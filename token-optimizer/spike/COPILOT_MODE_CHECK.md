# Copilot Routing-Mode Check

**Purpose:** Determine whether a developer's VS Code + Copilot setup can be intercepted by the local LiteLLM proxy — the precondition for the M0-lite token-compression measurement. **Read-only, ~2 minutes.**
**Date:** 2026-06-02 · **Owner:** Raja
**Companion to:** `../planning/validate_with_copilot.md` (full decision table) · `check_copilot_mode.ps1` / `check_copilot_mode.sh` (automated, all-platform)

---

## Why this matters

The Token Optimizer spike works by routing a coding assistant's LLM traffic through a localhost proxy (`http://localhost:4000`) that compresses the context. That only works if the assistant's request **leaves the developer's machine** — a proxy on the laptop cannot compress a request it never sees.

GitHub Copilot can route in three different ways, and only one is interceptable. Before the measurement window (6/8–6/12), every cohort dev must be confirmed in — or moved to — that interceptable mode. Otherwise the proxy sees zero traffic and the measured savings are meaningless. This check tells you which mode a machine is in.

## The three modes

| Mode | How it's set up | Who makes the LLM call | Proxyable? | Cost basis |
|---|---|---|---|---|
| **A** (default) | Copilot's built-in model dropdown | **GitHub's servers** | No | Flat subscription (~$19–39/mo) |
| **B** (BYOK) | AITO's Anthropic key handed to GitHub Copilot | **GitHub's servers** | No | Per-token on AITO's key |
| **C** (custom endpoint) | Copilot pointed at a custom Anthropic-compatible `baseUrl` in `settings.json` | **The developer's machine** | **Yes** | Per-token on AITO's key |

**Mode C is the goal.** Because the request originates locally and targets a base URL you control, you point that URL at `localhost:4000`; LiteLLM receives it, the LLMLingua-2 hook compresses the context, and it forwards to the real provider. The proxy is in the path — the entire premise of the spike. Mode C also implies real per-token spend, so there is something to save (the verdict's `S = $100/dev/mo` input).

In **Modes A and B**, GitHub's servers make the call, so a laptop-local proxy never sees the traffic. Mode A additionally has no per-token cost at all (flat subscription), so even the economics don't apply.

**The catch:** Mode C is not the default — someone must deliberately configure the custom endpoint, and your Copilot version has to honor it for the model in use (Claude Opus 4.6). The cohort is on the default "Copilot + Opus via the dropdown," which is the Mode A signature. So this check usually comes back A/B, and reaching Mode C means *configuring* it (or doing the tool-swap below), not just detecting it.

## The check (macOS)

Open **Terminal** and paste the whole block. When it prints *"NOW send one Copilot Chat message,"* switch to VS Code and fire a Copilot Chat prompt so there is live traffic to capture during the 12-second window.

```bash
# ===== Copilot routing-mode check (macOS) — read-only, ~2 min =====
S1="$HOME/Library/Application Support/Code/User/settings.json"
S2="$HOME/Library/Application Support/Code - Insiders/User/settings.json"

echo "== [1] Custom endpoint in VS Code settings? (the decisive check) =="
if grep -nEi 'baseUrl|endpoint|copilot\.advanced|anthropic|byok' "$S1" "$S2" ./.vscode/settings.json 2>/dev/null; then
  echo ">> FOUND custom-endpoint keys -> possibly Mode C (proxyable). Confirm with [2]+[3]."
else
  echo ">> Nothing found -> default model picker = Mode A/B. Proxy CANNOT interpose as-is."
fi

echo
echo "== [3] Where is VS Code's traffic going? =="
echo "   >>> NOW send one Copilot Chat message; capturing for ~12s..."
( for i in $(seq 1 24); do
    lsof -PiTCP -sTCP:ESTABLISHED +c 0 2>/dev/null | grep -iE 'code|copilot' | awk '{print $NF}'
    sleep 0.5
  done ) | sort -u | sed 's/^/   /'
echo "   Read: api.anthropic.com -> Mode C (proxyable) | only *.githubcopilot.com / *.github.com -> Mode A/B | only IPs/CDN -> trust [1]."

echo
echo "== [2] MANUAL: per-token Anthropic billing (splits A from B) =="
echo "   https://console.anthropic.com -> Settings -> Billing -> Usage (last 30 days)."
echo "   ~\$100 x dev-count on AITO's key -> per-token exists (B/C) | no usage -> flat-rate (Mode A, S=\$100 wrong)."
```

## Reading the result

- **`[1]` prints a `baseUrl`/endpoint AND `[3]` shows `api.anthropic.com`** → **Mode C**. Proxyable — proceed; this is the path you want for the cohort.
- **`[1]` prints nothing AND `[3]` shows only `*.githubcopilot.com` / `*.github.com`** → **Mode A/B**. Server-routed; the proxy can't interpose as-is. Use `[2]` to split:
  - per-token Anthropic spend exists → **Mode B** → swap the cohort to Claude Code / Cursor / Aider pointed at `localhost:4000` for the measurement (document the caveat: you're measuring that tool, not Copilot).
  - no per-token spend → **Mode A** → `S = $100/dev/mo` is wrong; stop and re-open the M1 gate.
- **`[3]` shows only raw IPs / CDN names** (Cloudflare, Azure) → ignore it and trust `[1]`. The settings check is the reliable signal.

`[1]` alone answers ~90% of it in 10 seconds; `[2]` and `[3]` confirm.

## Reaching Mode C if you're not already there

Two routes to a proxyable state:

1. **Configure true Mode C** — set a custom Anthropic-compatible endpoint in VS Code `settings.json` so Copilot calls out locally, then point it at `localhost:4000`. Only works if your Copilot version honors a custom base URL for Opus 4.6 — confirm before committing the cohort.
2. **Tool-swap (Mode-C-equivalent)** — leave Copilot alone; route Claude Code / Cursor / Aider through `localhost:4000` for the measurement week. Always works; measures a tool that isn't the daily driver (note the caveat).

Either way, also confirm: (a) the per-token spend lands near **`S = $100/dev/mo`** or the verdict math breaks regardless of mode; and (b) each dev actually does **real code-heavy work** through the proxied setup all week, or the data is thin.

## Other platforms

- **Windows:** `powershell -ExecutionPolicy Bypass -File .\check_copilot_mode.ps1`
- **Linux:** `bash check_copilot_mode.sh`

Both live alongside this file in `spike/` and run the same three checks. Full decision table and per-mode actions: `../planning/validate_with_copilot.md`.
