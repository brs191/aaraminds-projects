package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/aaraminds/transcript-agent/internal/audit"
	"github.com/aaraminds/transcript-agent/internal/domain"
	"github.com/aaraminds/transcript-agent/internal/tools"
)

func decodeJSON(r *http.Request, v any) error {
	if r.Body == nil {
		return nil
	}
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(v); err != nil {
		if errors.Is(err, io.EOF) { // empty body is legal for {}-optional endpoints
			return nil
		}
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			return domain.E(domain.CodeRequestTooLarge, "request body exceeds the %d byte limit", mbe.Limit)
		}
		return domain.E(domain.CodeValidationError, "invalid JSON body: %v", err)
	}
	return nil
}

func parseUUID(raw, what string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, domain.E(domain.CodeValidationError, "invalid %s %q", what, raw)
	}
	return id, nil
}

func uploadExt(filename string) (string, bool) {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
	for _, s := range domain.SupportedUploadExtensions {
		if ext == s {
			return ext, true
		}
	}
	return ext, false
}

func uploadMime(ext string) string {
	switch ext {
	case "mp3":
		return "audio/mpeg"
	case "m4a":
		return "audio/mp4"
	case "wav":
		return "audio/wav"
	case "mp4":
		return "video/mp4"
	case "mov":
		return "video/quicktime"
	}
	return "application/octet-stream"
}

func sanitizeUploadFilename(filename string) string {
	base := filepath.Base(filename)
	var b strings.Builder
	for _, r := range base {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "upload"
	}
	return b.String()
}

func (s *Server) loadJob(r *http.Request) (*domain.Job, error) {
	id, err := parseUUID(r.PathValue("jobID"), "job_id")
	if err != nil {
		return nil, err
	}
	return s.Tools.Stores.Jobs.GetJob(r.Context(), id)
}

// handleUploadMedia implements POST /api/v1/uploads (PRD R1; frozen contract
// addition). The multipart "file" part streams straight into the object store
// under uploads/<uuid><ext> — no full buffering — and is recorded as a staged
// source_media artifact. Jobs then reference it via upload://<uuid>.
func (s *Server) handleUploadMedia(w http.ResponseWriter, r *http.Request) {
	ident := identityFrom(r.Context())
	r.Body = http.MaxBytesReader(w, r.Body, s.MaxUploadBytes)

	mr, err := r.MultipartReader()
	if err != nil {
		writeError(w, domain.E(domain.CodeValidationError, "multipart/form-data body required: %v", err))
		return
	}
	var (
		part     *multipart.Part
		filename string
	)
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			var mbe *http.MaxBytesError
			if errors.As(err, &mbe) {
				writeError(w, domain.E(domain.CodeRequestTooLarge,
					"upload exceeds the %d byte limit", s.MaxUploadBytes))
				return
			}
			writeError(w, domain.E(domain.CodeValidationError, "read multipart body: %v", err))
			return
		}
		if p.FormName() == "file" {
			part = p
			filename = sanitizeUploadFilename(p.FileName())
			break
		}
		_ = p.Close()
	}
	if part == nil {
		writeError(w, domain.E(domain.CodeValidationError, "multipart field %q is required", "file"))
		return
	}
	defer part.Close()

	ext, ok := uploadExt(filename)
	if !ok {
		writeError(w, domain.E(domain.CodeUnsupportedFormat,
			"this file type is not supported (got %q); supported: %s",
			ext, strings.Join(domain.SupportedUploadExtensions, ", ")))
		return
	}

	uploadID := uuid.New()
	key := "uploads/" + uploadID.String() + "." + ext
	uri, size, err := s.Objects.PutStream(r.Context(), key, part)
	if err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			writeError(w, domain.E(domain.CodeRequestTooLarge,
				"upload exceeds the %d byte limit", s.MaxUploadBytes))
			return
		}
		writeError(w, domain.E(domain.CodeArtifactWriteFailed, "store upload: %v", err))
		return
	}
	mime := uploadMime(ext)
	// The staged artifact's ID IS the upload URI's uuid: upload://<uuid>
	// resolves via the artifact store only (audit M5).
	art := &domain.MediaArtifact{
		ArtifactID:   uploadID,
		ArtifactType: domain.ArtifactSourceMedia,
		URI:          uri,
		MimeType:     mime,
		SizeBytes:    size,
		CreatedAt:    time.Now().UTC(),
	}
	if err := s.Tools.Stores.Artifacts.CreateArtifact(r.Context(), art); err != nil {
		writeError(w, err)
		return
	}
	uploadURI := tools.UploadURIScheme + uploadID.String()
	if err := s.Tools.Audit(r.Context(), nil, audit.ActorUser, ident.UserID, "upload.media_staged", map[string]any{
		"upload_uri":      uploadURI,
		"source_uri_hash": tools.URIHash(uri),
		"filename":        filename,
		"size_bytes":      size,
		"mime_type":       mime,
	}); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"upload_uri": uploadURI,
		"filename":   filename,
		"size_bytes": size,
		"mime_type":  mime,
	})
}

