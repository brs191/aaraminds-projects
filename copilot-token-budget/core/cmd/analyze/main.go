// cmd/analyze produces a one-shot credit budget report from local session data.
//
// Usage: go run ./cmd/analyze [flags] [workspace-root]
//
//	--json   print a machine-readable export.Report as JSON to stdout, nothing else
//	--csv    print per-session CSV to stdout, nothing else
//
// If workspace-root is omitted, os.Getwd() is used. The --json and --csv flags
// suppress the ANSI report entirely (machine-readable output only) so the stream
// is safe to pipe. Exit 0 = success, Exit 1 = fatal error.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/aaraminds/copilot-token-budget/internal/analytics"
	"github.com/aaraminds/copilot-token-budget/internal/budget"
	"github.com/aaraminds/copilot-token-budget/internal/cli"
	"github.com/aaraminds/copilot-token-budget/internal/export"
	"github.com/aaraminds/copilot-token-budget/internal/instructions"
	"github.com/aaraminds/copilot-token-budget/internal/livebilling"
	"github.com/aaraminds/copilot-token-budget/internal/pricing"
	"github.com/aaraminds/copilot-token-budget/internal/render"
	"github.com/aaraminds/copilot-token-budget/internal/session"
)

// Build-time version metadata, injected via -ldflags "-X main.version=...".
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	jsonOut := flag.Bool("json", false, "print export.Report as JSON to stdout and exit (suppresses the ANSI report)")
	csvOut := flag.Bool("csv", false, "print per-session CSV to stdout and exit (suppresses the ANSI report)")
	showVersion := flag.Bool("version", false, "print version information and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("copilot-analyze %s (commit %s, built %s)\n", version, commit, date)
		return
	}

	workspaceRoot, err := cli.ResolveWorkspaceRootFrom(flag.Args())
	if err != nil {
		cli.Fatalf("cannot resolve workspace root: %v", err)
	}

	cfg, err := pricing.Load()
	if err != nil {
		cli.Fatalf("cannot load pricing: %v", err)
	}

	allSessions, err := session.ReadAll()
	if err != nil {
		cli.Fatalf("cannot read sessions: %v", err)
	}

	// Refresh live billing data from cache or GitHub (Phase 8.7).
	// This is non-blocking; failures degrade to estimated mode.
	cfg2, _ := livebilling.Load()
	auth := livebilling.ResolveAuth(cfg2, nil)
	refresher := livebilling.NewRefresher(cfg2, auth)
	refresher.Refresh(context.Background())

	// Print per-source breakdown (Phase 6.2: multi-source reader)
	printSourceBreakdown(allSessions)

	// Machine-readable modes: emit only the requested format, then exit 0.
	if *jsonOut {
		data, err := export.ToJSON(buildReport(allSessions, cfg))
		if err != nil {
			cli.Fatalf("cannot encode JSON report: %v", err)
		}
		os.Stdout.Write(data)
		os.Stdout.Write([]byte("\n"))
		return
	}
	if *csvOut {
		if err := export.SessionsToCSV(os.Stdout, allSessions); err != nil {
			cli.Fatalf("cannot write CSV: %v", err)
		}
		return
	}

	instrFiles, err := instructions.ScanWorkspace(workspaceRoot)
	if err != nil {
		cli.Fatalf("cannot scan workspace: %v", err)
	}

	render.RenderReport(allSessions, cli.FilterThisMonth(allSessions), instrFiles, workspaceRoot, cfg)
}

// buildReport assembles a fully-populated export.Report from all sessions using
// the loaded pricing config for the budget allowance. The budget state is scoped
// to the current calendar month, matching the ANSI report's MONTHLY BUDGET section.
func buildReport(allSessions []session.Session, cfg pricing.Config) export.Report {
	monthly := cli.FilterThisMonth(allSessions)
	nano := make([]int64, 0, len(monthly))
	var premiumRequests int64
	for _, s := range monthly {
		nano = append(nano, s.TotalNanoAIU)
		premiumRequests += s.TotalPremiumRequests
	}

	const topN = 5
	return export.Report{
		GeneratedAt:        time.Now(),
		BudgetState:        budget.Calculate(nano, cfg.AllowanceCredits),
		PremiumRequests:    premiumRequests,
		OrgBillingSnapshot: export.LatestOrgBillingSnapshot(allSessions),
		Daily:              analytics.DailySeries(allSessions),
		TopSessions:        analytics.TopSessions(allSessions, topN),
		TopModels:          analytics.TopModels(allSessions, topN),
		TopProjects:        analytics.TopProjects(allSessions, topN),
		Sessions:           export.SessionViews(allSessions),
	}
}

// printSourceBreakdown displays per-source session and cost statistics.
// This provides visibility into which tools (CLI vs IDE) are generating usage.
// Phase 6: IDE costs are estimated (TokenCostSource="estimated"); Phase 7 will enrich with GitHub API.
func printSourceBreakdown(sessions []session.Session) {
	// Aggregate by source
	sourceStats := make(map[string]struct {
		count           int
		totalNanoAIU    int64
		tokenCostSource string
	})

	for _, s := range sessions {
		stats := sourceStats[s.Source]
		stats.count++
		stats.totalNanoAIU += s.TotalNanoAIU
		stats.tokenCostSource = s.TokenCostSource
		sourceStats[s.Source] = stats
	}

	// Print summary
	fmt.Printf("\n  Session Sources (Phase 6: IDE costs estimated)\n")
	fmt.Printf("  ──────────────────────────────────────────────\n")

	// Print each source
	sourceOrder := []string{"cli", "ide-chat", "ide-edit", "ide-agent"} // stable order
	for _, source := range sourceOrder {
		if stats, ok := sourceStats[source]; ok && stats.count > 0 {
			costLabel := ""
			if stats.tokenCostSource == "authoritative" {
				costLabel = fmt.Sprintf("%.2f cr (authoritative)", budget.FromNanoAIU(stats.totalNanoAIU))
			} else if stats.tokenCostSource == "estimated" {
				costLabel = fmt.Sprintf("%.2f cr (estimated - Phase 6)", budget.FromNanoAIU(stats.totalNanoAIU))
			} else {
				costLabel = "costs unavailable"
			}

			sourceLabel := source
			switch source {
			case "cli":
				sourceLabel = "CLI"
			case "ide-chat":
				sourceLabel = "IDE Chat"
			case "ide-edit":
				sourceLabel = "IDE Edit"
			case "ide-agent":
				sourceLabel = "IDE Agent"
			}

			fmt.Printf("  %s: %d sessions (%s)\n", sourceLabel, stats.count, costLabel)
		}
	}

	fmt.Printf("  ──────────────────────────────────────────────\n")
	totalSessions := 0
	for _, stats := range sourceStats {
		totalSessions += stats.count
	}
	fmt.Printf("  Total: %d sessions\n", totalSessions)
}
