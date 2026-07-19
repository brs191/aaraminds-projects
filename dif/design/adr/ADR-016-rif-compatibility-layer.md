# ADR-016: RIF Compatibility Layer

**Date:** 2026-07-08  
**Status:** Accepted for P0 design gate  
**Owners:** Engineering + Platform  
**Related decisions:** D-007, D-009  
**Related docs:** `DECISIONS.md`, `dif_prd.md`, `design-decisions.md`, `action_plan.md`

---

## 1. Context

DIF's core v1 differentiator is RIF+DIF federation: document blocks link to code entities so agents and users can ask:

- Which documents describe this code?
- Which code entities are described by this document section?
- Which documents may be stale after a code change?

The original D-007 direction correctly made federation core v1 scope. The RIF review refined the implementation assumption:

- RIF's canonical code graph is stored in Postgres + Apache AGE under schema `rif`.
- `rif_meta` is metadata plus optional relational shadow tables.
- Existing/local RIF databases may have a populated AGE graph but empty or absent `rif_meta` shadows.
- In local `rif_p19`, AGE labels such as `File`, `Class`, and `Method` are populated, while `rif_meta.file_nodes` and `rif_meta.method_nodes` are empty, `rif_meta.class_nodes` is absent, and pgvector/FTS columns are not present.

Therefore DIF must not directly rely on raw `rif_meta.file_nodes`, `rif_meta.method_nodes`, or `rif_meta.class_nodes` for cross-graph behavior. It needs a compatibility layer that abstracts how code entities are resolved.

---

## 2. Decision

DIF will use a **RIF compatibility layer** for all cross-graph features.

The compatibility layer is the only allowed boundary between DIF-owned data in `dif_meta` and RIF-owned code graph data. DIF implementation must not directly couple P1/P2 features to optional RIF shadow tables unless a deploy-time capability check proves those tables or views are present, populated, and compatible.

Allowed implementations:

1. **AGE-backed resolver/view** over schema `rif`.
2. **Populated `rif_meta` compatibility views/tables** when RIF provides them.
3. **RIF-provided API** exposing the same contract.

Initial P0/P1 default: **AGE-backed resolver/view first**, because existing local RIF evidence shows AGE is populated when `rif_meta` shadows may be empty.

---

## 3. Compatibility contract

The compatibility layer must expose these fields for resolvable code entities.

| Field | Required | Meaning |
|---|---:|---|
| `node_id` | Yes | Stable RIF code-entity ID. |
| `repo_id` | Yes | RIF repository identifier. |
| `kind` | Yes | Code entity kind, such as `FILE`, `CLASS`, `INTERFACE`, `ENUM`, `RECORD`, `METHOD`, `CONSTRUCTOR`, `FIELD`, `URL_ENDPOINT`, or `POINTCUT_EXPRESSION`. |
| `qualified_name` | Yes | Fully qualified code entity name or repo-relative file path. |
| `simple_name` | No | Short method/class/entity name when available. |
| `source_ref` | Yes | RIF source reference, usually `repo_id@sha:path:line`. |
| `origin` | Yes | RIF origin, usually `first_party` or `external_stub`. |
| `confidence` | Yes | RIF confidence, usually `exact` or `inferred`. |
| `code_version` | P2 | Code version evidence for drift. May be commit SHA, index version, or equivalent. |
| `content_hash` | P2 | Content/version hash or equivalent change evidence for `drift_report`. |

The contract is intentionally field-based. It does not require DIF to know whether the fields came from AGE, `rif_meta`, or a RIF API.

---

## 4. RIF capability statuses

The compatibility layer must return explicit statuses.

| Status | Meaning | Required behavior |
|---|---|---|
| `rif_not_deployed` | No RIF graph/schema is available in the database. | Cross-graph tools return explicit status and do not fabricate empty success. |
| `rif_incompatible` | RIF exists but does not expose required fields or labels. | Cross-graph tools return explicit status with missing capabilities. |
| `rif_shadow_empty` | Optional relational shadows are empty or absent but AGE/API may still be usable. | Resolver must try AGE/API path before failing. |
| `rif_compatible` | Required contract is available and tested. | Cross-graph tools may execute. |

