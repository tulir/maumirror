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
	"io/ioutil"
	"net/http"
	"os"
	"runtime/debug"
	"sync"

	"gopkg.in/go-playground/webhooks.v5/github"
	"gopkg.in/yaml.v2"

	"maunium.net/go/mauflag"
	log "maunium.net/go/maulogger/v2"
)

var configPath = mauflag.MakeFull("c", "config", "Path to config file", "config.yaml").String()
var wantHelp, _ = mauflag.MakeHelpFlag()

var config Config
var lock = NewPartitionLocker(&sync.Mutex{})
var hook, _ = github.New()

func main() {
	if err := mauflag.Parse(); err != nil {
		mauflag.PrintHelp()
		os.Exit(1)
	} else if *wantHelp {
		mauflag.PrintHelp()
		os.Exit(0)
	} else if configData, err := ioutil.ReadFile(*configPath); err != nil {
		log.Fatalln("Failed to read config:", err)
		os.Exit(10)
	} else if err := yaml.Unmarshal(configData, &config); err != nil {
		log.Fatalln("Failed to parse config:", err)
		os.Exit(11)
	}

	for name, repo := range config.Repositories {
		repo.Name = name
		repo.Log = log.Sub(name)
	}

	root := http.NewServeMux()
	root.HandleFunc(config.Server.WebhookEndpoint, handleWebhook)
	if config.Server.AdminEndpoint != "" {
		adminMux := http.NewServeMux()
		adminMux.HandleFunc("/create", createMirror)
		root.Handle(config.Server.AdminEndpoint, adminMux)
		log.Infoln("Admin listener enabled at", config.Server.AdminEndpoint)
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
	} else if err := ioutil.WriteFile(*configPath, data, 0600); err != nil {
		log.Errorln("Failed to write config:", err)
	}
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	defer func() {
		err := recover()
		if err != nil {
			log.Errorln("Handling request from", readUserIP(r), "panicked:", err)
			debug.PrintStack()
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Errorfln("Failed to read body in request from %s: %v", readUserIP(r), err)
		respondErr(w, r, github.ErrParsingPayload, http.StatusBadRequest)
		return
	} else if err = r.Body.Close(); err != nil {
		log.Errorfln("Failed to close body reader in request from %s: %v", readUserIP(r), err)
		respondErr(w, r, github.ErrParsingPayload, http.StatusBadRequest)
		return
	}
	r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

	rawEvt, err := hook.Parse(r, github.PushEvent, github.PingEvent)
	if err != nil {
		respondErr(w, r, err, http.StatusBadRequest)
		return
	}

	r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

	switch evt := rawEvt.(type) {
	case github.PingPayload:
		if repo, err, code := checkSig(r, evt.Repository.FullName); err != nil {
			respondErr(w, r, err, code)
		} else {
			repo.Log.Infoln("Received webhook ping from", readUserIP(r))
		}
	case github.PushPayload:
		if repo, err, code := checkSig(r, evt.Repository.FullName); err != nil {
			respondErr(w, r, err, code)
		} else {
			w.WriteHeader(handlePushEvent(repo, evt))
		}
	case github.ReleasePayload:
		if repo, err, code := checkSig(r, evt.Repository.FullName); err != nil {
			respondErr(w, r, err, code)
		} else {
			w.WriteHeader(handleReleaseEvent(repo, evt))
		}
	}
}
