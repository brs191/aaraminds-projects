# DIF Phase Gate Status

**Status:** Active tracking artifact  
**Date:** 2026-07-09  
**Owners:** Engineering + QA + Platform + Security  
**Source of truth:** `../action_plan.md` remains the operating source of truth. Update that file whenever this tracker changes materially.

---

## 1. Purpose

This tracker makes phase gates, owners, required evidence, and current status visible to production and engineering teams.

It tracks:

- ADR/design gates
- schema gates
- ingestion gates
- source-anchor gates
- RIF compatibility gates
- MCP/API gates
- evaluation gates
- security/governance gates
- production-readiness gates

---

## 2. Status markers

| Marker | Meaning |
|---|---|
| ✅ Complete | Gate evidence exists and is reflected in the repository. |
| 🟡 In progress | Gate has partial evidence or an active design artifact. |
| ⏳ Pending | Gate has not started. |
| 🚫 Blocked | Gate cannot proceed until a named dependency is resolved. |
| ⚠️ Risk | Gate has known unresolved risk. |

---

## 3. Current phase summary

| Phase | Status | Evidence | Next gate |
|---|---|---|---|
| P0 design baseline | ✅ Complete | Required P0 ADRs exist: ADR-003, ADR-005, ADR-006, ADR-007, ADR-008, ADR-009, ADR-010, ADR-011, ADR-012, ADR-013, ADR-016; initial `dif_meta` schema design, P0 evaluation plan, risk register, and P0 delivery plan exist. | Keep ADRs aligned when implementation uncovers conflicts. |
| P0 implementation | ✅ Complete | `code` Go module skeleton, typed config package, safe structured logging package, request/execution context package, migration runner/checker, corpus admission package, source-anchor package, ingestion-run lifecycle package, Markdown/TXT/DOCX/JSON extraction package, deterministic graph emitter, retrieval package, embeddings package, `search_docs` service-contract package, MCP/API boundary package, audit/usage writer package, health/readiness package, RIF compatibility package, executable `dif_meta` SQL migration, golden corpus fixture layout, source-anchor harness, JSON caveat harness, RIF compatibility harness, `search_docs` contract harness, audit/usage harness, degenerate-run harness, Golden P0 evaluation runner, CI baseline, and P0 exit/sanity review exist. | Maintain P0 gates while P1 advances. |
| P0 evaluation | ✅ Complete | Evaluation plan, golden corpus fixture layout, executable source-anchor harness, JSON caveat harness, RIF compatibility harness, `search_docs` contract harness, audit/usage harness, degenerate-run harness, path/CI baseline harness, Golden P0 runner, config/logging redaction unit tests, request-context propagation tests, migration inventory tests, corpus admission tests, source-anchor resolver tests, ingestion-run lifecycle tests, Markdown/TXT extraction tests, DOCX paragraph-model extraction tests, graph-emitter NDJSON tests, retrieval golden-query tests, embedding provider tests, `search_docs` service-contract tests, MCP/API auth/validation/governance tests, audit/usage writer tests, health/readiness tests, RIF compatibility service tests, JSON extraction/caveat tests, scratch DB migration validation, and component-root Go tests exist. | Keep the P0 runner updated as gates are added. |
| P1 federation | 🟡 In progress | P1-01 candidate detection and P1-02 RIF resolver/`DESCRIBES` edges are implemented in `code/libs/codeentities` (resolver, evidence-gated edge builder, SQL writers, measured resolution metrics) with `code/migrations/002_dif_meta_describes_edges.sql`; `docs_for_code` and `code_for_doc` are not implemented yet. | Start P1-03 cross-graph tools `docs_for_code`/`code_for_doc`. |
| P2 agentic intelligence | 🚫 Blocked | Depends on P1 federation. | Complete P1 gates first. |
| P3 production readiness | 🚫 Blocked | Depends on P0-P2 implementation and evaluation evidence. | Complete implementation, security, deployment, and observability gates. |

---

## 4. Gate register

