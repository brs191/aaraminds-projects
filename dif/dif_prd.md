# DIF — Documents Intelligence Factory
## Product Requirements Document (PRD)

**Version:** 0.3 (Draft)
**Date:** 2026-07-08
**Owner:** AaraMinds
**Status:** Draft — pending review
**Sibling product:** RIF (Repo Intelligence Factory)
**v0.2 changes:** market-trend review applied — parsing router replaces hand-built PDF extraction, rerank stage added, low-level retrieval tools added, MCP 2026-07-28 spec + OAuth 2.1 targeted, structural citations + groundedness scoring, embedding default refreshed, long-context routing note.
**v0.3 changes (D-007):** RIF+DIF federation promoted from v2 candidate to core v1 architecture — DIF deploys per project into the project's existing RIF Postgres (`dif_meta` beside `rif_meta`), `DESCRIBES` doc→code edges added (P1), cross-graph tools `docs_for_code`/`code_for_doc` (P1) and `drift_report` (P2), shared NodeIdComputer now a hard requirement.
**v0.3.1 changes (D-008):** `.json` added as a P0 first-class artifact (JSONPath anchors, expansion caps via ADR); `.xlsx` committed at v1.5 with cell/range anchors; format admission policy adopted from design-decisions DD-01.

---

## 1. Summary

DIF turns an organization's document corpus into a queryable, citation-grounded knowledge graph — the same way RIF does for code. It ingests enterprise documents (Word, PDF, PowerPoint) and engineering docs (Markdown, ADRs, wiki exports), extracts structure and cross-references into a document graph, embeds content for hybrid retrieval, and exposes the result through MCP tools and a citation-gated agent service.

The product answers questions no keyword search or naive RAG can: *"Which contracts reference the deprecated SLA clause?"*, *"What downstream documents are impacted if this architecture decision changes?"*, *"Show me every claim about data retention across our policy set, with sources."*

DIF reuses RIF's proven architecture: ingestion → extractors → Postgres + pgvector storage → hybrid retriever (vector + FTS + graph, RRF fusion) → MCP server → agent service. Same stack, same ops model, new extraction domain.

**DIF is code-aware by design (D-007).** DIF deploys per project, into the same Postgres database where that project's RIF code graph already lives (`dif_meta` beside `rif_meta`). Document blocks link to code entities via `DESCRIBES` edges, making cross-graph questions first-class: *"which documents describe this code?"*, *"this method changed — which docs are now stale?"* (documentation drift), *"this policy changed — which code is affected?"*. DIF works standalone on doc-only corpora, but co-working with RIF is the default deployment and the core differentiator — no competitor can answer these questions without owning a code graph.

## 2. Problem statement

Enterprise document estates share the pathologies of legacy codebases:

- **Knowledge is trapped in format.** The answer exists, but it's on slide 34 of a deck, in a table inside a PDF, or split across three versions of a Word doc.
- **Cross-references are invisible.** Documents cite, supersede, and contradict each other with no machine-readable link structure. Impact analysis ("what breaks if we change this policy?") is manual and unreliable.
- **Naive RAG hallucinates and can't cite.** Chunk-and-embed pipelines lose document structure (headings, tables, clause hierarchy) and produce answers without verifiable provenance. In contract, compliance, and engineering-decision contexts, an uncited answer is a liability.
- **Freshness and versioning are unmanaged.** Which version is authoritative? What changed between v3 and v4? Existing search tools don't answer this.

RIF proved the pattern for code: deterministic extraction → content-addressed graph → hybrid retrieval → citation-gated agents. DIF applies the same pattern to documents.

## 3. Goals and non-goals

### Goals (v1, delivered across P0-P3)

**Release vocabulary:** v1 is the full sellable pilot arc through P3. P0 is only the first executable skeleton. P0 must stay deliberately narrow: own repo + CI, auth-on-day-one service scaffolds, graph schema, `.md/.txt/.docx` ingestion from local/Git sources, and one cited-search flow in CI.

