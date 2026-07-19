# BA Agent Phase 2 Confluence Sandbox Evidence Package

This package prepares the evidence needed for the `P2-SBX-CONF-READ` candidate row before any Confluence sandbox execution can be requested. It is a preparation artifact only and does not authorize credentials, endpoint access, Confluence calls, non-synthetic data processing, external artifact storage, draft creation, publishing, comments, or any other write-like side effect.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Phase 2 Confluence Sandbox Evidence Package |
| Version | 0.1 |
| Status | Preparation package; execution blocked |
| Prepared date | 2026-07-13 |
| Accountable owner | RAJA |
| Candidate row | `P2-SBX-CONF-READ` / `get_confluence_source_pages` |
| Primary references | `docs/development/phase-2-sandbox-authorization-package.md` v0.7; `docs/development/mcp-validation-register.json` v0.7; `src/ba_agent/phase2/sandbox_mcp.py`; `tests/phase2/test_sandbox_mcp.py`; `docs/planning/decision-log.md` v2.2 |
| Explicit non-authorization | No sandbox execution, live integration, pilot start, production rollout, non-synthetic data path, external tool execution, Confluence draft/publish/comment action, external publish/storage, credential use, or write-like side effect |

## 1) Current verdict

`P2-SBX-CONF-READ` is ready for **owner evidence collection**, not execution.

The BA Agent now has a local preparation wrapper for the candidate broad root MCP surface. The wrapper allowlists only `confluence_search`, `confluence_get_page`, `confluence_list_spaces`, `confluence_space_pages`, `confluence_page_children`, `confluence_page_attachments`, and `confluence_page_comments`. Write-like Confluence tools and unrelated root-surface tools are blocked before any upstream client call. The validation register still blocks real adapter construction because the row is not fully validated for execution.

## 2) Evidence checklist for RAJA/tool-owner review

| Evidence ID | Required evidence | Current package status | Execution blocker |
| --- | --- | --- | --- |
| `SBX-EV-001` | Named Confluence owner and review lanes | RAJA remains acting owner for preparation; Confluence space owner, security/privacy reviewer, platform reviewer, and QA/control reviewer remain `[RAJA]`. | Name delegates or explicitly keep RAJA as acting owner for the execution row. |
| `SBX-EV-002` | Exact Confluence sandbox scope | Scope capture template is defined in Section 3; no space key, page tree, label filter, tenant endpoint, or restricted-page boundary is recorded in this repository. | Owner-approved scope must be recorded in the approved evidence location before execution. |
| `SBX-EV-003` | Data classification | Candidate classes are page metadata, page titles, labels, hierarchy, attachments metadata, comments metadata, and page bodies only if separately approved. | Security/privacy must approve allowed classes and prohibited classes. |
| `SBX-EV-004` | Field minimization and redaction | Metadata-minimum policy is proposed in Section 4. | Final field allowlist, body-content policy, attachment policy, comment policy, redaction, and restricted-page handling remain `[RAJA]`. |
| `SBX-EV-005` | Retention, residency, and deletion | Decision template is defined in Section 5. | Retention window, audit retention, residency, deletion/archive process, and log handling remain `[RAJA]`. |
| `SBX-EV-006` | Actual request/response schema validation | Schema-capture contract is defined in Section 6; local tools-list evidence is partial and not sufficient for execution. | Tool owner/platform must provide request/response schemas and schema-diff result for the approved scope. |
| `SBX-EV-007` | Auth and rate-limit guardrails | Guardrail template is defined in Section 7; current local evidence has auth/TLS/rate limiting disabled. | Least-privilege auth, credential storage boundary, timeout/retry, throttling, and degraded-mode behavior remain `[RAJA]`. |
| `SBX-EV-008` | Audit and failure-mode examples | Synthetic examples are defined in Section 8. | Control reviewer must accept allowed, denied, degraded, throttled, schema-mismatch, stale-source, blocked-write, and unlisted-root-tool audit behavior. |
| `SBX-EV-009` | No-write proof | Local tests prove Confluence write-like and unrelated root-surface tools are denied before upstream calls. | Security/control review must accept the allowlist and confirm drafts, publishes, edits, deletes, comments, labels, approval tools, subscriptions, and unlisted tools remain unreachable. |
| `SBX-EV-010` | RAJA row decision | Preparation approval exists; execution approval does not. | Decision log must record explicit row-level execution approval after all evidence above is complete. |

