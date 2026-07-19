# BA Agent Phase 2 Sandbox Dry-Run Result Template

Use this template only after one Phase 2 sandbox row has explicit RAJA execution authorization and a validated register row. This template is a result-capture artifact; it does not authorize sandbox execution, credentials, endpoint access, non-synthetic data processing, external tool calls, external artifact storage/publishing, or write-like side effects.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Phase 2 Sandbox Dry-Run Result Template |
| Version | 0.1 |
| Status | Reusable dry-run result template; non-authorizing |
| Prepared date | 2026-07-13 |
| Accountable owner | RAJA |
| Applies to | First future authorized read-only row only |
| Primary references | `docs/development/phase-2-sandbox-dry-run-plan.md` v0.2; `docs/development/phase-2-sandbox-authorization-package.md` v1.3; `docs/development/phase-2-sandbox-owner-review-package.md` v0.5; `docs/development/mcp-validation-register.json` v0.9; `docs/planning/decision-log.md` v2.8 |
| Explicit non-authorization | No sandbox execution, live integration, pilot start, production rollout, non-synthetic data path, external tool execution, external publish/storage, credential use, or write-like side effect |

## 1) Dry-run identity

| Field | Value |
| --- | --- |
| Candidate row | `[RAJA]` |
| Tool/system | `[RAJA]` |
| Approved scope reference | `[RAJA]` |
| RAJA execution authorization reference | `[RAJA]` |
| Register version | `[RAJA]` |
| Register row validation timestamp | `[RAJA]` |
| Operator | `[RAJA]` |
| Reviewer lanes | `[RAJA]` |
| Start timestamp | `[RAJA]` |
| End timestamp | `[RAJA]` |

## 2) Pre-run gate confirmation

| Gate | Required value | Observed value |
| --- | --- | --- |
| RAJA row authorization exists | Yes | `[RAJA]` |
| Candidate row validated | Yes | `[RAJA]` |
| No open blockers in row | Yes | `[RAJA]` |
| BA-EM-005 before run | `0` | `[RAJA]` |
| BA-EM-009 before run | `0` | `[RAJA]` |
| Synthetic fallback available | Yes | `[RAJA]` |
| Kill switch reviewed | Yes | `[RAJA]` |

If any observed value does not match the required value, stop and do not run the dry run.

## 3) Request summary

Do not include secrets, credentials, private endpoints, tenant IDs, restricted data values, raw source-code content, or unredacted personal data.

| Field | Value |
| --- | --- |
| Read action/tool used | `[RAJA]` |
| Request purpose | `[RAJA]` |
| Request scope reference | `[RAJA]` |
| Request fields | `[RAJA]` |
| Redacted/prohibited fields excluded | `[RAJA]` |
| Page size / limit used | `[RAJA]` |
| Timeout/retry policy applied | `[RAJA]` |

## 4) Minimized response summary

| Field | Value |
| --- | --- |
| Result status | Pass / degraded / blocked / failed `[RAJA]` |
| Records/items returned | `[RAJA]` |
| Source timestamp range | `[RAJA]` |
| Retrieved-at timestamp | `[RAJA]` |
| Evidence refs generated | `[RAJA]` |
| Prohibited fields observed | None / details routed separately `[RAJA]` |
| Redaction applied | `[RAJA]` |
| Data retained | `[RAJA]` |
| Data deleted/archived | `[RAJA]` |

## 5) Audit and failure-mode evidence

| Evidence item | Reference / result |
| --- | --- |
| Allowed-read audit record | `[RAJA]` |
| Denied-scope audit record, if exercised | `[RAJA]` |
| Throttled/degraded audit record, if exercised | `[RAJA]` |
| Schema-mismatch audit record, if exercised | `[RAJA]` |
| Stale-source audit record, if exercised | `[RAJA]` |
| Blocked-write audit record, if exercised | `[RAJA]` |
| No-write proof result | `[RAJA]` |
| Unlisted-tool proof result | `[RAJA]` |

## 6) Stop-trigger review

| Stop trigger | Occurred? | Response |
| --- | --- | --- |
| Unapproved field appeared | `[RAJA]` | `[RAJA]` |
| Schema differed from approved schema | `[RAJA]` | `[RAJA]` |
| Auth/scope mismatch occurred | `[RAJA]` | `[RAJA]` |
| Write-like action became reachable | `[RAJA]` | `[RAJA]` |
| MVP route received Phase 2 behavior | `[RAJA]` | `[RAJA]` |
| Retention/residency rule missing or violated | `[RAJA]` | `[RAJA]` |
| Secret or credential appeared in output/log | `[RAJA]` | `[RAJA]` |
| Rate-limit/throttling behavior differed from approval | `[RAJA]` | `[RAJA]` |

## 7) Post-run hard gates

| Gate | Required value | Observed value |
| --- | --- | --- |
| BA-EM-005 approval-gate bypass count | `0` | `[RAJA]` |
| BA-EM-009 phase-separation violations | `0` | `[RAJA]` |
| Write-like external side effects | `0` | `[RAJA]` |
| Unauthorized fields retained | `0` | `[RAJA]` |

## 8) Recommendation

| Decision option | Selected? | Notes |
| --- | --- | --- |
| Continue preparation | `[RAJA]` | `[RAJA]` |
| Rerun after fixes | `[RAJA]` | `[RAJA]` |
| Defer row | `[RAJA]` | `[RAJA]` |
| Reject row | `[RAJA]` | `[RAJA]` |
| Request next owner decision | `[RAJA]` | `[RAJA]` |

This result template cannot approve pilot, production, additional rows, write-like behavior, or broader data access.

## 9) Required follow-up updates

If a dry run is completed, update the following artifacts in the same change set:

1. `docs/planning/decision-log.md`
2. `docs/development/mcp-validation-register.json`
3. `docs/development/phase-2-sandbox-authorization-package.md`
4. The relevant row-level evidence package
5. Any traceability or risk/backlog artifact impacted by the result

If the dry run fails or validation becomes stale, return the row to blocked/preparation state and preserve only minimized, approved evidence.
