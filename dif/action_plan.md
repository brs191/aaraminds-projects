# DIF Action Plan — Single Source of Truth

**Date:** 2026-07-08  
**Last updated:** 2026-07-19 (P1-02 complete)  
**Audience:** Product, production, platform, and engineering teams  
**Status:** Active execution plan  
**Authority:** This file is the single source of truth for DIF execution status, next actions, gates, and ownership. `DECISIONS.md`, `dif_prd.md`, `dif_brd.md`, and `design-decisions.md` remain source documents for requirements and decisions, but this file is the operating plan.

---

## 0. How to use this plan

Update this file whenever work is completed, blocked, rescoped, or moved between phases.

Status markers:

| Marker | Meaning |
|---|---|
| ✅ Complete | Done and reflected in the repository. |
| 🟡 In progress | Active or partially complete. |
| ⏳ Pending | Not started. |
| 🚫 Blocked | Cannot proceed until a named dependency is resolved. |
| ⚠️ Risk | Known risk requiring mitigation. |

Change-control rules:

1. Do not start implementation work that contradicts an accepted D-entry in `DECISIONS.md`.
2. If a decision changes, add a new dated D-entry first, then update this plan.
3. Keep this file aligned with `.github/copilot-instructions.md` after material architecture or command changes.
4. When code exists, add exact build/test/lint/single-test commands here and in `.github/copilot-instructions.md`.
5. Do not mark a phase complete until its gate criteria in this file are satisfied.

---

## 0.1 Current repository status

| Area | Status | Notes |
|---|---|---|
| Product/business docs | ✅ Complete for current design baseline | `dif_prd.md` and `dif_brd.md` updated with D-009 RIF compatibility and D-010 uniform-corpus ACL posture. |
| Decision log | ✅ Complete for current design baseline | `DECISIONS.md` contains D-001 through D-010. |
| Design backlog | ✅ Complete for current design baseline | `design-decisions.md` updated with ADR-016/RIF compatibility and ADR-003/source ACL posture. |
| Copilot guidance | ✅ Complete | `.github/copilot-instructions.md` created and updated with MCP + RIF compatibility guidance. |
| RIF review | ✅ Complete | Reviewed `/Users/rb692q/projects/aaraminds-projects/cc-rif` and local Postgres databases available as `repointel` and `rif_p19`. |
| Action plan | 🟡 In progress | This file is the operating single source of truth. |
| Runnable DIF code | ✅ Complete for P0 baseline | Minimal Go module rooted at `code/` with module path `github.com/aaraminds/dif`; service entry-point placeholders, build metadata, typed config, safe structured logging, request/execution context, migration runner/checker, corpus admission, source-anchor, ingestion-run lifecycle, Markdown/TXT/DOCX/JSON extraction, graph-emitter, retrieval, embeddings, `search_docs` service-contract, MCP/API boundary, audit/usage writer, health/readiness, and RIF compatibility packages exist. |
| Build/test/lint commands | ✅ Complete for current baseline | From repo root: `python3 evaluation/run_p0.py` validated the full P0 golden gate. From `code/`: `go test ./...`, `go test ./libs/config ./libs/logging ./libs/requestctx ./libs/migrations ./libs/admission ./libs/sourceanchors ./libs/ingestionruns ./libs/extraction ./libs/graphemit ./libs/retrieval ./libs/embeddings ./libs/searchdocs ./libs/mcpapi ./libs/auditusage ./libs/health ./libs/rifcompat`, `go test ./... -run TestInitialMigrationIsDiscoverableFromComponentRoot`, and `go build ./...` validated. No lint command exists yet. |
| ADR folder | ✅ Complete | `design/adr/` exists. |
| ADR-016 RIF compatibility | ✅ Complete for design gate | `design/adr/ADR-016-rif-compatibility-layer.md` created. |
| ADR-003 source ACL posture | ✅ Complete for design gate | `design/adr/ADR-003-source-acl-posture.md` created; v1 is uniformly readable corpora only. |
| ADR-007 source anchor contract | ✅ Complete for design gate | `design/adr/ADR-007-source-anchor-contract.md` created. |
| ADR-006 JSON expansion limits | ✅ Complete for design gate | `design/adr/ADR-006-json-expansion-limits.md` created. |
| P0 skeleton | ✅ Complete for first runnable skeleton | Project folders, starter READMEs, minimal Go module, service entry-point placeholders, shared library package, unit tests, and migration discoverability test exist under `code/`. |
| P0 config/logging baseline | ✅ Complete | `code/libs/config` defines required typed project/corpus/database/environment/log/auth config; `code/libs/logging` defines safe structured logging helpers and redaction tests. |
| P0 request/execution context propagation | ✅ Complete | `code/libs/requestctx` defines explicit request/principal/tenant/project/corpus/tool/run context, operation-specific validation, context attach/extract helpers, structured missing-field errors, and propagation tests. |
| P0 migration runner/checker | ✅ Complete | `code/libs/migrations` and `code/cmd/dif-migrate` load ordered DIF SQL migrations, reject RIF-owned DDL, apply via `psql`, and validate the expected `dif_meta` table inventory. |
| P0 corpus admission | ✅ Complete | `code/libs/admission` enforces v1 `uniform_readable` corpus/source admission, returns `corpus_not_admitted` for rejected or missing corpus access, and records denied audit intent. |
| P0 source anchors | ✅ Complete | `code/libs/sourceanchors` implements canonical source refs, deterministic anchor IDs/content hashes, P0 Markdown/TXT/DOCX/JSON resolution, explicit resolver statuses, and golden-fixture tests. |
| P0 ingestion run lifecycle | ✅ Complete | `code/libs/ingestionruns` models run statuses/counts, validates run shape, allows promotion only for completed non-degenerate runs, and returns explicit non-promotable reasons. |
| P0 Markdown/TXT extraction | ✅ Complete | `code/libs/extraction` emits deterministic Markdown document/section/block records and TXT document/block records with source anchors, retrieval passages, stable IDs/hashes, and `CONTAINS` edges. |
| P0 JSON extraction | ✅ Complete | `code/libs/extraction` emits deterministic bounded JSON records with sorted traversal, JSONPath anchors, cap caveats, fail-closed parse/size errors, and secret-like passage redaction. |
| P0 DOCX paragraph-model extraction | ✅ Complete | `code/libs/extraction` emits deterministic fixture-backed DOCX document/section/block records with user-facing `requirements.docx#pN` paragraph anchors, retrieval passages, stable IDs/hashes, and `CONTAINS` edges. |
| P0 graph emitter and NDJSON writer | ✅ Complete | `code/libs/graphemit` validates extractor output and emits byte-stable NDJSON records for documents, nodes, edges, source anchors, retrieval passages, and caveats. |
| P0 retrieval passage and FTS path | ✅ Complete | `code/libs/retrieval` builds an anchored-only P0 lexical retrieval index, enforces corpus admission, returns `ok`, `no_evidence`, or `corpus_not_admitted`, and passes golden-query tests. |
| P0 embedding interface | ✅ Complete | `code/libs/embeddings` defines the provider seam and deterministic offline hash provider with usage-metering placeholders; no pgvector schema or real Voyage integration added. |
| P0 `search_docs` service contract | ✅ Complete | `code/libs/searchdocs` validates request scope, enforces corpus admission before retrieval, returns anchored-only evidence with scores/caveats, and exposes explicit fail-closed statuses without free-form answer generation. |
| P0 MCP/API skeleton for `search_docs` | ✅ Complete | `code/libs/mcpapi` adds an authenticated thin transport boundary with constant-time bearer-token validation, required-field checks, HTTP and tool-style entry points, service routing, structured errors, and grounded response envelopes. |
| P0 audit/usage write paths | ✅ Complete | `code/libs/auditusage` writes separate audit and non-PII usage records, hashes safe parameters, avoids raw query/snippet/document text, and is wired into the MCP/API path including unauthorized-attempt recording. |
| P0 health/readiness checks | ✅ Complete | `code/libs/health` verifies Postgres connectivity, validates `dif_meta` table inventory, reports RIF compatibility informationally, and exposes secret-safe HTTP health/readiness handlers. |
| P0 RIF compatibility status check | ✅ Complete | `code/libs/rifcompat` assesses ADR-016 RIF deployment states, allows AGE fallback when optional shadows are empty/incomplete, provides deterministic lookups and NUL-separated IDs, and persists status snapshots to `dif_meta.rif_compatibility_status`. |
| P0 golden evaluation runner | ✅ Complete | `evaluation/run_p0.py` runs 10 repeatable P0 checks: targeted Go component tests, full Go tests, Go build, source-anchor, JSON caveat, RIF compatibility, `search_docs`, audit/usage, degenerate-run, and path/CI baseline harnesses. |
| P0 CI baseline | ✅ Complete | `.github/workflows/ci.yml` runs the P0 golden evaluation and a PostgreSQL service-backed migration idempotency check without deployment secrets, container publishing, Azure login, or registry use. |
| P0 exit/sanity review | ✅ Complete | Required P0 ADR set is complete, source-of-truth docs are synchronized, P0 runner passes, and P1-01 was unblocked for execution. |
| P1 code-entity candidate detector | ✅ Complete | `code/libs/codeentities` detects qualified names, source paths, method/class references, backtick spans, code-fence content, service routes, and inline identifier heuristics from anchored document blocks; it persists unresolved candidate rows without creating RIF nodes or `DESCRIBES` edges. |
| P1 RIF resolver and `DESCRIBES` edges | ✅ Complete | `code/libs/codeentities/resolver.go` resolves candidates through `rifcompat` reports (qualified-name, source-path, simple-name, fuzzy), keeps ambiguous/unresolved/`rif_unavailable` outcomes explicit, creates `DESCRIBES` edges only with single-match resolver evidence using shared edge-ID semantics, measures per-corpus resolution rates, and persists via evidence-shaped SQL writers; `code/migrations/002_dif_meta_describes_edges.sql` additively enables `DESCRIBES` in `dif_meta.edges`. |
| Initial `dif_meta` migration design | ✅ Complete for design gate | `code/migrations/001_dif_meta_initial_design.md` created. |
| P0 evaluation plan | ✅ Complete for design gate | `evaluation/p0-evaluation-plan.md` created. |
| Phase-gate tracker | ✅ Complete for initial tracking | `tracking/phase-gate-status.md` created. |
| Risk register | ✅ Complete for initial tracking | `tracking/risk-register.md` created. |
| P0 delivery plan | ✅ Complete for execution planning | `planning/p0-delivery-plan.md` created. |
| Leadership process plan | ✅ Complete for review | `process_plan.md` created to explain the DIF implementation process for leadership review and benchmarking. |
| Prompt execution catalog | ✅ Complete | `prompts.md` created with paired implementation/QA prompts from current state through P0-P3, explicit Aara agent/skill routing, and recurring sanity-check prompts with short unique tags/progress indicators. |
| Executable `dif_meta` SQL migration | ✅ Complete | `code/migrations/001_dif_meta_initial.sql` created and validated by running twice against a scratch local PostgreSQL database; P0-16 added the idempotent unknown-scope auth-audit sentinel corpus for FK-safe unauthorized audit/usage writes. |
| Golden corpus fixture layout | ✅ Complete for initial layout | `evaluation/golden/` now contains synthetic source fixtures, manifest, golden queries, expected anchors, and expected caveats. |
| Source-anchor round-trip harness | ✅ Complete for scaffold | `evaluation/source_anchor_roundtrip.py` validates 5 golden anchors and 5 resolver failure cases. |
| JSON caveat harness | ✅ Complete for scaffold | `evaluation/json_caveat_checks.py` validates all 9 ADR-006 caveat codes and 2 failure behaviors. |
| RIF compatibility fixture | ✅ Complete for scaffold | `evaluation/rif_compatibility_checks.py` validates 5 ADR-016 variants and 5 lookup cases against synthetic fixture data. |
| `search_docs` contract harness | ✅ Complete for scaffold | `evaluation/search_docs_checks.py` validates 7 golden queries, 5 anchored retrieval cases, no-evidence behavior, `corpus_not_admitted`, and unanchored-result exclusion. |
| Audit/usage harness | ✅ Complete for scaffold | `evaluation/audit_usage_checks.py` validates audit and usage schema dimensions, separate write records, MCP call metering, and safe record content against `expected-audit-usage.json`. |
| Degenerate-run guard harness | ✅ Complete for scaffold | `evaluation/degenerate_run_checks.py` validates the ingestion-run promotion guard and 7 promotion cases, including empty, all-failed, anchorless, passageless, and non-complete runs. |

