// Package orchestrator drives the deterministic job workflow (PRD 11.1),
// exception handling (11.2, 19 failure matrix), and retry policy. It is a
// state-machine worker, not an LLM: high-risk transitions are never model
// decisions (PRD 12.3).
//
// Runtime shape: a bounded channel queue consumed by a small goroutine pool,
// plus a requeue scan ticker that picks up submitted/queued jobs (including
// jobs returned to queued after STT quota exhaustion, PRD 14.7). In Sync mode
// (tests, and useful for demos) Enqueue drives the job inline until it blocks.
package orchestrator

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/aaraminds/transcript-agent/internal/audit"
	"github.com/aaraminds/transcript-agent/internal/domain"
	"github.com/aaraminds/transcript-agent/internal/state"
	"github.com/aaraminds/transcript-agent/internal/tools"
)

// Orchestrator drives jobs through the pipeline.
type Orchestrator struct {
	Tools *tools.Toolset
	Log   *slog.Logger
	// Backoff is the wait before the single retry of a retryable failure.
	Backoff time.Duration
	// Sync makes Enqueue drive the job inline (deterministic tests/demos).
	Sync bool

	queue chan uuid.UUID

	mu         sync.Mutex
	pauseUntil time.Time // queue pause after STT quota exhaustion (PRD 19)
	running    map[uuid.UUID]bool
	// inFlight tracks IDs that are enqueued or being processed so the requeue
	// scanner never floods the queue with duplicates of the same job.
	inFlight map[uuid.UUID]bool
}

// New returns an orchestrator with a buffered queue.
func New(ts *tools.Toolset, log *slog.Logger, backoff time.Duration, sync bool) *Orchestrator {
	if log == nil {
		log = slog.Default()
	}
	if backoff <= 0 {
		backoff = 2 * time.Second
	}
	return &Orchestrator{
		Tools:    ts,
		Log:      log,
		Backoff:  backoff,
		Sync:     sync,
		queue:    make(chan uuid.UUID, 256),
		running:  map[uuid.UUID]bool{},
		inFlight: map[uuid.UUID]bool{},
	}
}

// Enqueue schedules a job for processing. Non-blocking in async mode; the
// requeue scan is the safety net if the queue is full. IDs already queued or
// processing are skipped (in-process inFlight set).
func (o *Orchestrator) Enqueue(jobID uuid.UUID) {
	if o.Sync {
		o.Drive(context.Background(), jobID)
		return
	}
	o.mu.Lock()
	if o.inFlight[jobID] {
		o.mu.Unlock()
		return
	}
	o.inFlight[jobID] = true
	o.mu.Unlock()
	select {
	case o.queue <- jobID:
	default:
		o.mu.Lock()
		delete(o.inFlight, jobID)
		o.mu.Unlock()
	}
}

// Start launches the worker pool and the requeue scanner. It returns
// immediately; workers stop when ctx is cancelled.
func (o *Orchestrator) Start(ctx context.Context, workers int, scanInterval time.Duration) {
	if workers <= 0 {
		workers = 2
	}
	if scanInterval <= 0 {
		scanInterval = 3 * time.Second
	}
	for i := 0; i < workers; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case id := <-o.queue:
					o.Drive(ctx, id)
					o.mu.Lock()
					delete(o.inFlight, id)
					o.mu.Unlock()
				}
			}
		}()
	}
	go func() {
		t := time.NewTicker(scanInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				o.scan(ctx)
			}
		}
	}()
}

// scan requeues jobs sitting in submitted/queued (crash recovery, quota
// resume). It respects the quota pause window.
func (o *Orchestrator) scan(ctx context.Context) {
	o.mu.Lock()
	paused := time.Now().Before(o.pauseUntil)
	o.mu.Unlock()
	if paused {
		return
	}
	jobs, err := o.Tools.Stores.Jobs.ListJobsByStatus(ctx, domain.StatusSubmitted, domain.StatusQueued)
	if err != nil {
		o.Log.Error("requeue scan failed", "error", err)
		return
	}
	for _, j := range jobs {
		o.Enqueue(j.JobID) // skips IDs already queued or processing
	}
}

// pauseQueue pauses pickup of queued jobs after quota exhaustion.
func (o *Orchestrator) pauseQueue(d time.Duration) {
	o.mu.Lock()
	o.pauseUntil = time.Now().Add(d)
	o.mu.Unlock()
}

func (o *Orchestrator) beginDrive(jobID uuid.UUID) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.running[jobID] {
		return false
	}
	o.running[jobID] = true
	return true
}

