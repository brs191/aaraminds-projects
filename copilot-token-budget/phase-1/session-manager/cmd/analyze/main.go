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
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/aaraminds/copilot-session-manager/internal/analytics"
	"github.com/aaraminds/copilot-session-manager/internal/budget"
	"github.com/aaraminds/copilot-session-manager/internal/cli"
	"github.com/aaraminds/copilot-session-manager/internal/export"
	"github.com/aaraminds/copilot-session-manager/internal/instructions"
	"github.com/aaraminds/copilot-session-manager/internal/pricing"
	"github.com/aaraminds/copilot-session-manager/internal/render"
	"github.com/aaraminds/copilot-session-manager/internal/session"
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
	for _, s := range monthly {
		nano = append(nano, s.TotalNanoAIU)
	}

	const topN = 5
	return export.Report{
		GeneratedAt: time.Now(),
		BudgetState: budget.Calculate(nano, cfg.AllowanceCredits),
		Daily:       analytics.DailySeries(allSessions),
		TopSessions: analytics.TopSessions(allSessions, topN),
		TopModels:   analytics.TopModels(allSessions, topN),
		TopProjects: analytics.TopProjects(allSessions, topN),
		Sessions:    export.SessionViews(allSessions),
	}
}
