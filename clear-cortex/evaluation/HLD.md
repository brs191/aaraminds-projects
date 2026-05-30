# High-Level Design — Credit Routing Service

**Status:** STUB — authored across P1 (breadth) → P2 (deepen) → P3 (finalize). Conforms to `HLD_Template.md`; scored by `Evaluation_Rubric.md`.

## 1 · Document control

| Field | Value |
|---|---|
| Subject system | Credit Routing Service (`com.att.creditcheck`) |
| Source repo | `apm0045942-credit-routing-service` |
| Pinned commit | `e17fe410` (branch `develop`, 2026-05-29) |
| Scope | Whole single-module service |
| Comprehension depth | _breadth (P1) → deepened areas (P2): TODO_ |
| Version / date | v0.1 / _TODO_ |

## 2 · Purpose & context
_TODO P1._

## 3 · Scope
_TODO P1 — what this HLD covers and, explicitly, what it does not (per the "not visible" rule)._

## 4 · Architecture overview
_TODO P1 — the shape in a few paragraphs. Diagram → `../design/`._

## 5 · Component view
_TODO P1 — routing (core), admin (incl. admin/rules DSL), csi (SOAP), iebus (eventing), cas (identity), policy, multiproduct, config/common/internal. Type-naming cap applies._

## 6 · Domain & data model
_TODO P1/P2 — the MongoDB collection model inferred from the 32 `@Document` classes; relationships by reference/embedding (marked `inferred`); lifecycle/state of the core credit-check result and routing-rule documents._

## 7 · Key runtime flows
_TODO P2 — credit-check v2: request → routing → admin/rules DSL eval → policy → result → IEBus event._

## 8 · External interfaces & integrations
_TODO P1 — REST v1/v2 under `/CreditCheck`; CSI/SOAP external credit services; IEBus/Kafka eventing; OIDC; ubct._

## 9 · Cross-cutting concerns

| Concern | Status | Evidence |
|---|---|---|
| Transactions | _TODO_ | |
| Validation | _TODO_ | |
| Security & authorization | _TODO_ | |
| Configuration | _TODO_ | |
| Error handling | _TODO_ | |
| Events & observability | _TODO_ | |
| Auditing | _TODO_ | |
| Concurrency | _TODO_ | |
| Multi-tenancy | _TODO_ | |
| Data-store portability | _TODO_ | |

## 10 · Key design decisions & patterns
_TODO P1 seed → P2 complete. One record per decision: Observed decision · Evidence · Likely rationale (`inferred`) · Trade-off._

## 11 · Observations (optional)
_TODO P3 — risks, coupling, tech debt; modernization notes seeded from `.github/appmod/appcat`._

---
_Evidence tables per section; provenance + confidence on every anchor. See `HLD_Template.md`._
