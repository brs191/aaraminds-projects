# Scrum Master Agent — Requirements Baseline

Traceable requirements baseline for the Scrum Master Agent, in the AaraMinds gated-requirements format. This document does not introduce scope: it restates the accepted PRD, Agent Blueprint, and Architecture as stable, citable requirement IDs so downstream work (tests, tool contracts, evaluation gates, release notes) can reference requirements instead of prose sections.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | Scrum Master Agent Requirements Baseline |
| Version | 0.1 |
| Status | Baseline derived from accepted PRD v0.1; new requirements need a decision-log entry |
| Prepared date | 2026-07-03 |
| Accountable owner | Raja |
| Primary sources | `../Scrum_Master_Agent_PRD.md` (PRD), `../design/Agent_Blueprint.md` (BLU), `../design/Architecture.md` (ARC), `../design/adr/0001-langgraph-orchestration.md` (ADR1) |
| Evidence markers | `[inferred]` = reasonable but not stated in a source; `[VERIFY]` = owner-dependent value not yet confirmed. No other marker scheme is used in this repo. |
| Sibling documents | `../design/MCP_Tool_Contracts.md`, `../evaluation/Evaluation_Harness.md`, `../operations/Operations_Model.md`, `../planning/Decision_Log.md` |

## ID families

`SM-MVP-FR-###` MVP functional · `SM-NFR-###` non-functional · `SM-INT-###` integration · `SM-HIL-###` human-in-the-loop controls · `SM-AUT-###` capability autonomy · `SM-QG-###` quality gates · `SM-RISK-###` risks (PRD §11 owns the register; IDs assigned here only when a requirement cites one).

---

## Defining Operational Constraint (restated)

**Human-approved writes by construction** (BLU §6). No mutation of Jira or any channel occurs without a persisted human approval, enforced structurally by the LangGraph `interrupt()` gate and the `recommendation → approval → action_audit` chain in Postgres — not by prompt or policy. Every requirement below is subordinate to this invariant.

---

## MVP functional requirements

| ID | Requirement | Source | Verified by |
| --- | --- | --- | --- |
| SM-MVP-FR-001 | The agent shall generate a Daily Scrum Brief for the active sprint, grouped by assignee, flagging blocked and stalled items with issue keys, on a schedule before standup. | PRD §6.1 | GTS-BRIEF |
| SM-MVP-FR-002 | The agent shall deliver briefs and recommendations to Microsoft Teams via a Power Automate Workflows webhook with Adaptive Card payloads (not the retired O365 connector path). | PRD §6.1, §8; ARC | teams-adapter unit tests |
| SM-MVP-FR-003 | The agent shall compute a Sprint Health Summary (On-track / At-risk / Off-track) from time-tracking fields (`timeoriginalestimate`, `timeestimate`, `timespent`), remaining-work trend, post-start scope additions, and spillover candidates, with named drivers rather than only a RAG color. | PRD §6.2 | GTS-HEALTH |
| SM-MVP-FR-004 | The agent shall detect dependency-blocked and time-in-status-stale issues using changelog-derived signals and configurable per-team thresholds, producing a ranked list with age and a suggested next action, with zero false positives on Done items. | PRD §6.3 | GTS-BLOCKER |
| SM-MVP-FR-005 | The agent shall review story quality against a configurable Definition-of-Ready (missing acceptance criteria, missing time estimate, vague description, no owner) and produce a concrete rewrite suggestion; it shall never auto-edit a description. | PRD §6.4 | GTS-QUALITY |
| SM-MVP-FR-006 | The agent shall generate Sprint Closing / Retro Insights (completion %, spillover, cycle-time trend, recurring blockers across the last K sprints) and, on approval, emit a `Report.md` with a navigable table of contents. | PRD §6.5 | GTS-RETRO |
| SM-MVP-FR-007 | Every recommendation shall cite the issue key(s) and the triggering signal so the reader can falsify it against Jira. | PRD §5.3; BLU §7 | GTS-* evidence checks |
| SM-MVP-FR-008 | Analytical determinations (status bucketing, time math, staleness, DoR checks) shall be computed in code, not delegated to the LLM; the LLM composes narrative over computed facts. | BLU §8 (Agent → LLM mitigation) | `test_brief.py` |
| SM-MVP-FR-009 | The MVP write surface is exactly: add Jira comment, add Jira label, create follow-up sub-task, and generate a local `Report.md`. No status transitions, description/field edits, or deletions. | PRD §5.5, §3 | GTS-GATE; contract allowlist |
| SM-MVP-FR-010 | The agent shall run from both scheduled triggers and Jira dynamic-webhook events, with scheduled JQL polling as the webhook fallback. | PRD §8; ARC | integration tests `[VERIFY]` (P1) |
| SM-MVP-FR-011 | When source data is missing or stale, the output shall state data freshness / insufficiency explicitly rather than guess. | BLU §8; Eval_Rubric "Trust" | GTS-BRIEF degraded case |

## Non-functional requirements

