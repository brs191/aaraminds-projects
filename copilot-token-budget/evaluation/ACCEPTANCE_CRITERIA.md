# Copilot Token Budget — Evaluation Criteria

**Status:** Phases 1–2 criteria defined and met. Phases 3–4 criteria defined.

---

## Phase 1 Gate — Go CLI Tool

| Criterion | Target | Result |
|---|---|---|
| `go build ./...` exits 0 | No compile errors | ✅ PASS |
| `go run ./cmd/analyze` produces output | 4 sections rendered | ✅ PASS |
| Credit total matches nanoAIU arithmetic | ≤ 1% rounding error | ✅ PASS |
| Month-scoping correct | Only current-month sessions | ✅ PASS |
| No duplicate instruction files | Dedup by canonical path | ✅ PASS |
| No panics on live data | Zero runtime errors | ✅ PASS |

---

## Phase 2 Gate — VS Code Extension

| Criterion | Target | Result |
|---|---|---|
| `tsc -p .` exits 0 | All 8 files compile clean | ✅ PASS |
| F5 launches Extension Development Host | No activation error | 🔲 To verify |
| Status bar shows budget badge | Correct credit display | 🔲 To verify |
| Sidebar shows 3 root nodes | Budget / Active Sessions / Instruction Files | 🔲 To verify |
| Dashboard webview opens | `Copilot Budget: Open Dashboard` command works | 🔲 To verify |
| Budget alert fires | Notification at 90% threshold | 🔲 To verify |

---

## Phase 3 Gate — Teams Alerts + Forecasting

| Criterion | Target |
|---|---|
| Teams alert fires at 60% threshold | Within next refresh cycle (30s) |
| Teams alert fires at 90% threshold | Within next refresh cycle (30s) |
| Alert deduplication | Same threshold fires max once per day |
| Burn rate calculation | Matches manual nanoAIU sum / days |
| Month-end forecast | Linear projection (projected total = used + dailyBurn × daysRemaining). Accuracy **UNVALIDATED** — automated tests only check the formula against itself. Real error pending G-backtest below. |
| Model routing recommendation | Flags models > 2x the session average cost/token |

**G-backtest — forecast accuracy (NOT YET RUN).** Owner: Raja. Run on macOS.
Replay a *completed* month's session events as of day N (truncate the event stream at
day N), compute the linear forecast (used + dailyBurn × daysRemaining), then compare
to the *actual* recorded month-end total. Record the percent error for several values
of N. Until this backtest is run on real data, the forecast carries **no validated
accuracy claim** — the existing G14 only checks the formula's arithmetic against itself.

---

## Phase 4 Gate — MCP Server

| Criterion | Target |
|---|---|
| `get_budget_status` returns correct credits | Matches CLI tool output |
| `get_sessions` returns all month sessions | All sessions this month with `isActive` flag (matches `inuse.*.lock` presence), sorted by credits desc |
| `get_instruction_overhead` returns file list | Matches `cmd/analyze` audit section |
| Copilot can query budget mid-session | "How's my budget?" returns a cited answer |
| MCP server startup time | ≤ 100ms |

---

## Success metrics (final product)

| Metric | Target |
|---|---|
| Status bar accuracy vs Copilot billing | ≤ 1% error |
| VS Code extension activation time | ≤ 50ms |
| Teams alert latency | ≤ 30s from threshold crossing |
| Month-end forecast | Linear projection; accuracy **UNVALIDATED** pending G-backtest (no measured error to claim) |
| Onboarding time (zero to badge) | ≤ 5 minutes |
