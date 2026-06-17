#!/usr/bin/env bash
# scripts/run.sh — Copilot Token Budget launcher
# Builds the core Go module, runs the one-shot budget report, then launches the live dashboard.
#
# Usage:
#   ./scripts/run.sh                    # defaults to aaraminds-projects workspace
#   ./scripts/run.sh /path/to/workspace # explicit workspace root
set -euo pipefail

# ── Path resolution ────────────────────────────────────────────────────────────
# Resolve SCRIPT_DIR without string manipulation so spaces in paths work.
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
MODULE_DIR="${SCRIPT_DIR}/../core"

if [ -n "${1:-}" ]; then
  WORKSPACE_ROOT="$(cd "$1" && pwd)"
else
  # Default: two levels up from scripts/ = the aaraminds-projects workspace.
  WORKSPACE_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
fi

# ── Stage [1/3]: Pre-flight ────────────────────────────────────────────────────
echo ""
echo "╔══════════════════════════════════════════════════════╗"
echo "║          Copilot Token Budget — Launcher            ║"
echo "╚══════════════════════════════════════════════════════╝"
echo ""
echo "[1/3] Pre-flight checks"

# Verify 'go' is in PATH.
if ! command -v go >/dev/null 2>&1; then
  echo "  ✗ 'go' not found in PATH." >&2
  echo "    Install Go from https://go.dev/dl/ and re-run." >&2
  exit 1
fi
echo "  ✓ Go found: $(go version)"

# Verify go.mod is where we expect it.
if [ ! -f "${MODULE_DIR}/go.mod" ]; then
  echo "  ✗ go.mod not found at ${MODULE_DIR}/go.mod" >&2
  echo "    Is the repository checked out correctly?" >&2
  exit 1
fi
echo "  ✓ go.mod found at ${MODULE_DIR}/go.mod"

# Warn (do not fail) if session-state directory is absent.
SESSION_STATE_DIR="${HOME}/.copilot/session-state"
if [ ! -d "${SESSION_STATE_DIR}" ]; then
  echo "  ⚠ ${SESSION_STATE_DIR} not found — no Copilot sessions detected yet."
  echo "    The tool will show empty data until you run at least one Copilot CLI session."
else
  SESSION_COUNT="$(ls -1 "${SESSION_STATE_DIR}" 2>/dev/null | wc -l | tr -d ' ')"
  echo "  ✓ Session state: ${SESSION_COUNT} session(s) at ${SESSION_STATE_DIR}"
fi

echo "  ✓ Workspace root: ${WORKSPACE_ROOT}"
echo ""

# ── Stage [2/3]: Build ─────────────────────────────────────────────────────────
echo "[2/3] Building Go module"
cd "${MODULE_DIR}"
go build ./...
echo "  ✓ Build succeeded"
echo ""

# ── Stage [3/3]: Analyze ───────────────────────────────────────────────────────
echo "[3/3] Running one-shot budget report"
echo "────────────────────────────────────────────────────────"
go run ./cmd/analyze "${WORKSPACE_ROOT}"
echo "────────────────────────────────────────────────────────"
echo ""
echo "Press Enter to launch live dashboard (Ctrl+C to exit) ..."
read -r

# ── Dashboard ─────────────────────────────────────────────────────────────────
# exec replaces this shell process — Ctrl+C exits cleanly with no orphan processes.
exec go run ./cmd/dashboard "${WORKSPACE_ROOT}"
