# Code Briefing — Credit Routing Service

**Status:** STUB — fill in P1 (breadth) then extend in P2 (deepen)
**Fixture:** `apm0045942-credit-routing-service` @ `e17fe410` (read-only working copy)
**Purpose:** organize *verified* facts from source in HLD section order, so `HLD.md` is written from real code, not memory. This is raw material, not the HLD.

> Every fact cites a repo-relative locator (`file › Type#member › L<start>–<end>`) and provenance. Mark `[not deep-read]` where inventoried but not read line-by-line. Build the repo first (`./mvnw clean compile`) so MapStruct/SOAP/OpenAPI generated members are present.

## 0 · Read coverage
_TODO P1 — list what was fully read vs. structurally mapped vs. not deep-read._

## 1 · Purpose & context (HLD §2)
_TODO P1 — from README, Credit.yaml, application.yml, copilot-instructions._

## 2 · Architecture overview & component view (HLD §4–5)
_TODO P1 — the 14 packages and their roles; the @Service / controller / repository shape. Decode `cas`, `ubct`, `iebus`._

## 3 · Domain & data model (HLD §6)
_TODO P1/P2 — the 32 `@Document` collections; inferred relationships (by reference/embedding); indexes._

## 4 · REST surface (HLD §8)
_TODO P1 — 28 controllers, ~107 endpoints, v1 vs v2 under `/CreditCheck`. Cross-check against `Credit.yaml`._

## 5 · External interfaces & integrations (HLD §8)
_TODO P1 — CSI/SOAP (external credit services), IEBus/Kafka eventing, OIDC, `ubct`._

## 6 · Cross-cutting (HLD §9)
_TODO P1 — security filter chain, GlobalExceptionHandler, audit, Caffeine cache, MDC logging, ShedLock, and the AOP aspects (what each intercepts)._

## 7 · Configuration & runtime
_TODO P1 — 21 `@Configuration`, 6 `@ConfigurationProperties`, key `application.yml` paths, 4 `@Scheduled` jobs._

## 8 · Candidate design decisions (HLD §10)
_TODO P1 seed → P2 complete — Mongo over relational; IEBus eventing wrapper; DSL rules engine; v1/v2 split; AOP cross-cutting._
