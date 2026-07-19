package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aaraminds/transcript-agent/internal/domain"
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
		domain.CodeUnsupportedFormat, domain.CodeLanguageUnsupported,
		domain.CodeFeedURLInvalid, domain.CodeFeedFetchFailed:
		return http.StatusBadRequest
	case domain.CodeUnauthenticated, domain.CodeTokenInvalid:
		return http.StatusUnauthorized
	case domain.CodeUserNotAuthorized:
		return http.StatusForbidden
	case domain.CodeJobNotFound, domain.CodeTranscriptNotFound, domain.CodeSegmentNotFound,
		domain.CodeSummaryNotFound, domain.CodeExportNotFound, domain.CodeQualityReportNotFound,
		domain.CodeMediaNotFound, domain.CodeAudioNotAvailable,
		domain.CodeFeedNotFound, domain.CodeEpisodeNotFound:
		return http.StatusNotFound
	case domain.CodeTranscriptVersionImmutable, domain.CodeTranscriptVersionNotReviewable,
		domain.CodeApprovedTranscriptRequired, domain.CodeJobNotInActionableState,
		domain.CodeJobAlreadyTerminal, domain.CodeInvalidStateTransition,
		domain.CodeDisabledInMVP, domain.CodeOpenCriticalIssues,
		domain.CodeStatusConflict,
		domain.CodeFeedAlreadyExists, domain.CodeEpisodeAlreadyTranscribed:
		return http.StatusConflict
	case domain.CodeRequestTooLarge:
		return http.StatusRequestEntityTooLarge
	case domain.CodeAuditUnavailable:
		// Audit failure pauses high-risk actions (PRD 19 audit row).
		return http.StatusServiceUnavailable
	case domain.CodeAuditWriteFailed:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// internalErrorLogger is implemented by the middleware's statusRecorder so
// writeError can log the full internal error with the request ID.
type internalErrorLogger interface {
	logInternalError(err error)
}

// writeError renders the frozen error envelope. Unknown/internal errors are
// sanitized: the full error is logged server-side with the request ID and the
// client receives a generic INTERNAL_ERROR — no pgx/OS strings leak out.
func writeError(w http.ResponseWriter, err error) {
	de := domain.AsError(err)
	status := httpStatusFor(de.Code)
	code, msg := de.Code, de.Message
	if status == http.StatusInternalServerError {
		if lg, ok := w.(internalErrorLogger); ok {
			lg.logInternalError(err)
		} else {
			slog.Default().Error("internal error", "error", err)
		}
		code, msg = domain.CodeInternalError, "internal error"
	}
	writeJSON(w, status, errorBody{Error: domain.ErrorInfo{Code: code, Message: msg}})
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
	// Additive library-mode fields (nil/false on regular jobs).
	LibraryMode bool      `json:"library_mode"`
	SourceBasis *string   `json:"source_basis"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
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
		LibraryMode:       job.LibraryMode,
		CreatedAt:         job.CreatedAt,
		UpdatedAt:         job.UpdatedAt,
	}
	if job.SourceBasis != "" {
		basis := job.SourceBasis
		out.SourceBasis = &basis
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
	ValidationStatus          string    `json:"validation_status"` // passed | needs_review | failed
	ValidationNotes           *string   `json:"validation_notes"`  // null when no notes
	CreatedAt                 time.Time `json:"created_at"`
}

func summaryView(sm *domain.Summary) summaryJSON {
	out := summaryJSON{
		SummaryID:                 sm.SummaryID.String(),
		Text:                      sm.Text,
		SourceTranscriptVersionID: sm.SourceTranscriptVersionID.String(),
		ValidationStatus:          sm.ValidationStatus,
		CreatedAt:                 sm.CreatedAt,
	}
	if sm.ValidationNotes != "" {
		notes := sm.ValidationNotes
		out.ValidationNotes = &notes
	}
	return out
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
	ExportID                    string    `json:"export_id"`
	Format                      string    `json:"format"`
	ValidationStatus            string    `json:"validation_status"`
	ApprovedTranscriptVersionID string    `json:"approved_transcript_version_id"`
	Superseded                  bool      `json:"superseded"`
	DownloadURL                 string    `json:"download_url"`
	CreatedAt                   time.Time `json:"created_at"`
}

// Signed-link kinds (frozen contract for POST /api/v1/signed-links).
const (
	signedKindExport = "export"
	signedKindAudio  = "audio"
)

// signedLinkTTL is the frozen 15-minute validity window for signed links.
const signedLinkTTL = 15 * time.Minute

// signToken builds an expiring token: HMAC-SHA256 over kind|id|expiry-unix
// keyed with SIGNING_SECRET, encoded as "<expiry-unix>.<hex mac>".
func (s *Server) signToken(kind, id string, expiry int64) string {
	mac := hmac.New(sha256.New, s.SigningSecret)
	fmt.Fprintf(mac, "%s|%s|%d", kind, id, expiry)
	return strconv.FormatInt(expiry, 10) + "." + hex.EncodeToString(mac.Sum(nil))
}

// validToken verifies an expiring signed token with a constant-time compare.
func (s *Server) validToken(kind, id, token string) bool {
	expStr, _, ok := strings.Cut(token, ".")
	if !ok {
		return false
	}
	expiry, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || time.Now().Unix() > expiry {
		return false
	}
	expected := s.signToken(kind, id, expiry)
	return hmac.Equal([]byte(token), []byte(expected))
}

// mintSignedURL builds the relative signed URL plus its expiry for a kind/id.
func (s *Server) mintSignedURL(kind, id string) (string, time.Time) {
	expiresAt := time.Now().Add(signedLinkTTL).UTC().Truncate(time.Second)
	token := s.signToken(kind, id, expiresAt.Unix())
	switch kind {
	case signedKindAudio:
		return "/api/v1/jobs/" + id + "/audio?token=" + token, expiresAt
	default:
		return "/api/v1/exports/" + id + "/download?token=" + token, expiresAt
	}
}

func (s *Server) exportView(e *domain.ExportRecord) exportJSON {
	id := e.ExportID.String()
	url, _ := s.mintSignedURL(signedKindExport, id)
	return exportJSON{
		ExportID:                    id,
		Format:                      e.Format,
		ValidationStatus:            e.ValidationStatus,
		ApprovedTranscriptVersionID: e.ApprovedTranscriptVersionID.String(),
		Superseded:                  e.Superseded,
		DownloadURL:                 url,
		CreatedAt:                   e.CreatedAt,
	}
}

// approvalListJSON is the frozen shape of GET /jobs/{jobID}/approvals items.
type approvalListJSON struct {
	ApprovalID                  string    `json:"approval_id"`
	ApprovedTranscriptVersionID string    `json:"approved_transcript_version_id"`
	ApprovedBy                  string    `json:"approved_by"`
	ApprovedAt                  time.Time `json:"approved_at"`
	ApprovalNote                string    `json:"approval_note"`
	SupersededByApprovalID      *string   `json:"superseded_by_approval_id"`
}

func approvalListView(a *domain.Approval) approvalListJSON {
	out := approvalListJSON{
		ApprovalID:                  a.ApprovalID.String(),
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
