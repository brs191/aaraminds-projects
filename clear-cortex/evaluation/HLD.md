# High-Level Design — Credit Routing Service

**Status:** P1 breadth authored — §§2–11 below at component altitude (gate pending). P2 deepens the ranked areas (`Code_Briefing.md` §9); P3 finalizes. Conforms to `HLD_Template.md`; scored by `Evaluation_Rubric.md`. Full per-claim evidence anchors live in `Code_Briefing.md`; this document uses claim-cluster references to those sections.

## 1 · Document control

| Field | Value |
|---|---|
| Subject system | Credit Routing Service (`com.att.creditcheck`) — internal name **CLEAR** |
| Source repo | `apm0045942-credit-routing-service` |
| Working copy | `/Users/rb692q/projects/aaraminds-projects-main/coderepos/clear/apm0045942-credit-routing-service` |
| Pinned commit | `44b6b8659a33ad7ac2227b6d88696d1946b9ce1a` — confirmed by Raja on his macOS P0-validation system (`.git/refs/heads/develop`, 2026-06-01); see SHA provenance note below. **P1 facts extracted from the workspace clone at `e17fe410`** — pending reconciliation |
| Scope | Whole single-module service |
| Comprehension depth | P1 breadth (whole-service, component altitude); body §§2–11 authored; P2 deepens |
| Version / date | v0.3 / 2026-06-01 (post-gate corrections applied) |

> **SHA provenance note.** The pinned commit above is the value Raja confirmed on his macOS system, where the P0 build and validation actually ran. It is recorded as `human_confirmed` (Raja, macOS), not workspace-verified: the clone accessible in *this* workspace — `coderepos/clear/apm0045942-credit-routing-service` — is at `e17fe410e79e8784d67c9e4dc505f210500e7cf1` (the AT&T `origin/develop` tip seen here), and `44b6b86…` is not present in that clone's objects, packed-refs, or reflog. **Before P1, reconcile the two copies to a single revision** (or state which one the comprehension is authored against) so the artifacts and the code stay reproducible together. *The P1 inventory was authored against the `e17fe410` clone.*

## 2 · Purpose & context

The Credit Routing Service ("CLEAR" — Credit Logging and Evaluation Assistance Repository) is AT&T's **decision-and-routing service for business credit checks**. It accepts credit-check requests, applies finance-owned configurable rules to decide which downstream credit engine and policy apply, dispatches to that backend, persists the decision, and publishes it as an event for the order lifecycle. Primary callers are a CLEAR web front-end (v2) plus legacy v1 front-ends, internal services, and credit-finance admins; the routing *targets* are Equifax (via the CSI SOAP ESB and direct UBCT/ICAAM) and the CAS REST gateway (UCCS + CSRM). *Evidence: `Code_Briefing.md` §1; `Credit.yaml` title; `README.md` L5-7.*

## 3 · Scope

Whole single-module Spring Boot service, at component altitude. **Covered:** the routing/decision core, the full REST surface (v1 + v2 + admin + internal), the MongoDB data model, the five external integrations, and the cross-cutting mechanisms (§§4–10). **Not visible in scoped code:** the metrics/log collection backend (OTel exporter / Grafana / Prometheus / Sentry are not wired in code — assume agent/infra-side); the internals of the external credit engines (CSI / CAS / Equifax). **Out of scope this pass:** deep per-subsystem internals — deferred to P2 per the ranked deepen list (`Code_Briefing.md` §9); and the runtime/production-data layer, deferred by method.

## 4 · Architecture overview

