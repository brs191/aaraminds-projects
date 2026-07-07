--
-- RIF Phase 1 — rif_meta schema DDL snapshot
-- Captured: 2026-06-10 from repointel (Postgres 14.23 + AGE 1.5.0)
-- Source of truth: phase-1/schema/relational_schema.sql
-- Purpose: version-controlled baseline for schema drift detection in CI.
--   Refresh with: pg_dump $DATABASE_URL --schema=rif_meta --schema-only --no-owner --no-acl
-- Note: \restrict / \unrestrict lines are pg_dump integrity nonces — not credentials.
--
-- PostgreSQL database dump
--

\restrict bl5qpfGAUpEdifoltD7dfDxNOJP85WLbMuBvqO3GCJnkSfe9X7wPn8Bly3vjkie

-- Dumped from database version 14.23 (Homebrew)
-- Dumped by pg_dump version 14.23 (Homebrew)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: rif_meta; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA rif_meta;


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: file_nodes; Type: TABLE; Schema: rif_meta; Owner: -
--

CREATE TABLE rif_meta.file_nodes (
    node_id text NOT NULL,
    repo_id text NOT NULL,
    qualified_name text NOT NULL,
    package text,
    line_count integer,
    source_ref text NOT NULL,
    index_version integer NOT NULL,
    origin text DEFAULT 'first_party'::text NOT NULL,
    upserted_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_fn_origin CHECK ((origin = ANY (ARRAY['first_party'::text, 'external_stub'::text])))
);


--
-- Name: TABLE file_nodes; Type: COMMENT; Schema: rif_meta; Owner: -
--

COMMENT ON TABLE rif_meta.file_nodes IS 'Relational shadow of AGE File vertices. Kept in sync by the Ingestion Service. Phase 2 adds embedding vector(768) for semantic file search (jina-code-embeddings-1.5b; see FINDINGS_MEMO.md §4).';


--
-- Name: index_runs; Type: TABLE; Schema: rif_meta; Owner: -
--

CREATE TABLE rif_meta.index_runs (
    run_id uuid DEFAULT gen_random_uuid() NOT NULL,
    repo_id text NOT NULL,
    sha character(40) NOT NULL,
    index_version integer NOT NULL,
    extractor_version text NOT NULL,
    started_at timestamp with time zone DEFAULT now() NOT NULL,
    completed_at timestamp with time zone,
    status text DEFAULT 'running'::text NOT NULL,
    node_count integer,
    edge_count integer,
    run_metrics jsonb,
    error_message text,
    CONSTRAINT chk_completed_at CHECK (((status = ANY (ARRAY['running'::text, 'cancelled'::text])) OR (completed_at IS NOT NULL))),
    CONSTRAINT chk_run_status CHECK ((status = ANY (ARRAY['running'::text, 'completed'::text, 'failed'::text, 'cancelled'::text])))
);


--
-- Name: TABLE index_runs; Type: COMMENT; Schema: rif_meta; Owner: -
--

COMMENT ON TABLE rif_meta.index_runs IS 'One row per extraction invocation. Includes failed and cancelled runs. The CI provenance gate updates status to ''completed'' or ''failed'' and writes the final node_count / edge_count / run_metrics here.';


--
-- Name: COLUMN index_runs.run_metrics; Type: COMMENT; Schema: rif_meta; Owner: -
--

COMMENT ON COLUMN rif_meta.index_runs.run_metrics IS 'JSONB map of extractor counters from CODE_MODEL.md §5.6: same_file_resolution_failure_count, unresolved_param_type_count, provenance_gap_count, unsupported_construct_count, stub_node_count, total_files_parsed.';


--
-- Name: index_versions; Type: TABLE; Schema: rif_meta; Owner: -
--

