# cc-rif

Repo Intelligence Factory builds a deterministic code graph from source code, enriches it with lexical/vector retrieval, and exposes analysis through MCP tools and an agent service.

## Layout

| Path | Purpose |
|---|---|
| `services/ingestion` | Repo registration, indexing, GitHub webhook ingestion, incremental worker startup |
| `services/retriever` | Hybrid retrieval and impact-ranking logic |
| `services/mcp-server` | MCP tool surface over graph/retrieval capabilities |
| `services/embedding-service` | FastAPI embedding service with local, LiteLLM, and hash providers |
| `services/agent-service` | FastAPI explanation and impact-investigation API |
| `libs/graphstore` | GraphStore interface plus JSON and Postgres/AGE implementations |
| `libs/phase5` | Incremental diff, queue, reconcile, and delta-load logic |
| `extractors/core-java` | Core Java AST extractor |
| `extractors/spring-java` | Spring DI/AOP/cross-service extractors |
| `data/migrations` | Database migrations |
| `platform` | Deployment and CI reference assets |

## Local validation

```bash
bash scripts/repo_hygiene_check.sh

(cd services/ingestion && go test ./...)
(cd services/retriever && go test ./...)
(cd services/mcp-server && go test ./...)
(cd libs/graphstore && go test ./...)
(cd libs/phase5 && go test ./...)

mvn -q -f extractors/core-java/pom.xml test
mvn -q -f extractors/spring-java/pom.xml test

(cd services/agent-service && uv run python -m pytest -q)
(cd services/embedding-service && uv run pytest -q)
```

## Clean install workflow

```bash
./clean.sh
./run.sh
```
