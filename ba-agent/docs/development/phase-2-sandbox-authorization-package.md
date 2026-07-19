# BA Agent Phase 2 Sandbox Authorization Package

This package defines the evidence required before the BA Agent may move from synthetic-first completion to any Phase 2 sandbox path. It is an authorization package for RAJA review; until RAJA records an explicit approval decision and every required evidence item is complete, sandbox execution remains blocked.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Phase 2 Sandbox Authorization Package |
| Version | 1.3 |
| Change note (v1.3) | Added dry-run result template linkage for future authorized dry-run evidence capture. |
| Change note (v1.2) | Added sandbox dry-run plan linkage for future authorized row execution; no row is authorized today. |
| Change note (v1.1) | Added reusable sandbox evidence intake template linkage for owner-provided row evidence. |
| Change note (v1.0) | Added Teams blocked package linkage; Teams remains excluded from approved preparation and execution lanes. |
| Change note (v0.9) | Added consolidated sandbox owner-review package linkage across Jira, Confluence, and Git/GitHub preparation rows; execution remains blocked. |
| Change note (v0.8) | Added Git/GitHub read-only row-level evidence package linkage; no endpoint/tool family is validated and execution remains blocked. |
| Change note (v0.7) | Added Confluence read-only wrapper evidence and row-level evidence package linkage while keeping execution blocked. |
| Change note (v0.6) | Added Jira row-level sandbox evidence package linkage for scope, classification, schema, auth/rate-limit, audit, and register-update conditions; execution remains blocked. |
| Change note (v0.5) | Added `P2-SBX-JIRA-READ` row-level evidence plan, owner-decision prompts, and register-linkage boundaries while keeping execution blocked. |
| Change note (v0.4) | Added BA Agent Phase 2 Jira read-only MCP wrapper evidence and tests proving advertised write-like Jira tools are blocked before upstream calls. |
| Change note (v0.3) | Added local MCP server evidence from the running `apm0045942-cc-mcp-server` on port 8000 and converted it into a preparation-only allowlist path for Phase 2. |
| Change note (v0.2) | Recorded RAJA approval for Jira, Confluence, and Git/GitHub read-only sandbox evidence preparation; Teams remains blocked and no execution is authorized. |
| Status | Jira, Confluence, and Git/GitHub read-only row-level evidence packages, owner-review package, intake template, dry-run plan, and dry-run result template prepared; Teams blocked package linked; sandbox execution not authorized |
| Prepared date | 2026-07-13 |
| Accountable owner | RAJA |
| Triggering artifact | `docs/development/phase-2-synthetic-completion-package.md` |
| Recommended posture | Read-only-first sandbox readiness; Jira, Confluence, and Git/GitHub approved for evidence preparation only; execution blocked until evidence completion |
| Primary references | `docs/development/phase-2-synthetic-completion-package.md`; `docs/development/p2-g4-tool-data-readiness.md`; `docs/development/phase-2-tool-approval-matrix.md`; `docs/development/phase-2-data-classification-plan.md`; `docs/development/phase-2-blocked-tool-data-artifact-backlog.md`; `docs/development/phase-2-jira-sandbox-evidence-package.md`; `docs/development/phase-2-confluence-sandbox-evidence-package.md`; `docs/development/phase-2-git-sandbox-evidence-package.md`; `docs/development/phase-2-teams-sandbox-blocked-package.md`; `docs/development/phase-2-sandbox-owner-review-package.md`; `docs/development/phase-2-sandbox-evidence-intake-template.md`; `docs/development/phase-2-sandbox-dry-run-plan.md`; `docs/development/phase-2-sandbox-dry-run-result-template.md`; `docs/development/mcp-validation-register.json`; `docs/planning/decision-log.md` |
| Explicit non-authorization | No sandbox execution, live integration, pilot start, production rollout, non-synthetic data processing, external publish/storage, external tool execution, write-like side effect, or credential use is authorized by this package |

## 1) Authorization recommendation

RAJA decision recorded on 2026-07-09: **approve Jira, Confluence, and Git/GitHub read-only sandbox preparation; do not approve Teams**.

This approval is limited to evidence preparation. It does **not** authorize sandbox execution, credentials, endpoint access, non-synthetic data processing, external tool calls, external publishing/storage, live/pilot/production activity, or any write-like side effect.

