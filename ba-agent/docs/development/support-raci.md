# BA Agent Pilot Support and RACI

This support model preserves RAJA as accountable owner while documenting review lanes needed before pilot execution.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Pilot Support and RACI |
| Version | 0.1 |
| Status | Draft for G6 authorization review |
| Prepared date | 2026-07-03 |
| Accountable owner | RAJA |
| Execution prompt | [P6B] |

## RACI baseline

RAJA remains accountable for this baseline. Role-specific delegates remain `[RAJA]` until named.

| Activity | Responsible | Accountable | Consulted | Informed |
| --- | --- | --- | --- | --- |
| Pilot scope decision | Product/Delivery lane [RAJA] | RAJA | Security, platform, tool owners | Pilot users [RAJA] |
| Output review | BA SME / QA lane [RAJA] | RAJA | Scrum Master [RAJA] | Delivery lane [RAJA] |
| Tool validation | Tool-owner lane [RAJA] | RAJA | Platform/security lanes [RAJA] | Product lane [RAJA] |
| Security/privacy review | Security/privacy lane [RAJA] | RAJA | Platform/tool-owner lanes [RAJA] | All pilot stakeholders |
| Incident response | Platform/security lane [RAJA] | RAJA | Delivery/architect lanes [RAJA] | Affected stakeholders |
| Rollback/kill switch | Platform lane [RAJA] | RAJA | Security/QA lanes [RAJA] | Pilot users [RAJA] |

## Support tiers

| Tier | Handles | Escalates when |
| --- | --- | --- |
| L1 — usage/output questions | Evidence-linked explanation questions, advisory/draft-label confusion, user education. | Missing evidence trail, unexpected tool status, or likely incorrect output. |
| L2 — platform/tool issues | Gateway errors, degraded/denied/throttled behavior, command failures, trace lookup. | Gate bypass, data exposure, failed audit, or repeated tool failure. |
| L3 — security/control incident | Approval-gate bypass, data exposure, prompt-injection incident, unauthorized write-like side effect. | Immediate stop condition; preserve evidence and disable affected capability/tool. |

## Incident triggers

| Trigger | Severity | Required action |
| --- | --- | --- |
| Write-like side effect succeeds without valid approval | Sev1 | Stop pilot, disable writes, preserve audit, notify RAJA/security lane. |
| Data exposure or unauthorized scope access | Sev1 | Stop pilot, preserve trace/audit, engage security/privacy lane. |
| Phase 2 capability exposed in MVP pilot | Sev2 | Disable route/capability, rerun phase-separation eval. |
| Output lacks evidence refs | Sev2 | Stop affected output path, review trace/audit, rerun evals. |
| Single integration degraded honestly | Sev3 | Continue only if data-quality status is visible and scope remains approved. |

## Traceability for support

Support analysis starts from:

1. `trace_id`
2. eval run ID
3. fixture version or validated tool schema version
4. gateway audit record
5. prompt/graph version

No support path should require reproducing a live issue with broader permissions.

## Operational commitments

No 24x7 coverage, SLA, named on-call staff, release cadence, watch window, or retention period is approved in this document. Those values remain `[RAJA]`.
