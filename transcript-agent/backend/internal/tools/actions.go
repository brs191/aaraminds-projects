package tools

import (
	"context"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	"github.com/aaraminds/transcript-agent/internal/domain"
	"github.com/aaraminds/transcript-agent/internal/exporter"
	"github.com/aaraminds/transcript-agent/internal/metrics"
	"github.com/aaraminds/transcript-agent/internal/objectstore"
	"github.com/aaraminds/transcript-agent/internal/state"
	"github.com/aaraminds/transcript-agent/internal/store"
)

// buildClone constructs (without persisting) a new version of the given type
// carrying copies of source's segments. Persistence happens either via
// CreateVersion (review drafts) or inside the atomic approve/reopen store
// operations.
func (t *Toolset) buildClone(ctx context.Context, job *domain.Job, source *domain.TranscriptVersion, versionType, createdBy string, immutable bool) (*domain.TranscriptVersion, []*domain.Segment, error) {
	segs, err := t.Stores.Transcripts.ListSegments(ctx, source.TranscriptVersionID)
	if err != nil {
		return nil, nil, err
	}
	srcID := source.TranscriptVersionID
	version := &domain.TranscriptVersion{
		TranscriptVersionID: uuid.New(), JobID: job.JobID,
		VersionType: versionType, SourceVersionID: &srcID,
		CreatedBy: createdBy, IsImmutable: immutable, CreatedAt: time.Now().UTC(),
	}
	newSegs := make([]*domain.Segment, len(segs))
	for i, s := range segs {
		c := *s
		c.SegmentID = uuid.New()
		c.TranscriptVersionID = version.TranscriptVersionID
		if s.Flags != nil {
			f := make(map[string]bool, len(s.Flags))
			for k, v := range s.Flags {
				f[k] = v
			}
			c.Flags = f
		}
		newSegs[i] = &c
	}
	return version, newSegs, nil
}

func versionCreatedPayload(version, source *domain.TranscriptVersion, segmentCount int) map[string]any {
	return map[string]any{
		"transcript_version_id": version.TranscriptVersionID.String(),
		"version_type":          version.VersionType,
		"source_version_id":     source.TranscriptVersionID.String(),
		"is_immutable":          version.IsImmutable,
		"segment_count":         segmentCount,
	}
}

// CloneToVersion copies the segments of source into a new persisted version
// of the given type. Used for reviewed drafts (R7).
func (t *Toolset) CloneToVersion(ctx context.Context, job *domain.Job, source *domain.TranscriptVersion, versionType, createdBy string, immutable bool) (*domain.TranscriptVersion, error) {
	version, newSegs, err := t.buildClone(ctx, job, source, versionType, createdBy, immutable)
	if err != nil {
		return nil, err
	}
	if err := t.Stores.Transcripts.CreateVersion(ctx, version, newSegs); err != nil {
		return nil, err
	}
	// Informational event: fire-and-forget (audit failures are logged/counted).
	t.Audit(ctx, &job.JobID, "user", createdBy, "transcript.version_created",
		versionCreatedPayload(version, source, len(newSegs)))
	return version, nil
}

// EditSegment applies a reviewer edit to a segment of a mutable reviewed
// version (PRD R7). Every edit is audited.
func (t *Toolset) EditSegment(ctx context.Context, versionID, segmentID uuid.UUID, text, speakerLabel *string, editedBy string) (*domain.Segment, error) {
	version, err := t.Stores.Transcripts.GetVersion(ctx, versionID)
	if err != nil {
		return nil, err
	}
	if version.VersionType != domain.VersionReviewed || version.IsImmutable {
		// Frozen API contract: editing anything but a mutable reviewed version
		// answers 409 TRANSCRIPT_VERSION_IMMUTABLE.
		return nil, domain.E(domain.CodeTranscriptVersionImmutable,
			"segments can be edited only on mutable reviewed versions; %s is %s (immutable=%t)",
			versionID, version.VersionType, version.IsImmutable)
	}
	seg, err := t.Stores.Transcripts.GetSegment(ctx, segmentID)
	if err != nil {
		return nil, err
	}
	if seg.TranscriptVersionID != versionID {
		return nil, domain.E(domain.CodeSegmentNotFound, "segment %s does not belong to version %s", segmentID, versionID)
	}
	payload := map[string]any{
		"transcript_version_id": versionID.String(),
		"segment_id":            segmentID.String(),
	}
	if text != nil && *text != seg.Text {
		payload["text_before"] = seg.Text
		payload["text_after"] = *text
		seg.Text = *text
	}
	if speakerLabel != nil && *speakerLabel != seg.SpeakerLabel {
		payload["speaker_before"] = seg.SpeakerLabel
		payload["speaker_after"] = *speakerLabel
		seg.SpeakerLabel = *speakerLabel
	}
	if err := t.Stores.Transcripts.UpdateSegment(ctx, seg); err != nil {
		return nil, err
	}
	// Informational event: fire-and-forget (audit failures are logged/counted).
	t.Audit(ctx, &version.JobID, "user", editedBy, "transcript.segment_edited", payload)
	return seg, nil
}

