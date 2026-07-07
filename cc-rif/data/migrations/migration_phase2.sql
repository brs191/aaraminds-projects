-- =============================================================================
-- RIF Phase 2 — Additive Migration
-- =============================================================================
-- Purpose : Extend the Phase 1 schema with Phase 2 graph elements.
--           All Phase 1 schema objects (graph "rif", all vertex labels,
--           all Tier-A edge labels, schema rif_meta and its tables) are
--           preserved intact. This script is ADDITIVE ONLY.
--
-- Stack   : Apache AGE 1.5.0 + Postgres 14.23 (local dev) / PG16 (Azure
--           Flexible Server). Syntax is identical on both versions.
--
-- Run as  : psql -U <superuser> -d <database> -f migration_phase2.sql
-- Reads with: phase-1/design/CODE_MODEL.md (§3 Tier-B and Tier-C edges)
--             phase-1/schema/age_schema.sql  (Phase 1 baseline)
--
-- Idempotent: safe to run multiple times against an existing database.
--             All DDL uses IF NOT EXISTS / ADD COLUMN IF NOT EXISTS guards.
--
-- Phase 1 schema version: preserved — no Phase 1 object is dropped, altered,
-- or renamed. Phase 1 baseline remains at the state produced by:
--   phase-1/schema/age_schema.sql
--   phase-1/schema/relational_schema.sql
--
-- Rollback (manual — no auto-rollback provided):
--   DROP TABLE IF EXISTS rif."URL_ENDPOINT";          -- new vertex label
--   DROP TABLE IF EXISTS rif."POINTCUT_EXPRESSION";   -- new vertex label
--   DROP TABLE IF EXISTS rif."REGISTERS";             -- new edge label
--   ALTER TABLE rif_meta.repositories
--       DROP COLUMN IF EXISTS application_context_node_id;
--   ALTER TABLE rif_meta.index_runs
--       DROP COLUMN IF EXISTS tier_b_edge_count,
--       DROP COLUMN IF EXISTS tier_c_edge_count;
--   -- Note: dropping AGE label tables must go via ag_catalog.drop_label()
--   -- on AGE 1.5.x; direct DROP TABLE on a label table is not supported.
--   -- Use: SELECT * FROM ag_catalog.drop_label('rif', 'URL_ENDPOINT');
--   --      SELECT * FROM ag_catalog.drop_label('rif', 'POINTCUT_EXPRESSION');
--   --      SELECT * FROM ag_catalog.drop_label('rif', 'REGISTERS');
--
-- Execution order:
--   1. Extension load + search_path
--   2. New vertex labels — URL_ENDPOINT + POINTCUT_EXPRESSION
--   3. New edge label    — REGISTERS
--   4. New vertex indexes — URL_ENDPOINT + POINTCUT_EXPRESSION
--   5. New edge indexes   — REGISTERS
--   6. Relational schema extensions (rif_meta)
--   7. Verification query
-- =============================================================================

-- ---------------------------------------------------------------------------
-- 0. Extension load and search_path
-- ---------------------------------------------------------------------------
-- AGE must already be installed (Phase 1 pre-condition).
-- LOAD is session-scoped; must be re-executed on every new connection.
-- The application's pgx pool AfterConnect hook must also issue this pair.

LOAD 'age';
SET search_path = ag_catalog, rif_meta, "$user", public;

-- ---------------------------------------------------------------------------
-- 1. New Vertex Label — URL_ENDPOINT
-- ---------------------------------------------------------------------------
-- Represents a synthetic REST endpoint target identified from RestTemplate,
-- WebClient, or @FeignClient usage when the target is a URL pattern rather
-- than a resolved class node in the graph.
-- See: CODE_MODEL.md §1.2 "URL_ENDPOINT (Synthetic Node — Phase 2)"
--
-- Phase 1 check: URL_ENDPOINT was NOT created in phase-1/schema/age_schema.sql.
-- This block creates it for the first time.

DO $$
DECLARE
    graph_oid oid;
BEGIN
    SELECT graphid INTO graph_oid
    FROM ag_catalog.ag_graph
    WHERE name = 'rif';

    IF graph_oid IS NULL THEN
        RAISE EXCEPTION 'Graph "rif" not found — run phase-1/schema/age_schema.sql first.';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM ag_catalog.ag_label
        WHERE name  = 'URL_ENDPOINT'
          AND graph = graph_oid
          AND kind  = 'v'
    ) THEN
        PERFORM ag_catalog.create_vlabel('rif', 'URL_ENDPOINT');
        RAISE NOTICE 'Vertex label "URL_ENDPOINT" created.';
    ELSE
        RAISE NOTICE 'Vertex label "URL_ENDPOINT" already exists — skipping.';
    END IF;
