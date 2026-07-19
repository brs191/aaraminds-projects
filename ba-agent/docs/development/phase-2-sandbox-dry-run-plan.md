# BA Agent Phase 2 Sandbox Dry-Run Plan

This plan defines how to run a future Phase 2 sandbox dry run after one candidate row is explicitly authorized. It is a preparation artifact only; it does not authorize sandbox execution, credentials, endpoint access, non-synthetic data processing, external tool calls, external artifact storage/publishing, or write-like side effects.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Phase 2 Sandbox Dry-Run Plan |
| Version | 0.2 |
| Change note (v0.2) | Added dry-run result template linkage for future authorized dry-run evidence capture. |
| Status | Draft dry-run plan; no row authorized for execution |
| Prepared date | 2026-07-13 |
| Accountable owner | RAJA |
| Applies to | First future authorized read-only row only |
| Primary references | `docs/development/phase-2-sandbox-authorization-package.md` v1.3; `docs/development/phase-2-sandbox-owner-review-package.md` v0.5; `docs/development/phase-2-sandbox-evidence-intake-template.md` v0.1; `docs/development/phase-2-sandbox-dry-run-result-template.md` v0.1; `docs/development/mcp-validation-register.json` v0.9; `docs/planning/decision-log.md` v2.8 |
| Explicit non-authorization | No sandbox execution, live integration, pilot start, production rollout, non-synthetic data path, external tool execution, external publish/storage, credential use, or write-like side effect |

## 1) Dry-run entry criteria

A dry run may be scheduled only after all criteria below are complete for exactly one candidate row:

| Criterion | Required state |
| --- | --- |
| RAJA row decision | Explicit execution authorization recorded outside agent-authored text and linked in `docs/planning/decision-log.md` |
| Register state | Candidate row in `docs/development/mcp-validation-register.json` is `implementation_status: ready`, `validation_status: validated`, has approved scopes, has `validated_at`, and has no `open_blockers` |
| Evidence package | Row evidence package has complete owner, scope, classification, schema, auth/rate-limit, audit, and no-write evidence |
| Intake template | `docs/development/phase-2-sandbox-evidence-intake-template.md` is completed or referenced for the row |
| Hard gates | BA-EM-005 = `0`; BA-EM-009 = `0` |
| Fallback | Synthetic fixture fallback is available |
| Kill switch | Row-disable and return-to-synthetic procedure is reviewed |

If any criterion is missing, do not run the dry run.

## 2) Dry-run scope

| Scope item | Rule |
| --- | --- |
| Candidate count | One row only |
| Access mode | Read-only only |
| Inputs | Approved sandbox scope only; no broad tenant/project/repo/space/channel access |
| Outputs | Local draft/advisory evidence report only |
| Writes | Prohibited, including comments, updates, sends, drafts, publishes, approval records, and subscriptions |
| Data handling | Use only owner-approved fields; redact or block all prohibited fields |
| Logs | No secrets, credentials, raw restricted values, private endpoints, tenant IDs, or raw source-code content |
| Review | Human review required before interpreting results as readiness evidence |

## 3) Pre-run checklist

| Check | Required result |
| --- | --- |
| Confirm row authorization | RAJA decision evidence points to the exact row and scope |
| Confirm row is validated | Register summary shows the row validated and all other rows blocked unless separately authorized |
| Confirm environment | Non-production/sandbox only |
| Confirm credentials | Credentials are available only through approved boundary; none are committed or pasted into docs |
| Confirm allowlist | Adapter/tool surface exposes only approved read tools |
| Confirm denylist | Write-like and unlisted tools fail before upstream calls |
| Confirm audit sink | Allowed, denied, degraded, throttled, stale-source, schema-mismatch, and blocked-write paths emit audit records |
| Confirm fallback | Synthetic mode can be restored without external dependency |

## 4) Dry-run execution outline

This is an outline, not permission to execute.

1. Record dry-run start note with candidate row, approved scope reference, operator, and timestamp.
2. Load runtime settings with sandbox-read mode and live writes disabled.
3. Load the validated MCP register.
4. Instantiate only the authorized row adapter.
5. Execute the smallest approved read request.
6. Capture minimized response metadata and audit record.
7. Verify no prohibited fields were returned or persisted.
8. Verify no write-like or unlisted tool was called.
9. Run BA-EM-005 and BA-EM-009 hard gates.
10. Record result as pass, degraded, blocked, or failed.

## 5) Stop triggers

Stop immediately and return to synthetic fallback if any trigger occurs:

| Trigger | Required response |
| --- | --- |
| Unapproved field appears | Stop row, preserve minimized audit, route to security/privacy review |
| Schema differs from approved schema | Stop row, mark validation stale, return register row to blocked |
| Auth/scope mismatch occurs | Stop row, route to platform/tool owner |
| Write-like action becomes reachable | Stop immediately; BA-EM-005 hard-gate failure |
| MVP route receives Phase 2 behavior | Stop immediately; BA-EM-009 hard-gate failure |
| Retention/residency rule missing or violated | Stop row and route to RAJA/security/privacy |
| Secret or credential appears in output/log | Stop row, preserve incident evidence, route to security/privacy |
| Rate limit or throttling behavior differs from approval | Stop row or mark degraded per owner policy |

## 6) Result package

A completed dry run must produce a local review package with:

1. Candidate row and approved scope reference.
2. Start/end timestamp.
3. Operator/reviewer lanes.
4. Minimal request description.
5. Minimized response summary.
6. Audit record references.
7. BA-EM-005 and BA-EM-009 results.
8. Any denied/degraded/throttled/schema-mismatch outcomes.
9. Stop-trigger status.
10. Recommendation: continue preparation, rerun after fixes, defer, reject, or request next owner decision.

The result package must not contain secrets, credentials, raw restricted data, private endpoints, tenant IDs, or raw source-code content.

Use `docs/development/phase-2-sandbox-dry-run-result-template.md` for the result package so required gate, audit, stop-trigger, and register-rollback evidence is captured consistently.

## 7) Register rollback rule

If the dry run fails, the candidate row must return to a blocked state unless the failure is explicitly accepted as a degraded-but-safe result by RAJA and the required reviewers.

Rollback updates must include:

1. `implementation_status` returned to a blocked/preparation state if validation is stale.
2. `validation_status` returned to `not_validated` or `schema_observed_not_validated_for_execution` as appropriate.
3. `open_blockers` populated with the failure reason.
4. Decision log updated with the outcome.
5. Synthetic fallback confirmed.

## 8) Current status

No dry run is authorized today. This plan is ready for future use only after row-level authorization is recorded.