func (o *Orchestrator) finishDrive(jobID uuid.UUID) {
	o.mu.Lock()
	delete(o.running, jobID)
	o.mu.Unlock()
}

// Drive advances one job until it blocks: needs_user_action, in_review,
// queued-after-quota, or a terminal state.
func (o *Orchestrator) Drive(ctx context.Context, jobID uuid.UUID) {
	if !o.beginDrive(jobID) {
		return
	}
	defer o.finishDrive(jobID)
	for i := 0; i < 32; i++ { // hard bound against transition loops
		job, err := o.Tools.Stores.Jobs.GetJob(ctx, jobID)
		if err != nil {
			o.Log.Error("drive: load job", "job_id", jobID, "error", err)
			return
		}
		var proceed bool
		switch job.Status {
		case domain.StatusSubmitted:
			proceed = o.setStatus(ctx, job, domain.StatusQueued)
		case domain.StatusQueued:
			proceed = o.setStatus(ctx, job, domain.StatusValidating)
		case domain.StatusValidating:
			proceed = o.stepValidate(ctx, job)
		case domain.StatusMetadataExtracted:
			proceed = o.stepRouteAfterMetadata(ctx, job)
		case domain.StatusCaptionChecked:
			proceed = o.stepCaptionRoute(ctx, job)
		case domain.StatusExtractingAudio:
			proceed = o.stepExtractAudio(ctx, job)
		case domain.StatusTranscribing:
			proceed = o.stepTranscribe(ctx, job)
		case domain.StatusNormalizing:
			proceed = o.stepNormalize(ctx, job)
		case domain.StatusQualityChecking:
			proceed = o.stepQualityCheck(ctx, job)
		case domain.StatusDrafted:
			proceed = o.setStatus(ctx, job, domain.StatusInReview)
		default:
			// in_review, approved, exported, needs_user_action, failed,
			// cancelled: nothing for the pipeline worker to do.
			return
		}
		if !proceed {
			return
		}
	}
	o.Log.Error("drive: transition bound exceeded", "job_id", jobID)
}

// setStatus applies and audits a status transition through the store's
// compare-and-swap primitive (audit C1/C2): the swap only succeeds when the
// job is still in the status this worker saw, so a user cancel (or another
// worker's claim) always wins and is never overwritten. The worker's local
// bookkeeping mutations (last_error, action_required, caption fields, ...)
// are carried into the swap. Returns false when the job moved concurrently
// (drop the step) or the transition is illegal (a bug, logged loudly).
func (o *Orchestrator) setStatus(ctx context.Context, job *domain.Job, to domain.Status) bool {
	from := job.Status
	updated, err := o.Tools.Stores.Jobs.TransitionJob(ctx, job.JobID, from, func(j *domain.Job) error {
		j.JobConfigID = job.JobConfigID
		j.DurationSeconds = job.DurationSeconds
		j.ActionRequired = job.ActionRequired
		j.LastError = job.LastError
		j.CaptionsAvailable = job.CaptionsAvailable
		j.CaptionTrackID = job.CaptionTrackID
		j.CaptionReuse = job.CaptionReuse
		return state.Transition(j, to)
	})
	if err != nil {
		switch domain.CodeOf(err) {
		case domain.CodeStatusConflict:
			o.Log.Info("status transition lost race; dropping step",
				"job_id", job.JobID, "from", from, "to", to)
		case domain.CodeInvalidStateTransition:
			o.Log.Error("illegal transition attempted", "job_id", job.JobID, "from", from, "to", to, "error", err)
		default:
			o.Log.Error("persist status", "job_id", job.JobID, "error", err)
		}
		return false
	}
	*job = *updated
	if err := o.Tools.Audit(ctx, &job.JobID, audit.ActorSystem, "orchestrator", "job.status_changed",
		map[string]any{"from": string(from), "to": string(to)}); err != nil {
		o.Log.Error("audit status transition", "job_id", job.JobID, "from", from, "to", to, "error", err)
		return false
	}
	return true
}

// toNeedsUserAction parks the job for user correction (PRD R9).
func (o *Orchestrator) toNeedsUserAction(ctx context.Context, job *domain.Job, action string, err error) {
	de := domain.AsError(err)
	job.ActionRequired = action
	job.LastError = &domain.ErrorInfo{Code: de.Code, Message: de.Message}
	if !o.setStatus(ctx, job, domain.StatusNeedsUserAction) {
		return
	}
	o.Tools.Audit(ctx, &job.JobID, audit.ActorSystem, "orchestrator", "job.needs_user_action",
		map[string]any{"action_required": action, "error_code": de.Code, "message": de.Message})
}

