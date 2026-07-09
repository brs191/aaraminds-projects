package app_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/aaraminds/transcript-agent/internal/app"
	"github.com/aaraminds/transcript-agent/internal/domain"
	"github.com/aaraminds/transcript-agent/internal/objectstore"
	capmock "github.com/aaraminds/transcript-agent/internal/providers/captions/mock"
	llmmock "github.com/aaraminds/transcript-agent/internal/providers/llm/mock"
	"github.com/aaraminds/transcript-agent/internal/providers/media"
	sttmock "github.com/aaraminds/transcript-agent/internal/providers/stt/mock"
	"github.com/aaraminds/transcript-agent/internal/store/memory"
)

// --- harness ----------------------------------------------------------------

type env struct {
	t   *testing.T
	app *app.App
	srv *httptest.Server
}

var (
	producer  = map[string]string{"X-User-Id": "alice", "X-User-Role": "producer"}
	producer2 = map[string]string{"X-User-Id": "mallory", "X-User-Role": "producer"}
	reviewer  = map[string]string{"X-User-Id": "bob", "X-User-Role": "reviewer"}
	admin     = map[string]string{"X-User-Id": "root", "X-User-Role": "admin"}
)

func newEnv(t *testing.T, defaults *domain.JobConfig) *env {
	t.Helper()
	objects, err := objectstore.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := app.New(app.Options{
		Stores:         memory.New().Stores(),
		Objects:        objects,
		STT:            sttmock.New(),
		LLM:            llmmock.New(),
		Media:          media.NewStub(),
		Captions:       capmock.New(),
		STTName:        "mock",
		ConfigDefaults: defaults,
		CORSOrigin:     "http://localhost:5173",
		Sync:           true, // drive jobs inline for deterministic tests
		Backoff:        time.Millisecond,
	})
	srv := httptest.NewServer(a.API.Handler())
	t.Cleanup(srv.Close)
	return &env{t: t, app: a, srv: srv}
}

// do performs a request and decodes the JSON response into out (if non-nil).
func (e *env) do(method, path string, headers map[string]string, body any, out any) int {
	e.t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			e.t.Fatal(err)
		}
		rdr = bytes.NewReader(raw)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, e.srv.URL+path, rdr)
	if err != nil {
		e.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer res.Body.Close()
	if out != nil {
		if err := json.NewDecoder(res.Body).Decode(out); err != nil {
			e.t.Fatalf("%s %s: decode response: %v", method, path, err)
		}
	}
	return res.StatusCode
}

func (e *env) must(status, want int, what string) {
	e.t.Helper()
	if status != want {
		e.t.Fatalf("%s: status %d, want %d", what, status, want)
	}
}

// --- contract-shaped response types ------------------------------------------

