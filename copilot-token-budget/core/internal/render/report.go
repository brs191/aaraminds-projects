// Package render provides the shared 4-section budget report renderer used by
// both cmd/analyze and cmd/dashboard. Keeping it here avoids duplicating
// formatting logic across the two binaries.
package render

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/aaraminds/copilot-token-budget/internal/analytics"
	"github.com/aaraminds/copilot-token-budget/internal/budget"
	"github.com/aaraminds/copilot-token-budget/internal/instructions"
	"github.com/aaraminds/copilot-token-budget/internal/livebilling"
	"github.com/aaraminds/copilot-token-budget/internal/pricing"
	"github.com/aaraminds/copilot-token-budget/internal/session"
)

// ANSI colour and style codes.
const (
	ansiReset  = "\033[0m"
	ansiBold   = "\033[1m"
	ansiDim    = "\033[2m"
	ansiRed    = "\033[31m"
	ansiYellow = "\033[33m"
	ansiGreen  = "\033[32m"
	ansiCyan   = "\033[36m"
)

const progressBarWidth = 40

// trendWindowDays is how many trailing days the USAGE TREND section shows.
const trendWindowDays = 14

// trendBarWidth is the max width of the tiny ASCII bar in the usage trend.
const trendBarWidth = 24

// RenderReport writes the full budget report to os.Stdout and returns the
// BudgetState computed from monthlySessions so callers can update the WezTerm
// badge. cfg supplies the allowance (so budget math is not hardcoded) and the
// per-model context windows used for the active-session context-% annotation.
//
// Sections, in order:
//  1. Active sessions (with context-window %)
//  2. Recent session history
//  3. Usage trend (last 14 days)
//  4. Top consumers (sessions / models / projects)
//  5. Monthly budget status
//  6. Instruction file audit
func RenderReport(
	allSessions []session.Session,
	monthlySessions []session.Session,
	instrFiles []instructions.InstructionFile,
	workspaceRoot string,
	cfg pricing.Config,
) budget.BudgetState {
	printSection1(allSessions, cfg)
	printSection2(allSessions)
	RenderUsageTrend(os.Stdout, allSessions)
	RenderTopConsumers(os.Stdout, allSessions)
	bs := printSection3(monthlySessions, cfg)
	printSection4(instrFiles, workspaceRoot)
	return bs
}

// ── Section 1: Active Sessions ────────────────────────────────────────────────

func printSection1(sessions []session.Session, cfg pricing.Config) {
	var active []session.Session
	for _, s := range sessions {
		if s.IsActive {
			active = append(active, s)
		}
	}

	fmt.Printf("\n%s%s▶  ACTIVE SESSIONS%s\n", ansiBold, ansiCyan, ansiReset)
	if len(active) == 0 {
		fmt.Printf("  %s(none)%s\n", ansiDim, ansiReset)
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "%sProject\tModel\tInput K\tOutput K\tCredits\tContext\tStatus%s\n", ansiBold, ansiReset)
	fmt.Fprintf(w, "%s%s%s\n", ansiDim, strings.Repeat("─", 80), ansiReset)
	for _, s := range active {
		credits := budget.FromNanoAIU(s.TotalNanoAIU)
		// Active sessions have not shut down, so their billing is a live
		// snapshot, not a settled total. Annotate it so partial/zero numbers
		// are never mistaken for final figures.
		creditsCell := fmt.Sprintf("%.2f", credits)
		if !s.IsFinal {
			creditsCell = fmt.Sprintf("%.2f%s (live)%s", credits, ansiYellow, ansiReset)
		}
		ctxPct := analytics.ContextWindowPct(s, cfg)
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\t%s%.0f%%%s\t%s● ACTIVE%s\n",
			projectName(s), modelShort(s.PrimaryModel),
			s.TotalInputTokens()/1000, s.TotalOutputTokens()/1000,
			creditsCell, ctxPctColor(ctxPct), ctxPct, ansiReset,
			ansiGreen, ansiReset,
		)
		ctxLine := fmt.Sprintf("%s  ↳ context: %d total | %d system (instructions) | %d conversation%s",
			ansiDim,
			s.Tokens.CurrentTokens, s.Tokens.SystemTokens, s.Tokens.ConversationTokens,
			ansiReset,
		)
		if !s.IsFinal {
			ctxLine += fmt.Sprintf("  %s↳ live — not yet finalized%s", ansiYellow, ansiReset)
		}
		fmt.Fprintf(w, "%s\n", ctxLine)
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "render: flush error (section 1): %v\n", err)
	}
}

