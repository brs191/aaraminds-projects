# WhisperX STT Sidecar

Local speech-to-text service for the Podcast Transcript Agent. Wraps
[WhisperX](https://github.com/m-bain/whisperX) (faster-whisper transcription +
wav2vec2 word alignment + optional pyannote speaker diarization) behind a
small async HTTP API. The Go backend talks to it via
`STT_PROVIDER=whisperx` (`backend/internal/providers/stt/whisperx/`).

Processing is **one job at a time** (single worker thread + in-process
queue). Results are kept in memory; uploaded temp files are deleted right
after processing and job records expire after 1 hour (`JOB_TTL_SECONDS`).

## Setup

From the repo root (creates `stt-sidecar/.venv`, installs dependencies —
multi-GB download, torch included):

```bash
./scripts/stt-setup.sh
```

Or manually:

```bash
cd stt-sidecar
python3 -m venv .venv
.venv/bin/pip install -r requirements.txt
```

> **First run downloads models (~1.5GB)** for `large-v3-turbo`, plus the
> alignment model on the first job. They are cached under
> `~/.cache/huggingface`. Budget time and disk accordingly.

### Diarization (speaker labels) — optional, needs an HF token

pyannote models are **gated** on Hugging Face:

1. Create a token at <https://huggingface.co/settings/tokens> (read scope).
2. Accept the model terms while logged in:
   - <https://huggingface.co/pyannote/speaker-diarization-3.1>
   - <https://huggingface.co/pyannote/segmentation-3.0>
3. Run the sidecar with `HF_TOKEN=<your token>`.

Without `HF_TOKEN`, `/healthz` reports `"diarization_available": false` and
jobs still succeed — segments carry `"speaker": null` and
`"diarization_applied": false`; the backend then labels everything
`Speaker 1` and flags the transcript for manual speaker review.

## Run

```bash
./scripts/stt-run.sh                      # http://127.0.0.1:9090
HF_TOKEN=hf_xxx ./scripts/stt-run.sh      # with diarization
PORT=9191 ./scripts/stt-run.sh            # different port
```

Or manually: `cd stt-sidecar && .venv/bin/uvicorn app:app --port 9090`.

## Environment variables

| Variable | Default | Purpose |
|---|---|---|
| `PORT` | `9090` | Listen port (consumed by the run script / uvicorn flag) |
| `WHISPER_MODEL` | `large-v3-turbo` | faster-whisper model (`small`, `medium`, `large-v3`, ...) |
| `DEVICE` | auto (`cuda` if available, else `cpu`) | Force `cpu` or `cuda` |
| `COMPUTE_TYPE` | `int8` on cpu / `float16` on cuda | ctranslate2 compute type |
| `HF_TOKEN` | — | Hugging Face token for gated pyannote diarization models |
| `BATCH_SIZE` | `8` | Transcription batch size (lower it on small GPUs / CPU) |
| `JOB_TTL_SECONDS` | `3600` | In-memory job record lifetime |

## GPU notes

- With an NVIDIA GPU, install a CUDA-enabled torch build and run with
  `DEVICE=cuda` (auto-detected when `torch.cuda.is_available()`); ctranslate2
  needs cuBLAS/cuDNN on the library path. `COMPUTE_TYPE` defaults to
  `float16` on cuda.
- CPU works fine but is slow: expect roughly real-time or slower for
  `large-v3-turbo` with `COMPUTE_TYPE=int8`. Use `WHISPER_MODEL=small` for
  quick local testing.
- The provided `Dockerfile` is CPU-only (and untested — see its header). For
  GPU containers, base on an `nvidia/cuda` image with cuDNN and run with
  `--gpus all`.

## API reference (frozen — mirrored by the Go provider)

### `GET /healthz`

```json
{"status": "ok", "model": "large-v3-turbo", "device": "cpu", "diarization_available": true}
```

### `POST /v1/jobs` → `202`

`multipart/form-data`:

| Field | Required | Notes |
|---|---|---|
| `file` | yes | Audio file (wav/mp3/m4a/... — anything ffmpeg decodes) |
| `language` | no | Default `en` |
| `enable_diarization` | no | `"true"` / `"false"`, default `"true"` |
| `min_speakers` / `max_speakers` | no | Diarization bounds (the backend sends `expected_speaker_count` as both) |

```json
{"job_id": "b0c1...", "status": "queued"}
```

### `GET /v1/jobs/{job_id}` → `200` (`404` when unknown/expired)

```json
{
  "job_id": "b0c1...",
  "status": "queued|processing|done|error",
  "error": null,
  "result": {
    "language": "en",
    "duration_seconds": 1234.5,
    "model": "large-v3-turbo",
    "diarization_applied": true,
    "segments": [
      {"start_ms": 0, "end_ms": 4200, "text": "...", "speaker": "SPEAKER_00", "confidence": 0.93}
    ]
  }
}
```

`error` is `{"code": "...", "message": "..."}` when `status` is `error`.
Codes: `LANGUAGE_UNSUPPORTED`, `AUDIO_DECODE_FAILED`, `TRANSCRIBE_FAILED`,
`CANCELLED`. `speaker` and `confidence` are nullable; confidence is
`exp(avg_logprob)` clamped to `[0,1]` (alignment word-score mean as
fallback).

### `DELETE /v1/jobs/{job_id}` → `204`

Cancel/cleanup. Queued jobs are dropped immediately; a job already
processing finishes internally but its result is discarded.

## Smoke test

```bash
curl -s localhost:9090/healthz
JOB=$(curl -s -X POST localhost:9090/v1/jobs -F file=@episode.mp3 -F language=en \
  -F enable_diarization=true | python3 -c 'import sys,json;print(json.load(sys.stdin)["job_id"])')
watch -n 5 "curl -s localhost:9090/v1/jobs/$JOB | head -c 400"
```
