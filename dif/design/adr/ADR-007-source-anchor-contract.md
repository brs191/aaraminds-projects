# ADR-007: Source Anchor Contract

**Date:** 2026-07-08  
**Status:** Accepted for P0 design gate  
**Owners:** Engineering + QA  
**Related decisions:** D-008, D-010  
**Related docs:** `DECISIONS.md`, `dif_prd.md`, `design-decisions.md`, `action_plan.md`

---

## 1. Context

DIF's product promise is evidence-backed document intelligence. Every retrieval result, MCP result, and future agent claim must point back to source content that can be resolved and inspected.

P0 supports:

- Markdown
- TXT
- DOCX
- JSON

P1+ adds:

- PDF
- PPTX
- XLSX at v1.5

This ADR defines the source-anchor contract for P0 and extension rules for later formats.

---

## 2. Decision

DIF will use explicit, persisted source anchors for every indexed document node and retrieval passage.

No source anchor means:

- the format is not admitted,
- the node/passage is not eligible for retrieval,
- and agent/MCP responses cannot cite it.

Every returned result from `search_docs` must include at least one resolvable source anchor. Future claim-block agent responses must include one or more source anchors per claim.

---

## 3. Canonical source reference

DIF source refs use a stable, parseable format:

```text
corpus_id@document_version_id:anchor_type:anchor_payload
```

Examples:

```text
engineering-docs@v000001:md:architecture/overview#L12-L18
engineering-docs@v000001:txt:policies/access.txt#L4-L9
engineering-docs@v000001:docx:requirements.docx#p17
engineering-docs@v000001:json:config/services.json#$.services[0].name
```

Rules:

1. `corpus_id` is stable and scoped to a DIF corpus.
2. `document_version_id` identifies an immutable document version.
3. `anchor_type` is one of the admitted anchor types.
4. `anchor_payload` is format-specific and must be resolver-safe.
5. Source refs must not depend on mutable display titles alone.

The storage schema may also keep structured anchor fields separately; the string form is for external references, logs, audit, MCP, and tests.

---

## 4. P0 anchor types

| Format | Anchor type | Required payload | Notes |
|---|---|---|---|
| Markdown | `md` | repo/path + heading path + line start/end | Heading path improves human readability; line range is the primary resolver. |
| TXT | `txt` | repo/path + line start/end | Plain line ranges only. |
| DOCX | `docx` | file path + paragraph index and optional heading path | Paragraph index is required because line numbers are not native to DOCX. |
| JSON | `json` | file path + JSONPath | JSONPath is the primary resolver; byte/span offsets may be added later. |

P1+ planned anchor types:

| Format | Anchor type | Required payload |
|---|---|---|
| PDF | `pdf` | page + block ID or bounding box |
| PPTX | `pptx` | slide + shape/text block ID |
| XLSX | `xlsx` | sheet + cell/range |

---

## 5. Source-anchor table fields

`dif_meta.source_anchors` or equivalent must support at least:

| Field | Required | Meaning |
|---|---:|---|
| `anchor_id` | Yes | Stable anchor ID. |
| `corpus_id` | Yes | Corpus scope. |
| `document_id` | Yes | Logical document. |
| `document_version_id` | Yes | Immutable document version. |
| `source_id` | Yes | Source file/object identity. |
| `anchor_type` | Yes | `md`, `txt`, `docx`, `json`, etc. |
| `source_ref` | Yes | Canonical external source ref string. |
| `path` | Yes | Repo-relative path, mounted path, or connector source path. |
| `heading_path` | Optional | Markdown/DOCX heading path where available. |
| `line_start` | Optional | Required for Markdown/TXT. |
| `line_end` | Optional | Required for Markdown/TXT. |
| `paragraph_index` | Optional | Required for DOCX. |
| `json_path` | Optional | Required for JSON. |
| `page_number` | Future | Required for PDF. |
| `bounding_box` | Future | Required where PDF page bbox is used. |
| `slide_number` | Future | Required for PPTX. |
| `shape_id` | Future | Required for PPTX shape-level anchors. |
| `sheet_name` | Future | Required for XLSX. |
| `cell_range` | Future | Required for XLSX. |
| `content_hash` | Yes | Hash of anchored source excerpt or normalized block. |
| `extractor_version` | Yes | Parser/extractor version that produced the anchor. |
| `caveats` | Optional | Unsupported content, inferred location, truncation, or parser limitations. |

---

## 6. Anchor IDs

Anchor IDs must be deterministic.

Recommended algorithm:

```text
anchor_id = sha256(corpus_id + NUL + document_version_id + NUL + anchor_type + NUL + normalized_anchor_payload)
```

The normalized payload must be stable across repeated extraction of unchanged content.

Examples:

- Markdown/TXT: normalized path + line start + line end.
- DOCX: normalized path + paragraph index + heading path if present.
- JSON: normalized path + canonical JSONPath.

---

## 7. Round-trip resolver behavior

The resolver must take `source_ref` or `anchor_id` and return:

