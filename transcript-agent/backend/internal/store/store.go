// Package store defines the persistence interfaces. Two implementations
// exist: memory (default / dev / tests) and postgres (pgx, used when
// DATABASE_URL is set).
package store

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/aaraminds/transcript-agent/internal/domain"
)

// JobStore persists jobs and job_config snapshots.
type JobStore interface {
	CreateJob(ctx context.Context, j *domain.Job) error
	GetJob(ctx context.Context, id uuid.UUID) (*domain.Job, error)
	UpdateJob(ctx context.Context, j *domain.Job) error
	ListJobs(ctx context.Context) ([]*domain.Job, error)
	ListJobsByStatus(ctx context.Context, statuses ...domain.Status) ([]*domain.Job, error)

	// TransitionJob is the compare-and-swap primitive every status change goes
	// through: it atomically verifies the job's current status equals from,
	// applies apply (which may mutate fields and/or change the status), and
	// persists the result. When the current status differs from from it
	// returns domain.ErrStatusConflict and persists nothing. apply errors
	// abort the swap.
	TransitionJob(ctx context.Context, jobID uuid.UUID, from domain.Status, apply func(*domain.Job) error) (*domain.Job, error)

	CreateJobConfig(ctx context.Context, c *domain.JobConfig) error
	GetJobConfig(ctx context.Context, id uuid.UUID) (*domain.JobConfig, error)
}

// TranscriptStore persists transcript versions and segments.
type TranscriptStore interface {
	CreateVersion(ctx context.Context, v *domain.TranscriptVersion, segments []*domain.Segment) error
	GetVersion(ctx context.Context, id uuid.UUID) (*domain.TranscriptVersion, error)
	ListVersions(ctx context.Context, jobID uuid.UUID) ([]*domain.TranscriptVersion, error)
	// LatestVersion returns the most recent version of the given type for the
	// job, or nil if none exists.
	LatestVersion(ctx context.Context, jobID uuid.UUID, versionType string) (*domain.TranscriptVersion, error)
	ListSegments(ctx context.Context, versionID uuid.UUID) ([]*domain.Segment, error)
	GetSegment(ctx context.Context, segmentID uuid.UUID) (*domain.Segment, error)
	UpdateSegment(ctx context.Context, s *domain.Segment) error
}

// SummaryStore persists summaries.
type SummaryStore interface {
	CreateSummary(ctx context.Context, s *domain.Summary) error
	GetSummary(ctx context.Context, id uuid.UUID) (*domain.Summary, error)
	LatestSummaryByJob(ctx context.Context, jobID uuid.UUID) (*domain.Summary, error)
	UpdateSummary(ctx context.Context, s *domain.Summary) error
}

// QualityStore persists quality reports.
type QualityStore interface {
	CreateReport(ctx context.Context, r *domain.QualityReport) error
	LatestReportByJob(ctx context.Context, jobID uuid.UUID) (*domain.QualityReport, error)
}

// ApprovalStore persists approvals, including the supersede chain (11.4).
type ApprovalStore interface {
	CreateApproval(ctx context.Context, a *domain.Approval) error
	GetApproval(ctx context.Context, id uuid.UUID) (*domain.Approval, error)
	ListApprovalsByJob(ctx context.Context, jobID uuid.UUID) ([]*domain.Approval, error)
	// CurrentApproval returns the latest approval with no superseding approval.
	CurrentApproval(ctx context.Context, jobID uuid.UUID) (*domain.Approval, error)
	UpdateApproval(ctx context.Context, a *domain.Approval) error
}

// AuditStore is append-only (PRD 13.1).
type AuditStore interface {
	Append(ctx context.Context, e *domain.AuditEvent) error
	ListByJob(ctx context.Context, jobID uuid.UUID) ([]*domain.AuditEvent, error)
}

// ArtifactStore persists media artifact records and export records.
type ArtifactStore interface {
	CreateArtifact(ctx context.Context, a *domain.MediaArtifact) error
	GetArtifact(ctx context.Context, id uuid.UUID) (*domain.MediaArtifact, error)
	ListArtifactsByJob(ctx context.Context, jobID uuid.UUID, artifactType string) ([]*domain.MediaArtifact, error)
	MarkArtifactsSuperseded(ctx context.Context, jobID uuid.UUID) error
	// ListExpiredArtifacts returns up to limit artifacts whose retention_until
	// is set and older than cutoff (retention sweep, PRD 16.4/R3). Exports are
	// never returned (they carry no retention_until by construction).
	ListExpiredArtifacts(ctx context.Context, cutoff time.Time, limit int) ([]*domain.MediaArtifact, error)
	// DeleteArtifact removes the artifact row after its bytes were deleted.
	DeleteArtifact(ctx context.Context, id uuid.UUID) error

	CreateExport(ctx context.Context, e *domain.ExportRecord) error
	GetExport(ctx context.Context, id uuid.UUID) (*domain.ExportRecord, error)
	ListExportsByJob(ctx context.Context, jobID uuid.UUID) ([]*domain.ExportRecord, error)
}

// ApproveJobParams carries the pre-built rows for the atomic approve
// operation. The store persists them only if the job CAS succeeds.
type ApproveJobParams struct {
	JobID uuid.UUID
	// ApprovedVersion is the immutable approved clone plus its segments.
	ApprovedVersion *domain.TranscriptVersion
	Segments        []*domain.Segment
	// Approval is the new approval row (SupersededByApprovalID must be nil).
	Approval *domain.Approval
}

// ReopenJobParams carries the pre-built reviewed clone for the atomic reopen
// operation (PRD 11.4 post-approval correction).
type ReopenJobParams struct {
	JobID uuid.UUID
	// ReviewedVersion is the mutable reviewed clone plus its segments.
	ReviewedVersion *domain.TranscriptVersion
	Segments        []*domain.Segment
}

// ReviewTxStore bundles the multi-row review-lifecycle operations that must
// be atomic (audit H1/M7): approve and reopen. Both implementations execute
// each operation as a single unit — one lock hold (memory) or one pgx.Tx
// (postgres) — so a concurrent double-approve yields exactly one approval.
type ReviewTxStore interface {
	// ApproveJob atomically: CAS job in_review -> approved (clearing
	// last_error/action_required), inserts the approved version + segments,
	// inserts the approval row, marks every prior current approval for the
	// job superseded by the new approval, and marks every prior export record
	// superseded (PRD 13.2 r5) — all under the same lock/transaction. Returns
	// the updated job and the IDs of superseded approvals.
	// domain.ErrStatusConflict when the job is not in_review anymore.
	ApproveJob(ctx context.Context, p ApproveJobParams) (*domain.Job, []uuid.UUID, error)
	// ReopenJob atomically: CAS job approved|exported -> in_review and inserts
	// the fresh reviewed version + segments. domain.ErrStatusConflict when the
	// job left approved/exported concurrently.
	ReopenJob(ctx context.Context, p ReopenJobParams) (*domain.Job, error)
}

// Stores bundles all persistence interfaces for wiring.
type Stores struct {
	Jobs        JobStore
	Transcripts TranscriptStore
	Summaries   SummaryStore
	Quality     QualityStore
	Approvals   ApprovalStore
	Audit       AuditStore
	Artifacts   ArtifactStore
	Review      ReviewTxStore
}
