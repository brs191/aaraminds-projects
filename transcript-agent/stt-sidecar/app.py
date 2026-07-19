"""WhisperX STT sidecar for the Podcast Transcript Agent.

Async transcription HTTP service consumed by the Go backend's ``whisperx``
STT provider (backend/internal/providers/stt/whisperx/). The API is frozen —
keep both sides in sync:

    GET    /healthz          -> {"status","model","device","diarization_available"}
    POST   /v1/jobs          multipart: file, language ("en"), enable_diarization
                             ("true"/"false"), min_speakers?, max_speakers?
                             -> 202 {"job_id","status":"queued"}
    GET    /v1/jobs/{id}     -> {"job_id","status","error","result"}
    DELETE /v1/jobs/{id}     -> 204

Pipeline per job: save upload to a tempdir -> whisperx.load_audio ->
faster-whisper transcription -> whisperx.align (word timestamps) -> optional
pyannote diarization (whisperx DiarizationPipeline + assign_word_speakers,
requires HF_TOKEN) -> segments with start_ms/end_ms, stripped text, speaker
(null when diarization is unavailable) and confidence (exp(avg_logprob)
clamped to [0,1], alignment word-score mean as fallback, null if neither).

Processing is strictly one job at a time (single worker thread + in-process
queue). Job records live in memory; temp files are deleted right after
processing and whole job records are purged after JOB_TTL_SECONDS (1h).

Run:  uvicorn app:app --host 127.0.0.1 --port 9090
Env:  PORT (run command only), WHISPER_MODEL, DEVICE, COMPUTE_TYPE, HF_TOKEN,
      BATCH_SIZE, JOB_TTL_SECONDS — see README.md.
"""

from __future__ import annotations

import logging
import math
import os
import queue
import shutil
import tempfile
import threading
import time
import uuid
from typing import Any, Optional

from fastapi import FastAPI, Form, HTTPException, UploadFile
from fastapi.responses import JSONResponse, Response

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s %(message)s")
log = logging.getLogger("stt-sidecar")

# --------------------------------------------------------------------------
# Configuration
# --------------------------------------------------------------------------

WHISPER_MODEL = os.environ.get("WHISPER_MODEL", "large-v3-turbo")
HF_TOKEN = os.environ.get("HF_TOKEN", "").strip()
BATCH_SIZE = int(os.environ.get("BATCH_SIZE", "8"))
JOB_TTL_SECONDS = int(os.environ.get("JOB_TTL_SECONDS", "3600"))


def _detect_device() -> str:
    forced = os.environ.get("DEVICE", "").strip().lower()
    if forced in ("cpu", "cuda"):
        return forced
    try:
        import torch

        return "cuda" if torch.cuda.is_available() else "cpu"
    except Exception:  # torch missing/broken -> whisperx will fail loudly later
        return "cpu"


DEVICE = _detect_device()
COMPUTE_TYPE = os.environ.get("COMPUTE_TYPE", "").strip() or (
    "float16" if DEVICE == "cuda" else "int8"
)

# --------------------------------------------------------------------------
# Job registry (in-memory, guarded by _jobs_lock)
# --------------------------------------------------------------------------

_jobs: dict[str, dict[str, Any]] = {}
_jobs_lock = threading.Lock()
_work_queue: "queue.Queue[str]" = queue.Queue()

# --------------------------------------------------------------------------
# Lazily loaded models (worker thread only -> no model lock needed)
# --------------------------------------------------------------------------

_asr_model = None
_align_models: dict[str, tuple[Any, Any]] = {}  # language -> (model, metadata)
_diarize_pipeline = None
_diarization_available = bool(HF_TOKEN)  # flipped off if the pipeline fails


def _get_asr_model():
    global _asr_model
    if _asr_model is None:
        import whisperx

        log.info("loading whisper model %s (device=%s compute_type=%s) — first run downloads ~1.5GB",
                 WHISPER_MODEL, DEVICE, COMPUTE_TYPE)
        _asr_model = whisperx.load_model(WHISPER_MODEL, DEVICE, compute_type=COMPUTE_TYPE)
    return _asr_model


def _get_align_model(language: str):
    if language not in _align_models:
        import whisperx

        model, metadata = whisperx.load_align_model(language_code=language, device=DEVICE)
        _align_models[language] = (model, metadata)
    return _align_models[language]


def _get_diarize_pipeline():
    """Return the pyannote diarization pipeline, or None when unavailable."""
    global _diarize_pipeline, _diarization_available
    if not _diarization_available:
        return None
    if _diarize_pipeline is None:
        try:
            import whisperx

            # whisperx < 3.4 exposes whisperx.DiarizationPipeline; newer
            # releases moved it to whisperx.diarize.DiarizationPipeline.
            pipeline_cls = getattr(whisperx, "DiarizationPipeline", None)
            if pipeline_cls is None:
                from whisperx.diarize import DiarizationPipeline as pipeline_cls
            _diarize_pipeline = pipeline_cls(use_auth_token=HF_TOKEN, device=DEVICE)
        except Exception:
            log.exception("diarization pipeline unavailable; continuing without speakers")
            _diarization_available = False
            return None
    return _diarize_pipeline