| ID | Requirement | Source |
| --- | --- | --- |
| SM-NFR-001 | Jira remains the system of record; the agent caches snapshots for analysis only and never holds authoritative state. | PRD §5.1 |
| SM-NFR-002 | All Jira API access is wrapped behind the `jira-mcp` server so API churn is absorbed in one place. | PRD §8 design rule |
| SM-NFR-003 | The agent honors Jira's points-based rate limits: `429` + `Retry-After`, exponential backoff, webhooks over polling, snapshot caching. | PRD §8, §11 |
| SM-NFR-004 | Sensitive issue fields are redacted before LLM calls; model access is region-pinned (Azure) with tenant isolation. | PRD §11; BLU §8 |
| SM-NFR-005 | Secrets live in Key Vault accessed via managed identity; no credentials in code or env files. | BLU §7; ARC |
| SM-NFR-006 | The approval gate fails closed: a malformed or empty resume payload is treated as reject. | BLU §7; `gate.py` |
| SM-NFR-007 | A delivery failure after approval records `action_audit.result = failed`; writes are never left half-recorded. | BLU §7; `test_gate.py` |
| SM-NFR-008 | Interrupted runs resume idempotently from the Postgres checkpointer: one recommendation row per run, completed nodes not replayed. | BLU §8; `test_doc_invariant.py` |

## Integrations

| ID | Integration | Decision | Source |
| --- | --- | --- | --- |
| SM-INT-001 | Jira Cloud | REST v3 + Agile 1.0; JQL via `POST /rest/api/3/search/jql` with `nextPageToken` (legacy `/search` removed); ADF for rich text; OAuth 2.0 3LO with granular scopes and `offline_access`. | PRD §8 |
| SM-INT-002 | Microsoft Teams | Power Automate Workflows webhook + Adaptive Card. | PRD §8 |
| SM-INT-003 | Jira webhooks | Dynamic Webhooks API (`manage:jira-webhook`); registrations expire ~30 days → refresh job; polling fallback. | PRD §8 |
| SM-INT-004 | LLM | Claude / GPT via the orchestrator only; MCP servers and adapters never call the model. | ARC; BLU §5 `[inferred]` for the "orchestrator only" restriction — confirm in contracts |
| SM-INT-005 | State | Postgres on Azure: config, snapshots, LangGraph checkpoints, recommendation/approval/audit chain. | ARC; ADR1 |

## Human-in-the-loop controls

| ID | Control | Requirement | Source |
| --- | --- | --- | --- |
| SM-HIL-001 | Write approval | Every Jira/channel write requires a persisted, per-recommendation human approval; approval is not bulk. | BLU §6, §8 |
| SM-HIL-002 | Rejection path | A rejected or change-requested recommendation produces no write and no `action_audit` row; the rejection is recorded on the approval row. | BLU §11; `test_gate.py` |
| SM-HIL-003 | Human-only decisions | Approval decisions, sprint planning commitments, and people/performance judgments are never delegated to the agent. | BLU §3 |
| SM-HIL-004 | Approval durability | The approval interrupt survives process restarts (Postgres checkpointer); approvals may arrive hours or days later. | ADR1; BLU §7 |

## Capability autonomy classification

| ID | Capability | Autonomy | Notes |
| --- | --- | --- | --- |
| SM-AUT-001 | Daily Brief, Health, Blocker, Quality, Retro analysis + Teams delivery of advisory output | Advisory | Read-only analysis; posting the advisory card to the approved channel is the delivery of the recommendation itself, not a system-of-record write `[inferred]` — confirm whether channel posts also require the gate in P1. |
| SM-AUT-002 | Jira comment / label / sub-task; brief-as-sprint-comment | Approval-gated | SM-HIL-001. |
| SM-AUT-003 | `Report.md` generation | Approval-gated artifact generation | Local artifact, not a remote write; still gated per PRD §6.5. |
| SM-AUT-004 | Status transitions, description edits, deletions, sprint scope changes | Prohibited in MVP | PRD §3; BLU §3. P3 controlled autonomy requires a new baseline. |

## Quality gates

| ID | Gate | Pass condition | Verified by |
| --- | --- | --- | --- |
| SM-QG-001 | DOC gate (hard) | Zero writes without a matching approval row, including under adversarial resume payloads. Not owner-discretionary. | GTS-GATE; `test_gate.py`, `test_doc_invariant.py` |
| SM-QG-002 | Write-surface gate (hard) | Zero write calls outside the SM-MVP-FR-009 allowlist. | GTS-GATE; contract allowlist |
| SM-QG-003 | Evidence gate | Sampled outputs: every claim carries a resolvable issue key + signal; threshold `[VERIFY]` (owner: Raja). | GTS-* + human sample |
| SM-QG-004 | Accuracy gate | Blocker/stale false-positive rate within target `[VERIFY]`; zero FPs on Done items (hard sub-condition). | GTS-BLOCKER |
| SM-QG-005 | Usefulness gate | Pilot SM confirms the Daily Brief is standup-ready with minimal edits (P1 gate condition). | Pilot review |

The P0→P3 phase gates themselves live in `../planning/Roadmap.md`; this table defines what each gate measures, not when it runs.

---

## Change control

New capabilities, write-surface expansion, or autonomy changes require: a PRD change or ADR, a `DEC-###` entry in `../planning/Decision_Log.md`, and new/updated GTS cases before code merges. Phase 2/3 items (backlog grooming, planning, Slack, auto-actions) stay out of this baseline until their gate opens.