## 3) Scope capture template

The exact sandbox scope must be approved outside this repository if it contains sensitive tenant, space, page, endpoint, or document identifiers. Do not commit secrets, credentials, tenant IDs, private endpoints, restricted page names, or restricted data values.

| Scope field | Required owner-provided value | Current value |
| --- | --- | --- |
| Confluence environment | Sandbox/non-production environment identifier or approved evidence reference | `[RAJA]` |
| Space boundary | Approved space key or equivalent evidence reference | `[RAJA]` |
| Page boundary | Approved page IDs, parent page, page tree, labels, or CQL boundary | `[RAJA]` |
| Restricted-page policy | Whether restricted pages are blocked, redacted, or allowed | `[RAJA]` |
| Body-content policy | Blocked by default; allowed only if classification review approves | `[RAJA]` |
| Comment policy | Metadata/read-only only if approved; writes remain blocked | `[RAJA]` |
| Attachment policy | Metadata blocked by default unless approved; file content blocked | `[RAJA]` |
| Label policy | Candidate read-only if approved; label creation/removal blocked | `[RAJA]` |
| Maximum page size | Owner/platform-approved limit | `[RAJA]` |
| Freshness window | Owner-approved source freshness boundary | `[RAJA]` |

## 4) Proposed field-minimization baseline

Default posture before owner approval:

| Field group | Proposed status | Rationale |
| --- | --- | --- |
| Space key/source ref | Candidate allow | Required for traceability if approved. |
| Page ID and title | Candidate allow with review | Useful for context, but title can expose sensitive business terms. |
| Page labels and hierarchy | Candidate allow with review | Useful for scoping and traceability. |
| Page body | Block by default | Higher risk of restricted business, customer, regulatory, or personal data. |
| Comments | Block by default except metadata if approved | Collaboration context and personal data risk. |
| Attachments | Block by default | Unknown classification and retention risk. |
| Authors/display names | Redact or block by default | Personal data minimization. |
| Links | Candidate allow with redaction | Useful for traceability, but URLs may expose internal identifiers. |
| Timestamps | Candidate allow | Needed for source freshness and audit if approved. |

Final policy remains `[RAJA]` until security/privacy and tool-owner review approve it.

## 5) Retention, residency, and deletion decision template

| Decision area | Required decision | Current value |
| --- | --- | --- |
| Input retention | Whether sandbox Confluence responses may be stored, and for how long | `[RAJA]` |
| Output retention | Whether generated discovery artifacts may retain Confluence-derived evidence refs and summaries | `[RAJA]` |
| Audit retention | Required audit-record retention duration and storage boundary | `[RAJA]` |
| Residency | Approved region/tenant/data boundary for any retained artifact | `[RAJA]` |
| Deletion | Deletion/archive procedure for sandbox evidence and derived local artifacts | `[RAJA]` |
| Log policy | Fields allowed in logs; secrets and raw restricted values remain prohibited | `[RAJA]` |

## 6) Schema validation contract

Before the register can move to `validation_status: validated`, capture actual request and response schemas for the approved scope.

| Allowed upstream tool | Request schema evidence required | Response schema evidence required | Current status |
| --- | --- | --- | --- |
| `confluence_search` | JSON schema or field list for approved CQL/search arguments | JSON schema or field list for search results, pagination, nullable fields, and errors | Partial tools-list evidence only; not validated for execution |
| `confluence_get_page` | JSON schema or field list for page lookup arguments | JSON schema or field list for page metadata/body fields allowed by policy | Partial tools-list evidence only; not validated for execution |
| `confluence_list_spaces` | JSON schema or field list for list/filter arguments | JSON schema or field list for space metadata and pagination | Partial tools-list evidence only; not validated for execution |
| `confluence_space_pages` | JSON schema or field list for space/page-list arguments | JSON schema or field list for page metadata and pagination | Partial tools-list evidence only; not validated for execution |
| `confluence_page_children` | JSON schema or field list for child-page arguments | JSON schema or field list for child metadata and hierarchy fields | Partial tools-list evidence only; not validated for execution |
| `confluence_page_attachments` | JSON schema or field list for attachment-metadata arguments | JSON schema or field list for attachment metadata, excluding file content unless separately approved | Partial tools-list evidence only; not validated for execution |
| `confluence_page_comments` | JSON schema or field list for comment-read arguments | JSON schema or field list for allowed comment metadata/content, if approved | Partial tools-list evidence only; not validated for execution |

