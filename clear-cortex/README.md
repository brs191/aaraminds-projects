# Credit Routing Service — Architecture Comprehension

**Owner:** Raja · **Stage:** P1 breadth complete — assistive gate PASS (85/100, 0 fabrications); 2nd human reviewer pending
**Subject repo:** `apm0045942-credit-routing-service` @ commit `e17fe410` (branch `develop`)
**Method:** Code Intelligence Factory (CIF), adapted — see `design/Method_Adaptation.md`

This is the working home for the architecture comprehension of the Credit Routing Service. It produces, by hand (Claude-assisted), the artifacts that should have existed — a **Code Briefing**, an **Inferred Product Spec**, and a whole-service **HLD** — each evidence-linked, with observed facts kept strictly separate from inferred ones.

> **The code repo stays clean.** `apm0045942-credit-routing-service` is treated as read-only. Every artifact, note, and tracking file lives here, never inside the repo.

## Folder map

```
clear-cortex/
├── README.md            ← you are here: project index
├── Exec_Summary.md      ⭐ exec one-pager — verdict + 4 decisions (start here for the bottom line)
├── instructions_plan.md execution blueprint — what to load & do each phase
├── Raja_Instructions.md saved P1 + gate prompts (Raja's working copy)
├── product/             the "why" — stable reference
│   └── Comprehension_Brief.md
├── design/              the "what & how" — method + diagrams
│   ├── Method_Adaptation.md
│   ├── architecture-component-view.svg      architecture / component view
│   └── credit-check-v2-runtime-flow.svg     credit-check v2 runtime flow
├── planning/            the "when and in what order"
│   ├── Roadmap.md
│   ├── Plan_Validation.md       Project-Planner audit of the plan (2026-05-30)
│   └── CIF_Bridge_Roadmap.md    how this links to the CIF product
├── evaluation/          the "how we know it's good" + the artifacts themselves
│   ├── HLD_Template.md          adapted from CIF
│   ├── Evaluation_Rubric.md     adapted from CIF
│   ├── Code_Briefing.md         the evidence layer — P1 breadth (§0–§9) + P2 deep-reads D1–D6 (§10–§15)
│   ├── Inferred_Product_Spec.md deliverable — capabilities/actors/value-flow
│   ├── HLD.md                   ⭐ the deliverable — final v1.0 (whole-service, 9 ranked risks)
│   ├── Scorecard.md             P3 score — 100/100, 20/20 anchors, 0 fabrications (assistive)
│   ├── P1_Gate_Review.md        P1 gate scorecard + verdict (assistive PASS)
│   ├── P2_Gate_Review.md        P2 gate — D1–D6 verdicts (all PASS, 0 fabrications)
│   └── Reviewer_Guide.md        brief + scorecard for the 2nd human reviewer
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
- **The P1 gate result** — `evaluation/P1_Gate_Review.md` (scorecard, verdict, corrections applied).
- **For the human reviewer** — `evaluation/Reviewer_Guide.md` (brief + scorecard to fill).
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
