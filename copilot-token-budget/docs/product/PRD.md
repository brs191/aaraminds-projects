# Copilot Token Budget — Product Requirements

**Project:** `copilot-token-budget`
**Status:** Active
**Last reviewed:** 2026-06-13

---

## Problem statement

AT&T engineers running GitHub Copilot CLI across multiple projects exhaust their 7,000-credit/month
allowance faster than expected. They have no real-time visibility into:

1. **How many credits they've used** this month
2. **Which sessions are burning the most** credit
3. **How much instruction file overhead** is adding to every single message
4. **Whether they'll hit the cap** before month-end

The result: engineers hit the wall mid-sprint with no warning, blocking AI-assisted development.

---

## North-star goal

> An AT&T engineer opens VS Code and sees their exact credit usage in the status bar, gets a
> Teams alert before they go over budget, and can ask Copilot "how's my budget?" and get an answer.

---

## Users

**Primary:** AT&T engineers with GitHub Copilot Enterprise access (attuid-scoped GitHub).

**Context:**
- Run 2–5 concurrent Copilot CLI sessions across multiple project directories
- Use WezTerm, VS Code, and Microsoft Teams
- Work on-premise / corporate network
- Corporate GitHub: `github.com` with attuid
- NO external LLM providers; NO Anthropic direct calls
- npm registry: AT&T Artifactory (requires auth); workaround: `--registry https://registry.npmjs.org`
- Container registry: JFrog Artifactory (ACR is AT&T anti-pattern)

---

## Personas

| Persona | Pain | Win |
|---|---|---|
| **The Power User** (3–5 sessions/day) | Hits cap by mid-month; has no warning | Sees real-time burn rate; gets Teams alert at 60% |
| **The Optimizer** (cost-conscious) | Doesn't know which instruction files are expensive | Gets per-file token breakdown in the sidebar |
| **The New User** (onboarding) | No idea what "7,000 credits" means in practice | Status bar + dashboard explains it in plain English |

---

## Requirements

### P0 — Must have at launch (Phases 1–2)

- [ ] Real-time credit usage — current month, exact nanoAIU arithmetic
- [ ] Active session count + per-session credit breakdown
- [ ] Instruction file audit — token count, cost per session, severity
- [ ] VS Code status bar badge (green/yellow/red)
- [ ] VS Code sidebar panel (Budget, Active Sessions, Instruction Files nodes)
- [ ] VS Code dashboard webview (full breakdown)
- [ ] Go CLI `analyze` command for terminal users
- [ ] Go CLI `dashboard` command with WezTerm badge support

### P1 — High value, Phase 3

- [ ] Microsoft Teams webhook alert at 60% and 90% budget thresholds
- [x] Daily burn rate calculation — also surfaced in the VS Code dashboard and tree (as of 2026-06-15)
- [x] Month-end forecast (projected month-end total = used + dailyBurn × daysRemaining) — also surfaced in the VS Code dashboard and tree (as of 2026-06-15)
- [ ] Model routing recommender (flag expensive models) — **Go alert binary + MCP only**; not surfaced in VS Code

> **Surfacing (2026-06-15):** burn rate and the projected month-end total now appear in the VS Code dashboard and tree (`src/forecast/model.ts`), not just in Teams alerts and the MCP server. The model-routing recommender stays in the Go alert binary and MCP only.

### P2 — Phase 4

- [ ] MCP server — `get_budget_status`, `get_sessions`, `get_instruction_overhead`
- [ ] Copilot can answer "how's my budget?" mid-session via MCP tool call

### P3 — Phase 5 / future

- [ ] `.vsix` package distributed via internal VS Code marketplace
- [ ] Go binary distributed via JFrog Artifactory
- [ ] Onboarding runbook for new AT&T engineers
- [ ] Auto-compact advisor (context > 70% threshold warning)

---

## Non-requirements (deliberately deferred)

- Cloud sync or multi-user dashboard
- GitHub API calls of any kind
- Modifying or intercepting Copilot traffic
- GitHub Actions budget gate (interesting but not the core product)
- Any external network calls whatsoever

---

## Success metrics

| Metric | Target |
|---|---|
| Status bar accuracy vs Copilot billing | ≤ 1% error (nanoAIU is exact) |
| Teams alert latency from threshold crossing | ≤ next refresh cycle (30s) |
| Month-end forecast | Linear projection: projected month-end total = used + dailyBurn × daysRemaining. **Accuracy UNVALIDATED** — current tests only check the formula against itself. Real error pending a backtest (see G-backtest). Do not claim a numeric error target until measured. |
| Onboarding time (zero to status bar badge showing) | ≤ 5 minutes |
| VS Code extension startup impact | ≤ 50ms activation time |