type jobResp struct {
	JobID             string `json:"job_id"`
	SourceType        string `json:"source_type"`
	SourceURI         string `json:"source_uri"`
	Status            string `json:"status"`
	SubmittedBy       string `json:"submitted_by"`
	OwnershipAttested bool   `json:"ownership_attested"`
	Language          string `json:"language"`
	JobConfig         *struct {
		ConfidenceThreshold float64 `json:"confidence_threshold"`
		EnableDiarization   bool    `json:"enable_diarization"`
		Language            string  `json:"language"`
		StylePolicyID       string  `json:"style_policy_id"`
		SummaryMaxWords     int     `json:"summary_max_words"`
		SummaryStyle        string  `json:"summary_style"`
		STTProvider         string  `json:"stt_provider"`
	} `json:"job_config"`
	DurationSeconds int    `json:"duration_seconds"`
	ActionRequired  string `json:"action_required"`
	LastError       *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"last_error"`
}

type versionResp struct {
	TranscriptVersionID string  `json:"transcript_version_id"`
	VersionType         string  `json:"version_type"`
	SourceVersionID     *string `json:"source_version_id"`
	CreatedBy           string  `json:"created_by"`
	IsImmutable         bool    `json:"is_immutable"`
}

type segmentResp struct {
	SegmentID    string          `json:"segment_id"`
	StartMS      int             `json:"start_ms"`
	EndMS        int             `json:"end_ms"`
	SpeakerLabel string          `json:"speaker_label"`
	Text         string          `json:"text"`
	Confidence   *float64        `json:"confidence"`
	Flags        map[string]bool `json:"flags"`
}

type approvalResp struct {
	ApprovalID                  string  `json:"approval_id"`
	ApprovedTranscriptVersionID string  `json:"approved_transcript_version_id"`
	SupersededByApprovalID      *string `json:"superseded_by_approval_id"`
}

type summaryResp struct {
	SummaryID                 string `json:"summary_id"`
	Text                      string `json:"text"`
	SourceTranscriptVersionID string `json:"source_transcript_version_id"`
	ValidationStatus          string `json:"validation_status"`
}

type qualityResp struct {
	QualityScore              *float64 `json:"quality_score"`
	ConfidenceThreshold       float64  `json:"confidence_threshold"`
	AverageConfidence         *float64 `json:"average_confidence"`
	LowConfidenceSegmentCount int      `json:"low_confidence_segment_count"`
	CoverageGapSeconds        int      `json:"coverage_gap_seconds"`
	TimestampGapCount         int      `json:"timestamp_gap_count"`
	DiarizationWarningCount   int      `json:"diarization_warning_count"`
	ConfidenceUnavailable     bool     `json:"confidence_unavailable"`
	Issues                    []struct {
		IssueType string `json:"issue_type"`
		Severity  string `json:"severity"`
	} `json:"issues"`
}

type exportResp struct {
	ExportID         string `json:"export_id"`
	Format           string `json:"format"`
	ValidationStatus string `json:"validation_status"`
	DownloadURL      string `json:"download_url"`
}

type errResp struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// --- helpers -------------------------------------------------------------------

func submitJob(e *env, sourceType, sourceURI string) jobResp {
	e.t.Helper()
	var job jobResp
	status := e.do("POST", "/api/v1/jobs", producer, map[string]any{
		"source_type":        sourceType,
		"source_uri":         sourceURI,
		"language":           "en",
		"ownership_attested": true,
	}, &job)
	e.must(status, http.StatusCreated, "submit job")
	return job
}

func listVersions(e *env, jobID string) []versionResp {
	e.t.Helper()
	var out struct {
		Versions []versionResp `json:"versions"`
	}
	e.must(e.do("GET", "/api/v1/jobs/"+jobID+"/transcripts", producer, nil, &out),
		http.StatusOK, "list transcripts")
	return out.Versions
}

func listSegments(e *env, versionID string) []segmentResp {
	e.t.Helper()
	var out struct {
		Segments []segmentResp `json:"segments"`
	}
	e.must(e.do("GET", "/api/v1/transcripts/"+versionID+"/segments", producer, nil, &out),
		http.StatusOK, "list segments")
	return out.Segments
}

func versionOfType(t *testing.T, versions []versionResp, vt string) versionResp {
	t.Helper()
	for i := len(versions) - 1; i >= 0; i-- {
		if versions[i].VersionType == vt {
			return versions[i]
		}
	}
	t.Fatalf("no %s version in %v", vt, versions)
	return versionResp{}
}

// runToApproved drives a fresh upload job through review and approval.
func runToApproved(e *env) (job jobResp, reviewedID, approvedID string) {
	e.t.Helper()
	job = submitJob(e, "upload", fmt.Sprintf("uploads/%s.mp3", uuid.NewString()))
	if job.Status != "in_review" {
		e.t.Fatalf("job status %s, want in_review", job.Status)
	}
	var reviewed versionResp
	e.must(e.do("POST", "/api/v1/jobs/"+job.JobID+"/review", reviewer, map[string]any{}, &reviewed),
		http.StatusCreated, "create review version")
	var approval approvalResp
	e.must(e.do("POST", "/api/v1/jobs/"+job.JobID+"/approve", reviewer, map[string]any{
		"reviewed_transcript_version_id": reviewed.TranscriptVersionID,
		"approval_note":                  "looks good",
	}, &approval), http.StatusCreated, "approve")
	return job, reviewed.TranscriptVersionID, approval.ApprovedTranscriptVersionID
}

// --- tests -----------------------------------------------------------------------

// Full pipeline: submit -> in_review -> review -> edit -> approve -> export all
// four formats -> validators pass -> audit trail complete (spec test 2).
func TestFullPipelineE2E(t *testing.T) {
	e := newEnv(t, nil)
	job := submitJob(e, "upload", "uploads/episode-one.mp3")

	if job.Status != "in_review" {
		t.Fatalf("status %s, want in_review", job.Status)
	}
	if job.JobConfig == nil {
		t.Fatal("job_config missing after validation snapshot")
	}
	if job.JobConfig.ConfidenceThreshold != 0.80 || job.JobConfig.SummaryMaxWords != 150 ||
		job.JobConfig.StylePolicyID != "default-clean-v1" || job.JobConfig.STTProvider != "mock" {
		t.Fatalf("job_config defaults wrong: %+v", *job.JobConfig)
	}
	if job.DurationSeconds == 0 {
		t.Fatal("duration_seconds not populated from metadata")
	}

	versions := listVersions(e, job.JobID)
	if len(versions) != 2 {
		t.Fatalf("want raw+clean versions, got %d", len(versions))
	}
	raw := versionOfType(t, versions, "raw")
	clean := versionOfType(t, versions, "clean")

	rawSegs := listSegments(e, raw.TranscriptVersionID)
	if len(rawSegs) == 0 {
		t.Fatal("raw transcript has no segments")
	}
	lowFlagged := 0
	speakers := map[string]bool{}
	for _, sg := range rawSegs {
		if sg.Confidence == nil {
			t.Fatal("STT-path segment missing confidence")
		}
		speakers[sg.SpeakerLabel] = true
		if sg.Flags["low_confidence"] {
			lowFlagged++
			if *sg.Confidence >= 0.80 {
				t.Errorf("segment flagged low_confidence with confidence %.2f", *sg.Confidence)
			}
		}
	}
	if lowFlagged == 0 {
		t.Fatal("expected some low-confidence flagged segments (PRD R5)")
	}
	if len(speakers) < 2 {
		t.Fatalf("expected diarized 2-speaker transcript, got %v", speakers)
	}

	// Clean version preserves timestamps/speakers and removes fillers.
	cleanSegs := listSegments(e, clean.TranscriptVersionID)
	if len(cleanSegs) != len(rawSegs) {
		t.Fatalf("clean segment count %d != raw %d", len(cleanSegs), len(rawSegs))
	}
	for i := range cleanSegs {
		if cleanSegs[i].StartMS != rawSegs[i].StartMS || cleanSegs[i].EndMS != rawSegs[i].EndMS {
			t.Fatal("cleanup changed timestamps")
		}
		if cleanSegs[i].SpeakerLabel != rawSegs[i].SpeakerLabel {
			t.Fatal("cleanup changed speaker labels")
		}
	}

	// Quality report exists for the STT path.
	var qr qualityResp
	e.must(e.do("GET", "/api/v1/jobs/"+job.JobID+"/quality-report", producer, nil, &qr),
		http.StatusOK, "quality report")
	if qr.ConfidenceUnavailable {
		t.Fatal("STT path must not be confidence_unavailable")
	}
	if qr.AverageConfidence == nil || qr.LowConfidenceSegmentCount == 0 {
		t.Fatalf("quality metrics missing: %+v", qr)
	}

	// Review + edit a segment.
	var reviewed versionResp
	e.must(e.do("POST", "/api/v1/jobs/"+job.JobID+"/review", reviewer, map[string]any{}, &reviewed),
		http.StatusCreated, "create review")
	if reviewed.VersionType != "reviewed" || reviewed.IsImmutable {
		t.Fatalf("reviewed version wrong: %+v", reviewed)
	}
	if reviewed.SourceVersionID == nil || *reviewed.SourceVersionID != clean.TranscriptVersionID {
		t.Fatal("reviewed version must be copied from latest clean")
	}
	revSegs := listSegments(e, reviewed.TranscriptVersionID)
	target := revSegs[0]
	var edited segmentResp
	e.must(e.do("PATCH",
		"/api/v1/transcripts/"+reviewed.TranscriptVersionID+"/segments/"+target.SegmentID,
		reviewer, map[string]any{"text": "Edited opening line.", "speaker_label": "Priya"}, &edited),
		http.StatusOK, "edit segment")
	if edited.Text != "Edited opening line." || edited.SpeakerLabel != "Priya" {
		t.Fatalf("edit not applied: %+v", edited)
	}

	// Approve.
	var approval approvalResp
	e.must(e.do("POST", "/api/v1/jobs/"+job.JobID+"/approve", reviewer, map[string]any{
		"reviewed_transcript_version_id": reviewed.TranscriptVersionID,
	}, &approval), http.StatusCreated, "approve")

	var jobNow jobResp
	e.must(e.do("GET", "/api/v1/jobs/"+job.JobID, producer, nil, &jobNow), http.StatusOK, "get job")
	if jobNow.Status != "approved" {
		t.Fatalf("status %s, want approved", jobNow.Status)
	}

	// The approved version is immutable and carries the reviewer edit.
	approvedSegs := listSegments(e, approval.ApprovedTranscriptVersionID)
	if approvedSegs[0].Text != "Edited opening line." || approvedSegs[0].SpeakerLabel != "Priya" {
		t.Fatal("approved version lost the reviewer edit")
	}

	// Export all four formats.
	var exports struct {
		Exports []exportResp `json:"exports"`
	}
	e.must(e.do("POST", "/api/v1/jobs/"+job.JobID+"/exports", producer, map[string]any{
		"formats": []string{"txt", "md", "srt", "vtt"},
	}, &exports), http.StatusCreated, "create exports")
	if len(exports.Exports) != 4 {
		t.Fatalf("want 4 exports, got %d", len(exports.Exports))
	}
	wantType := map[string]string{
		"txt": "text/plain", "md": "text/markdown",
		"srt": "application/x-subrip", "vtt": "text/vtt",
	}
	for _, ex := range exports.Exports {
		if ex.ValidationStatus != "passed" {
			t.Fatalf("export %s validation %s, want passed", ex.Format, ex.ValidationStatus)
		}
		if ex.DownloadURL == "" {
			t.Fatalf("export %s missing signed download_url", ex.Format)
		}
		res, err := http.Get(e.srv.URL + ex.DownloadURL) // signed link; no auth headers needed
		if err != nil {
			t.Fatal(err)
		}
		body := new(bytes.Buffer)
		_, _ = body.ReadFrom(res.Body)
		res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("download %s: status %d", ex.Format, res.StatusCode)
		}
		if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, wantType[ex.Format]) {
			t.Fatalf("download %s content-type %q", ex.Format, ct)
		}
		if cd := res.Header.Get("Content-Disposition"); !strings.Contains(cd, "attachment") {
			t.Fatalf("download %s disposition %q", ex.Format, cd)
		}
		if body.Len() == 0 {
			t.Fatalf("download %s empty body", ex.Format)
		}
		if ex.Format == "vtt" && !strings.HasPrefix(body.String(), "WEBVTT") {
			t.Fatal("vtt download missing WEBVTT header")
		}
		if ex.Format == "srt" && !strings.HasPrefix(body.String(), "1\n") {
			t.Fatal("srt download missing leading sequence number")
		}
	}
	res, err := http.Get(e.srv.URL + "/api/v1/exports/" + exports.Exports[0].ExportID + "/download")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("download without token status %d, want 401", res.StatusCode)
	}

	e.must(e.do("GET", "/api/v1/jobs/"+job.JobID, producer, nil, &jobNow), http.StatusOK, "get job")
	if jobNow.Status != "exported" {
		t.Fatalf("status %s, want exported", jobNow.Status)
	}

	// Audit trail (spec test 2 required events).
	var auditOut struct {
		Events []struct {
			EventType string `json:"event_type"`
		} `json:"events"`
	}
	e.must(e.do("GET", "/api/v1/jobs/"+job.JobID+"/audit", producer, nil, &auditOut),
		http.StatusOK, "audit")
	seen := map[string]bool{}
	for _, ev := range auditOut.Events {
		seen[ev.EventType] = true
	}
	for _, want := range []string{
		"job.submitted", "tool.get_media_metadata.completed",
		"tool.extract_audio.completed", "tool.transcribe_audio.completed",
		"tool.normalize_transcript.completed", "tool.quality_check_transcript.completed",
		"transcript.approved", "tool.export_transcript.completed",
	} {
		if !seen[want] {
			t.Errorf("audit trail missing %s (have %v)", want, seen)
		}
	}
}

// Caption reuse path (spec test 3): youtube job with official captions pauses
// for the producer decision, then produces a null-confidence raw transcript
// and a confidence_unavailable quality report.
func TestCaptionReusePathE2E(t *testing.T) {
	e := newEnv(t, nil)
	job := submitJob(e, "youtube", "https://www.youtube.com/watch?v=demo1&captions=1")

	if job.Status != "needs_user_action" || job.ActionRequired != "caption_decision" {
		t.Fatalf("want needs_user_action/caption_decision, got %s/%s", job.Status, job.ActionRequired)
	}

	var after jobResp
	e.must(e.do("POST", "/api/v1/jobs/"+job.JobID+"/caption-decision", producer,
		map[string]any{"reuse_captions": true}, &after), http.StatusOK, "caption decision")
	if after.Status != "in_review" {
		t.Fatalf("after reuse decision status %s, want in_review", after.Status)
	}

	versions := listVersions(e, job.JobID)
	raw := versionOfType(t, versions, "raw")
	segs := listSegments(e, raw.TranscriptVersionID)
	if len(segs) == 0 {
		t.Fatal("caption-derived raw transcript has no segments")
	}
	for _, sg := range segs {
		if sg.Confidence != nil {
			t.Fatal("caption-derived segments must carry null confidence (PRD 14.5)")
		}
		if !sg.Flags["caption_origin"] {
			t.Fatal("caption-derived segments must carry caption_origin flag")
		}
		if sg.SpeakerLabel != "Speaker 1" {
			t.Fatalf("caption-derived speaker %q, want Speaker 1", sg.SpeakerLabel)
		}
	}

	var qr qualityResp
	e.must(e.do("GET", "/api/v1/jobs/"+job.JobID+"/quality-report", producer, nil, &qr),
		http.StatusOK, "quality report")
	if !qr.ConfidenceUnavailable {
		t.Fatal("quality report must mark confidence_unavailable=true (PRD R5)")
	}
	if qr.AverageConfidence != nil || qr.LowConfidenceSegmentCount != 0 {
		t.Fatalf("caption path must skip threshold flagging: %+v", qr)
	}
	if qr.ConfidenceThreshold != 0.80 {
		t.Fatalf("threshold %v, want snapshot value 0.80", qr.ConfidenceThreshold)
	}

	// Fresh-transcription decision on a second captioned job goes through STT.
	job2 := submitJob(e, "youtube", "https://www.youtube.com/watch?v=demo2&captions=1")
	var after2 jobResp
	e.must(e.do("POST", "/api/v1/jobs/"+job2.JobID+"/caption-decision", producer,
		map[string]any{"reuse_captions": false}, &after2), http.StatusOK, "fresh decision")
	if after2.Status != "in_review" {
		t.Fatalf("fresh path status %s, want in_review", after2.Status)
	}
	raw2 := versionOfType(t, listVersions(e, job2.JobID), "raw")
	if segs2 := listSegments(e, raw2.TranscriptVersionID); segs2[0].Confidence == nil {
		t.Fatal("fresh transcription path must carry confidence scores")
	}

	// A youtube job without official captions never pauses.
	job3 := submitJob(e, "youtube", "https://www.youtube.com/watch?v=nocaps")
	if job3.Status != "in_review" {
		t.Fatalf("no-captions youtube job status %s, want in_review", job3.Status)
	}
}

// Export blocked before approval (spec test 4).
func TestExportBlockedBeforeApproval(t *testing.T) {
	e := newEnv(t, nil)
	job := submitJob(e, "upload", "uploads/blocked.mp3")
	var er errResp
	status := e.do("POST", "/api/v1/jobs/"+job.JobID+"/exports", producer,
		map[string]any{"formats": []string{"srt"}}, &er)
	e.must(status, http.StatusConflict, "export before approval")
	if er.Error.Code != "APPROVED_TRANSCRIPT_REQUIRED" {
		t.Fatalf("code %s, want APPROVED_TRANSCRIPT_REQUIRED", er.Error.Code)
	}
}

// Approval rules (spec test 6): reviewed version required, immutability
// enforced after approval, reopen supersedes the prior approval.
func TestApprovalImmutabilityAndReopen(t *testing.T) {
	e := newEnv(t, nil)
	job, reviewedID, _ := runToApproved(e)

	// PATCH on the now-approved flow's reviewed/approved versions -> 409.
	versions := listVersions(e, job.JobID)
	approvedV := versionOfType(t, versions, "approved")
	segs := listSegments(e, approvedV.TranscriptVersionID)
	var er errResp
	status := e.do("PATCH",
		"/api/v1/transcripts/"+approvedV.TranscriptVersionID+"/segments/"+segs[0].SegmentID,
		reviewer, map[string]any{"text": "tamper"}, &er)
	e.must(status, http.StatusConflict, "edit approved version")
	if er.Error.Code != "TRANSCRIPT_VERSION_IMMUTABLE" {
		t.Fatalf("code %s, want TRANSCRIPT_VERSION_IMMUTABLE", er.Error.Code)
	}

	// Approving a non-reviewed version is rejected.
	cleanV := versionOfType(t, versions, "clean")
	status = e.do("POST", "/api/v1/jobs/"+job.JobID+"/approve", reviewer, map[string]any{
		"reviewed_transcript_version_id": cleanV.TranscriptVersionID,
	}, &er)
	e.must(status, http.StatusConflict, "approve clean version")

	// Reopen (PRD 11.4): back to in_review with a fresh reviewed version
	// copied from the approved one.
	var reopened jobResp
	e.must(e.do("POST", "/api/v1/jobs/"+job.JobID+"/reopen", reviewer, map[string]any{}, &reopened),
		http.StatusOK, "reopen")
	if reopened.Status != "in_review" {
		t.Fatalf("status %s, want in_review", reopened.Status)
	}
	versions = listVersions(e, job.JobID)
	newReviewed := versionOfType(t, versions, "reviewed")
	if newReviewed.TranscriptVersionID == reviewedID {
		t.Fatal("reopen must create a NEW reviewed version")
	}
	if newReviewed.SourceVersionID == nil || *newReviewed.SourceVersionID != approvedV.TranscriptVersionID {
		t.Fatal("reopened reviewed version must be copied from the approved version")
	}

	// Re-approve: prior approval gets superseded_by_approval_id.
	var approval2 approvalResp
	e.must(e.do("POST", "/api/v1/jobs/"+job.JobID+"/approve", reviewer, map[string]any{
		"reviewed_transcript_version_id": newReviewed.TranscriptVersionID,
	}, &approval2), http.StatusCreated, "re-approve")

	jobUUID := uuid.MustParse(job.JobID)
	approvals, err := e.app.Tools.Stores.Approvals.ListApprovalsByJob(t.Context(), jobUUID)
	if err != nil {
		t.Fatal(err)
	}
	if len(approvals) != 2 {
		t.Fatalf("want 2 approvals, got %d", len(approvals))
	}
	first, second := approvals[0], approvals[1]
	if first.SupersededByApprovalID == nil || first.SupersededByApprovalID.String() != second.ApprovalID.String() {
		t.Fatal("prior approval must be superseded by the new approval (PRD 11.4)")
	}
	if second.SupersededByApprovalID != nil {
		t.Fatal("current approval must not be superseded")
	}
}

func TestApprovalBlocksCriticalQualityIssues(t *testing.T) {
	e := newEnv(t, nil)
	job := submitJob(e, "upload", "uploads/critical-quality.mp3")
	var reviewed versionResp
	e.must(e.do("POST", "/api/v1/jobs/"+job.JobID+"/review", reviewer, map[string]any{}, &reviewed),
		http.StatusCreated, "create review version")

	jobID := uuid.MustParse(job.JobID)
	currentJob, err := e.app.Tools.Stores.Jobs.GetJob(t.Context(), jobID)
	if err != nil {
		t.Fatal(err)
	}
	if currentJob.JobConfigID == nil {
		t.Fatal("job_config_id missing")
	}
	clean := versionOfType(t, listVersions(e, job.JobID), "clean")
	report := &domain.QualityReport{
		QualityReportID:     uuid.New(),
		JobID:               jobID,
		TranscriptVersionID: uuid.MustParse(clean.TranscriptVersionID),
		JobConfigID:         *currentJob.JobConfigID,
		ConfidenceThreshold: 0.8,
		Issues: []domain.QualityIssue{{
			IssueType: "MEANING_CHANGE_RISK",
			Severity:  "critical",
			StartMS:   0,
			EndMS:     1000,
			Message:   "Injected by test to verify approval policy.",
		}},
		CreatedAt: time.Now().UTC().Add(time.Hour),
	}
	if err := e.app.Tools.Stores.Quality.CreateReport(t.Context(), report); err != nil {
		t.Fatal(err)
	}

	var er errResp
	status := e.do("POST", "/api/v1/jobs/"+job.JobID+"/approve", reviewer, map[string]any{
		"reviewed_transcript_version_id": reviewed.TranscriptVersionID,
	}, &er)
	e.must(status, http.StatusConflict, "approve with critical quality issue")
	if er.Error.Code != "OPEN_CRITICAL_ISSUES" {
		t.Fatalf("code %s, want OPEN_CRITICAL_ISSUES", er.Error.Code)
	}
}

// Config centralization (spec test 7): transcribe, quality check, and summary
// all read the same job_config snapshot — assert with non-default values.
func TestConfigCentralization(t *testing.T) {
	defaults := domain.DefaultJobConfig("mock")
	defaults.ConfidenceThreshold = 0.90
	defaults.SummaryMaxWords = 25
	e := newEnv(t, &defaults)

	job := submitJob(e, "upload", "uploads/config-check.mp3")
	if job.JobConfig == nil || job.JobConfig.ConfidenceThreshold != 0.90 || job.JobConfig.SummaryMaxWords != 25 {
		t.Fatalf("job_config snapshot did not capture overridden defaults: %+v", job.JobConfig)
	}

	// transcribe_audio flagged with the snapshot threshold.
	raw := versionOfType(t, listVersions(e, job.JobID), "raw")
	lowCount := 0
	for _, sg := range listSegments(e, raw.TranscriptVersionID) {
		below := *sg.Confidence < 0.90
		if below != sg.Flags["low_confidence"] {
			t.Fatalf("segment flag inconsistent with snapshot threshold 0.90 (conf=%.2f flag=%v)",
				*sg.Confidence, sg.Flags["low_confidence"])
		}
		if below {
			lowCount++
		}
	}

	// quality_check_transcript used the same snapshot value and counts.
	var qr qualityResp
	e.must(e.do("GET", "/api/v1/jobs/"+job.JobID+"/quality-report", producer, nil, &qr),
		http.StatusOK, "quality report")
	if qr.ConfidenceThreshold != 0.90 {
		t.Fatalf("quality report threshold %v, want snapshot 0.90", qr.ConfidenceThreshold)
	}
	if qr.LowConfidenceSegmentCount != lowCount {
		t.Fatalf("report low-confidence count %d != segment flag count %d",
			qr.LowConfidenceSegmentCount, lowCount)
	}

	// generate_summary respected snapshot summary_max_words.
	var sum summaryResp
	e.must(e.do("POST", "/api/v1/jobs/"+job.JobID+"/summary", producer, map[string]any{}, &sum),
		http.StatusCreated, "summary")
	if n := len(strings.Fields(sum.Text)); n > 25 {
		t.Fatalf("summary word count %d exceeds snapshot summary_max_words 25", n)
	}
	if sum.ValidationStatus != "passed" {
		t.Fatalf("extractive mock summary must pass grounding, got %s", sum.ValidationStatus)
	}
}

// RBAC (spec test 8): producer cannot approve/review/edit or read another
// producer's jobs; missing or invalid identity is 401; health stays open.
func TestRBAC(t *testing.T) {
	e := newEnv(t, nil)

	// Missing identity -> 401.
	var er errResp
	e.must(e.do("GET", "/api/v1/jobs", nil, nil, &er), http.StatusUnauthorized, "no identity")
	if er.Error.Code != "UNAUTHENTICATED" {
		t.Fatalf("code %s, want UNAUTHENTICATED", er.Error.Code)
	}
	// Invalid role -> 401.
	badRole := map[string]string{"X-User-Id": "eve", "X-User-Role": "superuser"}
	e.must(e.do("GET", "/api/v1/jobs", badRole, nil, &er), http.StatusUnauthorized, "bad role")

	// Producer cannot approve / review / reopen / edit segments -> 403.
	fakeJob := uuid.NewString()
	fakeVersion := uuid.NewString()
	fakeSegment := uuid.NewString()
	for what, req := range map[string][2]string{
		"approve": {"POST", "/api/v1/jobs/" + fakeJob + "/approve"},
		"review":  {"POST", "/api/v1/jobs/" + fakeJob + "/review"},
		"reopen":  {"POST", "/api/v1/jobs/" + fakeJob + "/reopen"},
		"edit":    {"PATCH", "/api/v1/transcripts/" + fakeVersion + "/segments/" + fakeSegment},
	} {
		status := e.do(req[0], req[1], producer, map[string]any{}, &er)
		e.must(status, http.StatusForbidden, "producer "+what)
		if er.Error.Code != "USER_NOT_AUTHORIZED" {
			t.Fatalf("producer %s: code %s, want USER_NOT_AUTHORIZED", what, er.Error.Code)
		}
	}

	// Health check requires no identity.
	e.must(e.do("GET", "/healthz", nil, nil, nil), http.StatusOK, "healthz")
	e.must(e.do("GET", "/api/v1/healthz", nil, nil, nil), http.StatusOK, "api healthz")

	// Producers see their own jobs, not another producer's jobs.
	aliceJob := submitJob(e, "upload", "uploads/alice-owned.mp3")
	var malloryJob jobResp
	e.must(e.do("POST", "/api/v1/jobs", producer2, map[string]any{
		"source_type":        "upload",
		"source_uri":         "uploads/mallory-owned.mp3",
		"language":           "en",
		"ownership_attested": true,
	}, &malloryJob), http.StatusCreated, "mallory submit")
	e.must(e.do("GET", "/api/v1/jobs/"+malloryJob.JobID, producer, nil, &er),
		http.StatusForbidden, "producer cannot read another producer job")
	var jobs struct {
		Jobs []jobResp `json:"jobs"`
	}
	e.must(e.do("GET", "/api/v1/jobs", producer, nil, &jobs), http.StatusOK, "producer list jobs")
	seenAlice, seenMallory := false, false
	for _, job := range jobs.Jobs {
		if job.JobID == aliceJob.JobID {
			seenAlice = true
		}
		if job.JobID == malloryJob.JobID {
			seenMallory = true
		}
	}
	if !seenAlice || seenMallory {
		t.Fatalf("producer list isolation wrong: seenAlice=%v seenMallory=%v jobs=%+v", seenAlice, seenMallory, jobs.Jobs)
	}
}

// Missing ownership attestation blocks job creation entirely (PRD R1).
func TestOwnershipAttestationRequired(t *testing.T) {
	e := newEnv(t, nil)
	var er errResp
	status := e.do("POST", "/api/v1/jobs", producer, map[string]any{
		"source_type":        "upload",
		"source_uri":         "uploads/unattested.mp3",
		"language":           "en",
		"ownership_attested": false,
	}, &er)
	e.must(status, http.StatusBadRequest, "unattested submit")
	if er.Error.Code != "OWNERSHIP_ATTESTATION_MISSING" {
		t.Fatalf("code %s, want OWNERSHIP_ATTESTATION_MISSING", er.Error.Code)
	}
	var jobs struct {
		Jobs []jobResp `json:"jobs"`
	}
	e.must(e.do("GET", "/api/v1/jobs", producer, nil, &jobs), http.StatusOK, "list jobs")
	if len(jobs.Jobs) != 0 {
		t.Fatal("job must NOT be created without attestation")
	}

	// Unsupported upload format is a specific 400 at submission (PRD R1).
	status = e.do("POST", "/api/v1/jobs", producer, map[string]any{
		"source_type":        "upload",
		"source_uri":         "uploads/slides.pdf",
		"language":           "en",
		"ownership_attested": true,
	}, &er)
	e.must(status, http.StatusBadRequest, "unsupported format")
	if er.Error.Code != "UNSUPPORTED_FORMAT" {
		t.Fatalf("code %s, want UNSUPPORTED_FORMAT", er.Error.Code)
	}
}

func TestUploadMediaFlow(t *testing.T) {
	e := newEnv(t, nil)
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", "episode-upload.mp3")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("fake mp3 payload for mock media provider")); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", e.srv.URL+"/api/v1/uploads", &body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	for k, v := range producer {
		req.Header.Set(k, v)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var upload struct {
		SourceURI string `json:"source_uri"`
		Filename  string `json:"filename"`
		SizeBytes int64  `json:"size_bytes"`
	}
	if err := json.NewDecoder(res.Body).Decode(&upload); err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("upload status %d, want 201", res.StatusCode)
	}
	if !strings.HasPrefix(upload.SourceURI, "file://") || upload.Filename != "episode-upload.mp3" || upload.SizeBytes == 0 {
		t.Fatalf("bad upload response: %+v", upload)
	}

	var job jobResp
	e.must(e.do("POST", "/api/v1/jobs", producer, map[string]any{
		"source_type":        "upload",
		"source_uri":         upload.SourceURI,
		"language":           "en",
		"ownership_attested": true,
	}, &job), http.StatusCreated, "submit uploaded media")
	if job.Status != "in_review" {
		t.Fatalf("uploaded job status %s, want in_review", job.Status)
	}
}

// No-audio media parks in needs_user_action/replace_media; replace_job_media
// re-attests and restarts from queued (PRD 14.13, 19).
func TestReplaceMediaFlow(t *testing.T) {
	e := newEnv(t, nil)
	job := submitJob(e, "upload", "uploads/noaudio-clip.mp4")
	if job.Status != "needs_user_action" || job.ActionRequired != "replace_media" {
		t.Fatalf("want needs_user_action/replace_media, got %s/%s", job.Status, job.ActionRequired)
	}
	if job.LastError == nil || job.LastError.Code != "NO_AUDIO_TRACK" {
		t.Fatalf("last_error %+v, want NO_AUDIO_TRACK", job.LastError)
	}

	// Replacement without re-attestation is rejected.
	var er errResp
	status := e.do("POST", "/api/v1/jobs/"+job.JobID+"/replace-media", producer, map[string]any{
		"source_type": "upload", "source_uri": "uploads/fixed.mp3", "ownership_attested": false,
	}, &er)
	e.must(status, http.StatusBadRequest, "replace without attestation")
	if er.Error.Code != "OWNERSHIP_ATTESTATION_MISSING" {
		t.Fatalf("code %s, want OWNERSHIP_ATTESTATION_MISSING", er.Error.Code)
	}

	// Valid replacement restarts the pipeline and completes.
	var after jobResp
	e.must(e.do("POST", "/api/v1/jobs/"+job.JobID+"/replace-media", producer, map[string]any{
		"source_type": "upload", "source_uri": "uploads/fixed.mp3", "ownership_attested": true,
	}, &after), http.StatusOK, "replace media")
	if after.Status != "in_review" {
		t.Fatalf("after replacement status %s, want in_review", after.Status)
	}
	if after.LastError != nil || after.ActionRequired != "" {
		t.Fatalf("replacement must clear last_error/action_required: %+v", after)
	}

	// Replacement on a job that is not in needs_user_action -> 409.
	status = e.do("POST", "/api/v1/jobs/"+job.JobID+"/replace-media", producer, map[string]any{
		"source_type": "upload", "source_uri": "uploads/again.mp3", "ownership_attested": true,
	}, &er)
	e.must(status, http.StatusConflict, "replace in wrong state")
}

// STT failure matrix (PRD 19): quota returns the job to queued; a transient
// timeout is retried once and succeeds.
func TestSTTFailureMatrix(t *testing.T) {
	t.Run("quota returns job to queued", func(t *testing.T) {
		e := newEnv(t, nil)
		job := submitJob(e, "upload", "uploads/stt-quota-show.mp3")
		if job.Status != "queued" {
			t.Fatalf("status %s, want queued after quota exhaustion", job.Status)
		}
		if job.LastError == nil || job.LastError.Code != "STT_PROVIDER_QUOTA_EXCEEDED" {
			t.Fatalf("last_error %+v, want STT_PROVIDER_QUOTA_EXCEEDED", job.LastError)
		}
	})
	t.Run("timeout retried once then succeeds", func(t *testing.T) {
		e := newEnv(t, nil)
		job := submitJob(e, "upload", "uploads/stt-timeout-once-show.mp3")
		if job.Status != "in_review" {
			t.Fatalf("status %s, want in_review after retry", job.Status)
		}
		var auditOut struct {
			Events []struct {
				EventType string `json:"event_type"`
			} `json:"events"`
		}
		e.must(e.do("GET", "/api/v1/jobs/"+job.JobID+"/audit", producer, nil, &auditOut),
			http.StatusOK, "audit")
		failed, completed := false, false
		for _, ev := range auditOut.Events {
			if ev.EventType == "tool.transcribe_audio.failed" {
				failed = true
			}
			if ev.EventType == "tool.transcribe_audio.completed" {
				completed = true
			}
		}
		if !failed || !completed {
			t.Fatalf("expected failed-then-completed transcribe audit trail (failed=%v completed=%v)", failed, completed)
		}
	})
}

// Cancellation rules (PRD 14.14): submitter can cancel pre-approval; only
// admin can cancel after approval; terminal jobs reject cancellation; records
// are never deleted.
func TestCancelRules(t *testing.T) {
	e := newEnv(t, nil)

	// Pause a job at the caption decision so it is cancellable mid-flight.
	job := submitJob(e, "youtube", "https://www.youtube.com/watch?v=c1&captions=1")

	// A different producer cannot cancel someone else's job.
	var er errResp
	status := e.do("POST", "/api/v1/jobs/"+job.JobID+"/cancel", producer2,
		map[string]any{"reason": "not mine"}, &er)
	e.must(status, http.StatusForbidden, "foreign cancel")

	// The submitter can.
	var cancelled jobResp
	e.must(e.do("POST", "/api/v1/jobs/"+job.JobID+"/cancel", producer,
		map[string]any{"reason": "wrong episode"}, &cancelled), http.StatusOK, "cancel")
	if cancelled.Status != "cancelled" {
		t.Fatalf("status %s, want cancelled", cancelled.Status)
	}

	// Terminal jobs reject further cancellation.
	status = e.do("POST", "/api/v1/jobs/"+job.JobID+"/cancel", producer,
		map[string]any{"reason": "again"}, &er)
	e.must(status, http.StatusConflict, "cancel terminal")
	if er.Error.Code != "JOB_ALREADY_TERMINAL" {
		t.Fatalf("code %s, want JOB_ALREADY_TERMINAL", er.Error.Code)
	}

	// After approval, only admin may cancel.
	approvedJob, _, _ := runToApproved(e)
	status = e.do("POST", "/api/v1/jobs/"+approvedJob.JobID+"/cancel", producer,
		map[string]any{"reason": "changed my mind"}, &er)
	e.must(status, http.StatusForbidden, "post-approval cancel by producer")
	e.must(e.do("POST", "/api/v1/jobs/"+approvedJob.JobID+"/cancel", admin,
		map[string]any{"reason": "legal hold"}, &cancelled), http.StatusOK, "post-approval cancel by admin")

	// Cancellation never deletes transcript versions or approvals.
	if len(listVersions(e, approvedJob.JobID)) == 0 {
		t.Fatal("cancellation must not delete transcript versions")
	}
	approvals, err := e.app.Tools.Stores.Approvals.ListApprovalsByJob(
		t.Context(), uuid.MustParse(approvedJob.JobID))
	if err != nil || len(approvals) == 0 {
		t.Fatal("cancellation must not delete approvals")
	}
}

// Summary lifecycle: generate (grounded, versioned), fetch latest, edit.
func TestSummaryLifecycle(t *testing.T) {
	e := newEnv(t, nil)
	job, _, approvedID := runToApproved(e)

	// 404 before any summary exists.
	var er errResp
	e.must(e.do("GET", "/api/v1/jobs/"+job.JobID+"/summary", producer, nil, &er),
		http.StatusNotFound, "summary before generation")

	var sum summaryResp
	e.must(e.do("POST", "/api/v1/jobs/"+job.JobID+"/summary", producer, map[string]any{}, &sum),
		http.StatusCreated, "generate summary")
	if sum.SourceTranscriptVersionID != approvedID {
		t.Fatalf("summary source %s, want approved version %s (most authoritative)",
			sum.SourceTranscriptVersionID, approvedID)
	}
	if n := len(strings.Fields(sum.Text)); n > 150 {
		t.Fatalf("summary word count %d exceeds default 150", n)
	}
	if sum.ValidationStatus != "passed" {
		t.Fatalf("validation %s, want passed", sum.ValidationStatus)
	}

	var got summaryResp
	e.must(e.do("GET", "/api/v1/jobs/"+job.JobID+"/summary", producer, nil, &got),
		http.StatusOK, "get summary")
	if got.SummaryID != sum.SummaryID {
		t.Fatal("GET summary must return the latest summary")
	}

	var edited summaryResp
	e.must(e.do("PATCH", "/api/v1/summaries/"+sum.SummaryID, producer,
		map[string]any{"text": "Edited summary text."}, &edited), http.StatusOK, "edit summary")
	if edited.Text != "Edited summary text." {
		t.Fatalf("edit not applied: %q", edited.Text)
	}
}
