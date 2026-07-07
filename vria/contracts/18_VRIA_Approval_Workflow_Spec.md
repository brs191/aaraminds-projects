# VRIA Approval Workflow Specification

**Document set:** Value Realization Intelligence Agent  
**Version:** v1.2  
**Date:** 2026-07-05  
**Owner:** AaraMinds / Common Capabilities AI Portfolio  
**Status:** Implementation baseline  


---

## 1. Purpose

This document defines the approval lifecycle for VRIA. Approval is a first-class workflow, not a comment in a decision log.

## 2. State Machines

Two distinct lifecycles. The **approval request** tracks the review of a proposed action; the **target artifact** (scorecard, assessment, use-case record) tracks its own publication lifecycle. States are never shared between the two machines. Enums are authoritative in `contracts/17` (`ApprovalRequestState`, `ArtifactState`).

### 2.1 Approval Request Lifecycle (`ApprovalRequestState`)

```text
Draft
  ├─ submit_for_approval (requester) → Submitted
  ├─ withdraw (requester) → Withdrawn          [terminal]
Submitted
  ├─ approve (approver) → Approved             [terminal; unlocks artifact transition]
  ├─ reject (approver) → Rejected              [terminal; a new request is required]
  ├─ request_changes (approver) → ChangesRequested
ChangesRequested
  ├─ resubmit (requester) → Submitted
  ├─ withdraw (requester) → Withdrawn          [terminal]
```

`Rejected` is terminal by design: reopening a rejected request would erase the reviewer's decision from the active trail. Revisions go through a new request linked to the same target.

### 2.2 Target Artifact Lifecycle (`ArtifactState`)

```text
Draft
  ├─ approval request Approved → Approved
Approved
  ├─ publish_scorecard / execute (system, requires Approved request) → Published
  ├─ supersede (approval-gated) → Superseded   [terminal]
Published
  ├─ supersede (approval-gated) → Superseded   [terminal]
  ├─ invalidate (approval-gated) → Invalidated [terminal]
```

`Invalidated` replaces the former Published→Withdrawn transition: withdrawal is a requester action on a pending request and has no meaning for a published artifact.

## 3. Approval-Required Actions

| Action | Approval Required | Approver |
|---|---|---|
| Publish scorecard | Yes | Portfolio Lead or Executive Sponsor |
| Declare realized value | Yes | Value Owner + Portfolio Lead; Finance owner for financial claims |
| Update registry owner/tier/status | Yes | Portfolio Lead / Governance |
| Create external Jira/ADO task | Yes unless configured low-risk | Use-case Owner or Portfolio Lead |
| Supersede scorecard | Yes | Portfolio Lead |
| Invalidate assessment | Yes | Portfolio Lead or Governance Reviewer |
| Funding recommendation publication | Yes | Executive Sponsor / Portfolio Governance |

## 4. Tool Contracts

### `submit_for_approval`

```json
{
  "draft_id": "uuid",
  "action_type": "ApprovalActionType",
  "target_id": "string",
  "target_type": "Assessment | Scorecard | UseCase | FollowUpAction",
  "requested_by": "user_id",
  "approver_ids": ["user_id"],
  "rationale": "string",
  "risk_tier": "ToolPolicyTier"
}
```

Returns:

```json
{
  "approval_id": "uuid",
  "approval_state": "Submitted",
  "submitted_at": "datetime",
  "audit_id": "uuid"
}
```

### `approve_or_reject_draft`

```json
{
  "approval_id": "uuid",
  "decision": "Approved | Rejected | ChangesRequested",
  "decided_by": "user_id",
  "comments": "string"
}
```

`decided_by` is taken from the authenticated identity (Entra ID token), never from the payload, and is persisted on `approval_requests.decided_by`.

Returns:

```json
{
  "approval_id": "uuid",
  "approval_state": "ApprovalRequestState",
  "decision_record_id": "uuid",
  "audit_id": "uuid"
}
```

### `publish_scorecard`

May execute only when `approval_state=Approved`.

### `invalidate_assessment`

Requires reason and creates immutable audit record.

### `supersede_scorecard`

Creates link from old scorecard to replacement scorecard. Never edits published scorecards in place.

## 5. Audit Requirements

Approval audit must record:

- requested_by
- approver_ids
- decision
- comments
- target hash before approval
- target hash after publication/execution
- timestamp
- tool/action executed
- trace ID

## 6. SLA Guidance

| Approval Type | Target SLA |
|---|---|
| Scorecard publication | 2 business days |
| Registry update | 2 business days |
| Realized value declaration | 5 business days |
| External follow-up action | 1 business day |
| Supersession / invalidation | 1 business day for high severity |
