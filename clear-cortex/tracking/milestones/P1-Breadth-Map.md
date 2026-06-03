# P1 — Breadth Map (whole service, shallow)

**Goal:** a coherent shallow whole-service HLD + a ranked deepen list. **Effort:** ~2–4 days. **Status: produced 2026-06-01 — reviewer gate pending.**

## Deliverables
- `Code_Briefing.md` (breadth) — deterministic inventory across all packages. ✅ §2–§9.
- `Inferred_Product_Spec.md` (breadth) — capabilities, actors, value flow. ✅
- `HLD.md` (breadth) — §§1–11 at component altitude; §9 checklist filled; §10 seeded; §11 observations. ✅

## Tasks
- [x] Package-by-package roles; started with `routing/` (core) and `admin/rules` (DSL engine). (14 top-level packages.)
- [x] **Decoded `cas`, `ubct`, `iebus`**; confirmed `csi` = Credit Services Integration. (Plus CLEAR, CRSMS, UCCS, UBCT/ICAAM.)
- [x] REST surface: 27 controllers / **89 routable endpoints** (107 with v2 dual-mount aliases), v1 vs v2 — cross-checked `Credit.yaml` (covers only 6 of 89; drift flagged).
- [x] Mongo: **29 collections** (30 `@Document`) + 29 repositories; inferred relationships noted; 3-index risk flagged.
- [x] Integrations: CSI/SOAP, CAS (UCCS+CSRM), Equifax UBCT/ICAAM, IEBus/Kafka.
- [x] Cross-cutting (§9): security, error handling, audit, cache, ShedLock, MDC, **11 aspects** — checklist filled.
- [x] Seeded §10 decision records (Mongo/no-tx; two DSL evaluators; AOP-as-persistence; strategy/factory; IEBus wrapper; v1/v2).
- [x] `admin/` decision: **catalogued** (deep-read only `admin/rules`).
- [x] Ranked the P2 deepen list (`Code_Briefing.md` §9).

## Gate (reviewer)
**Assistive PASS — 85/100, zero fabrications** (6 adversarial verifiers + 1 scorer; see `evaluation/P1_Gate_Review.md`). The 7 accuracy corrections it raised are **applied + verified**. **Still required before P2:** a second human reviewer (rubric §6) and the `44b6b86…` ↔ `e17fe410` SHA reconciliation.
