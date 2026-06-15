// cmd/statusline prints a single ccusage-style status line to stdout and exits,
// designed to be embedded in a shell prompt or a WezTerm right-status command.
//
// Output (one line, Copilot credits not dollars):
//
//	🤖 {model} | 💰 {today} today / {month}/{allowance} ({pct}%) | 🔥 {burn}/day | 🧠 {ctx}%
//
// It is one-shot (no ticker, no network) and never panics: any read error or
// empty data set still produces a minimal safe line and exits 0, because a
// status line that aborts would break the host prompt. Colour honours NO_COLOR.
package main

import (
	"fmt"
	"time"

	"github.com/aaraminds/copilot-session-manager/internal/pricing"
	"github.com/aaraminds/copilot-session-manager/internal/render"
	"github.com/aaraminds/copilot-session-manager/internal/session"
)

func main() {
	// Pricing: fall back to bundled defaults on any error so the line still renders.
	cfg, err := pricing.Load()
	if err != nil {
		cfg = pricing.Default()
	}

	// Sessions: on a read error, render from an empty set (minimal safe line).
	sessions, err := session.ReadAll()
	if err != nil {
		sessions = nil
	}

	line := render.Statusline(sessions, cfg, time.Now(), render.ColorEnabled())
	fmt.Println(line)
}
