package app_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/aaraminds/transcript-agent/internal/app"
	"github.com/aaraminds/transcript-agent/internal/domain"
	"github.com/aaraminds/transcript-agent/internal/objectstore"
	"github.com/aaraminds/transcript-agent/internal/providers/stt"
	sttmock "github.com/aaraminds/transcript-agent/internal/providers/stt/mock"
	"github.com/aaraminds/transcript-agent/internal/store"
)

// --- fault-injection wrappers ------------------------------------------------

// faultyAudit fails Append while fail is set (store-fault injection for the
// audit-blocks-high-risk-actions rule).
type faultyAudit struct {
	inner store.AuditStore
	fail  atomic.Bool
}

func (f *faultyAudit) Append(ctx context.Context, e *domain.AuditEvent) error {
	if f.fail.Load() {
		return errors.New("injected audit store fault")
	}
	return f.inner.Append(ctx, e)
}

func (f *faultyAudit) ListByJob(ctx context.Context, jobID uuid.UUID) ([]*domain.AuditEvent, error) {
	return f.inner.ListByJob(ctx, jobID)
}

// faultyJobs fails ListJobsByStatus (healthz probe) and/or GetJob (sanitized
// 500 check) on demand; everything else passes through.
type faultyJobs struct {
	store.JobStore
	failList atomic.Bool
	failGet  atomic.Bool
}

func (f *faultyJobs) ListJobsByStatus(ctx context.Context, statuses ...domain.Status) ([]*domain.Job, error) {
	if f.failList.Load() {
		return nil, errors.New("injected job store fault")
	}
	return f.JobStore.ListJobsByStatus(ctx, statuses...)
}

func (f *faultyJobs) GetJob(ctx context.Context, id uuid.UUID) (*domain.Job, error) {
	if f.failGet.Load() {
		return nil, errors.New("pgx: connection refused to db-internal-host:5432")
	}
	return f.JobStore.GetJob(ctx, id)
}

// blockingSTT blocks until its context is cancelled (drain testing) with a
// generous fallback so a broken test cannot hang the suite.
type blockingSTT struct{ inner stt.Provider }

func (s blockingSTT) Transcribe(ctx context.Context, uri, language string, diarize bool) (*stt.Result, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(5 * time.Second):
	}
	return s.inner.Transcribe(ctx, uri, language, diarize)
}

// --- fix 1: audit failure blocks high-risk actions ---------------------------

