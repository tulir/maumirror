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
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/go-playground/webhooks/v6/github"
	"gopkg.in/yaml.v2"

	"maunium.net/go/mauflag"
	log "maunium.net/go/maulogger/v2"
)

var configPath = mauflag.MakeFull("c", "config", "Path to config file", "config.yaml").String()
var debugLogs = mauflag.MakeFull("d", "debug", "Print debug logs to stdout", "false").Bool()
var wantHelp, _ = mauflag.MakeHelpFlag()

var config Config
var lock = NewPartitionLocker(&sync.Mutex{})
var ghHook, _ = github.New()

func main() {
	if err := mauflag.Parse(); err != nil {
		mauflag.PrintHelp()
		os.Exit(1)
	} else if *wantHelp {
		mauflag.PrintHelp()
		os.Exit(0)
	} else if configData, err := os.ReadFile(*configPath); err != nil {
		log.Fatalln("Failed to read config:", err)
		os.Exit(10)
	} else if err := yaml.Unmarshal(configData, &config); err != nil {
		log.Fatalln("Failed to parse config:", err)
		os.Exit(11)
	}
	if *debugLogs {
		log.DefaultLogger.PrintLevel = log.LevelDebug.Severity
	}

	for name, repo := range config.Repositories {
		repo.Name = name
		repo.Log = log.Sub(name)
	}

	for _, repo := range config.CIRepositories {
		repo.checkSuiteIDs = make(map[string]int64)
		repo.checkRunIDs = make(map[int64]int64)
		repo.plock = NewPartitionLocker(&sync.Mutex{})
	}

	root := http.NewServeMux()
	root.HandleFunc(config.Server.WebhookEndpoint, handleWebhook)
	if len(config.GitHubApp.PrivateKey) > 0 && len(config.Server.CIWebhookEndpoint) > 0 {
		log.Debugfln("Initializing GitHub app client")
		initGHClient()
		root.HandleFunc(config.Server.CIWebhookEndpoint, handleCIWebhook)
	}
	if len(config.Server.AdminEndpoint) > 0 {
		log.Debugfln("Admin API is enabled")
		root.HandleFunc(fmt.Sprintf("%s/create", config.Server.AdminEndpoint), createMirror)
	}

	log.Infoln("Listening at", config.Server.Address)
	if err := http.ListenAndServe(config.Server.Address, root); err != nil {
		log.Fatalln("Fatal error in HTTP server")
		panic(err)
	}
}

func saveConfig() {
	if data, err := yaml.Marshal(&config); err != nil {
		log.Errorln("Failed to marshal config:", err)
		return
	} else if err := os.WriteFile(*configPath, data, 0600); err != nil {
		log.Errorln("Failed to write config:", err)
	}
}
