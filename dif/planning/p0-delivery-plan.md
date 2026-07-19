# DIF P0 Delivery Plan

**Status:** P0 execution complete; retained as historical delivery evidence  
**Date:** 2026-07-08  
**Owners:** Engineering + QA + Platform + Security  
**Source of truth:** `../action_plan.md` remains the operating source of truth. Update that file whenever this plan changes materially.

---

## 1. Purpose

This plan translated the accepted P0 design baseline into an engineering execution sequence. P0 execution is now complete; `../action_plan.md` remains the operating source of truth for current status and next work.

P0 delivery produced a runnable DIF baseline that can:

- create and own `dif_meta` without mutating RIF schemas
- admit only uniformly readable corpora
- ingest Markdown, TXT, DOCX, and JSON deterministically
- preserve source anchors for every indexed node and retrieval passage
- enforce JSON expansion caps and caveats
- expose anchored `search_docs`
- write audit and usage events separately
- detect RIF compatibility status without assuming populated `rif_meta` shadows
- run repeatable golden/evaluation checks

P0 does not implement `DESCRIBES`, `docs_for_code`, `code_for_doc`, or `drift_report`. Those remain blocked until the P1 federation gates advance past P1-01 candidate detection.

---

## 2. P0 entry criteria

P0 implementation may start because these design/planning gates exist:

| Entry criterion | Status | Evidence |
|---|---|---|
| Accepted source ACL posture | Complete | `../design/adr/ADR-003-source-acl-posture.md` |
| Accepted JSON expansion limits | Complete | `../design/adr/ADR-006-json-expansion-limits.md` |
| Accepted source-anchor contract | Complete | `../design/adr/ADR-007-source-anchor-contract.md` |
| Accepted RIF compatibility contract | Complete | `../design/adr/ADR-016-rif-compatibility-layer.md` |
| Initial `dif_meta` schema design | Complete | `../code/migrations/001_dif_meta_initial_design.md` |
| P0 evaluation plan | Complete | `../evaluation/p0-evaluation-plan.md` |
| Phase-gate tracker | Complete | `../tracking/phase-gate-status.md` |
| Risk register | Complete | `../tracking/risk-register.md` |

Implementation must stop and update ADRs if code-level work contradicts any accepted decision.

---

## 3. Workstream overview

| Workstream | Owner | Purpose | Depends on | Primary evidence |
|---|---|---|---|---|
| WS-1 Schema and persistence | Engineering + Platform | Create `dif_meta` SQL migration and data-access boundary. | Schema design | SQL migration and idempotency test. |
| WS-2 Corpus admission | Engineering + Security | Enforce uniformly readable corpus gate. | WS-1 | `corpus_not_admitted` fail-closed test. |
| WS-3 Golden fixtures and harness | QA + Engineering | Establish repeatable evaluation corpus and test runner. | WS-1 | Golden fixtures, expected anchors, documented command. |
| WS-4 Source anchors | Engineering + QA | Persist and resolve Markdown/TXT/DOCX/JSON anchors. | WS-1, WS-3 | Round-trip tests. |
| WS-5 JSON ingestion | Engineering + QA | Implement bounded deterministic JSON extraction. | WS-4 | Caveat tests for all ADR-006 codes. |
| WS-6 Markdown/TXT/DOCX ingestion | Engineering + QA | Implement deterministic document graph extraction. | WS-4 | Golden extraction tests. |
| WS-7 Retrieval passages and `search_docs` | Engineering + QA | Return source-anchored evidence. | WS-4, WS-5, WS-6 | Anchored retrieval tests. |
| WS-8 Audit, usage, and safe logging | Engineering + Security + Product | Add governance and metering evidence. | WS-1, WS-7 | Audit/usage write tests and logging checks. |
| WS-9 RIF compatibility gate | Engineering + Platform + QA | Detect RIF statuses and protect future federation. | WS-1, WS-3 | Executable ADR-016 fixture tests. |
| WS-10 MCP/API skeleton | Engineering + Platform + Security | Expose P0 `search_docs` through governed tool/API boundary. | WS-7, WS-8 | MCP/API contract tests. |

---

## 4. Execution sequence

### Step 1: Create executable `dif_meta` migration

Deliverable:

```text
../code/migrations/001_dif_meta_initial.sql
```

Scope:

1. Create schema `dif_meta`.
2. Create all P0 tables from `001_dif_meta_initial_design.md`.
3. Add primary keys, foreign keys, uniqueness rules, status checks, and essential indexes.
4. Keep vector columns out until embedding model/dimension is pinned.
5. Include FTS-ready text columns/indexes where useful for P0 retrieval.
6. Do not create, alter, or drop objects in `rif` or `rif_meta`.

Validation checkpoint:

- migration runs once successfully
- migration runs twice idempotently
- migration succeeds when RIF schemas are absent
- migration succeeds when RIF schemas are present
- table inventory matches the design document

---

### Step 2: Establish project toolchain and test harness

Deliverables:

```text
../code/README.md
../code/testdata/
```

Exact paths depend on implementation language selection, but the first runnable setup must define:

1. package/module layout
2. migration runner command
3. unit test command
4. targeted evaluation command
5. lint/type-check command, if applicable

After commands exist, update:

- `../action_plan.md`
- `../.github/copilot-instructions.md`
- `../tracking/phase-gate-status.md`

Validation checkpoint:

- commands run from the target project root, not workspace root
- commands are deterministic and safe for local developer use

---

### Step 3: Create golden corpus and expected outputs

Deliverables:

```text
../evaluation/golden/
```

Required fixture coverage:

1. Markdown heading and line anchors.
2. TXT line anchors.
3. DOCX paragraph anchors.
4. JSONPath anchors.
5. JSON cap/caveat cases.
6. Invalid JSON.
7. Non-admitted corpus sample.

Validation checkpoint:

- fixture files contain no customer data, secrets, credentials, or private source content
- expected anchors and expected query results are deterministic

---

### Step 4: Implement corpus admission

Scope:

1. Store corpus admission metadata.
2. Require `readability_model = uniform_readable` for v1.
3. Reject or fail closed for non-admitted corpora.
4. Return `corpus_not_admitted` for retrieval/MCP calls against non-admitted corpora.

Validation checkpoint:

- admitted corpus can ingest/search
- non-admitted corpus cannot ingest/promote/search
- audit event records the failed-closed outcome when applicable

---

### Step 5: Implement source anchors

Scope:

1. Persist source anchors for Markdown, TXT, DOCX, and JSON.
2. Generate canonical `source_ref` strings.
3. Implement resolver behavior for supported anchor types.
4. Return structured resolver failures:
   - `anchor_not_found`
   - `document_version_not_found`
   - `source_content_unavailable`
   - `anchor_out_of_range`
   - `anchor_type_unsupported`
   - `content_hash_mismatch`

Validation checkpoint:

- every indexed node/passage has `anchor_id` and `source_ref`
- all supported format anchors round trip to expected text/subtree
- resolver failures are structured, not silent empty results

---

### Step 6: Implement JSON ingestion

Scope:

1. Deterministic JSON traversal with sorted object keys and ascending array indices.
2. JSONPath anchors for every JSON-derived node/passage.
3. Bounded block/passage generation.
4. Caveats for every ADR-006 cap and failure.
5. Secret-safe logging.

Validation checkpoint:

- all required ADR-006 caveat codes are covered by tests
- invalid and too-large JSON do not emit partial graphs
- repeated extraction of unchanged JSON produces stable ordered output

---

### Step 7: Implement Markdown, TXT, and DOCX ingestion

Scope:

1. Create document, section, and block nodes.
2. Create `CONTAINS` edges.
3. Create retrieval passages mapped to anchors.
4. Preserve document versions and content hashes.
5. Block index promotion for degenerate runs.

Validation checkpoint:

- golden extraction outputs match expected IDs/anchors
- empty or all-failed runs cannot promote an index

---

### Step 8: Implement retrieval and `search_docs`

Scope:

1. Search admitted corpus passages.
2. Return only anchored results.
3. Include required result fields:
   - `corpus_id`
   - `document_id`
   - `document_version_id`
   - `node_id` or `passage_id`
   - `snippet`
   - `anchor_id`
   - `source_ref`
   - `score`
   - `caveats`
4. Avoid unsupported claims when no evidence exists.

Validation checkpoint:

- golden queries return expected source refs
- unknown query returns no unsupported answer
- unanchored results are excluded

---

### Step 9: Implement audit, usage, and logging checks

Scope:

1. Write audit events for MCP/API/security-relevant operations.
2. Write usage events separately for metering.
3. Hash parameters where raw payloads are not needed.
4. Do not log raw JSON secret-like values, credentials, tokens, or full private document contents by default.

Validation checkpoint:

- audit write test passes
- usage write test passes
- audit and usage events are not conflated
- logging checks cover prohibited raw-value patterns

---

### Step 10: Implement RIF compatibility status checks

Scope:

1. Detect `rif_not_deployed`.
2. Detect `rif_incompatible`.
3. Detect `rif_shadow_empty`.
4. Detect `rif_compatible`.
5. Prefer AGE-backed resolver/view when shadows are empty.
6. Store status in `dif_meta.rif_compatibility_status`.
7. Do not mutate RIF-owned schemas.

Validation checkpoint:

- all ADR-016 fixture variants pass
- cross-graph tools remain blocked unless compatible
- responses expose explicit RIF status instead of false empty success

---

### Step 11: Implement MCP/API skeleton

P0 MCP/API scope is intentionally narrow:

1. `search_docs`
2. health/status endpoint or tool
3. RIF compatibility status surface

Validation checkpoint:

- corpus authorization/admission is enforced
- responses include source anchors
- audit/usage events are emitted
- unsupported cross-graph tools return explicit unavailable/blocked status

---

## 5. Dependency rules

1. Do not implement P1 federation before RIF compatibility fixture tests pass.
2. Do not add vector schema until embedding model and dimension are pinned.
3. Do not add per-user ACL propagation in v1.
4. Do not index mixed-permission corpora.
5. Do not mutate RIF schemas from DIF migrations.
6. Do not return retrieval results without source anchors.
7. Do not log raw secrets, credentials, tokens, or full source documents by default.

---

## 6. P0 milestone plan

| Milestone | Scope | Exit evidence |
|---|---|---|
| M0: Executable foundation | SQL migration, toolchain, test harness | Migration and tests can run locally with documented commands. |
| M1: Governance baseline | Corpus admission, audit, usage, safe logging | Fail-closed and audit/usage tests pass. |
| M2: Evidence graph | Source anchors and Markdown/TXT/DOCX/JSON ingestion | Golden extraction and round-trip tests pass. |
| M3: Retrieval baseline | Retrieval passages and `search_docs` | Golden query tests pass with anchored results. |
| M4: RIF compatibility gate | RIF status checks and fixtures | ADR-016 fixture matrix passes. |
| M5: P0 exit review | Gate tracker, risk register, commands, evidence | P0 exit checklist is complete. |

---

## 7. Validation command policy

P0 validation is executable and CI-backed. Use the full P0 golden gate from the repository root:

```bash
cd /Users/rb692q/projects/aaraminds-projects/dif
python3 evaluation/run_p0.py
```

Targeted component validation runs from the Go component root:

```bash
cd /Users/rb692q/projects/aaraminds-projects/dif/code
go test ./...
go build ./...
```

Migration idempotency validation replays the executable SQL twice against a scratch PostgreSQL database and verifies the `dif_meta` table inventory. The CI baseline in `../.github/workflows/ci.yml` runs the P0 golden gate plus PostgreSQL-backed migration replay without deployment, registry publishing, Azure login, or secret usage.

Command policy:

1. Run commands from the target project root or `code/` component root, not the workspace root.
2. Keep `../action_plan.md` and `../.github/copilot-instructions.md` synchronized when commands change.
3. Prefer `python3 evaluation/run_p0.py` as the deterministic P0 gate.

---

## 8. Current P1 handoff

The first implementation task after P0 exit was completed:

```text
P1-01 code-entity candidate detector
```

It preserves source-anchor evidence and keeps detected document references unresolved until resolver evidence exists. The current next task is P1-02 RIF resolver and `DESCRIBES` edges.

Definition of done:

1. Candidate detection is deterministic.
2. Every candidate preserves source-anchor evidence.
3. Candidates remain unresolved until RIF resolver evidence exists.
4. P1-02 uses these unresolved candidates as resolver input.
