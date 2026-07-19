# BA Agent Phase 2 Git/GitHub Sandbox Evidence Package

This package prepares the evidence needed for the `P2-SBX-GIT-READ` candidate row before any Git or GitHub sandbox execution can be requested. It is a preparation artifact only and does not authorize credentials, endpoint access, GitHub API calls, repository reads, source-code processing, external artifact storage, comments, pull-request actions, webhook subscriptions, or any other write-like side effect.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Phase 2 Git/GitHub Sandbox Evidence Package |
| Version | 0.1 |
| Status | Preparation package; execution blocked; no endpoint validated |
| Prepared date | 2026-07-13 |
| Accountable owner | RAJA |
| Candidate row | `P2-SBX-GIT-READ` / `get_recent_activity` |
| Primary references | `docs/development/phase-2-sandbox-authorization-package.md` v0.8; `docs/development/mcp-validation-register.json` v0.8; `docs/development/phase-2-tool-approval-matrix.md` v0.7; `docs/planning/decision-log.md` v2.3 |
| Explicit non-authorization | No sandbox execution, live integration, pilot start, production rollout, non-synthetic data path, external tool execution, Git/GitHub read, source-code ingestion, comments, PR actions, webhook subscriptions, external publish/storage, credential use, or write-like side effect |

## 1) Current verdict

`P2-SBX-GIT-READ` is approved for **evidence preparation only**, but it is not implementation-ready.

No Git/GitHub MCP endpoint or tool family was identified in the available local MCP evidence. Therefore, no BA Agent adapter should be implemented yet. The next safe action is to collect owner/tool evidence that identifies the actual provider, allowed metadata surface, schema, auth, rate limits, and source-code classification boundary.

## 2) Evidence checklist for RAJA/tool-owner review

| Evidence ID | Required evidence | Current package status | Execution blocker |
| --- | --- | --- | --- |
| `SBX-EV-001` | Named Git/GitHub owner and review lanes | RAJA remains acting owner for preparation; repo owner, security/privacy reviewer, platform reviewer, and QA/control reviewer remain `[RAJA]`. | Name delegates or explicitly keep RAJA as acting owner for the execution row. |
| `SBX-EV-002` | Exact repository sandbox scope | Scope capture template is defined in Section 3; no repository, organization, branch, PR, commit, file, or path boundary is recorded in this repository. | Owner-approved scope must be recorded in the approved evidence location before execution. |
| `SBX-EV-003` | Data classification | Candidate classes are repository metadata, PR metadata, commit metadata, changed-file names, and source-code content only if separately approved. | Security/privacy and source-code owner must approve allowed classes and prohibited classes. |
| `SBX-EV-004` | Field minimization and redaction | Metadata-minimum policy is proposed in Section 4. | Final field allowlist, source-code policy, diff policy, author/identity policy, and secret-redaction policy remain `[RAJA]`. |
| `SBX-EV-005` | Retention, residency, and deletion | Decision template is defined in Section 5. | Retention window, audit retention, residency, deletion/archive process, and log handling remain `[RAJA]`. |
| `SBX-EV-006` | Actual request/response schema validation | Schema-capture contract is defined in Section 6; no endpoint/tool schema exists in this repo. | Tool owner/platform must provide actual endpoint/tool names, request/response schemas, and schema-diff result for the approved scope. |
| `SBX-EV-007` | Auth and rate-limit guardrails | Guardrail template is defined in Section 7. | Least-privilege auth, credential storage boundary, timeout/retry, throttling, and degraded-mode behavior remain `[RAJA]`. |
| `SBX-EV-008` | Audit and failure-mode examples | Synthetic examples are defined in Section 8. | Control reviewer must accept allowed, denied, degraded, throttled, schema-mismatch, stale-source, blocked-write, and source-code-blocked audit behavior. |
| `SBX-EV-009` | No-write proof | Not implemented because no endpoint/tool family exists. | Adapter cannot be implemented until the tool family exists; future tests must prove comments, PR updates, branch operations, pushes, webhook subscriptions, approval records, and unlisted tools are unreachable. |
| `SBX-EV-010` | RAJA row decision | Preparation approval exists; execution approval does not. | Decision log must record explicit row-level execution approval after all evidence above is complete. |

## 3) Scope capture template

