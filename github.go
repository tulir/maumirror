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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type CreateWebhookPayload struct {
	Name   string              `json:"name"`
	Active bool                `json:"active"`
	Events []string            `json:"events"`
	Config CreateWebhookConfig `json:"config"`
}

type CreateWebhookConfig struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Secret      string `json:"secret"`
	InsecureSSL string `json:"insecure_ssl"`
}

func NewCreateWebhookPayload(secret string) CreateWebhookPayload {
	if secret == "" {
		secret = RandString(50)
	}
	return CreateWebhookPayload{
		Name:   "maumirror",
		Active: true,
		Events: []string{"push"},
		Config: CreateWebhookConfig{
			URL:         config.Server.WebhookPublicURL,
			ContentType: "json",
			Secret:      secret,
			InsecureSSL: "1",
		},
	}
}

const WebhookAPIURL = "https://api.github.com/repos/%s/hooks"

func CreateWebhook(accessToken, repo, secret string) (string, error) {
	payload := NewCreateWebhookPayload(secret)
	var body bytes.Buffer

	if err := json.NewEncoder(&body).Encode(&payload); err != nil {
		return "", err
	} else if req, err := http.NewRequest(http.MethodPost, fmt.Sprintf(WebhookAPIURL, repo), &body); err != nil {
		return "", err
	} else if req.Header.Set("Authorization", "token "+accessToken); false {
		return "", errors.New("false = true")
	} else if resp, err := http.DefaultClient.Do(req); err != nil {
		return "", err
	} else if resp.StatusCode != http.StatusCreated {
		return "", errors.New(resp.Status)
	} else {
		return payload.Config.Secret, nil
	}
}
