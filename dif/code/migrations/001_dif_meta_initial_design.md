# 001 DIF Meta Initial Schema Design

**Status:** Design baseline implemented by `001_dif_meta_initial.sql`  
**Date:** 2026-07-08  
**Owners:** Engineering + Platform  
**Related ADRs:** ADR-003, ADR-006, ADR-007, ADR-016  
**Target schema:** `dif_meta`  
**Important boundary:** DIF migrations must not mutate RIF-owned schemas such as `rif`, `rif_meta`, or future RIF equivalents.

---

## 1. Purpose

This document defines the initial P0 schema shape for DIF. It has been implemented by `001_dif_meta_initial.sql`; future schema changes should be additive and captured in new migrations unless this design is explicitly superseded.

P0 schema must support:

- uniformly readable corpora
- deterministic document ingestion
- immutable document versions
- source-anchor round trips
- retrieval passages
- audit logging
- usage metering
- RIF compatibility status
- future `DESCRIBES` edges without implementing cross-graph federation yet

---

## 2. Migration principles

Executable migrations derived from this design must remain:

1. Idempotent.
2. Safe to run more than once.
3. Additive unless an explicit migration ADR approves a destructive change.
4. Scoped only to `dif_meta`.
5. Compatible with local Postgres and Azure Postgres Flexible Server.
6. Explicit about extensions required for vector/FTS.
7. Validated by a migration idempotency test before P0 exit.

---

## 3. Schema overview

Initial P0 tables:

| Table | Purpose |
|---|---|
| `dif_meta.corpora` | Corpus-level boundary, admission status, and v1 uniformly readable policy. |
| `dif_meta.sources` | Source roots/files/connectors included in a corpus. |
| `dif_meta.documents` | Logical documents independent of version. |
| `dif_meta.document_versions` | Immutable document versions and content hashes. |
| `dif_meta.nodes` | Document graph nodes: `document`, `section`, `block`. |
| `dif_meta.edges` | Document graph edges: P0 `CONTAINS`; later `REFERENCES`, `VERSION_OF`, `SUPERSEDES`, `DESCRIBES`. |
| `dif_meta.source_anchors` | Resolvable source anchors per ADR-007. |
| `dif_meta.retrieval_passages` | Derived retrieval units mapped back to anchors/nodes. |
| `dif_meta.ingestion_runs` | Run lifecycle, counts, metrics, and degenerate-run guard evidence. |
| `dif_meta.audit_log` | Security/audit trail for MCP/API calls. |
| `dif_meta.usage_events` | Non-PII usage/metering events, separate from audit. |
| `dif_meta.rif_compatibility_status` | RIF capability/status snapshots per ADR-016. |
| `dif_meta.code_entity_candidates` | Unresolved or pending code-entity references detected in document text. |

---

## 4. Common conventions

### 4.1 IDs

Use text IDs for deterministic/content-addressed identifiers.

Recommended ID patterns:

| ID | Algorithm |
|---|---|
| `document_id` | `sha256(corpus_id + NUL + normalized_source_uri)` |
| `document_version_id` | `sha256(document_id + NUL + content_hash + NUL + extractor_version)` |
| `node_id` | `sha256(corpus_id + NUL + document_version_id + NUL + node_kind + NUL + normalized_node_path)` |
| `edge_id` | `sha256(from_node_id + NUL + edge_kind + NUL + to_node_id + NUL + discriminator)` |
| `anchor_id` | `sha256(corpus_id + NUL + document_version_id + NUL + anchor_type + NUL + normalized_anchor_payload)` |
| `passage_id` | `sha256(anchor_id + NUL + passage_kind + NUL + normalized_passage_text_hash)` |

### 4.2 Status fields

Status columns should use checked text enums in SQL for P0. Native Postgres enum types can be considered later if migration ergonomics justify them.

### 4.3 Timestamps

Use `TIMESTAMPTZ NOT NULL DEFAULT now()` for creation timestamps. Immutable records should not have mutable `updated_at` unless they are intentionally stateful.

### 4.4 JSONB

Use `JSONB` for:

- caveats
- run metrics
- parser metadata
- usage dimensions
- compatibility capabilities

