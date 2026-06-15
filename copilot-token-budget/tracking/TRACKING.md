# Copilot Token Budget — Tracking

**Last updated:** 2026-06-16
**Reconciled to:** `IMPLEMENTATION_PLAYBOOK.md`

---

## Phase gate summary

| Phase | Gate | Status |
|---|---|---|
| Phase 0 | Data confirmed in `events.jsonl` | ✅ CLOSED |
| Phase 1 | Tool produces accurate output from real data | ✅ CLOSED |
| Phase 2 | Extension compiles; F5 launches dev host; `.vsix` packages | ✅ CLOSED |
| Phase 3 | Teams alert fires; forecast formulas defined (G10–G22) | ✅ CLOSED |
| Phase 4 | Copilot answers "how's my budget?" via MCP; parity confirmed | ✅ CLOSED (8/10 gates) |
| Phase 5 | Binary + vsix distributed via Artifactory | 🔲 NOT STARTED |

> Phase 4 closed with 8/10 automated gates green. Two gates remain before final distribution:
> **G31** (live Copilot CLI tool invocation) and **G32** (pin go-sdk to commit hash, not `v1.6.1` tag).

---

## Current sprint (Phase 5 kickoff)

| Item | Status | Notes |
|---|---|---|
| Step 5.1 — Windows compatibility audit | 🔲 Pending | `aara-project-builder`; verify `filepath.Join`/`path.join`, no `ioutil`, `.exe` suffix |
| Step 5.2 — CI/CD pipeline + JFrog distribution | 🔲 Pending | `azure-ops` skill; `release.yml` + `.goreleaser.yml` |
| Raise JFrog Artifactory provisioning ticket | 🔲 Pending | **Do now** — 1–2 week IT lead time |
| Phase 4 tail: G31 live MCP invocation | 🔲 Pending | Build `~/bin/copilot-budget-mcp`; verify in Copilot CLI session |
| Phase 4 tail: G32 commit-hash pin | 🔲 Pending | Replace `v1.6.1` tag with explicit commit hash in `phase-4/go.mod` |

---

## Closed sprints

- **Phase 1 close-out (2026-06-13):** Steps 1.1–1.8; review clean (3 MINOR fixed); 34/34 tests `-race`.
- **Phase 2 close-out (2026-06-14):** Steps 2.1–2.6; F5 + `.vsix` verified; review clean (3 MINOR fixed).
- **Phase 3 close-out (2026-06-14):** Steps 3.1–3.5; ADR-006 accepted; review fixed 1 CRITICAL webhook-leak + 1 MAJOR + 1 MINOR; gates G10–G22 defined.
- **Phase 4 close-out (2026-06-14):** Steps 4.1–4.3; MCP server, 4 tools; arithmetic parity diff=0.0017 cr; gates G23–G32, 8/10 green.
- **Code review fixes (2026-06-15):** active-session live billing (`isFinal` flag); end-time month scoping; model-rate correction (Opus 500/2,500, Haiku 100/500); forecast = projected month-end total + VS Code surfacing (recommender stays Go/MCP only); env var rename `COPILOT_BUDGET_TEAMS_WEBHOOK`; MCP tool rename `get_sessions`; symlink/path-traversal hardening; state.json fsync durability; UTC dedup; webhook-error redaction; jitter-per-process; CSP on webview; Go 1.25 requirement documented. Forecast accuracy remains UNVALIDATED pending G-backtest. See `STATUS.md` → "2026-06-15 — Code review fixes applied".
- **v1.1 usage-insight increment / Phase 7 close-out (2026-06-16):** Steps 7.1–7.6. New Go packages `internal/pricing` (overridable `pricing.json` — ADR-008), `internal/analytics` (UTC series, top-N, context%, anomaly mean+2σ), `internal/export` (JSON camelCase + CSV). New `cmd/statusline` (ccusage-style, never-panics, exit 0); `cmd/analyze --json/--csv` + USAGE TREND / TOP CONSUMERS / context-% sections. MCP gained **two tools** (`get_usage_timeseries`, `get_top_consumers`) → **six total**; `get_model_costs` rates now from `internal/pricing`. Extension: `src/pricing/config.ts` (+ `copilotBudget.pricingPath`), `src/analytics/model.ts`, `src/export/report.ts`, Usage Trend SVG chart, Top Consumers tables, context-% column, input/output split, richer tooltip, new `copilotBudget.exportUsage` command. Phase 6 groundwork landed (Source/Collector/dedup-by-ID, IDE collector **stub**); **IDE parser still pending Step 6.0 discovery**. ADR-008 + ADR-009 accepted. All builds + tests green; UTC bucketing parity Go↔TS; **independent review = SHIP after parity fixes**. Gates: `evaluation/PHASE7_ACCEPTANCE.md` (G38–G50). All costs are estimates; zero-network preserved (ADR-001).

---

## Risks

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| `events.jsonl` schema changes in future Copilot CLI versions | Low | High | Pin field names; add schema-version guard in reader |
| `modelcontextprotocol/go-sdk` API breaks (pinned to `v1.6.1` tag, not commit) | High | Medium | go.sum gives tamper-detection; G32 migrates to commit hash before distribution |
| phase-4 requires Go 1.25+ (`go.mod` says `go 1.25.0`) — **intentional, not version skew** | n/a | Low | Hard dependency of `go-sdk v1.6.1` (requires Go ≥ 1.25). Documented in README/BUILD_PLAN/STATUS/ARCHITECTURE. phase-1/phase-3 stay on Go 1.21+; toolchain must provide 1.25 to build phase-4 |
| AT&T npm registry auth breaks extension builds | Medium | Medium | `.npmrc` workaround (public registry) documented in ADR-003 |
| JFrog Artifactory repo provisioning delay | Medium | Medium | Raise ticket at Phase 5 kickoff (now) |
| Monthly allowance changes post-2026-09-01 | Medium | Low | `monthlyAllowance` configurable in VS Code settings |

---

## Open questions

| Question | Owner | Due |
|---|---|---|
| What happens to the 7,000 credit allowance after 2026-09-01? | Raja | Before Phase 5 distribution |
| JFrog Artifactory repo name for binary distribution? | Raja | Before Step 5.2 |
| Resolved — MCP transport: **stdio** (matches Copilot CLI's own MCP servers) | — | ✅ Closed in Phase 4 |
