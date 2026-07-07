-- =============================================================================
-- RIF Phase 1 — Apache AGE Graph Schema
-- =============================================================================
-- Stack : Apache AGE 1.5.0 + Postgres 14.23 (local dev) / PG16 (Azure Flexible Server)
-- Run as: psql -U <superuser> -d <database> -f age_schema.sql
-- Reads with: phase-1/design/CODE_MODEL.md (node/edge types, properties, tiers)
-- Idempotent: safe to run multiple times against an existing database.
--
-- Execution order matters:
--   1. Extension load (requires superuser or pg_extension_owner role on Azure)
--   2. Graph creation
--   3. Vertex labels  (node types)
--   4. Edge labels    (edge types — Tier-A populated Phase 1; Tier-B/C stub tables only)
--   5. Vertex indexes (node_id, repo_id per label)
--   6. Edge indexes   (start_id+end_id traversal, end_id reverse, edge_id dedup)
-- =============================================================================

-- ---------------------------------------------------------------------------
-- 0. Extension and search_path
-- ---------------------------------------------------------------------------
-- On Azure Postgres Flexible Server, AGE must first be added to
-- shared_preload_libraries via the server parameter blade, then the database
-- restarted, before CREATE EXTENSION will succeed.
-- On local PG14 built from source: ensure the AGE .so is in $libdir.

CREATE EXTENSION IF NOT EXISTS age;
LOAD 'age';

-- All subsequent DDL and index expressions must resolve the agtype ->> operator
-- from ag_catalog. This SET persists for the session; the application must also
-- execute "LOAD 'age'; SET search_path = ag_catalog, rif_meta, public" on every
-- new connection (see SCHEMA.md §6 for pgx pool AfterConnect setup).
SET search_path = ag_catalog, "$user", public;

-- ---------------------------------------------------------------------------
-- 1. Graph
-- ---------------------------------------------------------------------------
-- AGE 1.5.0 has no CREATE GRAPH IF NOT EXISTS — use the catalog check below.

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM ag_catalog.ag_graph WHERE name = 'rif'
    ) THEN
        PERFORM ag_catalog.create_graph('rif');
        RAISE NOTICE 'Graph "rif" created.';
    ELSE
        RAISE NOTICE 'Graph "rif" already exists — skipping create_graph.';
    END IF;
END;
$$;

-- ---------------------------------------------------------------------------
-- 2. Vertex Labels
-- ---------------------------------------------------------------------------
-- One label per node kind from CODE_MODEL.md §1.2.
-- Convention: PascalCase to match CODE_MODEL kind literals.
-- Properties are stored as agtype JSON on each vertex; no DDL column-per-property
-- is required — AGE is schema-optional for vertex properties.
-- All properties documented in CODE_MODEL.md are enforced by the extractor,
-- not by AGE DDL constraints.
--
-- Compatibility note:
-- Some AGE builds expose create_vlabel/create_elabel with cstring arguments.
-- We cast explicitly to cstring for broader compatibility across PG14/PG16
-- environments.

DO $$
DECLARE
    graph_oid  oid;
    labels     TEXT[] := ARRAY[
        'File',         -- §1.2 FILE     — one .java source file
        'Class',        -- §1.2 CLASS    — concrete/abstract class, inner, anonymous
        'Interface',    -- §1.2 INTERFACE
        'Enum',         -- §1.2 ENUM
        'Method',       -- §1.2 METHOD   — non-constructor method declaration
        'Constructor',  -- §1.2 CONSTRUCTOR
        'Field',        -- §1.2 FIELD    — one field variable declaration
        'Record'        -- §1.2 RECORD   — Java 16+ record type
    ];
    lbl TEXT;
BEGIN
    SELECT graphid INTO graph_oid FROM ag_catalog.ag_graph WHERE name = 'rif';

    FOREACH lbl IN ARRAY labels LOOP
        IF NOT EXISTS (
            SELECT 1 FROM ag_catalog.ag_label
            WHERE name = lbl
              AND graph = graph_oid
              AND kind = 'v'
        ) THEN
            PERFORM ag_catalog.create_vlabel('rif', lbl);
            RAISE NOTICE 'Vertex label "%" created.', lbl;
        ELSE
            RAISE NOTICE 'Vertex label "%" already exists — skipping.', lbl;
        END IF;
    END LOOP;