JSONB fields must not become a substitute for required indexed columns.

---

## 5. Table designs

## 5.1 `dif_meta.corpora`

Corpus-level authorization and admission boundary.

| Column | Type | Required | Notes |
|---|---|---:|---|
| `corpus_id` | TEXT | Yes | Primary key. Stable and safe for source refs. |
| `project_id` | TEXT | Yes | Project/deployment unit. Usually aligns to RIF project/repo context. |
| `display_name` | TEXT | Yes | Human-readable. |
| `admission_status` | TEXT | Yes | `pending`, `admitted`, `rejected`, `archived`. |
| `readability_model` | TEXT | Yes | P0 must be `uniform_readable`. |
| `admission_evidence` | JSONB | No | Owner confirmation, scope notes, review metadata. |
| `created_at` | TIMESTAMPTZ | Yes | Creation time. |
| `updated_at` | TIMESTAMPTZ | Yes | State changes. |

Required constraints:

- primary key on `corpus_id`
- check `readability_model = 'uniform_readable'` for P0
- check `admission_status in ('pending','admitted','rejected','archived')`

---

## 5.2 `dif_meta.sources`

Source file roots or connector scopes.

| Column | Type | Required | Notes |
|---|---|---:|---|
| `source_id` | TEXT | Yes | Primary key. |
| `corpus_id` | TEXT | Yes | FK to corpora. |
| `source_type` | TEXT | Yes | `local_tree`, `git`, `sharepoint`, `onedrive`, future. |
| `source_uri` | TEXT | Yes | Path, repo URL, connector object URI, etc. |
| `scope_path` | TEXT | No | Subtree/folder scope. |
| `admission_status` | TEXT | Yes | Mirrors corpus admission at source level. |
| `created_at` | TIMESTAMPTZ | Yes | Creation time. |

Required constraints:

- FK `corpus_id`
- check source type
- no source may be indexed unless corpus/source admission allows it

---

## 5.3 `dif_meta.documents`

Logical document identity.

| Column | Type | Required | Notes |
|---|---|---:|---|
| `document_id` | TEXT | Yes | Primary key. |
| `corpus_id` | TEXT | Yes | FK to corpora. |
| `source_id` | TEXT | Yes | FK to sources. |
| `source_uri` | TEXT | Yes | Stable source object/path. |
| `path` | TEXT | Yes | Normalized display/path. |
| `format` | TEXT | Yes | `md`, `txt`, `docx`, `json`, later `pdf`, `pptx`, `xlsx`. |
| `current_version_id` | TEXT | No | Points to current document version after promotion. |
| `created_at` | TIMESTAMPTZ | Yes | Creation time. |
| `updated_at` | TIMESTAMPTZ | Yes | State changes. |

Required indexes:

- `(corpus_id, path)`
- `(source_id, path)`

---

## 5.4 `dif_meta.document_versions`

Immutable document version record.

| Column | Type | Required | Notes |
|---|---|---:|---|
| `document_version_id` | TEXT | Yes | Primary key. |
| `document_id` | TEXT | Yes | FK to documents. |
| `corpus_id` | TEXT | Yes | Denormalized for scoping/indexes. |
| `source_id` | TEXT | Yes | Source scope. |
| `content_hash` | TEXT | Yes | Hash of original source content. |
| `source_size_bytes` | BIGINT | No | Source size. |
| `format` | TEXT | Yes | Admitted format. |
| `extractor_name` | TEXT | Yes | Parser/extractor. |
| `extractor_version` | TEXT | Yes | Parser/extractor version. |
| `parser_metadata` | JSONB | No | Format-specific metadata. |
| `created_at` | TIMESTAMPTZ | Yes | Immutable creation time. |

Required constraints:

- unique `(document_id, content_hash, extractor_version)`
- document versions are immutable; do not update content metadata in place

---

## 5.5 `dif_meta.nodes`

Document graph nodes.

