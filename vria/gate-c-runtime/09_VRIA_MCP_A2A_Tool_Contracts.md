# VRIA MCP / A2A Tool Contracts

**Document set:** Value Realization Intelligence Agent  
**Version:** v1.2  
**Date:** 2026-07-05  
**Owner:** AaraMinds / Common Capabilities AI Portfolio  
**Status:** Implementation baseline  


---

## 1. Purpose

This document defines strict tool, MCP, and A2A contracts for VRIA. A listed tool without accepted input, returned output, permissions, failure handling, audit record, and approval boundary is incomplete.

All payload enums are authoritative in `contracts/17_VRIA_Canonical_Schemas_and_Enums.md`.

## 2. Contract Standard

Each tool must define:

| Required Field | Meaning |
|---|---|
| Purpose | Why the tool exists. |
| Accepted input | Strict JSON payload. |
| Returned output | Strict JSON payload. |
| Permissions | Read-only, draft-write, or approval-gated write. |
| Failure handling | Error codes and safe degradation behavior. |
| Audit record | What must be logged. |
| Approval boundary | Whether execution requires approval. |
| Source of truth | Authoritative system. |
| Timeout/retry | Runtime limits. |
| Policy tier | Tool risk tier. |

## 3. Tool Contracts

### 3.1 `load_use_cases(source_id, source_type)`

**Purpose:** Load use cases from approved portfolio sources into the registry staging area.

**Input:**
```json
{
  "source_id": "string",
  "source_type": "Spreadsheet | Markdown | API | Manual",
  "requested_by": "user_id"
}
```

**Output:**
```json
{
  "import_batch_id": "uuid",
  "records_loaded": 0,
  "records_rejected": 0,
  "validation_errors": [
    {"row_ref": "string", "field": "string", "error_code": "string", "message": "string"}
  ],
  "audit_id": "uuid"
}
```

**Permissions:** Portfolio Lead or Admin.  
**Failure:** Return validation errors; do not partially promote to active registry without explicit user confirmation.  
**Audit:** user, source, row count, hash, timestamp.  
**Approval:** No approval for staging; approval required for promotion.  
**Policy tier:** Tier 2.

---

### 3.2 `get_use_case_status(use_case_id)`

**Purpose:** Retrieve current delivery/PTB/PTO status.

**Input:**
```json
{"use_case_id": "string"}
```

**Output:**
```json
{
  "use_case_id": "string",
  "delivery_status": "DeliveryStatus",
  "status_source": "string",
  "source_owner": "string",
  "updated_at": "datetime",
  "confidence": "High | Medium | Low",
  "audit_id": "uuid"
}
```

**Permissions:** Read-only.  
**Failure:** Return `delivery_status=Unknown`; do not infer from text.  
**Audit:** use_case_id, source, timestamp, caller.  
**Approval:** None for read.  
**Policy tier:** Tier 1.

---

### 3.3 `get_value_hypothesis(use_case_id)`

**Purpose:** Retrieve latest active value hypothesis.

**Input:**
```json
{"use_case_id": "string", "include_history": false}
```

**Output:**
```json
{
  "value_hypothesis": "ValueHypothesis",
  "version": 1,
  "approval_state": "ArtifactState",
  "missing_required_fields": ["string"],
  "audit_id": "uuid"
}
```

**Permissions:** Read-only.  
**Failure:** Return `NOT_FOUND` and missing-field list.  
**Audit:** use_case_id, version, caller.  
**Approval:** None for read.  
**Policy tier:** Tier 1.

---

### 3.4 `draft_use_case_update(use_case_id, proposed_changes)`

**Purpose:** Draft updates to registry/hypothesis fields without committing them.

**Allowed `proposed_changes` fields:**
```json
{
  "tier": "UseCaseTier",
  "domain": "string",
  "value_owner": "string",
  "delivery_owner": "string",
  "sponsor": "string",
  "primary_metric_id": "string",
  "expected_benefit": "string",
  "attribution_method": "AttributionMethod",
  "known_confounders": ["string"],
  "initiative_cost_period": "string",
  "net_value_check": "NetValueCheck"
}
```

**Output:**
```json
{
  "draft_id": "uuid",
  "validation_status": "Valid | Invalid",
  "validation_errors": ["string"],
  "requires_approval": true,
  "approval_action_type": "RegistryUpdate",
  "audit_id": "uuid"
}
```

