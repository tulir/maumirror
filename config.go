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
	"sync"
)

type Config struct {
	// Cloned repo storage directory.
	DataDir string `yaml:"datadir"`

	// HTTP server configuration.
	Server struct {
		// Endpoint for admin API (e.g. dynamically adding webhooks).
		AdminEndpoint string `yaml:"admin_endpoint,omitempty"`
		// Secret for accessing admin API.
		AdminSecret string `yaml:"admin_secret,omitempty"`
		// Endpoint for receiving webhooks.
		WebhookEndpoint string `yaml:"webhook_endpoint"`
		// Public URL where the webhook endpoint is accessible. Used for installing GitHub webhooks automatically.
		WebhookPublicURL string `yaml:"webhook_public_url,omitempty"`

		// Whether or not to trust X-Forwarded-For headers for logging.
		TrustForwardHeaders bool `yaml:"trust_forward_headers,omitempty"`
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
			Push *Script `yaml:"push,omitempty"`
		} `yaml:"scripts,omitempty"`
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
	sync.Mutex

	Remotes []Remote `yaml:"remotes" json:"remotes"`
	Source string `yaml:"source,omitempty" json:"source"`
	Secret string `yaml:"secret,omitempty" json:"secret"`
	// Target repo URL. Required.
	Target  string `yaml:"target" json:"target"`
	PushKey string `yaml:"push_key,omitempty" json:"push_key"`
	// Path to SSH key for pulling repo. If set, source repo URL defaults to ssh instead of https.
	PullKey string `yaml:"pull_key,omitempty" json:"pull_key"`

	Name string           `yaml:"-" json:"-"`
	Log  maulogger.Logger `yaml:"-" json:"-"`
}

type Remote struct {
	// ID that identifies
	ID string `yaml:"id" json:"id"`
	// Repository source URL. Optional, defaults to https.
	URL string `yaml:"url,omitempty" json:"url,omitempty"`
	// Webhook auth secret. Request signature is not checked if secret is not configured.
	Secret string `yaml:"secret,omitempty" json:"secret,omitempty"`
	// Path to SSH key for the remote.
	Key string `yaml:"key,omitempty" json:"key,omitempty"`
}
