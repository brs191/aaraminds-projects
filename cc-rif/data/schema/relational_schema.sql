-- =============================================================================
-- RIF Phase 1 — Relational Metadata Schema
-- =============================================================================
-- Stack : Postgres 14.23 (local dev) / PG16 (Azure Flexible Server)
-- Run as: psql -U <superuser> -d <database> -f relational_schema.sql
-- Reads with: phase-1/design/CODE_MODEL.md, age_schema.sql
-- Idempotent: safe to run multiple times.
--
-- This schema is PURE POSTGRES — no AGE dependency. It can be executed
-- before or after age_schema.sql.
--
-- Tables:
--   rif_meta.repositories        — registered repos (clone URL, current SHA, version)
--   rif_meta.index_versions      — immutable ledger of completed index versions
--   rif_meta.index_runs          — one row per extraction invocation
--   rif_meta.provenance_failures — written by CI gate on source_ref violations
--   rif_meta.file_nodes          — shadow table for File vertices (Phase 2 embeddings)
--   rif_meta.method_nodes        — shadow table for Method vertices (Phase 2 embeddings)
-- =============================================================================

CREATE SCHEMA IF NOT EXISTS rif_meta;

-- ---------------------------------------------------------------------------
-- 1. repositories
-- One row per tracked repository.
-- repo_id is the stable identifier written into every node/edge source_ref.
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS rif_meta.repositories (
    repo_id               TEXT        NOT NULL,
    clone_url             TEXT        NOT NULL,
    -- 40-char SHA-1 of the commit currently reflected in the AGE graph.
    -- NULL until the first successful index run completes.
    current_sha           CHAR(40),
    -- Monotonically increasing; incremented by Ingestion Service on each
    -- successful run. Matches the latest row in index_versions.
    current_index_version INTEGER     NOT NULL DEFAULT 0,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT pk_repositories PRIMARY KEY (repo_id),
    CONSTRAINT chk_repo_id_no_special
        CHECK (repo_id ~ '^[A-Za-z0-9_\-]+$')   -- repo_id must be safe for source_ref
);

COMMENT ON TABLE  rif_meta.repositories IS
    'One row per registered repository. repo_id is the stable identifier embedded '
    'in every node/edge source_ref value (format: repo_id@sha:path:line).';
COMMENT ON COLUMN rif_meta.repositories.current_sha IS
    '40-char SHA-1 of the commit currently indexed in the AGE graph. NULL until first successful run.';
COMMENT ON COLUMN rif_meta.repositories.current_index_version IS
    'Monotonically increasing counter; incremented on each successful ingestion run.';

-- ---------------------------------------------------------------------------
-- 2. index_versions
-- Immutable ledger — one row per successfully completed index version.
-- Acts as the authoritative "what is currently in the graph" audit trail.
-- The Ingestion Service inserts here only after the AGE graph load is confirmed
-- and the CI provenance gate has passed.
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS rif_meta.index_versions (
    repo_id    TEXT        NOT NULL,
    version    INTEGER     NOT NULL,
    sha        CHAR(40)    NOT NULL,
    -- extractor_version allows tracing which extractor binary produced this version
    extractor_version TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT pk_index_versions       PRIMARY KEY (repo_id, version),
    CONSTRAINT fk_index_versions_repo  FOREIGN KEY (repo_id)
        REFERENCES rif_meta.repositories(repo_id) ON DELETE CASCADE
);

COMMENT ON TABLE rif_meta.index_versions IS
    'Immutable ledger of every successfully completed index version per repo. '
    'Phase 2 adds SCIP-derived versions alongside Phase 1 AST versions.';

-- ---------------------------------------------------------------------------
-- 3. index_runs
-- One row per extraction invocation (including failed and cancelled runs).
-- run_metrics stores all per-run counters defined in CODE_MODEL.md §5.6.
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS rif_meta.index_runs (
    run_id            UUID        NOT NULL DEFAULT gen_random_uuid(),
    repo_id           TEXT        NOT NULL,
    sha               CHAR(40)    NOT NULL,
    index_version     INTEGER     NOT NULL,
    extractor_version TEXT        NOT NULL,
    started_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at      TIMESTAMPTZ,
    status            TEXT        NOT NULL DEFAULT 'running',
    -- Counts from the extractor output manifest
    node_count        INTEGER,
    edge_count        INTEGER,
    -- JSONB bag of all per-run counters from the extractor.
    -- Expected keys (CODE_MODEL.md §5.6):
    --   same_file_resolution_failure_count  INTEGER
    --   unresolved_param_type_count         INTEGER
    --   provenance_gap_count                INTEGER
    --   unsupported_construct_count         INTEGER
    --   stub_node_count                     INTEGER
    --   total_files_parsed                  INTEGER
    run_metrics       JSONB,
    error_message     TEXT,
    CONSTRAINT pk_index_runs    PRIMARY KEY (run_id),
    CONSTRAINT fk_run_repo      FOREIGN KEY (repo_id)
        REFERENCES rif_meta.repositories(repo_id),
    CONSTRAINT chk_run_status   CHECK (
        status IN ('running', 'completed', 'failed', 'cancelled')
    ),
    CONSTRAINT chk_completed_at CHECK (
        status IN ('running', 'cancelled') OR completed_at IS NOT NULL
    )
);

