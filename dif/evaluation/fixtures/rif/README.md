# RIF Compatibility Fixture

**Purpose:** define the pinned fixture shape for DIF's RIF compatibility contract tests.  
**Primary ADR:** `design/adr/ADR-016-rif-compatibility-layer.md`  
**Primary gate:** P0 RIF compatibility gate  
**Status:** Initial executable fixture data and harness created.

---

## 1. Why this fixture exists

DIF must attach to existing RIF deployments without assuming optional RIF relational shadow tables are populated.

The local RIF review found this important pattern:

- RIF canonical graph data can be present in Postgres + Apache AGE schema `rif`.
- `rif_meta` can exist but contain empty or incomplete relational shadow tables.
- `rif_meta.class_nodes`, vector columns, and FTS columns may be absent.

This fixture must prove DIF can still detect and use a compatible RIF graph when AGE-backed data is available and optional `rif_meta` shadows are empty.

---

## 2. Observed local reference pattern

Use the local `rif_p19` database as the reference pattern, not as committed test data.

Observed through local `psql`:

| Area | Observation |
|---|---|
| Database name available locally | `rif_p19` |
| Exact pgAdmin labels requested | `rif_dev`, `rif_p19-local` were not available through local `psql` |
| RIF repository | `apm0045942` |
| Index state | one completed index run |
| Graph backend | Postgres + Apache AGE |
| Canonical code graph schema | `rif` |
| Populated labels | `File`, `Class`, `Method`, `SAME_FILE_CALLS`; additional labels may exist |
| `rif_meta.file_nodes` | exists but empty |
| `rif_meta.method_nodes` | exists but empty |
| `rif_meta.class_nodes` | absent |
| pgvector/FTS columns | absent |

Do not commit a dump of local/customer code or proprietary source content. The executable fixture should use synthetic data that reproduces this structural pattern.

---

## 3. Fixture variants

The fixture suite must cover these deployment states.

| Variant | Required status | Purpose |
|---|---|---|
| `no-rif` | `rif_not_deployed` | No RIF schema or compatibility surface exists. |
| `rif-incompatible` | `rif_incompatible` | RIF-like schema exists but required fields/labels are missing. |
| `age-only-compatible` | `rif_compatible` with `rif_shadow_empty` caveat | AGE-like code graph is populated while optional `rif_meta` shadows are empty or absent. This is the most important fixture. |
| `shadow-compatible` | `rif_compatible` | Populated compatibility view/table is available through `rif_meta` or equivalent. |
| `shadow-empty-no-age` | `rif_incompatible` | Shadows exist but are empty and no AGE/API fallback is available. |

---

## 4. Synthetic fixture entities

Use synthetic names. Do not copy customer or local RIF code names into committed fixture data.

Minimum entities:

| Entity | Kind | Required fields |
|---|---|---|
| file | `FILE` | `node_id`, `repo_id`, `qualified_name`, `source_ref`, `origin`, `confidence` |
| class | `CLASS` | `node_id`, `repo_id`, `qualified_name`, `simple_name`, `source_ref`, `origin`, `confidence` |
| method | `METHOD` | `node_id`, `repo_id`, `qualified_name`, `simple_name`, `source_ref`, `origin`, `confidence` |
| second method with duplicate simple name | `METHOD` | used to test ambiguity |
| relationship | `SAME_FILE_CALLS` or equivalent | used to prove relationship labels can exist but are not required for simple entity resolution |

Recommended synthetic repo:

```text
repo_id: demo-rif
sha: 1111111111111111111111111111111111111111
```

Recommended synthetic code entities:

```text
src/main/java/com/example/payments/PaymentService.java
com.example.payments.PaymentService
com.example.payments.PaymentService#authorize(com.example.payments.PaymentRequest)
com.example.billing.PaymentService#authorize(com.example.billing.PaymentRequest)
```

The duplicate `authorize` simple name is intentional; it verifies deterministic ambiguity handling.

---

## 5. Required compatibility fields

Every resolvable code entity must expose the ADR-016 contract:

| Field | Required in P0/P1 fixture | Notes |
|---|---:|---|
| `node_id` | Yes | Stable RIF-compatible node ID. |
| `repo_id` | Yes | Use `demo-rif`. |
| `kind` | Yes | `FILE`, `CLASS`, `METHOD`, etc. |
| `qualified_name` | Yes | Fully qualified name or repo-relative path. |
| `simple_name` | Required where applicable | Required for class/method tests; may be empty for file. |
| `source_ref` | Yes | Format: `demo-rif@1111111111111111111111111111111111111111:path:line`. |
| `origin` | Yes | Use `first_party` for primary fixture entities. |
| `confidence` | Yes | Use `exact` for primary fixture entities. |
| `code_version` | Optional in P0 | Required later for drift tests. |
| `content_hash` | Optional in P0 | Required later for drift tests. |