// toFailed marks the job terminally failed (PRD R9: terminal only).
func (o *Orchestrator) toFailed(ctx context.Context, job *domain.Job, err error) {
	de := domain.AsError(err)
	job.ActionRequired = ""
	job.LastError = &domain.ErrorInfo{Code: de.Code, Message: de.Message}
	if !o.setStatus(ctx, job, domain.StatusFailed) {
		return
	}
	o.Tools.Audit(ctx, &job.JobID, audit.ActorSystem, "orchestrator", "job.failed",
		map[string]any{"error_code": de.Code, "message": de.Message})
}

func (o *Orchestrator) clearError(ctx context.Context, job *domain.Job) {
	if job.LastError == nil && job.ActionRequired == "" {
		return
	}
	updated, err := o.Tools.Stores.Jobs.TransitionJob(ctx, job.JobID, job.Status, func(j *domain.Job) error {
		j.LastError = nil
		j.ActionRequired = ""
		return nil
	})
	if err != nil {
		if domain.CodeOf(err) != domain.CodeStatusConflict {
			o.Log.Error("clear job error", "job_id", job.JobID, "error", err)
		}
		return
	}
	*job = *updated
}

// --- pipeline steps ------------------------------------------------------

// stepValidate creates the job_config snapshot (PRD 13.2 rule 7) and extracts
// media metadata. METADATA_TIMEOUT retries once (PRD 14.2); correctable input
// problems park in needs_user_action/replace_media.
func (o *Orchestrator) stepValidate(ctx context.Context, job *domain.Job) bool {
	if job.JobConfigID == nil {
		if _, err := o.Tools.CreateConfigSnapshot(ctx, job, "system:orchestrator"); err != nil {
			o.toFailed(ctx, job, err)
			return false
		}
	}
	_, err := o.Tools.GetMediaMetadata(ctx, job)
	if err != nil && domain.CodeOf(err) == domain.CodeMetadataTimeout {
		time.Sleep(o.Backoff) // retry once with backoff
		_, err = o.Tools.GetMediaMetadata(ctx, job)
	}
	if err != nil {
		switch domain.CodeOf(err) {
		case domain.CodeNoAudioTrack, domain.CodeUnsupportedFormat, domain.CodeMediaNotFound:
			// User can replace the media (PRD 19).
			o.toNeedsUserAction(ctx, job, domain.ActionReplaceMedia, err)
		default:
			o.toFailed(ctx, job, err)
		}
		return false
	}
	o.clearError(ctx, job)
	return o.setStatus(ctx, job, domain.StatusMetadataExtracted)
}

// stepRouteAfterMetadata runs the caption pre-check for YouTube jobs (PRD R2)
// or goes straight to audio extraction for uploads.
func (o *Orchestrator) stepRouteAfterMetadata(ctx context.Context, job *domain.Job) bool {
	if job.SourceType != domain.SourceYouTube {
		return o.setStatus(ctx, job, domain.StatusExtractingAudio)
	}
	_, err := o.Tools.CheckYouTubeCaptions(ctx, job)
	if err != nil && domain.CodeOf(err) == domain.CodeCaptionAPIUnavailable {
		time.Sleep(o.Backoff) // retry once (PRD 14.3)
		_, err = o.Tools.CheckYouTubeCaptions(ctx, job)
	}
	if err != nil {
		// Caption check failed or is unauthorized/unconfigured: transcription
		// can continue with a warning (PRD 19 "Caption check unavailable").
		o.Tools.Audit(ctx, &job.JobID, audit.ActorSystem, "orchestrator", "caption_check.skipped",
			map[string]any{"error_code": domain.CodeOf(err), "message": err.Error()})
		// setStatus carries the cleared caption flags into the CAS.
		job.CaptionsAvailable = false
		job.CaptionTrackID = ""
	}
	return o.setStatus(ctx, job, domain.StatusCaptionChecked)
}

