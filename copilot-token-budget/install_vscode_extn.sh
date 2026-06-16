#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
EXT_DIR="${SCRIPT_DIR}/phase-2/vscode-extension"

usage() {
  cat <<'EOF'
Usage:
  ./install_vscode_extn.sh [--vsix /path/to/file.vsix] [--skip-npm-install]

Behavior:
  - No --vsix: builds extension from local source, then installs it.
  - With --vsix: skips build and installs the provided VSIX.

Options:
  --vsix PATH          Install this VSIX file.
  --skip-npm-install   Skip npm install before packaging.
  -h, --help           Show this help.

Dashboard Metrics (Phase 1–7):
  ✨ Copilot Token Budget Dashboard provides comprehensive monitoring across:

  📊 BUDGET OVERVIEW (Top Section)
  • Used Credits / Allowed Credits (Billions format: e.g., 8.55 B / 7.00 B)
  • Remaining Credits (positive = under budget; negative = CRITICAL)
  • Budget Status (OK | WARNING @ 60% | CRITICAL @ 90%)

  🔀 SOURCE BREAKDOWN (Phase 6)
  • CLI Sessions Total (credits from copilot-cli)
  • IDE Sessions Total (credits from VS Code copilot-ide)
  • Combined Total (deduplicated, no double-counting)

  🚀 FORECAST & BURN RATE
  • Daily Burn Rate (credits/day, 7-day rolling average)
  • Projected Month-End Total (linear extrapolation)
  • Verdict (within/OVER allowance)

  📈 USAGE TREND (Last 14 Days)
  • Inline SVG bar chart with daily credits
  • Anomalous days flagged (mean + 2σ threshold)
  • Hover tooltip: date, credits, anomaly status

  👥 TOP CONSUMERS
  • Top 3 Sessions (by credits, name, model)
  • Top 3 Models (by credits, input/output K tokens)
  • Top 3 Projects (by credits, model used)

  🖥️ ACTIVE SESSIONS TABLE
  • Session ID, Project, Model, Credits (Billions), Source (CLI/IDE)
  • Status (Active | Final), Input/Output tokens
  • Context Window % fullness for active models

  ⚙️ INSTRUCTION FILES
  • Loaded .copilot/instructions/ files detected in workspace
  • Token cost estimation (5 Sonnet turns per session default)
  • File path, severity (info/warning/error), auto-include flag

  💾 EXPORT COMMANDS
  • Copilot Budget: Export Usage → JSON (full report, all metrics)
  • Copilot Budget: Export Usage → CSV (sessions, daily, consumers)

  📌 STATUS BAR INDICATOR
  • Real-time badge: $(icon) 💰 Used / Allowed (Billions format)
  • Colour-coded: $(check) OK | $(warning) WARNING | $(circle-filled) CRITICAL
  • Click to open dashboard
  • Hover tooltip: today, month, burn, forecast, context %

  🎯 CONFIGURATION (VS Code Settings)
  • copilotBudget.monthlyAllowance (override 7,000 default)
  • copilotBudget.pricingPath (custom pricing.json)
  • copilotBudget.checkInterval (refresh interval, ms)
  • copilotBudget.teamsWebhook (Microsoft Teams alert URL)

  🔔 TEAMS ALERTS
  • CRITICAL / WARNING alerts sent to configured Teams webhook
  • Alert suppression (no duplicate fires within 1 hour, UTC)
  • Message includes used %, burn rate, projected total

  📊 ANALYTICS (Phase 7)
  • Per-model usage distribution (input/output tokens split)
  • Daily/weekly/monthly series with anomaly flags
  • Context-window utilization % by model
  • Pricing configuration (Sonnet 300/1500, Opus 500/2500, Haiku 100/500 cr/M tokens)

  🌐 CROSS-PLATFORM
  • macOS: ~/.copilot/session-state/ (JSONL events)
  • Linux: ~/.copilot/session-state/ (same JSONL schema)
  • Windows: %APPDATA%/Copilot/session-state/ (coming soon)

  🔐 SECURITY & PRIVACY
  • All data local (zero network calls to monitor usage)
  • No Copilot API calls (reads local session files only)
  • IDE source detected via vscode.metadata.json marker
  • Dedup prevents double-counting across CLI and IDE
  • Teams webhook URL masked in error logs

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

  pushd "$EXT_DIR" >/dev/null

  if [[ $SKIP_NPM_INSTALL -eq 0 ]]; then
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
