package export

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/aaraminds/copilot-token-budget/internal/analytics"
	"github.com/aaraminds/copilot-token-budget/internal/budget"
	"github.com/aaraminds/copilot-token-budget/internal/livebilling"
	"github.com/aaraminds/copilot-token-budget/internal/session"
)

const credit = int64(1_000_000_000)

func sampleSessions() []session.Session {
	d := time.Date(2026, 6, 10, 9, 0, 0, 0, time.UTC)
	return []session.Session{
		{
			ID:                   "id1",
			Source:               "copilot-cli",
			ProjectName:          "alpha",
			PrimaryModel:         "claude-sonnet-4.6",
			StartTime:            d,
			EndTime:              d.Add(time.Hour),
			IsFinal:              true,
			TotalNanoAIU:         5 * credit,
			TotalPremiumRequests: 7,
			Tokens:               session.TokenBreakdown{SystemTokens: 1200, CurrentTokens: 30000},
			ModelMetrics: []session.ModelMetric{
				{Model: "claude-sonnet-4.6", InputTokens: 1000, OutputTokens: 100, NanoAIU: 5 * credit,
					CacheReadTokens: 9000, CacheWriteTokens: 300, ReasoningTokens: 42},
			},
			OrgBillingSnapshot: &livebilling.OrgBillingSnapshot{
				OrgSlug:         "att-enterprise",
				Scope:           livebilling.ScopeOrgAggregate,
				SourceLabel:     "org aggregate, ~24h ago",
				Availability:    livebilling.AvailabilityAvailable,
				LastRefreshedAt: time.Date(2026, 6, 17, 10, 0, 0, 0, time.UTC),
				AsOf:            time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC),
				Credits:         123.45,
			},
		},
		{
			ID:           "id2",
			Source:       "copilot-cli",
			ProjectName:  "beta",
			PrimaryModel: "claude-opus-4.8",
			StartTime:    d.Add(2 * time.Hour),
			IsActive:     true,
			IsFinal:      false,
			TotalNanoAIU: 2 * credit,
			Tokens:       session.TokenBreakdown{SystemTokens: 800, CurrentTokens: 15000},
			ModelMetrics: []session.ModelMetric{
				{Model: "claude-opus-4.8", InputTokens: 500, OutputTokens: 50, NanoAIU: 2 * credit},
			},
		},
	}
}

func TestToJSON_DeterministicAndShape(t *testing.T) {
	sessions := sampleSessions()
	r := Report{
		GeneratedAt:     time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC),
		BudgetState:     budget.Calculate([]int64{5 * credit, 2 * credit}, 0),
		PremiumRequests: 7,
		OrgBillingSnapshot: &livebilling.OrgBillingSnapshot{
			OrgSlug:         "att-enterprise",
			Scope:           livebilling.ScopeOrgAggregate,
			SourceLabel:     "org aggregate, ~24h ago",
			Availability:    livebilling.AvailabilityAvailable,
			LastRefreshedAt: time.Date(2026, 6, 17, 10, 0, 0, 0, time.UTC),
			AsOf:            time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC),
			Credits:         123.45,
		},
		Daily:       analytics.DailySeries(sessions),
		TopSessions: analytics.TopSessions(sessions, 5),
		TopModels:   analytics.TopModels(sessions, 5),
		TopProjects: analytics.TopProjects(sessions, 5),
		Sessions:    SessionViews(sessions),
	}

	a, err := ToJSON(r)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	b, err := ToJSON(r)
	if err != nil {
		t.Fatalf("ToJSON (2nd): %v", err)
	}
	if !bytes.Equal(a, b) {
		t.Error("ToJSON is not deterministic across calls")
	}

	// Round-trip into a generic map and check key presence + a value.
	var m map[string]any
	if err := json.Unmarshal(a, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, k := range []string{"generatedAt", "budgetState", "premiumRequests", "daily", "topSessions", "topModels", "topProjects", "sessions"} {
		if _, ok := m[k]; !ok {
			t.Errorf("missing key %q in JSON", k)
		}
	}
	if _, ok := m["orgBillingSnapshot"]; !ok {
		t.Fatal("missing orgBillingSnapshot in JSON")
	}

	// budgetState must serialize in camelCase to match the TS extension's
	// BudgetState wire shape (remainingCredits is plural). Locking this guards
	// against the json tags being dropped, which would re-emit PascalCase.
	bs, ok := m["budgetState"].(map[string]any)
	if !ok {
		t.Fatalf("budgetState is not an object: %T", m["budgetState"])
	}
	for _, k := range []string{"usedCredits", "allowedCredits", "usedPct", "remainingCredits", "status"} {
		if _, ok := bs[k]; !ok {
			t.Errorf("missing budgetState key %q (camelCase contract)", k)
		}
	}
	for _, k := range []string{"UsedCredits", "AllowedCredits", "UsedPct", "RemainingCredit", "RemainingCredits", "Status"} {
		if _, ok := bs[k]; ok {
			t.Errorf("budgetState has PascalCase key %q; want camelCase only", k)
		}
	}
	views := m["sessions"].([]any)
	if len(views) != 2 {
		t.Fatalf("sessions len = %d, want 2", len(views))
	}
	first := views[0].(map[string]any)
	if first["source"] != "copilot-cli" {
		t.Errorf("session source = %v, want copilot-cli", first["source"])
	}
	if first["billingDate"] != "2026-06-10" {
		t.Errorf("billingDate = %v, want 2026-06-10", first["billingDate"])
	}
	if _, ok := first["orgBillingSnapshot"].(map[string]any); !ok {
		t.Fatal("session missing orgBillingSnapshot object")
	}
	// The new cache/reasoning/premium fields must be present in camelCase and
	// carry the alpha session's aggregated values (JSON numbers decode to float64).
	for k, want := range map[string]float64{
		"cacheReadTokens":  9000,
		"cacheWriteTokens": 300,
		"reasoningTokens":  42,
		"premiumRequests":  7,
	} {
		got, ok := first[k].(float64)
		if !ok {
			t.Errorf("session missing %q (camelCase contract)", k)
			continue
		}
		if got != want {
			t.Errorf("session %q = %v, want %v", k, got, want)
		}
	}
}

