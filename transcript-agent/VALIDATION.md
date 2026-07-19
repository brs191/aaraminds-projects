# Validation Guide — Podcast Transcript Agent

Run these phases in order. Phases 1–2 take ~10 minutes and need no AI models.
Phase 4 is the one that answers the real question: transcript quality on your hardware.

## Phase 0 — Prerequisites

| Requirement | Check | Notes |
|---|---|---|
| Go 1.26+ | `go version` | backend build/tests |
| Node 20+ | `node --version` | UI build |
| Python 3.10+ | `python3 --version` | WhisperX sidecar |
| ffmpeg | `ffmpeg -version` | **required for real media** — `sudo apt install ffmpeg` |
| ~5 GB free disk | | Python deps (~3 GB) + models (~1.5 GB) |
| Hugging Face token (optional) | huggingface.co → Settings → Tokens | needed only for speaker labels (diarization) |

For the HF token: accept the terms on both model pages first —
`pyannote/speaker-diarization-3.1` and `pyannote/segmentation-3.0` — then create a **read** token.

## Phase 1 — Automated gates (~2 min)

```bash
cd ~/projects/aaraminds-projects/transcript-agent
(cd backend && go test -race -count=1 ./...)   # expect: all packages ok
(cd web && npm install && npm run build)        # expect: build green
```

Pass = every backend package `ok`, zero failures; vite build completes.

## Phase 2 — Mock-mode smoke (~5 min, no AI)

```bash
./scripts/run.sh    # → http://localhost:8080
```

In the browser (pick an identity in the header first):

1. Submit → advanced URI → `mock://demo` → attest ownership → Submit.
2. Job reaches `in_review` in seconds. Open it: segments visible, ~3 amber low-confidence highlights, audio player plays a silent WAV.
3. Switch identity to `reviewer-1` → Start review → edit one segment → Approve.
4. Exports tab → generate all four formats → all `passed` → download the `.vtt` (starts with `WEBVTT`).
5. Submit a YouTube-type job with URI containing `captions=1` → it pauses at *caption decision* → choose Reuse → transcript appears with a caption-origin banner, no confidence highlights.
6. Audit tab: full chain from `job.submitted` to `transcript.approved`.

Pass = all six behave as described. Stop the server (Ctrl-C).

## Phase 3 — Sidecar setup (one-time, ~15 min + downloads)

```bash
./scripts/stt-setup.sh                  # creates stt-sidecar/.venv, installs whisperx (multi-GB)
HF_TOKEN=hf_xxx ./scripts/stt-run.sh    # sidecar on :9090 (omit HF_TOKEN to skip diarization)
curl http://localhost:9090/healthz      # expect {"status":"ok",...,"diarization_available":true}
```

First transcription downloads ~1.5 GB of models — that's once.

## Phase 4 — Real transcription (the validation that matters)

In a second terminal:

```bash
MEDIA_PROVIDER=ffmpeg STT_PROVIDER=whisperx ./scripts/run.sh
```

> `MEDIA_PROVIDER=ffmpeg` is critical — the default mock media provider generates
> silent audio, and WhisperX would faithfully transcribe silence.

1. **Start small**: upload a 2–5 minute mp3 of real speech (a voice memo works). CPU with the default `large-v3-turbo` runs roughly real-time; a GPU is much faster. If CPU is painful: restart the sidecar with `WHISPER_MODEL=distil-large-v3`.
2. When it hits `in_review`, judge it against the PRD §17.2 targets:

| Check | Target | How |
|---|---|---|
| Word accuracy | ≲10% errors on clean speech | read 2–3 minutes against the audio |
| Timestamps | ≤2 s median drift | click 5 segments, does audio land on the words? |
| Speakers | ≥80% attribution on 2-speaker audio | are turns split at the right places? |
| Confidence flags | flags land on genuinely unclear audio | play the amber segments |
| Cleanup | no meaning changes | diff raw vs clean versions |

3. Then run one full-length episode (30–60 min) and repeat the spot checks.

Pass = you'd rather review this transcript than type one. Record what you find —
this doubles as episode 1 of the PRD's 3-episode pilot gate.

## Phase 5 — Library mode

With both processes still running:

1. Find a real feed URL: podcastindex.org → search a show you follow → copy RSS URL.
2. UI → Library → add feed (auto-transcribe on) → Poll now → episodes appear.
3. Transcribe one episode → wait for `drafted` (a 1-hour episode ≈ its runtime on CPU) → open it: transcript + auto-summary, **no approval step** — that's correct for library jobs.
4. Library → Search → search a phrase you remember from the episode → click the result → lands on the exact segment, audio seeks there.
5. Delete the feed → episodes leave the list, but the transcript stays reachable via Jobs and Search (by design).

## Phase 6 — Optional: Postgres via Docker

```bash
docker compose up --build
```

Honest label: this path is **untested** (built by inspection). Treat this run as its
first test. If it works, full-text search quality is noticeably better than in-memory,
and data survives restarts. Report anything broken.

## Troubleshooting

- **Sidecar 403 on model download** → you haven't accepted the pyannote terms on both HF pages.
- **`diarization_available: false`** → HF_TOKEN not set; transcripts still work, all segments labeled Speaker 1.
- **First job seems stuck in `transcribing`** → sidecar is downloading models; watch its terminal. Poll timeout is 2 h.
- **Transcript full of "..." / empty** → you're on the mock media provider; set `MEDIA_PROVIDER=ffmpeg`.
- **Port conflicts** → `PORT=8090 ./scripts/run.sh`, sidecar: `PORT=9091 ./scripts/stt-run.sh` + `WHISPERX_URL=http://localhost:9091`.
- **Job failed with STT_PROVIDER_TIMEOUT** → sidecar not running/unreachable; start it, then re-submit.