# --------------------------------------------------------------------------
# Transcription pipeline
# --------------------------------------------------------------------------


def _clamp01(value: float) -> float:
    return max(0.0, min(1.0, value))


def _segment_confidence(aligned_seg: dict, original_seg: Optional[dict]) -> Optional[float]:
    """Confidence for one segment.

    Preference order: exp(avg_logprob) from the faster-whisper segment
    (clamped to [0,1]); mean of alignment word scores (already 0..1); null.
    """
    for seg in (aligned_seg, original_seg):
        if seg is None:
            continue
        avg_logprob = seg.get("avg_logprob")
        if isinstance(avg_logprob, (int, float)):
            return _clamp01(math.exp(avg_logprob))
    scores = [
        w["score"]
        for w in aligned_seg.get("words", [])
        if isinstance(w.get("score"), (int, float))
    ]
    if scores:
        return _clamp01(sum(scores) / len(scores))
    return None


def _process(job_id: str, path: str, params: dict[str, Any]) -> dict[str, Any]:
    """Run the WhisperX pipeline for one job. Returns the result payload."""
    import whisperx

    language = params["language"]
    try:
        audio = whisperx.load_audio(path)
    except Exception as exc:
        raise SidecarError("AUDIO_DECODE_FAILED", f"could not decode audio: {exc}") from exc
    duration_seconds = round(float(len(audio)) / 16000.0, 3)  # load_audio -> 16kHz mono

    asr = _get_asr_model()
    result = asr.transcribe(audio, batch_size=BATCH_SIZE, language=language)
    detected_language = result.get("language") or language
    original_segments = list(result.get("segments") or [])

    # Word-level timestamps via wav2vec2 forced alignment. A missing align
    # model for the language is a hard error (LANGUAGE_UNSUPPORTED); any other
    # alignment failure degrades gracefully to the unaligned segments.
    aligned_segments = original_segments
    try:
        align_model, metadata = _get_align_model(detected_language)
        aligned = whisperx.align(
            original_segments, align_model, metadata, audio, DEVICE,
            return_char_alignments=False,
        )
        aligned_segments = list(aligned.get("segments") or [])
    except ValueError as exc:  # whisperx raises ValueError for unknown languages
        raise SidecarError(
            "LANGUAGE_UNSUPPORTED",
            f"no alignment model for language {detected_language!r}: {exc}",
        ) from exc
    except Exception:
        log.exception("alignment failed for job %s; using unaligned segments", job_id)
        aligned = {"segments": aligned_segments}

    diarization_applied = False
    if params["enable_diarization"]:
        pipeline = _get_diarize_pipeline()
        if pipeline is not None:
            try:
                kwargs: dict[str, Any] = {}
                if params.get("min_speakers"):
                    kwargs["min_speakers"] = params["min_speakers"]
                if params.get("max_speakers"):
                    kwargs["max_speakers"] = params["max_speakers"]
                diarize_segments = pipeline(audio, **kwargs)
                assigned = whisperx.assign_word_speakers(
                    diarize_segments, {"segments": aligned_segments}
                )
                aligned_segments = list(assigned.get("segments") or aligned_segments)
                diarization_applied = True
            except Exception:
                # Jobs still succeed without speakers; the Go side flags the
                # DIARIZATION_UNAVAILABLE warning path from diarization_applied.
                log.exception("diarization failed for job %s; returning null speakers", job_id)

    segments = []
    for idx, seg in enumerate(aligned_segments):
        text = (seg.get("text") or "").strip()
        if not text:
            continue
        start_ms = int(round(float(seg.get("start") or 0.0) * 1000))
        end_ms = int(round(float(seg.get("end") or 0.0) * 1000))
        if end_ms <= start_ms:
            end_ms = start_ms + 1
        original = original_segments[idx] if idx < len(original_segments) else None
        speaker = seg.get("speaker") if diarization_applied else None
        segments.append(
            {
                "start_ms": start_ms,
                "end_ms": end_ms,
                "text": text,
                "speaker": speaker,
                "confidence": _segment_confidence(seg, original),
            }
        )
    if not segments:
        raise SidecarError("TRANSCRIBE_FAILED", "transcription produced no segments")

    return {
        "language": detected_language,
        "duration_seconds": duration_seconds,
        "model": WHISPER_MODEL,
        "diarization_applied": diarization_applied,
        "segments": segments,
    }


class SidecarError(Exception):
    """Structured job error surfaced as {"error":{"code","message"}}."""

    def __init__(self, code: str, message: str):
        super().__init__(f"{code}: {message}")
        self.code = code
        self.message = message


# --------------------------------------------------------------------------
# Worker + TTL cleanup threads
# --------------------------------------------------------------------------


def _cleanup_files(job: dict[str, Any]) -> None:
    tmpdir = job.pop("tmpdir", None)
    job.pop("path", None)
    if tmpdir:
        shutil.rmtree(tmpdir, ignore_errors=True)