CREATE INDEX IF NOT EXISTS idx_index_runs_repo_status
    ON rif_meta.index_runs (repo_id, status);
CREATE INDEX IF NOT EXISTS idx_index_runs_started_at
    ON rif_meta.index_runs (started_at DESC);

COMMENT ON TABLE  rif_meta.index_runs IS
    'One row per extraction invocation. Includes failed and cancelled runs. '
    'The CI provenance gate updates status to ''completed'' or ''failed'' and '
    'writes the final node_count / edge_count / run_metrics here.';
COMMENT ON COLUMN rif_meta.index_runs.run_metrics IS
    'JSONB map of extractor counters from CODE_MODEL.md §5.6: '
    'same_file_resolution_failure_count, unresolved_param_type_count, '
    'provenance_gap_count, unsupported_construct_count, stub_node_count, '
    'total_files_parsed.';

-- ---------------------------------------------------------------------------
-- 4. provenance_failures
-- Written by the CI provenance gate (Step 1.8) whenever a node or edge from
-- a first-party source file (origin=first_party, provenance_kind=file) is
-- found with source_ref = UNAVAILABLE:... or a source_ref that fails the
-- repo@sha:path:line format check.
--
-- The CI gate asserts: SELECT COUNT(*) = 0 FROM rif_meta.provenance_failures
--                      WHERE run_id = $run_id;
-- Any non-zero count fails the gate.
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS rif_meta.provenance_failures (
    failure_id     BIGSERIAL   NOT NULL,
    run_id         UUID        NOT NULL,
    repo_id        TEXT        NOT NULL,
    -- 'node' or 'edge'
    entity_type    TEXT        NOT NULL,
    -- SHA-256 hex: node_id for vertices, edge_id for edges
    entity_id      TEXT        NOT NULL,
    -- AGE vertex or edge label (e.g. 'Method', 'IMPORTS')
    label          TEXT        NOT NULL,
    -- Human-readable identifier for the failing entity (for log readability)
    qualified_name TEXT,
    -- The actual invalid source_ref value that triggered the failure
    source_ref     TEXT,
    -- Machine-readable failure code (e.g. 'UNAVAILABLE', 'FORMAT_MISMATCH',
    -- 'MISSING_SOURCE_REF', 'STUB_IN_FIRST_PARTY')
    failure_reason TEXT        NOT NULL,
    detected_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT pk_provenance_failures PRIMARY KEY (failure_id),
    CONSTRAINT fk_pf_run              FOREIGN KEY (run_id)
        REFERENCES rif_meta.index_runs(run_id),
    CONSTRAINT chk_pf_entity_type     CHECK (entity_type IN ('node', 'edge'))
);

CREATE INDEX IF NOT EXISTS idx_pf_run_id
    ON rif_meta.provenance_failures (run_id);
CREATE INDEX IF NOT EXISTS idx_pf_repo_id
    ON rif_meta.provenance_failures (repo_id);

COMMENT ON TABLE  rif_meta.provenance_failures IS
    'Written by the CI provenance gate when a first-party node/edge has an '
    'invalid or missing source_ref. Gate assertion: zero rows for current run_id.';
