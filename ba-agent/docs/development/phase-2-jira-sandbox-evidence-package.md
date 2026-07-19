# BA Agent Phase 2 Jira Sandbox Evidence Package

This package prepares the evidence needed for the `P2-SBX-JIRA-READ` candidate row before any Jira sandbox execution can be requested. It is a preparation artifact only and does not authorize credentials, endpoint access, Jira calls, non-synthetic data processing, external artifact storage, or any write-like side effect.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Phase 2 Jira Sandbox Evidence Package |
| Version | 0.1 |
| Status | Preparation package; execution blocked |
| Prepared date | 2026-07-13 |
| Accountable owner | RAJA |
| Candidate row | `P2-SBX-JIRA-READ` / `get_sprint_status` |
| Primary references | `docs/development/phase-2-sandbox-authorization-package.md` v0.6; `docs/development/mcp-validation-register.json` v0.6; `src/ba_agent/phase2/sandbox_mcp.py`; `tests/phase2/test_sandbox_mcp.py`; `docs/planning/decision-log.md` v2.1 |
| Explicit non-authorization | No sandbox execution, live integration, pilot start, production rollout, non-synthetic data path, external tool execution, external publish/storage, credential use, or write-like side effect |

## 1) Current verdict

`P2-SBX-JIRA-READ` is ready for **owner evidence collection**, not execution.

The BA Agent has local preparation evidence for a read-only wrapper over the candidate `/jira-cloud/mcp` upstream. The wrapper allowlists `FetchItrackJiraIssuesList`, `GetJiraItrackJobStatus`, and `JiraItrackValidate`, and blocks advertised Jira write/destructive tools before any upstream client call. The validation register still blocks real adapter construction because the row is not fully validated for execution.

## 2) Evidence checklist for RAJA/tool-owner review

| Evidence ID | Required evidence | Current package status | Execution blocker |
| --- | --- | --- | --- |
| `SBX-EV-001` | Named Jira owner and review lanes | RAJA remains acting owner for preparation; Jira tool owner, security/privacy reviewer, platform reviewer, and QA/control reviewer remain `[RAJA]`. | Name delegates or explicitly keep RAJA as acting owner for the execution row. |
| `SBX-EV-002` | Exact Jira sandbox scope | Scope capture template is defined in Section 3; no project key, board, JQL, issue type, tenant endpoint, or field boundary is recorded in this repository. | Owner-approved scope must be recorded in the approved evidence location before execution. |
| `SBX-EV-003` | Data classification | Candidate classes are issue metadata, epics/stories, labels, statuses, links, timestamps, and optionally comments; no approval is recorded. | Security/privacy must approve allowed classes and prohibited classes. |
| `SBX-EV-004` | Field minimization and redaction | Metadata-minimum policy is proposed in Section 4. | Final field allowlist, redaction, and comment/attachment policy remain `[RAJA]`. |
| `SBX-EV-005` | Retention, residency, and deletion | Decision template is defined in Section 5. | Retention window, audit retention, residency, deletion/archive process, and log handling remain `[RAJA]`. |
| `SBX-EV-006` | Actual request/response schema validation | Schema-capture contract is defined in Section 6; local tools-list evidence is partial and not sufficient for execution. | Tool owner/platform must provide request/response schemas and schema-diff result for the approved scope. |
| `SBX-EV-007` | Auth and rate-limit guardrails | Guardrail template is defined in Section 7; current local evidence has auth/TLS/rate limiting disabled. | Least-privilege auth, credential storage boundary, timeout/retry, throttling, and degraded-mode behavior remain `[RAJA]`. |
| `SBX-EV-008` | Audit and failure-mode examples | Synthetic examples are defined in Section 8. | Control reviewer must accept allowed, denied, degraded, throttled, schema-mismatch, stale-source, and blocked-write audit behavior. |
| `SBX-EV-009` | No-write proof | Local tests prove advertised Jira write/destructive tools are denied before upstream calls. | Security/control review must accept the allowlist and confirm comments, updates, deletes, approval tools, subscriptions, and unlisted tools remain unreachable. |
| `SBX-EV-010` | RAJA row decision | Preparation approval exists; execution approval does not. | Decision log must record explicit row-level execution approval after all evidence above is complete. |

