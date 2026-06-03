# Roadmap — Credit Routing Service Comprehension

**Subject:** `apm0045942-credit-routing-service` @ `e17fe410` · **Owner:** Raja · **Status:** durable plan v2 (Project-Planner fixes applied 2026-06-01)

Gates, not checkbox counts, govern progress. Effort is dimensioned for **~1.0 FTE**; calendar = effort ÷ capacity. Scope choice (set with the user): **breadth map first, then deepen**.

**Fixed constraint:** **scope-floor** — the whole-service breadth HLD (through P1) is the must-ship. **Depth** (which P2 areas, how deep) and **time** are the levers; **quality** (evidence discipline, zero fabrications) is the floor, never a lever. Owner: **Raja** (≤ 1.0 FTE). Plan date vs. committed date are not set until the FTE/calendar is named.

## Phases & gates

### P0 — Foundations / fresh-start setup · ~0.5–1 day
- Pin the commit (`e17fe410`, `develop`) in `evaluation/HLD.md` Document Control.
- **Build the repo:** `docker-compose up -d` (local Mongo) → `./mvnw clean compile`. Confirms generated sources exist and the tree compiles at this SHA (the deterministic ground truth). *Run from a working copy — never write into the repo.*
- Mine the existing sources into the Code Briefing's raw-material section — especially `Credit.yaml` (endpoint inventory) and `.github/copilot-instructions.md` (conventions).
- **Gate:** SHA pinned · repo compiles · existing-doc facts captured · adapted template + rubric in place.

### P1 — Breadth map (whole service, shallow) · ~2–4 days
- **`Code_Briefing.md` (breadth):** deterministic inventory across all 14 packages — package roles; the controllers + endpoints (v1 vs v2, under `/CreditCheck`); the Mongo collections + repositories; the `@Service` layer; 21 MapStruct mappers; integration points (Mongo, IEBus/Kafka, CSI/SOAP, OIDC, `ubct`, `cas`); config surface; schedulers + ShedLock; cross-cutting (security filter chain, `GlobalExceptionHandler`, audit, Caffeine cache, MDC logging, aspects). **Decode `cas`/`ubct`/`iebus`.** Locator + provenance on every fact; mark `[not deep-read]` where shallow.
- **`Inferred_Product_Spec.md` (breadth):** capabilities (credit routing; single- & multi-product credit check; DSL rule evaluation; policy enforcement; identity verification; admin/config/rules management; analytics & monitoring), actors/callers, value flow. Marked inferred.
- **`HLD.md` (breadth, whole-service):** template §§1–11 at component altitude. Fill the §9 checklist (Covered / Not visible / Out of scope — no silent omission) and start §10 decision records for the obvious decisions (Mongo over relational; IEBus eventing wrapper; the DSL rules engine; v1/v2 API split; AOP cross-cutting).
- **Gate (rubric self-check):** completeness (every major component + integration named) · architectural correctness (sound decomposition) · critical-error rule (zero fabrications) · evidence anchors on non-trivial claims · altitude held. **Output:** a coherent shallow whole-service HLD + a ranked deepen list.

### P2 — Deepen highest-value areas · ~1–2 days each
Priority (adjust at the P1 gate):

1. **Core credit-check runtime flow (v2):** request → `routing` → `admin/rules` DSL evaluation → `policy` → credit-check result → IEBus event. Touches the most subsystems.
2. **The DSL rules engine (`admin/rules`):** the custom, novel, highest-risk subsystem — how rules are defined, stored (Mongo), and evaluated.
3. **Domain & data model (29 Mongo collections):** relationships by reference/embedding → inferred.
4. **External integrations:** CSI/SOAP (external credit services), IEBus/Kafka eventing, OIDC, `ubct`.
5. **v1 vs v2 + multi-product divergence**, and the `admin/` surface (deep-read vs. catalogue — 157 files).

Per area: extend the Briefing with deep-read facts; upgrade the HLD section inferred → evidence-backed; add step-level runtime detail; complete §10 decision records (observed / evidence / likely rationale `inferred` / trade-off).
- **Gate per area:** rubric altitude + accuracy + evidence bars met; every inferred claim carries a confidence band.

