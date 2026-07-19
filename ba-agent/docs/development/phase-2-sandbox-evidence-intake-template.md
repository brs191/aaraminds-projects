# BA Agent Phase 2 Sandbox Evidence Intake Template

Use this template to collect owner-provided evidence for one Phase 2 sandbox candidate row. It is an intake template only; completing it does not authorize sandbox execution, credentials, endpoint access, non-synthetic data processing, external tool calls, external artifact storage/publishing, or write-like side effects.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Phase 2 Sandbox Evidence Intake Template |
| Version | 0.1 |
| Status | Reusable evidence intake template; non-authorizing |
| Prepared date | 2026-07-13 |
| Accountable owner | RAJA |
| Applies to | `P2-SBX-JIRA-READ`; `P2-SBX-CONF-READ`; `P2-SBX-GIT-READ`; future Teams only if explicitly reopened |
| Primary references | `docs/development/phase-2-sandbox-owner-review-package.md` v0.3; `docs/development/phase-2-sandbox-authorization-package.md` v1.1; `docs/development/mcp-validation-register.json` v0.9 |
| Explicit non-authorization | No sandbox execution, live integration, pilot start, production rollout, non-synthetic data path, external tool execution, external publish/storage, credential use, or write-like side effect |

## 1) Intake rules

1. Do not include secrets, tokens, credentials, tenant IDs, private endpoints, restricted data values, real customer data, or raw source-code content in this repository.
2. If a required evidence item is sensitive, record only an approved evidence reference here and store the sensitive evidence in the approved owner-controlled location.
3. Use `[RAJA]` for owner-dependent decisions that are not closed.
4. Use `[inferred]` only for reasonable implementation interpretations that are not source-backed.
5. Leave the register row blocked until all evidence is reviewed and RAJA records explicit execution authorization.

## 2) Candidate row summary

| Field | Value |
| --- | --- |
| Candidate row | `[RAJA]` |
| Tool/system | `[RAJA]` |
| Requested decision | Continue preparation / Defer / Reject / Authorize execution `[RAJA]` |
| Evidence owner | `[RAJA]` |
| Tool owner | `[RAJA]` |
| Security/privacy reviewer | `[RAJA]` |
| Platform reviewer | `[RAJA]` |
| QA/control reviewer | `[RAJA]` |
| BA SME/Product Owner reviewer | `[RAJA]` |
| Sensitive evidence location/reference | `[RAJA]` |

## 3) Scope evidence

| Evidence field | Required value or reference | Status |
| --- | --- | --- |
| Environment | Sandbox/non-production boundary or approved reference | `[RAJA]` |
| System scope | Project/space/repo/channel/API scope or approved reference | `[RAJA]` |
| Query/filter boundary | JQL/CQL/repo filter/page tree/date range/path policy or equivalent | `[RAJA]` |
| Data fields allowed | Field allowlist or approved reference | `[RAJA]` |
| Data fields redacted | Redaction policy or approved reference | `[RAJA]` |
| Data fields prohibited | Prohibited fields/classes or approved reference | `[RAJA]` |
| Page size / rate boundary | Maximum request size/page size and frequency | `[RAJA]` |
| Freshness window | Source timestamp/retrieval freshness rule | `[RAJA]` |

## 4) Classification and data handling evidence

| Evidence field | Required value or reference | Status |
| --- | --- | --- |
| Classification label(s) allowed | Approved data classes | `[RAJA]` |
| Personal data handling | Allowed/redacted/prohibited policy | `[RAJA]` |
| Source-code handling | Allowed/redacted/prohibited policy, if applicable | `[RAJA]` |
| Restricted/security-sensitive handling | Allowed/redacted/prohibited policy | `[RAJA]` |
| Regulated/legal handling | Compliance/legal routing, if applicable | `[RAJA]` |
| Retention window | Input/output/audit retention decision | `[RAJA]` |
| Residency boundary | Approved storage/processing boundary | `[RAJA]` |
| Deletion/archive procedure | Owner-approved deletion/archive rule | `[RAJA]` |
| Logging policy | Fields allowed in logs; prohibited values | `[RAJA]` |

