# cc-rif Migration Risk Register

| ID | Risk | Impact | Likelihood | Mitigation | Owner | Status |
|---|---|---|---|---|---|---|
| R-001 | Go module/import paths break after relocation | High | High | Update module names + replace paths incrementally and test each service | Go services migrator | Open |
| R-002 | Python runtime/dependency mismatch in new paths | Medium | Medium | Rebuild env in `services/*`, run targeted test suites | Python services migrator | Open |
| R-003 | Schema migration order causes drift/regression | High | Medium | Deterministic validation runner added (`scripts/validate_schema_idempotency.sh`); pgvector prerequisite provisioned on local Postgres (2026-07-02); full end-to-end rerun with `DATABASE_URL='postgres:///postgres?sslmode=disable' scripts/validate_schema_idempotency.sh` passed pass-1/pass-2 through `migration_pgvector.sql` and `migration_fts.sql` (exit `0`). | Data schema migrator | Mitigated (full local idempotency pass confirmed) |
| R-004 | CI workflows still reference legacy paths | Medium | High | Rewire workflow paths and run validation in `cc-rif` branch | Platform CI migrator | Open |
| R-005 | Contract mismatch at cutover | High | Medium | Maintain compatibility report with snapshot diffs and explicit shim inventory/removal plan; block Go/No-Go if temporary shims remain active. | Compatibility auditor | Open |
| R-006 | Ownership/signoff gaps delay cutover | Medium | Medium | Complete go/no-go approvals in cutover plan | Governance owner | Open |
| R-007 | Undetected dependency vulnerabilities in migrated services | High | Medium | `platform/ci/services-ci.yml` generates CycloneDX SBOM artifacts and blocks CI on High/Critical findings via Anchore scan gate. Risk acceptance is currently manual via documented exception in this register + security signoff. | Platform CI migrator + Security owner | Mitigated |

## Temporary compatibility shim inventory

Current assessment after migrated code + compatibility report review: **none active**.

| Shim/adapter | Status | Owner | Decommission criteria | Timeline |
|---|---|---|---|---|
| None active | Closed | Compatibility auditor | Any new temporary compatibility shim/adapter must have a linked migration ticket, be listed in `docs/ops/compatibility-report.md`, and be removed once parity smoke checks pass with shim disabled. | No pending removal. Any future shim must be decommissioned before Go/No-Go approval and no later than stabilization window close. |
