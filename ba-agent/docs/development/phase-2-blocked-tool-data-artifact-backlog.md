# BA Agent Phase 2 Blocked Tool/Data/Artifact Backlog

This artifact closes the synthetic-maturation review for `P2-MAT-004`, `P2-MAT-005`, and `P2-MAT-006` by explicitly retaining the blocked-by-default posture for external tools, non-synthetic data, and artifact storage/publishing. It does not authorize sandbox, live, pilot, production, non-synthetic data, external publish/storage, or write-like behavior.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Phase 2 Blocked Tool/Data/Artifact Backlog |
| Version | 0.7 |
| Change note (v0.7) | Linked the Teams blocked package and clarified Teams remains excluded from approved preparation lanes. |
| Change note (v0.6) | Linked Jira, Confluence, and Git/GitHub row-level evidence packages plus the consolidated sandbox owner-review package while preserving blocked execution. |
| Change note (v0.5) | Linked the Jira row-level sandbox evidence plan while preserving blocked execution until all row evidence and RAJA authorization are complete. |
| Change note (v0.4) | Recorded BA Agent Jira read-only wrapper evidence while retaining execution block until row-level evidence and RAJA authorization are complete. |
| Change note (v0.3) | Recorded partial local MCP evidence for Jira and Confluence and preserved blocked-by-default execution posture until allowlist, auth/rate-limit, scope, classification, and no-write evidence are complete. |
| Change note (v0.2) | Linked the Phase 2 sandbox authorization package and preserved blocked posture pending row-level approval evidence. |
| Status | Blocked posture retained; Jira, Confluence, Git/GitHub evidence packages support preparation only; Teams blocked package records exclusion |
| Prepared date | 2026-07-13 |
| Accountable owner | RAJA |
| Related maturation items | `P2-MAT-004`, `P2-MAT-005`, `P2-MAT-006`, `P2-MAT-EXIT-007` |
| Related decisions | `P2-DEC-009`, `P2-DEC-010`, `P2-DEC-012` |
| Primary references | `docs/development/p2-g4-tool-data-readiness.md`; `docs/development/phase-2-tool-approval-matrix.md`; `docs/development/phase-2-data-classification-plan.md`; `docs/development/mcp-validation-register.json`; `docs/development/phase-2-synthetic-maturation-package.md`; `docs/development/phase-2-sandbox-authorization-package.md`; `docs/development/phase-2-sandbox-owner-review-package.md` |
| Explicit non-authorization | No sandbox, live, pilot, production, non-synthetic data, external publish/storage, or write-like side effect |

## 1) Closure verdict

For the current synthetic-only maturation boundary:

1. **P2-MAT-004 is closed as blocked:** all external tools remain blocked because owner, scope, security/privacy, platform, schema, auth, rate-limit, and approval evidence are incomplete.
2. **P2-MAT-005 is closed as blocked:** all non-synthetic data remains blocked because classification, redaction, retention, residency, and allowed-source decisions are not approved.
3. **P2-MAT-006 is closed as blocked:** external artifact storage/publishing remains blocked because no publish/storage policy or write-like side-effect approval evidence exists.

This is a closure of the review item, not an enablement decision. The paths remain unavailable until a separate owner-approved authorization package is provided.

The current separate authorization artifact is `docs/development/phase-2-sandbox-authorization-package.md` v0.9. It captures partial local MCP evidence for Jira and Confluence preparation, adds Jira and Confluence read-only wrapper proof, and links Jira, Confluence, and Git/GitHub row-level evidence packages plus a consolidated owner-review package. It does not approve, validate for execution, enable, or execute any sandbox row.

## 2) External tool backlog

