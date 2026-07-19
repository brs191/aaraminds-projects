# DIF Code Workspace

Runnable DIF implementation lives here.

Current status: minimal Go module skeleton plus SQL migration, typed configuration baseline, safe structured logging helpers, typed request/execution context propagation, corpus admission gate, source-anchor resolver, ingestion-run lifecycle guard, Markdown/TXT/DOCX/JSON extractors, deterministic graph/NDJSON emitter, P0 retrieval package, embedding provider seam, service-layer `search_docs` contract, MCP/API boundary skeleton, audit/usage write path, health/readiness checks, RIF compatibility status checks, P1 code-entity candidate detection, and executable evaluation harnesses. Agent business logic is intentionally not implemented yet.

## Component-root commands

Run Go commands from this directory:

```bash
cd /Users/rb692q/projects/aaraminds-projects/dif/code
```

Full unit test run:

```bash
go test ./...
```

Single-test run for the migration discoverability check:

```bash
go test ./... -run TestInitialMigrationIsDiscoverableFromComponentRoot
```

Build all Go packages and service entry points:

```bash
go build ./...
```

Configuration/logging/request-context/migration/admission/source-anchor/ingestion-run/extraction component tests are included in the full unit test run. They can also be run directly:

```bash
go test ./libs/config ./libs/logging ./libs/requestctx ./libs/migrations ./libs/admission ./libs/sourceanchors ./libs/ingestionruns ./libs/extraction ./libs/graphemit ./libs/retrieval ./libs/embeddings ./libs/searchdocs ./libs/mcpapi ./libs/auditusage ./libs/health ./libs/rifcompat ./libs/codeentities
```

Required P0 configuration environment variables:

| Variable | Purpose |
|---|---|
| `DIF_PROJECT_ID` | Project scope identifier. |
| `DIF_CORPUS_ID` | Corpus scope identifier. |
| `DIF_DATABASE_URL` | Existing per-project RIF Postgres URL. Credentials are redacted before logging. |
| `DIF_ENVIRONMENT` | One of `local`, `test`, `development`, `staging`, `production`. |
| `DIF_LOG_LEVEL` | One of `debug`, `info`, `warn`, `warning`, `error`. |
| `DIF_AUTH_MODE` | Auth mode placeholder: `bearer_token` or `oauth_pkce`. No live secret retrieval is implemented yet. |

Structured logging helpers live in `libs/logging` and intentionally support operational metadata only: IDs, paths, hashes, counts, caveat codes, latency, and statuses. They redact obvious credentials, bearer tokens, private-key markers, secret-like key/value pairs, and raw document text fields.

Corpus admission helpers live in `libs/admission`. They enforce the v1 `uniform_readable` corpus gate, return `corpus_not_admitted` for rejected or missing corpus access, model source admission, and produce denied-audit intent without implementing the full audit write path yet.

Source-anchor helpers live in `libs/sourceanchors`. They parse and format canonical `source_ref` values, compute deterministic anchor IDs and content hashes, resolve P0 Markdown/TXT/DOCX/JSON anchors, and return explicit resolver statuses such as `anchor_not_found`, `anchor_out_of_range`, and `content_hash_mismatch`.

Ingestion-run lifecycle helpers live in `libs/ingestionruns`. They model P0 run statuses and output counts, validate non-negative counts, allow promotion only for completed runs with documents, nodes, anchors, and passages, and return explicit non-promotable reasons such as `run_not_completed` or `degenerate_no_anchors`.

Markdown/TXT/DOCX/JSON extractors live in `libs/extraction`. They emit deterministic document graph records, source anchors, retrieval passages, stable IDs/hashes, and `CONTAINS` edges for the P0 Markdown, TXT, DOCX paragraph-model, and JSON fixtures without using LLMs. DOCX support intentionally starts from the committed paragraph-model fixture and emits user-facing `requirements.docx#pN` source refs without adding a heavy binary parser dependency. JSON extraction also enforces ADR-006 caps, emits machine-readable caveats, fails closed for invalid/too-large JSON, and redacts obvious secret-like values from passages.

Graph-emitter helpers live in `libs/graphemit`. They validate extractor output and emit byte-stable NDJSON records for documents, source anchors, nodes, `CONTAINS` edges, retrieval passages, and caveats. The emitter fails closed on dangling edges, unknown anchors, unanchored passages, and source-ref mismatches.

Retrieval helpers live in `libs/retrieval`. They build an anchored-only P0 lexical retrieval index from extractor output, enforce corpus admission through `libs/admission`, return explicit `ok`, `no_evidence`, or `corpus_not_admitted` statuses, and keep vector search out until embedding dimensions are pinned.

Embedding helpers live in `libs/embeddings`. They define the provider interface and deterministic offline hash provider, validate requests, emit normalized vectors, and record non-PII usage placeholders without adding pgvector schema or pinning production Voyage dimensions.

Search service helpers live in `libs/searchdocs`. They implement the service-layer `search_docs` contract with required scope validation, corpus admission before retrieval, anchored-only result validation, score/caveat propagation, explicit `ok`, `no_evidence`, `corpus_not_admitted`, and fail-closed statuses, and no free-form answer generation.

