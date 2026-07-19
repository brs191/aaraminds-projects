-- Synthetic populated-shadow compatibility fixture for future PostgreSQL tests.

CREATE SCHEMA IF NOT EXISTS rif_fixture;

CREATE TABLE IF NOT EXISTS rif_fixture.shadow_entities (
    node_id TEXT PRIMARY KEY,
    repo_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    qualified_name TEXT NOT NULL,
    simple_name TEXT,
    source_ref TEXT NOT NULL,
    origin TEXT NOT NULL,
    confidence TEXT NOT NULL
);

