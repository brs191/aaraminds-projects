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
make postgres-test # opt-in pgx store workflow test; set POSTGRES_TEST_DATABASE_URL
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
| `POSTGRES_TEST_DATABASE_URL` | — | Optional pgx DSN used only by `make postgres-test` / the skipped-by-default Postgres integration test |
| `MIGRATIONS_DIR` | `migrations` | SQL migration files (applied at boot, `schema_migrations` table) |
| `DATA_DIR` | `./data` | Local object store root (audio, captions, exports) |
| `STT_PROVIDER` | `mock` | `mock`, `whisperx` (local WhisperX sidecar — the default real-STT path) or `azure` (Azure Speech fast transcription) |
| `WHISPERX_URL` | `http://localhost:9090` | WhisperX sidecar base URL for `STT_PROVIDER=whisperx` |
| `WHISPERX_POLL_INTERVAL` | `5s` | Wait between sidecar job-status polls |
| `WHISPERX_TIMEOUT` | `2h` | Overall per-transcription deadline (submit + poll); exceeding it fails the attempt as retryable `STT_PROVIDER_TIMEOUT` |
| `AZURE_SPEECH_ENDPOINT` | — | Azure Speech/Foundry resource endpoint for `STT_PROVIDER=azure`, e.g. `https://<resource>.cognitiveservices.azure.com` |
| `AZURE_SPEECH_REGION` | — | Legacy fallback when `AZURE_SPEECH_ENDPOINT` is unset; builds `https://<region>.api.cognitive.microsoft.com` |
| `AZURE_SPEECH_KEY` | — | Azure Speech resource key for `STT_PROVIDER=azure` |
| `AZURE_SPEECH_MODEL` | — | Optional model/deployment label recorded in audit metadata |
| `LLM_PROVIDER` | `mock` | `mock` or `anthropic` (Claude Messages API) |
| `ANTHROPIC_API_KEY` | — | Required for `LLM_PROVIDER=anthropic` |
| `ANTHROPIC_CLEANUP_MODEL` | `claude-haiku-4-5` | Model for strict segment cleanup |
| `ANTHROPIC_SUMMARY_MODEL` | `claude-sonnet-4-5` | Model for grounded summaries |
| `ANTHROPIC_BASE_URL` | — | Optional Messages API URL override for tests/proxies |
| `CAPTION_PROVIDER` | `mock` | `mock` or `youtube` (Data API v3 skeleton, needs `YOUTUBE_OAUTH_TOKEN`, `YOUTUBE_CHANNEL_OWNED=true`) |
| `MEDIA_PROVIDER` | `mock` | `mock`, or `ffmpeg`/`auto` (uses ffmpeg/ffprobe when on PATH) |
| `CORS_ORIGIN` | `http://localhost:5173` | Allowed browser origin |
| `AUTH_PROXY_SECRET` | — | Optional shared secret required in `X-Auth-Proxy-Secret` from a trusted auth proxy; leave unset only for local dev |
| `SIGNING_SECRET` | random per boot | HMAC-SHA256 key for signed download/audio links (15-min TTL). When unset the server warns and uses a random per-boot secret — signed links stop working on restart |
| `MAX_UPLOAD_BYTES` | `2147483648` (2 GiB) | Body limit for `POST /api/v1/uploads` (413 `REQUEST_TOO_LARGE` above it) |
| `WORKERS` | `2` | Orchestrator worker goroutines |
| `REQUEUE_INTERVAL` | `3s` | Scan interval for submitted/queued jobs, stuck-job reclaim, and retention sweep |
| `RETRY_BACKOFF` | `2s` | Wait before the single retry of retryable failures (context-aware: shutdown interrupts it) |
| `DRAIN_TIMEOUT` | `30s` | SIGTERM drain: intake stops immediately, in-flight steps get this long to finish. Interrupted steps never mark the job failed — it stays in its durable state for reclaim |
| `STUCK_JOB_THRESHOLD` | `10m` | Jobs sitting in a mid-pipeline state (`validating`, `metadata_extracted`, `caption_checked`, `extracting_audio`, `transcribing`, `normalizing`, `quality_checking`) with `updated_at` older than this are CAS'd back to `queued` by the scanner (ALERT-logged + audited); jobs in flight in this process are never reclaimed |
| `RETENTION_DAYS` | `30` | `media_artifacts.retention_until` for `source_media`, `audio_extract`, `caption_source` at creation. The scan loop deletes expired artifacts (object bytes + row) and audit-logs each deletion. Exports and approved transcripts are exempt — never swept |
| `LIBRARY_POLL_INTERVAL` | `30m` | Library mode: RSS feed poll cadence in the orchestrator scan loop (first scan after boot polls immediately; per-feed polls also via `POST /library/feeds/{id}/poll`) |
| `LIBRARY_AUTO_PER_POLL` | `3` | Library mode: max NEW episodes auto-transcribed per feed per poll for `auto_transcribe` feeds (backfilled episodes are never auto-transcribed) |
| `LIBRARY_MAX_DOWNLOAD_BYTES` | `524288000` (500 MiB) | Library mode: enclosure download size cap. Over-cap episodes park their job in `needs_user_action`/`replace_media` with `LIBRARY_DOWNLOAD_TOO_LARGE` |
| `MAX_DURATION_SECONDS` | `0` (disabled) | Media duration cap (PRD 20.2), snapshotted into `job_config`. Over-limit jobs park in `needs_user_action`/`duration_exceeded` with `DURATION_LIMIT_EXCEEDED`; resolution is replace-media with a shorter file or cancel (no override endpoint in MVP) |
| `DEFAULT_CONFIDENCE_THRESHOLD` | `0.80` | job_config snapshot default (PRD R5) |
| `DEFAULT_SUMMARY_MAX_WORDS` | `150` | job_config snapshot default (PRD R10) |
| `DEFAULT_SUMMARY_STYLE` | `neutral-professional` | job_config snapshot default |
| `DEFAULT_STYLE_POLICY_ID` | `default-clean-v1` | Cleanup policy id (PRD 15.2) |

