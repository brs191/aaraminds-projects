# P2 — Deepen Highest-Value Areas

**Goal:** upgrade the priority areas from inferred to evidence-backed. **Effort:** ~1–2 days each.

## Priority (confirmed at the P1 gate — see `../../evaluation/Code_Briefing.md` §9)
1. [x] **DSL rules engine + two-evaluator duplication** — ✅ deepened (P2-D1, 2026-06-01): semantic divergence (`gt` = `>=` vs `>`), admin-only operators throw in routing, hot-path **not** cache-fronted, out-of-band audit. See `Code_Briefing.md` §10 + `HLD.md` §7/§10/§11.
2. [x] **Core credit-check runtime flow** — ✅ deepened (P2-D2, 2026-06-01): the dispatch table (7 routable of 13), the multi-product state machine + fail-fast parallel legs, the concrete §11(1) atomicity (8 failure modes), eventing gating, test gaps. See `Code_Briefing.md` §11 + `HLD.md` §7/§10/§11.
3. [x] **Domain & data model** — ✅ deepened (P2-D3, 2026-06-01): verified relationships (shared-PK + denormalized-FK, no DBRefs), the escalated indexing risk (`auto-index-creation` OFF → effectively `_id`-only + nightly full-scans), unbounded growth / no retention, 10 out-of-band audit collections. See `Code_Briefing.md` §12.
4. [x] **External integrations** — ✅ deepened (P2-D4, 2026-06-02): CSI SOAP (raw SAAJ, 9 ops = 4 primary routes + 5 sub-calls, plaintext-creds-in-header / no signing), CAS UCCS+CSRM (generated clients, models+Basic-auth only), Equifax UBCT/ICAAM (canary-gated, `@Async`+poll). **Both P1 §8 corrections re-verified** (`CreditApi` unused; ICAAM = Equifax OAuth, not Entra). New top risk: **no fault-tolerance layer** on the synchronous downstreams (no breaker/pool, 60 s timeout, retry only on CSI). See `Code_Briefing.md` §13 + `HLD.md` §8/§9/§10/§11(9).
5. [x] **Security model** — ✅ deepened (P2-D5, 2026-06-02): 3 inbound token regimes (Entra JWT · Entra-twin · eLogin opaque) + Kafka OAUTHBEARER; **authz categorically authentication-only** (RBAC computed at login, never enforced — even role-grant endpoints unguarded); both dead components confirmed (unwired caching introspector → per-request introspection; unwired entry point); `isJwtShaped` ignores the token (routes by env). NEW: unvalidated JWT audience, unauth `/v2/internal`+`/public/oidc`+`/sync` surface, unsigned Halo `JWT.decode`, **6 fully-plaintext prod-style secrets**. See `Code_Briefing.md` §14 + `HLD.md` §9/§10/§11(2)(3)(6).
6. [x] **admin/ analytics + audit** — ✅ deepened (P2-D6, 2026-06-02, catalogue-depth): stats scheduler = **11 `@Scheduled` jobs (not 12)**; the **recompute job** is the sharp risk (unindexed, sort-mismatched, 5×-fan-out scan of `creditCheckResult`); audit = 10 `*_audit` snapshot collections **written out-of-band** (`AuditService` read-only; 2 mechanisms — in-process stamps vs out-of-band snapshots); admin surface = **19 controllers / 62 endpoints (not ~66)**; no transactional-data audit. See `Code_Briefing.md` §15 + `HLD.md` §9.

## Per-area tasks (all areas D1–D6)
- [x] Extend `Code_Briefing.md` with deep-read facts (locators + provenance). — §10–§15
- [x] Upgrade the matching `HLD.md` section inferred → evidence-backed. — v0.8
- [x] Add step-level runtime detail for flows. — §11 (D2)
- [x] Complete §10 decision records (observed / evidence / likely rationale `inferred` / trade-off). — incl. fail-fast + sync-downstream + authn-only

## Gate (per area)
Rubric altitude + accuracy + evidence bars met; every inferred claim carries a confidence band.

✅ **Gate run 2026-06-02 — assistive PASS on all 6 areas (D1–D6), zero fabrications.** 6 fresh independent adversarial reviewers re-derived every load-bearing claim from `e17fe410`; 7 precision fixes applied (D2 publish-anchor, D5 grep-description + 7→6 secrets, D6 §7 16/12→15/11, D4 "67×", D3 TTL ladder). Scorecard: `../../evaluation/P2_Gate_Review.md`. Formal sign-off still needs the 2nd human reviewer + the P3 reviewer pass.
