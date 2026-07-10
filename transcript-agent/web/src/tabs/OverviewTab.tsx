import { useState, type FormEvent } from "react";
import { useCancelJob, useCaptionDecision, useQualityReport, useReplaceMedia } from "../api/hooks";
import {
  CAPTION_PIPELINE_ORDER,
  PIPELINE_ORDER,
  TERMINAL_STATUSES,
  type Job,
  type SourceType,
} from "../api/types";
import { canCancel, useIdentity } from "../identity";
import { ErrorBox, formatDuration, formatTimestamp } from "../components/ui";

function recoveryHint(code: string, actionRequired: Job["action_required"]): string | null {
  switch (code) {
    case "NOT_CONFIGURED":
      return "Provider configuration is missing on the backend. Set the provider env vars, restart, then resubmit or replace media.";
    case "STT_PROVIDER_TIMEOUT":
      return "Transcription timed out after retry. Try again when the provider is healthy, or replace media with shorter, cleaner audio.";
    case "STT_PROVIDER_QUOTA_EXCEEDED":
      return "Transcription is paused for provider quota. Queued jobs resume after the quota pause clears.";
    case "LLM_PROVIDER_TIMEOUT":
      return "The cleanup or summary provider timed out. Retry after the provider recovers.";
    case "LLM_OUTPUT_INVALID":
      return "The model response did not match the transcript contract, so it was rejected instead of being saved.";
    case "AUDIT_UNAVAILABLE":
      return "The requested action was not applied because the audit event could not be written. Retry after audit storage is healthy.";
    case "UNSUPPORTED_FORMAT":
      return "Replace media with a supported file type: mp3, m4a, wav, mp4, or mov.";
    case "NO_AUDIO_TRACK":
      return "Replace media with a file that contains an audio track.";
    case "MEDIA_NOT_FOUND":
      return "Replace media with a source the backend can access.";
    case "DURATION_LIMIT_EXCEEDED":
      return "Replace media with a shorter source or cancel the job.";
    case "INVALID_SOURCE_URI":
      return actionRequired === "replace_media"
        ? "Use a staged upload:// URI or a valid YouTube URL for replacement media."
        : "Use a staged upload:// URI for uploads or a valid YouTube URL.";
    default:
      return null;
  }
}

function StatusTimeline({ job, captionPath }: { job: Job; captionPath: boolean }) {
  // M10: on the caption-reuse path, extracting_audio / transcribing never ran —
  // do not render them as completed steps.
  const order = captionPath ? CAPTION_PIPELINE_ORDER : PIPELINE_ORDER;
  const offPipeline = !order.includes(job.status);
  const currentIdx = order.indexOf(job.status);
  const hint = job.last_error ? recoveryHint(job.last_error.code, job.action_required) : null;
  return (
    <div className="card">
      <h2>Status</h2>
      <ol className="timeline">
        {order.map((s, i) => {
          let cls = "timeline-step";
          if (!offPipeline && i < currentIdx) cls += " done";
          if (!offPipeline && i === currentIdx) cls += " current";
          return (
            <li key={s} className={cls}>
              {s.replace(/_/g, " ")}
            </li>
          );
        })}
      </ol>
      {captionPath && (
        <p className="muted hint">
          Caption-reuse path — audio extraction and transcription were skipped.
        </p>
      )}
      {offPipeline && (
        <p className={job.status === "needs_user_action" ? "warn-text" : "error-text"}>
          Job is currently <strong>{job.status.replace(/_/g, " ")}</strong>
          {job.action_required ? ` — action required: ${job.action_required.replace(/_/g, " ")}` : ""}.
        </p>
      )}
      {job.last_error && (
        <div className="error-box">
          Last error: <code>{job.last_error.code}</code> — {job.last_error.message}
          {hint && <p className="error-hint">{hint}</p>}
        </div>
      )}
    </div>
  );
}

