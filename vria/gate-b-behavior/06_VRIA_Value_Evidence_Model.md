# VRIA Value Evidence Model

**Document set:** Value Realization Intelligence Agent  
**Version:** v1.2.1  
**Date:** 2026-07-05  
**Owner:** AaraMinds / Common Capabilities AI Portfolio  
**Status:** Implementation baseline  


---

## 1. Purpose

This document defines the evidence model used by VRIA to determine whether an AI use case has measurable, credible, attributed, net-positive, and reviewable value.

## 2. Evidence Principle

> No evidence, no value claim.

The agent may describe expected value, potential value, or hypothesis value, but it cannot report realized value unless evidence supports it.

## 3. Authoritative Schemas

All entity schemas and enums are defined in `contracts/17_VRIA_Canonical_Schemas_and_Enums.md`.

## 4. Evidence Quality Dimensions

| Dimension | Description |
|---|---|
| Authority | Whether the source is the system of record or secondary. |
| Freshness | Whether the data is current for the reporting window. |
| Completeness | Whether baseline, current, target, and period exist. |
| Attribution | Whether movement can be linked to the initiative. |
| Net value | Whether benefit is greater than initiative cost where applicable. |
| Conflict status | Whether sources disagree. |
| Approval state | Whether claim/scorecard is approved for publication. |
| Sustainment | Whether realized value is maintained over time (threshold and check cadence defined in `20`, section 7). |

## 5. Attribution Methods

| Method | Confidence Default | Notes |
|---|---|---|
| DirectMeasurement | High | Directly measured workflow or system output. |
| A_BComparison | High | Treatment/control comparison. |
| MatchedComparison | Medium | Comparable group or matched period. |
| BeforeAfter | Medium | Requires confounders. |
| ExpertJudgement | Low | Not enough for Realized unless paired with evidence. |
| ProxyMetric | Low | Use for early signal only. |
| Unknown | Low | Blocks Realized state. |

## 6. Net Value Rule

For cost, productivity, and revenue claims:

```text
net_value = measured_benefit - initiative_cost_period
```

A use case cannot be **Realized** when `net_value_check` is `Unknown` or `Negative`, or initiative cost is missing for a financial/productivity claim. For non-financial risk/quality claims, `NotApplicable` is allowed with rationale.

## 7. Known Confounders

Confounders must be captured when attribution is not direct. Examples include seasonality, parallel modernization, staffing changes, incident volume changes, cloud pricing change, adoption campaigns, and source-system changes.

## 8. Freshness Rules

| Freshness | Definition | Behavior |
|---|---|---|
| Fresh | Within accepted reporting window | Usable for scoring. |
| Aging | Slightly older but accepted by owner | Usable with confidence caveat. |
| Stale | Outside reporting window | Caps score and cannot support Realized. |
| Unknown | Date unavailable | Evidence gap. |

### Reporting Window and Cycle Cadence

Every metric carries a `reporting_window` (set at hypothesis approval; default **monthly**). It drives both freshness classification and the sustainment check schedule:

| Snapshot age (in reporting windows) | Freshness |
|---|---|
| Within current window | Fresh |
| 1 missed window | Aging |
| 2+ missed windows | Stale |

One evidence freshness cycle = one reporting window. Sustainment checks (`contracts/20` section 7) run once per reporting window per Realized use case. The next check date is always computable: `last_check + reporting_window`.

## 9. Conflict Resolution

Prefer authoritative system of record, then newer source, then approved decision log over draft notes. Surface unresolved conflicts; do not average conflicting metrics unless the metric owner approves.

## 10. Final Output Evidence Contract

Every published assessment must expose source IDs, citation pointers, metric period, data freshness, authority status, attribution method, known confounders, net-value check, approval state, scoring rule version, and model/prompt version.
