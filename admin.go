// maumirror - A GitHub repo mirroring system using webhooks.
// Copyright (C) 2019 Tulir Asokan
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
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"gopkg.in/go-playground/webhooks.v5/github"

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
}

func writeKey(key, path, name string) error {
	if key != "" {
		if path == "" {
			home, _ := os.UserHomeDir()
			path = filepath.Join(home, ".ssh", "push", name)
		}
		_ = os.MkdirAll(filepath.Dir(path), 0700)
		err := ioutil.WriteFile(path, []byte(key), 0600)
		if err != nil {
			log.Warnln("Failed to write SSH key for", name, "to", path + ":", err)
			return err
		}
		log.Infoln("Wrote SSH key for", name, "to", path)
	}
	return nil
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
	if data, err := ioutil.ReadAll(r.Body); err != nil {
		respondErr(w, r, github.ErrParsingPayload, http.StatusBadRequest)
		return
	} else if err = json.Unmarshal(data, &req); err != nil {
		respondErr(w, r, err, http.StatusBadRequest)
		return
	}
	if err := writeKey(req.PushKey, req.Repo.PushKey, req.Name); err != nil {
		respondErr(w, r, err, http.StatusInternalServerError)
		return
	} else if err := writeKey(req.PullKey, req.Repo.PullKey, req.Name); err != nil {
		respondErr(w, r, err, http.StatusInternalServerError)
		return
	}

	if req.GitHubToken != "" {
		var err error
		req.Repo.Secret, err = CreateWebhook(req.GitHubToken, req.Name, req.Repo.Secret)
		if err != nil {
			respondErr(w, r, err, http.StatusInternalServerError)
			return
		}
	}

	log.Infoln("Adding", req.Name, "with push target", req.Repo.Target, "to repos")
	config.Repositories[req.Name] = &req.Repo
	log.Debugln("Saving config...")
	saveConfig()

	w.WriteHeader(http.StatusOK)
}
