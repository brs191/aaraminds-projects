# Copilot Token Budget — Enterprise Onboarding Runbook

**Audience:** AT&T engineers installing the tool for the first time.
**Goal:** zero → a working status-bar budget badge in **≤ 5 minutes**.
**Platforms:** macOS, Linux (incl. Ubuntu), Windows.
**Phase 5 · Step 5.4.**

The tool is **local-first**. Every figure is computed on your machine from the session-state
files the GitHub Copilot CLI already writes. The only outbound network call in the whole tool is
the optional Teams alert webhook. There is no GitHub API call, no proxy, no traffic interception.

> **All cost and credit figures are estimates.** The tool reads local telemetry and applies a
> local price table; it never reconciles against GitHub's authoritative billing.

---

## Conventions used below

Throughout this runbook, replace these placeholders with your real values:

| Placeholder | Meaning |
|---|---|
| `<version>` | the release version you are installing, e.g. `v1.0.0` |
| `<JF_BINARY_REPO>` | JFrog Artifactory **generic** repo base URL for binaries |
| `<JF_VSIX_REPO>` | JFrog Artifactory base URL for the VS Code `.vsix` |
| `<YOUR_ATTUID>` | your AT&T user id (your home-directory name) |

The five published binaries (Windows names carry `.exe`):

`copilot-analyze` · `copilot-dashboard` · `copilot-statusline` · `copilot-alert` · `copilot-budget-mcp`

Distribution layout in Artifactory:

```
<JF_BINARY_REPO>/binaries/<version>/   # the 5 binaries (per OS/arch)
<JF_VSIX_REPO>/vsix/<version>/         # the VS Code .vsix
```

---

## 1. Prerequisites

### All platforms

1. **GitHub Copilot CLI installed and used at least once.** The tool reports on the session-state
   files the CLI writes; with no sessions there is nothing to report. Confirm the data directory
   exists and is non-empty:
   - **macOS / Linux:** `ls ~/.copilot/session-state/`
   - **Windows (PowerShell):** `dir $env:USERPROFILE\.copilot\session-state\`

   If the directory is empty, run any Copilot CLI prompt once, then re-check.

2. **JFrog Artifactory access.** Either:
   - the `jf` CLI installed and configured (`jf c add` against the AT&T Artifactory instance), **or**
   - browser access to the repo URLs above (use the curl/browser fallback in the quick start).

3. **VS Code** *(optional — only for the status-bar badge and the extension UI).*

### Building from source (NOT required for the distributed binaries)

The published binaries are static and need **nothing** installed to run — no Go, no Node. You only
need a toolchain if you build from source:

| If you build… | You need |
|---|---|
| phase-1 / phase-3 binaries | Go 1.21+ |
| phase-4 MCP server (`copilot-budget-mcp`) | **Go 1.25+** (hard requirement of `modelcontextprotocol/go-sdk`; `GOTOOLCHAIN=auto` will auto-fetch it) |
| VS Code extension `.vsix` | Node 18+ |

Most engineers should **download** rather than build. Build-from-source steps are noted where relevant.

---

## 2. Five-minute quick start

Pick your column. Run top to bottom.

### a. Download the binaries and put them on PATH

**Option A — `jf` CLI (recommended)**

macOS / Linux:

```bash
mkdir -p ~/bin
# Adjust the OS/arch suffix to match your machine, e.g. darwin_arm64, darwin_amd64,
# linux_amd64, linux_arm64. Below downloads all five into ~/bin.
for b in copilot-analyze copilot-dashboard copilot-statusline copilot-alert copilot-budget-mcp; do
  jf rt download "binaries/<version>/${b}" "$HOME/bin/${b}" --flat
done
chmod +x ~/bin/copilot-*
```

Windows (PowerShell):

```powershell
New-Item -ItemType Directory -Force "$env:USERPROFILE\bin" | Out-Null
foreach ($b in "copilot-analyze","copilot-dashboard","copilot-statusline","copilot-alert","copilot-budget-mcp") {
  jf rt download "binaries/<version>/$b.exe" "$env:USERPROFILE\bin\$b.exe" --flat
}
```

**Option B — plain curl / browser fallback (no `jf`)**

macOS / Linux:

```bash
mkdir -p ~/bin
for b in copilot-analyze copilot-dashboard copilot-statusline copilot-alert copilot-budget-mcp; do
  curl -fSL "<JF_BINARY_REPO>/binaries/<version>/${b}" -o "$HOME/bin/${b}"
