# BA Agent Phase 2 Sandbox Owner Review Package

This package consolidates the prepared Phase 2 sandbox evidence for RAJA, tool-owner, security/privacy, platform, and QA/control review. It is a review-routing artifact only; it does not authorize sandbox execution, credentials, endpoint access, non-synthetic data processing, external tool calls, external artifact storage/publishing, or write-like side effects.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Phase 2 Sandbox Owner Review Package |
| Version | 0.5 |
| Change note (v0.5) | Added dry-run result template linkage for future authorized dry-run evidence capture. |
| Change note (v0.4) | Added sandbox dry-run plan linkage for future authorized row execution. |
| Change note (v0.3) | Added reusable sandbox evidence intake template linkage for owner-provided row evidence. |
| Change note (v0.2) | Added Teams blocked package linkage and clarified Teams is not part of the approved preparation review lanes. |
| Status | Review package prepared; evidence intake template, dry-run plan, and dry-run result template linked; all sandbox execution blocked |
| Prepared date | 2026-07-13 |
| Accountable owner | RAJA |
| Candidate rows | `P2-SBX-JIRA-READ`; `P2-SBX-CONF-READ`; `P2-SBX-GIT-READ`; `P2-SBX-TEAMS-READ` blocked |
| Primary references | `docs/development/phase-2-sandbox-authorization-package.md` v1.3; `docs/development/phase-2-sandbox-evidence-intake-template.md` v0.1; `docs/development/phase-2-sandbox-dry-run-plan.md` v0.2; `docs/development/phase-2-sandbox-dry-run-result-template.md` v0.1; `docs/development/phase-2-jira-sandbox-evidence-package.md` v0.1; `docs/development/phase-2-confluence-sandbox-evidence-package.md` v0.1; `docs/development/phase-2-git-sandbox-evidence-package.md` v0.1; `docs/development/phase-2-teams-sandbox-blocked-package.md` v0.1; `docs/development/mcp-validation-register.json` v0.9; `docs/development/phase-2-tool-approval-matrix.md` v1.0; `docs/development/phase-2-data-classification-plan.md` v0.2; `docs/planning/decision-log.md` v2.8 |
| Explicit non-authorization | No sandbox execution, live integration, pilot start, production rollout, non-synthetic data path, external tool execution, external publish/storage, credential use, or write-like side effect |

## 1) Review verdict

The Phase 2 sandbox package is ready for **owner evidence review**, not execution.

| Candidate row | Current readiness | Adapter/control evidence | Execution posture |
| --- | --- | --- | --- |
| `P2-SBX-JIRA-READ` | Evidence package prepared; schema evidence partial | Local BA Agent read-only wrapper exists; write-like Jira tools are denied before upstream calls | Blocked until row evidence is complete and RAJA authorizes execution |
| `P2-SBX-CONF-READ` | Evidence package prepared; schema evidence partial | Local BA Agent read-only wrapper exists; Confluence write-like and broad-root tools are denied before upstream calls | Blocked until row evidence is complete and RAJA authorizes execution |
| `P2-SBX-GIT-READ` | Evidence package prepared; no endpoint/tool family identified | No adapter should be implemented until actual endpoint/tool family exists | Blocked until endpoint/tool evidence exists, row evidence is complete, and RAJA authorizes execution |
| `P2-SBX-TEAMS-READ` | Deferred/blocked per RAJA decision | Blocked package exists; no adapter or preparation lane is authorized | Blocked; requires separate RAJA reopen decision before preparation |

No row in `docs/development/mcp-validation-register.json` is validated or executable.

## 2) Cross-row owner review checklist

Each candidate row must satisfy every item below before the register can move to `ready` / `validated`.

| Evidence ID | Evidence requirement | Jira | Confluence | Git/GitHub | Teams |
| --- | --- | --- | --- | --- | --- |
| `SBX-EV-001` | Named owner and reviewer lanes | `[RAJA]` | `[RAJA]` | `[RAJA]` | Blocked; not approved for preparation |
| `SBX-EV-002` | Exact sandbox scope | Project/board/JQL/issue types/fields `[RAJA]` | Space/page/tree/label boundary `[RAJA]` | Provider/org/repo/branch/PR/commit/path boundary `[RAJA]` | Blocked; no channel/app scope approved |
| `SBX-EV-003` | Data classification | Ticket/issue data classes `[RAJA]` | Page/body/comment/attachment data classes `[RAJA]` | Repository metadata/source-code classes `[RAJA]` | Blocked; privacy review not approved |
| `SBX-EV-004` | Field minimization and redaction | Issue fields/comments/attachments policy `[RAJA]` | Body/comments/attachments/restricted-page policy `[RAJA]` | Metadata/diff/source-code/identity policy `[RAJA]` | Blocked; message metadata/content policy not approved |
| `SBX-EV-005` | Retention, residency, deletion | `[RAJA]` | `[RAJA]` | `[RAJA]` | Blocked; message retention/residency not approved |
| `SBX-EV-006` | Actual request/response schema validation | Partial tools-list evidence; response schema incomplete | Partial tools-list evidence; response schema incomplete | No endpoint/tool schema identified | Blocked; no Teams/Graph schema approved |
| `SBX-EV-007` | Auth, rate limits, timeout/retry, degraded mode | Local evidence has auth/rate limiting disabled | Local evidence has auth/rate limiting disabled | Not captured | Blocked; app/scopes/rate limits not approved |
| `SBX-EV-008` | Audit and failure-mode evidence | Synthetic examples prepared; owner acceptance pending | Synthetic examples prepared; owner acceptance pending | Synthetic examples prepared; owner acceptance pending | Blocked; no Teams audit model approved |
| `SBX-EV-009` | No-write proof | Local wrapper tests exist; control acceptance pending | Local wrapper tests exist; control acceptance pending | Not implementable until endpoint/tool family exists | Blocked; no tool surface approved |
| `SBX-EV-010` | RAJA row decision | Preparation only; execution not approved | Preparation only; execution not approved | Preparation only; execution not approved | Deferred/blocked; requires reopen decision |

