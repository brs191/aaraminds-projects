-- Synthetic AGE-like compatibility fixture.
-- This file intentionally avoids requiring Apache AGE so it can document the
-- compatibility-view shape in plain PostgreSQL. It must be applied only to a
-- scratch database/schema during future integration tests.

CREATE SCHEMA IF NOT EXISTS rif_fixture;

CREATE TABLE IF NOT EXISTS rif_fixture.age_entities (
    node_id TEXT PRIMARY KEY,
    repo_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    qualified_name TEXT NOT NULL,
    simple_name TEXT,
    source_ref TEXT NOT NULL,
    origin TEXT NOT NULL,
    confidence TEXT NOT NULL
);

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

-- Keep shadow_entities empty to model rif_shadow_empty with AGE fallback.

