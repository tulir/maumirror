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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"strconv"
	"sync"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/go-playground/webhooks/v6/gitlab"
	"github.com/google/go-github/v40/github"
	log "maunium.net/go/maulogger/v2"
)

var glHook, _ = gitlab.New()
var appGHClient *github.Client
var appTransport *ghinstallation.AppsTransport
var installationClients = make(map[int64]*github.Client)
var installationClientsLock sync.Mutex

func installationGHClient(installationID int64) *github.Client {
	installationClientsLock.Lock()
	cli, ok := installationClients[installationID]
	if !ok {
		cli = github.NewClient(&http.Client{
			Transport: ghinstallation.NewFromAppsTransport(appTransport, installationID),
		})
		installationClients[installationID] = cli
	}
	installationClientsLock.Unlock()
	return cli
}

func initGHClient() {
	var err error
	appTransport, err = ghinstallation.NewAppsTransport(http.DefaultTransport, config.GitHubApp.ID, []byte(config.GitHubApp.PrivateKey))
	if err != nil {
		panic(err)
	}
	appGHClient = github.NewClient(&http.Client{Transport: appTransport})

	installationIDChanged := false
	for _, repo := range config.CIRepositories {
		if repo.InstallationID != 0 {
			continue
		}
		installation, _, err := appGHClient.Apps.FindRepositoryInstallation(context.Background(), repo.Owner, repo.Name)
		if err != nil {
			log.Errorfln("Failed to find installation ID for %s/%s: %v", repo.Owner, repo.Name, err)
			repo.InstallationID = -1
		} else {
			log.Infofln("Found installation ID for %s/%s: %d", repo.Owner, repo.Name, installation.GetID())
			repo.InstallationID = installation.GetID()
		}
		installationIDChanged = true
	}
	if installationIDChanged {
		saveConfig()
	}
}

func ensureCheckSuiteExists(repo *CIRepository, ref, sha string) {
	repo.mapLock.RLock()
	_, ok := repo.checkSuiteIDs[sha]
	repo.mapLock.RUnlock()
	if ok {
		return
	}
	cli := installationGHClient(repo.InstallationID)
	suite, resp, err := cli.Checks.CreateCheckSuite(context.Background(), repo.Owner, repo.Name, github.CreateCheckSuiteOptions{
		HeadSHA:    sha,
		HeadBranch: &ref,
	})
	if err != nil {
		if resp.StatusCode == 422 {
			log.Debugfln("Got 422 while creating check suite for %s/%s in %s/%s", ref, sha, repo.Owner, repo.Name)
			repo.mapLock.Lock()
			repo.checkSuiteIDs[sha] = -1
			repo.mapLock.Unlock()
		} else {
			log.Errorfln("Failed to create check suite for %s/%s in %s/%s: %v", ref, sha, repo.Owner, repo.Name, err)
		}
	} else {
		log.Debugfln("Created check suite for %s/%s in %s/%s: %d", ref, sha, repo.Owner, repo.Name, *suite.ID)
		repo.mapLock.Lock()
		repo.checkSuiteIDs[sha] = suite.GetID()
		repo.mapLock.Unlock()
	}
}

func handlePipelineEvent(repo *CIRepository, evt gitlab.PipelineEventPayload) {
	repo.plock.Lock(evt.ObjectAttributes.SHA)
	defer repo.plock.Unlock(evt.ObjectAttributes.SHA)
	ensureCheckSuiteExists(repo, evt.ObjectAttributes.Ref, evt.ObjectAttributes.SHA)
}

var (
	statusInProgress = "in_progress"
	statusCompleted  = "completed"

	conclusionSuccess   = "success"
	conclusionCancelled = "cancelled"
	conclusionFailure   = "failure"
	conclusionNeutral   = "neutral"
	conclusionTimedOut  = "timed_out"
)

func makeUpdateFromCreate(opts github.CreateCheckRunOptions) github.UpdateCheckRunOptions {
	return github.UpdateCheckRunOptions{
		Name:        opts.Name,
		DetailsURL:  opts.DetailsURL,
		ExternalID:  opts.ExternalID,
		Status:      opts.Status,
		Conclusion:  opts.Conclusion,
		CompletedAt: opts.CompletedAt,
		Output:      opts.Output,
		Actions:     opts.Actions,
	}
}

func stringPtr(str string) *string {
	return &str
}

