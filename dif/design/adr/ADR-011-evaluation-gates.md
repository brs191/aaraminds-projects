# ADR-011: P0 Evaluation Gates

**Date:** 2026-07-13  
**Status:** Accepted for P0 exit  
**Owners:** QA + Engineering  
**Related docs:** `evaluation/p0-evaluation-plan.md`, `evaluation/run_p0.py`, `.github/workflows/ci.yml`

---

## 1. Context

DIF needs repeatable evidence for P0 behavior before P1 federation begins. The gate must validate current deterministic behavior without inventing production SLOs.

---

## 2. Decision

P0 exit requires one local command and one CI workflow to execute the current gate set.

Local command:

```bash
python3 evaluation/run_p0.py
```

The runner executes:

1. targeted Go component tests
2. full Go tests
3. Go build
4. source-anchor round-trip harness
5. JSON caveat harness
6. RIF compatibility harness
7. `search_docs` contract harness
8. audit/usage harness
9. degenerate-run guard harness
10. path/CI baseline harness

CI runs the same P0 gate and adds PostgreSQL-backed migration idempotency.

---

## 3. Consequences

- Metrics reported by the runner are measured durations and output summaries only.
- No quality target or production SLO is implied by P0 baseline metrics.
- New P0 gates must be added to the runner and docs together.

---

## 4. P0 evidence

- `evaluation/run_p0.py`
- `evaluation/path_checks.py`
- `.github/workflows/ci.yml`
- `tracking/phase-gate-status.md`
