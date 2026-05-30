# P2 — Deepen Highest-Value Areas

**Goal:** upgrade the priority areas from inferred to evidence-backed. **Effort:** ~1–2 days each.

## Priority (adjust at the P1 gate)
1. [ ] Core credit-check runtime flow (v2): request → `routing` → `admin/rules` DSL eval → `policy` → result → IEBus event.
2. [ ] DSL rules engine (`admin/rules`): how rules are defined, stored (Mongo), evaluated.
3. [ ] Domain & data model (32 Mongo collections): inferred relationships.
4. [ ] External integrations: CSI/SOAP, IEBus/Kafka, OIDC, `ubct`.
5. [ ] v1 vs v2 + multi-product divergence; `admin/` surface per the P1 decision.

## Per-area tasks
- [ ] Extend `Code_Briefing.md` with deep-read facts (locators + provenance).
- [ ] Upgrade the matching `HLD.md` section inferred → evidence-backed.
- [ ] Add step-level runtime detail for flows.
- [ ] Complete §10 decision records (observed / evidence / likely rationale `inferred` / trade-off).

## Gate (per area)
Rubric altitude + accuracy + evidence bars met; every inferred claim carries a confidence band.