The service is a single Spring Boot 3.3.9 / Java 17 application, layered **controller → processor → routing → strategy → backend client**, over MongoDB (persistence), Kafka (eventing), and SOAP/REST egress to credit bureaus. Its defining shape is a **rules-driven router**: an inbound request is reduced to a parameter map, a DSL rule set stored in MongoDB (and cache-fronted) is evaluated to a `CCTarget{system, api}`, and a strategy + factory pair dispatches to the named backend (CSI / CAS / Equifax, or an internal auto-approve / pre-approval / govt-exempt path). A request-scoped object, `CreditCheckRequestScope`, threads state through the call as a shared blackboard, and a thick layer of **AOP aspects** owns persistence, audit, and status transitions *around* the service methods rather than inside them. v2 extends v1 with multi-product evaluation (mobility + wireline in parallel) and identity verification. A diagram will be added to `../design/` in P3. *Evidence: `Code_Briefing.md` §2, §6, §8.*

## 5 · Component view

| Component | Responsibility | Key types (boundary/flow anchors) | Depends on |
|---|---|---|---|
| **Routing core** (`routing/`) | Orchestrate a credit check: enrich → route → validate → dispatch → post-process | `RoutingService`, `CreditCheckProcessor`, `CreditCheckRequestScope`, `CreditCheckServiceFactory`, `CreditCheckService` strategies | DSL engine, admin/rules, integration clients |
| **DSL rule engine** (`routing/services`, `admin/rules`) | Evaluate finance-owned `when/then` rules → `CCTarget`; CRUD + store the rules | `CCRuleExecutionService` (live), `RuleExecutionService` (admin/UBCT), `CCRule`, `Rule<CCTarget>` | MongoDB (`cCRule`), cache |
| **Admin surface** (`admin/`) | Reference-data + rules CRUD, RBAC, OIDC, analytics, monitoring/export | 19 controllers; `CreditAdminController`, `CreditCheckStatScheduler` | MongoDB, the result store |
| **CSI integration** (`csi/`) | Outbound SOAP to Equifax via the CSI ESB (9 operations) | `SoapCallService`, `CSIClient` | Equifax (SOAP) |
| **CAS integration** (`cas/`, `policy/`) | Outbound REST to the CAS gateway — UCCS (identity/credit) + CSRM (policy) | `IdentityVerificationServiceImpl`, `PolicyServiceImpl` | CAS gateway (REST) |
| **UBCT integration** (`ubct/`) | Outbound REST to Equifax business API (UBCT + ICAAM); poll scheduler | `UBCTAPIService`, `ICAAMAPIService`, `TokenProvider` | Equifax (REST) |
| **Eventing** (`iebus/`) | Publish credit decisions to Kafka; consume SAART segments | `MessageBrokerClient`, `PublishMessageServiceImpl` | Kafka/Confluent |
| **Cross-cutting** (`config/`, `common/`, `exceptions/`, `logging/`, `internal/`) | Security, config, error handling, MDC/tracing, the AOP audit substrate | `SecurityConfiguration`, `GlobalExceptionHandler`, `AspectHelperClass`, 11 aspects | Entra ID, MongoDB |

*Evidence: `Code_Briefing.md` §2 (full package table + anchors).*

## 6 · Domain & data model

The model is **29 MongoDB collections** (from 30 `@Document` classes, one of which — `AuditableEntity` — is a mapped superclass with no collection of its own), with **no relational store and no JPA associations** — relationships are by shared key or embedding only. The spine is the **credit-check result**: `creditCheckResult` (`@Id creditCheckProcessId`) is 1:1 with `creditCheckDetails` (same PK) and rolls up N:1 into `multiProductCreditCheckResult` (`@Id creditProcessNumber`); `creditCheckParameter` and `ubctTransactions` reference the same keys for monitoring and bureau-transaction tracking. The **routing rule** (`cCRule`, `@Indexed creditCheckType`) embeds the `Rule<CCTarget>` `when/then` definition and has an `ACTIVE/INACTIVE/DELETED` lifecycle (soft delete). Around these sit reference-data collections (config, product/policy/account-type/EIP/pre-approval/BCES/SAART mappings), RBAC (`roles`, `userroleassignments` — referenced by role *name*), one analytics snapshot collection, and ten generic `*_audit` collections that embed full document snapshots. All cross-document links are marked `inferred`. **Risk:** only three annotation-driven indexes exist service-wide, while the hottest collection (`creditCheckResult`) is queried by dynamic/regex fields with no declared secondary index. *Evidence: `Code_Briefing.md` §3.*

