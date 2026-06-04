# Phase 1 — Execution Prompt & Exit Gate

**Repository Intelligence Factory**
Prepared 2026-06-03. Target repo frozen at `apm0045942-credit-routing-service@44b6b86`.
Reads with: `baseline/TARGET_ARCHITECTURE.md` (the *what*), `baseline/IMPLEMENTATION_ROADMAP.md` (Phase 1), `phase-0/` (validated inputs: gold sets + AGE benchmark).

Path roots used below: factory files are relative to this repo's root; brain assets are under `~/projects/aaraminds/`.

---

## How to use this

Two prompts. **Part A** drives the build. **Part B** is the acceptance contract you run before declaring Phase 1 done. Part B's thresholds were calibrated and locked in Phase 0 — they are not negotiable post-hoc.

---

## Part A — Phase 1 Execution Prompt

> Paste from here. Compose the assets named in §0 first, then execute.

### Inputs

**Primary input — the code to extract:**

- A clean, **buildable** Maven checkout of `clear/apm0045942-credit-routing-service` at `44b6b86`. Check the SHA out into a fresh worktree — *not* the live working tree (it carries 5 uncommitted files including `pom.xml`, which would change the build and drift `path:line` provenance).
- It must compile (`mvn compile` succeeds — verified: `target/` holds 1,566 classes). Symbol resolution needs the classpath, and Lombok members + the 481 JAXB sources exist only post-build. Build/parse it sandboxed (network-denied, resource-capped) — it is untrusted source.

**Schema-design references (build step 1):** `baseline/TARGET_ARCHITECTURE.md` §6 and `~/projects/aaraminds/Repo_Context_Platform_Architecture_v1.0.md` — reconcile, don't re-invent.

**Composed assets (per §0):** the AI Engineering Architect persona + the four skills + the agents, loaded into context.

**Phase 0 carry-ins:** `phase-0/evalset/{understanding,impact}-goldset.csv` (build step 6 / gate G9) and `phase-0/age-benchmark/` (gate G7).

**Environment:** JDK 17 + Maven; the resolving parser (Eclipse JDT or JavaParser-with-symbol-solver); a dev Postgres 16 + AGE + pgvector.

**Parameters:** frozen SHA `44b6b86` · language Java · graph store = AGE behind `GraphStore` · output dir `phase-1/`.

### §0 — Compose these AaraMinds assets before starting

- **Persona (design authority):** `~/projects/aaraminds/instruction-os/Persona/AaraMinds_AI_Engineering_Architect_v1.2.md` + its required base modules (01 Layered Base, 02 Visual Identity, 04 Framework Creation).
- **Skills (read the SKILL.md routers):**
  - `codebase-comprehension` — designs the graph model (the M0 schema, provenance rule, Spring-stereotype modeling).
  - `codebase-extraction-engineering` — builds the extractor (parser choice, symbol resolution, build integration, deterministic IDs).
  - `azure-data-tier-design` + `data-access-engineering` — the AGE graph + pgvector + relational store, migrations, queries.
  - `test-engineering` — the provenance CI assertion and golden-fixture tests.
  - All under `~/projects/aaraminds/skills-pack/.claude/skills/<name>/SKILL.md`.
- **Agents:** dispatch `aara-senior-microservices-architect` and `aara-mcp-server-builder` as Claude subagents (they exist in `.claude/agents/`). The design/build specialists — `aara-code-model-designer`, `aara-codebase-extraction-engineer`, `aara-data-tier-designer` — are Copilot-format only; drive them via their underlying skills here, or run them directly in a Copilot session.

### §1 — Mission

Phase 1 is the **walking skeleton**: make one real repo flow end-to-end into a queryable, provenance-complete **deterministic** graph, fully on-stack. Prove the pipeline works; deliberately defer SCIP precision, embeddings, retrieval, and incrementality to later phases.

### §2 — Target (frozen)

`clear/apm0045942-credit-routing-service` @ `44b6b86` — Spring Boot 3.3.9 / Java 17 / Maven. ~604 source + 481 generated JAXB Java files; compiles clean (`target/` present, jar builds). 83 `@Service`, 28 `@RestController`, 12 `@Repository`, 11 `@Aspect`, 3 SOAP WSDLs (+39 XSDs), Lombok throughout. This profile is *why* the constraints below are non-negotiable: DI + AOP + SOAP + generated code are exactly where naive extractors fail silently.

### §3 — Non-negotiable constraints