---

## 0.2 Work completed so far

| Date | Status | Deliverable | Details |
|---|---|---|---|
| 2026-07-08 | ✅ Complete | Repository analysis | Confirmed DIF repository is documentation-only: `DECISIONS.md`, `design-decisions.md`, `dif_prd.md`, `dif_brd.md`, plus generated `.github/copilot-instructions.md` and this plan. |
| 2026-07-08 | ✅ Complete | `.github/copilot-instructions.md` | Created future Copilot guidance covering docs-only status, source docs, architecture, MCP server configuration, phase boundaries, and RIF compatibility. |
| 2026-07-08 | ✅ Complete | RIF code review | Reviewed `cc-rif` layout: ingestion, retriever, MCP server, embedding service, agent service, graphstore, phase-5 incremental libs, schemas, migrations. |
| 2026-07-08 | ✅ Complete | RIF local database review | Local exact `rif_dev` / `rif_p19-local` names were not present through `psql`; available relevant DBs were `repointel` and `rif_p19`. `rif_p19` has repo `apm0045942`, one completed index run, populated AGE graph, empty `rif_meta.file_nodes`/`method_nodes`, absent `class_nodes`, and no pgvector/FTS columns. |
| 2026-07-08 | ✅ Complete | D-009 decision | Added RIF compatibility layer decision to `DECISIONS.md`. |
| 2026-07-08 | ✅ Complete | PRD update | Updated `dif_prd.md` through v0.3.3 with AGE-backed RIF compatibility, P0/P1 gate changes, explicit RIF statuses, and D-010 uniform-readable corpus posture. |
| 2026-07-08 | ✅ Complete | BRD update | Updated `dif_brd.md` to v0.3.2 so business claims are gated by compatibility readiness and v1 is uniformly readable corpora only. |
| 2026-07-08 | ✅ Complete | Design backlog update | Updated `design-decisions.md` with ADR-016, corrected default posture, and RIF compatibility non-negotiable. |
| 2026-07-08 | ✅ Complete | Initial action plan | Created `action_plan.md`; then promoted it to this single-source-of-truth execution plan. |
| 2026-07-08 | ✅ Complete | ADR folder | Created `design/adr/`. |
| 2026-07-08 | ✅ Complete | ADR-016 RIF compatibility layer | Created `design/adr/ADR-016-rif-compatibility-layer.md` with compatibility fields, RIF statuses, AGE-first resolver strategy, MCP behavior, fixture plan, and contract-test requirements. |
| 2026-07-08 | ✅ Complete | RIF compatibility fixture specification | Created `evaluation/fixtures/rif/README.md` with fixture variants, synthetic entity contract, required fields, node/edge ID expectations, contract test matrix, resolver response shape, and implementation sequence. |
| 2026-07-08 | ✅ Complete | D-010 source ACL decision | Recorded v1 source ACL posture as uniformly readable corpora only; ACL propagation moves to post-production-readiness/GA v2. |
| 2026-07-08 | ✅ Complete | ADR-003 source ACL posture | Created `design/adr/ADR-003-source-acl-posture.md` with corpus admission rules, runtime behavior, connector constraints, sales language, and evaluation gates. |
| 2026-07-08 | ✅ Complete | ADR-007 source anchor contract | Created `design/adr/ADR-007-source-anchor-contract.md` with canonical source refs, P0 anchor types, table fields, round-trip resolver behavior, MCP/agent citation requirements, `DESCRIBES` linkage, and tests. |
| 2026-07-08 | ✅ Complete | ADR-006 JSON graph expansion limits | Created `design/adr/ADR-006-json-expansion-limits.md` with deterministic traversal, JSONPath anchors, expansion caps, caveat codes, retrieval passage rules, security notes, and tests. |
| 2026-07-08 | ✅ Complete | Project skeleton folders | Created `code/`, `code/services/`, `code/libs/`, `code/migrations/`, `code/testdata/`, `planning/`, `evaluation/`, `evaluation/golden/`, and `tracking/` with starter README files. |
| 2026-07-08 | ✅ Complete | Initial `dif_meta` migration design | Created `code/migrations/001_dif_meta_initial_design.md` with P0 table designs, ID conventions, source-anchor support, audit/usage separation, RIF compatibility status, and SQL acceptance criteria. |
| 2026-07-08 | ✅ Complete | P0 evaluation plan | Created `evaluation/p0-evaluation-plan.md` with golden corpus/query plan, source-anchor round-trip gates, JSON caveat coverage, RIF compatibility fixture gates, `search_docs` contract checks, audit/usage checks, and baseline metrics. |
| 2026-07-08 | ✅ Complete | Phase-gate tracker | Created `tracking/phase-gate-status.md` with P0-P3 phase summary, gate register, P0 exit checklist, blocked downstream gates, owners, evidence, and next actions. |
| 2026-07-08 | ✅ Complete | Risk register | Created `tracking/risk-register.md` with active blockers/risks, severity, owners, mitigations, watchlist, mitigation priorities, review cadence, and closure rules. |
| 2026-07-08 | ✅ Complete | P0 delivery plan | Created `planning/p0-delivery-plan.md` with P0 entry criteria, workstreams, execution sequence, dependency rules, milestones, validation command policy, and first implementation task. |
| 2026-07-08 | ✅ Complete | Leadership process plan | Created `process_plan.md` with leadership-facing process flow diagrams, completed implementation steps, decision points, current state, benchmarkable practices, and review questions. |
| 2026-07-09 | ✅ Complete | Prompt execution catalog | Created and updated `prompts.md` with paired implementation/QA prompts, status/result fields, guardrails, P0-P3 execution coverage, explicit Aara agent/skill routing, and sanity-check prompts. |
| 2026-07-08 | ✅ Complete | Executable `dif_meta` SQL migration | Created `code/migrations/001_dif_meta_initial.sql`; validated syntax and idempotency by running it twice against a scratch local PostgreSQL database and confirming 13 `dif_meta` tables. |
| 2026-07-08 | ✅ Complete | Golden corpus fixture layout | Created synthetic P0 golden corpus layout under `evaluation/golden/`, including admitted/restricted source fixtures, manifest, golden queries, expected anchors, DOCX paragraph fixture model, JSON caveat expectations, and non-admitted corpus behavior. |
| 2026-07-08 | ✅ Complete | Source-anchor round-trip harness scaffold | Created `evaluation/source_anchor_roundtrip.py`, a stdlib-only executable harness for Markdown, TXT, DOCX paragraph model, JSONPath, and resolver failure cases; verified with `python3 evaluation/source_anchor_roundtrip.py`. |
| 2026-07-08 | ✅ Complete | JSON caveat harness scaffold | Created `evaluation/json_caveat_checks.py`, a stdlib-only executable harness for JSON caveat coverage and failure behavior; verified with `python3 evaluation/json_caveat_checks.py`. |
| 2026-07-08 | ✅ Complete | RIF compatibility fixture data and harness | Created `evaluation/fixtures/rif/compat_entities.json`, `expected_resolutions.json`, SQL fixture sketches, and `evaluation/rif_compatibility_checks.py`; verified all 5 ADR-016 variants and 5 lookup cases. |
| 2026-07-08 | ✅ Complete | `search_docs` anchored retrieval harness scaffold | Created `evaluation/search_docs_checks.py`, a stdlib-only executable harness for golden query contract checks; verified 7 queries, 5 anchored retrieval cases, no-evidence behavior, `corpus_not_admitted`, and unanchored-result exclusion. |
| 2026-07-09 | ✅ Complete | Audit/usage write harness scaffold | Created `evaluation/golden/expected-audit-usage.json` and `evaluation/audit_usage_checks.py`, a stdlib-only executable harness for audit/usage schema dimensions, separate write records, MCP call metering, and safe record content; verified with `python3 evaluation/audit_usage_checks.py`. |
| 2026-07-09 | ✅ Complete | Degenerate-run guard harness scaffold | Created `evaluation/golden/expected-degenerate-runs.json` and `evaluation/degenerate_run_checks.py`, a stdlib-only executable harness for ingestion-run promotion safety; verified 7 promotion cases and blocked 6 degenerate/non-complete runs with `python3 evaluation/degenerate_run_checks.py`. |
| 2026-07-09 | ✅ Complete | First runnable Go skeleton | Created module `github.com/aaraminds/dif` under `code/`, service entry-point placeholders, shared build metadata, root unit test, and migration discoverability test; validated with `go test ./...`, `go test ./... -run TestInitialMigrationIsDiscoverableFromComponentRoot`, `go build ./...`, all scaffold evaluation harnesses, and P0-01 QA review with no blocking issues. |
| 2026-07-10 | ✅ Complete | Request/execution context propagation | Created `code/libs/requestctx` with typed DIF execution scope, operation-specific required-field validation, context attach/extract helpers without global mutable state, structured missing-field errors, logging attrs, and nested propagation tests; validated with `go test ./libs/config ./libs/logging ./libs/requestctx`, `go test ./...`, and `go build ./...`. |
| 2026-07-10 | ✅ Complete | Migration runner and schema inventory checks | Created `code/libs/migrations` and `code/cmd/dif-migrate` for deterministic SQL migration loading, RIF-owned DDL rejection, `psql`-backed apply/check operations, and expected `dif_meta` table inventory validation; validated with Go tests/build and by applying migrations twice plus inventory check against a scratch PostgreSQL database. |
| 2026-07-10 | ✅ Complete | Corpus admission implementation | Created `code/libs/admission` with `dif_meta.corpora`/`sources` semantics, v1 `uniform_readable` enforcement, fail-closed `corpus_not_admitted` handling for rejected/missing corpora, source admission checks, and denied audit intent; validated with Go tests/build and `python3 evaluation/search_docs_checks.py`. |
| 2026-07-10 | ✅ Complete | Source anchor model and resolver | Created `code/libs/sourceanchors` with canonical source-ref parsing/formatting, deterministic anchor IDs/content hashes, P0 Markdown/TXT/DOCX/JSON resolver behavior, explicit failure statuses, and golden anchor tests; validated with Go tests/build and `python3 evaluation/source_anchor_roundtrip.py`. |
| 2026-07-10 | ✅ Complete | Ingestion run lifecycle and degenerate-run guard | Created `code/libs/ingestionruns` with lifecycle statuses, non-negative count validation, promotion decisions matching the SQL guard, explicit non-promotable errors, safe write-shape metrics, and golden degenerate-run tests; validated with Go tests/build and `python3 evaluation/degenerate_run_checks.py`. |
| 2026-07-10 | ✅ Complete | Markdown and TXT extractors | Created `code/libs/extraction` with deterministic Markdown/TXT extraction records, source anchors, retrieval passages, stable IDs/hashes, `CONTAINS` edges, heading paths, and golden fixture tests; validated with Go tests/build and `python3 evaluation/source_anchor_roundtrip.py`. |
| 2026-07-10 | ✅ Complete | JSON extractor with expansion caps | Extended `code/libs/extraction` with deterministic JSON traversal, JSONPath source anchors, ADR-006 cap caveats, fail-closed invalid/too-large JSON behavior, secret-like passage redaction, and golden caveat tests; validated with Go tests/build and `python3 evaluation/json_caveat_checks.py`. |
| 2026-07-13 | ✅ Complete | DOCX paragraph-model adapter | Extended `code/libs/extraction` with fixture-backed DOCX paragraph-model extraction, user-facing `requirements.docx#pN` anchors, deterministic ordering, invalid fixture-shape rejection, and golden fixture tests; validated with Go tests/build and `python3 evaluation/source_anchor_roundtrip.py`. |
| 2026-07-13 | ✅ Complete | Deterministic graph emitter and NDJSON writer | Created `code/libs/graphemit` with validation-first byte-stable NDJSON output for documents, source anchors, nodes, edges, retrieval passages, and caveats; validated with Go tests/build and scaffold harnesses. |
| 2026-07-13 | ✅ Complete | Retrieval passage generator and FTS query path | Created `code/libs/retrieval` with anchored-only P0 lexical retrieval, corpus admission enforcement, deterministic ranking, explicit `no_evidence`/`corpus_not_admitted` statuses, and golden-query tests. |
| 2026-07-13 | ✅ Complete | Embedding interface with deterministic stub/hash provider | Created `code/libs/embeddings` with provider abstraction, deterministic offline hash embeddings, request validation, cancellation handling, normalized vectors, and non-PII usage placeholders without pinning production vector schema. |
| 2026-07-13 | ✅ Complete | Service-layer `search_docs` contract | Created `code/libs/searchdocs` with structured request/response types, scope validation, admission-before-retrieval enforcement, anchored-result validation, no-evidence/corpus-not-admitted behavior, and golden-query service tests. |
| 2026-07-13 | ✅ Complete | MCP/API skeleton for `search_docs` | Created `code/libs/mcpapi` with P0 bearer-token auth, required input validation, tool-style invocation, HTTP JSON handler, service-layer routing, structured failure envelopes, and transport tests; security review found no high-confidence issues. |
| 2026-07-13 | ✅ Complete | Audit logging and usage metering write paths | Created `code/libs/auditusage`, wired MCP/API governance recording, added the migration-backed unknown-scope auth-audit sentinel corpus, and passed security review after closing unauthorized-audit bypass findings. |
| 2026-07-13 | ✅ Complete | Postgres-backed health/readiness checks | Created `code/libs/health` with DB connectivity checks, `dif_meta` schema readiness validation, informational RIF status, secret-safe errors, HTTP handlers, and tests for healthy/unavailable/missing-schema states. |
| 2026-07-13 | ✅ Complete | RIF compatibility status check | Created `code/libs/rifcompat` with ADR-016 status assessment, AGE/shadow fallback handling, deterministic code-entity lookup, NUL-separated node/edge ID helpers, and `dif_meta.rif_compatibility_status` persistence. |
| 2026-07-13 | ✅ Complete | Golden P0 evaluation runner | Created `evaluation/run_p0.py` to execute the full P0 gate and report measured baseline durations/output summaries without inventing quality targets. |
| 2026-07-13 | ✅ Complete | CI baseline | Created `.github/workflows/ci.yml` plus `evaluation/path_checks.py`; CI runs P0 validation and Postgres-backed migration idempotency checks with no publish/deploy jobs or secrets. |
| 2026-07-13 | ✅ Complete | P0 exit/sanity review | Completed the missing P0 ADR set, synchronized action/tracking/prompt docs, reran the P0 gate, and unblocked P1-01 as the next prompt. |
| 2026-07-19 | ✅ Complete | P1-02 RIF resolver and `DESCRIBES` edges | Created `code/libs/codeentities/resolver.go` (resolver over `rifcompat` reports, evidence-gated `DESCRIBES` edge builder, SQL edge/resolution writers, measured per-corpus resolution metrics) with 12 resolver tests; added additive idempotent migration `code/migrations/002_dif_meta_describes_edges.sql` (ADR-016 minimum edge fields, `DESCRIBES` evidence-shape constraint); recreated missing `.github/workflows/ci.yml` and `.github/copilot-instructions.md` (absent on disk despite documented complete) and extended CI to apply migration 002; full P0 gate passed 10/10 checks. |