Teams remains explicitly blocked in `docs/development/phase-2-teams-sandbox-blocked-package.md`. Do not create a Teams adapter, Graph API path, approval-flow implementation, Adaptive Card send path, or Teams preparation package unless RAJA records a separate reopen decision.

The consolidated owner-review package is `docs/development/phase-2-sandbox-owner-review-package.md`. Use it to route row-by-row owner decisions and evidence collection across Jira, Confluence, and Git/GitHub.

Use `docs/development/phase-2-sandbox-evidence-intake-template.md` to collect owner-provided evidence for a single row without committing secrets, tenant identifiers, private endpoints, restricted data values, or raw source-code content.

Use `docs/development/phase-2-sandbox-dry-run-plan.md` only after a row is explicitly authorized and the register row is validated. No dry run is authorized today.

Use `docs/development/phase-2-sandbox-dry-run-result-template.md` only to capture the result of a future authorized dry run. It is not an approval artifact.

The next safe move is to collect and review evidence for a narrow, read-only sandbox candidate. Do not connect to Jira, Confluence, GitHub/Git, Teams, SharePoint, SQL/Data, ServiceNow, Miro/Draw.io, Azure DevOps, or test-management tools until the matching row in this package has:

1. Named owner.
2. Approved sandbox scope.
3. Security/privacy classification decision.
4. Retention, residency, and redaction decision.
5. Actual request/response schema validation.
6. Auth model and rate-limit evidence.
7. Audit and failure-mode evidence.
8. RAJA approval recorded in the decision log.

Missing any one item keeps the path **blocked**.

### Current local MCP evidence captured on 2026-07-10

RAJA provided a locally running MCP server at `localhost:8000` with code in `../apm0045942-cc-mcp-server`. This evidence is useful for preparation, but it does not by itself authorize sandbox execution.

| Evidence area | Observed evidence | Phase 2 interpretation |
| --- | --- | --- |
| Server health | `GET /healthz` and `GET /readyz` returned `200 OK` from the local server. | Confirms local MCP process availability for evidence preparation only. |
| MCP protocol | `/jira-cloud/mcp` completed MCP `initialize` and `tools/list`. | Supports partial actual-schema evidence for the Jira candidate row. |
| Jira tools | Advertised read-like tools include `FetchItrackJiraIssuesList`, `GetJiraItrackJobStatus`, and `JiraItrackValidate`; the same surface also advertises write/destructive tools including `CreateJiraCloudIssue`, `UpdateJiraCloudIssue`, `UpdateJiraCloudStatus`, `DeleteJiraCloudIssue`, and `RevertJiraItrackIssue`. | Jira is a viable preparation target only behind a BA Agent allowlist/gateway that exposes the read-like tools and blocks every write-like tool by code. |
| Confluence tools | Root MCP surface advertises Confluence read tools such as `confluence_search`, `confluence_get_page`, `confluence_list_spaces`, `confluence_space_pages`, `confluence_page_children`, `confluence_page_attachments`, and `confluence_page_comments`. | Confluence schema evidence is partial because the available root surface is mixed with unrelated/write-like tools; BA Agent now has a read-only allowlist wrapper, but execution still requires row evidence and approval. |
| Git/GitHub tools | No Git/GitHub read endpoint or tool family was identified in the running server. | `P2-SBX-GIT-READ` remains approved for preparation only but lacks implementation evidence. |
| Auth/rate limit | Local config evidence showed auth, TLS, and rate limiting disabled for the local server. Credential values are not recorded in this package. | Auth/rate-limit guardrails are not complete; execution remains blocked. |
| No-write proof | Raw advertised tool surfaces include write-like/destructive tools; BA Agent Jira wrapper tests now block those tools before upstream calls. | Jira no-write evidence is partial and local; control review and RAJA execution authorization remain required. |

Working posture: **use what exists for Phase 2 preparation by wrapping it, not by trusting the raw server surface**. The BA Agent should treat the raw MCP as an upstream candidate and expose only a narrow, code-enforced Phase 2 read-only adapter when and if RAJA later authorizes execution.

### BA Agent wrapper evidence added on 2026-07-10

`src/ba_agent/phase2/sandbox_mcp.py` now defines a Phase 2 Jira read-only MCP adapter over the existing `apm0045942-cc-mcp-server` upstream candidate.

