# ADR-005: Parser Strategy for P0 Formats

**Date:** 2026-07-13  
**Status:** Accepted for P0 exit  
**Owners:** Engineering + QA  
**Related docs:** `action_plan.md`, `evaluation/p0-evaluation-plan.md`, `code/libs/extraction`

---

## 1. Context

DIF P0 admits Markdown, TXT, DOCX, and JSON. Each format needs deterministic graph records, source anchors, extraction caveats where applicable, and repeatable tests before it can be considered supported.

---

## 2. Decision

DIF P0 uses deterministic in-process parsers and fixture-backed seams:

| Format | P0 strategy | Anchor type | Notes |
|---|---|---|---|
| Markdown | Line-oriented heading/block extraction | `md` line refs | Emits document/section/block nodes and heading paths. |
| TXT | Line-oriented block extraction | `txt` line refs | Emits document/block nodes and passages. |
| DOCX | Paragraph-model fixture adapter | `docx` paragraph refs | Uses committed paragraph fixture seam; binary parser is deferred. |
| JSON | Standard JSON parser with bounded traversal | `json` JSONPath refs | Sorted object traversal, ascending arrays, ADR-006 caveats. |

Unsupported formats must not be silently admitted. PDF/PPTX remain P1 and XLSX remains v1.5.

---

## 3. Consequences

- Parser output must be byte-stable for unchanged fixtures.
- Every emitted node/passage must have a source anchor.
- Format support requires parser, source-anchor type, graph mapping, caveats where applicable, tests, and cost profile.
- Future binary DOCX/PDF/PPTX/XLSX parsers must preserve the same source-anchor contract.

---

## 4. P0 evidence

- `code/libs/extraction`
- `code/libs/sourceanchors`
- `evaluation/source_anchor_roundtrip.py`
- `evaluation/json_caveat_checks.py`
- `evaluation/run_p0.py`
