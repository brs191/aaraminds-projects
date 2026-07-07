# VRIA Value Hypothesis Template

**Document set:** Value Realization Intelligence Agent  
**Version:** v1.2  
**Date:** 2026-07-05  
**Owner:** AaraMinds / Common Capabilities AI Portfolio  
**Status:** Implementation baseline  


---

## 1. Purpose

This template standardizes how expected value is captured for every AI use case before VRIA scores readiness or reports value.

## 2. Core Rule

> A value hypothesis is not a value claim.

Expected value remains a hypothesis until baseline, target, current metric, evidence source, attribution method, net-value check, and approval state are available.

## 3. YAML Template

```yaml
use_case_id:
use_case_name:
tier: Tool | Agent | Layer | Unclassified
domain:
value_owner:
delivery_owner:
sponsor:

business_objective:
problem_statement:
expected_benefit:
benefit_type: Cost | Productivity | CycleTime | Quality | Risk | Revenue | Adoption | DecisionQuality | Compliance
primary_metric_id:
primary_metric_name:
metric_unit:
baseline_value:
baseline_period:
target_value:
target_period:
current_value:
current_period:

initiative_cost_period:
initiative_cost_value:
net_value_check: Positive | Negative | Neutral | Unknown | NotApplicable
attribution_method: DirectMeasurement | A_BComparison | BeforeAfter | MatchedComparison | ExpertJudgement | ProxyMetric | Unknown
known_confounders:
  -
confidence_level: High | Medium | Low

primary_evidence_source:
evidence_owner:
evidence_freshness: Fresh | Aging | Stale | Unknown
evidence_access_status: Available | PendingAccess | Restricted | Missing

approval_state: ArtifactState per contracts/17 (Draft | Approved | Published | Superseded | Invalidated)
approval_required_for:
  - ScorecardPublication
  - StatusChange
  - RealizedValueDeclaration

expected_decision:
  - Build
  - ContinuePilot
  - Scale
  - Fix
  - Defer
  - Rebaseline
  - Stop

notes:
```

### Field Ownership Note

`current_value`, `current_period`, and `metric_unit` are operational fields sourced from `MetricSnapshot` (contracts/17 section 5) at assessment time; they appear in this template for owner convenience but are not persisted on the `ValueHypothesis` record. The canonical persisted fields are defined in contracts/17 section 4.

## 4. Required Before Gate A Exit

| Field | Requirement |
|---|---|
| `use_case_id` | Required |
| `value_owner` | Required |
| `expected_benefit` | Required |
| `primary_metric_id` | Required or marked missing |
| `baseline_value` | Required or marked unavailable |
| `target_value` | Required or marked unavailable |
| `primary_evidence_source` | Required or marked unavailable |
| `approval_required_for` | Required |

## 5. Required Before Realized Value Claim

A use case cannot be classified as **Realized** unless all are true:

- Baseline exists.
- Current value exists.
- Evidence source is authoritative.
- Evidence is fresh or accepted by owner.
- Attribution method is not `Unknown`.
- Known confounders are documented.
- Net-value check is `Positive` or `NotApplicable`.
- Approval state is `Approved` for publication.
- Sustainment check has no regression.
