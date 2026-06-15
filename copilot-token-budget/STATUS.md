# Copilot Token Budget — Project Status

**Last updated:** 2026-06-16
**Source of truth:** `IMPLEMENTATION_PLAYBOOK.md` (every step has an agent, a prompt, a test, and a recorded result). This file is reconciled to the Playbook.

---

## Phase Summary

| Phase | Name | Status | Gate |
|---|---|---|---|
| Phase 0 | Spike — validate data source | ✅ **COMPLETE** | All 4 bets confirmed |
| Phase 1 | Go CLI tool | ✅ **COMPLETE** | Steps 1.1–1.8 ✅ · review clean · 34/34 tests |
| Phase 2 | VS Code extension | ✅ **COMPLETE** | Steps 2.1–2.6 ✅ · F5 + .vsix verified 2026-06-14 |
| Phase 3 | Teams alerts + forecasting | ✅ **COMPLETE** | Steps 3.1–3.5 ✅ · review clean · gates G10–G22 defined |
| Phase 4 | MCP server | ✅ **COMPLETE** | Steps 4.1–4.3 ✅ · 8/10 gates green · G31–G32 pending |
| Phase 5 | Distribution + onboarding | 🔲 **NOT STARTED** | Steps 5.1–5.6 |

> Built 2026-06-13 → 2026-06-14 with agent routing for every step.
> See `IMPLEMENTATION_PLAYBOOK.md` for all prompts, results, and gate criteria.

---

## ⚠️ Note on the headline credit figure

Two different month-usage numbers appear across the project history. They are reconciled here:

- **14,144.66 cr (202%)** — the **Phase 0 spike** estimate (2026-06-13). The spike summed `totalNanoAiu` across **all** shutdown events without calendar-month scoping, so it over-counts (it includes sessions that started before June).
- **~8,200–8,550 cr (~118–122%)** — the **correct, month-scoped** figure produced by the Phase 1 tool once `ReadThisMonth()` (year **and** month filter) was added. This is the number to cite.

**Current authoritative figure (latest `cmd/analyze` run, 2026-06-14): ~8,554 cr / 7,000 = ~122% — CRITICAL, OVER BUDGET.** The value climbs through the month as sessions accrue.

---

## 2026-06-15 — Code review fixes applied

Post-Phase-4 code-review hardening. All landed in code (builds + tests green); docs reconciled to match:

- **Active-session live billing** — active sessions now report live/partial billing via a running snapshot; `Session` gained an `isFinal` flag.
- **End-time month scoping** — a session is attributed to a month by its END (shutdown) time, not its start time.
- **Model-rate correction** — authoritative GitHub Copilot rates (1 credit = $0.01): Sonnet 300/1,500, **Opus 500/2,500**, **Haiku 100/500** cr per M tokens (in/out). Prior Opus 1500/7500 and Haiku 80/400 were wrong.
- **Forecast = projected month-end TOTAL** — displayed forecast is now `usedCredits + dailyBurn × daysRemaining` (never hidden on the last day), in Go (phase-3/phase-4) and TS (phase-2). Burn rate + projected total are now **also surfaced in the VS Code dashboard and tree** (`src/forecast/model.ts`). Model-routing recommender stays Go-alert/MCP only. Forecast accuracy remains UNVALIDATED pending a real backtest (G-backtest).
- **Env var rename** — Teams webhook is now `COPILOT_BUDGET_TEAMS_WEBHOOK` (was `COPILOT_TEAMS_WEBHOOK`).
- **MCP tool rename** — `get_active_sessions` → **`get_sessions`** (returns all sessions this month with an `isActive` flag, sorted by credits desc).
- **Symlink / path-traversal hardening** — session-dir reads guarded against symlink escape and traversal.
- **state.json fsync durability** — atomic write now fsyncs before rename for crash durability.
- **UTC dedup** — alert dedup dates computed in UTC to avoid timezone double-fires.
- **Webhook-error redaction** — webhook URL never leaks through `*url.Error` or other error strings.
- **Jitter-per-process** — retry jitter seeded per process to avoid thundering-herd alignment.
- **CSP on webview** — Content-Security-Policy added to the VS Code dashboard webview.
- **Go 1.25 requirement documented** — phase-4 requires Go 1.25+ (hard dependency of `go-sdk v1.6.1`); phase-1/phase-3 stay on Go 1.21+. Intentional, not skew.

---

