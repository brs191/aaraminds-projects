-- DIF initial metadata schema.
--
-- Scope:
-- - Creates only objects in dif_meta.
-- - Does not create, alter, or drop objects in RIF-owned schemas such as rif or rif_meta.
-- - Is safe to run more than once.
--
-- Notes:
-- - pgcrypto is used for gen_random_uuid() defaults.
-- - pgvector is intentionally deferred until the embedding model and vector dimension are pinned.
-- - Full-text search uses built-in PostgreSQL tsvector and GIN indexing.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE SCHEMA IF NOT EXISTS dif_meta;

CREATE TABLE IF NOT EXISTS dif_meta.corpora (
    corpus_id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    display_name TEXT NOT NULL,
    admission_status TEXT NOT NULL DEFAULT 'pending',
    readability_model TEXT NOT NULL DEFAULT 'uniform_readable',
    admission_evidence JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT corpora_admission_status_check
        CHECK (admission_status IN ('pending', 'admitted', 'rejected', 'archived')),
    CONSTRAINT corpora_readability_model_check
        CHECK (readability_model = 'uniform_readable')
);

CREATE INDEX IF NOT EXISTS idx_corpora_project_id
    ON dif_meta.corpora (project_id);

CREATE INDEX IF NOT EXISTS idx_corpora_admission_status
    ON dif_meta.corpora (admission_status);

INSERT INTO dif_meta.corpora (
    corpus_id,
    project_id,
    display_name,
    admission_status,
    readability_model,
    admission_evidence
) VALUES (
    'dif-auth-unknown-corpus',
    'dif-auth-unknown-project',
    'DIF unknown-scope authentication audit sentinel',
    'archived',
    'uniform_readable',
    '{"purpose":"Allows denied authentication attempts without request scope to satisfy audit and usage corpus foreign keys."}'::jsonb
) ON CONFLICT (corpus_id) DO NOTHING;

CREATE TABLE IF NOT EXISTS dif_meta.sources (
    source_id TEXT PRIMARY KEY,
    corpus_id TEXT NOT NULL REFERENCES dif_meta.corpora (corpus_id) ON DELETE CASCADE,
    source_type TEXT NOT NULL,
    source_uri TEXT NOT NULL,
    scope_path TEXT,
    admission_status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT sources_source_type_check
        CHECK (source_type IN ('local_tree', 'git', 'sharepoint', 'onedrive')),
    CONSTRAINT sources_admission_status_check
        CHECK (admission_status IN ('pending', 'admitted', 'rejected', 'archived'))
);

CREATE INDEX IF NOT EXISTS idx_sources_corpus_id
    ON dif_meta.sources (corpus_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_sources_corpus_source_uri_scope
    ON dif_meta.sources (corpus_id, source_uri, COALESCE(scope_path, ''));

CREATE TABLE IF NOT EXISTS dif_meta.documents (
    document_id TEXT PRIMARY KEY,
    corpus_id TEXT NOT NULL REFERENCES dif_meta.corpora (corpus_id) ON DELETE CASCADE,
    source_id TEXT NOT NULL REFERENCES dif_meta.sources (source_id) ON DELETE CASCADE,
    source_uri TEXT NOT NULL,
    path TEXT NOT NULL,
    format TEXT NOT NULL,
    current_version_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT documents_format_check
        CHECK (format IN ('md', 'txt', 'docx', 'json'))
);

CREATE INDEX IF NOT EXISTS idx_documents_corpus_path
    ON dif_meta.documents (corpus_id, path);

CREATE INDEX IF NOT EXISTS idx_documents_source_path
    ON dif_meta.documents (source_id, path);

CREATE UNIQUE INDEX IF NOT EXISTS idx_documents_corpus_source_uri
    ON dif_meta.documents (corpus_id, source_uri);

CREATE TABLE IF NOT EXISTS dif_meta.document_versions (
    document_version_id TEXT PRIMARY KEY,
    document_id TEXT NOT NULL REFERENCES dif_meta.documents (document_id) ON DELETE CASCADE,
    corpus_id TEXT NOT NULL REFERENCES dif_meta.corpora (corpus_id) ON DELETE CASCADE,
    source_id TEXT NOT NULL REFERENCES dif_meta.sources (source_id) ON DELETE CASCADE,
    content_hash TEXT NOT NULL,
    source_size_bytes BIGINT,
    format TEXT NOT NULL,
    extractor_name TEXT NOT NULL,
    extractor_version TEXT NOT NULL,
    parser_metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT document_versions_format_check
        CHECK (format IN ('md', 'txt', 'docx', 'json')),
    CONSTRAINT document_versions_source_size_check
        CHECK (source_size_bytes IS NULL OR source_size_bytes >= 0),
    CONSTRAINT document_versions_document_hash_extractor_unique
        UNIQUE (document_id, content_hash, extractor_version)
);

CREATE INDEX IF NOT EXISTS idx_document_versions_corpus_id
    ON dif_meta.document_versions (corpus_id);

CREATE INDEX IF NOT EXISTS idx_document_versions_source_id
    ON dif_meta.document_versions (source_id);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'documents_current_version_fk'
          AND conrelid = 'dif_meta.documents'::regclass
    ) THEN
        ALTER TABLE dif_meta.documents
            ADD CONSTRAINT documents_current_version_fk
            FOREIGN KEY (current_version_id)
            REFERENCES dif_meta.document_versions (document_version_id);
    END IF;