| Gate | Owner | Status | Required evidence | Current evidence | Next action |
|---|---|---|---|---|---|
| Decision log gate | Product + Architecture | ✅ Complete for current baseline | Accepted D-entries for major constraints. | `DECISIONS.md` includes D-001 through D-010. | Add new D-entry before changing accepted direction. |
| ADR gate | Architecture + Engineering | ✅ Complete for P0 exit | Accepted ADRs for parser strategy, source ACL, JSON expansion, source anchors, MCP/auth, ingestion orchestration, embedding strategy, evaluation gates, observability/audit, security threat model, and RIF compatibility. | ADR-003, ADR-005, ADR-006, ADR-007, ADR-008, ADR-009, ADR-010, ADR-011, ADR-012, ADR-013, and ADR-016 exist. | Keep ADRs aligned when implementation uncovers conflicts. |
| Source ACL posture gate | Product + Security | ✅ Complete for P0 design | Uniformly readable corpus posture and admission behavior defined. | ADR-003 accepted. | Implement corpus admission and `corpus_not_admitted` fail-closed test. |
| RIF compatibility design gate | Engineering + Platform | ✅ Complete for P0 design | Required RIF fields, statuses, resolver strategy, and boundaries defined. | ADR-016, `evaluation/fixtures/rif/README.md`, and `evaluation/rif_compatibility_checks.py`. | Keep fixture expectations aligned with the eventual service-level resolver. |
| Source-anchor design gate | Engineering + QA | ✅ Complete for P0 design | Canonical source ref and P0 anchor contract defined. | ADR-007 accepted. | Keep implementation aligned as new formats are added. |
| JSON expansion design gate | Engineering + QA | ✅ Complete for P0 design | Deterministic traversal, caps, caveats, and failure behavior defined. | ADR-006 accepted. | Keep implementation aligned as JSON handling evolves. |
| Schema design gate | Engineering + Platform | ✅ Complete for design | P0 `dif_meta` tables and idempotency strategy defined. | `code/migrations/001_dif_meta_initial_design.md` and executable `code/migrations/001_dif_meta_initial.sql`. | Keep future schema changes additive through new migrations. |
| Schema implementation gate | Engineering + Platform | ✅ Complete | Idempotent SQL migration runs twice and does not mutate RIF schemas. | `code/migrations/001_dif_meta_initial.sql`; scratch local PostgreSQL validation ran migration twice, confirmed 13 `dif_meta` tables, and now seeds the FK-safe unknown-scope auth-audit sentinel corpus. Additive `code/migrations/002_dif_meta_describes_edges.sql` enables `DESCRIBES` edges (P1-02); CI applies both migrations twice; local scratch-DB double-apply of 002 pending `[VERIFY]`. | Keep migration additive; create next migration for schema changes. |
| Corpus admission implementation gate | Engineering + Security | ✅ Complete for scaffold | Admitted corpus succeeds; non-admitted corpus fails closed. | `evaluation/search_docs_checks.py` validates `corpus_not_admitted` behavior against `evaluation/golden/manifest.json`; `code/libs/searchdocs` and `code/libs/mcpapi` enforce the fail-closed path; `code/libs/auditusage` records denied attempts. | Keep admission wired as ingestion and transport expand. |
| Ingestion implementation gate | Engineering | ✅ Complete for P0 extractors | Markdown, TXT, DOCX, and JSON ingestion produce deterministic graph records. | `code/libs/extraction` implements deterministic Markdown, TXT, DOCX paragraph-model, and JSON records with source anchors, retrieval passages, cap caveats where applicable, and `CONTAINS` edges. | Integrate extractors with service persistence/load path when ingestion service lands. |
| Graph emitter gate | Engineering + QA | ✅ Complete | Extractor output emits byte-stable NDJSON without dangling edges or unanchored passages. | `code/libs/graphemit` validates nodes, anchors, edges, passages, source refs, and caveats; tests cover Markdown/TXT/DOCX/JSON byte stability, DOCX user-facing refs, dangling-edge rejection, unanchored-passage rejection, source-ref mismatch rejection, and caveat preservation. | Integrate emitted records with persistence/load path when ingestion service lands. |
| Degenerate-run guard gate | Engineering + QA | ✅ Complete | Empty/all-failed runs cannot promote an index. | `code/libs/ingestionruns` implements lifecycle statuses, non-negative count validation, promotion decisions matching the SQL guard, explicit non-promotable errors, and golden tests; `evaluation/degenerate_run_checks.py` validates the `ingestion_runs` promoted guard and 7 promotion cases. | Integrate the package with real ingestion service persistence when implementation lands. |
| Source-anchor implementation gate | Engineering + QA | ✅ Complete | Anchors resolve round trip for Markdown, TXT, DOCX, and JSON. | `code/libs/sourceanchors` implements canonical source refs, deterministic anchor IDs/content hashes, P0 resolver behavior, explicit failure statuses, and golden tests; `evaluation/source_anchor_roundtrip.py` validates 5 golden anchors and 5 resolver failure cases. | Integrate the source-anchor package with parser/retriever service code when implementation lands. |
| Retrieval gate | Engineering + QA | ✅ Complete for P0 retrieval package | `search_docs` returns anchored passages and excludes unanchored results. | `code/libs/retrieval` builds an anchored-only lexical index from extractor output, enforces corpus admission, returns `ok`, `no_evidence`, or `corpus_not_admitted`, and tests all 7 golden queries plus unanchored-passage exclusion; `evaluation/search_docs_checks.py` validates the external contract. | Continue with P0 release gate validation. |
| Embedding seam gate | Engineering + QA | ✅ Complete | Provider abstraction exists without pinning production dimensions or vector schema. | `code/libs/embeddings` defines provider/request/response/usage contracts and deterministic offline `HashProvider`; tests prove deterministic vectors, validation failures, cancellation handling, and usage placeholders. | Replace stub with shared RIF/LiteLLM provider after exact model/dimension spike exit. |
| `search_docs` service contract gate | Engineering + QA | ✅ Complete | Service layer validates scope, enforces admission before retrieval, and returns source-anchored evidence only. | `code/libs/searchdocs` exposes structured request/response types, explicit `ok`/`no_evidence`/`corpus_not_admitted`/fail-closed statuses, score/caveat fields, audit intent on denied admission, and tests against all golden queries plus invalid/unanchored cases. | Keep as the P0 MCP evidence contract. |
| RIF compatibility implementation gate | Engineering + Platform | ✅ Complete | `rif_not_deployed`, `rif_incompatible`, `rif_shadow_empty`, and `rif_compatible` are tested at fixture and service-package level. | `evaluation/rif_compatibility_checks.py` validates 5 ADR-016 variants and 5 lookup cases; `code/libs/rifcompat` implements status assessment, compatible AGE fallback when shadows are empty/incomplete, deterministic lookups, NUL-separated ID helpers, and DIF-owned status persistence. | Use `rifcompat` as the P1 federation gate input. |
| Code-entity candidate gate | Engineering + QA | ✅ Complete | Document text candidate detection is deterministic, source-anchored, unresolved by default, and independent of optional RIF shadows. | `code/libs/codeentities` detects qualified names, source paths, method/class references, backtick spans, code-fence content, service routes, and inline identifier heuristics; tests cover deterministic output, source-anchor enforcement, unresolved shape validation, and SQL persistence to `dif_meta.code_entity_candidates`. | Candidates now feed the P1-02 resolver. |
| RIF resolver / `DESCRIBES` edge gate | Engineering + Platform + QA | ✅ Complete | Candidates resolve through the compatibility layer; `DESCRIBES` edges exist only with resolver evidence; ambiguous/unresolved/`rif_unavailable` outcomes are explicit; resolution rate is measured per corpus; RIF-owned schemas are not mutated. | `code/libs/codeentities/resolver.go` resolves qualified-name/source-path/simple-name/fuzzy modes against `rifcompat` reports, builds evidence-gated `DESCRIBES` edges with shared edge-ID semantics, and persists via `SQLEdgeStore`/`UpdateResolutions`; `code/migrations/002_dif_meta_describes_edges.sql` additively enables `DESCRIBES` in `dif_meta.edges` with an evidence-shape constraint; 12 resolver tests pass; CI applies migration 002 twice. Live scratch-DB double-apply of 002 not yet run locally `[VERIFY]` — covered by the CI migration-idempotency job. | Use resolver outcomes and `dif_meta.edges` `DESCRIBES` rows as P1-03 inputs. |
| MCP/API gate | Engineering + Platform | ✅ Complete for skeleton | `search_docs` MCP/API contract returns evidence and explicit statuses. | `code/libs/mcpapi` requires bearer auth on tool-style and HTTP entry points, validates required inputs non-empty, routes to `code/libs/searchdocs`, returns grounded source refs or explicit failure statuses, records audit/usage events when configured, and has tests for auth failure, missing fields, routing, governance writes, and no answer field. | Keep transport thin as additional tools are added. |
| Audit gate | Engineering + Security | ✅ Complete | MCP/API access writes audit events with required dimensions. | `code/libs/auditusage` validates and writes `dif_meta.audit_log` records; `code/libs/mcpapi` records success, denied corpus, and unauthorized attempts; unauthorized attempts use the migration-backed sentinel corpus; security review found no high-confidence issues. | Keep audit writer wired as transport expands. |
| Usage metering gate | Engineering + Product | ✅ Complete | Non-PII usage events are separate from audit events. | `code/libs/auditusage` validates and writes separate `dif_meta.usage_events`; usage records exclude principal/source refs/raw parameters and carry non-PII counts; `evaluation/audit_usage_checks.py` and Go tests cover safety. | Keep usage writer wired as transport expands. |
| Health/readiness gate | Engineering + Platform | ✅ Complete | Health verifies Postgres connectivity and readiness verifies `dif_meta` availability. | `code/libs/health` pings Postgres, validates expected `dif_meta` tables, exposes HTTP health/readiness handlers, returns secret-safe errors, and reports RIF compatibility status as informational for doc-only mode; security review found no high-confidence issues. | Keep RIF status informational for P0 doc-only mode. |
| Logging safety gate | Engineering + Security | ✅ Complete for P0 baseline | Logs avoid raw secrets, tokens, and full private document contents. | `code/libs/logging` exposes operational metadata helpers and redaction tests for raw document text, credentials, tokens, private-key markers, database URLs, and secret-like values; ADR-006, P0 evaluation plan, and `evaluation/audit_usage_checks.py` safe-record checks also exist. | Integrate the same safe logger into service entry points as they are implemented. |
| Golden corpus gate | QA | ✅ Complete for initial layout | Golden fixtures and expected anchors/queries exist. | `evaluation/golden/` contains synthetic source fixtures, manifest, queries, expected anchors, and expected caveats. | Add deterministic generated IDs after extractor implementation lands. |
| Evaluation harness gate | QA + Engineering | ✅ Complete for current P0 baseline | Tests can run repeatably with documented commands. | `evaluation/run_p0.py` runs 10 checks: targeted Go component tests, full Go tests, Go build, source-anchor, JSON caveat, RIF compatibility, `search_docs`, audit/usage, degenerate-run, and path/CI baseline harnesses. Last measured run passed all 10 checks. | Keep the runner updated as gates are added. |
| Build/test/lint gate | Engineering | ✅ Complete for current baseline | Exact commands documented in `action_plan.md`, `.github/copilot-instructions.md`, and `code/README.md`. | From repo root: `python3 evaluation/run_p0.py` validated the full P0 gate. From `code/`: `go test ./...`, `go test ./libs/config ./libs/logging ./libs/requestctx ./libs/migrations ./libs/admission ./libs/sourceanchors ./libs/ingestionruns ./libs/extraction ./libs/graphemit ./libs/retrieval ./libs/embeddings ./libs/searchdocs ./libs/mcpapi ./libs/auditusage ./libs/health ./libs/rifcompat`, `go test ./... -run TestInitialMigrationIsDiscoverableFromComponentRoot`, and `go build ./...` validated. `.github/workflows/ci.yml` runs the same P0 gate and PostgreSQL service-backed migration idempotency. No lint command exists yet. | Add lint command when a lint runner is introduced. |
| CI baseline gate | Engineering + DevOps + QA | ✅ Complete | CI runs component tests, harnesses, and migration checks without deployment/publish side effects. | `.github/workflows/ci.yml` runs `python3 evaluation/run_p0.py`, applies `code/migrations/001_dif_meta_initial.sql` twice against PostgreSQL 16, verifies 13 `dif_meta` tables, and has no Azure login, secrets, registry, or publish jobs; `evaluation/path_checks.py` verifies required workflow terms and safety exclusions. | Keep CI aligned with the P0 runner as gates are added. |
| Production deployment gate | Platform | 🚫 Blocked | Deployment, secrets, identity, observability, rollback, and runbooks exist. | Not started. | Wait for implementation skeleton and deployment architecture. |
| Security review gate | Security | ✅ Complete for P0 baseline | Auth, audit, prompt-injection controls, logging, and supply chain reviewed for P0 scope. | ADR-013 covers P0 threat model; `code/libs/mcpapi`, `code/libs/auditusage`, `code/libs/logging`, `evaluation/path_checks.py`, and `.github/workflows/ci.yml` provide P0 evidence. | Reopen for pilot/remote OAuth, production deployment, or new tool exposure. |

