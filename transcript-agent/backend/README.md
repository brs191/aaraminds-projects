# Podcast Transcript Agent — Backend (PRD v1.5 P0)

Go 1.26 backend for the Podcast Transcript Agent MVP. Implements the P0 scope
of `../Podcast_Transcript_Agent_PRD.md`: intake with ownership attestation,
caption pre-check and reuse, audio extraction, batch STT with diarization and
confidence flags, raw/clean transcript versions, quality reports, human review
and immutable approval, txt/md/srt/vtt exports with parse-back validation,
summaries, and a full append-only audit trail. Level 2 drafting agent only —
`publish_caption_file` is a stub that always returns `DISABLED_IN_MVP`.

Dependencies: stdlib `net/http` (Go 1.22+ method routing), `log/slog`,
`github.com/google/uuid`, `github.com/jackc/pgx/v5`. Nothing else.

## Run

```bash
make run          # in-memory storage + mock providers on :8080
make test         # go test ./...
make vet build    # static checks + bin/server
```

With the React UI (`../web`): start the backend on `:8080`, then run the UI
dev server (`npm run dev`, Vite on `:5173`). CORS for `http://localhost:5173`
is allowed by default; the UI sends `X-User-Id` / `X-User-Role` headers.

## Environment variables

| Variable | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | HTTP listen port |
| `STORAGE` | `memory` | `memory` or `postgres` |
| `DATABASE_URL` | — | pgx DSN, required for `STORAGE=postgres` |
| `MIGRATIONS_DIR` | `migrations` | SQL migration files (applied at boot, `schema_migrations` table) |
| `DATA_DIR` | `./data` | Local object store root (audio, captions, exports) |
| `STT_PROVIDER` | `mock` | `mock` or `azure` (Azure Speech batch skeleton, needs `AZURE_SPEECH_REGION`/`AZURE_SPEECH_KEY`) |
| `LLM_PROVIDER` | `mock` | `mock` or `anthropic` (skeleton, needs `ANTHROPIC_API_KEY`) |
| `CAPTION_PROVIDER` | `mock` | `mock` or `youtube` (Data API v3 skeleton, needs `YOUTUBE_OAUTH_TOKEN`, `YOUTUBE_CHANNEL_OWNED=true`) |
| `MEDIA_PROVIDER` | `mock` | `mock`, or `ffmpeg`/`auto` (uses ffmpeg/ffprobe when on PATH) |
| `CORS_ORIGIN` | `http://localhost:5173` | Allowed browser origin |
| `AUTH_PROXY_SECRET` | — | Optional shared secret required in `X-Auth-Proxy-Secret` from a trusted auth proxy; leave unset only for local dev |
| `SIGNING_SECRET` | random per boot | HMAC-SHA256 key for signed download/audio links (15-min TTL). When unset the server warns and uses a random per-boot secret — signed links stop working on restart |
| `MAX_UPLOAD_BYTES` | `2147483648` (2 GiB) | Body limit for `POST /api/v1/uploads` (413 `REQUEST_TOO_LARGE` above it) |
| `WORKERS` | `2` | Orchestrator worker goroutines |
| `REQUEUE_INTERVAL` | `3s` | Scan interval for submitted/queued jobs |
| `RETRY_BACKOFF` | `2s` | Wait before the single retry of retryable failures |
| `DEFAULT_CONFIDENCE_THRESHOLD` | `0.80` | job_config snapshot default (PRD R5) |
| `DEFAULT_SUMMARY_MAX_WORDS` | `150` | job_config snapshot default (PRD R10) |
| `DEFAULT_SUMMARY_STYLE` | `neutral-professional` | job_config snapshot default |
| `DEFAULT_STYLE_POLICY_ID` | `default-clean-v1` | Cleanup policy id (PRD 15.2) |

All configuration is snapshotted into `job_config` when a job enters
`validating`; tools read the snapshot via `job_config_id` and never accept
thresholds/style/summary parameters as inputs (PRD 13.2 rule 7).

The HTTP server runs with `ReadTimeout=30s`, `WriteTimeout=120s` (audio
streaming headroom), `IdleTimeout=120s`, and a 1 MiB body cap on every JSON
route (413 `REQUEST_TOO_LARGE`); only `POST /api/v1/uploads` uses the larger
`MAX_UPLOAD_BYTES` limit.

## Uploads, signed links, audio playback

