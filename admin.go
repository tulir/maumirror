// maumirror - A GitHub repo mirroring system using webhooks.
// Copyright (C) 2021 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-playground/webhooks/v6/github"

	log "maunium.net/go/maulogger/v2"
)

var (
	ErrInvalidAdminSecret = errors.New("invalid admin secret")
)

type CreateMirrorRequest struct {
	Name    string     `json:"name"`
	Repo    Repository `json:"repo"`
	PushKey string     `json:"push_key"`
	PullKey string     `json:"pull_key"`

	GitHubToken string `json:"github_token"`

	GitLabProjectID int64  `json:"gitlab_project_id"`
	GitLabToken     string `json:"gitlab_token"`
	GitLabURL       string `json:"gitlab_url"`
}

func writeKey(key, path, name string) (string, error) {
	if key != "" {
		if path == "" {
			home, _ := os.UserHomeDir()
			path = filepath.Join(home, ".ssh", "push", name)
		}
		_ = os.MkdirAll(filepath.Dir(path), 0700)
		err := os.WriteFile(path, []byte(key), 0600)
		if err != nil {
			log.Warnfln("Failed to write SSH key for %s to %s: %v", name, path, err)
			return path, err
		}
		log.Infoln("Wrote SSH key for", name, "to", path)
	}
	return path, nil
}

func createMirror(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondErr(w, r, github.ErrInvalidHTTPMethod, http.StatusBadRequest)
		return
	}
	header := r.Header.Get("Authorization")
	if config.Server.AdminSecret != "" && header != "Bearer "+config.Server.AdminSecret {
		respondErr(w, r, ErrInvalidAdminSecret, http.StatusUnauthorized)
		return
	}

	var req CreateMirrorRequest
	if data, err := io.ReadAll(r.Body); err != nil {
		respondErr(w, r, github.ErrParsingPayload, http.StatusBadRequest)
		return
	} else if err = json.Unmarshal(data, &req); err != nil {
		respondErr(w, r, err, http.StatusBadRequest)
		return
	}

	repo := &req.Repo
	repo.Name = req.Name
	repo.Log = log.Sub(repo.Name)

	log.Debugln("Create mirror request from %s: %s to %s", readUserIP(r), repo.Name, repo.Target)

	log.Infoln("Adding", repo.Name, "with push target", repo.Target, "to repos")
	config.Repositories[repo.Name] = repo

	var err error
	if req.GitHubToken != "" {
		repo.Secret, err = CreateGitHubWebhook(req.GitHubToken, repo.Name, repo.Secret)
		if err != nil {
			respondErr(w, r, err, http.StatusInternalServerError)
			return
		}
	}
	if repo.PushKey, err = writeKey(req.PushKey, repo.PushKey, repo.Name); err != nil {
		respondErr(w, r, err, http.StatusInternalServerError)
		return
	} else if repo.PullKey, err = writeKey(req.PullKey, repo.PullKey, repo.Name); err != nil {
		respondErr(w, r, err, http.StatusInternalServerError)
		return
	}

	if appGHClient != nil && req.GitLabProjectID != 0 && req.GitLabToken != "" && req.GitLabURL != "" {
		parts := strings.Split(repo.Name, "/")
		ciRepo := &CIRepository{
			Secret:        RandString(50),
			Owner:         parts[0],
			Name:          parts[1],
			plock:         NewPartitionLocker(&sync.Mutex{}),
			checkSuiteIDs: make(map[string]int64),
			checkRunIDs:   make(map[int64]int64),
		}
		installation, _, err := appGHClient.Apps.FindRepositoryInstallation(r.Context(), ciRepo.Owner, ciRepo.Name)
		if err != nil {
			respondErr(w, r, fmt.Errorf("failed to find GitHub app installation ID: %w", err), http.StatusInternalServerError)
			return
		}
		log.Debugfln("Found installation ID for %s/%s: %d", ciRepo.Owner, ciRepo.Name, installation.GetID())
		ciRepo.InstallationID = installation.GetID()

		log.Debugln("Creating CI webhook for", req.GitLabProjectID)
		err = CreateGitLabWebhook(req.GitLabURL, req.GitLabToken, req.GitLabProjectID, ciRepo.Secret)
		if err != nil {
			respondErr(w, r, fmt.Errorf("failed to create CI webhook: %w", err), http.StatusInternalServerError)
			return
		}
		log.Infofln("Successfully created CI webhook for %d to mirror status to %s/%s", req.GitLabProjectID, ciRepo.Owner, ciRepo.Name)

		config.CIRepositories[req.GitLabProjectID] = ciRepo
	}

	log.Debugln("Saving config...")
	saveConfig()

	w.WriteHeader(http.StatusOK)
}
