# Architecture Overview

This synthetic fixture describes a small service used only for DIF P0 evaluation.

## Ownership

The architecture service is owned by Platform Architecture.
The owning group reviews ingestion, retrieval, and source-anchor behavior.
Production rollout requires the owner to confirm the corpus is uniformly readable.

## Runtime

The service stores document metadata in `dif_meta`.
Retrieval responses must cite source anchors.
Unsupported claims must return insufficient evidence instead of a fabricated answer.

