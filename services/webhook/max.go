// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webhook

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	webhook_model "code.gitea.io/gitea/models/webhook"
	"code.gitea.io/gitea/modules/git"
	api "code.gitea.io/gitea/modules/structs"
	webhook_module "code.gitea.io/gitea/modules/webhook"
)

type (
	// MaxPayload represents a MAX messenger webhook payload.
	MaxPayload struct {
		Text string `json:"text"`
	}

	maxConvertor struct{}
)

func createMaxPayload(text string) MaxPayload {
	return MaxPayload{Text: strings.TrimSpace(text)}
}

// Create implements PayloadConvertor Create method
func (m maxConvertor) Create(p *api.CreatePayload) (MaxPayload, error) {
	refName := git.RefName(p.Ref).ShortName()
	return createMaxPayload(fmt.Sprintf("[%s] %s %s created", p.Repo.FullName, p.RefType, refName)), nil
}

// Delete implements PayloadConvertor Delete method
func (m maxConvertor) Delete(p *api.DeletePayload) (MaxPayload, error) {
	refName := git.RefName(p.Ref).ShortName()
	return createMaxPayload(fmt.Sprintf("[%s] %s %s deleted", p.Repo.FullName, p.RefType, refName)), nil
}

// Fork implements PayloadConvertor Fork method
func (m maxConvertor) Fork(p *api.ForkPayload) (MaxPayload, error) {
	return createMaxPayload(fmt.Sprintf("%s is forked to %s", p.Forkee.FullName, p.Repo.FullName)), nil
}

// Push implements PayloadConvertor Push method
func (m maxConvertor) Push(p *api.PushPayload) (MaxPayload, error) {
	branchName := git.RefName(p.Ref).ShortName()

	var text strings.Builder
	if p.TotalCommits == 1 {
		fmt.Fprintf(&text, "[%s:%s] 1 new commit", p.Repo.FullName, branchName)
	} else {
		fmt.Fprintf(&text, "[%s:%s] %d new commits", p.Repo.FullName, branchName, p.TotalCommits)
	}

	for _, commit := range p.Commits {
		authorName := ""
		if commit.Author != nil {
			authorName = " - " + commit.Author.Name
		}
		fmt.Fprintf(&text, "\n[%s] %s%s", commit.ID[:7], strings.TrimRight(commit.Message, "\r\n"), authorName)
	}

	return createMaxPayload(text.String()), nil
}

// Issue implements PayloadConvertor Issue method
func (m maxConvertor) Issue(p *api.IssuePayload) (MaxPayload, error) {
	text, _, extraMarkdown, _ := getIssuesPayloadInfo(p, noneLinkFormatter, true)
	if extraMarkdown != "" {
		text += "\n\n" + extraMarkdown
	}
	return createMaxPayload(text), nil
}

// IssueComment implements PayloadConvertor IssueComment method
func (m maxConvertor) IssueComment(p *api.IssueCommentPayload) (MaxPayload, error) {
	text, _, _ := getIssueCommentPayloadInfo(p, noneLinkFormatter, true)
	if p.Comment.Body != "" {
		text += "\n" + p.Comment.Body
	}
	return createMaxPayload(text), nil
}

// PullRequest implements PayloadConvertor PullRequest method
func (m maxConvertor) PullRequest(p *api.PullRequestPayload) (MaxPayload, error) {
	text, _, extraMarkdown, _ := getPullRequestPayloadInfo(p, noneLinkFormatter, true)
	if extraMarkdown != "" {
		text += "\n" + extraMarkdown
	}
	return createMaxPayload(text), nil
}

// Review implements PayloadConvertor Review method
func (m maxConvertor) Review(p *api.PullRequestPayload, event webhook_module.HookEventType) (MaxPayload, error) {
	var text string
	switch p.Action {
	case api.HookIssueReviewed:
		action, err := parseHookPullRequestEventType(event)
		if err != nil {
			return MaxPayload{}, err
		}
		text = fmt.Sprintf("[%s] Pull request review %s: #%d %s", p.Repository.FullName, action, p.Index, p.PullRequest.Title)
		if p.Review.Content != "" {
			text += "\n" + p.Review.Content
		}
	}

	return createMaxPayload(text), nil
}

// Repository implements PayloadConvertor Repository method
func (m maxConvertor) Repository(p *api.RepositoryPayload) (MaxPayload, error) {
	switch p.Action {
	case api.HookRepoCreated:
		return createMaxPayload(fmt.Sprintf("[%s] Repository created", p.Repository.FullName)), nil
	case api.HookRepoDeleted:
		return createMaxPayload(fmt.Sprintf("[%s] Repository deleted", p.Repository.FullName)), nil
	}
	return MaxPayload{}, nil
}

// Wiki implements PayloadConvertor Wiki method
func (m maxConvertor) Wiki(p *api.WikiPayload) (MaxPayload, error) {
	text, _, _ := getWikiPayloadInfo(p, noneLinkFormatter, true)
	return createMaxPayload(text), nil
}

// Release implements PayloadConvertor Release method
func (m maxConvertor) Release(p *api.ReleasePayload) (MaxPayload, error) {
	text, _ := getReleasePayloadInfo(p, noneLinkFormatter, true)
	return createMaxPayload(text), nil
}

func (m maxConvertor) Package(p *api.PackagePayload) (MaxPayload, error) {
	text, _ := getPackagePayloadInfo(p, noneLinkFormatter, true)
	return createMaxPayload(text), nil
}

func (m maxConvertor) Status(p *api.CommitStatusPayload) (MaxPayload, error) {
	text, _ := getStatusPayloadInfo(p, noneLinkFormatter, true)
	return createMaxPayload(text), nil
}

func (maxConvertor) WorkflowRun(p *api.WorkflowRunPayload) (MaxPayload, error) {
	text, _ := getWorkflowRunPayloadInfo(p, noneLinkFormatter, true)
	return createMaxPayload(text), nil
}

func (maxConvertor) WorkflowJob(p *api.WorkflowJobPayload) (MaxPayload, error) {
	text, _ := getWorkflowJobPayloadInfo(p, noneLinkFormatter, true)
	return createMaxPayload(text), nil
}

func newMaxRequest(_ context.Context, w *webhook_model.Webhook, t *webhook_model.HookTask) (*http.Request, []byte, error) {
	var pc payloadConvertor[MaxPayload] = maxConvertor{}
	return newJSONRequest(pc, w, t, true)
}

func init() {
	RegisterWebhookRequester(webhook_module.MAX, newMaxRequest)
}
