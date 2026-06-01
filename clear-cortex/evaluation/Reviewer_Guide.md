# P1 HLD — Reviewer Brief & Guidelines

**You are the second, independent reviewer** for the P1 breadth-comprehension gate of the Credit Routing Service HLD. The rubric (`Evaluation_Rubric.md` §6) requires **two qualified reviewers** to score the HLD independently and reconcile. A first *assistive* pass scored **85/100 — PASS** (zero fabrications; 7 accuracy defects since corrected). **Score independently — do not anchor to that pass.**

**Subject:** `apm0045942-credit-routing-service` @ `e17fe410` (the workspace clone the facts were read from; a `44b6b86…` Mac pin is pending reconciliation). **Milestone:** P1 = *breadth*, component altitude. **There is no golden HLD yet** → score on the rubric's anchored 0–4 scales against the actual code.

---

## What to open (and the role of each)

| Document | Your use |
|---|---|
| **`HLD.md`** | **The deliverable you score** — the only doc scored on the six dimensions. |
| **`Evaluation_Rubric.md`** | Your **scorecard** — dimensions, weights, the gate, the blank scorecard. |
| **`HLD_Template.md`** | The **conformance contract** — required sections, evidence format, the §9 "no silent omission" rule, milestone rules. Check the HLD against it. |
| **`Code_Briefing.md`** | The **evidence layer** — the HLD's claim-cluster references point here; per-claim `file › Type#member › L–L` anchors live here. Trace a sample. |
| **Source code** at `coderepos/clear/apm0045942-credit-routing-service` | **Ground truth** — spot-check that anchors and the decomposition hold against the real code. |
| `Inferred_Product_Spec.md` | **Context only** (not scored) — what the service does. |
| `P1_Gate_Review.md` | **Open only AFTER you have scored** — the first reviewer's scorecard, for reconciliation. |

---

## Ground rules

- **Score first, compare later.** Produce your own six scores before opening `P1_Gate_Review.md`, so you don't anchor to it.
- **Milestone-aware.** This is P1 breadth — do **not** penalize intentional P2/P3 depth deferral or `[not deep-read]` markers.
- **Critical-error rule.** A *single* fabricated component, data flow, or integration caps **Factual accuracy at 1** and **fails the document**, regardless of other scores. Trust is binary.
- **Not your own homework.** You must not be whoever authored or drove the HLD.
- **Anchored scales, not vibes.** No golden HLD exists; judge each dimension on the 0–4 scale below against the code.

---

## How to review (≈60–90 min)

1. **Read** `HLD.md` §§1–11 once, for altitude + clarity.
2. **Conformance pass** vs `HLD_Template.md`: every section present? §9 checklist has all **10** rows marked Covered / Not-visible / Out-of-scope (no blanks)? Claims carry anchors? Inferences marked with a confidence band?
3. **Spot-check ~10–12 anchors** spanning components, the runtime flow, the data model, integrations, and §9. For each: open `Code_Briefing.md` → the cited `file:line` → confirm the code says what's claimed. **Actively hunt for a fabricated component / flow / integration.**
4. **Sanity-check the headline counts** with a quick grep: ~89 endpoints, 29 collections, 11 aspects, 14 packages, 27 controllers.
5. **Score** the six dimensions.

---

## Per-dimension 0–4 scale (condensed)

- **Factual accuracy (×30):** 4 = zero errors · 3 = ≤2 trivial imprecisions, nothing a reviewer calls wrong · 2 = minor inaccuracies · 1 = one fabrication (critical-error) or several wrong · 0 = multiple false.
- **Completeness / coverage (×20):** 4 = every major element covered (milestone-aware) · 3 = nearly complete · 2 = main components covered, some gaps · 1 = a major component/concern missing · 0 = whole subsystems missing.
- **Architectural correctness (×20):** 4 = decomposition & dependencies match how a knowledgeable engineer would describe it · 3 = sound, minor imprecision · 2 = reasonable, some boundaries arguable · 1 = partly wrong · 0 = misleading.
- **Altitude (×10):** 4 = consistently HLD altitude · 3 = occasional brief drift · 2 = drifts too deep/shallow in places · 1 = frequently wrong · 0 = wrong throughout (code dump or hand-waving).
- **Clarity & usefulness (×10):** 4 = genuinely useful for onboarding · 3 = clear, minor rough edges · 2 = understandable but uneven · 1 = hard to follow · 0 = unusable.
- **Evidence & traceability (×10):** 4 = every non-trivial claim has a conformant anchor + inferences confidence-marked · 3 = most anchored, a few gaps · 2 = major claims traceable, routine ones unlinked · 1 = sparse/malformed · 0 = none.

---

## Scorecard (copy and fill)

```
Reviewer: ______________    Date: __________
Repo / commit: apm0045942-credit-routing-service / e17fe410

Dimension                       Score(0-4)   Weight   Contribution
1 Factual accuracy              ____         30       ____
2 Completeness / coverage       ____         20       ____
3 Architectural correctness     ____         20       ____
4 Altitude                      ____         10       ____
5 Clarity & usefulness          ____         10       ____
6 Evidence & traceability       ____         10       ____
WEIGHTED TOTAL                  ________ / 100        (contribution = (score/4) x weight)

Critical-error rule (any fabricated component/flow/integration)?   YES / NO
New gate-blocking issues found: ___________________________________

GATE (a) — the only gate (the §5b no-graph bar does NOT apply to a hand-written HLD):
   total >= 70  AND  factual accuracy >= 3  AND  no dimension = 0  AND  zero fabrications
   RESULT:   PASS / FAIL
```

---

## Verdict & reconciliation

1. Compute your weighted total and the gate result **on your own scores first.**
2. **Now** open `P1_Gate_Review.md`. For any dimension where you and the first reviewer differ by **more than 1 point**, discuss and reconcile — and **record both raw scores** (do not silently average).
3. The gate is **formally cleared** only when both reviewers' reconciled result is PASS. Note any new gate-blocking issue you found that the first pass missed.
4. Hand back: your filled scorecard, the reconciled result, and a one-line sign-off (or the specific fixes required to pass).