Schema evidence must include:

1. Capture date/time and source of schema evidence.
2. Schema owner or reviewer.
3. Diff against BA Agent expected fields.
4. Handling for missing, nullable, unexpected, stale, restricted, or extra fields.
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
| Degraded mode | User-visible behavior when Confluence is unavailable, denied, throttled, or schema-stale | `[RAJA]` |
| Kill switch | How the row returns to blocked/synthetic fallback | `[RAJA]` |

## 8) Synthetic audit and failure-mode examples

These examples describe the audit shape expected from a future approved row. They are synthetic and do not prove execution readiness.

| Scenario | Expected status | Required audit fields | Evidence/ref behavior |
| --- | --- | --- | --- |
| Allowed read | `ok` | `trace_id`, candidate row, allowed upstream tool, input hash, approved scope ref, source system, `source_timestamp`, `retrieved_at`, result status | Evidence refs point to approved Confluence source refs or redacted synthetic placeholders in dry runs. |
| Denied scope | `denied` | Same core audit fields plus denial reason | No upstream call beyond authorization/scope check; no raw data stored. |
| Throttled | `throttled` | Same core audit fields plus retry/degraded metadata | Output reports degraded state and asks reviewer to retry or use synthetic fallback. |
| Degraded upstream | `degraded` | Same core audit fields plus upstream health/error class | Draft output marks source coverage incomplete and avoids unsupported conclusions. |
| Schema mismatch | `blocked` | Same core audit fields plus schema version/diff reference | Row is marked stale; sandbox execution stops until validation is refreshed. |
| Stale source | `degraded` | Same core audit fields plus source freshness window | Output labels source staleness and routes to review. |
| Write-like tool requested | `blocked` or `rejected` | Same core audit fields plus denied tool name and no-write control reference | No upstream call occurs; BA-EM-005 remains zero. |
| Unlisted root-surface tool requested | `blocked` | Same core audit fields plus allowlist miss reason | No upstream call occurs. |

Minimum audit assertions for future tests:

1. Every allowed or denied sandbox tool attempt emits an audit record.
2. `source_timestamp` and `retrieved_at` remain distinct when source data is returned.
3. Input values are hashed or minimized; secrets and raw restricted values are not logged.
4. Denied write-like and unlisted root-surface tools do not call the upstream client.
5. Schema mismatch and missing approval keep the row blocked.

## 9) Register update conditions

Do not update the `get_confluence_source_pages` row to `implementation_status: ready` or `validation_status: validated` until all of the following are true:

1. `approved_scopes` contains at least one approved, non-sensitive scope reference.
2. `actual_request_schema_ref`, `actual_response_schema_ref`, and `schema_diff_ref` point to approved evidence.
3. `auth_model_ref` and `rate_limit_ref` point to approved evidence.
4. `approval_evidence_ref` points to the explicit RAJA execution authorization decision.
5. `validated_at` is populated.
6. `open_blockers` is empty.
7. BA-EM-005 and BA-EM-009 remain zero in the relevant regression run.

Until then, the checked-in register must continue to block adapter construction for real execution.

## 10) Recommended next owner action

Route this package to RAJA/Confluence owner review and collect evidence for `SBX-EV-001` through `SBX-EV-010`.

If any owner cannot provide complete scope, classification, schema, auth, rate-limit, audit, or no-write evidence, keep `P2-SBX-CONF-READ` in preparation-only status and continue using synthetic fixtures as the fallback.