END;
$$;

-- ---------------------------------------------------------------------------
-- 3. Edge Labels
-- ---------------------------------------------------------------------------
-- Convention: UPPER_SNAKE_CASE to match CODE_MODEL.md edge type labels.
--
-- Tier-A  (Phase 1 — populated by JavaParser extractor):
--   IMPORTS, SAME_FILE_CALLS, EXTENDS, IMPLEMENTS, DECLARES_FIELD
--
-- Tier-B  (Phase 2 stub — table created now, zero rows until Phase 2):
--   INJECTS, PRODUCES
--
-- Tier-C  (Phase 2 stub — table created now, zero rows until Phase 2):
--   ADVISES, CALLS_SOAP, CALLS_REST
--
-- Creating stub edge label tables in Phase 1 ensures no destructive ALTER TABLE
-- is needed in Phase 2 when those edge types are first populated.

DO $$
DECLARE
    graph_oid  oid;
    labels     TEXT[] := ARRAY[
        -- Tier-A: exact, AST-derived (populated Phase 1)
        'IMPORTS',          -- §2.2  FILE → CLASS|INTERFACE|ENUM
        'SAME_FILE_CALLS',  -- §2.3  METHOD|CONSTRUCTOR → METHOD|CONSTRUCTOR
        'EXTENDS',          -- §2.4  CLASS → CLASS | INTERFACE → INTERFACE
        'IMPLEMENTS',       -- §2.5  CLASS → INTERFACE
        'DECLARES_FIELD',   -- §2.6  CLASS|INTERFACE|ENUM → FIELD
        -- Tier-B: probable, Spring annotation scan (Phase 2 stub)
        'INJECTS',          -- §3.1  RECEIVER_CLASS → FIELD|CONSTRUCTOR|METHOD
        'PRODUCES',         -- §3.1  METHOD → CLASS
        -- Tier-C: inferred, heuristic/cross-service (Phase 2 stub)
        'ADVISES',          -- §3.2  CLASS(aspect) → METHOD|CLASS
        'CALLS_SOAP',       -- §3.2  METHOD → CLASS(endpoint)
        'CALLS_REST'        -- §3.2  METHOD → CLASS(endpoint)
    ];
    lbl TEXT;
BEGIN
    SELECT graphid INTO graph_oid FROM ag_catalog.ag_graph WHERE name = 'rif';

    FOREACH lbl IN ARRAY labels LOOP
        IF NOT EXISTS (
            SELECT 1 FROM ag_catalog.ag_label
            WHERE name = lbl
              AND graph = graph_oid
              AND kind = 'e'
        ) THEN
            PERFORM ag_catalog.create_elabel('rif', lbl);
            RAISE NOTICE 'Edge label "%" created.', lbl;
        ELSE
            RAISE NOTICE 'Edge label "%" already exists — skipping.', lbl;
        END IF;
    END LOOP;
END;
$$;

-- ---------------------------------------------------------------------------
-- 4. Vertex Indexes
-- ---------------------------------------------------------------------------
-- Each vertex label gets two btree indexes:
--   node_id  — primary business-key lookup (SHA-256 content-addressed, 64 hex chars)
--   repo_id  — per-repository graph queries and bulk deletion / re-index
--
-- Method gets an additional index on simple_name for "find method by name" queries.
--
-- Expression syntax: ((properties -> '"key"'::agtype)::text) extracts TEXT from agtype using the
-- ->> operator registered in ag_catalog. The SET search_path above ensures it
-- resolves correctly when the index is created and when it is used.
--
-- PG14/PG16 compatibility: expression indexes on agtype are identical on both
-- versions since the operator is defined by AGE, not by Postgres core.

-- File
CREATE INDEX IF NOT EXISTS idx_rif_file_node_id
    ON rif."File" USING btree (((properties -> '"node_id"'::agtype)::text));
CREATE INDEX IF NOT EXISTS idx_rif_file_repo_id
    ON rif."File" USING btree (((properties -> '"repo_id"'::agtype)::text));

-- Class
CREATE INDEX IF NOT EXISTS idx_rif_class_node_id
    ON rif."Class" USING btree (((properties -> '"node_id"'::agtype)::text));
