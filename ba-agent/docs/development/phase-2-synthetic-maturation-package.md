# BA Agent Phase 2 Synthetic-First Maturation Package

This package defines the next synthetic-only work after the `P2-G5` Continue decision. It is a closure and maturation plan for the first Phase 2 requirement-discovery slice; it does not authorize sandbox, live, pilot, production, non-synthetic data, or external write-like behavior.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Phase 2 Synthetic-First Maturation Package |
| Version | 1.1 |
| Change note (v1.1) | Completed the minimum synthetic GTS-P2-REQ case set by adding executable P2REQ-002 and P2REQ-005 through P2REQ-008 coverage plus the synthetic completion package. |
| Change note (v1.0) | Added executable P2REQ-004 CC-RIF-inspired missing-business-rule coverage and updated the synthetic case inventory. |
| Change note (v0.9) | Added executable P2REQ-003 conflict-case coverage and updated the synthetic case inventory. |
| Change note (v0.8) | Closed P2-MAT-004/005/006 as blocked by recording the tool/data/artifact backlog and separate authorization evidence rule. |
| Change note (v0.7) | Closed P2-MAT-003 by recording non-hard BA-EM metrics as informational while preserving BA-EM-005 and BA-EM-009 hard gates at zero. |
| Change note (v0.6) | Closed P2-MAT-EXIT-003 by recording the synthetic case inventory review and retaining P2REQ-002 through P2REQ-008 as expansion backlog. |
| Change note (v0.5) | Closed P2-MAT-008 by approving rollback triggers and documentation-control operations for synthetic-only maturation. |
| Change note (v0.4) | Closed P2-MAT-007 by approving the first-slice architecture delta for synthetic-only maturation. |
| Change note (v0.3) | Closed P2-MAT-002 by approving the current output contract and project-context memory schema as-is for synthetic-only maturation. |
| Change note (v0.2) | Recorded RAJA as acting owner for all Phase 2 synthetic maturation review lanes and closed P2-MAT-001 for the synthetic-only boundary. |
| Status | Synthetic-first maturation complete for minimum GTS-P2-REQ coverage; non-authorizing |
| Prepared date | 2026-07-08 |
| Accountable owner | RAJA |
| Triggering decision | `P2-G5` outcome: Continue synthetic-first only |
| Primary references | `docs/development/p2-g5-candidate-review.md`; `docs/development/p2-g4-tool-data-readiness.md`; `docs/planning/phase-2-implementation-plan.md`; `docs/planning/decision-log.md`; `docs/planning/phase-2-traceability-matrix.md`; `docs/development/p2-g1-technical-baseline.md`; `docs/development/p2-g2-synthetic-thin-slice.md`; `docs/development/p2-g3-evaluation-control-hardening.md` |
| Explicit non-authorization | No sandbox execution, live integration, pilot start, production rollout, non-synthetic data path, external publish/storage, or write-like side effect |

## 1) Current posture

`P2-G5` selected **Continue** with a synthetic-first boundary. The first Phase 2 slice can continue maturing local synthetic requirement-discovery behavior, output contracts, review readiness, and evidence controls. It cannot enable external tools, non-synthetic data, sandbox execution, live reads/writes, Teams sends, Confluence/Jira updates, approval-record creation, webhook subscriptions, or production deployment.

### Evidence baseline

| Evidence area | Current baseline | Source |
| --- | --- | --- |
| Candidate outcome | Continue synthetic-first only; sandbox/live/production remain separately authorized | `p2-g5-candidate-review.md` Section 7 |
| Hard gates | BA-EM-005 = 0 and BA-EM-009 = 0 in recorded control evidence | `p2-g5-candidate-review.md` Section 2; `p2-g3-evaluation-control-hardening.md` |
| Tool/data posture | Blocked by default; no external path enabled | `p2-g4-tool-data-readiness.md` |
| Phase 2 scope | Synthetic requirement discovery only for first slice | `phase-2-implementation-plan.md` Sections 2 and 6 |
| Traceability | First-slice requirement/control rows active | `phase-2-traceability-matrix.md` |
| Decision gaps | Delegate, contract ownership, thresholds, tool/data evidence, architecture closeout, and doc-control closeout require owner attention | `p2-g5-candidate-review.md` Section 3 |

## 2) Maturation objective

Mature the Phase 2 first slice from a proven synthetic candidate into a stronger review-ready baseline by:

1. Closing or explicitly deferring owner decisions that block broader progression.
2. Strengthening the synthetic case set and output-review evidence without adding non-synthetic inputs.
3. Making review routing, output ownership, architecture boundaries, rollback, and documentation-control responsibilities explicit.
4. Preserving blocked-by-default tool/data posture until a separate authorization package exists.

This package does not change the Phase 2 scope. Full BRD/FRD/PRD generation, final story/acceptance-criteria generation, process mapping, gap/impact analysis, HLD generation, sandbox use, and live/pilot activity remain out of scope unless RAJA records a separate scope-change decision.

## 3) Decision-gap closure plan

