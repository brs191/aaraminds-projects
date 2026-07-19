# BA Agent MVP Pilot Release Notes

Release-note package for the MVP pilot candidate. This document does not authorize live pilot execution.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent MVP Pilot Release Notes |
| Version | 0.1 |
| Status | Draft for G6 authorization review |
| Prepared date | 2026-07-03 |
| Accountable owner | RAJA |
| Execution prompt | [P6C] |

## Version identifiers

| Artifact | Version / ID |
| --- | --- |
| Python package | `ba-agent` 0.1.0 |
| Prompt pack | `prompts.md` v0.6 |
| Fleet guide | `fleet_prompt.md` v0.4 |
| Standup fixture set | `synthetic-standup-v1` |
| MVP synthetic fixture marker | `mvp-synthetic-v1` |
| Standup graph version | `phase2-synthetic-standup` |
| Gateway control version | `gateway-control-local` |
| Model version | No model integration in this release |
| Container image | Not applicable; no container build/publish |

## Harness run IDs and gate results

| Eval set | Run ID | Result |
| --- | --- | --- |
| GTS-STANDUP | `run-GTS-STANDUP-synthetic` | Passed |
| GTS-ROUTER | `run-GTS-ROUTER-synthetic` | Passed; BA-EM-009 = 0 |
| GTS-GATE | `run-GTS-GATE-synthetic` | Passed; BA-EM-005 = 0 |
| GTS-PLANNING | `run-GTS-PLANNING-synthetic` | Passed |
| GTS-RETRO | `run-GTS-RETRO-synthetic` | Passed |
| GTS-HEALTH | `run-GTS-HEALTH-synthetic` | Passed |
| GTS-MVP | `run-GTS-MVP-synthetic` | Passed |

## Hard-gate evidence

| Gate | Required | Actual |
| --- | --- | --- |
| BA-EM-005 approval-gate bypass count | 0 | 0 |
| BA-EM-009 Phase-separation violations | 0 | 0 |

## Known limitations

- No actual sandbox MCP server schema is validated.
- No live Jira/Git/Teams/Confluence/Calendar/Graph/model/MCP integration is enabled.
- Pilot scope is `[RAJA]`.
- Security/privacy/classification approval is `[RAJA]`.
- Tool-owner scopes are `[RAJA]`.
- Teams channel approval is `[RAJA]`.
- Owner-threshold metrics remain measured/no-threshold until RAJA sets values.

## Blocked tools

| Tool | Status |
| --- | --- |
| `get_sprint_status` | Not validated |
| `get_recent_activity` | Not validated |
| `send_adaptive_card` | Blocked |
| All write-like tools | Blocked/approval-gated |

## Rollback references

- Code rollback: no git baseline initialized; use file checkpoint/backup process [RAJA] until version control is initialized.
- Prompt/graph rollback: use checked-in prompt/graph artifacts from this working tree.
- Model rollback: not applicable; no model integration.
- Container rollback: not applicable; no container image or Artifactory tag exists.

No Azure ACR, stored cloud credential, out-of-band production prompt edit, or registry publishing path is introduced.