// ---------------------------------------------------------------------
// 14.10 approve_transcript
// ---------------------------------------------------------------------

// ApproveTranscript creates the immutable approved version and the approval
// record (PRD 14.10). If a prior approval exists, it is superseded via
// approvals.superseded_by_approval_id (PRD 11.4).
func (t *Toolset) ApproveTranscript(ctx context.Context, job *domain.Job, reviewedVersionID uuid.UUID, approvedBy, note string) (*domain.Approval, *domain.TranscriptVersion, error) {
	if job.Status != domain.StatusInReview {
		return nil, nil, domain.E(domain.CodeTranscriptVersionNotReviewable,
			"job %s is %s; approval requires in_review", job.JobID, job.Status)
	}
	reviewed, err := t.Stores.Transcripts.GetVersion(ctx, reviewedVersionID)
	if err != nil {
		return nil, nil, err
	}
	if reviewed.JobID != job.JobID {
		return nil, nil, domain.E(domain.CodeTranscriptVersionNotReviewable,
			"version %s does not belong to job %s", reviewedVersionID, job.JobID)
	}
	if reviewed.VersionType != domain.VersionReviewed || reviewed.IsImmutable {
		return nil, nil, domain.E(domain.CodeTranscriptVersionNotReviewable,
			"version %s is %s (immutable=%t); only mutable reviewed versions can be approved",
			reviewedVersionID, reviewed.VersionType, reviewed.IsImmutable)
	}
	report, err := t.Stores.Quality.LatestReportByJob(ctx, job.JobID)
	if err != nil && domain.CodeOf(err) != domain.CodeQualityReportNotFound {
		return nil, nil, err
	}
	if report != nil {
		var critical []string
		for _, issue := range report.Issues {
			if strings.EqualFold(issue.Severity, "critical") {
				critical = append(critical, issue.IssueType)
			}
		}
		if len(critical) > 0 {
			return nil, nil, domain.E(domain.CodeOpenCriticalIssues,
				"approval blocked because the latest quality report has critical issues: %s",
				strings.Join(critical, ", "))
		}
	}
	// Build the immutable approved clone and the approval row, then persist
	// them atomically together with the in_review -> approved CAS (audit H1):
	// under a concurrent double-approve exactly one caller wins, the other
	// receives STATUS_CONFLICT and nothing is persisted for it.
	approvedVersion, approvedSegs, err := t.buildClone(ctx, job, reviewed, domain.VersionApproved, approvedBy, true)
	if err != nil {
		return nil, nil, err
	}
	approval := &domain.Approval{
		ApprovalID:                  uuid.New(),
		JobID:                       job.JobID,
		ApprovedTranscriptVersionID: approvedVersion.TranscriptVersionID,
		ApprovedBy:                  approvedBy,
		ApprovedAt:                  time.Now().UTC(),
		ApprovalNote:                note,
	}
	if err := t.AuditStrict(ctx, &job.JobID, "user", approvedBy, "transcript.approval_requested", map[string]any{
		"approval_id":                    approval.ApprovalID.String(),
		"reviewed_transcript_version_id": reviewedVersionID.String(),
		"approved_transcript_version_id": approvedVersion.TranscriptVersionID.String(),
		"approval_note":                  note,
	}); err != nil {
		return nil, nil, err
	}
	updatedJob, superseded, err := t.Stores.Review.ApproveJob(ctx, store.ApproveJobParams{
		JobID:           job.JobID,
		ApprovedVersion: approvedVersion,
		Segments:        approvedSegs,
		Approval:        approval,
	})
	if err != nil {
		return nil, nil, err
	}
	*job = *updatedJob
	// Approval already passed its strict pre-mutation audit reservation above.
	// Completion events are best-effort so a post-commit audit outage never makes
	// the API answer 503 after the approval has already been durably committed.
	t.Audit(ctx, &job.JobID, "user", approvedBy, "transcript.version_created",
		versionCreatedPayload(approvedVersion, reviewed, len(approvedSegs)))
	for _, priorID := range superseded {
		t.Audit(ctx, &job.JobID, "system", "approve_transcript", "approval.superseded", map[string]any{
			"superseded_approval_id":    priorID.String(),
			"superseded_by_approval_id": approval.ApprovalID.String(),
		})
	}
	t.Audit(ctx, &job.JobID, "user", approvedBy, "transcript.approved", map[string]any{
		"approval_id":                    approval.ApprovalID.String(),
		"reviewed_transcript_version_id": reviewedVersionID.String(),
		"approved_transcript_version_id": approvedVersion.TranscriptVersionID.String(),
		"approval_note":                  note,
	})
	return approval, approvedVersion, nil
}

