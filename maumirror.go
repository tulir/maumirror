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
	"net/http"
	"os"
	"runtime/debug"
	"sync"

	"gopkg.in/go-playground/webhooks.v5/github"
	"maunium.net/go/mauflag"
)

var config Config
var lock = NewPartitionLocker(&sync.Mutex{})
var hook, _ = github.New()

func main() {
	var configPath = mauflag.MakeFull("c", "config", "Path to config file", "config.json").String()
	var wantHelp, _ = mauflag.MakeHelpFlag()

	if err := mauflag.Parse(); err != nil {
		mauflag.PrintHelp()
		os.Exit(1)
	} else if *wantHelp {
		mauflag.PrintHelp()
		os.Exit(0)
	} else if configData, err := ioutil.ReadFile(*configPath); err != nil {
		printErr("Failed to read config:", err)
		os.Exit(10)
	} else if err := json.Unmarshal(configData, &config); err != nil {
		printErr("Failed to parse config:", err)
		os.Exit(11)
	}

	http.HandleFunc(config.ListenPath, handleWebhook)
	if err := http.ListenAndServe(config.ListenAddress, nil); err != nil {
		printErr("Fatal error in HTTP server")
		panic(err)
	}
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	defer func() {
		err := recover()
		if err != nil {
			printErr("Event handler panicked:", err)
			debug.PrintStack()
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	rawEvt, err := hook.Parse(r, github.PushEvent)
	if err != nil {
		respondErr(w, err)
		return
	}

	switch evt := rawEvt.(type) {
	case github.PushPayload:
		if repo, err := checkSig(r, evt.Repository.FullName); err != nil {
			respondErr(w, err)
		} else {
			w.WriteHeader(handlePushEvent(repo, evt))
		}
	case github.ReleasePayload:
		if repo, err := checkSig(r, evt.Repository.FullName); err != nil {
			respondErr(w, err)
		} else {
			w.WriteHeader(handleReleaseEvent(repo, evt))
		}
	}
}
