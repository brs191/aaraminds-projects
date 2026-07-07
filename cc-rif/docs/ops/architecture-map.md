# cc-rif Baseline Architecture Map

## Scope

This baseline captures the source-to-target migration entry point from `repo-intelligence-factory` into `cc-rif`.

## Runtime capability map

| Capability | Source path | Target path | Status |
|---|---|---|---|
| Ingestion API + queue/reconcile wiring | `phase-1/ingestion` | `services/ingestion` | Migrated (copy baseline) |
| Retriever | `phase-3/retriever` | `services/retriever` | Migrated (copy baseline) |
| MCP server | `phase-4/mcp-server` | `services/mcp-server` | Migrated (copy baseline) |
| Embedding service | `phase-2/embedding-service` | `services/embedding-service` | Migrated (copy baseline) |
| Agent service | `phase-4/agent-service` | `services/agent-service` | Migrated (copy baseline) |
| Core extractor (Java) | `phase-1/extractor` | `extractors/core-java` | Migrated (copy baseline) |
| Spring extractors (Java) | `phase-2/extractor` | `extractors/spring-java` | Migrated (copy baseline) |
| Graphstore library | `phase-1/graphstore` | `libs/graphstore` | Migrated (copy baseline) |
| Schema + migrations | `phase-1/schema`, `phase-2/schema` | `data/schema`, `data/migrations` | Migrated (copy baseline) |
| CI/deploy workflows | `.github/workflows/*` | `platform/ci`, `platform/deploy` | Migrated (copy baseline) |
| Governance docs | root governance files | `governance/*` | Migrated |

## Current migration phase

- Phase 0 (baseline capture): complete.
- Phase 1 (scaffold): complete.
- Phase 2 (governance/docs preservation): complete.
- Phase 3+ (code adaptation for independent build/runtime): in progress.

## Validation entry commands (baseline)

- `cd services/ingestion && go test ./...`
- `cd services/retriever && go test ./...`
- `cd services/mcp-server && go test ./...`
- `cd services/agent-service && uv sync --system-certs --quiet && uv run python -m pytest tests/test_agents.py -q && uv run python -m pytest tests/test_e2e.py -q`
- `cd services/embedding-service && uv sync --system-certs --quiet && uv run pytest -q`

