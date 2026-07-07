--
-- RIF Phase 1 — AGE rif graph schema DDL snapshot
-- Captured: 2026-06-10 from repointel (Postgres 14.23 + AGE 1.5.0)
-- Source of truth: phase-1/schema/age_schema.sql
-- Purpose: version-controlled baseline for graph schema drift detection in CI.
--   Refresh with: pg_dump $DATABASE_URL --schema=rif --schema-only --no-owner --no-acl
-- Note: \restrict / \unrestrict lines are pg_dump integrity nonces — not credentials.
--
-- PostgreSQL database dump
--

\restrict AKVfibxLilVEeAj1t7ghITfvrOBdzGmQyXygasVrq1D0APM5MQT2krvtNJz3T7n

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
-- Name: rif; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA rif;


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: _ag_label_edge; Type: TABLE; Schema: rif; Owner: -
--

CREATE TABLE rif._ag_label_edge (
    id ag_catalog.graphid NOT NULL,
    start_id ag_catalog.graphid NOT NULL,
    end_id ag_catalog.graphid NOT NULL,
    properties ag_catalog.agtype DEFAULT ag_catalog.agtype_build_map() NOT NULL
);


--
-- Name: ADVISES; Type: TABLE; Schema: rif; Owner: -
--

CREATE TABLE rif."ADVISES" (
)
INHERITS (rif._ag_label_edge);


--
-- Name: ADVISES_id_seq; Type: SEQUENCE; Schema: rif; Owner: -
--

CREATE SEQUENCE rif."ADVISES_id_seq"
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 281474976710655
    CACHE 1;


--
-- Name: ADVISES_id_seq; Type: SEQUENCE OWNED BY; Schema: rif; Owner: -
--

ALTER SEQUENCE rif."ADVISES_id_seq" OWNED BY rif."ADVISES".id;


--
-- Name: CALLS_REST; Type: TABLE; Schema: rif; Owner: -
--

CREATE TABLE rif."CALLS_REST" (
)
INHERITS (rif._ag_label_edge);


--
-- Name: CALLS_REST_id_seq; Type: SEQUENCE; Schema: rif; Owner: -
--

CREATE SEQUENCE rif."CALLS_REST_id_seq"
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 281474976710655
    CACHE 1;


--
-- Name: CALLS_REST_id_seq; Type: SEQUENCE OWNED BY; Schema: rif; Owner: -
--

ALTER SEQUENCE rif."CALLS_REST_id_seq" OWNED BY rif."CALLS_REST".id;


--
-- Name: CALLS_SOAP; Type: TABLE; Schema: rif; Owner: -
--

CREATE TABLE rif."CALLS_SOAP" (
)
INHERITS (rif._ag_label_edge);


--
-- Name: CALLS_SOAP_id_seq; Type: SEQUENCE; Schema: rif; Owner: -
--

CREATE SEQUENCE rif."CALLS_SOAP_id_seq"
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 281474976710655
    CACHE 1;


--
-- Name: CALLS_SOAP_id_seq; Type: SEQUENCE OWNED BY; Schema: rif; Owner: -
--

ALTER SEQUENCE rif."CALLS_SOAP_id_seq" OWNED BY rif."CALLS_SOAP".id;


--
-- Name: _ag_label_vertex; Type: TABLE; Schema: rif; Owner: -
--

CREATE TABLE rif._ag_label_vertex (
    id ag_catalog.graphid NOT NULL,
    properties ag_catalog.agtype DEFAULT ag_catalog.agtype_build_map() NOT NULL
);


--
-- Name: Class; Type: TABLE; Schema: rif; Owner: -
--

CREATE TABLE rif."Class" (
)
INHERITS (rif._ag_label_vertex);


--
-- Name: Class_id_seq; Type: SEQUENCE; Schema: rif; Owner: -
--

CREATE SEQUENCE rif."Class_id_seq"
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 281474976710655
    CACHE 1;