---

## 0.3 Critical review findings

The first version of this plan was directionally correct but needed these corrections to be production-ready:

| Finding | Severity | Resolution |
|---|---|---|
| It did not explicitly record completed work. | High | Added section `0.2 Work completed so far`. |
| It did not declare itself as the operating source of truth. | High | Renamed title and added authority/change-control rules. |
| It had no live status register. | High | Added current repository status and execution board sections. |
| It listed workstreams but did not separate immediate next actions from later phase work strongly enough. | Medium | Added immediate execution board and near-term gate order. |
| It did not make blocked/not-started states visible. | Medium | Added status markers and current-state table. |
| It did not capture the RIF database discovery details in the plan itself. | Medium | Added RIF review/database review entry under completed work. |

---

## 0.4 Immediate execution board

These are the next items to execute in order. Do not start P1 federation implementation until items 1-9 are complete.

| # | Status | Owner | Work item | Deliverable | Gate / acceptance |
|---|---|---|---|---|---|
| 1 | ✅ Complete | Engineering + Platform | Create ADR folder | `design/adr/` | Folder exists and is referenced by this plan. |
| 2 | ✅ Complete | Engineering + Platform | Draft ADR-016 RIF compatibility layer | `design/adr/ADR-016-rif-compatibility-layer.md` | Required fields, statuses, AGE vs `rif_meta` strategy, and fixture tests defined. |
| 3 | ✅ Complete | Engineering + QA | Create RIF compatibility fixture spec | `evaluation/fixtures/rif/README.md` | Fixture captures populated AGE graph and empty/absent shadows from `rif_p19` pattern; executable fixture data and harness now exist. |
| 4 | ✅ Complete | Product + Security | Draft ADR-003 source ACL posture | `design/adr/ADR-003-source-acl-posture.md` | v1 limitation language and admissible corpus rules approved. |
| 5 | ✅ Complete | Engineering + QA | Draft ADR-007 source anchor contract | `design/adr/ADR-007-source-anchor-contract.md` | P0 source anchors and round-trip tests specified. |
| 6 | ✅ Complete | Engineering | Draft ADR-006 JSON expansion limits | `design/adr/ADR-006-json-expansion-limits.md` | JSON caps and caveat behavior specified. |
| 7 | ✅ Complete | Engineering | Create project skeleton | `code/`, `planning/`, `evaluation/`, `tracking/` | Folders exist with starter README files. |
| 8 | ✅ Complete | Engineering | Create initial `dif_meta` migration design | `code/migrations/001_dif_meta_initial_design.md` | Tables and idempotency strategy reviewed before code implementation. |
| 9 | ✅ Complete | QA | Create P0 evaluation plan | `evaluation/p0-evaluation-plan.md` | Golden corpus and required checks defined. |
| 10 | ✅ Complete | Engineering + QA | Create phase-gate tracker | `tracking/phase-gate-status.md` | P0-P3 gates, owners, evidence, and status are visible. |
| 11 | ✅ Complete | Engineering + Production | Create risk register | `tracking/risk-register.md` | Blockers, risks, owners, mitigations, and review cadence are visible. |
| 12 | ✅ Complete | Engineering + QA | Create P0 delivery plan | `planning/p0-delivery-plan.md` | P0 implementation workstreams, sequencing, dependencies, and validation checkpoints are clear before code starts. |
| 12a | ✅ Complete | Product + Engineering | Create leadership process plan | `process_plan.md` | Process followed so far is visible for leadership review, feedback, and benchmarking. |
| 13 | ✅ Complete | Engineering + Platform | Create executable `dif_meta` SQL migration | `code/migrations/001_dif_meta_initial.sql` | Migration is idempotent, creates only `dif_meta` objects, and is safe with or without RIF schemas present. |
| 14 | ✅ Complete | Engineering + QA | Create golden corpus fixture layout | `evaluation/golden/` fixture files and expectations | Golden corpus covers Markdown, TXT, DOCX, JSON, JSON caveats, invalid JSON, and non-admitted corpus behavior. |
| 15 | ✅ Complete | Engineering + QA | Create executable source-anchor round-trip test plan or harness scaffold | `evaluation/source_anchor_roundtrip.py` | Round-trip checks for Markdown, TXT, DOCX paragraph model, JSONPath, and resolver failures are executable. |
| 16 | ✅ Complete | Engineering + QA | Create JSON caveat test harness scaffold | `evaluation/json_caveat_checks.py` | JSON caveat expectations in `expected-caveats.json` are executable and cover all ADR-006 caveat codes. |
| 17 | ✅ Complete | Engineering + QA + Platform | Create executable RIF compatibility fixture data/scripts | `evaluation/fixtures/rif/` fixture data and `evaluation/rif_compatibility_checks.py` | ADR-016 statuses are executable: `rif_not_deployed`, `rif_incompatible`, `rif_shadow_empty`, and `rif_compatible`. |
| 18 | ✅ Complete | Engineering + QA | Create `search_docs` anchored retrieval test scaffold | `evaluation/search_docs_checks.py` | Golden queries validate anchored results, no-evidence behavior, non-admitted corpus fail-closed status, and exclusion of unanchored results. |
| 19 | ✅ Complete | Engineering + QA + Security | Create audit/usage write test scaffold | `evaluation/audit_usage_checks.py` and `evaluation/golden/expected-audit-usage.json` | Required audit and usage dimensions are checked separately and do not log raw sensitive payloads. |
| 20 | ✅ Complete | Engineering + QA | Create degenerate-run guard test scaffold | `evaluation/degenerate_run_checks.py` and `evaluation/golden/expected-degenerate-runs.json` | Empty or all-failed ingestion runs cannot promote an index. |
| 21 | ✅ Complete | Engineering | Establish implementation skeleton and project toolchain | `code/go.mod`, `code/services/`, `code/libs/buildinfo/`, `code/skeleton_test.go`, `code/README.md` | First runnable setup defines package/module layout, unit test command, single-test command, build command, component-root validation policy, and targeted evaluation harness commands. |
| 22 | ✅ Complete | Engineering + Security | Add P0 config and structured logging baseline | `code/libs/config/`, `code/libs/logging/`, `code/README.md` | Required config fields fail explicitly when missing; operational log helpers allow IDs, paths, hashes, counts, caveat codes, latency, and statuses; redaction tests prove logs avoid raw document text, credentials, tokens, and secret-like values. |
| 23 | ✅ Complete | Engineering | Add request ID and execution context propagation | `code/libs/requestctx/`, `code/README.md` | Typed request/principal/tenant/project/corpus/tool/run context validates required fields per operation and propagates through nested calls without global mutable state. |
| 24 | ✅ Complete | Engineering + Platform | Add migration runner and schema inventory checks | `code/libs/migrations/`, `code/cmd/dif-migrate/`, `code/migrations/README.md` | Ordered DIF SQL migrations can be applied idempotently with `psql`, RIF-owned DDL is rejected, and expected `dif_meta` table inventory is checked. |
| 25 | ✅ Complete | Engineering + Security | Add corpus admission implementation | `code/libs/admission/`, `code/README.md`, `code/libs/README.md` | V1 uniformly readable corpus/source gate fails closed with `corpus_not_admitted` for rejected or missing corpora and records denied audit intent. |
| 26 | ✅ Complete | Engineering + QA | Add source anchor model and resolver | `code/libs/sourceanchors/`, `code/README.md`, `code/libs/README.md` | Canonical source refs parse/format, deterministic anchor IDs/content hashes, P0 anchor types resolve, and resolver failures are explicit. |
| 27 | ✅ Complete | Engineering + QA | Add ingestion run lifecycle and degenerate-run guard | `code/libs/ingestionruns/`, `code/README.md`, `code/libs/README.md` | Only completed runs with documents, nodes, anchors, and passages can promote; failed/running/cancelled and degenerate runs return explicit reasons. |
| 28 | ✅ Complete | Engineering + QA | Add deterministic Markdown and TXT extractors | `code/libs/extraction/`, `code/README.md`, `code/libs/README.md` | Markdown emits document/section/block records; TXT emits document/block records; both preserve line anchors, stable IDs/hashes, passages, and `CONTAINS` edges. |
| 29 | ✅ Complete | Engineering + QA + Security | Add JSON extractor with expansion caps | `code/libs/extraction/`, `code/README.md`, `code/libs/README.md` | JSON traversal is deterministic, JSONPath anchors are emitted, all ADR-006 caveat codes are covered, invalid/too-large JSON fails closed, and secret-like values are redacted from passages. |
| 30 | ✅ Complete | Engineering + QA | Add DOCX paragraph-model adapter | `code/libs/extraction/`, `code/README.md`, `code/libs/README.md` | DOCX fixture model emits document/section/block records with user-facing `requirements.docx#pN` paragraph anchors, deterministic ordering, stable IDs/hashes, passages, and `CONTAINS` edges. |
| 31 | ✅ Complete | Engineering + QA | Add deterministic graph emitter and NDJSON writer | `code/libs/graphemit/`, `code/README.md`, `code/libs/README.md` | Same extractor output emits byte-identical NDJSON; every passage has an anchor/source ref; every `CONTAINS` edge points to valid nodes; caveats are preserved. |
| 32 | ✅ Complete | Engineering + QA | Add retrieval passage generator and FTS query path | `code/libs/retrieval/`, `code/libs/extraction/`, `code/README.md`, `code/libs/README.md` | Golden queries return anchored results for Markdown/TXT/DOCX/JSON; unanchored passages are excluded; unknown queries return `no_evidence`; non-admitted corpora return `corpus_not_admitted`. |
| 33 | ✅ Complete | Engineering + QA | Add embedding interface with deterministic stub/hash provider | `code/libs/embeddings/`, `code/README.md`, `code/libs/README.md` | Provider interface supports future RIF/LiteLLM integration; hash provider is deterministic/offline; usage placeholders are non-PII; no pgvector schema or production dimensions are pinned. |
| 34 | ✅ Complete | Engineering + QA | Add service-layer `search_docs` contract | `code/libs/searchdocs/`, `code/README.md`, `code/libs/README.md` | Required tenant/project/corpus/request fields are validated; corpus admission happens before retrieval; only anchored results with score/caveats are returned; `no_evidence` and `corpus_not_admitted` are explicit. |
| 35 | ✅ Complete | Engineering + Security + QA | Add MCP/API skeleton for `search_docs` | `code/libs/mcpapi/`, `code/README.md`, `code/libs/README.md` | Every entry point requires bearer auth; missing fields return structured errors; transport routes to `searchdocs` without duplicating retrieval/ranking; grounded source refs or explicit failure statuses are returned. |
| 36 | ✅ Complete | Engineering + Security + QA | Add audit logging and usage metering write paths | `code/libs/auditusage/`, `code/libs/mcpapi/`, `code/migrations/001_dif_meta_initial.sql` | Audit and usage records are separated; audit includes security dimensions/source refs; usage counts are non-PII; raw parameters/query/snippets/document text are not stored; unauthorized attempts use the FK-safe sentinel scope. |
| 37 | ✅ Complete | Engineering + QA | Add Postgres-backed health/readiness checks | `code/libs/health/`, `code/README.md`, `code/libs/README.md` | Health verifies DB connectivity; readiness verifies `dif_meta` schema inventory; RIF status is informational; errors are explicit but secret-safe. |
| 38 | ✅ Complete | Engineering + Platform + QA | Add RIF compatibility status check | `code/libs/rifcompat/`, `code/README.md`, `code/libs/README.md` | ADR-016 statuses are assessed; empty/incomplete shadows do not mask compatible AGE fallback; missing/incompatible RIF returns explicit non-success statuses; RIF-owned schemas are not mutated. |
| 39 | ✅ Complete | Engineering + QA | Add Golden P0 evaluation runner | `evaluation/run_p0.py`, `evaluation/README.md`, `code/README.md` | One command runs all required P0 Go and Python checks and reports measured run metrics only. |
| 40 | ✅ Complete | Engineering + DevOps + QA | Add CI baseline | `.github/workflows/ci.yml`, `evaluation/path_checks.py`, `evaluation/README.md`, `code/README.md` | CI runs `python3 evaluation/run_p0.py`, checks SQL migration idempotency against a PostgreSQL service, and avoids secrets, deployment, and container publishing. |
| 41 | ✅ Complete | Engineering + QA + Security | Run P0 exit/sanity review | `design/adr/ADR-005-parser-strategy.md`, `design/adr/ADR-008-mcp-gateway-auth-model.md`, `design/adr/ADR-009-ingestion-orchestration.md`, `design/adr/ADR-010-embedding-strategy.md`, `design/adr/ADR-011-evaluation-gates.md`, `design/adr/ADR-012-observability-audit-schema.md`, `design/adr/ADR-013-security-threat-model.md`, `prompts.md`, `tracking/phase-gate-status.md` | Required P0 ADRs exist, docs are synchronized, P0 runner passes, and P1 execution is unblocked. |
| 42 | ✅ Complete | Engineering + QA | Add P1 code-entity candidate detector | `code/libs/codeentities/`, `code/README.md`, `code/libs/README.md`, `prompts.md`, `tracking/phase-gate-status.md` | Anchored document blocks emit deterministic unresolved code-entity candidates with source refs, match modes, confidence, caveats, and SQL persistence that preserves later resolver evidence. |
| 43 | ✅ Complete | Engineering + Platform + QA | Add RIF resolver and `DESCRIBES` edge creation | `code/libs/codeentities/resolver.go`, `code/libs/codeentities/resolver_test.go`, `code/migrations/002_dif_meta_describes_edges.sql` | Candidates resolve through `rifcompat` reports (qualified-name, source-path, simple-name, fuzzy with PascalCase fallback); `DESCRIBES` edges are created only for single-match resolver evidence with shared edge-ID semantics; ambiguous/unresolved/`rif_unavailable` outcomes are explicit; per-corpus resolution-rate metrics are measured; migration 002 additively enables `DESCRIBES` in `dif_meta.edges` with an evidence-shape constraint. |
| 44 | ⏳ Pending | Engineering + Platform + QA | Add cross-graph tools `docs_for_code` and `code_for_doc` (P1-03) | MCP tool contracts, service wiring, tests | Explicit tenant/project/corpus/repo scope; source-anchored responses only; explicit `rif_not_deployed`/`rif_incompatible` statuses; audit/usage on every call; mandatory security gate after implementation. |

