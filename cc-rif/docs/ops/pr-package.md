# cc-rif Migration PR Package

Last updated: 2026-07-01T19:24:32.004+05:30
Todo: `migration-pr-prep` (PR package preparation)

## Phased cutover readiness (assessment)

- Assessment timestamp: 2026-07-01T19:24:32.004+05:30
- Verdict: **Not Ready**
- Reason: hard blockers remain open (pgvector-dependent full schema idempotency run, remote GitHub Actions evidence, and governance approvals).

| Hard blocker | Required trigger to mark unblocked |
|---|---|
| pgvector prerequisite (`R-003`) | `pg_available_extensions` confirms `vector` is available **and** `DATABASE_URL=... scripts/validate_schema_idempotency.sh` completes both passes through `migration_pgvector.sql` + `migration_fts.sql` with exit code `0`. |
| Remote GitHub Actions evidence (`R-004`) | Workflows are synced to `.github/workflows` on remote, visible in `gh workflow list`, and latest `gh run view --log` for each is successful (including SBOM/vulnerability gate). |
| Governance approvals | Engineering, Operations, and Security approval rows in `docs/ops/cutover-and-rollback-plan.md` contain named approver + decision + date, and all decisions are `APPROVED`. |

## 1) Completed migration scope summary

- Migrated runtime/service layout into cc-rif structure (`services/*`, `libs/*`, `extractors/*`, `data/*`, `platform/*`).
- Rewired Go module replace paths for ingestion/retriever/mcp-server and validated service test lanes.
- Rewired CI/deploy workflow definitions to migrated paths under `platform/ci` and `platform/deploy`.
- Added deterministic schema idempotency validator (`scripts/validate_schema_idempotency.sh`) and validated dry-run + partial live execution.
- Added SBOM + vulnerability scan gate in `platform/ci/services-ci.yml`.
- Confirmed compatibility parity for ingestion HTTP endpoints, MCP schema/contracts, and agent-service API surface.

Primary migration evidence log: `docs/ops/move-log.md`.

## 2) Evidence checklist (attach to PR)

| Evidence item | Status | Link |
|---|---|---|
| Compatibility report (contracts + parity verdicts) | ✅ Available | [`docs/ops/compatibility-report.md`](./compatibility-report.md) |
| Risk register (open/partial risks + mitigations) | ✅ Available | [`docs/ops/risk-register.md`](./risk-register.md) |
| Move log (what moved + rewires/fixes) | ✅ Available | [`docs/ops/move-log.md`](./move-log.md) |
| Cutover and rollback plan | ✅ Available | [`docs/ops/cutover-and-rollback-plan.md`](./cutover-and-rollback-plan.md) |
| Final repository tree snapshot | ✅ Available | [`docs/ops/final-tree.txt`](./final-tree.txt) |

## 3) Open blockers / external prerequisites

These are explicit PR/cutover blockers still requiring external completion:

1. **pgvector prerequisite (hard blocker for full schema idempotency proof)**
   - Current state: live migration validation halts at `data/migrations/migration_pgvector.sql` because pgvector extension files are unavailable.
   - Needed action: provision/enable pgvector on target Postgres, then rerun full live validator.
   - Evidence refs: `docs/ops/compatibility-report.md`, `docs/ops/cutover-and-rollback-plan.md` (Go/No-Go table), `docs/ops/risk-register.md` (`R-003`).

2. **Remote GitHub Actions evidence not yet captured (hard blocker)**
   - Current state: migrated workflow files are under `platform/ci`, not yet executed remotely from `.github/workflows`.
   - Needed action: sync workflows to `.github/workflows`, push, capture `gh run` logs.
   - Evidence refs: `docs/ops/compatibility-report.md`, `docs/ops/cutover-and-rollback-plan.md` (`R-004` / Go-No-Go).

3. **Governance approvals pending (hard blocker)**
   - Current state: Engineering/Ops/Security cutover approvals are placeholders.
   - Needed action: capture named approvers, decisions, and dates in cutover plan.
   - Evidence ref: `docs/ops/cutover-and-rollback-plan.md` (Go/No-Go approvals).

## 4) Attach-ready validation/test outcomes

Use this as PR-ready evidence summary (latest captured outcomes):

| Command / check | Outcome | Evidence pointer |
|---|---|---|
| `bash scripts/repo_hygiene_check.sh` | ✅ Passed | `docs/ops/compatibility-report.md` (CI workflow evidence section) |
| `(cd services/ingestion && go test ./...)` | ✅ Passed | `docs/ops/compatibility-report.md` |
| `(cd services/retriever && go test ./...)` | ✅ Passed | `docs/ops/compatibility-report.md` |
| `(cd services/mcp-server && go test ./...)` | ✅ Passed | `docs/ops/compatibility-report.md` |
| `(cd services/agent-service && uv sync --system-certs --quiet && uv run python -m pytest tests/test_agents.py -q && uv run python -m pytest tests/test_e2e.py -q)` | ✅ Passed (`2 passed, 1 skipped`; `1 passed`) | `docs/ops/compatibility-report.md` |
| `(cd services/embedding-service && uv sync --system-certs --quiet && uv run pytest -q)` | ✅ Passed (`11 passed, 1 skipped`) | `docs/ops/compatibility-report.md` |
| `scripts/validate_schema_idempotency.sh --dry-run` | ✅ Passed | `docs/ops/compatibility-report.md` |
| `DATABASE_URL=... scripts/validate_schema_idempotency.sh` | ⚠️ Partial pass (blocked at `migration_pgvector.sql`) | `docs/ops/compatibility-report.md`, `docs/ops/risk-register.md` |
| CI workflow YAML parse + path rewiring checks (`platform/ci/*.yml`) | ✅ Passed locally | `docs/ops/compatibility-report.md` |
| Remote GitHub Actions workflow runs (`gh run list/view`) | ⛔ Not yet available | `docs/ops/compatibility-report.md`, `docs/ops/cutover-and-rollback-plan.md` |

## 5) Todo and blocker alignment

- `migration-pr-prep`: this package prepared and evidence pointers consolidated.
- Overall migration readiness remains **blocked** by external prerequisites listed in section 3.
