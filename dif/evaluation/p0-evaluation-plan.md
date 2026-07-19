# DIF P0 Evaluation Plan

**Status:** P0 evaluation design gate with scaffold harnesses implemented  
**Date:** 2026-07-08  
**Owners:** QA + Engineering + Platform  
**Related ADRs:** ADR-003, ADR-006, ADR-007, ADR-016  
**Related docs:** `action_plan.md`, `DECISIONS.md`, `dif_prd.md`, `design-decisions.md`, `evaluation/fixtures/rif/README.md`

---

## 1. Purpose

This plan defines the minimum evidence required before DIF P0 implementation can be considered ready for phase exit.

P0 evaluation must prove that DIF can:

- admit only uniformly readable corpora
- ingest supported document types deterministically
- preserve resolvable source anchors
- retrieve evidence-backed passages
- apply JSON expansion caps and caveats
- detect RIF compatibility status without assuming populated RIF shadow tables
- separate audit events from usage events
- fail closed when required gates are not met

This plan defines evaluation scope and pass/fail criteria. The current repository now includes synthetic golden fixtures plus scaffold harnesses for source anchors, JSON caveats, RIF compatibility, `search_docs`, audit/usage, and degenerate-run promotion safety; service-level evaluation must wire those expectations into the real implementation as code lands.

---

## 2. P0 quality gates

| Gate | Required for P0 exit | Evidence |
|---|---:|---|
| Corpus admission | Yes | Admitted corpus succeeds; non-admitted corpus fails closed with `corpus_not_admitted`. |
| Source anchors | Yes | Every indexed node/passage has a valid `anchor_id` and `source_ref`; round-trip resolver succeeds. |
| Retrieval grounding | Yes | `search_docs` results include source anchors and do not return uncited claims. |
| JSON bounded expansion | Yes | Caps produce deterministic caveats instead of silent drops. |
| RIF compatibility | Yes | Compatibility status distinguishes no RIF, incompatible RIF, empty shadows, and compatible RIF. |
| Audit logging | Yes | MCP/API calls write audit events with principal, corpus, outcome, latency, and source-anchor evidence. |
| Usage metering | Yes | Non-PII usage events are written separately from audit events. |
| Degenerate-run guard | Yes | Empty or all-failed ingestion runs cannot promote an index. |
| Determinism | Yes | Re-running unchanged fixtures produces stable IDs and stable ordered outputs. |

---

## 3. Golden corpus plan

P0 requires a small deterministic golden corpus under `evaluation/golden/`.

Recommended fixture set:

| Fixture | Format | Purpose |
|---|---|---|
| `architecture-overview.md` | Markdown | Heading-path anchors, line-range anchors, section/block graph, retrieval grounding. |
| `runbook.txt` | TXT | Plain line anchors and simple passage retrieval. |
| `requirements.docx` | DOCX | Paragraph-index anchors and optional heading path extraction. |
| `service-config.json` | JSON | JSONPath anchors, code-entity candidate text, bounded object/array traversal. |
| `large-capped.json` | JSON | Depth, array, object-property, scalar, block-text, and total-text caveats. |
| `invalid.json` | JSON | `json_parse_error` failure behavior. |
| `restricted-corpus-sample.md` | Markdown | Non-admitted corpus fail-closed behavior. |

Golden corpus records must include:

1. `corpus_id`
2. `project_id`
3. source path
4. expected document ID inputs
5. expected source anchors
6. expected retrieval passages
7. expected caveats
8. expected audit/usage dimensions

Do not add real customer documents, credentials, or private source content to evaluation fixtures.

---

## 4. Golden query set

P0 search evaluation should use deterministic queries with explicit expected anchors.

| Query ID | Query | Expected behavior |
|---|---|---|
| `q-architecture-owner` | "Who owns the architecture service?" | Returns grounded Markdown passage with `md` source ref. |
| `q-runbook-retry` | "What is the retry procedure?" | Returns TXT line-range source ref. |
| `q-docx-requirement` | "Which requirement mentions approval?" | Returns DOCX paragraph source ref. |
| `q-json-service-owner` | "Which service owner is configured for payments?" | Returns JSONPath source ref. |
| `q-json-class-reference` | "Which config mentions PaymentService?" | Returns JSON passage and code-entity candidate, not a resolved code link unless RIF compatibility is enabled. |
| `q-unknown` | "What is the disaster recovery RTO?" | Returns no unsupported claim; response must be empty or explicit insufficient evidence. |