def _worker_loop() -> None:
    while True:
        job_id = _work_queue.get()
        with _jobs_lock:
            job = _jobs.get(job_id)
            if job is None or job["status"] != "queued":
                continue  # cancelled or purged while queued
            job["status"] = "processing"
            job["updated_at"] = time.time()
            path = job["path"]
            params = job["params"]
        try:
            result = _process(job_id, path, params)
            error = None
        except SidecarError as exc:
            result, error = None, {"code": exc.code, "message": exc.message}
        except Exception as exc:  # never kill the worker thread
            log.exception("job %s failed", job_id)
            result, error = None, {"code": "TRANSCRIBE_FAILED", "message": str(exc)}
        with _jobs_lock:
            job = _jobs.get(job_id)
            if job is None:
                continue  # deleted while processing; files already cleaned
            _cleanup_files(job)
            if job.get("cancelled"):
                job["status"] = "error"
                job["error"] = {"code": "CANCELLED", "message": "job was cancelled"}
            elif error is not None:
                job["status"] = "error"
                job["error"] = error
            else:
                job["status"] = "done"
                job["result"] = result
            job["updated_at"] = time.time()


def _ttl_loop() -> None:
    while True:
        time.sleep(60)
        cutoff = time.time() - JOB_TTL_SECONDS
        with _jobs_lock:
            expired = [jid for jid, j in _jobs.items() if j["updated_at"] < cutoff]
            for jid in expired:
                _cleanup_files(_jobs[jid])
                del _jobs[jid]
        if expired:
            log.info("ttl cleanup: purged %d job(s)", len(expired))


threading.Thread(target=_worker_loop, name="stt-worker", daemon=True).start()
threading.Thread(target=_ttl_loop, name="stt-ttl", daemon=True).start()

# --------------------------------------------------------------------------
# HTTP API
# --------------------------------------------------------------------------

app = FastAPI(title="WhisperX STT sidecar", version="1.0.0")


@app.get("/healthz")
def healthz() -> dict[str, Any]:
    return {
        "status": "ok",
        "model": WHISPER_MODEL,
        "device": DEVICE,
        "diarization_available": _diarization_available,
    }


def _parse_bool(raw: str, field: str) -> bool:
    value = raw.strip().lower()
    if value in ("true", "1", "yes"):
        return True
    if value in ("false", "0", "no", ""):
        return False
    raise HTTPException(status_code=400, detail=f"{field} must be 'true' or 'false'")


@app.post("/v1/jobs", status_code=202)
async def submit_job(
    file: UploadFile,
    language: str = Form("en"),
    enable_diarization: str = Form("true"),
    min_speakers: Optional[int] = Form(None),
    max_speakers: Optional[int] = Form(None),
) -> dict[str, str]:
    params = {
        "language": (language or "en").strip() or "en",
        "enable_diarization": _parse_bool(enable_diarization, "enable_diarization"),
        "min_speakers": min_speakers,
        "max_speakers": max_speakers,
    }
    job_id = str(uuid.uuid4())
    tmpdir = tempfile.mkdtemp(prefix=f"stt-{job_id[:8]}-")
    filename = os.path.basename(file.filename or "") or "audio"
    path = os.path.join(tmpdir, filename)
    try:
        with open(path, "wb") as out:
            while chunk := await file.read(1 << 20):
                out.write(chunk)
    except Exception:
        shutil.rmtree(tmpdir, ignore_errors=True)
        raise
    now = time.time()
    with _jobs_lock:
        _jobs[job_id] = {
            "status": "queued",
            "created_at": now,
            "updated_at": now,
            "params": params,
            "tmpdir": tmpdir,
            "path": path,
            "error": None,
            "result": None,
            "cancelled": False,
        }
    _work_queue.put(job_id)
    log.info("job %s queued (%s, language=%s, diarization=%s)",
             job_id, filename, params["language"], params["enable_diarization"])
    return {"job_id": job_id, "status": "queued"}


@app.get("/v1/jobs/{job_id}")
def get_job(job_id: str) -> JSONResponse:
    with _jobs_lock:
        job = _jobs.get(job_id)
        if job is None:
            return JSONResponse(
                status_code=404,
                content={"error": {"code": "JOB_NOT_FOUND", "message": f"unknown job {job_id}"}},
            )
        return JSONResponse(
            content={
                "job_id": job_id,
                "status": job["status"],
                "error": job["error"],
                "result": job["result"],
            }
        )


@app.delete("/v1/jobs/{job_id}", status_code=204)
def delete_job(job_id: str) -> Response:
    with _jobs_lock:
        job = _jobs.get(job_id)
        if job is None:
            raise HTTPException(status_code=404, detail=f"unknown job {job_id}")
        if job["status"] == "processing":
            # The whisper pass cannot be interrupted mid-flight; mark it so
            # the worker discards the result when it finishes.
            job["cancelled"] = True
        else:
            _cleanup_files(job)
            del _jobs[job_id]
    return Response(status_code=204)
