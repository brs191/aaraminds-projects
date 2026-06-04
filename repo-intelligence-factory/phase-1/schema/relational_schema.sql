-- Relational projection + provenance enforcement for the M0 graph.
-- Phase 1 — Repository Intelligence Factory. Lives in the SAME Postgres instance
-- as the AGE graph (one datastore, per TARGET_ARCHITECTURE §7). The graph is the
-- traversal engine; this projection is where uniqueness, provenance NOT-NULL, and
-- commit-consistent versioning are ENFORCED, and what you JOIN graph results against.

CREATE TABLE IF NOT EXISTS index_versions (
    index_version   TEXT PRIMARY KEY,          -- e.g. 'extractor-1.1.0'
    repo            TEXT NOT NULL,
    repo_sha        TEXT NOT NULL,
    scip_tier       TEXT NOT NULL DEFAULT 'ast' CHECK (scip_tier IN ('ast','scip')),
    complete        BOOLEAN NOT NULL DEFAULT FALSE,   -- readers pin only to complete versions
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Nodes projection. Every graph node is mirrored here for the integrity guarantees
-- AGE cannot give. The deterministic id is the PK, so re-running the same SHA
-- upserts idempotently and the build-to-build diff is meaningful.
CREATE TABLE IF NOT EXISTS symbols (
    id              TEXT PRIMARY KEY,                  -- deterministic: 'type:FQN', 'method:FQN#name(params)', ...
    label           TEXT NOT NULL CHECK (label IN
                      ('Repository','Module','Package','File','Type','Method','Field',
                       'Endpoint','DataStore','Aspect','Generated','BuildMeta')),
    name            TEXT NOT NULL,
    kind            TEXT,
    stereotype      TEXT,
    provenance      TEXT NOT NULL CHECK (provenance IN ('deterministic','inferred','generated','external','system')),
    confidence      TEXT NOT NULL CHECK (confidence IN ('exact','probable','inferred')),
    evidence        TEXT,
    source_ref      TEXT,                              -- repo@sha:path:line-range
    index_version   TEXT NOT NULL REFERENCES index_versions(index_version),
    -- THE 100% self-citation gate, enforced structurally: every non-external,
    -- non-system node MUST carry a resolvable source_ref.
    CONSTRAINT symbols_provenance_gate CHECK (
        provenance IN ('external','system') OR source_ref IS NOT NULL
    )
);
CREATE INDEX IF NOT EXISTS symbols_label_idx ON symbols(label);
CREATE INDEX IF NOT EXISTS symbols_iv_idx    ON symbols(index_version);

-- Edges projection. Same provenance gate; FK to both endpoints so a dangling edge
-- cannot exist (the extractor must emit a target node for every edge target).
CREATE TABLE IF NOT EXISTS edges (
    id              TEXT PRIMARY KEY,                  -- 'edge:TYPE:src->dst'
    type            TEXT NOT NULL CHECK (type IN
                      ('CONTAINS','DEFINES','IMPORTS','EXTENDS','IMPLEMENTS','CALLS',
                       'INJECTS','EXPOSES','READS_FROM','WRITES_TO','ADVISES','CALLS_SERVICE')),
    src_id          TEXT NOT NULL REFERENCES symbols(id),
    dst_id          TEXT NOT NULL REFERENCES symbols(id),
    provenance      TEXT NOT NULL CHECK (provenance IN ('deterministic','inferred','generated','external','system')),
    confidence      TEXT NOT NULL CHECK (confidence IN ('exact','probable','inferred')),
    evidence        TEXT,
    source_ref      TEXT,
    call_site       TEXT,                              -- CALLS only
    weave_kind      TEXT,                              -- ADVISES only: before|after|around
    index_version   TEXT NOT NULL REFERENCES index_versions(index_version),
    CONSTRAINT edges_provenance_gate CHECK (
        provenance IN ('external','system') OR source_ref IS NOT NULL
    )
);
CREATE INDEX IF NOT EXISTS edges_src_idx  ON edges(src_id);
CREATE INDEX IF NOT EXISTS edges_dst_idx  ON edges(dst_id);
CREATE INDEX IF NOT EXISTS edges_type_idx ON edges(type);

-- Endpoint detail (queried directly for the API-surface answers).
CREATE TABLE IF NOT EXISTS endpoints (
    id              TEXT PRIMARY KEY REFERENCES symbols(id),
    http_method     TEXT NOT NULL,
    path            TEXT NOT NULL,
    handler_id      TEXT REFERENCES symbols(id),
    index_version   TEXT NOT NULL REFERENCES index_versions(index_version)
);

-- Incremental-extraction bookkeeping (Phase 5 uses it; Phase 1 just records jobs).
CREATE TABLE IF NOT EXISTS jobs (
    id              BIGSERIAL PRIMARY KEY,
    repo            TEXT NOT NULL,
    from_sha        TEXT,
    to_sha          TEXT NOT NULL,
    index_version   TEXT REFERENCES index_versions(index_version),
    status          TEXT NOT NULL DEFAULT 'queued' CHECK (status IN ('queued','running','done','failed')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Provenance audit (the CI gate, expressed in SQL — must return 0 rows):
--   SELECT id FROM symbols WHERE provenance NOT IN ('external','system') AND source_ref IS NULL
--   UNION ALL
--   SELECT id FROM edges   WHERE provenance NOT IN ('external','system') AND source_ref IS NULL;