---

## 0.5 Current blockers and risks

| Item | Type | Status | Impact | Mitigation |
|---|---|---|---|---|
| RIF local DB names differ from pgAdmin labels | Risk | ⚠️ Risk | Automation may target wrong DB names. | Use explicit `DATABASE_URL`; document `rif_p19` fixture setup. |
| Existing RIF shadows may be empty | Risk | ⚠️ Risk | P1/P2 cross-graph queries could return false empty results if they bypass `rifcompat`. | Keep all P1 federation work behind `code/libs/rifcompat` and ADR-016 AGE/shadow fallback semantics. |
| Embedding model and vector dimension are not pinned | Risk | ⚠️ Risk | Vector schema could be reworked if created too early. | Keep production vector schema deferred until the model/dimension spike exits. |
| P1 cross-graph tools not implemented | Risk | ⚠️ Risk | P1-02 resolver and `DESCRIBES` edges exist, but `docs_for_code`/`code_for_doc` (P1-03) and traversal tools (P1-04) are not implemented yet. | Build P1-03 on `code/libs/codeentities` resolver outcomes and `dif_meta.edges` `DESCRIBES` rows; keep responses anchored with explicit RIF statuses. |
| `.github/` was missing from this working copy | Risk | ⚠️ Risk | `ci.yml` and `copilot-instructions.md` were documented as complete but absent on disk (likely lost syncing the hidden directory from the original machine); the P0 gate failed its path check until they were recreated on 2026-07-19. | Files recreated to the `evaluation/path_checks.py` contract; verify against the original machine's copies and confirm CI runs green on the hosted repository. `[VERIFY]` |

