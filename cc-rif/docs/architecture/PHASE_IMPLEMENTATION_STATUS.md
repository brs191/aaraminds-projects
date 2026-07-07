# Phase Implementation Status

## Status Legend
- **Implemented**: code/artifacts present and integrated
- **Deferred**: planned but not implemented in repository
- **[VERIFY]**: claim needs runtime/environment confirmation

## Phase-by-Phase Status

| Phase | Status | What is implemented | Evidence |
|---|---|---|---|
| 0 — Technical bets | Implemented | Findings memo, benchmark evidence, evalsets | (source: phase-0/FINDINGS_MEMO.md#L1-L23) |
| 1 — Deterministic graph ingestion | Implemented | Ingestion service routes + extractor + graphstore + loader + smoke scripts | (source: phase-1/ingestion/main.go#L5-L13) (source: phase-1/scripts/e2e_smoke.sh#L4-L8) |
| 2 — Tier-B/C + embeddings + migrations | Implemented | DI/AOP/CrossService extractors, embedding service, schema migrations, infra bicep | (source: phase-2/extractor/di/src/main/java/com/att/rif/extractor/di/SpringDiExtractor.java) (source: phase-2/embedding-service/app.py#L255-L315) (source: phase-2/schema/migration_pgvector.sql#L27-L35) |
| 3 — Hybrid retrieval | Implemented | Retriever with fusion and impact scoring + AB report | (source: phase-3/retriever/retriever.go#L107-L220) (source: phase-3/eval/AB_EVAL_REPORT.md#L13-L18) |
| 4 — MCP + agent service | Implemented | MCP server with 5 tools and FastAPI agent endpoints | (source: phase-4/mcp-server/tools.schema.json#L6-L71) (source: phase-4/agent-service/app.py#L43-L62) |
| 5 — Incremental updates | Implemented | Queue worker, diff classification, reconciliation, delta loader CAS/fallback | (source: phase-5/ingestion/queue/worker.go#L47-L69) (source: phase-5/loader/delta_load.go#L50-L75) |
| 6 — Production hardening | Deferred | Terraform, observability, security runbook are still pending | (source: prompts/playbook.md#L877-L960) (source: RepoIntelligenceFactory-STATUS.md#L68-L68) |

## Contradictions Found and Resolved

| Topic | Conflict | Resolution | Why |
|---|---|---|---|
| Overall completion | `RepoIntelligenceFactory-STATUS.md` claims 0–5 complete, but includes older “Phase 2 pending” section in same file | Treat “0–5 complete” as active status; keep older section as stale historical text | Code paths and artifacts for phases 3–5 are present and testable |
| Build-plan progress | Build plan marks Phase 2 as current/not complete | Treat as stale planning state | Build plan dated older snapshot and conflicts with current code state |
| Playbook header vs table | Header says “Phase 2 in progress”; table marks phases 0–5 accepted | Table + code reality wins; header is stale | Internal contradiction in same file |
| Embedding model narrative | Architecture/engine docs still mention Jina/1536 in places; implementation uses 768 + `text-embedding-3-small` default | Use migration + service code as source of truth | Schema and service runtime are implementation facts |

## Source-of-Truth Decision Record (applied policy)
1. **Code/config first**: ingestion, retriever, MCP, agent, incremental components are directly present. (source: phase-1/ingestion/main.go#L95-L121) (source: phase-3/retriever/retriever.go#L107-L152)
2. **Test/result artifacts second**: AB report and module tests support implemented claims. (source: phase-3/eval/AB_EVAL_REPORT.md#L5-L18)
3. **Docs third**: roadmap/status/playbook used only after conflict filtering.

## Assumptions
- “Implemented” does not imply full production hardening.
- Environment-level validations (network isolation, OTel pipelines, Terraform apply) remain phase-6 scope.

