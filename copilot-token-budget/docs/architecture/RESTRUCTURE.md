# Repository Restructure — Phase-based → Domain (DDD) Layout

**Date:** 2026-06-17
**Status:** Applied
**Scope:** Directory + module layout only. No behavior, no public CLI/MCP/extension surface changed.

---

## Why

The repository was organized by **build phase** (`phase-0` … `phase-4`), which was useful
while the tool was being built sequentially but is the wrong axis for an operated product:

- A new engineer cannot tell what `phase-3` *does* without reading it.
- The phase number encodes *when* code was written, not *what bounded context* it belongs to.
- Build/release tooling, scripts, status docs, and ADRs were scattered at the repo root.

The repository is now organized by **domain / bounded context**. The phase record is preserved
verbatim under `docs/history/` — nothing was deleted.

---

## Target layout

```
copilot-token-budget/
├── core/        Go module — domain engine (ingestion, budgeting, analytics,
│                instructions, presentation) + the CLI surface (cmd/)
├── alerting/    Go module — Teams alerting + forecasting bounded context
├── mcp/         Go module — MCP server surface (Go 1.25)
├── extension/   TypeScript — VS Code extension surface
├── scripts/     install / remove / run + discovery/ (IDE-source probes)
├── docs/
│   ├── architecture/  ARCHITECTURE.md, architecture-diagram.svg, adr/, this file
│   ├── product/       PRD.md
│   ├── research/      dashboard-feature-analysis.md
│   ├── runbooks/      onboarding-runbook.md
│   └── history/       archived phase record (read-only)
├── go.work      workspace stitching core + alerting + mcp
├── .goreleaser.yaml, .github/, .copilot/, README.md, USAGE.md, LICENSE
```

The five domains the team reasons about map onto the tree as follows. Ingestion, budgeting,
analytics and instructions are **packages inside the `core` module** (one module, because Go's
`internal/` visibility and the shared domain types make a single library the right grain);
surfaces and distribution are **cross-cutting**:

| Domain        | Where it lives |
|---------------|----------------|
| Ingestion     | `core/internal/session` (CLI reader, dedup, IDE collector stub) |
| Budgeting     | `core/internal/budget`, `core/internal/pricing` |
| Analytics     | `core/internal/analytics`, `alerting/internal/forecast` |
| Instructions  | `core/internal/instructions` |
| Surfaces      | `core/cmd/*` (CLI), `extension/` (VS Code), `alerting/cmd/alert` (Teams), `mcp/` (MCP) |
| Distribution  | `.goreleaser.yaml`, `.github/workflows/`, `scripts/` |

---

## Old → new mapping

| Old path | New path |
|---|---|
| `phase-1/session-manager/` | `core/` |
| `phase-3/` | `alerting/` |
| `phase-4/` | `mcp/` |
| `phase-2/vscode-extension/` | `extension/` |
| `install_vscode_extn.sh`, `remove_vscode_extn.sh` | `scripts/` |
| `phase-1/run.sh` | `scripts/run.sh` |
| `phase-0/discover-ide-usage.{sh,ps1}` | `scripts/discovery/` |
| `design/ARCHITECTURE.md`, `design/adr/`, `design/*.svg` | `docs/architecture/` |
| `product/PRD.md` | `docs/product/PRD.md` |
| `research/dashboard-feature-analysis.md` | `docs/research/` |
| `docs/onboarding-runbook.md` | `docs/runbooks/onboarding-runbook.md` |
| `IMPLEMENTATION_PLAYBOOK.md`, `STATUS.md`, `BUILD_PLAN.md` | `docs/history/` |
| `tracking/TRACKING.md` | `docs/history/TRACKING.md` |
| `evaluation/*` | `docs/history/evaluation/` |
| `phase-0/findings/` | `docs/history/discovery/findings/` |

All moves were done with `git mv`, so per-file history is preserved.

## Go module path changes

The legacy module name `copilot-session-manager` (a phase-1 artifact) was renamed to match the
repository. The prefix-nesting that lets the satellite modules import `core`'s `internal/`
packages is preserved (`alerting` and `mcp` remain children of the `core` module path).

| Old module path | New module path |
|---|---|
| `github.com/aaraminds/copilot-session-manager` | `github.com/aaraminds/copilot-token-budget` |
| `github.com/aaraminds/copilot-session-manager/phase3` | `github.com/aaraminds/copilot-token-budget/alerting` |
| `github.com/aaraminds/copilot-session-manager/phase4` | `github.com/aaraminds/copilot-token-budget/mcp` |

`replace` directives in `alerting/go.mod` and `mcp/go.mod` now point at `../core`. A new
`go.work` at the repo root stitches the three modules for local development and CI.

---

## Decisions and rationale

**`core`'s `internal/` packages were *not* renamed or re-nested.** They already carry
single-responsibility, domain-aligned names (`session`, `budget`, `pricing`, `analytics`,
`instructions`, `render`, `platform`). A flat `internal/<pkg>` layout is idiomatic Go; forcing a
`domain/`-nested tree would have churned ~70 import sites and every cross-module importer for no
compile-time or readability gain. The domain grouping is documented in the table above and in the
README tree instead.

**Three Go modules were kept (not merged into one).** `mcp` must pin Go 1.25 (a hard requirement
of `modelcontextprotocol/go-sdk v1.6.1`); `alerting` and `core` stay on Go 1.21+. Merging would
force 1.25 on everything and couple unrelated dependency sets.

**Phase history archived, not rewritten.** `docs/history/` keeps the playbook, status, tracking,
build plan, acceptance gates, and discovery findings exactly as written. They intentionally still
speak in phase terms — they are the record of how the tool was built.

---

## Verification

After the move, all gates were re-run green (matching the pre-move baseline):

- `go build ./...`, `go vet ./...`, `go test ./...` on `core`, `alerting`, `mcp` (mcp on Go 1.25)
- the `mcp` integration test (builds binaries by import path → confirms cross-module resolution)
- `tsc` strict on `extension`
- `goreleaser check` + `goreleaser build --snapshot`
- `actionlint` on `.github/workflows/`

## Rollback

The change is pure file moves + path string edits, staged via `git mv`. To revert before commit:
`git reset --hard HEAD` (scoped to this project's paths). A pre-restructure snapshot also exists at
`../Copilot-token-budget-with-phasesv0.9.zip`.
