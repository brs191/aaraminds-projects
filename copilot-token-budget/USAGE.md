# Copilot Token Budget — Usage Guide

How to run the outcome of **each phase**. Everything here is local-first and zero-network
(the only outbound call is the optional Teams webhook in Phase 3).

**Phase 6.2 update (2026-06-17):** The tool now captures **both** GitHub Copilot CLI and VS Code IDE Chat sessions locally. The same `analyze` / `dashboard` commands show per-source breakdown (CLI with token costs, IDE sessions with metadata). No changes needed — if you have active IDE Chat sessions, they appear automatically.

> **Just want to install and use it?** End users (any OS) should follow
> [`docs/runbooks/onboarding-runbook.md`](docs/runbooks/onboarding-runbook.md) — the ≤5-minute install guide with full
> macOS / Linux / Windows steps. **This file** is the developer/source-build reference (run each phase
> from the repo). Commands default to **bash** (macOS/Linux); **Windows** variants (PowerShell) are
> given inline where they differ. On Windows, `run.sh` requires **Git Bash or WSL**; the IDE-discovery
> step has a native PowerShell port (`scripts/discovery/discover-ide-usage.ps1`).

> **A note on two "workspace" concepts.** Credit/session data is *always* read from
> `~/.copilot/session-state/` regardless of arguments. The optional `[workspace-root]` argument on
> `analyze`/`dashboard`/`alert` only tells the tool where to scan `.github/instructions/**` for the
> **instruction-file overhead audit**. If you omit it, the current directory is used.

---

## Supported platforms

**macOS, Linux (incl. Ubuntu), and Windows.** The core is pure Go stdlib with cross-platform path
helpers (`os.UserHomeDir`, `os.UserConfigDir`, `filepath.Join`) and the TypeScript extension uses
`path`/`os` only — there are no hardcoded OS paths. Verified building (`linux/amd64`) and running on
**Ubuntu 20.04** (analyze, statusline, all v1.1 sections). Config lives at `~/.config/copilot-token-budget/`
on Linux/macOS and `%AppData%\copilot-token-budget\` on Windows; session data is read from
`~/.copilot/session-state/` (or `%USERPROFILE%\.copilot\session-state\`) on every OS.

> The IDE-discovery script (`scripts/discovery/discover-ide-usage.sh`) is OS-aware: it scans
> `~/Library/Application Support/Code*` on macOS and `~/.config/Code*` + `~/.vscode-server` on Linux.
> Phase 5 distribution (GoReleaser) builds for `darwin/amd64+arm64`, `linux/amd64+arm64`, and `windows/amd64`.

## Prerequisites

| Need | Why |
|---|---|
| **Go 1.21+** | builds core and alerting |
| **Go 1.25+** | builds mcp (the MCP server) — hard requirement of `modelcontextprotocol/go-sdk v1.6.1`. `GOTOOLCHAIN=auto` (default) will auto-fetch it. |
| **Node 18+** | to `npm run compile` / F5 the extension. **Node 22+** is required to `npm run package` the `.vsix` (`@vscode/vsce` 3.x). |
| GitHub Copilot CLI that has produced sessions under `~/.copilot/session-state/` (`%USERPROFILE%\.copilot\session-state\` on Windows) | there is nothing to report without real session data |
| Windows only: **Git Bash or WSL** | to run the `.sh` helper scripts (`run.sh`, `discover-ide-usage.sh`) |

Repo root referenced below as `copilot-token-budget/`. **Windows** users: in command blocks, use the
PowerShell variant where shown; `~` maps to `%USERPROFILE%`, and built binaries take a `.exe` suffix.

---

## Phase 0 — Data-source validation (spike)

**Outcome:** confirmation that `~/.copilot/session-state/` carries the billing fields. Findings live in `docs/history/discovery/findings/FINDINGS_MEMO.md` + `sample_event.json` (no command needed to re-run).

**IDE discovery spike (Phase 6 prerequisite, Step 6.0)** — run on your Mac to map VS Code IDE Copilot usage:

```bash
bash scripts/discovery/discover-ide-usage.sh > ide-usage-report.txt
cat ide-usage-report.txt   # read-only, zero-network, redacts PII
```

**Windows (native PowerShell):**

```powershell
powershell -ExecutionPolicy Bypass -File scripts\discovery\discover-ide-usage.ps1 > ide-usage-report.txt
Get-Content ide-usage-report.txt   # read-only, zero-network, redacts PII
```

It scans `%USERPROFILE%\.copilot` and `%APPDATA%\Code*` (incl. Insiders / VSCodium). Under **WSL/Git Bash**
you can run the `.sh` version instead, which inspects the Linux-side VS Code dirs.

---

## Phase 1 — Go CLI (analyze + dashboard)

```bash
cd core