--
-- Name: Class_id_seq; Type: SEQUENCE OWNED BY; Schema: rif; Owner: -
--

ALTER SEQUENCE rif."Class_id_seq" OWNED BY rif."Class".id;


--
-- Name: Constructor; Type: TABLE; Schema: rif; Owner: -
--

CREATE TABLE rif."Constructor" (
)
INHERITS (rif._ag_label_vertex);


--
-- Name: Constructor_id_seq; Type: SEQUENCE; Schema: rif; Owner: -
--

CREATE SEQUENCE rif."Constructor_id_seq"
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 281474976710655
    CACHE 1;


--
-- Name: Constructor_id_seq; Type: SEQUENCE OWNED BY; Schema: rif; Owner: -
--

ALTER SEQUENCE rif."Constructor_id_seq" OWNED BY rif."Constructor".id;


--
-- Name: DECLARES_FIELD; Type: TABLE; Schema: rif; Owner: -
--

CREATE TABLE rif."DECLARES_FIELD" (
)
INHERITS (rif._ag_label_edge);


--
-- Name: DECLARES_FIELD_id_seq; Type: SEQUENCE; Schema: rif; Owner: -
--

CREATE SEQUENCE rif."DECLARES_FIELD_id_seq"
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 281474976710655
    CACHE 1;


--
-- Name: DECLARES_FIELD_id_seq; Type: SEQUENCE OWNED BY; Schema: rif; Owner: -
--

ALTER SEQUENCE rif."DECLARES_FIELD_id_seq" OWNED BY rif."DECLARES_FIELD".id;


--
-- Name: EXTENDS; Type: TABLE; Schema: rif; Owner: -
--

CREATE TABLE rif."EXTENDS" (
)
INHERITS (rif._ag_label_edge);


--
-- Name: EXTENDS_id_seq; Type: SEQUENCE; Schema: rif; Owner: -
--

CREATE SEQUENCE rif."EXTENDS_id_seq"
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 281474976710655
    CACHE 1;


--
-- Name: EXTENDS_id_seq; Type: SEQUENCE OWNED BY; Schema: rif; Owner: -
--

ALTER SEQUENCE rif."EXTENDS_id_seq" OWNED BY rif."EXTENDS".id;


--
-- Name: Enum; Type: TABLE; Schema: rif; Owner: -
--

CREATE TABLE rif."Enum" (
)
INHERITS (rif._ag_label_vertex);


--
-- Name: Enum_id_seq; Type: SEQUENCE; Schema: rif; Owner: -
--

CREATE SEQUENCE rif."Enum_id_seq"
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 281474976710655
    CACHE 1;


--
-- Name: Enum_id_seq; Type: SEQUENCE OWNED BY; Schema: rif; Owner: -
--

ALTER SEQUENCE rif."Enum_id_seq" OWNED BY rif."Enum".id;


--
-- Name: Field; Type: TABLE; Schema: rif; Owner: -
--

CREATE TABLE rif."Field" (
)
INHERITS (rif._ag_label_vertex);


--
-- Name: Field_id_seq; Type: SEQUENCE; Schema: rif; Owner: -
--

CREATE SEQUENCE rif."Field_id_seq"
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 281474976710655
    CACHE 1;


--
-- Name: Field_id_seq; Type: SEQUENCE OWNED BY; Schema: rif; Owner: -
--

ALTER SEQUENCE rif."Field_id_seq" OWNED BY rif."Field".id;


--
-- Name: File; Type: TABLE; Schema: rif; Owner: -
--

CREATE TABLE rif."File" (
)
INHERITS (rif._ag_label_vertex);


--
-- Name: File_id_seq; Type: SEQUENCE; Schema: rif; Owner: -
--

CREATE SEQUENCE rif."File_id_seq"
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 281474976710655
    CACHE 1;


--
-- Name: File_id_seq; Type: SEQUENCE OWNED BY; Schema: rif; Owner: -
--

ALTER SEQUENCE rif."File_id_seq" OWNED BY rif."File".id;


