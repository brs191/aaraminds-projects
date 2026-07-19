#!/usr/bin/env bash
# validate.sh — automated validation for the Podcast Transcript Agent.
# Mirrors VALIDATION.md. Phases 0-2 are fully automatic; 4 and 5 need inputs.
#
# Usage:
#   ./scripts/validate.sh                    # phases 0, 1, 2 (no AI models needed)
#   ./scripts/validate.sh --phase 2          # one phase only
#   ./scripts/validate.sh --phase 4 --audio ~/clip.mp3     # real WhisperX transcription
#   ./scripts/validate.sh --phase 5 --feed https://...rss  # library mode
#   ./scripts/validate.sh --phase all --audio f.mp3 --feed URL
#
# Env overrides: VALIDATE_PORT (18099), WHISPERX_URL (http://localhost:9090),
#                REAL_TIMEOUT seconds for phase 4 polling (3600)

set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PORT="${VALIDATE_PORT:-18099}"
BASE="http://localhost:$PORT/api/v1"
WHISPERX_URL="${WHISPERX_URL:-http://localhost:9090}"
REAL_TIMEOUT="${REAL_TIMEOUT:-3600}"
PASS=0; FAIL=0; SKIPPED=0
SERVER_PID=""
PHASES="0 1 2"; AUDIO=""; FEED=""

while [ $# -gt 0 ]; do
  case "$1" in
    --phase) shift; [ "$1" = "all" ] && PHASES="0 1 2 3 4 5" || PHASES="$1";;
    --audio) shift; AUDIO="$1";;
    --feed)  shift; FEED="$1";;
    *) echo "unknown arg: $1"; exit 2;;
  esac; shift
done

say()  { printf '%s\n' "$*"; }
hdr()  { printf '\n\033[1m== %s ==\033[0m\n' "$*"; }
ok()   { PASS=$((PASS+1));    say "  PASS  $*"; }
bad()  { FAIL=$((FAIL+1));    say "  FAIL  $*"; }
skp()  { SKIPPED=$((SKIPPED+1)); say "  SKIP  $*"; }

PRODUCER=(-H "X-User-Id: producer-1" -H "X-User-Role: producer")
REVIEWER=(-H "X-User-Id: reviewer-1" -H "X-User-Role: reviewer")

# jget <dotted.path>  — reads JSON on stdin, prints value ("" if missing)
jget() {
  python3 -c '
import sys, json
try: d = json.load(sys.stdin)
except Exception: print(""); sys.exit()
for k in sys.argv[1].split("."):
    if isinstance(d, list):
        try: d = d[int(k)]
        except Exception: d = None
    elif isinstance(d, dict): d = d.get(k)
    else: d = None
print("" if d is None else d)' "$1"
}

start_server() { # $@ = extra env KEY=VAL pairs
  local bin="$ROOT/backend/bin/server"
  if [ ! -x "$bin" ]; then
    say "  building backend binary..."
    (cd "$ROOT/backend" && go build -o bin/server ./cmd/server) || { bad "backend build"; return 1; }
  fi
  local dist=""
  [ -d "$ROOT/web/dist" ] && dist="$ROOT/web/dist"
  env PORT="$PORT" DATA_DIR="$ROOT/backend/data-validate" WEB_DIST="$dist" "$@" \
    "$bin" >"$ROOT/backend/validate-server.log" 2>&1 &
  SERVER_PID=$!
  for _ in $(seq 1 30); do
    curl -sf "http://localhost:$PORT/healthz" >/dev/null 2>&1 && return 0
    sleep 0.5
  done
  bad "server did not become healthy on :$PORT (see backend/validate-server.log)"
  return 1
}

stop_server() {
  [ -n "$SERVER_PID" ] && kill "$SERVER_PID" 2>/dev/null && wait "$SERVER_PID" 2>/dev/null
  SERVER_PID=""
}
trap stop_server EXIT

wait_job() { # $1 job_id, $2 want-status, $3 timeout-seconds -> 0 if reached
  local end=$(( $(date +%s) + $3 )) st=""
  while [ "$(date +%s)" -lt "$end" ]; do
    st="$(curl -s "${PRODUCER[@]}" "$BASE/jobs/$1" | jget status)"
    [ "$st" = "$2" ] && return 0
    case "$st" in failed|cancelled) say "      job ended in '$st'"; return 1;; esac
    sleep 2
  done
  say "      timed out waiting for '$2' (last: '$st')"; return 1
}

phase0() {
  hdr "Phase 0 — prerequisites"
  for t in "go:go version" "node:node --version" "python3:python3 --version" "ffmpeg:ffmpeg -version" "curl:curl --version"; do
    name="${t%%:*}"; cmd="${t#*:}"
    if v=$($cmd 2>/dev/null | head -1); then ok "$name — $v"; else
      if [ "$name" = ffmpeg ]; then skp "ffmpeg missing — required only for phase 4 real media"; else bad "$name missing"; fi
    fi
  done
}

