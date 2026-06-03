-- Phase 0 - Workstream C: enable Apache AGE and create the benchmark graph.
-- Run AFTER provision.sh has set azure.extensions + shared_preload_libraries = AGE
-- and the server has restarted. Connect with psql, then: \i setup.sql

CREATE EXTENSION IF NOT EXISTS age CASCADE;
LOAD 'age';
SET search_path = ag_catalog, "$user", public;

-- Recreate the graph cleanly if re-running.
SELECT drop_graph('codegraph', true) FROM ag_graph WHERE name = 'codegraph';
SELECT create_graph('codegraph');

-- ---------------------------------------------------------------------------
-- Load nodes/edges. Two options:
--
-- (A) FAITHFUL: export the deterministic graph from the potpie/Neo4j spike
--     (Workstream A) - Function/Class/File nodes + CALLS/IMPORTS edges - and
--     bulk-insert them as Cypher CREATE statements here. This is the real test.
--
-- (B) QUICK: let benchmark.py synthesize a graph at the repo's scale:
--       python benchmark.py --generate --nodes 2500 --avg-degree 4
--     Proves AGE's traversal engine in isolation, not on real shape.
--
-- Node + edge shape the benchmark expects:
--   SELECT * FROM cypher('codegraph', $$
--     CREATE (:Function {fqn:'com.att.credit.Router.route', file:'Router.java'}) $$) AS (v agtype);
--   SELECT * FROM cypher('codegraph', $$
--     MATCH (a:Function {fqn:'A'}),(b:Function {fqn:'B'}) CREATE (a)-[:CALLS]->(b) $$) AS (e agtype);
-- ---------------------------------------------------------------------------

-- Sanity check after loading:
-- SELECT * FROM cypher('codegraph', $$ MATCH (f:Function) RETURN count(f) $$) AS (n agtype);
