# M0 Graph Schema — Repository Intelligence Factory (Phase 1)

**Status:** M0 deliverable (settle before the extractor writes a node). Reconciles `baseline/TARGET_ARCHITECTURE.md` §6–§7 with `~/projects/aaraminds/Repo_Context_Platform_Architecture_v1.0.md`, specialized for the Spring/JAXB/Lombok target. Date 2026-06-03.

## Reconciliation verdict

The two docs are not competing graph designs. `Repo_Context_Platform_Architecture_v1.0.md` (human-consumer-first) **evicts** graph/vector indexing from its spine and parks it as an optional Phase-3 enhancement — so it contributes the **deterministic-extraction discipline, the provenance/freshness invariant, and the Spring fact set**, not labels. `TARGET_ARCHITECTURE.md` §6 is the binding graph design and supplies the labels, edge types, and confidence tiers. The `codebase-comprehension` skill's `graph-schema-and-ontology` reference is the authoritative property/identity/versioning model. **M0 = TARGET §6 labels + skill property model, reconciled, specialized for this repo.**

Reconciled disagreements:
- **Module vs Package** — TARGET collapses them; the skill separates them. *Kept both* (Maven multi-module is real; the `CONTAINS` chain needs both rungs).
- **Edge naming** `READS_FROM`/`WRITES_TO` (TARGET) vs `READS`/`WRITES` (skill) — *TARGET wins* (binding doc).
- **Generated code** — TARGET is silent; the skill flags it. This repo forces the issue (481 JAXB + Lombok). *Generated members are first-class `Generated*` nodes*, extracted post-annotation-processing.
- **Property collisions** `buildVersion` (skill) folded into `index_version` (TARGET); `confidence` is the tier enum.

## Node labels

| Label | Meaning | Provenance |
|---|---|---|
| `Repository` | one Git repo at a commit | deterministic |
| `Module` | Maven reactor module | deterministic |
| `Package` | Java package | deterministic |
| `File` | one source file | deterministic |
| `Type` | Class / Interface / Enum / Record (`kind`, `stereotype`) | deterministic |
| `Method` | method / constructor, overload-aware | deterministic |
| `Field` | declared field (carries DI target) | deterministic |
| `Endpoint` | HTTP route from a `@RestController` method | deterministic (annotation) |
| `DataStore` | persistence target (MongoDB collection here) | deterministic |
| `Aspect` | a `@Aspect` type | deterministic |
| `Generated` | JAXB / Lombok / proxy member (`origin`, `generator`) | generated |
| `BuildMeta` | one node per index pass (`index_version`, `repo_sha`, `complete`, `scip_tier`) | system |

Un-annotated component clustering (Design layer) is **inferred** and kept as a separate, visibly-tagged layer — never blended into the deterministic nodes above.

## Edge types and Phase-1 confidence tier

| Edge | Direction | Tier |
|---|---|---|
| `CONTAINS` | Repository→Module→Package→File→Type→Method; Type→Field | A `exact` |
| `DEFINES` | File→Type, File→Method | A `exact` |
| `IMPORTS` | File→Type (resolved) | A `exact` |
| `EXTENDS` | Type→Type | A `exact` |
| `IMPLEMENTS` | Type→Interface | A `exact` |
| `CALLS` | Method→Method (`call_site`) | A `exact` resolved · B `probable` dynamic dispatch |
| `INJECTS` | Type→Type (Spring DI: ctor param / `@Autowired`) | **C `inferred`** (DI never `exact`) |
| `EXPOSES` | Type(`@RestController`)→Endpoint | A `exact` (annotation) |
| `READS_FROM` / `WRITES_TO` | Method→DataStore | A `exact` from Spring Data · B otherwise |
| `ADVISES` | Aspect→Method (`weave_kind`) | **C `inferred`** (pointcut matches at runtime) |
| `CALLS_SERVICE` | Method→Endpoint (route/WSDL match) | **C `inferred`** |

## Property set — on EVERY node and edge

`id` (deterministic) · `name` · `kind` · `source_ref` = `repo@commit:path:line-range` · `provenance` ∈ {`deterministic`,`inferred`,`generated`,`external`} · `confidence` ∈ {`exact`,`probable`,`inferred`} · `evidence` ∈ {`ast`,`scip`,`annotation`,`config`,`route-match`} · `index_version`. Node extras: `stereotype` (Type), `http_method`/`path` (Endpoint). Edge extras: `call_site` (CALLS), `weave_kind` (ADVISES).

`external` nodes (library types like `ObjectMapper`) are exempt from `source_ref` (no in-repo declaration); the provenance gate treats them specially.

## Deterministic identity rule

`id = label-prefix + natural_key`, natural key being the **overload-aware FQN** — never a sequence number, never a line:
- `Type` → `type:{FQN}`
- `Method` → `method:{FQN}#{name}({orderedParamFQNs})`
- `Field` → `field:{FQN}#{name}`
- `Endpoint` → `endpoint:{httpMethod} {normalizedPath}`
- `DataStore` → `datastore:{collection}`
- `Repository` → `repo:{name}@{sha}`
- Edge → `edge:{TYPE}:{srcId}->{dstId}`

No line numbers / no `now()` in keys → same SHA yields identical IDs across rebuilds; `MERGE` on `id` is idempotent; build-to-build diffs are meaningful. One `UNIQUE(id)` per label. Schema evolves **additively only** — breaking the identity scheme forces a full rebuild and invalidates stored citations.

## Provenance & versioning

Two-layer, never blended: the parsed layer (`deterministic`/`exact`) and the inference layer (`inferred` — `INJECTS`, `ADVISES`, un-annotated clusters) coexist, tagged distinctly. Every element stamped `index_version`; a `BuildMeta` node marks a finished pass and readers **pin to a version** (commit-consistency). `scip_tier` distinguishes fast-AST-lane edges from full-SCIP-reconciled edges. The "100% traceability" guarantee is **self-citation completeness** — asserted in CI (`eval/provenance_check.py`) and enforced at the relational tier by `NOT NULL` on `source_ref` for non-`external` rows.