// An injected audit store fault turns approve (and admin cancel) into 503
// AUDIT_UNAVAILABLE but never breaks read paths like job listing.
func TestAuditFaultBlocksHighRiskActions(t *testing.T) {
	var fa *faultyAudit
	e := newEnvWith(t, nil, func(o *app.Options) {
		fa = &faultyAudit{inner: o.Stores.Audit}
		o.Stores.Audit = fa
	})
	job := submitJob(e, "upload", "mock://uploads/audit-fault.mp3")
	if job.Status != "in_review" {
		t.Fatalf("status %s, want in_review", job.Status)
	}
	var reviewed versionResp
	e.must(e.do("POST", "/api/v1/jobs/"+job.JobID+"/review", reviewer, map[string]any{}, &reviewed),
		http.StatusCreated, "create review version")

	fa.fail.Store(true)

	var er errResp
	status := e.do("POST", "/api/v1/jobs/"+job.JobID+"/approve", reviewer, map[string]any{
		"reviewed_transcript_version_id": reviewed.TranscriptVersionID,
	}, &er)
	e.must(status, http.StatusServiceUnavailable, "approve under audit fault")
	if er.Error.Code != "AUDIT_UNAVAILABLE" {
		t.Fatalf("code %s, want AUDIT_UNAVAILABLE", er.Error.Code)
	}
	got, err := e.app.Tools.Stores.Jobs.GetJob(t.Context(), uuid.MustParse(job.JobID))
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.StatusInReview {
		t.Fatalf("audit-failed approve changed job status to %s, want in_review", got.Status)
	}
	if approved, err := e.app.Tools.Stores.Transcripts.LatestVersion(t.Context(), got.JobID, domain.VersionApproved); err != nil {
		t.Fatal(err)
	} else if approved != nil {
		t.Fatalf("audit-failed approve created approved version %s", approved.TranscriptVersionID)
	}
	approvals, err := e.app.Tools.Stores.Approvals.ListApprovalsByJob(t.Context(), got.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if len(approvals) != 0 {
		t.Fatalf("audit-failed approve created %d approval rows", len(approvals))
	}

	// Cancel is high-risk too: 503 under the same fault.
	other := submitJob(e, "upload", "mock://uploads/audit-fault-cancel.mp3")
	status = e.do("POST", "/api/v1/jobs/"+other.JobID+"/cancel", admin,
		map[string]any{"reason": "audit down"}, &er)
	e.must(status, http.StatusServiceUnavailable, "cancel under audit fault")
	if er.Error.Code != "AUDIT_UNAVAILABLE" {
		t.Fatalf("cancel code %s, want AUDIT_UNAVAILABLE", er.Error.Code)
	}
	got, err = e.app.Tools.Stores.Jobs.GetJob(t.Context(), uuid.MustParse(other.JobID))
	if err != nil {
		t.Fatal(err)
	}
	if got.Status == domain.StatusCancelled {
		t.Fatal("audit-failed cancel must not cancel the job")
	}
	if got.Status != domain.StatusInReview {
		t.Fatalf("audit-failed cancel changed job status to %s, want in_review", got.Status)
	}

	// Job listing (read path) is unaffected by the audit fault.
	var jobs struct {
		Jobs []jobResp `json:"jobs"`
	}
	e.must(e.do("GET", "/api/v1/jobs", reviewer, nil, &jobs), http.StatusOK, "list jobs under audit fault")
	if len(jobs.Jobs) == 0 {
		t.Fatal("job listing must still return jobs while audit is down")
	}

	// Informational paths stay fire-and-forget: a new submission succeeds even
	// though its audit events cannot be written.
	if j := submitJob(e, "upload", "mock://uploads/audit-fault-submit.mp3"); j.Status != "in_review" {
		t.Fatalf("submit under audit fault: status %s, want in_review", j.Status)
	}
}

// --- fix 2: graceful drain never fails a job on shutdown ---------------------

// Shutdown while a worker sits inside a blocked provider call: the drain
// window expires, the in-flight step is cancelled, and the job stays in its
// durable mid-pipeline state — never failed.
func TestDrainDoesNotFailInFlightJobs(t *testing.T) {
	e := newEnvWith(t, nil, func(o *app.Options) {
		o.Sync = false
		o.STT = blockingSTT{inner: sttmock.New()}
		o.DrainTimeout = 100 * time.Millisecond
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	e.app.Orch.Start(ctx, 1, 20*time.Millisecond)

	job := submitJob(e, "upload", "mock://uploads/drain-me.mp3")
	waitForJob(e, job.JobID, 10*time.Second, "transcribing")

	cancel()          // SIGTERM: stop intake, drain in-flight work
	e.app.Orch.Wait() // returns once the drain (bounded by DrainTimeout) is done

	got, err := e.app.Tools.Stores.Jobs.GetJob(t.Context(), uuid.MustParse(job.JobID))
	if err != nil {
		t.Fatal(err)
	}
	if got.Status == domain.StatusFailed {
		t.Fatal("shutdown-interrupted step must NOT mark the job failed")
	}
	if got.Status != domain.StatusTranscribing {
		t.Fatalf("job status %s, want transcribing (durable state kept for reclaim)", got.Status)
	}
}

// --- fix 3: stuck-job reclaim -------------------------------------------------

// A job artificially aged in a mid-pipeline state is CAS'd back to queued by
// the scanner and completes the pipeline.
func TestStuckJobReclaim(t *testing.T) {
	e := newEnvWith(t, nil, func(o *app.Options) {
		o.Sync = false
		o.StuckJobThreshold = 100 * time.Millisecond
	})
	old := time.Now().UTC().Add(-time.Hour)
	stuck := &domain.Job{
		JobID: uuid.New(), SourceType: "upload",
		SourceURI:   "mock://uploads/stuck-worker.mp3",
		Status:      domain.StatusTranscribing, // abandoned by a "crashed" worker
		SubmittedBy: "alice", OwnershipAttested: true, Language: "en",
		CreatedAt: old, UpdatedAt: old,
	}
	if err := e.app.Tools.Stores.Jobs.CreateJob(t.Context(), stuck); err != nil {
		t.Fatal(err)
	}

	e.app.Orch.Start(t.Context(), 2, 20*time.Millisecond)
	waitForJob(e, stuck.JobID.String(), 10*time.Second, "in_review")

	var auditOut struct {
		Events []struct {
			EventType string `json:"event_type"`
		} `json:"events"`
	}
	e.must(e.do("GET", "/api/v1/jobs/"+stuck.JobID.String()+"/audit", reviewer, nil, &auditOut),
		http.StatusOK, "audit")
	reclaimed := false
	for _, ev := range auditOut.Events {
		if ev.EventType == "job.reclaimed_stuck" {
			reclaimed = true
		}
	}
	if !reclaimed {
		t.Fatal("audit trail missing job.reclaimed_stuck")
	}
}

// --- fix 4 + approvals contract: supersede chain ------------------------------

// Re-approval supersedes prior exports (same tx), the approvals endpoint
// shows the chain newest first, and superseded downloads carry X-Superseded.
func TestExportSupersedeAndApprovalsChain(t *testing.T) {
	e := newEnv(t, nil)
	job, _, approvedID1 := runToApproved(e)

	var exports struct {
		Exports []exportResp `json:"exports"`
	}
	e.must(e.do("POST", "/api/v1/jobs/"+job.JobID+"/exports", reviewer,
		map[string]any{"formats": []string{"txt", "srt"}}, &exports), http.StatusCreated, "create exports")
	for _, ex := range exports.Exports {
		if ex.Superseded {
			t.Fatalf("fresh export %s must not be superseded", ex.ExportID)
		}
		if ex.ApprovedTranscriptVersionID != approvedID1 {
			t.Fatalf("export approved_transcript_version_id %s, want %s",
				ex.ApprovedTranscriptVersionID, approvedID1)
		}
	}
	firstExportID := exports.Exports[0].ExportID

	// Reopen and re-approve with a fresh reviewed version.
	var reopened jobResp
	e.must(e.do("POST", "/api/v1/jobs/"+job.JobID+"/reopen", reviewer, map[string]any{}, &reopened),
		http.StatusOK, "reopen")
	newReviewed := versionOfType(t, listVersions(e, job.JobID), "reviewed")
	var approval2 approvalResp
	e.must(e.do("POST", "/api/v1/jobs/"+job.JobID+"/approve", reviewer, map[string]any{
		"reviewed_transcript_version_id": newReviewed.TranscriptVersionID,
	}, &approval2), http.StatusCreated, "re-approve")

	// Approvals endpoint: newest first, supersede chain intact, frozen shape.
	var approvalsOut struct {
		Approvals []struct {
			ApprovalID                  string  `json:"approval_id"`
			ApprovedTranscriptVersionID string  `json:"approved_transcript_version_id"`
			ApprovedBy                  string  `json:"approved_by"`
			ApprovedAt                  string  `json:"approved_at"`
			ApprovalNote                string  `json:"approval_note"`
			SupersededByApprovalID      *string `json:"superseded_by_approval_id"`
		} `json:"approvals"`
	}
	e.must(e.do("GET", "/api/v1/jobs/"+job.JobID+"/approvals", producer, nil, &approvalsOut),
		http.StatusOK, "list approvals")
	if len(approvalsOut.Approvals) != 2 {
		t.Fatalf("want 2 approvals, got %d", len(approvalsOut.Approvals))
	}
	newest, prior := approvalsOut.Approvals[0], approvalsOut.Approvals[1]
	if newest.ApprovalID != approval2.ApprovalID {
		t.Fatal("approvals must be listed newest first")
	}
	if newest.SupersededByApprovalID != nil {
		t.Fatal("current approval must not be superseded")
	}
	if prior.SupersededByApprovalID == nil || *prior.SupersededByApprovalID != approval2.ApprovalID {
		t.Fatal("prior approval must point at the superseding approval")
	}
	if newest.ApprovedBy == "" || newest.ApprovedAt == "" {
		t.Fatal("approvals entries must carry approved_by/approved_at")
	}

	// Prior exports are now superseded; the list shows it.
	e.must(e.do("GET", "/api/v1/jobs/"+job.JobID+"/exports", producer, nil, &exports),
		http.StatusOK, "list exports after re-approve")
	for _, ex := range exports.Exports {
		if !ex.Superseded {
			t.Fatalf("export %s must be superseded after re-approval", ex.ExportID)
		}
	}

	// Superseded downloads still work but carry X-Superseded: true.
	res, body := e.get("/api/v1/exports/"+firstExportID+"/download", producer, "")
	if res.StatusCode != http.StatusOK || len(body) == 0 {
		t.Fatalf("superseded download status %d (%d bytes), want 200 with content", res.StatusCode, len(body))
	}
	if res.Header.Get("X-Superseded") != "true" {
		t.Fatalf("superseded download missing X-Superseded: true (got %q)", res.Header.Get("X-Superseded"))
	}

	// A new export from the new approval is not superseded and links to it.
	var exports2 struct {
		Exports []exportResp `json:"exports"`
	}
	e.must(e.do("POST", "/api/v1/jobs/"+job.JobID+"/exports", reviewer,
		map[string]any{"formats": []string{"txt"}}, &exports2), http.StatusCreated, "re-export")
	fresh := exports2.Exports[0]
	if fresh.Superseded {
		t.Fatal("new export must not be superseded")
	}
	if fresh.ApprovedTranscriptVersionID != approval2.ApprovedTranscriptVersionID {
		t.Fatal("new export must link to the new approved version")
	}
	res, _ = e.get("/api/v1/exports/"+fresh.ExportID+"/download", producer, "")
	if res.Header.Get("X-Superseded") != "" {
		t.Fatal("fresh export download must not carry X-Superseded")
	}
}

// --- fix 5: retention sweep ----------------------------------------------------

func TestRetentionSweep(t *testing.T) {
	var objects objectstore.ObjectStore
	e := newEnvWith(t, nil, func(o *app.Options) { objects = o.Objects })
	ctx := t.Context()

	// Pipeline artifacts carry retention_until at creation.
	job := submitJob(e, "upload", "mock://uploads/retention-run.mp3")
	audioArts, err := e.app.Tools.Stores.Artifacts.ListArtifactsByJob(
		ctx, uuid.MustParse(job.JobID), domain.ArtifactAudioExtract)
	if err != nil || len(audioArts) == 0 {
		t.Fatalf("no audio artifact: %v", err)
	}
	if audioArts[0].RetentionUntil == nil {
		t.Fatal("audio_extract artifact must carry retention_until at creation (PRD 16.4)")
	}
	if until := *audioArts[0].RetentionUntil; time.Until(until) < 29*24*time.Hour {
		t.Fatalf("retention_until %s not ~30 days out", until)
	}

	// An artificially expired audio artifact is swept: bytes and row deleted.
	staleURI, err := objects.Put(ctx, "sweep/audio/old.wav", []byte("stale bytes"))
	if err != nil {
		t.Fatal(err)
	}
	past := time.Now().UTC().Add(-time.Hour)
	expired := &domain.MediaArtifact{
		ArtifactID: uuid.New(), JobID: uuid.New(),
		ArtifactType: domain.ArtifactAudioExtract, URI: staleURI,
		MimeType: "audio/wav", SizeBytes: 11,
		RetentionUntil: &past, CreatedAt: past.Add(-24 * time.Hour),
	}
	if err := e.app.Tools.Stores.Artifacts.CreateArtifact(ctx, expired); err != nil {
		t.Fatal(err)
	}

	// An old export artifact has no retention_until and is never swept.
	exportURI, err := objects.Put(ctx, "sweep/exports/keep.txt", []byte("keep me"))
	if err != nil {
		t.Fatal(err)
	}
	exportArt := &domain.MediaArtifact{
		ArtifactID: uuid.New(), JobID: expired.JobID,
		ArtifactType: domain.ArtifactExport, URI: exportURI,
		MimeType: "text/plain", SizeBytes: 7,
		RetentionUntil: nil, CreatedAt: past.Add(-24 * time.Hour),
	}
	if err := e.app.Tools.Stores.Artifacts.CreateArtifact(ctx, exportArt); err != nil {
		t.Fatal(err)
	}

	e.app.Orch.SweepRetention(ctx)

	if _, err := e.app.Tools.Stores.Artifacts.GetArtifact(ctx, expired.ArtifactID); err == nil {
		t.Fatal("expired artifact row must be deleted by the sweep")
	}
	if _, err := objects.Get(ctx, staleURI); err == nil {
		t.Fatal("expired artifact bytes must be deleted by the sweep")
	}
	if _, err := e.app.Tools.Stores.Artifacts.GetArtifact(ctx, exportArt.ArtifactID); err != nil {
		t.Fatalf("export artifact must survive the sweep: %v", err)
	}
	if _, err := objects.Get(ctx, exportURI); err != nil {
		t.Fatalf("export bytes must survive the sweep: %v", err)
	}
	// The unexpired pipeline artifact survives too.
	if _, err := e.app.Tools.Stores.Artifacts.GetArtifact(ctx, audioArts[0].ArtifactID); err != nil {
		t.Fatalf("unexpired artifact must survive the sweep: %v", err)
	}
}

// --- fix 6: max_duration_seconds guardrail --------------------------------------

// Snapshot max_duration_seconds below the stub's 120s duration: the job parks
// in needs_user_action/duration_exceeded with DURATION_LIMIT_EXCEEDED; the
// admin path out is replace-media or cancel (no override endpoint in MVP).
func TestDurationCapGuardrail(t *testing.T) {
	defaults := domain.DefaultJobConfig("mock")
	maxDur := 60
	defaults.MaxDurationSeconds = &maxDur
	e := newEnv(t, &defaults)

	job := submitJob(e, "upload", "mock://uploads/way-too-long.mp3") // stub duration: 120s
	if job.Status != "needs_user_action" || job.ActionRequired != "duration_exceeded" {
		t.Fatalf("want needs_user_action/duration_exceeded, got %s/%s", job.Status, job.ActionRequired)
	}
	if job.LastError == nil || job.LastError.Code != "DURATION_LIMIT_EXCEEDED" {
		t.Fatalf("last_error %+v, want DURATION_LIMIT_EXCEEDED", job.LastError)
	}
	if !strings.Contains(job.LastError.Message, "120") || !strings.Contains(job.LastError.Message, "60") {
		t.Fatalf("last_error message must name actual and maximum duration: %q", job.LastError.Message)
	}

	// Documented resolution path: cancel (or replace media).
	var cancelled jobResp
	e.must(e.do("POST", "/api/v1/jobs/"+job.JobID+"/cancel", producer,
		map[string]any{"reason": "episode too long"}, &cancelled), http.StatusOK, "cancel over-limit job")
	if cancelled.Status != "cancelled" {
		t.Fatalf("status %s, want cancelled", cancelled.Status)
	}
}

// --- fix 7: metrics + healthz -----------------------------------------------------

func TestMetricsEndpoint(t *testing.T) {
	e := newEnv(t, nil)
	submitJob(e, "upload", "mock://uploads/metrics-run.mp3")

	res, body := e.get("/debug/vars", nil, "") // auth-exempt, internal
	if res.StatusCode != http.StatusOK {
		t.Fatalf("/debug/vars status %d, want 200", res.StatusCode)
	}
	var vars map[string]json.RawMessage
	if err := json.Unmarshal(body, &vars); err != nil {
		t.Fatalf("/debug/vars is not JSON: %v", err)
	}
	for _, key := range []string{
		"jobs_submitted", "jobs_completed", "jobs_failed_total", "tool_failures_total",
		"retries_total", "export_validation_failures", "audit_write_failures",
		"stt_seconds_processed",
	} {
		if _, ok := vars[key]; !ok {
			t.Errorf("/debug/vars missing %s", key)
		}
	}
	// Counters are process-global across test envs; presence plus non-zero
	// submissions is the reliable assertion.
	var submitted int64
	if err := json.Unmarshal(vars["jobs_submitted"], &submitted); err != nil || submitted < 1 {
		t.Fatalf("jobs_submitted = %d (%v), want >= 1", submitted, err)
	}
}

func TestHealthzDegradedOnStoreFault(t *testing.T) {
	var fj *faultyJobs
	e := newEnvWith(t, nil, func(o *app.Options) {
		fj = &faultyJobs{JobStore: o.Stores.Jobs}
		o.Stores.Jobs = fj
	})

	res, body := e.get("/healthz", nil, "")
	if res.StatusCode != http.StatusOK || !strings.Contains(string(body), `"ok"`) {
		t.Fatalf("healthy healthz: status %d body %s", res.StatusCode, body)
	}

	fj.failList.Store(true)
	res, body = e.get("/healthz", nil, "")
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("degraded healthz status %d, want 503", res.StatusCode)
	}
	var out map[string]string
	if err := json.Unmarshal(body, &out); err != nil || out["status"] != "degraded" {
		t.Fatalf("degraded healthz body %s, want {\"status\":\"degraded\"}", body)
	}
}

// --- fix 8: sanitized 500s ----------------------------------------------------------

// Internal store errors never leak pgx/OS strings: clients get the generic
// INTERNAL_ERROR envelope.
func TestInternalErrorsAreSanitized(t *testing.T) {
	var fj *faultyJobs
	e := newEnvWith(t, nil, func(o *app.Options) {
		fj = &faultyJobs{JobStore: o.Stores.Jobs}
		o.Stores.Jobs = fj
	})
	fj.failGet.Store(true)

	req, err := http.NewRequest("GET", e.srv.URL+"/api/v1/jobs/"+uuid.NewString(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range reviewer {
		req.Header.Set(k, v)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var er errResp
	raw := new(strings.Builder)
	dec := json.NewDecoder(io.TeeReader(res.Body, raw))
	if err := dec.Decode(&er); err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", res.StatusCode)
	}
	if er.Error.Code != "INTERNAL_ERROR" || er.Error.Message != "internal error" {
		t.Fatalf("error %+v, want generic INTERNAL_ERROR/internal error", er.Error)
	}
	if strings.Contains(raw.String(), "pgx") || strings.Contains(raw.String(), "db-internal-host") {
		t.Fatalf("internal details leaked to client: %s", raw.String())
	}
	if res.Header.Get("X-Request-Id") == "" {
		t.Fatal("responses must carry X-Request-Id for server-side error correlation")
	}
}

// --- summary contract addition ------------------------------------------------------

// Summary JSON exposes validation_notes (null when there are none).
func TestSummaryValidationNotesField(t *testing.T) {
	e := newEnv(t, nil)
	job, _, _ := runToApproved(e)
	var raw map[string]json.RawMessage
	e.must(e.do("POST", "/api/v1/jobs/"+job.JobID+"/summary", producer, map[string]any{}, &raw),
		http.StatusCreated, "generate summary")
	notes, ok := raw["validation_notes"]
	if !ok {
		t.Fatal("summary JSON missing validation_notes")
	}
	if string(notes) != "null" {
		t.Fatalf("grounded mock summary must have null validation_notes, got %s", notes)
	}
}
