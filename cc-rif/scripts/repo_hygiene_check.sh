#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

echo "[repo-hygiene] checking required migration structure..."
required_paths=(
  "services/ingestion"
  "services/retriever"
  "services/mcp-server"
  "services/embedding-service"
  "services/agent-service"
  "extractors/core-java"
  "extractors/spring-java"
  "libs/graphstore"
  "data/schema"
  "data/migrations"
  "platform/ci"
  "governance"
  "docs/ops"
)

for p in "${required_paths[@]}"; do
  if [[ ! -e "${p}" ]]; then
    echo "[repo-hygiene] missing required path: ${p}" >&2
    exit 1
  fi
done

echo "[repo-hygiene] checking tracked binaries..."
binary_count=0
while IFS= read -r tracked_file; do
  [[ -f "${tracked_file}" ]] || continue
  mime_type="$(file --brief --mime-type "${tracked_file}")"
  case "${mime_type}" in
    application/x-executable|application/x-pie-executable|application/x-mach-binary|application/x-sharedlib)
      echo "[repo-hygiene] tracked binary is not allowed: ${tracked_file} (${mime_type})" >&2
      binary_count=$((binary_count + 1))
      ;;
  esac
done < <(git ls-files)

if [[ "${binary_count}" -gt 0 ]]; then
  exit 1
fi

echo "[repo-hygiene] passed."