--
-- Name: IMPLEMENTS; Type: TABLE; Schema: rif; Owner: -
--

CREATE TABLE rif."IMPLEMENTS" (
)
INHERITS (rif._ag_label_edge);


--
-- Name: IMPLEMENTS_id_seq; Type: SEQUENCE; Schema: rif; Owner: -
--

CREATE SEQUENCE rif."IMPLEMENTS_id_seq"
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 281474976710655
    CACHE 1;


--
-- Name: IMPLEMENTS_id_seq; Type: SEQUENCE OWNED BY; Schema: rif; Owner: -
--

ALTER SEQUENCE rif."IMPLEMENTS_id_seq" OWNED BY rif."IMPLEMENTS".id;


--
-- Name: IMPORTS; Type: TABLE; Schema: rif; Owner: -
--

CREATE TABLE rif."IMPORTS" (
)
INHERITS (rif._ag_label_edge);


--
-- Name: IMPORTS_id_seq; Type: SEQUENCE; Schema: rif; Owner: -
--

CREATE SEQUENCE rif."IMPORTS_id_seq"
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 281474976710655
    CACHE 1;


--
-- Name: IMPORTS_id_seq; Type: SEQUENCE OWNED BY; Schema: rif; Owner: -
--

ALTER SEQUENCE rif."IMPORTS_id_seq" OWNED BY rif."IMPORTS".id;


--
-- Name: INJECTS; Type: TABLE; Schema: rif; Owner: -
--

CREATE TABLE rif."INJECTS" (
)
INHERITS (rif._ag_label_edge);


--
-- Name: INJECTS_id_seq; Type: SEQUENCE; Schema: rif; Owner: -
--

CREATE SEQUENCE rif."INJECTS_id_seq"
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 281474976710655
    CACHE 1;


--
-- Name: INJECTS_id_seq; Type: SEQUENCE OWNED BY; Schema: rif; Owner: -
--

ALTER SEQUENCE rif."INJECTS_id_seq" OWNED BY rif."INJECTS".id;


--
-- Name: Interface; Type: TABLE; Schema: rif; Owner: -
--

CREATE TABLE rif."Interface" (
)
INHERITS (rif._ag_label_vertex);


--
-- Name: Interface_id_seq; Type: SEQUENCE; Schema: rif; Owner: -
--

CREATE SEQUENCE rif."Interface_id_seq"
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 281474976710655
    CACHE 1;


--
-- Name: Interface_id_seq; Type: SEQUENCE OWNED BY; Schema: rif; Owner: -
--

ALTER SEQUENCE rif."Interface_id_seq" OWNED BY rif."Interface".id;


--
-- Name: Method; Type: TABLE; Schema: rif; Owner: -
--

CREATE TABLE rif."Method" (
)
INHERITS (rif._ag_label_vertex);


--
-- Name: Method_id_seq; Type: SEQUENCE; Schema: rif; Owner: -
--

CREATE SEQUENCE rif."Method_id_seq"
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 281474976710655
    CACHE 1;


--
-- Name: Method_id_seq; Type: SEQUENCE OWNED BY; Schema: rif; Owner: -
--

ALTER SEQUENCE rif."Method_id_seq" OWNED BY rif."Method".id;


--
-- Name: PRODUCES; Type: TABLE; Schema: rif; Owner: -
--

CREATE TABLE rif."PRODUCES" (
)
INHERITS (rif._ag_label_edge);


--
-- Name: PRODUCES_id_seq; Type: SEQUENCE; Schema: rif; Owner: -
--

CREATE SEQUENCE rif."PRODUCES_id_seq"
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 281474976710655
    CACHE 1;


--
-- Name: PRODUCES_id_seq; Type: SEQUENCE OWNED BY; Schema: rif; Owner: -
--

ALTER SEQUENCE rif."PRODUCES_id_seq" OWNED BY rif."PRODUCES".id;


--
-- Name: Record; Type: TABLE; Schema: rif; Owner: -
--