END;
$$;

-- ---------------------------------------------------------------------------
-- 2. New Vertex Label — POINTCUT_EXPRESSION
-- ---------------------------------------------------------------------------
-- Represents a synthetic pointcut expression target emitted by the AOP extractor.
-- See: phase-2/extractor/aop/SpringAopExtractor.java

DO $$
DECLARE
    graph_oid oid;
BEGIN
    SELECT graphid INTO graph_oid
    FROM ag_catalog.ag_graph
    WHERE name = 'rif';

    IF graph_oid IS NULL THEN
        RAISE EXCEPTION 'Graph "rif" not found — run phase-1/schema/age_schema.sql first.';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM ag_catalog.ag_label
        WHERE name  = 'POINTCUT_EXPRESSION'
          AND graph = graph_oid
          AND kind  = 'v'
    ) THEN
        PERFORM ag_catalog.create_vlabel('rif', 'POINTCUT_EXPRESSION');
        RAISE NOTICE 'Vertex label "POINTCUT_EXPRESSION" created.';
    ELSE
        RAISE NOTICE 'Vertex label "POINTCUT_EXPRESSION" already exists — skipping.';
    END IF;
END;
$$;

-- ---------------------------------------------------------------------------
-- 3. New Edge Label — REGISTERS
-- ---------------------------------------------------------------------------
-- Tier-B (probable): a Spring-stereotyped class registers itself as a bean
-- in the application context. Direction: CLASS → APPLICATION_CONTEXT (virtual).
-- See: CODE_MODEL.md §3.1 REGISTERS
--
-- Phase 1 check: REGISTERS was NOT in the stub edge-label list in
-- phase-1/schema/age_schema.sql (only INJECTS, PRODUCES, ADVISES, CALLS_SOAP,
-- CALLS_REST were pre-created as stubs). This block creates it now.

DO $$
DECLARE
    graph_oid oid;
BEGIN
    SELECT graphid INTO graph_oid
    FROM ag_catalog.ag_graph
    WHERE name = 'rif';

    IF graph_oid IS NULL THEN
        RAISE EXCEPTION 'Graph "rif" not found — run phase-1/schema/age_schema.sql first.';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM ag_catalog.ag_label
        WHERE name  = 'REGISTERS'
          AND graph = graph_oid
          AND kind  = 'e'
    ) THEN
        PERFORM ag_catalog.create_elabel('rif', 'REGISTERS');
        RAISE NOTICE 'Edge label "REGISTERS" created.';
    ELSE
        RAISE NOTICE 'Edge label "REGISTERS" already exists — skipping.';
    END IF;
END;
$$;

-- ---------------------------------------------------------------------------
-- 4. Vertex Indexes — URL_ENDPOINT + POINTCUT_EXPRESSION
-- ---------------------------------------------------------------------------
-- Same index convention as Phase 1:
--   node_id  — primary business-key lookup (SHA-256, 64 hex chars)
--   repo_id  — per-repository bulk queries and re-index
--
-- Expression syntax: ((properties -> '"key"'::agtype)::text)
-- Compatible with PG14+AGE1.5.0 and PG16+AGE1.5.0 (operator defined by AGE).

CREATE INDEX IF NOT EXISTS idx_rif_url_endpoint_node_id
    ON rif."URL_ENDPOINT" USING btree (((properties -> '"node_id"'::agtype)::text));

CREATE INDEX IF NOT EXISTS idx_rif_url_endpoint_repo_id
    ON rif."URL_ENDPOINT" USING btree (((properties -> '"repo_id"'::agtype)::text));

CREATE INDEX IF NOT EXISTS idx_rif_pointcut_expr_node_id
    ON rif."POINTCUT_EXPRESSION" USING btree (((properties -> '"node_id"'::agtype)::text));

CREATE INDEX IF NOT EXISTS idx_rif_pointcut_expr_repo_id
    ON rif."POINTCUT_EXPRESSION" USING btree (((properties -> '"repo_id"'::agtype)::text));

-- ---------------------------------------------------------------------------
-- 5. Edge Indexes — REGISTERS
-- ---------------------------------------------------------------------------
-- Same three-index convention as Phase 1 edge labels:
--   (start_id, end_id) — forward traversal (outgoing from a vertex)
--   (end_id)           — reverse traversal / blast-radius queries
--   (edge_id)          — deduplication and idempotent upsert

