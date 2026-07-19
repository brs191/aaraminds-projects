# BA Agent G6 Authorization Package

This package prepares G6 authorization review. It does not authorize live pilot execution and does not enable live configuration.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent G6 Authorization Package |
| Version | 0.1 |
| Status | Blocked pending external non-agent-controlled RAJA approval and required review evidence |
| Prepared date | 2026-07-03 |
| Accountable owner | RAJA |
| Execution prompt | [P6E] |
| G5 evidence | `docs/development/g5-candidate-review.md` |

## Authorization verdict

G6 authorization is **not complete**. This package is ready for RAJA/reviewer inspection, but live pilot execution is blocked because required external approval and scope artifacts are missing.

## Required evidence checklist

| Evidence | Status |
| --- | --- |
| G5 candidate evidence | Present: `docs/development/g5-candidate-review.md` |
| Release notes with harness run IDs | Present: `docs/development/pilot-release-notes.md` |
| Security/privacy/classification approval | Missing [RAJA] |
| Tool-owner approvals for exact Jira project/repo/Teams/Confluence/calendar scopes | Missing [RAJA] |
| Teams tenant/app/channel approval | Missing [RAJA] |
| Support/RACI sign-off | Draft only: `docs/development/support-raci.md` |
| Rollback/kill-switch drill evidence | Present: `docs/development/rollback-kill-switch-drill.md` |
| Limited pilot scope/start-stop criteria | Draft only: `docs/development/pilot-runbook.md` |
| External non-agent-controlled RAJA approval artifact | Missing |

## Pilot execution status

Pilot execution is blocked. `[P6F]` must not run until an external non-agent-controlled RAJA approval artifact exists and matches the exact pilot scope.

## Live enablement status

| Item | Status |
| --- | --- |
| Live Jira/Git/Confluence/Calendar/Teams/Graph/model/MCP access | Not enabled |
| Live writes | Not enabled |
| Teams posting | Not enabled |
| Sandbox read replacement | Not enabled |
| Production deployment | Not enabled |

## Required next decisions

1. Confirm exact pilot team/project/repo/channel/calendar/Confluence scopes.
2. Complete security/privacy/classification review.
3. Complete tool-owner validation for required reads.
4. Complete Teams sandbox/channel approval.
5. Record external non-agent-controlled RAJA approval artifact.

Until those are complete, F6 stops at authorization readiness.
