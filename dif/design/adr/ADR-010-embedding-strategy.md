# ADR-010: Embedding Strategy and Provider Seam

**Date:** 2026-07-13  
**Status:** Accepted for P0 exit  
**Owners:** Engineering + Product  
**Related docs:** `action_plan.md`, `code/libs/embeddings`

---

## 1. Context

DIF needs an embedding abstraction for future hybrid retrieval, but P0 does not need to pin production model dimensions or add vector schema before the model/dimension spike exits.

---

## 2. Decision

P0 defines the embedding provider interface and uses a deterministic offline hash provider for tests.

Production embedding choices remain:

- Voyage with Matryoshka truncation to <=1024 dimensions as the accepted prose default.
- Qwen3-Embedding as the self-host/sovereignty fallback.

The exact Voyage model and dimension remain open until the P0 spike exit. P0 intentionally does not add pgvector schema or production embedding calls.

---

## 3. Consequences

- Tests stay offline and deterministic.
- Usage metering shape exists without real provider charges.
- Vector schema must be added only after dimensions are pinned.
- Retrieval remains anchored lexical in P0.

---

## 4. P0 evidence

- `code/libs/embeddings`
- `code/libs/retrieval`
- `evaluation/run_p0.py`
