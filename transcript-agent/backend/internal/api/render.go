package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/aaraminds/transcript-agent/internal/domain"
	"github.com/google/uuid"
)

// errorBody is the frozen error envelope: {"error":{"code","message"}}.
type errorBody struct {
	Error domain.ErrorInfo `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// httpStatusFor maps structured error codes (PRD 14/19) to HTTP statuses.
func httpStatusFor(code string) int {
	switch code {
	case domain.CodeValidationError, domain.CodeOwnershipAttestationMissing,
		domain.CodeUnsupportedSourceType, domain.CodeInvalidSourceURI,
		domain.CodeUnsupportedFormat, domain.CodeLanguageUnsupported:
		return http.StatusBadRequest
	case domain.CodeUnauthenticated:
		return http.StatusUnauthorized
	case domain.CodeUserNotAuthorized:
		return http.StatusForbidden
	case domain.CodeJobNotFound, domain.CodeTranscriptNotFound, domain.CodeSegmentNotFound,
		domain.CodeSummaryNotFound, domain.CodeExportNotFound, domain.CodeQualityReportNotFound,
		domain.CodeMediaNotFound:
		return http.StatusNotFound
	case domain.CodeTranscriptVersionImmutable, domain.CodeTranscriptVersionNotReviewable,
		domain.CodeApprovedTranscriptRequired, domain.CodeJobNotInActionableState,
		domain.CodeJobAlreadyTerminal, domain.CodeInvalidStateTransition,
		domain.CodeDisabledInMVP, domain.CodeOpenCriticalIssues:
		return http.StatusConflict
	case domain.CodeAuditWriteFailed:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

func writeError(w http.ResponseWriter, err error) {
	de := domain.AsError(err)
	writeJSON(w, httpStatusFor(de.Code), errorBody{Error: domain.ErrorInfo{Code: de.Code, Message: de.Message}})
}

// --- frozen API JSON shapes ----------------------------------------------

type jobConfigJSON struct {
	ConfidenceThreshold float64 `json:"confidence_threshold"`
	EnableDiarization   bool    `json:"enable_diarization"`
	Language            string  `json:"language"`
	StylePolicyID       string  `json:"style_policy_id"`
	SummaryMaxWords     int     `json:"summary_max_words"`
	SummaryStyle        string  `json:"summary_style"`
	STTProvider         string  `json:"stt_provider"`
}

type jobJSON struct {
	JobID             string            `json:"job_id"`
	SourceType        string            `json:"source_type"`
	SourceURI         string            `json:"source_uri"`
	Status            string            `json:"status"`
	SubmittedBy       string            `json:"submitted_by"`
	OwnershipAttested bool              `json:"ownership_attested"`
	Language          string            `json:"language"`
	JobConfig         *jobConfigJSON    `json:"job_config"`
	DurationSeconds   int               `json:"duration_seconds"`
	ActionRequired    string            `json:"action_required"`
	LastError         *domain.ErrorInfo `json:"last_error"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

func (s *Server) jobView(ctx context.Context, job *domain.Job) jobJSON {
	out := jobJSON{
		JobID:             job.JobID.String(),
		SourceType:        job.SourceType,
		SourceURI:         job.SourceURI,
		Status:            string(job.Status),
		SubmittedBy:       job.SubmittedBy,
		OwnershipAttested: job.OwnershipAttested,
		Language:          job.Language,
		DurationSeconds:   job.DurationSeconds,
		ActionRequired:    job.ActionRequired,
		LastError:         job.LastError,
		CreatedAt:         job.CreatedAt,
		UpdatedAt:         job.UpdatedAt,
	}
	if job.JobConfigID != nil {
		if cfg, err := s.Tools.Stores.Jobs.GetJobConfig(ctx, *job.JobConfigID); err == nil {
			out.JobConfig = &jobConfigJSON{
				ConfidenceThreshold: cfg.ConfidenceThreshold,
				EnableDiarization:   cfg.EnableDiarization,
				Language:            cfg.Language,
				StylePolicyID:       cfg.StylePolicyID,
				SummaryMaxWords:     cfg.SummaryMaxWords,
				SummaryStyle:        cfg.SummaryStyle,
				STTProvider:         cfg.STTProvider,
			}
		}
	}
	return out
}

type versionJSON struct {
	TranscriptVersionID string    `json:"transcript_version_id"`
	VersionType         string    `json:"version_type"`
	SourceVersionID     *string   `json:"source_version_id"`
	CreatedBy           string    `json:"created_by"`
	IsImmutable         bool      `json:"is_immutable"`
	CreatedAt           time.Time `json:"created_at"`
}

func versionView(v *domain.TranscriptVersion) versionJSON {
	out := versionJSON{
		TranscriptVersionID: v.TranscriptVersionID.String(),
		VersionType:         v.VersionType,
		CreatedBy:           v.CreatedBy,
		IsImmutable:         v.IsImmutable,
		CreatedAt:           v.CreatedAt,
	}
	if v.SourceVersionID != nil {
		id := v.SourceVersionID.String()
		out.SourceVersionID = &id
	}
	return out
}

type segmentJSON struct {
	SegmentID    string          `json:"segment_id"`
	StartMS      int             `json:"start_ms"`
	EndMS        int             `json:"end_ms"`
	SpeakerLabel string          `json:"speaker_label"`
	Text         string          `json:"text"`
	Confidence   *float64        `json:"confidence"`
	Flags        map[string]bool `json:"flags"`
}

func segmentView(sg *domain.Segment) segmentJSON {
	return segmentJSON{
		SegmentID:    sg.SegmentID.String(),
		StartMS:      sg.StartMS,
		EndMS:        sg.EndMS,
		SpeakerLabel: sg.SpeakerLabel,
		Text:         sg.Text,
		Confidence:   sg.Confidence,
		Flags:        sg.Flags,
	}
}

type approvalJSON struct {
	ApprovalID                  string    `json:"approval_id"`
	JobID                       string    `json:"job_id"`
	ApprovedTranscriptVersionID string    `json:"approved_transcript_version_id"`
	ApprovedBy                  string    `json:"approved_by"`
	ApprovedAt                  time.Time `json:"approved_at"`
	ApprovalNote                string    `json:"approval_note"`
	SupersededByApprovalID      *string   `json:"superseded_by_approval_id"`
}

func approvalView(a *domain.Approval) approvalJSON {
	out := approvalJSON{
		ApprovalID:                  a.ApprovalID.String(),
		JobID:                       a.JobID.String(),
		ApprovedTranscriptVersionID: a.ApprovedTranscriptVersionID.String(),
		ApprovedBy:                  a.ApprovedBy,
		ApprovedAt:                  a.ApprovedAt,
		ApprovalNote:                a.ApprovalNote,
	}
	if a.SupersededByApprovalID != nil {
		id := a.SupersededByApprovalID.String()
		out.SupersededByApprovalID = &id
	}
	return out
}

type summaryJSON struct {
	SummaryID                 string    `json:"summary_id"`
	Text                      string    `json:"text"`
	SourceTranscriptVersionID string    `json:"source_transcript_version_id"`
	ValidationStatus          string    `json:"validation_status"`
	CreatedAt                 time.Time `json:"created_at"`
}

func summaryView(sm *domain.Summary) summaryJSON {
	return summaryJSON{
		SummaryID:                 sm.SummaryID.String(),
		Text:                      sm.Text,
		SourceTranscriptVersionID: sm.SourceTranscriptVersionID.String(),
		ValidationStatus:          sm.ValidationStatus,
		CreatedAt:                 sm.CreatedAt,
	}
}

type qualityReportJSON struct {
	QualityScore              *float64              `json:"quality_score"`
	ConfidenceThreshold       float64               `json:"confidence_threshold"`
	AverageConfidence         *float64              `json:"average_confidence"`
	LowConfidenceSegmentCount int                   `json:"low_confidence_segment_count"`
	CoverageGapSeconds        int                   `json:"coverage_gap_seconds"`
	TimestampGapCount         int                   `json:"timestamp_gap_count"`
	DiarizationWarningCount   int                   `json:"diarization_warning_count"`
	ConfidenceUnavailable     bool                  `json:"confidence_unavailable"`
	Issues                    []domain.QualityIssue `json:"issues"`
}

func qualityReportView(r *domain.QualityReport) qualityReportJSON {
	issues := r.Issues
	if issues == nil {
		issues = []domain.QualityIssue{}
	}
	return qualityReportJSON{
		QualityScore:              r.QualityScore,
		ConfidenceThreshold:       r.ConfidenceThreshold,
		AverageConfidence:         r.AverageConfidence,
		LowConfidenceSegmentCount: r.LowConfidenceSegmentCount,
		CoverageGapSeconds:        r.CoverageGapSeconds,
		TimestampGapCount:         r.TimestampGapCount,
		DiarizationWarningCount:   r.DiarizationWarningCount,
		ConfidenceUnavailable:     r.ConfidenceUnavailable,
		Issues:                    issues,
	}
}

type exportJSON struct {
	ExportID         string    `json:"export_id"`
	Format           string    `json:"format"`
	ValidationStatus string    `json:"validation_status"`
	DownloadURL      string    `json:"download_url"`
	CreatedAt        time.Time `json:"created_at"`
}

func (s *Server) exportDownloadToken(exportID uuid.UUID) string {
	mac := hmac.New(sha256.New, s.DownloadTokenSecret)
	_, _ = mac.Write([]byte(exportID.String()))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *Server) validExportDownloadToken(exportID uuid.UUID, token string) bool {
	if token == "" {
		return false
	}
	expected := s.exportDownloadToken(exportID)
	return hmac.Equal([]byte(token), []byte(expected))
}

func (s *Server) exportView(e *domain.ExportRecord) exportJSON {
	id := e.ExportID.String()
	return exportJSON{
		ExportID:         id,
		Format:           e.Format,
		ValidationStatus: e.ValidationStatus,
		DownloadURL:      "/api/v1/exports/" + id + "/download?token=" + s.exportDownloadToken(e.ExportID),
		CreatedAt:        e.CreatedAt,
	}
}

type auditEventJSON struct {
	AuditEventID string         `json:"audit_event_id"`
	ActorType    string         `json:"actor_type"`
	ActorID      string         `json:"actor_id"`
	EventType    string         `json:"event_type"`
	EventPayload map[string]any `json:"event_payload"`
	CreatedAt    time.Time      `json:"created_at"`
}

func auditEventView(e *domain.AuditEvent) auditEventJSON {
	payload := e.EventPayload
	if payload == nil {
		payload = map[string]any{}
	}
	return auditEventJSON{
		AuditEventID: e.AuditEventID.String(),
		ActorType:    e.ActorType,
		ActorID:      e.ActorID,
		EventType:    e.EventType,
		EventPayload: payload,
		CreatedAt:    e.CreatedAt,
	}
}
