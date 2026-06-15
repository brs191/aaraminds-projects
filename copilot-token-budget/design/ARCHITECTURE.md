# Copilot Token Budget — Architecture

**Status:** Phases 1–4 implemented; v1.1 usage-insight increment (Phase 7) shipped. Phases 5–6 pending.
**Last updated:** 2026-06-16

---

## Core principle

> **Local-first, deterministic, zero-network.** Every credit figure is computed client-side from
> files the Copilot CLI already writes. No GitHub API. No Copilot API. No token interception.
> No proxy. Works offline. Works on corporate networks with no external egress.

This is the founding constraint that drives every architectural decision.

---

## Component map

```
~/.copilot/session-state/
  <uuid>/
    events.jsonl        ← billing data (totalNanoAiu, systemTokens, modelMetrics)
    workspace.yaml      ← session metadata (workspaceDir, startTime)
    inuse.<pid>.lock    ← active session indicator
          │
          │  read (local file I/O only)
          ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Core Data Layer (Go)                        │
│  internal/session/reader.go   — Source + Collector + ReadAll     │
│      cliCollector (live)  ·  ideCollector (Phase 6 stub)          │
│      dedup-by-ID (final wins, else higher TotalNanoAIU)          │
│  internal/budget/tracker.go   — nanoAIU → credits → dollars     │
│  internal/instructions/       — scan .github/instructions/      │
│  internal/pricing/            — overridable pricing.json (ADR-008)│
│  internal/analytics/          — series, top-N, context%, anomaly  │
│      (UTC bucketing — Go↔TS parity)                              │
│  internal/export/             — Report→JSON, Sessions/DailyToCSV  │
└────────────────────┬────────────────────────────────────────────┘
                     │
     ┌───────────────┼───────────┬──────────────┬──────────────┐
     ▼               ▼           ▼              ▼              ▼
  ┌────────┐ ┌──────────┐ ┌────────────┐ ┌───────────┐ ┌───────────┐
  │ cmd/   │ │ cmd/     │ │ cmd/        │ │ VS Code   │ │ MCP Server│
  │analyze │ │dashboard │ │ statusline  │ │ Extension │ │ (Go) Ph 4 │
  │ (Go)   │ │  (Go)    │ │ (ccusage-   │ │ (TS)      │ │ SIX tools │
  │ --json │ │          │ │  style)     │ │           │ │           │
  │ --csv  │ │          │ │             │ │           │ │           │
  └────────┘ └──────────┘ └─────────────┘ └─────┬─────┘ └─────┬─────┘
                                                │             │
                                         ┌──────┴──────┐      ▼
                                         │ Status bar  │ Copilot CLI
                                         │ Sidebar     │ (MCP client)
                                         │ Dashboard   │
                                         │  (Usage     │
                                         │   Trend SVG,│
                                         │   Top       │
                                         │   Consumers,│
                                         │   context%) │
                                         │ exportUsage │
                                         └──────┬──────┘
                                                │
                                         ┌──────┴──────────┐
                                         │  Teams Webhook  │  (Phase 3)
                                         │  (Go HTTP POST) │
                                         └─────────────────┘
```

---

## Data flow (Phase 1 + 2, with v1.1 layers)

```
1. Copilot CLI writes events.jsonl on session.shutdown
2. session reader runs each Collector:
     cliCollector scans ~/.copilot/session-state/ for all session dirs
     ideCollector (Phase 6 stub) returns nothing pending Step 6.0 discovery
3. Finds inuse.*.lock → marks session active (IsFinal=false); sets Source
4. Parses NDJSON → extracts totalNanoAiu, systemTokens, modelMetrics
5. ReadAll dedups by session ID (final wins, else higher TotalNanoAIU) → no double-count
6. tracker.go: sum(totalNanoAiu for sessions this month) / 1e9 = credits
7. instructions/analyzer.go: scan .github/instructions/ → estimate token counts
8. pricing.Load(): bundled defaults merged over ConfigDir()/pricing.json (ADR-008)
9. analytics: DailySeries/WeeklySeries/MonthlySeries (UTC bucketing), TopSessions/
   TopModels/TopProjects, ContextWindowPct, AnomalousDays (mean + 2·σ)
10. export: Report→JSON (camelCase) / SessionsToCSV / DailyToCSV (RFC-4180 quoted)
11. UI layer (Go CLI, statusline, TS extension, MCP) renders the data
```

**UTC bucketing invariant:** analytics normalizes billing time to UTC before computing the
day/week/month boundary, so Go and TS produce identical bucket keys regardless of host timezone.

---

## Billing arithmetic

```
credits = totalNanoAiu / 1_000_000_000
dollars = credits * 0.01
pctUsed = credits / monthlyAllowance * 100

// AT&T promo: 7,000 credits/month until 2026-09-01
```

Model rate card (credits per M tokens; source: GitHub Copilot models-and-pricing,
1 credit = $0.01, credits/M token = USD/Mtoken × 100):

| Model | Input | Output |
|---|---|---|
| Claude Sonnet | 300 | 1,500 |
| Claude Opus | 500 | 2,500 |
| Claude Haiku | 100 | 500 |

These rates (plus the allowance and a 200,000-token context window per model) are the **bundled
defaults**; since v1.1 they are overridable via `ConfigDir()/pricing.json` (ADR-008). All costs are
estimates.

---

## Phase 3 design — Teams alerts

```
GET ~/.copilot/session-state/ (every 30s in dashboard loop)
  │
  ├── budget crosses 60% threshold? → POST to Teams webhook (warning)
  ├── budget crosses 90% threshold? → POST to Teams webhook (critical)
  └── burn rate > (allowance / days_in_month) * 1.5? → POST (pace warning)

Teams webhook payload: Adaptive Card with budget gauge, top sessions, forecast
```

