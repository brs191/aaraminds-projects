# Copilot instructions for the BA Agent repository

## Project state and validation

This repository is currently documentation-only: the product baseline lives under `docs/requirements/`, planning and gate decisions live under `docs/planning/`, and execution orchestration lives in root-level `prompts.md` and `fleet_prompt.md`. No source tree, build manifest, lint config, test runner, Makefile, or runnable evaluation harness is checked in yet, so there are no build/test/lint commands or single-test commands to run from this repo.

Do not invent validation commands. For documentation changes, validate by reading the changed Markdown and cross-checking requirement IDs, source citations, document-control metadata, and companion-doc references. `docs/requirements/ba_agent_evaluation_harness.md` defines the intended golden-test harness, but it is a specification, not an executable harness in this repository.

## Source-of-truth documents

- `docs/requirements/business-analyst-agent-requirements.md` is the primary baseline. It is a v0.4 draft for human review, not an approved delivery commitment.
- Companion docs extend the baseline: `ba_agent_runtime_architecture.md`, `ba_agent_mcp_tool_contracts.md`, `ba_agent_evaluation_harness.md`, and `ba_agent_operations_model.md`.
- `docs/planning/project-development-plan.md` and `docs/planning/decision-log.md` govern gate sequencing and current RAJA decisions.
- `prompts.md` is the execution contract for prompt-by-prompt implementation; `fleet_prompt.md` is the fleet orchestration guide.
- `docs/requirements/ba-requirements-prompt.md` is task framing. Do not cite it as product evidence for functional or non-functional requirements.
- The requirements doc references external source IDs `S1`-`S6`; if those external source files are not available in the current session, preserve existing citations and mark new unsupported conclusions as `[inferred]` or `[RAJA]`.

## Global Aara assets

The user's Aara agents, skills, and personas are globally available from `~/.copilot`. Do not duplicate those global files into this repository; use this repo file only for project-specific routing.

For BA Agent requirements work, prefer the global `aara-business-analyst` agent as the primary drafting/review assistant. Use supporting global agents only for their lane: `aara-agent-blueprint-advisor` for agent controls/governance, `aara-ai-application-architect` for AI runtime topology, `aara-ai-evaluation-engineer` for gates and golden datasets, `aara-project-architect` for system integration implications, `aara-project-planner` for sequencing, and `aara-executive-narrative-advisor` only for executive-summary polish.

Apply global persona lenses as review lenses, not as source evidence. If agent/skill invocation is unavailable in a session, apply the same routing descriptions manually and state that invocation was unavailable.

## High-level architecture

The product is split into an Agile/Scrum MVP and later Phase 2 enterprise BA capabilities. Keep these separate: MVP covers standup summaries, sprint-planning recommendations, retrospective reports, and sprint-health monitoring; Phase 2 covers requirement discovery, story and acceptance-criteria drafting, process mapping, gap/impact analysis, traceability, BRD/FRD/PRD drafts, and test-scenario inputs.

The source-fixed MVP shape is Teams/Copilot 365 as the user surface, LangGraph orchestration, and MCP-mediated access to Jira, Git, Confluence, Calendar, and Teams. The proposed runtime architecture uses a Python `orchestrator-svc` with a LangGraph router and four capability graphs, a separate `mcp-gateway-svc` enforcing auth/scope/approval/audit controls, per-system MCP server containers, Postgres for LangGraph checkpoints plus approval/idempotency records, Event Hubs for audit/webhook fan-out, and Azure OpenAI called only by the orchestrator.

Writes to systems of record are approval-gated. The gateway validates `approval_ref` and idempotency before any write tool can update Jira, publish Confluence content, or send approved outbound actions. Git and Calendar are read-only in the current contracts.

## Codebase-specific conventions

- Maintain evidence discipline: source-backed claims get source citations; reasonable but unsupported conclusions get `[inferred]`; owner-dependent values, thresholds, names, and implementation details get `[RAJA]`.
- Keep stable requirement ID families intact: `BA-MVP-FR-*`, `BA-P2-FR-*`, `BA-NFR-*`, `BA-INT-*`, `BA-HIL-*`, `BA-AUT-*`, `BA-DSPC-*`, `BA-QG-*`, `BA-EM-*`, `BA-RISK-*`, `BA-DEP-*`, and `BA-OQ-*`.
- Preserve human-gated wording. Agent output is advisory, drafting, or approval-gated; no MVP or Phase 2 capability is autonomous.
- Keep Teams/Copilot 365 as the collaboration surface. Do not introduce Slack language.
- If registry, CI/CD, or Azure infrastructure appears in implementation docs, align with the existing proposal: GitHub Actions OIDC, Terraform AzureRM, Key Vault with managed identities, Azure-primary services, and JFrog Artifactory for container images.
- MCP tool contracts use explicit audit records, `degraded`/`denied`/`throttled` statuses, `source_timestamp` distinct from `retrieved_at`, idempotent reads, and `approval_ref` plus `idempotency_key` for writes. Treat every external side effect as write-like, including drafts, webhook subscriptions, Teams posts/escalations, and approval records; `approval_request_id` is never an `approval_ref`.
- Evaluation gates are defined in the harness spec. Treat all owner-set numeric thresholds as `[RAJA]`; hard gates are zero approval-gate bypasses and zero MVP/Phase 2 separation violations.
- For execution artifacts, `prompts.md` and `fleet_prompt.md` are the active execution contract: do not introduce the old VERIFY marker; use `[RAJA]` for owner-dependent execution decisions.
- When changing any requirements or companion document, update its document-control table and keep sibling-document references consistent.
