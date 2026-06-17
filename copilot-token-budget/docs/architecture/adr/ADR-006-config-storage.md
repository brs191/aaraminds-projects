# ADR-006 — Cross-platform config and state storage

**Status:** Accepted
**Date:** 2026-06-14

## Context

Phase 3 introduces two new storage needs:

1. **Alert deduplication state** — the Teams alert engine must track which threshold
   levels (WARNING at 60%, CRITICAL at 90%) have already fired today, so engineers
   are not spammed with repeated notifications.

2. **Teams webhook URL** — the URL used to POST alert cards to Microsoft Teams.
   This is a secret-adjacent value (anyone with it can post to the team channel).

The tool targets 1,000+ AT&T engineers on macOS today, Windows in a future release.
Any storage decision must work on both platforms without a code change.

Phase 1 already ships `platform.ConfigDir()` in `internal/platform/paths.go`, which
returns `filepath.Join(os.UserConfigDir(), "copilot-token-budget")` and creates the
directory on first call. A second path-construction pattern must not be introduced.

## Decision

### 1. Config/state directory

| Layer | Path | How |
|---|---|---|
| **Go binary** | `platform.ConfigDir()` → `~/.config/copilot-token-budget/` (macOS/Linux), `%AppData%\copilot-token-budget\` (Windows) | Existing helper — no new code |
| **VS Code extension** | `vscode.ExtensionContext.globalStorageUri` | VS Code sandboxes this per extension per user — no path construction needed |

The Go binary writes `state.json` to `platform.ConfigDir()`. The VS Code extension
reads `globalStorageUri` for its own UI state. They do not share a directory — this
is intentional (the extension reads alert state from the Go binary's stdout/exit code,
not from a shared file).

### 2. state.json structure and atomicity

Location: `platform.ConfigDir() + "/state.json"`

Schema:
```json
{
  "thresholdAlerts": {
    "60": "2026-06-13",
    "90": "2026-06-13"
  }
}
```

Keys are threshold percentages as strings; values are ISO 8601 date strings (UTC, date
only). On each alert check:

- If the key is absent or the stored date is before today → fire the alert, write today's date.
- If the stored date equals today → skip (already alerted today).

**Atomic write protocol** (mandatory — 1,000 machines, any can be killed mid-write):

```
1. Marshal JSON to bytes
2. Write to state.json.tmp (same directory — same filesystem, so rename is atomic)
3. os.Rename("state.json.tmp", "state.json")
```

`os.Rename` is an atomic syscall on POSIX (macOS/Linux) and atomic on NTFS (Windows)
when source and destination are on the same volume. This prevents a half-written file
from being read as corrupt state.

File permissions: `0600` (owner read/write only — no other user on the machine can
read the alert history).

### 3. Teams webhook URL — environment variable, not CLI flag

The webhook URL is **never** stored in `state.json`.

| Where | How |
|---|---|
| **Storage** | VS Code setting `copilotBudget.teamsWebhookUrl` — encrypted at rest by VS Code's secret storage layer |
| **Delivery to Go binary** | Environment variable `COPILOT_BUDGET_TEAMS_WEBHOOK` injected by the extension before exec |
| **NOT a CLI flag** | CLI flags appear in `ps aux` output and are visible to all users on a shared machine. Environment variables are per-process and not visible to other users. |

The Go binary reads `os.Getenv("COPILOT_BUDGET_TEAMS_WEBHOOK")`. If the variable is
empty, it exits 0 silently (no webhook configured — not an error).

### 4. state.json security invariant

`state.json` contains **only** threshold IDs (integers as strings) and ISO 8601 dates.
It must never contain:
- Webhook URLs
- API tokens or credentials
- Personal data
- Session content or file paths

This invariant is enforced by the schema above. Any future additions to `state.json`
must be reviewed against this invariant before merging.

## Rationale

- **`platform.ConfigDir()` reuse** — single path-construction pattern; tested; Windows
  path already correct via `os.UserConfigDir()`.
- **Atomic rename** — prevents corruption on any of 1,000 machines regardless of kill
  signal timing. Rename within the same directory is always same-filesystem.
- **Env var for webhook** — `ps aux` visibility of CLI flags is a real threat on shared
  developer machines. Environment variables are the POSIX-standard way to pass secrets
  to subprocesses.
- **VS Code `globalStorageUri`** — VS Code manages the path, handles migration, and
  is available without any path computation in TypeScript code.
- **No shared state file between Go and TypeScript** — avoids file-locking complexity.
  The extension gets alert outcomes from the binary's exit code and stdout, not from
  polling a shared file.

## Consequences

- Go binary requires `COPILOT_BUDGET_TEAMS_WEBHOOK` env var to send alerts — the VS
  Code extension is responsible for injecting it; running the binary standalone without
  the env var is a no-op (correct for CLI use).
- `state.json` date-based dedup resets at midnight UTC — this is intentional. An
  engineer who spends into a new day gets a fresh alert. Month-boundary behaviour is
  correct: new month = new allowance = new alert cycle.
- Windows path `%AppData%\copilot-token-budget\state.json` is created automatically by
  `platform.ConfigDir()` on first run — no installer action required.

## Alternatives considered

| Alternative | Rejected because |
|---|---|
| Store webhook URL in `state.json` | Secret-adjacent value must not be in a plain file at 0600 on a shared machine |
| Pass webhook URL as `--webhook` CLI flag | Visible in `ps aux` to all users on the machine |
| Use a dedicated `.env` file | Adds a second path-construction pattern; no benefit over VS Code settings |
| SQLite instead of `state.json` | Overkill for a two-key dedup record; adds a dependency |
| Share state file between Go and TypeScript | Requires file locking; Go and Node.js lock APIs differ on Windows |
