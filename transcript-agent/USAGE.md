# Usage Guide — Podcast Transcript Agent

How to run and use the software day-to-day. (Setup validation lives in `VALIDATION.md`.)

## What this is

Two workflows in one app:

1. **Transcript workflow** (governed): submit media you own → automatic transcript with speaker labels and confidence flags → human review and edit → immutable approval → validated exports (`.txt` `.md` `.srt` `.vtt`). For content you intend to publish.
2. **Personal library** (zero-touch): subscribe to podcast RSS feeds → episodes auto-download and transcribe → read, skim summaries, and search across everything. No review step. For your own listening/learning.

Everything runs on your machine. Audio never leaves it.

---

## Starting the software

Two terminals from the repo root (`~/projects/aaraminds-projects/transcript-agent`):

```bash
# Terminal 1 — the transcription engine (WhisperX sidecar)
HF_TOKEN=hf_xxx ./scripts/stt-run.sh
```

- `HF_TOKEN` is optional: without it everything works but all speech is labeled "Speaker 1" (no speaker separation).
- First-ever transcription downloads ~1.5 GB of models; after that it's instant startup.
- Leave this terminal running.

```bash
# Terminal 2 — the app
MEDIA_PROVIDER=ffmpeg STT_PROVIDER=whisperx ./scripts/run.sh
```

Open **http://localhost:8080**.

> Tip: put `export MEDIA_PROVIDER=ffmpeg STT_PROVIDER=whisperx` in your `~/.bashrc`
> so a plain `./scripts/run.sh` does the right thing.

**Mock mode** (no AI, instant fake transcripts — for demos/UI testing): just `./scripts/run.sh` with no env vars.

## Identities

Top-right dropdown. There's no login yet (local use only — don't expose this to a network):

| Identity | Can do |
|---|---|
| `producer-1` | submit jobs, download approved exports, use the library |
| `reviewer-1` | everything producers can, plus edit/approve transcripts and generate exports |
| `admin-1` | everything |

---

## Workflow 1 — transcribe something you own

1. **Submit** → choose a file (`mp3` `m4a` `wav` `mp4` `mov`), tick the ownership attestation, Submit.
2. Watch the job on the **Jobs** page — it moves through the pipeline automatically. A 1-hour episode takes roughly its own runtime on CPU (much faster with a GPU).
3. When it reaches **in_review**, open it:
   - **Review tab** — transcript with timestamps and speakers. Amber segments = low confidence, review those first. Click any segment's ▶ to hear that exact moment. Click text to edit; type in the speaker box to rename (or use rename-everywhere).
   - Switch identity to `reviewer-1` → **Start review** → make your edits → **Approve**. Approval is permanent — corrections afterward go through **Reopen**, which creates a new version and keeps the full history (Approvals card).
4. **Exports tab** → pick formats → Generate → download via the button (links are secure and expire after 15 min — just click again if one goes stale).
5. **Summary tab** → Generate for an editable ≤150-word summary grounded in the transcript.

Everything is audit-logged (Audit tab): who submitted, approved, exported, and when.

## Workflow 2 — the personal library

1. Find a podcast's RSS feed URL: search the show on **podcastindex.org** → copy the RSS link.
2. **Library** → paste the URL → leave **auto-transcribe** on → Add.
3. That's it. The poller checks feeds every 30 min; new episodes download and transcribe themselves and land as readable transcripts with summaries — no review step, by design. Older (backfill) episodes: click **Transcribe** on any episode you want.
4. **Library → Search** — full-text search across every transcript. Click a result to land on the exact segment with audio cued up.

Notes:
- Deleting a feed removes it and its episodes from the listing, but keeps any transcripts you already made (still findable via Jobs and Search).
- The library is for personal use. Don't republish transcripts of other people's shows.
- YouTube links are not supported — by design. Almost every video podcast also has an audio RSS feed; use that.

---

## Configuration quick reference

Set as env vars before `./scripts/run.sh`:

| Variable | Default | What it does |
|---|---|---|
| `STT_PROVIDER` | `mock` | `whisperx` for real transcription |
| `MEDIA_PROVIDER` | `mock` | `ffmpeg` for real media (required with whisperx) |
| `WHISPERX_URL` | `http://localhost:9090` | sidecar address |
| `PORT` | `8080` | app port |
| `STORAGE` | `memory` | `postgres` + `DATABASE_URL` for persistence & better search |
| `RETENTION_DAYS` | `30` | days to keep source audio after approval |
| `MAX_DURATION_SECONDS` | `0` (off) | reject episodes longer than this |
| `LIBRARY_POLL_INTERVAL` | `30m` | RSS check frequency |
| `LIBRARY_AUTO_PER_POLL` | `3` | max auto-transcriptions per feed per poll |

Sidecar (before `./scripts/stt-run.sh`): `HF_TOKEN`, `WHISPER_MODEL` (default `large-v3-turbo`; use `distil-large-v3` if CPU is slow), `DEVICE` (auto).

**Important:** with the default `STORAGE=memory`, jobs/feeds/transcripts are lost when the server restarts (downloaded files remain in `backend/data/`). For a library you keep, run Postgres: `docker compose up` (first real test of that path) or point `DATABASE_URL` at your own Postgres 16.

## Troubleshooting

| Symptom | Cause / fix |
|---|---|
| Job stuck in `transcribing` on first run | models downloading — watch the sidecar terminal; one-time |
| Transcript is gibberish/empty | you're on mock media — set `MEDIA_PROVIDER=ffmpeg` |
| All speakers are "Speaker 1" | no `HF_TOKEN`, or pyannote terms not accepted on huggingface.co |
| Job failed `STT_PROVIDER_TIMEOUT` | sidecar not running — start terminal 1, resubmit |
| Feed add fails `FEED_FETCH_FAILED` | wrong URL (needs the RSS/XML link, not the show's webpage) |
| Download link 401 | link expired (15 min) — click the download button again |
| Everything gone after restart | memory storage — see Postgres note above |

Health checks: `curl localhost:8080/healthz` (app) · `curl localhost:9090/healthz` (sidecar) · metrics at `localhost:8080/debug/vars`.
