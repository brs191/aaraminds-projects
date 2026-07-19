# BA Agent MVP Pilot Runbook

This runbook prepares a controlled MVP pilot. It does not authorize live use. Limited live execution requires explicit external, non-agent-controlled RAJA approval recorded in the G6 authorization package.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent MVP Pilot Runbook |
| Version | 0.1 |
| Status | Draft for G6 authorization review |
| Prepared date | 2026-07-03 |
| Accountable owner | RAJA |
| Execution prompt | [P6A] |
| G5 evidence | `docs/development/g5-candidate-review.md` |

## Pilot objective

Validate whether the MVP Scrum-support BA Agent can provide evidence-linked standup, planning, retrospective, and health-monitoring assistance within a narrow approved pilot scope.

## Candidate pilot scope

All scope values are `[RAJA]` until confirmed through a non-agent-controlled approval artifact.

| Scope item | Candidate value | Status |
| --- | --- | --- |
| Pilot team | [RAJA] | Not approved |
| Jira project | [RAJA] | Not approved |
| Git repository | [RAJA] | Not approved |
| Teams channel | [RAJA] | Not approved |
| Confluence space | [RAJA] | Not approved |
| Calendar scope | [RAJA] | Not approved |
| Approvers | [RAJA] | Not approved |

## Entry criteria

The pilot cannot start until all are complete:

1. G5 accepted by RAJA.
2. Security/privacy/classification handling approved.
3. Exact pilot scopes approved.
4. Tool-owner validation completed for any sandbox/live read.
5. Teams tenant/app/channel approval completed.
6. Support/RACI package reviewed.
7. Rollback/kill-switch drill completed.
8. Release notes prepared with current harness run IDs.
9. External non-agent-controlled RAJA approval artifact is recorded.

## In-scope capabilities

- Standup summary assistance.
- Sprint planning recommendation only; no sprint-scope publish.
- Retrospective draft-only report generation.
- Sprint health advisory findings only.

## Out of scope

- Phase 2 Enterprise BA capabilities.
- Any autonomous system-of-record update.
- Jira sprint scope mutation.
- Confluence publish.
- Teams escalation/send without validated approval.
- Production rollout.

## Data handling and evidence expectations

- Every factual claim must carry evidence refs.
- Every run must carry `trace_id`.
- Prompt/output retention, audit retention, and data residency are `[RAJA]` until approved.
- Calendar data remains aggregate-only; no event titles, bodies, or attendee details.

## Operating cadence

| Activity | Cadence | Owner |
| --- | --- | --- |
| Pilot check-in | [RAJA] | RAJA / delegate [RAJA] |
| Audit spot-check | [RAJA] | Security/privacy lane [RAJA] |
| Output sample review | [RAJA] | BA SME / QA lane [RAJA] |
| Support triage review | [RAJA] | Delivery/platform lane [RAJA] |

## Stop conditions

Stop the pilot immediately if any occur:

1. BA-EM-005 approval-gate bypass count is greater than zero.
2. Data exposure or unauthorized scope access is suspected.
3. Any unapproved write-like side effect succeeds.
4. Phase 2 capability is exposed in the MVP pilot.
5. Evidence-link coverage collapses or factual output cannot be traced.
6. Severe output regression appears in sampled review.

## Rollback / fallback

- Disable write-like tools at the gateway/control layer.
- Disable affected capability through local capability controls where supported.
- Revert to synthetic-only mode.
- Preserve audit evidence and trace IDs.
- Do not retry with broader permissions.
