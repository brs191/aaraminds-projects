# BA Agent Sandbox Validation Plan

This plan prepares G4 sandbox validation without enabling sandbox reads, live pilot use, production deployment, or write-like tools.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Sandbox Validation Plan |
| Version | 0.1 |
| Status | Draft for G4 readiness |
| Prepared date | 2026-07-03 |
| Accountable owner | RAJA |
| Execution prompt | [P4A] |
| Prior gate evidence | `docs/development/g3-readiness.md` |

## Scope

G4 prepares validated **read-only sandbox replacement** for synthetic Jira/Git data. It does not authorize live pilot use, production use, unapproved tools, or system-of-record writes.

## Candidate sandbox reads

| Tool | Purpose | Initial G4 status |
| --- | --- | --- |
| `get_sprint_status` | Jira sprint status/story evidence for standup. | Not validated |
| `get_recent_activity` | Git commit/PR evidence for standup. | Not validated |

Other tools remain out of scope unless RAJA explicitly adds them to the validation register.

## Validation requirements

Before any sandbox read can replace synthetic fixtures, the tool row must have:

1. Named tool owner.
2. Actual MCP server name.
3. Actual request/response schema captured from approved sandbox metadata.
4. Proposed-vs-actual schema diff recorded.
5. Auth model documented.
6. Approved scopes listed.
7. Rate limits documented.
8. Validation status set to `validated`.
9. No open blockers.

## Rollback

If validation fails or sandbox access is unavailable, the system remains in synthetic mode. Synthetic fixtures are the fallback and default.

## Current validation register

The current working register is:

`docs/development/mcp-validation-register.json`

No read tool is marked validated yet.