// ReopenJob implements post-approval correction (PRD 11.4): atomically CAS
// approved/exported -> in_review and create a fresh mutable reviewed version
// cloned from the current approved version (audit M7).
func (t *Toolset) ReopenJob(ctx context.Context, job *domain.Job, reopenedBy string) (*domain.Job, *domain.TranscriptVersion, error) {
	if job.Status != domain.StatusApproved && job.Status != domain.StatusExported {
		return nil, nil, domain.E(domain.CodeJobNotInActionableState,
			"reopen applies only to approved or exported jobs; job is %s", job.Status)
	}
	approved, err := t.Stores.Transcripts.LatestVersion(ctx, job.JobID, domain.VersionApproved)
	if err != nil {
		return nil, nil, err
	}
	if approved == nil {
		return nil, nil, domain.E(domain.CodeTranscriptNotFound, "job %s has no approved version to reopen from", job.JobID)
	}
	reviewed, segs, err := t.buildClone(ctx, job, approved, domain.VersionReviewed, reopenedBy, false)
	if err != nil {
		return nil, nil, err
	}
	if err := t.AuditStrict(ctx, &job.JobID, "user", reopenedBy, "job.reopen_requested",
		map[string]any{
			"from_approved_version_id": approved.TranscriptVersionID.String(),
			"reviewed_version_id":      reviewed.TranscriptVersionID.String(),
		}); err != nil {
		return nil, nil, err
	}
	updatedJob, err := t.Stores.Review.ReopenJob(ctx, store.ReopenJobParams{
		JobID:           job.JobID,
		ReviewedVersion: reviewed,
		Segments:        segs,
	})
	if err != nil {
		return nil, nil, err
	}
	*job = *updatedJob
	t.Audit(ctx, &job.JobID, "user", reopenedBy, "transcript.version_created",
		versionCreatedPayload(reviewed, approved, len(segs)))
	t.Audit(ctx, &job.JobID, "user", reopenedBy, "job.reopened",
		map[string]any{"from_approved_version_id": approved.TranscriptVersionID.String()})
	return job, reviewed, nil
}

// ---------------------------------------------------------------------
// 14.11 generate_summary
// ---------------------------------------------------------------------

func summaryWords(text string) map[string]bool {
	set := map[string]bool{}
	for _, w := range strings.Fields(text) {
		c := strings.ToLower(strings.TrimFunc(w, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) }))
		if c != "" {
			set[c] = true
		}
	}
	return set
}