| Control | Evidence | Remaining boundary |
| --- | --- | --- |
| Register gate | Adapter construction requires `BA_AGENT_DATA_SOURCE_MODE=sandbox_read`, `LIVE_INTEGRATIONS_ENABLED=false`, and a fully validated `get_sprint_status` row in `docs/development/mcp-validation-register.json`. | Current register row remains `schema_observed_not_validated_for_execution`, so real execution is still blocked. |
| Jira read mapping | Adapter maps Phase 2 Jira metadata reads only to `FetchItrackJiraIssuesList`, `GetJiraItrackJobStatus`, and `JiraItrackValidate`. | Scope, classification, auth, rate limits, and approved fields remain `[RAJA]`/incomplete. |
| Write-like denial | Tests prove `CreateJiraCloudIssue`, `UpdateJiraCloudIssue`, `UpdateJiraCloudStatus`, `DeleteJiraCloudIssue`, and `RevertJiraItrackIssue` raise before any upstream client call. | RAJA execution authorization is still required after all row evidence is complete. |
| No network in tests | Wrapper tests use a fake MCP client and the repository no-network fixture. | No real Jira call has been made or authorized. |

### BA Agent Confluence wrapper evidence added on 2026-07-13

`src/ba_agent/phase2/sandbox_mcp.py` now defines a Phase 2 Confluence read-only MCP adapter over the existing broad root MCP surface.

| Control | Evidence | Remaining boundary |
| --- | --- | --- |
| Register gate | Adapter construction requires `BA_AGENT_DATA_SOURCE_MODE=sandbox_read`, `LIVE_INTEGRATIONS_ENABLED=false`, and a fully validated `get_confluence_source_pages` row in `docs/development/mcp-validation-register.json`. | Current register row remains `schema_observed_not_validated_for_execution`, so real execution is still blocked. |
| Confluence read mapping | Adapter maps Phase 2 Confluence metadata/page reads only to `confluence_search`, `confluence_get_page`, `confluence_list_spaces`, `confluence_space_pages`, `confluence_page_children`, `confluence_page_attachments`, and `confluence_page_comments`. | Scope, classification, auth, rate limits, restricted-page policy, body/comment/attachment policy, and approved fields remain `[RAJA]`/incomplete. |
| Write-like and broad-root denial | Tests prove Confluence write-like tools and unrelated root-surface tools raise before any upstream client call. | RAJA execution authorization is still required after all row evidence is complete. |
| No network in tests | Wrapper tests use a fake MCP client and the repository no-network fixture. | No real Confluence call has been made or authorized. |

### `P2-SBX-CONF-READ` row-level evidence plan added on 2026-07-13

The detailed row-level evidence package is maintained in `docs/development/phase-2-confluence-sandbox-evidence-package.md`. That package defines the scope capture template, proposed field-minimization baseline, retention/residency decision template, schema validation contract, auth/rate-limit template, synthetic audit examples, and register update conditions for `P2-SBX-CONF-READ`.

### `P2-SBX-GIT-READ` row-level evidence plan added on 2026-07-13

The detailed row-level evidence package is maintained in `docs/development/phase-2-git-sandbox-evidence-package.md`. That package defines the provider/repository scope capture template, proposed metadata-minimum baseline, source-code classification boundary, retention/residency decision template, schema validation contract, auth/rate-limit template, synthetic audit examples, and register update conditions for `P2-SBX-GIT-READ`.

Unlike Jira and Confluence, no Git/GitHub MCP endpoint or tool family has been identified in the current local evidence. Therefore no BA Agent Git/GitHub adapter is implemented yet, and the register row must remain `not_validated` until the actual endpoint/tool family exists and is reviewed.

### `P2-SBX-JIRA-READ` row-level evidence plan added on 2026-07-10

This plan converts the available Jira wrapper and local MCP observations into the exact evidence needed before the row can move from preparation to RAJA authorization review. It is a preparation artifact only. It does not authorize credentials, endpoint access, sandbox calls, non-synthetic input, external tool execution, external artifact storage, or write-like side effects.

The detailed row-level evidence package is now maintained in `docs/development/phase-2-jira-sandbox-evidence-package.md`. That package defines the scope capture template, proposed field-minimization baseline, retention/residency decision template, schema validation contract, auth/rate-limit template, synthetic audit examples, and register update conditions for `P2-SBX-JIRA-READ`.