END
$$;

CREATE TABLE IF NOT EXISTS dif_meta.nodes (
    node_id TEXT PRIMARY KEY,
    corpus_id TEXT NOT NULL REFERENCES dif_meta.corpora (corpus_id) ON DELETE CASCADE,
    document_id TEXT NOT NULL REFERENCES dif_meta.documents (document_id) ON DELETE CASCADE,
    document_version_id TEXT NOT NULL REFERENCES dif_meta.document_versions (document_version_id) ON DELETE CASCADE,
    node_kind TEXT NOT NULL,
    parent_node_id TEXT REFERENCES dif_meta.nodes (node_id) ON DELETE CASCADE,
    ordinal INTEGER NOT NULL,
    heading_path TEXT,
    anchor_id TEXT,
    text_hash TEXT,
    caveats JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT nodes_node_kind_check
        CHECK (node_kind IN ('document', 'section', 'block')),
    CONSTRAINT nodes_ordinal_check
        CHECK (ordinal >= 0)
);

CREATE INDEX IF NOT EXISTS idx_nodes_corpus_document_version
    ON dif_meta.nodes (corpus_id, document_version_id);

CREATE INDEX IF NOT EXISTS idx_nodes_document_version_kind_ordinal
    ON dif_meta.nodes (document_version_id, node_kind, ordinal);

CREATE INDEX IF NOT EXISTS idx_nodes_parent_node_id
    ON dif_meta.nodes (parent_node_id);

CREATE INDEX IF NOT EXISTS idx_nodes_anchor_id
    ON dif_meta.nodes (anchor_id);

CREATE TABLE IF NOT EXISTS dif_meta.source_anchors (
    anchor_id TEXT PRIMARY KEY,
    corpus_id TEXT NOT NULL REFERENCES dif_meta.corpora (corpus_id) ON DELETE CASCADE,
    document_id TEXT NOT NULL REFERENCES dif_meta.documents (document_id) ON DELETE CASCADE,
    document_version_id TEXT NOT NULL REFERENCES dif_meta.document_versions (document_version_id) ON DELETE CASCADE,
    source_id TEXT NOT NULL REFERENCES dif_meta.sources (source_id) ON DELETE CASCADE,
    anchor_type TEXT NOT NULL,
    source_ref TEXT NOT NULL,
    path TEXT NOT NULL,
    heading_path TEXT,
    line_start INTEGER,
    line_end INTEGER,
    paragraph_index INTEGER,
    json_path TEXT,
    content_hash TEXT NOT NULL,
    extractor_version TEXT NOT NULL,
    caveats JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT source_anchors_anchor_type_check
        CHECK (anchor_type IN ('md', 'txt', 'docx', 'json')),
    CONSTRAINT source_anchors_line_range_check
        CHECK (
            (line_start IS NULL AND line_end IS NULL)
            OR (line_start IS NOT NULL AND line_end IS NOT NULL AND line_start > 0 AND line_end >= line_start)
        ),
    CONSTRAINT source_anchors_paragraph_index_check
        CHECK (paragraph_index IS NULL OR paragraph_index >= 0),
    CONSTRAINT source_anchors_type_payload_check
        CHECK (
            (anchor_type IN ('md', 'txt') AND line_start IS NOT NULL AND line_end IS NOT NULL)
            OR (anchor_type = 'docx' AND paragraph_index IS NOT NULL)
            OR (anchor_type = 'json' AND json_path IS NOT NULL)
        )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_source_anchors_source_ref
    ON dif_meta.source_anchors (source_ref);

CREATE INDEX IF NOT EXISTS idx_source_anchors_corpus_document_version
    ON dif_meta.source_anchors (corpus_id, document_version_id);

CREATE INDEX IF NOT EXISTS idx_source_anchors_anchor_type
    ON dif_meta.source_anchors (anchor_type);

CREATE INDEX IF NOT EXISTS idx_source_anchors_json_path
    ON dif_meta.source_anchors (json_path)
    WHERE anchor_type = 'json';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'nodes_anchor_fk'
          AND conrelid = 'dif_meta.nodes'::regclass
    ) THEN
        ALTER TABLE dif_meta.nodes
            ADD CONSTRAINT nodes_anchor_fk
            FOREIGN KEY (anchor_id)
            REFERENCES dif_meta.source_anchors (anchor_id);
    END IF;
