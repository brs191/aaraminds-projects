# cc-rif Move Log

## 2026-07-01 — FLEETMIG execution batch 1

| Source | Target | Action | Notes |
|---|---|---|---|
| `phase-1/ingestion/**` | `services/ingestion/**` | Copied | Baseline import path updates pending |
| `phase-3/retriever/**` | `services/retriever/**` | Copied | Baseline import path updates pending |
| `phase-4/mcp-server/**` | `services/mcp-server/**` | Copied | Baseline import path updates pending |
| `phase-2/embedding-service/**` | `services/embedding-service/**` | Copied | Python runtime parity pending |
| `phase-4/agent-service/**` | `services/agent-service/**` | Copied | Python runtime parity pending |
| `phase-1/extractor/**` | `extractors/core-java/**` | Copied | Build validation pending |
| `phase-2/extractor/**` | `extractors/spring-java/**` | Copied | Build validation pending |
| `phase-1/graphstore/**` | `libs/graphstore/**` | Copied | Go module path updates pending |
| `phase-5/**` | `libs/phase5/**` | Copied | Introduced to satisfy ingestion incremental dependencies |
| `phase-1/schema/*.sql` | `data/schema/*.sql` | Copied | Canonical ordering pending |
| `phase-2/schema/*.sql` | `data/migrations/*.sql` | Copied | Idempotency validation pending |
| `.github/workflows/services-ci.yml` | `platform/ci/services-ci.yml` | Copied | Path rewiring pending |
| `.github/workflows/repo-hygiene.yml` | `platform/ci/repo-hygiene.yml` | Copied | Path rewiring pending |
| `.github/workflows/deploy-ingestion.yml` | `platform/deploy/deploy-ingestion.yml` | Copied | Path rewiring pending |
| `CODEOWNERS` | `governance/CODEOWNERS` | Copied | - |
| `SECURITY.md` | `governance/SECURITY.md` | Copied | - |
| `CONTRIBUTING.md` | `governance/CONTRIBUTING.md` | Copied | - |
| `RELEASE.md` | `governance/RELEASE.md` | Copied | - |

## 2026-07-01 — Rewire + post-copy fixes (completed so far)

| Date | Area | Old path/scope | New path/scope | Status | Notes |
|---|---|---|---|---|---|
| 2026-07-01 | Go module replace rewires | `phase-1/ingestion/go.mod`: `../graphstore`, `../../phase-5` | `services/ingestion/go.mod`: `../../libs/graphstore`, `../../libs/phase5` | Done | Updated replace targets for migrated monorepo layout. |
| 2026-07-01 | Go module replace rewires | `phase-3/retriever/go.mod`: `../../phase-1/graphstore` | `services/retriever/go.mod`: `../../libs/graphstore` | Done | Retriever now resolves graphstore from `libs/`. |
| 2026-07-01 | Go module replace rewires | `phase-4/mcp-server/go.mod`: `../../phase-1/graphstore`, `../../phase-3/retriever` | `services/mcp-server/go.mod`: `../../libs/graphstore`, `../retriever` | Done | MCP server module rewired to migrated service/lib locations. |
| 2026-07-01 | Agent service e2e path fix | `phase-4/agent-service/tests/test_e2e.py`: `REPO_ROOT / "phase-4" / "mcp-server"` | `services/agent-service/tests/test_e2e.py`: `REPO_ROOT / "services" / "mcp-server"` | Done | Restores fixture MCP binary build path in migrated tree. |
| 2026-07-01 | CI workflow rewires | `.github/workflows/services-ci.yml` paths/`cd` commands under `phase-*` | `platform/ci/services-ci.yml` paths/`cd` commands under `services/*` + `libs/**` | Done | Includes embedding-service test lane in migrated CI. |
| 2026-07-01 | Deploy workflow rewires | `.github/workflows/deploy-ingestion.yml` using `phase-1/ingestion`, `phase-1/extractor`, `phase-1/` build context | `platform/deploy/deploy-ingestion.yml` using `services/ingestion`, `extractors/core-java`, repo-root build context | Done | Trigger paths and build/package references moved to cc-rif layout. |
| 2026-07-01 | Ingestion Dockerfile rewires | `phase-1/ingestion/Dockerfile` with `ingestion/`, `graphstore/`, `extractor/target` | `services/ingestion/Dockerfile` with `services/ingestion/`, `libs/graphstore/`, `extractors/core-java/target` | Done | Docker build context contract updated for repo-root build. |
| 2026-07-01 | Schema validator creation | No deterministic idempotency runner in legacy tree | Added `scripts/validate_schema_idempotency.sh` for ordered schema+migration validation | Done | Supports `--dry-run` static checks and double-apply execution mode. |
| 2026-07-01 | age_schema compatibility fix | `data/schema/age_schema.sql`: `create_vlabel/create_elabel` called with `::cstring` casts | Updated calls to `ag_catalog.create_vlabel('rif', lbl)` and `ag_catalog.create_elabel('rif', lbl)` | Done | Removes cast signature friction for compatibility in migrated execution path. |
| 2026-07-01 | pgvector migration blocker note | N/A | `scripts/validate_schema_idempotency.sh` live run halts at `data/migrations/migration_pgvector.sql` | Blocked | Requires pgvector extension on target Postgres (`vector.control` prerequisite). |
| 2026-07-01 | SBOM + vuln gate updates | Legacy CI had no SBOM/vuln gate stage | Added `supply-chain-gates` job in `platform/ci/services-ci.yml` (`anchore/sbom-action` + `anchore/scan-action`, severity cutoff `high`) | Done | Generates CycloneDX SBOM artifacts and fails on High/Critical vulnerabilities. |

## Validation notes

- `services/ingestion`: `go test ./...` passed after replace-path rewiring.
- `services/retriever`: `go test ./...` passed after replace-path rewiring.
- `services/mcp-server`: `go test ./...` passed after replace-path rewiring.
- `services/embedding-service`: `uv run pytest -q` passed.
- `services/agent-service`: targeted suites (`test_agents.py`, `test_e2e.py`) passed after e2e MCP path update.
