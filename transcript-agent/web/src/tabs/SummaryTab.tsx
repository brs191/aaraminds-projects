import { useEffect, useState } from "react";
import { useGenerateSummary, usePatchSummary, useSummary } from "../api/hooks";
import type { Job } from "../api/types";
import { EmptyState, ErrorBox, Loading, formatTimestamp } from "../components/ui";

export default function SummaryTab({ job }: { job: Job }) {
  const summaryQuery = useSummary(job.job_id);
  const generate = useGenerateSummary(job.job_id);
  const patch = usePatchSummary(job.job_id);

  const summary = summaryQuery.data ?? null;
  const [text, setText] = useState("");
  useEffect(() => {
    if (summary) setText(summary.text);
  }, [summary]);

  if (summaryQuery.isLoading) return <Loading label="Loading summary…" />;
  if (summaryQuery.error)
    return <ErrorBox error={summaryQuery.error} prefix="Could not load summary:" />;

  const dirty = summary !== null && text !== summary.text;
  const maxWords = job.job_config?.summary_max_words ?? 150;

  return (
    <div className="stack">
      <div className="card">
        <div className="page-head">
          <h2>Summary</h2>
          <button
            className="primary"
            disabled={generate.isPending}
            onClick={() => generate.mutate()}
          >
            {generate.isPending
              ? "Generating…"
              : summary
                ? "Regenerate summary"
                : "Generate summary"}
          </button>
        </div>
        <p className="muted hint">
          Transcript-grounded, target ≤{maxWords} words ({job.job_config?.summary_style ?? "neutral-professional"}).
          Regenerate or reconfirm after approval.
        </p>
        <ErrorBox error={generate.error} prefix="Generation failed:" />

        {!summary ? (
          <EmptyState>
            No summary yet. Generate one once a transcript version exists.
          </EmptyState>
        ) : (
          <>
            {summary.validation_status === "failed" && (
              <div className="error-box" role="alert">
                <strong>Validation failed</strong> — this summary did not pass grounding
                validation. Do not use it as-is; regenerate or rewrite it against the transcript.
                {summary.validation_notes && (
                  <p className="validation-notes">{summary.validation_notes}</p>
                )}
              </div>
            )}
            {summary.validation_status === "needs_review" && (
              <div className="warn-banner">
                Validation flagged this summary as <strong>needs review</strong> — check it for
                claims not grounded in the transcript before using it.
                {summary.validation_notes && (
                  <p className="validation-notes">{summary.validation_notes}</p>
                )}
              </div>
            )}
            <p className="muted">
              Source version: <span className="mono">{summary.source_transcript_version_id.slice(0, 8)}</span>{" "}
              · generated {formatTimestamp(summary.created_at)} · status: {summary.validation_status}
            </p>
            <textarea
              className="summary-editor"
              rows={8}
              value={text}
              onChange={(e) => setText(e.target.value)}
              aria-label="Summary text"
            />
            <ErrorBox error={patch.error} prefix="Save failed:" />
            <div className="button-row">
              <button
                className="primary"
                disabled={!dirty || patch.isPending}
                onClick={() => patch.mutate({ summaryId: summary.summary_id, text })}
              >
                {patch.isPending ? "Saving…" : dirty ? "Save changes" : "Saved"}
              </button>
              {dirty && (
                <button onClick={() => setText(summary.text)} disabled={patch.isPending}>
                  Discard changes
                </button>
              )}
            </div>
          </>
        )}
      </div>
    </div>
  );
}