| Evidence ID | Jira row evidence target | Current preparation status | Blocking decision before execution |
| --- | --- | --- | --- |
| `SBX-EV-001` | Named Jira owner and review lanes | RAJA remains acting owner for preparation; Jira tool owner, security/privacy reviewer, and platform reviewer are not named in this artifact. | RAJA must name delegates or explicitly retain acting-owner responsibility for the Jira row. |
| `SBX-EV-002` | Exact Jira sandbox scope | No project key, board, JQL, issue-type scope, comment policy, or tenant endpoint is recorded here. | RAJA/tool owner must approve the exact sandbox scope outside this public checklist if sensitive. |
| `SBX-EV-003` | Data classification | Candidate data classes are issue metadata, epics/stories, labels, statuses, links, and optionally comments; approval is not recorded. | Security/privacy must approve allowed classes and prohibited fields before any non-synthetic read. |
| `SBX-EV-004` | Field minimization and redaction | Proposed default is metadata-minimum: issue key, summary/title if approved, status, type, labels, links, timestamps, and source refs; descriptions/comments remain blocked unless separately approved. | RAJA/security/privacy must approve the final field allowlist and redaction rules. |
| `SBX-EV-005` | Retention, residency, deletion | No retention window, audit retention, deletion procedure, or residency decision is recorded. | RAJA/security/privacy/platform must approve storage and log-handling rules. |
| `SBX-EV-006` | Actual schema validation | Local `/jira-cloud/mcp` `tools/list` evidence identified `FetchItrackJiraIssuesList`, `GetJiraItrackJobStatus`, and `JiraItrackValidate`; response schema remains incomplete for execution. | Tool owner/platform must provide actual request/response schemas, schema diff, and validation result for the approved scope. |
| `SBX-EV-007` | Auth and rate limits | Current local evidence shows auth, TLS, and rate limiting disabled. | Platform/tool owner must approve least-privilege auth, credential handling, throttling, timeout, retry, and degraded-mode behavior. |
| `SBX-EV-008` | Audit and failure modes | Synthetic audit and failure-mode examples are prepared in `docs/development/phase-2-jira-sandbox-evidence-package.md`; execution evidence is not complete. | Control review must accept allowed, denied, degraded, throttled, schema-mismatch, stale-source, and blocked-write audit behavior without exposing secrets or real restricted data. |
| `SBX-EV-009` | No-write proof | Local wrapper tests deny advertised Jira write/destructive tools before upstream calls. | Security/control review must accept the allowlist/no-write proof and confirm comments, updates, deletes, approval tools, and subscriptions remain unreachable. |
| `SBX-EV-010` | RAJA row decision | Preparation approval is recorded; execution authorization is not. | Decision log must record explicit RAJA approval for this exact Jira row after all evidence above is complete. |

#### Jira schema evidence request template

When a tool owner is ready to provide evidence, capture the following without secrets, credentials, tenant identifiers, or restricted data values:

| Evidence field | Required content |
| --- | --- |
| Candidate row | `P2-SBX-JIRA-READ` / `get_sprint_status` |
| Approved scope reference | Owner-approved project/board/JQL/issue-type boundary, stored in the approved evidence location if sensitive |
| Allowed upstream tools | `FetchItrackJiraIssuesList`, `GetJiraItrackJobStatus`, `JiraItrackValidate` only |
| Denied upstream tools | `CreateJiraCloudIssue`, `UpdateJiraCloudIssue`, `UpdateJiraCloudStatus`, `DeleteJiraCloudIssue`, `RevertJiraItrackIssue`, comments, approval tools, subscriptions, and any unlisted Jira tool |
| Request schema | JSON schema or equivalent structured field list for each allowed tool under the approved scope |
| Response schema | JSON schema or equivalent structured field list for each allowed tool under the approved scope, including nullable/optional fields |
| Field policy | Allowed, redacted, and prohibited fields, especially descriptions, comments, attachments, user identities, and links |
| Auth/rate limit | Auth mechanism, least-privilege scopes, token storage boundary, rate limits, timeout/retry policy, and degraded-mode response |
| Audit proof | Example allowed-read, denied-write, schema-mismatch, throttled/degraded, and stale-source audit records using synthetic or redacted data |
| Approval evidence | RAJA decision-log row and reviewer signoffs for owner, security/privacy, platform/tool owner, and QA/control review |

Until this evidence is complete and the register row is updated to `implementation_status: ready`, `validation_status: validated`, with no open blockers, the BA Agent must continue to reject construction of the Jira sandbox adapter against the checked-in register.

## 2) Allowed preparation work before authorization

The following preparation is allowed because it does not touch external systems:

