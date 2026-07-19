# DIF Libraries

Shared code should live here once implementation begins.

Planned libraries:

| Library area | Purpose |
|---|---|
| source anchors | Implemented in `sourceanchors`; canonical source refs, deterministic anchor IDs/content hashes, P0 Markdown/TXT/DOCX/JSON round-trip resolvers, and explicit failure statuses. |
| ingestion runs | Implemented in `ingestionruns`; lifecycle statuses, output counts, promotion decisions, non-promotable errors, and degenerate-run guard semantics. |
| extraction contracts | Implemented initially in `extraction`; deterministic Markdown/TXT/DOCX/JSON records, source anchors, retrieval passages, stable IDs/hashes, caveats, and `CONTAINS` edges. |
| graph emitter | Implemented in `graphemit`; validation-first byte-stable NDJSON records for documents, source anchors, nodes, `CONTAINS` edges, retrieval passages, and caveats. |
| retrieval primitives | Implemented in `retrieval`; anchored-only P0 lexical retrieval, corpus admission enforcement, deterministic ranking, and explicit `ok` / `no_evidence` / `corpus_not_admitted` statuses. |
| embedding providers | Implemented in `embeddings`; provider interface, deterministic offline hash provider, request validation, normalized vectors, and non-PII usage placeholders. |
| search service | Implemented in `searchdocs`; service-layer `search_docs` request/response contract, scope validation, admission-before-retrieval guard, anchored-only results, scores, caveats, and fail-closed statuses. |
| MCP/API boundary | Implemented in `mcpapi`; P0 bearer auth, required field validation, HTTP/tool-style `search_docs` transport, service routing, structured errors, and grounded response envelopes. |
| audit/usage writes | Implemented in `auditusage`; separated audit and non-PII usage write shapes, SQL writer seam, safe parameter hashing, source-ref audit recording, and MCP/API governance integration. |
| health/readiness | Implemented in `health`; Postgres connectivity checks, `dif_meta` schema inventory readiness, informational RIF status, secret-safe errors, and HTTP handlers. |
| RIF compatibility | Implemented in `rifcompat`; ADR-016 status assessment, AGE/shadow fallback handling, deterministic code-entity resolver contract, NUL-separated RIF node/edge ID helpers, and DIF-owned status persistence. |
| code-entity candidates | Implemented in `codeentities`; deterministic unresolved candidate detection for anchored document blocks, syntax-level match metadata, caveats, and SQL persistence to `dif_meta.code_entity_candidates` without minting RIF nodes. |
| RIF resolver / `DESCRIBES` edges | Implemented in `codeentities` (`resolver.go`); resolves candidates against `rifcompat` reports (qualified-name/source-path/simple-name/fuzzy), explicit ambiguous/unresolved/`rif_unavailable` outcomes, evidence-gated `DESCRIBES` edge creation with shared edge-ID semantics, per-corpus measured resolution rates, and SQL writers for `dif_meta.edges` and candidate resolution updates (migration 002). |
| corpus admission | Uniform-readable v1 corpus/source admission gate and denied-audit intent. |

Avoid importing legacy RIF package namespaces directly. Use neutral AaraMinds modules or compatibility tests for shared semantics.