| Column | Type | Required | Notes |
|---|---|---:|---|
| `node_id` | TEXT | Yes | Primary key. |
| `corpus_id` | TEXT | Yes | Scope. |
| `document_id` | TEXT | Yes | Logical doc. |
| `document_version_id` | TEXT | Yes | Immutable version. |
| `node_kind` | TEXT | Yes | `document`, `section`, `block`. |
| `parent_node_id` | TEXT | No | Parent in document hierarchy. |
| `ordinal` | INTEGER | Yes | Stable ordering among siblings. |
| `heading_path` | TEXT | No | For Markdown/DOCX sections. |
| `anchor_id` | TEXT | No | Primary anchor when applicable. |
| `text_hash` | TEXT | No | Hash of normalized node text. |
| `caveats` | JSONB | No | Extraction caveats. |
| `created_at` | TIMESTAMPTZ | Yes | Creation time. |

Required indexes:

- `(corpus_id, document_version_id)`
- `(document_version_id, node_kind, ordinal)`
- `(parent_node_id)`

---

## 5.6 `dif_meta.edges`

Document graph edges.

| Column | Type | Required | Notes |
|---|---|---:|---|
| `edge_id` | TEXT | Yes | Primary key. |
| `corpus_id` | TEXT | Yes | Scope. |
| `document_version_id` | TEXT | Yes | Version scope. |
| `edge_kind` | TEXT | Yes | P0 `CONTAINS`; later `REFERENCES`, `VERSION_OF`, `SUPERSEDES`, `DESCRIBES`. |
| `from_node_id` | TEXT | Yes | DIF node ID. |
| `to_node_id` | TEXT | No | DIF node ID for intra-DIF edges. |
| `to_external_node_id` | TEXT | No | RIF code node ID for future `DESCRIBES`. |
| `external_system` | TEXT | No | Example: `rif`. |
| `confidence` | TEXT | Yes | `exact`, `inferred`. |
| `anchor_id` | TEXT | No | Source anchor for evidence-bearing edges. |
| `caveats` | JSONB | No | Edge caveats. |
| `created_at` | TIMESTAMPTZ | Yes | Creation time. |

Required constraints:

- P0 permits only `CONTAINS`
- future `DESCRIBES` requires `to_external_node_id` and `external_system = 'rif'`

Required indexes:

- `(corpus_id, edge_kind)`
- `(from_node_id)`
- `(to_node_id)`
- `(to_external_node_id)` for future cross-graph lookups

---

## 5.7 `dif_meta.source_anchors`

Source-anchor contract from ADR-007.

| Column | Type | Required | Notes |
|---|---|---:|---|
| `anchor_id` | TEXT | Yes | Primary key. |
| `corpus_id` | TEXT | Yes | Scope. |
| `document_id` | TEXT | Yes | Logical doc. |
| `document_version_id` | TEXT | Yes | Immutable version. |
| `source_id` | TEXT | Yes | Source scope. |
| `anchor_type` | TEXT | Yes | `md`, `txt`, `docx`, `json`, future. |
| `source_ref` | TEXT | Yes | Canonical source ref. |
| `path` | TEXT | Yes | Source path. |
| `heading_path` | TEXT | No | Markdown/DOCX. |
| `line_start` | INTEGER | No | Markdown/TXT. |
| `line_end` | INTEGER | No | Markdown/TXT. |
| `paragraph_index` | INTEGER | No | DOCX. |
| `json_path` | TEXT | No | JSON. |
| `content_hash` | TEXT | Yes | Hash of anchored excerpt/block. |
| `extractor_version` | TEXT | Yes | Extractor version. |
| `caveats` | JSONB | No | Caveats. |
| `created_at` | TIMESTAMPTZ | Yes | Creation time. |

Required indexes:

- unique `source_ref`
- `(corpus_id, document_version_id)`
- `(anchor_type)`
- `(json_path)` where anchor type is JSON

---

## 5.8 `dif_meta.retrieval_passages`

Derived retrieval units.