### P3 — Consolidate, verify, finalize · ~1–2 days
- Assemble the final `HLD.md` + `Code_Briefing.md` + `Inferred_Product_Spec.md`; add the architecture + runtime-flow diagrams to `design/`.
- **Verification** (Rubric §6): self-score with the Scorecard (≥ 70/100, accuracy ≥ 3/4, no zero dimension); spot-check ~15 evidence anchors resolve to real code at `e17fe410`; run the no-silent-omission check on §9; confirm zero fabricated components/flows/integrations; ideally a second-reviewer pass (Raja or a peer) on Part B.
- Seed §11 Observations + modernization notes from `appcat` — without scope-creeping into modernization *execution*.

## Risks & adaptations (register)

Probability / Impact = H/M/L. Response ∈ avoid · mitigate · accept · transfer.

| Risk | P·I | Response | Owner | Trigger signal |
|---|---|---|---|---|
| **Build needs internal artifact / VPN access** — `settings.xml` mirrors all deps to `artifact.it.att.com`; `com.att.ttrace` is internal (the #1 P0 blocker) | H·H | mitigate: build on the AT&T network with a JFrog token. **Fallback:** comprehend from source, flag the generated-code blind spot | Raja | `./mvnw` can't resolve deps off-network |
| **Second human reviewer unavailable** — the rubric (§6) needs two; an external dependency | M·H | mitigate: line up a senior peer early. **Fallback:** single-reviewer sign-off with the limitation documented | Raja | no 2nd reviewer by the gate date |
| **SHA unreconciled** — workspace `e17fe410` vs Mac `44b6b86…` differ; P2 must deepen one revision | M·H | mitigate: pin one revision before P2; re-pin only deliberately | Raja | the two clones differ at P2 start |
| **Count discipline** — the P1 gate caught off-by-N counts | M·M | mitigate: grep-verify every count as it is written | Raja | a reviewer finds a wrong count |
| Generated code (MapStruct/SOAP/OpenAPI) invisible in source | M·M | mitigate: build first; read `target/generated-sources`; tag `generated` | Raja | members missing from the model |
| Heavy AOP (`cas`/`ubct`/`iebus`) hides behavior off the call site | M·M | mitigate: map aspects; record woven concerns in §9 | Raja | a flow's behavior unexplained at the call site |
| Large `admin/` (157 files) blows the P2 budget | M·M | accept: catalogued at P1; deep-read only `admin/rules` | Raja | P2 admin deep-read exceeds budget |
| MongoDB, not relational | H·L | accept: describe collections from `@Document`; relationships `inferred` | Raja | — |
| README/layout drift mislocates subsystems | M·L | mitigate: trust the code; record drift as a finding | Raja | doc-vs-code mismatch found |

## Assumptions (each is a replan trigger if invalidated)

- The workspace clone (`e17fe410`) and the Mac copy (`44b6b86…`) are the **same** revision — *unverified; reconcile before P2.*
- ~1.0 FTE (Raja) is available; no committed calendar exists without that.
- The DSL engine + AOP are comprehensible from a **static** read (no runtime needed for breadth/deepen).
- The 209 Spock tests are a usable behavior oracle for P2.
- A second qualified human reviewer can be secured for the gate.

## Replan triggers

Replan (produce a new honest baseline — do **not** silently absorb) if any fires: a phase slips past its buffer · the working copy can't be pinned to one revision at P2 start · P1 breadth reveals the service is materially larger / more coupled than the ~768-file estimate · the second human reviewer can't be secured · capacity drops below ~1.0 FTE · a load-bearing assumption above is invalidated.

## Buffer

One **phase-boundary buffer before P3** (sign-off), sized to the P2 outcome and owned by the delivery lead (Raja), who decides when to release it. Per-phase estimates are honest mid-points; uncertainty lives in this buffer, not padded into tasks.

## What this is NOT

Not building the Eclipse-JDT → graph → generator pipeline (a separate option). Not LLD (class-by-class). Not a rewrite or greenfield redesign. Not modernization *execution* (observations only). Not multi-repo. This is manual, Claude-assisted comprehension of **one repo at one pinned SHA**, with the code repo kept read-only.

**Rough effort:** an onboarding-grade draft (P0–P1) ≈ 1 week at 1.0 FTE; a sign-off-quality whole-service HLD (through P3) ≈ 2–3 weeks. Don't commit a calendar without naming the FTE.
