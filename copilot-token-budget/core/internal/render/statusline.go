package render

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aaraminds/copilot-token-budget/internal/analytics"
	"github.com/aaraminds/copilot-token-budget/internal/budget"
	"github.com/aaraminds/copilot-token-budget/internal/pricing"
	"github.com/aaraminds/copilot-token-budget/internal/session"
)

// Statusline is the ccusage-style one-liner used by cmd/statusline. It is a pure
// function of its inputs (no clock, no file system) so it is deterministic and
// testable: pass time.Now() from the caller.
//
// Format (Copilot credits, not dollars):
//
//	🤖 {model} | 💰 {todayCr} today / {monthCr}/{allowance} ({pct}%) | 🔥 {burn}/day | 🧠 {ctx}%
//
// Where:
//   - model      = primary model of the newest active session (blank → "idle")
//   - todayCr    = credits in today's daily bucket (by BillingTime)
//   - monthCr    = credits across the current calendar month
//   - allowance  = cfg.AllowanceCredits
//   - pct        = budget used percentage for the month
//   - burn       = month credits / days elapsed in the month (daily burn rate)
//   - ctx        = context-window % of the newest active session (omitted if none)
//
// color controls ANSI colouring of the budget percentage; callers should pass
// false when NO_COLOR is set. Statusline never panics and degrades gracefully
// when there is no data (empty session slice yields a minimal safe line).
func Statusline(sessions []session.Session, cfg pricing.Config, now time.Time, color bool) string {
	newest := newestActive(sessions)

	model := "idle"
	ctxField := ""
	if newest != nil {
		if m := modelShort(newest.PrimaryModel); m != "" {
			model = m
		}
		ctxField = fmt.Sprintf(" | 🧠 %.0f%%", analytics.ContextWindowPct(*newest, cfg))
	}

	// Today's credits from the daily series (BillingTime-bucketed).
	todayKey := now.Format("2006-01-02")
	var todayCr float64
	for _, b := range analytics.DailySeries(sessions) {
		if b.Key == todayKey {
			todayCr = b.Credits
			break
		}
	}

	// Month total + budget state.
	monthly := filterMonth(sessions, now)
	nano := make([]int64, 0, len(monthly))
	for _, s := range monthly {
		nano = append(nano, s.TotalNanoAIU)
	}
	state := budget.Calculate(nano, cfg.AllowanceCredits)

	// Daily burn = month credits / days elapsed (clamped to >= 1 day).
	daysElapsed := now.Day()
	if daysElapsed < 1 {
		daysElapsed = 1
	}
	burn := state.UsedCredits / float64(daysElapsed)

	pctField := fmt.Sprintf("%.0f%%", state.UsedPct)
	if color {
		pctField = statuslineColor(state.Status) + pctField + ansiReset
	}

	return fmt.Sprintf("🤖 %s | 💰 %.0f today / %.0f/%d (%s) | 🔥 %.0f/day%s",
		model, todayCr, state.UsedCredits, state.AllowedCredits, pctField, burn, ctxField,
	)
}

// newestActive returns a pointer to the newest active session by StartTime, or
// nil when there are no active sessions. ReadAll already sorts newest-first, but
// this scans defensively so any input order is handled.
func newestActive(sessions []session.Session) *session.Session {
	var best *session.Session
	for i := range sessions {
		s := &sessions[i]
		if !s.IsActive {
			continue
		}
		if best == nil || s.StartTime.After(best.StartTime) {
			best = s
		}
	}
	return best
}

// filterMonth returns sessions whose BillingTime falls in now's calendar month.
// Both sides are normalized to UTC to match the analytics daily/monthly buckets
// (which bucket BillingTime in UTC), so a session near a month boundary is
// attributed to the same month the buckets use regardless of the host timezone.
func filterMonth(sessions []session.Session, now time.Time) []session.Session {
	nowUTC := now.UTC()
	var out []session.Session
	for _, s := range sessions {
		bt := s.BillingTime().UTC()
		if bt.Year() == nowUTC.Year() && bt.Month() == nowUTC.Month() {
			out = append(out, s)
		}
	}
	return out
}

// statuslineColor maps a budget status to an ANSI colour for the percentage cell.
func statuslineColor(s budget.BudgetStatus) string {
	switch s {
	case budget.StatusCritical:
		return ansiRed
	case budget.StatusWarning:
		return ansiYellow
	default:
		return ansiGreen
	}
}

// ColorEnabled reports whether ANSI colour should be emitted, honouring the
// NO_COLOR convention (https://no-color.org): any non-empty NO_COLOR disables it.
func ColorEnabled() bool {
	return strings.TrimSpace(os.Getenv("NO_COLOR")) == ""
}