CREATE TABLE rif."Record" (
)
INHERITS (rif._ag_label_vertex);


--
-- Name: Record_id_seq; Type: SEQUENCE; Schema: rif; Owner: -
--

CREATE SEQUENCE rif."Record_id_seq"
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 281474976710655
    CACHE 1;


--
-- Name: Record_id_seq; Type: SEQUENCE OWNED BY; Schema: rif; Owner: -
--

ALTER SEQUENCE rif."Record_id_seq" OWNED BY rif."Record".id;


--
-- Name: SAME_FILE_CALLS; Type: TABLE; Schema: rif; Owner: -
--

CREATE TABLE rif."SAME_FILE_CALLS" (
)
INHERITS (rif._ag_label_edge);


--
-- Name: SAME_FILE_CALLS_id_seq; Type: SEQUENCE; Schema: rif; Owner: -
--

CREATE SEQUENCE rif."SAME_FILE_CALLS_id_seq"
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 281474976710655
    CACHE 1;


--
-- Name: SAME_FILE_CALLS_id_seq; Type: SEQUENCE OWNED BY; Schema: rif; Owner: -
--

ALTER SEQUENCE rif."SAME_FILE_CALLS_id_seq" OWNED BY rif."SAME_FILE_CALLS".id;


--
-- Name: _ag_label_edge_id_seq; Type: SEQUENCE; Schema: rif; Owner: -
--

CREATE SEQUENCE rif._ag_label_edge_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 281474976710655
    CACHE 1;


--
-- Name: _ag_label_edge_id_seq; Type: SEQUENCE OWNED BY; Schema: rif; Owner: -
--

ALTER SEQUENCE rif._ag_label_edge_id_seq OWNED BY rif._ag_label_edge.id;


--
-- Name: _ag_label_vertex_id_seq; Type: SEQUENCE; Schema: rif; Owner: -
--

CREATE SEQUENCE rif._ag_label_vertex_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 281474976710655
    CACHE 1;


--
-- Name: _ag_label_vertex_id_seq; Type: SEQUENCE OWNED BY; Schema: rif; Owner: -
--

ALTER SEQUENCE rif._ag_label_vertex_id_seq OWNED BY rif._ag_label_vertex.id;


--
-- Name: _label_id_seq; Type: SEQUENCE; Schema: rif; Owner: -
--

CREATE SEQUENCE rif._label_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 65535
    CACHE 1
    CYCLE;


--
-- Name: ADVISES id; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."ADVISES" ALTER COLUMN id SET DEFAULT ag_catalog._graphid((ag_catalog._label_id('rif'::name, 'ADVISES'::name))::integer, nextval('rif."ADVISES_id_seq"'::regclass));


--
-- Name: ADVISES properties; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."ADVISES" ALTER COLUMN properties SET DEFAULT ag_catalog.agtype_build_map();


--
-- Name: CALLS_REST id; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."CALLS_REST" ALTER COLUMN id SET DEFAULT ag_catalog._graphid((ag_catalog._label_id('rif'::name, 'CALLS_REST'::name))::integer, nextval('rif."CALLS_REST_id_seq"'::regclass));


--
-- Name: CALLS_REST properties; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."CALLS_REST" ALTER COLUMN properties SET DEFAULT ag_catalog.agtype_build_map();


--
-- Name: CALLS_SOAP id; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."CALLS_SOAP" ALTER COLUMN id SET DEFAULT ag_catalog._graphid((ag_catalog._label_id('rif'::name, 'CALLS_SOAP'::name))::integer, nextval('rif."CALLS_SOAP_id_seq"'::regclass));


--
-- Name: CALLS_SOAP properties; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."CALLS_SOAP" ALTER COLUMN properties SET DEFAULT ag_catalog.agtype_build_map();


--
-- Name: Class id; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."Class" ALTER COLUMN id SET DEFAULT ag_catalog._graphid((ag_catalog._label_id('rif'::name, 'Class'::name))::integer, nextval('rif."Class_id_seq"'::regclass));


