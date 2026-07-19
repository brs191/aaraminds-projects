import { useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { ApiError } from "../api/client";
import {
  useApprovals,
  useApprove,
  useAudioLink,
  usePatchSegment,
  useQualityReport,
  useRenameSpeaker,
  useReopen,
  useSegments,
  useStartReview,
  useTranscriptVersions,
} from "../api/hooks";
import type { Approval, Job, Segment, TranscriptVersion } from "../api/types";
import { canApprove, canReopen as canReopenRole, useIdentity } from "../identity";
import { EmptyState, ErrorBox, Loading, formatMs, formatTimestamp } from "../components/ui";

function pickDefaultVersion(job: Job, versions: TranscriptVersion[]): TranscriptVersion | null {
  if (versions.length === 0) return null;
  const byType = (t: TranscriptVersion["version_type"]) =>
    [...versions].reverse().find((v) => v.version_type === t) ?? null;
  if (job.status === "approved" || job.status === "exported") {
    const approved = byType("approved");
    if (approved) return approved;
  }
  return byType("reviewed") ?? byType("clean") ?? byType("raw") ?? versions[versions.length - 1];
}

/**
 * H5: the server flag is authoritative. The client-side comparison is a
 * fallback applied ONLY when the job config (and thus the real threshold) is
 * loaded — there is no hardcoded default threshold.
 */
function isLowConfidence(seg: Segment, threshold: number | null): boolean {
  if (seg.flags?.low_confidence) return true;
  if (threshold === null) return false;
  return seg.confidence !== null && seg.confidence < threshold;
}

interface SegmentRowProps {
  segment: Segment;
  threshold: number | null;
  editable: boolean;
  onSave: (patch: { text?: string; speaker_label?: string }) => void;
  /** Present only when the job has playable audio. */
  onPlay?: () => void;
  /** True while audio playback is inside this segment's time range. */
  playing?: boolean;
  /** DOM id so deep links (?t=) can scroll to the row. */
  domId?: string;
  /** True when this row is the ?t= deep-link target. */
  deepLinked?: boolean;
}

function SegmentRow({
  segment,
  threshold,
  editable,
  onSave,
  onPlay,
  playing,
  domId,
  deepLinked,
}: SegmentRowProps) {
  const [editingText, setEditingText] = useState(false);
  const [text, setText] = useState(segment.text);
  const [speaker, setSpeaker] = useState(segment.speaker_label);
  // H3a: refs mirror the editing state so the server-resync effects never
  // clobber local text/speaker while the row is actively being edited.
  const editingTextRef = useRef(false);
  const editingSpeakerRef = useRef(false);

  const startTextEdit = () => {
    editingTextRef.current = true;
    setEditingText(true);
  };
  const stopTextEdit = () => {
    editingTextRef.current = false;
    setEditingText(false);
  };

  // Re-sync local state when the server copy changes (e.g. after invalidation)
  // — but never while the user is mid-edit (H3a).
  useEffect(() => {
    if (!editingTextRef.current) setText(segment.text);
  }, [segment.text]);
  useEffect(() => {
    if (!editingSpeakerRef.current) setSpeaker(segment.speaker_label);
  }, [segment.speaker_label]);

  const low = isLowConfidence(segment, threshold);

  const saveText = () => {
    stopTextEdit();
    if (text !== segment.text) onSave({ text });
  };
  const saveSpeaker = () => {
    editingSpeakerRef.current = false;
    const trimmed = speaker.trim();
    if (trimmed && trimmed !== segment.speaker_label) onSave({ speaker_label: trimmed });
    else setSpeaker(segment.speaker_label);
  };

  const rowClass = [
    "segment",
    low ? "low-confidence" : "",
    playing ? "playing" : "",
    deepLinked ? "deep-linked" : "",
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div className={rowClass} id={domId}>
      <div className="segment-meta">
        {onPlay && (
          <button
            type="button"
            className="seg-play"
            aria-label={`Play from ${formatMs(segment.start_ms)}`}
            title={`Play from ${formatMs(segment.start_ms)}`}
            onClick={onPlay}
          >
            ▶
          </button>
        )}
        <span className="mono segment-time">
          {formatMs(segment.start_ms)}–{formatMs(segment.end_ms)}
        </span>
        {editable ? (
          <input
            className="speaker-input"
            value={speaker}
            aria-label="Speaker label"
            onFocus={() => {
              editingSpeakerRef.current = true;
            }}
            onChange={(e) => setSpeaker(e.target.value)}
            onBlur={saveSpeaker}
            onKeyDown={(e) => {
              if (e.key === "Enter") (e.target as HTMLInputElement).blur();
            }}
          />
        ) : (
          <span className="speaker-label">{segment.speaker_label}</span>
        )}
        {low && <span className="badge chip-low">low confidence</span>}
        {segment.confidence !== null && (
          <span className="muted confidence-value">{segment.confidence.toFixed(2)}</span>
        )}
      </div>
      {editable && editingText ? (
        <textarea
          className="segment-editor"
          value={text}
          autoFocus
          rows={Math.max(2, Math.ceil(text.length / 80))}
          onChange={(e) => setText(e.target.value)}
          onBlur={saveText}
        />
      ) : (
        <p
          className={editable ? "segment-text editable" : "segment-text"}
          title={editable ? "Click to edit" : undefined}
          // M5: click-to-edit must also be keyboard-operable.
          role={editable ? "button" : undefined}
          tabIndex={editable ? 0 : undefined}
          aria-label={editable ? "Edit segment text" : undefined}
          onClick={editable ? startTextEdit : undefined}
          onKeyDown={
            editable
              ? (e) => {
                  if (e.key === "Enter" || e.key === " ") {
                    e.preventDefault();
                    startTextEdit();
                  }
                }
              : undefined
          }
        >
          {segment.text}
        </p>
      )}
    </div>
  );
}

function RenameSpeakerBar({
  segments,
  onRename,
  busy,
}: {
  segments: Segment[];
  onRename: (from: string, to: string) => void;
  busy: boolean;
}) {
  const labels = useMemo(
    () => Array.from(new Set(segments.map((s) => s.speaker_label))).sort(),
    [segments],
  );
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  // C3: when the target label already exists on other segments, renaming is a
  // merge — require explicit confirmation first.
  const [pendingMerge, setPendingMerge] = useState<{ from: string; to: string } | null>(null);

  useEffect(() => {
    if (!from && labels.length > 0) setFrom(labels[0]);
    else if (from && !labels.includes(from) && labels.length > 0) setFrom(labels[0]);
  }, [labels, from]);

  const startRename = () => {
    const target = to.trim();
    if (!target || target === from) return;
    if (labels.includes(target)) {
      setPendingMerge({ from, to: target });
      return;
    }
    onRename(from, target);
    setTo("");
  };

  const mergeCount = pendingMerge
    ? segments.filter((s) => s.speaker_label === pendingMerge.from).length
    : 0;

  return (
    <div className="rename-bar-wrap">
      <div className="rename-bar">
        <span className="muted">Rename speaker everywhere:</span>
        <select value={from} onChange={(e) => setFrom(e.target.value)} aria-label="Speaker to rename">
          {labels.map((l) => (
            <option key={l} value={l}>
              {l}
            </option>
          ))}
        </select>
        <span aria-hidden>→</span>
        <input
          value={to}
          onChange={(e) => setTo(e.target.value)}
          placeholder="New name"
          aria-label="New speaker name"
        />
        <button disabled={busy || !from || !to.trim() || to.trim() === from} onClick={startRename}>
          {busy ? "Renaming…" : "Rename"}
        </button>
      </div>
      {pendingMerge && (
        <div className="notice-banner" role="alertdialog" aria-label="Confirm speaker merge">
          This merges <strong>{pendingMerge.from}</strong> into <strong>{pendingMerge.to}</strong>{" "}
          — {mergeCount} segment{mergeCount === 1 ? "" : "s"}. Continue?
          <div className="button-row">
            <button
              className="primary"
              disabled={busy}
              onClick={() => {
                onRename(pendingMerge.from, pendingMerge.to);
                setPendingMerge(null);
                setTo("");
              }}
            >
              Continue
            </button>
            <button disabled={busy} onClick={() => setPendingMerge(null)}>
              Cancel
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

function ApproveDialog({
  onConfirm,
  onCancel,
  busy,
  editsPending,
  error,
  version,
  lowConfidenceCount,
}: {
  onConfirm: (note: string) => void;
  onCancel: () => void;
  busy: boolean;
  /** A segment PATCH (or bulk rename) is still in flight — block confirmation. */
  editsPending: boolean;
  error: unknown;
  /** The reviewed version being approved (M3). */
  version: TranscriptVersion;
  /** Low-confidence segment count from the quality report; null when no report. */
  lowConfidenceCount: number | null;
}) {
  const [note, setNote] = useState("");
  const conflict =
    error instanceof ApiError && (error.code === "STATUS_CONFLICT" || error.status === 409);
  return (
    <div className="modal-overlay" role="dialog" aria-modal="true" aria-label="Approve transcript">
      <div className="modal">
        <h2>Approve transcript</h2>
        <p>
          Approving version{" "}
          <span className="mono">{version.transcript_version_id.slice(0, 8)}</span> (reviewed,
          created {formatTimestamp(version.created_at)}).
        </p>
        {lowConfidenceCount !== null && lowConfidenceCount > 0 && (
          <p className="warn-text">
            {lowConfidenceCount} segment{lowConfidenceCount === 1 ? " is" : "s are"} still flagged
            low-confidence in the quality report — double-check them before approving.
          </p>
        )}
        <p>
          Per the PRD: <em>"approval creates an immutable version"</em>. After approval, further
          corrections require reopening the job and approving a new superseding version.
        </p>
        <div className="field">
          <label htmlFor="approval-note">Approval note (optional)</label>
          <textarea
            id="approval-note"
            rows={3}
            value={note}
            onChange={(e) => setNote(e.target.value)}
          />
        </div>
        {conflict ? (
          <div className="error-box" role="alert">
            Job state changed — refresh. The job was updated elsewhere (its latest state has been
            re-fetched). Close this dialog, review the current state, and approve again if it
            still applies.
          </div>
        ) : (
          <ErrorBox error={error} prefix="Approval failed:" />
        )}
        {editsPending && (
          <p className="muted hint" role="status">
            Waiting for a segment edit to finish saving…
          </p>
        )}
        <div className="button-row">
          <button
            className="primary"
            disabled={busy || editsPending}
            onClick={() => onConfirm(note.trim())}
          >
            {busy ? "Approving…" : "Confirm approval"}
          </button>
          <button disabled={busy} onClick={onCancel}>
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
}

/** Approvals chain (newest first) from GET /jobs/{id}/approvals. */
function ApprovalsCard({ approvals }: { approvals: Approval[] }) {
  if (approvals.length === 0) return null;
  return (
    <div className="card">
      <h2>Approvals</h2>
      <ol className="approvals-list">
        {approvals.map((a) => (
          <li key={a.approval_id} id={`approval-${a.approval_id}`} className="approval-item">
            <div className="approval-head">
              <span className="mono">{a.approval_id.slice(0, 8)}</span>
              {" — "}
              <strong>{a.approved_by}</strong> approved version{" "}
              <span className="mono">{a.approved_transcript_version_id.slice(0, 8)}</span>{" "}
              <span className="muted">{formatTimestamp(a.approved_at)}</span>{" "}
              {a.superseded_by_approval_id !== null && (
                <>
                  <span className="badge chip-superseded">superseded</span>{" "}
                  <a href={`#approval-${a.superseded_by_approval_id}`} className="muted">
                    by {a.superseded_by_approval_id.slice(0, 8)}
                  </a>
                </>
              )}
            </div>
            {a.approval_note && <p className="muted approval-note">"{a.approval_note}"</p>}
          </li>
        ))}
      </ol>
    </div>
  );
}

export default function ReviewTab({ job }: { job: Job }) {
  const { identity } = useIdentity();
  const versionsQuery = useTranscriptVersions(job.job_id);
  const versions = useMemo(
    () => versionsQuery.data?.versions ?? [],
    [versionsQuery.data],
  );

  const [selectedId, setSelectedId] = useState<string | null>(null);
  useEffect(() => {
    if (versions.length === 0) return;
    if (selectedId && versions.some((v) => v.transcript_version_id === selectedId)) return;
    setSelectedId(pickDefaultVersion(job, versions)?.transcript_version_id ?? null);
  }, [versions, selectedId, job]);

  const selected = versions.find((v) => v.transcript_version_id === selectedId) ?? null;
  const segmentsQuery = useSegments(selected ? selected.transcript_version_id : null);
  const qualityQuery = useQualityReport(job.job_id);
  const approvalsQuery = useApprovals(job.job_id);

  const startReview = useStartReview(job.job_id);
  const patchSegment = usePatchSegment(
    selected ? selected.transcript_version_id : null,
    job.job_id,
  );
  const renameSpeaker = useRenameSpeaker(
    selected ? selected.transcript_version_id : null,
    job.job_id,
  );
  const approve = useApprove(job.job_id);
  const reopen = useReopen(job.job_id);

  const [showApprove, setShowApprove] = useState(false);
  // C3: after a partial rename failure, which segments failed and what target
  // label to retry them with.
  const [renameFailures, setRenameFailures] = useState<{
    to: string;
    segmentIds: string[];
  } | null>(null);

  // Audio playback (PRD R7). Signed link resolves to null when the job has no
  // audio artifact (caption-reuse path) — render a quiet note instead of a player.
  const audioLink = useAudioLink(job.job_id);
  const audioRef = useRef<HTMLAudioElement | null>(null);
  const remintedOnce = useRef(false);
  const [audioBroken, setAudioBroken] = useState(false);
  const [playingSegmentId, setPlayingSegmentId] = useState<string | null>(null);

  // Deep link from library search: /jobs/{id}?t=<start_ms>. Highlight + scroll
  // to the segment containing t once segments load, and seek the audio player
  // there when its metadata is ready. Each applies exactly once per mount.
  const [searchParams] = useSearchParams();
  const deepLinkMs = useMemo(() => {
    const raw = searchParams.get("t");
    if (raw === null) return null;
    const ms = Number(raw);
    return Number.isFinite(ms) && ms >= 0 ? ms : null;
  }, [searchParams]);
  const [deepLinkSegmentId, setDeepLinkSegmentId] = useState<string | null>(null);
  const deepLinkScrolled = useRef(false);
  const deepLinkSeeked = useRef(false);
  const loadedSegments = segmentsQuery.data?.segments;
  useEffect(() => {
    if (deepLinkScrolled.current || deepLinkMs === null) return;
    if (!loadedSegments || loadedSegments.length === 0) return;
    deepLinkScrolled.current = true;
    const target =
      loadedSegments.find((s) => deepLinkMs >= s.start_ms && deepLinkMs < s.end_ms) ??
      [...loadedSegments].reverse().find((s) => s.start_ms <= deepLinkMs) ??
      loadedSegments[0];
    setDeepLinkSegmentId(target.segment_id);
    // Wait a frame so the row exists in the DOM before scrolling.
    requestAnimationFrame(() => {
      document
        .getElementById(`segment-${target.segment_id}`)
        ?.scrollIntoView({ block: "center", behavior: "smooth" });
    });
  }, [deepLinkMs, loadedSegments]);

  if (versionsQuery.isLoading) return <Loading label="Loading transcript versions…" />;
  if (versionsQuery.error)
    return <ErrorBox error={versionsQuery.error} prefix="Could not load transcript versions:" />;

  if (versions.length === 0) {
    return (
      <EmptyState>
        No transcript versions yet. Versions appear once transcription (or caption parsing)
        completes.
      </EmptyState>
    );
  }

  const reviewedVersion =
    [...versions].reverse().find((v) => v.version_type === "reviewed" && !v.is_immutable) ?? null;
  const hasReviewed = versions.some((v) => v.version_type === "reviewed");
  const editable =
    selected !== null && selected.version_type === "reviewed" && !selected.is_immutable;
  const segments = segmentsQuery.data?.segments ?? [];
  // H5: no hardcoded fallback — the client-side comparison only applies once
  // job_config (and its threshold) is loaded. Server flags always apply.
  const threshold = job.job_config?.confidence_threshold ?? null;
  // H6: caption-origin is signalled by the quality report OR by per-segment
  // caption_origin flags (the report may not exist yet for the visible version).
  const captionOrigin =
    qualityQuery.data?.confidence_unavailable === true ||
    segments.some((s) => s.flags?.caption_origin === true);
  const jobTerminal = job.status === "failed" || job.status === "cancelled";
  const canStartReview = !hasReviewed && !jobTerminal && job.status !== "approved";
  const approveAllowed = canApprove(identity);
  const reopenAllowed = canReopenRole(identity);
  const canApproveNow =
    reviewedVersion !== null && !jobTerminal && job.status !== "approved" && job.status !== "exported";
  const canReopen = job.status === "approved" || job.status === "exported";
  const approvals = approvalsQuery.data?.approvals ?? [];

  const audioAvailable = !!audioLink.data && !audioBroken;

  const seekTo = (startMs: number) => {
    const el = audioRef.current;
    if (!el) return;
    el.currentTime = startMs / 1000;
    void el.play().catch(() => {
      // Autoplay restrictions or a mid-load seek — user can press play manually.
    });
  };

  const handleTimeUpdate = () => {
    const el = audioRef.current;
    if (!el) return;
    const ms = el.currentTime * 1000;
    // Cheap linear scan — segment lists are small enough at MVP scale.
    const active = segments.find((s) => ms >= s.start_ms && ms < s.end_ms);
    const id = active?.segment_id ?? null;
    setPlayingSegmentId((prev) => (prev === id ? prev : id));
  };

  // Seek to the ?t= deep-link position once (without autoplay) as soon as the
  // audio element knows its duration — seeking before metadata is unreliable.
  const handleLoadedMetadata = () => {
    if (deepLinkMs === null || deepLinkSeeked.current) return;
    deepLinkSeeked.current = true;
    const el = audioRef.current;
    if (el) el.currentTime = deepLinkMs / 1000;
  };

  const handleAudioError = () => {
    // Most likely the 15-minute token expired: re-mint the link once automatically.
    if (!remintedOnce.current) {
      remintedOnce.current = true;
      void audioLink.refetch();
      return;
    }
    setAudioBroken(true);
  };

  const runRename = (segmentIds: string[], to: string) => {
    if (!selected || segmentIds.length === 0) return;
    setRenameFailures(null);
    renameSpeaker.mutate(
      { segmentIds, to },
      {
        onSuccess: (result) => {
          if (result.failed.length > 0) {
            setRenameFailures({ to, segmentIds: result.failed.map((f) => f.segmentId) });
          }
        },
      },
    );
  };

  const renameEverywhere = (from: string, to: string) => {
    runRename(
      segments.filter((s) => s.speaker_label === from).map((s) => s.segment_id),
      to,
    );
  };

  const renaming = renameSpeaker.isPending;
  const describeSegment = (segmentId: string): string => {
    const seg = segments.find((s) => s.segment_id === segmentId);
    return seg ? `${formatMs(seg.start_ms)}–${formatMs(seg.end_ms)}` : segmentId.slice(0, 8);
  };

  return (
    <div className="stack">
      {captionOrigin && (
        <div className="notice-banner">
          This transcript is caption-derived: provider confidence scores are unavailable and
          diarization did not run. Threshold-based flagging is skipped — review the full text.
        </div>
      )}

      <div className="card">
        <div className="review-toolbar">
          <div className="field inline">
            <label htmlFor="version-select">Version</label>
            <select
              id="version-select"
              value={selectedId ?? ""}
              onChange={(e) => setSelectedId(e.target.value)}
            >
              {versions.map((v) => (
                <option key={v.transcript_version_id} value={v.transcript_version_id}>
                  {v.version_type}
                  {v.is_immutable ? " (immutable)" : ""} — {formatTimestamp(v.created_at)}
                </option>
              ))}
            </select>
          </div>

          <div className="button-row">
            {canStartReview && (
              <button
                className="primary"
                disabled={startReview.isPending}
                onClick={() =>
                  startReview.mutate(undefined, {
                    onSuccess: (v) => setSelectedId(v.transcript_version_id),
                  })
                }
              >
                {startReview.isPending ? "Starting…" : "Start review"}
              </button>
            )}
            {canApproveNow && approveAllowed && (
              <button
                className="primary"
                onClick={() => {
                  approve.reset(); // do not carry a stale error into a fresh dialog
                  setShowApprove(true);
                }}
              >
                Approve…
              </button>
            )}
            {canApproveNow && !approveAllowed && (
              <span className="muted hint">
                Approval requires the reviewer or admin role (currently {identity.role}).
              </span>
            )}
            {canReopen && reopenAllowed && (
              <button disabled={reopen.isPending} onClick={() => reopen.mutate()}>
                {reopen.isPending ? "Reopening…" : "Reopen for correction"}
              </button>
            )}
            {canReopen && !reopenAllowed && (
              <span className="muted hint">
                Reopening requires the reviewer or admin role (currently {identity.role}).
              </span>
            )}
          </div>
        </div>

        <ErrorBox error={startReview.error} prefix="Start review failed:" />
        <ErrorBox error={reopen.error} prefix="Reopen failed:" />
        <ErrorBox error={patchSegment.error} prefix="Edit failed:" />
        <ErrorBox error={renameSpeaker.error} prefix="Rename failed:" />
        {renameFailures && (
          <div className="error-box" role="alert">
            <strong>Rename partially failed:</strong> {renameFailures.segmentIds.length} segment
            {renameFailures.segmentIds.length === 1 ? "" : "s"} could not be renamed to{" "}
            <strong>{renameFailures.to}</strong>:
            <ul className="failed-segment-list">
              {renameFailures.segmentIds.map((id) => (
                <li key={id} className="mono">
                  {describeSegment(id)}
                </li>
              ))}
            </ul>
            <div className="button-row">
              <button
                disabled={renaming}
                onClick={() => runRename(renameFailures.segmentIds, renameFailures.to)}
              >
                {renaming ? "Retrying…" : "Retry failed"}
              </button>
              <button disabled={renaming} onClick={() => setRenameFailures(null)}>
                Dismiss
              </button>
            </div>
          </div>
        )}

        {editable && (
          <>
            <p className="muted hint">
              This reviewed version is editable — click a segment's text to edit it, or change a
              speaker label inline. Changes save on blur.
            </p>
            <RenameSpeakerBar segments={segments} onRename={renameEverywhere} busy={renaming} />
          </>
        )}
        {selected && !editable && hasReviewed && selected.version_type !== "reviewed" && (
          <p className="muted hint">
            This {selected.version_type} version is read-only. Switch to the reviewed version to
            edit.
          </p>
        )}

        {audioLink.data ? (
          <div className="audio-bar">
            <audio
              key={audioLink.data.url}
              ref={audioRef}
              controls
              preload="metadata"
              src={audioLink.data.url}
              onTimeUpdate={handleTimeUpdate}
              onLoadedMetadata={handleLoadedMetadata}
              onError={handleAudioError}
            />
            {audioBroken && (
              <span className="error-text hint">
                Audio playback failed even after refreshing the link — reload the page to retry.
              </span>
            )}
          </div>
        ) : audioLink.data === null ? (
          <p className="muted hint">No audio available — caption-derived transcript.</p>
        ) : audioLink.isError ? (
          <ErrorBox error={audioLink.error} prefix="Could not get an audio link:" />
        ) : null}

        {segmentsQuery.isLoading ? (
          <Loading label="Loading segments…" />
        ) : segmentsQuery.error ? (
          <ErrorBox error={segmentsQuery.error} prefix="Could not load segments:" />
        ) : segments.length === 0 ? (
          <EmptyState>This version has no segments.</EmptyState>
        ) : (
          <div className="segments">
            {segments.map((seg) => (
              <SegmentRow
                key={seg.segment_id}
                segment={seg}
                threshold={threshold}
                editable={editable}
                onSave={(patch) => patchSegment.mutate({ segmentId: seg.segment_id, ...patch })}
                onPlay={audioAvailable ? () => seekTo(seg.start_ms) : undefined}
                playing={seg.segment_id === playingSegmentId}
                domId={`segment-${seg.segment_id}`}
                deepLinked={seg.segment_id === deepLinkSegmentId}
              />
            ))}
          </div>
        )}
      </div>

      {approvalsQuery.error ? (
        <ErrorBox error={approvalsQuery.error} prefix="Could not load approvals:" />
      ) : (
        <ApprovalsCard approvals={approvals} />
      )}

      {showApprove && reviewedVersion && (
        <ApproveDialog
          busy={approve.isPending}
          editsPending={patchSegment.isPending || renaming}
          error={approve.error}
          version={reviewedVersion}
          lowConfidenceCount={qualityQuery.data?.low_confidence_segment_count ?? null}
          onCancel={() => setShowApprove(false)}
          onConfirm={(note) =>
            approve.mutate(
              {
                reviewed_transcript_version_id: reviewedVersion.transcript_version_id,
                ...(note ? { approval_note: note } : {}),
              },
              { onSuccess: () => setShowApprove(false) },
            )
          }
        />
      )}
    </div>
  );
}
