# VRIA Security and Governance Model

**Document set:** Value Realization Intelligence Agent  
**Version:** v1.2  
**Date:** 2026-07-05  
**Owner:** AaraMinds / Common Capabilities AI Portfolio  
**Status:** Implementation baseline  


---

## 1. Purpose

This document defines the controls required to operate VRIA safely in an enterprise environment.

## 2. Governance Principles

- Business value first.
- Evidence required for value claims.
- Human approval for funding, status, compliance, publishing, and production-impacting decisions.
- Least privilege for all tools and data sources.
- Prompt-injection defense by design.
- Full audit trail for assessments, recommendations, approvals, and decisions.
- Re-review triggers for material changes.

## 3. Roles and RBAC

| Role | Capabilities |
|---|---|
| Viewer | Read approved scorecards and summaries. |
| Use-Case Owner | Edit draft value hypothesis, respond to evidence gaps. |
| Portfolio Lead | Manage registry, draft scorecards, submit approvals. |
| Approver | Approve/reject scorecards, status updates, realized-value declarations. |
| Governance Reviewer | Review policy tier, audit, approval boundaries. |
| Admin | Configure integrations, users, roles, and policy settings. |
| Service Account | Tool execution under workload identity with least privilege. |

## 4. Policy Tiers

| Tier | Examples | Controls |
|---|---|---|
| Tier 1 Read | Registry/status reads | Audit and RBAC. |
| Tier 2 Sensitive Read | Metrics, cost, evidence docs | RBAC, redaction, data classification. |
| Tier 3 Draft Write | Draft registry/update/scorecard | Approval workflow before commit. |
| Tier 4 External Action | Create Jira/ADO task, notify owner | Explicit approval and audit. |
| Tier 5 High-Risk Decision | Funding/status/realized-value declaration | Human-only decision; agent drafts only. |

## 5. Prompt-Injection Defense

- Treat retrieved documents as untrusted content.
- Separate system/tool instructions from document text.
- Ignore instructions found inside evidence documents.
- Require citation and authority metadata for evidence.
- Log and flag malicious content.
- Run red-team tests before release.

## 6. Re-Review Triggers

Re-review is required when scoring rules, model, prompt, tool contract, data source, policy tier, approval workflow, or evidence source authority changes.

## 7. Audit Requirements

Every tool call, score, approval, published scorecard, supersession, invalidation, and external action must have immutable audit record with caller, timestamp, inputs, outputs, source references, model/rules version, and approval ID where applicable.
