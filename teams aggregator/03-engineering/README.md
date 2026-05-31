# 03 · Engineering

Implementation artifacts — specs, decisions, runbooks, and (eventually) code.

## Suggested layout (create as needed)

- `specs/` — per-feature technical specs (Adaptive Card schema, AskAT&T client, change-notification handler, etc.)
- `adr/` — Architecture Decision Records (or keep them under `../02-architecture/adr/`)
- `runbooks/` — incident response, deployment, scheduled-digest backfill
- `api/` — OpenAPI specs, Adaptive Card JSON schemas
- `code/` — the Bot Framework app, or a `.gitkeep` if the code lives in a separate git repo (recommended)

## Conventions

- One spec per feature, prefixed with the PRD section it implements (e.g., `06.3-on-demand-summary.md`)
- ADR filenames: `NNNN-decision-title.md` with date in the front-matter
- All specs link back to the PRD section they implement, and forward to the diagram(s) they reference
