# cc-rif Cutover and Rollback Plan

## Phased cutover readiness verdict

- Assessment timestamp: 2026-07-01T19:24:32.004+05:30
- Verdict: **Not Ready**
- Decision: phased cutover **must not** start yet.

### Why Not Ready (hard blockers) and exact unblock conditions

| Blocker | Current status | Exact unblock condition (trigger to clear blocker) |
|---|---|---|
| `R-003` pgvector prerequisite for full schema idempotency proof | `migration_pgvector.sql` cannot run to completion without pgvector | 1) `psql "$DATABASE_URL" -X -v ON_ERROR_STOP=1 -c "SELECT name, installed_version FROM pg_available_extensions WHERE name='vector';"` returns `vector` with non-null `installed_version`; 2) `DATABASE_URL="$DATABASE_URL" scripts/validate_schema_idempotency.sh` exits `0` and pass-1/pass-2 complete through `migration_pgvector.sql` and `migration_fts.sql`. |
| `R-004` Remote GitHub Actions evidence missing | Workflows validated locally only; no remote runs captured | 1) `platform/ci/repo-hygiene.yml` and `platform/ci/services-ci.yml` are present in `.github/workflows/` on remote branch; 2) `gh workflow list` shows both workflows; 3) latest `gh run view <run-id> --log` for both workflows is successful and includes green service test lanes + SBOM/Anchore gate. |
| Governance approvals not recorded | Engineering/Ops/Security rows are placeholders (`PENDING`) | Go/No-Go table in this file is updated with named approver identity, explicit `APPROVED`/`REJECTED` decision, and approval date for Engineering, Operations, and Security. Cutover can proceed only if all three are `APPROVED`. |

### Trigger condition to move from Not Ready -> Ready

Set verdict to **Ready** only when all hard blockers above are cleared at the same time, no new Sev-1/Sev-2 cutover risk is open, and no active compatibility shim is introduced.

## Current migration state (must match before cutover)

- Compatibility report: `docs/ops/compatibility-report.md`
  - Ingestion HTTP endpoints: **Pass**
  - MCP tool schema/contracts: **Pass**
  - Agent-service APIs: **Pass**
  - Schema idempotency: **Partial** (blocked at `migration_pgvector.sql` until pgvector is available)
  - Platform CI workflows: **Partial (local validated)** (remote GitHub Actions evidence pending)
- Risk register: `docs/ops/risk-register.md`
  - `R-003`: **Partial (pgvector prerequisite)**
  - `R-004`: **Open** (remote CI workflow execution evidence)
  - `R-005`: **Open** (must remain no-shim at Go/No-Go)

## Cutover prerequisites

1. Compatibility report remains at current status or better, with no new unresolved shims.
2. Service tests pass for migrated services.
3. Schema idempotency:
   - dry-run passes
   - live double-apply passes through `migration_pgvector.sql` and `migration_fts.sql`
4. SBOM + vulnerability gates pass, or approved risk acceptance is documented.
5. Engineering, Operations, and Security approvals are recorded in this file.

## Staged cutover sequence (explicit)

### Stage 0 — Unblock hard prerequisites

1. Confirm pgvector is available on target Postgres:
   ```bash
   psql "$DATABASE_URL" -X -v ON_ERROR_STOP=1 -c "SELECT name, installed_version FROM pg_available_extensions WHERE name='vector';"
   ```
2. Execute full schema idempotency validation:
   ```bash
   DATABASE_URL="$DATABASE_URL" scripts/validate_schema_idempotency.sh
   ```
3. Sync CI workflows to GitHub Actions path and push:
   ```bash
   mkdir -p .github/workflows
   cp platform/ci/repo-hygiene.yml .github/workflows/repo-hygiene.yml
   cp platform/ci/services-ci.yml .github/workflows/services-ci.yml
   git add .github/workflows/repo-hygiene.yml .github/workflows/services-ci.yml
   git commit -m "ci: enable migrated workflows under .github/workflows"
   git push origin main
   ```