| Column | Type | Required | Notes |
|---|---|---:|---|
| `passage_id` | TEXT | Yes | Primary key. |
| `corpus_id` | TEXT | Yes | Scope. |
| `document_id` | TEXT | Yes | Logical doc. |
| `document_version_id` | TEXT | Yes | Version. |
| `node_id` | TEXT | Yes | Source graph node. |
| `anchor_id` | TEXT | Yes | Round-trip source anchor. |
| `passage_kind` | TEXT | Yes | `structural`, `json_subtree`, etc. |
| `text` | TEXT | Yes | Bounded retrieval text. |
| `text_hash` | TEXT | Yes | Hash of retrieval text. |
| `fts_vector` | TSVECTOR | Future/P0 optional | Full-text search vector. |
| `embedding` | VECTOR | Future/P0 optional | pgvector embedding; exact dimension decided later. |
| `embedding_model` | TEXT | No | Model identifier. |
| `caveats` | JSONB | No | Truncation/parser caveats. |
| `created_at` | TIMESTAMPTZ | Yes | Creation time. |

P0 may start with FTS only plus a stub/hash embedding provider. If pgvector is enabled in P0, dimension must follow ADR-010 once accepted.

Required indexes:

- `(corpus_id, document_version_id)`
- `(anchor_id)`
- FTS GIN index when `fts_vector` exists
- vector ANN index when `embedding` exists

---

## 5.9 `dif_meta.ingestion_runs`

Ingestion lifecycle and promotion evidence.

| Column | Type | Required | Notes |
|---|---|---:|---|
| `run_id` | UUID | Yes | Primary key. |
| `corpus_id` | TEXT | Yes | Scope. |
| `source_id` | TEXT | No | Optional source scope. |
| `started_at` | TIMESTAMPTZ | Yes | Start time. |
| `completed_at` | TIMESTAMPTZ | No | Completion time. |
| `status` | TEXT | Yes | `running`, `completed`, `failed`, `cancelled`. |
| `stage` | TEXT | No | Current stage. |
| `document_count` | INTEGER | No | Count. |
| `node_count` | INTEGER | No | Count. |
| `edge_count` | INTEGER | No | Count. |
| `anchor_count` | INTEGER | No | Count. |
| `passage_count` | INTEGER | No | Count. |
| `caveat_count` | INTEGER | No | Count. |
| `run_metrics` | JSONB | No | Parser/extraction metrics. |
| `error_message` | TEXT | No | Failure details. |
| `promoted` | BOOLEAN | Yes | True only after atomic promotion. |

Degenerate-run guard:

- zero usable documents/nodes must not promote
- all-failed extraction must not promote
- promotion must be recorded explicitly

---

## 5.10 `dif_meta.audit_log`

Security/audit trail.

| Column | Type | Required | Notes |
|---|---|---:|---|
| `audit_id` | BIGSERIAL | Yes | Primary key. |
| `occurred_at` | TIMESTAMPTZ | Yes | Event time. |
| `principal_id` | TEXT | Yes | Caller identity. |
| `tenant_id` | TEXT | No | Tenant/customer. |
| `project_id` | TEXT | Yes | Project. |
| `corpus_id` | TEXT | Yes | Corpus. |
| `tool_name` | TEXT | Yes | MCP/API tool. |
| `tool_version` | TEXT | No | Tool schema version. |
| `parameters_hash` | TEXT | Yes | Hash only; not raw parameters. |
| `outcome` | TEXT | Yes | `success`, `error`, `denied`. |
| `latency_ms` | INTEGER | No | Latency. |
| `source_refs` | JSONB | No | Returned source refs only; no raw excerpts. |
| `error_class` | TEXT | No | Machine-readable error. |

Audit logs must not store raw enterprise document text by default.

---

## 5.11 `dif_meta.usage_events`

Usage/metering events separate from audit.

| Column | Type | Required | Notes |
|---|---|---:|---|
| `usage_event_id` | UUID | Yes | Primary key. |
| `occurred_at` | TIMESTAMPTZ | Yes | Event time. |
| `event_type` | TEXT | Yes | `ingestion_run`, `document_indexed`, `embedding_batch`, `mcp_tool_call`, `agent_request`, `connector_sync`. |
| `tenant_id` | TEXT | No | Tenant/customer. |
| `project_id` | TEXT | Yes | Project. |
| `corpus_id` | TEXT | Yes | Corpus. |
| `connector_id` | TEXT | No | Connector. |
| `counts` | JSONB | No | Counts. |
| `latency_ms` | INTEGER | No | Latency. |
| `token_units` | INTEGER | No | Model token units. |
| `embedding_units` | INTEGER | No | Embedding units. |
| `error_class` | TEXT | No | Error class. |