// ── Section 2: Recent Session History ─────────────────────────────────────────

func printSection2(sessions []session.Session) {
	var history []session.Session
	for _, s := range sessions {
		if s.TotalNanoAIU > 0 {
			history = append(history, s)
		}
		if len(history) == 20 {
			break
		}
	}

	fmt.Printf("\n%s%s▶  RECENT SESSION HISTORY%s %s(last 20 with credit data)%s\n",
		ansiBold, ansiCyan, ansiReset, ansiDim, ansiReset)
	if len(history) == 0 {
		fmt.Printf("  %s(no completed sessions found)%s\n", ansiDim, ansiReset)
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "%sDate\tProject\tModel\tInput K\tOutput K\tCredits%s\n", ansiBold, ansiReset)
	fmt.Fprintf(w, "%s%s%s\n", ansiDim, strings.Repeat("─", 72), ansiReset)
	for _, s := range history {
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d\t%.2f\n",
			s.StartTime.Format("Jan 02 15:04"),
			projectName(s), modelShort(s.PrimaryModel),
			s.TotalInputTokens()/1000, s.TotalOutputTokens()/1000,
			budget.FromNanoAIU(s.TotalNanoAIU),
		)
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "render: flush error (section 2): %v\n", err)
	}
}

// ── Section 3: Monthly Budget Status ──────────────────────────────────────────

