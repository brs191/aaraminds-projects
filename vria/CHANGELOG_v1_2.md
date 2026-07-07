# VRIA Changelog

## v1.3 implementation drop (2026-07-07)

`impl/` now covers Epics 1-5 + 7: registry (staging→promotion, real-inventory import test), hypothesis workflow (approval-gated commits), assessment generation + sustainment scheduler, scorecard lifecycle with GE-007 publication gate, MCP servers for metrics/documents (reference adapters), golden eval harness (15/15, critical gating), 62-record volume dataset (all four accuracy gates at 100%), agent system prompt v1.0, CI release-gate workflow. 10 Go packages, all tests green, zero external dependencies.

## v1.3 (2026-07-07)

Phase 0 remediation per `prompts.md` P0.1–P0.9.

1. **P0.1** GE-006, PRD §6, and `20` §5 aligned on two-consecutive-failures Regressed trigger.
2. **P0.2** Employer data scrubbed from `13` and `02`; real pilot mapping moved to `internal/99`.
3. **P0.3** Schema alignment: `initiative_cost_period` standardized as object; `evidence_source_ids`, `sustainment_threshold`, `sustainment_status` added to ValueAssessment; `assessment_evidence` join table; `approval_requests.decided_by`; canonical `Scorecard` and `DecisionRecord` schemas (17 §9–10); payloads for `get_pending_approvals`, `create_follow_up_action`, `append_decision_log`; `03` field-ownership note.
4. **P0.4** Executable component formulas added (`20` §3a) with lookup tables, metric-movement interpolation, and 3 worked examples; strategic-alignment rubric extension marked `[DECISION NEEDED]`.
5. **P0.5** Reporting-window cadence defined (`06` §8); sustainment checks schedulable.
6. **P0.6** NFR section added to `04`; matching Gate D checklist rows in `16`.
7. **P0.7** Deployment architecture + ADR-01..04 in `08`; open ORs decided (Go/Container Apps, pgvector, React-first, A2A post-MVP).
8. **P0.8** `/api/v1` conventions (Entra ID OIDC, cursor pagination); 4 missing endpoint groups; per-event payload schemas; A2A per-purpose payload schemas. No `{}` payloads remain.
9. **P0.9** Approval split into ApprovalRequestState and ArtifactState machines (`18` §2); `Rejected` documented terminal; `Invalidated` replaces Published→Withdrawn; dashboard badges updated (`15`).
10. Physical model hardening: enum CHECK constraints, `pre_cap_score` column, append-only enforcement (REVOKE + trigger), missing indexes.

### Build Readiness Notes — resolution

| v1.2 open item | Resolution |
|---|---|
| Source systems for metrics/PTB-PTO | Reference adapters first (CSV/file-drop metrics, pgvector documents) per `prompts.md` P3.1; real adapters at pilot onboarding. |
| Identity provider / RBAC | Entra ID app roles (`21` §2, `08` deployment view). |
| Dashboard technology | React (ADR-03). |
| A2A MVP or post-MVP | Post-MVP (ADR-04); envelope frozen. |
| Data/audit retention | NFR-07/08 in `04` `[VERIFY with governance]`. |


## v1.2.1 (2026-07-05)

1. Sustainment threshold defined (`20` section 7): 80% of target default, checked each freshness cycle, two consecutive failures -> Regressed. GE-006 is now testable.
2. Volume evaluation dataset reinstated (`07` section 4); golden tests tagged critical/non-critical; critical tests gate at 100%; percentage accuracy gates measured only against the volume dataset.
3. Approval-state score cap renamed publication-readiness cap; explicitly excluded from evidential-quality trending.
4. Gate A intake: verified baseline (15 pts) split from planned baseline (8 pts).
5. Typo fixes.

# VRIA v1.2 Changelog

**Document set:** Value Realization Intelligence Agent  
**Version:** v1.2  
**Date:** 2026-07-05  
**Owner:** AaraMinds / Common Capabilities AI Portfolio  
**Status:** Implementation baseline  


---

## Summary

v1.2 converts the pack from enterprise-review ready to implementation baseline by fixing version drift, centralizing schemas, defining approval as workflow, tightening tool contracts, adding physical data model, and making scoring executable.

## Key Changes

1. All documents updated to v1.2.
2. Added canonical enums and JSON schemas.
3. Added approval workflow state machine and tools.
4. Added PostgreSQL physical data model.
5. Added executable scoring rules, caps, state mapping, and recommendation mapping.
6. Added REST API and event contracts.
7. Rewrote MCP/A2A tool contracts with strict payloads and audit.
8. Propagated Regressed, net value, attribution, known confounders, initiative cost, and approval state.

## Build Readiness Notes

Remaining implementation decisions before coding:

- Confirm exact source systems for metrics and PTB/PTO status.
- Confirm identity provider and RBAC implementation.
- Confirm dashboard technology and deployment platform.
- Confirm whether A2A is MVP or post-MVP.
- Confirm data retention and audit retention requirements.
