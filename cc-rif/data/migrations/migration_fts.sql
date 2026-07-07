-- =============================================================================
-- RIF Phase 2 — Full-Text Search (FTS) Migration
-- =============================================================================
-- Adds tsvector columns and auto-update triggers to the rif_meta shadow
-- tables for File, Method, and Class-family nodes (class_nodes).
--
-- Note: rif_meta.class_nodes is added by this migration if not already
-- present (Phase 1 only has file_nodes and method_nodes shadow tables).
-- class_nodes mirrors AGE Class vertices for FTS and is used by the
-- hybrid retriever in Phase 3.
--
-- Prerequisites:
--   1. Phase 1 schema applied (relational_schema.sql)
--   2. migration_pgvector.sql applied (for consistency — not a hard dep)
--
-- Run:
--   psql $DATABASE_URL -f phase-2/schema/migration_fts.sql
--
-- Idempotent: safe to run multiple times.
-- Phase 1 schema: UNCHANGED — additive only.
-- =============================================================================

-- ---------------------------------------------------------------------------
-- 1. Create class_nodes shadow table (Phase 2 addition)
--    Mirrors AGE Class/Interface/Enum/Record vertices for FTS and Phase 3
--    hybrid retrieval. Phase 1 did not need a class shadow table because
--    graph traversal queries went directly through AGE.
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS rif_meta.class_nodes (
    node_id        TEXT        NOT NULL,
    repo_id        TEXT        NOT NULL,
    qualified_name TEXT        NOT NULL,
    simple_name    TEXT        NOT NULL,
    -- 'CLASS' | 'INTERFACE' | 'ENUM' | 'RECORD'
    kind           TEXT        NOT NULL DEFAULT 'CLASS',
    -- Null until populated by Phase 2 embedding worker
    summary        TEXT,
    source_ref     TEXT        NOT NULL,
    index_version  INTEGER     NOT NULL,
    origin         TEXT        NOT NULL DEFAULT 'first_party',
    upserted_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Phase 2 FTS
    fts_vector     TSVECTOR,
    -- Phase 2 vector embedding (same pattern as file_nodes/method_nodes)
    embedding       vector(768),
    embedding_model TEXT,
    CONSTRAINT pk_class_nodes   PRIMARY KEY (node_id),
    CONSTRAINT fk_cn_repo       FOREIGN KEY (repo_id)
        REFERENCES rif_meta.repositories(repo_id),
    CONSTRAINT chk_cn_origin    CHECK (origin IN ('first_party', 'external_stub')),
    CONSTRAINT chk_cn_kind      CHECK (kind IN ('CLASS', 'INTERFACE', 'ENUM', 'RECORD'))
);

CREATE INDEX IF NOT EXISTS idx_class_nodes_repo_id
    ON rif_meta.class_nodes (repo_id);
CREATE INDEX IF NOT EXISTS idx_class_nodes_simple_name
    ON rif_meta.class_nodes (repo_id, simple_name);

COMMENT ON TABLE rif_meta.class_nodes IS
    'Phase 2 — relational shadow of AGE Class/Interface/Enum/Record vertices. '
    'Populated by the Ingestion Service alongside file_nodes and method_nodes. '
    'Used for FTS (fts_vector) and Phase 3 hybrid retrieval.';

-- ---------------------------------------------------------------------------
-- 2. Add fts_vector + summary columns to existing shadow tables
-- ---------------------------------------------------------------------------

-- file_nodes
ALTER TABLE rif_meta.file_nodes
    ADD COLUMN IF NOT EXISTS fts_vector TSVECTOR,
    ADD COLUMN IF NOT EXISTS summary    TEXT;

COMMENT ON COLUMN rif_meta.file_nodes.fts_vector IS
    'Phase 2 — auto-maintained tsvector for full-text search. '
    'Derived from: qualified_name (path) + summary. '
    'Updated by trigger on insert/update.';

COMMENT ON COLUMN rif_meta.file_nodes.summary IS
    'Phase 2 — LLM-generated one-sentence summary of the file contents. '
    'NULL until populated by Phase 4 agent. Included in fts_vector when present.';