1. **Resolving parser, not syntax-only.** The extractor MUST use **Eclipse JDT** or **JavaParser-with-symbol-solver** (or `scip-java`), run against a **built Maven checkout** with the dependency classpath assembled. Tree-sitter / regex as the extractor is the banned anti-pattern (`codebase-extraction-engineering`): no symbol resolution means it cannot bind a call to its target, distinguish overloads, or follow interface dispatch — the call graph is wrong in ways nothing downstream can detect. Tree-sitter is permitted **only** for cheap structural chunking, later (Phase 2 cAST).
2. **Schema first (M0).** Settle the node/edge taxonomy, identity scheme, provenance, and versioning **before** the extractor writes a node (`codebase-comprehension`). Reconcile the two existing designs first — `baseline/TARGET_ARCHITECTURE.md` §6 and `~/projects/aaraminds/Repo_Context_Platform_Architecture_v1.0.md` — do not invent a third.
3. **Deterministic vs inferred, never blended.** Tag every fact. Tier-A `exact` from AST + resolution; DI/stereotype edges deterministic; anything inferred is visibly separate and lower-confidence.
4. **Spring-stereotype-aware.** Read `@RestController` / `@Service` / `@Repository` / `@Autowired` as first-class **deterministic** edges. This moves the Design layer from guessed to parsed and is the highest-leverage decision in the pipeline.
5. **Generated code is handled.** Extract against post-annotation-processing output so Lombok members and the 481 JAXB stubs (the SOAP contract surface) are visible. A source-text-only extractor is silently incomplete.
6. **Provenance on everything.** Every node and edge carries `repo@sha:path:line-range`, `confidence` (`exact`/`probable`/`inferred`), `evidence`, `index_version`. This is the project's "100% traceability" definition — self-citation completeness, asserted in CI.
7. **Deterministic output.** Same SHA in → byte-identical model out (sort by stable ID before emit; stamp the commit, never `now()`). Stable IDs across rebuilds so the model diffs cleanly.
8. **`GraphStore` interface.** All graph reads/writes go through it so the spike ↔ AGE ↔ fallback swap is one adapter.
9. **On-stack only.** Go services on Azure Container Apps; Postgres Flexible Server + AGE + pgvector; Terraform AzureRM (RBAC mode); GitHub Actions OIDC; Key Vault via managed identity. No AWS, Neo4j-in-prod, Bicep, GitLab, Datadog.

### §4 — Build order (thin end-to-end slice first; do not perfect one stage before the pipeline runs)

1. **M0 schema** — nodes (Repository / Module / Package / File / Type / Method / Endpoint) + Tier-A edges (CONTAINS, DEFINES, IMPORTS, EXTENDS/IMPLEMENTS, resolved CALLS, INJECTS, EXPOSES), with provenance + identity. Deliverable: schema doc + AGE DDL + relational-metadata DDL + `index_version`.
2. **`GraphStore` interface + store** — Postgres schema (relational + AGE graph + FTS stub); migrations via `data-access-engineering`.
3. **Resolving extractor** — parse → resolve → walk → emit over the built checkout; emit nodes + Tier-A + Spring-stereotype edges + provenance; deterministic IDs. Deliverable: `graph.json` and a direct AGE load path.
4. **Load + query** — load the real graph into AGE; demonstrate `find_callers`, `dependents@depth≤3`, `list endpoints`, `DI wiring` as Cypher returning cited nodes/edges.
5. **Provenance CI gate** — assert 100% of nodes/edges carry a resolvable `source_ref`; fail the build otherwise.
6. **Eval sanity** — run the graph-answerable subset of `phase-0/evalset/*` (callers, DI, endpoints, inheritance + impact traversal) and record results against the locked thresholds.

### §5 — Working method

Thin slice before depth. Cite every claim to a source location. Confidence-tier every edge. Lead with fatal flaws before helping execute. Never present a static edge as proof a path runs at runtime — static reachability is possibility, not observation.

### Outputs

1. **M0 schema** (`phase-1/schema/`) — node/edge taxonomy doc + AGE graph DDL + relational-metadata DDL, with the identity / provenance / `index_version` scheme.
2. **`GraphStore` interface + store** — the Go interface, the AGE-backed implementation, and Postgres migrations.
3. **Resolving extractor + `phase-1/graph.json`** — the extractor program plus its emitted graph (file/type/method nodes + Tier-A + Spring-stereotype edges; every element carries `repo@44b6b86:path:line`, confidence, `index_version`).
4. **Loaded AGE graph** + the 4 demo Cypher queries (`find_callers`, `dependents@depth≤3`, endpoints, DI wiring) returning cited rows.
5. **Provenance CI gate** — the assertion that fails the build when any node/edge lacks a resolvable `source_ref`.
6. **Eval-sanity record** — the graph-answerable gold-set subset + impact traversal, scored per tier vs the locked thresholds.

