# Agent Roster — reconciliation

The playbooks reference several `aara-*` agents. They come from **two different homes** with two
different purposes. This note removes the ambiguity flagged in review (the playbook cited agents that
are not in the engineering skills-pack).

## Engineering agents (skills-pack) — `~/projects/aaraminds/skills-pack/.claude/agents/`

These are the durable, native-format agents that orchestrate the engineering skills:

| Agent | Role |
|---|---|
| `aara-network-topology-reviewer` | reachability/severity review + cost/gen orchestration (produces the *report*). Now also composes `azure-iac-policy-as-code` (policy gate) and `azure-defender-signal-ingestion` (consume Defender). |
| `aara-topology-visualizer` | produces the *diagram* (Phase 4); consumes the analyzer for severity. **Activated + wired (2026-06-15).** |
| `aara-mcp-server-builder` | builds the Go MCP engine |
| `aara-senior-microservices-architect` | microservices architecture |
| `aara-azure-cost-reviewer` | billing actuals / FinOps |

> **Status (2026-06-15):** `aara-topology-visualizer` is now installed in
> `skills-pack/.claude/agents/` and wired (`wire-skills.sh`). The three new skills
> (`azure-network-topology-visualization`, `azure-iac-policy-as-code`,
> `azure-defender-signal-ingestion`) are installed and wired; the reviewer agent invokes the latter two.

## Project-delivery agents — `~/projects/for-submission/*.agent.md`

The playbook's `aara-project-architect`, `aara-project-builder`, `aara-project-reviewer`,
`aara-project-debugger`, `aara-python-ai-developer`, `aara-ai-evaluation-engineer` are
**project-delivery / orchestration** agents, not engineering-pack agents. The architect / planner /
reviewer variants exist as `project-architect.agent.md`, `project-planner.agent.md`,
`project-reviewer.agent.md` under `~/projects/for-submission/`. The builder / debugger /
python-ai-developer / ai-evaluation-engineer variants are referenced by the playbook but are **not yet
authored** anywhere — they are aspirational.

## Action

1. ~~Activate `aara-topology-visualizer`~~ — **done (2026-06-15):** installed + wired; reviewer agent
   updated to compose the two new skills. Engineering-pack agent coverage for the adoption roadmap is complete.
2. Decide on the project-delivery agents: either (a) author the missing four and standardize all six
   into one location, or (b) update `IMPLEMENTATION_PLAYBOOK.md` to reference only agents that exist.
   Until then, treat playbook agent prompts as role descriptions, not guaranteed-present agents.
