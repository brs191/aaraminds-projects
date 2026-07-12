# Podcast Transcript Agent

A Level-2 drafting agent (PRD v1.5, P0 scope) that turns podcast episodes into reviewed, approved transcripts: media intake with ownership attestation, caption pre-check and reuse, audio extraction, batch STT with diarization and confidence flags, LLM cleanup, quality reports, human review with immutable approvals, txt/md/srt/vtt exports with parse-back validation, grounded summaries, and a full append-only audit trail. Go backend (`backend/`), React review UI (`web/`), one server serves both.

## Quickstart

```bash
./scripts/run.sh
# → http://localhost:8080  (PORT=9090 ./scripts/run.sh to change)
```

That builds the UI and backend on first run (needs Go 1.26+ and Node 22+), then starts a single server with mock providers and in-memory storage — no external services, no credentials. `./scripts/build.sh` builds without running.

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
| `STT_PROVIDER` / `LLM_PROVIDER` / `CAPTION_PROVIDER` | `mock` | `azure` / `anthropic` / `youtube` when credentials are wired |

## Docker (Postgres mode) — untested

`docker-compose.yml` + `Dockerfile` build the UI and backend into one distroless image (UI baked in via `WEB_DIST`, migrations via `MIGRATIONS_DIR`) and run it against `postgres:16` with a persistent volume:

```bash
docker compose up --build   # → http://localhost:8080
```

Honesty note: docker is not available in the environment this was authored in, so the compose path is **correct by inspection but untested**. The scripts path above is smoke-tested.

## Architecture

- **Backend** (`backend/`, Go 1.26, stdlib + uuid + pgx only): REST API under `/api/v1` → orchestrator workers drive each job through a tool pipeline (metadata → captions check → audio extract → STT → normalize → quality) with CAS status transitions, retries, stuck-job reclaim, and retention sweep.
- **Providers** are interfaces with mock implementations (deterministic demo) and real skeletons: Azure Speech (STT), Anthropic Claude (cleanup/summaries), YouTube Data API (captions).
- **UI** (`web/`, Vite + React 19 + TanStack Query): submit → job list → tabbed review (transcript editor, summary, quality report, exports, audit), talking to the same origin in production or proxied from the Vite dev server in dev.

More depth: [backend/README.md](backend/README.md) (API contract, env vars, curl flow, mock markers) and [web/README.md](web/README.md) (UI layout, dev identity, role mirroring).

## Status — read before deploying

Mock providers are the default and what the demo uses. Azure Speech, Anthropic, and YouTube captions wiring exists but is **pending credentials** and unverified against live services. Identity is a **dev stub** (client-chosen headers) pending SSO integration — anyone who can reach the port can be anyone. **Do not expose this service to untrusted networks**; front it with an authenticating reverse proxy (`AUTH_PROXY_SECRET`) at minimum.
