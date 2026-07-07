# Architecture Deep Dive

## Scope
This deep dive documents the implemented architecture for phases 0–5 and calls out deferred phase-6 capabilities.

## Core Architectural Principle
Deterministic extraction is the system-of-record for graph edges; LLM components narrate and investigate but do not author graph facts. (source: RepoIntelligenceFactory-architecture.md#L10-L16)

## Runtime Topology (Implemented)

| Layer | Primary modules | Design notes |
|---|---|---|
| Extraction + ingestion | `phase-1/extractor`, `phase-1/ingestion`, `phase-2/extractor` | Async pipeline, provenance/degenerate guards, optional Phase-2 extraction merge (source: phase-1/ingestion/service/index_service.go#L151-L165) |
| Graph + metadata | `phase-1/schema`, AGE/pgvector migrations | AGE graph + rif_meta relational + vector columns and ANN indexes (source: phase-2/schema/migration_pgvector.sql#L27-L35) |
| Retrieval | `phase-3/retriever` | Search fan-out (vector/FTS/graph), RRF fusion, bounded depth impact scoring (source: phase-3/retriever/retriever.go#L107-L152) |
| Tool interface | `phase-4/mcp-server` | Official MCP server + streamable HTTP handler + raw tool-call bridge (source: phase-4/mcp-server/main.go#L32-L50) |
| Agent interface | `phase-4/agent-service` | FastAPI endpoints wrapping architecture/impact agents (source: phase-4/agent-service/app.py#L43-L62) |
| Incremental freshness | `phase-5/ingestion`, `phase-5/loader` | Queue + coalescing + reconcile + CAS/fallback swap design (source: phase-5/design/INCREMENTAL_UPDATE_ADR.md#L17-L25) |

## Ingestion and Indexing Flow
1. API entrypoints include repo registration, index trigger, status, and webhook enqueue. (source: phase-1/ingestion/main.go#L5-L9) (source: phase-1/ingestion/main.go#L116-L121)
2. `runPipeline` performs clone → extract → load → swap in one orchestrated flow. (source: phase-1/ingestion/service/index_service.go#L90-L246)
3. Phase-2 extractor outputs are appended into the same NDJSON stream before load when enabled. (source: phase-1/ingestion/service/index_service.go#L461-L509)
4. Provenance-gap metrics are parsed and can fail runs. (source: phase-1/ingestion/service/index_service.go#L430-L459)
5. Version swap uses expected-version compare-and-set semantics to prevent split-brain visibility. (source: phase-5/loader/delta_load.go#L77-L107)

## Retrieval and Analysis Flow
1. Search request embeds query, runs vector + FTS + graph expansion, then fuses via RRF. (source: phase-3/retriever/retriever.go#L129-L151)
2. Impact request computes blast radius and ranks by tier, depth, and hub damping. (source: phase-3/retriever/retriever.go#L182-L219)
3. MCP server exposes five tools; tool schemas constrain required inputs. (source: phase-4/mcp-server/tools.schema.json#L6-L71)
4. MCP app sanitizes tool input tokens and writes audit log entries. (source: phase-4/mcp-server/app.go#L30-L31) (source: phase-4/mcp-server/app.go#L177-L190)

## Agent Plane
- Agent service is FastAPI-based and returns typed responses for explanation and impact narratives. (source: phase-4/agent-service/app.py#L39-L62)
- Agent configuration includes MCP endpoint, model, hop limits, and timeout controls. (source: phase-4/agent-service/config.py#L10-L15)

## Incremental Indexing Architecture (Phase 5)
- Webhook handler classifies changes into lanes and enqueues jobs per classification. (source: phase-1/ingestion/handler/webhook.go#L88-L137)
- Worker coalesces jobs in window and dispatches selected work; marks coalesced rows explicitly. (source: phase-5/ingestion/queue/worker.go#L47-L68)
- ADR defines 30s coalescing and 15m reconciliation policy. (source: phase-5/design/INCREMENTAL_UPDATE_ADR.md#L19-L24) (source: phase-5/design/INCREMENTAL_UPDATE_ADR.md#L104-L109)

## Deferred Architecture Boundary (Phase 6)
Terraformized production topology, private networking hardening, and observability stack are still deferred and should not be documented as live capability. (source: RepoIntelligenceFactory-STATUS.md#L68-L68) (source: prompts/playbook.md#L877-L960)

## Assumptions
- The architecture baseline is inferred from currently checked-in code and scripts, not from superseded planning docs.
- “Deferred” means planned content may exist in prompts/playbooks but no production implementation path exists in this repo yet.