END
$$;

CREATE TABLE IF NOT EXISTS dif_meta.edges (
    edge_id TEXT PRIMARY KEY,
    corpus_id TEXT NOT NULL REFERENCES dif_meta.corpora (corpus_id) ON DELETE CASCADE,
    document_version_id TEXT NOT NULL REFERENCES dif_meta.document_versions (document_version_id) ON DELETE CASCADE,
    edge_kind TEXT NOT NULL,
    from_node_id TEXT NOT NULL REFERENCES dif_meta.nodes (node_id) ON DELETE CASCADE,
    to_node_id TEXT REFERENCES dif_meta.nodes (node_id) ON DELETE CASCADE,
    to_external_node_id TEXT,
    external_system TEXT,
    confidence TEXT NOT NULL,
    anchor_id TEXT REFERENCES dif_meta.source_anchors (anchor_id),
    caveats JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT edges_edge_kind_check
        CHECK (edge_kind IN ('CONTAINS')),
    CONSTRAINT edges_confidence_check
        CHECK (confidence IN ('exact', 'inferred')),
    CONSTRAINT edges_contains_shape_check
        CHECK (
            edge_kind <> 'CONTAINS'
            OR (
                to_node_id IS NOT NULL
                AND to_external_node_id IS NULL
                AND external_system IS NULL
            )
        )
);

CREATE INDEX IF NOT EXISTS idx_edges_corpus_edge_kind
    ON dif_meta.edges (corpus_id, edge_kind);

CREATE INDEX IF NOT EXISTS idx_edges_from_node_id
    ON dif_meta.edges (from_node_id);

CREATE INDEX IF NOT EXISTS idx_edges_to_node_id
    ON dif_meta.edges (to_node_id);

CREATE INDEX IF NOT EXISTS idx_edges_to_external_node_id
    ON dif_meta.edges (to_external_node_id);

CREATE INDEX IF NOT EXISTS idx_edges_anchor_id
    ON dif_meta.edges (anchor_id);

CREATE TABLE IF NOT EXISTS dif_meta.retrieval_passages (
    passage_id TEXT PRIMARY KEY,
    corpus_id TEXT NOT NULL REFERENCES dif_meta.corpora (corpus_id) ON DELETE CASCADE,
    document_id TEXT NOT NULL REFERENCES dif_meta.documents (document_id) ON DELETE CASCADE,
    document_version_id TEXT NOT NULL REFERENCES dif_meta.document_versions (document_version_id) ON DELETE CASCADE,
    node_id TEXT NOT NULL REFERENCES dif_meta.nodes (node_id) ON DELETE CASCADE,
    anchor_id TEXT NOT NULL REFERENCES dif_meta.source_anchors (anchor_id) ON DELETE RESTRICT,
    passage_kind TEXT NOT NULL,
    text TEXT NOT NULL,
    text_hash TEXT NOT NULL,
    fts_vector TSVECTOR GENERATED ALWAYS AS (to_tsvector('simple', COALESCE(text, ''))) STORED,
    embedding_model TEXT,
    caveats JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT retrieval_passages_passage_kind_check
        CHECK (passage_kind IN ('structural', 'json_subtree')),
    CONSTRAINT retrieval_passages_text_nonempty_check
        CHECK (length(btrim(text)) > 0)
);

CREATE INDEX IF NOT EXISTS idx_retrieval_passages_corpus_document_version
    ON dif_meta.retrieval_passages (corpus_id, document_version_id);

