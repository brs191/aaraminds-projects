// Package teams posts messages to a Microsoft Teams channel via a Power Automate
// "Workflows" incoming webhook.
//
// IMPORTANT (verified 2026-05-31): the legacy Office 365 "Incoming Webhook" connector
// — and its MessageCard payload — is being retired in Teams, with the final rollout
// 2026-05-18..22. New connectors have been blocked since 2024-08-15. So this adapter
// targets a Power Automate Workflows webhook and sends an Adaptive Card wrapped in the
// Workflows `attachments` envelope. (Workflows also accepts MessageCard, but it cannot
// render action buttons — which the HITL approval flow will need — so we standardize on
// Adaptive Card now.) TEAMS_WEBHOOK_URL must be a Workflows flow URL, not a connector URL.
package teams

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Message is the payload the orchestrator sends to POST /post.
type Message struct {
	Title    string `json:"title"`
	Markdown string `json:"markdown"`
	Channel  string `json:"channel,omitempty"`
}

// Poster delivers messages to Teams.
type Poster struct {
	WebhookURL string
	client     *http.Client
}

// NewPoster returns a Poster. An empty webhookURL means "stub mode" — Post
// reports the message as not delivered so the caller can log a preview instead.
func NewPoster(webhookURL string) *Poster {
	return &Poster{
		WebhookURL: webhookURL,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

// Post sends msg to Teams as an Adaptive Card via a Workflows webhook. Returns
// delivered=false (no error) when no webhook is configured (stub mode).
func (p *Poster) Post(msg Message) (delivered bool, err error) {
	if p.WebhookURL == "" {
		return false, nil
	}
	// Adaptive Card wrapped in the Teams