function CaptionDecisionPanel({ job }: { job: Job }) {
  const decision = useCaptionDecision(job.job_id);
  // L6: reuse is a lossy, irreversible-ish choice — confirm before committing.
  const [confirmingReuse, setConfirmingReuse] = useState(false);
  return (
    <div className="card action-panel">
      <h2>Caption decision required</h2>
      <p>
        Official captions were found for this video. Reuse them (no confidence scores, no
        diarization) or transcribe fresh?
      </p>
      <ErrorBox error={decision.error} />
      {confirmingReuse ? (
        <div className="notice-banner" role="alertdialog" aria-label="Confirm caption reuse">
          Reuses official captions: no confidence scores or speaker detection. Continue?
          <div className="button-row">
            <button
              className="primary"
              disabled={decision.isPending}
              onClick={() =>
                decision.mutate(true, { onSettled: () => setConfirmingReuse(false) })
              }
            >
              {decision.isPending ? "Applying…" : "Continue"}
            </button>
            <button disabled={decision.isPending} onClick={() => setConfirmingReuse(false)}>
              Back
            </button>
          </div>
        </div>
      ) : (
        <div className="button-row">
          <button
            className="primary"
            disabled={decision.isPending}
            onClick={() => setConfirmingReuse(true)}
          >
            Reuse captions
          </button>
          <button disabled={decision.isPending} onClick={() => decision.mutate(false)}>
            Transcribe fresh
          </button>
        </div>
      )}
    </div>
  );
}

function ReplaceMediaPanel({ job, durationExceeded }: { job: Job; durationExceeded: boolean }) {
  const replace = useReplaceMedia(job.job_id);
  const [sourceType, setSourceType] = useState<SourceType>(job.source_type);
  const [sourceUri, setSourceUri] = useState("");
  const [attested, setAttested] = useState(false);
  const canSubmit = attested && sourceUri.trim().length > 0 && !replace.isPending;

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    if (!canSubmit) return;
    replace.mutate({
      source_type: sourceType,
      source_uri: sourceUri.trim(),
      ownership_attested: attested,
    });
  };

  return (
    <div className="card action-panel">
      <h2>{durationExceeded ? "Media too long" : "Replace media"}</h2>
      {durationExceeded ? (
        <p>
          The source media exceeds the maximum allowed duration. Replace it with shorter media
          below, or cancel the job.
        </p>
      ) : (
        <p>The source media cannot be processed. Provide replacement media to resume this job.</p>
      )}
      <form onSubmit={onSubmit} className="form">
        <div className="field">
          <label htmlFor="replace-type">Source type</label>
          <select
            id="replace-type"
            value={sourceType}
            onChange={(e) => setSourceType(e.target.value as SourceType)}
          >
            <option value="youtube">youtube</option>
            <option value="upload">upload</option>
          </select>
        </div>
        <div className="field">
          <label htmlFor="replace-uri">Source URI</label>
          <input
            id="replace-uri"
            type="text"
            value={sourceUri}
            onChange={(e) => setSourceUri(e.target.value)}
            placeholder={
              sourceType === "youtube"
                ? "https://www.youtube.com/watch?v=…"
                : "upload://… or mock://…"
            }
          />
        </div>
        <div className="field checkbox-field">
          <label>
            <input
              type="checkbox"
              checked={attested}
              onChange={(e) => setAttested(e.target.checked)}
            />{" "}
            I attest ownership/licensing for the replacement media.
          </label>
        </div>
        <ErrorBox error={replace.error} />
        <button type="submit" className="primary" disabled={!canSubmit}>
          {replace.isPending ? "Replacing…" : "Replace media"}
        </button>
      </form>
    </div>
  );
}