4. Capture remote CI evidence:
   ```bash
   gh workflow list
   gh run list --workflow repo-hygiene.yml
   gh run list --workflow services-ci.yml
   gh run view <run-id> --log
   ```

### Stage 1 — Pre-cutover validation baseline

1. Repo hygiene + services tests:
   ```bash
   bash scripts/repo_hygiene_check.sh
   (cd services/ingestion && go test ./...)
   (cd services/retriever && go test ./...)
   (cd services/mcp-server && go test ./...)
   (cd services/agent-service && uv sync --system-certs --quiet && uv run python -m pytest tests/test_agents.py -q && uv run python -m pytest tests/test_e2e.py -q)
   (cd services/embedding-service && uv sync --system-certs --quiet && uv run pytest -q)
   ```
2. Schema dry-run guard:
   ```bash
   scripts/validate_schema_idempotency.sh --dry-run
   ```
3. Confirm no compatibility shim is active (manual review against compatibility + risk register).

### Stage 2 — Deploy migrated ingestion path (shadow then promote)

1. Build + push + deploy through workflow (`platform/deploy/deploy-ingestion.yml`) after approvals.
2. Verify ingress health endpoint:
   ```bash
   FQDN=$(az containerapp show --name "$CONTAINER_APP_NAME" --resource-group "$RESOURCE_GROUP" --query "properties.configuration.ingress.fqdn" --output tsv)
   curl -fsS "https://${FQDN}/healthz"
   curl -fsS "https://${FQDN}/health"
   ```

### Stage 3 — Promote full migrated runtime

Promotion order: `ingestion -> retriever -> mcp-server -> embedding-service -> agent-service`.

For each service promotion:
1. Deploy migrated artifact.
2. Run service-specific health/smoke checks.
3. Proceed only if health and parity checks pass.

### Stage 4 — Stabilization window

1. Keep legacy rollback path and known-good artifact references available.
2. Monitor error rate, latency, and parity signals.
3. Close stabilization only when no rollback trigger is active.

## Rollback triggers

- Contract parity regression versus compatibility baseline.
- Schema integrity/idempotency failure.
- Failed health checks after promotion.
- Critical security gate failure.
- Any Sev-1/Sev-2 incident attributable to migrated runtime.

## Exact rollback commands (where feasible)

### A) Ingestion service (Container App) rollback

Rollback to previous known-good image tag:

```bash
export IMAGE_REF="<JFROG_URL>/<JFROG_DOCKER_REPO>/rif-ingestion"
export LAST_GOOD_TAG="<previous-good-tag>"

az containerapp update \
  --name "$CONTAINER_APP_NAME" \
  --resource-group "$RESOURCE_GROUP" \
  --image "${IMAGE_REF}:${LAST_GOOD_TAG}"
```

If latest deploy came from a specific commit, capture and pin that commit's tag before rollback.

### B) CI workflow rollback (if new workflows cause failures)

Disable workflow(s) immediately:

```bash
gh workflow disable repo-hygiene.yml
gh workflow disable services-ci.yml
```

Revert workflow-introducing commit and push:

```bash
git revert <workflow-enable-commit-sha>
git push origin main
```

(Alternative emergency path if commit SHA is unknown: remove `.github/workflows/*.yml`, commit, and push.)

### C) Schema/migration rollback

No automated down-migration scripts are maintained for the full stack; use DB snapshot restore as authoritative rollback.

Pre-cutover backup command:

```bash
pg_dump "$DATABASE_URL" --format=custom --file data/backups/pre_cutover_$(date +%Y%m%d_%H%M%S).dump
```

Rollback restore command:

```bash
pg_restore --clean --if-exists --no-owner --no-privileges --dbname "$DATABASE_URL" data/backups/<pre_cutover_backup>.dump
```

Phase-2 manual rollback snippets (only if explicitly approved for partial revert):

