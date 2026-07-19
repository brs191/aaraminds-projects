#!/bin/sh
# One-time setup for the WhisperX STT sidecar: create stt-sidecar/.venv and
# install dependencies (multi-GB: torch, whisperx, pyannote). POSIX sh; works
# from any cwd. Re-runnable. See stt-sidecar/README.md for the HF_TOKEN steps
# required by speaker diarization.
set -eu

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROOT=$(dirname -- "$SCRIPT_DIR")
SIDECAR="$ROOT/stt-sidecar"

PYTHON="${PYTHON:-python3}"
if ! command -v "$PYTHON" >/dev/null 2>&1; then
    echo "error: $PYTHON not found (need Python 3.10+)" >&2
    exit 1
fi

if [ ! -x "$SIDECAR/.venv/bin/pip" ]; then
    echo "==> creating venv at stt-sidecar/.venv"
    "$PYTHON" -m venv "$SIDECAR/.venv"
fi

echo "==> installing requirements (this pulls torch + whisperx; several GB)"
"$SIDECAR/.venv/bin/pip" install --upgrade pip
"$SIDECAR/.venv/bin/pip" install -r "$SIDECAR/requirements.txt"

echo ""
echo "Done. Start the sidecar with:  ./scripts/stt-run.sh"
echo "Diarization (speaker labels) needs HF_TOKEN — see stt-sidecar/README.md."
echo "First transcription downloads the whisper model (~1.5GB)."
