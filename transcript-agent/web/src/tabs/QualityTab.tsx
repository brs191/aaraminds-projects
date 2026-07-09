import { useQualityReport } from "../api/hooks";
import type { Job } from "../api/types";
import { EmptyState, ErrorBox, Loading, formatMs } from "../components/ui";

function Metric({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="metric">
      <div className="metric-value">{value}</div>
      <div className="metric-label">{label}</div>
    </div>
  );
}

export default function QualityTab({ job }: { job: Job }) {
  const { data, isLoading, error } = useQualityReport(job.job_id);

  if (isLoading) return <Loading label="Loading quality report…" />;
  if (error) return <ErrorBox error={error} prefix="Could not load quality report:" />;

  const report = data ?? null;
  if (!report) {
    return (
      <EmptyState>
        No quality report yet. It is produced during the quality-checking stage.
      </EmptyState>
    );
  }

  return (
    <div className="stack">
      {report.confidence_unavailable && (
        <div className="notice-banner">
          Confidence unavailable — this transcript is caption-derived, so provider confidence
          metrics do not apply.
        </div>
      )}

      <div className="card">
        <h2>Quality metrics</h2>
        <div className="metrics-grid">
          <Metric
            label="Quality score"
            value={report.quality_score !== null ? report.quality_score.toFixed(2) : "—"}
          />
          <Metric
            label="Average confidence"
            value={
              report.confidence_unavailable || report.average_confidence === null
                ? "n/a"
                : report.average_confidence.toFixed(2)
            }
          />
          <Metric label="Confidence threshold" value={report.confidence_threshold} />
          <Metric label="Low-confidence segments" value={report.low_confidence_segment_count} />
          <Metric label="Coverage gap (s)" value={report.coverage_gap_seconds} />
          <Metric label="Timestamp gaps" value={report.timestamp_gap_count} />
          <Metric label="Diarization warnings" value={report.diarization_warning_count} />
        </div>
      </div>

      <div className="card">
        <h2>Issues</h2>
        {report.issues.length === 0 ? (
          <EmptyState>No issues reported.</EmptyState>
        ) : (
          <table className="table">
            <thead>
              <tr>
                <th>Type</th>
                <th>Severity</th>
                <th>Range</th>
                <th>Message</th>
              </tr>
            </thead>
            <tbody>
              {report.issues.map((issue, i) => (
                <tr key={i}>
                  <td className="mono">{issue.issue_type}</td>
                  <td>
                    <span className={`badge severity-${issue.severity}`}>{issue.severity}</span>
                  </td>
                  <td className="mono">
                    {formatMs(issue.start_ms)}–{formatMs(issue.end_ms)}
                  </td>
                  <td>{issue.message}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
