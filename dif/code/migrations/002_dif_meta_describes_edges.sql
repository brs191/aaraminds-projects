-- 002_dif_meta_describes_edges.sql
--
-- P1-02: enable DESCRIBES doc->code edges in dif_meta.edges (ADR-016 §9).
--
-- Additive to 001_dif_meta_initial.sql. Creates and alters dif_meta objects
-- only; never touches RIF-owned schemas (rif, rif_meta). Idempotent: safe to
-- run twice (ADD COLUMN IF NOT EXISTS; constraints are dropped and re-added
-- to the same definition).
--
-- DESCRIBES edges reference RIF code nodes via to_external_node_id with
-- external_system = 'rif'. No foreign key is created against RIF-owned
-- tables: RIF shadows may be empty or absent (ADR-016), and DIF must not
-- depend on RIF schema shape at the database level.

-- ADR-016 minimum fields for DESCRIBES edges that 001 did not carry.
ALTER TABLE dif_meta.edges
    ADD COLUMN IF NOT EXISTS repo_id TEXT;

-- DESCRIBES edges exist only with resolver evidence; if the evidencing
-- candidate row is removed, the edge is removed with it (CASCADE) instead of
-- violating edges_describes_shape_check via SET NULL.
ALTER TABLE dif_meta.edges
    ADD COLUMN IF NOT EXISTS candidate_id TEXT
        REFERENCES dif_meta.code_entity_candidates (candidate_id) ON DELETE CASCADE;

ALTER TABLE dif_meta.edges
    ADD COLUMN IF NOT EXISTS match_mode TEXT;

ALTER TABLE dif_meta.edges
    ADD COLUMN IF NOT EXISTS code_source_ref TEXT;

-- Extend the edge-kind vocabulary: P0 CONTAINS plus P1 DESCRIBES.
ALTER TABLE dif_meta.edges
    DROP CONSTRAINT IF EXISTS edges_edge_kind_check;

ALTER TABLE dif_meta.edges
    ADD CONSTRAINT edges_edge_kind_check
        CHECK (edge_kind IN ('CONTAINS', 'DESCRIBES'));

ALTER TABLE dif_meta.edges
    DROP CONSTRAINT IF EXISTS edges_match_mode_check;

ALTER TABLE dif_meta.edges
    ADD CONSTRAINT edges_match_mode_check
        CHECK (
            match_mode IS NULL
            OR match_mode IN ('qualified-name', 'source-path', 'node-id', 'simple-name', 'fuzzy')
        );

-- DESCRIBES shape: doc node -> external RIF node, resolver evidence required.
ALTER TABLE dif_meta.edges
    DROP CONSTRAINT IF EXISTS edges_describes_shape_check;

ALTER TABLE dif_meta.edges
    ADD CONSTRAINT edges_describes_shape_check
        CHECK (
            edge_kind <> 'DESCRIBES'
            OR (
                to_node_id IS NULL
                AND to_external_node_id IS NOT NULL
                AND external_system = 'rif'
                AND repo_id IS NOT NULL
                AND match_mode IS NOT NULL
                AND anchor_id IS NOT NULL
                AND candidate_id IS NOT NULL
            )
        );

CREATE INDEX IF NOT EXISTS idx_edges_candidate_id
    ON dif_meta.edges (candidate_id);

CREATE INDEX IF NOT EXISTS idx_edges_repo_external_node
    ON dif_meta.edges (repo_id, to_external_node_id);