Each golden query must define:

- accepted result count range
- required top anchor IDs or source refs
- required caveat behavior, if any
- prohibited behavior, especially uncited answers

---

## 5. Source-anchor round-trip tests

P0 must validate ADR-007 for every supported format.

| Format | Required anchor behavior |
|---|---|
| Markdown | Resolve heading path and line range back to the expected excerpt. |
| TXT | Resolve line range back to the expected excerpt. |
| DOCX | Resolve paragraph index back to the expected paragraph text. |
| JSON | Resolve JSONPath back to the expected scalar/subtree. |

Required failure cases:

| Failure status | Scenario |
|---|---|
| `anchor_not_found` | Anchor ID/source ref does not exist. |
| `document_version_not_found` | Document version is missing or not indexed. |
| `source_content_unavailable` | Source content is unavailable to the resolver. |
| `anchor_out_of_range` | Line/paragraph range exceeds document bounds. |
| `anchor_type_unsupported` | Unsupported anchor type is requested in P0. |
| `content_hash_mismatch` | Source content no longer matches anchored content hash. |

Pass criteria:

1. Every retrieval passage has at least one resolvable source anchor.
2. Every resolver failure returns a structured status, not a silent empty result.
3. Source refs are stable and parseable in the canonical form:

```text
corpus_id@document_version_id:anchor_type:anchor_payload
```

---

## 6. JSON cap and caveat tests

P0 must validate ADR-006 with deterministic JSON fixtures.

Required caveat coverage:

| Caveat code | Required fixture condition |
|---|---|
| `json_depth_capped` | Nested object exceeds depth 12. |
| `json_block_count_capped` | Document would emit more than 2,000 JSON blocks. |
| `json_object_properties_capped` | Object has more than 200 properties. |
| `json_array_elements_capped` | Array has more than 100 elements. |
| `json_scalar_truncated` | Scalar string exceeds 8,192 characters. |
| `json_block_text_truncated` | Normalized block text exceeds 16,384 characters. |
| `json_total_text_capped` | Total emitted text exceeds 5 MB. |
| `json_file_too_large` | File exceeds 25 MB P0 parser cap. |
| `json_parse_error` | Invalid JSON input. |

Pass criteria:

1. Object keys are traversed in deterministic sorted order.
2. Arrays are traversed by ascending index.
3. Caveats are machine-readable and include `code`, `message`, `json_path`, and limit/observed values when applicable.
4. Cap conditions emit partial bounded output where allowed.
5. Invalid JSON and too-large JSON do not emit partial graphs.
6. Raw secret-like JSON values are not written to logs.

---

## 7. RIF compatibility fixture gates

P0 must validate ADR-016 using the fixture variants defined in `evaluation/fixtures/rif/README.md`.

| Fixture variant | Expected status | Required behavior |
|---|---|---|
| `no-rif` | `rif_not_deployed` | Cross-graph tools return explicit status and do not fabricate empty success. |
| `rif-incompatible` | `rif_incompatible` | Missing required fields/capabilities are listed. |
| `age-only-compatible` | `rif_shadow_empty` or `rif_compatible` | Resolver uses AGE path even when `rif_meta` shadows are empty/absent. |
| `shadow-compatible` | `rif_compatible` | Resolver can use compatibility shadows/views when populated. |
| `shadow-empty-no-age` | `rif_incompatible` | Empty shadows without usable AGE/API are not treated as success. |

Pass criteria:

1. DIF never directly depends on populated `rif_meta.file_nodes`, `rif_meta.method_nodes`, or `rif_meta.class_nodes`.
2. RIF status is persisted in `dif_meta.rif_compatibility_status` or equivalent.
3. Cross-graph MCP responses include explicit RIF status when compatibility is not ready.
4. RIF-owned schemas are not mutated by DIF tests or migrations.

