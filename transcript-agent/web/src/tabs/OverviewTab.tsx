import { useState, type FormEvent } from "react";
import { useCancelJob, useCaptionDecision, useReplaceMedia } from "../api/hooks";
import { PIPELINE_ORDER, TERMINAL_STATUSES, type Job, type SourceType } from "../api/types";
import { ErrorBox, formatDuration, formatTimestamp } from "../components/ui";

function StatusTimeline({ job }: { job: Job }) {
  const offPipeline = !PIPELINE_ORDER.includes(job.status);
  const currentIdx = PIPELINE_ORDER.indexOf(job.status);
  return (
    <div className="card">
      <h2>Status</h2>
      <ol className="timeline">
        {PIPELINE_ORDER.map((s, i) => {
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
      {offPipeline && (
        <p className={job.status === "needs_user_action" ? "warn-text" : "error-text"}>
          Job is currently <strong>{job.status.replace(/_/g, " ")}</strong>
          {job.action_required ? ` — action required: ${job.action_required.replace(/_/g, " ")}` : ""}.
        </p>
      )}
      {job.last_error && (
        <div className="error-box">
          Last error: <code>{job.last_error.code}</code> — {job.last_error.message}
        </div>
      )}
    </div>
  );
}

function CaptionDecisionPanel({ job }: { job: Job }) {
  const decision = useCaptionDecision(job.job_id);
  return (
    <div className="card action-panel">
      <h2>Caption decision required</h2>
      <p>
        Official captions were found for this video. Reuse them (no confidence scores, no
        diarization) or transcribe fresh?
      </p>
      <ErrorBox error={decision.error} />
      <div className="button-row">
        <button
          className="primary"
          disabled={decision.isPending}
          onClick={() => decision.mutate(true)}
        >
          Reuse captions
        </button>
        <button disabled={decision.isPending} onClick={() => decision.mutate(false)}>
          Transcribe fresh
        </button>
      </div>
    </div>
  );
}

function ReplaceMediaPanel({ job }: { job: Job }) {
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
      <h2>Replace media</h2>
      <p>The source media cannot be processed. Provide replacement media to resume this job.</p>
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
            placeholder={sourceType === "youtube" ? "https://www.youtube.com/watch?v=…" : "uploads/…"}
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
  const cancel = useCancelJob(job.job_id);
  const cancellable = !TERMINAL_STATUSES.includes(job.status) && job.status !== "exported";

  const onCancel = () => {
    const reason = window.prompt("Reason for cancelling this job:");
    if (reason === null) return;
    cancel.mutate(reason.trim() || "cancelled by user");
  };

  const cfg = job.job_config;

  return (
    <div className="stack">
      <StatusTimeline job={job} />

      {job.action_required === "caption_decision" && <CaptionDecisionPanel job={job} />}
      {job.action_required === "replace_media" && <ReplaceMediaPanel job={job} />}

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
          <button className="danger" disabled={cancel.isPending} onClick={onCancel}>
            {cancel.isPending ? "Cancelling…" : "Cancel job"}
          </button>
        </div>
      )}
    </div>
  );
}
