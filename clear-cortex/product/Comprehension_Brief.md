# Comprehension Brief — Credit Routing Service

**Subject:** `apm0045942-credit-routing-service` @ `e17fe410` · **Owner:** Raja · **Status:** stable reference (the "why")

## The problem

The repo documents how to *build and run* the service (README, Copilot instructions) but has **no architecture document** — no component decomposition, no runtime-flow map, no integration inventory, no record of *why* the design is the way it is. Anyone picking the service up reverse-engineers it from 768 Java files across 14 subsystems, several with opaque names (`csi`, `cas`, `ubct`, `iebus`).

## What we're producing, and for whom

For engineers onboarding to or maintaining the service, and for architecture review — the map a senior engineer would want before touching the code.

| Artifact | Question it answers | Provenance |
|---|---|---|
| `Code_Briefing.md` | What is *verifiably* in the code? | deterministic facts |
| `Inferred_Product_Spec.md` | What does the service *do* as a product? | inferred, marked |
| `HLD.md` | How is the system shaped, and why? | the deliverable |

## What "good" means

- Conforms to `evaluation/HLD_Template.md`; scores **≥ 70 / 100** on `evaluation/Evaluation_Rubric.md`, with **factual accuracy ≥ 3 / 4** and **zero fabricated** components, data flows, or integrations (the critical-error rule).
- Every non-trivial claim carries an evidence anchor; every inference is phrased as inference and carries a confidence band.
- Deterministic facts and inferred judgements are never blurred — the single discipline the whole method rests on.

## Constraints

- The code repo is **read-only**. All work product lives in this project.
- One repo, one pinned commit (`e17fe410`). No multi-repo, no rewrite, no modernization execution — see `planning/Roadmap.md` §"What this is NOT".