All configuration is snapshotted into `job_config` when a job enters
`validating`; tools read the snapshot via `job_config_id` and never accept
thresholds/style/summary parameters as inputs (PRD 13.2 rule 7).

## WhisperX STT provider (`STT_PROVIDER=whisperx`)

Local, no-cloud-credentials real STT via the WhisperX sidecar in
`../stt-sidecar/` (faster-whisper + word alignment + optional pyannote
diarization). Start the sidecar (`../scripts/stt-setup.sh` once, then
`../scripts/stt-run.sh`), then run the backend with `STT_PROVIDER=whisperx`
(and `WHISPERX_URL` if not `http://localhost:9090`). Sidecar setup, env vars
(`WHISPER_MODEL`, `HF_TOKEN`, GPU notes) and the frozen HTTP API are
documented in `../stt-sidecar/README.md`.

Provider behavior (`internal/providers/stt/whisperx/`):

- Reads the `local://` audio artifact from `DATA_DIR` (same resolution rules
  as the azure provider), streams it to `POST /v1/jobs`, then polls
  `GET /v1/jobs/{id}` every `WHISPERX_POLL_INTERVAL` until done/error, bounded
  by `WHISPERX_TIMEOUT` (context-aware; abandoned sidecar jobs get a
  best-effort `DELETE`).
- Sidecar unreachable / job lost / deadline exceeded map to
  `STT_PROVIDER_TIMEOUT` (retryable per the PRD 19 matrix); HTTP 429 maps to
  `STT_PROVIDER_QUOTA_EXCEEDED`; sidecar `LANGUAGE_UNSUPPORTED` passes
  through.
- Diarization speaker IDs (`SPEAKER_00`, ...) map to `Speaker 1`,
  `Speaker 2`, ... by first appearance; when the sidecar runs without
  `HF_TOKEN` the job still succeeds with everything labeled `Speaker 1` and
  segments flagged `diarization_unavailable`.
- `job_config.expected_speaker_count` is forwarded as
  `min_speakers`/`max_speakers` (via the optional `stt.SpeakerHinter`
  interface — mock and azure are unaffected).