| Preparation item | Allowed action | Boundary |
| --- | --- | --- |
| Evidence checklist | Draft row-level evidence requirements and review questions | No credentials, endpoints, tenant IDs, project keys, repo names, channel IDs, or real data |
| Schema request template | Define what actual schema evidence must be captured by tool owners later | Do not call the tool or infer real schemas from design docs |
| Review routing | Assign or keep `[RAJA]` acting-owner lanes | No automatic approval |
| Dry-run plan | Define a future dry-run procedure | Do not execute dry run |
| Rollback plan | Define disable/fallback triggers | Do not claim production kill-switch readiness |

## 3) Candidate read-only sandbox scope

This package intentionally starts with **read-only** candidates. Write-like behavior remains blocked.

| Candidate ID | Tool/path | Candidate use | Data class | Read/write posture | Current status | Required approval before execution |
| --- | --- | --- | --- | --- | --- | --- |
| `P2-SBX-JIRA-READ` | Jira | Read requirement-related issue/ticket metadata for Phase 2 requirement discovery and traceability | Issues, epics, stories, labels, statuses, links, comments `[RAJA]` | Read-only candidate; writes/comments blocked | Approved for evidence preparation only; execution blocked | Jira owner, project scope, field/comment policy, schema validation, auth/rate limits, classification |
| `P2-SBX-CONF-READ` | Confluence | Read approved sandbox source pages for requirement context | Pages, labels, page metadata `[RAJA]` | Read-only candidate; drafts/publishes blocked | Approved for evidence preparation only; execution blocked; row-level evidence package prepared | Complete `docs/development/phase-2-confluence-sandbox-evidence-package.md`: space owner, page scope, restricted-page handling, body/comment/attachment policy, schema validation, auth/rate limits, classification, audit/control review |
| `P2-SBX-GIT-READ` | GitHub/Git | Read approved implementation metadata for traceability | PR metadata, commits, files only if approved `[RAJA]` | Metadata read-only candidate; comments/writes blocked | Approved for evidence preparation only; execution blocked; no endpoint/tool family validated; row-level evidence package prepared | Complete `docs/development/phase-2-git-sandbox-evidence-package.md`: provider, repo scope, source-code classification, metadata/diff/comment policy, schema validation, auth/rate limits, audit/control review |
| `P2-SBX-TEAMS-READ` | Teams/Copilot 365 | Read approved sandbox clarification/review interaction metadata only if privacy review allows it | Message metadata or approved test-channel content `[RAJA]` | Read-only candidate; sends/cards blocked | Deferred / blocked per RAJA decision | Channel owner, tenant/app approval, retention policy, privacy review, schema validation, separate RAJA approval |

Excluded from the first sandbox candidate: SQL/Data raw rows, SharePoint documents, ServiceNow, Azure DevOps, Miro/Draw.io, test-management tools, external artifact publishing, and every write-like action.

## 4) Mandatory evidence checklist

Each candidate row must complete this checklist before status can change from `Blocked` to `Ready for RAJA authorization`.

| Evidence ID | Evidence requirement | Acceptable proof | Current status |
| --- | --- | --- | --- |
| `SBX-EV-001` | Named owner and reviewer lane | Named owner/delegate, or RAJA explicitly remains acting owner for that row | Jira: preparation owner recorded as RAJA acting owner; delegate/reviewer names still `[RAJA]` |
| `SBX-EV-002` | Approved sandbox scope | Exact sandbox project/space/repo/channel/data boundary recorded outside this public checklist if sensitive | Jira: scope request template prepared; actual project/board/JQL/field boundary still `[RAJA]` |
| `SBX-EV-003` | Data classification decision | Security/privacy review states allowed data class and prohibited fields | Jira: candidate data classes listed; classification approval still `[RAJA]` |
| `SBX-EV-004` | Redaction/minimization rule | Field-level list of allowed metadata and redacted/prohibited content | Jira: metadata-minimum default proposed; final field/comment policy still `[RAJA]` |
| `SBX-EV-005` | Retention/residency rule | Retention, deletion/archive, audit retention, and residency decision | Jira: not ready; retention/residency/deletion rules still `[RAJA]` |
| `SBX-EV-006` | Actual schema validation | Actual request/response schema refs, schema diff, and validation result | Jira: partial local MCP tools-list evidence plus evidence request template; not validated for execution |
| `SBX-EV-007` | Auth/rate-limit guardrails | Auth model, least-privilege scopes, rate limits, throttling/degraded behavior | Jira: not ready; local evidence shows auth/rate limiting disabled |
| `SBX-EV-008` | Audit evidence | Trace ID, evidence refs, source timestamp, retrieved timestamp, and denied/degraded audit behavior | Jira: audit proof requirements defined; evidence not complete |
| `SBX-EV-009` | No-write proof | Tests or control review proving comments/sends/publishes/updates/drafts remain blocked | Jira: local wrapper tests prove advertised write-like tools are blocked before upstream calls; security/control approval still required |
| `SBX-EV-010` | RAJA decision | Decision-log row updated with explicit approved/denied/deferred outcome | Complete for preparation approval: Jira, Confluence, and Git/GitHub approved; Teams deferred/blocked |

