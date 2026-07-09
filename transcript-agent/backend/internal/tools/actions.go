package tools

import (
	"context"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	"github.com/aaraminds/transcript-agent/internal/domain"
	"github.com/aaraminds/transcript-agent/internal/exporter"
	"github.com/aaraminds/transcript-agent/internal/objectstore"
	"github.com/aaraminds/transcript-agent/internal/state"
)

// CloneToVersion copies the segments of source into a new version of the
// given type. Used for reviewed drafts (R7) and immutable approved versions.
func (t *Toolset) CloneToVersion(ctx context.Context, job *domain.Job, source *domain.TranscriptVersion, versionType, createdBy string, immutable bool) (*domain.TranscriptVersion, error) {
	segs, err := t.Stores.Transcripts.ListSegments(ctx, source.TranscriptVersionID)
	if err != nil {
		return nil, err
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
	if err := t.Stores.Transcripts.CreateVersion(ctx, version, newSegs); err != nil {
		return nil, err
	}
	if err := t.Audit(ctx, &job.JobID, "user", createdBy, "transcript.version_created", map[string]any{
		"transcript_version_id": version.TranscriptVersionID.String(),
		"version_type":          versionType,
		"source_version_id":     source.TranscriptVersionID.String(),
		"is_immutable":          immutable,
		"segment_count":         len(newSegs),
	}); err != nil {
		return nil, err
	}
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
	if err := t.Audit(ctx, &version.JobID, "user", editedBy, "transcript.segment_edited", payload); err != nil {
		return nil, err
	}
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
	approvedVersion, err := t.CloneToVersion(ctx, job, reviewed, domain.VersionApproved, approvedBy, true)
	if err != nil {
		return nil, nil, err
	}
	prior, err := t.Stores.Approvals.CurrentApproval(ctx, job.JobID)
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
	if err := t.Stores.Approvals.CreateApproval(ctx, approval); err != nil {
		return nil, nil, err
	}
	if prior != nil {
		prior.SupersededByApprovalID = &approval.ApprovalID
		if err := t.Stores.Approvals.UpdateApproval(ctx, prior); err != nil {
			return nil, nil, err
		}
		if err := t.Audit(ctx, &job.JobID, "system", "approve_transcript", "approval.superseded", map[string]any{
			"superseded_approval_id":    prior.ApprovalID.String(),
			"superseded_by_approval_id": approval.ApprovalID.String(),
			"superseded_version_id":     prior.ApprovedTranscriptVersionID.String(),
		}); err != nil {
			return nil, nil, err
		}
	}
	if err := state.Transition(job, domain.StatusApproved); err != nil {
		return nil, nil, err
	}
	job.LastError = nil
	if err := t.Stores.Jobs.UpdateJob(ctx, job); err != nil {
		return nil, nil, err
	}
	if err := t.Audit(ctx, &job.JobID, "user", approvedBy, "transcript.approved", map[string]any{
		"approval_id":                    approval.ApprovalID.String(),
		"reviewed_transcript_version_id": reviewedVersionID.String(),
		"approved_transcript_version_id": approvedVersion.TranscriptVersionID.String(),
		"approval_note":                  note,
	}); err != nil {
		return nil, nil, err
	}
	return approval, approvedVersion, nil
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
	if err := t.Audit(ctx, &job.JobID, "tool", "generate_summary", "tool.generate_summary.completed",
		map[string]any{
			"summary_id":                   summary.SummaryID.String(),
			"source_transcript_version_id": source.TranscriptVersionID.String(),
			"job_config_id":                cfg.JobConfigID.String(),
			"summary_max_words":            cfg.SummaryMaxWords,
			"summary_style":                cfg.SummaryStyle,
			"validation_status":            validation,
		}); err != nil {
		return nil, err
	}
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
		if err := state.Transition(job, domain.StatusExported); err != nil {
			return nil, err
		}
		if err := t.Stores.Jobs.UpdateJob(ctx, job); err != nil {
			return nil, err
		}
	}
	ids := make([]string, len(records))
	for i, r := range records {
		ids[i] = r.ExportID.String()
	}
	if err := t.Audit(ctx, &job.JobID, "tool", "export_transcript", "tool.export_transcript.completed",
		map[string]any{
			"approved_transcript_version_id": approved.TranscriptVersionID.String(),
			"formats":                        formats,
			"export_ids":                     ids,
			"requested_by":                   requestedBy,
		}); err != nil {
		return nil, err
	}
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
	priorHash := URIHash(job.SourceURI)
	// Prior artifacts stay in storage under retention, marked superseded.
	if err := t.Stores.Artifacts.MarkArtifactsSuperseded(ctx, job.JobID); err != nil {
		return nil, err
	}
	job.SourceType = in.SourceType
	job.SourceURI = in.SourceURI
	job.OwnershipAttested = true
	job.DurationSeconds = 0
	job.ActionRequired = ""
	job.LastError = nil
	job.CaptionsAvailable = false
	job.CaptionTrackID = ""
	job.CaptionReuse = nil
	if err := state.Transition(job, domain.StatusQueued); err != nil {
		return nil, err
	}
	if err := t.Stores.Jobs.UpdateJob(ctx, job); err != nil {
		return nil, err
	}
	if err := t.Audit(ctx, &job.JobID, "user", in.ReplacedBy, "job.media_replaced", map[string]any{
		"prior_source_uri_hash": priorHash,
		"new_source_uri_hash":   URIHash(in.SourceURI),
		"source_type":           in.SourceType,
		"ownership_attested":    true,
	}); err != nil {
		return nil, err
	}
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
	job.CancelReason = reason
	job.ActionRequired = ""
	if err := state.Transition(job, domain.StatusCancelled); err != nil {
		return nil, err
	}
	if err := t.Stores.Jobs.UpdateJob(ctx, job); err != nil {
		return nil, err
	}
	if err := t.Audit(ctx, &job.JobID, "user", cancelledBy, "job.cancelled", map[string]any{
		"reason":       reason,
		"prior_status": string(priorStatus),
	}); err != nil {
		return nil, err
	}
	return job, nil
}
