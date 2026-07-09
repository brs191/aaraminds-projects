import type { ReactNode } from "react";
import { ApiError } from "../api/client";
import type { JobStatus } from "../api/types";

const STATUS_KIND: Record<JobStatus, string> = {
  submitted: "active",
  queued: "active",
  validating: "active",
  metadata_extracted: "active",
  caption_checked: "active",
  needs_user_action: "action",
  extracting_audio: "active",
  transcribing: "active",
  normalizing: "active",
  quality_checking: "active",
  drafted: "review",
  in_review: "review",
  approved: "approved",
  exported: "approved",
  failed: "failed",
  cancelled: "failed",
};

export function StatusBadge({ status }: { status: JobStatus }) {
  return <span className={`badge status-${STATUS_KIND[status]}`}>{status}</span>;
}

export function ActionChip({ action }: { action: string }) {
  if (!action) return null;
  return <span className="badge chip-action">{action.replace(/_/g, " ")}</span>;
}

export function ErrorBox({ error, prefix }: { error: unknown; prefix?: string }) {
  if (!error) return null;
  let code = "UNKNOWN";
  let message = "Something went wrong.";
  if (error instanceof ApiError) {
    code = error.code;
    message = error.message;
  } else if (error instanceof Error) {
    message = error.message;
  }
  return (
    <div className="error-box" role="alert">
      {prefix ? <strong>{prefix} </strong> : null}
      <code>{code}</code> — {message}
    </div>
  );
}

export function Loading({ label = "Loading…" }: { label?: string }) {
  return <div className="muted loading">{label}</div>;
}

export function EmptyState({ children }: { children: ReactNode }) {
  return <div className="empty-state">{children}</div>;
}

export function formatMs(ms: number): string {
  const totalSeconds = Math.floor(ms / 1000);
  const h = Math.floor(totalSeconds / 3600);
  const m = Math.floor((totalSeconds % 3600) / 60);
  const s = totalSeconds % 60;
  const mm = String(m).padStart(2, "0");
  const ss = String(s).padStart(2, "0");
  return h > 0 ? `${h}:${mm}:${ss}` : `${mm}:${ss}`;
}

export function formatDuration(seconds: number): string {
  if (!seconds) return "—";
  return formatMs(seconds * 1000);
}

export function formatTimestamp(iso: string): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString();
}
