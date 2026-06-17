package render

import (
	"strings"
	"testing"
	"time"

	"github.com/aaraminds/copilot-token-budget/internal/pricing"
	"github.com/aaraminds/copilot-token-budget/internal/session"
)

const credit = int64(1_000_000_000)

// fakeConfig is a deterministic pricing config independent of any on-disk file.
func fakeConfig() pricing.Config {
	return pricing.Config{
		AllowanceCredits: 7000,
		Models: map[string]pricing.ModelRate{
			"sonnet": {InputPerMillion: 300, OutputPerMillion: 1500, ContextWindowTokens: 200000},
		},
		Default: pricing.ModelRate{InputPerMillion: 300, OutputPerMillion: 1500, ContextWindowTokens: 200000},
	}
}

// TestStatusline_AssemblesFromFakeSessions checks the full one-liner against a
// known session set, asserting model, today/month credits, percentage, burn and
// the context-% field — all on a fixed clock so the math is deterministic.
func TestStatusline_AssemblesFromFakeSessions(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC) // day 10 of the month
	today := time.Date(2026, 6, 10, 9, 0, 0, 0, time.UTC)
	earlier := time.Date(2026, 6, 3, 9, 0, 0, 0, time.UTC)

	sessions := []session.Session{
		// Active, newest → drives the model + context-% fields.
		{
			ID:           "active",
			PrimaryModel: "claude-sonnet-4.6",
			IsActive:     true,
			StartTime:    today,
			EndTime:      time.Time{}, // active → BillingTime falls back to StartTime (today)
			TotalNanoAIU: 100 * credit,
			Tokens:       session.TokenBreakdown{CurrentTokens: 100000}, // 50% of 200k window
			ModelMetrics: []session.ModelMetric{{Model: "claude-sonnet-4.6", NanoAIU: 100 * credit}},
		},
		// Earlier this month, finished → adds to month total but not today.
		{
			ID:           "older",
			PrimaryModel: "claude-opus-4.8",
			StartTime:    earlier,
			EndTime:      earlier.Add(time.Hour),
			IsFinal:      true,
			TotalNanoAIU: 200 * credit,
			ModelMetrics: []session.ModelMetric{{Model: "claude-opus-4.8", NanoAIU: 200 * credit}},
		},
	}

	got := Statusline(sessions, fakeConfig(), now, false)

	// Model of the newest active session, prefix-stripped by modelShort.
	if !strings.Contains(got, "🤖 sonnet-4.6") {
		t.Errorf("expected newest-active model 'sonnet-4.6' in line, got %q", got)
	}
	// Today = 100 credits (active session billed to today via StartTime).
	if !strings.Contains(got, "💰 100 today") {
		t.Errorf("expected '💰 100 today' in line, got %q", got)
	}
	// Month total = 300 credits / 7000 allowance.
	if !strings.Contains(got, "300/7000") {
		t.Errorf("expected month '300/7000' in line, got %q", got)
	}
	// Pct = 300/7000 ≈ 4% (rounded).
	if !strings.Contains(got, "(4%)") {
		t.Errorf("expected '(4%%)' in line, got %q", got)
	}
	// Burn = 300 / 10 days = 30/day.
	if !strings.Contains(got, "🔥 30/day") {
		t.Errorf("expected '🔥 30/day' in line, got %q", got)
	}
	// Context = 100000 / 200000 = 50%.
	if !strings.Contains(got, "🧠 50%") {
		t.Errorf("expected '🧠 50%%' in line, got %q", got)
	}
}

// TestStatusline_NoData yields a minimal, safe single line with no active model
// and no context field — never a panic, never a partial-format crash.
func TestStatusline_NoData(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	got := Statusline(nil, fakeConfig(), now, false)

	if strings.Contains(got, "🧠") {
		t.Errorf("no active session → context field should be omitted, got %q", got)
	}
	if !strings.Contains(got, "🤖 idle") {
		t.Errorf("no active session → model should be 'idle', got %q", got)
	}
	if !strings.Contains(got, "💰 0 today") {
		t.Errorf("no data → '0 today' expected, got %q", got)
	}
	if !strings.Contains(got, "0/7000") {
		t.Errorf("no data → month '0/7000' expected, got %q", got)
	}
	if strings.Count(got, "\n") != 0 {
		t.Errorf("statusline must be a single line, got %q", got)
	}
}

// TestStatusline_ColorWrapsPercentage verifies that color=true wraps the
// percentage in an ANSI sequence and color=false leaves it bare.
func TestStatusline_ColorWrapsPercentage(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	colored := Statusline(nil, fakeConfig(), now, true)
	plain := Statusline(nil, fakeConfig(), now, false)

	if !strings.Contains(colored, "\033[") {
		t.Errorf("color=true should emit an ANSI escape, got %q", colored)
	}
	if strings.Contains(plain, "\033[") {
		t.Errorf("color=false should emit no ANSI escape, got %q", plain)
	}
}
