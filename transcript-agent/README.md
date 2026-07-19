# Podcast Transcript Agent

A Level-2 drafting agent (PRD v1.5, P0 scope) that turns podcast episodes into reviewed, approved transcripts: media intake with ownership attestation, caption pre-check and reuse, audio extraction, batch STT with diarization and confidence flags, LLM cleanup, quality reports, human review with immutable approvals, txt/md/srt/vtt exports with parse-back validation, grounded summaries, and a full append-only audit trail. Go backend (`backend/`), React review UI (`web/`), one server serves both.

## Quickstart

```bash
./scripts/run.sh
# → http://localhost:8080  (PORT=9090 ./scripts/run.sh to change)
```

That builds the UI and backend on first run (needs Go 1.26+ and Node 22+), then starts a single server with mock providers and in-memory storage — no external services, no credentials. `./scripts/build.sh` builds without running.

**Real transcription (local, no cloud account):** the WhisperX sidecar (`stt-sidecar/`) runs Whisper + speaker diarization on your machine:

```bash
./scripts/stt-setup.sh          # once: venv + deps (multi-GB; Python 3.10+)
./scripts/stt-run.sh            # sidecar on :9090 (HF_TOKEN=... enables speaker labels)
STT_PROVIDER=whisperx ./scripts/run.sh
```

First transcription downloads the whisper model (~1.5GB). Diarization needs a free Hugging Face token for the gated pyannote models — steps in [stt-sidecar/README.md](stt-sidecar/README.md); without it, jobs still succeed single-speaker.

### Library mode: add a feed → auto transcript → search

Library mode (personal-use extension) subscribes to open podcast RSS feeds,
transcribes episodes automatically, and makes every transcript full-text
searchable — no review gate, transcripts are readable as soon as they are
drafted:

```bash
B=http://localhost:8080/api/v1
H='-H "Content-Type: application/json" -H "X-User-Id: me" -H "X-User-Role: producer"'

# 1. Add a feed (validated by fetching it; existing episodes are backfilled
#    but not transcribed). auto_transcribe picks up NEW episodes on each poll.
curl -s -X POST $B/library/feeds -H 'Content-Type: application/json' \
  -H 'X-User-Id: me' -H 'X-User-Role: producer' \
  -d '{"feed_url":"https://example.com/podcast.xml","auto_transcribe":true}'

# 2. List episodes; transcribe any of them on demand (the poller handles new
#    ones automatically every LIBRARY_POLL_INTERVAL, default 30m).
curl -s "$B/library/episodes" -H 'X-User-Id: me' -H 'X-User-Role: producer'
curl -s -X POST $B/library/episodes/<episodeID>/transcribe \
  -H 'Content-Type: application/json' -H 'X-User-Id: me' -H 'X-User-Role: producer' -d '{}'

# 3. The job downloads the enclosure, transcribes, and stops at `drafted` with
#    an auto-generated summary. Search across all library transcripts:
curl -s "$B/library/search?q=approval+gates" -H 'X-User-Id: me' -H 'X-User-Role: producer'
```