**Permissions:** Draft-write for Portfolio Lead or Use-Case Owner.  
**Failure:** Reject disallowed fields; preserve original record.  
**Audit:** previous value, proposed value, caller, timestamp.  
**Approval:** Required before commit.  
**Policy tier:** Tier 3.

---

### 3.5 `get_metric_snapshot(metric_id, period)`

**Purpose:** Retrieve baseline/current/target metric snapshot from authoritative metric source.

**Input:**
```json
{
  "metric_id": "string",
  "period": {"start": "date", "end": "date"},
  "use_case_id": "string"
}
```

**Output:**
```json
{
  "metric_snapshot": "MetricSnapshot",
  "initiative_cost_period": {"start": "date", "end": "date", "cost": 0, "currency": "string"},
  "source_owner": "string",
  "freshness": "EvidenceFreshness",
  "authority": "Authoritative | Secondary | Unknown",
  "audit_id": "uuid"
}
```

**Permissions:** Read-only metric access.  
**Failure:** Return `METRIC_UNAVAILABLE`; score must cap per `20`.  
**Audit:** metric_id, period, source, caller.  
**Approval:** None for read.  
**Policy tier:** Tier 2.

---

### 3.6 `search_evidence_documents(query, filters)`

**Purpose:** Retrieve cited supporting evidence.

**Input:**
```json
{
  "query": "string",
  "filters": {
    "use_case_id": "string",
    "document_type": ["BRD", "PRD", "Deck", "DecisionNote", "MetricDefinition"],
    "date_from": "date",
    "date_to": "date"
  },
  "top_k": 10
}
```

**Output:**
```json
{
  "results": [
    {
      "document_id": "uuid",
      "title": "string",
      "citation_pointer": "string",
      "authority": "Authoritative | Secondary | Unknown",
      "freshness": "EvidenceFreshness",
      "evidence_quality": "High | Medium | Low",
      "snippet": "string"
    }
  ],
  "audit_id": "uuid"
}
```

**Permissions:** Read-only, role-filtered.  
**Failure:** Return empty result with `NO_EVIDENCE_FOUND`; do not fabricate citations.  
**Audit:** query hash, filters, caller.  
**Approval:** None for read.  
**Policy tier:** Tier 2.

---

### 3.7 `score_value_realization(use_case_id, assessment_context)`

**Purpose:** Apply executable scoring logic from `20`.

**Input:**
```json
{
  "use_case_id": "string",
  "assessment_context": {
    "value_hypothesis_id": "uuid",
    "metric_snapshot_id": "uuid",
    "evidence_source_ids": ["uuid"],
    "scoring_rule_version": "v1.2"
  }
}
```

**Output:**
```json
{
  "assessment_id": "uuid",
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
  "value_state": "ValueState",
  "recommendation": "Recommendation",
  "confidence": "ConfidenceLevel",
  "attribution_method": "AttributionMethod",
  "known_confounders": ["string"],
  "net_value_check": "NetValueCheck",
  "initiative_cost_period": {"start": "date", "end": "date", "cost": 0, "currency": "string"},
  "sustainment_threshold": 0,
  "sustainment_status": "SustainmentStatus",
  "evidence_source_ids": ["uuid"],
  "missing_evidence": ["string"],
  "rationale": "string",
  "approval_state": "Draft",
  "audit_id": "uuid"
}
```

**Permissions:** Internal compute.  
**Failure:** Return `SCORING_INPUT_INCOMPLETE`; no score if required context is absent.  
**Audit:** inputs, rules version, model version, prompt version, output hash.  
**Approval:** Score is draft until approved.  
**Policy tier:** Tier 2.

---

### 3.8 Approval and Publishing Tools

