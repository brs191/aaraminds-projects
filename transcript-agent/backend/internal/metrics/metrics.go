// Package metrics exposes the PRD 18.2 operational counters via the stdlib
// expvar registry, served at GET /debug/vars. Stdlib-only by design (no
// Prometheus client in the MVP); counters are process-lifetime monotonic.
package metrics

import "expvar"

var (
	// JobsSubmitted counts accepted submit_media_job calls.
	JobsSubmitted = expvar.NewInt("jobs_submitted")
	// JobsCompleted counts jobs that reached in_review (pipeline complete).
	JobsCompleted = expvar.NewInt("jobs_completed")
	// JobsFailed counts terminal pipeline failures.
	JobsFailed = expvar.NewInt("jobs_failed_total")
	// ToolFailures counts tool contract failures keyed by tool name.
	ToolFailures = expvar.NewMap("tool_failures_total")
	// Retries counts single-retry attempts after retryable tool failures.
	Retries = expvar.NewInt("retries_total")
	// ExportValidationFailures counts exports whose parse-back validation failed.
	ExportValidationFailures = expvar.NewInt("export_validation_failures")
	// AuditWriteFailures counts failed audit appends (control failures, PRD 19).
	AuditWriteFailures = expvar.NewInt("audit_write_failures")
	// STTSecondsProcessed accumulates media seconds successfully transcribed.
	STTSecondsProcessed = expvar.NewInt("stt_seconds_processed")
	// StuckJobsReclaimed counts mid-pipeline jobs the scanner CAS'd back to
	// queued (PRD 18.4/18.5).
	StuckJobsReclaimed = expvar.NewInt("stuck_jobs_reclaimed_total")
	// ArtifactsSwept counts media artifacts deleted by the retention sweep
	// (PRD 16.4, R3).
	ArtifactsSwept = expvar.NewInt("artifacts_swept_total")
)
