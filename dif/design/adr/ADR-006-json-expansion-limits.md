# ADR-006: JSON Graph Expansion Limits

**Date:** 2026-07-08  
**Status:** Accepted for P0 design gate  
**Owners:** Engineering + QA  
**Related decisions:** D-008  
**Related docs:** `DECISIONS.md`, `dif_prd.md`, `design-decisions.md`, `action_plan.md`, `design/adr/ADR-007-source-anchor-contract.md`

---

## 1. Context

DIF treats JSON as a first-class P0 artifact because engineering corpora often include:

- service configuration
- policies as code
- environment inventories
- API metadata
- dependency/config references to code entities

JSON must participate in retrieval, source anchoring, and future `DESCRIBES` detection. However, JSON can be deeply nested, repetitive, machine-generated, or extremely large. Without expansion limits, a single pathological file could flood `dif_meta` with nodes/edges/passages and make indexing unpredictable.

This ADR defines deterministic JSON traversal, graph mapping, caps, caveats, and tests for P0.

---

## 2. Decision

DIF will ingest JSON deterministically with bounded graph expansion.

P0 JSON extraction will:

1. Parse valid JSON only.
2. Preserve JSONPath source anchors.
3. Emit bounded `document`, `section`, and `block` nodes.
4. Generate retrieval passages from selected JSON subtrees.
5. Cap depth, node count, array expansion, object property expansion, scalar length, and total extracted bytes.
6. Add explicit caveats when content is skipped, summarized, or capped.
7. Refuse index promotion for fully degenerate JSON extraction.

JSON extraction must never silently drop unsupported or capped content.

---

## 3. P0 JSON graph model

JSON remains a document artifact in the DIF graph.

Minimum mapping:

| JSON element | DIF node | Notes |
|---|---|---|
| root document | `document` | One per JSON file/version. |
| top-level object property | `section` or `block` | Section if object/array subtree is significant; block for scalar/small values. |
| nested object | `block` | Bounded by depth and node caps. |
| array | `block` | Expanded for first N representative elements; caveat records cap. |
| scalar | `block` | Preserved as normalized key/value text. |

P0 does not create first-class JSON entity nodes. Entity extraction from JSON remains future work; P0 only stores candidate text and JSONPath anchors for retrieval and later `DESCRIBES` detection.

---

## 4. Source anchors

Every JSON-derived node and retrieval passage must carry a JSONPath source anchor as defined by ADR-007.

Canonical source ref example:

```text
engineering-docs@v000001:json:config/services.json#$.services[0].name
```

Anchor rules:

1. Use canonical JSONPath with `$` root.
2. Object keys are emitted in sorted order for deterministic traversal.
3. Array indices are explicit.
4. Escaped/special keys must use bracket notation.
5. If an entire object/array is represented as one block, the anchor is the subtree JSONPath.
6. If a scalar is represented, the anchor is the scalar JSONPath.

---

## 5. Expansion caps

Default P0 caps:

| Cap | Default | Behavior when exceeded |
|---|---:|---|
| Maximum nesting depth | 12 | Stop descending; emit caveat on parent block. |
| Maximum emitted JSON blocks per document | 2,000 | Stop emitting additional blocks; record truncation caveat. |
| Maximum object properties per object | 200 | Emit sorted first 200 keys; record skipped count. |
| Maximum array elements per array | 100 | Emit first 100 elements; record skipped count. |
| Maximum scalar string length | 8,192 characters | Truncate scalar text for retrieval; preserve source anchor and caveat. |
| Maximum normalized block text | 16,384 characters | Truncate retrieval text; preserve source anchor and caveat. |
| Maximum JSON file size for P0 parser | 25 MB | Reject file as unsupported for P0 with explicit caveat/status. |
| Maximum total emitted text per document | 5 MB | Stop passage generation; record truncation caveat. |

These defaults are P0 operational guardrails. They may become corpus-level configuration later, but P0 should keep them static and testable.

---

## 6. Deterministic traversal

Traversal rules:

1. Object keys sorted lexicographically by Unicode code point.
2. Arrays traversed by ascending index.
3. Null, boolean, number, and string scalar formatting is canonical.
4. Whitespace in emitted block text is normalized.
5. Output nodes are sorted by JSONPath before persistence/NDJSON emission.
6. Caveats are sorted deterministically.

Repeated extraction of unchanged JSON must produce byte-identical output.

---

## 7. Caveat model

Every cap or unsupported condition must produce a machine-readable caveat.

Required caveat fields:

| Field | Meaning |
|---|---|
| `code` | Machine-readable caveat code. |
| `message` | Human-readable explanation. |
| `json_path` | JSONPath where caveat occurred. |
| `limit` | Limit that was hit, when applicable. |
| `observed` | Observed value, when applicable. |

Required caveat codes:

