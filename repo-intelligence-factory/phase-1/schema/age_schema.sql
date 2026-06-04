-- M0 graph schema for Apache AGE (openCypher on Postgres).
-- Phase 1 — Repository Intelligence Factory. Run after AGE is enabled:
--   CREATE EXTENSION IF NOT EXISTS age CASCADE; LOAD 'age';
--   SET search_path = ag_catalog, "$user", public;
-- Then: \i age_schema.sql
--
-- AGE does not enforce a property schema, so this file (a) creates the graph,
-- (b) declares every node/edge label so labels exist before MERGE, and
-- (c) adds property indexes on the deterministic id + on source_ref/index_version,
-- which are the hot lookup + provenance-audit paths. The authoritative
-- uniqueness/NOT-NULL guarantees live in relational_schema.sql (the projection).

SELECT drop_graph('codegraph', true) FROM ag_graph WHERE name = 'codegraph';
SELECT create_graph('codegraph');

-- Node labels (vlabels) -------------------------------------------------------
SELECT create_vlabel('codegraph','Repository');
SELECT create_vlabel('codegraph','Module');
SELECT create_vlabel('codegraph','Package');
SELECT create_vlabel('codegraph','File');
SELECT create_vlabel('codegraph','Type');
SELECT create_vlabel('codegraph','Method');
SELECT create_vlabel('codegraph','Field');
SELECT create_vlabel('codegraph','Endpoint');
SELECT create_vlabel('codegraph','DataStore');
SELECT create_vlabel('codegraph','Aspect');
SELECT create_vlabel('codegraph','Generated');
SELECT create_vlabel('codegraph','BuildMeta');

-- Edge labels (elabels) -------------------------------------------------------
SELECT create_elabel('codegraph','CONTAINS');
SELECT create_elabel('codegraph','DEFINES');
SELECT create_elabel('codegraph','IMPORTS');
SELECT create_elabel('codegraph','EXTENDS');
SELECT create_elabel('codegraph','IMPLEMENTS');
SELECT create_elabel('codegraph','CALLS');
SELECT create_elabel('codegraph','INJECTS');
SELECT create_elabel('codegraph','EXPOSES');
SELECT create_elabel('codegraph','READS_FROM');
SELECT create_elabel('codegraph','WRITES_TO');
SELECT create_elabel('codegraph','ADVISES');
SELECT create_elabel('codegraph','CALLS_SERVICE');

-- Property indexes ------------------------------------------------------------
-- id is the MERGE key + the join key against the relational projection; index it
-- per label that is read at query time. (AGE stores properties as agtype; index
-- the extracted scalar.) Bounded-depth traversals start from an id lookup, so
-- these indexes are what keep the AGE go/no-go (gate G7) inside budget.
CREATE INDEX IF NOT EXISTS type_id_idx     ON codegraph."Type"     USING btree (agtype_access_operator(properties, '"id"'::agtype));
CREATE INDEX IF NOT EXISTS method_id_idx   ON codegraph."Method"   USING btree (agtype_access_operator(properties, '"id"'::agtype));
CREATE INDEX IF NOT EXISTS endpoint_id_idx ON codegraph."Endpoint" USING btree (agtype_access_operator(properties, '"id"'::agtype));

-- Sanity check after a load:
--   SELECT * FROM cypher('codegraph', $$ MATCH (n) RETURN labels(n), count(*) $$) AS (lbl agtype, n agtype);
--   SELECT * FROM cypher('codegraph', $$ MATCH ()-[e]->() RETURN type(e), count(*) $$) AS (t agtype, n agtype);
