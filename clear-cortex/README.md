# Credit Routing Service — Architecture Comprehension

**Owner:** Raja · **Stage:** plan complete, entering P0 (Foundations)
**Subject repo:** `apm0045942-credit-routing-service` @ commit `e17fe410` (branch `develop`)
**Method:** Code Intelligence Factory (CIF), adapted — see `design/Method_Adaptation.md`

This is the working home for the architecture comprehension of the Credit Routing Service. It produces, by hand (Claude-assisted), the artifacts that should have existed — a **Code Briefing**, an **Inferred Product Spec**, and a whole-service **HLD** — each evidence-linked, with observed facts kept strictly separate from inferred ones.

> **The code repo stays clean.** `apm0045942-credit-routing-service` is treated as read-only. Every artifact, note, and tracking file lives here, never inside the repo.

## Folder map

```
clear-cortex/
├── README.md            ← you are here: project index
├── instructions_plan.md execution blueprint — what to load & do each phase
├── product/             the "why" — stable reference
│   └── Comprehension_Brief.md
├── design/              the "what & how" — method adaptation (+ diagrams later)
│   └── Method_Adaptation.md
├── planning/            the "when and in what order"
│   ├── Roadmap.md
│   ├── Plan_Validation.md       Project-Planner audit of the plan (2026-05-30)
│   └── CIF_Bridge_Roadmap.md    how this links to the CIF product
├── evaluation/          the "how we know it's good" + the artifacts themselves
│   ├── HLD_Template.md          adapted from CIF
│   ├── Evaluation_Rubric.md     adapted from CIF
│   ├── Code_Briefing.md         deliverable (stub → P1/P2)
│   ├── Inferred_Product_Spec.md deliverable (stub → P1)
│   └── HLD.md                   the deliverable (stub → P1–P3)
└── tracking/            live execution state
    ├── Status.md        the dashboard — open this each session
    └── milestones/
        ├── P0-Foundations.md
        ├── P1-Breadth-Map.md
        ├── P2-Deepen.md
        └── P3-Finalize.md
```

## Where to find things

- **What to load and do in each phase (so you don't get sidetracked)** — `instructions_plan.md`. Open it first.
- **Why we're doing this, for whom** — `product/Comprehension_Brief.md`.
- **How the CIF method maps onto a MongoDB / Kafka / SOAP / AOP service** — `design/Method_Adaptation.md` (includes the grounded snapshot of the repo).
- **The plan** — `planning/Roadmap.md`: four phases (P0 → P3), each with a gate.
- **Is the plan sound?** — `planning/Plan_Validation.md`: a Project-Planner audit with the open fixes.
- **The quality bar** — `evaluation/Evaluation_Rubric.md` and `evaluation/HLD_Template.md`.
- **What's done and what's next** — `tracking/Status.md`.
- **How this links to the CIF product** — `planning/CIF_Bridge_Roadmap.md` (clear-cortex = the manual proof; `../code-intelligence-factory` = the automated platform).

## How the tracking system works

`tracking/Status.md` is the dashboard — active phase, gate states, open threads. Open it at the start of every working session. Each file under `tracking/milestones/` is the working checklist for one phase: its deliverables, its gate, and a checkbox list. A phase is **not done until its gate passes** — gates, not checkbox counts, govern progress.

## Planning vs. execution split

`planning/Roadmap.md` is the durable plan — it changes only when strategy changes. `tracking/` is the volatile execution layer — it changes constantly. Keeping them separate keeps the plan stable while day-to-day state churns.

## Relationship to Code Intelligence Factory

clear-cortex is the **manual, single-repo proof** of the CIF method; the **automated platform** lives next door in `../code-intelligence-factory`. The credit-routing service is CIF's natural **pilot repo**, and this project's **P3 HLD becomes CIF's first golden-HLD fixture** — the ground truth that validates CIF's Reverse-Engineering and Business-Analyst agents (Phase 1). Full handoff and sequencing: `planning/CIF_Bridge_Roadmap.md`.

## Inspiration

Structure and method are modeled on `aaraminds-delivery/product-research/Code Intelligence Factory` — its README's product / design / planning / evaluation / tracking split, its `HLD_Template.md`, and its `Evaluation_Rubric.md`. This project applies that method to one real service; it does not rebuild the CIF tooling.
