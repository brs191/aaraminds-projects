# BA Agent MCP Schema Validation Process

This process defines how actual MCP server schemas are validated against proposed contracts before sandbox read tools are enabled.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent MCP Schema Validation Process |
| Version | 0.1 |
| Status | Draft for G4 readiness |
| Prepared date | 2026-07-03 |
| Accountable owner | RAJA |
| Execution prompt | [P4B] |

## Validation workflow

For each candidate tool:

1. Name owner and review lane.
2. Identify actual MCP server.
3. Capture actual request schema from approved sandbox metadata.
4. Capture actual response schema from approved sandbox metadata.
5. Diff actual schema against `docs/requirements/ba_agent_mcp_tool_contracts.md`.
6. Record auth model, scopes, rate limits, and failure modes.
7. Confirm no write-like side effect exists for the read path.
8. Record approved sandbox scopes.
9. Mark row `validated` only when all checks pass.

`validated` rows must include `environment=sandbox`, read permission, non-`[RAJA]` owner and MCP server name, approved scopes, actual request/response schema refs, schema-diff ref, auth-model ref, rate-limit ref, external approval evidence ref, validation timestamp, and no open blockers.

## Current priority

Validate read-only tools first:

1. `get_sprint_status`
2. `get_recent_activity`

## Blocked assumptions

- Proposed contracts are not build-authoritative.
- Missing sandbox access is a blocker, not permission to infer schema.
- No tenant ID, token, secret, endpoint, or live credential is committed.
- Unvalidated tools remain blocked by config and gateway controls.

## Local validation command

The local command validates the shape of the working register only:

```bash
make validate-mcp
```

It does not call MCP servers or live endpoints.
