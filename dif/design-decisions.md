# DIF - Design Decisions for Enterprise Implementation

**Version:** 1.1 Draft  
**Date:** 2026-07-08  
**Owner:** AaraMinds  
**Related:** `dif_brd.md`, `dif_prd.md`, `DECISIONS.md`, RIF  
**Purpose:** Decision **backlog** — the design decisions the engineering team must resolve before building DIF at enterprise/B2B scale. This file lists questions, options, and recommended defaults. Decisions actually made are recorded in `DECISIONS.md` (the decision log) and marked **RESOLVED** here. Where this backlog and a dated D-entry conflict, the D-entry wins.

**v1.1:** reconciled with DECISIONS.md — DD-02/03/11/14 marked resolved (D-001/D-002/D-003), DD-01 resolved (D-008), DD-28 rewritten per D-007 (federation is core v1), DD-15 superseded by PRD R12a, MCP spec references updated to the 2026-07-28 release.

---

## 1. Decision Context

DIF is not a document chatbot. It is an enterprise-grade intelligence backend that converts documents and structured knowledge artifacts into a citation-grounded, queryable knowledge graph for agents, APIs, and human workflows.

The expected quality bar is AT&T/B2B scale:

- Multi-tenant enterprise deployment.
- Large document estates across teams, business units, and regulated domains.
- Strict source attribution, provenance, auditability, and access control.
- Production-grade observability, cost controls, and operational runbooks.
- Secure MCP/tool exposure for internal and external AI agents.
- CI-enforced quality gates before pilots, not after customer pressure.

Current market direction supports this architecture: enterprise AI is moving from generic assistants toward task-specific agents, AI-ready governed data, retrieval with provenance, MCP-style tool integration, and GenAI observability. Gartner predicts broad adoption of task-specific agents in enterprise applications by 2026, while MCP, OWASP LLM guidance, and OpenTelemetry GenAI conventions are becoming important design inputs for production agent platforms.

Reference anchors:

- [Gartner: task-specific AI agents in enterprise applications](https://www.gartner.com/en/newsroom/press-releases/2025-08-26-gartner-predicts-40-percent-of-enterprise-apps-will-feature-task-specific-ai-agents-by-2026-up-from-less-than-5-percent-in-2025)
- [Gartner: AI agents and AI-ready data](https://www.gartner.com/en/newsroom/press-releases/2025-08-05-gartner-hype-cycle-identifies-top-ai-innovations-in-2025)
- [MCP specification — target the 2026-07-28 release (stateless core, final ships July 28, 2026)](https://modelcontextprotocol.io/specification)
- [MCP authorization (OAuth 2.1 + PKCE, RFC 9728/8707) per the 2026-07-28 release](https://modelcontextprotocol.io/specification)
- [MCP security best practices](https://modelcontextprotocol.io/docs/tutorials/security/security_best_practices)
- [OWASP Top 10 for LLM Applications](https://owasp.org/www-project-top-10-for-large-language-model-applications/)
- [OpenTelemetry GenAI observability](https://opentelemetry.io/blog/2026/genai-observability/)
- [NIST AI Risk Management Framework](https://www.nist.gov/itl/ai-risk-management-framework)

---

## 2. Decision Principles

| Principle | Engineering Meaning |
|---|---|
| Deterministic before generative | Extraction, IDs, versioning, ACLs, and citations must be deterministic. LLMs can explain; they must not invent the source of truth. |
| Evidence is the product | Every claim must resolve to a source anchor: page, section, slide, line, JSONPath, or Excel cell/range. |
| Secure by default | Auth, tenant isolation, audit logging, prompt-injection controls, and secret handling must exist from day one. |
| Enterprise scale is a design input | The system must assume large corpora, concurrent ingestion, retries, throttling, partial failure, and cost pressure. |
| Agent-ready, not agent-dependent | DIF should expose reliable tools and APIs usable by agents, but core intelligence must not depend on one agent framework. |
| Evaluation gates releases | Retrieval quality, citation integrity, ACL correctness, extraction determinism, and latency must be release gates. |
| Reuse RIF deliberately | Reuse RIF patterns where proven; require ADRs for divergence. Do not copy/paste shared primitives. |

---

## 3. Decision Backlog

### DD-01 - Product Boundary: Document Intelligence vs General Data Platform

**✅ RESOLVED (D-008, 2026-07-08):** Documents + file-based structured artifacts — `.md/.txt/.docx/.json` (P0), `.pdf/.pptx` (P1), `.xlsx` (v1.5). Format admission policy applies to every new type. See `DECISIONS.md`.

**Decision needed:** Define what DIF owns and what it explicitly does not own.

| Option | Notes |
|---|---|
| Documents only | Simpler, but misses JSON configs, policies-as-code, inventories, and spreadsheet systems of record. |
| Documents + file-based structured artifacts | Recommended. Supports `.docx`, `.pdf`, `.pptx`, `.md`, `.txt`, `.json`, and v1.5 `.xlsx`. |
| General enterprise data platform | Too broad. Would pull DIF into streaming, CDC, relational warehouse, and analytics scope. |

**Recommended default:** Documents + file-based structured artifacts.

**Before implementation:** Add a format admission policy. Every new source type must define parser, source anchor, graph nodes, extraction caveats, golden tests, and cost profile.

---

### DD-02 - Deployment Model

**✅ RESOLVED (D-001, 2026-07-08):** Managed deployment on customer Azure tenancy (BYOC), AaraMinds-operated. Additionally per D-007: the deployment unit is **per project**, co-located with the project's RIF stack. See `DECISIONS.md`.

**Decision needed:** Decide whether DIF is self-hosted, managed in customer cloud, or fully SaaS.

| Option | Fit |
|---|---|
| Self-hosted in customer tenancy | Strong for AT&T-scale security reviews and data residency. |
| Managed deployment in customer Azure tenant | Recommended initial default. AaraMinds can operate while data stays in customer boundary. |
| AaraMinds multi-tenant SaaS | Faster operations later, but harder for regulated B2B pilots. |

**Recommended default:** Managed deployment in customer Azure tenancy, with self-hosted option for high-control customers.

**Before implementation:** Decide tenancy boundary, network model, Key Vault ownership, logging export, upgrade process, and support access model.

---

### DD-03 - Multi-Tenancy Isolation

**✅ RESOLVED (D-001 + D-007, 2026-07-08):** Isolation by customer tenancy (BYOC) and by project database (DIF lands in each project's RIF Postgres) — DB-per-tenant by construction. Revisit only if an AaraMinds-hosted multi-tenant tier is added. See `DECISIONS.md`.

**Decision needed:** Choose tenant isolation model before schema and migration work.

| Option | Trade-off |
|---|---|
| Database per tenant | Strong isolation, simpler customer audits, higher operations overhead. |
| Schema per tenant | Middle ground, but operational mistakes can still cross boundaries. |
| Row-level tenancy | Efficient, but highest blast radius if controls fail. |

**Recommended default:** Database per tenant for enterprise/B2B customers; row-level tenancy only for internal/demo environments.

**Before implementation:** Produce ADR covering tenant lifecycle, migrations, backups, key separation, audit export, and restore drills.

---

### DD-04 - Identity, Authorization, and Source ACLs

**Decision needed:** Decide how user identity and source permissions flow into DIF.

| Area | Recommended Direction |
|---|---|
| User auth | OIDC/SAML via enterprise IdP; Entra ID first for Microsoft-heavy customers. |
| Service auth | Managed identity / workload identity; no static cloud credentials. |
| Source ACLs | v1 pilots may use uniformly-readable corpora; v2 must enforce source ACL propagation. |
| Authorization model | RBAC for admin actions, ABAC/resource ACLs for retrieval. |

**Why it matters:** In AT&T/B2B scale, retrieval without permission filtering is a procurement blocker.

**Before implementation:** Define the v1 pilot rule clearly: either uniformly-readable corpora only, or ACL-enforced retrieval. Do not leave this ambiguous.

---

### DD-05 - MCP Exposure and Tool Governance

**Decision needed:** Decide whether each service exposes MCP directly or all tools go through a governed MCP gateway.

| Option | Trade-off |
|---|---|
| Direct MCP server per service | Simple early build, weaker governance. |
| Central MCP gateway | Recommended for enterprise scale: auth, audit, policy, tool registry, throttling, and versioning in one place. |
| MCP only inside agent service | Easier, but limits reuse by external agent clients. |

**Recommended default:** Central MCP gateway plus internally versioned tool contracts.

**Must include:**

- MCP authorization aligned with the MCP HTTP authorization specification.
- Tool schema generation from code.
- Tool allowlists per tenant/client.
- Tool versioning and deprecation policy.
- Capability attestation or signed tool registry before broad enterprise rollout.
- Audit logs for every tool invocation, including principal, tenant, tool, parameters hash, source corpus, and outcome.

---

### DD-06 - Agent Framework and Responsibility Boundary

**Decision needed:** Decide whether DIF builds agents, exposes tools, or does both.

| Layer | Recommendation |
|---|---|
| Core system | Deterministic services and APIs. |
| MCP layer | First-class product surface. |
| Agent service | Thin, citation-gated narrative layer. |
| Autonomous workflows | Defer until evals, safety policies, and human-approval flows are mature. |

**Framework options:** LangGraph, Semantic Kernel, OpenAI Agents SDK, custom orchestrator, or hybrid.

**Recommended default:** Keep core DIF framework-neutral. Use a thin agent service for `/explain` and `/investigate_impact`. Use LangGraph or equivalent only where stateful multi-step reasoning is required and testable.

**Before implementation:** Define exactly which actions agents may perform. For v1, prefer read-only tools.

---

### DD-07 - Workflow Orchestration: Durable vs In-Process

**Decision needed:** Pick orchestration patterns separately for ingestion and agent reasoning.

| Workload | Recommended Pattern |
|---|---|
| Long-running ingestion, connector sync, retries, backfills | Durable orchestration such as Temporal, Azure Durable Functions, or queue + worker with idempotency. |
| Short agent reasoning plans | LangGraph/Semantic Kernel/custom state machine, bounded by timeouts and evals. |
| Embedding backfill | Batch worker with resumable checkpoints and per-batch persistence. |

**Recommended default:** Durable orchestration for ingestion; lightweight bounded orchestration for agent calls.

**Before implementation:** Define idempotency keys, retry policy, dead-letter queues, checkpoint schema, and replay behavior.

---

### DD-08 - Ingestion Architecture

**Decision needed:** Choose ingestion topology for large corpora and unreliable connectors.

| Area | Recommended Direction |
|---|---|
| Triggering | Webhook where available, scheduled polling otherwise. |
| Processing | Async queue-based pipeline with bounded concurrency. |
| Idempotency | Content hash + source version + tenant + connector ID. |
| Failure handling | Partial failure isolation, dead-letter queues, retry budgets. |
| Index swap | Atomic version swap; failed extraction must not replace serving index. |

**Before implementation:** Create ingestion state machine covering discovered, fetched, parsed, extracted, embedded, indexed, validated, promoted, failed, and quarantined states.

---

### DD-09 - Parser and Extraction Strategy

**Decision needed:** Decide parser stack by format rather than choosing one universal parser.

| Format | Recommended Starting Point |
|---|---|
| Markdown/text | Native parser with line anchors. |
| JSON | Standard parser with deterministic traversal and JSONPath anchors. |
| DOCX/PPTX | OOXML parser; Apache POI fallback if Go libraries are insufficient. |
| Text-layer PDF | PDF text extraction with layout preservation and quality gates. |
| Excel v1.5 | `.xlsx` parser with visible sheets, ranges, formulas, named ranges, and caveats. |
| OCR/image documents v2 | Azure Document Intelligence, AWS Textract, Docling, or specialized OCR pipeline after evaluation. |

**Recommended default:** Use deterministic parsers first; use LLMs for interpretation only after extraction quality is measured.

**Before implementation:** Each extractor must emit completeness caveats, byte-stable output, source anchors, and golden-test fixtures.

---

### DD-10 - OCR and Multimodal Document Understanding

**Decision needed:** Decide when and how to support scanned PDFs, charts, figures, and image-heavy decks.

| Option | Trade-off |
|---|---|
| v1 text-layer only | Safe and deterministic, but excludes common enterprise PDFs. |
| Add OCR immediately | Better coverage, higher cost/latency/noise. |
| Add document-quality profiler in v1, OCR in v2 | Recommended. Gives customers visibility without derailing v1. |

**Recommended default:** v1 text-layer only plus quality profiler; v2 OCR/multimodal extraction.

**Before implementation:** Define quality signals: text coverage, pages with no text, table density, image density, OCR-needed percentage, extraction confidence, and unsupported features.

---

### DD-11 - Knowledge Graph Storage

**✅ RESOLVED (D-003 + D-007, 2026-07-08):** Postgres relational adjacency with recursive CTEs, as `dif_meta` schema in the project's existing RIF Postgres beside `rif_meta`. See `DECISIONS.md`.

**Decision needed:** Choose graph storage model.

| Option | Trade-off |
|---|---|
| Postgres relational adjacency | Recommended default: simple operations, fits pgvector/FTS, fewer moving parts. |
| Apache AGE on Postgres | Useful graph query syntax, but adds operational complexity. |
| Neo4j or external graph DB | Strong graph ergonomics, but another platform to run and secure. |

**Recommended default:** Postgres adjacency tables for v1; reconsider AGE/external graph only if graph-query complexity demands it.

**Before implementation:** Define schema versioning, migration rollback, tenant isolation, source anchor model, and graph query performance tests.

---

### DD-12 - Vector Store and Retrieval Stack

**Decision needed:** Decide whether pgvector is enough or whether a dedicated vector database is needed.

| Option | Fit |
|---|---|
| Postgres + pgvector + FTS | Recommended for v1; simpler and matches PRD. |
| OpenSearch/Elasticsearch + vector | Better search features, higher operational surface. |
| Dedicated vector DB | Consider only if corpus size, latency, or recall cannot be met. |

**Recommended default:** Postgres + pgvector + FTS + graph signal + RRF fusion.

**Before implementation:** Run retrieval benchmarks on realistic corpora, not toy PDFs. Measure precision@10, latency p50/p95/p99, index size, and re-index cost.

---

### DD-13 - Chunking vs Structural Nodes

**Decision needed:** Decide whether retrieval indexes chunks or graph-native structural units.

| Option | Trade-off |
|---|---|
| Fixed-size chunks | Simple but loses headings, tables, clauses, and citations. |
| Structural nodes only | Better provenance, may hurt semantic recall if nodes are too small/large. |
| Structural nodes + derived retrieval passages | Recommended. Preserve graph truth while optimizing retrieval. |

**Recommended default:** Source graph as canonical; retrieval passages are derived artifacts with reversible source anchors.

**Before implementation:** Define passage-generation rules per format and prove every passage maps back to exact source nodes.

---

### DD-14 - Embedding Model Strategy

**✅ PARTIALLY RESOLVED (D-002, 2026-07-08):** Prose default is Voyage via the shared LiteLLM abstraction, ≤1024d Matryoshka; Qwen3-Embedding self-host fallback; exact model/dimension pinned at P0 spike exit (D-005). Still open: per-artifact-type model choices (tables/JSON) and the eval set. See `DECISIONS.md`.

**Decision needed:** Decide model/provider strategy for prose, tables, JSON, and future multimodal content.

| Area | Decision Needed |
|---|---|
| Provider abstraction | LiteLLM-compatible or equivalent abstraction to support hosted and local models. |
| Model selection | Different models may be needed for prose vs code/config vs table-heavy content. |
| Local fallback | Required for sensitive/on-prem deployments or outage resilience. |
| Cost controls | Batch caps, token-aware truncation, concurrency limits, caching, and usage metering. |

**Recommended default:** Provider abstraction from day one; model choice is a corpus-level configuration with eval-backed defaults.

**Before implementation:** Establish embedding eval set and cost benchmark for each supported artifact type.

---

### DD-15 - Reranking and Hybrid Retrieval

**Decision needed:** Decide whether to add reranking in v1.

| Option | Trade-off |
|---|---|
| Vector + FTS only | Fast, but weaker precision on enterprise queries. |
| Vector + FTS + graph RRF | PRD baseline and recommended minimum. |
| Add cross-encoder or LLM reranker | Better precision, more latency/cost. |

**⚠️ SUPERSEDED by PRD R12a (v0.2 market review):** reranking is **P1, not optional-later** — cross-encoder rerank is the single highest-leverage retrieval component in 2026 benchmarks (+17pp MRR@3 over unreranked hybrid). Provider-abstracted; rerank scores recorded alongside RRF for evaluation.

**Original recommended default (superseded):** RRF fusion in v1; add optional reranker behind feature flag after baseline metrics.

**Before implementation:** Define query classes: exact lookup, policy/contract question, impact analysis, version diff, JSON path/config lookup, and table/spreadsheet lookup.

---

### DD-16 - Citation and Source Anchor Contract

**Decision needed:** Define the citation contract before building retrieval APIs.

| Artifact | Required Anchor |
|---|---|
| PDF | document version + page + block/offset when possible. |
| DOCX/Markdown | document version + heading path + paragraph/line anchor. |
| PPTX | deck version + slide + shape/text block anchor. |
| JSON | document version + JSONPath. |
| Excel v1.5 | workbook version + sheet + cell/range. |

**Recommended default:** No source anchor, no answer. Agent responses without valid source refs fail closed.

**Before implementation:** Build a source-anchor resolver test suite. Every citation in test output must round-trip to source content.

---

### DD-17 - Versioning and Change Impact

**Decision needed:** Decide how versions, supersedence, and diffs are represented.

| Area | Recommended Direction |
|---|---|
| Version identity | Source version + content hash + ingestion run. |
| Change detection | Content-addressed section/block/artifact IDs. |
| Supersedence | Explicit metadata where available; inferred edges separately marked. |
| Diff | Structural diff first, narrative explanation second. |

**Before implementation:** Define impact-analysis semantics. "Impacted" must distinguish direct reference, transitive reference, shared entity, superseded version, and inferred relationship.

---

### DD-18 - Evaluation Harness

**Decision needed:** Define release gates before building features.

| Eval Area | Required Metric |
|---|---|
| Extraction determinism | Same input produces byte-identical graph output. |
| Retrieval quality | precision@10 / recall where golden answers exist. |
| Citation integrity | 100% source refs resolve. |
| Grounding | Claims must be supported by retrieved excerpts, not citation-string echo. |
| ACL correctness | No unauthorized result leakage in negative tests. |
| Tool safety | Prompt-injection and tool-misuse tests pass. |
| Performance | p95/p99 ingestion and query targets agreed per pilot. |

**Recommended default:** Golden evals are required for each design partner corpus before success criteria are signed.

---

### DD-19 - Security Threat Model

**Decision needed:** Define the AI-specific and MCP-specific threat model.

Required threat categories:

- Prompt injection from documents, tables, JSON values, and connector metadata.
- Indirect prompt injection through retrieved content.
- Tool poisoning and misleading tool descriptions.
- Cross-tenant data leakage.
- Sensitive information disclosure in generated answers.
- Insecure plugin/tool design.
- Overbroad agent permissions.
- Secret leakage through logs, traces, embeddings, and prompts.
- Supply-chain compromise in parsers and model libraries.

**Recommended default:** Use STRIDE + OWASP LLM Top 10 + MCP-specific threat review.

**Before implementation:** Produce a threat model and security test checklist before P0 exits.

---

### DD-20 - Data Privacy, PII, and Retention

**Decision needed:** Decide how sensitive data is detected, stored, masked, retained, and deleted.

| Area | Recommended Direction |
|---|---|
| PII detection | Configurable classifiers per tenant/corpus. |
| Embeddings | Treat embeddings as sensitive derived data. |
| Logs/traces | Never log raw document text by default. |
| Deletion | Source deletion must propagate to graph, embeddings, cache, and audit-visible tombstones. |
| Retention | Tenant-configurable retention for raw extracted text, embeddings, audit logs, and generated answers. |

**Before implementation:** Define data classification policy and deletion SLA.

---

### DD-21 - Observability and GenAI Telemetry

**Decision needed:** Define telemetry from day one.

| Signal | Required Examples |
|---|---|
| Logs | Structured logs with tenant, request ID, run ID, connector ID, tool ID. |
| Metrics | ingestion throughput, extraction failure rate, embedding queue depth, query latency, citation failure rate, token cost. |
| Traces | request-to-retrieval-to-agent trace using OpenTelemetry. |
| GenAI telemetry | model name, token counts, latency, retries, safety/fallback events, no raw sensitive prompts by default. |
| Audit | immutable business/security audit log for tool calls and index promotions. |

**Recommended default:** OpenTelemetry-first instrumentation with GenAI semantic conventions where appropriate.

---

### DD-22 - Reliability and SLOs

**Decision needed:** Define SLOs before production pilots.

Initial SLO categories:

- Query availability.
- Query p95/p99 latency.
- Ingestion freshness.
- Ingestion success rate.
- Index promotion correctness.
- Citation resolution rate.
- ACL enforcement correctness.
- Backup/restore RTO/RPO.

**Recommended default:** Separate SLOs for query-serving path and ingestion path. Ingestion failures must not take down query serving.

---

### DD-23 - Cost and Capacity Model

**Decision needed:** Decide how cost is measured, limited, and exposed.

Cost drivers:

- Document download/connector calls.
- Parser CPU/memory.
- OCR/multimodal processing.
- Embedding tokens and vector storage.
- LLM calls for explanations/reranking.
- Postgres storage, indexes, and backups.
- Audit/log/trace volume.

**Recommended default:** Per-tenant metering from P1: documents, versions, pages/slides, JSON nodes, Excel cells/ranges, embedding tokens, LLM tokens, tool calls, and storage.

**Before implementation:** Build cost guardrails: corpus quotas, batch caps, rate limits, OCR feature flags, and budget alerts.

---

### DD-24 - API and Contract Strategy

**Decision needed:** Define stable APIs before agent clients depend on them.

Required contracts:

- REST APIs for ingestion status, search, explain, impact, diff.
- MCP tool schemas generated from code.
- Source reference schema.
- Error schema with retryability and user-action hints.
- Audit event schema.
- Eval result schema.

**Recommended default:** OpenAPI for HTTP APIs, generated MCP schemas for tools, and JSON Schema for internal artifacts.

---

### DD-25 - CI/CD, Supply Chain, and Release Governance

**Decision needed:** Define build/release standards before first commit.

Required controls:

- CODEOWNERS and branch protection.
- Unit, integration, e2e, and golden-eval workflows.
- Dependency scanning and container vulnerability scanning.
- SBOM generation.
- Signed container images.
- Non-root containers.
- IaC scanning.
- Migration tests.
- Doc-path validation.
- No client namespaces or client-specific policy files.

**Recommended default:** CI gates are part of P0, not hardening.

---

### DD-26 - Connector Strategy

**Decision needed:** Decide connector architecture and rollout order.

| Connector | Recommendation |
|---|---|
| File tree | P0 baseline. |
| Git docs | P0/P1; useful for engineering docs and ADRs. |
| SharePoint/OneDrive | P3; enterprise-critical but auth/throttling heavy. |
| Confluence/ServiceNow/Jira | Later, only after document/file artifact foundation is stable. |

**Before implementation:** Connector SDK must define auth, throttling, incremental sync, deletion handling, ACL extraction, retries, and audit events.

---

### DD-27 - Human Review and Admin UX

**Decision needed:** Decide how admins inspect ingestion quality and trust.

Recommended admin capabilities:

- Corpus inventory.
- Unsupported-file report.
- Extraction quality report.
- Citation resolver.
- Ingestion run history.
- Failed/quarantined documents.
- Golden-query dashboard.
- ACL coverage report.
- Cost and usage dashboard.

**Recommended default:** Build a minimal admin console or operational API before pilots. Enterprise customers will ask how they know the system is trustworthy.

---

### DD-28 - RIF/DIF Federation

**✅ RESOLVED — AND REVERSED (D-007, 2026-07-08):** Federation is **core v1 architecture**, not deferred. The original "don't build joint queries in v1" default is overridden: each project already runs a RIF Postgres, so DIF deploys into it (`dif_meta` beside `rif_meta`), `DESCRIBES` doc→code edges land at P1, `docs_for_code`/`code_for_doc` tools at P1, `drift_report` at P2. See `DECISIONS.md` D-007 and PRD v0.3 (R7a, R11, R13c, R14).

Design constraints, now hard requirements:

- Shared NodeIdComputer implementation (one artifact, not a copy) — PRD R6.
- Shared source reference model spanning doc anchors and code `source_ref`s.
- Shared MCP conventions; cross-graph tools return `rif_not_deployed` explicitly on standalone deployments.
- Shared embedding service contracts.
- Documented minimum `rif_meta` schema version, checked at deploy time; cross-graph queries behind a view layer (PRD R11, risk table).
- Cross-product entity model and versioned graph export/import remain v2 work.

---

## 4. Mandatory ADRs Before Build

| ADR | Decision | Required Before |
|---|---|---|
| ADR-001 | Deployment and tenancy model — **resolved by D-001/D-007** (BYOC, per-project, co-located with RIF) | ~~P0 start~~ done |
| ADR-002 | Multi-tenant database isolation — **resolved by D-001/D-007** (per-tenancy + per-project DB) | ~~P0 start~~ done |
| ADR-003 | Source ACL posture for v1 pilots — direction set (uniformly-readable or per-boundary indexes, BRD BR4/BR9); formalize | P0 start |
| ADR-004 | Graph store — **resolved by D-003** (Postgres adjacency + recursive CTEs) | ~~P0 schema work~~ done |
| ADR-005 | Parser strategy per supported format | P0 extraction work |
| ADR-006 | JSON graph expansion limits | P0 JSON ingestion |
| ADR-007 | Source anchor contract | P0 retrieval work |
| ADR-008 | MCP gateway and authorization model | P0 MCP work |
| ADR-009 | Ingestion orchestration pattern | P0 ingestion work |
| ADR-010 | Embedding provider/model strategy — **prose default resolved by D-002** (Voyage, ≤1024d); per-artifact-type choices remain | P0 embedding integration |
| ADR-011 | Evaluation harness and release gates | P0 CI work |
| ADR-012 | Observability and audit event schema | P0 service scaffolding |
| ADR-013 | Security threat model and prompt-injection controls | P0 exit |
| ADR-014 | Excel v1.5 scope and parser choice | v1.5 planning |
| ADR-015 | OCR/multimodal v2 strategy | v2 planning |

---

## 5. Recommended Default Architecture Posture

| Area | Default |
|---|---|
| Cloud posture | Azure-primary, customer-tenant deployment preferred for regulated B2B. |
| Tenancy | DB-per-tenant for enterprise customers. |
| Storage | Postgres + pgvector + FTS + relational graph adjacency. |
| Retrieval | Hybrid vector + FTS + graph signal with RRF; optional reranker later. |
| Ingestion | Async queue/workflow pipeline with idempotency, checkpoints, and atomic index promotion. |
| Extraction | Deterministic parsers first; LLMs only for bounded explanation or later enrichment. |
| JSON | v1 first-class artifact with JSONPath anchors and graph expansion caps. |
| Excel | v1.5 controlled `.xlsx` extraction with caveats. |
| OCR | v2, preceded by v1 document quality profiler. |
| MCP | Governed MCP gateway, generated schemas, auth, audit, tool versioning. |
| Agents | Thin citation-gated agent service; read-only tools in v1. |
| Security | OIDC, RBAC/ABAC, prompt-injection controls, no raw text in logs, OWASP LLM controls. |
| Observability | OpenTelemetry traces/metrics/logs plus GenAI telemetry and immutable audit events. |
| Release gates | Golden evals, citation integrity, ACL negative tests, performance tests, vuln scans. |

---

## 6. Non-Negotiables for Production Grade

1. No unauthenticated MCP or HTTP surface.
2. No generated answer without resolvable source references.
3. No tenant sharing without an explicit tenancy ADR.
4. No source ACL overclaiming in v1.
5. No parser that silently drops unsupported content.
6. No ingestion run may replace a healthy serving index with degenerate output.
7. No raw enterprise document text in logs by default.
8. No hand-maintained MCP schemas when code generation is feasible.
9. No release without golden-query evaluation.
10. No pilot without baseline metrics and written success criteria.

---

## 7. First Engineering Checklist

- Create the ADR folder and write ADR-001 through ADR-004 before coding service internals.
- Define the source reference schema and citation resolver tests.
- Build P0 with `.md`, `.docx`, and `.json` only, but design the extractor interface for PDF/PPTX/Excel/OCR.
- Implement tenant, run, document, node, edge, source anchor, embedding, audit, and eval schemas early.
- Stand up CI with lint, unit tests, e2e doc ingestion, golden evals, vuln scan, container scan, and doc-link validation.
- Create a small public demo corpus covering Word, Markdown, JSON, PDF text-layer, PPTX, and later Excel.
- Write the security threat model before exposing MCP to any external client.
- Define the cost-metering model before the first design-partner pilot.

---

## 8. Final Engineering View

The strongest implementation path is not to chase every document-AI feature at once. Build the trustworthy intelligence substrate first:

1. Deterministic extraction.
2. Stable graph and source anchors.
3. Hybrid retrieval.
4. Citation-gated agent layer.
5. Secure MCP gateway.
6. Eval-driven release gates.
7. Enterprise observability and audit.

Once that foundation holds, Excel, OCR, multimodal extraction, contradiction detection, and RIF+DIF federation become extensions of the factory instead of expensive rewrites.