# One-shot report (active sessions, history, budget, instruction audit, usage trend, top consumers)
go run ./cmd/analyze [workspace-root]

# Live dashboard — refreshes every 10s, updates the WezTerm tab badge, Ctrl+C to exit
go run ./cmd/dashboard [workspace-root]
```

**Guided launcher** (preflight → build → one-shot report → dashboard):

```bash
./scripts/run.sh [workspace-root]     # defaults to the aaraminds-projects workspace if omitted
```

Build static binaries instead of `go run`:

```bash
# macOS / Linux
cd core
go build -o ~/bin/copilot-analyze    ./cmd/analyze
go build -o ~/bin/copilot-dashboard  ./cmd/dashboard
go build -o ~/bin/copilot-statusline ./cmd/statusline
```

```powershell
# Windows (PowerShell) — note the .exe suffix and %USERPROFILE%\bin
cd core
go build -o "$env:USERPROFILE\bin\copilot-analyze.exe"    ./cmd/analyze
go build -o "$env:USERPROFILE\bin\copilot-dashboard.exe"  ./cmd/dashboard
go build -o "$env:USERPROFILE\bin\copilot-statusline.exe" ./cmd/statusline
# add %USERPROFILE%\bin to PATH once:  setx PATH "$($env:PATH);$env:USERPROFILE\bin"
```

> On **Windows**, `./scripts/run.sh` only works under Git Bash/WSL. Otherwise run `go run ./cmd/analyze`
> and `go run ./cmd/dashboard` directly (PowerShell), which is what `run.sh` does.

---

## Phase 2 — VS Code extension

**Run from source (F5):**

```bash
cd extension
npm install --registry https://registry.npmjs.org   # first time (AT&T proxy workaround)
npm run compile
# In VS Code: File → Open Folder → extension, then press F5 → Extension Development Host
```

**Build + install a `.vsix`:**

```bash
cd extension
npm run package                       # produces copilot-token-budget-*.vsix (needs Node 22+ for vsce)
code --install-extension copilot-token-budget-*.vsix
```

**Install / uninstall via helper scripts** (repo root; bash — use Git Bash/WSL on Windows):

```bash
./scripts/install_vscode_extn.sh                         # build from source + install (Node 22+ to package)
./scripts/install_vscode_extn.sh --vsix /path/file.vsix  # install a prebuilt .vsix (skips build)
./scripts/install_vscode_extn.sh --help                  # all options + a feature summary

./scripts/remove_vscode_extn.sh                          # uninstall the extension (no-op if absent)
./scripts/remove_vscode_extn.sh --purge-config --remove-binaries --yes   # full teardown, no prompts
```

`remove_vscode_extn.sh` never touches your Copilot session data (`~/.copilot/session-state/`);
`--purge-config` removes the local `pricing.json` + Teams-alert `state.json`, and `--remove-binaries`
removes the five `~/bin` CLIs. To unregister the MCP server, delete the `copilot-token-budget` entry
from your workspace `.copilot/mcp.json` by hand.

**Commands** (Command Palette): `Copilot Budget: Show Dashboard`, `: Refresh`, `: Open Settings`,
`: Export Usage` (`copilotBudget.exportUsage` — JSON/CSV save dialog).

**Settings** (`copilotBudget.*`): `monthlyAllowance`, `workspacePath`, `refreshIntervalSec`,
`teamsWebhookUrl`, `alertThresholdWarn`, `alertThresholdCrit`, `alertBinaryPath`, `pricingPath`.

---

## Phase 3 — Teams alerts + forecasting

The webhook URL comes **only** from the `COPILOT_BUDGET_TEAMS_WEBHOOK` env var (never a flag).
The `<workspace-root>` argument is **required**.

```bash
cd alerting

# Safe preview — builds the Adaptive Card JSON, makes NO network call
go run ./cmd/alert --dry-run [--allowance 7000] <workspace-root>

