#!/usr/bin/env bash
set -euo pipefail

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "Error: this installer is macOS-only." >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUNDLE_DIR=""
BIN_DIR="${HOME}/bin"
CODE_CMD="code"
UPDATE_PATH=1
INSTALL_EXTENSION=1

usage() {
  cat <<'EOF'
Usage:
  ./install.sh [--bundle DIR] [--bin-dir DIR] [--code-cmd code] [--no-path-update] [--skip-extension]

Behavior:
  - Installs the macOS binaries for your host architecture into ~/bin by default.
  - Installs the VS Code .vsix from the bundle unless --skip-extension is passed.
  - Adds ~/bin to your shell PATH via ~/.zprofile unless --no-path-update is passed.

Options:
  --bundle DIR        Bundle root containing binaries/, extension/, and manifest.json.
  --bin-dir DIR       Target directory for the binaries. Default: ~/bin
  --code-cmd CMD      VS Code CLI command. Default: code
  --no-path-update    Do not update shell profile PATH entries.
  --skip-extension    Skip installing the VS Code extension.
  -h, --help          Show this help.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --bundle)
      [[ $# -ge 2 ]] || { echo "Error: --bundle requires a directory" >&2; exit 1; }
      BUNDLE_DIR="$2"
      shift 2
      ;;
    --bin-dir)
      [[ $# -ge 2 ]] || { echo "Error: --bin-dir requires a directory" >&2; exit 1; }
      BIN_DIR="$2"
      shift 2
      ;;
    --code-cmd)
      [[ $# -ge 2 ]] || { echo "Error: --code-cmd requires a command" >&2; exit 1; }
      CODE_CMD="$2"
      shift 2
      ;;
    --no-path-update)
      UPDATE_PATH=0
      shift
      ;;
    --skip-extension)
      INSTALL_EXTENSION=0
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Error: unknown argument: $1" >&2
      usage
      exit 1
      ;;
  esac
done

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Error: required command not found: $1" >&2
    exit 1
  fi
}

require_cmd cp
require_cmd mkdir
require_cmd chmod

resolve_bundle_dir() {
  local candidate

  if [[ -n "$BUNDLE_DIR" ]]; then
    candidate="$BUNDLE_DIR"
  elif [[ -f "${SCRIPT_DIR}/manifest.json" && -d "${SCRIPT_DIR}/binaries" ]]; then
    candidate="$SCRIPT_DIR"
  elif [[ -f "${SCRIPT_DIR}/../manifest.json" && -d "${SCRIPT_DIR}/../binaries" ]]; then
    candidate="$(cd "${SCRIPT_DIR}/.." && pwd)"
  else
    echo "Error: bundle directory not found. Pass --bundle /path/to/copilot-token-budget-macos-<version>." >&2
    exit 1
  fi

  if [[ ! -f "${candidate}/manifest.json" || ! -d "${candidate}/binaries" ]]; then
    echo "Error: invalid bundle layout at: ${candidate}" >&2
    exit 1
  fi

  printf '%s\n' "$candidate"
}

bundle_root="$(resolve_bundle_dir)"

case "$(uname -m)" in
  arm64) host_arch="darwin_arm64" ;;
  x86_64) host_arch="darwin_amd64" ;;
  *)
    echo "Error: unsupported macOS architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

install_path() {
  local src="$1"
  local dst="$2"
  mkdir -p "$(dirname "$dst")"
  cp "$src" "$dst"
  chmod +x "$dst"
  if command -v xattr >/dev/null 2>&1; then
    xattr -d com.apple.quarantine "$dst" 2>/dev/null || true
  fi
}

ensure_shell_path() {
  [[ $UPDATE_PATH -eq 1 ]] || return 0

  local profile="${HOME}/.zprofile"
  local marker="# Copilot Token Budget macOS bootstrap"
  local line='export PATH="$HOME/bin:$PATH"'

  if [[ -f "$profile" ]] && grep -Fqx "$line" "$profile"; then
    return 0
  fi

  {
    echo "$marker"
    echo "$line"
    echo
  } >> "$profile"
}

BIN_SOURCE_DIR="${bundle_root}/binaries/${host_arch}"
if [[ ! -d "$BIN_SOURCE_DIR" ]]; then
  echo "Error: missing architecture bundle: $BIN_SOURCE_DIR" >&2
  exit 1
fi

mkdir -p "$BIN_DIR"

bins=(
  copilot-analyze
  copilot-dashboard
  copilot-statusline
  copilot-alert
  copilot-budget-mcp
)

for bin in "${bins[@]}"; do
  src="${BIN_SOURCE_DIR}/${bin}"
  if [[ ! -f "$src" ]]; then
    echo "Error: missing binary in bundle: $src" >&2
    exit 1
  fi
  install_path "$src" "${BIN_DIR}/${bin}"
done

if [[ $INSTALL_EXTENSION -eq 1 ]]; then
  require_cmd "$CODE_CMD"
  shopt -s nullglob
  vsix_candidates=("${bundle_root}/extension"/*.vsix)
  shopt -u nullglob
  if [[ ${#vsix_candidates[@]} -ne 1 ]]; then
    echo "Error: expected exactly one VSIX in ${bundle_root}/extension/" >&2
    exit 1
  fi
  if command -v xattr >/dev/null 2>&1; then
    xattr -d com.apple.quarantine "${vsix_candidates[0]}" 2>/dev/null || true
  fi
  "$CODE_CMD" --install-extension "${vsix_candidates[0]}" --force
fi

ensure_shell_path

echo "Installed macOS bundle from: ${bundle_root}"
echo "Binaries: ${BIN_DIR}"
echo "Architecture: ${host_arch}"
echo "Next: open VS Code and run 'Copilot Budget: Show Dashboard'"
echo "Caveman demo (optional): run ./launch-caveman-demo.sh to open examples/token-optimization-demo"