--
-- Name: Class properties; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."Class" ALTER COLUMN properties SET DEFAULT ag_catalog.agtype_build_map();


--
-- Name: Constructor id; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."Constructor" ALTER COLUMN id SET DEFAULT ag_catalog._graphid((ag_catalog._label_id('rif'::name, 'Constructor'::name))::integer, nextval('rif."Constructor_id_seq"'::regclass));


--
-- Name: Constructor properties; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."Constructor" ALTER COLUMN properties SET DEFAULT ag_catalog.agtype_build_map();


--
-- Name: DECLARES_FIELD id; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."DECLARES_FIELD" ALTER COLUMN id SET DEFAULT ag_catalog._graphid((ag_catalog._label_id('rif'::name, 'DECLARES_FIELD'::name))::integer, nextval('rif."DECLARES_FIELD_id_seq"'::regclass));


--
-- Name: DECLARES_FIELD properties; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."DECLARES_FIELD" ALTER COLUMN properties SET DEFAULT ag_catalog.agtype_build_map();


--
-- Name: EXTENDS id; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."EXTENDS" ALTER COLUMN id SET DEFAULT ag_catalog._graphid((ag_catalog._label_id('rif'::name, 'EXTENDS'::name))::integer, nextval('rif."EXTENDS_id_seq"'::regclass));


--
-- Name: EXTENDS properties; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."EXTENDS" ALTER COLUMN properties SET DEFAULT ag_catalog.agtype_build_map();


--
-- Name: Enum id; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."Enum" ALTER COLUMN id SET DEFAULT ag_catalog._graphid((ag_catalog._label_id('rif'::name, 'Enum'::name))::integer, nextval('rif."Enum_id_seq"'::regclass));


--
-- Name: Enum properties; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."Enum" ALTER COLUMN properties SET DEFAULT ag_catalog.agtype_build_map();


--
-- Name: Field id; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."Field" ALTER COLUMN id SET DEFAULT ag_catalog._graphid((ag_catalog._label_id('rif'::name, 'Field'::name))::integer, nextval('rif."Field_id_seq"'::regclass));


--
-- Name: Field properties; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."Field" ALTER COLUMN properties SET DEFAULT ag_catalog.agtype_build_map();


--
-- Name: File id; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."File" ALTER COLUMN id SET DEFAULT ag_catalog._graphid((ag_catalog._label_id('rif'::name, 'File'::name))::integer, nextval('rif."File_id_seq"'::regclass));


--
-- Name: File properties; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."File" ALTER COLUMN properties SET DEFAULT ag_catalog.agtype_build_map();


--
-- Name: IMPLEMENTS id; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."IMPLEMENTS" ALTER COLUMN id SET DEFAULT ag_catalog._graphid((ag_catalog._label_id('rif'::name, 'IMPLEMENTS'::name))::integer, nextval('rif."IMPLEMENTS_id_seq"'::regclass));


--
-- Name: IMPLEMENTS properties; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."IMPLEMENTS" ALTER COLUMN properties SET DEFAULT ag_catalog.agtype_build_map();


--
-- Name: IMPORTS id; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."IMPORTS" ALTER COLUMN id SET DEFAULT ag_catalog._graphid((ag_catalog._label_id('rif'::name, 'IMPORTS'::name))::integer, nextval('rif."IMPORTS_id_seq"'::regclass));


--
-- Name: IMPORTS properties; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."IMPORTS" ALTER COLUMN properties SET DEFAULT ag_catalog.agtype_build_map();


--
-- Name: INJECTS id; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."INJECTS" ALTER COLUMN id SET DEFAULT ag_catalog._graphid((ag_catalog._label_id('rif'::name, 'INJECTS'::name))::integer, nextval('rif."INJECTS_id_seq"'::regclass));


--
-- Name: INJECTS properties; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."INJECTS" ALTER COLUMN properties SET DEFAULT ag_catalog.agtype_build_map();


