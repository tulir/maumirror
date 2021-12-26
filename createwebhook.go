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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	log "maunium.net/go/maulogger/v2"
)

type GHCreateWebhookPayload struct {
	Name   string                `json:"name"`
	Active bool                  `json:"active"`
	Events []string              `json:"events"`
	Config GHCreateWebhookConfig `json:"config"`
}

type GHCreateWebhookConfig struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Secret      string `json:"secret"`
	InsecureSSL string `json:"insecure_ssl"`
}

func NewGHCreateWebhookPayload(secret string) GHCreateWebhookPayload {
	if secret == "" {
		secret = RandString(50)
	}
	return GHCreateWebhookPayload{
		Name:   "web",
		Active: true,
		Events: []string{"push"},
		Config: GHCreateWebhookConfig{
			URL:         config.Server.WebhookPublicURL,
			ContentType: "json",
			Secret:      secret,
		},
	}
}

const GHWebhookAPIURL = "https://api.github.com/repos/%s/hooks"

func CreateGitHubWebhook(accessToken, repo, secret string) (string, error) {
	payload := NewGHCreateWebhookPayload(secret)
	var body bytes.Buffer

	log.Debugln("Creating webhook for", repo)
	if err := json.NewEncoder(&body).Encode(&payload); err != nil {
		return "", fmt.Errorf("failed to encode webhook create body: %w", err)
	} else if req, err := http.NewRequest(http.MethodPost, fmt.Sprintf(GHWebhookAPIURL, repo), &body); err != nil {
		return "", fmt.Errorf("failed to create webhook create request: %w", err)
	} else if req.Header.Set("Authorization", "token "+accessToken); false {
		return "", errors.New("false = true")
	} else if resp, err := http.DefaultClient.Do(req); err != nil {
		return "", fmt.Errorf("failed to send webhook create request: %w", err)
	} else if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected HTTP status %d: %s", resp.Status, string(respBody))
	} else {
		log.Infoln("Created webhook for", repo)
		return payload.Config.Secret, nil
	}
}

type GLCreateWebhookPayload struct {
	URL   string `json:"url"`
	Token string `json:"token"`

	EnableSSLVerification bool `json:"enable_ssl_verification"`

	ConfidentialIssuesEvents bool   `json:"confidential_issues_events"`
	ConfidentialNoteEvents   bool   `json:"confidential_note_events"`
	DeploymentEvents         bool   `json:"deployment_events"`
	IssuesEvents             bool   `json:"issues_events"`
	JobEvents                bool   `json:"job_events"`
	MergeRequestsEvents      bool   `json:"merge_requests_events"`
	NoteEvents               bool   `json:"note_events"`
	PipelineEvents           bool   `json:"pipeline_events"`
	PushEventsBranchFilter   string `json:"push_events_branch_filter"`
	PushEvents               bool   `json:"push_events"`
	TagPushEvents            bool   `json:"tag_push_events"`
	WikiPageEvents           bool   `json:"wiki_page_events"`
}

func CreateGitLabWebhook(baseURL, accessToken string, projectID int64, secret string) error {
	url := fmt.Sprintf("%s/api/v4/projects/%d/hooks", baseURL, projectID)
	payload := &GLCreateWebhookPayload{
		URL:   config.Server.CIWebhookPublicURL,
		Token: secret,

		EnableSSLVerification: true,

		JobEvents:      true,
		PipelineEvents: true,
	}
	var body bytes.Buffer

	if err := json.NewEncoder(&body).Encode(&payload); err != nil {
		log.Warnfln("Failed to create CI webhook for %s: %v", projectID, err)
		return fmt.Errorf("failed to encode webhook create body: %w", err)
	} else if req, err := http.NewRequest(http.MethodPost, url, &body); err != nil {
		log.Warnfln("Failed to create CI webhook for %s: %v", projectID, err)
		return fmt.Errorf("failed to create webhook create request: %w", err)
	} else if req.Header.Set("PRIVATE-TOKEN", accessToken); false {
		return errors.New("false = true")
	} else if req.Header.Set("Content-Type", "application/json"); false {
		return errors.New("false = true")
	} else if resp, err := http.DefaultClient.Do(req); err != nil {
		return fmt.Errorf("failed to send webhook create request: %w", err)
	} else if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected HTTP status %d: %s", resp.Status, string(respBody))
	} else {
		return nil
	}
}
