# BA Agent Phase 0 Baseline Review

This Phase 0 review confirms the current planning baseline before implementation starts. It is a docs-only gate artifact and does not approve source scaffolding beyond the G0 synthetic-only boundary.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Phase 0 Baseline Review |
| Version | 0.1 |
| Status | Completed for F0 review |
| Prepared date | 2026-07-03 |
| Accountable owner | RAJA |
| Execution prompt | [P0A] |
| Planning baseline | `docs/planning/project-development-plan.md` v0.3 |
| Decision baseline | `docs/planning/decision-log.md` v0.3 |
| Requirement baseline | `docs/requirements/business-analyst-agent-requirements.md` v0.4 |

## Baseline confirmation

| Baseline item | Confirmation | Evidence |
| --- | --- | --- |
| Accountable owner | RAJA is the accountable owner for the planning baseline. | `decision-log.md` DEC-002; `project-development-plan.md` document control |
| Build-start scope | G0 clears synthetic-only engineering foundation work. | `decision-log.md` G0 build-start assessment |
| First build target | Synthetic Teams standup summary thin slice using synthetic Jira/Git fixtures, evidence refs, `trace_id`, Adaptive Card payload, and no live writes. | `decision-log.md` DEC-003 |
| User surface | Teams/Copilot 365 is the target collaboration surface. | `business-analyst-agent-requirements.md` BA-MVP-FR-001; `project-development-plan.md` constraints |
| Orchestration and tool constraints | LangGraph-compatible orchestration and MCP-mediated tool access remain design constraints. | `business-analyst-agent-requirements.md` design constraints; `ba_agent_runtime_architecture.md` source-fixed constraints |
| Live access | No live system-of-record reads or writes are authorized by G0. | `decision-log.md` G0 build-start assessment |
| MCP state | MCP integrations remain stubbed/blocked until validated. | `decision-log.md` DEC-006 |
| Phase separation | Phase 2 Enterprise BA capabilities remain readiness/planning-only until G7 and a separate plan. | `project-development-plan.md` Phase 7; `business-analyst-agent-requirements.md` BA-QG-008 |

## DEC-001 through DEC-007 comparison

| Decision | Decision log status | Development-plan gate impact | Baseline review |
| --- | --- | --- | --- |
| DEC-001 — Product name and positioning | Closed | Clears stakeholder/UX naming for G0. | Use BA Agent as working name and Business Analyst AI Agent as formal document name. |
| DEC-002 — Accountable ownership | Closed | Clears accountability/RACI blocker for synthetic-only build start. | RAJA is accountable across roles; delegates may be added later without changing accountability. |
| DEC-003 — Thin-slice scope/no-live-write rule | Closed | Clears G0/G2 scope. | Synthetic standup thin slice is the first build target; live writes remain blocked. |
| DEC-004 — Pilot boundaries | Deferred | Blocks G4/G6, not G1-G3. | Does not block synthetic/local work; pilot team, Jira project, repo, Teams channel, Confluence space, and calendar scope remain [RAJA]. |
| DEC-005 — Classification handling path | Conditional | Blocks non-synthetic data use before G4/G6. | Synthetic data is allowed for G1-G3; non-synthetic/restricted data remains blocked until RAJA confirms handling rules. |
| DEC-006 — MCP validation | Conditional | Blocks G4 sandbox replacement of fixtures. | Proposed MCP contracts are not build-authoritative; live/sandbox tools remain stubbed/blocked until validation. |
| DEC-007 — Approval-record and `approval_ref` semantics | Closed | Establishes G3 control design baseline. | Writes fail closed unless a valid, single-use, scope-bound `approval_ref` is presented; G3 must prove BA-EM-005 = 0. |

## Source-alignment findings

| Finding | Impact | Disposition |
| --- | --- | --- |
| G0 is already recorded as clear for synthetic-only engineering foundation work. | [F1] may start only after [F0] evidence is complete and RAJA/G0 acceptance is recorded or explicitly waived by RAJA. | No blocker for [F0]. |
| DEC-004/005/006 are not fully closed. | They block later sandbox/non-synthetic/live work, not local synthetic [F1]. | Track in risk/open-question triage. |
| Phase 1 owns technical baseline, source scaffold, safe local commands, and fixture/eval placeholders only. | Prevents Phase 1 from prematurely building actual standup fixtures or seed evals. | Actual fixture loading and seed cases remain [F2]/[F3] work. |
| `prompts.md` and `fleet_prompt.md` are execution artifacts, not product evidence. | Prevents execution instructions from being cited as functional requirement evidence. | Use only for implementation orchestration. |

## Out-of-scope for this baseline review

- Source code scaffolding.
- Runtime command creation.
- CI, IaC, or cloud deployment.
- Live Jira, Git, Confluence, Calendar, Teams, Copilot 365, Graph API, model, or MCP connectivity.
- Phase 2 Enterprise BA implementation.

## QA handoff

[P0A] is ready for [Q0A]. No implementation files were created or modified by this prompt.