---

## 1. Executive objective

Move Documents Intelligence Factory (DIF) from design into a production-grade P0 skeleton while preserving the accepted product direction:

- DIF is a code-aware document intelligence backend.
- DIF deploys per project into the same Postgres database as RIF.
- DIF owns `dif_meta`; it must not mutate RIF-owned schemas.
- RIF+DIF federation remains core v1 scope.
- Cross-graph features must use a RIF compatibility layer because existing RIF deployments may have populated AGE graph data while `rif_meta` shadow tables are empty or absent.

The immediate execution goal is **P0 readiness**: deterministic document ingestion, source-anchored retrieval, `search_docs` MCP tool, usage/audit events, and a proven RIF compatibility contract.

---

## 2. Key decisions already accepted

| Decision | Summary | Impact |
|---|---|---|
| D-001 | Customer-tenant Azure BYOC deployment. | Production must plan for customer Azure tenancy, not AaraMinds-hosted multi-tenant SaaS first. |
| D-002 | Voyage is the default prose embedding direction via shared LiteLLM abstraction. | Exact model/dimension decided after P0 spike. |
| D-003 | DIF graph storage uses Postgres relational adjacency, not Apache AGE. | DIF graph lives in `dif_meta`; RIF may still use AGE. |
| D-007 | RIF+DIF federation is core v1 architecture. | `DESCRIBES`, `docs_for_code`, `code_for_doc`, and `drift_report` stay in scope. |
| D-008 | JSON is a first-class P0 artifact; Excel is v1.5. | JSONPath anchors and JSON expansion caps are required in P0. |
| D-009 | RIF compatibility layer is mandatory. | DIF cannot assume populated `rif_meta.file_nodes` / `method_nodes`; AGE-backed RIF resolution must be supported or abstracted. |