```sql
SELECT * FROM ag_catalog.drop_label('rif', 'URL_ENDPOINT');
SELECT * FROM ag_catalog.drop_label('rif', 'POINTCUT_EXPRESSION');
SELECT * FROM ag_catalog.drop_label('rif', 'REGISTERS');
ALTER TABLE rif_meta.repositories DROP COLUMN IF EXISTS application_context_node_id;
ALTER TABLE rif_meta.index_runs DROP COLUMN IF EXISTS tier_b_edge_count, DROP COLUMN IF EXISTS tier_c_edge_count;
```

## Post-rollback verification commands

1. Container App health:
   ```bash
   FQDN=$(az containerapp show --name "$CONTAINER_APP_NAME" --resource-group "$RESOURCE_GROUP" --query "properties.configuration.ingress.fqdn" --output tsv)
   curl -fsS "https://${FQDN}/healthz"
   curl -fsS "https://${FQDN}/health"
   ```
2. Local parity-critical service tests:
   ```bash
   (cd services/ingestion && go test ./...)
   (cd services/mcp-server && go test ./...)
   (cd services/agent-service && uv run python -m pytest tests/test_agents.py -q)
   ```
3. Schema sanity checks:
   ```bash
   psql "$DATABASE_URL" -X -v ON_ERROR_STOP=1 -c "SELECT extname FROM pg_extension WHERE extname IN ('age','vector');"
   psql "$DATABASE_URL" -X -v ON_ERROR_STOP=1 -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='rif_meta' AND table_name IN ('repositories','index_runs','file_nodes','method_nodes');"
   ```
4. CI rollback verification:
   ```bash
   gh workflow list
   gh run list --limit 20
   ```

## Go/No-Go checklist (aligned to current blockers)

| Check | Required Evidence | Status (now) | Gate |
|---|---|---|---|
| Contract parity: ingestion/MCP/agent-service | `docs/ops/compatibility-report.md` area verdicts are Pass | ✅ Complete | Go |
| No active compatibility shims | Compatibility report + risk register shim inventory remains none | ✅ Complete | Go |
| Schema idempotency full live pass (including pgvector) | `DATABASE_URL=... scripts/validate_schema_idempotency.sh` succeeds fully | ⛔ Blocked (`pgvector` prerequisite) | No-Go until cleared |
| Remote GitHub Actions CI evidence | Workflows in `.github/workflows` with successful `gh run view` logs | ⛔ Blocked (not yet synced/executed remotely) | No-Go until cleared |
| SBOM/vulnerability gates | `services-ci.yml` supply-chain job green or signed exception | ⚠️ Pending remote evidence | Conditional |
| Governance approvals | Engineering + Ops + Security signoff rows completed | ⚠️ Pending | No-Go until completed |

## Go/No-Go approvals

**Hard gate:** Cutover execution is prohibited until all three approvals below are replaced with real approver identity, explicit decision, and date.
**Current state (2026-07-02):** Real approver identity/decision/date values are not available in current repository context. This gate remains externally blocked pending user/org input.

| Role | Approver Identity | Decision | Status | External dependency input required |
|---|---|---|---|---|
| Engineering owner | `MISSING (external approval required)` | `PENDING` | No-Go | Provide `approver_name`, `approver_role_or_title`, `decision` (`APPROVED` or `REJECTED`), `decision_date` (`YYYY-MM-DD`), and `evidence_ref` (PR/ticket/comment URL or ID). |
| Operations owner | `MISSING (external approval required)` | `PENDING` | No-Go | Provide `approver_name`, `approver_role_or_title`, `decision` (`APPROVED` or `REJECTED`), `decision_date` (`YYYY-MM-DD`), and `evidence_ref` (change record/on-call approval URL or ID). |
| Security owner | `MISSING (external approval required)` | `PENDING` | No-Go | Provide `approver_name`, `approver_role_or_title`, `decision` (`APPROVED` or `REJECTED`), `decision_date` (`YYYY-MM-DD`), and `evidence_ref` (security review artifact URL or ID). |

### External dependency handoff (required from user/org)

Submit one record for each required role using this exact schema:
`role | approver_name | approver_role_or_title | decision(APPROVED/REJECTED) | decision_date(YYYY-MM-DD) | evidence_ref`

The Go/No-Go gate remains **No-Go** until all three role records are provided and entered above.