func printSection3(sessions []session.Session, cfg pricing.Config) budget.BudgetState {
	now := time.Now()
	fmt.Printf("\n%s%s▶  MONTHLY BUDGET STATUS — %s %d%s\n",
		ansiBold, ansiCyan, now.Month().String(), now.Year(), ansiReset)

	var nanoValues []int64
	var premiumRequests int64
	for _, s := range sessions {
		nanoValues = append(nanoValues, s.TotalNanoAIU)
		premiumRequests += s.TotalPremiumRequests
	}
	state := budget.Calculate(nanoValues, cfg.AllowanceCredits)
	color := colorForStatus(state.Status)
	label := liveBillingLabel(sessions)

	fmt.Printf("  Status:    %s%s%s%s\n", ansiBold, color, state.Status, ansiReset)
	fmt.Printf("  Used:      %s%.2f%s / %d credits  (%s$%.2f%s)\n",
		color, state.UsedCredits, ansiReset,
		state.AllowedCredits,
		ansiDim, budget.ToDollars(state.UsedCredits), ansiReset,
	)
	fmt.Printf("  Remaining: %.2f credits\n", state.RemainingCredit)
	fmt.Printf("  Usage:     %.1f%%\n", state.UsedPct)
	fmt.Printf("  Premium requests this month: %d\n\n", premiumRequests)
	fmt.Printf("  Live billing source: %s\n", label)

	filled := int(state.UsedPct / 100 * progressBarWidth)
	if filled > progressBarWidth {
		filled = progressBarWidth
	}
	if filled < 0 {
		filled = 0
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", progressBarWidth-filled)
	fmt.Printf("  %s[%s%s%s] %.1f%%\n  %s\n",
		ansiDim, ansiReset+color, bar, ansiReset+ansiDim, state.UsedPct, ansiReset)

	fmt.Printf("  %sAT&T Copilot Enterprise promo — 7,000 cr/month until 2026-09-01%s\n",
		ansiDim, ansiReset)

	return state
}

// ── Section 4: Instruction File Audit ─────────────────────────────────────────

func printSection4(files []instructions.InstructionFile, workspaceRoot string) {
	fmt.Printf("\n%s%s▶  INSTRUCTION FILE AUDIT%s\n", ansiBold, ansiCyan, ansiReset)
	if len(files) == 0 {
		fmt.Printf("  %s(no instruction files found under %s)%s\n",
			ansiDim, workspaceRoot, ansiReset)
		return
	}

	var wsRoot, projScoped []instructions.InstructionFile
	for _, f := range files {
		if f.Scope == "workspace-root" {
			wsRoot = append(wsRoot, f)
		} else {
			projScoped = append(projScoped, f)
		}
	}

	printInstructionGroup("Always loaded (workspace-root)", wsRoot, workspaceRoot)
	printInstructionGroup("Project-scoped", projScoped, workspaceRoot)

	summary := instructions.BuildOptimizationSummary(files)
	if summary.AlwaysLoadedTokens > 0 {
		cr, usd := budget.EstimateInstructionCostPerSession(summary.AlwaysLoadedTokens)
		fmt.Printf("\n  %sAlways-loaded overhead:%s ~%d tokens → %s%.2f cr / $%.2f%s per 50-turn session\n",
			ansiBold, ansiReset, summary.AlwaysLoadedTokens, ansiYellow, cr, usd, ansiReset,
		)
		if summary.ReducibleTokens > 0 {
			targetCr, targetUSD := budget.EstimateInstructionCostPerSession(summary.TargetTokens)
			fmt.Printf("  %sOptimization target:%s ~%d tokens → %.2f cr / $%.2f per session\n",
				ansiBold, ansiReset, summary.TargetTokens, targetCr, targetUSD,
			)
			fmt.Printf("  %sPotential savings:%s reduce ~%d tokens → save ~%.2f cr / $%.2f per session\n",
				ansiBold, ansiReset, summary.ReducibleTokens, cr-targetCr, usd-targetUSD,
			)

			fmt.Printf("\n  %sTop optimization opportunities:%s\n", ansiBold, ansiReset)
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "  %sFile\tScope\tCurrent\tTarget\tReduce\tAction%s\n", ansiBold, ansiReset)
			fmt.Fprintf(w, "  %s%s%s\n", ansiDim, strings.Repeat("─", 92), ansiReset)
			limit := 3
			if len(summary.Opportunities) < limit {
				limit = len(summary.Opportunities)
			}
			for i := 0; i < limit; i++ {
				o := summary.Opportunities[i]
				rel, err := filepath.Rel(workspaceRoot, o.Path)
				if err != nil {
					rel = o.Path
				}
				fmt.Fprintf(w, "  %s\t%s\t%d\t%d\t%s-%d%s\t%s\n",
					rel, o.Scope, o.CurrentTokens, o.TargetTokens,
					tokenColor(o.ReducibleTokens), o.ReducibleTokens, ansiReset,
					o.Recommendation,
				)
			}
			if err := w.Flush(); err != nil {
				fmt.Fprintf(os.Stderr, "render: flush error (section 4 optimization): %v\n", err)
			}
		}
	}
}

func printInstructionGroup(label string, files []instructions.InstructionFile, workspaceRoot string) {
	fmt.Printf("\n  %s%s:%s\n", ansiBold, label, ansiReset)
	if len(files) == 0 {
		fmt.Printf("    %s(none)%s\n", ansiDim, ansiReset)
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "  %sFile\t~Tokens\tRecommendation%s\n", ansiBold, ansiReset)
	fmt.Fprintf(w, "  %s%s%s\n", ansiDim, strings.Repeat("─", 60), ansiReset)
	for _, f := range files {
		rel, err := filepath.Rel(workspaceRoot, f.Path)
		if err != nil {
			rel = f.Path
		}
		fmt.Fprintf(w, "  %s\t%s%d%s\t%s\n",
			rel, tokenColor(f.EstimatedToks), f.EstimatedToks, ansiReset,
			f.SavingsRecommendation(),
		)
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "render: flush error (section 4): %v\n", err)
	}
}

// ── Usage Trend (last N days) ─────────────────────────────────────────────────