---

## 8. `search_docs` expected behavior

P0 `search_docs` must be evidence-first.

Required result fields:

| Field | Required | Notes |
|---|---:|---|
| `corpus_id` | Yes | Must match authorized/admitted corpus. |
| `document_id` | Yes | Logical document identity. |
| `document_version_id` | Yes | Immutable version identity. |
| `node_id` or `passage_id` | Yes | Result identity. |
| `snippet` | Yes | Human-readable excerpt. |
| `anchor_id` | Yes | Primary source anchor. |
| `source_ref` | Yes | Canonical source ref. |
| `score` | Yes | Ranking score or deterministic fallback score. |
| `caveats` | Yes | Empty array if none. |

Required behavior:

1. Non-admitted corpora fail closed with `corpus_not_admitted`.
2. Results without source anchors are excluded.
3. Unsupported formats are not silently indexed.
4. Empty result sets are allowed only when accompanied by a clear no-evidence status.
5. Tool responses must not answer beyond retrieved evidence.

---

## 9. Audit, usage, and logging checks

### 9.1 Audit events

Audit events must capture security-relevant access without storing raw sensitive payloads.

Required audit dimensions:

- principal or service identity
- tenant/project
- corpus
- tool/API name
- parameters hash
- outcome/status
- latency
- returned source-anchor IDs or count
- timestamp

### 9.2 Usage events

Usage events must be separate from audit events and suitable for metering.

Required usage dimensions:

- project/corpus
- operation
- request count
- result count
- token or model usage when applicable
- storage/indexing counters when applicable
- timestamp

### 9.3 Logging

Logs may include:

- source path
- document ID
- anchor ID
- caveat code
- counts
- hashes
- latency

Logs must not include:

- raw JSON secret-like values
- credentials
- access tokens
- full private document contents by default

---

## 10. Baseline metrics

Do not set production SLOs before executable baselines exist. P0 should collect these baseline metrics:

| Metric | Purpose |
|---|---|
| Ingestion documents/sec | Parser throughput baseline. |
| Ingestion failure rate | Degenerate-run and parser-quality signal. |
| Anchor round-trip pass rate | Citation reliability. |
| Retrieval top-1 anchor hit rate | Search quality baseline. |
| Retrieval top-3 anchor hit rate | Search quality baseline. |
| JSON caveat count by code | Cap tuning and corpus-quality signal. |
| RIF compatibility status by environment | Deployment readiness. |
| Audit write success rate | Governance readiness. |
| Usage write success rate | Metering readiness. |

Initial P0 pass/fail should be based on deterministic functional gates, not arbitrary production-scale performance targets.

---

## 11. Phase-exit checklist

P0 evaluation is complete only when:

1. Golden corpus fixtures exist.
2. Golden query expectations exist.
3. Source-anchor round-trip tests exist for Markdown, TXT, DOCX, and JSON.
4. JSON cap/caveat tests cover all ADR-006 caveat codes.
5. RIF compatibility fixture tests cover all ADR-016 statuses.
6. Non-admitted corpus fail-closed behavior is tested.
7. `search_docs` excludes unanchored results.
8. Audit and usage event checks are implemented.
9. Degenerate ingestion run guard is tested.
10. Baseline metrics are captured from executable tests.

---

## 12. Next implementation artifacts

Create these after this plan is accepted:

| Order | Artifact | Purpose |
|---:|---|---|
| 1 | `tracking/phase-gate-status.md` | Track design, implementation, evaluation, and production-readiness gates. |
| 2 | `tracking/risk-register.md` | Track blockers, risks, owners, and mitigation dates. |
| 3 | `evaluation/golden/README.md` update or fixture files | Define executable golden corpus layout. |
| 4 | RIF fixture JSON/SQL files | Make ADR-016 fixture spec executable. |
| 5 | Executable `dif_meta` SQL migration | Implement schema design in `code/migrations/001_dif_meta_initial.sql`. |
| 6 | Test runner choice | Define actual command once code/toolchain exists. |