func handleJobEvent(repo *CIRepository, evt gitlab.JobEventPayload) {
	repo.plock.Lock(evt.SHA)
	defer repo.plock.Unlock(evt.SHA)
	log.Debugfln("Received build event in %d (%s) for build %d (%s). Current status is %s/%s",
		evt.ProjectID, evt.ProjectName, evt.BuildID, evt.BuildName, evt.BuildStatus, evt.BuildFailureReason)
	ensureCheckSuiteExists(repo, evt.Ref, evt.SHA)

	detailsURL := fmt.Sprintf("%s/-/jobs/%d", evt.Repository.Homepage, evt.BuildID)
	externalID := strconv.FormatInt(evt.BuildID, 10)
	opts := github.CreateCheckRunOptions{
		Name:       evt.BuildName,
		HeadSHA:    evt.SHA,
		DetailsURL: &detailsURL,
		ExternalID: &externalID,
		Output:     &github.CheckRunOutput{},
	}
	switch evt.BuildStatus {
	case "created":
		opts.Output.Title = stringPtr("Job created")
		opts.Output.Summary = stringPtr("This job has not been triggered yet. It depends on upstream jobs that need to succeed in order for this job to be triggered.")
	case "pending":
		opts.Output.Title = stringPtr("Job pending")
		opts.Output.Summary = stringPtr("This job is waiting for a runner to pick it up.")
	case "running":
		opts.Status = &statusInProgress
		opts.StartedAt = &github.Timestamp{Time: evt.BuildStartedAt.Time}
		opts.Output.Title = stringPtr("Job running")
		opts.Output.Summary = stringPtr("This job is running.")
	case "success":
		opts.Status = &statusCompleted
		opts.Conclusion = &conclusionSuccess
		opts.StartedAt = &github.Timestamp{Time: evt.BuildStartedAt.Time}
		opts.CompletedAt = &github.Timestamp{Time: evt.BuildFinishedAt.Time}
		opts.Output.Title = stringPtr("Job successful")
		opts.Output.Summary = stringPtr("This job is completed successfully.")
	case "canceled":
		opts.Status = &statusCompleted
		opts.Conclusion = &conclusionCancelled
		opts.StartedAt = &github.Timestamp{Time: evt.BuildStartedAt.Time}
		opts.CompletedAt = &github.Timestamp{Time: evt.BuildFinishedAt.Time}
		opts.Output.Title = stringPtr("Job canceled")
		opts.Output.Summary = stringPtr("This job was canceled.")
	case "failed":
		opts.Status = &statusCompleted
		opts.Conclusion = &conclusionFailure
		opts.Output.Title = stringPtr("Job failed")
		opts.Output.Summary = stringPtr("This job failed.")
		if evt.BuildFailureReason == "job_execution_timeout" {
			opts.Conclusion = &conclusionTimedOut
			opts.Output.Title = stringPtr("Job timed out")
			opts.Output.Summary = stringPtr("The script exceeded the maximum execution time set for the job.")
		} else if evt.BuildAllowFailure {
			opts.Conclusion = &conclusionNeutral
			opts.Output.Summary = stringPtr("This job failed, but the job is marked to allow failures.")
		}
		opts.StartedAt = &github.Timestamp{Time: evt.BuildStartedAt.Time}
		opts.CompletedAt = &github.Timestamp{Time: evt.BuildFinishedAt.Time}
	default:
		log.Warnfln("Unknown build status %s", evt.BuildStatus)
		return
	}

	repo.mapLock.RLock()
	runID, ok := repo.checkRunIDs[evt.BuildID]
	repo.mapLock.RUnlock()

	var run *github.CheckRun
	var err error
	var action string

	cli := installationGHClient(repo.InstallationID)
	// For running we have to create a new check run, because the go-github library doesn't expose StartedAt in the update fields
	if !ok || evt.BuildStatus == "running" {
		run, _, err = cli.Checks.CreateCheckRun(context.Background(), repo.Owner, repo.Name, opts)
		action = "create"
	} else {
		run, _, err = cli.Checks.UpdateCheckRun(context.Background(), repo.Owner, repo.Name, runID, makeUpdateFromCreate(opts))
		action = "update"
	}
	if err != nil {
		log.Errorfln("Failed to %s check run for %s/%s/%s in %s/%s: %v", action, evt.Ref, evt.SHA, evt.BuildName, repo.Owner, repo.Name, err)
	} else {
		log.Infofln("Successfully %sd check run for %s/%s/%s#%d in %s/%s. Run ID: %d, status: %s %s", action, evt.Ref, evt.SHA, evt.BuildName, evt.BuildID, repo.Owner, repo.Name, run.GetID(), run.GetStatus(), run.GetConclusion())
		if run.ID != nil && *run.ID != runID {
			repo.mapLock.Lock()
			repo.checkRunIDs[evt.BuildID] = *run.ID
			repo.mapLock.Unlock()
		}
	}
}

func handleCIWebhook(w http.ResponseWriter, r *http.Request) {
	defer func() {
		err := recover()
		if err != nil {
			log.Errorln("Handling GitLab webhook from", readUserIP(r), "panicked:", err)
			debug.PrintStack()
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondErr(w, r, err, http.StatusBadRequest)
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	rawEvt, err := glHook.Parse(r, gitlab.BuildEvents, gitlab.JobEvents, gitlab.PipelineEvents)
	if err != nil {
		respondErr(w, r, err, http.StatusBadRequest)
		return
	}

	// The webhook library is a bit of a mess, so fix it here
	if _, ok := rawEvt.(gitlab.BuildEventPayload); ok {
		var fixedPayload gitlab.JobEventPayload
		err = json.Unmarshal(body, &fixedPayload)
		if err != nil {
			respondErr(w, r, err, http.StatusBadRequest)
			return
		}
		rawEvt = fixedPayload
	}

	switch evt := rawEvt.(type) {
	case gitlab.JobEventPayload:
		if repo, err, code := checkGLToken(r, evt.ProjectID); err != nil {
			respondErr(w, r, err, code)
		} else {
			log.Debugfln("Handling job event from %d", evt.ProjectID)
			handleJobEvent(repo, evt)
			w.WriteHeader(http.StatusOK)
		}
	case gitlab.PipelineEventPayload:
		if repo, err, code := checkGLToken(r, evt.Project.ID); err != nil {
			respondErr(w, r, err, code)
		} else {
			log.Debugfln("Handling pipeline event from %d", evt.Project.ID)
			handlePipelineEvent(repo, evt)
			w.WriteHeader(http.StatusOK)
		}
	default:
		log.Errorfln("Unexpected event type %T", evt)
	}
}
