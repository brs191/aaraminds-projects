export type SourceType = "youtube" | "upload";

export type JobStatus =
  | "submitted"
  | "queued"
  | "validating"
  | "metadata_extracted"
  | "caption_checked"
  | "needs_user_action"
  | "extracting_audio"
  | "transcribing"
  | "normalizing"
  | "quality_checking"
  | "drafted"
  | "in_review"
  | "approved"
  | "exported"
  | "failed"
  | "cancelled";

export type ActionRequired =
  | ""
  | "caption_decision"
  | "replace_media"
  | "duration_exceeded";

export interface JobConfig {
  confidence_threshold: number;
  enable_diarization: boolean;
  language: string;
  style_policy_id: string;
  summary_max_words: number;
  summary_style: string;
  stt_provider: string;
}

export interface ApiErrorBody {
  error: { code: string; message: string };
}

export interface Job {
  job_id: string;
  source_type: SourceType;
  source_uri: string;
  status: JobStatus;
  submitted_by: string;
  ownership_attested: boolean;
  language: string;
  /** Snapshotted when the job enters `validating`; null before that. */
  job_config: JobConfig | null;
  duration_seconds: number;
  action_required: ActionRequired;
  last_error: { code: string; message: string } | null;
  created_at: string;
  updated_at: string;
}

export type VersionType = "raw" | "clean" | "reviewed" | "approved";

export interface TranscriptVersion {
  transcript_version_id: string;
  version_type: VersionType;
  source_version_id: string | null;
  created_by: string;
  is_immutable: boolean;
  created_at: string;
}

export interface SegmentFlags {
  low_confidence?: boolean;
  caption_origin?: boolean;
}

export interface Segment {
  segment_id: string;
  start_ms: number;
  end_ms: number;
  speaker_label: string;
  text: string;
  confidence: number | null;
  flags: SegmentFlags | null;
}

export interface Summary {
  summary_id: string;
  text: string;
  source_transcript_version_id: string;
  validation_status: "passed" | "needs_review" | "failed";
  validation_notes: string | null;
  created_at: string;
}

export interface QualityIssue {
  issue_type: string;
  severity: string;
  start_ms: number;
  end_ms: number;
  message: string;
}

export interface QualityReport {
  quality_score: number | null;
  confidence_threshold: number;
  average_confidence: number | null;
  low_confidence_segment_count: number;
  coverage_gap_seconds: number;
  timestamp_gap_count: number;
  diarization_warning_count: number;
  confidence_unavailable: boolean;
  issues: QualityIssue[];
}

export type ExportFormat = "txt" | "md" | "srt" | "vtt";

export interface ExportArtifact {
  export_id: string;
  format: ExportFormat;
  validation_status: string;
  download_url: string;
  /** Transcript version this export was generated from. */
  approved_transcript_version_id: string;
  /** True when a newer approval superseded the version behind this export. */
  superseded: boolean;
  created_at: string;
}

/** Item of GET /jobs/{jobID}/approvals — newest first. */
export interface Approval {
  approval_id: string;
  approved_transcript_version_id: string;
  approved_by: string;
  approved_at: string;
  approval_note: string;
  superseded_by_approval_id: string | null;
}

/** Response of POST /uploads (multipart). */
export interface UploadResponse {
  upload_uri: string;
  filename: string;
  size_bytes: number;
  mime_type: string;
}

export type SignedLinkKind = "export" | "audio";

/** Response of POST /signed-links. `url` is site-relative and embeds ?token=; valid 15 min. */
export interface SignedLink {
  url: string;
  expires_at: string;
}

export interface AuditEvent {
  audit_event_id: string;
  actor_type: string;
  actor_id: string;
  event_type: string;
  event_payload: unknown;
  created_at: string;
}

/** Statuses where the backend is actively working — worth polling. */
export const ACTIVE_STATUSES: JobStatus[] = [
  "submitted",
  "queued",
  "validating",
  "metadata_extracted",
  "caption_checked",
  "extracting_audio",
  "transcribing",
  "normalizing",
  "quality_checking",
];

export const TERMINAL_STATUSES: JobStatus[] = ["failed", "cancelled"];

export function isJobActive(job: Job): boolean {
  return ACTIVE_STATUSES.includes(job.status);
}

/** Happy-path pipeline order, used to render the status timeline. */
export const PIPELINE_ORDER: JobStatus[] = [
  "submitted",
  "queued",
  "validating",
  "metadata_extracted",
  "caption_checked",
  "extracting_audio",
  "transcribing",
  "normalizing",
  "quality_checking",
  "drafted",
  "in_review",
  "approved",
  "exported",
];

/**
 * Pipeline order for the caption-reuse path: audio extraction and transcription
 * never run, so they must not render as completed steps.
 */
export const CAPTION_PIPELINE_ORDER: JobStatus[] = PIPELINE_ORDER.filter(
  (s) => s !== "extracting_audio" && s !== "transcribing",
);

// ---------- Library (podcast feeds) ----------

export interface Feed {
  feed_id: string;
  feed_url: string;
  title: string;
  description: string;
  image_url: string | null;
  auto_transcribe: boolean;
  episode_count: number;
  last_polled_at: string | null;
  poll_error: string | null;
  created_at: string;
}

export interface Episode {
  episode_id: string;
  feed_id: string;
  feed_title: string;
  title: string;
  description: string;
  published_at: string | null;
  duration_seconds: number | null;
  audio_url: string;
  /** Null until a transcription job exists for this episode. */
  job_id: string | null;
  job_status: string | null;
  created_at: string;
}

/**
 * Item of GET /library/search. `snippet` is plain text with matches wrapped in
 * <b>…</b> — the ONLY markup the contract allows. Parse the markers manually;
 * never inject the string as HTML. Episode/feed fields are null for hits from
 * approved non-library transcripts (the search spans both).
 */
export interface LibrarySearchResult {
  episode_id: string | null;
  episode_title: string | null;
  feed_title: string | null;
  job_id: string;
  transcript_version_id: string;
  segment_id: string;
  start_ms: number;
  snippet: string;
  rank: number;
}

const JOB_STATUS_SET = new Set<string>([
  ...PIPELINE_ORDER,
  "needs_user_action",
  "failed",
  "cancelled",
]);

/** Type guard for the loosely-typed `Episode.job_status` string. */
export function isJobStatus(status: string): status is JobStatus {
  return JOB_STATUS_SET.has(status);
}

/**
 * Library jobs stop at "drafted". An episode's job is "active" (worth polling)
 * when it has a status that is neither terminal nor drafted/in_review or later.
 */
const EPISODE_SETTLED_STATUSES = new Set<string>([
  "drafted",
  "in_review",
  "approved",
  "exported",
  "failed",
  "cancelled",
]);

export function isEpisodeJobActive(episode: Episode): boolean {
  return episode.job_status !== null && !EPISODE_SETTLED_STATUSES.has(episode.job_status);
}
