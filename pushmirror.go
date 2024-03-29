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
	_ "embed"
	"io"
	"net/http"
	"os/exec"
	"runtime/debug"

	"github.com/go-playground/webhooks/v6/github"
	log "maunium.net/go/maulogger/v2"
)

//go:embed push_script.sh
var PushScript string

func handlePushEvent(repo *Repository, evt github.PushPayload) int {
	lock.Lock(evt.Repository.FullName)
	defer lock.Unlock(evt.Repository.FullName)

	cmd := exec.Command(config.Shell.Command, config.Shell.Args...)
	cmd.Dir = config.DataDir
	cmd.Env = append(cmd.Env,
		"MM_REPOSITORY_NAME="+evt.Repository.Name,
		"MM_REPOSITORY_OWNER="+evt.Repository.Owner.Login,
		"MM_SOURCE_URL="+evt.Repository.GitURL,
		"MM_SOURCE_URL_OVERRIDE="+repo.Source,
		"MM_SOURCE_KEY_PATH="+repo.PullKey,

		"MM_TARGET_URL="+repo.Target,
		"MM_TARGET_KEY_PATH="+repo.PushKey)
	cmd.Stderr = repo.Log.Writer(log.LevelError)
	cmd.Stdout = repo.Log.Writer(log.LevelInfo)

	script := PushScript
	if config.Shell.Scripts.Push != nil && len(config.Shell.Scripts.Push.Data) > 0 {
		repo.Log.Debugln("Using push handler script from", config.Shell.Scripts.Push.Path)
		script = config.Shell.Scripts.Push.Data
	}

	if stdin, err := cmd.StdinPipe(); err != nil {
		repo.Log.Errorln("Failed to open stdin pipe for subprocess:", err)
		return http.StatusInternalServerError
	} else if _, err = stdin.Write([]byte(script)); err != nil {
		repo.Log.Errorln("Failed to write script to stdin of subprocess:", err)
		return http.StatusInternalServerError
	} else if err = cmd.Start(); err != nil {
		repo.Log.Errorln("Failed to start command:", err)
		return http.StatusInternalServerError
	} else if err = stdin.Close(); err != nil {
		repo.Log.Warnln("Failed to close stdin:", err)
	}
	if err := cmd.Wait(); err != nil {
		repo.Log.Errorln("Error waiting for command:", err)
		return http.StatusInternalServerError
	}
	return http.StatusOK
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	defer func() {
		err := recover()
		if err != nil {
			log.Errorln("Handling GitHub webhook from", readUserIP(r), "panicked:", err)
			debug.PrintStack()
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Errorfln("Failed to read body in request from %s: %v", readUserIP(r), err)
		respondErr(w, r, github.ErrParsingPayload, http.StatusBadRequest)
		return
	} else if err = r.Body.Close(); err != nil {
		log.Errorfln("Failed to close body reader in request from %s: %v", readUserIP(r), err)
		respondErr(w, r, github.ErrParsingPayload, http.StatusBadRequest)
		return
	}
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	rawEvt, err := ghHook.Parse(r, github.PushEvent, github.PingEvent)
	if err != nil {
		respondErr(w, r, err, http.StatusBadRequest)
		return
	}

	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

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
	}
}
