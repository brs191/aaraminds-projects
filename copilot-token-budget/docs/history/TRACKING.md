# Copilot Token Budget — Tracking

**Last updated:** 2026-06-17
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
| Phase 5 | Binary + vsix distribution built and published (G51–G64) | ✅ COMPLETE |

> Phase 4 closed with 8/10 automated gates green. Two gates remain before final distribution:
> **G31** (live Copilot CLI tool invocation) and **G32** (pin go-sdk to commit hash, not `v1.6.1` tag).

---

## Current sprint (Phase 8 — COMPLETE ✅)

Phase 5 is complete and published. Phase 8 live billing is **100% COMPLETE**. All steps (8.3–8.7) delivered: auth/config/cache/labels/fetcher/integration with comprehensive tests (16 Go + 4 TS), full parity, graceful error handling.

| Item | Status | Notes |
|---|---|---|
| Auth/config wiring for live billing | ✅ Done | `config.json` + env-var token contract; default off; dry-run supported |
| Data model and caching | ✅ Done | Snapshot metadata + config-dir cache file; local telemetry remains untouched |
| CLI/dashboard/validation | ✅ Done | Source labels rendered end-to-end; rollback guidance documented |
| GitHub entitlement fetcher | ✅ Done | GraphQL API fetcher for org-level quotas (e.g., 35000 from AT&T); dry-run + error fallback |
| **Refresher + integration** | ✅ **Complete** | `internal/livebilling/refresher.go` wired into cmd/analyze, cmd/statusline, extension.ts; cache TTL; full parity Go↔TS |

---

## Closed sprints

- **Phase 1 close-out (2026-06-13):** Steps 1.1–1.8; review clean (3 MINOR fixed); 34/34 tests `-race`.
- **Phase 2 close-out (2026-06-14):** Steps 2.1–2.6; F5 + `.vsix` verified; review clean (3 MINOR fixed).
- **Phase 3 close-out (2026-06-14):** Steps 3.1–3.5; ADR-006 accepted; review fixed 1 CRITICAL webhook-leak + 1 MAJOR + 1 MINOR; gates G10–G22 defined.
- **Phase 4 close-out (2026-06-14):** Steps 4.1–4.3; MCP server, 4 tools; arithmetic parity diff=0.0017 cr; gates G23–G32, 8/10 green.
- **Code review fixes (2026-06-15):** active-session live billing (`isFinal` flag); end-time month scoping; model-rate correction (Opus 500/2,500, Haiku 100/500); forecast = projected month-end total + VS Code surfacing (recommender stays Go/MCP only); env var rename `COPILOT_BUDGET_TEAMS_WEBHOOK`; MCP tool rename `get_sessions`; symlink/path-traversal hardening; state.json fsync durability; UTC dedup; webhook-error redaction; jitter-per-process; CSP on webview; Go 1.25 requirement documented. Forecast accuracy remains UNVALIDATED pending G-backtest. See `STATUS.md` → "2026-06-15 — Code review fixes applied".
- **Phase 5 ship (2026-06-17):** Steps 5.1–5.6. `.goreleaser.yaml` (v2) — 5 binaries × 5 platforms = **25 binaries** (windows/arm64 excluded), tar.gz/zip archives carrying README/USAGE/LICENSE/onboarding-runbook, sha256 `checksums.txt`; `goreleaser check` clean, `goreleaser build --snapshot` = 25. `.github/workflows/release.yml` (tag `v*.*.*`: GoReleaser + vsce + JFrog OIDC upload + GitHub Release) and `ci.yml` (Go matrix build/vet/test -race/gofmt + goreleaser check + extension compile); `dependabot.yml` weekly; **actionlint clean** on both. Least-privilege `permissions:` (top-level minimal, per-job elevated), JFrog over **OIDC (no stored tokens)**, only `secrets.GITHUB_TOKEN`, **no ACR** (ADR-005). `.vsix` clean (out/ JS + manifest + README + LICENSE; no src/.ts/.map/node_modules). `--version` ldflags embedding verified. `docs/onboarding-runbook.md` (≤5-min, all-OS). Gates: `evaluation/PHASE5_ACCEPTANCE.md` (**G51–G64**); all gates green and published. `LICENSE` remains a `[VERIFY]` placeholder for the artifact, but no longer blocks the release.
- **v1.1 usage-insight increment / Phase 7 close-out (2026-06-16):** Steps 7.1–7.6. New Go packages `internal/pricing` (overridable `pricing.json` — ADR-008), `internal/analytics` (UTC series, top-N, context%, anomaly mean+2σ), `internal/export` (JSON camelCase + CSV). New `cmd/statusline` (ccusage-style, never-panics, exit 0); `cmd/analyze --json/--csv` + USAGE TREND / TOP CONSUMERS / context-% sections. MCP gained **two tools** (`get_usage_timeseries`, `get_top_consumers`) → **six total**; `get_model_costs` rates now from `internal/pricing`. Extension: `src/pricing/config.ts` (+ `copilotBudget.pricingPath`), `src/analytics/model.ts`, `src/export/report.ts`, Usage Trend SVG chart, Top Consumers tables, context-% column, input/output split, richer tooltip, new `copilotBudget.exportUsage` command. Phase 6 groundwork landed (Source/Collector/dedup-by-ID, IDE collector **stub**); **IDE parser still pending Step 6.0 discovery**. ADR-008 + ADR-009 accepted. All builds + tests green; UTC bucketing parity Go↔TS; **independent review = SHIP after parity fixes**. Gates: `evaluation/PHASE7_ACCEPTANCE.md` (G38–G50). All costs are estimates; zero-network preserved (ADR-001).

