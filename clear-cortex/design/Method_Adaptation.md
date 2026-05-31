# Method Adaptation — CIF applied to the Credit Routing Service

**Subject:** `apm0045942-credit-routing-service` @ `e17fe410` · **Owner:** Raja
**Source method:** `aaraminds-delivery/product-research/Code Intelligence Factory/` — `code intelligence framework.md`, `evaluation/HLD_Template.md` (v0.4), `evaluation/Evaluation_Rubric.md` (v0.3)

## The CIF method, in brief

Turn an undocumented codebase into a trustworthy model, then render artifacts from it. Two rules carry everything: **extract deterministic facts separately from inferred judgements**, and **anchor every non-trivial claim to evidence**. For a manual pass we apply the same discipline by hand — read the code, record facts with file/line locators, infer on top, and mark the inference. The artifact trio is `Code_Briefing.md` (deterministic) → `Inferred_Product_Spec.md` (inferred) → `HLD.md` (the rendered deliverable).

## Grounded snapshot @ `e17fe410`

Single Maven module, **Spring Boot 3.3.9 / Java 17**, app `CreditRoutingApplication`, context-path **`/CreditCheck`**. Proprietary AT&T code → near-zero training-data contamination.

| Dimension | Count / fact |
|---|---|
| Main Java files | 768 |
| Tests | 209 Spock (Groovy) + 14 JUnit — a large behavior oracle |
| REST controllers / endpoints | 28 controllers · ~107 `@*Mapping` · v1 + v2 surfaces |
| `@Service` / `@Repository` / `@Component` | 83 / 12 / 26 |
| `@Configuration` / `@ConfigurationProperties` | 21 / 6 |
| Persistence | **MongoDB** — 32 `@Document` collections, 29 `MongoRepository` (no relational DB) |
| Generated code | 21 MapStruct `@Mapper`, SOAP WSDL stubs (3 WSDLs), OpenAPI models |
| Eventing | Kafka, wrapped by `iebus/servicebus` — 1 `@KafkaListener`, 0 `KafkaTemplate` |
| Scheduling | 4 `@Scheduled` + ShedLock (multi-instance) |
| Security | OAuth2 / OIDC, JWT (Spring Security) |

**Subsystem map — the 14 packages under `com.att.creditcheck`:**

| Package | Files | Decoded role |
|---|---:|---|
| `routing/` | 179 | **Core domain** — credit routing + check; `v1` (`CreditRoutingController`) and `v2` (`singleproduct`, `multiproduct`, `internal`) APIs; policy controller |
| `admin/` | 157 | Admin surface (19 controllers): rules, policies, product mapping, roles, analytics, monitoring, pre-approval, audit. **Contains `admin/rules` — the custom DSL rules engine** |
| `csi/` | 49 | **Credit Services Integration** — SOAP clients to external credit services; request/response transformers + WSDL-generated stubs |
| `ubct/` | 29 | UBCT integration (acronym **to decode**) — `UBCTAPIServiceAspect` (AOP-wrapped external API) |
| `cas/` | 16 | Identity verification (`IdentityVerificationServiceAspect`) |
| `policy/` | 10 | Credit policy enforcement; `CRSMErrorHandlingService` |
| `iebus/` | 7 | **Enterprise messaging** — `servicebus/MessageBrokerClient`, `RetrieveKafkaConnection`; the real Kafka publish path |
| `multiproduct/` | 5 | Multi-product credit-check result models |
| `config/`, `common/`, `exceptions/`, `internal/`, `logging/`, `util/` | — | Spring config, shared infra, `GlobalExceptionHandler`, actuator/cache controllers, MDC logging |

**Existing knowledge to mine (don't re-derive):** `README.md`, `.github/copilot-instructions.md` (335 lines), `.github/instructions/Java.instructions.md`, `Credit.yaml` (OpenAPI, 1100 lines — authoritative API contract), `src/main/resources/application.yml`, `helm/`, `.github/appmod/appcat` (Azure app-modernization assessment output).

> **Drift already spotted:** the README's "Project Structure" lists `rules/`, `creditcheckresult/`, `audit/`, `cache/`, `oidc/` as if top-level under `creditcheck`, but the real layout nests them (the rules engine is `admin/rules`). Trust the code, not the README — and record the drift as a finding.

## Five adaptations from the CIF (Postgres/JPA) fixtures to this service

The CIF reference fixtures are Postgres/JPA services. This one is MongoDB + Kafka + SOAP + heavy AOP. The method holds; five specifics change:

1. **Persistence is document, not relational.** HLD §6 describes **Mongo collections** inferred from `@Document` (+ `@Indexed`/`@CompoundIndex`), not tables. No Flyway/Liquibase to "verify against" (the CIF M2 step) — verify against `@Document` annotations, repository query methods, and `application.yml`. Document relationships are by reference/embedding → **inferred** (the analog of the contract-builder "no JPA associations" challenge).
2. **Generated code must exist before extraction.** MapStruct mapper impls, SOAP WSDL stubs, and OpenAPI models are generated at build time. **Build first** (`./mvnw clean compile`) so `target/generated-sources` and `target/classes` are populated, or generated members — and every call through them — are invisible.
3. **AOP is load-bearing.** `cas`, `ubct`, `iebus` use aspects (`*Aspect.java`) that weave behavior off the call site. Map aspects explicitly; record what each intercepts in HLD §9.
4. **Eventing is wrapped.** Kafka goes through `iebus/servicebus/MessageBrokerClient` (+ `RetrieveKafkaConnection`), not `KafkaTemplate`. Trace the real publish path; the single `@KafkaListener` is the inbound consumer.
5. **SOAP integration in `csi/`.** External credit services are called via WSDL-generated clients + transformer services — an integration boundary for HLD §8, not hand-written REST clients.

The HLD §9 cross-cutting checklist and §10 decision-record structure carry over unchanged.

## Diagrams

Produced in P2–P3 and stored here in `design/`: an architecture / component diagram and a core runtime-flow diagram (credit-check v2). Per `HLD_Template.md` §4, prose must stand alone; diagrams support it.