## 2026-06-16 — v1.1 usage-insight increment (Phase 7) · ✅ SHIPPED

A verified "usage-insight" increment landed across Go and TS. All builds + tests green
in-sandbox; **independent review verdict = SHIP** (after Go↔TS parity fixes). Acceptance:
`evaluation/PHASE7_ACCEPTANCE.md` (gates **G38–G50**).

- **New Go packages (phase-1):** `internal/pricing` (overridable `ConfigDir()/pricing.json`;
  bundled defaults sonnet 300/1,500, opus 500/2,500, haiku 100/500, allowance 7,000, context
  window 200,000; merge-over-defaults + graceful fallback — ADR-008); `internal/analytics`
  (Daily/Weekly/Monthly series with **UTC** bucketing; TopSessions/TopModels/TopProjects;
  ContextWindowPct; AnomalousDays = mean + 2·σ); `internal/export` (Report→JSON camelCase,
  SessionsToCSV, DailyToCSV).
- **New CLI:** `cmd/analyze` gained `--json`/`--csv` + sections "USAGE TREND (last 14 days)"
  (anomaly flags), "TOP CONSUMERS", context-window % on active sessions; `cmd/dashboard` mirrors
  it; **new `cmd/statusline`** (ccusage-style one-liner, credits-based, NO_COLOR-aware, never
  panics, exits 0).
- **MCP (phase-4):** `models.go` now sources rates from `internal/pricing`; **two new tools**
  (`get_usage_timeseries`, `get_top_consumers`) — the server now exposes **six tools**.
- **Extension (phase-2):** `src/pricing/config.ts` (+ setting `copilotBudget.pricingPath`),
  `src/analytics/model.ts`, `src/export/report.ts`; dashboard gained a Usage Trend inline-SVG
  chart, Top Consumers tables, context-% column, input/output split; richer status-bar tooltip
  (today/month/allowance%/burn/projected/context%); **new command `copilotBudget.exportUsage`**
  (JSON/CSV save dialog). Allowance precedence: explicit `copilotBudget.monthlyAllowance` wins,
  else `pricing.allowanceCredits`.
- **Phase 6 groundwork (landed):** a `Source` field (`copilot-cli`/`copilot-ide`), a `Collector`
  interface, a CLI collector, an **IDE collector STUB** (returns nothing), and `ReadAll`
  dedup-by-ID (winner = `IsFinal` else higher `TotalNanoAIU`) — mirrored in Go and TS.
  **The IDE PARSER is still pending the Step 6.0 discovery spike.** Today `ReadAll` ≡ the CLI source.
- **ADRs:** ADR-008 (overridable pricing config) and ADR-009 (usage analytics + source
  abstraction) accepted. Everything stays local-first / zero-network; analytics bucketing is UTC
  on both sides. All cost figures are estimates.

---

## Phase 0 — Spike · ✅ COMPLETE (2026-06-13)

Confirmed `~/.copilot/session-state/` contains billing data sufficient for a local credit tracker.

| Bet | Field | Verdict |
|---|---|---|
| Billing field | `data.totalNanoAiu` in `session.shutdown` | ✅ Confirmed |
| Active session detection | `inuse.<pid>.lock` file presence | ✅ Confirmed |
| Instruction overhead | `data.systemTokens` in `session.shutdown` | ✅ Confirmed |
| Month-scoped timestamp | `timestamp` (ISO 8601 UTC, every event) | ✅ Confirmed |

**Artifacts:** `phase-0/findings/FINDINGS_MEMO.md`, `phase-0/findings/sample_event.json`

---

## Phase 1 — Go CLI tool · ✅ COMPLETE (2026-06-13)

Exact credit usage from real session data in the terminal, zero external dependencies.

- Steps 1.1–1.8 all ✅. Packages: `platform`, `session`, `budget`, `instructions`, `wezterm`, `render`, `cli`. Commands: `cmd/analyze`, `cmd/dashboard`. Launcher: `phase-1/run.sh`.
- Code review (Step 1.8): no CRITICAL/MAJOR; 3 MINOR fixed inline. `go vet` + `go build` clean; **34/34 tests pass with `-race`** (budget 11, instructions 8, platform 4, session 8, wezterm 3).

**Artifact:** `phase-1/session-manager/`

---

## Phase 2 — VS Code extension · ✅ COMPLETE (2026-06-14)

Status bar badge + sidebar tree + dashboard webview inside VS Code.

