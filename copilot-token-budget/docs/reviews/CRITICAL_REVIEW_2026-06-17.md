# Critical Review — Copilot Token Budget

**Date:** 2026-06-17
**Reviewer:** independent code + test audit (two parallel deep reviews, every CRITICAL/MAJOR finding re-verified against source by hand)
**Scope:** stability and accuracy of the shipped code (`core`, `alerting`, `mcp`, `extension`). Not a security pentest; not a distribution/infra review.

---

## What was run

| Check | Result |
|---|---|
| `go build ./...` — core, alerting, mcp | ✅ clean (mcp on Go 1.25) |
| `go vet ./...` — all 3 modules | ✅ clean |
| `go test ./... -race -count=1` — all 3 modules | ✅ all green, **no data races** |
| `gofmt -l` | ✅ clean |
| `tsc -p ./ --noEmit` (extension, strict) | ✅ clean, zero `any` |
| `npm test` (extension) | ❌ **no `test` script** — fails immediately |
| `goreleaser check` / `build --snapshot` | ✅ 25 binaries |
| `actionlint` | ✅ clean |

### Go test coverage (statement %)

| Package | Cov | Package | Cov |
|---|---|---|---|
| budget | **100.0** | session | 68.0 |
| analytics | **96.4** | platform | 68.8 |
| instructions | 89.7 | alerts | 35.0 |
| pricing | 89.1 | wezterm | 33.3 |
| export | 85.2 | cli | 28.0 |
| forecast | 38.5 | render | 21.9 |
| mcp/tools | 20.1 | | |

The math core is well covered. **The surfaces that users and Copilot actually call — MCP tools (20%), Teams alerts (35%), forecast (38%), terminal render (22%) — are thinly tested.** Forecast accuracy is, by the project's own admission, UNVALIDATED (no backtest).

---

## Headline assessment

The accuracy *engine* is correct: nanoAIU→credit→dollar conversion, the model rate table, UTC month-scoping **in the Go core**, dedup, and the deliberate non-billing of cache/reasoning tokens (billing uses the authoritative upstream `totalNanoAiu`, so no double-counting). Stability is good: race-clean, robust malformed-input handling, atomic+fsync state writes, and disciplined Teams-webhook-URL redaction.

The real risks cluster into three themes, none of which is data-loss but all of which are **silent inaccuracies**:

1. **Local-vs-UTC time handling is inconsistent across surfaces.** The Go core is UTC; the VS Code extension and the Go statusline "today" figure slip into local time. This is the single most impactful accuracy issue because it affects every user not on UTC, and only near day/month boundaries — i.e., exactly when it's hard to notice.
2. **Go ↔ TypeScript have drifted.** The extension is not a faithful mirror: month window, forecast window, and export schema differ. Two tools, two answers.
3. **The TypeScript test suite is theater.** It tests inline reimplementations, not the real code, and `npm test` doesn't even run. It cannot catch any of the above.

---

## Findings

