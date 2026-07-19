# BA Agent Phase 2 Tool Approval Matrix

This matrix prepares Phase 2 integration review. It does not enable any tool, approve any scope, grant any credential, or authorize any live integration.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Phase 2 Tool Approval Matrix |
| Version | 1.0 |
| Change note (v1.0) | Added reusable sandbox evidence intake template linkage for owner-provided row evidence. |
| Change note (v0.9) | Added Teams blocked package linkage; Teams remains excluded from approved preparation and execution lanes. |
| Change note (v0.8) | Added consolidated sandbox owner-review package linkage for Jira, Confluence, and Git/GitHub row decisions. |
| Change note (v0.7) | Added Git/GitHub row-level evidence package linkage; no endpoint/tool family is validated and execution remains blocked. |
| Change note (v0.6) | Added Confluence read-only wrapper and row-level evidence package linkage; execution remains blocked. |
| Change note (v0.5) | Added Jira sandbox evidence package linkage for row-level scope, classification, schema, auth/rate-limit, audit, and register-update review. |
| Change note (v0.4) | Added Phase 2 Jira read-only wrapper implementation evidence; write-like Jira tools are denied before upstream calls, but execution remains blocked. |
| Change note (v0.3) | Reflected partial local MCP schema evidence from `apm0045942-cc-mcp-server` while preserving execution-blocked posture and no-write requirements. |
| Change note (v0.2) | Recorded RAJA preparation-only approval for Jira, Confluence, and GitHub/Git read-only sandbox evidence collection; Teams remains blocked. |
| Status | Draft for Phase 2 readiness review; consolidated owner-review package, intake template, and Teams blocked package prepared; execution remains blocked |
| Prepared date | 2026-07-13 |
| Accountable owner | RAJA |
| Execution prompt | [P7B] |
| Requirement baseline | `docs/requirements/business-analyst-agent-requirements.md` v0.4 |
| Phase 2 prioritization input | `docs/development/phase-2-prioritization-brief.md` |

## Matrix rules

1. Every tool defaults to **blocked** until owner, security/privacy, platform, and schema validation are complete.
2. MVP tool readiness does not transfer to Phase 2. Phase 2 requires separate approval because data classes, artifacts, and write intent differ.
3. Candidate writes are draft-only or human-gated by default; no autonomous writes are allowed.
4. No credentials, tenant IDs, endpoint URLs, project keys, repo names, channel IDs, database names, ServiceNow queues, or test-management project IDs are recorded in this matrix.
5. Unknown owners, scopes, data classes, and approval paths remain `[RAJA]`.
6. Tool contracts are not build-authoritative until actual server/schema/auth/scope/rate-limit validation is complete.

## Phase 2 candidate tool matrix

| Tool | Candidate Phase 2 use | Candidate data classes | Read/write intent | Owner | Security/privacy review | Scopes | Validation status | Default action | Blockers |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Jira | Requirements discovery from tickets, backlog context, release items, traceability to delivery work. | Issues, epics, stories, labels, statuses, links, comments [RAJA]. | Read-only first; story/field updates blocked unless human-gated later. | RAJA acting owner | Required | [RAJA] | Partial local MCP schema evidence plus BA Agent read-only wrapper and row-level evidence package; not validated for execution | Approved for sandbox evidence preparation only; execution blocked | Complete `docs/development/phase-2-jira-sandbox-evidence-package.md`: project scope, field mapping, comment privacy, classification, retention/residency, schemas, auth/rate-limit evidence, audit/control review, and RAJA execution authorization. |
| Confluence | Source documents, BRD/FRD/PRD drafts, retro/learning artifacts, traceability pages. | Pages, spaces, labels, draft artifacts, comments [RAJA]. | Read and draft-only candidate; publish human-gated. | RAJA acting owner | Required | [RAJA] | Partial local MCP schema evidence plus BA Agent read-only wrapper and row-level evidence package; not validated for execution | Approved for sandbox evidence preparation only; execution blocked | Complete `docs/development/phase-2-confluence-sandbox-evidence-package.md`: space scope, page/body/comment/attachment policy, restricted-page handling, classification, retention/residency, schemas, auth/rate-limit evidence, audit/control review, and RAJA execution authorization. |
| GitHub | Engineering context, PR/commit evidence, release links, traceability to implementation. | Issues, PR metadata, commits, files where approved [RAJA]. | Read-only first; comments/PR writes blocked unless human-gated later. | RAJA acting owner | Required | [RAJA] | Not validated; no endpoint/tool family identified; row-level evidence package prepared | Approved for sandbox evidence preparation only; execution blocked | Complete `docs/development/phase-2-git-sandbox-evidence-package.md`: provider, repo scope, source-code classification, metadata/diff/comment policy, schema/auth/rate-limit evidence, audit/control review, and RAJA execution authorization. |
| Azure DevOps | Work items, repos, pipelines, release items, test-plan linkage where used. | Work items, repos, pipeline metadata, release metadata [RAJA]. | Read-only first; work-item updates blocked unless human-gated later. | [RAJA] | Required | [RAJA] | Not validated | Blocked | Organization/project scope, pipeline metadata sensitivity, write policy. |
| SharePoint | Source documents, stakeholder files, policy/process docs, approved templates. | Documents, folders, metadata, sharing labels [RAJA]. | Read-only first; document creation/update blocked unless human-gated later. | [RAJA] | Required | [RAJA] | Not validated | Blocked | Information classification, permissions, retention, document scope. |
| Teams | Collaboration surface, approval cards, review notifications, stakeholder clarification workflows. | Messages, Adaptive Cards, approval records, interaction metadata [RAJA]. | Local/draft first; sends are human-gated or policy-approved only. | [RAJA] | Required | [RAJA] | Not validated; Teams preparation not approved; blocked package prepared | Blocked; excluded from approved preparation lanes | Requires separate RAJA reopen decision before any Teams preparation, channel scope, tenant/app approval, message retention, send policy, schema/API validation, or adapter work. |
| Miro/Draw.io | Process maps, current/future state diagrams, gap-analysis visuals. | Boards/diagrams, shapes, comments, exports [RAJA]. | Draft diagram generation candidate; publish/share human-gated. | [RAJA] | Required | [RAJA] | Not validated | Blocked | Tool choice, board ownership, export handling, visual review workflow. |
| SQL/Data | Data touchpoint analysis, reporting impact, source-system lookup where approved. | Schemas, table metadata, data dictionaries, sample rows only if approved [RAJA]. | Metadata read-only first; data reads blocked until classification approval. | [RAJA] | Required | [RAJA] | Not validated | Blocked | Data classification, PII/regulated data, query scope, no raw data until approved. |
| ServiceNow | Operational impact, support process inputs, incident/change context. | Incidents, changes, catalog items, assignment groups [RAJA]. | Read-only first; ticket creation/update blocked unless human-gated later. | [RAJA] | Required | [RAJA] | Not validated | Blocked | Queue scope, assignment policy, operational data classification. |
| Test-management tools / TestRail / QA | Test scenario traceability, test-case inputs, QA acceptance linkage. | Test suites, cases, runs, defects, trace links [RAJA]. | Draft test-scenario inputs first; test-case creation/update human-gated. | [RAJA] | Required | [RAJA] | Not validated | Blocked | Tool selection, project scope, QA ownership, write policy. |

