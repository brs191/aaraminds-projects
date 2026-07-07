# Operations Runbook

## Intent
Operational baseline for running, validating, and troubleshooting the current services/libs/extractors implementation from repository artifacts.

## Preconditions
- Postgres connection string available as `DATABASE_URL`.
- Java, Go, Python/uv, and `psql` available for local execution paths.
- For embedding service local mode, a local model path is required; LiteLLM mode requires endpoint + key. (source: services/embedding-service/app.py)

## Core Bring-Up Sequence (Local/Validation)
1. **Validate repository structure**
   - Run `bash scripts/repo_hygiene_check.sh`.
2. **Validate DB migrations**
   - Run `DATABASE_URL=... scripts/validate_schema_idempotency.sh`.
3. **Run module tests**
   - Run `go test ./...` in `services/ingestion`, `services/retriever`, `services/mcp-server`, `libs/graphstore`, and `libs/phase5`.
   - Run `mvn -q -f extractors/core-java/pom.xml test` and `mvn -q -f extractors/spring-java/pom.xml test`.
   - Run targeted pytest for `services/embedding-service` and `services/agent-service`.

## Service Runtime Notes

### Ingestion Service
- Exposes routes for repo registration/index/status/webhook. (source: services/ingestion/main.go)
- Uses graceful shutdown and background workers for incremental mode. (source: services/ingestion/main.go)

### Embedding Service
- `/health` and `/embed` are stable API surfaces across providers. (source: services/embedding-service/app.py)
- Dimension mismatch raises runtime error to prevent silent bad writes. (source: services/embedding-service/app.py)

### MCP Server
- `/health` endpoint and `/mcp` handler are available.
- Audit table created if missing at startup. (source: services/mcp-server/main.go) (source: services/mcp-server/app.go)

### Agent Service
- `/explain` and `/investigate_impact` map to MCP-backed workflows with explicit HTTP error mapping. (source: services/agent-service/app.py)

## Troubleshooting Guide

| Symptom | Likely cause | Check |
|---|---|---|
| Index run fails before load | Provenance-gap, clone allowlist, extractor, or requested-SHA checkout issue | inspect ingestion logs and `services/ingestion/service/index_service.go` |
| Webhook accepted but no fresh index | lane enqueue/coalescing/reconcile behavior | inspect queue worker + reconcile loops under `libs/phase5/ingestion` |
| MCP request returns validation/tool error | bad input schema or unknown tool | validate against `services/mcp-server/tools.schema.json` and raw tool dispatch |
| Embedding service 500 | provider config, LiteLLM transport/auth, or dimension mismatch | check provider env vars and `services/embedding-service/app.py` |

## Known Operational Gaps
1. Full production observability and Terraformized deployment are phase-6 deferred. (source: prompts/playbook.md#L877-L960)
2. Production deploy workflow coverage is currently ingestion-focused; add equivalent deploy workflows before claiming full-service production readiness.

## Local E2E Evidence — 2026-07-02

Dedicated local database:

```bash
createdb cc_rif_e2e_610d4206
DATABASE_URL='postgres:///cc_rif_e2e_610d4206?sslmode=disable'
psql "$DATABASE_URL" -X -v ON_ERROR_STOP=1 -c 'CREATE EXTENSION IF NOT EXISTS age; CREATE EXTENSION IF NOT EXISTS vector;'
PGOPTIONS='--client-min-messages=warning' scripts/validate_schema_idempotency.sh
```

Service stack:

```bash
EMBEDDING_PROVIDER=hash EMBEDDING_DIM=768 PORT=18000 uv run uvicorn app:app --host 127.0.0.1 --port 18000
DATABASE_URL='postgres:///cc_rif_e2e_610d4206?sslmode=disable' EXTRACTOR_JAR_PATH='../../extractors/core-java/target/rif-extractor-1.0.0-SNAPSHOT-shaded.jar' PORT=18080 RIF_API_TOKEN='local-e2e-token' EMBEDDING_SERVICE_URL='http://127.0.0.1:18000' ALLOWED_CLONE_HOSTS='github.com' go run .
DATABASE_URL='postgres:///cc_rif_e2e_610d4206?sslmode=disable' EMBEDDING_SERVICE_URL='http://127.0.0.1:18000/embed' MCP_SERVER_ADDR='127.0.0.1:18081' go run .
MCP_SERVER_URL='http://127.0.0.1:18081/mcp' PORT=18082 uv run uvicorn app:app --host 127.0.0.1 --port 18082
```

Validated health endpoints:

| Service | Endpoint | Result |
|---|---|---|
| Embedding | `GET http://127.0.0.1:18000/health` | `{"status":"ok","model":"hash-deterministic-v1-768","dim":768}` |
| Ingestion | `GET http://127.0.0.1:18080/healthz` | `{"status":"ok"}` |
| MCP | `GET http://127.0.0.1:18081/health` | `{"status":"ok"}` |
| Agent | `GET http://127.0.0.1:18082/health` | `{"status":"ok","model":"ollama/llama3.1:8b","max_hops":3}` |

Indexed fixture repo:

```text
repo_id: gs-rest-service-e2e-2
clone_url: https://github.com/spring-guides/gs-rest-service.git
sha: 2ef8e28f7139ebd1b9b7a9226f748d43e9f9145f
run_id: 76799e6a-85b0-4299-838c-98ed53a87340
result: complete, node_count=19, edge_count=10, current_index_version=1
```

Validated MCP tools:

| Tool | Input | Result |
|---|---|---|
| `search_code` | `GreetingController`, top 5 | Returned source refs including `complete/src/main/java/com/example/restservice/GreetingController.java:16` |
| `dependency_analysis` | `GreetingController`, depth 3 | Returned valid empty direct/transitive dependency sets |
| `find_callers` | `GreetingController`, depth 2 | Returned valid empty result set |
| `impact_analysis` | `GreetingController`, depth 3 | Returned valid empty impacted set with bounded-graph caveat |

Validated agent endpoints:

| Endpoint | Result |
|---|---|
| `POST /explain` | Returned explanation plus 3 source refs |
| `POST /investigate_impact` | Returned narrative, empty tier map, and 3 source refs |

Runtime fixes discovered during validation:

1. Ingestion expects `EMBEDDING_SERVICE_URL` as the service base URL; MCP expects the `/embed` endpoint URL.
2. Retriever SQL now derives `confidence='exact'` for shadow-table search because `file_nodes`, `method_nodes`, and `class_nodes` do not store a `confidence` column.
3. AGE `DirectCallers` now uses `MATCH (caller)-[e]->... WHERE label(e) IN [...]` because AGE rejected the previous `:SAME_FILE_CALLS|IMPORTS` edge label syntax.

## Assumptions
- This runbook is repository-scoped and intentionally avoids undocumented cloud-environment specifics.
- Any SLO/SLA numbers beyond benchmark and AB report outputs are `[VERIFY]`.