phase1() {
  hdr "Phase 1 — automated gates"
  say "  running backend tests (-race, ~1 min)..."
  if (cd "$ROOT/backend" && go test -race -count=1 ./... >/tmp/validate-go.log 2>&1); then
    ok "go test -race ./... ($(grep -c '^ok' /tmp/validate-go.log) packages ok)"
  else bad "go test failed — see /tmp/validate-go.log"; fi
  say "  building UI..."
  if (cd "$ROOT/web" && npm run build >/tmp/validate-npm.log 2>&1); then ok "npm run build"
  else bad "npm run build failed — see /tmp/validate-npm.log"; fi
}

phase2() {
  hdr "Phase 2 — mock-mode smoke (API-level)"
  start_server || return

  # attestation negative
  code=$(curl -s -o /dev/null -w '%{http_code}' "${PRODUCER[@]}" -H 'Content-Type: application/json' \
    -d '{"source_type":"upload","source_uri":"mock://demo.mp3","language":"en","ownership_attested":false}' "$BASE/jobs")
  [ "$code" = 400 ] && ok "missing attestation rejected (400)" || bad "attestation: expected 400, got $code"

  # happy path: submit -> in_review -> review -> approve -> export -> signed download
  job=$(curl -s "${PRODUCER[@]}" -H 'Content-Type: application/json' \
    -d '{"source_type":"upload","source_uri":"mock://demo.mp3","language":"en","ownership_attested":true}' "$BASE/jobs" | jget job_id)
  [ -n "$job" ] && ok "job submitted ($job)" || { bad "job submission"; stop_server; return; }
  wait_job "$job" in_review 60 && ok "reached in_review" || { bad "pipeline to in_review"; stop_server; return; }

  segs=$(curl -s "${PRODUCER[@]}" "$BASE/jobs/$job/transcripts" | jget versions.0.transcript_version_id)
  n=$(curl -s "${PRODUCER[@]}" "$BASE/transcripts/$segs/segments" | python3 -c 'import sys,json;print(len(json.load(sys.stdin)["segments"]))')
  [ "${n:-0}" -gt 0 ] && ok "transcript has $n segments" || bad "no segments"

  rev=$(curl -s "${REVIEWER[@]}" -X POST "$BASE/jobs/$job/review" | jget transcript_version_id)
  [ -n "$rev" ] && ok "review version created" || bad "review creation"
  ap=$(curl -s "${REVIEWER[@]}" -H 'Content-Type: application/json' \
    -d "{\"reviewed_transcript_version_id\":\"$rev\"}" "$BASE/jobs/$job/approve" | jget approved_transcript_version_id)
  [ -n "$ap" ] && ok "approved (immutable version $ap)" || bad "approval"

  exp=$(curl -s "${REVIEWER[@]}" -H 'Content-Type: application/json' \
    -d '{"formats":["txt","md","srt","vtt"]}' "$BASE/jobs/$job/exports")
  vtt=$(echo "$exp" | python3 -c 'import sys,json
for e in json.load(sys.stdin)["exports"]:
    if e["format"]=="vtt": print(e["export_id"])')
  nexp=$(echo "$exp" | python3 -c 'import sys,json;print(len(json.load(sys.stdin)["exports"]))')
  [ "${nexp:-0}" = 4 ] && ok "4 exports generated, all validated" || bad "exports: got ${nexp:-0}/4"
  url=$(curl -s "${REVIEWER[@]}" -H 'Content-Type: application/json' \
    -d "{\"kind\":\"export\",\"id\":\"$vtt\"}" "$BASE/signed-links" | jget url)
  body=$(curl -s "http://localhost:$PORT$url" | head -1)
  [ "$body" = "WEBVTT" ] && ok "signed download works, VTT valid" || bad "signed download (got: $body)"
  code=$(curl -s -o /dev/null -w '%{http_code}' "$BASE/exports/$vtt/download")
  [ "$code" = 401 ] && ok "unauthenticated download blocked (401)" || bad "open download: expected 401, got $code"

  # caption path
  cj=$(curl -s "${PRODUCER[@]}" -H 'Content-Type: application/json' \
    -d '{"source_type":"youtube","source_uri":"mock://video?captions=1","language":"en","ownership_attested":true}' "$BASE/jobs" | jget job_id)
  wait_job "$cj" needs_user_action 30 \
    && ok "caption-decision pause reached" || bad "caption path pause"
  curl -s "${PRODUCER[@]}" -H 'Content-Type: application/json' -d '{"reuse_captions":true}' \
    "$BASE/jobs/$cj/caption-decision" >/dev/null
  wait_job "$cj" in_review 30 && ok "caption reuse → in_review" || bad "caption reuse path"
  cu=$(curl -s "${PRODUCER[@]}" "$BASE/jobs/$cj/quality-report" | jget confidence_unavailable)
  [ "$cu" = "True" ] || [ "$cu" = "true" ] && ok "caption transcript marked confidence_unavailable" || bad "confidence_unavailable flag"

  stop_server
}

phase3() {
  hdr "Phase 3 — WhisperX sidecar"
  h=$(curl -sf "$WHISPERX_URL/healthz" 2>/dev/null)
  if [ -z "$h" ]; then
    skp "sidecar not running at $WHISPERX_URL — run: ./scripts/stt-setup.sh then HF_TOKEN=... ./scripts/stt-run.sh"
    return
  fi
  ok "sidecar healthy: model=$(echo "$h" | jget model) device=$(echo "$h" | jget device)"
  d=$(echo "$h" | jget diarization_available)
  { [ "$d" = "True" ] || [ "$d" = "true" ]; } && ok "diarization available" \
    || skp "diarization unavailable (no HF_TOKEN) — speakers will all be 'Speaker 1'"
}

phase4() {
  hdr "Phase 4 — real transcription (WhisperX)"
  if ! curl -sf "$WHISPERX_URL/healthz" >/dev/null 2>&1; then skp "sidecar not running — see phase 3"; return; fi
  if ! command -v ffmpeg >/dev/null 2>&1; then bad "ffmpeg required for real media"; return; fi
  if [ -z "$AUDIO" ] || [ ! -f "$AUDIO" ]; then skp "no audio file — rerun with --audio /path/to/clip.mp3"; return; fi

  start_server MEDIA_PROVIDER=ffmpeg STT_PROVIDER=whisperx WHISPERX_URL="$WHISPERX_URL" || return
  up=$(curl -s "${PRODUCER[@]}" -F "file=@$AUDIO" "$BASE/uploads" | jget upload_uri)
  [ -n "$up" ] && ok "uploaded $(basename "$AUDIO")" || { bad "upload"; stop_server; return; }
  job=$(curl -s "${PRODUCER[@]}" -H 'Content-Type: application/json' \
    -d "{\"source_type\":\"upload\",\"source_uri\":\"$up\",\"language\":\"en\",\"ownership_attested\":true}" "$BASE/jobs" | jget job_id)
  say "  transcribing (first run also downloads ~1.5GB of models; timeout ${REAL_TIMEOUT}s)..."
  if wait_job "$job" in_review "$REAL_TIMEOUT"; then
    q=$(curl -s "${PRODUCER[@]}" "$BASE/jobs/$job/quality-report")
    ok "real transcript ready — job $job"
    say "      avg confidence: $(echo "$q" | jget average_confidence)   low-confidence segments: $(echo "$q" | jget low_confidence_segment_count)   speakers: $(echo "$q" | jget speaker_count)"
    say ""
    say "  MANUAL CHECKS (PRD §17.2) — open http://localhost:8080 after ./scripts/run.sh, or re-run server:"
    say "    - read 2-3 min against audio: <=10% word errors on clean speech?"
    say "    - click 5 segments: audio lands on the words (<=2s drift)?"
    say "    - two-speaker audio: turns split correctly (>=80%)?"
    say "    - amber flags land on genuinely unclear audio?"
  else
    bad "real transcription failed — see backend/validate-server.log and the sidecar terminal"
  fi
  stop_server
}

phase5() {
  hdr "Phase 5 — library mode"
  if [ -z "$FEED" ]; then skp "no feed URL — rerun with --feed <rss-url> (find one on podcastindex.org)"; return; fi
  start_server || return
  f=$(curl -s "${PRODUCER[@]}" -H 'Content-Type: application/json' \
    -d "{\"feed_url\":\"$FEED\",\"auto_transcribe\":false}" "$BASE/library/feeds")
  fid=$(echo "$f" | jget feed_id)
  [ -n "$fid" ] && ok "feed added: $(echo "$f" | jget title) ($(echo "$f" | jget episode_count) episodes)" \
    || { bad "feed add: $(echo "$f" | jget error.code) $(echo "$f" | jget error.message)"; stop_server; return; }
  n=$(curl -s "${PRODUCER[@]}" "$BASE/library/episodes?feed_id=$fid" | python3 -c 'import sys,json;print(len(json.load(sys.stdin)["episodes"]))')
  [ "${n:-0}" -gt 0 ] && ok "$n episodes listed" || bad "no episodes parsed from feed"
  say "  NOTE: transcribing a full episode takes ~its runtime on CPU."
  say "  Do it from the UI (Library → Transcribe) with the sidecar running, then try Library → Search."
  stop_server
}

for p in $PHASES; do "phase$p"; done

hdr "Summary"
say "  PASS: $PASS   FAIL: $FAIL   SKIP: $SKIPPED"
[ "$FAIL" -eq 0 ] && say "  All executed checks passed." || say "  Failures above — logs: backend/validate-server.log, /tmp/validate-go.log, /tmp/validate-npm.log"
exit "$([ "$FAIL" -eq 0 ] && echo 0 || echo 1)"
