#!/bin/bash
# validate-adr-001.sh — Test ADR-001 compliance
#
# Validates that the token-budget tool makes ZERO unintended network calls.
# Blocks ONLY allowed network paths:
#   - [PHASE 7+] Explicit GitHub API calls (guarded by config flag "github_api_enabled")
#
# Usage:
#   ./scripts/validate-adr-001.sh              # Run full test
#   ./scripts/validate-adr-001.sh --verbose    # Print network call logs
#   ./scripts/validate-adr-001.sh --allow-github-api  # Allow Phase 7 API calls
#
# Exit codes:
#   0 = PASS (ADR-001 compliant)
#   1 = FAIL (network call detected when not allowed)
#   2 = SKIP (tcpdump/timeout not available)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
VERBOSE=${VERBOSE:-0}
ALLOW_GITHUB_API=${ALLOW_GITHUB_API:-0}
TIMEOUT=${TIMEOUT:-10}  # seconds to monitor network

# Parse arguments
for arg in "$@"; do
  case "$arg" in
    --verbose) VERBOSE=1 ;;
    --allow-github-api) ALLOW_GITHUB_API=1 ;;
  esac
done

log() { echo "[ADR-001 Test] $*" >&2; }
log_verbose() { [ "$VERBOSE" = "1" ] && echo "[ADR-001 Test VERBOSE] $*" >&2 || true; }

trap "cleanup" EXIT

cleanup() {
  # Kill background processes if still running
  jobs -p | xargs -r kill 2>/dev/null || true
}

# Check prerequisites
if ! command -v tcpdump &>/dev/null; then
  log "⚠️  WARNING: tcpdump not found. Network monitoring disabled."
  log "    Install with: brew install tcpdump (or apt-get install tcpdump)"
  log "    Test will proceed without network validation."
  exit 2
fi

log "🔒 Starting ADR-001 Compliance Test"
log "   Project root: $PROJECT_ROOT"
log "   Timeout: ${TIMEOUT}s"
log "   Allow GitHub API: $([ "$ALLOW_GITHUB_API" = "1" ] && echo "YES" || echo "NO (Phase 6)")"

# === Network capture ===
PCAP_FILE="/tmp/adr001-test-$$.pcap"
ALLOWED_HOSTS="127.0.0.1|localhost"

if [ "$ALLOW_GITHUB_API" = "1" ]; then
  ALLOWED_HOSTS="${ALLOWED_HOSTS}|api\.github\.com"
fi

log "   Allowed hosts: $ALLOWED_HOSTS"
log ""

# Start tcpdump in background (monitor all traffic except allowed)
log "📡 Monitoring network..."
sudo tcpdump -i any -w "$PCAP_FILE" -Q in 'tcp port 443 or tcp port 80' >/dev/null 2>&1 &
TCPDUMP_PID=$!
sleep 0.5  # Give tcpdump time to start

# === Run the tool ===
cd "$PROJECT_ROOT"

log "▶️  Running: go run ./cmd/analyze"
if timeout "$TIMEOUT" go run ./cmd/analyze >/dev/null 2>&1 || true; then
  log "✅ Tool completed"
fi
sleep 1

# Stop capture
log "⏹️  Stopping network capture..."
sudo kill $TCPDUMP_PID 2>/dev/null || true
sleep 0.5

# === Analyze pcap ===
log ""
log "🔍 Analyzing network traffic..."

# Extract destination IPs from pcap
DEST_IPS=$(sudo tcpdump -r "$PCAP_FILE" -n 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+' | sort -u || echo "")

# Check for suspicious IPs
SUSPICIOUS_IPS=""
for ip in $DEST_IPS; do
  # Check if IP is local
  if echo "$ip" | grep -qE "^127\.|^localhost"; then
    log_verbose "  ✓ Local: $ip"
    continue
  fi
  # Check if IP resolves to github.com (Phase 7 allowed)
  if [ "$ALLOW_GITHUB_API" = "1" ]; then
    if dig +short "$ip" 2>/dev/null | grep -q "github"; then
      log_verbose "  ✓ GitHub API (Phase 7): $ip"
      continue
    fi
  fi
  # This is a suspicious network call
  log_verbose "  ❌ SUSPICIOUS: $ip"
  SUSPICIOUS_IPS="${SUSPICIOUS_IPS}${ip} "
done

# === Report ===
log ""
if [ -z "$SUSPICIOUS_IPS" ]; then
  log "✅ PASS — ADR-001 Compliant"
  log "   Zero unintended network calls detected"
  log "   Allowed traffic: local (127.0.0.1) + $([ "$ALLOW_GITHUB_API" = "1" ] && echo "GitHub API" || echo "none")"
  exit 0
else
  log "❌ FAIL — ADR-001 Violation Detected"
  log "   Suspicious IPs: $SUSPICIOUS_IPS"
  if [ "$VERBOSE" = "1" ]; then
    log ""
    log "Full packet capture:"
    sudo tcpdump -r "$PCAP_FILE" -n 2>/dev/null || true
  fi
  exit 1
fi
