// Package anthropic is a config-gated skeleton client for the Claude API,
// covering the two MVP LLM tasks: transcript cleanup (strict transformation
// rules, PRD 15.2) and grounded summarization (PRD 15.3). Never called in
// MODE=mock.
//
// Endpoint: POST https://api.anthropic.com/v1/messages
// Headers:  x-api-key: <key>, anthropic-version: 2023-06-01
// Cleanup should use a low/mid-cost model; summary a mid-tier model
// (PRD 12.3 model selection guidance).
package anthropic

import (
	"context"
	"net/http"
	"time"

	"github.com/aaraminds/transcript-agent/internal/domain"
	"github.com/aaraminds/transcript-agent/internal/providers/llm"
)

// Config for the Claude client. API key from secrets manager in production
// (PRD 16.6).
type Config struct {
	APIKey       string
	CleanupModel string // low/mid-cost model for strict transformation
	SummaryModel string // mid-tier model for summarization
}

// Provider is the Claude API client skeleton.
type Provider struct {
	cfg    Config
	client *http.Client
}

// New returns the provider.
func New(cfg Config) *Provider {
	return &Provider{cfg: cfg, client: &http.Client{Timeout: 120 * time.Second}}
}

// Configured reports whether an API key is present.
func (p *Provider) Configured() bool { return p.cfg.APIKey != "" }

// Cleanup would send segment texts as a numbered list with a system prompt
// that encodes PRD 15.2 verbatim (allowed edits: fillers, false starts,
// punctuation, capitalization; forbidden: paraphrase, reorder, add facts,
// remove sensitive statements). Transcript content is passed strictly as
// data, never as instructions (PRD 16.5 prompt-injection control).
// The response must return exactly one line per input segment; anything else
// maps to LLM_OUTPUT_INVALID so the retry policy in 14.8 applies.
func (p *Provider) Cleanup(ctx context.Context, segmentTexts []string, stylePolicyID string) ([]string, llm.CleanupStats, error) {
	if !p.Configured() {
		return nil, llm.CleanupStats{}, domain.E(domain.CodeNotConfigured,
			"anthropic provider not configured (set ANTHROPIC_API_KEY)")
	}
	_ = ctx
	return nil, llm.CleanupStats{}, domain.E(domain.CodeNotConfigured,
		"anthropic cleanup client is a skeleton; use MODE=mock")
}

// Summarize would prompt with the transcript as quoted data plus the
// job_config constraints (summary_max_words, summary_style) and require the
// model to use only transcript content. Timeouts map to LLM_PROVIDER_TIMEOUT.
func (p *Provider) Summarize(ctx context.Context, transcriptText string, maxWords int, style string) (string, error) {
	if !p.Configured() {
		return "", domain.E(domain.CodeNotConfigured,
			"anthropic provider not configured (set ANTHROPIC_API_KEY)")
	}
	_ = ctx
	return "", domain.E(domain.CodeNotConfigured,
		"anthropic summary client is a skeleton; use MODE=mock")
}

var _ llm.Provider = (*Provider)(nil)
