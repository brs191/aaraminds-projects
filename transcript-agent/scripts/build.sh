#!/bin/sh
# Build the Podcast Transcript Agent: React UI (vite) + Go backend.
# POSIX sh; works from any cwd (repo root is resolved from the script path).
set -eu

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(dirname -- "$SCRIPT_DIR")

echo "==> web: building UI (vite)"
cd "$ROOT/web"
if [ ! -d node_modules ]; then
    echo "==> web: node_modules missing — running npm ci"
    npm ci
fi
npm run build

echo "==> backend: go build -o bin/server ./cmd/server"
cd "$ROOT/backend"
go build -o bin/server ./cmd/server

echo "==> build complete:"
echo "    $ROOT/backend/bin/server"
echo "    $ROOT/web/dist"
