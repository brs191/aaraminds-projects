# API and Tooling Reference

## Scope
This reference covers currently implemented service APIs, MCP tools, and core operational tooling in this repository.

## Ingestion Service API (Phase 1 + Phase 5 hooks)

| Method | Route | Purpose | Evidence |
|---|---|---|---|
| POST | `/repos` | Register repository (`repo_id`, `clone_url`) with clone-host allowlist | (source: services/ingestion/handler/repos.go) |
| POST | `/repos/{repoID}/index` | Trigger async index run (optional SHA) | (source: services/ingestion/handler/index.go) |
| GET | `/repos/{repoID}/status` | Retrieve latest run status and counts | (source: services/ingestion/handler/status.go) |
| POST | `/webhook/github` | Enqueue incremental jobs from push payloads | (source: services/ingestion/handler/webhook.go) |
| GET | `/healthz` | Service health endpoint | (source: services/ingestion/main.go) |

### Ingestion Runtime Controls
- `PHASE2_EXTRACTORS_ENABLED`, jar-path env vars, embedding settings, `RIF_API_TOKEN`, and `ALLOWED_CLONE_HOSTS` are loaded from config. (source: services/ingestion/config/config.go)
- Incremental worker and reconciler start only when incremental mode is enabled. (source: services/ingestion/main.go)

## Embedding Service API (Phase 2)

| Method | Route | Purpose | Evidence |
|---|---|---|---|
| GET | `/health` | Return service/model/dim status | (source: services/embedding-service/app.py) |
| POST | `/embed` | Embed batch inputs (`node_id`, `text`) | (source: services/embedding-service/app.py) |

### Provider Modes
- `local/jina`, `litellm`, and `hash` providers are supported in current implementation. (source: services/embedding-service/app.py)
- Default model constant is `text-embedding-3-small`; schema migration uses 768-d vectors. (source: services/embedding-service/app.py) (source: data/migrations/migration_pgvector.sql)

## MCP Server Tools (Phase 4)

| Tool | Required inputs | Purpose | Evidence |
|---|---|---|---|
| `search_code` | `repo_id`, `query` | Hybrid code search | (source: services/mcp-server/tools.schema.json) |
| `find_callers` | `repo_id`, `qualified_name` | Caller lookup | (source: services/mcp-server/tools.schema.json) |
| `impact_analysis` | `repo_id`, `changed_entity` | Ranked impact result | (source: services/mcp-server/tools.schema.json) |
| `explain_architecture` | `repo_id`, `component` | Architecture summary path | (source: services/mcp-server/tools.schema.json) |
| `dependency_analysis` | `repo_id`, `entity` | Dependency depth analysis | (source: services/mcp-server/tools.schema.json) |

### MCP Operational Notes
- Server exposes `/health` and `/mcp` endpoints and supports raw JSON-RPC tool calls. (source: services/mcp-server/main.go)
- Query sanitization and audit-log table provisioning are implemented in app initialization. (source: services/mcp-server/app.go)

## Agent Service API (Phase 4)

| Method | Route | Purpose | Evidence |
|---|---|---|---|
| GET | `/health` | Agent runtime health/model/hops | (source: services/agent-service/app.py) |
| POST | `/explain` | Architecture explanation via agent workflow | (source: services/agent-service/app.py) |
| POST | `/investigate_impact` | Multi-step impact investigation | (source: services/agent-service/app.py) |

## Tooling and Scripts

| Tool/Script | Path | Purpose |
|---|---|---|
| Repository hygiene | `scripts/repo_hygiene_check.sh` | Checks migrated structure and tracked binary hygiene |
| Schema idempotency | `scripts/validate_schema_idempotency.sh` | Validates database migrations/idempotency |

## Assumptions
- API contracts listed here are based on source definitions and not external gateway overlays.
- Authentication/authorization front-door behavior is `[VERIFY]` for deployed environments because phase-6 hardening is deferred.
