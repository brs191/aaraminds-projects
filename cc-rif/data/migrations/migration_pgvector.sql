-- =============================================================================
-- RIF Phase 2 — pgvector Migration
-- =============================================================================
-- Adds vector embedding columns to the rif_meta shadow tables (file_nodes,
-- method_nodes) and installs HNSW indexes for cosine ANN search.
--
-- Prerequisites:
--   1. Phase 1 schema applied (relational_schema.sql)
--   2. pgvector extension available on the server:
--        SELECT * FROM pg_available_extensions WHERE name = 'vector';
--      On Azure Postgres Flexible Server: enable via Extensions blade before
--      running this migration.
--
-- Run:
--   psql $DATABASE_URL -f phase-2/schema/migration_pgvector.sql
--
-- Idempotent: safe to run multiple times (all statements use IF NOT EXISTS
-- or DO $$ … END guards).
--
-- Phase 1 schema: UNCHANGED — no DROP, no ALTER COLUMN, no RENAME.
-- =============================================================================

-- ---------------------------------------------------------------------------
-- 1. Install pgvector extension
-- ---------------------------------------------------------------------------

CREATE EXTENSION IF NOT EXISTS vector;

-- ---------------------------------------------------------------------------
-- 2. file_nodes — add embedding + embedding_model columns
-- ---------------------------------------------------------------------------

ALTER TABLE rif_meta.file_nodes
    ADD COLUMN IF NOT EXISTS embedding       vector(768),
    ADD COLUMN IF NOT EXISTS embedding_model TEXT;

COMMENT ON COLUMN rif_meta.file_nodes.embedding IS
    'Phase 2 — 768-dim code embedding from text-embedding-3-small. '
    'NULL until the batch_embed.py worker processes this node. '
    'Used for semantic file-level similarity search (cosine ANN via HNSW).';

COMMENT ON COLUMN rif_meta.file_nodes.embedding_model IS
    'Model identifier used to produce the embedding, '
    'e.g. ''text-embedding-3-small''. '
    'NULL until embedding is populated. Allows detecting stale embeddings '
    'after a model version change.';

-- ---------------------------------------------------------------------------
-- 3. method_nodes — add embedding + embedding_model columns
-- ---------------------------------------------------------------------------

ALTER TABLE rif_meta.method_nodes
    ADD COLUMN IF NOT EXISTS embedding       vector(768),
    ADD COLUMN IF NOT EXISTS embedding_model TEXT;

COMMENT ON COLUMN rif_meta.method_nodes.embedding IS
    'Phase 2 — 768-dim code embedding from text-embedding-3-small. '
    'NULL until the batch_embed.py worker processes this node. '
    'Core retrieval signal for hybrid impact-analysis (graph traversal + vector similarity).';

COMMENT ON COLUMN rif_meta.method_nodes.embedding_model IS
    'Model identifier used to produce the embedding. NULL until populated.';

-- ---------------------------------------------------------------------------
-- 4. HNSW indexes — cosine distance, m=16, ef_construction=64
--    HNSW chosen over IVFFlat because it does not require a training step
--    (no VACUUM ANALYZE needed before first query) and gives better recall
--    at low ef_construction values for this dataset size (~6,600 methods).
--
--    Index creation is skipped if already present (DO $$ guard).
-- ---------------------------------------------------------------------------

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE schemaname = 'rif_meta'
          AND tablename  = 'file_nodes'
          AND indexname  = 'idx_file_nodes_embedding_hnsw'
    ) THEN
        CREATE INDEX idx_file_nodes_embedding_hnsw
            ON rif_meta.file_nodes
            USING hnsw (embedding vector_cosine_ops)
            WITH (m = 16, ef_construction = 64);
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE schemaname = 'rif_meta'
          AND tablename  = 'method_nodes'
          AND indexname  = 'idx_method_nodes_embedding_hnsw'
    ) THEN
        CREATE INDEX idx_method_nodes_embedding_hnsw
            ON rif_meta.method_nodes
            USING hnsw (embedding vector_cosine_ops)
            WITH (m = 16, ef_construction = 64);
    END IF;
END$$;

-- ---------------------------------------------------------------------------
-- 5. Backfill query — methods with no embedding yet
--    Used by batch_embed.py to enumerate work. Returns method nodes ordered
--    by most-recent index run first (prioritises freshly indexed repos).
-- ---------------------------------------------------------------------------

-- NOTE: This is a reference query, not executed here.
-- batch_embed.py runs this query to find pending work:
--
-- SELECT
--     mn.node_id,
--     mn.repo_id,
--     mn.qualified_name,
--     mn.simple_name,
--     mn.source_ref,
--     ir.run_id,
--     ir.started_at AS index_run_started_at
-- FROM rif_meta.method_nodes mn
-- JOIN rif_meta.index_runs ir
--   ON ir.repo_id = mn.repo_id
--  AND ir.status  = 'completed'
-- WHERE mn.embedding IS NULL
--   AND mn.origin    = 'first_party'
-- ORDER BY ir.started_at DESC
-- LIMIT :batch_size;

-- ---------------------------------------------------------------------------
-- 6. Verification
-- ---------------------------------------------------------------------------

DO $$
DECLARE
    v_ext_present   INTEGER;
    v_fn_col        INTEGER;
    v_mn_col        INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_ext_present
    FROM pg_extension WHERE extname = 'vector';

    SELECT COUNT(*) INTO v_fn_col
    FROM information_schema.columns
    WHERE table_schema = 'rif_meta'
      AND table_name   = 'file_nodes'
      AND column_name  = 'embedding';

    SELECT COUNT(*) INTO v_mn_col
    FROM information_schema.columns
    WHERE table_schema = 'rif_meta'
      AND table_name   = 'method_nodes'
      AND column_name  = 'embedding';

    IF v_ext_present = 0 THEN
        RAISE EXCEPTION 'pgvector extension not installed';
    END IF;
    IF v_fn_col = 0 THEN
        RAISE EXCEPTION 'file_nodes.embedding column missing';
    END IF;
    IF v_mn_col = 0 THEN
        RAISE EXCEPTION 'method_nodes.embedding column missing';
    END IF;

    RAISE NOTICE 'migration_pgvector.sql: OK — vector extension present, embedding columns on file_nodes and method_nodes confirmed';
END$$;

-- =============================================================================
-- End of migration_pgvector.sql
-- =============================================================================
