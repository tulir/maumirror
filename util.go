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
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"net/http"

	"gopkg.in/go-playground/webhooks.v5/github"

	log "maunium.net/go/maulogger/v2"
)

func checkSig(r *http.Request, repoName string) (repo *Repository, err error) {
	signature := r.Header.Get("X-Hub-Signature")
	if len(signature) == 0 {
		err = github.ErrMissingHubSignatureHeader
		return
	}
	repo, ok := config.Repositories[repoName]
	if !ok {
		err = errors.New("unknown repository")
		return
	}
	mac := hmac.New(sha1.New, []byte(repo.Secret))

	payload, err := ioutil.ReadAll(r.Body)
	if err != nil || len(payload) == 0 {
		err = github.ErrParsingPayload
		return
	}

	_, _ = mac.Write(payload)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature[5:]), []byte(expectedMAC)) {
		err = github.ErrHMACVerificationFailed
	}
	return
}

func respondErr(w http.ResponseWriter, err error) {
	log.Errorln("Failed to handle request:", err.Error())
	_, _ = w.Write([]byte(err.Error()))
	w.WriteHeader(http.StatusBadRequest)
}