MCP/API helpers live in `libs/mcpapi`. They implement a thin P0 authenticated transport boundary for `search_docs` with constant-time bearer-token validation, required input checks, tool-style invocation, an HTTP JSON handler, service-layer routing, structured error envelopes, and grounded source-ref responses. Pilot/remote deployments must replace the P0 bearer-token gate with OAuth 2.1 + PKCE.

Audit/usage helpers live in `libs/auditusage`. They validate and write separate `dif_meta.audit_log` and `dif_meta.usage_events` records, hash safe parameters, keep usage records non-PII, and avoid storing raw queries, snippets, document text, or request parameter payloads. The initial migration also seeds a migration-backed unknown-scope auth-audit sentinel corpus for denied authentication attempts that arrive without valid request scope.

Health helpers live in `libs/health`. They implement Postgres-backed health/readiness checks, validate `dif_meta` table inventory, report RIF compatibility status as informational for P0 doc-only mode, avoid leaking connection strings in errors, and expose HTTP health/readiness handlers.

RIF compatibility helpers live in `libs/rifcompat`. They assess ADR-016 deployment states (`rif_not_deployed`, `rif_incompatible`, `rif_shadow_empty`, `rif_compatible`), avoid treating empty/incomplete optional shadows as success or as fatal when AGE/API fallback is compatible, provide deterministic code-entity lookups, compute shared NUL-separated RIF node/edge IDs, and persist status snapshots to `dif_meta.rif_compatibility_status` without mutating RIF-owned schemas.

Code-entity candidate helpers live in `libs/codeentities`. They detect qualified names, source paths, method/class references, backtick spans, code-fence content, service routes, and inline identifier heuristics from anchored document blocks, persist only unresolved candidates to `dif_meta.code_entity_candidates`, preserve source refs, and never create RIF nodes.

The P1-02 resolver also lives in `libs/codeentities` (`resolver.go`). It resolves candidates against `rifcompat` compatibility reports (qualified-name, source-path, simple-name, and fuzzy with PascalCase fallback), keeps ambiguous, unresolved, and `rif_unavailable` outcomes explicit, records exact/inferred confidence and caveats, and measures per-corpus resolution rates. `DESCRIBES` edges are created only from single-match resolver evidence, use the shared RIF/DIF edge-ID algorithm, and are written to `dif_meta.edges` (enabled by migration `002_dif_meta_describes_edges.sql`) — never to RIF-owned schemas. `UpdateResolutions` persists resolver outcomes onto existing candidate rows without inserting new ones.

## Migration runner

`cmd/dif-migrate` applies ordered SQL migrations from `migrations/` and checks the expected `dif_meta` table inventory. It is intentionally scoped to DIF-owned migrations and does not require RIF schemas to exist.

```bash
cd /Users/rb692q/projects/aaraminds-projects/dif/code
DIF_DATABASE_URL='postgres://localhost:5432/dif_migration_check?sslmode=disable' go run ./cmd/dif-migrate apply
DIF_DATABASE_URL='postgres://localhost:5432/dif_migration_check?sslmode=disable' go run ./cmd/dif-migrate apply
DIF_DATABASE_URL='postgres://localhost:5432/dif_migration_check?sslmode=disable' go run ./cmd/dif-migrate check
```

The runner shells out to `psql` with `ON_ERROR_STOP=1`, so local scratch validation still requires the PostgreSQL client tools.

## Targeted evaluation harness commands

Run scaffold evaluation harnesses from the repository root:

```bash
cd /Users/rb692q/projects/aaraminds-projects/dif
python3 evaluation/run_p0.py
python3 evaluation/source_anchor_roundtrip.py
python3 evaluation/json_caveat_checks.py
python3 evaluation/rif_compatibility_checks.py
python3 evaluation/search_docs_checks.py
python3 evaluation/audit_usage_checks.py
python3 evaluation/degenerate_run_checks.py
python3 evaluation/path_checks.py
```

## Layout

| Path | Purpose |
|---|---|
| `go.mod` | Minimal Go module rooted at `github.com/aaraminds/dif`. |
| `services/` | Service entry points: ingestion, retriever, MCP server, later agent service. |
| `cmd/dif-migrate/` | Component-root migration apply/check command for scratch PostgreSQL validation. |
| `libs/` | Shared DIF libraries, including build metadata, typed config, safe structured logging, request/execution context propagation, migration loading/inventory checks, corpus admission, source-anchor resolution, ingestion-run lifecycle guards, deterministic extraction, graph emission, retrieval, embeddings, search service contract, MCP/API boundary, audit/usage writes, health/readiness checks, RIF compatibility, and code-entity candidate detection. |
| `migrations/` | Idempotent SQL migrations for `dif_meta`. |
| `testdata/` | Local test fixtures that are safe to commit. |

Implementation must follow `action_plan.md`, `prompts.md`, and the ADRs under `design/adr/`.
