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

import "maunium.net/go/maulogger/v2"

type Config struct {
	DataDir string `json:"datadir"`

	ListenPath    string `json:"endpoint"`
	ListenAddress string `json:"address"`

	Repositories map[string]*Repository `json:"repositories"`
}

type Repository struct {
	Source  string `json:"source"`
	Secret  string `json:"secret"`
	Target  string `json:"target"`
	PushKey string `json:"push_key"`
	PullKey string `json:"pull_key"`

	Name string `json:"-"`
	Log maulogger.Logger `json:"-"`
}