| Code | Meaning |
|---|---|
| `json_depth_capped` | Max depth reached. |
| `json_block_count_capped` | Max emitted blocks reached. |
| `json_object_properties_capped` | Object property cap reached. |
| `json_array_elements_capped` | Array element cap reached. |
| `json_scalar_truncated` | Scalar string truncated. |
| `json_block_text_truncated` | Normalized block text truncated. |
| `json_total_text_capped` | Total emitted text cap reached. |
| `json_file_too_large` | File exceeds P0 parser size cap. |
| `json_parse_error` | Invalid JSON. |

---

## 8. Failure behavior

| Condition | Behavior |
|---|---|
| Invalid JSON | Mark file extraction failed with `json_parse_error`; do not emit partial graph for that file. |
| File larger than P0 cap | Mark unsupported for P0 with `json_file_too_large`; do not parse. |
| One JSON file fails but corpus has valid docs | Ingestion run may continue, with caveat/failure record. |
| All files fail or emit zero usable nodes | Degenerate-run guard blocks index promotion. |
| Cap reached | Emit partial bounded graph with caveats; do not fail the entire file solely because a cap was reached. |

No failure may be silently converted into success.

---

## 9. Retrieval passage rules

JSON retrieval passages must be human-readable and source-resolvable.

Recommended text shape:

```text
JSON path: $.services[0]
name: payments-api
owner: platform
class: com.example.payments.PaymentService
```

Rules:

1. Include JSONPath in passage metadata.
2. Include nearby key context for scalar values.
3. Do not flatten the entire file into one passage unless it is small.
4. Preserve enough structured text for code-entity detection.
5. Apply block text cap before embedding/FTS.

---

## 10. Relationship to `DESCRIBES`

JSON artifacts participate in future `DESCRIBES` detection.

Candidate examples:

- class names
- package names
- method-like strings
- file paths
- service names that map to RIF entities
- endpoint paths

P0 may store candidate text and JSONPath anchors. P1 resolves candidates through the RIF compatibility layer.

`DESCRIBES` edges from JSON must preserve:

- JSON anchor ID
- JSONPath source ref
- match mode
- confidence
- caveats caused by truncation or expansion caps

---

## 11. Schema implications

`dif_meta.source_anchors` must support `json_path`.

JSON extraction metadata should record:

| Field | Meaning |
|---|---|
| `parser_name` | JSON parser identifier. |
| `parser_version` | Parser/extractor version. |
| `max_depth` | Applied depth cap. |
| `max_blocks` | Applied block cap. |
| `max_array_elements` | Applied array cap. |
| `max_object_properties` | Applied property cap. |
| `caveat_count` | Number of caveats emitted. |
| `truncated` | True if any cap caused truncation/skipping. |

---

## 12. Security and privacy

JSON may contain secrets or credentials.

P0 must not log raw JSON values by default. Operational logs may include:

- path
- JSONPath
- caveat code
- counts
- hashes

P0 should include a simple secret-pattern caveat/redaction pass for retrieval text before embedding/FTS if obvious secret keys are encountered, such as:

- `password`
- `secret`
- `token`
- `apiKey`
- `clientSecret`
- `privateKey`

Secret handling must be refined in ADR-013. This ADR only requires no raw JSON logging and explicit caveats/redaction for obvious secret-like keys in emitted retrieval passages.

---

## 13. Tests required

P0 tests:

1. Object keys are traversed in deterministic sorted order.
2. Arrays are traversed by ascending index.
3. JSONPath anchors round-trip to source values.
4. Repeated extraction of unchanged JSON is byte-identical.
5. Maximum depth cap emits `json_depth_capped`.
6. Maximum block count cap emits `json_block_count_capped`.
7. Object property cap emits `json_object_properties_capped`.
8. Array element cap emits `json_array_elements_capped`.
9. Scalar truncation emits `json_scalar_truncated`.
10. Invalid JSON emits `json_parse_error`.
11. Oversized JSON emits `json_file_too_large`.
12. Capped JSON still emits valid source-anchored retrieval passages.
13. Secret-like keys are not logged raw by default.
14. Degenerate all-failed JSON corpus cannot promote an index.

---

## 14. Consequences

Positive:

- JSON becomes safe to admit in P0.
- Pathological JSON cannot flood `dif_meta`.
- Retrieval and future `DESCRIBES` can cite exact JSONPath anchors.
- Caps and caveats are testable.

Trade-offs:

- Some large/generated JSON content is partially represented in P0.
- P0 caps are static rather than corpus-tuned.
- Secret redaction is conservative and must be improved during security hardening.

---

## 15. Acceptance criteria

ADR-006 is accepted when:

- P0 expansion caps are defined.
- Deterministic traversal rules are defined.
- JSONPath source-anchor behavior is defined.
- Caveat codes are defined.
- Failure behavior is explicit.
- Retrieval passage rules are defined.
- `DESCRIBES` relationship is defined.
- Required tests are listed.

