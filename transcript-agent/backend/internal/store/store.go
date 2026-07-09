// Package store defines the persistence interfaces. Two implementations
// exist: memory (default / dev / tests) and postgres (pgx, used when
// DATABASE_URL is set).
package store

import (
	"context"

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
	ListArtifactsByJob(ctx context.Context, jobID uuid.UUID, artifactType string) ([]*domain.MediaArtifact, error)
	MarkArtifactsSuperseded(ctx context.Context, jobID uuid.UUID) error

	CreateExport(ctx context.Context, e *domain.ExportRecord) error
	GetExport(ctx context.Context, id uuid.UUID) (*domain.ExportRecord, error)
	ListExportsByJob(ctx context.Context, jobID uuid.UUID) ([]*domain.ExportRecord, error)
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
}
