# Validate Copilot routing mode — pre-flight for M0-lite

**Created:** 2026-05-27  ·  **Updated:** 2026-05-31  ·  **Owner:** Raja  ·  **Status:** 🔴 **BLOCKING — UNRESOLVED.** Must be settled before the 6/8–6/12 measurement window; it should have gated the 5/28 PoC.
**Blocks:** the M0-lite measurement (`../tracking/milestones/M0-lite.md`) **and** the M1 GREEN verdict's `S = $100/dev/mo` input (`../tracking/milestones/M1-Decision-Gate.md`).
**Source context:** This document captures the Copilot architecture-fit finding surfaced during M0-lite cohort screening on 2026-05-27, before the PoC starts.

> **🔴 2026-05-31 status — still unresolved.** The 5/28–5/29 PoC window has elapsed with no recorded mode determination. Any PoC data collected may therefore be from a mix of modes — or from none at all, if devs were in server-routed Mode A and the proxy saw zero traffic. This is the project's single highest-impact open question: it gates not just the R measurement but the `S = $100/dev/mo` input the GREEN verdict rests on. If the cohort is on flat-rate Copilot subscriptions, there may be little or no per-token spend to compress against, and the verdict's economic math (50 × $100 × 0.20 = $1,000/mo, 5-month payback) does not hold. **Settle this before the 6/8 window or that week burns too.**

## Why this validation exists

The M0-lite spike measures token reduction by interposing a localhost LiteLLM proxy between the developer's AI coding assistant and the backend LLM API. The cohort screening confirmed the 7 committed engineers use **GitHub Copilot with Claude Opus 4.6** as their primary VS Code assistant.

Copilot is a proprietary VS Code extension that routes through GitHub's servers (`api.githubcopilot.com`). **Whether the LiteLLM proxy can actually interpose depends on which routing mode Copilot is operating in.** If we start the PoC Thursday without confirming this, we risk burning the 2-day measurement discovering that the proxy received zero requests.

## Three modes — only one of which works with M0-lite as built

### Mode A — Copilot's built-in model picker (GitHub-routed)

You select "Claude Opus 4.6" in Copilot Chat's model dropdown. VS Code calls GitHub's Copilot service → GitHub calls Anthropic server-side using GitHub-managed infrastructure → response back to VS Code.

- The API call to Anthropic **never leaves GitHub's servers** into your network. The proxy cannot see it.
- AITO's cost: Copilot subscription only (~$19/mo Pro, ~$39/mo Business; possibly metered for premium models). **NOT per-token.**
- Economic implication: **M1 verdict's S = $100/dev/mo doesn't fit this mode** — there's nothing to compress against because Copilot is flat-rate. The verdict's economic math collapses.

### Mode B — Copilot BYOK with AITO's Anthropic key (still GitHub-routed)

AITO has provided an Anthropic API key to GitHub Copilot Enterprise. GitHub still makes the API call server-side, but bills it against AITO's Anthropic account instead of GitHub-managed infrastructure.

- The API call still leaves **from GitHub's servers**, not your VS Code. The proxy still cannot interpose.
- AITO's cost: Copilot subscription + per-token Anthropic spend on AITO's key.
- Economic implication: **S = $100/dev/mo is plausible** (real per-token spend exists). But proxy interposition impossible. **Cohort must use a different assistant (Claude Code, Cursor, Continue, Aider) pointed at localhost:4000 for the measurement window.**

### Mode C — Copilot with custom endpoint configured in VS Code settings

VS Code 1.99+ Language Model API supports configuring a custom Anthropic-compatible `baseUrl`. If `settings.json` has a custom `github.copilot.advanced` block with an explicit Anthropic endpoint, VS Code makes the API call **directly from your local machine**.

- The proxy **CAN interpose** — VS Code's outbound HTTPS to `api.anthropic.com` (or wherever the custom endpoint points) can be redirected to `localhost:4000`.
- AITO's cost: per-token Anthropic spend.
- Economic implication: **S = $100/dev/mo plausible AND the project works as designed.** M0-lite proceeds as planned.

## Validation checks — run these on a developer machine

