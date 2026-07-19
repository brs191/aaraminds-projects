# ADR-009: Ingestion Orchestration and Promotion Guard

**Date:** 2026-07-13  
**Status:** Accepted for P0 exit  
**Owners:** Engineering + Platform  
**Related docs:** `action_plan.md`, `code/libs/ingestionruns`, `code/migrations/001_dif_meta_initial.sql`

---

## 1. Context

Ingestion turns source files into documents, nodes, edges, source anchors, retrieval passages, caveats, and future code-entity candidates. P0 must prevent empty or failed extraction output from replacing a healthy serving index.

---

## 2. Decision

DIF tracks ingestion as explicit runs with statuses:

- `running`
- `completed`
- `failed`
- `cancelled`

A run can be promoted only when it is completed and has nonzero documents, nodes, source anchors, and retrieval passages. Degenerate output is persisted as evidence but cannot promote.

Promotion decisions must be deterministic and should be atomic with optimistic locking when wired into service persistence.

---

## 3. Consequences

- Failed/running/cancelled runs cannot promote.
- Completed runs with zero documents, nodes, anchors, or passages cannot promote.
- Future connector retries/checkpoints must preserve run identity and idempotency.
- Service entry points must integrate the guard before production ingestion.

---

## 4. P0 evidence

- `code/libs/ingestionruns`
- `code/migrations/001_dif_meta_initial.sql`
- `evaluation/degenerate_run_checks.py`
- `evaluation/run_p0.py`