These statuses are user-visible in MCP tool responses where relevant and are stored in `dif_meta.rif_compatibility_status` or equivalent.

---

## 5. Resolver behavior

The compatibility layer must support deterministic resolution modes.

| Mode | Match input | Confidence | Notes |
|---|---|---|---|
| Exact qualified-name | Fully qualified class/method/entity name | `exact` | Highest priority. |
| Exact source path | Repo-relative file path or source_ref path | `exact` | Required for file-level doc references. |
| Exact node ID | 64-character RIF node ID | `exact` | Required when upstream tools already know the code node. |
| Simple-name | Method/class simple name | `inferred` unless unique and policy says exact | Must surface ambiguity caveats. |
| Fuzzy/path contains | Partial path or partial qualified name | `inferred` | Must be deterministic and capped. |
| Unknown | No match | unresolved | Store candidate as unresolved; never mint dangling code nodes. |

Resolver output ordering must be deterministic:

1. exact node ID
2. exact qualified name
3. exact source path
4. unique simple name
5. deterministic inferred match
6. unresolved

Tie-breakers:

1. confidence
2. kind priority: `METHOD`, `CLASS`, `INTERFACE`, `RECORD`, `ENUM`, `FILE`, other
3. shortest qualified name
4. lexicographic `qualified_name`
5. lexicographic `node_id`

---

## 6. Node ID compatibility

DIF must match RIF node/edge ID semantics when referring to RIF code entities.

Normal RIF code node ID:

```text
sha256(repoId + NUL + qualifiedName + NUL + kind)
```

Normal RIF edge ID:

```text
sha256(fromNodeId + NUL + label + NUL + toNodeId)
```

Known synthetic RIF nodes use prefixed strings before hashing, such as:

- `APPLICATION_CONTEXT:`
- `POINTCUT_EXPR:`
- `URL_ENDPOINT:`

DIF must not import legacy `com.att.rif` or `github.com/att/rif` packages directly unless RIF first moves the shared primitive into a neutral AaraMinds module. Until then, DIF must implement compatibility tests against the exact algorithm.

---

## 7. Data ownership boundaries

DIF owns:

- `dif_meta`
- DIF document nodes and edges
- source anchors
- retrieval passages
- audit events
- usage events
- RIF compatibility status snapshots
- code-entity candidates
- `DESCRIBES` edges from document blocks to RIF code nodes

RIF owns:

- schema `rif`
- schema `rif_meta`
- AGE graph labels and edges
- RIF index runs and repository metadata
- RIF code entity source refs
- RIF embedding/FTS internals when present

DIF migrations must not mutate RIF-owned schemas. If a compatibility view is needed inside RIF-owned schema, it must be delivered by RIF or explicitly approved as a RIF migration, not hidden inside a DIF migration.

---

## 8. Minimum P0 implementation

P0 does not need full `DESCRIBES` resolution, but it must establish the compatibility gate.

P0 must provide:

1. Capability check for RIF presence.
2. Capability check for required AGE labels or compatibility views.
3. Capability check for optional `rif_meta` shadows without assuming they are populated.
4. Explicit status result:
   - `rif_not_deployed`
   - `rif_incompatible`
   - `rif_shadow_empty`
   - `rif_compatible`
5. Contract fixture based on the `rif_p19` pattern.
6. Contract test proving AGE-backed RIF can be compatible even when shadows are empty.

P0 may store unresolved code-entity candidates from documents, but it must not ship `docs_for_code`, `code_for_doc`, `drift_report`, or production `DESCRIBES` behavior until the P1 contract checks pass.

---

## 9. P1 federation requirements

P1 may implement `DESCRIBES`, `docs_for_code`, and `code_for_doc` only after:

1. ADR-016 is accepted.
2. RIF compatibility fixture passes.
3. The resolver can resolve representative `FILE`, `CLASS`, and `METHOD` entities.
4. Ambiguous or missing entities are stored as unresolved candidates.
5. MCP tool responses include explicit RIF status and caveats.
6. Resolution-rate metric is recorded per corpus.

`DESCRIBES` edge minimum fields:

| Field | Meaning |
|---|---|
| `from_doc_node_id` | DIF block/section node. |
| `to_code_node_id` | RIF code entity node ID. |
| `repo_id` | RIF repo ID. |
| `match_mode` | qualified-name, source-path, node-id, simple-name, fuzzy, unresolved. |
| `confidence` | exact or inferred. |
| `source_anchor_id` | DIF source anchor where the candidate appeared. |
| `code_source_ref` | RIF source ref. |
| `caveats` | Ambiguity, staleness, unresolved, external stub, or other limitations. |

---

## 10. P2 drift requirements

`drift_report` requires version/change evidence. P2 must not infer drift from a missing hash.

The compatibility layer must expose at least one reliable change signal:

- code node content hash
- indexed commit SHA + source_ref path/line
- RIF index version plus node update timestamp
- RIF-provided drift evidence API

Drift output must remain heuristic:

```text
code changed != document is wrong
```

The product must surface candidates for human review, not claim semantic contradiction.

---

## 11. Fixture plan

Create a RIF compatibility fixture under one of:

```text
evaluation/fixtures/rif/
code/testdata/rif/
```

The fixture must model:

- schema `rif` with populated AGE-like code entity data
- `File`, `Class`, `Method` examples
- at least one relationship edge, such as `SAME_FILE_CALLS`
- empty `rif_meta.file_nodes`
- empty `rif_meta.method_nodes`
- absent `rif_meta.class_nodes`
- no pgvector/FTS requirement

The fixture may be a SQL fixture, serialized JSON fixture, or documented local setup script. It must not depend on private customer data.

---

## 12. Required contract tests

Minimum tests:

1. Detect no RIF schema -> `rif_not_deployed`.
2. Detect RIF schema without required labels/fields -> `rif_incompatible`.
3. Detect empty optional shadows -> `rif_shadow_empty` but continue to AGE resolver.
4. Resolve exact method qualified name.
5. Resolve exact file path.
6. Resolve class qualified name.
7. Resolve simple name with deterministic caveat.
8. Return unresolved for unknown entity.
9. Return deterministic ordering for multiple matches.
10. Verify normal RIF node ID hash algorithm.
11. Verify normal RIF edge ID hash algorithm.
12. Verify no DIF migration mutates RIF-owned schemas.

---

## 13. MCP behavior

Cross-graph MCP tools must never return empty success when RIF is missing or incompatible.

Required response pattern:

```json
{
  "rif_status": "rif_compatible",
  "results": [],
  "caveats": []
}
```

For incompatible/missing RIF:

```json
{
  "rif_status": "rif_incompatible",
  "results": [],
  "caveats": ["Required RIF compatibility fields are unavailable."]
}
```

The exact schema can evolve, but the semantics cannot: missing/incompatible RIF must be explicit.

---

## 14. Consequences

Positive:

- DIF can attach to existing RIF deployments safely.
- Existing AGE-backed RIF graphs are usable even when relational shadows are empty.
- Federation failures become explicit and diagnosable.
- `DESCRIBES` resolution rate becomes a meaningful quality metric.

Trade-offs:

- P0 must include compatibility gate work before federation implementation.
- P1 cross-graph tools cannot be built as direct joins against optional shadows.
- Drift reporting requires additional version/content evidence.

---

## 15. Open questions

1. Should RIF expose a stable compatibility view such as `rif_meta.compat_code_entities`, or should DIF own an AGE-backed resolver?
2. Should capability status be recomputed on every tool call, cached per deployment, or refreshed per index version?
3. What is the canonical code content hash for drift if RIF currently stores source refs but not content hashes?
4. How should simple-name ambiguity thresholds be tuned for large Java repos?
5. Should unresolved code-entity candidates be upgraded automatically on the next RIF index, or only on the next DIF re-index?

---

## 16. Acceptance criteria

ADR-016 is accepted when:

- The required compatibility fields are documented.
- The explicit RIF statuses are documented.
- AGE-backed resolution is the default for existing deployments.
- Optional `rif_meta` shadows are treated as an optimization, not a dependency.
- P0, P1, and P2 requirements are separated.
- Fixture and contract test requirements are defined.
- MCP missing/incompatible behavior is explicit.
- The plan blocks cross-graph P1 tools until the compatibility fixture passes.

