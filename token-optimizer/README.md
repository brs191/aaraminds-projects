# AI Token Optimizer

**Owner:** Raja  ·  **Stage:** pre-build — measurement spike scoped, product gate-contingent

The AI Token Optimizer is a proposed local-first developer tool that compresses the context sent to AI coding assistants to cut token spend, with a measured guarantee that answer quality does not degrade. The prior-art scan found the concept already ships, so this is not a build — it is a decision. The initiative is sequenced **spike → gate → conditional build**: a 2–4 week measurement spike produces real savings numbers on AITO's own usage, a decision gate turns those numbers into a verdict, and only a Green verdict opens a product build.

This folder is the working home for the project — its thinking, its plan, and its live execution state.

## Folder map

```
token-optimizer/
├── README.md            ← you are here: project index
├── product/             the "why" — stable reference
│   ├── AI_Token_Optimizer_Product_Brief_2026-05-24.md
│   ├── AI_Token_Optimizer_Product_Brief_Infographic.html
│   ├── AI_Token_Optimizer_Executive_Deck.pptx   leadership decision deck
│   ├── PRD_M2.md         the production PRD for the M2 conditional build
│   └── AI_Token_Optimizer_Prior_Art_Landscape_2026-05-21.md
├── design/              the "what" — architecture
│   ├── AI_Token_Optimizer_Agent_Blueprint_v0.1.md
│   └── Product_Architecture.svg   production architecture diagram (M2 target)
├── planning/            the "when and in what order"
│   ├── Roadmap.md       the durable plan — milestones and gates
│   ├── Delivery_Plan.md the executable layer — breakdown, estimates, risks
│   └── AI_Token_Optimizer_Build_vs_Adopt_2026-05-21.md
├── evaluation/          the "how we know it's good"
│   ├── AI_Token_Optimizer_Systems_Review_2026-05-21.md
│   └── Project_Readiness_Evaluation.md   dual-lens launch readiness
├── tracking/            live execution state
│   ├── Status.md        the dashboard — start here each working session
│   └── milestones/
│       ├── M0-Spike.md
│       ├── M1-Decision-Gate.md
│       └── M2-Conditional-Build.md
└── spike/               the runnable measurement kit (Phase 1)
    ├── SPIKE_PLAN.md    objective, scope, metrics, timeline, decision gate
    ├── README.md        setup and run instructions
    ├── architecture.svg the kit's architecture diagram
    └── ...              Docker, the compression hook, the A/B harness, fixtures
```

## Where to find things

- **Why we're building this, for whom, and the core product decisions** — `product/AI_Token_Optimizer_Product_Brief_2026-05-24.md` (with a one-page infographic alongside it). The market context behind it — why this is hard — is in `product/AI_Token_Optimizer_Prior_Art_Landscape_2026-05-21.md`. The production PRD that specifies *what* the M2 build must do — numbered requirements, acceptance criteria, success metrics — is `product/PRD_M2.md`, gate-contingent on M1 Green.
- **How the system is shaped** — `design/AI_Token_Optimizer_Agent_Blueprint_v0.1.md`: the Module 8 blueprint and the locked architecture decisions (bundled sidecar, localhost loopback proxy, metadata-only egress). The production architecture diagram — the M2 target with the Module 5 fixes folded in — is in `design/Product_Architecture.svg`.
- **The plan** — `planning/Roadmap.md`: the three milestones (M0 spike → M1 gate → M2 conditional build), the kill/continue gates, and the component-to-milestone mapping. The executable delivery layer — work breakdown to a binary Definition of Done, effort estimates, the critical path, and the risk register — is in `planning/Delivery_Plan.md`. The reasoning behind the whole approach — why a spike rather than an outright build — is in `planning/AI_Token_Optimizer_Build_vs_Adopt_2026-05-21.md`.
- **How output quality is judged** — `evaluation/AI_Token_Optimizer_Systems_Review_2026-05-21.md`: the Module 5 conformance review of the blueprint, with ten findings and the four Required Fixes a build must fold in. The launch-readiness assessment through the AI Engineering Architect and Project Planner lenses is in `evaluation/Project_Readiness_Evaluation.md` — read this before kickoff.
- **What's done and what's next** — `tracking/Status.md` and the per-milestone files.
- **The runnable spike** — `spike/`: the LiteLLM + LLMLingua-2 measurement rig. Start with `spike/SPIKE_PLAN.md`.

## How the tracking system works

`tracking/Status.md` is the dashboard. It shows which milestone is active, the state of each gate, and overall progress. Open it at the start of every working session.

Each file under `tracking/milestones/` is the working checklist for one milestone — its deliverables, its gate, the owner, and a checkbox list of tasks. As work proceeds, tick items in the milestone file and reflect the milestone's roll-up state in `Status.md`.

The rule the plan sets: a milestone is not "done" until its **gate** passes. Tasks can all be ticked and the milestone can still be open if the gate has not been cleared. Gates, not task counts, govern progress.

## Planning vs. execution split

`planning/` is the durable plan — it changes only when the strategy changes. `tracking/` is the volatile execution layer — it changes constantly. Keeping them in separate folders keeps the plan stable while the day-to-day state churns.

## Current state

Product definition is complete: the brief, the prior-art scan, the build-vs-adopt analysis, and the systems review are all written. The Phase 1 spike kit is built and ready to run (`spike/`). Current work is **M0 — Spike**: calibrate the decision-gate thresholds to AITO's economics, then run the 2–4 week measurement. See `tracking/milestones/M0-Spike.md`.