CREATE TABLE rif_meta.index_versions (
    repo_id text NOT NULL,
    version integer NOT NULL,
    sha character(40) NOT NULL,
    extractor_version text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE index_versions; Type: COMMENT; Schema: rif_meta; Owner: -
--

COMMENT ON TABLE rif_meta.index_versions IS 'Immutable ledger of every successfully completed index version per repo. Phase 2 adds SCIP-derived versions alongside Phase 1 AST versions.';


--
-- Name: method_nodes; Type: TABLE; Schema: rif_meta; Owner: -
--

CREATE TABLE rif_meta.method_nodes (
    node_id text NOT NULL,
    repo_id text NOT NULL,
    qualified_name text NOT NULL,
    simple_name text NOT NULL,
    return_type text,
    visibility text,
    is_static boolean,
    source_ref text NOT NULL,
    index_version integer NOT NULL,
    origin text DEFAULT 'first_party'::text NOT NULL,
    upserted_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_mn_origin CHECK ((origin = ANY (ARRAY['first_party'::text, 'external_stub'::text]))),
    CONSTRAINT chk_mn_visibility CHECK (((visibility IS NULL) OR (visibility = ANY (ARRAY['public'::text, 'protected'::text, 'package'::text, 'private'::text]))))
);


--
-- Name: TABLE method_nodes; Type: COMMENT; Schema: rif_meta; Owner: -
--

COMMENT ON TABLE rif_meta.method_nodes IS 'Relational shadow of AGE Method vertices. Kept in sync by the Ingestion Service. Phase 2 adds embedding vector(768) for semantic method search and hybrid impact-analysis retrieval (graph traversal + vector similarity).';


--
-- Name: provenance_failures; Type: TABLE; Schema: rif_meta; Owner: -
--

CREATE TABLE rif_meta.provenance_failures (
    failure_id bigint NOT NULL,
    run_id uuid NOT NULL,
    repo_id text NOT NULL,
    entity_type text NOT NULL,
    entity_id text NOT NULL,
    label text NOT NULL,
    qualified_name text,
    source_ref text,
    failure_reason text NOT NULL,
    detected_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_pf_entity_type CHECK ((entity_type = ANY (ARRAY['node'::text, 'edge'::text])))
);


--
-- Name: TABLE provenance_failures; Type: COMMENT; Schema: rif_meta; Owner: -
--

COMMENT ON TABLE rif_meta.provenance_failures IS 'Written by the CI provenance gate when a first-party node/edge has an invalid or missing source_ref. Gate assertion: zero rows for current run_id.';


--
-- Name: COLUMN provenance_failures.failure_reason; Type: COMMENT; Schema: rif_meta; Owner: -
--

COMMENT ON COLUMN rif_meta.provenance_failures.failure_reason IS 'Machine-readable code. Values: UNAVAILABLE (source_ref starts with ''UNAVAILABLE:''), FORMAT_MISMATCH (does not match repo@sha:path:line), MISSING_SOURCE_REF (null or empty), STUB_IN_FIRST_PARTY (stub marker found on a first_party origin node).';


--
-- Name: provenance_failures_failure_id_seq; Type: SEQUENCE; Schema: rif_meta; Owner: -
--

CREATE SEQUENCE rif_meta.provenance_failures_failure_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: provenance_failures_failure_id_seq; Type: SEQUENCE OWNED BY; Schema: rif_meta; Owner: -
--

ALTER SEQUENCE rif_meta.provenance_failures_failure_id_seq OWNED BY rif_meta.provenance_failures.failure_id;


--
-- Name: repositories; Type: TABLE; Schema: rif_meta; Owner: -
--

CREATE TABLE rif_meta.repositories (
    repo_id text NOT NULL,
    clone_url text NOT NULL,
    current_sha character(40),
    current_index_version integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_repo_id_no_special CHECK ((repo_id ~ '^[A-Za-z0-9_\-]+$'::text))
);


--
-- Name: TABLE repositories; Type: COMMENT; Schema: rif_meta; Owner: -
--

COMMENT ON TABLE rif_meta.repositories IS 'One row per registered repository. repo_id is the stable identifier embedded in every node/edge source_ref value (format: repo_id@sha:path:line).';


--
-- Name: COLUMN repositories.current_sha; Type: COMMENT; Schema: rif_meta; Owner: -
--

COMMENT ON COLUMN rif_meta.repositories.current_sha IS '40-char SHA-1 of the commit currently indexed in the AGE graph. NULL until first successful run.';


--
-- Name: COLUMN repositories.current_index_version; Type: COMMENT; Schema: rif_meta; Owner: -
--

COMMENT ON COLUMN rif_meta.repositories.current_index_version IS 'Monotonically increasing counter; incremented on each successful ingestion run.';


--
-- Name: provenance_failures failure_id; Type: DEFAULT; Schema: rif_meta; Owner: -
--

ALTER TABLE ONLY rif_meta.provenance_failures ALTER COLUMN failure_id SET DEFAULT nextval('rif_meta.provenance_failures_failure_id_seq'::regclass);


--
-- Name: file_nodes pk_file_nodes; Type: CONSTRAINT; Schema: rif_meta; Owner: -
--

ALTER TABLE ONLY rif_meta.file_nodes
    ADD CONSTRAINT pk_file_nodes PRIMARY KEY (node_id);


--
-- Name: index_runs pk_index_runs; Type: CONSTRAINT; Schema: rif_meta; Owner: -
--

ALTER TABLE ONLY rif_meta.index_runs
    ADD CONSTRAINT pk_index_runs PRIMARY KEY (run_id);


--
-- Name: index_versions pk_index_versions; Type: CONSTRAINT; Schema: rif_meta; Owner: -
--

ALTER TABLE ONLY rif_meta.index_versions
    ADD CONSTRAINT pk_index_versions PRIMARY KEY (repo_id, version);


--
-- Name: method_nodes pk_method_nodes; Type: CONSTRAINT; Schema: rif_meta; Owner: -
--

ALTER TABLE ONLY rif_meta.method_nodes
    ADD CONSTRAINT pk_method_nodes PRIMARY KEY (node_id);


--
-- Name: provenance_failures pk_provenance_failures; Type: CONSTRAINT; Schema: rif_meta; Owner: -
--

ALTER TABLE ONLY rif_meta.provenance_failures
    ADD CONSTRAINT pk_provenance_failures PRIMARY KEY (failure_id);


--
-- Name: repositories pk_repositories; Type: CONSTRAINT; Schema: rif_meta; Owner: -
--

ALTER TABLE ONLY rif_meta.repositories
    ADD CONSTRAINT pk_repositories PRIMARY KEY (repo_id);


--
-- Name: idx_file_nodes_package; Type: INDEX; Schema: rif_meta; Owner: -
--

CREATE INDEX idx_file_nodes_package ON rif_meta.file_nodes USING btree (repo_id, package) WHERE (package IS NOT NULL);


--
-- Name: idx_file_nodes_repo_id; Type: INDEX; Schema: rif_meta; Owner: -
--

CREATE INDEX idx_file_nodes_repo_id ON rif_meta.file_nodes USING btree (repo_id);


--
-- Name: idx_index_runs_repo_status; Type: INDEX; Schema: rif_meta; Owner: -
--

CREATE INDEX idx_index_runs_repo_status ON rif_meta.index_runs USING btree (repo_id, status);


--
-- Name: idx_index_runs_started_at; Type: INDEX; Schema: rif_meta; Owner: -
--

CREATE INDEX idx_index_runs_started_at ON rif_meta.index_runs USING btree (started_at DESC);


--
-- Name: idx_method_nodes_repo_id; Type: INDEX; Schema: rif_meta; Owner: -
--

CREATE INDEX idx_method_nodes_repo_id ON rif_meta.method_nodes USING btree (repo_id);


--
-- Name: idx_method_nodes_simple_name; Type: INDEX; Schema: rif_meta; Owner: -
--

CREATE INDEX idx_method_nodes_simple_name ON rif_meta.method_nodes USING btree (repo_id, simple_name);


--
-- Name: idx_pf_repo_id; Type: INDEX; Schema: rif_meta; Owner: -
--

CREATE INDEX idx_pf_repo_id ON rif_meta.provenance_failures USING btree (repo_id);


--
-- Name: idx_pf_run_id; Type: INDEX; Schema: rif_meta; Owner: -
--

CREATE INDEX idx_pf_run_id ON rif_meta.provenance_failures USING btree (run_id);


--
-- Name: file_nodes fk_fn_repo; Type: FK CONSTRAINT; Schema: rif_meta; Owner: -
--

ALTER TABLE ONLY rif_meta.file_nodes
    ADD CONSTRAINT fk_fn_repo FOREIGN KEY (repo_id) REFERENCES rif_meta.repositories(repo_id);


--
-- Name: index_versions fk_index_versions_repo; Type: FK CONSTRAINT; Schema: rif_meta; Owner: -
--

ALTER TABLE ONLY rif_meta.index_versions
    ADD CONSTRAINT fk_index_versions_repo FOREIGN KEY (repo_id) REFERENCES rif_meta.repositories(repo_id) ON DELETE CASCADE;


--
-- Name: method_nodes fk_mn_repo; Type: FK CONSTRAINT; Schema: rif_meta; Owner: -
--

ALTER TABLE ONLY rif_meta.method_nodes
    ADD CONSTRAINT fk_mn_repo FOREIGN KEY (repo_id) REFERENCES rif_meta.repositories(repo_id);


--
-- Name: provenance_failures fk_pf_run; Type: FK CONSTRAINT; Schema: rif_meta; Owner: -
--

ALTER TABLE ONLY rif_meta.provenance_failures
    ADD CONSTRAINT fk_pf_run FOREIGN KEY (run_id) REFERENCES rif_meta.index_runs(run_id);


--
-- Name: index_runs fk_run_repo; Type: FK CONSTRAINT; Schema: rif_meta; Owner: -
--

ALTER TABLE ONLY rif_meta.index_runs
    ADD CONSTRAINT fk_run_repo FOREIGN KEY (repo_id) REFERENCES rif_meta.repositories(repo_id);


--
-- PostgreSQL database dump complete
--

\unrestrict bl5qpfGAUpEdifoltD7dfDxNOJP85WLbMuBvqO3GCJnkSfe9X7wPn8Bly3vjkie