-- method_nodes
ALTER TABLE rif_meta.method_nodes
    ADD COLUMN IF NOT EXISTS fts_vector TSVECTOR,
    ADD COLUMN IF NOT EXISTS summary    TEXT;

COMMENT ON COLUMN rif_meta.method_nodes.fts_vector IS
    'Phase 2 — auto-maintained tsvector. Derived from: simple_name + qualified_name + summary.';

COMMENT ON COLUMN rif_meta.method_nodes.summary IS
    'Phase 2 — LLM-generated one-sentence summary of the method. NULL until Phase 4.';

-- ---------------------------------------------------------------------------
-- 3. GIN indexes on fts_vector
-- ---------------------------------------------------------------------------

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE schemaname = 'rif_meta'
          AND tablename  = 'file_nodes'
          AND indexname  = 'idx_file_nodes_fts'
    ) THEN
        CREATE INDEX idx_file_nodes_fts
            ON rif_meta.file_nodes USING gin(fts_vector);
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE schemaname = 'rif_meta'
          AND tablename  = 'method_nodes'
          AND indexname  = 'idx_method_nodes_fts'
    ) THEN
        CREATE INDEX idx_method_nodes_fts
            ON rif_meta.method_nodes USING gin(fts_vector);
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE schemaname = 'rif_meta'
          AND tablename  = 'class_nodes'
          AND indexname  = 'idx_class_nodes_fts'
    ) THEN
        CREATE INDEX idx_class_nodes_fts
            ON rif_meta.class_nodes USING gin(fts_vector);
    END IF;
END$$;

-- ---------------------------------------------------------------------------
-- 4. Trigger functions — auto-update fts_vector on INSERT or UPDATE
--    Uses 'english' dictionary. Weights:
--      simple_name / path tail — 'A' (highest)
--      qualified_name          — 'B'
--      summary                 — 'C'
-- ---------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION rif_meta.file_nodes_fts_update()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.fts_vector :=
        setweight(to_tsvector('english',
            coalesce(
                -- Use just the filename (last path segment) for highest weight
                reverse(split_part(reverse(coalesce(NEW.qualified_name, '')), '/', 1)),
                ''
            )
        ), 'A') ||
        setweight(to_tsvector('english',
            coalesce(NEW.qualified_name, '')
        ), 'B') ||
        setweight(to_tsvector('english',
            coalesce(NEW.summary, '')
        ), 'C');
    RETURN NEW;
END$$;

CREATE OR REPLACE FUNCTION rif_meta.method_nodes_fts_update()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.fts_vector :=
        setweight(to_tsvector('english',
            coalesce(NEW.simple_name, '')
        ), 'A') ||
        setweight(to_tsvector('english',
            coalesce(NEW.qualified_name, '')
        ), 'B') ||
        setweight(to_tsvector('english',
            coalesce(NEW.summary, '')
        ), 'C');
    RETURN NEW;
END$$;

CREATE OR REPLACE FUNCTION rif_meta.class_nodes_fts_update()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.fts_vector :=
        setweight(to_tsvector('english',
            coalesce(NEW.simple_name, '')
        ), 'A') ||
        setweight(to_tsvector('english',
            coalesce(NEW.qualified_name, '')
        ), 'B') ||
        setweight(to_tsvector('english',
            coalesce(NEW.summary, '')
        ), 'C');
    RETURN NEW;
END$$;

-- ---------------------------------------------------------------------------
-- 5. Attach triggers (idempotent via DROP IF EXISTS + CREATE)
-- ---------------------------------------------------------------------------

DROP TRIGGER IF EXISTS trg_file_nodes_fts   ON rif_meta.file_nodes;
DROP TRIGGER IF EXISTS trg_method_nodes_fts ON rif_meta.method_nodes;
DROP TRIGGER IF EXISTS trg_class_nodes_fts  ON rif_meta.class_nodes;

CREATE TRIGGER trg_file_nodes_fts
    BEFORE INSERT OR UPDATE ON rif_meta.file_nodes
    FOR EACH ROW EXECUTE FUNCTION rif_meta.file_nodes_fts_update();