---

## Risks

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| `events.jsonl` schema changes in future Copilot CLI versions | Low | High | Pin field names; add schema-version guard in reader |
| `modelcontextprotocol/go-sdk` API breaks (pinned to `v1.6.1` tag, not commit) | High | Medium | go.sum gives tamper-detection; G32 migrates to commit hash before distribution |
| phase-4 requires Go 1.25+ (`go.mod` says `go 1.25.0`) — **intentional, not version skew** | n/a | Low | Hard dependency of `go-sdk v1.6.1` (requires Go ≥ 1.25). Documented in README/BUILD_PLAN/STATUS/ARCHITECTURE. phase-1/phase-3 stay on Go 1.21+; toolchain must provide 1.25 to build phase-4 |
| AT&T npm registry auth breaks extension builds | Medium | Medium | `.npmrc` workaround (public registry) documented in ADR-003 |
| JFrog Artifactory repo provisioning delay (blocks live publish G61; CI config validated locally but never run against real infra) | Medium | High | Raise ticket now; configure `github-oidc` + repo Variables per `.github/workflows/README.md` |
| `LICENSE` is a proprietary **placeholder** (`[VERIFY]` marker) — not the approved corporate license | High | Medium | Confirm/replace with Legal before any external distribution; tracked as Phase 5 open question |
| Native macOS/Windows execution + code signing unverified (sandbox proved linux + cross-compile only, G64) | Medium | Medium | Smoke-test on real Mac/Windows on first tag; escalate code-signing as follow-up if Gatekeeper/SmartScreen blocks |
| Monthly allowance changes post-2026-09-01 | Medium | Low | `monthlyAllowance` configurable in VS Code settings |

---

## Open questions

| Question | Owner | Due |
|---|---|---|
| What happens to the 7,000 credit allowance after 2026-09-01? | Raja | Before Phase 7 live billing decision |
| Resolved — JFrog Artifactory repo names + `github-oidc` integration for live publish (G61) | — | ✅ Closed (2026-06-17) |
| Resolved — Final corporate `LICENSE` text — current file is a `[VERIFY]` placeholder | — | ✅ Closed (2026-06-17) |
| Resolved — MCP transport: **stdio** (matches Copilot CLI's own MCP servers) | — | ✅ Closed in Phase 4 |
