# DIF — Copilot instructions

DIF (Document Intelligence Fabric) ingests uniformly readable document corpora
into a deterministic, source-anchored graph in the `dif_meta` Postgres schema,
co-located with a project's RIF code graph, and exposes evidence-only retrieval
via MCP. `action_plan.md` is the operating single source of truth;
`DECISIONS.md` and `design/adr/` hold accepted decisions.

## Hard rules

1. Never mutate RIF-owned schemas (`rif`, `rif_meta`). DIF writes to `dif_meta` only.
2. Never assume populated `rif_meta` shadows. All cross-graph behavior goes
   through `code/libs/rifcompat` (ADR-016): `rif_not_deployed`,
   `rif_incompatible`, `rif_shadow_empty`, and `rif_compatible` are explicit;
   the AGE fallback must work when shadows are empty.
3. Every retrieval result must carry a source anchor. No unanchored passages,
   no success-shaped empty results, no answers without source refs.
4. Corpus admission fails closed: rejected or unknown corpora return
   `corpus_not_admitted` (`code/libs/admission`).
5. MCP/API entry points require bearer auth, write audit events (with security
   dimensions) and separate non-PII usage events (`code/libs/mcpapi`,
   `code/libs/auditusage`).
6. Migrations are ordered, additive, idempotent SQL under `code/migrations/`;
   schema changes get a new numbered migration. The runner rejects RIF-owned DDL.
7. Extraction and graph emission are deterministic: same input, byte-identical
   NDJSON (`code/libs/extraction`, `code/libs/graphemit`).
8. `DESCRIBES` edges require resolver evidence (`code/libs/codeentities`
   resolver + `rifcompat`); ambiguous or unresolved candidates never create edges.
9. Do not log raw document text, credentials, tokens, or secret-like values
   (`code/libs/logging`).
10. Metrics are measured, not invented. Unverified numbers are marked `[VERIFY]`.

## Build and test commands

From the repository root:

```bash
python3 evaluation/run_p0.py          # full Golden P0 gate (Go + Python harnesses)
```

From `code/`:

```bash
go build ./...
go test ./...
go test ./libs/config ./libs/logging ./libs/requestctx ./libs/migrations \
  ./libs/admission ./libs/sourceanchors ./libs/ingestionruns ./libs/extraction \
  ./libs/graphemit ./libs/retrieval ./libs/embeddings ./libs/searchdocs \
  ./libs/mcpapi ./libs/auditusage ./libs/health ./libs/rifcompat ./libs/codeentities
go test ./... -run TestInitialMigrationIsDiscoverableFromComponentRoot
```

Single test: `go test ./libs/<pkg> -run <TestName>`. No lint runner exists yet.

## Layout

- `code/libs/` — component packages (see `code/libs/README.md`)
- `code/migrations/` — ordered `dif_meta` SQL migrations
- `evaluation/` — golden fixtures and executable P0 harnesses
- `design/adr/` — accepted ADRs; `tracking/` — phase gates and risk register
