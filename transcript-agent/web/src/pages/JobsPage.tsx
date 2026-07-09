import { Link } from "react-router-dom";
import { useJobs } from "../api/hooks";
import {
  ActionChip,
  EmptyState,
  ErrorBox,
  Loading,
  StatusBadge,
  formatDuration,
  formatTimestamp,
} from "../components/ui";

export default function JobsPage() {
  const { data, isLoading, error } = useJobs();

  if (isLoading) return <Loading label="Loading jobs…" />;
  if (error) return <ErrorBox error={error} prefix="Could not load jobs:" />;

  const jobs = data?.jobs ?? [];

  return (
    <div className="page">
      <div className="page-head">
        <h1>Jobs</h1>
        <Link to="/submit" className="button primary">
          Submit episode
        </Link>
      </div>

      {jobs.length === 0 ? (
        <EmptyState>
          No jobs yet. <Link to="/submit">Submit the first episode</Link> to get started.
        </EmptyState>
      ) : (
        <table className="table">
          <thead>
            <tr>
              <th>Job</th>
              <th>Source</th>
              <th>Status</th>
              <th>Duration</th>
              <th>Created</th>
              <th>Updated</th>
            </tr>
          </thead>
          <tbody>
            {jobs.map((job) => (
              <tr key={job.job_id}>
                <td>
                  <Link to={`/jobs/${job.job_id}`} className="mono">
                    {job.job_id.slice(0, 8)}
                  </Link>
                </td>
                <td>
                  <span className="badge chip-source">{job.source_type}</span>{" "}
                  <span className="uri" title={job.source_uri}>
                    {job.source_uri}
                  </span>
                </td>
                <td>
                  <StatusBadge status={job.status} /> <ActionChip action={job.action_required} />
                </td>
                <td>{formatDuration(job.duration_seconds)}</td>
                <td className="muted">{formatTimestamp(job.created_at)}</td>
                <td className="muted">{formatTimestamp(job.updated_at)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