## 3) Required reviewer lanes

RAJA remains accountable until delegates are named.

| Review lane | Required decision | Applies to |
| --- | --- | --- |
| Tool owner | Approve exact tool/server, scope, schema source, and rate limits | Jira, Confluence, Git/GitHub; Teams only if reopened |
| Security/privacy | Approve classification, redaction, field minimization, personal-data handling, source-code handling, collaboration-data handling, and prohibited fields | Jira, Confluence, Git/GitHub; Teams only if reopened |
| Platform | Approve auth model, credential storage boundary, network/TLS posture, timeout/retry, throttling, and degraded mode | Jira, Confluence, Git/GitHub; Teams only if reopened |
| QA/control | Accept no-write proof, audit behavior, schema-mismatch handling, and hard-gate regression evidence | Jira, Confluence, Git/GitHub; Teams only if reopened |
| BA SME/Product Owner | Confirm row value for Phase 2 requirement discovery and traceability | Jira, Confluence, Git/GitHub; Teams only if reopened |
| Compliance/legal | Review regulated/legal/source-document implications where applicable | Confluence, Git/GitHub where source content is requested |

Use `docs/development/phase-2-sandbox-evidence-intake-template.md` when a reviewer or tool owner is ready to provide evidence for one candidate row. Do not paste secrets, tenant identifiers, private endpoints, restricted data values, or raw source-code content into this repository.

Use `docs/development/phase-2-sandbox-dry-run-plan.md` only after a row has explicit RAJA execution authorization and a validated register row. It is not a substitute for approval.

Use `docs/development/phase-2-sandbox-dry-run-result-template.md` to capture future authorized dry-run results without secrets, tenant identifiers, private endpoints, restricted data values, or raw source-code content.

## 4) Decision options by row

| Decision option | Meaning | Required repository update |
| --- | --- | --- |
| Continue preparation | Keep collecting evidence; no execution | Update row evidence package and `mcp-validation-register.json` blockers if evidence changes |
| Defer row | Keep row blocked with rationale | Update this package, sandbox authorization package, decision log, and register blockers |
| Reject row | Remove from near-term sandbox candidate list | Update tool matrix, sandbox package, decision log, and register posture |
| Authorize execution | Allow exactly one row under approved scope | Only after all `SBX-EV-001` through `SBX-EV-010` are complete; update register to `ready` / `validated`, record RAJA decision, and re-run hard gates |
| Reopen Teams preparation | Move Teams from blocked to preparation review | Requires separate explicit RAJA decision before any Teams package, adapter, schema, or test work |
| Schedule dry run | Execute the smallest approved read for one authorized row | Only after row authorization and register validation; follow `phase-2-sandbox-dry-run-plan.md` |

Authorizing one row does not authorize any other row.

## 5) Register update rule

The register row for a candidate may move to `implementation_status: ready` and `validation_status: validated` only when:

1. The row has a named owner or RAJA explicitly remains acting owner for execution.
2. Approved scopes are present and do not expose sensitive identifiers in this repository.
3. Request/response schema refs and schema-diff refs point to reviewed evidence.
4. Auth and rate-limit refs point to reviewed evidence.
5. Approval evidence points to an explicit RAJA execution decision.
6. `validated_at` is populated.
7. `open_blockers` is empty.
8. BA-EM-005 and BA-EM-009 remain zero.

Until then, rows stay blocked and synthetic fixtures remain the only approved Phase 2 input source.

## 6) Recommended next human action

Route this package to RAJA for a row-by-row owner review:

1. Pick at most one first execution candidate, if any.
2. Name the tool owner, security/privacy, platform, and QA/control reviewers for that row.
3. Provide approved scope/classification/schema/auth/rate-limit/audit evidence, or explicitly keep the row in preparation-only status.

No agent-authored artifact in this repository can create the execution approval.
