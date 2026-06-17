#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
EXT_DIR="${SCRIPT_DIR}/../extension"

usage() {
  cat <<'EOF'
Usage:
  ./scripts/install_vscode_extn.sh [--vsix /path/to/file.vsix] [--skip-npm-install]

Behavior:
  - No --vsix: builds extension from local source, then installs it.
  - With --vsix: skips build and installs the provided VSIX.

Options:
  --vsix PATH          Install this VSIX file.
  --skip-npm-install   Skip npm install before packaging.
  -h, --help           Show this help.

What the extension shows (reads local files only — zero network):

  BUDGET OVERVIEW
  • Used / Allowed / Remaining credits (raw credits, e.g. 8,550 / 7,000)
  • Status: OK | WARNING (>=60%) | CRITICAL (>=90%)

  FORECAST & BURN RATE
  • Daily burn rate (month credits / days elapsed)
  • Projected month-end total (used + burn * days remaining)
  • Premium-request count for the month

  USAGE TREND (last 14 days)
  • Inline SVG bar chart; anomalous days flagged (mean + 2 sigma)

  TOP CONSUMERS
  • Top sessions / models / projects by credits
  • Per-model prompt-cache reads (cache-read tokens, where present)

  SESSIONS TABLE
  • Project, model, credits, source, input/output tokens, context-window %
  • Active sessions show a live (not-yet-final) indicator

  INSTRUCTION FILES (.github/instructions/)
  • Per-file token estimate + overhead cost (50-turn session model)
  • Optimization opportunities

  EXPORT
  • "Copilot Budget: Export Usage" -> JSON or CSV (chosen by file extension)

  STATUS BAR
  • Badge: used / allowed credits, colour-coded OK/WARNING/CRITICAL
  • Tooltip: today, month, burn, projected, context %

  SETTINGS (copilotBudget.*)
  • monthlyAllowance, pricingPath, refreshIntervalSec (seconds),
    teamsWebhookUrl, alertThresholdWarn, alertThresholdCrit, alertBinaryPath,
    workspacePath

  TEAMS ALERTS (opt-in)
  • CRITICAL/WARNING Adaptive Card to a configured webhook
  • Deduped: same threshold fires at most once per day (UTC)
  • Use a Power Automate "Workflows" webhook (legacy O365 connector retired ~May 2026)

  CROSS-PLATFORM
  • macOS / Linux: ~/.copilot/session-state/ (JSONL)
  • Windows:       %USERPROFILE%\.copilot\session-state\ (same schema)

  SCOPE NOTE
  • Captures GitHub Copilot **CLI** usage today. VS Code Copilot **Chat** is a
    SEPARATE local source (chatSessions/transcripts under VS Code user data) and is
    NOT captured yet — Phase 6 (see ADR-007). All cost figures are estimates.

EOF
}

require_cmd() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "Error: required command not found: $cmd" >&2
    exit 1
  fi
}

VSIX_PATH=""
SKIP_NPM_INSTALL=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --vsix)
      [[ $# -ge 2 ]] || { echo "Error: --vsix requires a file path" >&2; exit 1; }
      VSIX_PATH="$2"
      shift 2
      ;;
    --skip-npm-install)
      SKIP_NPM_INSTALL=1
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

require_cmd code

if [[ -z "$VSIX_PATH" ]]; then
  if [[ ! -d "$EXT_DIR" ]]; then
    echo "Error: extension directory not found: $EXT_DIR" >&2
    exit 1
  fi

  require_cmd npm
  require_cmd npx

  require_cmd node
  # Packaging uses @vscode/vsce 3.x, which requires Node.js >= 22.
  node_major="$(node -p 'process.versions.node.split(".")[0]' 2>/dev/null || echo 0)"
  if [[ "$node_major" -lt 22 ]]; then
    echo "Error: packaging the .vsix needs Node.js >= 22 (found v$(node -v 2>/dev/null | tr -d 'v'))." >&2
    echo "       Upgrade Node, or pass a prebuilt VSIX with --vsix /path/to/file.vsix." >&2
    exit 1
  fi

  pushd "$EXT_DIR" >/dev/null

  if [[ $SKIP_NPM_INSTALL -eq 0 ]]; then
    # The extension dir ships a .npmrc pointing at the public registry (ADR-003),
    # so plain `npm install` avoids the AT&T Artifactory proxy hang.
    echo "Installing npm dependencies..."
    npm install
  fi

  echo "Packaging VS Code extension..."
  npm run package

  VSIX_PATH="$(ls -1t ./*.vsix 2>/dev/null | head -n 1 || true)"
  popd >/dev/null

  if [[ -z "$VSIX_PATH" ]]; then
    echo "Error: no .vsix file produced in $EXT_DIR" >&2
    exit 1
  fi

  VSIX_PATH="${EXT_DIR}/${VSIX_PATH#./}"
fi

if [[ ! -f "$VSIX_PATH" ]]; then
  echo "Error: VSIX not found: $VSIX_PATH" >&2
  exit 1
fi

echo "Installing extension from: $VSIX_PATH"
code --install-extension "$VSIX_PATH" --force

echo
echo "Done. Open VS Code command palette and run: Copilot Budget: Show Dashboard"
