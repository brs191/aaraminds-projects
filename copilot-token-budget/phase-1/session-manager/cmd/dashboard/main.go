// cmd/dashboard is a live-refreshing terminal dashboard for the Copilot credit budget.
//
// Usage: go run ./cmd/dashboard [workspace-root]
// Refreshes every 10 seconds. Press Ctrl+C or send SIGTERM to exit cleanly.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aaraminds/copilot-session-manager/internal/cli"
	"github.com/aaraminds/copilot-session-manager/internal/instructions"
	"github.com/aaraminds/copilot-session-manager/internal/pricing"
	"github.com/aaraminds/copilot-session-manager/internal/render"
	"github.com/aaraminds/copilot-session-manager/internal/session"
	"github.com/aaraminds/copilot-session-manager/internal/wezterm"
)

const clearScreen = "\033[2J\033[H"

func main() {
	workspaceRoot, err := cli.ResolveWorkspaceRoot()
	if err != nil {
		cli.Fatalf("cannot resolve workspace root: %v", err)
	}

	// Pricing is stable for the life of the process; load it once. The full
	// report (usage trend, top consumers, context-% on active sessions) is
	// rendered by render.RenderReport, so the dashboard surfaces the new
	// sections automatically by passing cfg.
	cfg, err := pricing.Load()
	if err != nil {
		cli.Fatalf("cannot load pricing: %v", err)
	}

	// Signal channel — buffer 1 so the sender is never blocked.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Render one cycle: read data, clear screen, render, update badge.
	tick := func() {
		all, err := session.ReadAll()
		if err != nil {
			// Log but keep the dashboard running — transient read errors are expected.
			fmt.Fprintf(os.Stderr, "dashboard: read error: %v\n", err)
			return
		}
		instr, err := instructions.ScanWorkspace(workspaceRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "dashboard: scan error: %v\n", err)
		}

		fmt.Print(clearScreen)
		bs := render.RenderReport(all, cli.FilterThisMonth(all), instr, workspaceRoot, cfg)

		wezterm.SetBadge(wezterm.BudgetBadgeText(
			bs.UsedCredits,
			bs.AllowedCredits,
			string(bs.Status),
		))
	}

	// Render immediately on start, then every 10 seconds.
	tick()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tick()
		case <-sigCh:
			fmt.Println()
			wezterm.SetBadge("") // restore tab title to default
			os.Exit(0)
		}
	}
}