## 7 · Key runtime flows

**v2 single-product credit check (component altitude):** the controller seeds `CreditCheckRequestScope`; a processor aspect hydrates and persists a draft `CreditCheckResult` and buffers the payload; `CreditCheckProcessor` runs the enricher chain, then `RoutingService` evaluates the DSL rules → `CCTarget`; a route-specific `CustomValidator` runs; `CreditCheckServiceFactory` resolves the concrete `CreditCheckService`, which calls the chosen backend (CSI/CAS/Equifax) and maps the result back onto the scope; a closing aspect persists the result + details + the monitoring trail and stamps the `creditProcessNumber` on the response. **Multi-product** splits the request into mobility + wireline legs, runs both single-product flows in parallel on a thread pool, and aggregates. **Eventing** is *not* on the single-product happy path — a credit decision is published to Kafka only on specific refresh/UBCT paths. Step-level detail (and the aspect-driven state machine) is the top P2 deepen item. *Evidence: `Code_Briefing.md` §6, §8; routing subagent inventory.*

## 8 · External interfaces & integrations

**Inbound:** 89 routable REST endpoints (104 counting the v2 dual-mount aliases, which are live on 3 controllers) under `/CreditCheck` — v1 (3), v2 single-product (4), v2 multi-product (8), v2 policy (3), v2 internal (2), admin (62) + ubct config (3), internal/ops ping + cache (4) — the breakdown sums to 89. The published `Credit.yaml` contract covers only **6 of 89** (the v2 single-product happy path + cancel + ping) and **omits the entire multi-product + policy surface** its own SpringDoc groups expose, and lacks a server/base-path — a documentation-drift finding; treat the SpringDoc runtime groups as authoritative. **Outbound:** CSI SOAP → Equifax (9 ops); CAS REST → UCCS (identity/credit) + CSRM (policy); Equifax direct → UBCT + ICAAM (OAuth2); Kafka publish → `…creditcheckupdates` and consume → SAART segments. **Auth dependency:** Microsoft Entra ID underpins the resource-server JWT and the Kafka OAUTHBEARER token; the ICAAM token is issued by Equifax's own OAuth2 (not Entra), and opaque-token introspection uses the AT&T eLogin IdP (`oidc.stage.elogin.att.com`). The policy/order-update endpoints are served by a **hand-written** `CreditPolicyController` over the generated `creditPolicy` models — the generated `CreditApi` server interface itself is **unused**. *Evidence: `Code_Briefing.md` §4, §5.*

## 9 · Cross-cutting concerns

| Concern | Status | Evidence (see `Code_Briefing.md` §6–§7) |
|---|---|---|
| Transactions | **Not visible (effectively absent)** | No `@Transactional` / `MongoTransactionManager` anywhere; multi-document writes (often inside AOP advice) are **non-atomic**. **High-severity gap.** |
| Validation | **Covered** | Bean Validation (`@Valid`) + custom validators; `GlobalExceptionHandler` maps to structured 400s. |
| Security & authorization | **Partially covered** | AuthN covered (OAuth2 resource server: JWT Entra/Halo + opaque introspection). **AuthZ is `authenticated()`-only — no roles/scopes/method security.** Three dead/buggy components (unwired token cache + entry point; `isJwtShaped` routes by env, not token). |
| Configuration | **Covered** | Externalized `application.yml` + 6 `@ConfigurationProperties` + `@ConditionalOnProperty` gates. **Flag:** secrets are plaintext defaults in yml. |
| Error handling | **Covered** | `GlobalExceptionHandler` (`@RestControllerAdvice`, 14 handlers), uniform `ErrorResponse`. |
| Events & observability | **Partially covered** | Micrometer tracing (W3C `traceparent`) + MDC + the aspect audit trail are **in-code**; actuator health/metrics exposed. **No Sentry / OTel exporter / Grafana / Prometheus wiring in scoped code** — assume infra/agent-side. |
| Auditing | **Covered** | `@EnableMongoAuditing` (created/updated stamps) + the aspect payload trail + 10 `*_audit` snapshot collections. |
| Concurrency | **Covered** | Tuned thread pools with MDC/trace propagation; `@Async`; **ShedLock (MongoLockProvider)** guards scheduled jobs across instances. |
| Multi-tenancy | **Not visible** | No tenant resolver/isolation; `sourceSystem`/environment are labels, not isolation boundaries. |
| Data-store portability | **Not portable** | Direct `MongoTemplate` usage, Mongo-specific ShedLock + auditing, Atlas SRV URI — no portability seam. |