// stepCaptionRoute implements PRD 11.1 steps 6-7: pause for the producer's
// reuse decision, then either parse captions into a raw transcript (skipping
// STT) or fall through to audio extraction.
func (o *Orchestrator) stepCaptionRoute(ctx context.Context, job *domain.Job) bool {
	if job.CaptionsAvailable && job.CaptionReuse == nil {
		// Reusable official captions exist: producer must choose reuse vs
		// fresh transcription (PRD R2 acceptance).
		job.LastError = nil
		job.ActionRequired = domain.ActionCaptionDecision
		if !o.setStatus(ctx, job, domain.StatusNeedsUserAction) {
			return false
		}
		o.Tools.Audit(ctx, &job.JobID, audit.ActorSystem, "orchestrator", "job.needs_user_action",
			map[string]any{"action_required": domain.ActionCaptionDecision})
		return false
	}
	if job.CaptionsAvailable && job.CaptionReuse != nil && *job.CaptionReuse {
		cfg, err := o.Tools.Config(ctx, job)
		if err != nil {
			o.toFailed(ctx, job, err)
			return false
		}
		uri, err := o.Tools.FetchExistingCaptions(ctx, job, job.CaptionTrackID)
		if err == nil {
			_, _, err = o.Tools.ParseCaptionsToTranscript(ctx, job, uri, cfg)
		}
		if err != nil {
			// Caption fetch/parse failures fall back to transcription
			// (PRD 14.4/14.5 failure handling).
			o.Tools.Audit(ctx, &job.JobID, audit.ActorSystem, "orchestrator", "caption_reuse.fallback_to_transcription",
				map[string]any{"error_code": domain.CodeOf(err), "message": err.Error()})
			return o.setStatus(ctx, job, domain.StatusExtractingAudio)
		}
		// Caption path proceeds directly to normalization (PRD 11.1 step 7).
		return o.setStatus(ctx, job, domain.StatusNormalizing)
	}
	return o.setStatus(ctx, job, domain.StatusExtractingAudio)
}

// stepExtractAudio extracts normalized audio (PRD 14.6). EXTRACTION_FAILED
// and ARTIFACT_WRITE_FAILED retry once; NO_AUDIO_TRACK parks for replacement.
func (o *Orchestrator) stepExtractAudio(ctx context.Context, job *domain.Job) bool {
	_, err := o.Tools.ExtractAudio(ctx, job)
	if err != nil {
		switch domain.CodeOf(err) {
		case domain.CodeExtractionFailed, domain.CodeArtifactWriteFailed:
			time.Sleep(o.Backoff)
			_, err = o.Tools.ExtractAudio(ctx, job)
		}
	}
	if err != nil {
		if domain.CodeOf(err) == domain.CodeNoAudioTrack {
			o.toNeedsUserAction(ctx, job, domain.ActionReplaceMedia, err)
		} else {
			o.toFailed(ctx, job, err)
		}
		return false
	}
	return o.setStatus(ctx, job, domain.StatusTranscribing)
}

// stepTranscribe runs batch STT (PRD 14.7). Timeout retries once with
// backoff then fails; quota exhaustion returns the job to queued and pauses
// the queue (PRD 19).
func (o *Orchestrator) stepTranscribe(ctx context.Context, job *domain.Job) bool {
	cfg, err := o.Tools.Config(ctx, job)
	if err != nil {
		o.toFailed(ctx, job, err)
		return false
	}
	audioURI, err := o.Tools.LatestAudioArtifactURI(ctx, job.JobID)
	if err != nil {
		o.toFailed(ctx, job, err)
		return false
	}
	_, err = o.Tools.TranscribeAudio(ctx, job, audioURI, cfg)
	if err != nil && domain.CodeOf(err) == domain.CodeSTTProviderTimeout {
		time.Sleep(o.Backoff) // retry with backoff (PRD 19)
		_, err = o.Tools.TranscribeAudio(ctx, job, audioURI, cfg)
	}
	if err != nil {
		switch domain.CodeOf(err) {
		case domain.CodeSTTProviderQuotaExceeded:
			// Return to queued, pause queue, alert admin (PRD 14.7/19).
			de := domain.AsError(err)
			job.LastError = &domain.ErrorInfo{Code: de.Code, Message: "Transcription is paused due to provider quota."}
			if o.setStatus(ctx, job, domain.StatusQueued) {
				o.pauseQueue(5 * time.Minute)
				o.Log.Error("ALERT: STT provider quota exceeded; queue paused", "job_id", job.JobID)
				o.Tools.Audit(ctx, &job.JobID, audit.ActorSystem, "orchestrator", "queue.paused_quota",
					map[string]any{"error_code": de.Code})
			}
		default:
			o.toFailed(ctx, job, err)
		}
		return false
	}
	o.clearError(ctx, job)
	return o.setStatus(ctx, job, domain.StatusNormalizing)
}