- `POST /api/v1/uploads` (auth required) — `multipart/form-data` field
  `file`, streamed to the object store under `uploads/<uuid><ext>` →
  `201 {"upload_uri":"upload://<uuid>","filename","size_bytes","mime_type"}`.
  Extensions outside mp3/m4a/wav/mp4/mov → 400 `UNSUPPORTED_FORMAT`; oversize
  → 413 `REQUEST_TOO_LARGE`.
- `POST /api/v1/jobs` with `source_type=upload` now **requires** an
  `upload://` URI that resolves to a staged artifact (or a `mock://` URI in
  mock/demo mode). Raw filesystem paths and `file://` URIs are rejected with
  400 `INVALID_SOURCE_URI` — the backend never reads arbitrary server paths.
- `POST /api/v1/signed-links` (auth required, any role) —
  `{"kind":"export"|"audio","id":"<exportID or jobID>"}` →
  `201 {"url":"...?token=...","expires_at":"RFC3339"}`. Tokens are
  HMAC-SHA256 over `kind|id|expiry-unix` keyed by `SIGNING_SECRET`, valid 15
  minutes, compared in constant time.
- `GET /api/v1/exports/{id}/download` — requires EITHER a valid `?token=` OR
  auth headers (no longer open). Invalid/expired token → 401 `TOKEN_INVALID`.
- `GET /api/v1/jobs/{jobID}/audio?token=...` (token or auth headers) —
  streams the job's audio (`audio_extract` artifact if present, else the
  uploaded `source_media` when it is audio) with correct `Content-Type` and
  HTTP Range support. Caption-reuse jobs have no audio → 404
  `AUDIO_NOT_AVAILABLE`. In mock mode the extractor writes a real ~2-minute
  silent WAV so playback and seeking work in demos.

Every job status change goes through a compare-and-swap store primitive:
concurrent writers (double approve, cancel vs. worker, duplicate workers)
lose the race with 409 `STATUS_CONFLICT` instead of overwriting state, and
approve/reopen are single atomic store operations (one pgx transaction on
Postgres).

## Mock walkthrough

Mock providers are deterministic. Upload-type demo jobs use `mock://` source
URIs (e.g. `mock://uploads/episode1.mp3`); real uploads use `upload://` URIs
from `POST /api/v1/uploads`. Behavior markers in `source_uri`:

| Marker | Effect |
|---|---|
| `captions=1` (youtube) | Official authorized captions found → pause at `needs_user_action`/`caption_decision` |
| `noaudio` | `NO_AUDIO_TRACK` → `needs_user_action`/`replace_media` |
| `missing` | `MEDIA_NOT_FOUND` → `needs_user_action`/`replace_media` |
| `meta-timeout-once` | First metadata probe times out, retry succeeds |
| `stt-timeout-once` | First STT call times out, retry succeeds |
| `stt-quota` | `STT_PROVIDER_QUOTA_EXCEEDED` → job returns to `queued`, queue pauses |
| `no-diarization` | Single-speaker output + diarization warning in the quality report |

The mock STT emits a two-speaker, ~2-minute episode with filler words and
several segments under the 0.80 confidence threshold (confidence values get a
deterministic per-job jitter in [0.55, 0.99]). The mock LLM cleanup removes
only standalone `um`/`uh`/`you know,`/`like,` fillers; the mock summary is
extractive (grounded by construction). The mock audio extractor writes a real
silent WAV artifact so `GET /api/v1/jobs/{id}/audio` is playable.

## curl example flow