---

## 3. Team responsibilities

| Team | Primary responsibilities |
|---|---|
| Product | Confirm phase scope, success criteria, pilot constraints, and sales-facing claims. |
| Engineering | Implement schemas, extractors, retriever, MCP server, eval harness, and tests. |
| Production/platform | Define deployment posture, CI/CD, observability, secrets, identity, container hardening, and operational runbooks. |
| Security/governance | Review auth, audit, prompt-injection controls, source ACL posture, logging policy, and supply-chain controls. |
| Evaluation/QA | Own golden corpus, golden queries, deterministic extraction checks, source-anchor validation, and RIF compatibility contract tests. |

---

## 4. Immediate priority: ADR-016 RIF compatibility layer

ADR-016 is the highest-priority gate because it protects the core DIF differentiator: joining documents to code.

### 4.1 Problem to solve

The local RIF review showed:

- RIF canonical graph is Postgres + Apache AGE under schema `rif`.
- `rif_meta` exists but may only contain metadata or optional shadows.
- Local `rif_p19` has populated AGE graph data but empty `rif_meta.file_nodes` and `rif_meta.method_nodes`.
- `rif_meta.class_nodes`, pgvector columns, and FTS columns may be absent.

Therefore DIF cross-graph features must use a compatibility layer rather than direct assumptions about raw RIF shadow tables.

### 4.2 ADR-016 deliverables

Create:

```text
design/adr/ADR-016-rif-compatibility-layer.md
```

Define:

- Required fields: `node_id`, `repo_id`, `kind`, `qualified_name`, `simple_name`, `source_ref`, `origin`, `confidence`.
- Drift fields: code version, content hash, or equivalent change evidence.
- Supported data sources:
  - AGE schema `rif`.
  - Populated `rif_meta` shadows.
  - Future RIF-provided compatibility view/API.
- Explicit statuses:
  - `rif_not_deployed`
  - `rif_incompatible`
  - `rif_shadow_empty`
  - `rif_compatible`
- Resolver modes:
  - exact qualified-name match
  - file path match
  - simple-name match
  - inferred/fuzzy match with caveat
  - unresolved candidate

### 4.3 ADR-016 acceptance criteria

ADR-016 is complete when:

- It defines the compatibility contract fields and statuses.
- It documents AGE-first resolution for existing RIF deployments.
- It explains when populated `rif_meta` shadows can be used.
- It defines contract tests against a pinned local RIF fixture.
- It blocks P1 federation tools until the contract passes.

---

## 5. Required ADR backlog before build

Create `design/adr/` and complete these ADRs before or during P0 setup.

| ADR | Required before | Owner | Purpose |
|---|---|---|---|
| ADR-003 Source ACL posture | ✅ Complete | Product + Security | v1 uses uniformly readable corpora only; ACL propagation is post-production-readiness/GA v2. |
| ADR-005 Parser strategy | ✅ Complete | Engineering | Choose parsers for Markdown, TXT, DOCX, JSON, PDF, PPTX. |
| ADR-006 JSON graph expansion limits | ✅ Complete | Engineering + QA | Set max JSON depth, node count, array/object caps, and caveat behavior. |
| ADR-007 Source anchor contract | ✅ Complete | Engineering + QA | Define source anchors and round-trip resolver requirements. |
| ADR-008 MCP gateway/auth model | ✅ Complete | Platform + Security | Define auth, gateway posture, tool allowlists, and schema versioning. |
| ADR-009 Ingestion orchestration | ✅ Complete | Engineering | Define jobs, retries, idempotency, checkpoints, and atomic promotion. |
| ADR-010 Embedding strategy | ✅ Complete | Engineering + Product | Pin provider abstraction, model choices, dimensions, fallback, and metering. |
| ADR-011 Evaluation gates | ✅ Complete | Evaluation/QA | Define golden tests and release gates. |
| ADR-012 Observability/audit schema | ✅ Complete | Platform | Define logs, metrics, traces, audit events, usage events. |
| ADR-013 Security threat model | ✅ Complete | Security | Cover MCP, prompt injection, secrets, logging, supply chain. |
| ADR-016 RIF compatibility layer | ✅ Complete | Engineering + Platform | Define AGE vs `rif_meta` compatibility contract and fixture tests. |

---

## 6. Repository structure to create

Recommended structure:

```text
design/
  adr/
product/
planning/
evaluation/
tracking/
code/
  services/
  libs/
  migrations/
  testdata/
```

Recommended document moves or links:

| Current file | Recommended location |
|---|---|
| `dif_prd.md` | `product/dif_prd.md` or keep root with link from `product/` |
| `dif_brd.md` | `product/dif_brd.md` or keep root with link from `product/` |
| `design-decisions.md` | `design/design-decisions.md` or keep root with link from `design/` |
| `DECISIONS.md` | Keep root as decision log, or move to `design/DECISIONS.md` with root link |
| `action_plan.md` | Keep root as execution plan |

If files are moved, update `.github/copilot-instructions.md`.

---

## 7. P0 scope definition

P0 must prove this path:

```text
local docs / git docs
  -> deterministic extraction
  -> dif_meta nodes, edges, source anchors
  -> retrieval passages
  -> FTS / embedding interface
  -> search_docs
  -> source-anchored MCP result
```

### 7.1 P0 included

- Repository skeleton.
- CI baseline.
- `dif_meta` schema.
- P0 formats:
  - `.md`
  - `.txt`
  - `.docx`
  - `.json`