- Audit metadata: `provider=whisperx`, `model` from the sidecar (e.g.
  `large-v3-turbo`), `request_id` = sidecar job id.

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
- `POST /api/v1/signed-links` (auth required, any role with job access) —
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

## Approvals, export supersede semantics

- `GET /api/v1/jobs/{jobID}/approvals` (auth, any role with job access) →
  `200 {"approvals":[{approval_id, approved_transcript_version_id,
  approved_by, approved_at, approval_note,
  superseded_by_approval_id (string|null)}]}`, newest first. The supersede
  chain records post-approval corrections (PRD 11.4).
- Export JSON objects (POST/GET `/jobs/{id}/exports`) carry
  `approved_transcript_version_id` and `superseded`. Re-approving a job marks
  **all** prior exports superseded inside the approve transaction
  (PRD 13.2 r5). Superseded exports remain downloadable; the download response
  carries `X-Superseded: true`.
- Summary JSON carries `validation_status` (`passed` | `needs_review` |
  `failed`) and `validation_notes` (`string|null`).

## Library mode (personal-use extension)

Library mode turns the agent into a personal podcast transcript library:
subscribe to open RSS feeds, transcribe episodes (manually or automatically),
and full-text-search the transcripts. It is a personal-use extension outside
the PRD's review workflow.

**Concepts**

- A **feed** is a subscribed RSS 2.0 URL (enclosures + `itunes:duration`;
  parsed with stdlib `encoding/xml`, Atom not supported). Adding a feed
  validates it by fetching synchronously (10s timeout, 5 MiB cap) and
  backfills the current episodes **without** transcribing them.
- An **episode** is one feed item, unique per `(feed_id, guid)` (missing GUIDs
  fall back to the enclosure URL). Deleting a feed is a soft delete: the feed
  and its episodes leave the listings, but episodes, jobs, and transcripts are
  kept.
- A **library job** is a normal job with `library_mode=true` and upload
  semantics: no caption pre-check, and the pipeline **stops at `drafted`** —
  there is no review gate. The summary is auto-generated right after the
  quality check (fire-and-forget; a summary failure leaves the episode
  drafted). Transcripts are readable and searchable immediately at `drafted`.
- **Ownership**: instead of the manual attestation, library jobs set
  `ownership_attested=true` programmatically and record
  `source_basis="open_rss_personal_use"` on the job plus a
  `job.ownership_attested` audit event noting the basis.
- **Media**: the enclosure is downloaded as the job's first pipeline step
  (HTTP GET streamed to the object store under `library/<episode_id>.<ext>`,
  capped by `LIBRARY_MAX_DOWNLOAD_BYTES`, 10-minute timeout) so the worker
  pool bounds download concurrency. The job's `source_uri` is
  `library://<episode_id>`, resolved through the object store exactly like
  `upload://`. Over-cap or repeatedly failing downloads park the job in
  `needs_user_action`/`replace_media` (documented choice for the cap failure
  mode).
- **Poller**: runs in the orchestrator scan loop every
  `LIBRARY_POLL_INTERVAL`; for `auto_transcribe` feeds it enqueues NEW
  episodes only (never the backfill), at most `LIBRARY_AUTO_PER_POLL` per feed
  per poll. Poll failures record `feeds.poll_error` and never kill the poller.

**API** (base `/api/v1`, standard auth headers, standard error envelope; any
authenticated role — the library is a shared personal space, so library jobs
and their transcripts are readable by every authenticated user):

- `POST /library/feeds` `{"feed_url","auto_transcribe"}` → `201 Feed`.
  Errors: `FEED_URL_INVALID` (400), `FEED_FETCH_FAILED` (400),
  `FEED_ALREADY_EXISTS` (409).
- `GET /library/feeds` → `{"feeds":[Feed]}` — Feed carries `episode_count`,
  `last_polled_at`, `poll_error`.
- `DELETE /library/feeds/{feedID}` → `204` (soft delete).
- `POST /library/feeds/{feedID}/poll` → `202 {"status":"poll_queued"}`.
- `GET /library/episodes?feed_id=&q=&transcribed=true|false` →
  `{"episodes":[Episode]}` newest `published_at` first (nulls last); Episode
  carries `feed_title`, `job_id`, `job_status`.