CREATE INDEX IF NOT EXISTS idx_rif_class_repo_id
    ON rif."Class" USING btree (((properties -> '"repo_id"'::agtype)::text));

-- Interface
CREATE INDEX IF NOT EXISTS idx_rif_interface_node_id
    ON rif."Interface" USING btree (((properties -> '"node_id"'::agtype)::text));
CREATE INDEX IF NOT EXISTS idx_rif_interface_repo_id
    ON rif."Interface" USING btree (((properties -> '"repo_id"'::agtype)::text));

-- Enum
CREATE INDEX IF NOT EXISTS idx_rif_enum_node_id
    ON rif."Enum" USING btree (((properties -> '"node_id"'::agtype)::text));
CREATE INDEX IF NOT EXISTS idx_rif_enum_repo_id
    ON rif."Enum" USING btree (((properties -> '"repo_id"'::agtype)::text));

-- Method — extra index on simple_name for "find all methods named X" queries
CREATE INDEX IF NOT EXISTS idx_rif_method_node_id
    ON rif."Method" USING btree (((properties -> '"node_id"'::agtype)::text));
CREATE INDEX IF NOT EXISTS idx_rif_method_repo_id
    ON rif."Method" USING btree (((properties -> '"repo_id"'::agtype)::text));
CREATE INDEX IF NOT EXISTS idx_rif_method_simple_name
    ON rif."Method" USING btree (((properties -> '"simple_name"'::agtype)::text));

-- Constructor
CREATE INDEX IF NOT EXISTS idx_rif_constructor_node_id
    ON rif."Constructor" USING btree (((properties -> '"node_id"'::agtype)::text));
CREATE INDEX IF NOT EXISTS idx_rif_constructor_repo_id
    ON rif."Constructor" USING btree (((properties -> '"repo_id"'::agtype)::text));

-- Field
CREATE INDEX IF NOT EXISTS idx_rif_field_node_id
    ON rif."Field" USING btree (((properties -> '"node_id"'::agtype)::text));
CREATE INDEX IF NOT EXISTS idx_rif_field_repo_id
    ON rif."Field" USING btree (((properties -> '"repo_id"'::agtype)::text));

-- Record
CREATE INDEX IF NOT EXISTS idx_rif_record_node_id
    ON rif."Record" USING btree (((properties -> '"node_id"'::agtype)::text));
CREATE INDEX IF NOT EXISTS idx_rif_record_repo_id
    ON rif."Record" USING btree (((properties -> '"repo_id"'::agtype)::text));

-- ---------------------------------------------------------------------------
-- 5. Edge Indexes
-- ---------------------------------------------------------------------------
-- Each edge label gets three indexes:
--   (start_id, end_id) — forward traversal (outgoing edges from a vertex)
--   (end_id)           — reverse traversal (incoming edges = blast-radius queries)
--   (edge_id)          — deduplication and idempotent upsert during extraction
--
-- AGE does NOT auto-create traversal indexes on edge labels; these are required
-- for the p50 < 500ms / p95 < 1500ms traversal gate (Phase 0 benchmark).
--
-- Naming: idx_rif_{label_lower}_{suffix}
--   Abbreviations: sfc = SAME_FILE_CALLS, df = DECLARES_FIELD

-- IMPORTS
CREATE INDEX IF NOT EXISTS idx_rif_imports_fwd
    ON rif."IMPORTS" USING btree (start_id, end_id);
CREATE INDEX IF NOT EXISTS idx_rif_imports_rev
    ON rif."IMPORTS" USING btree (end_id);
CREATE INDEX IF NOT EXISTS idx_rif_imports_edge_id
    ON rif."IMPORTS" USING btree (((properties -> '"edge_id"'::agtype)::text));

-- SAME_FILE_CALLS
CREATE INDEX IF NOT EXISTS idx_rif_sfc_fwd
    ON rif."SAME_FILE_CALLS" USING btree (start_id, end_id);
CREATE INDEX IF NOT EXISTS idx_rif_sfc_rev
    ON rif."SAME_FILE_CALLS" USING btree (end_id);
CREATE INDEX IF NOT EXISTS idx_rif_sfc_edge_id
    ON rif."SAME_FILE_CALLS" USING btree (((properties -> '"edge_id"'::agtype)::text));