func TestSessionsToCSV(t *testing.T) {
	var buf bytes.Buffer
	if err := SessionsToCSV(&buf, sampleSessions()); err != nil {
		t.Fatalf("SessionsToCSV: %v", err)
	}

	rows, err := csv.NewReader(strings.NewReader(buf.String())).ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want 3 (header + 2)", len(rows))
	}
	wantHeader := []string{
		"date", "project", "model", "source", "credits", "inputTokens", "outputTokens", "systemTokens",
		"cacheReadTokens", "cacheWriteTokens", "reasoningTokens", "premiumRequests", "isActive", "isFinal",
	}
	for i, h := range wantHeader {
		if rows[0][i] != h {
			t.Errorf("header[%d] = %q, want %q", i, rows[0][i], h)
		}
	}
	// First data row spot checks.
	got := rows[1]
	if got[0] != "2026-06-10" || got[1] != "alpha" || got[3] != "copilot-cli" {
		t.Errorf("row1 = %v, unexpected date/project/source", got)
	}
	if got[4] != "5" {
		t.Errorf("row1 credits = %q, want 5", got[4])
	}
	if got[7] != "1200" {
		t.Errorf("row1 systemTokens = %q, want 1200", got[7])
	}
	if got[8] != "9000" || got[9] != "300" || got[10] != "42" {
		t.Errorf("row1 cacheRead/cacheWrite/reasoning = %q/%q/%q, want 9000/300/42", got[8], got[9], got[10])
	}
	if got[11] != "7" {
		t.Errorf("row1 premiumRequests = %q, want 7", got[11])
	}
	if got[12] != "false" || got[13] != "true" {
		t.Errorf("row1 isActive/isFinal = %q/%q, want false/true", got[12], got[13])
	}
}

func TestDailyToCSV(t *testing.T) {
	var buf bytes.Buffer
	daily := analytics.DailySeries(sampleSessions())
	if err := DailyToCSV(&buf, daily); err != nil {
		t.Fatalf("DailyToCSV: %v", err)
	}
	rows, err := csv.NewReader(strings.NewReader(buf.String())).ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	wantHeader := []string{"date", "sessions", "credits", "inputTokens", "outputTokens"}
	for i, h := range wantHeader {
		if rows[0][i] != h {
			t.Errorf("header[%d] = %q, want %q", i, rows[0][i], h)
		}
	}
	// Both sample sessions are on the same day -> one data row.
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2 (header + 1)", len(rows))
	}
	if rows[1][0] != "2026-06-10" || rows[1][1] != "2" {
		t.Errorf("daily row = %v, want date 2026-06-10 sessions 2", rows[1])
	}
	if rows[1][2] != "7" {
		t.Errorf("daily credits = %q, want 7", rows[1][2])
	}
}
