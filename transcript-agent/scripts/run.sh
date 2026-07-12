#!/bin/sh
# Run the Podcast Transcript Agent: single server on http://localhost:8080
# serving both the API and the built React UI. Mock providers, in-memory
# store — no external services needed. Override the port with PORT=9090.
# POSIX sh; works from any cwd (repo root is resolved from the script path).
set -eu

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(dirname -- "$SCRIPT_DIR")

if [ ! -x "$ROOT/backend/bin/server" ] || [ ! -f "$ROOT/web/dist/index.html" ]; then
    echo "==> artifacts missing — building first"
    "$SCRIPT_DIR/build.sh"
fi

PORT="${PORT:-8080}"

echo ""
echo "Podcast Transcript Agent  →  http://localhost:$PORT"
echo ""
echo "Demo identities (switcher in the UI header; sent as X-User-Id/X-User-Role):"
echo "  producer-1 / producer   submit jobs, upload media, cancel own jobs"
echo "  reviewer-1 / reviewer   review, edit, approve, generate exports"
echo "  admin-1    / admin      everything, including cancel any job"
echo ""
echo "Storage is in-memory: jobs are lost on restart. Ctrl-C to stop."
echo ""

cd "$ROOT/backend"
PORT="$PORT" \
STORAGE=memory \
STT_PROVIDER=mock \
LLM_PROVIDER=mock \
CAPTION_PROVIDER=mock \
MEDIA_PROVIDER=mock \
DATA_DIR="$ROOT/backend/data" \
WEB_DIST="$ROOT/web/dist" \
exec ./bin/server
