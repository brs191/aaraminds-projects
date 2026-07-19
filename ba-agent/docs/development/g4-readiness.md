# BA Agent G4 Readiness Evidence

This document records Phase 4 sandbox-readiness evidence. It recommends readiness for RAJA/G4 review of the validation process; it does not approve any sandbox read, live pilot use, production deployment, or system-of-record write.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent G4 Readiness Evidence |
| Version | 0.1 |
| Status | Draft for RAJA/G4 review |
| Prepared date | 2026-07-03 |
| Accountable owner | RAJA |
| Execution prompts | [P4A], [P4B], [P4C], [P4D], [P4E] |
| G3 evidence | `docs/development/g3-readiness.md` |

## G4 readiness verdict

G4 readiness artifacts are in place. No actual MCP tool is validated, no sandbox read is enabled, and all unvalidated tools remain blocked.

## Deliverables

| Area | Delivered path |
| --- | --- |
| Sandbox validation plan | `docs/development/sandbox-validation-plan.md` |
| MCP validation process | `docs/development/mcp-schema-validation-process.md` |
| Validation register | `docs/development/mcp-validation-register.json` |
| Read-only replacement path | `docs/development/read-only-sandbox-replacement.md` |
| Teams readiness | `docs/development/teams-sandbox-readiness.md` |
| Adapter boundary | `src/ba_agent/adapters.py` |
| Validation tooling | `src/ba_agent/validation.py`, `make validate-mcp` |

## Validation register summary

| Tool | Status | Blocker |
| --- | --- | --- |
| `get_sprint_status` | Not validated | Server name, owner, scope, and actual schema remain [RAJA]. |
| `get_recent_activity` | Not validated | Git provider, repo scope, owner, and actual schema remain [RAJA]. |
| `send_adaptive_card` | Blocked | Teams sandbox channel and auto-response policy are not approved. |

## Command evidence

| Check | Command | Result |
| --- | --- | --- |
| Full local check | `make check` | Passed |
| Validation register shape | `make validate-mcp` | Passed; reports no validated read tools |

`validated` rows require sandbox environment, read permission, named owner/server, approved scopes, actual request/response schema refs, schema diff, auth model, rate-limit documentation, external approval evidence, validation timestamp, and no open blockers.

## Guardrail evidence

- Synthetic mode remains default.
- `sandbox_read` mode fails closed unless Jira/Git read tools are validated.
- Teams posting remains disabled.
- Write-like actions remain rejected by gateway controls.
- G4 does not authorize live pilot or production use.

## G5/G6 blockers

- Tool owners remain [RAJA].
- Actual MCP server names remain [RAJA].
- Approved Jira project/repo/channel scopes remain [RAJA].
- Actual schemas are not captured.
- Teams sandbox channel is not approved.
- Write permissions remain blocked.