1. Ingest .docx, .pdf, .pptx, .md/.txt, and .json artifacts (.xlsx at v1.5) from configurable sources (file drop, Git repos, SharePoint/OneDrive connector), staged by phase (D-008).
2. Extract a **document graph**: document → section → block (paragraph/table/figure) nodes; REFERENCES, SUPERSEDES, VERSION_OF, CONTAINS edges; content-addressed IDs for change detection.
3. Hybrid retrieval (vector + full-text + graph signal, RRF-fused) over the graph with source-anchored results (doc, section, page/slide, line where applicable).
4. MCP server exposing search, reference-tracing, impact-analysis, and explain tools — consumable from Claude, Copilot, or any MCP client.
5. Agent service producing **citation-gated narratives**: every claim block carries at least one validated `source_ref`; unsupported claims fail closed or become caveats.
6. Incremental re-indexing: only changed documents/sections re-extract and re-embed.
7. **RIF federation (D-007):** per-project deployment into the project's RIF Postgres; `DESCRIBES` edges linking doc blocks to `rif_meta` code nodes; cross-graph tools (`docs_for_code`, `code_for_doc`, `drift_report`). P0 lays the foundation (co-located schema, compatible node IDs) at zero extra cost; P1–P2 deliver the capability.

### Non-goals (v1)

- Document *authoring* or editing — DIF reads, it does not write documents.
- Scanned/image-only PDF corpora as a primary target (v2). v1's parsing router (R2a) handles born-digital PDFs including complex layout via Docling/VLM; corpora that are *predominantly* scans are out of pilot scope and screened out at qualification.
- Real-time collaborative-editor integration (Google Docs live sync, O365 co-authoring events).
- Semantic contradiction detection between documents (v2 candidate; v1 surfaces the references, humans judge).
- Fine-tuned/custom embedding models — v1 uses the same embedding service and model options as RIF.
- Email, chat, or ticket ingestion — documents only.
- Per-user source ACL propagation for mixed-permission corpora — v1 pilots use uniformly-readable corpora or separately indexed corpora per access boundary; ACL inheritance is v2.

## 4. Users and use cases

**Primary personas:**

- **Engineering lead / architect** — traces ADRs, design docs, runbooks; asks "what documents are impacted if we change X?"
- **Compliance / contracts analyst** — finds every clause referencing a policy, with page-level citations.
- **AI platform team** — wires DIF's MCP tools into internal agents so those agents answer from governed, cited sources instead of hallucinating.

**Representative use cases:**

| # | Use case | Tool path |
|---|----------|-----------|
| U1 | "Find every document that references the Q3 data-retention policy" | `search_docs` → `trace_references` |
| U2 | "We're changing the SLA clause — what's impacted?" | `impact_of_change` |
| U3 | "Summarize our position on encryption-at-rest across all policies, with sources" | agent `/explain` (citation-gated) |
| U4 | "What changed between contract v3 and v4?" | `diff_versions` |
| U5 | Internal agent needs grounded doc context via MCP | any tool, from any MCP client |
| U6 | "Which documents describe `PaymentProcessor`, and are they current?" | `docs_for_code` (cross-graph, P1) |
| U7 | "`AmountValidator` changed last sprint — which runbooks/ADRs are now stale?" | `drift_report` (P2) |

## 5. Product requirements

### 5.1 Ingestion

