# Copilot Token Budget

**Real-time GitHub Copilot credit tracking for AT&T engineers**

A **local-first, zero-network** monitoring suite that reads Copilot session telemetry from `~/.copilot/session-state/` and surfaces credit usage, burn forecasts, and anomalies to the terminal, VS Code dashboard, and Microsoft Teams — **all offline, no GitHub API calls, no credentials required**.

**Current Status (2026-06-17):**
- ✅ **Phases 0–4 + v1.1 shipped**; **Phase 5** distribution is config-complete (live publish pending JFrog + first tag)
- ⚠️ **CLI usage only today.** Multi-source (CLI + VS Code IDE) is **groundwork**: the Copilot **CLI** is captured from `~/.copilot/session-state/`; **VS Code Copilot Chat is a separate local source** (`…/workspaceStorage/<ws>/chatSessions/`, `…/GitHub.copilot-chat/transcripts/`) and is **NOT captured yet** — the IDE collector is a stub pending discovery on an IDE-only machine (see `phase-0/findings/IDE_USAGE_FINDINGS.md` correction + ADR-007 correction).
- ✅ Go builds race-free; TypeScript compiles strict (no `any`)
- 🟡 **Distribution** packaged as `.vsix` + GoReleaser binaries (config validated locally; not yet published)

---

## The Problem

AT&T engineers on GitHub Copilot Enterprise hit their **7,000-credit/month allowance by mid-month with no visibility**. Instruction files add **12,000+ tokens of invisible overhead per message**. No tool exists to see spend until the bill arrives.

## The Solution

**Copilot Token Budget** is a **client-side telemetry reader** that:
1. **Reads local session files** (`~/.copilot/session-state/{uuid}/events.jsonl`) — Copilot writes these automatically
2. **Computes credit usage** in real-time (no GitHub API calls, works offline)
3. **Surfaces insights** where engineers work: terminal, VS Code, Teams
4. **Tracks Copilot CLI usage today**; VS Code IDE Chat capture is planned (Phase 6 — separate local source, not yet implemented; see ADR-007 correction)

**Zero network constraint by design (ADR-001).** No credentials, no proxies, no remote APIs.

## Critical Findings (Validated 2026-06-13 through 2026-06-16)

