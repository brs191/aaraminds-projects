import { useEffect, useMemo, useRef, useState } from "react";
import { ApiError } from "../api/client";
import {
  useApprove,
  useAudioLink,
  usePatchSegment,
  useQualityReport,
  useReopen,
  useSegments,
  useStartReview,
  useTranscriptVersions,
} from "../api/hooks";
import type { Job, Segment, TranscriptVersion } from "../api/types";
import { canApprove, useIdentity } from "../identity";
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

function isLowConfidence(seg: Segment, threshold: number): boolean {
  if (seg.flags?.low_confidence) return true;
  return seg.confidence !== null && seg.confidence < threshold;
}

interface SegmentRowProps {
  segment: Segment;
  threshold: number;
  editable: boolean;
  onSave: (patch: { text?: string; speaker_label?: string }) => void;
  /** Present only when the job has playable audio. */
  onPlay?: () => void;
  /** True while audio playback is inside this segment's time range. */
  playing?: boolean;
}

function SegmentRow({ segment, threshold, editable, onSave, onPlay, playing }: SegmentRowProps) {
  const [editingText, setEditingText] = useState(false);
  const [text, setText] = useState(segment.text);
  const [speaker, setSpeaker] = useState(segment.speaker_label);

  // Re-sync local state when the server copy changes (e.g. after invalidation).
  useEffect(() => {
    setText(segment.text);
    setSpeaker(segment.speaker_label);
  }, [segment.text, segment.speaker_label]);

  const low = isLowConfidence(segment, threshold);

  const saveText = () => {
    setEditingText(false);
    if (text !== segment.text) onSave({ text });
  };
  const saveSpeaker = () => {
    const trimmed = speaker.trim();
    if (trimmed && trimmed !== segment.speaker_label) onSave({ speaker_label: trimmed });
    else setSpeaker(segment.speaker_label);
  };

  const rowClass = [
    "segment",
    low ? "low-confidence" : "",
    playing ? "playing" : "",
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div className={rowClass}>
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
          onClick={editable ? () => setEditingText(true) : undefined}
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

  useEffect(() => {
    if (!from && labels.length > 0) setFrom(labels[0]);
    else if (from && !labels.includes(from) && labels.length > 0) setFrom(labels[0]);
  }, [labels, from]);

  return (
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
      <button
        disabled={busy || !from || !to.trim() || to.trim() === from}
        onClick={() => {
          onRename(from, to.trim());
          setTo("");
        }}
      >
        {busy ? "Renaming…" : "Rename"}
      </button>
    </div>
  );
}

function ApproveDialog({
  onConfirm,
  onCancel,
  busy,
  editsPending,
  error,
}: {
  onConfirm: (note: string) => void;
  onCancel: () => void;
  busy: boolean;
  /** A segment PATCH (or bulk rename) is still in flight — block confirmation. */
  editsPending: boolean;
  error: unknown;
}) {
  const [note, setNote] = useState("");
  const conflict =
    error instanceof ApiError && (error.code === "STATUS_CONFLICT" || error.status === 409);
  return (
    <div className="modal-overlay" role="dialog" aria-modal="true" aria-label="Approve transcript">
      <div className="modal">
        <h2>Approve transcript</h2>
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

  const startReview = useStartReview(job.job_id);
  const patchSegment = usePatchSegment(selected ? selected.transcript_version_id : null);
  const approve = useApprove(job.job_id);
  const reopen = useReopen(job.job_id);

  const [showApprove, setShowApprove] = useState(false);
  const [renaming, setRenaming] = useState(false);
  const [renameError, setRenameError] = useState<unknown>(null);

  // Audio playback (PRD R7). Signed link resolves to null when the job has no
  // audio artifact (caption-reuse path) — render a quiet note instead of a player.
  const audioLink = useAudioLink(job.job_id);
  const audioRef = useRef<HTMLAudioElement | null>(null);
  const remintedOnce = useRef(false);
  const [audioBroken, setAudioBroken] = useState(false);
  const [playingSegmentId, setPlayingSegmentId] = useState<string | null>(null);

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
  const threshold = job.job_config?.confidence_threshold ?? 0.8;
  const captionOrigin = qualityQuery.data?.confidence_unavailable === true;
  const jobTerminal = job.status === "failed" || job.status === "cancelled";
  const canStartReview = !hasReviewed && !jobTerminal && job.status !== "approved";
  const approveAllowed = canApprove(identity);
  const canApproveNow =
    reviewedVersion !== null && !jobTerminal && job.status !== "approved" && job.status !== "exported";
  const canReopen = job.status === "approved" || job.status === "exported";

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

  const handleAudioError = () => {
    // Most likely the 15-minute token expired: re-mint the link once automatically.
    if (!remintedOnce.current) {
      remintedOnce.current = true;
      void audioLink.refetch();
      return;
    }
    setAudioBroken(true);
  };

  const renameEverywhere = async (from: string, to: string) => {
    if (!selected) return;
    setRenaming(true);
    setRenameError(null);
    try {
      const targets = segments.filter((s) => s.speaker_label === from);
      for (const seg of targets) {
        await patchSegment.mutateAsync({ segmentId: seg.segment_id, speaker_label: to });
      }
    } catch (err) {
      setRenameError(err);
    } finally {
      setRenaming(false);
    }
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
            {canReopen && (
              <button disabled={reopen.isPending} onClick={() => reopen.mutate()}>
                {reopen.isPending ? "Reopening…" : "Reopen for correction"}
              </button>
            )}
          </div>
        </div>

        <ErrorBox error={startReview.error} prefix="Start review failed:" />
        <ErrorBox error={reopen.error} prefix="Reopen failed:" />
        <ErrorBox error={patchSegment.error} prefix="Edit failed:" />
        <ErrorBox error={renameError} prefix="Rename failed:" />

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
              />
            ))}
          </div>
        )}
      </div>

      {showApprove && reviewedVersion && (
        <ApproveDialog
          busy={approve.isPending}
          editsPending={patchSegment.isPending || renaming}
          error={approve.error}
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