# Real alert — POSTs to Teams only if a threshold (60% warn / 90% crit) is crossed
#   and that threshold hasn't already fired today (dedup, UTC, per-day)
COPILOT_BUDGET_TEAMS_WEBHOOK="https://<your-teams-webhook>" \
  go run ./cmd/alert <workspace-root>
```

```powershell
# Windows (PowerShell) — set the env var for this process, then run
$env:COPILOT_BUDGET_TEAMS_WEBHOOK = "https://<your-teams-webhook>"
go run ./cmd/alert <workspace-root>
```

> **Teams webhook:** the legacy O365 "Incoming Webhook" connector is retired (~May 2026). Create a
> **Power Automate "Workflows"** webhook ("Post to a channel when a webhook request is received") — our
> Adaptive Card payload works with it. Steps are in `docs/runbooks/onboarding-runbook.md`.

Exit codes: `0` = no alert needed / already sent today · `1` = alert fired (or dry-run printed) · `2` = error.

**Run it on a schedule:**

```bash
# macOS / Linux — cron, every 30 min during work hours
*/30 9-18 * * 1-5  COPILOT_BUDGET_TEAMS_WEBHOOK="https://..." /path/to/copilot-alert /path/to/workspace
```

```powershell
# Windows — Task Scheduler: a wrapper .ps1 that sets the env var then runs the binary,
# scheduled via schtasks. Example wrapper (alert.ps1):
#   $env:COPILOT_BUDGET_TEAMS_WEBHOOK = "https://..."
#   & "$env:USERPROFILE\bin\copilot-alert.exe" "C:\path\to\workspace"
schtasks /create /tn "CopilotBudgetAlert" /tr "powershell -File C:\path\to\alert.ps1" /sc minute /mo 30
```

> The alert engine is also invoked automatically by the VS Code extension's refresh loop when
> `copilotBudget.teamsWebhookUrl` is set and the `copilot-alert` binary is found
> (`copilotBudget.alertBinaryPath` or `~/bin/copilot-alert`).

---

## Phase 4 — MCP server (six tools)

```bash
# macOS / Linux
cd mcp
go build -ldflags "-X main.version=v0.1.0" -o ~/bin/copilot-budget-mcp ./cmd/mcp-server
```

```powershell
# Windows (PowerShell)
cd mcp
go build -ldflags "-X main.version=v0.1.0" -o "$env:USERPROFILE\bin\copilot-budget-mcp.exe" ./cmd/mcp-server
```

Register it in `.copilot/mcp.json` (already scaffolded at the repo root) — **use an absolute path**;
`~` / `%USERPROFILE%` are NOT expanded by the process launcher, so write the full literal path:

```jsonc
// macOS / Linux
{ "mcpServers": { "copilot-token-budget": {
  "command": "/Users/<you>/bin/copilot-budget-mcp", "args": [], "env": {} } } }
```

```jsonc
// Windows — full path + .exe, escaped backslashes
{ "mcpServers": { "copilot-token-budget": {
  "command": "C:\\Users\\<you>\\bin\\copilot-budget-mcp.exe", "args": [], "env": {} } } }
