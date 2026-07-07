# VRIA Portfolio Intake Model

**Document set:** Value Realization Intelligence Agent  
**Version:** v1.2  
**Date:** 2026-07-05  
**Owner:** AaraMinds / Common Capabilities AI Portfolio  
**Status:** Implementation baseline  


---

## 1. Purpose

This document defines how AI use cases are captured, normalized, tiered, prioritized, and selected for MVP/pilot review by the Value Realization Intelligence Agent.

All enums and schemas referenced here are authoritative in `contracts/17_VRIA_Canonical_Schemas_and_Enums.md`.

## 2. Intake Sources

| Source | Purpose | Authority | Record Type |
|---|---|---|---|
| Ideas.md / portfolio spreadsheet | Initial use-case inventory | Portfolio lead | Read-only import |
| Use-case BRDs / PRDs | Scope and expected outcomes | Use-case owner / product owner | Evidence document |
| PTB/PTO tracker | Delivery and compliance status | Governance / platform process owner | Read-only status |
| Jira / ADO | Delivery progress and backlog evidence | Delivery team | Read-only status, approval-gated task creation |
| Metrics systems | Baseline and current values | Metric owner | Read-only metric snapshot |
| Cost systems | Cost/savings/initiative effort | Finance / FinOps owner | Read-only metric snapshot |
| Leadership decks | Narrative and decision history | Portfolio lead | Evidence document |

## 3. Intake Principles

1. Intake is not approval.
2. A use case without owner is **NotReady**.
3. A use case without metric is **HypothesisOnly** at best.
4. A use case without baseline cannot report realized value.
5. A financial/productivity value claim requires initiative cost and net-value check.
6. Conflicting source data must be surfaced, not silently merged.
7. Employer-specific inventory remains only in `internal/99_Source_AI_Use_Case_Inventory.md`.

## 4. Normalized Intake Fields

| Field | Required | Source | Notes |
|---|---:|---|---|
| `use_case_id` | Yes | Registry/import | Stable ID. |
| `name` | Yes | Registry/import | Human-readable name. |
| `tier` | Yes | Portfolio lead | Tool, Agent, Layer, Unclassified. |
| `domain` | Yes | Portfolio lead | Business or technical domain. |
| `value_owner` | Yes for pilot | Use-case owner | Accountable for value, not only delivery. |
| `delivery_owner` | Yes | Delivery team | Accountable for execution. |
| `sponsor` | Required before Scale | Leadership | Named sponsor. |
| `delivery_status` | Yes | PTB/PTO/Jira/ADO | Canonical delivery status enum. |
| `value_hypothesis_id` | Required for Gate A exit | Template | Links to value hypothesis. |
| `primary_metric_id` | Required for pilot | Metric owner | Links to baseline/current/target. |
| `approval_state` | Yes | Approval workflow | Draft, Submitted, Approved, etc. |

## 5. Tiering Rules

| Tier | Definition | Example Value Focus | Commitment |
|---|---|---|---|
| Tool | Exposes one system capability to AI or automation | MTTR, query time, analyst time | Low |
| Agent | Automates a known multi-step workflow | Cycle time, quality, rework, effort | Medium |
| Layer | Creates intelligence layer over fragmented domain | Revenue, risk, accuracy, decision quality | High |
| Unclassified | Not enough information | Needs triage | Unknown |

## 6. Gate A Readiness Questions

For each use case, VRIA asks:

1. What business outcome does this support?
2. Who owns value realization?
3. What metric proves success?
4. What is the baseline?
5. What is the target?
6. What is the current evidence source?
7. What is the reporting period?
8. What is the attribution method?
9. What known confounders may distort value?
10. What cost/effort is required to calculate net value?
11. What approval is needed before status, funding, or published scorecard changes?

## 7. Pilot Candidate Selection

A use case is a strong pilot candidate when:

- Owner is named.
- Metric is measurable.
- Baseline is available or can be established within two weeks.
- Evidence source is authoritative.
- Tool/data access is feasible.
- Recommendation can influence a real decision.
- Risk is manageable under Level 2 Drafting autonomy.

## 8. Intake Output

The intake workflow produces:

```json
{
  "use_case_id": "UC-XXXX",
  "tier": "Agent",
  "delivery_status": "PTB_InProgress",
  "intake_readiness_score": 78,
  "value_state": "BaselineReady",
  "missing_fields": ["attribution_method"],
  "recommended_next_action": "NeedsEvidence",
  "approval_state": "Draft"
}
```

## 9. Gate A Exit Criteria

Gate A is passed only when:

- Use case exists in registry.
- Tier is assigned.
- Owner is identified.
- Value hypothesis exists.
- Metric and baseline status are known.
- Evidence source is identified.
- Approval boundary is recorded.
