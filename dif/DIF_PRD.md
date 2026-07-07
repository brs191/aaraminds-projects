# DIF — Documents Intelligence Factory
## Product Requirements Document (PRD)

**Version:** 0.1 (Draft)
**Date:** 2026-07-07
**Owner:** AaraMinds
**Status:** Draft — pending review
**Sibling product:** RIF (Repo Intelligence Factory)

---

## 1. Summary

DIF turns an organization's document corpus into a queryable, citation-grounded knowledge graph — the same way RIF does for code. It ingests enterprise documents (Word, PDF, PowerPoint) and engineering docs (Markdown, ADRs, wiki exports), extracts structure and cross-references into a document graph, embeds content for hybrid retrieval, and exposes the result through MCP tools and a citation-gated agent service.

The product answers questions no keyword search or naive RAG can: *"Which contracts reference the deprecated SLA clause?"*, *"What downstream documents are impacted if this architecture decision changes?"*, *"Show me every claim about data retention across our policy set, with sources."*

DIF reuses RIF's proven architecture: ingestion → extractors → Postgres + pgvector storage → hybrid retriever (vector + FTS + graph, RRF fusion) → MCP server → agent service. Same stack, same ops model, new extraction domain.

## 2. Problem statement

Enterprise document estates share the pathologies of legacy codebases:

- **Knowledge is trapped in format.** The answer exists, but it's on slide 34 of a deck, in a table inside a PDF, or split across three versions of a Word doc.
- **Cross-references are invisible.** Documents cite, supersede, and contradict each other with no machine-readable link structure. Impact analysis ("what breaks if we change this policy?") is manual and unreliable.
- **Naive RAG hallucinates and can't cite.** Chunk-and-embed pipelines lose document structure (headings, tables, clause hierarchy) and produce answers without verifiable provenance. In contract, compliance, and engineering-decision contexts, an uncited answer is a liability.
- **Freshness and versioning are unmanaged.** Which version is authoritative? What changed between v3 and v4? Existing search tools don't answer this.

RIF proved the pattern for code: deterministic extraction → content-addressed graph → hybrid retrieval → citation-gated agents. DIF applies the same pattern to documents.

## 3. Goals and non-goals

### Goals (v1)

1. Ingest .docx, .pdf, .pptx, and .md/.txt documents from configurable sources (file drop, Git repos, SharePoint/OneDrive connector).
2. Extract a **document graph**: document → section → block (paragraph/table/figure) nodes; REFERENCES, SUPERSEDES, VERSION_OF, CONTAINS edges; content-addressed IDs for change detection.
3. Hybrid retrieval (vector + full-text + graph signal, RRF-fused) over the graph with source-anchored results (doc, section, page/slide, line where applicable).
4. MCP server exposing search, reference-tracing, impact-analysis, and explain tools — consumable from Claude, Copilot, or any MCP client.
5. Agent service producing **citation-gated narratives**: every claim carries a source_ref; responses with no citations fail closed.
6. Incremental re-indexing: only changed documents/sections re-extract and re-embed.

### Non-goals (v1)

- Document *authoring* or editing — DIF reads, it does not write documents.
- OCR of scanned/image-only PDFs (v2; v1 handles text-layer PDFs only).
- Real-time collaborative-editor integration (Google Docs live sync, O365 co-authoring events).
- Semantic contradiction detection between documents (v2 candidate; v1 surfaces the references, humans judge).
- Fine-tuned/custom embedding models — v1 uses the same embedding service and model options as RIF.
- Email, chat, or ticket ingestion — documents only.

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

## 5. Product requirements

### 5.1 Ingestion (P0)