These six outputs are exactly what the Exit Gate (Part B) consumes.

---

## Part B — Phase 1 Exit Gate

Phase 1 is **done** only when every gate passes. Run this as the acceptance checklist and produce a one-page **Phase 1 Acceptance Memo** (mirrors the Phase 0 findings memo). Thresholds were locked in Phase 0 — do not move them.

### Inputs

- **All six Part A outputs** — especially `graph.json`, the loaded AGE graph, the extractor (for the determinism re-run), and the `GraphStore` interface with ≥ 2 implementations.
- **Two extraction runs** of the same SHA, for the G4 determinism diff.
- **`phase-0/age-benchmark/benchmark.py` + `setup.sql`**, pointed at the **real** graph (not synthetic) — gate G7.
- **`phase-0/evalset/*.csv`** — gate G9, scored against the **locked thresholds** already in the table.
- **A dev Postgres 16 + AGE** to load and benchmark; and, only if G7 fails, access to a fallback store (Cosmos Gremlin, or FalkorDB on Container Apps).

### Gates

| # | Gate | Pass condition | How to verify | Fail action |
|---|---|---|---|---|
| G1 | Walking skeleton | `credit-routing-service@44b6b86` ingested end-to-end; Cypher returns cited nodes/edges | Run the 4 demo queries; each row carries `repo@sha:path:line` | Fix the failing pipeline stage |
| G2 | Resolving extractor | Call edges built from **resolved bindings**, not name-matching; overloads distinct; interface dispatch followed | Spot-check 20 edges against source; confirm no syntax-only fallback | Adopt JDT / JavaParser-symbol-solver; rebuild |
| G3 | Generated code | Lombok members + JAXB stubs present in the model | Query a `@RequiredArgsConstructor` class's constructor edge + a JAXB-generated type | Run extraction post-annotation-processing |
| G4 | Determinism | Same SHA → byte-identical model on re-run; stable IDs across rebuilds | Diff two runs; assert empty diff | Remove `now()` / unordered iteration; sort before emit |
| G5 | Provenance (100%) | Every node + edge carries a resolvable `source_ref` + confidence + `index_version` | CI assertion fails the build if any element lacks it | Block merge until complete |
| G6 | Spring-stereotype edges | `@RestController` / `@Service` / `@Repository` / DI modeled as **deterministic** edges | Confirm `CCRoutingService`'s 8 injected beans appear as INJECTS edges | Add the stereotype pass |
| **G7** | **AGE go/no-go (folded exit test)** | The 5 real traversal queries at depth ≤ 3 on the **real extracted graph** hit **p95 < 1500 ms / p50 < 500 ms** | `phase-0/age-benchmark/benchmark.py` loaded with the real graph (not synthetic) | **Fallback via `GraphStore`:** Cosmos Gremlin (strict managed) or FalkorDB on Container Apps |
| G8 | `GraphStore` swap | The spike ↔ AGE ↔ fallback swap is one adapter | Show the interface + ≥2 implementations compiling against it | Refactor all reads behind the interface |
| G9 | Eval baseline | Graph-answerable subset of the gold sets scored; impact traversal reported **per tier** | Capability (structural subset) ≥ 50% with citations; impact Tier-A recall ≥ 0.80 / precision ≥ 0.70, Tier-B DI recall ≥ 0.50 | Record the gap; Tier-C (SOAP/AOP) deferred to Phase 2 |
| G10 | On-stack | Go on Container Apps; Terraform AzureRM; Key Vault via MI; no off-stack tech | Review IaC + service manifests | Replace any drift |

**Scope honesty (state this in the memo):** Phase 1 proves the deterministic pipeline. Full semantic Q&A (needs embeddings + retrieval) is a **Phase 3** gate; SCIP precision and AOP/SOAP Tier-C edges are **Phase 2**; incremental freshness is **Phase 5**. The Phase 1 eval therefore grades only what the Tier-A + stereotype graph can answer — callers, DI wiring, endpoints, inheritance, and static/DI impact traversal — not the full gold set.

### Outputs

- **The filled G1–G10 checklist** — pass/fail with evidence per gate.
- **The AGE go/no-go verdict** + the p50/p95 latency table from the real-graph benchmark.
- **The store decision** — Phase 2 proceeds on AGE, or on the named fallback. This is the one architectural decision the gate exists to make.
- **The one-page Phase 1 Acceptance Memo** — the above plus the deferred-scope note (semantic Q&A → Phase 3, SCIP / Tier-C → Phase 2, incrementality → Phase 5). This memo is the artifact that authorizes Phase 2.