export default function OverviewTab({ job }: { job: Job }) {
  const { identity } = useIdentity();
  const cancel = useCancelJob(job.job_id);
  const qualityQuery = useQualityReport(job.job_id);
  const cancellable = !TERMINAL_STATUSES.includes(job.status) && job.status !== "exported";
  // M1 (PRD 16.2): cancel is submitter-or-admin. UX mirror only — server enforces.
  const cancelAllowed = canCancel(identity, job.submitted_by);

  // M4: inline reason input instead of window.prompt. An empty reason is sent
  // as empty — no fabricated default.
  const [cancelOpen, setCancelOpen] = useState(false);
  const [cancelReason, setCancelReason] = useState("");

  const cfg = job.job_config;
  const captionPath = qualityQuery.data?.confidence_unavailable === true;

  return (
    <div className="stack">
      <StatusTimeline job={job} captionPath={captionPath} />

      {job.action_required === "caption_decision" && <CaptionDecisionPanel job={job} />}
      {(job.action_required === "replace_media" ||
        job.action_required === "duration_exceeded") && (
        <ReplaceMediaPanel
          job={job}
          durationExceeded={job.action_required === "duration_exceeded"}
        />
      )}

      <div className="card">
        <h2>Job details</h2>
        <dl className="detail-grid">
          <dt>Source</dt>
          <dd>
            {job.source_type} — <span className="uri">{job.source_uri}</span>
          </dd>
          <dt>Submitted by</dt>
          <dd>{job.submitted_by}</dd>
          <dt>Ownership attested</dt>
          <dd>{job.ownership_attested ? "yes" : "no"}</dd>
          <dt>Duration</dt>
          <dd>{formatDuration(job.duration_seconds)}</dd>
          <dt>Created</dt>
          <dd>{formatTimestamp(job.created_at)}</dd>
          <dt>Updated</dt>
          <dd>{formatTimestamp(job.updated_at)}</dd>
        </dl>
      </div>

      <div className="card">
        <h2>Job configuration</h2>
        {cfg ? (
          <dl className="detail-grid">
            <dt>Language</dt>
            <dd>{cfg.language}</dd>
            <dt>Confidence threshold</dt>
            <dd>{cfg.confidence_threshold}</dd>
            <dt>Diarization</dt>
            <dd>{cfg.enable_diarization ? "enabled" : "disabled"}</dd>
            <dt>Style policy</dt>
            <dd>{cfg.style_policy_id}</dd>
            <dt>Summary max words</dt>
            <dd>{cfg.summary_max_words}</dd>
            <dt>Summary style</dt>
            <dd>{cfg.summary_style}</dd>
            <dt>STT provider</dt>
            <dd>{cfg.stt_provider}</dd>
          </dl>
        ) : (
          <p className="muted">Configuration snapshot not created yet (set during validation).</p>
        )}
      </div>

      {cancellable && (
        <div className="card danger-zone">
          <h2>Cancel job</h2>
          <p className="muted">
            Cancellation stops processing. Approvals, transcript versions, and audit records are
            retained.
          </p>
          <ErrorBox error={cancel.error} />
          {!cancelAllowed ? (
            <p className="muted hint">
              Only the submitter ({job.submitted_by}) or an admin can cancel this job (currently{" "}
              {identity.userId}/{identity.role}).
            </p>
          ) : !cancelOpen ? (
            <button className="danger" disabled={cancel.isPending} onClick={() => setCancelOpen(true)}>
              Cancel job
            </button>
          ) : (
            <div className="cancel-form">
              <div className="field">
                <label htmlFor="cancel-reason">Reason (optional)</label>
                <input
                  id="cancel-reason"
                  type="text"
                  value={cancelReason}
                  onChange={(e) => setCancelReason(e.target.value)}
                  placeholder="Why is this job being cancelled?"
                />
              </div>
              <div className="button-row">
                <button
                  className="danger"
                  disabled={cancel.isPending}
                  onClick={() =>
                    cancel.mutate(cancelReason.trim(), {
                      onSuccess: () => setCancelOpen(false),
                    })
                  }
                >
                  {cancel.isPending ? "Cancelling…" : "Confirm cancel"}
                </button>
                <button
                  disabled={cancel.isPending}
                  onClick={() => {
                    setCancelOpen(false);
                    setCancelReason("");
                  }}
                >
                  Keep job
                </button>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
