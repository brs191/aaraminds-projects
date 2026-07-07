# System Overview

## Purpose
Repo Intelligence Factory is a brownfield code-intelligence platform that builds a deterministic graph from source code, enriches retrieval with vector + lexical + graph signals, and exposes analysis through MCP tools and an agent service. The implementation is present through phases 0–5; phase 6 is deferred. (source: RepoIntelligenceFactory-STATUS.md#L3-L8) (source: phase-1/ingestion/main.go#L1-L13)

## Current Maturity Snapshot

| Area | State | Evidence |
|---|---|---|
| Deterministic ingestion graph pipeline | Implemented | `phase-1` extractor/ingestion/graphstore/loader exist and are wired (source: phase-1/ingestion/main.go#L95-L121) |
| Phase-2 extractor + embedding integration | Implemented in code path | Phase-2 extractors invoked from ingestion pipeline (source: phase-1/ingestion/service/index_service.go#L161-L165) |
| Hybrid retrieval | Implemented and benchmarked | Hybrid service and AB report exist (source: phase-3/retriever/retriever.go#L107-L152) (source: phase-3/eval/AB_EVAL_REPORT.md#L5-L18) |
| MCP tool surface | Implemented | 5 tools are registered and schema-defined (source: phase-4/mcp-server/main.go#L32-L50) (source: phase-4/mcp-server/tools.schema.json#L6-L71) |
| Agent endpoints | Implemented | `/health`, `/explain`, `/investigate_impact` exist (source: phase-4/agent-service/app.py#L39-L62) |
| Incremental indexing | Implemented | Queue worker, reconciliation, delta loader, ADR present (source: phase-5/ingestion/queue/worker.go#L13-L69) (source: phase-5/loader/delta_load.go#L39-L75) |
| Production hardening (Terraform/observability/private networking) | Deferred | Phase 6 deferred in status/playbook (source: RepoIntelligenceFactory-STATUS.md#L68-L68) (source: prompts/playbook.md#L34-L35) |

## High-Level Component Model

| Plane | Components | Responsibility |
|---|---|---|
| Ingestion | Phase-1 ingestion service + Phase-1/2 extractors + loader | Register repos, run extraction pipeline, bulk load graph data, manage index version swaps |
| Storage | Postgres + AGE + pgvector + relational metadata | Graph traversal, vector search, metadata and run tracking |
| Query | Phase-3 retriever + Phase-4 MCP server | Hybrid retrieval and tool-facing API |
| Reasoning | Phase-4 agent-service | Narrative explanation and impact investigation over MCP tools |
| Freshness | Phase-5 queue/reconcile/delta loader | Coalesced incremental updates with fallback to full reindex |

## Request/Index Lifecycle (Implemented)
1. Repo registered and indexing triggered over ingestion API. (source: phase-1/ingestion/handler/repos.go#L10-L44) (source: phase-1/ingestion/handler/index.go#L13-L50)
2. Ingestion pipeline clones, extracts Tier-A and optional Phase-2 edges, parses NDJSON, applies degenerate/provenance safeguards, bulk loads, and swaps index version atomically. (source: phase-1/ingestion/service/index_service.go#L93-L246)
3. Retriever executes vector + FTS + graph-signal fusion for search and graph-based impact scoring. (source: phase-3/retriever/retriever.go#L107-L220)
4. MCP server exposes normalized tooling (`search_code`, `find_callers`, `impact_analysis`, `explain_architecture`, `dependency_analysis`). (source: phase-4/mcp-server/tools.schema.json#L6-L71)
5. Agent-service wraps MCP tool use for `/explain` and `/investigate_impact`. (source: phase-4/agent-service/app.py#L43-L62)
6. Incremental webhook path classifies lane A/B/C, dispatches queue work, and falls back to full reindex on swap exhaustion. (source: phase-1/ingestion/handler/webhook.go#L36-L167) (source: phase-5/loader/delta_load.go#L58-L75)

## Assumptions
- “Implemented” means code and key artifacts are present in this repository and major targeted tests run successfully in this workspace.
- Environment-specific production assertions (Azure networking, live OTel, Terraform applies) remain deferred unless Phase-6 artifacts exist.
- Where documents conflict, this overview follows code + test/result artifacts first.

