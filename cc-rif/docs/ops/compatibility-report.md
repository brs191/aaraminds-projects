# cc-rif Compatibility Report

## Scope

Contract parity evidence between legacy repo (`repo-intelligence-factory`) and migrated repo (`cc-rif`) for:
1. ingestion HTTP endpoints
2. MCP tool schema/contracts
3. agent-service APIs

Snapshot timestamp: 2026-07-02T07:36:13+05:30

## Deterministic evidence (file-level parity)

| Legacy file | Migrated file | SHA-256 (legacy) | SHA-256 (migrated) | Match |
|---|---|---|---|---|
| `phase-1/ingestion/main.go` | `services/ingestion/main.go` | `8b8ca50f32101c24ff5b00992d0902e24de761145c366675cc9d309a69016b88` | `8b8ca50f32101c24ff5b00992d0902e24de761145c366675cc9d309a69016b88` | YES |
| `phase-1/ingestion/handler/repos.go` | `services/ingestion/handler/repos.go` | `b215e0b2300df4f27b38c6dfa53eff10cae012b270a66f22438a5ddf76c942c0` | `b215e0b2300df4f27b38c6dfa53eff10cae012b270a66f22438a5ddf76c942c0` | YES |
| `phase-1/ingestion/handler/index.go` | `services/ingestion/handler/index.go` | `fc8bba7d7eb051c90e903f627daf1c8b462c6c5bef666dea20a841e390a4e1b5` | `fc8bba7d7eb051c90e903f627daf1c8b462c6c5bef666dea20a841e390a4e1b5` | YES |
| `phase-1/ingestion/handler/status.go` | `services/ingestion/handler/status.go` | `2e34408e6b7997cd081489dbcf959460367bb4289a8d05a62e28517d4ac471fa` | `2e34408e6b7997cd081489dbcf959460367bb4289a8d05a62e28517d4ac471fa` | YES |
| `phase-1/ingestion/handler/webhook.go` | `services/ingestion/handler/webhook.go` | `b3d53216ec419013965000585f4f9ca176689a94e4a14284b58cd3caafb82fbb` | `b3d53216ec419013965000585f4f9ca176689a94e4a14284b58cd3caafb82fbb` | YES |
| `phase-4/mcp-server/tools.schema.json` | `services/mcp-server/tools.schema.json` | `a4eeb4e6943ed887c57c21c78811f12cd211f31314ca3b3c9b7ac7776e6aa3ef` | `a4eeb4e6943ed887c57c21c78811f12cd211f31314ca3b3c9b7ac7776e6aa3ef` | YES |
| `phase-4/agent-service/app.py` | `services/agent-service/app.py` | `6d408ecba5de89d3a79a08604f5ad98b3912ae53763d0b9139081f463a47348c` | `6d408ecba5de89d3a79a08604f5ad98b3912ae53763d0b9139081f463a47348c` | YES |
| `phase-4/agent-service/models.py` | `services/agent-service/models.py` | `880c2e1664bd09a526a9bc0f47c045c64fcfcdedfbc6c6b56b81cdea60aa89e8` | `880c2e1664bd09a526a9bc0f47c045c64fcfcdedfbc6c6b56b81cdea60aa89e8` | YES |

## Area verdicts

| Area | Status | Findings | Shim notes |
|---|---|---|---|
| Ingestion HTTP endpoints | **Pass** | Endpoint surface and handler contracts are byte-identical. | No shim required. |
| MCP tool schema/contracts | **Pass** | `tools.schema.json` is byte-identical; all tool names and required fields match. | No shim required. |
| Agent-service APIs | **Pass** | FastAPI route surface and Pydantic request/response models are byte-identical. | No shim required. |
| Schema + migration idempotency | **Pass (local Postgres)** | Deterministic validator created at `scripts/validate_schema_idempotency.sh`; live pass-1/pass-2 execution passed on 2026-07-02 after pgvector prerequisite was provisioned. | No compatibility shim required. |
| Platform CI workflows (`platform/ci`) | **Partial (remote attempted, runner-blocked)** | Workflows were synced into `.github/workflows`, pushed, and a live `repo-hygiene.yml` run was created on PR #1; execution failed before job start because hosted runners are disabled. `services-ci.yml` is not yet queryable by filename on default branch. | Enable runners, merge workflow files to `main`, then trigger/capture live `services-ci` run evidence. |

## Compatibility shim inventory and decommission plan

Code and contract review result (services, libs, extractors, scripts + this report): **no active temporary compatibility shims/adapters** were found in migrated runtime paths.

