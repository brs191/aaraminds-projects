# Document Consistency Matrix

## 1) Claim Traceability Matrix

| Claim | Document section | Evidence | Status |
|---|---|---|---|
| Ingestion exposes repo/index/status/webhook APIs | `API_AND_TOOLING_REFERENCE.md` | (source: phase-1/ingestion/main.go#L5-L9) (source: phase-1/ingestion/main.go#L116-L121) | Verified |
| Phase-2 extractors are wired in ingestion path | `SYSTEM_OVERVIEW.md`, `ARCHITECTURE_DEEP_DIVE.md` | (source: phase-1/ingestion/service/index_service.go#L161-L165) | Verified |
| Hybrid retrieval uses vector + FTS + graph signal with fusion | `ARCHITECTURE_DEEP_DIVE.md`, `PHASE_IMPLEMENTATION_STATUS.md` | (source: phase-3/retriever/retriever.go#L107-L152) | Verified |
| MCP exposes 5 tools with typed schemas | `API_AND_TOOLING_REFERENCE.md` | (source: phase-4/mcp-server/tools.schema.json#L6-L71) | Verified |
| Agent service exposes `/explain` and `/investigate_impact` | `API_AND_TOOLING_REFERENCE.md` | (source: phase-4/agent-service/app.py#L43-L62) | Verified |
| Incremental indexing uses queue/coalescing/reconcile/CAS fallback | `ARCHITECTURE_DEEP_DIVE.md`, `OPERATIONS_RUNBOOK.md` | (source: phase-5/ingestion/queue/worker.go#L47-L69) (source: phase-5/loader/delta_load.go#L58-L75) | Verified |
| Phase 6 is deferred | `SYSTEM_OVERVIEW.md`, `PHASE_IMPLEMENTATION_STATUS.md`, `KNOWN_GAPS_AND_RISKS.md` | (source: RepoIntelligenceFactory-STATUS.md#L68-L68) (source: prompts/playbook.md#L877-L960) | Verified |

## 2) Contradiction Register

| Conflict ID | Conflicting statements | Sources | Selected truth source | Rationale | Resolution |
|---|---|---|---|---|---|
| C1 | “Phases 0–5 complete” vs embedded “Phase 2 pending” historical section | `RepoIntelligenceFactory-STATUS.md` top and lower sections | Code + module artifacts | Implementation for phases 3–5 exists | Resolved; lower section treated as stale history |
| C2 | Build plan says Phase 2 current/not complete | `RepoIntelligenceFactory-build-plan.md` | Code + newer status docs | Build plan is older planning snapshot | Resolved; mark build plan as historical planning unless refreshed |
| C3 | Playbook header says Phase 2 in progress; phase table says 0–5 accepted | `prompts/playbook.md` | Playbook phase table + code | Internal inconsistency in same file | Resolved; header considered stale |
| C4 | Embedding architecture mentions Jina/1536; implementation uses 768 + `text-embedding-3-small` defaults | `RepoIntelligenceFactory-engine-plan.md`, `RepoIntelligenceFactory-architecture.md`, `phase-2/schema/migration_pgvector.sql`, `phase-2/embedding-service/app.py` | Schema + service code | Runtime/schema truth overrides older design-target narrative | Resolved; keep Jina notes as historical/alternative context |
| C5 | Closure doc references `phase-4/agent-service/main.py`; repository contains `app.py` | `FINAL_SESSION_CLOSURE.md`, `phase-4/agent-service/app.py` | Repository path reality | File-path mismatch | Resolved; use actual path in all new docs |

## 3) Coverage Matrix

| Required topic | Covered in | Completeness |
|---|---|---|
| System purpose and maturity | `SYSTEM_OVERVIEW.md` | Complete |
| Detailed architecture | `ARCHITECTURE_DEEP_DIVE.md` | Complete |
| Phase-by-phase status (0–6) | `PHASE_IMPLEMENTATION_STATUS.md` | Complete |
| APIs and tooling contracts | `API_AND_TOOLING_REFERENCE.md` | Complete |
| Operations and troubleshooting | `OPERATIONS_RUNBOOK.md` | Complete |
| Known gaps and risks | `KNOWN_GAPS_AND_RISKS.md` | Complete |
| Contradiction and traceability governance | `DOCUMENT_CONSISTENCY_MATRIX.md` | Complete |

## 4) Open `[VERIFY]` Items
1. Production environment enforcement for private networking and OTel signal export (phase 6 scope).
2. Repository hygiene policy execution for generated artifacts and nested fixture repo state.

## Assumptions
- This matrix tracks consistency for the generated `doc/` package and major repository sources referenced during this run.
- It does not replace future CI-based link/path validation and should be refreshed when phase status changes.
