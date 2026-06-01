# Code Briefing — Credit Routing Service

**Status:** STUB — fill in P1 (breadth) then extend in P2 (deepen)
**Fixture:** `apm0045942-credit-routing-service` @ `44b6b8659a33ad7ac2227b6d88696d1946b9ce1a` (read-only working copy — confirmed by Raja on his macOS P0-validation system; see §0 SHA note)
**Purpose:** organize _verified_ facts from source in HLD section order, so `HLD.md` is written from real code, not memory. This is raw material, not the HLD.

> Every fact cites a repo-relative locator (`file › Type#member › L<start>–<end>`) and provenance. Mark `[not deep-read]` where inventoried but not read line-by-line. Build the repo first (`./mvnw clean compile`) so MapStruct/SOAP/OpenAPI generated members are present.

## 0 · Read coverage

- **Fully read (P0):** `README.md` (overview + build/run guidance), `.github/copilot-instructions.md` (build conventions + package map), `Credit.yaml` (contract header + top-level paths), `src/main/resources/application.yml` (runtime/config surface), and `start-credit-service.sh` (current helper build/run flow). Sample locators: `README.md` › `## Overview` / `## Building the Service` / `## Running the Service` › L5/L55/L82; `.github/copilot-instructions.md` › `Maven Wrapper` / `MongoDB` / `application.yml` guidance › L24-L41; `Credit.yaml` › `info.title` / `info.version` / `/credit-checks` / `/ping` › L3/L7/L15/L155; `src/main/resources/application.yml` › SpringDoc groups / Mongo / security / context path / IEBus / CAS › L1-L14/L29-L33/L42-L69/L70-L71/L336-L352/L354-L359; `start-credit-service.sh` › Java selection / Maven selection / generated-source cleanup / build / run › L9-L16/L27-L31/L33-L38/L41-L43. Provenance: deterministic.
- **Working copy pinned [human_confirmed — Raja, macOS]:** on Raja's macOS P0-validation system, local branch ref `develop` resolved to `44b6b8659a33ad7ac2227b6d88696d1946b9ce1a` (Locators: `.git/HEAD` › L1, `.git/refs/heads/develop` › L1). **Reconciliation note:** this could not be re-verified from the workspace-accessible clone (`coderepos/clear/apm0045942-credit-routing-service`), which is at `e17fe410e79e8784d67c9e4dc505f210500e7cf1` (= AT&T `origin/develop` tip seen here); `44b6b86…` is absent from that clone's objects, packed-refs, and reflog. Reconcile the two copies to one revision before P1.
- **Generated-source surface verified [deterministic + runtime-observed]:** `start-credit-service.sh` was rerun in this session and regenerated code under the expected slices, including MapStruct annotations output plus SOAP/OpenAPI-derived client/server sources. Sample anchors: `target/generated-sources/annotations/com/att/creditcheck/policy/mapper/EUCPCMapperImpl.java` › package / `@Generated` › L1/L20; `target/generated-sources/csi/com/att/csi/csi/namespaces/orderandsubscriptionmanagementmobility/types/_public/commondatamodel/package-info.java` › package declaration › L9; `target/generated-sources/openapi-client/src/main/java/com/att/creditcheck/client/api/UnifiedCreditCheckServiceApi.java` › package declaration › L1; `target/generated-sources/crsms/src/main/java/com/att/creditcheck/client/crsms/api/CustomerRiskSystemManagerServiceApi.java` › package declaration › L1. Provenance: deterministic + runtime-observed.
- **Build-gate status [runtime-observed]:** P0 build confirmation now passes. The checked-in `start-credit-service.sh` executed through package/build, regenerated generated-source output, produced the application jar, and launched the Spring Boot service. A follow-up HTTP check to `/CreditCheck/health` returned `401 Unauthorized`, which confirms the service was serving under the configured context path with security enabled. Runtime Kafka listener authentication errors were still present after startup, but those occur after the requested P0 build/generation gate and do not invalidate the P0 deliverable. Locators: `start-credit-service.sh` › package command / jar launch › L41-L43/L52-L60; `src/main/resources/application.yml` › `server.servlet.context-path` / `management.endpoints.web` › L70-L71/L88-L91.

## 1 · Purpose & context (HLD §2)

- **Service identity [deterministic]:** the repo describes Credit Routing Service as a Spring Boot 3.3.9 microservice that orchestrates credit-check requests using configurable business rules, supports multi-product evaluation, persists data in MongoDB, and publishes asynchronous events via Kafka. Locators: `README.md` › overview paragraph › L5-L7; `.github/copilot-instructions.md` › repository overview bullets › L8-L16.
- **Security posture [deterministic]:** the repo advertises OAuth2/OIDC security with JWT-based authentication, and runtime configuration includes Spring Security resource-server issuer/audience settings plus Microsoft Entra and opaque-token settings. Locators: `README.md` › key features › L16-L19; `src/main/resources/application.yml` › `spring.security.oauth2.resourceserver` › L42-L69.
- **API surface framing [deterministic]:** the checked-in OpenAPI contract identifies the service as `Credit Logging and Evaluation Assistance Repository API`, version `2.0.0`, with `/credit-checks` and `/ping` among its top-level paths; runtime SpringDoc groups include `credit-check-v2`, `multi-product-credit-check`, `credit-check-v1`, `credit-checks-sentry`, and `admin`. Locators: `Credit.yaml` › `info.title` / `info.version` / `/credit-checks` / `/ping` › L3/L7/L15/L155; `src/main/resources/application.yml` › `springdoc.group-configs` › L1-L14.
- **Runtime context path and observability [deterministic]:** the service is mounted under `/CreditCheck`, exposes health and metrics at the management web base path `/`, and enables distributed tracing by default. Locator: `src/main/resources/application.yml` › `server.servlet.context-path` / `management.endpoints.web` / `management.tracing` › L70-L71/L83-L91.
- **Persistence and local-run defaults [deterministic]:** the primary datastore is MongoDB, configured through `spring.data.mongodb.database` and `spring.data.mongodb.uri`; defaults point at a remote Atlas-style host and the repo guidance still documents Docker Compose for local MongoDB. Locators: `src/main/resources/application.yml` › `spring.data.mongodb` › L29-L33; `README.md` › local MongoDB setup › L38-L51; `.github/copilot-instructions.md` › MongoDB environment setup › L28-L33.
- **Named external integrations [deterministic]:** runtime configuration names CSI SOAP endpoints, UBCT, IEBus/Kafka, and CAS APIs, while repo guidance maps the package names `csi`, `ubct`, `iebus`, and `cas` to those integration roles. Locators: `src/main/resources/application.yml` › `csi.host` and CSI API blocks › L115-L211, `ubct.host` / `ubct.api` › L256-L263, `ie.bootstrap.servers` / `iebus.topic` › L336-L352, `cas-api.host` › L354-L359; `.github/copilot-instructions.md` › package map bullets › L8-L16.
- **Build convention vs helper reality [deterministic + runtime-observed]:** repo guidance expects Java 17 and the Maven Wrapper, while the helper script uses system `mvn` and local `java -jar`. On this machine, the rerun executed under Java 25 because no Java 17 installation was available, but the script still completed package/build and launched the application. That environment mismatch is important for later runtime debugging, but it did not block the P0 deliverable. Locators: `.github/copilot-instructions.md` › Java 17 / Maven Wrapper guidance › L22-L27; `README.md` › Maven Wrapper build steps › L55-L80; `start-credit-service.sh` › Java selection › L9-L16, `MVN="mvn"` › L27, package command › L43, `java -jar` launch › L60.

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