- Steps 2.1–2.6 all ✅. `tsc` strict, zero `any`, zero runtime deps, publisher `att-internal`.
- F5 Extension Development Host verified; `.vsix` packaging verified (`@vscode/vsce` added to devDependencies; public-registry `.npmrc` workaround for AT&T proxy).
- Code review (Step 2.6): no CRITICAL/MAJOR; 3 MINOR fixed inline.

**Artifact:** `phase-2/vscode-extension/`

---

## Phase 3 — Teams alerts + forecasting · ✅ COMPLETE (2026-06-14)

Proactive Teams alerts; daily burn rate; month-end forecast; model routing recommender.

> **Forecast now in VS Code (2026-06-15):** daily burn rate and the projected month-end total (`used + dailyBurn × daysRemaining`) are now surfaced in the VS Code dashboard and tree (`phase-2/.../src/forecast/model.ts`) — previously Teams/MCP-only. The **model-routing recommender remains Go-alert-binary + MCP only** (not in VS Code). Forecast accuracy is a linear projection and remains **UNVALIDATED** pending a real backtest (see `evaluation/PHASE3_ACCEPTANCE.md` → G-backtest).

- Steps 3.1–3.5 all ✅. ADR-006 (config storage) accepted. Go alert engine (`phase-3/`): `alerts/teams.go`, `alerts/dedup.go`, `forecast/model.go`, `cmd/alert`. Wired into the VS Code extension (`src/alerts/teamsAlert.ts`).
- Webhook URL via `COPILOT_BUDGET_TEAMS_WEBHOOK` env var only; atomic `state.json`; jitter retry; division-by-zero guard.
- Code review (Step 3.4): **1 CRITICAL fixed** (webhook URL leak through `*url.Error`), 1 MAJOR + 1 MINOR fixed. `go test -race` and `tsc` clean.
- Acceptance: `evaluation/PHASE3_ACCEPTANCE.md` — gates **G10–G22** defined.

**Artifacts:** `phase-3/`, `phase-2/vscode-extension/src/alerts/teamsAlert.ts`, `design/adr/ADR-006-config-storage.md`

---

## Phase 4 — MCP server · ✅ COMPLETE (2026-06-14)

Copilot CLI can answer "how's my budget?" mid-session via MCP tool call.

- Steps 4.1–4.3 all ✅. Go stdio MCP server (`phase-4/`) using `modelcontextprotocol/go-sdk v1.6.1`. Four tools at Phase 4 close: `get_budget_status`, `get_sessions`, `get_instruction_overhead`, `get_model_costs`. Reuses the Phase 1/3 data layer directly. *(v1.1 added `get_usage_timeseries` + `get_top_consumers` → six tools — see the 2026-06-16 section above.)*
- **Go version:** phase-4 **requires Go 1.25+** (`go.mod` declares `go 1.25.0`) — a hard dependency of `go-sdk v1.6.1`, not version skew. phase-1/phase-3 build on Go 1.21+.
- Path-traversal guards (absolute + within home), no stdout pollution, no panics, race-clean. **Arithmetic parity with `cmd/analyze` confirmed: diff = 0.0017 cr.**
- Acceptance: `evaluation/PHASE4_ACCEPTANCE.md` — gates **G23–G32**; **8/10 automated gates green**.
- **Open before distribution:** G31 (Copilot CLI invokes the 4 tools live — needs `~/bin/copilot-budget-mcp` build) and G32 (pin go-sdk to a commit hash, not the `v1.6.1` semver tag — tracked as tech debt).

**Artifacts:** `phase-4/`, `.copilot/mcp.json`

---

## Phase 5 — Distribution + onboarding · 🔲 NOT STARTED

Goal: any AT&T engineer installs the tool in ≤ 5 minutes from JFrog Artifactory.

Steps 5.1–5.6 (all 🔲): Windows compatibility audit → CI/CD + JFrog distribution → `.vsix` distribution hardening → onboarding runbook → final code review → Phase 5 eval criteria.

> ⚠️ Raise the **JFrog Artifactory provisioning ticket now** if not already open — 1–2 week IT lead time.

---

## Next action

Execute **Step 5.1 — Windows compatibility audit** using `aara-project-builder` — prompt in `IMPLEMENTATION_PLAYBOOK.md`.
Also close out Phase 4 tail: **G31** (live Copilot CLI tool invocation) and **G32** (commit-hash pin) before final distribution.
