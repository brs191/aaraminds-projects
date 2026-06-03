#!/usr/bin/env bash
# Copilot routing-mode check (Mac / Linux) - Token Optimizer M0-lite pre-flight.
# Run on ONE developer machine. Canonical decision table: ../planning/validate_with_copilot.md
#
#   baseUrl set? | per-token Anthropic billing? | traffic to          | Mode | Action
#   No           | No                           | *.githubcopilot.com | A    | STOP (proxy blind, S wrong)
#   No           | Yes                          | *.githubcopilot.com | B    | Swap cohort to Claude Code / Cursor
#   Yes          | Yes                          | api.anthropic.com   | C    | PROCEED (proxyable)
#
# Usage:
#   1) Open VS Code, send ONE Copilot Chat message so a request is in flight.
#   2) Within ~20s:  bash check_copilot_mode.sh

set -uo pipefail
echo "==== Copilot routing-mode check ===="

# --- Check 1 (DECISIVE): custom endpoint in VS Code settings ---
echo
echo "[1] Scanning VS Code settings for a custom endpoint / baseUrl..."
case "$(uname -s)" in
  Darwin) UA="$HOME/Library/Application Support/Code/User/settings.json"
          UI="$HOME/Library/Application Support/Code - Insiders/User/settings.json" ;;
  *)      UA="$HOME/.config/Code/User/settings.json"
          UI="$HOME/.config/Code - Insiders/User/settings.json" ;;
esac
WS=".vscode/settings.json"
PAT='baseUrl|endpoint|copilot\.advanced|anthropic|azure|byok|github\.copilot\.chat'
found=0
for f in "$UA" "$UI" "$WS"; do
  if [ -f "$f" ]; then
    echo "  file: $f"
    if hits=$(grep -nEi "$PAT" "$f"); then
      found=1
      echo "$hits" | sed 's/^/    > /'
    else
      echo "    (no endpoint/baseUrl/byok keys found)"
    fi
  else
    echo "  file: $f  (not present)"
  fi
done
if [ "$found" -eq 0 ]; then
  echo "  RESULT: no custom baseUrl/endpoint -> consistent with Mode A or B (default model picker)."
else
  echo "  RESULT: custom endpoint keys present -> possibly Mode C (confirm traffic + billing)."
fi

# --- Check 3 (confirmatory): where does VS Code traffic go? ---
echo
echo "[3] Observing VS Code outbound connections (trigger a Copilot request NOW)..."
tmp=$(mktemp)
for i in $(seq 1 20); do
  if command -v lsof >/dev/null 2>&1; then
    lsof -PiTCP -sTCP:ESTABLISHED 2>/dev/null | grep -iE 'code|copilot|electron' \
      | awk '{print $9}' | sed 's/.*->//' >> "$tmp"
  elif command -v ss >/dev/null 2>&1; then
    ss -tnp 2>/dev/null | grep -iE 'code|electron' | awk '{print $5}' >> "$tmp"
  fi
  sleep 0.75
done
remotes=$(sort -u "$tmp" | grep -vE '127\.0\.0\.1|::1|^$' || true); rm -f "$tmp"
if [ -z "$remotes" ]; then
  echo "  No external connections captured. Re-run WHILE a Copilot request is in flight."
else
  echo "  Remote endpoints seen:"; echo "$remotes" | sed 's/^/    /'
  if echo "$remotes" | grep -qiE 'anthropic'; then
    echo "  -> Direct anthropic traffic: consistent with Mode C (PROXYABLE)."
  elif echo "$remotes" | grep -qiE 'githubcopilot|copilot|github'; then
    echo "  -> Only GitHub/Copilot hosts: consistent with Mode A or B (NOT proxyable)."
  else
    echo "  -> Only CDN/IP names matched; network check is confirmatory only - trust [1] + [2]."
  fi
fi

# --- Check 2 (MANUAL): per-token Anthropic billing distinguishes A vs B ---
echo
echo "[2] MANUAL - confirm per-token Anthropic billing (distinguishes Mode A from B):"
echo "    https://console.anthropic.com -> Settings -> Billing -> Usage (last 30 days)."
echo "    Spend ~\$100 x dev-count on AITO's key -> per-token exists (Mode B or C)."
echo "    No AITO account / no usage            -> flat-rate only (Mode A: S=\$100 is wrong)."
echo
echo "==== Combine [1] baseUrl + [2] billing + [3] traffic -> Mode -> Action ===="
echo "  No baseUrl + default picker is the Mode A signature: the build as scoped cannot intercept."
echo "  Full decision table: ../planning/validate_with_copilot.md"