| Maturation ID | Related item | Required owner action | Proposed closure evidence | Status |
| --- | --- | --- | --- | --- |
| P2-MAT-001 | `P2-G5-OD-01`, `P2-DEC-004` | Name review delegates for BA SME, Product Owner, QA/evaluation, architect, security/privacy, compliance/legal, platform, and tool-owner lanes, or explicitly keep RAJA as acting owner for each lane. | Delegate roster table records RAJA as acting owner for all lanes. | Closed for synthetic maturation |
| P2-MAT-002 | `P2-G5-OD-02`, `P2-DEC-005`, `P2-DEC-006` | Approve, adjust, or defer the first-slice output contract and project-context memory schema. | Approved as-is for synthetic-only maturation. Evidence: RAJA selected "Approve as-is for synthetic-only maturation" for both the output contract and memory schema in the 2026-07-08 review. | Closed for synthetic maturation |
| P2-MAT-003 | `P2-G5-OD-03`, `P2-DEC-008` | Decide whether non-hard-gate BA-EM metrics remain informational or receive owner thresholds. | Non-hard metrics remain informational with no numeric thresholds set yet. Evidence: `docs/development/phase-2-metric-threshold-posture.md` v0.1. Hard gates remain BA-EM-005 = 0 and BA-EM-009 = 0. | Closed for synthetic maturation |
| P2-MAT-004 | `P2-G5-OD-04`, `P2-DEC-009` | Keep all external tools blocked unless owner/scope/schema/rate-limit/security evidence is complete. | Closed as blocked for synthetic-only maturation. Evidence: `docs/development/phase-2-blocked-tool-data-artifact-backlog.md` v0.1. | Closed as blocked |
| P2-MAT-005 | `P2-G5-OD-05`, `P2-DEC-010` | Keep non-synthetic data blocked unless classification, redaction, retention, residency, and allowed-data decisions are approved. | Closed as blocked for synthetic-only maturation. Evidence: `docs/development/phase-2-blocked-tool-data-artifact-backlog.md` v0.1. | Closed as blocked |
| P2-MAT-006 | `P2-G5-OD-07`, `P2-DEC-012` | Confirm artifact storage/publishing remains local-only/no-external-publish, or define an approved future policy. | Closed as blocked for external artifact paths; local synthetic artifacts remain local/test-only. Evidence: `docs/development/phase-2-blocked-tool-data-artifact-backlog.md` v0.1. | Closed as blocked |
| P2-MAT-007 | `P2-G5-OD-09`, `P2-DEC-013` | Close or defer the architecture delta approval for route isolation, scaffold, output contract, memory schema, and gateway carry-forward. | Approved for synthetic-only maturation. Evidence: RAJA selected "Close as approved for synthetic-only maturation" in the 2026-07-08 review; architecture basis remains `p2-g1-technical-baseline.md` v0.1 plus P2-MAT-002 contract/schema approval. | Closed for synthetic maturation |
| P2-MAT-008 | `P2-G5-OD-08`, `P2-DEC-015` | Confirm rollback triggers and documentation-control operations for continued synthetic maturation. | Approved for synthetic-only maturation. Evidence: RAJA selected "Close as approved for synthetic-only maturation" in the 2026-07-08 review; accepted P2-RB-001 through P2-RB-004 triggers, route-disable/revert/preserve-evidence procedure, and decision-log/traceability update rules. | Closed for synthetic maturation |
| P2-MAT-009 | `P2-MAT-EXIT-003`, `P2-DEC-007` | Review synthetic case inventory and first-slice evaluation coverage. | Inventory reviewed in `docs/development/phase-2-synthetic-case-inventory.md` v0.4. `P2REQ-001` through `P2REQ-008` are executable synthetic fixtures with aggregate `GTS-P2-REQ` eval coverage. | Closed for synthetic completion |

## 4) Synthetic maturation workstreams

| Workstream | Purpose | Allowed work | Not allowed | Exit evidence |
| --- | --- | --- | --- | --- |
| WS-1 Review routing | Make human-review lanes operationally clear. | Define delegate roster, review checklist, and escalation lane for unresolved questions. | Do not approve requirements or assign real system scopes without owner evidence. | P2-MAT-001 closure table. |
| WS-2 Output contract hardening | Improve the draft/advisory output shape. | Refine required sections, labels, trace IDs, evidence refs, conflict handling, and non-approval statement. | Do not create final BRD/FRD/PRD, final stories, or final acceptance criteria. | P2-MAT-002 closure plus updated traceability rows if contract changes. |
| WS-3 Synthetic case expansion | Broaden deterministic coverage without real data. | Add or refine fictional GTS-P2-REQ cases for conflict, missing rules, regulatory-style summaries, product ideas, process pain, and tool-origin metadata. | Do not use real meeting notes, tickets, documents, project keys, repositories, people, tenants, or customer data. | Synthetic-case inventory and eval evidence. |
| WS-4 Metric interpretation | Make review outcomes easier to judge. | Keep BA-EM-005/009 hard gates; propose owner-threshold placeholders for evidence, structure, unsupported claims, and citation correctness. | Do not invent numeric pass thresholds as approved. | P2-MAT-003 threshold decision table. |
| WS-5 Architecture closeout | Close the first-slice architecture delta. | Confirm route isolation, `src/ba_agent/phase2/` boundaries, gateway fail-closed carry-forward, and no MVP leakage. | Do not enable new model, MCP, Graph, Jira, Confluence, Teams, or storage clients. | P2-MAT-007 architecture closeout note. |
| WS-6 Rollback/doc-control | Keep the baseline recoverable and traceable. | Define rollback triggers for synthetic-route disablement and update rules for decision log, traceability matrix, and development artifacts. | Do not claim production rollback or live kill-switch readiness. | P2-MAT-008 closeout note. |
| WS-7 Tool/data backlog | Prepare future authorization evidence without enablement. | Maintain blocked tool/data backlog and evidence checklist. | Do not mark any tool/data path enabled or validated without actual approval evidence. | P2-MAT-004/005/006 backlog decisions. |

