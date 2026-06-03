# P2 Gate Review — Deepen areas D1–D6

**Date:** 2026-06-02 · **Subject:** `apm0045942-credit-routing-service` @ `e17fe410` · **Deliverables under gate:** `Code_Briefing.md` §10–§15 (P2 deep-reads) + the matching `HLD.md` v0.8 upgrades.

**Method.** Six **fresh, independent** adversarial reviewers — one per deepened area, none of which authored the section it reviewed — each ran the per-area P2 gate prompt (`Raja_Instructions.md` → "P2 Gate Prompt") against the `e17fe410` source. Each re-derived every load-bearing claim from code (re-opening anchors, re-running greps), hunted for fabrications, checked that P1 carry-forwards were actually closed (not silently asserted), verified correction lineage, and checked local no-regression. The cross-document consistency sweep (correction lineage across *all* sections; global no-regression) was run in synthesis.

> **Status:** assistive single-pass gate — like the P1 gate, this is **not** the formal sign-off. The formal gate still needs the **second human reviewer** (rubric §6) and the **P3 reviewer pass**. The **SHA reconciliation** (Mac `44b6b86…` vs workspace `e17fe410`) also remains open.

## Verdict — PASS (all six areas), zero fabrications

| Area | Section | Verdict | Fabrications | P1 carry-forwards | Notes |
|---|---|---|---|---|---|
| **D1** DSL rules engine | §10 | **PASS** | none | all closed | `gt` `>=`-vs-`>` divergence, uncached hot-path, 4 admin-only operators, out-of-band `cCRule_audit` — all re-derived exact. The two P1 corrections (cache-fronted; new `gt` divergence) verified + properly attributed. |
| **D2** runtime flow | §11 | **PASS** (1 anchor fix) | none | all closed | 7-of-13 routable, `EIP` dead, fail-fast legs, free-form-String + unused enum, only-POST-`FAILED`, 0 `@Transactional`, write-counts — all confirmed. **Fixed:** the S7 "emitted inside `proceed()` (L545)" anchor (the multi-product aspect has no `@Around`/`proceed()`) → file-qualified to `MultiProductCreditCheckProcessorImpl#refreshMultiProductCreditCheck L545` → aspect `L373`. |
| **D3** data model | §12 | **PASS** | none | all closed | 3 declared indexes / `auto-index-creation` off / `_id`-only hot collections, shared-PK, 0 DBRefs, 10 audit collections, 1 TTL — all exact. The §11(5) escalation and the 10-not-11 audit count both verified code-true. |
| **D4** integrations | §13 | **PASS** (cosmetic) | none | both P1 §8 corrections re-verified | 9 SOAP ops (4+5), `@Retryable` 3×/2 s gated, no WS-signing, **`CreditApi` unused (0 refs)**, **ICAAM = Equifax OAuth (`api.uat.equifax.com`)**, no circuit-breaker/pool — all confirmed. **Fixed:** the "67×" framing (reworded to "67 grep lines / ~24 setter calls / 12 transformers"). Residual ±1 op-table start-line anchors are within method ranges — deferred to P3 polish. |
| **D5** security | §14 | **PASS** (2 precision fixes) | none | §11(2)/(3) now concrete | **Highest-stakes claim verified, NOT overstated:** authz is categorically authentication-only, RBAC never enforced (login-time DTO only), role-grant endpoints unguarded; `isJwtShaped` ignores the token; both dead components confirmed; JWT `aud` unvalidated; unsigned Halo `JWT.decode`. **Fixed:** (a) §14.3 grep description split into two greps (authz-construct pattern → 0 = no gates; `authorizeHttpRequests` → 2, both `.anyRequest().authenticated()`); (b) "7 plaintext secrets" → **6 fully-plaintext** (the eLogin secret is asterisk-masked even in main) + test-masking nuance — propagated to HLD §11(6), Status, P2-Deepen. |
| **D6** admin/audit | §15 | **PASS** (1 consistency fix) | none | all closed | **Cleanest area** — every count exact: 11 `@Scheduled` (not 12), recompute 5×-fan-out, `AuditService` read-only / 0 in-process audit writes, 10 audit collections, 19 controllers / 62 endpoints, no transactional-data audit. **Fixed:** the lingering breadth contradiction — `Code_Briefing.md` §7 still printed the old "16 methods / 12 stats" as current fact → corrected to "15 / 11" with a `⚠ corrected — see §15.1` marker. |

## Fixes applied at the gate (this pass)

1. **§11.3 S7 (D2)** — Kafka-before-persist anchor file-qualified; removed the inaccurate `proceed()` phrasing.
2. **§7 (D6)** — scheduler counts corrected 16/12 → 15/11 with a correction marker (removes the last breadth-vs-deep contradiction).
3. **§14.3 (D5)** — the authorization grep claim split into two greps (the construct-pattern returns 0, which is itself the proof; `authorizeHttpRequests` returns 2); added the adversarial cross-check line.
4. **§14.4 (D5)** — `new CachingOpaqueTokenIntrospector(` → "0 instantiations (lone match is javadoc)".
5. **§14.6 + §14.8 (D5)** — "7 plaintext" → "6 fully-plaintext credential values" + the eLogin-masked-in-main + test-masking nuance; propagated to HLD §11(6), Status.md, P2-Deepen.md.
6. **§13.2 (D4)** — "67×" reworded to be reproducible-and-honest about what it counts.
7. **§12.3 (D3)** — TTL "~3 yr" → "ladder DAILY=3 yr … YEARLY=7 yr (see §15.1)" for cross-section consistency.

## Residual (non-blocking, for P3 polish)

- D4 §13.1 op-table: ESOCC/ICCR/IUCCR `invokeBackendAPI` start anchors are L57 vs the method-body L58 (and IUCAD L53 vs L54); the cited end-line ranges already span the methods. Cosmetic.
- D1 §10.4: add the `UBCTService:215` line-anchor to the UBCT row for parity; note `getCCRuleByCCTypeAndAgreementType` has zero callers (its `@Cacheable` is fully dead, not merely routing-irrelevant).
- D3 §12.1/§12.2: two repo-path locators are slightly off (line numbers exact).

## Gate bars (milestone-aware, P2 depth)

All six areas met the per-area bars: **altitude** (component-to-line, no code-dump) · **accuracy** (zero fabrications; counts right after the fixes above) · **evidence** (every non-trivial claim anchored; inferences carry confidence bands). The full 6-dimension rubric re-score is deferred to **P3**.

## Open before formal sign-off

1. **Second human reviewer** (rubric §6) — still required.
2. **SHA reconciliation** — Mac `44b6b86…` vs workspace `e17fe410`.
3. **P3 reviewer pass** — the formal scored gate over the consolidated HLD.