done
chmod +x ~/bin/copilot-*
```

Windows (PowerShell):

```powershell
New-Item -ItemType Directory -Force "$env:USERPROFILE\bin" | Out-Null
foreach ($b in "copilot-analyze","copilot-dashboard","copilot-statusline","copilot-alert","copilot-budget-mcp") {
  Invoke-WebRequest "<JF_BINARY_REPO>/binaries/<version>/$b.exe" -OutFile "$env:USERPROFILE\bin\$b.exe"
}
```

Or just open `<JF_BINARY_REPO>/binaries/<version>/` in a browser and download each file into your
`bin` directory.

**Put `bin` on PATH** (one-time):

```bash
# macOS / Linux — add to ~/.zshrc or ~/.bashrc
export PATH="$HOME/bin:$PATH"
```

```powershell
# Windows (PowerShell, user-scoped, persists)
[Environment]::SetEnvironmentVariable("Path", "$env:USERPROFILE\bin;" + [Environment]::GetEnvironmentVariable("Path","User"), "User")
```

> If you prefer a system-wide location on macOS/Linux, use `/usr/local/bin` instead of `~/bin`
> (`sudo mv ~/bin/copilot-* /usr/local/bin/`).

**macOS only — clear the Gatekeeper quarantine attribute.** Files downloaded via browser/curl are
quarantined and macOS will refuse to run them with a "cannot verify developer" error:

```bash
xattr -d com.apple.quarantine ~/bin/copilot-* 2>/dev/null || true
```

### b. Verify

```bash
copilot-analyze --version     # every binary supports --version
copilot-analyze               # one-shot report: budget, history, top consumers, instruction audit
```

Windows:

```powershell
copilot-analyze.exe --version
copilot-analyze.exe
```

You should see a report ending in something like
`💰 4950/7000 (71%)`. If it says **"no sessions found,"** the Copilot CLI hasn't written
session-state yet — see Troubleshooting.

> **Scope note:** the tool currently tracks the **GitHub Copilot CLI** only. VS Code **IDE** usage
> capture is groundwork-only (the IDE collector is a stub); it is not yet reflected in the numbers.

### c. (Optional) add the status line to your prompt / WezTerm

`copilot-statusline` prints a single line and always exits 0 (it honours `NO_COLOR` and never panics):

```bash
copilot-statusline
# 🤖 sonnet-4.6 | 💰 0 today / 4950/7000 (71%) | 🔥 309/day | 🧠 75%
```

Embed it in your shell prompt or WezTerm right-status by calling the built binary, e.g. in
`~/.wezterm.lua` run `copilot-statusline` on a timer and set it as the right-status text.

### d. Install the VS Code extension and see the badge

```bash
# Download the .vsix (jf CLI)
jf rt download "vsix/<version>/copilot-token-budget-<version>.vsix" ./ --flat
# …or curl:
curl -fSL "<JF_VSIX_REPO>/vsix/<version>/copilot-token-budget-<version>.vsix" -o copilot-token-budget-<version>.vsix

