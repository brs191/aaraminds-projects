# VRIA ValueOps Dashboard Specification

**Document set:** Value Realization Intelligence Agent  
**Version:** v1.2  
**Date:** 2026-07-05  
**Owner:** AaraMinds / Common Capabilities AI Portfolio  
**Status:** Implementation baseline  


---

## 1. Purpose

The ValueOps Dashboard provides portfolio-level and use-case-level views of AI value realization, evidence readiness, governance status, and decision recommendations.

## 2. Primary Views

| View | Audience | Purpose |
|---|---|---|
| Portfolio Overview | Leadership / portfolio lead | See portfolio health and value states. |
| Use-Case Detail | Use-case owner / product owner | Review score, evidence, gaps, actions. |
| Evidence Gaps | Owners / governance | See missing baseline/source/approval items. |
| Recommendations | Leadership / portfolio governance | Review build/defer/scale/stop recommendations. |
| Approval Queue | Approvers | Approve/reject/request changes. |
| Decision Log | Audit / portfolio lead | Review decision history and rationale. |
| Evaluation Health | Engineering / governance | Review eval pass rates and regressions. |

## 3. Required Portfolio Fields

- Use-case ID/name/tier/domain.
- Value owner and delivery owner.
- Delivery status.
- Value state.
- Readiness/realization score.
- Confidence.
- Recommendation.
- Evidence coverage.
- Attribution method.
- Net-value check.
- Approval state.
- Last assessment date.
- Decision log pointer.

## 4. Visual Rules

- Never show a value claim without evidence status.
- Show **Unproven** explicitly, not as blank.
- Show **Regressed** separately from AtRisk.
- Show caveats for stale evidence, unknown attribution, and missing net value.
- Badge artifact lifecycle (`ArtifactState`): Draft / Approved / Published / Superseded / Invalidated.
- Badge pending review separately from the request lifecycle (`ApprovalRequestState`): Submitted / ChangesRequested / Rejected.
