# BA Agent Phase 2 Traceability Matrix

Traceability baseline for the first Phase 2 synthetic requirement-discovery slice.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Phase 2 Traceability Matrix |
| Version | 0.6 |
| Change note (v0.6) | Added HLD lane traceability rows for draft/advisory architecture creation after RAJA scope-change decision. |
| Change note (v0.5) | Recorded `P2REQ-001` through `P2REQ-008` as executable, completing minimum synthetic GTS-P2-REQ coverage. |
| Change note (v0.4) | Recorded `P2REQ-004` as executable missing-business-rule coverage; remaining referenced cases stay planned until fixture/test expansion. |
| Change note (v0.3) | Recorded `P2REQ-003` as executable conflict-case coverage; remaining referenced cases stay planned until fixture/test expansion. |
| Status | Execution baseline with complete minimum synthetic GTS-P2-REQ coverage and active HLD traceability addendum |
| Prepared date | 2026-07-06 |
| Accountable owner | RAJA |
| Plan reference | `docs/planning/phase-2-implementation-plan.md` v0.4; `docs/planning/phase-2-hld-creation-plan.md` v0.1 |
| Requirement baseline | `docs/requirements/business-analyst-agent-requirements.md` v0.4 |
| Evaluation reference | `docs/development/gts-p2-req-evaluation-approach.md` |

## Current executable coverage note

`P2REQ-001` through `P2REQ-008` are executable synthetic fixtures. This is complete for the current minimum synthetic GTS-P2-REQ coverage boundary and remains non-authorizing for sandbox, live, pilot, production, non-synthetic data, external tools, or write-like side effects.

## Purpose

Keep a verifiable map from requirement intent to:

1. Phase 2 plan scope and gate controls.
2. Expected synthetic output sections.
3. GTS-P2-REQ evaluation cases.
4. Decision-log evidence references.

## First-slice traceability map

| Trace ID | Requirement / control | Plan references | Required output / behavior | Eval coverage | Decision linkage | Status |
| --- | --- | --- | --- | --- | --- | --- |
| P2-TM-001 | `BA-P2-FR-001` requirement discovery | Plan Sections 2 and 6 | Draft/advisory requirement-discovery summary with problem/objective, facts, and evidence refs | P2REQ-001, P2REQ-006 | P2-DEC-002 | Active |
| P2-TM-002 | `BA-P2-FR-002` risk/dependency and unresolved-decision surfacing | Plan Sections 2 and 6 | Risks, dependencies, conflicts, unresolved decisions remain explicit | P2REQ-002, P2REQ-003, P2REQ-007 | P2-DEC-006 | Active |
| P2-TM-003 | `BA-P2-FR-009` clarification questions | Plan Sections 2 and 6 | Targeted open questions instead of invented rules | P2REQ-004 | P2-DEC-006 | Active |
| P2-TM-004 | `BA-P2-FR-011` traceability | Plan Section 6 (traceability skeleton) | Evidence → objective → draft requirement/story links with trace IDs | P2REQ-001 through P2REQ-008 | P2-DEC-006 | Active |
| P2-TM-005 | `BA-P2-FR-014` project-context memory | Plan Section 6 (project-context memory schema) | Stable schema with unknowns marked `[RAJA]` and no inferred business rules | P2REQ-001, P2REQ-006, P2REQ-008 | P2-DEC-005 | Active |
| P2-TM-006 | `BA-P2-FR-016` uncertainty transparency | Plan Sections 2, 6, and 9 | Clear separation of facts, assumptions, `[inferred]`, and open questions | P2REQ-003, P2REQ-004, P2REQ-005 | P2-DEC-006 | Active |
| P2-TM-007 | `BA-EM-005` hard gate | Plan Section 9 and Section 12 | No unapproved write-like side effect; fail closed on breach | `P2-G3` hard-gate checks | P2-DEC-015 | Active |
| P2-TM-008 | `BA-EM-009` hard gate | Plan Sections 3, 4, and 9 | No MVP/Phase 2 route leakage | `P2-G3` hard-gate checks | P2-DEC-015 | Active |
| P2-TM-009 | `BA-NFR-001` evidence discipline | Plan Sections 2, 6, and 9 | Facts are evidence-backed; unsupported points are explicitly separated | P2REQ-001 through P2REQ-008; BA-EM-002/003 reviews | P2-DEC-006 | Active |
| P2-TM-010 | `BA-NFR-003` uncertainty honesty | Plan Sections 2, 6, and 9 | Unknowns/conflicts/open questions remain explicit; no silent assumption promotion | P2REQ-003, P2REQ-004, P2REQ-005 | P2-DEC-006 | Active |
| P2-TM-011 | `BA-AC-PROD-001` non-approval behavior | Plan Sections 2, 6, and 9 | Outputs remain draft/advisory and include explicit non-approval statement | P2REQ-001 through P2REQ-008 | P2-DEC-006 | Active |
| P2-TM-012 | `BA-HIL-003`, `BA-HIL-004`, `BA-HIL-005` human-review routing | Plan Sections 2, 9, and 13 | Required review lanes are identified and preserved in output routing | P2REQ-003, P2REQ-004, P2REQ-005; human-review rubric checks | P2-DEC-004 | Active |
| P2-TM-013 | `BA-QG-007` synthetic eval gate | Plan Sections 2, 4, and 9 | First-slice quality gate uses synthetic GTS-P2-REQ coverage with hard-gate enforcement | P2REQ-001 through P2REQ-008; BA-EM-007/008 | P2-DEC-007 | Active |
| P2-TM-014 | HLD draft/advisory architecture artifact `[RAJA]` | HLD plan Sections 1-4 | Draft HLD is repository-evidence-only, clearly non-authoritative, and separates source-backed, `[inferred]`, and `[RAJA]` content | [P9B]/[Q9B] docs review | P2-DEC-017 | Active |
| P2-TM-015 | HLD non-authorization and control boundaries | HLD plan Sections 2-4 and 6 | HLD must not authorize sandbox/live/non-synthetic/production/write-like behavior and must preserve BA-EM-005/009 hard-gate language | [P9B]/[Q9B], [P9C]/[Q9C] | P2-DEC-016, P2-DEC-017 | Active |

## Update rules

| Trigger | Required matrix update |
| --- | --- |
| Requirement IDs added/removed from first slice | Add/remove trace rows and update plan/eval references |
| Output contract change | Update "Required output / behavior" and impacted eval cases |
| Eval case change | Update "Eval coverage" links for affected trace rows |
| Gate/decision outcome change | Update "Decision linkage" and status for impacted trace rows |

## Change log

| Version | Date | Summary |
| --- | --- | --- |
| 0.6 | 2026-07-13 | Added HLD lane traceability rows after RAJA scope-change decision. |
| 0.5 | 2026-07-09 | Recorded `P2REQ-001` through `P2REQ-008` as executable minimum synthetic GTS-P2-REQ coverage. |
| 0.4 | 2026-07-08 | Recorded `P2REQ-004` as executable missing-business-rule coverage. |
| 0.3 | 2026-07-08 | Recorded `P2REQ-003` as executable conflict-case coverage. |
| 0.2 | 2026-07-06 | Added NFR/HIL/QG control trace rows and updated plan reference to v0.3. |
| 0.1 | 2026-07-06 | Initial traceability baseline for Phase 2 first slice. |
