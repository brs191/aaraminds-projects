# Live Billing Enablement (Engineer Test Guide)

This is a practical guide to enable and test **live billing** in this project.

## 1) What live billing does

- Keeps local session metrics as-is (CLI/IDE local telemetry remains primary).
- Adds an org-level GitHub billing snapshot as an enrichment layer.
- Requires explicit opt-in and a PAT with `manage_billing:copilot`.

---

## 2) Prerequisites

1. GitHub PAT with scope: `manage_billing:copilot`
2. Org slug (example: `your-org`)
3. Token set in environment variable (default: `COPILOT_BILLING_TOKEN`)

> Do **not** put PAT in source code or committed files.

---

## 3) Config file location

- macOS/Linux: `~/.config/copilot-token-budget/config.json`
- Windows: `%AppData%\copilot-token-budget\config.json`

---

## 4) Config format to use (current implementation)

Use this exact top-level JSON shape:

```json
{
  "enabled": true,
  "orgSlug": "your-org",
  "tokenEnvVar": "COPILOT_BILLING_TOKEN",
  "cacheMaxAgeHours": 24,
  "requestTimeoutSecs": 10,
  "gitHubAPIUrl": "https://api.github.com",
  "dryRun": false
}
```

Then set token:

```bash
export COPILOT_BILLING_TOKEN="ghp_xxx"
```

PowerShell:

```powershell
$env:COPILOT_BILLING_TOKEN = "ghp_xxx"
```

---

## 5) Test via CLI (`cmd/analyze`)

```bash
cd core
go run ./cmd/analyze
```

Expected behavior:
- Local metrics still render normally.
- Live billing attempts fetch and caches result.
- Cache file appears at:
  - macOS/Linux: `~/.config/copilot-token-budget/live-billing-cache.json`
  - Windows: `%AppData%\copilot-token-budget\live-billing-cache.json`

---

## 6) Test via VS Code extension

1. Install latest VSIX.
2. Reload VS Code.
3. Run command: **Copilot Budget: Refresh Now**
4. Open: **Copilot Budget: Show Dashboard**

Expected behavior:
- Dashboard metrics still come from local telemetry.
- Billing note should no longer stay purely `(estimated)` once live snapshot is available.

---

## 7) Safe dry-run mode (no HTTP calls)

Set in `config.json`:

```json
{
  "enabled": true,
  "orgSlug": "your-org",
  "tokenEnvVar": "COPILOT_BILLING_TOKEN",
  "cacheMaxAgeHours": 24,
  "requestTimeoutSecs": 10,
  "dryRun": true
}
```

Use this for config-path validation without network requests.

---

## 8) Disable/rollback immediately

Either:
1. Set `"enabled": false` in `config.json`, or
2. Delete `config.json`

This returns behavior to estimated local-only mode.
