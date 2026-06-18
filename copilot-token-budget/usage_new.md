# Copilot Token Budget — Usage Guide (Updated Layout)

This guide mirrors `USAGE.md` but is rewritten for the current component-based folder layout.
All metrics are local-first. Teams webhook is optional.

---

## Current folder layout

| Path | Purpose |
|---|---|
| `core/` | Main Go CLI: analyze, dashboard, statusline |
| `extension/` | VS Code extension (`.vsix`) |
| `alerting/` | Teams alert binary |
| `mcp/` | MCP server binary |
| `scripts/` | Helper scripts (install/remove/package/discovery) |
| `distr/v1.0.2/` | Latest macOS distribution bundle + zip |
| `docs/runbooks/` | End-user onboarding docs |

---

## Where metrics come from

### CLI sessions
- `~/.copilot/session-state/<uuid>/events.jsonl`
- `~/.copilot/session-state/<uuid>/workspace.yaml` (workspace metadata)

### IDE sessions
- VS Code user data stores (scanner looks for `chatSessions`, `transcripts`, `emptyWindowChatSessions`)
- macOS:
  - `~/Library/Application Support/Code/User/...`
  - `~/Library/Application Support/Code - Insiders/User/...`
- Windows:
  - `%APPDATA%\Code\User\...`
  - `%APPDATA%\Code - Insiders\User\...`
- Linux:
  - `~/.config/Code/User/...`
  - `~/.config/Code - Insiders/User/...`

Optional IDE-activity marker:
- `~/.copilot/vscode.session.metadata.cache.json`

---

## Prerequisites

| Need | Why |
|---|---|
| Go 1.21+ | Build `core` and `alerting` |
| Go 1.25+ | Build `mcp` |
| Node 18+ | Compile extension |
| Node 22+ | Package VSIX (`vsce`) |
| VS Code | Install and run extension |

---

## 1) Core CLI (analyze + dashboard + statusline)

```bash
cd core

# one-shot report
go run ./cmd/analyze [workspace-root]

# live dashboard (10s refresh)
go run ./cmd/dashboard [workspace-root]

# one-line status output
go run ./cmd/statusline
```

Build local binaries:

```bash
cd core
go build -o ~/bin/copilot-analyze ./cmd/analyze
go build -o ~/bin/copilot-dashboard ./cmd/dashboard
go build -o ~/bin/copilot-statusline ./cmd/statusline
```

---

## 2) VS Code extension

From source:

```bash
cd extension
npm install --registry https://registry.npmjs.org
npm run compile
```

Package + install VSIX:

```bash
cd extension
npm run package
code --install-extension copilot-token-budget-*.vsix --force
```

---

## 3) Teams alerts

```bash
cd alerting

# dry run
go run ./cmd/alert --dry-run <workspace-root>

# real alert
COPILOT_BUDGET_TEAMS_WEBHOOK="https://<your-webhook>" \
  go run ./cmd/alert <workspace-root>
```

---

## 4) MCP server

```bash
cd mcp
go build -ldflags "-X main.version=v1.0.2" -o ~/bin/copilot-budget-mcp ./cmd/mcp-server
~/bin/copilot-budget-mcp --version
```

Register in `.copilot/mcp.json` using an absolute path.

---

## 5) macOS out-of-box distribution (latest)

Use:

- Folder: `distr/v1.0.2/copilot-token-budget-macos-v1.0.2/`
- Zip: `distr/v1.0.2/copilot-token-budget-macos-v1.0.2.zip`

Install/remove from bundle:

```bash
cd distr/v1.0.2/copilot-token-budget-macos-v1.0.2
bash ./install.sh
bash ./remove.sh
```

The bundle contains:
- `binaries/darwin_amd64/` and `binaries/darwin_arm64/`
- `extension/copilot-token-budget-1.0.2.vsix`
- `manifest.json`, `LICENSE`, `USAGE.md`, `docs/runbooks/onboarding-runbook.md`

---

## 6) Build cross-platform release artifacts

```bash
goreleaser check
goreleaser build --snapshot --clean
goreleaser release --snapshot --clean --skip=publish
```

Extension package:

```bash
cd extension
npx @vscode/vsce package --no-dependencies
```

---

## 7) Quick troubleshooting

| Issue | Check |
|---|---|
| Dashboard shows zero | Ensure local session files exist under `~/.copilot/session-state/` and VS Code transcript paths |
| Extension installs but no data | Reload VS Code, then run `Copilot Budget: Refresh Now` |
| VSIX package fails | Upgrade to Node 22+ |
| MCP not found | Use absolute binary path in `.copilot/mcp.json` |
