# Agent Roster — reconciliation

The playbooks reference several `aara-*` agents. They come from **two different homes** with two
different purposes. This note removes the ambiguity flagged in review (the playbook cited agents that
are not in the engineering skills-pack).

## Engineering agents (skills-pack) — `~/projects/aaraminds/skills-pack/.claude/agents/`

These are the durable, native-format agents that orchestrate the engineering skills:

| Agent | Role |
|---|---|
| `aara-network-topology-reviewer` | reachability/severity review + cost/gen orchestration (produces the *report*) |
| `aara-topology-visualizer` | **NEW** — produces the *diagram* (Phase 4); consumes the analyzer for severity |
| `aara-mcp-server-builder` | builds the Go MCP engine |
| `aara-senior-microservices-architect` | microservices architecture |
| `aara-azure-cost-reviewer` | billing actuals / FinOps |

> `aara-topology-visualizer` was authored in `skill-staging/agents/` (the `.claude/agents/` dir is
> write-protected in-session). Move it into `skills-pack/.claude/agents/` and re-run the wiring to
> activate, same as the visualization skill.

## Project-delivery agents — `~/projects/for-submission/*.agent.md`

The playbook's `aara-project-architect`, `aara-project-builder`, `aara-project-reviewer`,
`aara-project-debugger`, `aara-python-ai-developer`, `aara-ai-evaluation-engineer` are
**project-delivery / orchestration** agents, not engineering-pack agents. The architect / planner /
reviewer variants exist as `project-architect.agent.md`, `project-planner.agent.md`,
`project-reviewer.agent.md` under `~/projects/for-submission/`. The builder / debugger /
python-ai-developer / ai-evaluation-engineer variants are referenced by the playbook but are **not yet
authored** anywhere — they are aspirational.

## Action

1. Activate `aara-topology-visualizer` (move + wire) — closes the Phase-4 agent gap.
2. Decide on the project-delivery agents: either (a) author the missing four and standardize all six
   into one location, or (b) update `IMPLEMENTATION_PLAYBOOK.md` to reference only agents that exist.
   Until then, treat playbook agent prompts as role descriptions, not guaranteed-present agents.