// GenerateSummary produces a transcript-grounded summary (PRD 14.11, 15.3).
// Length and style come from job_config only. A deterministic grounding
// check marks summaries containing off-transcript words as needs_review.
func (t *Toolset) GenerateSummary(ctx context.Context, job *domain.Job, source *domain.TranscriptVersion, cfg *domain.JobConfig, createdBy string) (*domain.Summary, error) {
	segs, err := t.Stores.Transcripts.ListSegments(ctx, source.TranscriptVersionID)
	if err != nil {
		return nil, err
	}
	if len(segs) == 0 {
		return nil, domain.E(domain.CodeTranscriptNotFound, "transcript version %s has no segments", source.TranscriptVersionID)
	}
	var parts []string
	for _, s := range segs {
		parts = append(parts, s.Text)
	}
	transcriptText := strings.Join(parts, " ")
	text, err := t.LLM.Summarize(ctx, transcriptText, cfg.SummaryMaxWords, cfg.SummaryStyle)
	if err != nil {
		t.Audit(ctx, &job.JobID, "tool", "generate_summary", "tool.generate_summary.failed",
			map[string]any{"error_code": domain.CodeOf(err), "error": err.Error()})
		return nil, err
	}
	// Grounding validation: every summary word must appear in the transcript
	// (PRD 15.3 rule 2). Failures mark needs_review, never silently pass.
	validation := "passed"
	notes := ""
	tset := summaryWords(transcriptText)
	var ungrounded []string
	for w := range summaryWords(text) {
		if !tset[w] {
			ungrounded = append(ungrounded, w)
		}
	}
	if len(ungrounded) > 0 {
		validation = "needs_review"
		notes = "summary contains words not present in the transcript: " + strings.Join(ungrounded, ", ")
	}
	summary := &domain.Summary{
		SummaryID:                 uuid.New(),
		JobID:                     job.JobID,
		SourceTranscriptVersionID: source.TranscriptVersionID,
		Text:                      text,
		ValidationStatus:          validation,
		ValidationNotes:           notes,
		CreatedBy:                 createdBy,
		CreatedAt:                 time.Now().UTC(),
	}
	if err := t.Stores.Summaries.CreateSummary(ctx, summary); err != nil {
		return nil, err
	}
	// Informational tool event: fire-and-forget.
	t.Audit(ctx, &job.JobID, "tool", "generate_summary", "tool.generate_summary.completed",
		map[string]any{
			"summary_id":                   summary.SummaryID.String(),
			"source_transcript_version_id": source.TranscriptVersionID.String(),
			"job_config_id":                cfg.JobConfigID.String(),
			"summary_max_words":            cfg.SummaryMaxWords,
			"summary_style":                cfg.SummaryStyle,
			"validation_status":            validation,
		})
	return summary, nil
}

// ---------------------------------------------------------------------
// 14.12 export_transcript
// ---------------------------------------------------------------------

