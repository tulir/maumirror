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
)

const pushScript = `#!/bin/bash
if [[ ! -d $MM_REPOSITORY_OWNER ]]; then
	mkdir $MM_REPOSITORY_OWNER
fi
cd $MM_REPOSITORY_OWNER
if [[ ! -d $MM_REPOSITORY_NAME.git ]]; then
	git clone --mirror $MM_SOURCE_URL $MM_REPOSITORY_NAME.git
	cd $MM_REPOSITORY_NAME.git
	git remote set-url --push origin $MM_TARGET_URL
else
	cd $MM_REPOSITORY_NAME.git
	git fetch -p origin
fi
git push --mirror
`

func handlePushEvent(repo Repository, evt github.PushPayload) int {
	lock.Lock(evt.Repository.FullName)
	defer lock.Unlock(evt.Repository.FullName)

	cmd := exec.Command("/bin/bash", "/dev/stdin")
	cmd.Env = append(cmd.Env, "MM_SOURCE_URL="+evt.Repository.GitURL,
		"MM_TARGET_URL="+repo.Target,
		"MM_REPOSITORY_NAME="+evt.Repository.Name,
		"MM_REPOSITORY_OWNER="+evt.Repository.Owner.Login)
	cmd.Dir = config.DataDir

	if pipe, err := cmd.StdinPipe(); err != nil {
		printErr("Failed to open stdin pipe for subprocess:", err)
		return http.StatusInternalServerError
	} else if _, err := pipe.Write([]byte(pushScript)); err != nil {
		printErr("Failed to write script to stdin of subprocess:", err)
		return http.StatusInternalServerError
	} else if err := cmd.Run(); err != nil {
		printErr("Failed to run command:", err)
		return http.StatusInternalServerError
	}
	return http.StatusOK
}
