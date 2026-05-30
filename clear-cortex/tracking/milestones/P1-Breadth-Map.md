# P1 — Breadth Map (whole service, shallow)

**Goal:** a coherent shallow whole-service HLD + a ranked deepen list. **Effort:** ~2–4 days.

## Deliverables
- `Code_Briefing.md` (breadth) — deterministic inventory across all 14 packages.
- `Inferred_Product_Spec.md` (breadth) — capabilities, actors, value flow.
- `HLD.md` (breadth) — §§1–11 at component altitude; §9 checklist filled; §10 seeded.

## Tasks
- [ ] Package-by-package roles; start with `routing/` (core) and `admin/rules` (DSL engine).
- [ ] **Decode `cas`, `ubct`, `iebus`**; confirm `csi` = Credit Services Integration.
- [ ] REST surface: 28 controllers / ~107 endpoints, v1 vs v2 — cross-check `Credit.yaml`.
- [ ] Mongo: 32 `@Document` collections + 29 repositories; note inferred relationships.
- [ ] Integrations: CSI/SOAP, IEBus/Kafka, OIDC, `ubct`.
- [ ] Cross-cutting (§9): security, error handling, audit, cache, ShedLock, MDC, **aspects**.
- [ ] Seed §10 decision records (Mongo vs relational; IEBus wrapper; DSL engine; v1/v2; AOP).
- [ ] Decide: deep-read vs. catalogue the `admin/` surface.
- [ ] Rank the P2 deepen list.

## Gate
Completeness (every major component + integration named) · architectural correctness · **zero fabrications** · evidence anchors on non-trivial claims · altitude held.
