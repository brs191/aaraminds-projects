# Phase 1 — Walking Skeleton (thin slice executed)

Deterministic code-knowledge-graph pipeline for `apm0045942-credit-routing-service@44b6b86`, built per `PHASE_1_PROMPT.md` and the corrected approach (resolving parser, **not** tree-sitter; schema-first; provenance on everything). This folder contains the real artifacts; a verified **thin slice** of the graph was run end-to-end to prove the pipeline.

## Layout

```
phase-1/
├── schema/         SCHEMA.md (reconciled M0) · age_schema.sql · relational_schema.sql
├── extractor/      pom.xml · src/.../Extractor.java · IdGen.java · run.sh   (JavaParser+SymbolSolver)
├── graph/          build_thin_slice.py -> graph.thin-slice.json             (verified golden fixture)
├── graphstore/     graphstore.go        (the GraphStore interface + JSONStore + AGEStore)
├── loader/         load_age.py -> load.cypher
├── eval/           provenance_check.py · eval_sanity.py
├── query/          demo_queries.py
└── README.md
```

## What ran here (green, reproducible)

The sandbox has a JDK 11 JRE only (no `javac`/`mvn`) and no live AGE, so the pipeline was exercised over the **hand-verified thin slice** — the v1 credit-check chain (controller → routing → execution → 9 CSI clients → SOAP), its Spring DI fan-in, and the two advising aspects. Every line was verified against the repo.

- **Schema (M0)** — `schema/SCHEMA.md` reconciles `TARGET_ARCHITECTURE §6` with the brain's `Repo_Context_Platform_Architecture_v1.0.md` (no third design); DDL in `age_schema.sql` + `relational_schema.sql` (the latter **enforces** the provenance gate with `NOT NULL source_ref`).
- **Graph** — `build_thin_slice.py` emits `graph.thin-slice.json`: **35 nodes, 48 edges, byte-identical across runs** (determinism), 25 `exact` / 23 `inferred`. The fixture is the node/edge + provenance oracle; match it to extractor output on the `owner#name` method prefix (the extractor's `IdGen` appends resolved param FQNs).
- **G5 provenance gate** — `eval/provenance_check.py`: **PASS**, 33/33 citable elements carry a resolvable `repo@sha:path:line` (2 exempt: external `ObjectMapper`, the `BuildMeta` marker).
- **G1 demo queries** — `query/demo_queries.py`: `find_callers`, `dependents@depth≤3` (10-node blast radius), `endpoints`, `DI wiring` (8 beans) — every row cited.
- **G9 eval-sanity** — `eval/eval_sanity.py`: **6/6 in-slice** understanding questions answerable (100% ≥ the locked 50% bar); 10/16 are out-of-slice (need full extraction / embeddings).
- **Loader** — `loader/load_age.py`: 35 node + 48 edge idempotent `MERGE` statements, 83 Cypher statements validated.

## What needs the proper environment (and why)

| Step | Needs | Why it couldn't run here |
|---|---|---|
| Compile + run the extractor on the **full** repo | JDK 17 + Maven | sandbox is JDK 11 JRE, no `javac`/`mvn`. `extractor/run.sh` harvests the classpath offline from the fat jar's `BOOT-INF/lib` (194 jars) — no `.m2` needed — then `mvn package` + run. |
| Load the full graph into AGE | Postgres 16 + AGE | no DB in sandbox. `load_age.py` + `schema/age_schema.sql` are ready. |
| **G7 AGE go/no-go** | the above | run `phase-0/age-benchmark/benchmark.py` on the loaded **real** graph; gate = `p95 < 1500 ms` at depth ≤ 3. |
| Build the Go services | Go ≥ 1.16 | sandbox Go is 1.13. `graphstore.go` is the production interface (compile-time proof both adapters satisfy it). |

## Build-order status (PHASE_1_PROMPT §4)

1. M0 schema — **done** (`schema/`)
2. GraphStore + store DDL — **done** (`graphstore/graphstore.go`, `schema/*.sql`)
3. Resolving extractor — **authored** (`extractor/`); compile + run on full repo pending JDK 17
4. Load + query — **thin slice done** (loader + demo queries); full load pending AGE
5. Provenance CI gate — **done & green** (`eval/provenance_check.py`)
6. Eval sanity — **thin slice done** (`eval/eval_sanity.py`)

## Exit-gate status (PHASE_1_PROMPT Part B)

Green on the thin slice: **G1** (cited queries), **G5** (provenance 100%), **G6** (Spring INJECTS deterministic), **G8** (one interface, two adapters), **G9** (structural subset ≥ 50%). Pending the full env: **G2/G3** (run the resolving extractor on all 604+481 files), **G4** (byte-diff two full runs — the emitter is already deterministic), **G7** (AGE latency on the real graph), **G10** (Terraform/Container Apps deploy). The thin slice de-risks every gate that does not require infra.

## Finish it

```bash
# in a JDK 17 + Maven box, with a clean checkout of the SHA:
extractor/run.sh /path/to/credit-routing-service@44b6b86 ./_work     # -> _work/graph.json + provenance PASS
python3 loader/load_age.py   # (point it at _work/graph.json) -> load.cypher
# provision dev AGE, load, then run the go/no-go:
psql "$CONN" -f schema/age_schema.sql && psql "$CONN" -f loader/load.cypher
python3 ../phase-0/age-benchmark/benchmark.py --iterations 50        # gate G7
```

Scope honesty (per the prompt): full semantic Q&A is **Phase 3**; SCIP precision + AOP/SOAP Tier-C edges are **Phase 2**; incremental freshness is **Phase 5**. Phase 1 proves the deterministic pipeline only.