---

## 6. Node and edge ID expectations

The fixture must include expected hashes for:

```text
sha256(repoId + NUL + qualifiedName + NUL + kind)
sha256(fromNodeId + NUL + label + NUL + toNodeId)
```

The test implementation should compute expected IDs dynamically from fixture inputs rather than hard-coding opaque hashes in documentation.

Required checks:

1. Normal node ID uses NUL separators.
2. Normal edge ID uses NUL separators.
3. Legacy space-separated hashes are not accepted.

---

## 7. Contract test matrix

| Test | Fixture variant | Expected result |
|---|---|---|
| missing RIF schema | `no-rif` | `rif_not_deployed` |
| RIF schema without required fields | `rif-incompatible` | `rif_incompatible` |
| empty shadows with AGE fallback | `age-only-compatible` | `rif_compatible` plus empty-shadow caveat/status |
| populated compatibility shadow | `shadow-compatible` | `rif_compatible` |
| empty shadows without fallback | `shadow-empty-no-age` | `rif_incompatible` |
| exact method qualified-name lookup | `age-only-compatible` | one exact method result |
| exact file path lookup | `age-only-compatible` | one exact file result |
| exact class qualified-name lookup | `age-only-compatible` | one exact class result |
| simple-name ambiguous lookup | `age-only-compatible` | deterministic inferred result plus ambiguity caveat, or explicit ambiguous result set |
| unknown entity lookup | any compatible fixture | unresolved candidate; no minted code node |
| deterministic ordering | `age-only-compatible` | stable order across repeated runs |
| no mutation of RIF schemas | all writable tests | DIF migrations do not alter `rif` or `rif_meta` |

---

## 8. Expected resolver response shape

The exact implementation type can change, but contract tests should assert these semantics.

Compatible result:

```json
{
  "rif_status": "rif_compatible",
  "shadow_status": "rif_shadow_empty",
  "matches": [
    {
      "node_id": "<stable-rif-node-id>",
      "repo_id": "demo-rif",
      "kind": "METHOD",
      "qualified_name": "com.example.payments.PaymentService#authorize(com.example.payments.PaymentRequest)",
      "simple_name": "authorize",
      "source_ref": "demo-rif@1111111111111111111111111111111111111111:src/main/java/com/example/payments/PaymentService.java:42",
      "origin": "first_party",
      "confidence": "exact",
      "match_mode": "qualified_name",
      "caveats": []
    }
  ],
  "caveats": []
}
```

Missing/incompatible result:

```json
{
  "rif_status": "rif_incompatible",
  "matches": [],
  "caveats": ["Required RIF compatibility fields are unavailable."]
}
```

---

## 9. Implementation options

The first executable fixture can be implemented in one of three ways.

| Option | Pros | Cons | Recommendation |
|---|---|---|---|
| SQL fixture with AGE installed | Closest to RIF reality | Requires AGE in CI | Best integration test, not first unit test. |
| SQL fixture with compatibility view only | Easy Postgres-only CI | Does not exercise AGE parsing | Good first CI contract. |
| JSON fixture consumed by resolver tests | Fastest and deterministic | Less database-realistic | Best first unit-level spec. |

Recommended sequence:

1. Start with JSON fixture for deterministic resolver behavior.
2. Add Postgres compatibility-view fixture.
3. Add AGE-backed fixture when CI image supports AGE.

---

## 10. Executable fixture files

Current executable fixture files:

```text
evaluation/fixtures/rif/compat_entities.json
evaluation/fixtures/rif/expected_resolutions.json
evaluation/fixtures/rif/sql/age_only_compatible.sql
evaluation/fixtures/rif/sql/shadow_compatible.sql
evaluation/fixtures/rif/sql/rif_incompatible.sql
evaluation/rif_compatibility_checks.py
```

Run the fixture harness from the repository root:

```bash
python3 evaluation/rif_compatibility_checks.py
```

The harness validates:

1. all five fixture variants,
2. all ADR-016 RIF status outcomes,
3. exact qualified-name, file-path, and class lookups,
4. simple-name ambiguity behavior,
5. unknown entity unresolved behavior,
6. NUL-separated RIF node ID and edge ID hashing,
7. rejection of legacy space-separated hash compatibility.

The SQL files are plain PostgreSQL fixture sketches for future integration tests. They intentionally avoid requiring Apache AGE until the CI image supports AGE.