| Field | Required | Meaning |
|---|---:|---|
| `anchor_id` | Yes | Resolved anchor. |
| `source_ref` | Yes | Canonical source ref. |
| `document_version_id` | Yes | Immutable version. |
| `excerpt` | Yes | Source content excerpt covered by the anchor. |
| `content_hash` | Yes | Hash of returned excerpt. |
| `caveats` | Optional | Any known limitations. |

Resolution failure must be explicit.

Allowed failure statuses:

| Status | Meaning |
|---|---|
| `anchor_not_found` | Anchor ID/source ref does not exist. |
| `document_version_not_found` | Referenced immutable document version is missing. |
| `source_content_unavailable` | Metadata exists but source blob/content is unavailable. |
| `anchor_out_of_range` | Stored range no longer resolves against stored content. |
| `anchor_type_unsupported` | No resolver exists for that anchor type. |
| `content_hash_mismatch` | Resolved excerpt hash differs from stored hash. |

Failures must not be silently converted into empty results.

---

## 8. Retrieval and MCP requirements

`search_docs` results must include:

| Field | Required | Meaning |
|---|---:|---|
| `document_id` | Yes | Logical document. |
| `document_version_id` | Yes | Immutable version. |
| `anchor_id` | Yes | Primary source anchor. |
| `source_ref` | Yes | Canonical source ref. |
| `excerpt` | Yes | Bounded source excerpt. |
| `score` | Yes | Retrieval score. |
| `signals` | Yes | Retrieval signals used. |
| `caveats` | Optional | Extraction/retrieval limitations. |

MCP responses must return source refs in structured fields, not only inside natural-language text.

---

## 9. Agent claim requirements

Future agent responses must use claim blocks:

```json
{
  "text": "The policy requires all production changes to be reviewed.",
  "source_refs": ["engineering-docs@v000001:md:policy/change-management.md#L12-L14"],
  "caveats": []
}
```

Rules:

1. Every claim has `source_refs` with minimum length 1.
2. Every `source_ref` must resolve.
3. Unsupported claims are dropped or converted into caveats.
4. A response with no supported claim fails closed.

---

## 10. Relationship to `DESCRIBES`

`DESCRIBES` edges link document blocks to RIF code nodes. Each `DESCRIBES` edge must keep the source anchor where the code-entity candidate appeared.

Minimum fields:

| Field | Meaning |
|---|---|
| `from_doc_node_id` | DIF block/section node. |
| `to_code_node_id` | RIF node ID. |
| `anchor_id` | DIF source anchor where the reference appeared. |
| `source_ref` | DIF source ref. |
| `code_source_ref` | RIF code source ref. |
| `match_mode` | qualified-name, source-path, node-id, simple-name, fuzzy. |
| `confidence` | exact or inferred. |
| `caveats` | Ambiguity or unresolved limitations. |

This lets `docs_for_code`, `code_for_doc`, and `drift_report` return both document anchors and code source refs.

---

## 11. Format admission rule

A format is not admitted until it defines:

1. Parser.
2. Source-anchor type.
3. Source-anchor resolver.
4. Graph node mapping.
5. Extraction caveats.
6. Golden tests.
7. Cost/performance profile.

For P0, Markdown, TXT, DOCX, and JSON must satisfy this rule.

---

## 12. Logging, audit, and privacy

Audit events may include:

- `anchor_id`
- `source_ref`
- document/corpus IDs
- tool name
- parameters hash
- outcome

Audit and operational logs must not include raw enterprise document text by default.

Usage events are separate from audit events and may include counts, latency, and status codes, but not raw source excerpts.

---

## 13. Tests required

P0 tests:

1. Markdown source ref round-trips to exact line range.
2. TXT source ref round-trips to exact line range.
3. DOCX source ref round-trips to paragraph content.
4. JSON source ref round-trips via JSONPath.
5. Anchor IDs are deterministic across repeated extraction.
6. Changed content creates a new document version and does not mutate old anchors.
7. Missing anchor returns `anchor_not_found`.
8. Out-of-range anchor returns `anchor_out_of_range`.
9. Hash mismatch returns `content_hash_mismatch`.
10. `search_docs` returns only source-anchored results.
11. MCP response exposes `anchor_id` and `source_ref` structurally.
12. Logs do not contain raw document text by default.

---

## 14. Consequences

Positive:

- Makes citation integrity testable.
- Prevents ungrounded retrieval/agent responses.
- Gives MCP clients deterministic source refs.
- Gives future `DESCRIBES` edges traceable provenance.

Trade-offs:

- Parsers must preserve enough location information.
- DOCX anchors are paragraph-based until a stronger layout model exists.
- PDF/PPTX/XLSX cannot be admitted until their anchor resolvers are defined.

---

## 15. Acceptance criteria

ADR-007 is accepted when:

- P0 anchor types are defined.
- Canonical `source_ref` format is defined.
- Source-anchor table fields are defined.
- Anchor ID algorithm is defined.
- Round-trip resolver behavior and failure statuses are defined.
- `search_docs` source-anchor requirements are defined.
- Agent claim-block citation requirements are defined.
- `DESCRIBES` relationship to anchors is defined.
- P0 tests are listed.

