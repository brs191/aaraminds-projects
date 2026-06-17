package alerts

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/aaraminds/copilot-token-budget/internal/budget"
	"github.com/aaraminds/copilot-token-budget/internal/session"
)

func TestProgressBar(t *testing.T) {
	tests := []struct {
		name  string
		pct   float64
		width int
		want  string
	}{
		{"0%", 0, 4, "░░░░"},
		{"25%", 25, 4, "█░░░"},
		{"50%", 50, 4, "██░░"},
		{"75%", 75, 4, "███░"},
		{"100%", 100, 4, "████"},
		{"over 100%", 120, 4, "████"},
		{"negative clamped", -10, 4, "░░░░"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := progressBar(tc.pct, tc.width)
			if got != tc.want {
				t.Errorf("progressBar(%.0f%%, %d) = %q, want %q", tc.pct, tc.width, got, tc.want)
			}
		})
	}
}

func TestNewBudgetCardStructure(t *testing.T) {
	state := budget.BudgetState{
		UsedCredits:     5000,
		AllowedCredits:  7000,
		UsedPct:         71.4,
		RemainingCredit: 2000,
		Status:          budget.StatusWarning,
	}

	// Third arg is the projected month-end TOTAL.
	card := NewBudgetCard(state, nil, 6500, nil)

	if card.Type != "message" {
		t.Errorf("card.Type = %q, want %q", card.Type, "message")
	}
	if len(card.Attachments) != 1 {
		t.Fatalf("len(attachments) = %d, want 1", len(card.Attachments))
	}
	att := card.Attachments[0]
	if att.ContentType != "application/vnd.microsoft.card.adaptive" {
		t.Errorf("ContentType = %q", att.ContentType)
	}
	if att.Content.Type != "AdaptiveCard" {
		t.Errorf("Content.Type = %q, want AdaptiveCard", att.Content.Type)
	}
	if att.Content.Version != "1.4" {
		t.Errorf("Content.Version = %q, want 1.4", att.Content.Version)
	}
	if len(att.Content.Body) == 0 {
		t.Error("card body is empty")
	}

	// Card must marshal to valid JSON without error.
	_, err := json.Marshal(card)
	if err != nil {
		t.Fatalf("json.Marshal(card) error: %v", err)
	}
}

func TestNewBudgetCardWithSessions(t *testing.T) {
	state := budget.BudgetState{
		UsedCredits:     8000,
		AllowedCredits:  7000,
		UsedPct:         114.3,
		RemainingCredit: -1000,
		Status:          budget.StatusCritical,
	}
	sessions := []session.Session{
		{ProjectName: "project-a", TotalNanoAIU: 3_000_000_000, PrimaryModel: "claude-sonnet"},
		{ProjectName: "project-b", TotalNanoAIU: 2_000_000_000, PrimaryModel: "claude-opus"},
		{ProjectName: "project-c", TotalNanoAIU: 1_000_000_000, PrimaryModel: "claude-haiku"},
		{ProjectName: "project-d", TotalNanoAIU: 500_000_000, PrimaryModel: "claude-haiku"},
	}
	recs := []string{"claude-opus → consider claude-haiku"}

	// Projected month-end total of 9500 cr (over the 7000 allowance).
	card := NewBudgetCard(state, sessions, 9500, recs)
	payload, err := json.Marshal(card)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	// Verify sessions appear in payload; project-d (4th) should be excluded (top 3 only).
	s := string(payload)
	if !strings.Contains(s, "project-a") {
		t.Error("expected project-a in card")
	}
	if !strings.Contains(s, "project-b") {
		t.Error("expected project-b in card")
	}
	if strings.Contains(s, "project-d") {
		t.Error("project-d (4th) should be excluded from top-3")
	}
	// The projected month-end total must surface as a positive total.
	if !strings.Contains(s, "Projected Month-End Total") {
		t.Error("expected projected month-end total block in card")
	}
	if !strings.Contains(s, "9500 cr") {
		t.Error("expected projected total of 9500 cr in card")
	}
}

// TestNewBudgetCardForecastShowsOnPositiveTotal proves a positive projected total is
// always shown, even when daysRemaining would have been 0 under the old remaining-only
// model (the projected total equals current used credits on the last day).
func TestNewBudgetCardForecastShowsOnPositiveTotal(t *testing.T) {
	state := budget.BudgetState{
		UsedCredits:     6800,
		AllowedCredits:  7000,
		UsedPct:         97.1,
		RemainingCredit: 200,
		Status:          budget.StatusCritical,
	}
	// Simulating the last day of the month: projected total == used credits.
	card := NewBudgetCard(state, nil, 6800, nil)
	payload, err := json.Marshal(card)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if !strings.Contains(string(payload), "Projected Month-End Total") {
		t.Error("forecast block must show on the last day when projected total is positive")
	}
}

func TestStatusColor(t *testing.T) {
	tests := []struct {
		status budget.BudgetStatus
		want   string
	}{
		{budget.StatusCritical, "Attention"},
		{budget.StatusWarning, "Warning"},
		{budget.StatusOK, "Good"},
	}
	for _, tc := range tests {
		got := statusColor(tc.status)
		if got != tc.want {
			t.Errorf("statusColor(%v) = %q, want %q", tc.status, got, tc.want)
		}
	}
}