## 3) Scope capture template

The exact sandbox scope must be approved outside this repository if it contains sensitive tenant, project, or endpoint identifiers. Do not commit secrets, credentials, tenant IDs, private endpoints, or restricted data values.

| Scope field | Required owner-provided value | Current value |
| --- | --- | --- |
| Jira environment | Sandbox/non-production environment identifier or approved evidence reference | `[RAJA]` |
| Jira project or board boundary | Approved project key, board, or equivalent evidence reference | `[RAJA]` |
| Query boundary | Approved JQL or query policy, including date/status/issue-type limits | `[RAJA]` |
| Issue types | Approved issue types, such as epics/stories/tasks/bugs, if allowed | `[RAJA]` |
| Field boundary | Allowed, redacted, and prohibited fields | `[RAJA]` |
| Comment policy | Blocked by default; allowed only if privacy review approves | `[RAJA]` |
| Attachment policy | Blocked by default | `[RAJA]` |
| Link policy | Allowed only if approved and redacted as needed | `[RAJA]` |
| Maximum page size | Owner/platform-approved limit | `[RAJA]` |
| Time window | Owner-approved lookback or sprint boundary | `[RAJA]` |

## 4) Proposed field-minimization baseline

Default posture before owner approval:

| Field group | Proposed status | Rationale |
| --- | --- | --- |
| Issue key/source ref | Candidate allow | Required for traceability if approved. |
| Issue type/status/labels | Candidate allow | Useful for requirement discovery and lower sensitivity than full body text. |
| Summary/title | Candidate allow with review | May contain sensitive business context; needs classification approval. |
| Description/body | Block by default | Higher risk of restricted business, customer, or personal data. |
| Comments | Block by default | Higher privacy and collaboration-context risk. |
| Attachments | Block by default | Unknown classification and retention risk. |
| Reporter/assignee/display names | Redact or block by default | Personal data minimization. |
| Links | Candidate allow with redaction | Useful for traceability, but URLs may expose internal identifiers. |
| Timestamps | Candidate allow | Needed for source freshness and audit if approved. |

Final policy remains `[RAJA]` until security/privacy and tool-owner review approve it.

## 5) Retention, residency, and deletion decision template

| Decision area | Required decision | Current value |
| --- | --- | --- |
| Input retention | Whether sandbox Jira responses may be stored, and for how long | `[RAJA]` |
| Output retention | Whether generated discovery artifacts may retain Jira-derived evidence refs and summaries | `[RAJA]` |
| Audit retention | Required audit-record retention duration and storage boundary | `[RAJA]` |
| Residency | Approved region/tenant/data boundary for any retained artifact | `[RAJA]` |
| Deletion | Deletion/archive procedure for sandbox evidence and derived local artifacts | `[RAJA]` |
| Log policy | Fields allowed in logs; secrets and raw restricted values remain prohibited | `[RAJA]` |

## 6) Schema validation contract

Before the register can move to `validation_status: validated`, capture actual request and response schemas for the approved scope.

| Allowed upstream tool | Request schema evidence required | Response schema evidence required | Current status |
| --- | --- | --- | --- |
| `FetchItrackJiraIssuesList` | JSON schema or field list for the approved query/scope arguments | JSON schema or field list for returned issues, pagination/status metadata, nullable fields, and errors | Partial tools-list evidence only; response schema incomplete |
| `GetJiraItrackJobStatus` | JSON schema or field list for job identifier and status lookup arguments | JSON schema or field list for status response, terminal/error states, and timestamps | Partial local output schema observed; not validated for execution |
| `JiraItrackValidate` | JSON schema or field list for validation request arguments | JSON schema or field list for validation result and mismatch/error fields | Partial local output schema observed; not validated for execution |

Schema evidence must include:

