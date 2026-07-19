# BA Agent Phase 2 Teams Sandbox Blocked Package

This package records why `P2-SBX-TEAMS-READ` and Teams write-like actions remain blocked for Phase 2 sandbox progression. It is a blocked-path artifact only and does not authorize Teams/Copilot 365 reads, sends, Adaptive Cards, approval records, channel access, message metadata processing, credentials, app registration, Graph API access, or any other external side effect.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Phase 2 Teams Sandbox Blocked Package |
| Version | 0.1 |
| Status | Blocked; not approved for preparation or execution |
| Prepared date | 2026-07-13 |
| Accountable owner | RAJA |
| Candidate row | `P2-SBX-TEAMS-READ` / `send_adaptive_card` |
| Primary references | `docs/development/phase-2-sandbox-authorization-package.md` v1.0; `docs/development/phase-2-sandbox-owner-review-package.md` v0.2; `docs/development/mcp-validation-register.json` v0.9; `docs/development/phase-2-tool-approval-matrix.md` v0.9; `docs/planning/decision-log.md` v2.5 |
| Explicit non-authorization | No Teams/Copilot 365 read, message metadata access, channel access, Adaptive Card send, approval record, webhook subscription, Graph API call, credential use, external publish/storage, or write-like side effect |

## 1) Current verdict

`P2-SBX-TEAMS-READ` remains **deferred/blocked**. RAJA approved preparation for Jira, Confluence, and Git/GitHub only; Teams was explicitly not approved for preparation in the current sandbox evidence lane.

No Teams adapter, MCP wrapper, Graph API client, channel configuration, app registration, approval-flow implementation, or Adaptive Card send path should be implemented from the current Phase 2 authorization package.

## 2) Blocked evidence checklist

The following evidence would be required before Teams can even move from blocked to preparation review.

| Evidence ID | Required evidence | Current status | Blocker |
| --- | --- | --- | --- |
| `TMS-EV-001` | Explicit RAJA approval to start Teams preparation | Not approved | Current RAJA decision excludes Teams from preparation. |
| `TMS-EV-002` | Channel/app owner and review lanes | `[RAJA]` | Channel owner, tenant/app owner, security/privacy, platform, compliance/legal, and QA/control reviewers are not named. |
| `TMS-EV-003` | Exact channel/app scope | `[RAJA]` | No channel, tenant, app registration, bot/app identity, or approved scope is recorded. |
| `TMS-EV-004` | Privacy and message-retention review | `[RAJA]` | Message metadata/content handling, retention, residency, and deletion are not approved. |
| `TMS-EV-005` | Read-vs-send policy | Blocked | Teams sends, Adaptive Cards, approval records, and webhook subscriptions are write-like external side effects. |
| `TMS-EV-006` | Actual schema/API validation | Not captured | No Teams/Copilot 365 MCP or Graph schema is validated. |
| `TMS-EV-007` | Auth, tenant, and rate-limit posture | Not captured | Least-privilege scopes, credential boundary, tenant consent, rate limits, and throttling are not approved. |
| `TMS-EV-008` | Audit and no-write proof | Not implemented | No local no-write proof can be accepted until a specific tool/API surface exists and Teams preparation is approved. |
| `TMS-EV-009` | BA-EM hard-gate evidence | Not applicable yet | BA-EM-005 and BA-EM-009 must remain zero before any future Teams path. |
| `TMS-EV-010` | RAJA row decision | Blocked | A separate RAJA decision is required before any Teams preparation or execution work. |

## 3) Default Teams policy

| Capability | Default posture | Rationale |
| --- | --- | --- |
| Read Teams message content | Blocked | Privacy, retention, and channel-scope decisions are not approved. |
| Read Teams message metadata | Blocked | Metadata can still expose personal/collaboration context. |
| Send Teams message or Adaptive Card | Blocked | External side effect; write-like and approval-gated. |
| Create approval record | Blocked | Approval records are write-like and cannot be agent-created as execution approval. |
| Subscribe to Teams events/webhooks | Blocked | External side effect and data-ingestion expansion. |
| Use Graph API | Blocked | App registration, scopes, tenant consent, rate limits, and data policy are not approved. |
| Use Teams/Copilot 365 as conceptual user surface | Allowed in documentation | This preserves the product surface convention, but it does not authorize integration. |

## 4) Reopen conditions

Do not create a Teams preparation package, adapter, schema contract, or test seam until RAJA records a new decision that explicitly authorizes Teams preparation.

If RAJA later reopens Teams, the first package must define:

1. Whether the candidate is read-only, send-only, approval-only, or review-metadata-only.
2. Exact tenant/channel/app scope or approved sensitive evidence reference.
3. Privacy, retention, residency, and deletion policy.
4. App registration, auth scopes, credential storage, throttling, and degraded-mode behavior.
5. Actual Teams/Copilot 365/Microsoft Graph request and response schemas.
6. No-write proof for any unapproved send, approval, subscription, update, or external side effect.
7. Audit examples for allowed, denied, degraded, throttled, schema-mismatch, and blocked-write outcomes.
8. Explicit RAJA decision-log evidence.

Until then, Teams remains blocked and should not be included in sandbox execution planning.
