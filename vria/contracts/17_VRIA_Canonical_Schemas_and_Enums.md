# VRIA Canonical Schemas and Enums

**Document set:** Value Realization Intelligence Agent  
**Version:** v1.2  
**Date:** 2026-07-05  
**Owner:** AaraMinds / Common Capabilities AI Portfolio  
**Status:** Implementation baseline  


---

## 1. Purpose

This document is the authoritative source for VRIA enums, JSON schemas, and payload types. Other documents must reference these definitions instead of redefining them.

## 2. Canonical Enums

```text
UseCaseTier = Tool | Agent | Layer | Unclassified
DeliveryStatus = Draft | Discovery | Training | PTB_NotStarted | PTB_InProgress | PTB_Approved | PTO_NotStarted | PTO_InProgress | PTO_Approved | InProgress | Pilot | Production | Blocked | Stopped | Unknown
ValueState = NotReady | HypothesisOnly | BaselineReady | OnTrack | AtRisk | Realized | NotRealized | Regressed | Unproven
Recommendation = Build | ContinuePilot | Scale | Fix | Defer | Rebaseline | Stop | NeedsSponsor | NeedsEvidence
ConfidenceLevel = High | Medium | Low
EvidenceFreshness = Fresh | Aging | Stale | Unknown
EvidenceAuthority = Authoritative | Secondary | Unknown
ApprovalRequestState = Draft | Submitted | ChangesRequested | Approved | Rejected | Withdrawn
ArtifactState = Draft | Approved | Published | Superseded | Invalidated
SustainmentStatus = NotStarted | Ok | AtRisk | Regressed
ApprovalActionType = ScorecardPublication | RegistryUpdate | StatusChange | RealizedValueDeclaration | FollowUpAction | AssessmentInvalidation | ScorecardSupersession
AttributionMethod = DirectMeasurement | A_BComparison | BeforeAfter | MatchedComparison | ExpertJudgement | ProxyMetric | Unknown
NetValueCheck = Positive | Negative | Neutral | Unknown | NotApplicable
ToolPolicyTier = Tier1_Read | Tier2_SensitiveRead | Tier3_DraftWrite | Tier4_ExternalAction | Tier5_HighRiskDecision
```

## 3. `UseCase` Schema

```json
{
  "use_case_id": "string",
  "name": "string",
  "tier": "UseCaseTier",
  "domain": "string",
  "value_owner": "string",
  "delivery_owner": "string",
  "sponsor": "string|null",
  "delivery_status": "DeliveryStatus",
  "primary_metric_id": "string|null",
  "approval_state": "ArtifactState",
  "created_at": "datetime",
  "updated_at": "datetime",
  "record_version": 1
}
```

## 4. `ValueHypothesis` Schema

```json
{
  "value_hypothesis_id": "uuid",
  "use_case_id": "string",
  "business_objective": "string",
  "expected_benefit": "string",
  "benefit_type": "Cost | Productivity | CycleTime | Quality | Risk | Revenue | Adoption | DecisionQuality | Compliance",
  "primary_metric_id": "string",
  "baseline_value": "number|string|null",
  "baseline_period": {"start": "date", "end": "date"},
  "target_value": "number|string|null",
  "target_period": {"start": "date", "end": "date"},
  "initiative_cost_period": {"start": "date", "end": "date", "cost": "number|null", "currency": "string|null"},
  "attribution_method": "AttributionMethod",
  "known_confounders": ["string"],
  "net_value_check": "NetValueCheck",
  "evidence_source_ids": ["uuid"],
  "approval_state": "ArtifactState",
  "record_version": 1
}
```

## 5. `MetricSnapshot` Schema

```json
{
  "metric_snapshot_id": "uuid",
  "metric_id": "string",
  "use_case_id": "string",
  "period": {"start": "date", "end": "date"},
  "baseline_value": "number|string|null",
  "current_value": "number|string|null",
  "target_value": "number|string|null",
  "metric_unit": "string",
  "source_system": "string",
  "source_owner": "string",
  "authority": "EvidenceAuthority",
  "freshness": "EvidenceFreshness",
  "initiative_cost_period": {"start": "date", "end": "date", "cost": "number|null", "currency": "string|null"},
  "created_at": "datetime"
}
```

## 6. `EvidenceSource` Schema

```json
{
  "evidence_source_id": "uuid",
  "use_case_id": "string",
  "source_type": "Metric | Cost | Document | DeliveryStatus | Operational | UserFeedback | Evaluation",
  "source_system": "string",
  "source_owner": "string",
  "citation_pointer": "string",
  "authority": "EvidenceAuthority",
  "freshness": "EvidenceFreshness",
  "access_classification": "Public | Internal | Confidential | Restricted",
  "retrieved_at": "datetime",
  "content_hash": "string"
}
```

## 7. `ValueAssessment` Schema

```json
{
  "assessment_id": "uuid",
  "use_case_id": "string",
  "value_state": "ValueState",
  "realization_score": 0,
  "score_breakdown": {
    "strategic_alignment": 0,
    "baseline_quality": 0,
    "evidence_quality": 0,
    "metric_movement": 0,
    "attribution_confidence": 0,
    "net_value": 0,
    "sustainment": 0,
    "governance_readiness": 0
  },
  "applied_caps": ["string"],
  "recommendation": "Recommendation",
  "confidence": "ConfidenceLevel",
  "attribution_method": "AttributionMethod",
  "known_confounders": ["string"],
  "net_value_check": "NetValueCheck",
  "initiative_cost_period": {"start": "date", "end": "date", "cost": "number|null", "currency": "string|null"},
  "evidence_source_ids": ["uuid"],
  "sustainment_threshold": "number|null",
  "sustainment_status": "SustainmentStatus",
  "missing_evidence": ["string"],
  "rationale": "string",
  "approval_state": "ArtifactState",
  "scoring_rule_version": "string",
  "model_version": "string",
  "prompt_version": "string",
  "created_at": "datetime"
}
```

## 8. `ApprovalRequest` Schema

```json
{
  "approval_id": "uuid",
  "action_type": "ApprovalActionType",
  "target_id": "string",
  "target_type": "Assessment | Scorecard | UseCase | FollowUpAction",
  "requested_by": "user_id",
  "approver_ids": ["user_id"],
  "approval_state": "ApprovalRequestState",
  "decided_by": "user_id|null",
  "risk_tier": "ToolPolicyTier",
  "rationale": "string",
  "submitted_at": "datetime",
  "decided_at": "datetime|null",
  "decision_comments": "string|null"
}
```

## 9. `Scorecard` Schema

```json
{
  "scorecard_id": "uuid",
  "title": "string",
  "summary": "string",
  "period": {"start": "date", "end": "date"},
  "evidence_coverage_summary": {"assessments_total": 0, "with_citations": 0, "with_gaps": 0},
  "artifact_state": "ArtifactState",
  "assessment_ids": ["uuid"],
  "supersedes_scorecard_id": "uuid|null",
  "decision_log_pointer": "uuid|null",
  "published_at": "datetime|null",
  "created_by": "user_id",
  "created_at": "datetime"
}
```

## 10. `DecisionRecord` Schema

```json
{
  "decision_record_id": "uuid",
  "decision_type": "ApprovalActionType",
  "target_id": "string",
  "target_type": "Assessment | Scorecard | UseCase | FollowUpAction",
  "decision": "Approved | Rejected | ChangesRequested",
  "rationale": "string",
  "decided_by": "user_id",
  "approval_id": "uuid",
  "created_at": "datetime"
}
```

## 11. Compatibility Rule

Any schema change requires version increment, migration plan, golden eval run, and dashboard/API contract review.
