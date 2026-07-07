# VRIA API and Event Contracts

**Document set:** Value Realization Intelligence Agent  
**Version:** v1.2  
**Date:** 2026-07-05  
**Owner:** AaraMinds / Common Capabilities AI Portfolio  
**Status:** Implementation baseline  


---

## 1. Purpose

This document defines service-level REST APIs and event contracts for VRIA implementation.

## 2. Conventions

- **Base path:** all endpoints are prefixed `/api/v1` (table below omits the prefix for brevity).
- **Auth:** Entra ID OIDC bearer tokens; roles map to app roles per `gate-c-runtime/10` section 3. Approver identity is always taken from the token, never the payload.
- **Pagination:** list endpoints accept `cursor` + `limit` (default 20, max 100) and return `next_cursor`.
- **Errors:** standard envelope in section 3.

## 2a. REST API Contracts

| Method | Endpoint | Purpose | Approval Required |
|---|---|---|---|
| GET | `/api/use-cases` | List use cases | No |
| GET | `/api/use-cases/{id}` | Get use-case detail | No |
| POST | `/api/use-cases/import` | Stage import batch | No for staging |
| POST | `/api/use-cases/{id}/draft-update` | Draft registry/hypothesis update | Approval before commit |
| GET | `/api/use-cases/{id}/hypothesis` | Get value hypothesis | No |
| POST | `/api/use-cases/{id}/assessments` | Generate draft assessment | No publication |
| GET | `/api/assessments/{id}` | Read assessment | No |
| POST | `/api/scorecards` | Generate draft scorecard | Approval before publish |
| POST | `/api/approvals` | Submit approval request | Creates request only |
| GET | `/api/approvals/pending` | Get pending approvals | No |
| POST | `/api/approvals/{id}/decision` | Approve/reject/request changes | Approver only |
| POST | `/api/scorecards/{id}/publish` | Publish scorecard | Requires Approved approval |
| POST | `/api/assessments/{id}/invalidate` | Invalidate assessment | Requires approval |
| POST | `/api/scorecards/{id}/supersede` | Supersede scorecard | Requires approval |
| POST | `/api/follow-up-actions` | Create external Jira/ADO task | Requires Approved `FollowUpAction` request |
| GET | `/api/decision-log` | List decision records (filter by target) | No (role-filtered) |
| POST | `/api/metric-snapshots` | Ingest metric snapshot | No (Tier 2 source identity) |
| POST | `/api/evidence-sources` | Register evidence source | No (Tier 2 source identity) |

## 3. Standard Error Envelope

```json
{
  "error_code": "string",
  "message": "string",
  "safe_state": "Unknown | Draft | NoActionTaken",
  "trace_id": "uuid",
  "retryable": false
}
```

## 4. Event Contracts

All events must include:

```json
{
  "event_id": "uuid",
  "event_type": "string",
  "occurred_at": "datetime",
  "actor_id": "string",
  "target_id": "string",
  "schema_version": "v1.3",
  "payload": "object conforming to the per-event schema in section 5"
}
```

## 5. Event Types and Payloads

Transport: Azure Service Bus topics. Each payload references canonical types in `contracts/17`; no event ships an empty payload.

| Event | Payload fields (all required unless noted) |
|---|---|
| `use_case.imported` | `import_batch_id`, `use_case_ids[]`, `stage: Staged\|Promoted`, `records_rejected` |
| `value_hypothesis.updated` | `value_hypothesis_id`, `use_case_id`, `record_version`, `artifact_state`, `changed_fields[]` |
| `evidence_source.registered` | `EvidenceSource` (17 §6) |
| `metric_snapshot.ingested` | `MetricSnapshot` (17 §5) |
| `assessment.generated` | `assessment_id`, `use_case_id`, `value_state`, `realization_score`, `pre_cap_score`, `recommendation`, `scoring_rule_version` |
| `assessment.invalidated` | `assessment_id`, `reason`, `approval_id` |
| `approval.submitted` | `ApprovalRequest` (17 §8) |
| `approval.decided` | `approval_id`, `decision`, `decided_by`, `decision_record_id` |
| `scorecard.generated` | `scorecard_id`, `period`, `assessment_ids[]` |
| `scorecard.published` | `scorecard_id`, `approval_id`, `evidence_coverage_summary` (17 §9), `decision_log_pointer` |
| `scorecard.superseded` | `scorecard_id`, `replacement_id`, `approval_id` |
| `eval.failed` | `eval_id`, `suite: Golden\|Volume\|RedTeam\|Online`, `test_ids[]`, `severity: Critical\|NonCritical` |
| `drift.detected` | `metric`, `window`, `baseline_value`, `observed_value`, `threshold` |

## 6. Event Handling Rules

- Events are immutable.
- Consumers must be idempotent using `event_id`.
- Failed consumers should retry with backoff.
- High-risk events must include approval ID.
- Published scorecard events must include evidence coverage summary and decision log pointer.