```

Then, in a Copilot CLI session, the model can call: `get_budget_status`, `get_sessions`,
`get_instruction_overhead`, `get_model_costs`, `get_usage_timeseries` (daily/weekly/monthly),
`get_top_consumers` (top-N sessions/models/projects). All read local files only.

Smoke-test the binary directly:

```bash
~/bin/copilot-budget-mcp --version
```

---

## Phase 5 — Distribution + onboarding · CONFIG-COMPLETE (live publish pending)

The release config is built and locally validated; the live publish path (JFrog upload + GitHub
Release on a real tag) is pending JFrog provisioning + the first tag. See `docs/history/evaluation/PHASE5_ACCEPTANCE.md`.

**End-user install (any OS):** follow [`docs/runbooks/onboarding-runbook.md`](docs/runbooks/onboarding-runbook.md) — the
≤5-minute guide with macOS / Linux / Windows steps (download from Artifactory, install the `.vsix`,
register MCP, configure Teams).

**Build the release artifacts locally** (requires GoReleaser v2):

```bash
goreleaser check                       # validate .goreleaser.yaml
goreleaser build --snapshot --clean    # cross-compile 25 binaries (5 × 5 platforms) into dist/
goreleaser release --snapshot --clean --skip=publish   # also produce archives + checksums.txt
```

Targets: `darwin/amd64`, `darwin/arm64`, `linux/amd64`, `linux/arm64`, `windows/amd64`
(windows/arm64 intentionally excluded). Archives bundle README/USAGE/LICENSE/onboarding-runbook.

**Package the extension `.vsix`** (needs Node 22):

```bash
cd extension
npm install --registry https://registry.npmjs.org
npx @vscode/vsce package --no-dependencies
```

**CI/CD:** `.github/workflows/ci.yml` (build/test/lint on push/PR) and `.github/workflows/release.yml`
(tag `v*.*.*` → GoReleaser + vsce + JFrog OIDC upload + GitHub Release). Required repo Variables
(`JF_URL`, `JF_BINARY_REPO`, `JF_VSIX_REPO`) + OIDC setup are documented in `.github/workflows/README.md`.

---

## How to Distribute to Your Team

**Goal:** Build distribution artifacts (binaries + extension) and share them with your team for installation.

### Quick Start (5 minutes)

**macOS out-of-the-box bundle (recommended):**

1. Download `copilot-token-budget-macos-<version>.zip`.
2. Unzip it.
3. Run `./install.sh`.
4. Run `./launch-caveman-demo.sh` if you want the Caveman walkthrough.

**To create the zip file for v1.0.0:**

```bash
# Build the release artifacts first
goreleaser check && goreleaser build --snapshot --clean

# Package the macOS bundle as v1.0.0
bash ./scripts/package_macos_oob.sh \
  --artifact-dir dist \
  --vsix dist-vsix/copilot-token-budget-v1.0.0.vsix \
  --version v1.0.0 \
  --output-dir distr/v1.0.0

# Result:
# distr/v1.0.0/copilot-token-budget-macos-v1.0.0.zip
# distr/v1.0.0/copilot-token-budget-macos-v1.0.0.zip.sha256
```

**One-command build + package everything:**

```bash
# Build all release artifacts (binaries for macOS/Linux/Windows + extension .vsix)
goreleaser check && \
goreleaser build --snapshot --clean && \
cd extension && npm install --registry https://registry.npmjs.org && \
npx @vscode/vsce package --no-dependencies && \
cd .. && \
echo "✅ Distribution artifacts ready in dist/ and extension/"
```

**Result:** All binaries + `.vsix` in `dist/` and `extension/` ready to distribute.

---

### Distribution Methods

#### **Option 1: Local File Share (Quickest for Small Teams)**

**Step 1: Build locally**
```bash
cd /path/to/copilot-token-budget
goreleaser build --snapshot --clean    # creates dist/copilot-token-budget_*/
cd extension && npm install --registry https://registry.npmjs.org
npx @vscode/vsce package --no-dependencies   # creates copilot-token-budget-*.vsix
```

**Step 2: Share files with your team**

Create a shared folder (e.g., AT&T Sharepoint, Teams Files, or internal file server):

```
shared/copilot-token-budget-v1.0.0/
├── README.md                           (copy from repo root)
├── USAGE.md                            (copy from repo root)
├── docs/runbooks/onboarding-runbook.md (detailed install steps)
├── binaries/
│   ├── copilot-analyze-darwin-amd64              (macOS Intel)
│   ├── copilot-analyze-darwin-arm64              (macOS Apple Silicon)
│   ├── copilot-analyze-linux-amd64               (Linux x64)
│   ├── copilot-analyze-windows-amd64.exe         (Windows)
│   ├── copilot-dashboard-*                       (same platforms)
│   ├── copilot-statusline-*                      (same platforms)
│   ├── copilot-alert-*                           (same platforms)
│   └── copilot-budget-mcp-*                      (same platforms)
├── extension/
│   ├── copilot-token-budget-1.0.0.vsix           (VS Code extension)
│   └── copilot-token-budget-1.0.0.vsix.sha256sum (integrity check)
└── checksums.txt                       (SHA256 hashes for all binaries)
```

**Step 3: Team installs**

Each team member follows [`docs/runbooks/onboarding-runbook.md`](docs/runbooks/onboarding-runbook.md):
- Download binaries for their OS from `shared/binaries/`
- Install extension `.vsix` via VS Code
- Register MCP server
- Configure Teams webhook (optional)

---

#### **Option 2: GitHub Releases (Best for Public / Open-Source Teams)**

**Step 1: Tag and push**
```bash
# On main branch, create a version tag
git tag v1.0.0
git push origin v1.0.0
```

**Step 2: GitHub Actions builds + releases**

The `.github/workflows/release.yml` workflow automatically:
1. Detects the `v*.*.*` tag
2. Builds 25 binaries (5 platforms × 5 binaries)
3. Packages extension `.vsix`
4. Creates a GitHub Release with all artifacts
5. (If JFrog provisioned) Uploads to Artifactory

**Step 3: Team downloads**

Navigate to `https://github.com/<your-org>/copilot-token-budget/releases/v1.0.0` and download:
- Platform-specific binary (or use package installer if built)
- Extension `.vsix`
- README + onboarding runbook

