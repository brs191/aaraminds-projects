# Copilot Token Budget — Phased Build Plan

**Status:** Current — Phases 0–4 complete; Phase 5 next.
**Reads with:** `design/ARCHITECTURE.md` (the *what*) and `IMPLEMENTATION_PLAYBOOK.md` (the *how* — source of truth for step results).
**Date:** 2026-06-15 (reconciled to Playbook)

---

## The one decision that shapes everything

Build a **local-first, deterministic credit tracker** — every figure is computed client-side from
the session state files the Copilot CLI already writes. No GitHub API. No proxy. No interception.
No network calls. The LLM narrates findings; the arithmetic is exact.

This constraint drives the forced sequence below and is why the tool works on corporate networks
with no external egress. Mid-June evidence: **8,314.9/7,000 credits (118.8%)** computed in
< 50ms from local files only.

---

## The forced sequence

```
Phase 0 — Spike ──► Phase 1 — Go CLI ──► Phase 2 — VS Code Ext
                                                    │
                                                    ▼
                                         Phase 3 — Teams + Forecast
                                                    │
                                                    ▼
                                         Phase 4 — MCP Server
                                                    │
                                              ┌─────┴─────┐
                                              ▼           ▼
                                         Phase 5 — Distribution
```

Phase 3 gates Phase 4: the MCP server's `get_budget_status` tool reuses the forecasting model
built in Phase 3. Starting Phase 4 first produces a tool with no forecast capability.

---

## Phase 0 — Spike: validate data source · ✅ **COMPLETE** (2026-06-13)

**Goal:** Confirm `~/.copilot/session-state/` contains billing data sufficient for a local tracker.

**Outcomes:**

| Bet | Verdict |
|---|---|
| `events.jsonl` contains billing telemetry | ✅ `session.shutdown.totalNanoAiu` confirmed |
| Active session detection is possible | ✅ `inuse.*.lock` file presence |
| Instruction file overhead is measurable | ✅ `systemTokens` per session confirmed |
| Month-scoped budget is computable | ✅ `session.start.startTime` ISO 8601 |

**Key numbers:**

| Metric | Value |
|---|---|
| June budget consumed (13 days in) | **8,314.9 / 7,000 credits (118.8%)** |
| Instruction overhead per message | 12,238–12,323 tokens |
| Largest single instruction file | `apm0045942-credit-routing-app` = 7,779 tokens |
| Active sessions found | 2 |

**Decisions locked:**

| Decision | Choice |
|---|---|
| Data source | `~/.copilot/session-state/<uuid>/events.jsonl` |
| Billing field | `totalNanoAiu` in `session.shutdown` event |
| Active detection | `inuse.*.lock` presence |
| Implementation language | Go (zero deps, static binary) |

**Risk retired:** Data source viability — the fields are real, stable, and sufficient.

**Artifact:** `phase-0/findings/FINDINGS_MEMO.md`

---

## Phase 1 — Go CLI Tool · ✅ **COMPLETE** (2026-06-13)

**Goal:** Exact credit usage from real data, in the terminal, with zero external dependencies.

**Deliverables:**

| Artefact | Description |
|---|---|
| `cmd/analyze` | 4-section one-shot report: active sessions, history, budget, instruction audit |
| `cmd/dashboard` | 10-second refresh live dashboard + WezTerm tab badge |
| `internal/session/reader.go` | `ReadAll()`, `ReadThisMonth()`, `ReadSince()` — 1MB JSONL buffer |
| `internal/budget/tracker.go` | nanoAIU → credits → dollars; month-scoped |
| `internal/instructions/analyzer.go` | File scanner; dedup by canonical path; workspace-root vs project-scoped |
| `internal/wezterm/badge.go` | OSC escape sequences for WezTerm tab titles |

**Gate verdict:**

| Gate | Criterion | Verdict |
|---|---|---|
| G1 | `go build ./...` exits 0 | ✅ PASS |
| G2 | `cmd/analyze` produces accurate 4-section output | ✅ PASS |
| G3 | Budget scoped to current calendar month only | ✅ PASS |
| G4 | No duplicate instruction files (dedup by canonical path) | ✅ PASS |
| G5 | No panics on live data | ✅ PASS |

**Risk retired:** Core arithmetic correctness, deduplication edge cases, JSONL parsing robustness.

**Artifact:** `phase-1/session-manager/`

---

## Phase 2 — VS Code Extension · ✅ **COMPLETE** (compiled, 2026-06-13)

**Goal:** Status bar badge + sidebar panel + dashboard webview inside VS Code.

**Deliverables:**

| Artefact | Description |
|---|---|
| `src/extension.ts` | Activation entry; 3 commands; auto-refresh; budget alert notifications |
| `src/ui/statusBar.ts` | StatusBarManager; green/yellow/red based on budget % |
| `src/ui/sessionTree.ts` | TreeDataProvider; 3 root nodes: Budget, Active Sessions, Instruction Files |
| `src/ui/dashboardPanel.ts` | Full HTML webview with VS Code theme variables |
| `src/session/reader.ts` | TypeScript port of Go session reader; async JSONL via `readline` |
| `src/budget/tracker.ts` | Credit calculations + status bar text |
| `src/instructions/analyzer.ts` | Instruction file scanner + `severity()` |
| `.vscode/launch.json` | F5 → "Run Extension" → Extension Development Host |