COMMENT ON COLUMN rif_meta.provenance_failures.failure_reason IS
    'Machine-readable code. Values: UNAVAILABLE (source_ref starts with '
    '''UNAVAILABLE:''), FORMAT_MISMATCH (does not match repo@sha:path:line), '
    'MISSING_SOURCE_REF (null or empty), STUB_IN_FIRST_PARTY (stub marker '
    'found on a first_party origin node).';

-- ---------------------------------------------------------------------------
-- 5. Shadow Tables
-- ---------------------------------------------------------------------------
-- Relational mirrors of selected AGE vertex properties for File and Method.
-- These tables exist to support Phase 2 vector similarity search (pgvector).
-- AGE does not natively support vector types or ANN index scans; Phase 2
-- similarity queries run in SQL against these tables, then join back to the
-- AGE graph via node_id.
--
-- Phase 1: the embedding column is commented out below.
-- Phase 2 upgrade steps (non-destructive — no column removal or rename):
--   1. CREATE EXTENSION IF NOT EXISTS vector;
--   2. ALTER TABLE rif_meta.file_nodes   ADD COLUMN IF NOT EXISTS embedding vector(768);
--   3. ALTER TABLE rif_meta.method_nodes ADD COLUMN IF NOT EXISTS embedding vector(768);
--   4. CREATE INDEX ON rif_meta.file_nodes   USING ivfflat (embedding vector_cosine_ops)
--        WITH (lists = 100);
--   5. CREATE INDEX ON rif_meta.method_nodes USING ivfflat (embedding vector_cosine_ops)
--        WITH (lists = 100);
--   The Ingestion Service populates embeddings via the self-hosted
--   jina-code-embeddings-1.5b model (confirmed in phase-0/FINDINGS_MEMO.md).
-- ---------------------------------------------------------------------------

-- --- File shadow table ---

CREATE TABLE IF NOT EXISTS rif_meta.file_nodes (
    node_id        TEXT        NOT NULL,
    repo_id        TEXT        NOT NULL,
    -- Repo-relative path (no leading slash), e.g.:
    --   src/main/java/com/example/creditcheck/routing/v1/CCRoutingService.java
    qualified_name TEXT        NOT NULL,
    -- Java package declaration; NULL for default package
    package        TEXT,
    line_count     INTEGER,
    source_ref     TEXT        NOT NULL,
    index_version  INTEGER     NOT NULL,
    -- 'first_party' | 'external_stub' — matches CODE_MODEL.md §1.1
    origin         TEXT        NOT NULL DEFAULT 'first_party',
    upserted_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Phase 2: ADD COLUMN embedding vector(768)
    --   Populated by the embedding worker using jina-code-embeddings-1.5b.
    --   After adding: CREATE INDEX ON rif_meta.file_nodes
    --     USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
    CONSTRAINT pk_file_nodes    PRIMARY KEY (node_id),
    CONSTRAINT fk_fn_repo       FOREIGN KEY (repo_id)
        REFERENCES rif_meta.repositories(repo_id),
    CONSTRAINT chk_fn_origin    CHECK (origin IN ('first_party', 'external_stub'))
);

CREATE INDEX IF NOT EXISTS idx_file_nodes_repo_id
    ON rif_meta.file_nodes (repo_id);
CREATE INDEX IF NOT EXISTS idx_file_nodes_package
    ON rif_meta.file_nodes (repo_id, package) WHERE package IS NOT NULL;

COMMENT ON TABLE rif_meta.file_nodes IS
    'Relational shadow of AGE File vertices. Kept in sync by the Ingestion '
    'Service. Phase 2 adds embedding vector(768) for semantic file search '
    '(jina-code-embeddings-1.5b; see FINDINGS_MEMO.md §4).';

-- --- Method shadow table ---

CREATE TABLE IF NOT EXISTS rif_meta.method_nodes (
    node_id        TEXT        NOT NULL,
    repo_id        TEXT        NOT NULL,
    -- Fully-qualified with erased parameter types, e.g.:
    --   com.example...CCRoutingService#routeToCCApi(com.example...CreditCheckRequest)
    qualified_name TEXT        NOT NULL,
    simple_name    TEXT        NOT NULL,
    return_type    TEXT,
    visibility     TEXT,
    is_static      BOOLEAN,
    source_ref     TEXT        NOT NULL,
    index_version  INTEGER     NOT NULL,
    origin         TEXT        NOT NULL DEFAULT 'first_party',
    upserted_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Phase 2: ADD COLUMN embedding vector(768)
    --   Populated by the embedding worker. Used for semantic method search
    --   and impact-analysis retrieval in hybrid queries (graph + vector).
    --   After adding: CREATE INDEX ON rif_meta.method_nodes
    --     USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
    --   Note: install pgvector first — CREATE EXTENSION IF NOT EXISTS vector;
    CONSTRAINT pk_method_nodes  PRIMARY KEY (node_id),
    CONSTRAINT fk_mn_repo       FOREIGN KEY (repo_id)
        REFERENCES rif_meta.repositories(repo_id),
    CONSTRAINT chk_mn_origin    CHECK (origin IN ('first_party', 'external_stub')),
    CONSTRAINT chk_mn_visibility CHECK (
        visibility IS NULL OR visibility IN ('public','protected','package','private')
    )
);

CREATE INDEX IF NOT EXISTS idx_method_nodes_repo_id
    ON rif_meta.method_nodes (repo_id);
CREATE INDEX IF NOT EXISTS idx_method_nodes_simple_name
    ON rif_meta.method_nodes (repo_id, simple_name);

COMMENT ON TABLE rif_meta.method_nodes IS
    'Relational shadow of AGE Method vertices. Kept in sync by the Ingestion '
    'Service. Phase 2 adds embedding vector(768) for semantic method search '
    'and hybrid impact-analysis retrieval (graph traversal + vector similarity).';

-- ---------------------------------------------------------------------------
-- End of relational_schema.sql
-- ---------------------------------------------------------------------------
