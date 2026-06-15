package cli

import (
	"testing"
	"time"

	"github.com/aaraminds/copilot-session-manager/internal/session"
)

// TestFilterThisMonth_BillingTimeScoping covers BUG 2: sessions are attributed
// to the calendar month of their END (billing) time, not their start time.
func TestFilterThisMonth_BillingTimeScoping(t *testing.T) {
	now := time.Now()
	thisMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	prevMonth := thisMonthStart.AddDate(0, 0, -1) // last day of previous month

	sessions := []session.Session{
		// Starts in the previous month, ends this month → INCLUDED.
		{
			ID:        "spans-into-this-month",
			StartTime: prevMonth.Add(-2 * time.Hour),
			EndTime:   thisMonthStart.Add(2 * time.Hour),
		},
		// Starts this month, ended last month (degenerate but tests the rule) → EXCLUDED.
		{
			ID:        "ended-last-month",
			StartTime: thisMonthStart.Add(2 * time.Hour),
			EndTime:   prevMonth.Add(-2 * time.Hour),
		},
		// Active session this month, no EndTime → falls back to StartTime → INCLUDED.
		{
			ID:        "active-this-month",
			StartTime: thisMonthStart.Add(3 * time.Hour),
		},
	}

	got := FilterThisMonth(sessions)

	included := map[string]bool{}
	for _, s := range got {
		included[s.ID] = true
	}

	if !included["spans-into-this-month"] {
		t.Error("session starting last month but ending this month should be INCLUDED")
	}
	if included["ended-last-month"] {
		t.Error("session that ended last month should be EXCLUDED")
	}
	if !included["active-this-month"] {
		t.Error("active session started this month (no EndTime) should be INCLUDED")
	}
}