// --- jobs -----------------------------------------------------------------

func (s *Server) handleSubmitJob(w http.ResponseWriter, r *http.Request) {
	ident := identityFrom(r.Context())
	var in struct {
		SourceType        string `json:"source_type"`
		SourceURI         string `json:"source_uri"`
		Language          string `json:"language"`
		OwnershipAttested bool   `json:"ownership_attested"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	job, err := s.Tools.SubmitMediaJob(r.Context(), tools.SubmitMediaJobInput{
		SourceType:        in.SourceType,
		SourceURI:         in.SourceURI,
		Language:          in.Language,
		SubmittedBy:       ident.UserID,
		OwnershipAttested: in.OwnershipAttested,
	})
	if err != nil {
		writeError(w, err) // attestation missing -> 400, job NOT created
		return
	}
	s.Orch.Enqueue(job.JobID)
	// Re-read: in sync mode the pipeline has already advanced the job.
	fresh, err := s.Tools.Stores.Jobs.GetJob(r.Context(), job.JobID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, s.jobView(r.Context(), fresh))
}

func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	ident := identityFrom(r.Context())
	jobs, err := s.Tools.Stores.Jobs.ListJobs(r.Context()) // newest first
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]jobJSON, 0, len(jobs))
	for _, j := range jobs {
		if !canAccessJob(ident, j) {
			continue
		}
		out = append(out, s.jobView(r.Context(), j))
	}
	writeJSON(w, http.StatusOK, map[string]any{"jobs": out})
}

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	job, err := s.loadAuthorizedJob(r)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.jobView(r.Context(), job))
}

func (s *Server) handleCaptionDecision(w http.ResponseWriter, r *http.Request) {
	ident := identityFrom(r.Context())
	job, err := s.loadAuthorizedJob(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var in struct {
		ReuseCaptions bool `json:"reuse_captions"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	if err := s.Orch.ResumeAfterCaptionDecision(r.Context(), job, in.ReuseCaptions, ident.UserID); err != nil {
		writeError(w, err)
		return
	}
	fresh, err := s.Tools.Stores.Jobs.GetJob(r.Context(), job.JobID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.jobView(r.Context(), fresh))
}

func (s *Server) handleReplaceMedia(w http.ResponseWriter, r *http.Request) {
	ident := identityFrom(r.Context())
	job, err := s.loadAuthorizedJob(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var in struct {
		SourceType        string `json:"source_type"`
		SourceURI         string `json:"source_uri"`
		OwnershipAttested bool   `json:"ownership_attested"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	updated, err := s.Tools.ReplaceJobMedia(r.Context(), job, tools.ReplaceJobMediaInput{
		SourceType:        in.SourceType,
		SourceURI:         in.SourceURI,
		ReplacedBy:        ident.UserID,
		ReplacedByRole:    ident.Role,
		OwnershipAttested: in.OwnershipAttested,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	s.Orch.Enqueue(updated.JobID)
	fresh, err := s.Tools.Stores.Jobs.GetJob(r.Context(), updated.JobID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.jobView(r.Context(), fresh))
}

func (s *Server) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	ident := identityFrom(r.Context())
	job, err := s.loadAuthorizedJob(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var in struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	updated, err := s.Tools.CancelJob(r.Context(), job, ident.UserID, ident.Role, in.Reason)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.jobView(r.Context(), updated))
}

// --- transcripts / review --------------------------------------------------

func (s *Server) handleListTranscripts(w http.ResponseWriter, r *http.Request) {
	job, err := s.loadAuthorizedJob(r)
	if err != nil {
		writeError(w, err)
		return
	}
	versions, err := s.Tools.Stores.Transcripts.ListVersions(r.Context(), job.JobID)
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]versionJSON, 0, len(versions))
	for _, v := range versions {
		out = append(out, versionView(v))
	}
	writeJSON(w, http.StatusOK, map[string]any{"versions": out})
}

func (s *Server) handleListSegments(w http.ResponseWriter, r *http.Request) {
	ident := identityFrom(r.Context())
	version, err := s.requireVersionAccess(r.Context(), ident, r.PathValue("versionID"))
	if err != nil {
		writeError(w, err)
		return
	}
	segs, err := s.Tools.Stores.Transcripts.ListSegments(r.Context(), version.TranscriptVersionID)
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]segmentJSON, 0, len(segs))
	for _, sg := range segs {
		out = append(out, segmentView(sg))
	}
	writeJSON(w, http.StatusOK, map[string]any{"segments": out})
}

// handleCreateReview creates a mutable `reviewed` version copied from the
// latest clean (raw as fallback). Reviewer/admin only (PRD 16.2).
func (s *Server) handleCreateReview(w http.ResponseWriter, r *http.Request) {
	ident := identityFrom(r.Context())
	if err := requireRole(ident, domain.RoleReviewer, domain.RoleAdmin); err != nil {
		writeError(w, err)
		return
	}
	job, err := s.loadJob(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if job.Status != domain.StatusInReview {
		writeError(w, domain.E(domain.CodeJobNotInActionableState,
			"review versions can be created only while the job is in_review; job is %s", job.Status))
		return
	}
	source, err := s.Tools.Stores.Transcripts.LatestVersion(r.Context(), job.JobID, domain.VersionClean)
	if err == nil && source == nil {
		source, err = s.Tools.Stores.Transcripts.LatestVersion(r.Context(), job.JobID, domain.VersionRaw)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	if source == nil {
		writeError(w, domain.E(domain.CodeTranscriptNotFound, "job %s has no transcript version to review", job.JobID))
		return
	}
	reviewed, err := s.Tools.CloneToVersion(r.Context(), job, source, domain.VersionReviewed, ident.UserID, false)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, versionView(reviewed))
}

func (s *Server) handleEditSegment(w http.ResponseWriter, r *http.Request) {
	ident := identityFrom(r.Context())
	if err := requireRole(ident, domain.RoleReviewer, domain.RoleAdmin); err != nil {
		writeError(w, err)
		return
	}
	versionID, err := parseUUID(r.PathValue("versionID"), "transcript_version_id")
	if err != nil {
		writeError(w, err)
		return
	}
	segmentID, err := parseUUID(r.PathValue("segmentID"), "segment_id")
	if err != nil {
		writeError(w, err)
		return
	}
	var in struct {
		Text         *string `json:"text"`
		SpeakerLabel *string `json:"speaker_label"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	if in.Text == nil && in.SpeakerLabel == nil {
		writeError(w, domain.E(domain.CodeValidationError, "provide text and/or speaker_label"))
		return
	}
	seg, err := s.Tools.EditSegment(r.Context(), versionID, segmentID, in.Text, in.SpeakerLabel, ident.UserID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, segmentView(seg))
}

func (s *Server) handleApprove(w http.ResponseWriter, r *http.Request) {
	ident := identityFrom(r.Context())
	if err := requireRole(ident, domain.RoleReviewer, domain.RoleAdmin); err != nil {
		writeError(w, err)
		return
	}
	job, err := s.loadJob(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var in struct {
		ReviewedTranscriptVersionID string `json:"reviewed_transcript_version_id"`
		ApprovalNote                string `json:"approval_note"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	reviewedID, err := parseUUID(in.ReviewedTranscriptVersionID, "reviewed_transcript_version_id")
	if err != nil {
		writeError(w, err)
		return
	}
	approval, _, err := s.Tools.ApproveTranscript(r.Context(), job, reviewedID, ident.UserID, in.ApprovalNote)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, approvalView(approval))
}

// handleReopen implements post-approval correction (PRD 11.4): approved or
// exported jobs return to in_review with a fresh reviewed version copied from
// the approved one. The prior approval is superseded on re-approval.
func (s *Server) handleReopen(w http.ResponseWriter, r *http.Request) {
	ident := identityFrom(r.Context())
	if err := requireRole(ident, domain.RoleReviewer, domain.RoleAdmin); err != nil {
		writeError(w, err)
		return
	}
	job, err := s.loadJob(r)
	if err != nil {
		writeError(w, err)
		return
	}
	// Atomic CAS approved/exported -> in_review plus the reviewed clone, in
	// one store operation (audit M7).
	updated, _, err := s.Tools.ReopenJob(r.Context(), job, ident.UserID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.jobView(r.Context(), updated))
}

// --- summary ----------------------------------------------------------------

func (s *Server) handleGenerateSummary(w http.ResponseWriter, r *http.Request) {
	ident := identityFrom(r.Context())
	job, err := s.loadAuthorizedJob(r)
	if err != nil {
		writeError(w, err)
		return
	}
	cfg, err := s.Tools.Config(r.Context(), job)
	if err != nil {
		writeError(w, err)
		return
	}
	// Prefer the most authoritative version: approved > reviewed > clean > raw
	// (PRD R10: summaries can come from draft but should be reconfirmed after
	// approval; source version is always recorded).
	var source *domain.TranscriptVersion
	for _, vt := range []string{domain.VersionApproved, domain.VersionReviewed, domain.VersionClean, domain.VersionRaw} {
		source, err = s.Tools.Stores.Transcripts.LatestVersion(r.Context(), job.JobID, vt)
		if err != nil {
			writeError(w, err)
			return
		}
		if source != nil {
			break
		}
	}
	if source == nil {
		writeError(w, domain.E(domain.CodeTranscriptNotFound, "job %s has no transcript version to summarize", job.JobID))
		return
	}
	summary, err := s.Tools.GenerateSummary(r.Context(), job, source, cfg, ident.UserID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, summaryView(summary))
}

func (s *Server) handleGetSummary(w http.ResponseWriter, r *http.Request) {
	job, err := s.loadAuthorizedJob(r)
	if err != nil {
		writeError(w, err)
		return
	}
	summary, err := s.Tools.Stores.Summaries.LatestSummaryByJob(r.Context(), job.JobID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summaryView(summary))
}

func (s *Server) handleEditSummary(w http.ResponseWriter, r *http.Request) {
	ident := identityFrom(r.Context())
	summaryID, err := parseUUID(r.PathValue("summaryID"), "summary_id")
	if err != nil {
		writeError(w, err)
		return
	}
	var in struct {
		Text string `json:"text"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	if strings.TrimSpace(in.Text) == "" {
		writeError(w, domain.E(domain.CodeValidationError, "text is required"))
		return
	}
	summary, err := s.Tools.Stores.Summaries.GetSummary(r.Context(), summaryID)
	if err != nil {
		writeError(w, err)
		return
	}
	job, err := s.Tools.Stores.Jobs.GetJob(r.Context(), summary.JobID)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := requireJobAccess(ident, job); err != nil {
		writeError(w, err)
		return
	}
	now := time.Now().UTC()
	summary.Text = in.Text
	summary.UpdatedAt = &now
	if err := s.Tools.Stores.Summaries.UpdateSummary(r.Context(), summary); err != nil {
		writeError(w, err)
		return
	}
	if err := s.Tools.Audit(r.Context(), &summary.JobID, audit.ActorUser, ident.UserID, "summary.edited",
		map[string]any{"summary_id": summary.SummaryID.String()}); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summaryView(summary))
}

// --- quality report ---------------------------------------------------------

func (s *Server) handleQualityReport(w http.ResponseWriter, r *http.Request) {
	job, err := s.loadAuthorizedJob(r)
	if err != nil {
		writeError(w, err)
		return
	}
	report, err := s.Tools.Stores.Quality.LatestReportByJob(r.Context(), job.JobID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, qualityReportView(report))
}

// --- exports ------------------------------------------------------------------

func (s *Server) handleCreateExports(w http.ResponseWriter, r *http.Request) {
	ident := identityFrom(r.Context())
	job, err := s.loadAuthorizedJob(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var in struct {
		Formats []string `json:"formats"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	// Frozen contract + PRD R8: exports only from an approved transcript.
	if job.Status != domain.StatusApproved && job.Status != domain.StatusExported {
		writeError(w, domain.E(domain.CodeApprovedTranscriptRequired,
			"exports require an approved transcript; job is %s", job.Status))
		return
	}
	approval, err := s.Tools.Stores.Approvals.CurrentApproval(r.Context(), job.JobID)
	if err != nil {
		writeError(w, err)
		return
	}
	if approval == nil {
		writeError(w, domain.E(domain.CodeApprovedTranscriptRequired, "no current approval for job %s", job.JobID))
		return
	}
	approved, err := s.Tools.Stores.Transcripts.GetVersion(r.Context(), approval.ApprovedTranscriptVersionID)
	if err != nil {
		writeError(w, err)
		return
	}
	records, err := s.Tools.ExportTranscript(r.Context(), job, approved, in.Formats, ident.UserID)
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]exportJSON, 0, len(records))
	for _, rec := range records {
		out = append(out, s.exportView(rec))
	}
	writeJSON(w, http.StatusCreated, map[string]any{"exports": out})
}

func (s *Server) handleListExports(w http.ResponseWriter, r *http.Request) {
	job, err := s.loadAuthorizedJob(r)
	if err != nil {
		writeError(w, err)
		return
	}
	records, err := s.Tools.Stores.Artifacts.ListExportsByJob(r.Context(), job.JobID)
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]exportJSON, 0, len(records))
	for _, rec := range records {
		out = append(out, s.exportView(rec))
	}
	writeJSON(w, http.StatusOK, map[string]any{"exports": out})
}

// requireTokenOrAuth enforces the token-or-auth rule for signed-link
// endpoints (audit H2): a valid signed token authorizes the request; without
// a token, valid auth headers are required. Invalid/expired tokens are 401
// TOKEN_INVALID. Returns the identity (zero-valued when a token authorized)
// and whether the caller was token-authorized.
func (s *Server) requireTokenOrAuth(r *http.Request, kind, id string) (Identity, bool, error) {
	token := r.URL.Query().Get("token")
	if token != "" {
		if !s.validToken(kind, id, token) {
			return Identity{}, false, domain.E(domain.CodeTokenInvalid, "signed link token is invalid or expired")
		}
		return Identity{}, true, nil
	}
	ident := identityFrom(r.Context())
	if ident.UserID == "" {
		return Identity{}, false, domain.E(domain.CodeUnauthenticated,
			"a signed ?token= or authentication headers are required")
	}
	return ident, false, nil
}

// handleCreateSignedLink implements POST /api/v1/signed-links (frozen
// contract): any authenticated role mints a short-lived signed URL for an
// export download or a job's audio stream.
func (s *Server) handleCreateSignedLink(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Kind string `json:"kind"`
		ID   string `json:"id"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	switch in.Kind {
	case signedKindExport:
		exportID, err := parseUUID(in.ID, "export_id")
		if err != nil {
			writeError(w, err)
			return
		}
		if _, err := s.Tools.Stores.Artifacts.GetExport(r.Context(), exportID); err != nil {
			writeError(w, err)
			return
		}
	case signedKindAudio:
		jobID, err := parseUUID(in.ID, "job_id")
		if err != nil {
			writeError(w, err)
			return
		}
		if _, err := s.Tools.Stores.Jobs.GetJob(r.Context(), jobID); err != nil {
			writeError(w, err)
			return
		}
		// Mirror GET /jobs/{id}/audio: mint only when audio actually exists,
		// so the UI's "no audio (caption-reuse)" path triggers at mint time
		// instead of surfacing as a broken player.
		art, err := s.resolveAudioArtifact(r.Context(), jobID)
		if err != nil {
			writeError(w, err)
			return
		}
		if art == nil {
			writeError(w, domain.E(domain.CodeAudioNotAvailable,
				"job %s has no audio artifact (caption-reuse jobs skip audio extraction)", jobID))
			return
		}
	default:
		writeError(w, domain.E(domain.CodeValidationError,
			"kind must be %q or %q", signedKindExport, signedKindAudio))
		return
	}
	url, expiresAt := s.mintSignedURL(in.Kind, in.ID)
	writeJSON(w, http.StatusCreated, map[string]any{
		"url":        url,
		"expires_at": expiresAt.Format(time.RFC3339),
	})
}

// handleDownloadExport streams export bytes. Requires EITHER a valid signed
// ?token= OR auth headers (audit H2 — previously fully open).
func (s *Server) handleDownloadExport(w http.ResponseWriter, r *http.Request) {
	exportID, err := parseUUID(r.PathValue("exportID"), "export_id")
	if err != nil {
		writeError(w, err)
		return
	}
	if _, _, err := s.requireTokenOrAuth(r, signedKindExport, exportID.String()); err != nil {
		writeError(w, err)
		return
	}
	rec, err := s.Tools.Stores.Artifacts.GetExport(r.Context(), exportID)
	if err != nil {
		writeError(w, err)
		return
	}
	data, err := s.Objects.Get(r.Context(), rec.ArtifactURI)
	if err != nil {
		writeError(w, err)
		return
	}
	filename := fmt.Sprintf("transcript-%s.%s", strings.SplitN(rec.JobID.String(), "-", 2)[0], rec.Format)
	w.Header().Set("Content-Type", tools.ExportMime(rec.Format))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// audioMimes are the source_media types the audio endpoint may stream when
// no audio_extract artifact exists (PRD R7 playback).
var audioMimes = map[string]bool{
	"audio/mpeg": true, "audio/mp4": true,
	"audio/wav": true, "audio/x-wav": true,
}

// pickAudioArtifact returns the newest non-superseded artifact of the slice.
func pickAudioArtifact(arts []*domain.MediaArtifact) *domain.MediaArtifact {
	for i := len(arts) - 1; i >= 0; i-- {
		if !arts[i].Superseded {
			return arts[i]
		}
	}
	return nil
}

// resolveAudioArtifact returns the artifact GET /jobs/{id}/audio would stream
// for jobID — the newest audio_extract, else the source_media when it is
// audio — or nil when the job has no playable audio (caption-reuse path).
func (s *Server) resolveAudioArtifact(ctx context.Context, jobID uuid.UUID) (*domain.MediaArtifact, error) {
	extracts, err := s.Tools.Stores.Artifacts.ListArtifactsByJob(ctx, jobID, domain.ArtifactAudioExtract)
	if err != nil {
		return nil, err
	}
	if art := pickAudioArtifact(extracts); art != nil {
		return art, nil
	}
	sources, err := s.Tools.Stores.Artifacts.ListArtifactsByJob(ctx, jobID, domain.ArtifactSourceMedia)
	if err != nil {
		return nil, err
	}
	if src := pickAudioArtifact(sources); src != nil && audioMimes[src.MimeType] {
		return src, nil
	}
	return nil, nil
}

// handleJobAudio implements GET /api/v1/jobs/{jobID}/audio (frozen contract,
// enables PRD R7 playback). Streams the job's audio_extract artifact when
// present, else the source_media artifact when it is audio; Range requests
// are honored via http.ServeContent.
func (s *Server) handleJobAudio(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseUUID(r.PathValue("jobID"), "job_id")
	if err != nil {
		writeError(w, err)
		return
	}
	ident, byToken, err := s.requireTokenOrAuth(r, signedKindAudio, jobID.String())
	if err != nil {
		writeError(w, err)
		return
	}
	job, err := s.Tools.Stores.Jobs.GetJob(r.Context(), jobID)
	if err != nil {
		writeError(w, err)
		return
	}
	if !byToken {
		if err := requireJobAccess(ident, job); err != nil {
			writeError(w, err)
			return
		}
	}
	art, err := s.resolveAudioArtifact(r.Context(), jobID)
	if err != nil {
		writeError(w, err)
		return
	}
	if art == nil {
		writeError(w, domain.E(domain.CodeAudioNotAvailable,
			"job %s has no audio artifact (caption-reuse jobs skip audio extraction)", jobID))
		return
	}
	rc, _, modTime, err := s.Objects.Open(r.Context(), art.URI)
	if err != nil {
		writeError(w, err)
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Type", art.MimeType)
	http.ServeContent(w, r, "", modTime, rc)
}

// --- audit ---------------------------------------------------------------------

func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	job, err := s.loadAuthorizedJob(r)
	if err != nil {
		writeError(w, err)
		return
	}
	events, err := s.Tools.Stores.Audit.ListByJob(r.Context(), job.JobID)
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]auditEventJSON, 0, len(events))
	for _, e := range events {
		out = append(out, auditEventView(e))
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": out})
}
