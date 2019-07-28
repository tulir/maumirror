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
	"io/ioutil"
	"maunium.net/go/maulogger/v2"
)

type Config struct {
	// Cloned repo storage directory.
	DataDir string `yaml:"datadir"`

	// HTTP server configuration.
	Server struct {
		// Endpoint for admin API (e.g. dynamically adding webhooks).
		AdminEndpoint string `yaml:"admin_endpoint"`
		// Secret for accessing admin API.
		AdminSecret string `yaml:"admin_secret"`
		// Endpoint for receiving webhooks.
		WebhookEndpoint string `yaml:"webhook_endpoint"`
		// Public URL where the webhook endpoint is accessible. Used for installing GitHub webhooks automatically.
		WebhookPublicURL string `yaml:"webhook_public_url"`

		// Whether or not to trust X-Forwarded-For headers for logging.
		TrustForwardHeaders bool `yaml:"trust_forward_headers"`
		// IP and port where the server listens
		Address string `yaml:"address"`
	} `yaml:"server"`

	// Shell configuration
	Shell struct {
		// The command to start shells with
		Command string `yaml:"command"`
		// The arguments to pass to shells. The script is sent through stdin.
		Args []string `yaml:"args"`
		// Paths to scripts. If unset, will default to built-in handlers.
		Scripts struct {
			Push Script `yaml:"push"`
		} `yaml:"scripts"`
	} `yaml:"shell"`

	// Repository configuration
	Repositories map[string]*Repository `yaml:"repositories"`
}

type Script struct {
	Path string
	Data string
}

func (script *Script) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := unmarshal(&script.Path); err != nil {
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

func (script *Script) MarshalYAML() (interface{}, error) {
	if len(script.Path) > 0 {
		if err := ioutil.WriteFile(script.Path, []byte(script.Data), 0644); err != nil {
			return nil, err
		}
	}
	return script.Path, nil
}

type Repository struct {
	// Repository source URL. Optional, defaults to https.
	Source string `yaml:"source",json:"source"`
	// Webhook auth secret. Request signature is not checked if secret is not configured.
	Secret string `yaml:"secret",json:"secret"`
	// Target repo URL. Required.
	Target string `yaml:"target",json:"target"`
	// Path to SSH key for pushing repo.
	PushKey string `yaml:"push_key",json:"push_key"`
	// Path to SSH key for pulling repo. If set, source repo URL defaults to ssh instead of https.
	PullKey string `yaml:"pull_key",json:"pull_key"`

	Name string           `yaml:"-"`
	Log  maulogger.Logger `yaml:"-"`
}
