#!/usr/bin/env bash
set -euo pipefail

# Uninstall the Copilot Token Budget VS Code extension.
# Counterpart to install_vscode_extn.sh.

EXT_ID="att-internal.copilot-token-budget"

usage() {
  cat <<EOF
Usage:
  ./scripts/remove_vscode_extn.sh [--purge-config] [--remove-binaries] [--yes]

Behavior:
  - Uninstalls the "${EXT_ID}" VS Code extension (via 'code --uninstall-extension').
  - Does NOT touch your Copilot session data (~/.copilot/session-state) — ever.
  - Optional flags clean up things the tool created on this machine.

Options:
  --purge-config     Also delete the local config dir created by the CLI/extension:
                       macOS/Linux: ~/.config/copilot-token-budget/
                       Windows:     %APPDATA%\\copilot-token-budget\\
                     (holds pricing.json + the Teams-alert dedup state.json — NOT usage data)
  --remove-binaries  Also delete installed Go binaries from ~/bin (and %USERPROFILE%\\bin):
                       copilot-analyze, copilot-dashboard, copilot-statusline,
                       copilot-alert, copilot-budget-mcp (+ .exe on Windows)
  --yes, -y          Do not prompt for confirmation on destructive flags.
  -h, --help         Show this help.

Notes:
  - To also stop the MCP server being registered, remove the "copilot-token-budget"
    entry from your workspace .copilot/mcp.json by hand (we don't edit it for you).
EOF
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Error: required command not found: $1" >&2
    exit 1
  fi
}

PURGE_CONFIG=0
REMOVE_BINARIES=0
ASSUME_YES=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --purge-config)    PURGE_CONFIG=1; shift ;;
    --remove-binaries) REMOVE_BINARIES=1; shift ;;
    -y|--yes)          ASSUME_YES=1; shift ;;
    -h|--help)         usage; exit 0 ;;
    *) echo "Error: unknown argument: $1" >&2; usage; exit 1 ;;
  esac
done

confirm() {
  # confirm "<prompt>" -> returns 0 if the user agrees (or --yes was passed)
  [[ $ASSUME_YES -eq 1 ]] && return 0
  local reply
  read -r -p "$1 [y/N] " reply || true
  [[ "$reply" == "y" || "$reply" == "Y" ]]
}

# Resolve config dir per OS (mirrors Go os.UserConfigDir()).
config_dir() {
  case "$(uname -s)" in
    Darwin) echo "${HOME}/Library/Application Support/copilot-token-budget" ;;
    Linux)  echo "${XDG_CONFIG_HOME:-${HOME}/.config}/copilot-token-budget" ;;
    *)      echo "${APPDATA:-${HOME}/.config}/copilot-token-budget" ;;
  esac
}

# --- 1. Uninstall the extension ---
require_cmd code

if code --list-extensions 2>/dev/null | grep -qx "$EXT_ID"; then
  echo "Uninstalling extension: $EXT_ID"
  code --uninstall-extension "$EXT_ID"
  echo "Extension uninstalled. (Reload or restart VS Code to fully unload it.)"
else
  echo "Extension '$EXT_ID' is not installed — nothing to uninstall."
fi

# --- 2. Optional: purge local config (pricing.json + alert state.json) ---
if [[ $PURGE_CONFIG -eq 1 ]]; then
  CFG="$(config_dir)"
  if [[ -d "$CFG" ]]; then
    if confirm "Delete config dir '$CFG'? (pricing.json + alert state.json)"; then
      rm -rf "$CFG"
      echo "Removed: $CFG"
    else
      echo "Skipped config purge."
    fi
  else
    echo "No config dir at '$CFG' — nothing to purge."
  fi
fi

# --- 3. Optional: remove installed Go binaries ---
if [[ $REMOVE_BINARIES -eq 1 ]]; then
  BIN_DIR="${HOME}/bin"
  bins=(copilot-analyze copilot-dashboard copilot-statusline copilot-alert copilot-budget-mcp)
  found=0
  for b in "${bins[@]}"; do
    for f in "${BIN_DIR}/${b}" "${BIN_DIR}/${b}.exe"; do
      [[ -f "$f" ]] && { echo "  found: $f"; found=1; }
    done
  done
  if [[ $found -eq 1 ]]; then
    if confirm "Delete the Copilot Token Budget binaries listed above from '$BIN_DIR'?"; then
      for b in "${bins[@]}"; do
        rm -f "${BIN_DIR}/${b}" "${BIN_DIR}/${b}.exe"
      done
      echo "Removed binaries from $BIN_DIR."
    else
      echo "Skipped binary removal."
    fi
  else
    echo "No Copilot Token Budget binaries found in '$BIN_DIR' — nothing to remove."
  fi
fi

echo
echo "Done."
echo "Your Copilot session data (~/.copilot/session-state) was left untouched."
[[ $PURGE_CONFIG -eq 0 ]] && echo "Tip: pass --purge-config to also remove pricing.json + alert state; --remove-binaries for the CLI tools."