Ownership is recorded as `open_rss_personal_use` (personal-use basis for open
RSS enclosures) instead of the manual attestation. Postgres storage is
recommended for search (tsvector GIN index); the in-memory store falls back to
substring matching. Env knobs: `LIBRARY_POLL_INTERVAL` (30m),
`LIBRARY_AUTO_PER_POLL` (3), `LIBRARY_MAX_DOWNLOAD_BYTES` (500 MiB). Details
in [backend/README.md](backend/README.md#library-mode-personal-use-extension).

## Demo walkthrough

1. Open http://localhost:8080. Use the **identity switcher** in the header — `producer-1` (submits), `reviewer-1` (approves), `admin-1` (everything). It sets the `X-User-Id` / `X-User-Role` headers; there is no real login (see status note below).
2. As **producer-1**, submit a job: pick a real `.mp3`/`.m4a`/`.wav`/`.mp4`/`.mov` file on the Submit page (it is staged via `POST /api/v1/uploads`), attest ownership, submit. The mock pipeline finishes in seconds and the job lands in `in_review`.
3. Advanced: instead of a file, submit `mock://` source URIs with behavior markers to exercise the edge paths:
   - `https://www.youtube.com/watch?v=demo&captions=1` (source type **youtube**) — authorized official captions found; job pauses at `needs_user_action` for your caption **reuse vs. re-transcribe** decision.
   - `mock://uploads/noaudio-clip.mp4` — no audio track; job parks in `needs_user_action` / replace media.
   - `mock://uploads/stt-quota-show.mp3` — STT quota exhausted; job returns to `queued` and the queue pauses.
4. Switch to **reviewer-1**: open the job, review the transcript (low-confidence segments are flagged against the 0.80 threshold), edit segments, then **Approve**. Approval is immutable; re-approving supersedes prior exports.
5. Generate **exports** (txt/md/srt/vtt) and download them; check the Summary, Quality, and Audit tabs.

## Configuration

All configuration is environment variables. The important ones (full list in [backend/README.md](backend/README.md)):

| Variable | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | HTTP listen port |
| `WEB_DIST` | `../web/dist` (if it exists) | Directory with the built UI, served at `/` with SPA fallback; unset/missing = API-only |
| `STORAGE` | `memory` | `memory` or `postgres` (needs `DATABASE_URL`) |
| `DATA_DIR` | `./data` | Local object store (audio, captions, exports) |
| `SIGNING_SECRET` | random per boot | HMAC key for signed download/audio links (15-min TTL). Unset = links die on restart |
| `AUTH_PROXY_SECRET` | — | Shared secret a trusted reverse proxy must send; leave unset only for local dev |
| `RETENTION_DAYS` | `30` | Media artifact retention (approved policy default); expired artifacts are swept and audited. Exports and approved transcripts are never swept |
| `MAX_DURATION_SECONDS` | `0` (disabled) | Media duration cap; over-limit jobs park in `needs_user_action` |
| `MAX_UPLOAD_BYTES` | 2 GiB | Upload body limit |
| `STT_PROVIDER` | `mock` | `whisperx` (local sidecar, the default real-STT path) or `azure` (config-gated alternative, needs Azure credentials) |
| `WHISPERX_URL` | `http://localhost:9090` | WhisperX sidecar URL for `STT_PROVIDER=whisperx` |
| `LLM_PROVIDER` / `CAPTION_PROVIDER` | `mock` | `anthropic` / `youtube` when credentials are wired |

## Docker (Postgres mode) — untested

`docker-compose.yml` + `Dockerfile` build the UI and backend into one distroless image (UI baked in via `WEB_DIST`, migrations via `MIGRATIONS_DIR`) and run it against `postgres:16` with a persistent volume:

```bash
docker compose up --build   # → http://localhost:8080
```

Honesty note: docker is not available in the environment this was authored in, so the compose path is **correct by inspection but untested**. The scripts path above is smoke-tested.

## Architecture

- **Backend** (`backend/`, Go 1.26, stdlib + uuid + pgx only): REST API under `/api/v1` → orchestrator workers drive each job through a tool pipeline (metadata → captions check → audio extract → STT → normalize → quality) with CAS status transitions, retries, stuck-job reclaim, and retention sweep.
- **Providers** are interfaces with mock implementations (deterministic demo) and real implementations: WhisperX sidecar (local STT with diarization, `stt-sidecar/`), Azure Speech (STT, config-gated alternative), Anthropic Claude (cleanup/summaries), YouTube Data API (captions).
- **UI** (`web/`, Vite + React 19 + TanStack Query): submit → job list → tabbed review (transcript editor, summary, quality report, exports, audit), talking to the same origin in production or proxied from the Vite dev server in dev.

More depth: [backend/README.md](backend/README.md) (API contract, env vars, curl flow, mock markers) and [web/README.md](web/README.md) (UI layout, dev identity, role mirroring).

## Status — read before deploying

Mock providers are the default and what the demo uses. For real transcription, the **local WhisperX sidecar is the default path** (`STT_PROVIDER=whisperx`, no cloud account needed — see Quickstart); the sidecar's Go provider has full contract-test coverage, but the Python service itself is correct-by-inspection pending a run on a machine that can install torch/whisperx. Azure Speech remains a config-gated alternative (`STT_PROVIDER=azure`); it, Anthropic, and YouTube captions wiring exist but are **pending credentials** and unverified against live services. Identity is a **dev stub** (client-chosen headers) pending SSO integration — anyone who can reach the port can be anyone. **Do not expose this service to untrusted networks**; front it with an authenticating reverse proxy (`AUTH_PROXY_SECRET`) at minimum.
