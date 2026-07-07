#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${ROOT_DIR}"

echo "[clean] removing Python virtualenvs and caches..."
for svc in services/agent-service services/embedding-service; do
  rm -rf "${svc}/.venv" "${svc}/build"
  find "${svc}" -type d \( -name "__pycache__" -o -name ".pytest_cache" -o -name ".mypy_cache" -o -name ".ruff_cache" -o -name "*.egg-info" \) -prune -exec rm -rf {} +
done

echo "[clean] removing Java build output..."
find extractors -type d -name target -prune -exec rm -rf {} +

echo "[clean] clearing Go build/test cache..."
for mod in libs/graphstore libs/phase5 services/ingestion services/retriever services/mcp-server; do
  (cd "${mod}" && go clean -cache -testcache)
done

echo "[clean] removing local binaries..."
rm -rf bin dist services/mcp-server/mcp-server services/mcp-server/mcp-server-e2e

echo "[clean] done."