- `POST /library/episodes/{episodeID}/transcribe` → `202 Episode`;
  `409 EPISODE_ALREADY_TRANSCRIBED` when a job already exists.
- `GET /library/search?q=` (min 2 chars, else 400) →
  `{"results":[{episode_id, episode_title, feed_title, job_id,
  transcript_version_id, segment_id, start_ms, snippet, rank}]}` — snippets
  wrap matches in `<b>...</b>`, limit 50. Searches the latest drafted library
  transcript (clean, raw fallback) per library job **and** the current
  approved version of non-library jobs (those hits carry null episode/feed
  fields).

**Search backends**: on Postgres, search uses a generated `tsvector` column +
GIN index on `transcript_segments.text` (migration 0010) with
`websearch_to_tsquery`, `ts_headline`, and `ts_rank` — **Postgres is the
recommended storage for search**. The memory store ships a naive
case-insensitive substring implementation so everything works (and is tested)
without Postgres, but without stemming or ranking quality.

## Audit discipline, metrics, healthz

- **High-risk actions require a successful audit append** (PRD 19 audit row):
  approve, export generation, cancel, replace-media, caption-decision (and
  reopen) fail with `503 AUDIT_UNAVAILABLE` when the audit store is down.
  Informational tool/status events stay fire-and-forget (logged + counted).
- `GET /debug/vars` (auth-exempt, internal — keep off the public edge) serves
  expvar counters: `jobs_submitted`, `jobs_completed` (reached `in_review`),
  `jobs_failed_total`, `tool_failures_total` (per tool),
  `retries_total`, `export_validation_failures`, `audit_write_failures`,
  `stt_seconds_processed`, `stuck_jobs_reclaimed_total`,
  `artifacts_swept_total`.
- `GET /healthz` pings the job store (cheap status list) and the object store
  (small write probe); either failing answers `503 {"status":"degraded"}`.
- Unknown/internal errors are sanitized: the full error is logged server-side
  with the request ID (`X-Request-Id` response header) and clients receive
  `500 {"error":{"code":"INTERNAL_ERROR","message":"internal error"}}` — no
  pgx/OS strings leak.

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
  -H 'X-User-Id: bob' -H 'X-User-Role: reviewer' \
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
internal/orchestrator/ worker pool + requeue/reclaim/retention scan; graceful drain; retry/failure matrix (PRD 19)
internal/exporter/     deterministic txt/md/srt/vtt generators + parse-back validators
internal/api/          REST handlers, upload staging, auth/RBAC/logging/CORS middleware
internal/audit/        append-only audit writer (fire-and-forget + strict variants)
internal/metrics/      expvar counters served at /debug/vars (PRD 18.2)
internal/app/          shared wiring for main and the e2e test suite
internal/rss/          stdlib RSS 2.0 parser (enclosures, itunes:duration) for library mode
migrations/            PRD 13.3 schema (+ runtime columns in 0006, upload staging in 0007, export supersede in 0008, library feeds/episodes in 0009, segment search in 0010)
```

## RBAC (PRD 16.2 MVP-minimum)

- **producer** — submit, upload media, view own jobs, caption decision, replace own media, cancel own jobs, summaries, download approved exports, signed links for accessible jobs, audio playback.
- **reviewer** — everything producers can do, plus review versions, segment edits, approve, reopen, and generate exports (also acts as team lead).
- **admin** — everything, including cancel after approval.

Missing/invalid identity headers → `401`; role violations → `403`;
approve/review/reopen/segment-edit/export-generation require reviewer or admin.

**Security note:** identity transport is the `X-User-Id` / `X-User-Role`
header pair — a **pilot stub pending SSO integration**. It is only safe
behind a trusted reverse proxy that strips inbound copies of those headers
and attaches `X-Auth-Proxy-Secret` (set `AUTH_PROXY_SECRET`). Do not expose
the backend directly to untrusted networks with this scheme.
