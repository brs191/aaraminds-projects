# Inferred Product Spec — Credit Routing Service

**Status:** P1 breadth complete (inferred layer). · **Nature:** *inferred* — what the service does as a product, read off the code. Every claim is a hypothesis carrying a confidence band; kept strictly separate from `Code_Briefing.md` (deterministic facts).
**Provenance:** P1 inventory extracted from the workspace clone at `e17fe410`; the recorded pin `44b6b86…` (Raja's macOS) is pending reconciliation — see `HLD.md` §1.

## 1 · What the product does

The Credit Routing Service — internal name **CLEAR**, "Credit Logging and Evaluation Assistance Repository" (from the OpenAPI title, `Credit.yaml › info.title`) — is the **decision-and-routing brain for AT&T business credit checks** `[inferred: high]`. A caller submits a credit-check request; the service maps it to rule parameters, applies **finance-owned, configurable DSL rules** to decide *which* downstream credit engine and policy apply, dispatches to that backend (Equifax via the CSI SOAP ESB, the CAS REST gateway, or direct Equifax UBCT/ICAAM — or short-circuits to internal auto-approve / pre-approval / govt-exempt paths), persists the decision to MongoDB, and publishes it to a Kafka topic for the rest of the order lifecycle. Its name encodes a dual role — it both *evaluates/routes* and *logs/persists* every decision. `[inferred: high]` (`README.md › L5-7`; `routing/v2/RoutingService`; `iebus/publisher/PublishMessageServiceImpl`.)

## 2 · Capabilities

- **Credit routing / backend orchestration (core)** `[inferred: high]` — maps a request to rule params, evaluates DSL rules → a `CCTarget{system, api}`, and dispatches to the chosen credit engine. The reason the product exists. (`routing/v2/RoutingService#determineRoutingBackendSystem`.)
- **Single-product credit check** `[inferred: high]` — wireline + wireless, v2 (`routing/v2/singleproduct`) and legacy v1 (`routing/v1`).
- **Multi-product credit check** `[inferred: high]` — simultaneous mobility + wireline under one `creditProcessNumber`, evaluated in parallel (`routing/v2/multiproduct`). The headline v2 value-add over v1.
- **DSL-driven rule evaluation** `[inferred: high]` — a custom JSON `when`/`then` rule model with a rich operator set (`eq, gt, lt, bt, in, not in, bef, aft, contains one of …`), CRUD-managed by finance and cache-fronted; lets routing logic change *without a code deploy* (`admin/rules`, `routing/services/CCRuleExecutionService`).
- **Policy enforcement / management** `[inferred: high]` — small-business risk-mitigation policy via the CRSMS backend; product↔policy mapping admin (`policy/`, `admin/productpolicy`).
- **Identity verification** `[inferred: high]` — challenge-question fetch + answer validation (IUCVQ/VUCVA) via the CAS UCCS gateway (`cas/iucvq`).
- **Admin / reference-data management** `[inferred: high]` — a large finance-owned admin surface: rules, key-value config, product mappings, product-policy mappings, customer-account-type mappings, EIP limits, pre-approvals, BCES fallout, SAART segments, UBCT request-type config (`admin/*`, **19 controllers / 62 endpoints** — verified P2-D6).
- **RBAC + auth** `[inferred: high]` — roles, role assignments, OIDC token exchange; OAuth2 resource server (Entra/Halo JWT + opaque introspection) (`admin/roles`, `admin/roleassignment`, `admin/oidc`).
- **Analytics & monitoring** `[inferred: high]` — pre-aggregated stat snapshots with scheduled recompute, dashboard views (summary/dimension/trend/status), result search/export, and result-log tracing (`admin/analytics`, `admin/creditcheckresult`, `admin/ccresultmonitoring`).
- **Asynchronous event publication** `[inferred: high]` — completed credit decisions published to Kafka topic `com.att.clear.dev.creditcheckupdates`; a second consumer ingests SAART segment reference data (`iebus/publisher`, `admin/saartsegment/SaartSegmentConsumer`).
- **Pre-approval & EIP-limit gating** `[inferred: medium]` — `PreApprovalService` is injected into `RoutingService`, so pre-approval participates in the live routing decision, not just admin CRUD; EIP limits feed installment-plan eligibility.
- **Scheduled retry & UBCT batch processing** `[inferred: medium]` — ShedLock-coordinated retry of failed checks and asynchronous UBCT poll/submit (`csi/esocc` retry scheduler, `ubct/scheduler`).

## 3 · Actors & callers

- **v2 front-end (CLEAR UI / order capture)** `[inferred: high]` — primary v2 caller; CORS is locked to `https://clear.local.att.com:3000`, pointing to a CLEAR web front-end for sales/wireline + mobility order flows (`/v2/credit/credit-checks/**`).
- **v1 legacy front-ends** `[inferred: high]` — `/v1/public/api/credit-check` (README labels v1 "legacy"; separate Swagger group).
- **Internal service-to-service callers** `[inferred: high]` — `/v2/internal/credit-check/**` and the `/i1/**` internal tier (config onboarding, product import, role create, cache evict). The `getByOpptyIdAndRequestType` endpoint implies an upstream opportunity/quoting system.
- **Credit-finance admin users** `[inferred: high]` — own the entire admin surface (rules, reference data, analytics); README attributes rule ownership to "the credit finance team." Gated by RBAC.
- **Ops / SRE** `[inferred: medium]` — actuator health/metrics, ping endpoints, cache-evict.
- **Downstream event consumers** `[inferred: high]` — systems subscribing to `…creditcheckupdates` (order management, fulfillment, analytics) act on decisions asynchronously.
- **External credit engines (callees, not callers)** `[inferred: high]` — CSI SOAP ESB → Equifax; CAS REST (UCCS + CSRM); Equifax direct (UBCT/ICAAM). These are the routing *targets*.

## 4 · Value flow

```
[Front-end / internal caller]
   │ POST /CreditCheck/v2/credit/credit-checks(/multi-product)
   ▼
[1 INTAKE + AUTH]  OAuth2 JWT (Entra/Halo) or opaque-token introspection
   │ request → CCRequestRule (segment, sub-segment, enterprise type, affiliate, agreement…)
   ▼
[2 ROUTING DECISION]  RoutingService → fetch DSL rules (Mongo, cached) → CCRuleExecutionService.executeRule
   │ first when→then match  →  CCTarget{system, api}        ◄── primary value creation: the routing verdict
   ▼
[3 RULES + POLICY + REFERENCE DATA]  pre-approval, EIP limit, product↔policy, customer-account-type,
   │ SAART segment, BCES fallout, identity verification (CAS IUCVQ)
   ▼
[4 BACKEND CREDIT EVALUATION]  dispatch to chosen engine:
   │ CSI ECC/EUCC/ESOCC/USOCC (SOAP) · CAS UCCS/CSRM (REST) · Equifax UBCT/ICAAM
   ▼ credit decision (approve / decline / refer / debt / identity-challenge)
[5 RESULT]  CreditCheckResponse → persisted to MongoDB (powers monitoring/export/analytics) → returned sync
   ▼
[6 EVENT]  PublishMessageService → Kafka topic com.att.clear.dev.creditcheckupdates → downstream consumers
```

**Where business value is created** `[inferred: high]`: (1) **the routing verdict** (step 2) — finance-owned rules become a real-time decision of which engine + policy apply, with no code deploy; (2) **unified multi-product evaluation** (steps 3–4) — collapsing mobility + wireline into one orchestrated transaction; (3) **the persisted result + emitted event** (steps 5–6) — the decision is captured for audit/analytics and propagated so the order lifecycle can act on it.

## 5 · Confidence & open questions

- **Well-supported `[high]`:** the core routing/decision purpose, the capability set, the v2 vs v1 split, the external-engine targets, the event-publish path — all corroborated by code + README + config.
- **Weaker `[medium]`:** the exact decision-time use of pre-approval/EIP gating (inferred from wiring, not traced end-to-end); the precise actor behind the internal `getByOpptyIdAndRequestType` endpoint.
- **Acronyms still partly inferred:** `UBCT` (≈ Unified Business Credit Transaction — Equifax `/attgbs/` business API; not expanded in code), `ICAAM` ("UBCT2.0" per `ModelConstant`, letters unexpanded), `CAS` (gateway host; letters unexpanded). `CSI`, `IEBUS`, `CRSMS/CSRM`, `UCCS` are high-confidence/confirmed.
- **Needs an SME to confirm:** whether the published event is consumed by order-management/fulfillment as inferred; the business meaning of `BCES` and `SAART`; whether v1 is actively used or being sunset.