# Install
code --install-extension copilot-token-budget-<version>.vsix
```

Then set your monthly allowance so the badge has a denominator:

1. Command Palette → **Copilot Budget: Open Settings** (or open Settings and search `copilotBudget`).
2. Set **`copilotBudget.monthlyAllowance`** (e.g. `7000`).
3. The **status-bar budget badge** appears within one refresh cycle.

Useful extension commands (Command Palette):
`Copilot Budget: Show Dashboard` · `Refresh` · `Open Settings` · `Export Usage` (JSON/CSV).

Key settings (`copilotBudget.*`): `monthlyAllowance`, `workspacePath`, `refreshIntervalSec`,
`teamsWebhookUrl`, `alertThresholdWarn`, `alertThresholdCrit`, `alertBinaryPath`, `pricingPath`.

**That's the 5-minute path.** MCP and Teams below are optional power-ups.

---

## 3. MCP setup — ask Copilot "how's my budget?"

This registers `copilot-budget-mcp` so the Copilot CLI model can call the budget tools directly.

1. **Obtain the binary.** You already downloaded it in step 2a (it's in your `bin`). To build from
   source instead (requires **Go 1.25+**):

   ```bash
   cd phase-4
   go build -ldflags "-X main.Version=<version>" -o ~/bin/copilot-budget-mcp ./cmd/mcp-server
   # Windows:
   # go build -ldflags "-X main.Version=<version>" -o %USERPROFILE%\bin\copilot-budget-mcp.exe .\cmd\mcp-server
   ```

2. **Write `.copilot/mcp.json` in your workspace root.** The `command` field **must be an ABSOLUTE
   path** — `execve` does **not** expand `~`, and a leading tilde will fail to launch the server.

   macOS / Linux:

   ```json
   {
     "mcpServers": {
       "copilot-token-budget": {
         "command": "/Users/<YOUR_ATTUID>/bin/copilot-budget-mcp",
         "args": [],
         "env": {}
       }
     }
   }
   ```

   Linux absolute path example: `/home/<YOUR_ATTUID>/bin/copilot-budget-mcp`.

   Windows (use `.exe` and backslashes):

   ```json
   {
     "mcpServers": {
       "copilot-token-budget": {
         "command": "C:\\Users\\<YOUR_ATTUID>\\bin\\copilot-budget-mcp.exe",
         "args": [],
         "env": {}
       }
     }
   }
   ```

3. **Smoke-test** the binary before relying on the registration:

   ```bash
   ~/bin/copilot-budget-mcp --version
   ```

4. **Ask Copilot.** In a Copilot CLI session in that workspace, ask *"how's my budget?"* The model
   can call: `get_budget_status`, `get_sessions`, `get_instruction_overhead`, `get_model_costs`,
   `get_usage_timeseries`, `get_top_consumers`. All read local files only.

> The model-routing / cost-saving recommender surfaces through the **MCP tools and Teams alerts**,
> not the CLI report or IDE.

---

## 4. Teams alerts setup

Alerts POST an Adaptive Card to a Microsoft Teams channel when your usage crosses a threshold.

> **CRITICAL (current as of 2026-06-16):** the legacy Microsoft **O365 "Incoming Webhook"
> connector is retired** (rollout completing ~May 2026). **Do not** create an O365 connector — its
> URL will not work. Create a **Power Automate "Workflows" webhook** instead. Our Adaptive Card
> payload is compatible with Workflows webhooks.

### 4.1 Create the Power Automate Workflows webhook

1. In Teams, go to the **channel** you want alerts in.
2. Click the **⋯ (More options)** next to the channel name → **Workflows**.
   (Alternatively: **+ Apps → Workflows**, or **Power Automate**.)
3. Search the templates for **"Post to a channel when a webhook request is received."**
   (Listed in some clients as *"Send webhook alerts to a channel."*)
4. Select the template, confirm the connection/sign-in, then choose the **Team** and **Channel**
   that should receive the posts. **Save / Create.**
5. Teams shows the generated **HTTP POST URL** — **copy it.** This is your webhook URL.
   To retrieve it later: open the **Workflows** app → select the flow → **Edit** → expand the
   trigger **"When a Teams webhook request is received."**

### 4.2 Configure the webhook for the tool

The webhook URL is read from one of two places (never a command-line flag):

- **CLI / cron path** — environment variable `COPILOT_BUDGET_TEAMS_WEBHOOK` (read by `copilot-alert`).
- **Extension path** — the **`copilotBudget.teamsWebhookUrl`** setting. When set, the extension's
  refresh loop auto-invokes the `copilot-alert` binary (found via `copilotBudget.alertBinaryPath`,
  defaulting to `~/bin/copilot-alert`).

Set the env var:

```bash
# macOS / Linux — add to ~/.zshrc or ~/.bashrc
export COPILOT_BUDGET_TEAMS_WEBHOOK="https://<your-workflows-webhook-url>"
```

```powershell
# Windows (PowerShell, persists for the user)
[Environment]::SetEnvironmentVariable("COPILOT_BUDGET_TEAMS_WEBHOOK", "https://<your-workflows-webhook-url>", "User")
```

### 4.3 Test it

**Dry run first — builds the Adaptive Card JSON and makes NO network call.** The `<workspace>`
argument is required (it points the instruction-overhead audit at a workspace; session data is
always read from `~/.copilot/session-state/`):

```bash
copilot-alert --dry-run <workspace>
```

Then a **real run** — POSTs to Teams only if a threshold is crossed and that threshold hasn't
already fired today:

```bash
COPILOT_BUDGET_TEAMS_WEBHOOK="https://<your-workflows-webhook-url>" copilot-alert <workspace>
```

Exit codes: `0` = no alert needed / already sent today · `1` = alert fired (or dry-run printed) ·
`2` = error.

### 4.4 Thresholds and dedup

- **Warn at 60%**, **critical at 90%** of `monthlyAllowance` (override via
  `copilotBudget.alertThresholdWarn` / `alertThresholdCrit` in the extension).
- **Once-per-day dedup:** the same threshold will not re-fire on the same day (UTC, per-day). You
  get at most one warn and one crit notification per day.

**Run on a schedule** (cron, every 30 min during work hours):

```cron
*/30 9-18 * * 1-5  COPILOT_BUDGET_TEAMS_WEBHOOK="https://..." /home/<YOUR_ATTUID>/bin/copilot-alert /path/to/workspace
```

---

## 5. Customizing pricing and allowance

All costs are **estimates** computed from a local price table. Two ways to override it — no rebuild:

### 5.1 `pricing.json` in the config dir (applies to CLI + MCP)

Drop a `pricing.json` into the config directory. Partial files **merge** over the bundled defaults;
a missing or malformed file falls back safely. (See ADR-008.)

| OS | Config dir |
|---|---|
| macOS / Linux | `~/.config/copilot-token-budget/pricing.json` |
| Windows | `%AppData%\copilot-token-budget\pricing.json` |

Sample:

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

### 5.2 Extension settings (applies to the badge / dashboard)

- **`copilotBudget.monthlyAllowance`** — your monthly credit allowance (the badge's denominator).
- **`copilotBudget.pricingPath`** — point at a `pricing.json` of the same shape as above.

> If the AT&T credit allowance changes (e.g. post-2026-09-01), updating `monthlyAllowance` is a
> **settings change, not a code change.**

---

## 6. Troubleshooting

**"no sessions found" in `copilot-analyze`.**
The Copilot CLI hasn't written session-state yet. Confirm `~/.copilot/session-state/`
(`%USERPROFILE%\.copilot\session-state\` on Windows) exists and is non-empty. Run any Copilot CLI
prompt once, then re-run `copilot-analyze`.

**Status-bar badge not showing.**
1. Confirm the extension installed: `code --list-extensions | grep -i copilot-token-budget`.
2. Set **`copilotBudget.monthlyAllowance`** — without it the badge has no denominator.
3. Run **Copilot Budget: Refresh** from the Command Palette.
4. Check that there is real session data (see "no sessions found" above).

**MCP binary not found / server won't start.**
Almost always a **tilde-not-expanded** path. `execve` does not expand `~`. Edit
`.copilot/mcp.json` so `command` is a full absolute path
(`/Users/<YOUR_ATTUID>/bin/copilot-budget-mcp`, or `C:\Users\<YOUR_ATTUID>\bin\copilot-budget-mcp.exe`
on Windows). Verify the binary runs: `~/bin/copilot-budget-mcp --version`.

**macOS "cannot verify developer" / Gatekeeper block.**
The binary is quarantined. Clear it:
`xattr -d com.apple.quarantine ~/bin/copilot-*`. Then re-run.

**Teams alert not arriving.**
1. **Most common:** the webhook URL is from a **retired O365 connector.** Recreate it as a Power
   Automate **Workflows** webhook (§4.1) and update `COPILOT_BUDGET_TEAMS_WEBHOOK` /
   `copilotBudget.teamsWebhookUrl`.
2. Confirm a threshold actually crossed — below 60% nothing fires. Force the card build with
   `copilot-alert --dry-run <workspace>` to confirm the payload is generated.
3. Remember the **once-per-day dedup**: if the same threshold already fired today, it won't re-fire.
4. Confirm the env var is exported in the shell that runs the binary (cron jobs don't inherit your
   interactive shell — set it inside the cron line).

---

## 7. Uninstall

```bash
# macOS / Linux
rm -f ~/bin/copilot-analyze ~/bin/copilot-dashboard ~/bin/copilot-statusline \
      ~/bin/copilot-alert ~/bin/copilot-budget-mcp
