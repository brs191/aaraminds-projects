import { useState } from "react";
import { exportDownloadUrl } from "../api/client";
import { useCreateExports, useExports } from "../api/hooks";
import type { ExportFormat, Job } from "../api/types";
import { EmptyState, ErrorBox, Loading, formatTimestamp } from "../components/ui";

const ALL_FORMATS: ExportFormat[] = ["txt", "md", "srt", "vtt"];

export default function ExportsTab({ job }: { job: Job }) {
  const exportsQuery = useExports(job.job_id);
  const createExports = useCreateExports(job.job_id);
  const [selected, setSelected] = useState<ExportFormat[]>([...ALL_FORMATS]);

  const approved = job.status === "approved" || job.status === "exported";
  const canGenerate = approved && selected.length > 0 && !createExports.isPending;

  const toggle = (f: ExportFormat) =>
    setSelected((prev) => (prev.includes(f) ? prev.filter((x) => x !== f) : [...prev, f]));

  const items = exportsQuery.data?.exports ?? [];

  return (
    <div className="stack">
      <div className="card">
        <h2>Generate exports</h2>
        <div className="format-row">
          {ALL_FORMATS.map((f) => (
            <label key={f} className="format-check">
              <input type="checkbox" checked={selected.includes(f)} onChange={() => toggle(f)} />{" "}
              .{f}
            </label>
          ))}
        </div>
        {!approved && (
          <p className="muted hint">
            Exports are generated only from an approved transcript version. Approve the transcript
            in the Review tab first.
          </p>
        )}
        <ErrorBox error={createExports.error} prefix="Export failed:" />
        <button
          className="primary"
          disabled={!canGenerate}
          onClick={() => createExports.mutate(selected)}
        >
          {createExports.isPending ? "Generating…" : "Generate exports"}
        </button>
      </div>

      <div className="card">
        <h2>Export artifacts</h2>
        {exportsQuery.isLoading ? (
          <Loading label="Loading exports…" />
        ) : exportsQuery.error ? (
          <ErrorBox error={exportsQuery.error} prefix="Could not load exports:" />
        ) : items.length === 0 ? (
          <EmptyState>No exports yet.</EmptyState>
        ) : (
          <table className="table">
            <thead>
              <tr>
                <th>Format</th>
                <th>Validation</th>
                <th>Created</th>
                <th>Download</th>
              </tr>
            </thead>
            <tbody>
              {items.map((e) => (
                <tr key={e.export_id}>
                  <td className="mono">.{e.format}</td>
                  <td>
                    <span
                      className={
                        e.validation_status === "passed"
                          ? "badge status-approved"
                          : "badge status-failed"
                      }
                    >
                      {e.validation_status}
                    </span>
                  </td>
                  <td className="muted">{formatTimestamp(e.created_at)}</td>
                  <td>
                    <a href={exportDownloadUrl(e.download_url)} target="_blank" rel="noreferrer">
                      Download
                    </a>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