// RenderUsageTrend writes the trailing-window daily series with a tiny ASCII bar
// per day, flagging anomalous days (analytics.AnomalousDays) with a marker. It
// uses the full daily series for the anomaly distribution, then displays only the
// last trendWindowDays buckets so the section stays compact.
func RenderUsageTrend(out io.Writer, sessions []session.Session) {
	daily := analytics.DailySeries(sessions)

	fmt.Fprintf(out, "\n%s%s▶  USAGE TREND%s %s(last %d days)%s\n",
		ansiBold, ansiCyan, ansiReset, ansiDim, trendWindowDays, ansiReset)
	if len(daily) == 0 {
		fmt.Fprintf(out, "  %s(no usage data)%s\n", ansiDim, ansiReset)
		return
	}

	// Flag anomalies over the whole series, then key by bucket key for lookup.
	anomalous := make(map[string]bool)
	for _, b := range analytics.AnomalousDays(daily) {
		anomalous[b.Key] = true
	}

	window := lastBuckets(daily, trendWindowDays)
	maxCredits := 0.0
	for _, b := range window {
		if b.Credits > maxCredits {
			maxCredits = b.Credits
		}
	}

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "%sDate\tCredits\tTrend%s\n", ansiBold, ansiReset)
	fmt.Fprintf(w, "%s%s%s\n", ansiDim, strings.Repeat("─", 56), ansiReset)
	for _, b := range window {
		marker := ""
		if anomalous[b.Key] {
			marker = fmt.Sprintf("  %s⚠ anomaly%s", ansiRed, ansiReset)
		}
		fmt.Fprintf(w, "%s\t%.2f\t%s%s%s%s\n",
			b.Key, b.Credits,
			ansiCyan, dailyBar(b.Credits, maxCredits, trendBarWidth), ansiReset,
			marker,
		)
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "render: flush error (usage trend): %v\n", err)
	}
}

// ── Top Consumers ─────────────────────────────────────────────────────────────

// RenderTopConsumers writes the top-5 sessions, models, and projects by credits.
func RenderTopConsumers(out io.Writer, sessions []session.Session) {
	const topN = 5
	fmt.Fprintf(out, "\n%s%s▶  TOP CONSUMERS%s %s(top %d by credits)%s\n",
		ansiBold, ansiCyan, ansiReset, ansiDim, topN, ansiReset)

	printConsumerGroup(out, "Sessions", analytics.TopSessions(sessions, topN), true)
	topModels := analytics.TopModels(sessions, topN)
	printConsumerGroup(out, "Models", topModels, false)
	renderModelCacheReads(out, sessions, topModels)
	printConsumerGroup(out, "Projects", analytics.TopProjects(sessions, topN), false)
}

// renderModelCacheReads surfaces prompt-cache read tokens per model — data that
// is captured on ModelMetric but otherwise never shown. It prints one line per
// model in the supplied (already top-ranked) list that has any cache reads, and
// flags when cache reads dominate raw input for that model. Models with zero
// cache reads are skipped so the section stays compact; if none have cache reads
// nothing is printed.
func renderModelCacheReads(out io.Writer, sessions []session.Session, models []analytics.Consumer) {
	type tally struct{ cacheRead, input int64 }
	byModel := make(map[string]*tally)
	for _, s := range sessions {
		for _, m := range s.ModelMetrics {
			t := byModel[m.Model]
			if t == nil {
				t = &tally{}
				byModel[m.Model] = t
			}
			t.cacheRead += m.CacheReadTokens
			t.input += m.InputTokens
		}
	}

	var lines []string
	for _, m := range models {
		t := byModel[m.Name]
		if t == nil || t.cacheRead == 0 {
			continue
		}
		note := ""
		if t.cacheRead > t.input {
			note = fmt.Sprintf("  %scache reads dominate%s", ansiYellow, ansiReset)
		}
		lines = append(lines, fmt.Sprintf("    %s%s%s: %d cache-read tokens%s",
			ansiDim, modelShort(m.Name), ansiReset, t.cacheRead, note))
	}
	if len(lines) == 0 {
		return
	}
	fmt.Fprintf(out, "\n  %sCache reads (by model):%s\n", ansiBold, ansiReset)
	for _, ln := range lines {
		fmt.Fprintln(out, ln)
	}
}

