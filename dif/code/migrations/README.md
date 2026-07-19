# DIF Migrations

This folder contains idempotent SQL migrations for `dif_meta`.

Rules:

- DIF migrations must not mutate RIF-owned schemas such as `rif` or `rif_meta`.
- Migrations must be safe to run more than once.
- Schema changes must support source-anchor round trips, audit events, usage events, and RIF compatibility status checks.
- Migration validation commands must be documented here and in `.github/copilot-instructions.md` once they exist.

## Current migrations

| File | Purpose |
|---|---|
| `001_dif_meta_initial_design.md` | Design source for the initial `dif_meta` schema. |
| `001_dif_meta_initial.sql` | Idempotent PostgreSQL migration for the initial P0 `dif_meta` schema. |
| `002_dif_meta_describes_edges.sql` | Additive idempotent migration enabling P1-02 `DESCRIBES` edges in `dif_meta.edges` (ADR-016 minimum fields, evidence-shape constraint, indexes). |

## Local validation pattern

When a local PostgreSQL server is available, validate the migration in a scratch database:

```bash
cd /Users/rb692q/projects/aaraminds-projects/dif
createdb dif_migration_check
psql -v ON_ERROR_STOP=1 -d dif_migration_check -f code/migrations/001_dif_meta_initial.sql
psql -v ON_ERROR_STOP=1 -d dif_migration_check -f code/migrations/002_dif_meta_describes_edges.sql
psql -v ON_ERROR_STOP=1 -d dif_migration_check -f code/migrations/001_dif_meta_initial.sql
psql -v ON_ERROR_STOP=1 -d dif_migration_check -f code/migrations/002_dif_meta_describes_edges.sql
dropdb dif_migration_check
```

This confirms the migration can run twice without mutating RIF-owned schemas.

The component-root runner provides the same apply path plus table inventory validation:

```bash
cd /Users/rb692q/projects/aaraminds-projects/dif
createdb dif_migration_check
cd code
DIF_DATABASE_URL='postgres://localhost:5432/dif_migration_check?sslmode=disable' go run ./cmd/dif-migrate apply
DIF_DATABASE_URL='postgres://localhost:5432/dif_migration_check?sslmode=disable' go run ./cmd/dif-migrate apply
DIF_DATABASE_URL='postgres://localhost:5432/dif_migration_check?sslmode=disable' go run ./cmd/dif-migrate check
cd ..
dropdb dif_migration_check
```