Usage events must be non-PII and must not include raw document text.

---

## 5.12 `dif_meta.rif_compatibility_status`

RIF compatibility snapshot from ADR-016.

| Column | Type | Required | Notes |
|---|---|---:|---|
| `status_id` | UUID | Yes | Primary key. |
| `project_id` | TEXT | Yes | Project/deployment. |
| `checked_at` | TIMESTAMPTZ | Yes | Check time. |
| `rif_status` | TEXT | Yes | `rif_not_deployed`, `rif_incompatible`, `rif_shadow_empty`, `rif_compatible`. |
| `database_name` | TEXT | No | Database checked. |
| `capabilities` | JSONB | No | Labels/views/fields found. |
| `missing_capabilities` | JSONB | No | Missing fields/labels. |
| `caveats` | JSONB | No | Compatibility caveats. |

Required index:

- `(project_id, checked_at DESC)`

---

## 5.13 `dif_meta.code_entity_candidates`

Detected but unresolved or pending code references in documents.

| Column | Type | Required | Notes |
|---|---|---:|---|
| `candidate_id` | TEXT | Yes | Primary key. |
| `corpus_id` | TEXT | Yes | Scope. |
| `document_id` | TEXT | Yes | Logical doc. |
| `document_version_id` | TEXT | Yes | Version. |
| `node_id` | TEXT | Yes | Document node where candidate appeared. |
| `anchor_id` | TEXT | Yes | Source anchor. |
| `candidate_text` | TEXT | Yes | Bounded candidate string. |
| `candidate_kind` | TEXT | No | `class`, `method`, `file_path`, `service`, unknown. |
| `match_status` | TEXT | Yes | `unresolved`, `resolved`, `ambiguous`, `rif_unavailable`. |
| `resolved_rif_node_id` | TEXT | No | RIF node if resolved later. |
| `match_mode` | TEXT | No | qualified-name, source-path, simple-name, fuzzy. |
| `confidence` | TEXT | No | exact/inferred. |
| `caveats` | JSONB | No | Ambiguity/truncation notes. |
| `created_at` | TIMESTAMPTZ | Yes | Creation time. |
| `resolved_at` | TIMESTAMPTZ | No | Resolution time. |

P0 may populate candidates opportunistically but does not need to create `DESCRIBES` edges.

---

## 6. Extension requirements

Executable migration may require:

| Extension | Purpose | P0 requirement |
|---|---|---|
| `pgcrypto` | UUID/hash helpers if used in SQL | likely |
| `vector` | pgvector embeddings | optional until embedding dimension is pinned |

FTS uses built-in PostgreSQL `tsvector` / GIN support.

---

## 7. Open design items before SQL

Before writing executable SQL, decide:

1. Whether P0 creates pgvector columns immediately or defers until ADR-010 pins dimensions.
2. Whether `document.current_version_id` should be nullable until first promotion or moved to a serving pointer table.
3. Exact check constraints for status columns.
4. Whether audit and usage tables are partitioned later by date/tenant.
5. Whether source content blobs are stored in Postgres, filesystem, object storage, or only referenced.

Recommended P0 default:

- create all non-vector columns now
- defer vector column until embedding dimension is pinned
- use FTS in P0
- keep source content storage decision explicit in ingestion design

---

## 8. P0 acceptance criteria for executable migration

The executable migration derived from this design is accepted when:

1. It creates `dif_meta` idempotently.
2. It creates all P0 tables idempotently.
3. It does not alter `rif` or `rif_meta`.
4. It can run twice successfully on an empty database.
5. It can run against a database containing RIF schemas.
6. It supports source-anchor round-trip fields.
7. It supports corpus admission status.
8. It supports RIF compatibility status.
9. It separates audit logs from usage events.
10. It has a documented rollback/recreate approach for local dev.
