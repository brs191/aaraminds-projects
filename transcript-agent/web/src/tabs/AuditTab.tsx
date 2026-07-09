import { useAudit } from "../api/hooks";
import type { Job } from "../api/types";
import { EmptyState, ErrorBox, Loading, formatTimestamp } from "../components/ui";

export default function AuditTab({ job }: { job: Job }) {
  const { data, isLoading, error } = useAudit(job.job_id);

  if (isLoading) return <Loading label="Loading audit trail…" />;
  if (error) return <ErrorBox error={error} prefix="Could not load audit trail:" />;

  const events = data?.events ?? [];
  if (events.length === 0) return <EmptyState>No audit events recorded yet.</EmptyState>;

  return (
    <div className="card">
      <h2>Audit trail</h2>
      <table className="table">
        <thead>
          <tr>
            <th>Event</th>
            <th>Actor</th>
            <th>Time</th>
            <th>Payload</th>
          </tr>
        </thead>
        <tbody>
          {events.map((ev) => (
            <tr key={ev.audit_event_id}>
              <td className="mono">{ev.event_type}</td>
              <td>
                {ev.actor_id} <span className="muted">({ev.actor_type})</span>
              </td>
              <td className="muted">{formatTimestamp(ev.created_at)}</td>
              <td>
                <details>
                  <summary className="muted">payload</summary>
                  <pre className="payload">{JSON.stringify(ev.event_payload, null, 2)}</pre>
                </details>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
