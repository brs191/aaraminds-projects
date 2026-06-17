// Command alert checks the current Copilot credit budget and posts a Microsoft Teams
// Adaptive Card alert when the WARNING (60%) or CRITICAL (90%) threshold is crossed.
//
// Usage:
//
//	COPILOT_BUDGET_TEAMS_WEBHOOK=<url> alert [--dry-run] [--allowance N] <workspace-root>
//
// Exit codes:
//
//	0 = no alert needed (budget OK, or threshold already fired today)
//	1 = alert fired (or --dry-run printed card JSON)
//	2 = error
//
// The webhook URL is read from the COPILOT_BUDGET_TEAMS_WEBHOOK environment variable —
// never a CLI flag (CLI flags are visible in ps aux).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/aaraminds/copilot-token-budget/alerting/internal/alerts"
	forecastpkg "github.com/aaraminds/copilot-token-budget/alerting/internal/forecast"
	"github.com/aaraminds/copilot-token-budget/internal/budget"
	sessionsrc "github.com/aaraminds/copilot-token-budget/internal/session"
)

// Build-time version metadata, injected via -ldflags "-X main.version=...".
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "print card JSON to stdout instead of posting")
	allowance := flag.Int("allowance", 7000, "monthly credit allowance")
	showVersion := flag.Bool("version", false, "print version information and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("copilot-alert %s (commit %s, built %s)\n", version, commit, date)
		return
	}

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: alert [--dry-run] [--allowance N] <workspace-root>")
		os.Exit(2)
	}

	// Webhook URL from env var — NEVER from a flag (visible in ps aux).
	webhookURL := os.Getenv("COPILOT_BUDGET_TEAMS_WEBHOOK")
	if !*dryRun && webhookURL == "" {
		fmt.Fprintln(os.Stderr, "error: COPILOT_BUDGET_TEAMS_WEBHOOK environment variable is not set")
		os.Exit(2)
	}

	// Read this month's sessions.
	sessions, err := sessionsrc.ReadThisMonth()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: read sessions: %v\n", err)
		os.Exit(2)
	}

	// Collect nanoAIU values and compute budget state.
	nanoAIUs := make([]int64, 0, len(sessions))
	for _, s := range sessions {
		if s.TotalNanoAIU > 0 {
			nanoAIUs = append(nanoAIUs, s.TotalNanoAIU)
		}
	}
	state := budget.Calculate(nanoAIUs, *allowance)

	// Determine active alert threshold.
	var activeThreshold int
	switch state.Status {
	case budget.StatusCritical:
		activeThreshold = 90
	case budget.StatusWarning:
		activeThreshold = 60
	default:
		// Budget is OK — no alert needed.
		os.Exit(0)
	}

	// Deduplication: skip if this threshold already fired today.
	should, err := alerts.ShouldAlert(activeThreshold)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: check alert dedup: %v\n", err)
		os.Exit(2)
	}
	if !should {
		os.Exit(0) // already alerted today — silent skip
	}

	// Build month-end forecast. Use UTC throughout so daysElapsed and lastDay
	// share a timezone with the analytics buckets (which bucket in UTC); mixing a
	// local daysElapsed with a UTC lastDay skews daysRemaining near month boundaries.
	today := time.Now().UTC()
	daysElapsed := today.Day()
	// time.Date with day=0 returns the last day of the previous month,
	// so month+1, day=0 gives the last day of the current month.
	lastDay := time.Date(today.Year(), today.Month()+1, 0, 0, 0, 0, 0, time.UTC)
	daysRemaining := lastDay.Day() - daysElapsed

	dailyBurn := forecastpkg.DailyBurnRate(sessions, daysElapsed)
	// Projected month-end TOTAL (credits already used + projected remaining burn).
	// On the last day daysRemaining == 0, so this collapses to the current used total
	// rather than vanishing to zero.
	projectedMonthTotal := forecastpkg.ProjectedMonthEndTotal(state.UsedCredits, dailyBurn, daysRemaining)

	// Compute average cost per token across all sessions for model recommendations.
	var totalNano int64
	var totalTokens int64
	for _, s := range sessions {
		totalNano += s.TotalNanoAIU
		totalTokens += s.TotalInputTokens() + s.TotalOutputTokens()
	}
	var avgCostPerToken float64
	if totalTokens > 0 {
		avgCostPerToken = budget.FromNanoAIU(totalNano) / float64(totalTokens)
	}
	recommendations := forecastpkg.ModelRoutingRecommendation(sessions, avgCostPerToken)

	// Build the Adaptive Card.
	card := alerts.NewBudgetCard(state, sessions, projectedMonthTotal, recommendations)

	// Dry-run: print JSON and exit 1 (alert would have fired).
	if *dryRun {
		payload, err := json.MarshalIndent(card, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: marshal card: %v\n", err)
			os.Exit(2)
		}
		fmt.Println(string(payload))
		os.Exit(1)
	}

	// Post to Teams. webhookURL is NEVER printed — error messages carry no URL.
	if err := alerts.PostAdaptiveCard(context.Background(), webhookURL, card); err != nil {
		fmt.Fprintf(os.Stderr, "error: post alert: %v\n", err)
		os.Exit(2)
	}

	// Record that this threshold fired today (non-fatal if it fails).
	if err := alerts.MarkAlerted(activeThreshold); err != nil {
		fmt.Fprintf(os.Stderr, "warning: mark alerted: %v\n", err)
	}

	os.Exit(1) // exit 1 = alert fired
}