## MVP-to-Phase 2 relationship

| MVP tool | MVP use | Phase 2 relationship |
| --- | --- | --- |
| Jira | Scrum status, backlog, sprint metrics. | Phase 2 may use Jira for requirement traceability and delivery artifacts, but requires separate project/field/write approval. |
| Git/GitHub | Commits and PRs for standup evidence. | Phase 2 may use GitHub for implementation traceability, but source-code classification and repo scope must be re-reviewed. |
| Confluence | Retro artifacts. | Phase 2 may use Confluence for BRD/FRD/PRD drafts and traceability docs, but draft/publish policy must be separately approved. |
| Calendar | Aggregate availability for planning. | Phase 2 calendar use is not a default capability; any scheduling/availability use requires privacy review. |
| Teams | User surface and cards. | Phase 2 may use Teams for review/clarification workflows, but sends/approvals remain gated. |

## Write policy by default

| Write-like action | Default Phase 2 policy |
| --- | --- |
| Create/update Jira work item | Blocked until human-gated write policy and scope are approved. |
| Publish Confluence page | Blocked; draft-only until human-gated publish policy is approved. |
| Send Teams message/card | Blocked until channel/app approval and send policy are approved. |
| Comment on GitHub/Azure DevOps PR or issue | Blocked until human-gated comment policy is approved. |
| Create/update SharePoint document | Blocked until document owner, classification, and retention rules are approved. |
| Create/update ServiceNow ticket | Blocked until operational owner and assignment policy are approved. |
| Create/update test case | Blocked until QA owner and project scope are approved. |
| Create/update Miro/Draw.io board | Blocked until board ownership and sharing rules are approved. |
| Query raw SQL/data rows | Blocked until classification and query scope are approved. |

## Approval evidence required before enablement

Each tool must have:

1. Named owner and review lane.
2. Approved scopes.
3. Security/privacy classification review.
4. Actual MCP/server schema validation.
5. Auth model and rate limits documented.
6. Data retention and audit expectations.
7. Human-gated write policy, if any write-like action is needed.
8. GTS-P2-REQ coverage for tool-origin evidence claims where relevant.

## Current readiness conclusion

No Phase 2 tool is approved or enabled for execution. The local MCP server gives Jira and Confluence a practical preparation path, and both now have BA Agent read-only wrapper boundaries, no-write tests, and row-level evidence packages. Git/GitHub has an evidence package only because no endpoint/tool family has been identified. Teams has a blocked package only because it is excluded from the approved preparation lanes. Use `docs/development/phase-2-sandbox-owner-review-package.md` for consolidated RAJA/tool-owner review and `docs/development/phase-2-sandbox-evidence-intake-template.md` for owner-provided row evidence. Execution still requires completed scope, classification, auth/rate-limit, audit, control-review, and RAJA authorization evidence.
