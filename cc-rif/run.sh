#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${ROOT_DIR}"

for cmd in uv go mvn; do
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    echo "[install] missing required command: ${cmd}" >&2
    exit 1
  fi
done

echo "[install] syncing Python environments..."
(cd services/agent-service && uv sync --dev)
(cd services/embedding-service && uv sync --dev)

echo "[install] downloading Go module dependencies..."
for mod in libs/graphstore libs/phase5 services/ingestion services/retriever services/mcp-server; do
  (cd "${mod}" && go mod download)
done

echo "[install] downloading Maven dependencies..."
mvn -q -f extractors/core-java/pom.xml -DskipTests dependency:go-offline
mvn -q -f extractors/spring-java/pom.xml -DskipTests dependency:go-offline

echo "[install] done."
