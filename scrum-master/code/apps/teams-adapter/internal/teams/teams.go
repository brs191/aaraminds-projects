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
	// Adaptive Card wrapped in the Teams Workflows `attachments` envelope.
	payload := map[string]any{
		"type": "message",
		"attachments": []map[string]any{
			{
				"contentType": "application/vnd.microsoft.card.adaptive",
				"content": map[string]any{
					"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
					"type":    "AdaptiveCard",
					"version": "1.4",
					"body": []map[string]any{
						{"type": "TextBlock", "text": msg.Title, "weight": "Bolder", "size": "Large", "wrap": true},
						{"type": "TextBlock", "text": msg.Markdown, "wrap": true},
					},
				},
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return false, err
	}
	resp, err := p.client.Post(p.WebhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		return false, fmt.Errorf("teams webhook returned status %d", resp.StatusCode)
	}
	return true, nil
}