## 5) Proposed review-delegate roster template

RAJA remains accountable until delegates are named. For the current synthetic-only maturation boundary, RAJA is explicitly recorded as acting owner for every review lane.

| Review lane | Delegate / acting owner | Decision authority in this package | Evidence needed |
| --- | --- | --- | --- |
| BA SME | RAJA acting owner | Validate requirement-discovery usefulness, question quality, and BA terminology. | Output-contract review comments. |
| Product Owner | RAJA acting owner | Confirm draft requirement/story candidates remain non-committal and aligned to first-slice scope. | Scope and prioritization review comments. |
| QA / AI evaluation | RAJA acting owner | Review GTS-P2-REQ coverage, BA-EM metric mapping, and regression expectations. | Eval summary and case inventory review. |
| Architect | RAJA acting owner | Confirm route/scaffold/gateway boundaries and no MVP leakage. | Architecture closeout note. |
| Security/privacy | RAJA acting owner | Confirm synthetic-only data posture and non-synthetic blockers. | Data-classification decision or continued-blocked note. |
| Compliance/legal | RAJA acting owner | Review regulatory-style synthetic outputs and obligation non-approval wording. | Compliance/legal review comments for synthetic cases. |
| Platform/tool owner | RAJA acting owner | Confirm all external tools remain blocked pending evidence. | Tool evidence backlog review. |
| Delivery lead | RAJA acting owner | Confirm sequencing, gate stops, and documentation-control discipline. | Maturation exit checklist review. |

## 6) Synthetic-only guardrails

The following guardrails are mandatory for every maturation workstream:

1. Use fictional fixtures and synthetic evidence refs only.
2. Preserve `[inferred]` for unsupported but reasonable implementation conclusions.
3. Preserve `[RAJA]` for owner-dependent values, names, thresholds, decisions, scopes, dates, and approvals.
4. Keep all outputs labeled synthetic, draft, advisory, and non-approving.
5. Keep BA-EM-005 and BA-EM-009 as zero-tolerance hard gates.
6. Treat every external side effect as write-like, including sends, drafts, comments, publishes, approval records, and webhook subscriptions.
7. Keep Teams/Copilot 365 as the collaboration surface convention; do not introduce Slack.
8. If registry or artifact promotion is discussed later, keep the JFrog convention; do not introduce Azure ACR.

## 7) Maturation exit checklist

The synthetic maturation package is ready for RAJA review when all items below are either closed or explicitly deferred:

| Checklist ID | Exit criterion | Status |
| --- | --- | --- |
| P2-MAT-EXIT-001 | Review-delegate roster completed or RAJA acting-owner choices recorded. | Closed for synthetic maturation |
| P2-MAT-EXIT-002 | Output contract and project-context memory schema approved, adjusted, or deferred with rationale. | Closed for synthetic maturation |
| P2-MAT-EXIT-003 | Synthetic case inventory and evaluation coverage reviewed for first-slice adequacy. | Closed for synthetic completion; `P2REQ-001` through `P2REQ-008` are executable |
| P2-MAT-EXIT-004 | Non-hard-gate BA-EM threshold posture recorded. | Closed for synthetic maturation |
| P2-MAT-EXIT-005 | Architecture delta closeout recorded for `P2-DEC-013`. | Closed for synthetic maturation |
| P2-MAT-EXIT-006 | Rollback/documentation-control closeout recorded for `P2-DEC-015`. | Closed for synthetic maturation |
| P2-MAT-EXIT-007 | Tool/data/artifact paths remain blocked or have separate explicit authorization evidence. | Closed as blocked; separate authorization required before enablement |
| P2-MAT-EXIT-008 | Decision log and traceability matrix are updated if any decision state, output contract, or trace mapping changes. | Active control; update when triggered |

## 8) Recommended next execution order

1. Maintain `P2REQ-001` through `P2REQ-008` as the minimum synthetic regression set.
2. Use `docs/development/phase-2-synthetic-completion-package.md` as the synthetic-first closure artifact.
3. Prepare a separate authorization package if RAJA wants any sandbox, non-synthetic, external tool, or artifact-publishing path.

Do not start sandbox, pilot, live, production, or non-synthetic execution from this package.