Alert deduplication: store last-alerted threshold in `~/.config/copilot-token-budget/state.json`
so the same threshold doesn't fire more than once per day.

---

## Phase 4 design — MCP server

```go
// SIX tools exposed (v1.1):
get_budget_status        → BudgetState{credits, pct, status, daysLeft, forecast}
get_sessions             → []Session{name, credits, contextTokens, model, isActive}  // all sessions this month with isActive flag, sorted by credits desc
get_instruction_overhead → []InstructionFile{name, tokens, severity}
get_model_costs          → map[model]ModelCost{inputRate, outputRate, totalCredits}  // rates sourced from internal/pricing (ADR-008)
get_usage_timeseries     → {buckets:[{key, start (RFC3339), sessions, credits, inputTokens, outputTokens}]}  // granularity daily(default)/weekly/monthly; UTC bucketing
get_top_consumers        → {topSessions, topModels, topProjects: []{name, credits, inputTokens, outputTokens, model}}  // current month, n default 5
```

Both new tools call `validateWorkspacePath` (absolute + within home, symlink-resolved) and make
zero network calls, like the original four. Handlers are pure functions (no shared state).

Transport: stdio (same as Copilot CLI's own MCP servers).
Implementation: Go, `modelcontextprotocol/go-sdk` (v1.6.1).
Registration: `.copilot/mcp.json` in workspace root.

**Technology choice — Go version:** phase-4 (`go.mod` declares `go 1.25.0`) **requires Go 1.25+**
because `modelcontextprotocol/go-sdk v1.6.1` requires Go ≥ 1.25. This is an intentional,
hard dependency requirement — not version skew. phase-1 and phase-3 remain on **Go 1.21+**.

---

## Configuration and state

All config/state lives under `platform.ConfigDir()` (one cross-platform path helper — ADR-006):

| File | Purpose | ADR |
|---|---|---|
| `pricing.json` | Overridable per-model rates, allowance, context windows; merged over bundled defaults; graceful fallback | ADR-008 |
| `state.json` | Teams alert dedup (threshold → date); 0600; no secrets | ADR-006 |

| Platform | ConfigDir() |
|---|---|
| macOS/Linux | `~/.config/copilot-token-budget/` |
| Windows | `%AppData%\copilot-token-budget\` |

The TS extension reads its pricing override from the explicit setting `copilotBudget.pricingPath`
(it does not assume the Go config dir). Allowance precedence in the extension: an explicitly set
`copilotBudget.monthlyAllowance` wins, else `pricing.allowanceCredits` (ADR-008 §6).

---

## v1.1 usage-insight surfaces

| Surface | What v1.1 added |
|---|---|
| `cmd/analyze` | `--json` / `--csv`; sections "USAGE TREND (last 14 days)" with anomaly flags, "TOP CONSUMERS", context-window % on active sessions |
| `cmd/dashboard` | Same trend / top-consumers / context-% sections |
| `cmd/statusline` | New ccusage-style one-liner (credits-based, NO_COLOR-aware, never panics, exits 0) |
| MCP server | Two new tools (`get_usage_timeseries`, `get_top_consumers`) → six total |
| VS Code extension | Usage Trend inline-SVG chart; Top Consumers tables; context-% column; input/output split; richer status-bar tooltip (today/month/allowance%/burn/projected/context%); new command `copilotBudget.exportUsage` (JSON/CSV save dialog); setting `copilotBudget.pricingPath` |

All v1.1 figures are **estimates** (ADR-001 / ADR-008). Acceptance gates: `evaluation/PHASE7_ACCEPTANCE.md`.

---

## ADR index

| ADR | Decision | Status |
|---|---|---|
| [ADR-001](adr/ADR-001-local-file-only.md) | Local file read only — no GitHub API | Accepted |
| [ADR-002](adr/ADR-002-go-zero-deps.md) | Go tool with zero external dependencies | Accepted |
| [ADR-003](adr/ADR-003-vscode-ts.md) | VS Code extension in TypeScript, zero runtime deps | Accepted |
| [ADR-004](adr/ADR-004-teams-not-slack.md) | Microsoft Teams for alerts (not Slack) | Accepted |
| [ADR-005](adr/ADR-005-jfrog-registry.md) | JFrog Artifactory for distribution (not ACR) | Accepted |
| [ADR-006](adr/ADR-006-config-storage.md) | Cross-platform config and state storage | Accepted |
| [ADR-008](adr/ADR-008-overridable-pricing-config.md) | Overridable local pricing configuration | Accepted |
| [ADR-009](adr/ADR-009-usage-analytics-and-source-abstraction.md) | Usage analytics, export, and source abstraction | Accepted |

> ADR-007 (multi-source capture / IDE parser) is **planned**, not yet written — see Step 6.1
> in `IMPLEMENTATION_PLAYBOOK.md`. ADR-009 lands its groundwork (Source/Collector/dedup).

---

## Technology choices

| Layer | Technology | Rationale |
|---|---|---|
| Core data layer | Go 1.21, zero external deps | Static binary, fast startup, `go.sum` clean |
| VS Code extension | TypeScript 5.4, `@types/vscode` only | Official VS Code API; no runtime deps |
| Teams alerting | Go `net/http` | Webhook is a plain HTTPS POST; no SDK needed |
| MCP server | Go + `modelcontextprotocol/go-sdk` | Official SDK; same language as core |
| Distribution | JFrog Artifactory | AT&T standard — ACR is anti-pattern |
| npm registry | `registry.npmjs.org` (workaround) | AT&T Artifactory requires auth; public fallback |
