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
	"io/ioutil"
	"maunium.net/go/maulogger/v2"
)

type Config struct {
	// Cloned repo storage directory.
	DataDir string `json:"datadir"`

	// HTTP server configuration.
	Server struct {
		// Endpoint for admin API (e.g. dynamically adding webhooks).
		AdminEndpoint string `json:"admin_endpoint"`
		// Endpoint for receiving webhooks.
		WebhookEndpoint string `json:"webhook_endpoint"`

		// Whether or not to trust X-Forwarded-For headers for logging.
		TrustForwardHeaders bool `json:"trust_forward_headers"`
		// IP and port where the server listens
		Address string `json:"address"`
	} `json:"server"`

	// Shell configuration
	Shell struct {
		// The command to start shells with
		Command string `json:"command"`
		// The arguments to pass to shells. The script is sent through stdin.
		Args []string `json:"args"`
		// Paths to scripts. If unset, will default to built-in handlers.
		Scripts struct {
			Push Script `json:"push"`
		} `json:"scripts"`
	} `json:"shell"`

	// Repository configuration
	Repositories map[string]*Repository `json:"repositories"`
}

type Script struct {
	Path string
	Data string
}

func (script *Script) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &script.Path); err != nil {
		return err
	}
	if len(script.Path) > 0 {
		if fileData, err := ioutil.ReadFile(script.Path); err != nil {
			return err
		} else {
			script.Data = string(fileData)
		}
	}
	return nil
}

func (script *Script) MarshalJSON() ([]byte, error) {
	if len(script.Path) > 0 {
		if err := ioutil.WriteFile(script.Path, []byte(script.Data), 0644); err != nil {
			return nil, err
		}
	}
	return json.Marshal(script.Path)
}

type Repository struct {
	// Repository source URL.
	Source string `json:"source"`
	// Webhook auth secret. Request auth is not checked if secret is not configured.
	Secret string `json:"secret"`
	// Target repo URL. Required.
	Target string `json:"target"`
	// Path to SSH key for pushing repo.
	PushKey string `json:"push_key"`
	// Path to SSH key for pulling repo. If set, source repo URL defaults to ssh instead of https.
	PullKey string `json:"pull_key"`

	Name string           `json:"-"`
	Log  maulogger.Logger `json:"-"`
}