| Shim/adapter | Location | Status | Owner | Decommission criteria | Timeline |
|---|---|---|---|---|---|
| None active | N/A | Closed | Compatibility auditor | If any temporary shim is introduced, it must be logged here with a migration issue and removed after byte-level parity and production smoke checks pass without shim dependence. | No decommission work pending now. Any newly introduced shim must be removed before Go/No-Go approval and no later than end of stabilization window. |

## 1) Ingestion HTTP endpoint parity

Extracted endpoint inventory from legacy and migrated `main.go`:
- `GET /healthz`
- `GET /health`
- `POST /repos`
- `POST /repos/{repoID}/index`
- `GET /repos/{repoID}/status`
- `POST /webhook/github`

Key request/response compatibility notes:
- `POST /repos`: request body requires `repo_id`, `clone_url`; responses unchanged (`201` with `repo_id`, `400/409/500` error patterns).
- `POST /repos/{repoID}/index`: optional body field `sha`; response unchanged (`202` with `run_id`, `404/500` failure modes).
- `GET /repos/{repoID}/status`: response shape unchanged (`run_id`, `status`, `sha`, `node_count`, `edge_count`, `started_at`, `completed_at`).
- `POST /webhook/github`: webhook acceptance/ignore/queue response fields unchanged, including `force_reindex`, lane counters, `queued_sha`, and `enqueued_jobs`.
- `GET /healthz` and `GET /health`: unchanged health payload contract (`status`, optional `detail` on degraded).

## 2) MCP tool schema parity

Tool list parity (legacy == migrated):
- `search_code` (required: `repo_id`, `query`)
- `find_callers` (required: `repo_id`, `qualified_name`)
- `impact_analysis` (required: `repo_id`, `changed_entity`)
- `explain_architecture` (required: `repo_id`, `component`)
- `dependency_analysis` (required: `repo_id`, `entity`)

Compatibility note:
- Input schema constraints (types, min/max constraints, required fields) are unchanged.
- No explicit output schema is defined in either legacy or migrated contract; this is parity-preserved.

## 3) Agent-service API parity

Route parity (legacy == migrated):
- `GET /health` → `HealthResponse`
- `POST /explain` → `ExplainResponse`
- `POST /investigate_impact` → `InvestigateImpactResponse`

Model compatibility notes:
- Request models unchanged: `ExplainRequest(repo_id, component)`, `InvestigateImpactRequest(repo_id, changed_entity)`.
- Response models unchanged: citation structure (`tool_name`, `result_excerpt`, `confidence`), explain/impact response fields, and health payload (`status`, `model`, `max_hops`).
- Error mapping parity preserved (`502` for `MCPToolError`, `500` for generic exceptions).

## 4) Schema migration idempotency validation

Validation entrypoint:
- `scripts/validate_schema_idempotency.sh`

Deterministic apply order enforced by script:
1. `data/schema/age_schema.sql`
2. `data/schema/relational_schema.sql`
3. `data/migrations/migration_phase2.sql`
4. `data/migrations/migration_pgvector.sql`
5. `data/migrations/migration_fts.sql`

Execution evidence:
- 2026-07-01:
  - `scripts/validate_schema_idempotency.sh --dry-run` -> **passed**
  - `DATABASE_URL='postgres:///postgres?sslmode=disable' scripts/validate_schema_idempotency.sh` -> **partially passed**
    - `age_schema.sql`, `relational_schema.sql`, `migration_phase2.sql` applied successfully.
    - `migration_pgvector.sql` failed due missing pgvector extension (`vector.control` not found).
- 2026-07-02 (pgvector prerequisite resolved):
  - `psql "$DATABASE_URL" -X -v ON_ERROR_STOP=1 -c "SELECT name, installed_version FROM pg_available_extensions WHERE name='vector';"` -> **passed** (`vector` available).
  - `DATABASE_URL='postgres:///postgres?sslmode=disable' psql "$DATABASE_URL" -X -v ON_ERROR_STOP=1 -f data/migrations/migration_pgvector.sql` -> **passed** (`migration_pgvector.sql: OK` notice).
  - `DATABASE_URL='postgres:///postgres?sslmode=disable' PGOPTIONS='--client-min-messages=warning' scripts/validate_schema_idempotency.sh` -> **passed** (pass-1 + pass-2 re-apply completed without errors).

## Deploy workflow evidence (`platform/deploy/deploy-ingestion.yml`) (2026-07-01)

- Workflow YAML parse check: **passed** (`YAML.safe_load`).
- cc-rif path wiring checks: **passed** for trigger/build references:
  - `services/ingestion/**`, `extractors/core-java/**`, `libs/graphstore/**`
  - `services/ingestion/Dockerfile`, `extractors/core-java/pom.xml`
