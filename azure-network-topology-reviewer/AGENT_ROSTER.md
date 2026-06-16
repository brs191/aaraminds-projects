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

## Project-delivery agents (skills-pack) — **authored 2026-06-15**

The playbook's project-delivery / lifecycle agents now exist as native Claude agents in
`skills-pack/.claude/agents/` and are wired:

| Agent | Role |
|---|---|
| `aara-project-architect` | system design, decomposition, ADRs, brownfield evolution → design docs |
| `aara-project-planner` | outcome-defined phases, T-shirt estimates, critical path, risk register |
| `aara-project-builder` | execute a playbook step/ticket: code + tests + green gate + Result log |
| `aara-project-reviewer` | adversarial acceptance review → `PHASE_n_ACCEPTANCE_MEMO` (gates cited to file:line) |
| `aara-project-debugger` | reproduce → root-cause → minimal fix + regression test |
| `aara-python-ai-developer` | Python/LLM-orchestration (explainer, generator intent, reference engines, viz pipeline) |
| `aara-ai-evaluation-engineer` | build/run eval gates (precision/recall, diagram-eval, twin-drift, triggering); prove teeth |

> The `~/projects/for-submission/*.agent.md` files are the **GitHub Copilot-format** equivalents of the
> architect/planner/reviewer (different platform: `model: gpt-5`, `handoffs:`). The Claude-format pack
> agents above are the canonical ones the AaraMinds playbooks reference.

## Action — complete

1. ~~Activate `aara-topology-visualizer`~~ — **done:** installed + wired; reviewer composes the policy + Defender skills.
2. ~~Author the project-delivery agents~~ — **done:** all 7 authored in Claude format + wired; every
   playbook-referenced agent name now resolves. Optional follow-up: trim `IMPLEMENTATION_PLAYBOOK.md`
   prompts to point at these canonical names where they drifted.
