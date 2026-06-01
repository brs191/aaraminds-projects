# P1 Gate Review — Credit Routing Service HLD

**Date:** 2026-06-01 · **Subject:** `apm0045942-credit-routing-service` @ `e17fe410` (workspace clone the facts were read from) · **Reviewed:** `HLD.md` (breadth) + `Code_Briefing.md` (evidence layer)
**Method:** 6 independent adversarial code-verifiers (one per HLD slice, each checking claims against source) → 1 independent scorer applying `Evaluation_Rubric.md`. None authored the HLD.

## Verdict: PASS (assistive single-reviewer) — but soft, with 7 corrections required before sign-off

> **Update 2026-06-01:** all 7 corrections below have been **applied** to `HLD.md` + `Code_Briefing.md` and verified (no stale values remain; `HLD.md` → v0.3). A re-score would lift factual accuracy toward 4/4. A second human reviewer (rubric §6) is still required.

**Critical-error rule: NOT triggered — zero fabricated components / flows / integrations** (the trust-critical property). Six adversarial verifiers opened the code and found no invented component, data flow, or integration.

## Scorecard

| Dimension | Score | Weight | Contribution | Note |
|---|:--:|:--:|:--:|---|
| Factual accuracy | **3** / 4 | 30 | 22.5 | Zero fabrications; all major structural claims true. Held off 4 by a cluster of census off-by-ones + two false §8 claims + one internal inconsistency. **Binding constraint — one notch above fail.** |
| Completeness / coverage | 4 / 4 | 20 | 20.0 | Every major component, all 5 integrations, both flows, the collection model, and the full §9 checklist covered (milestone-aware for P1 breadth). |
| Architectural correctness | 3 / 4 | 20 | 15.0 | Sound decomposition; one wrong relationship (`CreditApi` "implemented by" the controller). |
| Altitude | 4 / 4 | 10 | 10.0 | Consistently component altitude; type-naming cap respected. |
| Clarity & usefulness | 4 / 4 | 10 | 10.0 | Genuinely usable for onboarding; leads with verdicts. |
| Evidence & traceability | 3 / 4 | 10 | 7.5 | Strong claim-cluster→Briefing model; a few anchors back wrong/unsupported claims or are loose. |
| **Weighted total** | | | **85.0 / 100** | Band: "Strong; minor review needed." |

**Gate (a) — the only gate in force (§5b no-graph bar N/A to a hand-written HLD):** total ≥ 70 ✅ (85) · factual accuracy ≥ 3 ✅ (3) · no zero dimension ✅ · zero fabrications ✅ → **PASS.**

**Honest caveat:** factual accuracy sits at the 3/4 floor-plus-one. A stricter second reviewer could reasonably score the count-errors + two false claims as **2/4 — which would FAIL** the gate. Treat this as "fix the 7, then it's a solid pass," not "done."

## Corrections required before this HLD is the source of truth (none gate-blocking)

1. **`CreditApi` relationship (HLD §8; Briefing §5 row 6) — FALSE.** The generated `CreditApi` server stub is *unused* (zero src refs); `CreditPolicyController` is hand-written over the generated *models* only. The one architectural-correctness error — correct the relationship and reframe integration #6.
2. **admin endpoints 66 → 62 (Briefing §4; HLD §8).** Self-contradicting: §8's area breakdown currently sums to 90 vs the 89 headline. Recount the 19 admin controllers and reconcile the breakdown to 89.
3. **"Entra underpins the ICAAM token" — FALSE.** ICAAM uses Equifax's own OAuth2. Entra→JWT and Entra→Kafka-OAUTHBEARER are correct; drop ICAAM from that sentence.
4. **Census off-by-ones:** `@Document` 29→**30**, collections 28→**29** (propagate to HLD §6), `@RestController` 28→**27** (grep substring artifact — `GlobalExceptionHandler` is `@RestControllerAdvice`), all-mounts 107→**104**, `@ExceptionHandler` 13→**14**, `@Configuration` 20→**21**.
5. **Delete the false rationale** "`KeyValueConfigAudit` backs two repos" (Briefing §3) — it backs one. Re-derive 30 `@Document` − 1 mapped superclass (`AuditableEntity`) = 29 collections.
6. **"18 ops / 25 ops"** (Briefing §8) mislabels per-file constant counts as operator counts — relabel or re-derive.
7. **Dual-mount prose** (HLD §8; Briefing §4): it's on **3** controllers, not single-product only.

**Nice-to-have:** name the opaque-token IdP (`oidc.stage.elogin.att.com`) in §8; drop/deep-read the unsupported `SaartSegmentService` programmatic-index claim; tighten a few loose anchor line-ranges; close the `44b6b86…` vs `e17fe410` SHA reconciliation.

## Deepen list — confirmed, no re-order

`Code_Briefing.md` §9 ranking stands (1 DSL engine + duplication → 2 runtime flow → 3 data model + atomicity → 4 integrations → 5 security → 6 admin/audit). Three of the must-fix defects are naturally repaired by deepening items #1 (the ops mislabel), #3 (the census/rationale), and #4 (the `CreditApi` relationship). The admin recount (#2) should be a standalone quick fix, not deferred to its low-priority depth pass.

## Status of the formal gate

Per rubric §6, the formal gate needs **two qualified human reviewers** scoring independently and reconciling differences > 1 point. This is **one assistive pass**. The gate is **not formally cleared** until a second human reviewer scores and the two reconcile.
