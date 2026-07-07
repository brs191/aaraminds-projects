# VRIA Product Requirements Document

**Document set:** Value Realization Intelligence Agent  
**Version:** v1.2.1  
**Date:** 2026-07-05  
**Owner:** AaraMinds / Common Capabilities AI Portfolio  
**Status:** Implementation baseline  


---

## 1. Product Summary

The **Value Realization Intelligence Agent** is a portfolio governance product that helps leaders and use-case owners understand whether AI initiatives are ready to fund, pilot, scale, fix, re-baseline, defer, or stop.

The product combines a registry, evidence model, scoring workflow, dashboard, approval workflow, leadership summary generator, decision audit, and ValueOps feedback loop.

## 2. Goals

- Normalize AI use-case portfolio data.
- Capture measurable value hypotheses.
- Detect missing baselines, weak attribution, stale evidence, and net-value gaps.
- Score use cases using transparent, executable criteria.
- Generate leadership-ready value scorecards.
- Preserve evidence lineage and decision history.
- Support monthly ValueOps review.

## 3. Non-Goals

- The agent will not make funding decisions.
- The agent will not change official PTB/PTO status without approval.
- The agent will not book savings or declare financial benefits officially.
- The agent will not replace finance, product, architecture, security, or governance approval.

## 4. Personas

| Persona | Needs |
|---|---|
| Executive sponsor | Portfolio-level clarity on value, risk, and funding choices. |
| Portfolio lead | Normalized registry, scorecard, evidence gaps, and decision log. |
| Use-case owner | Clear gaps and next actions to prove value. |
| Delivery owner | Traceable status and dependency visibility. |
| Governance reviewer | Approval boundaries, policy tiers, and audit evidence. |
| Finance / FinOps owner | Baseline, cost, attribution, and net-value validation. |
| Engineering owner | Tool contracts, data model, observability, and implementation backlog. |

## 5. Functional Requirements

| ID | Requirement | Acceptance Criteria |
|---|---|---|
| FR-01 | Maintain use-case registry | All imported use cases have canonical IDs, tier, owner status, and delivery status. |
| FR-02 | Capture value hypotheses | Template fields follow `03` and schema in `17`. |
| FR-03 | Detect evidence gaps | Missing baseline/current/target/source/owner/attribution/net-value fields are surfaced. |
| FR-04 | Score value realization | Scoring follows `20`; no ad-hoc score logic. |
| FR-05 | Classify value state | State uses canonical enum: NotReady, HypothesisOnly, BaselineReady, OnTrack, AtRisk, Realized, NotRealized, Regressed, Unproven. |
| FR-06 | Draft recommendations | Recommendation uses canonical enum and includes rationale, evidence, confidence, and owner action. |
| FR-07 | Generate scorecards | Scorecards include evidence coverage, caveats, approval state, and decision log pointer. |
| FR-08 | Manage approvals | Publishing, status changes, realized-value declarations, and follow-up actions use `18` workflow. |
| FR-09 | Preserve audit | Every assessment, tool call, approval, and published scorecard is auditable. |
| FR-10 | Support ValueOps loop | Production telemetry, owner feedback, online evals, and value evidence update backlog/evals/model. |

## 6. Value States

Authoritative enum is in `17`. Product behavior:

| State | Meaning |
|---|---|
| NotReady | Missing required owner, scope, metric, or governance information. |
| HypothesisOnly | Expected benefit exists, but evidence is insufficient. |
| BaselineReady | Baseline and target exist, but current value is not yet proven. |
| OnTrack | Evidence suggests progress toward target. |
| AtRisk | Delivery, evidence, value, or governance risk threatens realization. |
| Realized | Evidence-backed, attributed, net-positive or non-financial benefit approved for reporting. |
| NotRealized | Evidence shows target was not met. |
| Regressed | Previously realized value failed two consecutive sustainment checks (threshold and cadence defined in `contracts/20_VRIA_Scoring_Rules_Spec.md` section 7). |
| Unproven | Claim cannot be substantiated from available evidence. |

## 7. Non-Functional Requirements

Sized for an internal portfolio tool (~20–200 use cases, ~10–50 users, monthly ValueOps cadence).

| ID | Requirement | Target |
|---|---|---|
| NFR-01 | Availability (business hours, dashboard + API) | 99.5% monthly `[VERIFY]` |
| NFR-02 | Dashboard read latency | p95 < 2 s |
| NFR-03 | Draft assessment generation | p95 < 30 s `[VERIFY]` |
| NFR-04 | Approval decision write | p95 < 1 s |
| NFR-05 | Concurrency | 50 concurrent users without degradation |
| NFR-06 | RPO / RTO | 24 h / 8 h `[VERIFY with ops owner]` |
| NFR-07 | Retention — audit events, decision log | 7 years `[VERIFY with governance]` |
| NFR-08 | Retention — operational records | 2 years, then archive |
| NFR-09 | Accessibility | WCAG 2.1 AA |
| NFR-10 | Audit query (by target or trace ID) | p95 < 5 s |

## 8. Release Criteria

The MVP cannot be released unless:

- Golden eval release gate passes.
- Tool contracts are implemented or explicitly stubbed with failure behavior.
- Approval workflow is implemented for scorecard publication.
- Scoring rules are executable and tested.
- Data model supports immutable audit.
- Production monitoring is enabled for tool failures, scoring failures, and approval bypass attempts.