---

#### **Option 3: JFrog Artifactory (Recommended for AT&T)**

**Prerequisites:**
- JFrog instance URL (e.g., `artifactory.company.com`)
- Repository created for binaries (`copilot-token-budget-binaries`) and extension (`copilot-token-budget-vsix`)
- OIDC or API token configured (see `.github/workflows/README.md`)

**Step 1: Configure GitHub Actions**

Set repo Variables in `.github/settings/variables`:
- `JF_URL`: `https://artifactory.company.com`
- `JF_BINARY_REPO`: `copilot-token-budget-binaries`
- `JF_VSIX_REPO`: `copilot-token-budget-vsix`

(OIDC setup in `.github/workflows/README.md`)

**Step 2: Tag and push (same as GitHub Releases)**
```bash
git tag v1.0.0
git push origin v1.0.0
```

**Step 3: Workflow publishes**

`.github/workflows/release.yml` automatically uploads to JFrog:
- `https://artifactory.company.com/artifactory/copilot-token-budget-binaries/v1.0.0/...`
- `https://artifactory.company.com/artifactory/copilot-token-budget-vsix/v1.0.0/...`

**Step 4: Share Artifactory download links with team**

```
Install Guide for Copilot Token Budget v1.0.0
─────────────────────────────────────────────

1. Download for your OS:
   macOS (Intel):     https://artifactory.company.com/.../copilot-analyze-darwin-amd64
   macOS (Apple Si):  https://artifactory.company.com/.../copilot-analyze-darwin-arm64
   Linux x64:         https://artifactory.company.com/.../copilot-analyze-linux-amd64
   Windows:           https://artifactory.company.com/.../copilot-analyze-windows-amd64.exe

2. Follow: https://artifactory.company.com/.../onboarding-runbook.md
```

---

### Step-by-Step: Build & Share Artifacts (All Options)

#### **Step 1: Prepare repo**
```bash
cd /path/to/copilot-token-budget
git status                                 # ensure clean repo
go version                                 # check Go 1.25+
goreleaser --version                       # check GoReleaser v2
node --version                             # check Node 18+
```

#### **Step 2: Build binaries**
```bash
cd /path/to/copilot-token-budget

# Validate config
goreleaser check

# Build snapshot (no publish) — creates dist/
goreleaser build --snapshot --clean

# Verify output
ls -lh dist/copilot-token-budget_*/bin/
# Output: copilot-analyze, copilot-dashboard, copilot-statusline, copilot-alert, copilot-budget-mcp
```

#### **Step 3: Package extension**
```bash
cd extension

# Install dependencies
npm install --registry https://registry.npmjs.org

# Package .vsix
npx @vscode/vsce package --no-dependencies

# Verify
ls -lh copilot-token-budget-*.vsix
```

#### **Step 4a (Local Share): Copy to shared folder**
```bash
# Create distribution folder
mkdir -p /Volumes/shared/copilot-token-budget-v1.0.0/{binaries,extension,docs}

# Copy binaries
find dist/ -name "copilot-*" -type f -exec cp {} /Volumes/shared/copilot-token-budget-v1.0.0/binaries/ \;

# Copy extension
cp extension/copilot-token-budget-*.vsix /Volumes/shared/copilot-token-budget-v1.0.0/extension/

# Copy docs
cp USAGE.md README.md docs/runbooks/onboarding-runbook.md /Volumes/shared/copilot-token-budget-v1.0.0/docs/

# Generate checksums
cd /Volumes/shared/copilot-token-budget-v1.0.0
sha256sum binaries/* extension/* > checksums.txt

# Share via email, Sharepoint, Teams Files, or post link
echo "✅ Distribution ready at: /Volumes/shared/copilot-token-budget-v1.0.0/"
```