## 10 · Key design decisions & patterns

- **MongoDB document store, no transactions.** *Observed:* `@Document` model, zero `@Transactional`. *Evidence:* `Code_Briefing.md` §3, §6. *Likely rationale (`inferred`):* schema flexibility + horizontal scale for a high-volume decision/audit store. *Trade-off:* no atomicity across multi-write result+details+monitoring sequences.
- **DSL rule engine, externalized & cached — but duplicated.** *Observed:* a `when/then` `Rule<CCTarget>` model stored in `cCRule`, with **two forked evaluators** (`routing/services/CCRuleExecutionService` live; `admin/rules/RuleExecutionService` for UBCT). *Evidence:* `Code_Briefing.md` §8 (decision 2). *Likely rationale (`inferred`):* change routing logic without redeploying. *Trade-off:* duplicated evaluators with drifting operator sets — a consistency/maintenance risk.
- **AOP-as-persistence.** *Observed:* aspects own result persistence, status transitions, and the monitoring trail. *Evidence:* `Code_Briefing.md` §6. *Likely rationale (`inferred`):* keep processors decision-focused, centralize audit. *Trade-off:* heavy coupling to `CreditCheckRequestScope` + non-obvious control flow.
- **Strategy + factory routing dispatch.** *Observed:* `CCTarget{system,api}` drives factories → concrete backend strategy. *Evidence:* `Code_Briefing.md` §2, §8. *Rationale (`inferred`):* open/closed addition of new credit backends. *Trade-off:* indirection between request and the backend actually called.
- **Kafka eventing via a hand-rolled wrapper.** *Observed:* `MessageBrokerClient` wraps a raw `KafkaProducer` (not `KafkaTemplate`); publish only on select paths. *Evidence:* `Code_Briefing.md` §5, §8. *Rationale (`inferred`):* fine control over the IEBus event contract. *Trade-off:* bypasses Spring Kafka conveniences; publish coverage is uneven.

## 11 · Observations

Risks and debt surfaced during breadth (carry into P2/P3): **(1)** no transaction management → partial-write inconsistency on mid-sequence failure (highest severity); **(2)** authorization is authentication-only — any valid token reaches any protected endpoint; **(3)** three dead/buggy security components (unwired opaque-token Caffeine cache → uncached introspection per request; unwired `CustomAuthenticationEntryPoint`; `isJwtShaped` routes by env not token shape); **(4)** two duplicated DSL evaluators with drifting operator sets; **(5)** only three declared indexes service-wide, with `creditCheckResult` doing dynamic/regex queries unindexed; **(6)** plaintext secrets committed as `application.yml` defaults; **(7)** `Credit.yaml` is a stale 6-of-89 subset missing the multi-product + policy surface; **(8)** `CacheService.evictCache` uses a literal (non-SpEL) cache name — the manual evict likely targets the wrong cache. Modernization notes (from `.github/appmod/appcat`) to be folded in at P3.

---
_Evidence convention: claim-cluster references to `Code_Briefing.md` §2–§9, where every fact carries a file/line anchor + provenance tag. See `HLD_Template.md` §3._