CREATE INDEX IF NOT EXISTS idx_retrieval_passages_anchor_id
    ON dif_meta.retrieval_passages (anchor_id);

CREATE INDEX IF NOT EXISTS idx_retrieval_passages_node_id
    ON dif_meta.retrieval_passages (node_id);

CREATE INDEX IF NOT EXISTS idx_retrieval_passages_fts
    ON dif_meta.retrieval_passages
    USING GIN (fts_vector);

CREATE TABLE IF NOT EXISTS dif_meta.ingestion_runs (
    run_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    corpus_id TEXT NOT NULL REFERENCES dif_meta.corpora (corpus_id) ON DELETE CASCADE,
    source_id TEXT REFERENCES dif_meta.sources (source_id) ON DELETE SET NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'running',
    stage TEXT,
    document_count INTEGER NOT NULL DEFAULT 0,
    node_count INTEGER NOT NULL DEFAULT 0,
    edge_count INTEGER NOT NULL DEFAULT 0,
    anchor_count INTEGER NOT NULL DEFAULT 0,
    passage_count INTEGER NOT NULL DEFAULT 0,
    caveat_count INTEGER NOT NULL DEFAULT 0,
    run_metrics JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT,
    promoted BOOLEAN NOT NULL DEFAULT false,
    CONSTRAINT ingestion_runs_status_check
        CHECK (status IN ('running', 'completed', 'failed', 'cancelled')),
    CONSTRAINT ingestion_runs_counts_nonnegative_check
        CHECK (
            document_count >= 0
            AND node_count >= 0
            AND edge_count >= 0
            AND anchor_count >= 0
            AND passage_count >= 0
            AND caveat_count >= 0
        ),
    CONSTRAINT ingestion_runs_completed_at_check
        CHECK (completed_at IS NULL OR completed_at >= started_at),
    CONSTRAINT ingestion_runs_promoted_guard_check
        CHECK (
            promoted = false
            OR (
                status = 'completed'
                AND document_count > 0
                AND node_count > 0
                AND anchor_count > 0
                AND passage_count > 0
            )
        )
);