## 5) Schema evidence

| Evidence field | Required value or reference | Status |
| --- | --- | --- |
| MCP server / API endpoint name | Actual tool/server/API name or approved reference | `[RAJA]` |
| Allowed read tools/actions | Exact allowlist | `[RAJA]` |
| Denied tools/actions | Exact denylist, including write-like and unlisted tools | `[RAJA]` |
| Request schema reference | Reviewed request schema location/reference | `[RAJA]` |
| Response schema reference | Reviewed response schema location/reference | `[RAJA]` |
| Schema diff reference | Diff against BA Agent expected fields | `[RAJA]` |
| Nullable/missing-field behavior | Approved behavior | `[RAJA]` |
| Unexpected-field behavior | Approved behavior | `[RAJA]` |
| Stale-schema behavior | Approved behavior | `[RAJA]` |

## 6) Auth, rate-limit, and degraded-mode evidence

| Evidence field | Required value or reference | Status |
| --- | --- | --- |
| Auth mechanism | Least-privilege auth model | `[RAJA]` |
| Credential boundary | Key Vault/managed identity or approved local secret boundary | `[RAJA]` |
| TLS/network posture | Approved transport/network boundary | `[RAJA]` |
| Rate limits | Tool-owner-approved limits | `[RAJA]` |
| Timeout policy | Per-call and total-run timeout | `[RAJA]` |
| Retry policy | Retry count/backoff and non-retryable errors | `[RAJA]` |
| Throttled behavior | User-visible and audit behavior | `[RAJA]` |
| Degraded behavior | User-visible and audit behavior | `[RAJA]` |
| Kill switch / fallback | Return-to-synthetic or row-disable procedure | `[RAJA]` |

## 7) Audit and no-write evidence

| Evidence field | Required value or reference | Status |
| --- | --- | --- |
| Allowed-read audit example | Synthetic/redacted allowed-read audit record | `[RAJA]` |
| Denied-scope audit example | Synthetic/redacted denied audit record | `[RAJA]` |
| Throttled/degraded audit example | Synthetic/redacted degraded audit record | `[RAJA]` |
| Schema-mismatch audit example | Synthetic/redacted schema-mismatch record | `[RAJA]` |
| Stale-source audit example | Synthetic/redacted stale-source record | `[RAJA]` |
| Blocked-write audit example | Synthetic/redacted blocked-write record | `[RAJA]` |
| No-write test evidence | Test command/output or reviewed control evidence | `[RAJA]` |
| BA-EM-005 result | Must remain `0` | `[RAJA]` |
| BA-EM-009 result | Must remain `0` | `[RAJA]` |

## 8) Register update readiness

Do not update the candidate row to `implementation_status: ready` or `validation_status: validated` unless every item is complete.

| Register field | Ready value/reference | Ready? |
| --- | --- | --- |
| `mcp_server_name` | Actual approved server/API identifier | No |
| `approved_scopes` | Non-sensitive approved scope refs | No |
| `actual_request_schema_ref` | Reviewed schema evidence | No |
| `actual_response_schema_ref` | Reviewed schema evidence | No |
| `schema_diff_ref` | Reviewed schema diff | No |
| `auth_model_ref` | Reviewed auth evidence | No |
| `rate_limit_ref` | Reviewed rate-limit evidence | No |
| `approval_evidence_ref` | Explicit RAJA execution decision | No |
| `validated_at` | Validation timestamp | No |
| `open_blockers` | Empty | No |

## 9) RAJA decision record

| Decision field | Value |
| --- | --- |
| Decision | Continue preparation / Defer / Reject / Authorize execution `[RAJA]` |
| Decision boundary | `[RAJA]` |
| Approved row(s) | `[RAJA]` |
| Excluded row(s) | `[RAJA]` |
| Evidence accepted | `[RAJA]` |
| Remaining blockers | `[RAJA]` |
| Decision date/reference | `[RAJA]` |

No agent-authored completion of this section is sufficient for execution authorization. The decision must be recorded as an explicit RAJA approval artifact and reflected in `docs/planning/decision-log.md`.
