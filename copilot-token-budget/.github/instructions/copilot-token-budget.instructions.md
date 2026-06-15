---
applyTo: "copilot-token-budget/**"
---

# Copilot Token Budget — Workspace Instructions

## Project purpose

Real-time GitHub Copilot CLI credit/token tracking for AT&T engineers. Local-first, zero-network.
Reads `~/.copilot/session-state/<uuid>/events.jsonl` — no GitHub API, no external calls.

## Current state (2026-06-13)

- Phase 0 (spike): COMPLETE — `events.jsonl` validated, billing fields confirmed
- Phase 1 (Go CLI): COMPLETE — `cmd/analyze` + `cmd/dashboard` working with real data
- Phase 2 (VS Code extension): COMPLETE (compiled) — runtime test pending
- Phase 3–4: Not started

## Code directories and build commands

### Phase 1 — Go CLI
```bash
cd phase-1/session-manager
go build ./...                          # build all
go run ./cmd/analyze ~/projects/...     # one-shot budget report
go run ./cmd/dashboard ~/projects/...   # live 10-second dashboard
```
Go module: `github.com/aaraminds/copilot-session-manager`
Zero external dependencies.

### Phase 2 — VS Code Extension
```bash
cd phase-2/vscode-extension
npm install --registry https://registry.npmjs.org   # first time only
npm run compile                                      # tsc -p ./
```
Open the `phase-2/vscode-extension` folder in VS Code and press F5 to test.

## Key files

| File | Purpose |
|---|---|
| `phase-1/session-manager/internal/session/reader.go` | Core data layer — reads JSONL |
| `phase-1/session-manager/internal/budget/tracker.go` | nanoAIU → credits → dollars |
| `phase-1/session-manager/internal/instructions/analyzer.go` | Instruction file audit |
| `phase-2/vscode-extension/src/extension.ts` | VS Code activation entry point |
| `phase-2/vscode-extension/src/ui/dashboardPanel.ts` | Full HTML dashboard webview |
| `design/ARCHITECTURE.md` | Component map + data flow |
| `design/adr/` | All architectural decisions |

## Architecture constraints

- **Local file read only** — no GitHub API calls (ADR-001)
- **Go zero deps** — `go.sum` is empty (ADR-002)
- **VS Code zero runtime deps** — only devDependencies (ADR-003)
- **Microsoft Teams** for alerts, not Slack (ADR-004)
- **JFrog Artifactory** for distribution, not Azure ACR (ADR-005)

## AT&T environment facts

- GitHub: corporate `github.com` with attuid (NOT `api.github.com/enterprises/att`)
- npm registry: `artifact.it.att.com/artifactory/api/npm/npm-all/` (requires auth)
  - Workaround: `npm install --registry https://registry.npmjs.org`
- Container/binary registry: JFrog Artifactory (ACR is anti-pattern)
- Communication: Microsoft Teams (no Slack)
- Copilot budget: 7,000 credits/month promo until 2026-09-01

## Billing units

```
1 credit = 1,000,000,000 nanoAIU
1 credit = $0.01
Claude Sonnet: 300 cr/M input, 1500 cr/M output
```

## Agent routing

| Task | Agent |
|---|---|
| Planning / roadmap / milestone updates | `aara-project-planner` |
| Architecture decisions / ADRs | `aara-project-architect` |
| Feature implementation (Go or TypeScript) | `aara-project-builder` |
| Debugging runtime issues | `aara-project-debugger` |
| Code review / PR review | `aara-project-reviewer` |
| MCP server (Phase 4) | `aara-mcp-server-builder` |
| Eval criteria / success metrics | `aara-ai-evaluation-engineer` |
