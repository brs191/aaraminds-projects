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
| `DOWNLOAD_TOKEN_SECRET` | random per process | Secret used to sign export download URLs |
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

## Mock walkthrough

Mock providers are deterministic. Behavior markers in `source_uri`:

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
extractive (grounded by construction).

## curl example flow

```bash
B=http://localhost:8080/api/v1
P='-H "Content-Type: application/json" -H "X-User-Id: alice" -H "X-User-Role: producer"'
R='-H "Content-Type: application/json" -H "X-User-Id: bob" -H "X-User-Role: reviewer"'

# 1. Submit an upload job (attestation required — omit it and you get a 400).
JOB=$(curl -s -X POST $B/jobs -H 'Content-Type: application/json' \
  -H 'X-User-Id: alice' -H 'X-User-Role: producer' \
  -d '{"source_type":"upload","source_uri":"uploads/episode1.mp3","language":"en","ownership_attested":true}' \
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
# Signed download URLs need no auth headers, but expire only when DOWNLOAD_TOKEN_SECRET rotates.
curl -L "http://localhost:8080$EX"

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
migrations/            PRD 13.3 schema (+ contract-required runtime columns in 0006)
```

## RBAC (PRD 16.2 MVP-minimum)

- **producer** — submit, view own jobs, caption decision, replace own media, cancel own jobs, summaries, exports/downloads.
- **reviewer** — everything producers can do, plus review versions, segment edits, approve, reopen (also acts as team lead).
- **admin** — everything, including cancel after approval.

Missing/invalid identity headers → `401`; role violations → `403`;
approve/review/reopen/segment-edit require reviewer or admin.
