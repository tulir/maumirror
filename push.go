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
	"net/http"
	"os/exec"

	"gopkg.in/go-playground/webhooks.v5/github"

	log "maunium.net/go/maulogger/v2"
)

const pushScript = `#!/bin/bash
if [[ ! -d $MM_REPOSITORY_OWNER ]]; then
	echo "Creating $(pwd)/$MM_REPOSITORY_OWNER"
	mkdir $MM_REPOSITORY_OWNER
fi
cd $MM_REPOSITORY_OWNER
if [[ ! -z "$MM_SOURCE_KEY_PATH" ]]; then
	export GIT_SSH_COMMAND="ssh -o StrictHostKeyChecking=no -i $MM_SOURCE_KEY_PATH"
	SOURCE_URL="git@github.com:$MM_REPOSITORY_OWNER/$MM_REPOSITORY_NAME.git"
else
	SOURCE_URL="https://github.com/$MM_REPOSITORY_OWNER/$MM_REPOSITORY_NAME.git"
fi
if [[ ! -z "$MM_SOURCE_URL_OVERRIDE" ]]; then
	SOURCE_URL="$MM_SOURCE_URL_OVERRIDE"
fi
if [[ ! -d $MM_REPOSITORY_NAME.git ]]; then
	echo "Cloning $SOURCE_URL to $(pwd)/$MM_REPOSITORY_NAME.git"
	git clone --quiet --mirror $SOURCE_URL $MM_REPOSITORY_NAME.git
	cd $MM_REPOSITORY_NAME.git
	git remote set-url --push origin $MM_TARGET_URL
else
	cd $MM_REPOSITORY_NAME.git
	echo "Fetching $(git remote get-url origin)"
	git fetch --quiet -p origin
fi
if [[ ! -z "$MM_TARGET_KEY_PATH" ]]; then
	export GIT_SSH_COMMAND="ssh -o StrictHostKeyChecking=no -i $MM_TARGET_KEY_PATH"
else
	unset GIT_SSH_COMMAND
fi
echo "Pushing to $(git remote get-url --push origin)"
git push --quiet --mirror
echo "Mirroring complete"
exit 0
`

func handlePushEvent(repo *Repository, evt github.PushPayload) int {
	lock.Lock(evt.Repository.FullName)
	defer lock.Unlock(evt.Repository.FullName)

	cmd := exec.Command(config.Shell.Command, config.Shell.Args...)
	cmd.Dir = config.DataDir
	cmd.Env = append(cmd.Env,
		"MM_SOURCE_URL="+evt.Repository.GitURL,
		"MM_REPOSITORY_NAME="+evt.Repository.Name,
		"MM_REPOSITORY_OWNER="+evt.Repository.Owner.Login,
		"MM_SOURCE_URL_OVERRIDE="+repo.Source,

		"MM_SOURCE_KEY_PATH="+repo.PullKey,
		"MM_TARGET_URL="+repo.Target,
		"MM_TARGET_KEY_PATH="+repo.PushKey)
	cmd.Stderr = repo.Log.WithDefaultLevel(log.LevelError)
	cmd.Stdout = repo.Log.WithDefaultLevel(log.LevelInfo)

	script := pushScript
	if len(config.Shell.Scripts.Push.Data) > 0 {
		repo.Log.Debugln("Using push handler script from", config.Shell.Scripts.Push.Path)
		script = config.Shell.Scripts.Push.Data
	}

	if stdin, err := cmd.StdinPipe(); err != nil {
		repo.Log.Errorln("Failed to open stdin pipe for subprocess:", err)
		return http.StatusInternalServerError
	} else if _, err := stdin.Write([]byte(script)); err != nil {
		repo.Log.Errorln("Failed to write script to stdin of subprocess:", err)
		return http.StatusInternalServerError
	} else if err := cmd.Start(); err != nil {
		repo.Log.Errorln("Failed to start command:", err)
		return http.StatusInternalServerError
	} else {
		if err := cmd.Wait(); err != nil {
			repo.Log.Errorln("Error waiting for command:", err)
			return http.StatusInternalServerError
		}
	}
	return http.StatusOK
}