--
-- Name: Interface id; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."Interface" ALTER COLUMN id SET DEFAULT ag_catalog._graphid((ag_catalog._label_id('rif'::name, 'Interface'::name))::integer, nextval('rif."Interface_id_seq"'::regclass));


--
-- Name: Interface properties; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."Interface" ALTER COLUMN properties SET DEFAULT ag_catalog.agtype_build_map();


--
-- Name: Method id; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."Method" ALTER COLUMN id SET DEFAULT ag_catalog._graphid((ag_catalog._label_id('rif'::name, 'Method'::name))::integer, nextval('rif."Method_id_seq"'::regclass));


--
-- Name: Method properties; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."Method" ALTER COLUMN properties SET DEFAULT ag_catalog.agtype_build_map();


--
-- Name: PRODUCES id; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."PRODUCES" ALTER COLUMN id SET DEFAULT ag_catalog._graphid((ag_catalog._label_id('rif'::name, 'PRODUCES'::name))::integer, nextval('rif."PRODUCES_id_seq"'::regclass));


--
-- Name: PRODUCES properties; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."PRODUCES" ALTER COLUMN properties SET DEFAULT ag_catalog.agtype_build_map();


--
-- Name: Record id; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."Record" ALTER COLUMN id SET DEFAULT ag_catalog._graphid((ag_catalog._label_id('rif'::name, 'Record'::name))::integer, nextval('rif."Record_id_seq"'::regclass));


--
-- Name: Record properties; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."Record" ALTER COLUMN properties SET DEFAULT ag_catalog.agtype_build_map();


--
-- Name: SAME_FILE_CALLS id; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."SAME_FILE_CALLS" ALTER COLUMN id SET DEFAULT ag_catalog._graphid((ag_catalog._label_id('rif'::name, 'SAME_FILE_CALLS'::name))::integer, nextval('rif."SAME_FILE_CALLS_id_seq"'::regclass));


--
-- Name: SAME_FILE_CALLS properties; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif."SAME_FILE_CALLS" ALTER COLUMN properties SET DEFAULT ag_catalog.agtype_build_map();


--
-- Name: _ag_label_edge id; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif._ag_label_edge ALTER COLUMN id SET DEFAULT ag_catalog._graphid((ag_catalog._label_id('rif'::name, '_ag_label_edge'::name))::integer, nextval('rif._ag_label_edge_id_seq'::regclass));


--
-- Name: _ag_label_vertex id; Type: DEFAULT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif._ag_label_vertex ALTER COLUMN id SET DEFAULT ag_catalog._graphid((ag_catalog._label_id('rif'::name, '_ag_label_vertex'::name))::integer, nextval('rif._ag_label_vertex_id_seq'::regclass));


--
-- Name: _ag_label_edge _ag_label_edge_pkey; Type: CONSTRAINT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif._ag_label_edge
    ADD CONSTRAINT _ag_label_edge_pkey PRIMARY KEY (id);


--
-- Name: _ag_label_vertex _ag_label_vertex_pkey; Type: CONSTRAINT; Schema: rif; Owner: -
--

ALTER TABLE ONLY rif._ag_label_vertex
    ADD CONSTRAINT _ag_label_vertex_pkey PRIMARY KEY (id);


--
-- Name: idx_rif_advises_fwd; Type: INDEX; Schema: rif; Owner: -
--

CREATE INDEX idx_rif_advises_fwd ON rif."ADVISES" USING btree (start_id, end_id);


--
-- Name: idx_rif_advises_rev; Type: INDEX; Schema: rif; Owner: -
--

CREATE INDEX idx_rif_advises_rev ON rif."ADVISES" USING btree (end_id);


--
-- Name: idx_rif_calls_rest_fwd; Type: INDEX; Schema: rif; Owner: -
--

CREATE INDEX idx_rif_calls_rest_fwd ON rif."CALLS_REST" USING btree (start_id, end_id);