#### **Step 4b (GitHub Release): Create release**
```bash
cd /path/to/copilot-token-budget

# Tag version
git tag v1.0.0
git push origin v1.0.0

# GitHub Actions workflow runs automatically.
# Monitor: https://github.com/<your-org>/copilot-token-budget/actions

# Once complete, release available at:
# https://github.com/<your-org>/copilot-token-budget/releases/v1.0.0
```

#### **Step 4c (JFrog): Publish to Artifactory**

Same as GitHub Release (Step 4b), but `.github/workflows/release.yml` also uploads to JFrog (if configured).

```bash
git tag v1.0.0
git push origin v1.0.0

# Workflow runs. Check:
# https://artifactory.company.com/artifactory/webapp/#/artifacts/browse/tree/General/copilot-token-budget-binaries/v1.0.0
```

---

### What to Share with Your Team

Send each team member:

```markdown
# Copilot Token Budget v1.0.0 — Installation Guide

**What is it:** Local tool to track GitHub Copilot CLI + IDE Chat token usage, 
set monthly budgets, and get Teams alerts.

**Platforms:** macOS (Intel + Apple Silicon), Linux, Windows

**What you get:**
- `copilot-analyze` — one-shot usage report
- `copilot-dashboard` — live refresh (10s) with WezTerm badge support
- `copilot-alert` — Teams notifications (optional webhook)
- `copilot-statusline` — status line for shell prompt
- VS Code extension — inline dashboard + commands

**Install Steps:**

1. **Download binaries for your OS**
   [ Link to shared folder / GitHub Releases / JFrog ]

2. **Install extension**
   VS Code → Extensions → "Install from VSIX" → select copilot-token-budget-*.vsix

3. **Quick start**
   Terminal: copilot-analyze
   VS Code: Ctrl+Shift+P → "Copilot Budget: Show Dashboard"

4. **Configure Teams alerts (optional)**
   See: onboarding-runbook.md

**Support:** Post questions in #copilot-budget (Teams channel)

**Detailed guide:** Read USAGE.md and onboarding-runbook.md
```

---

### Verification Checklist (Before Sharing)

- [ ] All binaries build without errors (`goreleaser build --snapshot --clean`)
- [ ] Extension `.vsix` packages without warnings (`npm run package`)
- [ ] Test on real machine: `./copilot-analyze` shows sessions
- [ ] Test extension: Install `.vsix`, command palette works
- [ ] README + USAGE.md + onboarding-runbook.md in distribution folder
- [ ] `checksums.txt` generated and verified
- [ ] Distribution link/folder is accessible to all team members
- [ ] If GitHub Release: artifacts appear at release page
- [ ] If JFrog: artifacts appear in Artifactory repo

---

### Cross-Platform Notes

**macOS:**
- Both Intel (`darwin-amd64`) and Apple Silicon (`darwin-arm64`) binaries included
- Team members download correct binary for their chip (`uname -m`)

**Linux:**
- Tested on Ubuntu 20.04+ (`linux-amd64` and `linux-arm64`)
- Builds for standard glibc; no musl variant

**Windows:**
- `.exe` binaries ready-to-run (no dependencies)
- `copilot-analyze.exe`, `copilot-dashboard.exe`, etc.
- VS Code extension works identically

---

### Troubleshooting Distribution

| Issue | Solution |
|-------|----------|
| "GoReleaser not found" | Install: `brew install goreleaser` (macOS) or download from github.com/goreleaser/goreleaser |
| "vsce package fails" | Update Node to 22+: `nvm install 22 && nvm use 22` |
| Binaries don't run on team's machine | Verify architecture matches: `uname -m` (macOS/Linux) or `wmic os get osarchitecture` (Windows) |
| Team can't download from shared folder | Check permissions; if using Teams: Files → Sync to local drive first |
| Checksum mismatch | Re-download from source; network corruption during transfer; verify with: `sha256sum -c checksums.txt` |



---

## Phase 6 — Dual-source capture (Copilot CLI + VS Code IDE) · IDE SESSIONS LIVE (Phase 6.2 ✅)

