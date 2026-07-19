# BA Agent Rollback and Kill-Switch Drill

This document records the local/non-production rollback and kill-switch drill for G6 readiness. It does not affect live users, projects, channels, repos, or cloud infrastructure.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Rollback and Kill-Switch Drill |
| Version | 0.1 |
| Status | Completed local drill for G6 readiness |
| Prepared date | 2026-07-03 |
| Accountable owner | RAJA |
| Execution prompt | [P6D] |

## Kill-switch controls

| Control | Current local behavior |
| --- | --- |
| Write-like tools | Gateway rejects or blocks write-like actions. |
| Live integrations | `LIVE_INTEGRATIONS_ENABLED=true` is rejected by config. |
| Sandbox reads | `sandbox_read` mode fails closed without fully validated register rows. |
| Teams send | `send_adaptive_card_stub` raises a blocked error. |
| Phase 2 route | Router blocks Phase 2 Enterprise BA requests. |
| Capability placeholders | Planning/retro/health are advisory/draft/local only; no live sends/writes. |

## Drill evidence

| Drill | Evidence |
| --- | --- |
| Write disablement | `GTS-GATE` passes with approval_gate_bypass_count=0. |
| Route/capability boundary | `GTS-ROUTER` passes with phase_separation_violations=0. |
| Synthetic fallback | `make synthetic-demo` produces local Adaptive Card JSON. |
| Config live-mode rejection | `make no-live` validates default local config; tests reject live mode. |
| Sandbox-read block | `make validate-mcp` shows no validated read tools; adapter tests reject sandbox mode without complete validation evidence. |

## Rollback paths

| Axis | Rollback path |
| --- | --- |
| Code | Use working-tree checkpoint/backup process [RAJA] until git is initialized. |
| Prompt/graph | Revert to prior `prompts.md` / graph artifacts from checkpoint [RAJA]. |
| Model | Not applicable; no model integration. |
| Config | Restore default `BA_AGENT_ENV=local`, `LIVE_INTEGRATIONS_ENABLED=false`, `BA_AGENT_DATA_SOURCE_MODE=synthetic`. |
| Tools | Keep validation register rows non-validated/blocked. |

## Stop condition proof

If any write-like tool succeeds without approval, pilot must stop. Current local eval evidence shows this does not occur.

## Non-goals

- No cloud deployment.
- No container publishing.
- No live pilot.
- No production configuration.
- No live channel/project/repo mutation.