// printConsumerGroup renders one ranked leaderboard. showModel adds a Model
// column (useful for the per-session list where the model is informative).
func printConsumerGroup(out io.Writer, label string, rows []analytics.Consumer, showModel bool) {
	fmt.Fprintf(out, "\n  %s%s:%s\n", ansiBold, label, ansiReset)
	if len(rows) == 0 {
		fmt.Fprintf(out, "    %s(none)%s\n", ansiDim, ansiReset)
		return
	}
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	if showModel {
		fmt.Fprintf(w, "  %s#\tName\tModel\tCredits%s\n", ansiBold, ansiReset)
	} else {
		fmt.Fprintf(w, "  %s#\tName\tCredits%s\n", ansiBold, ansiReset)
	}
	fmt.Fprintf(w, "  %s%s%s\n", ansiDim, strings.Repeat("─", 56), ansiReset)
	for i, r := range rows {
		if showModel {
			fmt.Fprintf(w, "  %d\t%s\t%s\t%.2f\n", i+1, r.Name, modelShort(r.Model), r.Credits)
		} else {
			fmt.Fprintf(w, "  %d\t%s\t%.2f\n", i+1, r.Name, r.Credits)
		}
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "render: flush error (top %s): %v\n", strings.ToLower(label), err)
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// lastBuckets returns the trailing n elements of b (all of them when n >= len or
// n <= 0). It never allocates a copy — it reslices the input.
func lastBuckets(b []analytics.Bucket, n int) []analytics.Bucket {
	if n <= 0 || n >= len(b) {
		return b
	}
	return b[len(b)-n:]
}

// dailyBar renders a proportional ASCII bar of width up to maxWidth for value
// relative to max. A zero or negative max yields an empty bar; any positive
// value rounds up to at least one block so non-zero days are always visible.
func dailyBar(value, max float64, maxWidth int) string {
	if maxWidth <= 0 || max <= 0 || value <= 0 {
		return ""
	}
	filled := int(value / max * float64(maxWidth))
	if filled < 1 {
		filled = 1 // any non-zero spend shows at least one block
	}
	if filled > maxWidth {
		filled = maxWidth
	}
	return strings.Repeat("█", filled)
}

// ctxPctColor returns an ANSI colour for a context-window fill percentage:
// green under 60%, yellow 60–90%, red above 90% — matching the budget thresholds.
func ctxPctColor(pct float64) string {
	switch {
	case pct > 90:
		return ansiRed
	case pct >= 60:
		return ansiYellow
	default:
		return ansiGreen
	}
}

func colorForStatus(s budget.BudgetStatus) string {
	switch s {
	case budget.StatusCritical:
		return ansiRed
	case budget.StatusWarning:
		return ansiYellow
	default:
		return ansiGreen
	}
}

func tokenColor(toks int64) string {
	switch {
	case toks >= 5000:
		return ansiRed
	case toks >= 2000:
		return ansiYellow
	default:
		return ansiGreen
	}
}

func projectName(s session.Session) string {
	if s.ProjectName != "" {
		return s.ProjectName
	}
	return ansiDim + "(unknown)" + ansiReset
}

func modelShort(model string) string {
	model = strings.TrimPrefix(model, "claude-")
	model = strings.TrimPrefix(model, "gpt-")
	// Truncate on rune boundaries so a multibyte UTF-8 character is never split.
	if r := []rune(model); len(r) > 16 {
		return string(r[:16])
	}
	return model
}

func liveBillingLabel(sessions []session.Session) string {
	return livebilling.DisplayLabel(latestOrgBillingSnapshot(sessions), time.Now())
}

func latestOrgBillingSnapshot(sessions []session.Session) *livebilling.OrgBillingSnapshot {
	var latest *livebilling.OrgBillingSnapshot
	for i := range sessions {
		snap := sessions[i].OrgBillingSnapshot
		if snap == nil {
			continue
		}
		if latest == nil || snap.LastRefreshedAt.After(latest.LastRefreshedAt) {
			latest = snap
		}
	}
	return latest
}
