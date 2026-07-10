import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { api, apiMaybe, ApiError, mintSignedLink, uploadMediaFile } from "./client";
import {
  isJobActive,
  type Approval,
  type AuditEvent,
  type ExportArtifact,
  type ExportFormat,
  type Job,
  type QualityReport,
  type Segment,
  type SignedLink,
  type SourceType,
  type Summary,
  type TranscriptVersion,
  type UploadResponse,
} from "./types";

// ---------- Queries ----------

export function useJobs() {
  return useQuery({
    queryKey: ["jobs"],
    queryFn: () => api<{ jobs: Job[] }>("/jobs"),
    refetchInterval: (query) => {
      const jobs = query.state.data?.jobs ?? [];
      return jobs.some(isJobActive) ? 2000 : false;
    },
  });
}

export function useJob(jobId: string) {
  return useQuery({
    queryKey: ["job", jobId],
    queryFn: () => api<Job>(`/jobs/${jobId}`),
    refetchInterval: (query) => {
      const job = query.state.data;
      return job && isJobActive(job) ? 2000 : false;
    },
  });
}

export function useTranscriptVersions(jobId: string, enabled = true) {
  return useQuery({
    queryKey: ["versions", jobId],
    queryFn: () => api<{ versions: TranscriptVersion[] }>(`/jobs/${jobId}/transcripts`),
    enabled,
  });
}

export function useSegments(versionId: string | null) {
  return useQuery({
    queryKey: ["segments", versionId],
    queryFn: () => api<{ segments: Segment[] }>(`/transcripts/${versionId}/segments`),
    enabled: versionId !== null,
  });
}

export function useQualityReport(jobId: string) {
  return useQuery({
    queryKey: ["quality-report", jobId],
    queryFn: () => apiMaybe<QualityReport>(`/jobs/${jobId}/quality-report`),
  });
}

export function useSummary(jobId: string) {
  return useQuery({
    queryKey: ["summary", jobId],
    queryFn: () => apiMaybe<Summary>(`/jobs/${jobId}/summary`),
  });
}

export function useExports(jobId: string) {
  return useQuery({
    queryKey: ["exports", jobId],
    queryFn: () => api<{ exports: ExportArtifact[] }>(`/jobs/${jobId}/exports`),
  });
}

export function useAudit(jobId: string) {
  return useQuery({
    queryKey: ["audit", jobId],
    queryFn: () => api<{ events: AuditEvent[] }>(`/jobs/${jobId}/audit`),
  });
}

/** Approvals chain, newest first (GET /jobs/{jobID}/approvals). */
export function useApprovals(jobId: string) {
  return useQuery({
    queryKey: ["approvals", jobId],
    queryFn: () => api<{ approvals: Approval[] }>(`/jobs/${jobId}/approvals`),
  });
}

// ---------- Mutations ----------

/**
 * Every state-changing action writes an audit event (L3), so audit is
 * invalidated together with the job queries.
 */
function useInvalidateJob() {
  const qc = useQueryClient();
  return (jobId: string) => {
    void qc.invalidateQueries({ queryKey: ["jobs"] });
    void qc.invalidateQueries({ queryKey: ["job", jobId] });
    void qc.invalidateQueries({ queryKey: ["audit", jobId] });
  };
}

export interface SubmitJobInput {
  source_type: SourceType;
  source_uri: string;
  language: string;
  ownership_attested: boolean;
}

export function useUploadMedia() {
  return useMutation<UploadResponse, unknown, File>({
    mutationFn: (file) => uploadMediaFile(file),
  });
}

/**
 * Signed audio link for a job. Resolves to null when the job has no audio
 * artifact (404 AUDIO_NOT_AVAILABLE — e.g. the caption-reuse path).
 */
export function useAudioLink(jobId: string, enabled = true) {
  return useQuery<SignedLink | null>({
    queryKey: ["audio-link", jobId],
    queryFn: async () => {
      try {
        return await mintSignedLink("audio", jobId);
      } catch (err) {
        if (err instanceof ApiError && err.status === 404) return null;
        throw err;
      }
    },
    enabled,
    // Links expire after 15 min; treat as fresh slightly less than that.
    staleTime: 14 * 60 * 1000,
    retry: 1,
  });
}

/** Mint a signed download link for an export artifact on demand. */
export function useMintExportLink() {
  return useMutation<SignedLink, unknown, string>({
    mutationFn: (exportId) => mintSignedLink("export", exportId),
  });
}

export function useSubmitJob() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: SubmitJobInput) => api<Job>("/jobs", { method: "POST", json: input }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["jobs"] });
    },
  });
}

export function useCaptionDecision(jobId: string) {
  const invalidate = useInvalidateJob();
  return useMutation({
    mutationFn: (reuse_captions: boolean) =>
      api<Job>(`/jobs/${jobId}/caption-decision`, { method: "POST", json: { reuse_captions } }),
    onSuccess: () => invalidate(jobId),
  });
}