- Graph nodes:
  - `document`
  - `section`
  - `block`
- Edge:
  - `CONTAINS`
- MCP tool:
  - `search_docs`
- Audit log.
- Usage events.
- Health endpoint that checks Postgres.
- RIF compatibility check and contract fixture.

### 7.2 P0 excluded

- PDF/PPTX parsing.
- SharePoint/OneDrive connector.
- Generated explanation agent.
- `docs_for_code`.
- `code_for_doc`.
- `drift_report`.
- ACL propagation.
- Direct reliance on populated RIF shadow tables.

---

## 8. P0 `dif_meta` schema work

Engineering should create idempotent SQL migrations for:

1. `dif_meta.corpora`
2. `dif_meta.sources`
3. `dif_meta.documents`
4. `dif_meta.document_versions`
5. `dif_meta.nodes`
6. `dif_meta.edges`
7. `dif_meta.source_anchors`
8. `dif_meta.retrieval_passages`
9. `dif_meta.ingestion_runs`
10. `dif_meta.audit_log`
11. `dif_meta.usage_events`
12. `dif_meta.rif_compatibility_status`
13. `dif_meta.code_entity_candidates`

Schema rules:

- Migrations must be idempotent.
- DIF must not mutate RIF-owned schemas: `rif`, `rif_meta`, or future equivalents.
- Serving index promotion must be atomic.
- Unresolved references must be stored explicitly.
- Usage metering must be separate from audit logs.
- Source anchors must round-trip to original content.

---

## 9. Source-anchor contract

P0 source anchors:

| Format | Required anchor |
|---|---|
| Markdown | document version + heading path + line range |
| TXT | document version + line range |
| DOCX | document version + paragraph index and/or heading path |
| JSON | document version + JSONPath |

P1+ source anchors:

| Format | Required anchor |
|---|---|
| PDF | page + block or bounding box |
| PPTX | slide + shape/text block |
| XLSX | sheet + cell/range |

Acceptance criteria:

- Every retrieval result has a resolvable source anchor.
- Every source anchor can round-trip to source content.
- Unsupported or incomplete extraction produces caveats, not silent success.
- A format without a defined source anchor is not admitted.

---

## 10. Golden demo corpus

Create a public, non-sensitive fixture corpus under `code/testdata/` or `evaluation/fixtures/`.

Required files:

1. Markdown architecture sample.
2. Plain text policy sample.
3. DOCX requirements sample.
4. JSON config referencing code-like entities.
5. Cross-reference examples.
6. Broken reference examples.
7. Duplicate heading/path examples.
8. Large JSON sample for expansion-cap tests.

This corpus supports:

- demos
- deterministic extraction tests
- source-anchor tests
- golden query evaluation
- future `DESCRIBES` candidate detection

---

## 11. RIF compatibility fixture

Use local `rif_p19` as the reference fixture because it has populated AGE graph data and empty `rif_meta` shadows.

### 11.1 Fixture requirements

Capture or script:

- RIF schema presence.
- AGE graph labels:
  - `File`
  - `Class`
  - `Method`
  - `SAME_FILE_CALLS`
- Representative properties:
  - `node_id`
  - `repo_id`
  - `kind`
  - `qualified_name`
  - `simple_name`
  - `source_ref`
  - `origin`
  - `confidence`
- Empty or absent `rif_meta` shadows.

### 11.2 Contract tests

The fixture must validate:

- exact method qualified-name resolution
- file path resolution
- simple-name resolution with caveat
- unknown entity returns unresolved
- missing RIF returns `rif_not_deployed`
- present but unsupported RIF returns `rif_incompatible`
- empty optional shadow tables do not cause false empty success

---

## 12. Engineering implementation sequence

Recommended order:

1. Create project folders.
2. Add ADR-016.
3. Add remaining P0 ADRs.
4. ✅ Add SQL migration runner.
5. Add initial `dif_meta` migration.
6. ✅ Add config and structured logging.
7. ✅ Add request ID propagation.
8. ✅ Add corpus admission gate.
9. Add ingestion run lifecycle.
10. Add Markdown parser.
10. Add TXT parser.
11. Add JSON parser with expansion caps.
12. Add DOCX parser adapter.
13. Add node/edge/source-anchor emitter.
14. Add deterministic NDJSON writer.
15. Add degenerate-run guard.
16. Add atomic version promotion.
17. Add retrieval passage generator.
18. Add FTS query path.
19. Add embedding interface with stub/hash provider.
20. Add `search_docs` retriever.
21. Add MCP server with `search_docs`.
22. Add Postgres-backed health check.
23. Add audit logging.
24. Add usage metering.
25. Add RIF compatibility status check.
26. Add RIF contract test fixture.
27. Add golden P0 evaluation.

---

## 13. Production/platform workstream

Production and platform teams should prepare:

1. GitHub Actions CI baseline.
2. SQL migration idempotency job.
3. Secret handling pattern.
4. Managed identity and Key Vault design.
5. Container baseline:
   - non-root
   - `.dockerignore`
   - lockfile-based builds
   - `HEALTHCHECK`
6. Postgres connection policy.
7. Log redaction policy.
8. Audit-log retention policy.
9. Usage-event retention policy.
10. Observability baseline:
   - OpenTelemetry traces
   - service metrics
   - structured logs
   - MCP tool-call metrics
11. Deployment environments:
   - local dev
   - CI fixture DB
   - pilot Azure BYOC
12. Rollback plan for failed migrations or bad index promotion.

---

## 14. Security/governance workstream

Security should review and approve:

1. MCP auth model.
2. OAuth 2.1 + PKCE requirement for pilot/remote deployments.
3. P0 bearer-token allowance for internal deployments only.
4. Tool allowlist model.
5. Prompt-injection controls.
6. Source text logging restrictions.
7. Audit event shape.
8. Usage event shape.
9. Source ACL limitation language.
10. Dependency/vulnerability scan policy.
11. Container hardening.
12. Secret handling.

Security non-negotiables:

- No unauthenticated MCP or HTTP surface.
- No raw enterprise document text in logs by default.
- No generated answer without resolvable source refs.
- No overclaiming ACL propagation in v1.

---

## 15. Evaluation and QA workstream

QA must build an evaluation harness before P0 exit.

### 15.1 Required checks

1. Deterministic extraction:
   - same corpus twice produces byte-identical NDJSON.
2. Degenerate-run guard:
   - empty extraction cannot promote an index.
3. Source-anchor round trip:
   - Markdown
   - TXT
   - DOCX
   - JSON
4. JSON expansion caps:
   - oversized JSON returns caveats.
5. Search:
   - `search_docs` returns source-anchored results.
6. MCP:
   - missing required fields rejected.
   - audit event written.
7. Usage:
   - usage event written separately from audit.
8. RIF compatibility:
   - compatible AGE-backed RIF detected.
   - missing RIF detected.
   - incompatible RIF detected.
   - empty shadows do not produce false success.
9. Logging:
   - raw document text not emitted by default.

### 15.2 P0 metrics

Measure, do not invent:

- extraction determinism: pass/fail
- source-anchor validity: target 100%
- search precision baseline: measured on golden queries
- unresolved reference count
- `DESCRIBES` candidate count
- RIF compatibility status
- MCP latency baseline

---

## 16. CI/CD baseline

Add CI jobs as soon as code exists:

1. Go unit tests: from `code/`, run `go test ./...`.
2. Python unit tests.
3. SQL migration idempotency.
4. Source-anchor resolver tests: from `code/`, run `go test ./libs/sourceanchors`.
5. RIF compatibility contract tests.
6. Golden ingestion/search test.
7. Documentation path/link validation.
8. Vulnerability scan.
9. Container scan after Dockerfiles exist.

Current single-test command: from `code/`, run `go test ./... -run TestInitialMigrationIsDiscoverableFromComponentRoot`.

Current build command: from `code/`, run `go build ./...`.

Current targeted evaluation harness commands from repo root:

```bash
python3 evaluation/source_anchor_roundtrip.py
python3 evaluation/json_caveat_checks.py
python3 evaluation/rif_compatibility_checks.py
python3 evaluation/search_docs_checks.py
python3 evaluation/audit_usage_checks.py
python3 evaluation/degenerate_run_checks.py
```

---

## 17. P1 plan after P0 gate

P1 starts only after P0 and ADR-016 pass.

P1 scope:

