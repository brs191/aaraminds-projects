#!/bin/sh
# Run the WhisperX STT sidecar on http://127.0.0.1:9090 (override with PORT).
# Requires ./scripts/stt-setup.sh to have been run once. POSIX sh; works from
# any cwd. Pass HF_TOKEN=<hugging-face-token> for speaker diarization; see
# stt-sidecar/README.md. First run downloads models (~1.5GB).
set -eu

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(dirname -- "$SCRIPT_DIR")
SIDECAR="$ROOT/stt-sidecar"

if [ ! -x "$SIDECAR/.venv/bin/uvicorn" ]; then
    echo "==> venv missing — running setup first"
    "$SCRIPT_DIR/stt-setup.sh"
fi

PORT="${PORT:-9090}"

echo ""
echo "WhisperX STT sidecar  →  http://127.0.0.1:$PORT"
echo "  model:       ${WHISPER_MODEL:-large-v3-turbo}"
echo "  diarization: $([ -n "${HF_TOKEN:-}" ] && echo enabled || echo 'disabled (set HF_TOKEN — see stt-sidecar/README.md)')"
echo ""
echo "Point the backend at it:  STT_PROVIDER=whisperx WHISPERX_URL=http://localhost:$PORT ./scripts/run.sh"
echo ""

cd "$SIDECAR"
exec .venv/bin/uvicorn app:app --host 127.0.0.1 --port "$PORT"
