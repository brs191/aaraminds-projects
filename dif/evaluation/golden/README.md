# DIF P0 Golden Corpus

**Status:** Initial fixture layout  
**Date:** 2026-07-08  
**Purpose:** Synthetic P0 golden corpus for deterministic ingestion, source-anchor, retrieval, JSON caveat, corpus admission, audit, and usage checks.

---

## 1. Layout

| Path | Purpose |
|---|---|
| `manifest.json` | Corpus, source, document, and expected ID-input metadata. |
| `golden-queries.json` | P0 `search_docs` golden queries and expected result behavior. |
| `expected-anchors.json` | Expected source-anchor records and resolver round-trip excerpts. |
| `expected-caveats.json` | Expected JSON caveat coverage and failure behavior. |
| `expected-audit-usage.json` | Expected audit/usage write dimensions, safe record behavior, and MCP call metering cases. |
| `expected-degenerate-runs.json` | Expected ingestion-run promotion decisions for healthy, empty, failed, anchorless, passageless, and non-complete runs. |
| `sources/admitted/architecture-overview.md` | Markdown fixture for heading and line anchors. |
| `sources/admitted/runbook.txt` | TXT fixture for line anchors. |
| `sources/admitted/requirements.docx.fixture.json` | DOCX paragraph fixture until executable DOCX parser/test harness exists. |
| `sources/admitted/service-config.json` | JSON fixture for JSONPath anchors and code-entity candidate text. |
| `sources/admitted/large-capped.json` | Compact synthetic JSON fixture that declares cap-triggering expectations. |
| `sources/admitted/invalid.json` | Invalid JSON fixture for `json_parse_error`. |
| `sources/restricted/restricted-corpus-sample.md` | Non-admitted corpus fixture for fail-closed behavior. |

---

## 2. Fixture policy

1. Fixtures are synthetic and contain no customer data, credentials, secrets, or private source content.
2. IDs in expectation files are stable aliases until extractor code generates deterministic hashes.
3. Actual deterministic IDs must be added after the first extractor implementation lands.
4. `requirements.docx.fixture.json` models DOCX paragraphs without committing a binary DOCX yet. Replace or supplement it with a generated `.docx` once the parser/test harness is chosen.
5. Large-cap behavior is represented compactly with declared generation metadata. The executable harness may generate oversized JSON from those declarations to avoid storing very large files in the repository.

---

## 3. Coverage map

| P0 requirement | Fixture / expectation |
|---|---|
| Markdown source-anchor round trip | `architecture-overview.md`, `expected-anchors.json` |
| TXT source-anchor round trip | `runbook.txt`, `expected-anchors.json` |
| DOCX paragraph anchor round trip | `requirements.docx.fixture.json`, `expected-anchors.json` |
| JSONPath source-anchor round trip | `service-config.json`, `expected-anchors.json` |
| JSON caveat behavior | `large-capped.json`, `invalid.json`, `expected-caveats.json` |
| `search_docs` grounded results | `golden-queries.json` |
| Non-admitted corpus fail-closed behavior | `restricted-corpus-sample.md`, `golden-queries.json` |
| Audit/usage dimensions | `manifest.json`, `expected-audit-usage.json` |
| Degenerate-run promotion guard | `expected-degenerate-runs.json` |

---

## 4. Next step

The next implementation step is to establish the implementation skeleton and project toolchain that:

1. defines the package/module layout under `../../code/`,
2. adds the first component-root unit test command,
3. adds lint/type-check commands if applicable,
4. and records the exact commands in `../../action_plan.md` and `../../.github/copilot-instructions.md`.