**Gate verdict:**

| Gate | Criterion | Verdict |
|---|---|---|
| G6 | `tsc -p .` exits 0 — all 8 files compile clean | ✅ PASS |
| G7 | F5 launches Extension Development Host | ✅ PASS (verified 2026-06-14) |
| G8 | Status bar shows correct budget badge | ✅ PASS (verified 2026-06-14) |
| G9 | Sidebar shows 3 root nodes | ✅ PASS (verified 2026-06-14) |

> `.vsix` packaging also verified (Step 2.1/testing findings): `@vscode/vsce` added to devDependencies;
> public-registry `.npmrc` workaround for the AT&T proxy.

**Risk retired:** TypeScript compile-time correctness, VS Code API compatibility.

**Artifact:** `phase-2/vscode-extension/`

---

## Phase 3 — Teams Alerts + Budget Forecasting · ✅ **COMPLETE** (2026-06-14) · [S]

**Goal:** Proactive alerts in Microsoft Teams before budget is exhausted; daily burn rate and
month-end forecast.

> **Done:** Steps 3.1–3.5 ✅. ADR-006 accepted. Go alert engine (`phase-3/`) + VS Code wiring
> (`src/alerts/teamsAlert.ts`). Code review fixed **1 CRITICAL** (webhook-URL leak via `*url.Error`),
> 1 MAJOR, 1 MINOR. Acceptance gates **G10–G22** in `evaluation/PHASE3_ACCEPTANCE.md`.
> The exit-criteria items below were met (forecast formulas + dedup validated by automated gates;
> live Teams delivery G19–G21 remain manual integration checks).

**In scope:**

1. **Teams webhook alert** — POST Adaptive Card to Microsoft Teams webhook URL when budget crosses
   60% (warning) and 90% (critical) thresholds. Go `net/http`, no Teams SDK.
2. **Alert deduplication** — store last-alerted threshold in
   `~/.config/copilot-token-budget/state.json`; same threshold fires max once per day.
3. **Daily burn rate** — `creditsThisMonth / daysElapsed` → projected month-end usage.
4. **Month-end forecast** — linear extrapolation; flag if forecast exceeds allowance.
5. **Model routing recommender** — flag models costing > 2× the session average credits/token;
   suggest cheaper alternatives (e.g., Haiku instead of Opus).
6. **Wire into VS Code extension** — Phase 3 alert logic runs in the extension's refresh loop;
   Teams alert fires via the Go binary subprocess OR a native Node.js HTTPS call.

> ⚠️ **Start JFrog Artifactory ticket at Phase 3 kickoff.** IT lead time is 1–2 weeks. Waiting
> until Phase 5 will block distribution. Raise the ticket now.

**Exit criteria:**
- Teams alert fires within one refresh cycle (≤ 30s) of threshold crossing on test data
- Alert deduplication: same threshold does not re-fire same day
- Month-end forecast is a linear projection (projected month-end total = used + dailyBurn × daysRemaining). Accuracy is **UNVALIDATED** — current tests only check the formula against itself. **G-backtest (owner: Raja, run on macOS):** replay a completed month's events as of day N, forecast, compare to the actual month-end total, record the error. No numeric accuracy claim until this backtest runs on real data.
- Model recommender flags the correct model when manually injected session with high cost

**Risk retired:** Proactive awareness — engineers know before they hit the wall.

**Defers:** Webhook URL management UI (Phase 5).

---

## Phase 4 — MCP Server · ✅ **COMPLETE** (2026-06-14) · [S–M]

**Goal:** Copilot CLI can answer "how's my budget?" mid-session via an MCP tool call.

> **Done:** Steps 4.1–4.3 ✅. Go stdio MCP server (`phase-4/`) on `modelcontextprotocol/go-sdk v1.6.1`;
> four tools; reuses the Phase 1/3 data layer. Path-traversal guards, no stdout pollution, race-clean.
> **Arithmetic parity with `cmd/analyze` confirmed: diff = 0.0017 cr.** Acceptance gates **G23–G32**
> in `evaluation/PHASE4_ACCEPTANCE.md` — **8/10 automated gates green**.
> **Open before distribution:** G31 (live Copilot CLI invocation) and G32 (pin go-sdk to a commit hash,
> not the `v1.6.1` tag — tracked as tech debt).

**In scope:**

