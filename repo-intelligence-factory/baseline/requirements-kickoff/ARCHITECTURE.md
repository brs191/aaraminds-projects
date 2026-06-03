# Repository Intelligence Platform Architecture

## High-Level Flow
GitHub Repo -> Ingestion -> Parsing -> Metadata -> Knowledge Graph -> Vector Index -> GraphRAG -> Agents -> MCP

## Core Components
- Ingestion Service
- Tree-sitter Parsing Service
- Semgrep Analysis Service
- Knowledge Graph Builder
- Embedding Service
- GraphRAG Retrieval Engine
- LangGraph Agent Layer
- MCP Server

## Storage
- PostgreSQL
- pgvector
- Neo4j / Memgraph