1. Capture date/time and source of schema evidence.
2. Schema owner or reviewer.
3. Diff against BA Agent expected fields.
4. Handling for missing, nullable, unexpected, stale, or extra fields.
5. Confirmation that schema examples are synthetic or redacted and contain no secrets or restricted data values.

## 7) Auth, rate-limit, and degraded-mode template

| Control | Required evidence | Current value |
| --- | --- | --- |
| Auth mechanism | Least-privilege auth approach and scope list | `[RAJA]` |
| Credential storage | Key Vault/managed identity or approved local secret boundary; no committed secrets | `[RAJA]` |
| TLS/network boundary | Approved endpoint and transport security posture | `[RAJA]` |
| Rate limits | Tool-owner-approved limits and client-side throttling policy | `[RAJA]` |
| Timeout policy | Per-call timeout and total-run timeout | `[RAJA]` |
| Retry policy | Retry count/backoff and non-retryable errors | `[RAJA]` |
| Degraded mode | User-visible behavior when Jira is unavailable, denied, throttled, or schema-stale | `[RAJA]` |
| Kill switch | How the row returns to blocked/synthetic fallback | `[RAJA]` |

## 8) Synthetic audit and failure-mode examples

These examples describe the audit shape expected from a future approved row. They are synthetic and do not prove execution readiness.

| Scenario | Expected status | Required audit fields | Evidence/ref behavior |
| --- | --- | --- | --- |
| Allowed read | `ok` | `trace_id`, candidate row, allowed upstream tool, input hash, approved scope ref, source system, `source_timestamp`, `retrieved_at`, result status | Evidence refs point to approved Jira source refs or redacted synthetic placeholders in dry runs. |
| Denied scope | `denied` | Same core audit fields plus denial reason | No upstream call beyond authorization/scope check; no raw data stored. |
| Throttled | `throttled` | Same core audit fields plus retry/degraded metadata | Output reports degraded state and asks reviewer to retry or use synthetic fallback. |
| Degraded upstream | `degraded` | Same core audit fields plus upstream health/error class | Draft output marks source coverage incomplete and avoids unsupported conclusions. |
| Schema mismatch | `blocked` | Same core audit fields plus schema version/diff reference | Row is marked stale; sandbox execution stops until validation is refreshed. |
| Stale source | `degraded` | Same core audit fields plus source freshness window | Output labels source staleness and routes to review. |
| Write-like tool requested | `blocked` or `rejected` | Same core audit fields plus denied tool name and no-write control reference | No upstream call occurs; BA-EM-005 remains zero. |
| Unlisted tool requested | `blocked` | Same core audit fields plus allowlist miss reason | No upstream call occurs. |

Minimum audit assertions for future tests:

1. Every allowed or denied sandbox tool attempt emits an audit record.
2. `source_timestamp` and `retrieved_at` remain distinct when source data is returned.
3. Input values are hashed or minimized; secrets and raw restricted values are not logged.
4. Denied write-like tools do not call the upstream client.
5. Schema mismatch and missing approval keep the row blocked.

## 9) Register update conditions

Do not update the `get_sprint_status` row to `implementation_status: ready` or `validation_status: validated` until all of the following are true:

1. `approved_scopes` contains at least one approved, non-sensitive scope reference.
2. `actual_request_schema_ref`, `actual_response_schema_ref`, and `schema_diff_ref` point to approved evidence.
3. `auth_model_ref` and `rate_limit_ref` point to approved evidence.
4. `approval_evidence_ref` points to the explicit RAJA execution authorization decision.
5. `validated_at` is populated.
6. `open_blockers` is empty.
7. BA-EM-005 and BA-EM-009 remain zero in the relevant regression run.

Until then, the checked-in register must continue to block adapter construction for real execution.

## 10) Recommended next owner action

Route this package to RAJA/tool-owner review and collect evidence for `SBX-EV-001` through `SBX-EV-010`.

If any owner cannot provide complete scope, classification, schema, auth, rate-limit, audit, or no-write evidence, keep `P2-SBX-JIRA-READ` in preparation-only status and continue using synthetic fixtures as the fallback.
