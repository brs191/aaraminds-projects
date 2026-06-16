# Copilot Token Budget — Usage Guide

How to run the outcome of **each phase**. Everything here is local-first and zero-network
(the only outbound call is the optional Teams webhook in Phase 3).

> **A note on two "workspace" concepts.** Credit/session data is *always* read from
> `~/.copilot/session-state/` regardless of arguments. The optional `[workspace-root]` argument on
> `analyze`/`dashboard`/`alert` only tells the tool where to scan `.github/instructions/**` for the
> **instruction-file overhead audit**. If you omit it, the current directory is used.

---

## Supported platforms

**macOS, Linux (incl. Ubuntu), and Windows.** The core is pure Go stdlib with cross-platform path
helpers (`os.UserHomeDir`, `os.UserConfigDir`, `filepath.Join`) and the TypeScript extension uses
`path`/`os` only — there are no hardcoded OS paths. Verified building (`linux/amd64`) and running on
**Ubuntu 20.04** (analyze, statusline, all v1.1 sections). Config lives at `~/.config/copilot-token-budget/`
on Linux/macOS and `%AppData%\copilot-token-budget\` on Windows; session data is read from
`~/.copilot/session-state/` (or `%USERPROFILE%\.copilot\session-state\`) on every OS.

> The IDE-discovery script (`phase-0/discover-ide-usage.sh`) is OS-aware: it scans
> `~/Library/Application Support/Code*` on macOS and `~/.config/Code*` + `~/.vscode-server` on Linux.
> Phase 5 distribution will add `linux/amd64` + `linux/arm64` build targets alongside macOS/Windows.

## Prerequisites

| Need | Why |
|---|---|
| **Go 1.21+** | builds phase-1 and phase-3 |
| **Go 1.25+** | builds phase-4 (the MCP server) — hard requirement of `modelcontextprotocol/go-sdk v1.6.1`. `GOTOOLCHAIN=auto` (default) will auto-fetch it. |
| **Node 18+** | phase-2 VS Code extension build |
| GitHub Copilot CLI that has produced sessions under `~/.copilot/session-state/` | there is nothing to report without real session data |

Repo root referenced below as `copilot-token-budget/`.

---

## Phase 0 — Data-source validation (spike)

**Outcome:** confirmation that `~/.copilot/session-state/` carries the billing fields. Findings live in `phase-0/findings/FINDINGS_MEMO.md` + `sample_event.json` (no command needed to re-run).

**IDE discovery spike (Phase 6 prerequisite, Step 6.0)** — run on your Mac to map VS Code IDE Copilot usage:

```bash
bash phase-0/discover-ide-usage.sh > phase-0/findings/ide-usage-report.txt
cat phase-0/findings/ide-usage-report.txt   # read-only, zero-network, redacts PII
```

---

## Phase 1 — Go CLI (analyze + dashboard)

```bash
cd phase-1/session-manager

# One-shot report (active sessions, history, budget, instruction audit, usage trend, top consumers)
go run ./cmd/analyze [workspace-root]

# Live dashboard — refreshes every 10s, updates the WezTerm tab badge, Ctrl+C to exit
go run ./cmd/dashboard [workspace-root]
```

**Guided launcher** (preflight → build → one-shot report → dashboard):

```bash
./phase-1/run.sh [workspace-root]     # defaults to the aaraminds-projects workspace if omitted
```

Build static binaries instead of `go run`:

```bash
cd phase-1/session-manager
go build -o ~/bin/copilot-analyze   ./cmd/analyze
go build -o ~/bin/copilot-dashboard ./cmd/dashboard
```

---

## Phase 2 — VS Code extension

**Run from source (F5):**

```bash
cd phase-2/vscode-extension
npm install --registry https://registry.npmjs.org   # first time (AT&T proxy workaround)
npm run compile
# In VS Code: File → Open Folder → phase-2/vscode-extension, then press F5 → Extension Development Host
```

**Build + install a `.vsix`:**

```bash
cd phase-2/vscode-extension
npm run package                       # produces copilot-token-budget-*.vsix
code --install-extension copilot-token-budget-*.vsix
```

**Commands** (Command Palette): `Copilot Budget: Show Dashboard`, `: Refresh`, `: Open Settings`,
`: Export Usage` (`copilotBudget.exportUsage` — JSON/CSV save dialog).

**Settings** (`copilotBudget.*`): `monthlyAllowance`, `workspacePath`, `refreshIntervalSec`,
`teamsWebhookUrl`, `alertThresholdWarn`, `alertThresholdCrit`, `alertBinaryPath`, `pricingPath`.

---

## Phase 3 — Teams alerts + forecasting

The webhook URL comes **only** from the `COPILOT_BUDGET_TEAMS_WEBHOOK` env var (never a flag).
The `<workspace-root>` argument is **required**.

```bash
cd phase-3