rm -rf ~/.config/copilot-token-budget          # config dir (pricing.json etc.)
rm -f <workspace>/.copilot/mcp.json            # MCP registration
code --uninstall-extension <publisher>.copilot-token-budget   # VS Code extension
# remove the PATH export and COPILOT_BUDGET_TEAMS_WEBHOOK from your shell rc
```

```powershell
# Windows (PowerShell)
Remove-Item "$env:USERPROFILE\bin\copilot-*.exe" -Force
Remove-Item "$env:AppData\copilot-token-budget" -Recurse -Force
Remove-Item "<workspace>\.copilot\mcp.json" -Force
code --uninstall-extension <publisher>.copilot-token-budget
# remove the bin dir from PATH and delete COPILOT_BUDGET_TEAMS_WEBHOOK via System Environment Variables
```

The tool stores nothing else. It never wrote to `~/.copilot/session-state/` (read-only) — leave
that directory alone; it belongs to the Copilot CLI.

---

## References

- Microsoft — *Create incoming webhooks with Workflows for Microsoft Teams*:
  https://support.microsoft.com/en-us/teams/apps-service/create-incoming-webhooks-with-workflows-for-microsoft-teams
- Microsoft 365 Developer Blog — *Retirement of Office 365 connectors within Microsoft Teams*:
  https://devblogs.microsoft.com/microsoft365dev/retirement-of-office-365-connectors-within-microsoft-teams/
- Repo: `USAGE.md` (per-phase run guide), `.copilot/mcp.json` (MCP scaffold), ADR-008 (pricing override).
