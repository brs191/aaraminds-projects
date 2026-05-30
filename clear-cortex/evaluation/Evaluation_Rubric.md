# Evaluation Rubric (adapted) — Credit Routing Service HLD

**Adapted from:** `aaraminds-delivery/product-research/Code Intelligence Factory/evaluation/Evaluation_Rubric.md` v0.3. This is the quality gate for `HLD.md`. Scoring instrument and thresholds are unchanged from CIF; only the applicability notes are repo-specific.

## Part B — HLD quality (six dimensions, 0–4 each, weighted to 100)

| # | Dimension | Weight | 4/4 means |
|---|---|---:|---|
| 1 | Factual accuracy | 30 | Every claim verifiable against the code; zero factual errors |
| 2 | Completeness / coverage | 20 | Every major component, integration, data flow, and the collection model covered |
| 3 | Architectural correctness | 20 | Decomposition & boundaries match how a knowledgeable engineer would describe it |
| 4 | Altitude | 10 | Consistently at HLD altitude — not a class dump, not hand-waving |
| 5 | Clarity & usefulness | 10 | Genuinely useful as an onboarding document |
| 6 | Evidence & traceability | 10 | Every non-trivial claim carries a conformant anchor; every inference marked with confidence |

Contribution = `(score ÷ 4) × weight`.

**Critical-error rule.** A *single* fabricated component, data flow, or integration caps Factual accuracy at **1** and **fails the document overall**, regardless of other scores. Trust is binary.

## The gate (absolute quality bar)

- Weighted total **≥ 70 / 100**, and
- Factual accuracy **≥ 3 / 4**, and
- No dimension scored **0**.

> **The CIF §5b "value-of-graph" bar (≥15-point margin over a no-graph baseline) does NOT apply here.** That gate compares graph-grounded vs. naive *generation* and is only meaningful for the automated CIF pipeline. This is a hand-written HLD, so only the absolute quality bar (a) applies. If this work later feeds the automated pipeline, re-introduce §5b.

## Part A — knowledge graph (only if the automated pipeline is later pursued)

Not scored for a manual pass. If/when an extractor is built against this repo, score precision/recall vs. an independent Java parser: code layer ≥ 0.98/0.98, endpoints ≥ 0.95/0.98, structural edges ≥ 0.95/0.97, design layer (inferred) ≥ 0.80/0.85. Repo note: Mongo collections take the place of the Postgres table layer; MapStruct/SOAP generated members must be in the parsed set (build first).

## Scoring method

Part B is scored by a qualified human reviewer (a senior engineer) against the anchored scales — not the person who wrote the HLD. For a formal sign-off, two reviewers score independently and reconcile differences > 1 point.

## Scorecard (copy per scored run)

```
Credit Routing Service — HLD Evaluation Scorecard
Subject repo / commit:    apm0045942-credit-routing-service / e17fe410
Run / iteration:          ____________________
Date:                     ____________________
Scorer(s):                ____________________

PART B — HLD quality
  Dimension                    Score(0-4)  Weight  Contribution
  1 Factual accuracy           ____        30      ____
  2 Completeness / coverage    ____        20      ____
  3 Architectural correctness  ____        20      ____
  4 Altitude                   ____        10      ____
  5 Clarity & usefulness       ____        10      ____
  6 Evidence & traceability    ____        10      ____
  WEIGHTED TOTAL               ________ / 100

Critical-error rule triggered?     YES / NO

GATE
  total ≥ 70, accuracy ≥ 3, no zero dimension   PASS / FAIL
```