| Tool / path | Current posture | Missing evidence before enablement |
| --- | --- | --- |
| Jira | Blocked for execution; partial local MCP schema evidence, local BA Agent wrapper proof, and row-level evidence package captured | Named owner/delegates, exact project/board/JQL scope, classification, field/comment policy, retention/residency, complete schemas, auth/rate limits, audit examples, control-review/approval evidence. |
| Confluence | Blocked for execution; partial local MCP schema evidence, local BA Agent wrapper proof, and row-level evidence package captured | Space owner, restricted-page handling, body/comment/attachment policy, draft/publish policy, auth/rate limits, approval evidence. |
| GitHub / Git | Blocked for execution; row-level evidence package captured; no endpoint/tool family identified | Repo scope, source-code classification, comment/write policy, actual endpoint/tool family, schema validation, auth/rate limits, approval evidence. |
| Azure DevOps | Blocked | Organization/project scope, pipeline metadata handling, work-item write policy, actual schema validation, auth/rate limits, approval evidence. |
| SharePoint | Blocked | Document owner, classification, retention, scope, actual schema validation, auth/rate limits, approval evidence. |
| Teams | Blocked; excluded from approved preparation lanes; blocked package captured | Separate RAJA reopen decision, channel scope, tenant/app approval, message retention, send policy, actual schema validation, approval evidence. |
| Miro/Draw.io | Blocked | Tool choice, board ownership, export/sharing policy, schema/API validation, approval evidence. |
| SQL/Data | Blocked | Classification, data owner, query scope, metadata-vs-row policy, raw-data approval, approval evidence. |
| ServiceNow | Blocked | Queue scope, operational owner, assignment policy, schema validation, approval evidence. |
| Test-management tools | Blocked | Tool selection, QA owner, project scope, write policy, schema validation, approval evidence. |

## 3) MCP validation register status

| Register row | Permission | Current status | Blocking reason |
| --- | --- | --- | --- |
| `get_sprint_status` | read | Partial local MCP schema evidence plus Jira row-level evidence package; not validated for execution | Jira scope, classification, redaction, retention/residency, auth/rate limit, audit evidence, and control-approved no-write proof are not complete. |
| `get_confluence_source_pages` | read | Partial local MCP schema evidence plus Confluence row-level evidence package; not validated for execution | Confluence scope, classification, restricted-page/body/comment/attachment policy, retention/residency, auth/rate limit, audit evidence, and control-approved no-write proof are not complete. |
| `get_recent_activity` | read | Git/GitHub row-level evidence package prepared; not validated | No Git/GitHub MCP endpoint/tool family has been identified; Git provider, repository scope, source-code classification, and actual schema are not confirmed. |
| `send_adaptive_card` | write-like | Blocked | Teams sandbox channel, auto-response policy, and external side-effect approval are not approved. |

No register row is validated or enabled for Phase 2.

## 4) Non-synthetic data backlog

| Data class / decision | Current posture | Missing evidence before use |
| --- | --- | --- |
| Real meeting notes, emails, tickets, customer requests, source documents | Blocked | Classification, privacy, retention, redaction, allowed-source, and review-lane approvals. |
| Source code or restricted/internal/security-sensitive documents | Blocked | Security/privacy approval, source-code handling policy, minimization/redaction controls, and approved scope. |
| Regulated/legal text | Blocked | Compliance/legal review, obligation-handling policy, and non-approval controls. |
| Raw SQL/data rows | Blocked | Data owner, classification, query scope, row-level handling, retention/residency, and approval evidence. |
| Tool-origin evidence | Blocked | Tool row approval, schema validation, scope, auth/rate limit, and classification evidence. |

Synthetic GTS-P2-REQ fixtures remain the only approved Phase 2 input source.

## 5) Artifact storage/publishing backlog

| Artifact path | Current posture | Missing evidence before use |
| --- | --- | --- |
| External Confluence draft/page | Blocked | Space owner, classification, draft/publish policy, approval ref semantics, idempotency, audit, and human-gated publish policy. |
| Teams message/card send | Blocked | Channel/app approval, send policy, retention, approval ref semantics, idempotency, and audit. |
| Jira/GitHub/Azure DevOps comments or updates | Blocked | Tool owner, scope, comment/update policy, approval ref semantics, idempotency, and audit. |
| SharePoint/document storage | Blocked | Document owner, retention, classification, residency, approval ref semantics, and audit. |
| Local synthetic artifacts | Allowed only for local synthetic evaluation | Must remain local/test-only and must not contain non-synthetic data or credentials. |

All external artifact publication or storage remains write-like and fail-closed.

## 6) Reopen / enablement rule

Any future request to unblock one of these paths must provide a separate authorization package with:

1. Named owner and acting review lane.
2. Explicit allowed scope.
3. Security/privacy classification decision.
4. Retention, residency, and redaction decision where applicable.
5. Actual schema/auth/rate-limit validation evidence.
6. Write-like policy, approval-ref semantics, idempotency, and audit evidence for any side effect.
7. Updated decision-log evidence and traceability updates where mappings or eval coverage change.

Until all required evidence is present, the path remains blocked.
