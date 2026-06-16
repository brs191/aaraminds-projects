#!/usr/bin/env bash
# discover-ide-usage.sh — Phase 0.5 data-source discovery for VS Code IDE Copilot usage.
#
# READ-ONLY. ZERO-NETWORK. Makes no changes and contacts no servers.
# It enumerates where Copilot (CLI + VS Code IDE) writes local data and prints a
# REDACTED schema sample so we can build the reader against the real format.
#
# Redaction: long string values and email-like tokens are masked; JSON KEYS and
# NUMERIC values (token counts, credits) are preserved — those are what we need.
#
# Usage:   bash discover-ide-usage.sh > ide-usage-report.txt
# Then paste ide-usage-report.txt back into the chat.

set -uo pipefail
HOME_DIR="${HOME}"
section() { printf '\n========== %s ==========\n' "$1"; }

# Cross-platform: build the list of VS Code "user-data roots" (each holds User/ and logs/).
# macOS uses ~/Library/Application Support; Linux uses ~/.config; remote/WSL uses ~/.vscode-server.
OSNAME="$(uname -s)"
CODE_ROOTS=()
case "$OSNAME" in
  Darwin)
    for v in "Code" "Code - Insiders" "VSCodium"; do
      CODE_ROOTS+=("${HOME_DIR}/Library/Application Support/${v}")
    done ;;
  Linux)
    for v in "Code" "Code - Insiders" "VSCodium"; do
      CODE_ROOTS+=("${HOME_DIR}/.config/${v}")
    done
    # VS Code Remote/WSL server stores user data here (User/ + logs under data/):
    [ -d "${HOME_DIR}/.vscode-server/data" ] && CODE_ROOTS+=("${HOME_DIR}/.vscode-server/data")
    [ -d "${HOME_DIR}/.vscode-server-insiders/data" ] && CODE_ROOTS+=("${HOME_DIR}/.vscode-server-insiders/data") ;;
  *)
    CODE_ROOTS+=("${HOME_DIR}/.config/Code") ;;
esac

# redact: keep JSON keys + numbers; mask long strings and emails.
redact() {
  sed -E \
    -e 's/[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}/<EMAIL>/g' \
    -e 's/("(apiKey|token|authorization|access_token|webhook|url|content|text|prompt|completion|message)" *: *")[^"]*"/\1<REDACTED>"/gi' \
    -e 's/("[A-Za-z0-9_]+" *: *")[^"]{41,}"/\1<STR>"/g'
}

# show schema of a JSONL file: keys present + a redacted first/last record.
sample_jsonl() {
  local f="$1"
  printf -- '--- %s  (%s lines, %s)\n' "$f" "$(wc -l < "$f" 2>/dev/null | tr -d ' ')" "$(du -h "$f" 2>/dev/null | cut -f1)"
  if command -v python3 >/dev/null 2>&1; then
    printf '  top-level keys (union, first 50 lines): '
    head -50 "$f" 2>/dev/null | python3 -c '
import sys,json
keys=set()
for line in sys.stdin:
    line=line.strip()
    if not line: continue
    try: keys|=set(json.loads(line).keys())
    except Exception: pass
print(", ".join(sorted(keys)) or "(no parseable JSON objects)")' 2>/dev/null
    printf '  grep for billing/token fields: '
    grep -hoE '"(totalNanoAiu|nanoAiu|tokens|inputTokens|outputTokens|promptTokens|completionTokens|cachedTokens|usage|model|premiumRequests|credits|cost)"' "$f" 2>/dev/null | sort -u | tr '\n' ' '
    printf '\n  first record (redacted):\n'
    head -1 "$f" 2>/dev/null | redact | cut -c1-1200
    printf '\n  last record (redacted):\n'
    tail -1 "$f" 2>/dev/null | redact | cut -c1-1200
    printf '\n'
  else
    printf '  (python3 not found — raw first/last lines, redacted)\n'
    head -1 "$f" 2>/dev/null | redact | cut -c1-800; echo
    tail -1 "$f" 2>/dev/null | redact | cut -c1-800; echo
  fi
}