export interface ReplaceMediaInput {
  source_type: SourceType;
  source_uri: string;
  ownership_attested: boolean;
}

export function useReplaceMedia(jobId: string) {
  const invalidate = useInvalidateJob();
  return useMutation({
    mutationFn: (input: ReplaceMediaInput) =>
      api<Job>(`/jobs/${jobId}/replace-media`, { method: "POST", json: input }),
    onSuccess: () => invalidate(jobId),
  });
}

export function useCancelJob(jobId: string) {
  const invalidate = useInvalidateJob();
  return useMutation({
    mutationFn: (reason: string) =>
      api<Job>(`/jobs/${jobId}/cancel`, { method: "POST", json: { reason } }),
    onSuccess: () => invalidate(jobId),
  });
}

export function useStartReview(jobId: string) {
  const qc = useQueryClient();
  const invalidate = useInvalidateJob();
  return useMutation({
    mutationFn: () => api<TranscriptVersion>(`/jobs/${jobId}/review`, { method: "POST", json: {} }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["versions", jobId] });
      invalidate(jobId);
    },
  });
}

export interface PatchSegmentInput {
  segmentId: string;
  text?: string;
  speaker_label?: string;
}

export function usePatchSegment(versionId: string | null, jobId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ segmentId, ...patch }: PatchSegmentInput) =>
      api<Segment>(`/transcripts/${versionId}/segments/${segmentId}`, {
        method: "PATCH",
        json: patch,
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["segments", versionId] });
      void qc.invalidateQueries({ queryKey: ["audit", jobId] });
    },
  });
}

export interface RenameSpeakerInput {
  segmentIds: string[];
  to: string;
}

export interface RenameSpeakerResult {
  total: number;
  failed: { segmentId: string; error: unknown }[];
}

/**
 * Bulk speaker rename (C3): PATCHes every segment via Promise.allSettled so a
 * partial failure reports exactly which segments failed, and invalidates the
 * segments query ONCE after the whole batch instead of per PATCH.
 */
export function useRenameSpeaker(versionId: string | null, jobId: string) {
  const qc = useQueryClient();
  return useMutation<RenameSpeakerResult, unknown, RenameSpeakerInput>({
    mutationFn: async ({ segmentIds, to }) => {
      const results = await Promise.allSettled(
        segmentIds.map((segmentId) =>
          api<Segment>(`/transcripts/${versionId}/segments/${segmentId}`, {
            method: "PATCH",
            json: { speaker_label: to },
          }),
        ),
      );
      const failed: RenameSpeakerResult["failed"] = [];
      results.forEach((r, i) => {
        if (r.status === "rejected") failed.push({ segmentId: segmentIds[i], error: r.reason });
      });
      return { total: segmentIds.length, failed };
    },
    onSettled: () => {
      void qc.invalidateQueries({ queryKey: ["segments", versionId] });
      void qc.invalidateQueries({ queryKey: ["audit", jobId] });
    },
  });
}

export function useApprove(jobId: string) {
  const qc = useQueryClient();
  const invalidate = useInvalidateJob();
  return useMutation({
    mutationFn: (input: { reviewed_transcript_version_id: string; approval_note?: string }) =>
      api<unknown>(`/jobs/${jobId}/approve`, { method: "POST", json: input }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["versions", jobId] });
      void qc.invalidateQueries({ queryKey: ["approvals", jobId] });
      invalidate(jobId);
    },
    onError: (err) => {
      // 409 STATUS_CONFLICT: someone else changed the job state — refetch it.
      if (err instanceof ApiError && err.status === 409) invalidate(jobId);
    },
  });
}

export function useReopen(jobId: string) {
  const qc = useQueryClient();
  const invalidate = useInvalidateJob();
  return useMutation({
    mutationFn: () => api<Job>(`/jobs/${jobId}/reopen`, { method: "POST", json: {} }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["versions", jobId] });
      void qc.invalidateQueries({ queryKey: ["approvals", jobId] });
      invalidate(jobId);
    },
  });
}

export function useGenerateSummary(jobId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => api<Summary>(`/jobs/${jobId}/summary`, { method: "POST", json: {} }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["summary", jobId] });
      void qc.invalidateQueries({ queryKey: ["audit", jobId] });
    },
  });
}

export function usePatchSummary(jobId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ summaryId, text }: { summaryId: string; text: string }) =>
      api<Summary>(`/summaries/${summaryId}`, { method: "PATCH", json: { text } }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["summary", jobId] });
      void qc.invalidateQueries({ queryKey: ["audit", jobId] });
    },
  });
}

export function useCreateExports(jobId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (formats: ExportFormat[]) =>
      api<{ exports: ExportArtifact[] }>(`/jobs/${jobId}/exports`, {
        method: "POST",
        json: { formats },
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["exports", jobId] });
      void qc.invalidateQueries({ queryKey: ["job", jobId] });
      void qc.invalidateQueries({ queryKey: ["jobs"] });
      void qc.invalidateQueries({ queryKey: ["audit", jobId] });
    },
  });
}
