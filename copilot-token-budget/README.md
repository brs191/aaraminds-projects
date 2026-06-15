# Copilot Token Budget

Real-time GitHub Copilot CLI credit tracking for AT&T engineers — local, zero-network, evidence-first.

## The problem

AT&T engineers running multiple GitHub Copilot CLI sessions exhaust their 7,000-credit monthly
allowance faster than expected, with no visibility until the limit is hit. Instruction files add
12,000+ tokens of overhead *per message* — completely invisible to the user.

## The solution

A client-side tool suite that reads the session state files the Copilot CLI already writes to
`~/.copilot/session-state/`, computes exact credit usage, and surfaces it where engineers work:
terminal, VS Code, and Microsoft Teams.

**Zero network calls. No GitHub API. Works offline. Reads local files only.**

## Features

- Per-month credit total + % of allowance, color-coded green/yellow/red
- Active-session list with per-session credits and **context-window %**
- Instruction-file overhead audit
- Daily burn rate + projected month-end total
- **Usage trend** — daily/weekly/monthly series with **anomaly flags** (mean + 2·σ)
- **Top consumers** — most expensive sessions, models, and projects (top-N)
- **JSON / CSV export** of the full report
- **Statusline** — ccusage-style one-liner for shell prompts / WezTerm
- **Overridable pricing** — bundled rates editable via `pricing.json` (no rebuild)
- Teams threshold alerts; VS Code status bar, tree, and dashboard webview
- **Six MCP tools** so Copilot CLI can answer "how's my budget?" mid-session

All cost figures are **estimates** — the tool reads local telemetry and applies a local price
table; it never reconciles against GitHub's authoritative billing (by design).

## Key findings (data source validated 2026-06-13)

| Finding | Value |
|---|---|
| Monthly budget (AT&T promo, until 2026-09-01) | 7,000 credits |
| Budget consumed by mid-June 2026 (month-scoped tool) | **~8,300–8,550 credits (~119–122%) — OVER BUDGET** |
| Instruction file overhead per message | 12,238–12,323 tokens |
| Workspace always-loaded instruction tokens | 2,183 tokens |
| Largest single instruction file (`apm0045942-credit-routing-app`) | 7,779 tokens |
| Data source | `~/.copilot/session-state/<uuid>/events.jsonl` |
| Billing field | `session.shutdown.totalNanoAiu` |

> Note: the original Phase 0 spike reported 14,144.66 cr (202%) — an over-count, because it summed
> all shutdown events without calendar-month scoping. The figure above is the correct month-scoped
> number from the Phase 1 tool (`ReadThisMonth()`), and it climbs through the month. See `STATUS.md`.

## Folder map

```
copilot-token-budget/
  product/                    — PRD, north-star goals
  design/                     — Architecture, ADRs
    adr/                      — Architecture decision records
  planning/                   — Delivery roadmap, milestones
  evaluation/                 — Acceptance criteria, eval rubrics
  tracking/                   — Sprint state, phase-gate status
  phase-0/                    — Spike: data source validation (COMPLETE)
    findings/                 — FINDINGS_MEMO.md, raw data
  phase-1/                    — Go CLI tool: analyze + dashboard (COMPLETE)
    session-manager/          — Go module: cmd/analyze, cmd/dashboard
  phase-2/                    — VS Code extension (COMPLETE — F5 + .vsix verified)
    vscode-extension/         — TypeScript extension source + out/
  phase-3/                    — Teams alerts + budget forecasting (COMPLETE)
  phase-4/                    — MCP server: SIX tools, stdio (COMPLETE — 8/10 gates)
  phase-5/                    — Distribution + onboarding (NOT STARTED)
  .copilot/                   — mcp.json: registers the Phase 4 MCP server
  .github/
    instructions/             — Copilot CLI workspace instructions
    workflows/                — CI/CD (JFrog deploy)
```

## Quick start

**Prerequisites — Go version:** phase-1 and phase-3 build on **Go 1.21+**. **phase-4 (the MCP server) requires Go 1.25+** — this is a hard dependency: `modelcontextprotocol/go-sdk v1.6.1` requires Go ≥ 1.25, and `phase-4/go.mod` declares `go 1.25.0` deliberately. This is an intentional requirement, not version skew. (Node 18+ for the Phase 2 VS Code extension.)