```bash
B=http://localhost:8080/api/v1
P='-H "Content-Type: application/json" -H "X-User-Id: alice" -H "X-User-Role: producer"'
R='-H "Content-Type: application/json" -H "X-User-Id: bob" -H "X-User-Role: reviewer"'

# 0. (Real file) Stage an upload, get an upload:// URI back.
UP=$(curl -s -X POST $B/uploads -H 'X-User-Id: alice' -H 'X-User-Role: producer' \
  -F 'file=@episode1.mp3' \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["upload_uri"])')

# 1. Submit an upload job (attestation required — omit it and you get a 400).
#    Use $UP from step 0, or a mock:// URI when running with mock providers.
JOB=$(curl -s -X POST $B/jobs -H 'Content-Type: application/json' \
  -H 'X-User-Id: alice' -H 'X-User-Role: producer' \
  -d '{"source_type":"upload","source_uri":"mock://uploads/episode1.mp3","language":"en","ownership_attested":true}' \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["job_id"])')

# 2. Poll until in_review (mock pipeline finishes in seconds).
curl -s $B/jobs/$JOB -H 'X-User-Id: alice' -H 'X-User-Role: producer'

# 3. List versions, create a reviewed draft, edit a segment (reviewer role).
curl -s $B/jobs/$JOB/transcripts -H 'X-User-Id: bob' -H 'X-User-Role: reviewer'
REV=$(curl -s -X POST $B/jobs/$JOB/review -H 'X-User-Id: bob' -H 'X-User-Role: reviewer' -d '{}' \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["transcript_version_id"])')

# 4. Approve, export, download.
curl -s -X POST $B/jobs/$JOB/approve -H 'Content-Type: application/json' \
  -H 'X-User-Id: bob' -H 'X-User-Role: reviewer' \
  -d "{\"reviewed_transcript_version_id\":\"$REV\",\"approval_note\":\"ship it\"}"
EX=$(curl -s -X POST $B/jobs/$JOB/exports -H 'Content-Type: application/json' \
  -H 'X-User-Id: alice' -H 'X-User-Role: producer' \
  -d '{"formats":["txt","md","srt","vtt"]}' \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["exports"][0]["download_url"])')
# Signed download URLs need no auth headers; tokens expire after 15 minutes.
# Mint a fresh one anytime via POST $B/signed-links {"kind":"export","id":"<exportID>"}.
curl -L "http://localhost:8080$EX"

# 4b. Stream the job's audio (Range-capable; token or auth headers).
curl -s -H 'X-User-Id: alice' -H 'X-User-Role: producer' \
  -H 'Range: bytes=0-1023' "$B/jobs/$JOB/audio" -o clip.wav

# 5. Summary + audit trail.
curl -s -X POST $B/jobs/$JOB/summary -H 'X-User-Id: alice' -H 'X-User-Role: producer' -d '{}'
curl -s $B/jobs/$JOB/audit -H 'X-User-Id: alice' -H 'X-User-Role: producer'
```

Caption-reuse path: submit `{"source_type":"youtube","source_uri":"https://www.youtube.com/watch?v=demo&captions=1",...}`,
the job pauses at `needs_user_action`/`caption_decision`, then
`POST $B/jobs/$JOB/caption-decision -d '{"reuse_captions":true}'` continues
without STT (segments carry null confidence; quality report sets
`confidence_unavailable`).

## Layout

```
cmd/server/            env config, wiring, graceful shutdown
internal/domain/       types, canonical status enum, error codes
internal/state/        lifecycle state machine (PRD 11.1/11.4, R9)
internal/store/        interfaces; memory/ (default+tests); postgres/ (pgx + migrator)
internal/objectstore/  ObjectStore interface + local-FS impl (DATA_DIR)
internal/providers/    media (stub+ffmpeg), stt (mock+azure), llm (mock+anthropic), captions (mock+youtube)
internal/tools/        one Go function per PRD §14 tool contract
internal/orchestrator/ worker pool + requeue scan; retry/failure matrix (PRD 19)
internal/exporter/     deterministic txt/md/srt/vtt generators + parse-back validators
internal/api/          REST handlers, upload staging, auth/RBAC/logging/CORS middleware
internal/audit/        append-only audit writer
internal/app/          shared wiring for main and the e2e test suite
migrations/            PRD 13.3 schema (+ runtime columns in 0006, upload staging in 0007)
```

## RBAC (PRD 16.2 MVP-minimum)

- **producer** — submit, upload media, view own jobs, caption decision, replace own media, cancel own jobs, summaries, exports/downloads, signed links, audio playback.
- **reviewer** — everything producers can do, plus review versions, segment edits, approve, reopen (also acts as team lead).
- **admin** — everything, including cancel after approval.

Missing/invalid identity headers → `401`; role violations → `403`;
approve/review/reopen/segment-edit require reviewer or admin.

**Security note:** identity transport is the `X-User-Id` / `X-User-Role`
header pair — a **pilot stub pending SSO integration**. It is only safe
behind a trusted reverse proxy that strips inbound copies of those headers
and attaches `X-Auth-Proxy-Secret` (set `AUTH_PROXY_SECRET`). Do not expose
the backend directly to untrusted networks with this scheme.