- **R1.** Sources are phased. **P0:** local/mounted file trees and Git repositories (docs-in-repo). **P3:** SharePoint/OneDrive via Graph API connector. Webhook-triggered and scheduled ingestion are connector features, not P0 skeleton blockers.
- **R2.** Formats are phased (D-008). **P0:** `.md`, `.txt`, `.docx`, and `.json`. **P1:** `.pdf` and `.pptx`. **v1.5:** `.xlsx` (visible sheets, ranges, formulas, named ranges — with explicit caveats). Format detection is by content, not extension alone. Every new format enters via the **format admission policy**: parser, source-anchor type, graph node mapping, extraction caveats, golden tests, and cost profile — no ad hoc format additions.
- **R2b.** **JSON extraction (P0, D-008).** Deterministic traversal with JSONPath source anchors; graph expansion capped (max depth/nodes per document, ADR before P0 JSON ingestion) so a pathological config can't flood `dif_meta`. JSON artifacts participate in `DESCRIBES` detection (D-007) — service names, class references, and file paths in configs resolve against `rif_meta`, making policies-as-code and configuration first-class drift sources.
- **R2a.** PDF handling is a **parsing router, not a hand-built extractor**. Parsing is a commoditized layer in 2026 (Docling, Unstructured, Reducto, VLM-based OCR) — DIF's IP is the graph on top, not the parser. Route: text-layer fast path for clean born-digital PDFs; IBM Docling (self-hosted, TableFormer for tables) for complex layout, tables, and multi-column; VLM parse as fallback for pathological pages. Parsed output is stored as structured blocks with **page + bounding-box provenance** — bbox provenance is what makes clause-level citations possible. The router's outputs feed the same deterministic NDJSON contract as native extractors; determinism gates (R9) apply to the router's post-processing, with parser version pinned and recorded per run.
- **R3.** Every ingestion run is recorded with provenance (source, timestamp, content hashes). A run that extracts zero or degenerate content **must not** replace the prior index version (RIF's B1/B2 gate pattern — carry it over verbatim).
- **R4.** Atomic version swap with optimistic locking; a failed run leaves the previous index fully serving.
- **R5.** Incremental mode is P2: content-addressed section IDs mean unchanged sections are not re-extracted or re-embedded; fallback to full re-index with explicit logged reasons.

### 5.2 Extraction — the document graph

- **R6.** Node types: `document`, `section` (heading hierarchy), `block` (paragraph, table, figure, slide), `entity` (v1.5: named policies, systems, parties). Deterministic, content-addressed node IDs using the same NUL-separator SHA-256 scheme as RIF's NodeIdComputer. **One shared implementation is now a hard requirement, not hygiene (D-007):** `DESCRIBES` edges join `dif_meta` blocks to `rif_meta` code nodes by ID convention — a divergent ID scheme silently breaks federation, the product's core capability.
- **R7.** Edge types are phased. **P0:** `CONTAINS` structure. **P1:** `REFERENCES` (explicit citations, hyperlinks, "see section X"), `VERSION_OF`, `SUPERSEDES`. Every edge carries a confidence tier (`exact` / `inferred`) and completeness caveats, surfaced to consumers.
- **R7a.** **`DESCRIBES` edges — doc block → code node (P1, D-007).** Doc extractors run a code-entity detector over blocks: qualified names, file paths, method/class references (backtick spans, code fences, and inline identifiers). Candidates resolve against `rif_meta.method_nodes`/`file_nodes` at load time. Confidence tiers: `exact` (qualified-name match), `inferred` (path or fuzzy simple-name match). Unresolved candidates are flagged per R8, never silently minted. Resolution rate per corpus is recorded and surfaced — it is both a quality gate and the drift-detection denominator. When no `rif_meta` schema exists (standalone/doc-only deployment), the detector still runs and stores candidates as `unresolved`, so a later RIF deployment upgrades them in place on the next re-index.
- **R8.** Unresolvable references are flagged `unresolved:true` — never silently minted as dangling nodes (RIF review finding M20).
- **R9.** Extractors are deterministic: sorted traversal, stable output ordering, byte-reproducible NDJSON. Edge IDs include position discriminators to prevent duplicate-edge collisions (RIF finding M19).
- **R10.** Tables are extracted as structured blocks (rows/columns preserved), not flattened text.

### 5.3 Storage and retrieval

- **R11.** Postgres + pgvector for embeddings and FTS; graph edges as relational adjacency with recursive CTEs (D-003). **DIF lands in the project's existing RIF Postgres database as a sibling schema: `dif_meta` beside `rif_meta` (D-007).** Cross-graph queries are plain SQL joins across the two schemas — no federation protocol, no second database, shared BYOC ops footprint. `dif_meta` migrations are idempotent (RIF-validated approach) and never touch `rif_meta` objects; a documented minimum `rif_meta` schema version is a deploy-time compatibility check. Standalone mode (no `rif_meta` present) is supported: DIF creates only `dif_meta` and cross-graph tools return an explicit `rif_not_deployed` status, not empty results.
- **R12.** Hybrid retrieval: vector similarity + Postgres FTS + graph proximity signal, fused via RRF (reuse RIF's `rrf.go` and fusion logic). Vector queries must push `ORDER BY embedding <=> $n LIMIT k` into index-eligible per-table branches (RIF finding M5 — do not repeat).
- **R12a.** **Reranking stage (P1).** Broad hybrid recall → cross-encoder rerank before results ship. Reranking is the single highest-leverage retrieval component in 2026 benchmarks (+17pp MRR@3 over unreranked hybrid on text-and-table documents). Provider-abstracted (Cohere Rerank v4 API or open-weight cross-encoder self-hosted); rerank scores recorded alongside RRF scores for evaluation. pgvector notes: `halfvec` + HNSW as the default index recipe; pgvectorscale when a corpus passes ~50M vectors.
- **R13.** Every retrieval result carries a source anchor: `doc_id@version : section_path : page/slide/line` — plus `JSONPath` for JSON artifacts and `sheet!cell/range` for Excel at v1.5 (D-008). No anchor type defined = format not admitted (R2).
- **R13a.** Impact-analysis semantics are explicit. `REFERENCES` edges point from the citing node to the cited node. `impact_of_change(anchor)` traverses inbound `REFERENCES` paths from the changed anchor to affected citing documents, returns path evidence, depth, edge confidence, and unresolved-edge caveats, and does not claim semantic contradiction. Defaults: `max_depth=2`, hard cap `5`, `version_scope=current`, `edge_confidence=exact`; callers may opt into `inferred`, `all_versions`, or `as_of`.
- **R13b.** `trace_references(anchor)` supports `direction=outbound|inbound|both`, `max_depth`, `version_scope`, `edge_confidence`, and `include_unresolved`. Results are sorted deterministically by path length, confidence, document ID, then anchor.
- **R13c.** **Cross-graph traversal (P1–P2, D-007).** Impact analysis becomes bidirectional across schemas: `docs_for_code(entity)` resolves the entity in `rif_meta`, then follows inbound `DESCRIBES` edges to doc blocks; `code_for_doc(anchor)` follows outbound `DESCRIBES` edges from a doc anchor into code nodes. `drift_report(repo_id)` (P2) joins `DESCRIBES` edges against code-node content hashes: a doc block whose described code node changed after the doc's version timestamp is flagged as *potentially stale* — with the explicit caveat that staleness is heuristic (code changed ≠ doc wrong); DIF surfaces candidates, humans judge. All cross-graph results carry both doc anchors and code `source_ref`s.

### 5.4 MCP server

- **R14.** Tool surface is phased. **P0:** `search_docs`. **P1:** `trace_references`, `impact_of_change`, **`docs_for_code`, `code_for_doc` (D-007)**. **P2:** `diff_versions`, `explain_topic`, **`drift_report` (D-007)**. Tool schemas **generated from code**, not hand-maintained (RIF finding M3). Cross-graph tools return `rif_not_deployed` explicitly on standalone deployments (R11).
- **R14a.** **Low-level retrieval tools (P2):** `keyword_search`, `semantic_search`, `read_block` — alongside the fixed-function tools. The 2026 pattern is agentic retrieval: consuming agents iterate over retrieval primitives rather than accepting one-shot answers (the pattern Azure AI Search shipped GA in April 2026). Fixed-function tools serve simple clients; primitives serve capable agents. Both share the same auth, audit, and citation contracts.
- **R15.** All required fields validated non-empty server-side; ILIKE wildcards escaped.
- **R16.** Auth from day one, targeting the **MCP 2026-07-28 spec** (stateless core — no `Mcp-Session-Id`, any request routable to any instance): bearer-token (constant-time compare) is acceptable for P0 internal deployments only; remote/pilot deployments require **OAuth 2.1 + PKCE per the MCP auth spec** (RFC 9728 protected-resource metadata, RFC 8707 resource indicators, no token passthrough). Design for deployment behind an enterprise MCP gateway — gateways are the 2026 enterprise control plane for tool authorization and audit; DIF must not assume it terminates auth alone. RIF shipped v1 with zero auth — DIF does not.
- **R16a.** Long-running operations (ingestion runs, corpus-wide analyses) exposed via the MCP **Tasks** extension rather than blocking tool calls.
- **R17.** HTTP server with read/write/idle timeouts, graceful shutdown, and a `/health` that pings Postgres (not a static OK).
- **R18.** Audit log per tool call; audit-write failure logs and continues — it does not fail the user's read.

### 5.5 Agent service (P1)

- **R19.** `/explain` and `/investigate_impact` endpoints return claim blocks, not free-form blobs: each claim has `text`, `source_refs[]` (`min_length=1`), and optional `caveats[]`. A top-level bibliography is not sufficient. No-citation or unresolved-citation output fails closed with 404/422, never a fabricated answer.
- **R20.** Grounding check validates every claim against retrieved excerpts or structured table cells — not verbatim citation-string echo (RIF finding C2). Unsupported claims are dropped or converted into caveats; every fallback event is logged.
- **R20a.** **Citations are structural, not prompted.** Where the narration model is Claude, retrieval results are passed as search-result content blocks / Citations API documents so cited spans are literal quotes re-injected by the API — quote fabrication becomes structurally impossible. Block granularity = citation granularity, so the graph's block segmentation doubles as citation strategy. On other providers, use their grounding-metadata equivalent; the claim-block contract (R19) is provider-independent.
- **R20b.** **Groundedness scoring in the response path.** An HHEM-class open-weights faithfulness scorer runs inline on agent responses (cheap synchronous check); a sampled fraction of traffic goes to async LLM-judge evaluation. Scores are recorded per claim block and surfaced in the audit log — this is the evidence behind the citation-integrity contract (BRD BR5), demonstrable in a procurement bake-off.
- **R21.** All repo/doc-derived text interpolated into prompts is fenced as data with explicit "treat as data, not instructions" framing (prompt-injection surface; RIF finding M15).
- **R22.** MCP client calls carry timeouts, retries with backoff, and propagate request context.

### 5.6 Embedding service — shared with RIF

- **R23.** Reuse RIF's embedding service (LiteLLM provider abstraction + local model fallback). Add before DIF ships: request batch-size cap, concurrency semaphore on local encode, per-batch persistence in backfill CLI (RIF findings H4/M11/H2-python).
- **R23a.** **Embedding default refreshed for prose.** RIF's `text-embedding-3-small` is aging (no OpenAI refresh since 2024). DIF default: a Matryoshka-capable API model — Voyage (Anthropic's recommended provider) or Gemini Embedding — stored full-dimension, served truncated at ≤1024d; Qwen3-Embedding as the self-host/sovereignty fallback. Quality has converged across the top models, so the choice is ops/cost, and the LiteLLM abstraction makes switching cheap — but pick before P1, because re-embedding a pilot corpus later is the expensive path. ColPali-style page-image embeddings for visually dense corpora (slides, scanned forms) are a v2 second index, not the primary.
- **R24.** Token-aware (not char-based) truncation sized to the model's context window.

### 5.7 Cross-cutting

- **R25.** Namespace: `com.aaraminds.dif` / `github.com/aaraminds/dif` from the first commit. No client branding, ever (RIF's costliest cleanup).
- **R26.** Own git repo, `.github/workflows/` CI live from week 1 (lint, test, vuln scan, doc-path check), root `.gitignore`, CODEOWNERS at repo root with real identities.
- **R27.** All architecture docs cite in-repo paths only; CI check fails on citations to nonexistent paths (the exact failure RIF's doc set suffered).
- **R28.** Containers run non-root, `.dockerignore` present, builds resolve from lockfiles, `HEALTHCHECK` defined.
- **R29.** Structured logging with request IDs in every service. No silent excepts.
- **R30.** Usage metering is a product requirement before paid pilot. Emit non-PII usage events for `ingestion_run`, `document_indexed`, `embedding_batch`, `mcp_tool_call`, `agent_request`, and `connector_sync` with tenant/corpus IDs, connector ID, counts, latency, token/embedding units where applicable, and error class. Metering is separate from audit logs.

## 6. Architecture (adopted from RIF)

Deployment unit is **per project**: DIF services deploy beside the project's existing RIF stack and write to the same Postgres (`dif_meta` beside `rif_meta`). The Postgres box below is the project's RIF database, not a new one.

```
 Sources (files / git / SharePoint)
        │  webhook / schedule
        ▼
 ┌─────────────┐   NDJSON    ┌──────────────────┐
 │  Ingestion   │──nodes/────▶│ Postgres          │
 │  (Go)        │  edges      │  + pgvector + FTS │
 └──────┬──────┘             └────────┬─────────┘
        │ embed requests               │
        ▼                              ▼
 ┌─────────────┐             ┌──────────────────┐
 │ Embedding    │             │ Retriever (Go)    │
 │ svc (Py,     │             │ vector+FTS+graph  │
 │ shared w/RIF)│             │ RRF fusion        │
 └─────────────┘             └────────┬─────────┘
                                       │
                             ┌────────▼─────────┐      ┌───────────────┐
                             │ MCP server (Go)   │◀────▶│ Agent svc (Py) │
                             │ 5 tools, authed   │      │ citation-gated │
                             └──────────────────┘      └───────────────┘
```

Extractors: Go for format parsing where mature libraries exist; JVM (Apache POI) permitted for OOXML depth if Go libraries fall short; PDF goes through the parsing router (R2a — Docling/VLM integration, not hand-built) — decide via ADR per format. Stack constraints per AaraMinds standards: Azure-primary, Terraform AzureRM, GitHub Actions OIDC, Key Vault via managed identity, Grafana + Prometheus + OTel.

**Long-context routing (design note).** Retrieval and long context are complementary, not competing: single-document deep analysis (e.g., "analyze this one contract end to end") routes the whole document into a large context window with prompt caching; corpus-scale and high-QPS queries go through the retriever. The router is a product decision with easy economics — linear token cost and quadratic attention latency make retrieval the only viable path at corpus scale, while long context wins for one-off whole-document work. The agent service implements this routing; it is not exposed as user configuration.

## 7. Success metrics

Baselines to be established during pilot; no targets ship without a measured baseline.

- **Retrieval quality:** precision@10 on a curated golden-query set per corpus [VERIFY — build golden set in pilot].
- **Citation integrity:** % of individual claim blocks where every `source_ref` resolves to a valid source anchor and the cited excerpt supports the claim (target: 100%; structurally enforced).
- **Freshness:** p95 time from document change → queryable (target set after pilot measurement).
- **Adoption:** MCP tool calls/week from non-DIF-team consumers.
- **Extraction determinism:** repeated extraction of unchanged corpus yields byte-identical output (CI-enforced, boolean).
- **Metering completeness:** 100% of ingestion runs, MCP calls, and agent requests emit usage events before paid pilot.
- **`DESCRIBES` resolution rate (D-007):** % of detected code-entity candidates that resolve `exact` against `rif_meta`, per corpus — quality gate for federation and the drift-report denominator (baseline set in the reference-density spike).

## 8. Phasing

| Phase | Scope | Exit criteria |
|-------|-------|---------------|
| P0 — Skeleton | Own repo, CI, auth-on-day-one service scaffolds, **`dif_meta` schema in the RIF Postgres with RIF-compatible node IDs (D-007)**, local/Git ingestion, `.md/.txt/.docx`, `CONTAINS` graph, `search_docs`, audit/health/usage-event schema | E2E: docx in → cited search result out, in CI; golden demo corpus checked in; `dif_meta` migration idempotent against a live `rif_meta` database |
| P1 — Core graph + federation | PDF/pptx via parsing router, `REFERENCES`/`VERSION_OF`/`SUPERSEDES` edges, **`DESCRIBES` code-entity detector + resolution against `rif_meta`**, hybrid retriever, `trace_references`, `impact_of_change`, **`docs_for_code`, `code_for_doc`** | Golden-query set passing; impact-analysis semantics tested; determinism check green; `DESCRIBES` resolution rate measured on a real RIF project |
| P2 — Agents + incremental + drift | Agent service, claim-level citation gate, incremental re-index, `diff_versions`, `explain_topic`, **`drift_report`** | Claim citation gate 100%; incremental correctness proven; usage metering complete; drift report validated against a known code change |
| P3 — Connectors + hardening | SharePoint/OneDrive connector for uniformly-readable corpora, observability, Terraform, deployment hardening | Paid pilot deployment with a real admissible corpus |
| v2 candidates | Scanned-corpus OCR, ColPali-style page-image retrieval index, contradiction detection, entity extraction, ACL propagation | — |

Federation (D-007) is deliberately foundation-first: P0 pays the near-zero cost of landing in the right database with the right ID scheme; P1–P2 turn that into `DESCRIBES` edges, cross-graph tools, and drift detection — a feature rollout, not a re-architecture.

## 9. Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| PDF extraction quality varies wildly by producer | Garbage graph in, garbage answers out | Parsing router (R2a): text-layer fast path + Docling + VLM fallback; per-format quality gates; degenerate-run guard blocks index swap |
| Parser dependency drift (Docling/VLM versions change output) | Determinism story breaks | Parser versions pinned and recorded per run; determinism CI compares against pinned-parser baseline; re-parse is an explicit versioned event |
| OOXML edge cases (tracked changes, embedded objects) | Silent content loss | Explicit unsupported-feature caveats on nodes; completeness surfaced to consumers |
| SharePoint connector auth/throttling complexity | P3 slip | Isolate as connector module; file-drop and git paths carry v1 |
| Naive-RAG competitors ship "good enough" | Differentiation pressure | Lead with citation integrity + impact analysis — what chunk-RAG structurally cannot do |
| Repeating RIF's hygiene debt | Same cleanup cost twice | R25–R29 are P0 requirements, not follow-ups; CI-enforced |
| `rif_meta` schema evolves under DIF (D-007) | Cross-graph joins silently break | Documented minimum `rif_meta` version checked at deploy time (R11); cross-graph queries behind a thin view layer so RIF schema changes are absorbed in one place; contract test in CI against a pinned RIF fixture database |
| Low `DESCRIBES` resolution on real corpora | Federation value doesn't materialize | Reference-density spike measures doc→code resolution before P0 commits; `inferred` tier + resolution-rate metric make quality visible per corpus |

## 10. Open questions

1. ~~Graph store~~ — **DECIDED (D-003, 2026-07-08):** relational adjacency + recursive CTEs. See `DECISIONS.md`.
2. ~~Embedding model for prose~~ — **DECIDED (D-002, 2026-07-08):** Voyage via shared LiteLLM service, ≤1024d Matryoshka; exact model/dimension pinned at end of P0 spike (D-005).
3. ~~Multi-tenancy model~~ — **RESOLVED by D-001 (BYOC, 2026-07-08):** isolation by customer tenancy; no in-app row-level tenancy needed for pilot.
4. Access control inheritance v2 design: once v1 proves uniformly-readable corpora, decide whether ACL propagation is row-level filtering, per-corpus partitioning, or tenant-specific indexes (D-006, needs a committed roadmap date per BRD BR4).
5. `rif_meta` compatibility contract (D-007): which RIF schema version is the documented minimum, and does RIF need a version stamp table for DIF's deploy-time check? Decide during P0 against the current RIF schema.
6. JSON graph expansion limits (D-008): max traversal depth and nodes-per-document caps — ADR before P0 JSON ingestion (design-decisions ADR-006).

---
*Document conventions: unverified figures are marked [VERIFY]. This PRD cites no external market data — see BRD.*