CREATE INDEX IF NOT EXISTS idx_ingestion_runs_corpus_started
    ON dif_meta.ingestion_runs (corpus_id, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_ingestion_runs_source_started
    ON dif_meta.ingestion_runs (source_id, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_ingestion_runs_status
    ON dif_meta.ingestion_runs (status);

CREATE TABLE IF NOT EXISTS dif_meta.audit_log (
    audit_id BIGSERIAL PRIMARY KEY,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    principal_id TEXT NOT NULL,
    tenant_id TEXT,
    project_id TEXT NOT NULL,
    corpus_id TEXT NOT NULL REFERENCES dif_meta.corpora (corpus_id) ON DELETE RESTRICT,
    tool_name TEXT NOT NULL,
    tool_version TEXT,
    parameters_hash TEXT NOT NULL,
    outcome TEXT NOT NULL,
    latency_ms INTEGER,
    source_refs JSONB NOT NULL DEFAULT '[]'::jsonb,
    error_class TEXT,
    CONSTRAINT audit_log_outcome_check
        CHECK (outcome IN ('success', 'error', 'denied')),
    CONSTRAINT audit_log_latency_check
        CHECK (latency_ms IS NULL OR latency_ms >= 0)
);

CREATE INDEX IF NOT EXISTS idx_audit_log_corpus_occurred
    ON dif_meta.audit_log (corpus_id, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_log_principal_occurred
    ON dif_meta.audit_log (principal_id, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_log_tool_occurred
    ON dif_meta.audit_log (tool_name, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_log_outcome
    ON dif_meta.audit_log (outcome);

CREATE TABLE IF NOT EXISTS dif_meta.usage_events (
    usage_event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    event_type TEXT NOT NULL,
    tenant_id TEXT,
    project_id TEXT NOT NULL,
    corpus_id TEXT NOT NULL REFERENCES dif_meta.corpora (corpus_id) ON DELETE RESTRICT,
    connector_id TEXT,
    counts JSONB NOT NULL DEFAULT '{}'::jsonb,
    latency_ms INTEGER,
    token_units INTEGER,
    embedding_units INTEGER,
    error_class TEXT,
    CONSTRAINT usage_events_event_type_check
        CHECK (event_type IN (
            'ingestion_run',
            'document_indexed',
            'embedding_batch',
            'mcp_tool_call',
            'agent_request',
            'connector_sync'
        )),
    CONSTRAINT usage_events_latency_check
        CHECK (latency_ms IS NULL OR latency_ms >= 0),
    CONSTRAINT usage_events_token_units_check
        CHECK (token_units IS NULL OR token_units >= 0),
    CONSTRAINT usage_events_embedding_units_check
        CHECK (embedding_units IS NULL OR embedding_units >= 0)
);

CREATE INDEX IF NOT EXISTS idx_usage_events_corpus_occurred
    ON dif_meta.usage_events (corpus_id, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_usage_events_project_event_occurred
    ON dif_meta.usage_events (project_id, event_type, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_usage_events_event_type
    ON dif_meta.usage_events (event_type);

CREATE TABLE IF NOT EXISTS dif_meta.rif_compatibility_status (
    status_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id TEXT NOT NULL,
    checked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    rif_status TEXT NOT NULL,
    database_name TEXT,
    capabilities JSONB NOT NULL DEFAULT '{}'::jsonb,
    missing_capabilities JSONB NOT NULL DEFAULT '[]'::jsonb,
    caveats JSONB NOT NULL DEFAULT '[]'::jsonb,
    CONSTRAINT rif_compatibility_status_check
        CHECK (rif_status IN (
            'rif_not_deployed',
            'rif_incompatible',
            'rif_shadow_empty',
            'rif_compatible'
        ))
);

CREATE INDEX IF NOT EXISTS idx_rif_compatibility_status_project_checked
    ON dif_meta.rif_compatibility_status (project_id, checked_at DESC);

CREATE INDEX IF NOT EXISTS idx_rif_compatibility_status_status
    ON dif_meta.rif_compatibility_status (rif_status);

CREATE TABLE IF NOT EXISTS dif_meta.code_entity_candidates (
    candidate_id TEXT PRIMARY KEY,
    corpus_id TEXT NOT NULL REFERENCES dif_meta.corpora (corpus_id) ON DELETE CASCADE,
    document_id TEXT NOT NULL REFERENCES dif_meta.documents (document_id) ON DELETE CASCADE,
    document_version_id TEXT NOT NULL REFERENCES dif_meta.document_versions (document_version_id) ON DELETE CASCADE,
    node_id TEXT NOT NULL REFERENCES dif_meta.nodes (node_id) ON DELETE CASCADE,
    anchor_id TEXT NOT NULL REFERENCES dif_meta.source_anchors (anchor_id) ON DELETE RESTRICT,
    candidate_text TEXT NOT NULL,
    candidate_kind TEXT,
    match_status TEXT NOT NULL DEFAULT 'unresolved',
    resolved_rif_node_id TEXT,
    match_mode TEXT,
    confidence TEXT,
    caveats JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ,
    CONSTRAINT code_entity_candidates_candidate_kind_check
        CHECK (candidate_kind IS NULL OR candidate_kind IN ('class', 'method', 'file_path', 'service', 'unknown')),
    CONSTRAINT code_entity_candidates_match_status_check
        CHECK (match_status IN ('unresolved', 'resolved', 'ambiguous', 'rif_unavailable')),
    CONSTRAINT code_entity_candidates_match_mode_check
        CHECK (match_mode IS NULL OR match_mode IN ('qualified-name', 'source-path', 'simple-name', 'fuzzy')),
    CONSTRAINT code_entity_candidates_confidence_check
        CHECK (confidence IS NULL OR confidence IN ('exact', 'inferred')),
    CONSTRAINT code_entity_candidates_resolved_shape_check
        CHECK (
            (match_status = 'resolved' AND resolved_rif_node_id IS NOT NULL)
            OR (match_status <> 'resolved')
        )
);

CREATE INDEX IF NOT EXISTS idx_code_entity_candidates_corpus_document_version
    ON dif_meta.code_entity_candidates (corpus_id, document_version_id);

CREATE INDEX IF NOT EXISTS idx_code_entity_candidates_node_id
    ON dif_meta.code_entity_candidates (node_id);

CREATE INDEX IF NOT EXISTS idx_code_entity_candidates_anchor_id
    ON dif_meta.code_entity_candidates (anchor_id);

CREATE INDEX IF NOT EXISTS idx_code_entity_candidates_match_status
    ON dif_meta.code_entity_candidates (match_status);

CREATE INDEX IF NOT EXISTS idx_code_entity_candidates_resolved_rif_node_id
    ON dif_meta.code_entity_candidates (resolved_rif_node_id);
