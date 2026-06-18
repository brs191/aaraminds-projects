# GitHub Enterprise Entitlement Setup

This guide is for AT&T GitHub Enterprise admins who want to enable live Copilot billing in the Copilot Token Budget tool.

## Overview

The **live billing feature** fetches your organization's Copilot monthly quota directly from GitHub's internal GraphQL API, so you can see your **authoritative** usage (e.g., `28,646 / 35,000`) instead of only estimated local usage.

The feature is **opt-in** and **default-off**. When disabled, the tool works exactly as before (local-only, no network).

---

## Prerequisites

- GitHub Enterprise account with SAML/SSO
- A GitHub organization with Copilot licenses provisioned
- One of:
  - A **GitHub Personal Access Token** (PAT) with `manage_billing:copilot` scope, **or**
  - A **Copilot Billing Token** (internal GitHub entitlement service token, if available)

### Obtaining a Token

#### Option A: GitHub Personal Access Token (PAT)

1. Go to https://github.com/settings/tokens/new (or your GitHub Enterprise instance equivalent)
2. Click **Generate new token** → **Generate new token (classic)**
3. Give it a name (e.g., `copilot-billing-token`)
4. Select scopes: **`manage_billing:copilot`**
5. Click **Generate token** and **copy it immediately** (you won't see it again)

#### Option B: Copilot Billing Token (Enterprise-only)

Ask your GitHub Enterprise support contact for a dedicated Copilot entitlement service token. This is more restricted than a PAT.

---

## Configuration

### 1. Create the Config File

On your local machine, create `~/.copilot/config.json`:

```json
{
  "enabled": true,
  "orgSlug": "your-org-slug",
  "tokenEnvVar": "COPILOT_BILLING_TOKEN",
  "cacheMaxAgeHours": 24,
  "requestTimeoutSecs": 10,
  "gitHubAPIUrl": "https://api.github.com",
  "dryRun": false
}
```

Replace:
- `"your-org-slug"` with your GitHub organization slug (e.g., `att-org`)
- `"gitHubAPIUrl"` with your GitHub Enterprise API endpoint if not using github.com (e.g., `https://github.your-company.com/api`)

### 2. Set the Environment Variable

Store your token **securely** in your shell's environment. Do **not** commit it to version control.

#### macOS/Linux

Add to `~/.bashrc`, `~/.zshrc`, or equivalent:

```bash
export COPILOT_BILLING_TOKEN="ghp_your_pat_or_token_here"
```

Then reload:
```bash
source ~/.bashrc
# or
source ~/.zshrc
```

#### Windows (PowerShell)

```powershell
[Environment]::SetEnvironmentVariable("COPILOT_BILLING_TOKEN", "ghp_your_pat_or_token_here", "User")
```

Then restart your shell.

### 3. Verify the Setup

Run the CLI with verbose output:

```bash
copilot-budget analyze --json 2>&1 | grep -i "billing\|quota\|org"
```

If you see output like `"source": "(org aggregate, 24h ago)"` and a quota number, the feature is working.

### 4. (Optional) Test Before Enabling

Set `"dryRun": true` in `config.json` to test the config without making actual API calls:

```json
{
  "enabled": true,
  "orgSlug": "your-org-slug",
  "dryRun": true
}
```

Run the tool; you should see an error like `dry-run mode; no API call made`. If you see that, the config is valid.

Then set `"dryRun": false` to enable live fetching.

---

## Monitoring & Troubleshooting

### Check the Live-Billing Cache

The tool caches the org quota for 24 hours (configurable via `cacheMaxAgeHours`). The cache file is:

```
~/.copilot/live-billing-cache.json
```

Example:
```json
{
  "orgSlug": "att-org",
  "allowedCredits": 35000,
  "cachedAt": "2026-06-17T18:30:00Z",
  "ttlHours": 24
}
```

### Disable the Feature

To revert to local-only (no network calls):

1. Edit `~/.copilot/config.json` and set `"enabled": false`, **or**
2. Delete `~/.copilot/config.json` entirely (defaults to local-only)

The tool will immediately revert to showing estimated usage only.

### Common Errors

| Error | Cause | Fix |
|---|---|---|
| `live billing enabled but orgSlug is empty` | Config file missing `orgSlug` | Add `"orgSlug": "your-org"` to config.json |
| `GitHub API returned 401` | Token is invalid or expired | Re-generate a new PAT or use a fresh entitlement token |
| `org quota is 0 (zero or not set)` | Org has no Copilot quota provisioned | Contact your GitHub Enterprise admin to provision Copilot |
| `request timeout` | GitHub API took > 10 seconds | Increase `requestTimeoutSecs` in config.json (max 60) |

---

## Security Notes

- **Never commit your token** to version control or CI/CD pipelines. Use environment variables only.
- If you suspect your token is compromised, **revoke it immediately** on https://github.com/settings/tokens and generate a new one.
- The tool never logs or transmits your token; it's only used locally to call GitHub's API.
- All live-billing data is cached **locally** at `~/.copilot/live-billing-cache.json`; it never leaves your machine except in the query to GitHub.

---

## Support

If you encounter issues:

1. Check `docs/history/IMPLEMENTATION_PLAYBOOK.md` → **Step 8.6** for technical details
2. Enable debug logging (if available) and review `stderr` output
3. Consult your GitHub Enterprise support team if the issue is with quota provisioning or token access

---

## FAQs

**Q: Will live billing affect performance?**
A: The tool caches the org quota for 24 hours, so the API is called at most once per day. Network calls are asynchronous and never block local usage reporting.

**Q: What if the GitHub API is down?**
A: The tool gracefully degrades. If the API is unavailable or times out, live billing is marked `unavailable` and the tool shows estimated usage instead.

**Q: Can I use the same token for multiple machines?**
A: Yes, the token is global to your GitHub org. Set the same `COPILOT_BILLING_TOKEN` env var on each machine.

**Q: How do I rollback?**
A: Delete or rename `~/.copilot/config.json` and restart the tool. It will revert to local-only mode immediately.