---

## 5. P0 exit gate checklist

P0 cannot exit until all required gates below are complete:

| Required gate | Status | Evidence path |
|---|---|---|
| Accepted P0 ADRs | ✅ Complete for P0 exit | `design/adr/ADR-003-source-acl-posture.md`, `design/adr/ADR-005-parser-strategy.md`, `design/adr/ADR-006-json-expansion-limits.md`, `design/adr/ADR-007-source-anchor-contract.md`, `design/adr/ADR-008-mcp-gateway-auth-model.md`, `design/adr/ADR-009-ingestion-orchestration.md`, `design/adr/ADR-010-embedding-strategy.md`, `design/adr/ADR-011-evaluation-gates.md`, `design/adr/ADR-012-observability-audit-schema.md`, `design/adr/ADR-013-security-threat-model.md`, `design/adr/ADR-016-rif-compatibility-layer.md` |
| Initial `dif_meta` schema design | ✅ Complete for design | `code/migrations/001_dif_meta_initial_design.md` |
| P0 evaluation plan | ✅ Complete for design | `evaluation/p0-evaluation-plan.md` |
| Executable `dif_meta` SQL migration | ✅ Complete | `code/migrations/001_dif_meta_initial.sql` |
| Golden corpus fixtures | ✅ Complete for initial layout | `evaluation/golden/` |
| Source-anchor round-trip tests | ✅ Complete for scaffold | `evaluation/source_anchor_roundtrip.py` |
| Source-anchor resolver implementation | ✅ Complete | `code/libs/sourceanchors` |
| P0 extractors for Markdown/TXT/DOCX/JSON | ✅ Complete | `code/libs/extraction` |
| Deterministic graph emitter / NDJSON writer | ✅ Complete | `code/libs/graphemit` |
| Retrieval passage generator / P0 FTS path | ✅ Complete | `code/libs/retrieval` |
| Embedding provider seam | ✅ Complete | `code/libs/embeddings` |
| Service-layer `search_docs` contract | ✅ Complete | `code/libs/searchdocs` |
| MCP/API skeleton for `search_docs` | ✅ Complete | `code/libs/mcpapi` |
| Audit/usage write path | ✅ Complete | `code/libs/auditusage`, `code/libs/mcpapi` |
| Health/readiness checks | ✅ Complete | `code/libs/health` |
| RIF compatibility status check | ✅ Complete | `code/libs/rifcompat` |
| Golden P0 evaluation runner | ✅ Complete | `evaluation/run_p0.py` |
| CI baseline | ✅ Complete | `.github/workflows/ci.yml`, `evaluation/path_checks.py` |
| JSON cap/caveat tests | ✅ Complete for scaffold | `evaluation/json_caveat_checks.py` |
| RIF compatibility fixture tests | ✅ Complete for scaffold | `evaluation/rif_compatibility_checks.py`, `evaluation/fixtures/rif/` |
| `search_docs` anchored retrieval test | ✅ Complete for scaffold | `evaluation/search_docs_checks.py` |
| Audit/usage write tests | ✅ Complete for scaffold | `evaluation/audit_usage_checks.py`, `evaluation/golden/expected-audit-usage.json` |
| Degenerate-run guard test | ✅ Complete for scaffold | `evaluation/degenerate_run_checks.py`, `evaluation/golden/expected-degenerate-runs.json` |
| Ingestion run lifecycle implementation | ✅ Complete | `code/libs/ingestionruns` |
| Config/logging baseline | ✅ Complete for P0 baseline | `code/libs/config`, `code/libs/logging` |
| Build/test/lint commands documented | ✅ Complete for current baseline | `action_plan.md`, `.github/copilot-instructions.md`, `code/README.md` |

---

## 6. Blocked downstream gates

| Downstream feature | Blocked by | Required unblock evidence |
|---|---|---|
| `docs_for_code` | P1-03 implementation (resolver and `DESCRIBES` edges now exist) | Anchored responses, explicit RIF statuses, audit/usage on every call, security gate passed. |
| `code_for_doc` | P1-03 implementation (resolver and `DESCRIBES` edges now exist) | Anchored responses, explicit RIF statuses, audit/usage on every call, security gate passed. |
| `drift_report` | P1 federation plus code/content version evidence | RIF code version/content hash support and DIF document version evidence. |
| SharePoint/OneDrive connector | P0-P2 implementation and uniform-readable corpus gate | Connector scope restricted to uniformly readable folders/libraries. |
| Per-user source ACL propagation | Post-production-readiness/GA v2 | New decision/ADR; not part of v1. |

---

## 7. Update rules

Update this tracker when:

1. A gate changes status.
2. Evidence is added, moved, or superseded.
3. A blocker is discovered or resolved.
4. A new phase gate is introduced.
5. A runnable command becomes available.

After updating this file, also update `../action_plan.md`.
