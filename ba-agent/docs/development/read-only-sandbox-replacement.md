# BA Agent Read-Only Sandbox Replacement Path

This document defines the prepared path for replacing synthetic Jira/Git reads with validated sandbox MCP reads. No sandbox read is enabled by this document.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Read-Only Sandbox Replacement Path |
| Version | 0.1 |
| Status | Draft for G4 readiness |
| Prepared date | 2026-07-03 |
| Accountable owner | RAJA |
| Execution prompt | [P4C] |

## Data-source modes

| Mode | Status | Behavior |
| --- | --- | --- |
| `synthetic` | Default | Uses local synthetic fixtures only. |
| `sandbox_read` | Prepared, blocked | Requires validated `get_sprint_status` and `get_recent_activity` rows with approved scopes. |

The implementation exposes `BA_AGENT_DATA_SOURCE_MODE`, but sandbox mode fails closed unless the validation register permits the required read tools.

## Adapter boundary

`src/ba_agent/adapters.py` defines:

- `StandupReadAdapter`
- `SyntheticStandupReadAdapter`
- `SandboxReadAdapter`
- `build_standup_read_adapter`

The synthetic adapter remains the default. The sandbox adapter currently checks validation state and then raises a blocked error because actual sandbox reads are not implemented in G4.

## Evidence preservation requirements

Any future sandbox read adapter must preserve:

- `source_timestamp`
- `retrieved_at`
- evidence refs
- `trace_id`
- denied/degraded/throttled statuses
- audit records through the gateway/control layer

## Writes remain blocked

Jira and Git write actions remain absent or blocked. This replacement path is read-only.