- Extractor packaging contract: **passed** by running workflow-equivalent command:
  - `mvn -f extractors/core-java/pom.xml -q -DskipTests package`
  - Produced expected artifact: `extractors/core-java/target/rif-extractor-1.0.0-SNAPSHOT-shaded.jar` (~9.3 MB)
- Health probe compatibility: **passed** (`services/ingestion/main.go` exposes both `GET /healthz` and `GET /health`; deploy workflow polls `/healthz`).
- Local CLI prerequisite checks: **passed**
  - `az --version` present (`2.87.0`)
  - `az containerapp update -h` available
  - `docker build --help` available

### Deploy blockers

- Local Docker image build execution is **blocked** by host runtime state:
  - `docker build --file services/ingestion/Dockerfile --tag rif-ingestion:workflow-validation .`
  - Error: `Cannot connect to the Docker daemon ... Is the docker daemon running?`
- Full end-to-end cloud deploy/health poll remains **externally dependent** on configured GitHub Actions secrets/vars, OIDC trust, JFrog registry access, and reachable Azure Container App environment.

### Validation status for PR gate

- `deploy-ingestion.yml` is **validated for structural/path correctness and local command feasibility**.
- Remaining unvalidated scope is limited to external runtime/cloud dependencies.

## CI workflow evidence (`platform/ci/*.yml`) (2026-07-02)

- Workflow YAML parse check: **passed**
  - `ruby -ryaml -e 'YAML.load_file("platform/ci/repo-hygiene.yml"); YAML.load_file("platform/ci/services-ci.yml")'`
- cc-rif path wiring checks: **passed**
  - Verified present paths referenced by workflows: `scripts/repo_hygiene_check.sh`, `services/ingestion`, `services/retriever`, `services/mcp-server`, `services/agent-service`, `services/embedding-service`, `libs`.
- `repo-hygiene.yml` command equivalence: **passed**
  - `bash scripts/repo_hygiene_check.sh`
- `services-ci.yml` Go job command equivalence: **passed**
  - `cd services/ingestion && go test ./...`
  - `cd services/retriever && go test ./...`
  - `cd services/mcp-server && go test ./...`
- `services-ci.yml` Python job command equivalence: **passed**
  - `cd services/agent-service && uv sync --system-certs --quiet && uv run python -m pytest tests/test_agents.py -q && uv run python -m pytest tests/test_e2e.py -q`
    - Result: `2 passed, 1 skipped` and `1 passed`
  - `cd services/embedding-service && uv sync --system-certs --quiet && uv run pytest -q`
    - Result: `11 passed, 1 skipped`

### Remote GitHub Actions evidence (live)

- Workflow sync completed:
  - `platform/ci/repo-hygiene.yml` -> `.github/workflows/repo-hygiene.yml`
  - `platform/ci/services-ci.yml` -> `.github/workflows/services-ci.yml`
- Branch/commit/push:
  - Branch: `chore/remote-ci-evidence-live`
  - Commit: `4920fdf` (`ci: sync migrated workflows`)
  - PR: `https://github.com/aaraminds/cc-rif/pull/1`

CLI evidence:
- `gh workflow list`
  - `Repository Hygiene	active	305696506`
- `gh run list --workflow repo-hygiene.yml`
  - `28560372651` (`pull_request`, `chore/remote-ci-evidence-live`) -> `completed/failure` in `3s`.
  - `28560277063` (`pull_request`, `chore/remote-ci-evidence-live`) -> `completed/failure` in `5s`.
- `gh run view 28560372651`
  - Annotation: `GitHub Actions hosted runners are disabled for this repository. For more information please contact your GitHub Enterprise Administrator.`
- `gh run view 28560372651 --log`
  - `log not found: 84676629618`
- `gh run list --workflow services-ci.yml`
  - `HTTP 404: workflow services-ci.yml not found on the default branch`

### Remote blockers and next steps

1. **Runner policy blocker:** GitHub-hosted runners are disabled, so jobs do not execute and logs are unavailable.
2. **Workflow visibility blocker for `services-ci.yml`:** file exists on PR branch but not on default branch, so filename-based workflow lookup returns 404.

Next steps:
1. Enable GitHub-hosted runners (or configure compatible self-hosted runners).
2. Merge/sync `.github/workflows/services-ci.yml` to `main`.
3. Trigger a PR/push touching `services/**` or `libs/**` (or add `workflow_dispatch`) and capture:
   - `gh run list --workflow services-ci.yml`
   - `gh run view <run-id> --log`