CREATE TRIGGER trg_method_nodes_fts
    BEFORE INSERT OR UPDATE ON rif_meta.method_nodes
    FOR EACH ROW EXECUTE FUNCTION rif_meta.method_nodes_fts_update();

CREATE TRIGGER trg_class_nodes_fts
    BEFORE INSERT OR UPDATE ON rif_meta.class_nodes
    FOR EACH ROW EXECUTE FUNCTION rif_meta.class_nodes_fts_update();

-- ---------------------------------------------------------------------------
-- 6. Backfill existing rows (re-compute fts_vector for rows inserted before
--    the trigger existed — safe to run multiple times)
-- ---------------------------------------------------------------------------

UPDATE rif_meta.file_nodes SET upserted_at = upserted_at
WHERE fts_vector IS NULL;

UPDATE rif_meta.method_nodes SET upserted_at = upserted_at
WHERE fts_vector IS NULL;

-- class_nodes is new in Phase 2, no backfill needed.

-- ---------------------------------------------------------------------------
-- 7. Sample FTS query (reference — not executed here)
--    Phase 3 retriever uses ts_rank_cd for scored search:
--
-- SELECT
--     node_id,
--     repo_id,
--     qualified_name,
--     source_ref,
--     ts_rank_cd(fts_vector, query) AS rank
-- FROM rif_meta.method_nodes,
--      websearch_to_tsquery('english', :search_terms) query
-- WHERE fts_vector @@ query
--   AND repo_id = :repo_id
-- ORDER BY rank DESC
-- LIMIT :top_k;
-- ---------------------------------------------------------------------------

-- ---------------------------------------------------------------------------
-- 8. Verification
-- ---------------------------------------------------------------------------

DO $$
DECLARE
    v_fn_fts  INTEGER;
    v_mn_fts  INTEGER;
    v_cn_tbl  INTEGER;
    v_fn_trg  INTEGER;
    v_mn_trg  INTEGER;
    v_cn_trg  INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_fn_fts
    FROM information_schema.columns
    WHERE table_schema = 'rif_meta' AND table_name = 'file_nodes'
      AND column_name = 'fts_vector';

    SELECT COUNT(*) INTO v_mn_fts
    FROM information_schema.columns
    WHERE table_schema = 'rif_meta' AND table_name = 'method_nodes'
      AND column_name = 'fts_vector';

    SELECT COUNT(*) INTO v_cn_tbl
    FROM information_schema.tables
    WHERE table_schema = 'rif_meta' AND table_name = 'class_nodes';

    SELECT COUNT(*) INTO v_fn_trg
    FROM information_schema.triggers
    WHERE trigger_schema = 'rif_meta' AND event_object_table = 'file_nodes'
      AND trigger_name = 'trg_file_nodes_fts';

    SELECT COUNT(*) INTO v_mn_trg
    FROM information_schema.triggers
    WHERE trigger_schema = 'rif_meta' AND event_object_table = 'method_nodes'
      AND trigger_name = 'trg_method_nodes_fts';

    SELECT COUNT(*) INTO v_cn_trg
    FROM information_schema.triggers
    WHERE trigger_schema = 'rif_meta' AND event_object_table = 'class_nodes'
      AND trigger_name = 'trg_class_nodes_fts';

    IF v_fn_fts = 0 THEN RAISE EXCEPTION 'file_nodes.fts_vector missing'; END IF;
    IF v_mn_fts = 0 THEN RAISE EXCEPTION 'method_nodes.fts_vector missing'; END IF;
    IF v_cn_tbl = 0 THEN RAISE EXCEPTION 'class_nodes table missing'; END IF;
    IF v_fn_trg = 0 THEN RAISE EXCEPTION 'trigger trg_file_nodes_fts missing'; END IF;
    IF v_mn_trg = 0 THEN RAISE EXCEPTION 'trigger trg_method_nodes_fts missing'; END IF;
    IF v_cn_trg = 0 THEN RAISE EXCEPTION 'trigger trg_class_nodes_fts missing'; END IF;

    RAISE NOTICE 'migration_fts.sql: OK — fts_vector columns, GIN indexes, and triggers confirmed on file_nodes, method_nodes, class_nodes';
END$$;

-- =============================================================================
-- End of migration_fts.sql
-- =============================================================================