**What's new (Phase 6.2, 2026-06-17):**
- ✅ VS Code IDE Chat sessions now visible alongside CLI sessions
- ✅ Multi-source reader: CLI (authoritative token costs) + IDE (metadata, costs deferred)
- ✅ Automatic dedup: `{source}:{sessionId}` prevents cross-source ID collisions
- ✅ Per-source reporting: analyze/dashboard show breakdown (CLI vs. IDE)
- ✅ Token cost labels: "authoritative" (CLI from billing) vs. "estimated" (IDE metadata)

**What's included:**
- **CLI sessions:** all local data, token costs authoritative (from session.shutdown events)
- **IDE sessions:** ~116 sessions from `~/.config/github-copilot/ic/` (Xodus DB via Nitrite SDK) or JSON metadata fallback
- **Conversation history:** Session timestamps, turn counts, model names
- **Costs:** IDE costs marked "unavailable" for Phase 6 (Phase 7 will add GitHub API enrichment)

**Run the same commands as Phase 1** — they now include IDE sessions:

```bash
cd core

# One-shot report — now includes IDE Chat sessions + breakdown
go run ./cmd/analyze [workspace-root]

# Live dashboard — refreshes IDE sessions every 10s
go run ./cmd/dashboard [workspace-root]
```

**Example output (with IDE sessions):**
```
CLI Sessions: 53 (14,144.66 cr / authoritative)
IDE Chat Sessions: 116 (costs unavailable / Phase 6 limitation)
─────────────────────────────────────────────────────────────
Total: 169 sessions
```

**Known Phase 6 limitations (by design):**
- IDE token costs are estimated/unavailable (server-side only on GitHub)
- Per-turn granularity limited to metadata (Phase 7: Nitrite SDK integration for full detail)
- Model names in IDE metadata only (no cost per model for IDE)

**Phase 7 (coming soon):**
- GitHub API integration will populate real IDE token costs
- IDE cost labels will change from "estimated" to "authoritative"
- Per-turn granularity restored (Nitrite SDK + API data)

---

## Phase 7 — Usage insight (v1.1)

These ship inside the Phase 1/2/4 binaries you already built.

**Machine-readable export** (CLI):

```bash
cd core
go run ./cmd/analyze --json [workspace-root]   # full report as JSON (camelCase)
go run ./cmd/analyze --csv  [workspace-root]   # per-session CSV (RFC-4180)
```

**Status line** (ccusage-style one-liner; no args, never panics, exits 0, honours `NO_COLOR`):

```bash
go run ./cmd/statusline
# example: 🤖 sonnet-4.6 | 💰 0 today / 4950/7000 (71%) | 🔥 309/day | 🧠 75%
# macOS/Linux: embed the built binary in a shell prompt or WezTerm right-status
# Windows (PowerShell): call copilot-statusline.exe from your prompt function, e.g.
#   function prompt { (& "$env:USERPROFILE\bin\copilot-statusline.exe"); "PS $($pwd)> " }
```

**Override pricing / allowance / context window** (no rebuild) — drop a `pricing.json` in the config dir.
Partial files merge over the bundled defaults; a missing/malformed file falls back safely.

```
macOS/Linux : ~/.config/copilot-token-budget/pricing.json
Windows     : %AppData%\copilot-token-budget\pricing.json
```

```json
{
  "allowanceCredits": 7000,
  "models": {
    "sonnet": { "inputPerMillion": 300, "outputPerMillion": 1500, "contextWindowTokens": 200000 },
    "opus":   { "inputPerMillion": 500, "outputPerMillion": 2500, "contextWindowTokens": 200000 },
    "haiku":  { "inputPerMillion": 100, "outputPerMillion": 500,  "contextWindowTokens": 200000 }
  }
}
```

In the VS Code extension, point `copilotBudget.pricingPath` at a file of the same shape. (See ADR-008.)

> **All cost figures are estimates.** The tool reads local telemetry and applies a local price table;
> it never reconciles against GitHub's authoritative billing.

---

## Developer: build & test the whole repo

```bash
# Go (run in each module: core, alerting, mcp)
go build ./... && go vet ./... && go test ./... -race

# TypeScript extension
cd extension && npm run compile
```

Acceptance gates per phase live in `docs/history/evaluation/` (`ACCEPTANCE_CRITERIA.md`, `PHASE3/PHASE4/PHASE7_ACCEPTANCE.md`).