// ExportTranscript generates export artifacts from the approved version
// (PRD 14.12). Every artifact is validated before it is recorded; srt/vtt
// validation is a real parse-back (PRD R8).
func (t *Toolset) ExportTranscript(ctx context.Context, job *domain.Job, approved *domain.TranscriptVersion, formats []string, requestedBy string) ([]*domain.ExportRecord, error) {
	if approved == nil || approved.VersionType != domain.VersionApproved {
		return nil, domain.E(domain.CodeApprovedTranscriptRequired,
			"exports are generated only from an approved transcript version")
	}
	if len(formats) == 0 {
		return nil, domain.E(domain.CodeValidationError, "formats is required, e.g. [\"txt\",\"md\",\"srt\",\"vtt\"]")
	}
	for _, f := range formats {
		ok := false
		for _, s := range domain.ExportFormats {
			if f == s {
				ok = true
			}
		}
		if !ok {
			return nil, domain.E(domain.CodeValidationError,
				"unsupported export format %q; supported: %s", f, strings.Join(domain.ExportFormats, ", "))
		}
	}
	segs, err := t.Stores.Transcripts.ListSegments(ctx, approved.TranscriptVersionID)
	if err != nil {
		return nil, err
	}
	if err := t.AuditStrict(ctx, &job.JobID, "tool", "export_transcript", "tool.export_transcript.requested",
		map[string]any{
			"approved_transcript_version_id": approved.TranscriptVersionID.String(),
			"formats":                        formats,
			"requested_by":                   requestedBy,
		}); err != nil {
		return nil, err
	}
	var records []*domain.ExportRecord
	for _, format := range formats {
		data, err := exporter.Generate(format, segs)
		if err != nil {
			t.Audit(ctx, &job.JobID, "tool", "export_transcript", "tool.export_transcript.failed",
				map[string]any{"format": format, "error_code": domain.CodeOf(err), "error": err.Error()})
			return nil, err
		}
		validation := "passed"
		if err := exporter.Validate(format, data); err != nil {
			validation = "failed"
			metrics.ExportValidationFailures.Add(1)
			t.Audit(ctx, &job.JobID, "tool", "export_transcript", "export.validation_failed",
				map[string]any{"format": format, "error": err.Error()})
		}
		key := objectstore.KeyFor(job.JobID.String(), "exports",
			approved.TranscriptVersionID.String()+"."+format)
		uri, err := t.Objects.Put(ctx, key, data)
		if err != nil {
			return nil, err
		}
		art := &domain.MediaArtifact{
			ArtifactID: uuid.New(), JobID: job.JobID,
			ArtifactType: domain.ArtifactExport, URI: uri,
			MimeType: exportMime(format), SizeBytes: int64(len(data)), CreatedAt: time.Now().UTC(),
		}
		if err := t.Stores.Artifacts.CreateArtifact(ctx, art); err != nil {
			return nil, err
		}
		rec := &domain.ExportRecord{
			ExportID:                    uuid.New(),
			JobID:                       job.JobID,
			ApprovedTranscriptVersionID: approved.TranscriptVersionID,
			Format:                      format,
			ArtifactURI:                 uri,
			ValidationStatus:            validation,
			CreatedBy:                   requestedBy,
			CreatedAt:                   time.Now().UTC(),
		}
		if err := t.Stores.Artifacts.CreateExport(ctx, rec); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	if job.Status == domain.StatusApproved {
		updated, err := t.Stores.Jobs.TransitionJob(ctx, job.JobID, domain.StatusApproved, func(j *domain.Job) error {
			return state.Transition(j, domain.StatusExported)
		})
		switch {
		case err == nil:
			*job = *updated
		case domain.CodeOf(err) == domain.CodeStatusConflict:
			// A concurrent export/reopen/cancel moved the job first. The export
			// artifacts above are valid either way; do not overwrite the status.
		default:
			return nil, err
		}
	}
	ids := make([]string, len(records))
	for i, r := range records {
		ids[i] = r.ExportID.String()
	}
	t.Audit(ctx, &job.JobID, "tool", "export_transcript", "tool.export_transcript.completed",
		map[string]any{
			"approved_transcript_version_id": approved.TranscriptVersionID.String(),
			"formats":                        formats,
			"export_ids":                     ids,
			"requested_by":                   requestedBy,
		})
	return records, nil
}

func exportMime(format string) string {
	switch format {
	case "txt":
		return "text/plain; charset=utf-8"
	case "md":
		return "text/markdown; charset=utf-8"
	case "srt":
		return "application/x-subrip"
	case "vtt":
		return "text/vtt"
	}
	return "application/octet-stream"
}

// ExportMime exposes the content type mapping for download handlers.
func ExportMime(format string) string { return exportMime(format) }

// ---------------------------------------------------------------------
// 14.13 replace_job_media
// ---------------------------------------------------------------------

// ReplaceJobMediaInput is the 14.13 input contract.
type ReplaceJobMediaInput struct {
	SourceType        string `json:"source_type"`
	SourceURI         string `json:"source_uri"`
	ReplacedBy        string `json:"replaced_by"`
	ReplacedByRole    string `json:"-"`
	OwnershipAttested bool   `json:"ownership_attested"`
}

// ReplaceJobMedia swaps source media on a needs_user_action job and re-runs
// the workflow from the top (PRD 14.13).
func (t *Toolset) ReplaceJobMedia(ctx context.Context, job *domain.Job, in ReplaceJobMediaInput) (*domain.Job, error) {
	if job.Status != domain.StatusNeedsUserAction {
		return nil, domain.E(domain.CodeJobNotInActionableState,
			"media replacement is allowed only in needs_user_action; job is %s", job.Status)
	}
	if in.ReplacedBy != job.SubmittedBy && in.ReplacedByRole != domain.RoleAdmin && in.ReplacedByRole != domain.RoleReviewer {
		return nil, domain.E(domain.CodeUserNotAuthorized,
			"only the original submitter, team lead, or admin can replace job media")
	}
	if !in.OwnershipAttested {
		return nil, domain.E(domain.CodeOwnershipAttestationMissing,
			"ownership attestation is required again for replacement media")
	}
	if err := validateSourceURI(in.SourceType, in.SourceURI); err != nil {
		return nil, err
	}
	var staged *domain.MediaArtifact
	if in.SourceType == domain.SourceUpload && strings.HasPrefix(in.SourceURI, UploadURIScheme) {
		var err error
		if staged, err = t.ResolveUploadURI(ctx, in.SourceURI); err != nil {
			return nil, err
		}
	}
	priorHash := URIHash(job.SourceURI)
	if err := t.AuditStrict(ctx, &job.JobID, "user", in.ReplacedBy, "job.media_replace_requested", map[string]any{
		"prior_source_uri_hash": priorHash,
		"new_source_uri_hash":   URIHash(in.SourceURI),
		"source_type":           in.SourceType,
		"ownership_attested":    true,
	}); err != nil {
		return nil, err
	}
	// Prior artifacts stay in storage under retention, marked superseded.
	if err := t.Stores.Artifacts.MarkArtifactsSuperseded(ctx, job.JobID); err != nil {
		return nil, err
	}
	updated, err := t.Stores.Jobs.TransitionJob(ctx, job.JobID, domain.StatusNeedsUserAction, func(j *domain.Job) error {
		j.SourceType = in.SourceType
		j.SourceURI = in.SourceURI
		j.OwnershipAttested = true
		j.DurationSeconds = 0
		j.ActionRequired = ""
		j.LastError = nil
		j.CaptionsAvailable = false
		j.CaptionTrackID = ""
		j.CaptionReuse = nil
		return state.Transition(j, domain.StatusQueued)
	})
	if err != nil {
		return nil, err
	}
	*job = *updated
	if staged != nil {
		now := time.Now().UTC()
		if err := t.Stores.Artifacts.CreateArtifact(ctx, &domain.MediaArtifact{
			ArtifactID: uuid.New(), JobID: job.JobID,
			ArtifactType: domain.ArtifactSourceMedia, URI: staged.URI,
			MimeType: staged.MimeType, SizeBytes: staged.SizeBytes,
			RetentionUntil: t.RetentionUntil(now), CreatedAt: now,
		}); err != nil {
			return nil, err
		}
	}
	t.Audit(ctx, &job.JobID, "user", in.ReplacedBy, "job.media_replaced", map[string]any{
		"prior_source_uri_hash": priorHash,
		"new_source_uri_hash":   URIHash(in.SourceURI),
		"source_type":           in.SourceType,
		"ownership_attested":    true,
	})
	return job, nil
}

// ---------------------------------------------------------------------
// 14.14 cancel_job
// ---------------------------------------------------------------------

// CancelJob cancels a non-terminal job (PRD 14.14). After approval only an
// admin may cancel. Cancellation never deletes approvals, versions, or audit.
func (t *Toolset) CancelJob(ctx context.Context, job *domain.Job, cancelledBy, role, reason string) (*domain.Job, error) {
	if job.Status.Terminal() {
		return nil, domain.E(domain.CodeJobAlreadyTerminal,
			"job is already terminal (%s)", job.Status)
	}
	if (job.Status == domain.StatusApproved || job.Status == domain.StatusExported) && role != domain.RoleAdmin {
		return nil, domain.E(domain.CodeUserNotAuthorized,
			"cancellation after approval requires admin")
	}
	if cancelledBy != job.SubmittedBy && role != domain.RoleAdmin && role != domain.RoleReviewer {
		return nil, domain.E(domain.CodeUserNotAuthorized,
			"only the original submitter, team lead, or admin can cancel a job")
	}
	priorStatus := job.Status
	// CAS from the status the caller saw: if the pipeline advanced (or another
	// actor won) in the meantime the cancel answers 409 STATUS_CONFLICT and
	// the caller retries against the fresh state.
	if err := t.AuditStrict(ctx, &job.JobID, "user", cancelledBy, "job.cancel_requested", map[string]any{
		"reason":       reason,
		"prior_status": string(priorStatus),
	}); err != nil {
		return nil, err
	}
	updated, err := t.Stores.Jobs.TransitionJob(ctx, job.JobID, priorStatus, func(j *domain.Job) error {
		j.CancelReason = reason
		j.ActionRequired = ""
		return state.Transition(j, domain.StatusCancelled)
	})
	if err != nil {
		return nil, err
	}
	*job = *updated
	t.Audit(ctx, &job.JobID, "user", cancelledBy, "job.cancelled", map[string]any{
		"reason":       reason,
		"prior_status": string(priorStatus),
	})
	return job, nil
}