--
-- Name: idx_rif_calls_rest_rev; Type: INDEX; Schema: rif; Owner: -
--

CREATE INDEX idx_rif_calls_rest_rev ON rif."CALLS_REST" USING btree (end_id);


--
-- Name: idx_rif_calls_soap_fwd; Type: INDEX; Schema: rif; Owner: -
--

CREATE INDEX idx_rif_calls_soap_fwd ON rif."CALLS_SOAP" USING btree (start_id, end_id);


--
-- Name: idx_rif_calls_soap_rev; Type: INDEX; Schema: rif; Owner: -
--

CREATE INDEX idx_rif_calls_soap_rev ON rif."CALLS_SOAP" USING btree (end_id);


--
-- Name: idx_rif_df_fwd; Type: INDEX; Schema: rif; Owner: -
--

CREATE INDEX idx_rif_df_fwd ON rif."DECLARES_FIELD" USING btree (start_id, end_id);


--
-- Name: idx_rif_df_rev; Type: INDEX; Schema: rif; Owner: -
--

CREATE INDEX idx_rif_df_rev ON rif."DECLARES_FIELD" USING btree (end_id);


--
-- Name: idx_rif_extends_fwd; Type: INDEX; Schema: rif; Owner: -
--

CREATE INDEX idx_rif_extends_fwd ON rif."EXTENDS" USING btree (start_id, end_id);


--
-- Name: idx_rif_extends_rev; Type: INDEX; Schema: rif; Owner: -
--

CREATE INDEX idx_rif_extends_rev ON rif."EXTENDS" USING btree (end_id);


--
-- Name: idx_rif_implements_fwd; Type: INDEX; Schema: rif; Owner: -
--

CREATE INDEX idx_rif_implements_fwd ON rif."IMPLEMENTS" USING btree (start_id, end_id);


--
-- Name: idx_rif_implements_rev; Type: INDEX; Schema: rif; Owner: -
--

CREATE INDEX idx_rif_implements_rev ON rif."IMPLEMENTS" USING btree (end_id);


--
-- Name: idx_rif_imports_fwd; Type: INDEX; Schema: rif; Owner: -
--

CREATE INDEX idx_rif_imports_fwd ON rif."IMPORTS" USING btree (start_id, end_id);


--
-- Name: idx_rif_imports_rev; Type: INDEX; Schema: rif; Owner: -
--

CREATE INDEX idx_rif_imports_rev ON rif."IMPORTS" USING btree (end_id);


--
-- Name: idx_rif_injects_fwd; Type: INDEX; Schema: rif; Owner: -
--

CREATE INDEX idx_rif_injects_fwd ON rif."INJECTS" USING btree (start_id, end_id);


--
-- Name: idx_rif_injects_rev; Type: INDEX; Schema: rif; Owner: -
--

CREATE INDEX idx_rif_injects_rev ON rif."INJECTS" USING btree (end_id);


--
-- Name: idx_rif_produces_fwd; Type: INDEX; Schema: rif; Owner: -
--

CREATE INDEX idx_rif_produces_fwd ON rif."PRODUCES" USING btree (start_id, end_id);


--
-- Name: idx_rif_produces_rev; Type: INDEX; Schema: rif; Owner: -
--

CREATE INDEX idx_rif_produces_rev ON rif."PRODUCES" USING btree (end_id);


--
-- Name: idx_rif_sfc_fwd; Type: INDEX; Schema: rif; Owner: -
--

CREATE INDEX idx_rif_sfc_fwd ON rif."SAME_FILE_CALLS" USING btree (start_id, end_id);


--
-- Name: idx_rif_sfc_rev; Type: INDEX; Schema: rif; Owner: -
--

CREATE INDEX idx_rif_sfc_rev ON rif."SAME_FILE_CALLS" USING btree (end_id);


--
-- PostgreSQL database dump complete
--

\unrestrict AKVfibxLilVEeAj1t7ghITfvrOBdzGmQyXygasVrq1D0APM5MQT2krvtNJz3T7n