**Check 1 — VS Code settings.**

1. Open VS Code.
2. Open Settings JSON: `Cmd/Ctrl + Shift + P` → "Preferences: Open User Settings (JSON)".
3. Search the file for any of:
   - `copilot.advanced`
   - `copilot.chat`
   - `anthropic.baseUrl`
   - `endpoint`
   - `baseUrl`
4. Also check Workspace settings JSON (same command, but workspace scope).
5. Also check `.vscode/settings.json` in the AITO project repo if there is one.

Record what you find below.

**Check 2 — Anthropic billing console.**

1. Go to https://console.anthropic.com
2. Sign in with the AITO account that holds the team's Anthropic key (if there is one).
3. Navigate to: Settings → Billing → Usage.
4. Look at the last 30 days of token spend.
5. Check whether it matches roughly $100/dev × your effective dev count.

Record what you find below.

**Check 3 (optional — confirmatory) — network observation.**

While Copilot is making a request (e.g., a Copilot Chat message):

- **Mac:** Open Terminal, run `lsof -i -n -P | grep -i -E '(code|copilot)'` and look at the destination hostnames.
- **Linux:** `ss -tunp | grep -i code` or `netstat -anp | grep code`.
- **Windows:** `Get-NetTCPConnection | Where-Object {$_.OwningProcess -in (Get-Process Code).Id}` in PowerShell.

What you're looking for: outbound HTTPS connections from VS Code to:
- `api.anthropic.com` → **Mode C** (client-side call, proxyable).
- `api.githubcopilot.com` or `*.githubcopilot.com` only → **Mode A or B** (server-side mediation).

## Results — fill in below

**Date validated:** _________________
**Validated by:** _________________
**Machine / OS:** _________________
**VS Code version:** _________________
**GitHub Copilot extension version:** _________________

### Check 1 — VS Code settings

- [ ] Searched user settings JSON
- [ ] Searched workspace settings JSON
- [ ] Searched repo `.vscode/settings.json`

**Findings:**

```
(paste any copilot.advanced / baseUrl / endpoint configuration found, or write "none")
```

### Check 2 — Anthropic billing

- [ ] Logged into console.anthropic.com
- [ ] Reviewed last 30 days of usage

**Findings:**

```
(record: total token spend last 30 days, rough $/dev if calculable, or write "no per-token spend visible — no AITO Anthropic account / no usage")
```

### Check 3 — Network observation (optional)

- [ ] Triggered a Copilot request
- [ ] Observed outbound connections

**Findings:**

```
(record destination hostnames seen — api.anthropic.com? api.githubcopilot.com? both? something else?)
```

## Decision — circle one

| baseUrl set? | Anthropic billing? | Network goes to | Mode | Action |
|---|---|---|---|---|
| No | No | githubcopilot only | **A** | **STOP.** M1 verdict's S input wrong by order of magnitude. Re-litigate the gate before M0-lite is meaningful. |
| No | Yes | githubcopilot only | **B** | **Tooling swap.** Cohort uses Claude Code or Cursor (pointed at localhost:4000) for the measurement window. Document methodological caveat: measurement covers proxyable-assistant usage, not Copilot. PoC proceeds Thursday with the swap. |
| Yes | Yes | api.anthropic.com (direct or via VS Code) | **C** | **Proceed as planned.** Thursday's PoC starts; no change to M0-lite plan. |
| Yes | No | — | (uncommon) | Configuration likely incorrect — endpoint set but no per-token spend. Verify endpoint is actually active and being used. |

**Determined mode:** _________________
**Action chosen:** _________________

## What happens next once mode is determined

- **Mode C** → Mark task #21 (this validation) complete; proceed to the 6/8–6/12 measurement window per `../tracking/milestones/M0-lite.md`. (The 5/28–5/29 PoC window has already elapsed — record its outcome there.)
- **Mode B** → Mark this validation complete with the methodological caveat; update `../tracking/milestones/M0-lite.md` to specify the assistant being used by the cohort during the measurement (Claude Code or Cursor); update `../planning/M0-lite_Cohort_Recruitmen