**Go CLI (Phase 1 + v1.1):**
```bash
cd phase-1/session-manager
go run ./cmd/analyze ~/projects/aaraminds-projects
go run ./cmd/analyze --json ~/projects/aaraminds-projects   # full report as JSON (camelCase)
go run ./cmd/analyze --csv  ~/projects/aaraminds-projects   # sessions/daily as CSV
go run ./cmd/dashboard ~/projects/aaraminds-projects

# ccusage-style status line (one shot, no network, never panics, NO_COLOR-aware):
go run ./cmd/statusline
# embed in WezTerm right-status or a shell prompt; honours NO_COLOR=1
```

**Overridable pricing (v1.1):** drop a `pricing.json` into the config dir to override rates,
allowance, or context windows — partial files merge over the bundled defaults; a missing or
malformed file falls back to defaults. Path: `~/.config/copilot-token-budget/pricing.json`
(macOS/Linux) or `%AppData%\copilot-token-budget\pricing.json` (Windows). See ADR-008.

**VS Code Extension (Phase 2 + v1.1):**
```
File → Open Folder → phase-2/vscode-extension
Press F5 → Extension Development Host opens
```
- Dashboard adds a **Usage Trend** inline chart, **Top Consumers** tables, a context-% column,
  and an input/output split; the status-bar tooltip shows today/month/allowance%/burn/projected/context%.
- Command **`Copilot Budget: Export Usage`** (`copilotBudget.exportUsage`) — JSON/CSV save dialog.
- Setting **`copilotBudget.pricingPath`** — path to a pricing override file (mirrors `pricing.json`).
- Allowance: an explicit `copilotBudget.monthlyAllowance` wins, else the pricing config's allowance.

**Teams alerts (Phase 3):**
```bash
cd phase-3
COPILOT_BUDGET_TEAMS_WEBHOOK="<webhook>" go run ./cmd/alert ~/projects/aaraminds-projects
go run ./cmd/alert --dry-run ~/projects/aaraminds-projects   # prints Adaptive Card JSON, no POST
```

**MCP server (Phase 4 + v1.1):**
```bash
cd phase-4
go build -o ~/bin/copilot-budget-mcp ./cmd/mcp-server
# registered for Copilot CLI via .copilot/mcp.json (stdio transport)
```
Exposes **six tools**: `get_budget_status`, `get_sessions`, `get_instruction_overhead`,
`get_model_costs`, `get_usage_timeseries` (daily/weekly/monthly), `get_top_consumers` (top-N
sessions/models/projects). All read local files only — zero network.

## Billing reference

| Unit | Value |
|---|---|
| 1 AI Credit | $0.01 |
| 1 credit | 1,000,000,000 nanoAIU |
| AT&T monthly allowance | 7,000 credits/month (promo until 2026-09-01) |
| Claude Sonnet input | 300 credits/M tokens |
| Claude Sonnet output | 1,500 credits/M tokens |
| Claude Opus input | 500 credits/M tokens |
| Claude Opus output | 2,500 credits/M tokens |
| Claude Haiku input | 100 credits/M tokens |
| Claude Haiku output | 500 credits/M tokens |

Rate source: GitHub Copilot models-and-pricing (1 credit = $0.01; credits/M token = USD/Mtoken × 100).
Context window: 200,000 tokens per model (Copilot default / non-extended).

> Since v1.1, these rates, the allowance, and the context window are **bundled defaults** that are
> overridable via `pricing.json` (ADR-008) — change them without a rebuild. All costs are estimates.

## Status

**Phases 0–4 complete; Phase 5 (distribution) is next.** Phase 4 closed with 8/10 automated gates green —
G31 (live Copilot CLI invocation) and G32 (pin go-sdk to a commit hash) remain before final distribution.

See [STATUS.md](STATUS.md) for the live phase dashboard.
See [BUILD_PLAN.md](BUILD_PLAN.md) for the phased build plan with gates.
See [IMPLEMENTATION_PLAYBOOK.md](IMPLEMENTATION_PLAYBOOK.md) for the execution log.