- **Go MCP server** using official `modelcontextprotocol/go-sdk` (v1.6.1); **stdio transport** (same as
  Copilot CLI's own MCP servers; no HTTP server needed).
- **Go version requirement:** phase-4 **requires Go 1.25+** (`go.mod` declares `go 1.25.0`) because
  `modelcontextprotocol/go-sdk v1.6.1` requires Go ≥ 1.25. This is an intentional hard dependency,
  not version skew — phase-1 and phase-3 build on Go 1.21+.
- **Tools exposed:**

  | Tool | Returns |
  |---|---|
  | `get_budget_status` | `{credits, pct, status, daysLeft, forecast}` |
  | `get_sessions` | `[]{name, credits, contextTokens, model, isActive}` — all sessions this month with `isActive` flag, sorted by credits desc |
  | `get_instruction_overhead` | `[]{name, tokens, severity, savingsPerSession}` |
  | `get_model_costs` | `{model: {inputRate, outputRate, totalCredits}}` |

- **Registration:** `.copilot/mcp.json` in workspace root pointing to the compiled binary.
- **Reuses Phase 1 data layer** — the Go MCP server imports `internal/session`, `internal/budget`,
  `internal/instructions` directly; no duplication.

**Exit criteria:**
- `get_budget_status` returns exact credit total matching `cmd/analyze` output
- Copilot answers "how's my budget?" with a cited, current figure mid-session
- MCP server startup time ≤ 100ms
- Works offline (no network call in any tool handler)

> ⚠️ **`modelcontextprotocol/go-sdk` is pre-1.0.** Pin to an explicit commit hash at start.
> Add an integration test that fails if the tool schema breaks. Named fallback: if go-sdk is
> unstable, ship Phase 5 without MCP and add MCP as a patch after the SDK stabilises.

**Risk retired:** Copilot-native budget awareness — the feedback loop closes.

---

## Phase 5 — Distribution + Onboarding · 🔲 **NEXT** · [S]

**Goal:** Any AT&T engineer can install the tool in ≤ 5 minutes from the Artifactory repo.

> Steps 5.1–5.6 not started. Begin with **5.1 Windows compatibility audit**, then 5.2 CI/CD + JFrog.
> ⚠️ Raise the JFrog Artifactory provisioning ticket **now** — 1–2 week IT lead time.

**In scope:**

- **Go binary** — `goreleaser` cross-compile for Darwin arm64/amd64; publish to JFrog Artifactory.
- **VS Code `.vsix`** — `vsce package`; publish to internal VS Code marketplace or Artifactory.
- **GitHub Actions CI** — `jf` CLI (`jfrog/setup-jfrog-cli`); `JFROG_ACCESS_TOKEN` secret;
  `jf rt upload` for binary; triggered on tag push.
- **`.npmrc` in `phase-2/vscode-extension/`** — `registry=https://registry.npmjs.org` to avoid
  AT&T Artifactory auth blocking CI builds.
- **Onboarding runbook** — step-by-step: install binary, open VS Code extension, configure
  monthly allowance in VS Code settings, set Teams webhook URL.
- **Settings validation** — if `monthlyAllowance` changes post-2026-09-01, it is a config update
  in VS Code settings, not a code change.

**Exit criteria:**
- Engineer follows onboarding runbook; status bar badge shows within 5 minutes
- Go binary installs on Darwin arm64 and amd64 from Artifactory URL
- `.vsix` installs cleanly from file
- CI pipeline runs on tag push; artefacts appear in Artifactory

**Risk retired:** Adoption — the tool reaches engineers beyond the author.

---

## Deliberately deferred

- Cloud sync or multi-user dashboard
- GitHub API calls of any kind
- Modifying or intercepting Copilot traffic (proxy approach — invalidated in Phase 0)
- GitHub Actions budget gate (interesting; add post-v1 if engineers request it)
- Languages beyond Go/TypeScript
- Multi-machine usage aggregation

---

## Risk register

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| `events.jsonl` schema changes in a Copilot CLI update | Low | High | Pin field names; add schema-version guard in `reader.go`; alert on unexpected zero-values |
| AT&T credit allowance changes post-2026-09-01 | Medium | Low | `monthlyAllowance` is VS Code setting; no code change needed |
| JFrog Artifactory repo provisioning delay | Medium | Medium | **Raise ticket at Phase 3 kickoff** (not Phase 5) — 1–2 week IT lead time |
| `modelcontextprotocol/go-sdk` API breaks (pre-1.0) | High | Medium | Pin to commit hash; integration test on tool schema; fallback: defer MCP to patch |
| nanoAIU-to-credit rate changes | Low | High | Rate is a named constant in `tracker.go`; update is a one-line change |

---

## Replan triggers

The plan holds unless one of these fires:

1. `events.jsonl` schema changes in a Copilot CLI release — triggers re-validation of Phase 1 reader
2. AT&T credit allowance drops below 1,000 credits/month post-promo — triggers Phase 3 forecasting threshold recalibration
3. `modelcontextprotocol/go-sdk` has a breaking API change before Phase 4 — triggers fallback decision
4. JFrog Artifactory ticket is blocked or denied — triggers alternative distribution channel evaluation

---

## Definition of done (v1)

An AT&T engineer opens VS Code and sees their exact credit usage in the status bar.
They get a Microsoft Teams alert before they exceed their monthly allowance.
They can ask Copilot "how's my budget?" mid-session and get a cited answer.
