# VRIA Implementation (Phase 0 + core build)

Go implementation of the VRIA spec pack (v1.3). Pure, deterministic core —
no I/O in scoring, no LLM calls anywhere in this module.

## Layout

| Path | Implements |
|---|---|
| `migrations/` | `contracts/19` DDL, extracted verbatim from the doc (regenerate, don't hand-edit) |
| `internal/enums` | `contracts/17` canonical enums |
| `internal/scoring` | `contracts/20` — component formulas (§3a), caps (§4), state mapping (§5), recommendations (§6), sustainment (§7) |
| `internal/approval` | `contracts/18` — request + artifact state machines, publication gate, append-only decision log |
| `internal/registry` | `gate-a-value/02` — status normalization mapping table, staged import (reject-don't-guess) |
| `internal/evidence` | `gate-b-behavior/06` — freshness cadence, sustainment scheduling, conflict resolution |
| `internal/hypothesis` | Epic 2 — hypothesis drafts, 09 §3.4 field validation, approval-gated commit, versioning |
| `internal/assessment` | Epic 4 — assessment generation over the scoring engine; sustainment scheduler (P4.2) |
| `internal/scorecard` | Epic 5 — scorecard lifecycle: draft, approval, publish (GE-007 gate), supersede, invalidate |
| `internal/mcpserver` | P3.1 — get_metric_snapshot + search_evidence_documents servers with reference adapters |
| `cmd/vria-mcp-metrics`, `cmd/vria-mcp-evidence` | MCP server entrypoints (VRIA_METRICS_CSV / VRIA_EVIDENCE_DIR) |
| `goldeneval/volume/` | P7.1 — 62-record labeled volume dataset + 28 normalization cases; accuracy gates in `volume_test.go` |
| `agentprompt/` | P7.3 — VRIA agent system prompt v1.0 (validated by the golden harness) |
| `ci/` | P7.1 — GitHub Actions release-gate workflow, ready to copy to `.github/workflows/` |
| `internal/api` | `contracts/21` registry + hypothesis + approval-decision slices — `/api/v1` handlers, error envelope, principal boundary |
| `cmd/vria-api` | Local/dev HTTP entrypoint (`VRIA_ADDR`, default `:8080`) |
| `goldeneval/` | `gate-b-behavior/07` — GE-001..GE-015 as executable tests; `...Critical` tests are release-blocking |

## Run

```sh
go test ./...
```

Release gate (per `07` §3): every `Test...Critical` must pass — CI blocks
merge on failure (`ci/github-actions-release-gate.yml`). Percentage gates run
against the 62-record volume dataset: schema validation 100%, value-state
accuracy 100%, recommendation accuracy 100%, normalization accuracy 100%
(all above the 90/95% thresholds).

Migrations have paired `.down.sql` rollbacks. Live apply/rollback verification
requires PostgreSQL (not in this sandbox) — wire into CI per P7.1.

## Not in this module (remaining per prompts.md)

- PostgreSQL `Store` implementation (interface ready in `internal/registry`); live migration apply/rollback in CI
- Azure Service Bus event emission (`contracts/21` §4-5) — event structs exist, transport pending
- React dashboard — P6.x (`design:*` + `frontend-engineering`)
- Real metric/document source adapters behind the MCP servers (CSV/file reference adapters shipped)
- Phases 8-9 are process: pilot execution, production readiness run