## 5) Decision options

| Option | Meaning | Effect |
| --- | --- | --- |
| Prepare only | Approve evidence collection and package refinement only | Recommended current option; no sandbox calls |
| Authorize one read-only row | Approve exactly one candidate row after all evidence is complete | Only that row may be implemented/executed under approved scope |
| Authorize multiple read-only rows | Approve multiple rows after all evidence is complete per row | Only approved rows may be implemented/executed |
| Defer | Keep synthetic-only completion as the final state for now | All sandbox paths remain blocked |
| Reject | Stop sandbox progression | Keep blocked posture and document rationale |

## 6) Gate rules for first sandbox execution

No first sandbox execution may occur until:

1. `SBX-EV-001` through `SBX-EV-010` are complete for at least one candidate row.
2. `docs/development/mcp-validation-register.json` is updated for the exact approved row.
3. `docs/planning/decision-log.md` records the RAJA authorization decision.
4. BA-EM-005 remains `0`.
5. BA-EM-009 remains `0`.
6. Synthetic fixtures remain the fallback path.

## 7) Rollback and stop triggers

If a future sandbox row is authorized and execution later begins, stop and revert to synthetic mode if any trigger occurs:

| Trigger | Required response |
| --- | --- |
| Unapproved data field appears | Stop run, preserve audit, route to security/privacy review |
| Tool schema differs from approved schema | Stop row, mark validation stale, return to blocked |
| Auth/scope mismatch occurs | Stop row, revoke candidate status, route to platform/tool owner |
| Write-like action becomes reachable | Stop immediately; BA-EM-005 hard gate failure |
| MVP route receives Phase 2 behavior | Stop immediately; BA-EM-009 hard gate failure |
| Retention/residency rule missing or violated | Stop row and route to RAJA/security/privacy |

## 8) Current recommendation

Proceed with **Prepare only** using the existing local MCP as upstream evidence.

1. Prioritize `P2-SBX-JIRA-READ` because `/jira-cloud/mcp` has protocol-level evidence and identifiable read-like tools.
2. Use the BA Agent wrapper that permits only `FetchItrackJiraIssuesList`, `GetJiraItrackJobStatus`, and `JiraItrackValidate`; explicitly deny `CreateJiraCloudIssue`, `UpdateJiraCloudIssue`, `UpdateJiraCloudStatus`, `DeleteJiraCloudIssue`, `RevertJiraItrackIssue`, approval tools, and any comments/updates.
3. Complete `docs/development/phase-2-jira-sandbox-evidence-package.md` before any execution request: owner/reviewer names, exact scope, classification, field policy, retention/residency, complete schemas, auth/rate-limit posture, audit acceptance, and security/control review.
4. Treat `P2-SBX-CONF-READ` as second priority: the BA Agent read-only allowlist boundary is now prepared, but execution still requires complete row-level scope, classification, schema, auth/rate-limit, audit/control review, and explicit RAJA authorization.
5. Keep `P2-SBX-GIT-READ` in preparation-only backlog until a Git/GitHub read endpoint/tool family exists; use `docs/development/phase-2-git-sandbox-evidence-package.md` as the evidence checklist.
6. Keep `P2-SBX-TEAMS-READ` deferred/blocked; use `docs/development/phase-2-teams-sandbox-blocked-package.md` only as a blocked-path record.
7. Use `docs/development/phase-2-sandbox-owner-review-package.md` as the cross-row RAJA/tool-owner review artifact before any row-level execution decision.
6. Keep `P2-SBX-TEAMS-READ` deferred/blocked.

Until row-level owner, scope, classification, auth/rate-limit, audit, no-write proof, updated register evidence, and explicit RAJA execution authorization are complete, all sandbox execution paths remain blocked.
