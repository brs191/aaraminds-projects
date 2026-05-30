# Roadmap — Credit Routing Service Comprehension

**Subject:** `apm0045942-credit-routing-service` @ `e17fe410` · **Owner:** Raja · **Status:** durable plan v1

Gates, not checkbox counts, govern progress. Effort is dimensioned for **~1.0 FTE**; calendar = effort ÷ capacity. Scope choice (set with the user): **breadth map first, then deepen**.

## Phases & gates

### P0 — Foundations / fresh-start setup · ~0.5–1 day
- Pin the commit (`e17fe410`, `develop`) in `evaluation/HLD.md` Document Control.
- **Build the repo:** `docker-compose up -d` (local Mongo) → `./mvnw clean compile`. Confirms generated sources exist and the tree compiles at this SHA (the deterministic ground truth). *Run from a working copy — never write into the repo.*
- Mine the existing sources into the Code Briefing's raw-material section — especially `Credit.yaml` (endpoint inventory) and `.github/copilot-instructions.md` (conventions).
- **Gate:** SHA pinned · repo compiles · existing-doc facts captured · adapted template + rubric in place.

### P1 — Breadth map (whole service, shallow) · ~2–4 days
- **`Code_Briefing.md` (breadth):** deterministic inventory across all 14 packages — package roles; 28 controllers + ~107 endpoints (v1 vs v2, under `/CreditCheck`); 32 Mongo collections + 29 repositories; the `@Service` layer; 21 MapStruct mappers; integration points (Mongo, IEBus/Kafka, CSI/SOAP, OIDC, `ubct`, `cas`); config surface; schedulers + ShedLock; cross-cutting (security filter chain, `GlobalExceptionHandler`, audit, Caffeine cache, MDC logging, aspects). **Decode `cas`/`ubct`/`iebus`.** Locator + provenance on every fact; mark `[not deep-read]` where shallow.
- **`Inferred_Product_Spec.md` (breadth):** capabilities (credit routing; single- & multi-product credit check; DSL rule evaluation; policy enforcement; identity verification; admin/config/rules management; analytics & monitoring), actors/callers, value flow. Marked inferred.
- **`HLD.md` (breadth, whole-service):** template §§1–11 at component altitude. Fill the §9 checklist (Covered / Not visible / Out of scope — no silent omission) and start §10 decision records for the obvious decisions (Mongo over relational; IEBus eventing wrapper; the DSL rules engine; v1/v2 API split; AOP cross-cutting).
- **Gate (rubric self-check):** completeness (every major component + integration named) · architectural correctness (sound decomposition) · critical-error rule (zero fabrications) · evidence anchors on non-trivial claims · altitude held. **Output:** a coherent shallow whole-service HLD + a ranked deepen list.

### P2 — Deepen highest-value areas · ~1–2 days each
Priority (adjust at the P1 gate):

1. **Core credit-check runtime flow (v2):** request → `routing` → `admin/rules` DSL evaluation → `policy` → credit-check result → IEBus event. Touches the most subsystems.
2. **The DSL rules engine (`admin/rules`):** the custom, novel, highest-risk subsystem — how rules are defined, stored (Mongo), and evaluated.
3. **Domain & data model (32 Mongo collections):** relationships by reference/embedding → inferred.
4. **External integrations:** CSI/SOAP (external credit services), IEBus/Kafka eventing, OIDC, `ubct`.
5. **v1 vs v2 + multi-product divergence**, and the `admin/` surface (deep-read vs. catalogue — 157 files).

Per area: extend the Briefing with deep-read facts; upgrade the HLD section inferred → evidence-backed; add step-level runtime detail; complete §10 decision records (observed / evidence / likely rationale `inferred` / trade-off).
- **Gate per area:** rubric altitude + accuracy + evidence bars met; every inferred claim carries a confidence band.

### P3 — Consolidate, verify, finalize · ~1–2 days
- Assemble the final `HLD.md` + `Code_Briefing.md` + `Inferred_Product_Spec.md`; add the architecture + runtime-flow diagrams to `design/`.
- **Verification** (Rubric §6): self-score with the Scorecard (≥ 70/100, accuracy ≥ 3/4, no zero dimension); spot-check ~15 evidence anchors resolve to real code at `e17fe410`; run the no-silent-omission check on §9; confirm zero fabricated components/flows/integrations; ideally a second-reviewer pass (Raja or a peer) on Part B.
- Seed §11 Observations + modernization notes from `appcat` — without scope-creeping into modernization *execution*.

## Risks & adaptations (repo-specific)

| Risk | Why it bites | Mitigation |
|---|---|---|
| MongoDB, not relational | Template assumes JPA tables/migrations | Describe collections from `@Document`; verify against indexes + `application.yml`; relationships `inferred` |
| Generated code (MapStruct, SOAP, OpenAPI) | Members & call edges invisible in raw source | `./mvnw clean compile` first; read `target/generated-sources`; tag members `generated` |
| Heavy AOP (`cas`/`ubct`/`iebus`) | Behavior woven off the call site | Map aspects explicitly; record what each intercepts in §9 |
| Kafka wrapped by IEBus | `grep KafkaTemplate` finds nothing | Trace `iebus/servicebus/MessageBrokerClient` for the real publish path |
| README/layout drift | Following the README mislocates subsystems | Trust the code; record drift as a finding |
| Large `admin/` (157 files) | Deepening all of it blows the budget | Decide deep-read vs. catalogue at the P1 gate |
| Acronyms (`cas`, `ubct`, `iebus`) | Opaque names hide responsibilities | Decode in P1 from code + `copilot-instructions.md` + config |
| `develop` moves | Active branch (PR #1275 today) | Pin `e17fe410`; re-pin only deliberately |

## What this is NOT

Not building the Eclipse-JDT → graph → generator pipeline (a separate option). Not LLD (class-by-class). Not a rewrite or greenfield redesign. Not modernization *execution* (observations only). Not multi-repo. This is manual, Claude-assisted comprehension of **one repo at one pinned SHA**, with the code repo kept read-only.

**Rough effort:** an onboarding-grade draft (P0–P1) ≈ 1 week at 1.0 FTE; a sign-off-quality whole-service HLD (through P3) ≈ 2–3 weeks. Don't commit a calendar without naming the FTE.
