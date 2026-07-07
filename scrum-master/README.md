# Scrum Master Agent — Project Home

**Owner:** Raja · **Stage:** plan complete, entering P0 (Foundations)
**Anchor spec:** `Scrum_Master_Agent_PRD.md` · **Workspace:** AaraMinds

This is the working home for building the **Scrum Master Agent** — a Jira-connected Scrum Intelligence Layer that reads sprint state from Jira Cloud, analyzes it, and delivers recommendations into Microsoft Teams (Slack in a later phase) under a strict human-in-the-loop write model: **Read → Analyze → Recommend → Approve → Write**.

> **Code lives in [`code/`](code/).** The LangGraph orchestrator and Go MCP server are nested here under `code/`, alongside the specs, decisions, plans, and tracking that govern them. Docs and code, one home.

## Folder map

```
scrum-master/
├── README.md                    ← you are here: project index
├── Scrum_Master_Agent_PRD.md    the anchor spec — open for the full picture
├── Raja_Instructions.md         drop-zone for your own notes/overrides
├── Persona_Skill_Agent_Usage.md how the persona/skill/agent pass shaped this
├── .github/
│   └── copilot-instructions.md  repo conventions for AI assistants working here
├── product/                     the "why" — stable reference
│   └── Product_Brief.md
├── requirements/                stable requirement IDs (cite these, not prose)
│   └── Scrum_Master_Agent_Requirements.md   SM-* baseline derived from the PRD
├── design/                      the "what & how"
│   ├── Architecture.md          components, stack, Jira integration, data model
│   ├── Agent_Blueprint.md       Module 8 blueprint — boundary, DOC, controls, FMEA
│   ├── MCP_Tool_Contracts.md    jira-mcp/teams-adapter contracts + validation register
│   └── adr/
│       └── 0001-langgraph-orchestration.md
├── planning/                    the "when and in what order"
│   ├── Roadmap.md               four phases (P0 → P3), each with a gate
│   ├── Decision_Log.md          DEC-### locked decisions — reverse via new entry only
│   └── Open_Questions.md        unresolved decisions
├── evaluation/                  the "how we know it's good"
│   ├── Acceptance_Criteria.md   per-feature, testable
│   ├── Success_Metrics.md       leading + lagging indicators
│   ├── Eval_Rubric.md           quality bar / Definition of Done
│   ├── Evaluation_Harness.md    SM-EM metrics, hard gates, golden test sets (GTS-*)
│   └── Test_Strategy.md         test pyramid, DOC-weighted coverage, cases
├── operations/                  how it's run once live
│   └── Operations_Model.md      solo-operator ops: incidents, rollback, release, cadence
├── tracking/                    live execution state
│   ├── Status.md                the dashboard — open this each session
│   └── milestones/
│       ├── P0-Foundations.md
│       ├── P1-MVP.md
│       ├── P2-Expand.md
│       └── P3-Autonomy.md
└── code/                        the implementation (monorepo) — see code/README.md
    ├── apps/                    orchestrator (Python/LangGraph) · jira-mcp + teams-adapter (Go)
    ├── db/migrations/           Postgres schema
    ├── infra/ · .github/        Terraform stub · CI
    └── docker-compose.yml       run the P0 slice
```

## Where to find things

- **The full picture** — `Scrum_Master_Agent_PRD.md`. The anchor; everything else distils or executes it.
- **Why we're building this, for whom** — `product/Product_Brief.md`.
- **How it's built** — `design/Architecture.md` (components, stack, Jira integration, data model).
- **The agent blueprint** — `design/Agent_Blueprint.md` (boundary, Defining Operational Constraint, control plane, failure modes, workflow).
- **Why LangGraph (a fixed-stack exception)** — `design/adr/0001-langgraph-orchestration.md`.
- **The plan** — `planning/Roadmap.md`: four phases (P0 → P3), each with a gate.
- **What's still undecided** — `planning/Open_Questions.md`.
- **The quality bar** — `evaluation/Eval_Rubric.md`, `evaluation/Acceptance_Criteria.md`, `evaluation/Success_Metrics.md`.
- **What's done and what's next** — `tracking/Status.md`.
- **The code** — `code/` (the monorepo; run it via `code/README.md`).

## How the tracking system works

`tracking/Status.md` is the dashboard — active phase, gate states, locked decisions, open threads. Open it at the start of every working session. Each file under `tracking/milestones/` is the working checklist for one phase: its deliverables, its gate, and a checkbox list. A phase is **not done until its gate passes** — gates, not checkbox counts, govern progress.

## Planning vs. execution split

`planning/Roadmap.md` is the durable plan — it changes only when strategy changes. `tracking/` is the volatile execution layer — it changes constantly. Keeping them separate keeps the plan stable while day-to-day state churns.

## Implementation (`code/`)

The implementation lives in [`code/`](code/) — a monorepo holding the Python/LangGraph orchestrator, the Go `jira-mcp` server, the Go `teams-adapter`, the Postgres schema, and infra/CI. It is the **P0 vertical slice**: the Daily Brief wired end-to-end through every layer, with Jira data stubbed so it runs with zero credentials. Run instructions are in `code/README.md`; live status is in `tracking/Status.md`.

This nests code inside the project home rather than a separate repo — docs and code in one place. If `code/` later needs independent versioning or deployment, it can be split into its own repo without disturbing the brain.

## Inspiration

Structure and conventions are modeled on `../clear-cortex` — its README index, the product / design / planning / evaluation / tracking split, the Status dashboard, and milestone files governed by gates. This project applies that working-home pattern to building the Scrum Master Agent.