// stepNormalize creates the clean transcript from the latest raw version
// (PRD 14.8). LLM_OUTPUT_INVALID retries once; STYLE_POLICY_NOT_FOUND is
// terminal in MVP (no config-edit endpoint exists to correct the snapshot).
func (o *Orchestrator) stepNormalize(ctx context.Context, job *domain.Job) bool {
	cfg, err := o.Tools.Config(ctx, job)
	if err != nil {
		o.toFailed(ctx, job, err)
		return false
	}
	raw, err := o.Tools.Stores.Transcripts.LatestVersion(ctx, job.JobID, domain.VersionRaw)
	if err != nil || raw == nil {
		o.toFailed(ctx, job, domain.E(domain.CodeTranscriptNotFound, "no raw transcript version for job %s", job.JobID))
		return false
	}
	_, _, err = o.Tools.NormalizeTranscript(ctx, job, raw, cfg)
	if err != nil && domain.CodeOf(err) == domain.CodeLLMOutputInvalid {
		time.Sleep(o.Backoff) // retry once (PRD 14.8)
		_, _, err = o.Tools.NormalizeTranscript(ctx, job, raw, cfg)
	}
	if err != nil {
		o.toFailed(ctx, job, err)
		return false
	}
	return o.setStatus(ctx, job, domain.StatusQualityChecking)
}

// stepQualityCheck runs quality checks on the clean version (raw as
// fallback). Per PRD 14.9, a failed quality check does not block review — the
// job proceeds to drafted with a warning.
func (o *Orchestrator) stepQualityCheck(ctx context.Context, job *domain.Job) bool {
	cfg, err := o.Tools.Config(ctx, job)
	if err != nil {
		o.toFailed(ctx, job, err)
		return false
	}
	version, err := o.Tools.Stores.Transcripts.LatestVersion(ctx, job.JobID, domain.VersionClean)
	if err == nil && version == nil {
		version, err = o.Tools.Stores.Transcripts.LatestVersion(ctx, job.JobID, domain.VersionRaw)
	}
	if err != nil || version == nil {
		o.toFailed(ctx, job, domain.E(domain.CodeTranscriptNotFound, "no transcript version to quality-check for job %s", job.JobID))
		return false
	}
	if _, qerr := o.Tools.QualityCheckTranscript(ctx, job, version, cfg); qerr != nil {
		// Job can still enter review with warning (PRD 14.9). setStatus carries
		// the warning into the CAS below.
		de := domain.AsError(qerr)
		job.LastError = &domain.ErrorInfo{Code: domain.CodeQualityCheckFailed, Message: de.Message}
		o.Tools.Audit(ctx, &job.JobID, audit.ActorSystem, "orchestrator", "quality_check.warning",
			map[string]any{"error_code": de.Code, "message": de.Message})
	}
	return o.setStatus(ctx, job, domain.StatusDrafted)
}

// ResumeAfterCaptionDecision records the producer's caption decision and
// resumes the pipeline (PRD 11.1 step 6). Called from the API layer.
func (o *Orchestrator) ResumeAfterCaptionDecision(ctx context.Context, job *domain.Job, reuse bool, decidedBy string) error {
	if job.Status != domain.StatusNeedsUserAction || job.ActionRequired != domain.ActionCaptionDecision {
		return domain.E(domain.CodeJobNotInActionableState,
			"caption decision applies only in needs_user_action/caption_decision; job is %s/%s", job.Status, job.ActionRequired)
	}
	updated, err := o.Tools.Stores.Jobs.TransitionJob(ctx, job.JobID, domain.StatusNeedsUserAction, func(j *domain.Job) error {
		if j.ActionRequired != domain.ActionCaptionDecision {
			return domain.E(domain.CodeJobNotInActionableState,
				"caption decision applies only in needs_user_action/caption_decision; job is %s/%s", j.Status, j.ActionRequired)
		}
		j.CaptionReuse = &reuse
		j.ActionRequired = ""
		j.LastError = nil
		return state.Transition(j, domain.StatusCaptionChecked)
	})
	if err != nil {
		return err
	}
	*job = *updated
	if err := o.Tools.Audit(ctx, &job.JobID, audit.ActorUser, decidedBy, "job.caption_decision_recorded",
		map[string]any{"reuse_captions": reuse}); err != nil {
		return err
	}
	o.Enqueue(job.JobID)
	return nil
}

// helper used by tests/demos to detect mock fault-injection markers.
func HasMarker(uri, marker string) bool { return strings.Contains(uri, marker) }