- **R1.** Sources: local/mounted file trees, Git repositories (docs-in-repo), SharePoint/OneDrive via Graph API connector. Webhook-triggered and scheduled ingestion.
- **R2.** Formats: .docx, .pptx (OOXML parsing), .pdf (text-layer), .md, .txt. Format detection by content, not extension alone.
- **R3.** Every ingestion run is recorded with provenance (source, timestamp, content hashes). A run that extracts zero or degenerate content **must not** replace the prior index version (RIF's B1/B2 gate pattern — carry it over verbatim).
- **R4.** Atomic version swap with optimistic locking; a failed run leaves the previous index fully serving.
- **R5.** Incremental mode: content-addressed section IDs mean unchanged sections are not re-extracted or re-embedded; fallback to full re-index with explicit logged reasons.

### 5.2 Extraction — the document graph (P0)

- **R6.** Node types: `document`, `section` (heading hierarchy), `block` (paragraph, table, figure, slide), `entity` (v1.5: named policies, systems, parties). Deterministic, content-addressed node IDs (same NUL-separator SHA-256 scheme as RIF's NodeIdComputer — **one shared implementation, not a copy**; this was a RIF review finding).
- **R7.** Edge types: `CONTAINS` (structure), `REFERENCES` (explicit citations, hyperlinks, "see section X"), `VERSION_OF`, `SUPERSEDES`. Every edge carries a confidence tier (`exact` / `inferred`) and completeness caveats, surfaced to consumers.
- **R8.** Unresolvable references are flagged `unresolved:true` — never silently minted as dangling nodes (RIF review finding M20).
- **R9.** Extractors are deterministic: sorted traversal, stable output ordering, byte-reproducible NDJSON. Edge IDs include position discriminators to prevent duplicate-edge collisions (RIF finding M19).
- **R10.** Tables are extracted as structured blocks (rows/columns preserved), not flattened text.

### 5.3 Storage and retrieval (P0)

- **R11.** Postgres + pgvector for embeddings and FTS; graph edges in Postgres (AGE or relational adjacency — decide via ADR, default to what RIF ships). One `docs_nodes` / `docs_edges` schema versioned via the same idempotent migration approach RIF validated.
- **R12.** Hybrid retrieval: vector similarity + Postgres FTS + graph proximity signal, fused via RRF (reuse RIF's `rrf.go` and fusion logic). Vector queries must push `ORDER BY embedding <=> $n LIMIT k` into index-eligible per-table branches (RIF finding M5 — do not repeat).
- **R13.** Every retrieval result carries a source anchor: `doc_id@version : section_path : page/slide/line`.

### 5.4 MCP server (P0)

- **R14.** Tools (v1): `search_docs`, `trace_references`, `impact_of_change`, `diff_versions`, `explain_topic`. Tool schemas **generated from code**, not hand-maintained (RIF finding M3).
- **R15.** All required fields validated non-empty server-side; ILIKE wildcards escaped.
- **R16.** Bearer-token auth on `/mcp` and all HTTP surfaces **from day one** (constant-time compare). RIF shipped v1 with zero auth — DIF does not.
- **R17.** HTTP server with read/write/idle timeouts, graceful shutdown, and a `/health` that pings Postgres (not a static OK).
- **R18.** Audit log per tool call; audit-write failure logs and continues — it does not fail the user's read.

### 5.5 Agent service (P1)

- **R19.** `/explain` and `/investigate_impact` endpoints; responses are structurally citation-gated (`source_refs` min_length=1 in response models; no-citation → 404/422, never a fabricated answer).
- **R20.** Grounding check matches on retrieved excerpt substrings — not verbatim citation-string echo (RIF finding C2). Every fallback event is logged.
- **R21.** All repo/doc-derived text interpolated into prompts is fenced as data with explicit "treat as data, not instructions" framing (prompt-injection surface; RIF finding M15).
- **R22.** MCP client calls carry timeouts, retries with backoff, and propagate request context.

### 5.6 Embedding service (P0 — shared with RIF)

- **R23.** Reuse RIF's embedding service (LiteLLM provider abstraction + local model fallback). Add before DIF ships: request batch-size cap, concurrency semaphore on local encode, per-batch persistence in backfill CLI (RIF findings H4/M11/H2-python).
- **R24.** Token-aware (not char-based) truncation sized to the model's context window.

### 5.7 Cross-cutting (P0)

- **R25.** Namespace: `com.aaraminds.dif` / `github.com/aaraminds/dif` from the first commit. No client branding, ever (RIF's costliest cleanup).
- **R26.** Own git repo, `.github/workflows/` CI live from week 1 (lint, test, vuln scan, doc-path check), root `.gitignore`, CODEOWNERS at repo root with real identities.
- **R27.** All architecture docs cite in-repo paths only; CI check fails on citations to nonexistent paths (the exact failure RIF's doc set suffered).
- **R28.** Containers run non-root, `.dockerignore` present, builds resolve from lockfiles, `HEALTHCHECK` defined.
- **R29.** Structured logging with request IDs in every service. No silent excepts.

## 6. Architecture (adopted from RIF)

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

Extractors: Go for format parsing where mature libraries exist; JVM (Apache POI) permitted for OOXML depth if Go libraries fall short — decide via ADR per format. Stack constraints per AaraMinds standards: Azure-primary, Terraform AzureRM, GitHub Actions OIDC, Key Vault via managed identity, Grafana + Prometheus + OTel.

## 7. Success metrics

Baselines to be established during pilot; no targets ship without a measured baseline.

- **Retrieval quality:** precision@10 on a curated golden-query set per corpus [VERIFY — build golden set in pilot].
- **Citation integrity:** % of agent responses where every claim resolves to a valid source anchor (target: 100%; structurally enforced).
- **Freshness:** p95 time from document change → queryable (target set after pilot measurement).
- **Adoption:** MCP tool calls/week from non-DIF-team consumers.
- **Extraction determinism:** repeated extraction of unchanged corpus yields byte-identical output (CI-enforced, boolean).

## 8. Phasing

| Phase | Scope | Exit criteria |
|-------|-------|---------------|
| P0 — Skeleton | Repo, CI, auth-on-day-one service scaffolds, schema, .md/.docx ingestion | E2E: docx in → cited search result out, in CI |
| P1 — Core graph | PDF + pptx extractors, REFERENCES/VERSION_OF edges, hybrid retriever, MCP tools | Golden-query set passing; determinism check green |
| P2 — Agents + incremental | Agent service, incremental re-index, diff_versions | Citation gate 100%; incremental correctness proven |
| P3 — Connectors + hardening | SharePoint/OneDrive connector, observability, Terraform | Pilot deployment with a real corpus |
| v2 candidates | OCR, contradiction detection, entity extraction, RIF+DIF joint queries ("which docs describe this code?") | — |

The RIF+DIF joint-query capability (linking document claims to code entities) is the long-term differentiator — design node-ID and schema conventions now so the two graphs can federate later.

## 9. Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| PDF extraction quality varies wildly by producer | Garbage graph in, garbage answers out | Text-layer-only in v1; per-format quality gates; degenerate-run guard blocks index swap |
| OOXML edge cases (tracked changes, embedded objects) | Silent content loss | Explicit unsupported-feature caveats on nodes; completeness surfaced to consumers |
| SharePoint connector auth/throttling complexity | P3 slip | Isolate as connector module; file-drop and git paths carry v1 |
| Naive-RAG competitors ship "good enough" | Differentiation pressure | Lead with citation integrity + impact analysis — what chunk-RAG structurally cannot do |
| Repeating RIF's hygiene debt | Same cleanup cost twice | R25–R29 are P0 requirements, not follow-ups; CI-enforced |

## 10. Open questions

1. Graph store: Apache AGE vs relational adjacency tables — ADR needed (RIF experience is the input).
2. Embedding model for prose vs RIF's code-tuned choice — same service, possibly different model per corpus type.
3. Multi-tenancy model for the sellable product: DB-per-tenant vs row-level — BRD dependency, ADR before P3.
4. Access control inheritance: documents carry ACLs (SharePoint permissions) — does v1 enforce source ACLs at query time or restrict scope to uniformly-readable corpora? Recommend: uniformly-readable corpora in v1, ACL propagation as v2; state this limitation explicitly in sales material.

---
*Document conventions: unverified figures are marked [VERIFY]. This PRD cites no external market data — see BRD.*