The exact sandbox scope must be approved outside this repository if it contains sensitive organization, repository, endpoint, branch, PR, commit, or source-code identifiers. Do not commit secrets, credentials, tenant IDs, private endpoints, repository names, branch names, file paths, or restricted source-code values unless explicitly approved for public documentation.

| Scope field | Required owner-provided value | Current value |
| --- | --- | --- |
| Git provider | GitHub, GitHub Enterprise, local Git, or another approved provider | `[RAJA]` |
| Environment | Sandbox/non-production environment or approved evidence reference | `[RAJA]` |
| Organization/repository boundary | Approved org/repo or equivalent evidence reference | `[RAJA]` |
| Branch/tag boundary | Approved branch, tag, or release boundary | `[RAJA]` |
| PR/commit boundary | Approved PR, commit, date, author, or release-filter policy | `[RAJA]` |
| File/path boundary | Approved metadata-only path policy; source-code content blocked by default | `[RAJA]` |
| Diff policy | Blocked by default unless source-code classification review approves | `[RAJA]` |
| Comment/write policy | Comments and PR/issue writes blocked | `[RAJA]` |
| Webhook/subscription policy | Blocked by default | `[RAJA]` |
| Maximum page size | Owner/platform-approved limit | `[RAJA]` |
| Freshness window | Owner-approved source freshness boundary | `[RAJA]` |

## 4) Proposed field-minimization baseline

Default posture before owner approval:

| Field group | Proposed status | Rationale |
| --- | --- | --- |
| Repository/source ref | Candidate allow by approved reference only | Required for traceability if approved. |
| PR number/title/status | Candidate allow with review | Useful for implementation traceability; titles may expose sensitive context. |
| Commit SHA and timestamp | Candidate allow | Useful for evidence freshness and traceability. |
| Changed file names | Candidate allow with review | File paths may expose system structure; require classification review. |
| Diff/source-code content | Block by default | Higher risk of restricted source code, secrets, IP, and sensitive business logic. |
| Author names/emails | Redact or block by default | Personal data minimization. |
| Comments/review text | Block by default | Collaboration-context and personal data risk. |
| Links | Candidate allow with redaction | Useful for traceability, but URLs may expose internal identifiers. |
| Webhook payloads | Block by default | External side-effect and data-volume risk. |

Final policy remains `[RAJA]` until security/privacy, source-code owner, and tool-owner review approve it.

## 5) Retention, residency, and deletion decision template

| Decision area | Required decision | Current value |
| --- | --- | --- |
| Input retention | Whether sandbox Git/GitHub responses may be stored, and for how long | `[RAJA]` |
| Output retention | Whether generated discovery artifacts may retain Git-derived evidence refs and summaries | `[RAJA]` |
| Audit retention | Required audit-record retention duration and storage boundary | `[RAJA]` |
| Residency | Approved region/tenant/data boundary for any retained artifact | `[RAJA]` |
| Deletion | Deletion/archive procedure for sandbox evidence and derived local artifacts | `[RAJA]` |
| Log policy | Fields allowed in logs; secrets, raw restricted values, and source-code content remain prohibited by default | `[RAJA]` |

## 6) Schema validation contract

Before the register can move to `validation_status: validated`, identify the actual Git/GitHub tool family and capture request/response schemas for the approved scope.

| Candidate operation | Request schema evidence required | Response schema evidence required | Current status |
| --- | --- | --- | --- |
| Repository metadata read | JSON schema or field list for provider/org/repo lookup arguments | JSON schema or field list for repository metadata, visibility, timestamps, and errors | No endpoint/tool schema identified |
| Recent activity read | JSON schema or field list for PR/commit/date/path filters | JSON schema or field list for PR/commit metadata, pagination, nullable fields, and errors | No endpoint/tool schema identified |
| Pull request metadata read | JSON schema or field list for PR lookup arguments | JSON schema or field list for PR metadata without comments/content unless approved | No endpoint/tool schema identified |
| Commit metadata read | JSON schema or field list for commit lookup arguments | JSON schema or field list for commit metadata without diff/source content unless approved | No endpoint/tool schema identified |
| File metadata read | JSON schema or field list for path lookup arguments | JSON schema or field list for file metadata; source content blocked unless separately approved | No endpoint/tool schema identified |

Schema evidence must include:

1. Capture date/time and source of schema evidence.
2. Schema owner or reviewer.
3. Diff against BA Agent expected fields.
4. Handling for missing, nullable, unexpected, stale, restricted, or extra fields.
5. Confirmation that schema examples are synthetic or redacted and contain no secrets, source-code content, or restricted data values unless explicitly approved.

## 7) Auth, rate-limit, and degraded-mode template

| Control | Required evidence | Current value |
| --- | --- | --- |
| Auth mechanism | Least-privilege auth approach and scope list | `[RAJA]` |
| Credential storage | Key Vault/managed identity or approved local secret boundary; no committed secrets | `[RAJA]` |
| TLS/network boundary | Approved endpoint and transport security posture | `[RAJA]` |
| Rate limits | Tool-owner-approved limits and client-side throttling policy | `[RAJA]` |
| Timeout policy | Per-call timeout and total-run timeout | `[RAJA]` |
| Retry policy | Retry count/backoff and non-retryable errors | `[RAJA]` |
| Degraded mode | User-visible behavior when Git/GitHub is unavailable, denied, throttled, or schema-stale | `[RAJA]` |
| Kill switch | How the row returns to blocked/synthetic fallback | `[RAJA]` |

## 8) Synthetic audit and failure-mode examples

These examples describe the audit shape expected from a future approved row. They are synthetic and do not prove execution readiness.

| Scenario | Expected status | Required audit fields | Evidence/ref behavior |
| --- | --- | --- | --- |
| Allowed metadata read | `ok` | `trace_id`, candidate row, allowed upstream tool, input hash, approved scope ref, source system, `source_timestamp`, `retrieved_at`, result status | Evidence refs point to approved Git/GitHub source refs or redacted synthetic placeholders in dry runs. |
| Denied scope | `denied` | Same core audit fields plus denial reason | No upstream call beyond authorization/scope check; no raw data stored. |
| Source-code content requested before approval | `blocked` | Same core audit fields plus source-code policy reason | No source-code content retrieved or stored. |
| Throttled | `throttled` | Same core audit fields plus retry/degraded metadata | Output reports degraded state and asks reviewer to retry or use synthetic fallback. |
| Degraded upstream | `degraded` | Same core audit fields plus upstream health/error class | Draft output marks source coverage incomplete and avoids unsupported conclusions. |
| Schema mismatch | `blocked` | Same core audit fields plus schema version/diff reference | Row is marked stale; sandbox execution stops until validation is refreshed. |
| Stale source | `degraded` | Same core audit fields plus source freshness window | Output labels source staleness and routes to review. |
| Write-like tool requested | `blocked` or `rejected` | Same core audit fields plus denied tool name and no-write control reference | No upstream call occurs; BA-EM-005 remains zero. |

Minimum audit assertions for future tests:

1. Every allowed or denied sandbox tool attempt emits an audit record.
2. `source_timestamp` and `retrieved_at` remain distinct when source data is returned.
3. Input values are hashed or minimized; secrets, raw restricted values, and source-code content are not logged.
4. Denied write-like and unlisted tools do not call the upstream client.
5. Schema mismatch, missing approval, and unapproved source-code content keep the row blocked.

## 9) Register update conditions

Do not update the `get_recent_activity` row to `implementation_status: ready` or `validation_status: validated` until all of the following are true:

1. `mcp_server_name` identifies the actual approved Git/GitHub MCP server or read endpoint.
2. `approved_scopes` contains at least one approved, non-sensitive scope reference.
3. `actual_request_schema_ref`, `actual_response_schema_ref`, and `schema_diff_ref` point to approved evidence.
4. `auth_model_ref` and `rate_limit_ref` point to approved evidence.
5. `approval_evidence_ref` points to the explicit RAJA execution authorization decision.
6. `validated_at` is populated.
7. `open_blockers` is empty.
8. BA-EM-005 and BA-EM-009 remain zero in the relevant regression run.

Until then, the checked-in register must continue to block adapter construction for real execution.

## 10) Recommended next owner action

Route this package to RAJA/Git or GitHub owner review and collect evidence for `SBX-EV-001` through `SBX-EV-010`.

If no Git/GitHub MCP endpoint or approved read API is available, keep `P2-SBX-GIT-READ` in preparation-only status and continue using synthetic fixtures as the fallback.