CREATE INDEX IF NOT EXISTS idx_rif_registers_fwd
    ON rif."REGISTERS" USING btree (start_id, end_id);

CREATE INDEX IF NOT EXISTS idx_rif_registers_rev
    ON rif."REGISTERS" USING btree (end_id);

CREATE INDEX IF NOT EXISTS idx_rif_registers_edge_id
    ON rif."REGISTERS" USING btree (((properties -> '"edge_id"'::agtype)::text));

-- ---------------------------------------------------------------------------
-- 6. Relational Schema Extensions — rif_meta.repositories
-- ---------------------------------------------------------------------------
-- Add application_context_node_id to track the SHA-256 node_id of the
-- APPLICATION_CONTEXT virtual node for each repository.
--
-- The APPLICATION_CONTEXT node is a synthetic singleton (one per repo) stored
-- in the AGE graph as a Class vertex with origin='virtual'. Its node_id is:
--   SHA-256("APPLICATION_CONTEXT:{repo_id}")
--
-- The extractor writes this value here after creating the virtual node so
-- that REGISTERS edge emission can read the target node_id from a simple
-- relational lookup without issuing a Cypher graph query.
--
-- See: CODE_MODEL.md §1.2 "APPLICATION_CONTEXT (Virtual Node — Phase 2)"

ALTER TABLE rif_meta.repositories
    ADD COLUMN IF NOT EXISTS application_context_node_id TEXT;

COMMENT ON COLUMN rif_meta.repositories.application_context_node_id IS
    'SHA-256 node_id of the APPLICATION_CONTEXT virtual Class vertex for this '
    'repository. Set by the Phase 2 extractor when the virtual node is first '
    'created. Format: lowercase hex, 64 chars. '
    'Algorithm: SHA-256("APPLICATION_CONTEXT:{repo_id}"). '
    'See CODE_MODEL.md §1.2 APPLICATION_CONTEXT.';

-- ---------------------------------------------------------------------------
-- 7. Relational Schema Extensions — rif_meta.index_runs
-- ---------------------------------------------------------------------------
-- Add Phase 2 edge-count tracking columns so run reports can distinguish
-- Tier-B (probable) and Tier-C (inferred) edge populations from Tier-A.
-- These counters are set by the Phase 2 extractor in its run manifest and
-- written here by the Ingestion Service alongside node_count / edge_count.

ALTER TABLE rif_meta.index_runs
    ADD COLUMN IF NOT EXISTS tier_b_edge_count INTEGER DEFAULT 0;

ALTER TABLE rif_meta.index_runs
    ADD COLUMN IF NOT EXISTS tier_c_edge_count INTEGER DEFAULT 0;

COMMENT ON COLUMN rif_meta.index_runs.tier_b_edge_count IS
    'Count of Tier-B (probable) edges written during this run: '
    'INJECTS + PRODUCES + REGISTERS. Set by Phase 2 extractor.';

COMMENT ON COLUMN rif_meta.index_runs.tier_c_edge_count IS
    'Count of Tier-C (inferred) edges written during this run: '
    'ADVISES + CALLS_SOAP + CALLS_REST. Set by Phase 2 extractor.';

-- ---------------------------------------------------------------------------
-- 8. Verification Query
-- ---------------------------------------------------------------------------
-- Lists all edge labels currently registered in the "rif" graph.
-- Run after migration to confirm REGISTERS (new) and all Phase 1 stubs
-- (INJECTS, PRODUCES, ADVISES, CALLS_SOAP, CALLS_REST) are present.
-- Expected labels (11 total after Phase 2 migration):
--   Tier-A (5): IMPORTS, SAME_FILE_CALLS, EXTENDS, IMPLEMENTS, DECLARES_FIELD
--   Tier-B (3): INJECTS, PRODUCES, REGISTERS
--   Tier-C (3): ADVISES, CALLS_SOAP, CALLS_REST

SELECT
    al.name        AS edge_label,
    al.kind        AS kind,       -- should always be 'e' in this query
    ag.name        AS graph_name,
    -- Approximate row count from pg_class for a quick sanity check.
    -- Zero rows expected for all Tier-B/C labels on a freshly migrated DB.
    pc.reltuples::bigint AS approx_row_count
FROM ag_catalog.ag_label  al
JOIN ag_catalog.ag_graph  ag ON ag.graphid = al.graph
JOIN pg_catalog.pg_class  pc ON pc.oid     = al.relation
WHERE ag.name = 'rif'
  AND al.kind = 'e'
ORDER BY al.name;

-- ---------------------------------------------------------------------------
-- End of migration_phase2.sql
-- ---------------------------------------------------------------------------
