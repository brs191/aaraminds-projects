# Fleet-Ready Migration Plan

**Tag:** `[FLEETMIG]`

## Updated observations

1. `rif-migration.md` is strong and execution-ready, and now includes modern guardrails (sandboxing, GitOps model, SBOM/vuln scanning, go/no-go approvals).
2. The migration should be run as a phased program, not a single large refactor PR.
3. Highest delivery risk is coordination drift across multiple agents; this is controlled by strict path ownership + centralized ops artifacts.
4. Cutover risk is manageable if compatibility report, rollback plan, and ownership approvals are treated as hard gates.
5. For this repository, preserve deterministic extraction + graph-first truth model, and keep legacy phase paths alive until final cutover approval.

## Fleet task matrix

| Agent | Scope / Owned Paths | Prompt to Run | Done When |
|---|---|---|---|
| `fleet-orchestrator` | `cc-rif/docs/ops/*` (coordination only) | Execute `rif-migration.md` phase-by-phase. Do not implement service code. Create/maintain architecture-map, move-log, compatibility-report, risk-register, cutover-and-rollback-plan. Track blockers/dependencies and gate phase completion. | All phase gates tracked, dependencies clear, artifacts updated every phase |
| `baseline-inventory` | read-only legacy tree + `cc-rif/docs/ops/architecture-map.md` | Inventory phase-* services, endpoints, schemas, workflows, tests. Produce baseline architecture map with source paths + validation commands. | Baseline map complete and auditable |
| `scaffold-owner` | `cc-rif/**` (structure only) | Create target tree exactly per `rif-migration.md`. Add short READMEs for top-level folders. No functional moves yet. | Target tree complete, no behavior change |
| `go-services-migrator` | `cc-rif/services/{ingestion,retriever,mcp-server}`, `cc-rif/libs/graphstore`, `cc-rif/libs/contracts` | Migrate Go services incrementally with compatibility shims; preserve API/tool contracts; keep legacy runnable until cutover. | Go tests pass in migrated paths; parity evidence logged |
| `python-services-migrator` | `cc-rif/services/{embedding-service,agent-service}` | Migrate Python services preserving endpoint semantics, config behavior, and test expectations. | Python suites pass in migrated paths; parity evidence logged |
| `extractor-migrator` | `cc-rif/extractors/{core-java,spring-java}` | Migrate Java extractors/build files and preserve deterministic extraction contracts. | Extractor builds pass; output compatibility recorded |
| `data-schema-migrator` | `cc-rif/data/{schema,migrations,seeds}` | Consolidate canonical schema + migration layout and enforce deterministic order + idempotency checks. | Schema unified, drift checks passing |
| `platform-ci-migrator` | `cc-rif/platform/{infra/terraform,ci,deploy}` | Port CI/deploy workflows to new paths; keep Azure OIDC + JFrog constraints; add SBOM/vuln scan gates. | CI green on new paths; security/supply-chain gates active |
| `test-pyramid-owner` | `cc-rif/tests/{unit,integration,e2e,perf,security,fixtures}` | Rehome tests into pyramid layout; preserve fixtures; add migration parity suites. | Test layout complete + parity tests passing |
| `governance-owner` | `cc-rif/governance/*`, `cc-rif/docs/{runbooks,ops}` | Migrate governance files and formalize ownership/signoff/checklist artifacts. | Governance complete + signoff template ready |
| `compatibility-auditor` | `cc-rif/docs/ops/{compatibility-report.md,risk-register.md}` | Validate endpoint/tool/schema parity, list shims/deltas, and produce risk-ranked findings with mitigations. | Compatibility + risk artifacts complete |
| `cutover-manager` | `cc-rif/docs/ops/cutover-and-rollback-plan.md` | Produce phased cutover and rollback plan with explicit triggers, commands, and verification steps. | Cutover plan finalized and gate-ready |

## Execution order

1. `fleet-orchestrator` + `baseline-inventory`
2. `scaffold-owner`
3. Parallel migration wave: `go-services-migrator`, `python-services-migrator`, `extractor-migrator`, `data-schema-migrator`
4. `platform-ci-migrator` + `test-pyramid-owner`
5. `compatibility-auditor`
6. `governance-owner`
7. `cutover-manager`
8. `fleet-orchestrator` final go/no-go checkpoint

## Hard merge gates for every phase

1. `bash scripts/repo_hygiene_check.sh`
2. Go service tests (legacy + migrated equivalents)
3. Python service tests (legacy + migrated equivalents)
4. Updated contract parity evidence in `compatibility-report.md`
5. SBOM + vulnerability scan pass (or explicit risk acceptance in `risk-register.md`)
6. Updated `move-log.md` with exact old -> new path mappings

## Run tag usage

Use **`[FLEETMIG]`** when launching your Fleet run so agents execute this exact plan and artifact contract.