# Safe preview — builds the Adaptive Card JSON, makes NO network call
go run ./cmd/alert --dry-run [--allowance 7000] <workspace-root>

# Real alert — POSTs to Teams only if a threshold (60% warn / 90% crit) is crossed
#   and that threshold hasn't already fired today (dedup, UTC, per-day)
COPILOT_BUDGET_TEAMS_WEBHOOK="https://<your-teams-webhook>" \
  go run ./cmd/alert <workspace-root>
```

Exit codes: `0` = no alert needed / already sent today · `1` = alert fired (or dry-run printed) · `2` = error.

**Run it on a schedule** (cron example, every 30 min during work hours):

```bash
*/30 9-18 * * 1-5  COPILOT_BUDGET_TEAMS_WEBHOOK="https://..." /path/to/copilot-alert /path/to/workspace
```

> The alert engine is also invoked automatically by the VS Code extension's refresh loop when
> `copilotBudget.teamsWebhookUrl` is set and the `copilot-alert` binary is found
> (`copilotBudget.alertBinaryPath` or `~/bin/copilot-alert`).

---

## Phase 4 — MCP server (six tools)

```bash
cd phase-4
go build -ldflags "-X main.Version=v0.1.0" -o ~/bin/copilot-budget-mcp ./cmd/mcp-server
```

Register it in `.copilot/mcp.json` (already scaffolded at the repo root) — **use an absolute path**;
`~` is not expanded by `execve`:

```json
{
  "mcpServers": {
    "copilot-token-budget": {
      "command": "/Users/<you>/bin/copilot-budget-mcp",
      "args": [],
      "env": {}
    }
  }
}
```

Then, in a Copilot CLI session, the model can call: `get_budget_status`, `get_sessions`,
`get_instruction_overhead`, `get_model_costs`, `get_usage_timeseries` (daily/weekly/monthly),
`get_top_consumers` (top-N sessions/models/projects). All read local files only.

Smoke-test the binary directly:

```bash
~/bin/copilot-budget-mcp --version
```

---

## Phase 5 — Distribution + onboarding · NOT STARTED

No runnable outcome yet. Planned: `goreleaser` cross-compiled binaries + `.vsix` published to JFrog
Artifactory via GitHub Actions on tag push, plus an onboarding runbook. See `BUILD_PLAN.md` Phase 5.

---

## Phase 6 — Dual-source capture (Copilot CLI + VS Code IDE) · GROUNDWORK ONLY

The Source/Collector abstraction and dedup are in place, but the **IDE collector is a stub** — today
`ReadAll()` returns CLI sessions only. To unblock the IDE parser, run the Phase 0 discovery script
above and share the output. No new commands until the schema is known (see `IMPLEMENTATION_PLAYBOOK.md`
Phase 6 and ADR-009).

---

## Phase 7 — Usage insight (v1.1)

These ship inside the Phase 1/2/4 binaries you already built.

**Machine-readable export** (CLI):

```bash
cd phase-1/session-manager
go run ./cmd/analyze --json [workspace-root]   # full report as JSON (camelCase)
go run ./cmd/analyze --csv  [workspace-root]   # per-session CSV (RFC-4180)
```

**Status line** (ccusage-style one-liner; no args, never panics, exits 0, honours `NO_COLOR`):

```bash
go run ./cmd/statusline
# example: 🤖 sonnet-4.6 | 💰 0 today / 4950/7000 (71%) | 🔥 309/day | 🧠 75%
# embed in a shell prompt or WezTerm right-status by calling the built binary
```

**Override pricing / allowance / context window** (no rebuild) — drop a `pricing.json` in the config dir.
Partial files merge over the bundled defaults; a missing/malformed file falls back safely.

```
macOS/Linux : ~/.config/copilot-token-budget/pricing.json
Windows     : %AppData%\copilot-token-budget\pricing.json
```

```json
{
  "allowanceCredits": 7000,
  "models": {
    "sonnet": { "inputPerMillion": 300, "outputPerMillion": 1500, "contextWindowTokens": 200000 },
    "opus":   { "inputPerMillion": 500, "outputPerMillion": 2500, "contextWindowTokens": 200000 },
    "haiku":  { "inputPerMillion": 100, "outputPerMillion": 500,  "contextWindowTokens": 200000 }
  }
}
```

In the VS Code extension, point `copilotBudget.pricingPath` at a file of the same shape. (See ADR-008.)

> **All cost figures are estimates.** The tool reads local telemetry and applies a local price table;
> it never reconciles against GitHub's authoritative billing.

---

## Developer: build & test the whole repo

```bash
# Go (run in each module: phase-1/session-manager, phase-3, phase-4)
go build ./... && go vet ./... && go test ./... -race

# TypeScript extension
cd phase-2/vscode-extension && npm run compile
```

Acceptance gates per phase live in `evaluation/` (`ACCEPTANCE_CRITERIA.md`, `PHASE3/PHASE4/PHASE7_ACCEPTANCE.md`).
