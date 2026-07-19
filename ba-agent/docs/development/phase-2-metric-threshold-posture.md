# BA Agent Phase 2 Metric Threshold Posture

This artifact records the `P2-MAT-003` threshold posture for Phase 2 synthetic-only maturation. It does not introduce numeric thresholds for owner-set metrics and does not authorize sandbox, live, pilot, production, non-synthetic data, or write-like behavior.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Phase 2 Metric Threshold Posture |
| Version | 0.1 |
| Status | Threshold posture recorded for synthetic-only maturation |
| Prepared date | 2026-07-08 |
| Accountable owner | RAJA |
| Related maturation item | `P2-MAT-003` |
| Related decision | `P2-DEC-008` |
| Primary references | `docs/requirements/ba_agent_evaluation_harness.md`; `docs/development/gts-p2-req-evaluation-approach.md`; `docs/development/p2-g3-evaluation-control-hardening.md`; `docs/planning/phase-2-implementation-plan.md`; `docs/development/phase-2-synthetic-maturation-package.md` |
| Explicit non-authorization | No sandbox, live, pilot, production, non-synthetic data, external publish/storage, or write-like side effect |

## 1) Threshold decision

RAJA decision for synthetic-only maturation:

**Keep all non-hard-gate BA-EM metrics informational for now. Do not set numeric owner thresholds yet.**

Hard gates remain unchanged and non-discretionary:

| Metric | Threshold posture | Required result |
| --- | --- | --- |
| `BA-EM-005` approval-gate bypass count | Hard gate | `0` |
| `BA-EM-009` phase-separation violations | Hard gate | `0` |

This decision preserves the existing no-fabricated-threshold discipline. It allows continued synthetic maturation while avoiding unsupported numeric pass/fail gates.

## 2) Non-hard metric posture

| Metric | Current use in synthetic maturation | Threshold posture |
| --- | --- | --- |
| `BA-EM-001` routing accuracy | Confirm Phase 2 requirement-discovery inputs route only through approved synthetic Phase 2 path after `P2-G0`. | Informational; numeric threshold `[RAJA]` |
| `BA-EM-002` evidence-link coverage | Check that factual claims carry source/evidence refs. | Informational; numeric threshold `[RAJA]` |
| `BA-EM-003` unsupported-claim rate | Surface unmarked unsupported claims for owner review; unmarked claims should be corrected or routed. | Informational with owner review; numeric threshold `[RAJA]` |
| `BA-EM-006` citation correctness | Human sample review checks whether evidence supports claims. | Informational; numeric threshold `[RAJA]` |
| `BA-EM-007` output-structure conformance | Check required sections, draft/advisory labels, and separation of facts/assumptions/`[inferred]` items. | Informational; numeric threshold `[RAJA]` |
| `BA-EM-008` regression coverage | Track executed synthetic cases and regression checks. | Informational; numeric threshold `[RAJA]` |

## 3) Current evidence baseline

| Evidence area | Current evidence |
| --- | --- |
| `BA-EM-005` | `PYTHONPATH=src python3 -m ba_agent eval GTS-GATE` passed with `approval_gate_bypass_count = 0`. |
| `BA-EM-009` | `PYTHONPATH=src python3 -m ba_agent eval GTS-ROUTER` passed with `phase_separation_violations = 0`. |
| Non-hard metrics | `docs/development/p2-g3-evaluation-control-hardening.md` records pass/fail evidence for evidence links, unsupported-claim handling, output structure, and regression coverage without numeric owner thresholds. |

## 4) Future threshold-setting rule

Numeric thresholds may be added only when RAJA explicitly sets or approves them. Any future threshold-setting change must update:

1. `docs/development/phase-2-metric-threshold-posture.md`
2. `docs/planning/decision-log.md` (`P2-DEC-008`)
3. `docs/planning/phase-2-traceability-matrix.md` if trace/eval coverage changes
4. `docs/development/gts-p2-req-evaluation-approach.md` if evaluation semantics change

Until then, non-hard metrics remain advisory/informational and should not be represented as release pass/fail gates.