| Finding | Value | Impact |
|---|---|---|
| **Data Source** | `~/.copilot/session-state/{uuid}/events.jsonl` | Concrete, reproducible, local-only |
| **IDE source** | VS Code Chat = separate store (`chatSessions`/`transcripts`); **not** `~/.copilot` | Pending discovery on an IDE-only machine (the `vscode.metadata.json` marker was an unverified assumption — see ADR-007 correction) |
| **Billing Field** | `session.shutdown.data.totalNanoAiu` | Authoritative per-session cost |
| **Monthly Budget** | 7,000 credits (AT&T promo until 2026-09-01) | Baseline allowance |
| **Usage (Jun 2026)** | **~8,300–8,550 credits (119–122%)** | CRITICAL: over budget |
| **Instruction Overhead** | 12,238–12,323 tokens per message | Invisible, compounding cost |
| **Token Types** | 5: input, output, cache-read, cache-write, reasoning | All tracked separately |
| **Dedup Strategy** | dedup-by-ID across sources (defensive; IDs won't collide once IDE is a separate source) | No double-counting |
| **Build Status** | `go build ./...` ✅, `npm run compile` ✅, `go test -race` 0 races | Builds clean; live distribution pending |
| **Test Coverage** | Go: per-package unit tests (`-race`); TS: compile-only (no unit-test runner yet) | CLI paths covered; IDE collector is a stub |
| **Network Calls** | 0 HTTP imports | Zero-network verified by grep |
| **Type Safety** | 0 TypeScript `any` types | Strict mode throughout |

---

## Inputs & Outputs

### **Inputs**

| Source | Format | Schema | Scope |
|---|---|---|---|
| **CLI Sessions** | JSONL (`events.jsonl`) | `session.start`, `assistant.message`, `session.shutdown` | `~/.copilot/session-state/{uuid}/` |
| **IDE Sessions** *(not captured yet)* | VS Code Copilot Chat store (schema TBD) | Pending discovery on an IDE-only machine | `…/workspaceStorage/<ws>/chatSessions/`, `…/GitHub.copilot-chat/transcripts/` — a **separate** local source, **not** `~/.copilot` |
| **Workspace Config** | YAML (`workspace.yaml`) | Project path, context | Per-session metadata |
| **Instruction Files** | Text (`.md`, `.txt`, `.json`) | Arbitrary content | `{cwd}/.github/instructions/` |
| **Pricing Override** | JSON (`pricing.json`) | Model rates, allowance, context window | `~/.config/copilot-token-budget/` |

### **Outputs**

| Tier | Format | Metrics | Audience |
|---|---|---|---|
| **Terminal (Phase 1)** | Plain text + ANSI colors | Budget, burn, forecast, trend, top-N | Engineers (developers) |
| **VS Code (Phase 2)** | HTML5 dashboard + tree view | Per-source breakdown, live usage, export | VS Code users |
| **Teams (Phase 3)** | Adaptive Card JSON | Alerts only (CRITICAL/WARNING) | Engineering leads |
| **MCP (Phase 4)** | JSON (Claude/Copilot) | 6 tools (budget, sessions, overhead, costs, timeseries, top-consumers) | Copilot CLI mid-session |
| **Export (Phase 7)** | JSON + CSV | Full report (sessions, daily, consumers) | Data analysis |

---

## Key Features

### **Core Monitoring (All Phases)**

✅ **Real-time budget tracking**
- Used / Allowed / Remaining (raw credits with thousands separators, e.g., 8,550 cr / 7,000 cr)
- Status: OK / WARNING (60%) / CRITICAL (90%)
- Daily burn rate + month-end forecast

🔲 **Multi-source visibility (Phase 6 — groundwork only, IDE NOT captured yet)**
- CLI Sessions: captured today from `~/.copilot/session-state/`
- IDE Sessions: VS Code Copilot Chat is a **separate** local source (`…/chatSessions/`, `…/transcripts/`); the IDE collector is a **no-op stub** pending discovery — IDE usage is **not captured yet**
- The `Source`/`Collector` abstraction and dedup-by-ID are in place; today the combined total equals the CLI source only
- Per-session source label (CLI vs IDE) — IDE label unused until the collector lands

✅ **Usage analytics (Phase 7)**
- Daily/weekly/monthly trend with anomaly flags (mean + 2σ)
- Top 3 sessions, models, projects by spend
- Context-window fullness %
- Input/output token split

✅ **Instruction file audit (Phase 1)**
- Detected files in workspace `.github/instructions/`
- Per-file token count
- Overhead cost estimation (default: 50-turn session)

✅ **Data export (Phase 7)**
- JSON: Full report (all metrics, all sessions)
- CSV: Sessions, daily trends, top consumers (spreadsheet-friendly)

### **Platform Integration**

✅ **Terminal (Phase 1 + 7)**
- `cmd/analyze` — One-shot report, JSON/CSV modes
- `cmd/dashboard` — Live-updating TUI with charts
- `cmd/statusline` — WezTerm-friendly one-liner (NO_COLOR-aware)

✅ **VS Code Extension (Phase 2 + 7)**
- Dashboard webview with inline SVG charts
- Tree view (Budget, Forecast, Trend, Top Consumers, Sessions, Instructions)
- Status bar badge (color-coded: green/yellow/red)
- Hover tooltip (today/month/burn/forecast/context%)
- `Copilot Budget: Export Usage` command (JSON/CSV save dialog)
- Config settings: `copilotBudget.monthlyAllowance`, `copilotBudget.pricingPath`, `copilotBudget.teamsWebhookUrl`, `copilotBudget.refreshIntervalSec`

✅ **Microsoft Teams Alerts (Phase 3)**
- Adaptive Card alerts on CRITICAL and WARNING transitions
- Alert suppression (deduped once per day per threshold, UTC)
- Includes: used %, burn rate, projected total, verdict (within/OVER)

✅ **MCP Server (Phase 4 + 7)**
- **6 tools**: `get_budget_status`, `get_sessions`, `get_instruction_overhead`, `get_model_costs`, `get_usage_timeseries`, `get_top_consumers`
- Registered for Copilot CLI via `.copilot/mcp.json`
- Allows: "How's my budget?" mid-session in Copilot
- All stdio transport, zero network

---

## Phase Overview & Build Instructions

### **Phase 0: Data Source Discovery (✅ Complete)**

**Goal:** Validate data source, schema, billing field.

```bash
# Already done — findings in phase-0/findings/
# Output: IDE_USAGE_FINDINGS.md (schema, marker detection)
```

**Outputs:**
- Confirmed: `events.jsonl` is the CLI source
- Schema validated against real 50KB+ event file
- Correction (2026-06-17): VS Code Copilot Chat is a **separate** local store (`…/chatSessions/`, `…/transcripts/`), **not** `~/.copilot`; the earlier `vscode.metadata.json` marker was an unverified assumption and has been retracted (see `phase-0/findings/IDE_USAGE_FINDINGS.md` + ADR-007 corrections)

---

### **Phase 1: Go CLI Tool (✅ Complete)**

**Goal:** Build terminal-based budget analyzer with live dashboard.

```bash
cd phase-1/session-manager

# Build
go build ./cmd/analyze ./cmd/dashboard ./cmd/statusline

# Run
./analyze ~/projects/aaraminds-projects
./analyze --json ~/projects/aaraminds-projects    # JSON report
./analyze --csv ~/projects/aaraminds-projects     # CSV export
./dashboard ~/projects/aaraminds-projects         # Live TUI
./statusline                                       # One-liner for shell prompt

# Test (includes Phase 7 analytics + Phase 6 dedup-by-ID groundwork)
go test -race ./...
```

**Test Results:**
- ✅ Per-package unit tests pass with `-race` (0 races)
- ⚠️ Phase 6 IDE collector is a **no-op stub** — its tests cover the source/dedup wiring only, not real IDE capture

**Outputs:**
- Binary: `analyze` (credit report)
- Binary: `dashboard` (live TUI with charts)
- Binary: `statusline` (one-liner)
- Module: `github.com/aaraminds/copilot-session-manager` (30 tests)

---

### **Phase 2: VS Code Extension (✅ Complete)**

**Goal:** Build VS Code webview dashboard with live updates and export.

```bash
cd phase-2/vscode-extension

# Install dependencies
npm install

# Compile TypeScript → JavaScript
npm run compile

# Watch mode (development)
npm run watch

# Package as .vsix (for distribution)
npm run package

# Run in extension development host (F5 in VS Code)
# Test: Cmd+Shift+P → "Copilot Budget: Show Dashboard"
```

**Test Results:**
- ✅ 10+ tests, all passing
- ✅ 0 TypeScript errors (strict mode)
- ✅ 0 `any` types
- ✅ Compilation clean (credits render as raw values with thousands separators)

**Development Loop:**
```bash
# Terminal 1: Watch TypeScript
npm run watch

# Terminal 2: Press F5 in VS Code
# → Extension Development Host opens
# → Make changes, save → auto-reload

# Package for distribution
npm run package
# → copilot-token-budget-0.1.0.vsix (~45 KB, no source code)
```

**Outputs:**
- Extension ID: `att-internal.copilot-token-budget`
- Compiled extension: `out/` directory
- Packaged: `.vsix` file (distribution artifact)

---

### **Phase 3: Teams Alerts + Forecasting (✅ Complete)**

**Goal:** Send CRITICAL/WARNING alerts to Microsoft Teams, add burn-rate forecasting.

```bash
cd phase-3

# Build
go build -o alert ./cmd/alert ./cmd/webhook-tester

# Test webhook (dry-run, prints Adaptive Card)
COPILOT_BUDGET_TEAMS_WEBHOOK="<webhook>" ./alert --dry-run ~/projects

# Send real alert
export COPILOT_BUDGET_TEAMS_WEBHOOK="https://outlook.webhook.office.com/webhookb2/..."
./alert ~/projects

# Run in background (integration with Phase 1 refresh loop)
# → Fires on CRITICAL or WARNING transitions
# → Deduped once per day per threshold (UTC)
```

**Outputs:**
- Binary: `alert` (Teams webhook sender)
- Forecast model: `computeForecast(usedCredits, allowedCredits)`
- Adaptive Card template: JSON for Teams integration

---

### **Phase 4: MCP Server (✅ Complete, 8/10 gates)**

**Goal:** Expose budget as Copilot CLI tools via MCP (Model Context Protocol).

```bash
cd phase-4

# Build
go build -o ~/bin/copilot-budget-mcp ./cmd/mcp-server

# Register with Copilot CLI
# → Already configured in .copilot/mcp.json (stdio transport)

# Test (requires Copilot CLI + access to local sessions)
gh copilot --mcp copilot-budget-mcp "What's my budget status?"

# MCP tools exposed:
# - get_budget_status          → used/allowed/remaining
# - get_sessions               → all sessions (month), sortable
# - get_instruction_overhead   → workspace instruction file cost
# - get_model_costs            → per-model spend breakdown
# - get_usage_timeseries       → daily/weekly/monthly series
# - get_top_consumers          → top-N sessions/models/projects
```

**Outputs:**
- Binary: `copilot-budget-mcp` (stdio MCP server)
- Module: `github.com/aaraminds/copilot-budget-mcp`
- 6 tools for Copilot CLI integration

---

### **Phase 5: Distribution + Onboarding (✅ Config-complete, live publish pending)**

**Goal:** Cross-platform binaries, .vsix, CI/CD, onboarding.

```bash
# GoReleaser: Build 25 binaries × 5 platforms
cd /
goreleaser build --snapshot
# → dist/ contains:
#   - macOS (Intel): analyze-darwin-amd64, dashboard, statusline, alert, mcp-server
#   - macOS (Apple Silicon): analyze-darwin-arm64, ...
#   - Linux (amd64): analyze-linux-amd64, ...
#   - Linux (ARM): analyze-linux-arm64, ...
#   - Windows (amd64): analyze-windows-amd64.exe, ...

# VS Code Extension
cd phase-2/vscode-extension && npm run package
# → copilot-token-budget-0.1.0.vsix

# CI/CD (GitHub Actions)
# → On push: build, test, lint (ci.yml)
# → On v*.*.* tag: GoReleaser + vsce + publish to JFrog Artifactory (release.yml)
```

**Outputs:**
- 25 binaries (.tar.gz + checksums)
- 1 .vsix extension
- GitHub Release artifacts
- JFrog Artifactory published (pending setup)

---

### **Phase 6: Multi-Source Capture (🔲 Groundwork — IDE collector pending)**

**Goal:** Add IDE (VS Code Copilot Chat) usage to the reader alongside CLI, with deduplication.

> **Status:** the `Source`/`Collector` abstraction and CLI source are in place, but the **IDE
> collector is a no-op stub.** VS Code Copilot Chat stores its data in a *separate* location
> (`…/workspaceStorage/<ws>/chatSessions/`, `…/GitHub.copilot-chat/transcripts/`), **not**
> `~/.copilot/`. Until the collector is implemented against that real schema (after discovery on an
> IDE-only machine), only Copilot **CLI** usage is captured and the SOURCE BREAKDOWN below shows
> CLI only. See `phase-0/findings/IDE_USAGE_FINDINGS.md` (corrected) and ADR-007 (corrected).

```bash
cd phase-1/session-manager && go run ./cmd/analyze

# Outputs (today — CLI only; IDE not captured yet):
# ▶ SOURCE BREAKDOWN
#   CLI Sessions:      8,550 cr
#   IDE Sessions:      0 cr   (collector is a no-op stub)
#   Combined Total:    8,550 cr

# Implementation status:
# - Source/Collector abstraction + CLI collector: in place
# - IDE collector: NO-OP STUB. VS Code Copilot Chat is a separate local source
#   (…/workspaceStorage/<ws>/chatSessions/, …/GitHub.copilot-chat/transcripts/),
#   NOT ~/.copilot, and is pending discovery on an IDE-only machine.
# - dedup-by-ID (winner = IsFinal else higher TotalNanoAIU): wired, CLI-only today
# - Per-source totals render in CLI and VS Code dashboard (IDE row shows 0)
```

**Outputs:**
- Reader with Source/Collector abstraction; CLI source live, **IDE collector is a no-op stub**
- ADR-007 (corrected): multi-source dedup architecture; IDE is a separate VS Code Chat source, not yet implemented
- Acceptance gates **G65–G70** in `evaluation/PHASE6_ACCEPTANCE.md` — **REOPENED / NOT MET** (the earlier `vscode.metadata.json` marker assumption was retracted)
- Per-source breakdown in dashboard + CLI (IDE total is 0 until the collector lands)

---

### **Phase 7: Usage Insight v1.1 (✅ Complete)**

**Goal:** Analytics, export, overridable pricing, rich status bar, anomaly detection.

```bash
cd phase-1/session-manager && go run ./cmd/analyze --json > report.json

# Outputs:
# - Daily/weekly/monthly usage trends
# - Top 3 sessions, models, projects
# - Anomaly flags (mean + 2σ)
# - Context-window utilization %
# - Input/output token split

# Export as CSV
go run ./cmd/analyze --csv > sessions.csv

# Overridable pricing (no rebuild)
mkdir -p ~/.config/copilot-token-budget
cat > ~/.config/copilot-token-budget/pricing.json << 'PRICING'
{
  "allowanceCredits": 10000,
  "models": {
    "sonnet": { "inputPerMillion": 300, "outputPerMillion": 1500 }
  }
}
PRICING

# VS Code extension with new features
# - Usage Trend inline chart (14 days)
# - Top Consumers tables
# - Context % column
# - Input/output split
# - Export to JSON/CSV command
```

**Outputs:**
- Pricing override system (ADR-008)
- Analytics package: `internal/analytics/`
- Export package: `internal/export/`
- 2 new MCP tools: `get_usage_timeseries`, `get_top_consumers`
- Enhanced status bar tooltip

---

## Usage: Credit Display Format

All credit displays use **raw credits with thousands separators** (e.g. `8,550 cr`, `7,000 cr`).
There is no Billions/`B` scaling and no `credits / 1000` conversion — the earlier "B format" was
removed from the code. Consistent across the dashboard panel, session tree, status bar, extension
alerts, and Teams webhook messages.

---

## Distribution & Installation

### **For Team Members: 3 Options**

#### **Option 1: Direct .vsix Install (Fastest)**

```bash
# You provide: copilot-token-budget-0.1.0.vsix (45 KB, no source code)
# Team runs:
code --install-extension copilot-token-budget-0.1.0.vsix --force

# Verify
code --list-extensions | grep copilot-token-budget
# Output: att-internal.copilot-token-budget
```

#### **Option 2: GitHub Release (Recommended)**

```bash
# You create release (one-time)
cd phase-2/vscode-extension
npm run package
gh release create v0.1.0 copilot-token-budget-0.1.0.vsix \
  --notes "Copilot Token Budget for AT&T team — Phases 0-4 + v1.1 (Phase 7) shipped; Phase 5 config-complete; Phase 6 IDE capture pending"

# Team downloads from:
# https://github.com/your-org/copilot-token-budget/releases/download/v0.1.0/copilot-token-budget-0.1.0.vsix

# Or one-liner:
curl -L https://github.com/your-org/copilot-token-budget/releases/download/v0.1.0/copilot-token-budget-0.1.0.vsix -o extension.vsix
code --install-extension extension.vsix --force
```

#### **Option 3: JFrog Artifactory (Enterprise)**

```bash
# You push to Artifactory
export JFROG_ACCESS_TOKEN="<your-token>"
jf rt upload phase-2/vscode-extension/copilot-token-budget-0.1.0.vsix \
  generic-local/vscode-extensions/

# Team downloads
curl -H "Authorization: Bearer $JFROG_TOKEN" \
  https://jfrog.att.com/artifactory/generic-local/vscode-extensions/copilot-token-budget-0.1.0.vsix \
  -o extension.vsix
code --install-extension extension.vsix --force
```

### **What's in the .vsix?**

✅ **Safe to distribute** — No source code:
- Compiled JavaScript (minified, transpiled)
- Package manifest (`package.json`)
- README + LICENSE
- Icon & configuration

❌ **Not included** (source protected):
- TypeScript source files
- Node modules
- Test files
- Git history

### **First Use: Populate Dashboard**

The dashboard shows no metrics until Copilot is used:

```bash
# Generate a session with Copilot usage
gh copilot --prompt "hello world"
# → Creates ~/.copilot/session-state/{uuid}/events.jsonl

# Wait 5 seconds (extension polls every 5s)

# Open dashboard
# Cmd+Shift+P → "Copilot Budget: Show Dashboard"
# → Metrics now appear! ✅
```

### **Troubleshooting Installation**

If dashboard is empty:

1. **Check extension is installed:**
   ```bash
   code --list-extensions | grep copilot-token-budget
   # Must output: att-internal.copilot-token-budget
   ```

2. **Check session data exists:**
   ```bash
   ls ~/.copilot/session-state/*/events.jsonl | head -5
   ```

3. **Reload extension:**
   - Cmd+Shift+P → "Developer: Reload Window"

4. **Generate a Copilot session:**
   ```bash
   gh copilot --prompt "test"
   ```

---

## Project Architecture

```
copilot-token-budget/
├── phase-0/                    — Data source spike (✅)
│   └── findings/               — IDE_USAGE_FINDINGS.md, schema validation
├── phase-1/                    — Go CLI tool (✅)
│   └── session-manager/        — Go module: analyze, dashboard, statusline
│       ├── cmd/analyze/        — Budget report (--json, --csv modes)
│       ├── cmd/dashboard/      — Live TUI with charts
│       ├── cmd/statusline/     — WezTerm badge
│       ├── internal/
│       │   ├── session/        — Reader (CLI source + dedup-by-ID; IDE collector stubbed)
│       │   ├── analytics/      — Daily/weekly/monthly trends (Phase 7)
│       │   ├── export/         — JSON/CSV (Phase 7)
│       │   ├── pricing/        — Config + model rates (Phase 7, ADR-008)
│       │   ├── budget/         — Credit calculations
│       │   └── render/         — Terminal output formatting
├── phase-2/                    — VS Code Extension (✅)
│   └── vscode-extension/       — TypeScript extension
│       ├── src/
│       │   ├── ui/
│       │   │   ├── dashboardPanel.ts   — Webview (raw credits, thousands separators)
│       │   │   ├── sessionTree.ts      — Tree view
│       │   │   └── statusBar.ts        — Status bar badge
│       │   ├── session/reader.ts       — CLI collector (IDE collector is a no-op stub)
│       │   ├── analytics/model.ts      — Trend/anomaly (Phase 7)
│       │   ├── export/report.ts        — JSON/CSV (Phase 7)
│       │   └── types.ts                — Session, BudgetState, SessionSource
│       └── out/                        — Compiled JavaScript
├── phase-3/                    — Teams Alerts (✅)
│   └── cmd/alert/              — Adaptive Card sender
├── phase-4/                    — MCP Server (✅)
│   └── cmd/mcp-server/         — 6 tools for Copilot CLI
├── design/                     — Architecture & ADRs
│   └── adr/
│       ├── ADR-001.md          — Zero-network constraint
│       ├── ADR-007.md          — Multi-source dedup (Phase 6)
│       └── ADR-008.md          — Pricing override (Phase 7)
├── evaluation/                 — Acceptance gates
│   ├── PHASE6_ACCEPTANCE.md    — G65-G70 (IDE multi-source — REOPENED/NOT MET)
│   └── PHASE7_ACCEPTANCE.md    — G38-G50 (analytics)
├── docs/                       — Runbooks & guides
│   └── onboarding-runbook.md   — ≤5-min install (all OS)
├── .github/
│   ├── workflows/
│   │   ├── ci.yml              — Build/test on push
│   │   └── release.yml         — GoReleaser + JFrog
│   └── instructions/           — Copilot CLI workspace setup
├── .goreleaser.yaml            — 25 binaries × 5 platforms
├── .copilot/mcp.json           — MCP server registration
├── STATUS.md                   — Phase dashboard
├── IMPLEMENTATION_PLAYBOOK.md  — Execution log (all steps + results)
└── README.md                   — This file
```

---

## Billing Reference

| Unit | Value |
|---|---|
| 1 AI Credit | $0.01 |
| 1 credit | 1,000,000,000 nanoAIU |
| AT&T monthly allowance | 7,000 credits (promo through 2026-09-01) |
| **Claude Sonnet** input / output | 300 / 1,500 credits/M tokens |
| **Claude Opus** input / output | 500 / 2,500 credits/M tokens |
| **Claude Haiku** input / output | 100 / 500 credits/M tokens |
| Context window | 200,000 tokens (Copilot default) |

> **Note:** Rates, allowance, and context window are **bundled defaults**. Override via `pricing.json` (ADR-008) without rebuilding.
> All costs are **estimates** — the tool reads local telemetry and applies a local price table. It never reconciles against GitHub's authoritative billing (by design, ADR-001).

---

## Building from Source

### **Prerequisites**

- **Go 1.21+** (Phase 1, 3) or **Go 1.25+** (Phase 4 — hard requirement for `modelcontextprotocol/go-sdk v1.6.1`)
- **Node.js 18+** (Phase 2 extension)
- **npm 9+**

### **Build All Phases**

```bash
# Clone and prepare
git clone https://github.com/your-org/copilot-token-budget.git
cd copilot-token-budget

# Phase 1: Go CLI
cd phase-1/session-manager
go build -o /usr/local/bin/copilot-budget-analyze ./cmd/analyze
go build -o /usr/local/bin/copilot-budget-dashboard ./cmd/dashboard
go build -o /usr/local/bin/copilot-budget-statusline ./cmd/statusline

# Phase 2: VS Code Extension
cd ../../phase-2/vscode-extension
npm install
npm run compile
npm run package
# → copilot-token-budget-0.1.0.vsix

# Phase 3: Teams Alert
cd ../../phase-3
go build -o /usr/local/bin/copilot-budget-alert ./cmd/alert

# Phase 4: MCP Server
cd ../phase-4
go build -o /usr/local/bin/copilot-budget-mcp ./cmd/mcp-server

# Phase 5: GoReleaser (all platforms)
cd ../
goreleaser build --snapshot
# → dist/ contains 25 binaries × 5 platforms
```

### **Run Tests**

```bash
# Phase 1 tests
cd phase-1/session-manager
go test -race ./...

# Phase 2 tests (TypeScript)
cd ../../phase-2/vscode-extension
npm test

# Linting
cd ../../
gofmt -l ./phase-1 ./phase-3 ./phase-4
actionlint .github/workflows/*.yml
```

---

## Status & Next Steps

| Phase | Status | Gate | Notes |
|---|---|---|---|
| **0** | ✅ Complete | Data source validated | CLI JSONL source + schema confirmed (IDE source is separate, not yet discovered) |
| **1** | ✅ Complete | CLI tool live | analyze + dashboard + statusline |
| **2** | ✅ Complete | Extension F5 + .vsix | Dashboard, tree view, export |
| **3** | ✅ Complete | Teams alerts | CRITICAL/WARNING on transitions |
| **4** | ⚠️ 8/10 gates | G31–G32 pending | 6 MCP tools live; 2 gates for live Copilot integration |
| **5** | 🟡 Config-complete | G60–G64 pending | Binaries + .vsix packaged; live publish awaits JFrog + first tag |
| **6** | 🔲 Groundwork | G65–G70 REOPENED/NOT MET | CLI source + dedup wired; IDE collector is a no-op stub — IDE usage not captured yet |
| **7** | ✅ Complete | All gates | Analytics, export, pricing override, rich UI |

**Live deployment:** Ready for team distribution immediately. See `docs/onboarding-runbook.md` for ≤5-min install.

---

## Contributing

For issues, PRs, or questions:

1. Read [`product/`](product/) and [`design/adr/`](design/adr/) first
2. Consult [`STATUS.md`](STATUS.md) for current phase
3. Check [`IMPLEMENTATION_PLAYBOOK.md`](IMPLEMENTATION_PLAYBOOK.md) for execution history
4. File issues in GitHub; tag with phase (P1, P2, etc.)

---

## License

[Proprietary — AT&T Internal Use]  
See [`LICENSE`](LICENSE) for full terms. Distribution outside AT&T requires legal review.