| # | Severity | Location | Issue | Fix |
|---|---|---|---|---|
| 1 | **MAJOR** | `extension/src/session/reader.ts:174-177` | `readThisMonth` filters with **local** `getFullYear()/getMonth()`; Go `core/internal/session/reader.go:291-295` uses **UTC**. A session ending 2026-06-30 23:30 UTC is attributed to June by the CLI/MCP but to a different month by the extension on any non-UTC machine. The TS comment even claims "Mirrors Go ReadThisMonth" — it doesn't. Shifts used credits, used %, status, forecast near month edges. | Use `getUTCFullYear()/getUTCMonth()` on both `bt` and `now`. |
| 2 | **MAJOR** | `extension/src/forecast/model.ts:29-33` | `monthWindow` uses local `getDate()/getMonth()/getFullYear()`; the Go alert (`alerting/cmd/alert/main.go:103-108`) uses UTC. `daysElapsed`/`daysRemaining` → `dailyBurn` and `projectedMonthEndTotal` diverge from the alert near boundaries. | Use `getUTCDate()`; build days-in-month via `new Date(Date.UTC(y, m+1, 0)).getUTCDate()`. |
| 3 | **MAJOR** | `core/internal/render/statusline.go:48` vs `analytics.go` | "Today" credits key is `now.Format("2006-01-02")` in **local** time, but `DailySeries` buckets in **UTC** (and `filterMonth` in the same file correctly uses `.UTC()`). On machines where local date ≠ UTC date, the statusline "today" figure misses its bucket → shows 0 or the wrong day. Internally inconsistent. | `todayKey := now.UTC().Format("2006-01-02")`. |
| 4 | **MAJOR** | `extension/src/export/report.ts:73-95` (CSV) & `:23-48` (JSON) | Export schema drift. Go `SessionsToCSV` emits **14** columns incl. `cacheReadTokens,cacheWriteTokens,reasoningTokens,premiumRequests`; TS emits **10** (those four missing). Go JSON `Report` has top-level `premiumRequests` and `SessionView` has the four token/request fields; TS omits them. Exports from CLI vs extension won't diff; downstream parsers keyed to Go break. | Add the four columns (Go order) + the JSON fields to the TS exporter. |
| 5 | **MAJOR** | `extension` test suite (`src/session/reader.test.ts`, `package.json`) | Tests import **nothing** from `reader.ts`; they assert on inline reimplementations and hardcoded literals. No `test` script, no runner in devDependencies, so `npm test` errors. False confidence — cannot regression-guard findings 1, 2, 4. | Add a `test` script + runner; rewrite tests to call real `reader`/`analytics`/`budget`/`export` functions against fixtures. |
| 6 | MINOR | `core/internal/budget/tracker.go:91` (and TS `budget/tracker.ts`, webview `statusFor`) | Threshold `case pct > 90` → **exactly 90.0% returns WARNING**, but spec/docs say CRITICAL ≥ 90%. Go↔TS agree (parity holds); deviation is from spec. Real incidence is low (needs float pct == 90.000…), but it's a clear off-by-one at the documented alert boundary. | Change to `pct >= 90` on **both** sides; add boundary tests (89.99 / 90.0 / 90.01). |
| 7 | MINOR | `core/internal/budget/tracker.go:58-60` | `totalNano += v` sums `int64` with no overflow guard. A corrupt huge value (or ~1e6+ sessions) wraps negative → negative credits → false `StatusOK`. Not reachable in normal use; defensive gap only. The "large values" test sums `MaxInt64/4` twice and never wraps. | Guard the add (`totalNano > math.MaxInt64 - v`) or accumulate in `float64`/`big.Int`; add a real overflow test. |
| 8 | MINOR | `core/internal/session/reader.go:319-326` | Session-state enumeration uses `os.ReadDir` + `entry.IsDir()` with **no symlink/containment guard**, unlike `mcp/internal/tools/validate.go` (which resolves + contains) and the instructions scanner. A symlink planted under `~/.copilot/session-state/` would be followed and parsed. Low threat (user's own home dir) but inconsistent with the codebase's other path-handling. | `Lstat` and skip symlinks, or `EvalSymlinks` + verify the result stays under the state dir. |
| 9 | MINOR | `core/internal/budget/tracker.go:81-86` | `EstimateInstructionCostPerSession` hardcodes `SonnetInputRate` (300) and ignores `pricing.Config`. If a user overrides the Sonnet input rate via `pricing.json`, the report's instruction-overhead cost still uses 300 — inconsistent with every other pricing-driven figure. | Thread the effective input rate from `cfg` into the estimator, or document it as a fixed-rate estimate. |
| 10 | MINOR | `extension/package.json` (`copilotBudget.alertThresholdWarn`, `alertThresholdCrit`) | Declared as settings but **read nowhere** in `src/`. Changing them does nothing — implies configurability that doesn't exist. | Wire them into `statusFor`, or remove from `package.json`. |
| 11 | MINOR | `extension/src/ui/dashboardPanel.ts` (`savingsRec`, consumer/session credit cells) | Webview re-implements savings-recommendation text differently from `analyzer.ts`/Go; some credit cells use raw `.toFixed(2)` without thousands separators while the rest use `formatCreditsDisplay`. Cosmetic divergence only — **no `/1000` "B"/billions mis-scaling** (that past bug is gone). | Reuse `savingsRecommendation()` + `formatCreditsDisplay()` everywhere. |
| 12 | MINOR | `core/internal/analytics/analytics.go:80-81` | `WeeklySeries` keys by ISO week but derives `Start` (Monday) from the calendar year/month/day; at an ISO/calendar year boundary the two bases disagree (first-write-wins keeps one Start). Cosmetic. | Derive `Start` from the ISO week, or document. |

### Verified correct (no finding)

Conversions (`÷1e9`, `×0.01`); rate table (Sonnet 300/1500, Opus 500/2500, Haiku 100/500, ctx 200k, allowance 7000); unknown-model → sonnet default; pricing.json merge/override/fallback (never hard-fails); cache/reasoning tokens displayed but not re-billed (no double-count); Go month scoping (END time, UTC, year+month); dedup (final wins, else higher nanoAIU; IDE stub contributes zero so it can't drop/dup CLI sessions); malformed-JSONL line skip + missing-dir tolerance; anomaly = mean + 2σ (population, <3-pt guard, strict `>`); Teams webhook redaction (env-only, `execFile` not shell, errors stripped of `*url.Error`); atomic state write (tmp→fsync→rename→fsync dir, 0600); UTC alert dedup; webview CSP + nonce + escaped user strings; `-race` clean across all modules. Go↔TS numeric parity holds for conversions, rate table, analytics bucketing, dedup, instruction estimate, and the absence of the credit mis-scaling regression.

---

## Verdicts

**Accuracy:** The core arithmetic is trustworthy on a UTC machine using the CLI/MCP. It becomes unreliable at day/month boundaries on non-UTC machines, and the VS Code extension can report a different month total and forecast than the CLI for the same data (findings 1–4). Fix the timezone handling and the export schema before claiming "the CLI, extension, and Copilot all agree."

**Stability:** Strong. No panics, no races, graceful degradation on bad/missing data, durable state writes, no webhook leakage. The gaps are defensive (overflow guard, symlink guard) and process-level (the TS test suite is non-functional and several surfaces are under-tested).

**Recommended order of work:** (1) extension UTC fix + statusline today fix [findings 1–3], (2) export schema parity [4], (3) rebuild the TS test suite + add a `test` script [5], (4) threshold `>=` + overflow/symlink hardening [6–8], (5) raise coverage on mcp/tools, alerts, and forecast, and run the long-promised forecast backtest.

> Severity calibration note: the two automated reviews labeled findings 1 and 7 as CRITICAL. On hand re-verification I down-rated them — finding 6/7's real-world incidence is near-zero (exact-float boundary / absurd data volume), and finding 1, while genuinely impactful, is a bounded silent error near month edges rather than a continuous miscalculation. No finding is a ship-stopping data-corruption bug in normal use.
