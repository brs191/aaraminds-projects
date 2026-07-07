# RIF Migration Prompt

**Tag:** `[RIFMIG]`

```text
You are executing a brownfield migration program in `/repo-intelligence-factory`.

# Mission
Create a **new** project directory `cc-rif/` (side-by-side in the current repo root), then incrementally migrate from the current phase-based layout to this target capability-based layout without breaking existing behavior:

cc-rif/
  services/{ingestion,retriever,mcp-server,embedding-service,agent-service}
  extractors/{core-java,spring-java}
  libs/{graphstore,contracts}
  data/{schema,migrations,seeds}
  platform/{infra/terraform,ci,deploy}
  tests/{unit,integration,e2e,perf,security,fixtures}
  docs/{architecture,adr,runbooks,ops}
  governance/{CODEOWNERS,SECURITY.md,CONTRIBUTING.md,RELEASE.md}

# Hard guardrails (non-negotiable)
1. Do not destructively rewrite or delete existing phase-* project content.
2. No invented functionality, no speculative features, no fake metrics.
3. No broad `catch (Exception)`, `except Exception: pass`, silent fallback, or swallow-and-continue patterns.
4. No destructive git commands (`reset --hard`, `clean -fd`, force-push, history rewrite).
5. Preserve existing endpoint contracts and payload semantics unless explicitly migrated with compatibility shims.
6. Keep deterministic extraction and graph-first source-of-truth architecture.
7. Keep Azure-primary + Terraform AzureRM direction; GitHub Actions with Azure OIDC; JFrog Artifactory image publishing; no cloud/tool drift.
8. Mark any uncertain version/parameter claim as `[VERIFY]`.
9. Execute AI-agent-driven refactor tasks in sandboxed environments with restricted filesystem/network scope whenever possible.
10. All infrastructure/runtime desired state must be Git-declared and reconciled via GitOps-compatible workflows.

# Migration strategy
- Work in small, reversible phases.
- Prefer copy + adapt + parity tests, then switch references.
- Keep old and new paths runnable in parallel until cutover phase.
- Record every move/edit in `cc-rif/docs/ops/move-log.md`.

# Operating model (GitOps + security)
- Treat Git as the single source of truth for deployable app and infrastructure state.
- Prefer pull-based reconciliation for environment convergence where platform supports it.
- Require immutable audit trail for every deployment-affecting change (commit + PR + workflow run).
- Produce SBOM artifacts for migrated services and enforce vulnerability scan gates in CI before cutover.

# Definitions (avoid ambiguity)
- **Contract parity:** request/response/tool-schema behavior in `cc-rif` matches legacy behavior unless an approved compatibility shim is documented.
- **Migration complete (phase):** phase tasks done, acceptance criteria passed, and required evidence artifacts updated.
- **Cutover-ready:** compatibility report complete, rollback validated, go/no-go approvals captured.
- **Compatibility shim:** temporary adapter preserving legacy contract while internal path/module changes.

# Explicit move-map examples (apply this pattern repository-wide)
- `phase-1/ingestion/**` -> `cc-rif/services/ingestion/**`
- `phase-3/retriever/**` -> `cc-rif/services/retriever/**`
- `phase-4/mcp-server/**` -> `cc-rif/services/mcp-server/**`
- `phase-2/embedding-service/**` -> `cc-rif/services/embedding-service/**`
- `phase-4/agent-service/**` -> `cc-rif/services/agent-service/**`
- `phase-1/extractor/src/**` -> `cc-rif/extractors/core-java/src/**`
- `phase-2/extractor/src/**` -> `cc-rif/extractors/spring-java/src/**`
- `phase-1/graphstore/**` -> `cc-rif/libs/graphstore/**`
- shared API contracts from service code -> `cc-rif/libs/contracts/**`
- `phase-1/schema/*.sql` + `phase-2/schema/*.sql` + `database/phase-*/schema/*.sql` -> `cc-rif/data/schema/` and `cc-rif/data/migrations/`
- `.github/workflows/services-ci.yml` -> `cc-rif/platform/ci/services-ci.yml`
- `.github/workflows/repo-hygiene.yml` -> `cc-rif/platform/ci/repo-hygiene.yml`
- `.github/workflows/deploy-ingestion.yml` -> `cc-rif/platform/deploy/deploy-ingestion.yml`
- `phase-1/infra/*` + `phase-2/infra/*` -> `cc-rif/platform/infra/terraform/` (convert non-Terraform IaC only via explicit compatibility notes)
- `doc/*.md` + architecture/adrs/findings across phases -> `cc-rif/docs/{architecture,adr,runbooks,ops}/`
- root governance files -> `cc-rif/governance/`

# Implementation phases with acceptance criteria

## Phase 0 — Baseline capture
Tasks:
- Inventory current tree, services, endpoints, schema files, workflows, test suites.
- Create `cc-rif/docs/ops/architecture-map.md` and initial `move-log.md`.

Acceptance:
- Baseline inventory committed.
- No code behavior changes.

## Phase 1 — Scaffold new project
Tasks:
- Create full `cc-rif/` directory skeleton.
- Add placeholder README files in each top-level area describing scope and ownership.

Acceptance:
- Target tree exists.
- Legacy tree untouched.

## Phase 2 — Governance + docs preservation
Tasks:
- Copy and adapt `CODEOWNERS`, `SECURITY.md`, `CONTRIBUTING.md`, `RELEASE.md` into `cc-rif/governance/`.
- Migrate operational and architecture docs into `cc-rif/docs/...` with source references.
- Preserve evidence artifacts (findings, memos, reports).

Acceptance:
- Governance files present and valid.
- No lost docs/evidence; mapping table included.

## Phase 3 — Service and library migration (incremental)
Tasks:
- Migrate services one by one in this order: ingestion -> retriever -> mcp-server -> embedding-service -> agent-service.
- Migrate `graphstore` and extract contract interfaces into `libs/contracts`.
- Keep import/module compatibility shims where needed.

Acceptance (per service):
- Builds in new location.
- Existing API contract parity verified.
- Old path still runnable or shimmed.

## Phase 4 — Data unification
Tasks:
- Consolidate schema and migration SQL under `data/schema` + `data/migrations`.
- Add deterministic migration order and idempotency checks.
- Preserve current schema behavior and extension assumptions.

Acceptance:
- Unified schema layout complete.
- Migration test path documented and passing.
- No schema drift vs baseline.

## Phase 5 — Platform structure
Tasks:
- Establish `platform/infra/terraform`, `platform/ci`, `platform/deploy`.
- Port CI/deploy workflows with updated paths.
- Keep Azure OIDC + JFrog constraints explicit.

Acceptance:
- CI workflows reference `cc-rif` paths.
- Infra directory has valid Terraform scaffold and validation instructions.

## Phase 6 — Test pyramid reorganization
Tasks:
- Rehome tests into `tests/{unit,integration,e2e,perf,security,fixtures}`.
- Keep language-specific test runners intact.
- Add migration parity tests (old vs new behavior checks).

Acceptance:
- Tests organized by pyramid.
- Critical service tests passing from new paths.
- Fixtures preserved.

## Phase 7 — Compatibility, cutover, rollback readiness
Tasks:
- Produce compatibility report for endpoints, schemas, and workflows.
- Define cutover sequence and rollback triggers/commands.
- Keep legacy phase paths until final approval gate.

Acceptance:
- Cutover plan and rollback plan approved artifacts.
- No breaking changes without shim + documentation.

# Required validation commands (run and report output)
Run baseline (legacy) and migrated (`cc-rif`) equivalents where available.

## Repo hygiene
- `bash scripts/repo_hygiene_check.sh`

## Go services (legacy baseline)
- `cd phase-1/ingestion && go test ./...`
- `cd phase-3/retriever && go test ./...`
- `cd phase-4/mcp-server && go test ./...`

## Python services (legacy baseline)
- `cd phase-2/embedding-service && uv sync --system-certs --quiet && uv run pytest -q`
- `cd phase-4/agent-service && uv sync --system-certs --quiet && uv run python -m pytest tests/test_agents.py -q && uv run python -m pytest tests/test_e2e.py -q`

## Migration parity checks
- Run corresponding `go test ./...` and `pytest` commands from `cc-rif/services/*`.
- Compare endpoint contract snapshots before/after migration.
- Verify schema migration checks against unified `data/` layout.

## Supply-chain/security checks
- Generate SBOM for each migrated service (`cc-rif/services/*`) and publish as CI artifact.
- Run dependency vulnerability scans and fail on High/Critical findings unless explicitly risk-accepted in `risk-register.md`.

# Deliverables (must exist before completion)
1. `cc-rif/docs/ops/architecture-map.md`
2. `cc-rif/docs/ops/move-log.md` (old path -> new path, commit-by-commit)
3. `cc-rif/docs/ops/compatibility-report.md` (API/schema/workflow parity + shims)
4. `cc-rif/docs/ops/risk-register.md` (risk, impact, mitigation, owner, status)
5. `cc-rif/docs/ops/cutover-and-rollback-plan.md`
6. `cc-rif/docs/ops/final-tree.txt` (final directory tree snapshot)

# Rollback strategy
- Use phased rollback by capability (service/data/platform), not all-at-once.
- Keep legacy phase paths as fallback during cutover window.
- Rollback trigger examples: contract mismatch, failed migration idempotency, failed health checks, CI gate regressions.
- For each phase, document:
  - trigger condition
  - exact rollback steps
  - data safety considerations
  - verification commands post-rollback

# Go/No-Go gate (mandatory before cutover)
- Record explicit approvals in `cc-rif/docs/ops/cutover-and-rollback-plan.md` from:
  - Engineering owner
  - Operations owner
  - Security owner
- Do not proceed to cutover if any of the following are open:
  - unresolved High/Critical security findings
  - failed contract parity checks
  - missing rollback verification evidence
  - missing ownership sign-off

# Execution mode
Proceed phase-by-phase. After each phase: commit scoped changes, run validations, and update deliverables. Do not skip failed checks. Stop and report blockers with concrete remediation options.
```