-- EXTENDS
CREATE INDEX IF NOT EXISTS idx_rif_extends_fwd
    ON rif."EXTENDS" USING btree (start_id, end_id);
CREATE INDEX IF NOT EXISTS idx_rif_extends_rev
    ON rif."EXTENDS" USING btree (end_id);
CREATE INDEX IF NOT EXISTS idx_rif_extends_edge_id
    ON rif."EXTENDS" USING btree (((properties -> '"edge_id"'::agtype)::text));

-- IMPLEMENTS
CREATE INDEX IF NOT EXISTS idx_rif_implements_fwd
    ON rif."IMPLEMENTS" USING btree (start_id, end_id);
CREATE INDEX IF NOT EXISTS idx_rif_implements_rev
    ON rif."IMPLEMENTS" USING btree (end_id);
CREATE INDEX IF NOT EXISTS idx_rif_implements_edge_id
    ON rif."IMPLEMENTS" USING btree (((properties -> '"edge_id"'::agtype)::text));

-- DECLARES_FIELD
CREATE INDEX IF NOT EXISTS idx_rif_df_fwd
    ON rif."DECLARES_FIELD" USING btree (start_id, end_id);
CREATE INDEX IF NOT EXISTS idx_rif_df_rev
    ON rif."DECLARES_FIELD" USING btree (end_id);
CREATE INDEX IF NOT EXISTS idx_rif_df_edge_id
    ON rif."DECLARES_FIELD" USING btree (((properties -> '"edge_id"'::agtype)::text));

-- INJECTS (Phase 2 stub — indexes created now for zero-migration Phase 2 load)
CREATE INDEX IF NOT EXISTS idx_rif_injects_fwd
    ON rif."INJECTS" USING btree (start_id, end_id);
CREATE INDEX IF NOT EXISTS idx_rif_injects_rev
    ON rif."INJECTS" USING btree (end_id);
CREATE INDEX IF NOT EXISTS idx_rif_injects_edge_id
    ON rif."INJECTS" USING btree (((properties -> '"edge_id"'::agtype)::text));

-- PRODUCES (Phase 2 stub)
CREATE INDEX IF NOT EXISTS idx_rif_produces_fwd
    ON rif."PRODUCES" USING btree (start_id, end_id);
CREATE INDEX IF NOT EXISTS idx_rif_produces_rev
    ON rif."PRODUCES" USING btree (end_id);
CREATE INDEX IF NOT EXISTS idx_rif_produces_edge_id
    ON rif."PRODUCES" USING btree (((properties -> '"edge_id"'::agtype)::text));

-- ADVISES (Phase 2 stub)
CREATE INDEX IF NOT EXISTS idx_rif_advises_fwd
    ON rif."ADVISES" USING btree (start_id, end_id);
CREATE INDEX IF NOT EXISTS idx_rif_advises_rev
    ON rif."ADVISES" USING btree (end_id);
CREATE INDEX IF NOT EXISTS idx_rif_advises_edge_id
    ON rif."ADVISES" USING btree (((properties -> '"edge_id"'::agtype)::text));

-- CALLS_SOAP (Phase 2 stub)
CREATE INDEX IF NOT EXISTS idx_rif_calls_soap_fwd
    ON rif."CALLS_SOAP" USING btree (start_id, end_id);
CREATE INDEX IF NOT EXISTS idx_rif_calls_soap_rev
    ON rif."CALLS_SOAP" USING btree (end_id);
CREATE INDEX IF NOT EXISTS idx_rif_calls_soap_edge_id
    ON rif."CALLS_SOAP" USING btree (((properties -> '"edge_id"'::agtype)::text));

-- CALLS_REST (Phase 2 stub)
CREATE INDEX IF NOT EXISTS idx_rif_calls_rest_fwd
    ON rif."CALLS_REST" USING btree (start_id, end_id);
CREATE INDEX IF NOT EXISTS idx_rif_calls_rest_rev
    ON rif."CALLS_REST" USING btree (end_id);
CREATE INDEX IF NOT EXISTS idx_rif_calls_rest_edge_id
    ON rif."CALLS_REST" USING btree (((properties -> '"edge_id"'::agtype)::text));

-- ---------------------------------------------------------------------------
-- End of age_schema.sql
-- ---------------------------------------------------------------------------