1. Code-entity detector over document blocks.
2. `code_entity_candidates` population.
3. RIF compatibility-layer resolution.
4. `DESCRIBES` edges.
5. `DESCRIBES` resolution-rate metric.
6. `docs_for_code`.
7. `code_for_doc`.
8. `impact_of_change`.
9. `trace_references`.
10. PDF parsing router.
11. PPTX parsing router.
12. Cross-encoder reranking.
13. Federation golden evals.

P1 exit criteria:

- Golden-query set passing.
- Impact-analysis semantics tested.
- Determinism check green.
- `DESCRIBES` resolution rate measured on a real RIF project.
- AGE-backed RIF works even if `rif_meta` shadows are empty.

---

## 18. P2 plan after P1 gate

P2 scope:

1. Agent service.
2. Claim-block response contract.
3. Groundedness scorer.
4. `/explain`.
5. `/investigate_impact`.
6. `diff_versions`.
7. `explain_topic`.
8. `drift_report`.
9. Incremental re-indexing.
10. Full usage metering.

P2 exit criteria:

- Claim citation gate at 100%.
- Incremental correctness proven.
- Usage metering complete.
- Drift report validated against a known code change.

---

## 19. P3 plan after P2 gate

P3 scope:

1. SharePoint/OneDrive connector.
2. Connector auth.
3. Connector throttling and retry handling.
4. Uniformly-readable corpus qualification.
5. Terraform AzureRM deployment.
6. Managed identity and Key Vault.
7. Private networking posture.
8. Container hardening.
9. Observability dashboards.
10. Paid pilot deployment checklist.

P3 exit criteria:

- Paid pilot deployment with real admissible corpus.
- Production observability enabled.
- Deployment rollback documented.
- Security baseline approved.

---

## 20. Tracking artifacts status

These artifacts support this plan. `action_plan.md` remains the operating source of truth; supporting artifacts provide detail.

| Artifact | Status | Purpose | Next action |
|---|---|---|---|
| `action_plan.md` | 🟡 In progress | Operating source of truth for status, gates, and next work. | Keep updated after every completed/blocked item. |
| `prompts.md` | ✅ Complete | Copy/paste-ready paired implementation/QA prompt catalog for P0-P3 execution with Aara agent/skill routing and recurring sanity-check prompts. | Update prompt result blocks after each prompt execution. |
| `process_plan.md` | ✅ Complete | Leadership-facing process flow and benchmark artifact. | Use for leadership review and feedback. |
| `design/adr/ADR-016-rif-compatibility-layer.md` | ✅ Complete | RIF compatibility contract and federation gate. | Use as input for fixture/test spec. |
| `planning/roadmap.md` | ⏳ Pending | Phase roadmap and sequencing. | Create after implementation skeleton/toolchain choice if a roadmap beyond `planning/p0-delivery-plan.md` is still needed. |
| `planning/p0-delivery-plan.md` | ✅ Complete | Detailed P0 work breakdown and sprint sequence. | Use to execute P0 implementation in dependency order. |
| `evaluation/p0-evaluation-plan.md` | ✅ Complete | Golden corpus, metrics, and P0 evaluation gates. | Use as input for executable fixtures and tests. |
| `tracking/phase-gate-status.md` | ✅ Complete | Gate checklist and phase status. | Keep synchronized with this plan as gates move. |
| `tracking/risk-register.md` | ✅ Complete | Risk, mitigation, owner, status. | Review during every P0 planning/status checkpoint. |

Minimum gates to track:

- ADR gate
- schema gate
- ingestion gate
- source-anchor gate
- RIF compatibility gate
- MCP gate
- evaluation gate
- security gate
- production-readiness gate

---

## 21. Immediate next 10 tasks

The authoritative immediate execution queue is section `0.4 Immediate execution board`. The next 10 items are summarized here for quick handoff and must stay synchronized with section `0.4`.

| Priority | Status | Task |
|---|---|---|
| 1 | ✅ Complete | Add source anchor model and resolver. |
| 2 | ✅ Complete | Implement ingestion run lifecycle and degenerate-run guard. |
| 3 | ✅ Complete | Implement deterministic Markdown and TXT extractors. |
| 4 | ✅ Complete | Implement JSON extractor with expansion caps. |
| 5 | ✅ Complete | Implement DOCX paragraph-model adapter. |
| 6 | ✅ Complete | Implement deterministic graph emitter and NDJSON writer. |
| 7 | ✅ Complete | Implement retrieval passage generator and P0 FTS query path. |
| 8 | ✅ Complete | Implement P0 `search_docs` MCP/API contract. |
| 9 | ✅ Complete | Integrate audit/usage writes with real MCP/API paths. |
| 10 | ✅ Complete | Run P0 implementation/evaluation sanity pass. |
| 11 | ✅ Complete | Create `tracking/risk-register.md`. |
| 12 | ✅ Complete | Create `planning/p0-delivery-plan.md`. |
| 12a | ✅ Complete | Create `process_plan.md`. |
| 13 | ✅ Complete | Create `code/migrations/001_dif_meta_initial.sql`. |
| 14 | ✅ Complete | Create golden corpus fixture layout under `evaluation/golden/`. |
| 15 | ✅ Complete | Create executable source-anchor round-trip test plan or harness scaffold. |
| 16 | ✅ Complete | Create JSON caveat test harness scaffold. |
| 17 | ✅ Complete | Create executable RIF compatibility fixture data/scripts. |
| 18 | ✅ Complete | Create `search_docs` anchored retrieval test scaffold. |
| 19 | ✅ Complete | Create audit/usage write test scaffold. |
| 20 | ✅ Complete | Create degenerate-run guard test scaffold. |
| 21 | ✅ Complete | Establish implementation skeleton and project toolchain. |
| 22 | ✅ Complete | Add P0 config and structured logging baseline. |
| 23 | ✅ Complete | Add request ID and execution context propagation. |
| 24 | ✅ Complete | Add migration runner and schema inventory checks. |
| 25 | ✅ Complete | Add corpus admission implementation. |
| 26 | ✅ Complete | Add source anchor model and resolver. |
| 27 | ✅ Complete | Add ingestion run lifecycle and degenerate-run guard. |
| 28 | ✅ Complete | Add deterministic Markdown and TXT extractors. |
| 29 | ✅ Complete | Add JSON extractor with expansion caps. |
| 30 | ✅ Complete | Add DOCX paragraph-model adapter. |
| 31 | ✅ Complete | Add deterministic graph emitter and NDJSON writer. |
| 32 | ✅ Complete | Add retrieval passage generator and P0 FTS query path. |
| 33 | ✅ Complete | Add embedding interface with deterministic stub/hash provider. |
| 34 | ✅ Complete | Add service-layer `search_docs` contract. |
| 35 | ✅ Complete | Add MCP/API skeleton for `search_docs`. |
| 36 | ✅ Complete | Add audit logging and usage metering write paths. |
| 37 | ✅ Complete | Add Postgres-backed health/readiness checks. |
| 38 | ✅ Complete | Add RIF compatibility status check. |
| 39 | ✅ Complete | Add Golden P0 evaluation runner. |
| 40 | ✅ Complete | Add CI baseline. |
| 41 | ✅ Complete | Run P0 exit/sanity review. |
| 42 | ✅ Complete | Add P1 code-entity candidate detector. |
| 43 | ✅ Complete | Add RIF resolver and `DESCRIBES` edge creation. |
| 44 | ⏳ Pending | Add cross-graph tools `docs_for_code` and `code_for_doc` (P1-03). |

The P1 resolver and `DESCRIBES` gate is complete: resolver evidence exists in `code/libs/codeentities` and `dif_meta.edges` supports `DESCRIBES` rows. `docs_for_code` and `code_for_doc` (P1-03) are now unblocked and must return anchored responses with explicit RIF statuses; `drift_report` remains blocked on P2 version/change evidence.

---

## 22. Definition of P0 done

**Status:** ✅ Complete as of 2026-07-13 21:19 IST.  
**Validation evidence:** `python3 evaluation/run_p0.py` passed all 10 checks after the exit/sanity updates, including required ADR path checks and CI safety checks.

P0 is done only when:

- ADR-016 and required P0 ADRs are accepted.
- `dif_meta` migrations are idempotent.
- Markdown/TXT/DOCX/JSON ingestion works.
- Source anchors round-trip.
- Deterministic extraction passes.
- Degenerate-run guard passes.
- `search_docs` returns source-anchored results.
- MCP `search_docs` is authenticated and audited.
- Usage events are emitted.
- RIF compatibility check handles compatible, missing, and incompatible states.
- Golden evaluation runs in CI.
- No raw document text is logged by default.
