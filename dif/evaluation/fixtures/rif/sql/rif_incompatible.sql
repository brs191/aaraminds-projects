-- Synthetic incompatible RIF-like fixture.
-- Missing source_ref and confidence by design.

CREATE SCHEMA IF NOT EXISTS rif_fixture;

CREATE TABLE IF NOT EXISTS rif_fixture.incomplete_entities (
    node_id TEXT PRIMARY KEY,
    repo_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    qualified_name TEXT NOT NULL,
    simple_name TEXT,
    origin TEXT NOT NULL
);