| Tool | Purpose | Approval Boundary |
|---|---|---|
| `submit_for_approval(draft_id, action_type, approver_ids)` | Submit draft scorecard/update/action. | Creates approval request; does not execute action. |
| `get_pending_approvals(user_id)` | Return approval queue. | Read-only. |
| `approve_or_reject_draft(approval_id, decision, comments)` | Approve, reject, or request changes. | Approver-only; immutable decision log. |
| `publish_scorecard(scorecard_id, approval_id)` | Publish approved scorecard. | Requires Approved approval state. |
| `invalidate_assessment(assessment_id, reason)` | Mark assessment invalid. | Requires Portfolio Lead or Governance approval. |
| `supersede_scorecard(scorecard_id, replacement_id)` | Supersede prior published scorecard. | Requires approval and audit. |
| `append_decision_log(decision_record)` | Append decision record. | Append-only; no update/delete. |
| `create_follow_up_action(source_id, target_system, assignee, due_date)` | Create Jira/ADO task. | Requires approval unless configured as low-risk. |

Payloads for `submit_for_approval`, `approve_or_reject_draft`, `publish_scorecard`, `invalidate_assessment`, and `supersede_scorecard` are defined in `contracts/18`. The remaining three:

### `get_pending_approvals(user_id)`

**Input:** `{"user_id": "string", "cursor": "string|null", "limit": 20}`
**Output:** `{"approvals": ["ApprovalRequest"], "next_cursor": "string|null", "audit_id": "uuid"}`
**Permissions:** Approver or Portfolio Lead (own queue only). **Failure:** empty list with `NO_PENDING_APPROVALS`. **Policy tier:** Tier 1.

### `create_follow_up_action(source_id, target_system, assignee, due_date)`

**Input:**
```json
{
  "source_id": "uuid",
  "source_type": "Assessment | Scorecard",
  "target_system": "Jira | ADO",
  "title": "string",
  "description": "string",
  "assignee": "string",
  "due_date": "date",
  "approval_id": "uuid"
}
```
**Output:** `{"external_task_id": "string", "external_url": "string", "audit_id": "uuid"}`
**Permissions:** Approval-gated write (Tier 4); `approval_id` must reference an Approved request of type `FollowUpAction`. **Failure:** `EXTERNAL_SYSTEM_UNAVAILABLE`; no retry without new approval check. **Policy tier:** Tier 4.

### `append_decision_log(decision_record)`

**Input:** `{"decision_record": "DecisionRecord"}` (schema in `contracts/17` section 10)
**Output:** `{"decision_record_id": "uuid", "audit_id": "uuid"}`
**Permissions:** System-internal, append-only; no update or delete surface exists. **Failure:** `DECISION_LOG_WRITE_FAILED`; the triggering action must roll back. **Policy tier:** Tier 3.

## 4. A2A Contract

A2A calls are allowed only to approved specialist agents.

**Request envelope:**
```json
{
  "request_id": "uuid",
  "calling_agent_id": "vria-agent-prod",
  "target_agent_id": "string",
  "purpose": "EvidenceRequest | SpecialistAssessment | MetricExplanation",
  "use_case_id": "string",
  "input_payload_schema": "string",
  "input_payload": "object conforming to the per-purpose input schema below",
  "required_evidence_provenance": true,
  "timeout_ms": 10000
}
```

**Response envelope:**
```json
{
  "request_id": "uuid",
  "target_agent_id": "string",
  "status": "Success | Partial | Failed",
  "output_payload_schema": "string",
  "output_payload": "object conforming to the per-purpose output schema below",
  "evidence_provenance": [
    {"source_id": "string", "citation_pointer": "string", "authority": "string"}
  ],
  "confidence": "High | Medium | Low",
  "errors": ["string"]
}
```

**Per-purpose payload schemas** (referenced by `input_payload_schema` / `output_payload_schema`; no free-form `{}` payloads):

| Purpose | Input payload | Output payload |
|---|---|---|
| `EvidenceRequest` | `{"use_case_id", "evidence_types[]": "Metric\|Cost\|Document", "period": {"start","end"}}` | `{"evidence": ["EvidenceSource"], "not_found[]": "string"}` |
| `SpecialistAssessment` | `{"use_case_id", "assessment_scope": "Architecture\|FinOps\|Quality", "context_document_ids[]": "uuid"}` | `{"findings[]": {"finding","severity","citation_pointer"}, "recommendation": "string"}` |
| `MetricExplanation` | `{"metric_id", "period": {"start","end"}, "observed_anomaly": "string"}` | `{"explanation": "string", "contributing_factors[]": "string", "confidence": "High\|Medium\|Low"}` |

**Trust requirements:** workload identity, allowlisted agent ID, schema validation, timeout, provenance required, no cascading actions without VRIA approval workflow.