section "ENVIRONMENT"
echo "date: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "uname: $(uname -a)"
echo "code CLI present: $(command -v code >/dev/null 2>&1 && echo yes || echo no)"

section "~/.copilot TREE (CLI + any shared streams)"
if [ -d "${HOME_DIR}/.copilot" ]; then
  find "${HOME_DIR}/.copilot" -maxdepth 3 -type d 2>/dev/null | sed "s#${HOME_DIR}#~#"
  echo "--- file types/counts under ~/.copilot:"
  find "${HOME_DIR}/.copilot" -type f 2>/dev/null | sed -E 's/.*(\.[A-Za-z0-9]+)$/\1/' | sort | uniq -c
  echo "--- newest 15 files:"
  find "${HOME_DIR}/.copilot" -type f -print0 2>/dev/null | xargs -0 ls -lt 2>/dev/null | head -15 | sed "s#${HOME_DIR}#~#"
else
  echo "~/.copilot NOT found"
fi

section "~/.copilot/otel SAMPLES (ccusage reads these)"
for f in "${HOME_DIR}"/.copilot/otel/*.jsonl; do [ -e "$f" ] && sample_jsonl "$f"; done

section "~/.copilot OTHER *.jsonl / *.log SAMPLES"
find "${HOME_DIR}/.copilot" -type f \( -name '*.jsonl' -o -name '*.log' \) 2>/dev/null \
  | grep -v '/otel/' | grep -v '/session-state/' | while read -r f; do sample_jsonl "$f"; done

section "VS CODE — Copilot extension storage  (OS: ${OSNAME})"
echo "Scanning user-data roots: ${CODE_ROOTS[*]/#${HOME_DIR}/~}"
for root in "${CODE_ROOTS[@]}"; do
  [ -d "$root" ] || continue
  GS="${root}/User/globalStorage"
  if [ -d "$GS" ]; then
    echo "--- ${root#${HOME_DIR}/}/User/globalStorage entries matching copilot:"
    ls -la "$GS" 2>/dev/null | grep -i copilot | sed "s#${HOME_DIR}#~#"
    find "$GS" -ipath '*copilot*' -type f 2>/dev/null | sed "s#${HOME_DIR}#~#" | head -40
  fi
  WS="${root}/User/workspaceStorage"
  if [ -d "$WS" ]; then
    echo "--- ${root#${HOME_DIR}/}/User/workspaceStorage copilot files (first 20):"
    find "$WS" -ipath '*copilot*' -type f 2>/dev/null | sed "s#${HOME_DIR}#~#" | head -20
  fi
done

section "VS CODE — Copilot logs (diagnostic; check if any carry token/usage)"
for root in "${CODE_ROOTS[@]}"; do
  LOGS="${root}/logs"
  [ -d "$LOGS" ] || continue
  echo "--- ${root#${HOME_DIR}/} newest copilot log files:"
  find "$LOGS" -type f -ipath '*copilot*' 2>/dev/null | xargs ls -lt 2>/dev/null | head -10 | sed "s#${HOME_DIR}#~#"
  newest="$(find "$LOGS" -type f -ipath '*copilot*' 2>/dev/null | xargs ls -t 2>/dev/null | head -1)"
  if [ -n "${newest:-}" ]; then
    echo "--- token/usage/premium mentions in newest copilot log ($(basename "$newest")):"
    grep -ioE '(token[s]?|usage|premium|quota|model|credit)[^,;]{0,40}' "$newest" 2>/dev/null | sort -u | head -25
  fi
done

section "ANY OTHER likely usage DBs (state.vscdb / sqlite)"
for root in "${CODE_ROOTS[@]}"; do
  find "${root}/User" -name 'state.vscdb' 2>/dev/null | sed "s#${HOME_DIR}#~#" | head
done
echo "(If a copilot usage table lives in state.vscdb, note it — it's SQLite.)"

section "DONE"
echo "Paste this whole report back. Nothing was modified; no network calls were made."
