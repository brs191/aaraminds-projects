// Package postgres implements the store interfaces on PostgreSQL via pgx/v5.
// Selected with STORAGE=postgres DATABASE_URL=... ; the schema comes from the
// files in backend/migrations applied by Migrate (schema_migrations table).
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aaraminds/transcript-agent/internal/domain"
	"github.com/aaraminds/transcript-agent/internal/store"
)

// Store implements every interface in package store on a pgx pool.
type Store struct {
	pool *pgxpool.Pool
}

// New returns a Postgres-backed store.
func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Stores returns the store.Stores bundle backed by this instance.
func (s *Store) Stores() store.Stores {
	return store.Stores{
		Jobs: s, Transcripts: s, Summaries: s, Quality: s,
		Approvals: s, Audit: s, Artifacts: s, Review: s,
		Library: s, Search: s,
	}
}

func notFound(code, format string, args ...any) error {
	return domain.E(code, format, args...)
}

// --- JobStore --------------------------------------------------------------

const jobColumns = `job_id, source_type, source_uri, status, submitted_by,
	ownership_attested, language, job_config_id, duration_seconds,
	action_required, last_error_code, last_error_message,
	captions_available, caption_track_id, caption_reuse, cancel_reason,
	library_mode, source_basis, created_at, updated_at`

func scanJob(row pgx.Row) (*domain.Job, error) {
	var (
		j           domain.Job
		status      string
		errCode     string
		errMsg      string
		jobConfigID *uuid.UUID
		reuse       *bool
	)
	err := row.Scan(&j.JobID, &j.SourceType, &j.SourceURI, &status, &j.SubmittedBy,
		&j.OwnershipAttested, &j.Language, &jobConfigID, &j.DurationSeconds,
		&j.ActionRequired, &errCode, &errMsg,
		&j.CaptionsAvailable, &j.CaptionTrackID, &reuse, &j.CancelReason,
		&j.LibraryMode, &j.SourceBasis, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		return nil, err
	}
	j.Status = domain.Status(status)
	j.JobConfigID = jobConfigID
	j.CaptionReuse = reuse
	if errCode != "" {
		j.LastError = &domain.ErrorInfo{Code: errCode, Message: errMsg}
	}
	return &j, nil
}

func jobArgs(j *domain.Job) []any {
	errCode, errMsg := "", ""
	if j.LastError != nil {
		errCode, errMsg = j.LastError.Code, j.LastError.Message
	}
	return []any{
		j.JobID, j.SourceType, j.SourceURI, string(j.Status), j.SubmittedBy,
		j.OwnershipAttested, j.Language, j.JobConfigID, j.DurationSeconds,
		j.ActionRequired, errCode, errMsg,
		j.CaptionsAvailable, j.CaptionTrackID, j.CaptionReuse, j.CancelReason,
		j.LibraryMode, j.SourceBasis, j.CreatedAt, j.UpdatedAt,
	}
}

func (s *Store) CreateJob(ctx context.Context, j *domain.Job) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO jobs (`+jobColumns+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)`,
		jobArgs(j)...)
	return err
}

func (s *Store) GetJob(ctx context.Context, id uuid.UUID) (*domain.Job, error) {
	j, err := scanJob(s.pool.QueryRow(ctx,
		`SELECT `+jobColumns+` FROM jobs WHERE job_id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound(domain.CodeJobNotFound, "job %s not found", id)
	}
	return j, err
}

// updateJobSQL deliberately never touches created_at (audit L2): the insert
// owns it, updates must not rewrite it.
const updateJobSQL = `
	UPDATE jobs SET source_type=$2, source_uri=$3, status=$4, submitted_by=$5,
		ownership_attested=$6, language=$7, job_config_id=$8, duration_seconds=$9,
		action_required=$10, last_error_code=$11, last_error_message=$12,
		captions_available=$13, caption_track_id=$14, caption_reuse=$15,
		cancel_reason=$16, library_mode=$17, source_basis=$18, updated_at=$19
	WHERE job_id=$1`

func updateJobArgs(j *domain.Job) []any {
	errCode, errMsg := "", ""
	if j.LastError != nil {
		errCode, errMsg = j.LastError.Code, j.LastError.Message
	}
	return []any{
		j.JobID, j.SourceType, j.SourceURI, string(j.Status), j.SubmittedBy,
		j.OwnershipAttested, j.Language, j.JobConfigID, j.DurationSeconds,
		j.ActionRequired, errCode, errMsg,
		j.CaptionsAvailable, j.CaptionTrackID, j.CaptionReuse, j.CancelReason,
		j.LibraryMode, j.SourceBasis, j.UpdatedAt,
	}
}

func (s *Store) UpdateJob(ctx context.Context, j *domain.Job) error {
	tag, err := s.pool.Exec(ctx, updateJobSQL, updateJobArgs(j)...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return notFound(domain.CodeJobNotFound, "job %s not found", j.JobID)
	}
	return nil
}

// TransitionJob implements the compare-and-swap status primitive with a
// SELECT ... FOR UPDATE inside one transaction: verify current status == from,
// apply, persist — atomically. Losing the race returns
// domain.ErrStatusConflict.
func (s *Store) TransitionJob(ctx context.Context, jobID uuid.UUID, from domain.Status, apply func(*domain.Job) error) (*domain.Job, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	j, err := scanJob(tx.QueryRow(ctx,
		`SELECT `+jobColumns+` FROM jobs WHERE job_id = $1 FOR UPDATE`, jobID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound(domain.CodeJobNotFound, "job %s not found", jobID)
	}
	if err != nil {
		return nil, err
	}
	if j.Status != from {
		return nil, domain.ErrStatusConflict
	}
	if err := apply(j); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, updateJobSQL, updateJobArgs(j)...); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return j, nil
}

func (s *Store) listJobs(ctx context.Context, where string, args ...any) ([]*domain.Job, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+jobColumns+` FROM jobs `+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Job
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func (s *Store) ListJobs(ctx context.Context) ([]*domain.Job, error) {
	return s.listJobs(ctx, `ORDER BY created_at DESC`)
}

func (s *Store) ListJobsByStatus(ctx context.Context, statuses ...domain.Status) ([]*domain.Job, error) {
	vals := make([]string, len(statuses))
	for i, st := range statuses {
		vals[i] = string(st)
	}
	return s.listJobs(ctx, `WHERE status = ANY($1) ORDER BY created_at ASC`, vals)
}

func (s *Store) CreateJobConfig(ctx context.Context, c *domain.JobConfig) error {
	var sttModel *string
	if c.STTModel != "" {
		sttModel = &c.STTModel
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO job_config (job_config_id, job_id, language, confidence_threshold,
			enable_diarization, expected_speaker_count, style_policy_id,
			summary_max_words, summary_style, stt_provider, stt_model,
			max_duration_seconds, created_by, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		c.JobConfigID, c.JobID, c.Language, c.ConfidenceThreshold,
		c.EnableDiarization, c.ExpectedSpeakerCount, c.StylePolicyID,
		c.SummaryMaxWords, c.SummaryStyle, c.STTProvider, sttModel,
		c.MaxDurationSeconds, c.CreatedBy, c.CreatedAt)
	return err
}

func (s *Store) GetJobConfig(ctx context.Context, id uuid.UUID) (*domain.JobConfig, error) {
	var (
		c        domain.JobConfig
		sttModel *string
	)
	err := s.pool.QueryRow(ctx, `
		SELECT job_config_id, job_id, language, confidence_threshold,
			enable_diarization, expected_speaker_count, style_policy_id,
			summary_max_words, summary_style, stt_provider, stt_model,
			max_duration_seconds, created_by, created_at
		FROM job_config WHERE job_config_id = $1`, id).Scan(
		&c.JobConfigID, &c.JobID, &c.Language, &c.ConfidenceThreshold,
		&c.EnableDiarization, &c.ExpectedSpeakerCount, &c.StylePolicyID,
		&c.SummaryMaxWords, &c.SummaryStyle, &c.STTProvider, &sttModel,
		&c.MaxDurationSeconds, &c.CreatedBy, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound(domain.CodeValidationError, "job_config %s not found", id)
	}
	if err != nil {
		return nil, err
	}
	if sttModel != nil {
		c.STTModel = *sttModel
	}
	return &c, nil
}

// --- TranscriptStore ---------------------------------------------------------

// insertVersionTx inserts a transcript version plus its segments inside tx.
func insertVersionTx(ctx context.Context, tx pgx.Tx, v *domain.TranscriptVersion, segments []*domain.Segment) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO transcript_versions (transcript_version_id, job_id, version_type,
			source_version_id, created_by, is_immutable, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		v.TranscriptVersionID, v.JobID, v.VersionType, v.SourceVersionID,
		v.CreatedBy, v.IsImmutable, v.CreatedAt); err != nil {
		return err
	}
	for _, sg := range segments {
		flags, err := json.Marshal(nonNilFlags(sg.Flags))
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO transcript_segments (segment_id, transcript_version_id,
				start_ms, end_ms, speaker_label, text, confidence, flags)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			sg.SegmentID, sg.TranscriptVersionID, sg.StartMS, sg.EndMS,
			sg.SpeakerLabel, sg.Text, sg.Confidence, flags); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) CreateVersion(ctx context.Context, v *domain.TranscriptVersion, segments []*domain.Segment) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := insertVersionTx(ctx, tx, v, segments); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func nonNilFlags(f map[string]bool) map[string]bool {
	if f == nil {
		return map[string]bool{}
	}
	return f
}

func scanVersion(row pgx.Row) (*domain.TranscriptVersion, error) {
	var v domain.TranscriptVersion
	err := row.Scan(&v.TranscriptVersionID, &v.JobID, &v.VersionType,
		&v.SourceVersionID, &v.CreatedBy, &v.IsImmutable, &v.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

const versionColumns = `transcript_version_id, job_id, version_type,
	source_version_id, created_by, is_immutable, created_at`

func (s *Store) GetVersion(ctx context.Context, id uuid.UUID) (*domain.TranscriptVersion, error) {
	v, err := scanVersion(s.pool.QueryRow(ctx,
		`SELECT `+versionColumns+` FROM transcript_versions WHERE transcript_version_id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound(domain.CodeTranscriptNotFound, "transcript version %s not found", id)
	}
	return v, err
}

func (s *Store) ListVersions(ctx context.Context, jobID uuid.UUID) ([]*domain.TranscriptVersion, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+versionColumns+` FROM transcript_versions WHERE job_id = $1 ORDER BY created_at ASC`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.TranscriptVersion
	for rows.Next() {
		v, err := scanVersion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) LatestVersion(ctx context.Context, jobID uuid.UUID, versionType string) (*domain.TranscriptVersion, error) {
	v, err := scanVersion(s.pool.QueryRow(ctx, `
		SELECT `+versionColumns+` FROM transcript_versions
		WHERE job_id = $1 AND version_type = $2
		ORDER BY created_at DESC LIMIT 1`, jobID, versionType))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return v, err
}

const segmentColumns = `segment_id, transcript_version_id, start_ms, end_ms,
	speaker_label, text, confidence, flags`

func scanSegment(row pgx.Row) (*domain.Segment, error) {
	var (
		sg    domain.Segment
		flags []byte
	)
	err := row.Scan(&sg.SegmentID, &sg.TranscriptVersionID, &sg.StartMS, &sg.EndMS,
		&sg.SpeakerLabel, &sg.Text, &sg.Confidence, &flags)
	if err != nil {
		return nil, err
	}
	if len(flags) > 0 {
		if err := json.Unmarshal(flags, &sg.Flags); err != nil {
			return nil, fmt.Errorf("decode segment flags: %w", err)
		}
	}
	return &sg, nil
}

func (s *Store) ListSegments(ctx context.Context, versionID uuid.UUID) ([]*domain.Segment, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+segmentColumns+` FROM transcript_segments
		 WHERE transcript_version_id = $1 ORDER BY start_ms ASC`, versionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Segment
	for rows.Next() {
		sg, err := scanSegment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sg)
	}
	return out, rows.Err()
}

func (s *Store) GetSegment(ctx context.Context, segmentID uuid.UUID) (*domain.Segment, error) {
	sg, err := scanSegment(s.pool.QueryRow(ctx,
		`SELECT `+segmentColumns+` FROM transcript_segments WHERE segment_id = $1`, segmentID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound(domain.CodeSegmentNotFound, "segment %s not found", segmentID)
	}
	return sg, err
}

func (s *Store) UpdateSegment(ctx context.Context, sg *domain.Segment) error {
	flags, err := json.Marshal(nonNilFlags(sg.Flags))
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE transcript_segments
		SET start_ms=$2, end_ms=$3, speaker_label=$4, text=$5, confidence=$6, flags=$7
		WHERE segment_id=$1`,
		sg.SegmentID, sg.StartMS, sg.EndMS, sg.SpeakerLabel, sg.Text, sg.Confidence, flags)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return notFound(domain.CodeSegmentNotFound, "segment %s not found", sg.SegmentID)
	}
	return nil
}

// --- SummaryStore -------------------------------------------------------------

const summaryColumns = `summary_id, job_id, source_transcript_version_id, text,
	validation_status, validation_notes, created_by, created_at, updated_at`

func scanSummary(row pgx.Row) (*domain.Summary, error) {
	var (
		sm    domain.Summary
		notes *string
	)
	err := row.Scan(&sm.SummaryID, &sm.JobID, &sm.SourceTranscriptVersionID, &sm.Text,
		&sm.ValidationStatus, &notes, &sm.CreatedBy, &sm.CreatedAt, &sm.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if notes != nil {
		sm.ValidationNotes = *notes
	}
	return &sm, nil
}

func (s *Store) CreateSummary(ctx context.Context, sm *domain.Summary) error {
	var notes *string
	if sm.ValidationNotes != "" {
		notes = &sm.ValidationNotes
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO summaries (`+summaryColumns+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		sm.SummaryID, sm.JobID, sm.SourceTranscriptVersionID, sm.Text,
		sm.ValidationStatus, notes, sm.CreatedBy, sm.CreatedAt, sm.UpdatedAt)
	return err
}

func (s *Store) GetSummary(ctx context.Context, id uuid.UUID) (*domain.Summary, error) {
	sm, err := scanSummary(s.pool.QueryRow(ctx,
		`SELECT `+summaryColumns+` FROM summaries WHERE summary_id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound(domain.CodeSummaryNotFound, "summary %s not found", id)
	}
	return sm, err
}

func (s *Store) LatestSummaryByJob(ctx context.Context, jobID uuid.UUID) (*domain.Summary, error) {
	sm, err := scanSummary(s.pool.QueryRow(ctx, `
		SELECT `+summaryColumns+` FROM summaries
		WHERE job_id = $1 ORDER BY created_at DESC LIMIT 1`, jobID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound(domain.CodeSummaryNotFound, "no summary for job %s", jobID)
	}
	return sm, err
}

func (s *Store) UpdateSummary(ctx context.Context, sm *domain.Summary) error {
	var notes *string
	if sm.ValidationNotes != "" {
		notes = &sm.ValidationNotes
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE summaries SET text=$2, validation_status=$3, validation_notes=$4, updated_at=$5
		WHERE summary_id=$1`,
		sm.SummaryID, sm.Text, sm.ValidationStatus, notes, sm.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return notFound(domain.CodeSummaryNotFound, "summary %s not found", sm.SummaryID)
	}
	return nil
}

// --- QualityStore --------------------------------------------------------------

func (s *Store) CreateReport(ctx context.Context, r *domain.QualityReport) error {
	issues := r.Issues
	if issues == nil {
		issues = []domain.QualityIssue{}
	}
	issueJSON, err := json.Marshal(issues)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO quality_reports (quality_report_id, job_id, transcript_version_id,
			job_config_id, confidence_threshold, quality_score, average_confidence,
			low_confidence_segment_count, coverage_gap_seconds, timestamp_gap_count,
			diarization_warning_count, confidence_unavailable, issue_summary_json, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		r.QualityReportID, r.JobID, r.TranscriptVersionID, r.JobConfigID,
		r.ConfidenceThreshold, r.QualityScore, r.AverageConfidence,
		r.LowConfidenceSegmentCount, r.CoverageGapSeconds, r.TimestampGapCount,
		r.DiarizationWarningCount, r.ConfidenceUnavailable, issueJSON, r.CreatedAt)
	return err
}

func (s *Store) LatestReportByJob(ctx context.Context, jobID uuid.UUID) (*domain.QualityReport, error) {
	var (
		r         domain.QualityReport
		issueJSON []byte
	)
	err := s.pool.QueryRow(ctx, `
		SELECT quality_report_id, job_id, transcript_version_id, job_config_id,
			confidence_threshold, quality_score, average_confidence,
			low_confidence_segment_count, coverage_gap_seconds, timestamp_gap_count,
			diarization_warning_count, confidence_unavailable, issue_summary_json, created_at
		FROM quality_reports WHERE job_id = $1 ORDER BY created_at DESC LIMIT 1`, jobID).Scan(
		&r.QualityReportID, &r.JobID, &r.TranscriptVersionID, &r.JobConfigID,
		&r.ConfidenceThreshold, &r.QualityScore, &r.AverageConfidence,
		&r.LowConfidenceSegmentCount, &r.CoverageGapSeconds, &r.TimestampGapCount,
		&r.DiarizationWarningCount, &r.ConfidenceUnavailable, &issueJSON, &r.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound(domain.CodeQualityReportNotFound, "no quality report for job %s", jobID)
	}
	if err != nil {
		return nil, err
	}
	if len(issueJSON) > 0 {
		if err := json.Unmarshal(issueJSON, &r.Issues); err != nil {
			return nil, fmt.Errorf("decode issue_summary_json: %w", err)
		}
	}
	return &r, nil
}

// --- ApprovalStore ---------------------------------------------------------------

const approvalColumns = `approval_id, job_id, approved_transcript_version_id,
	approved_by, approved_at, approval_note, superseded_by_approval_id`

func scanApproval(row pgx.Row) (*domain.Approval, error) {
	var (
		a    domain.Approval
		note *string
	)
	err := row.Scan(&a.ApprovalID, &a.JobID, &a.ApprovedTranscriptVersionID,
		&a.ApprovedBy, &a.ApprovedAt, &note, &a.SupersededByApprovalID)
	if err != nil {
		return nil, err
	}
	if note != nil {
		a.ApprovalNote = *note
	}
	return &a, nil
}

func (s *Store) CreateApproval(ctx context.Context, a *domain.Approval) error {
	var note *string
	if a.ApprovalNote != "" {
		note = &a.ApprovalNote
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO approvals (`+approvalColumns+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		a.ApprovalID, a.JobID, a.ApprovedTranscriptVersionID,
		a.ApprovedBy, a.ApprovedAt, note, a.SupersededByApprovalID)
	return err
}

func (s *Store) GetApproval(ctx context.Context, id uuid.UUID) (*domain.Approval, error) {
	a, err := scanApproval(s.pool.QueryRow(ctx,
		`SELECT `+approvalColumns+` FROM approvals WHERE approval_id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound(domain.CodeValidationError, "approval %s not found", id)
	}
	return a, err
}

func (s *Store) ListApprovalsByJob(ctx context.Context, jobID uuid.UUID) ([]*domain.Approval, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+approvalColumns+` FROM approvals WHERE job_id = $1 ORDER BY approved_at ASC`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Approval
	for rows.Next() {
		a, err := scanApproval(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) CurrentApproval(ctx context.Context, jobID uuid.UUID) (*domain.Approval, error) {
	a, err := scanApproval(s.pool.QueryRow(ctx, `
		SELECT `+approvalColumns+` FROM approvals
		WHERE job_id = $1 AND superseded_by_approval_id IS NULL
		ORDER BY approved_at DESC LIMIT 1`, jobID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return a, err
}

func (s *Store) UpdateApproval(ctx context.Context, a *domain.Approval) error {
	var note *string
	if a.ApprovalNote != "" {
		note = &a.ApprovalNote
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE approvals SET approved_transcript_version_id=$2, approved_by=$3,
			approved_at=$4, approval_note=$5, superseded_by_approval_id=$6
		WHERE approval_id=$1`,
		a.ApprovalID, a.ApprovedTranscriptVersionID, a.ApprovedBy,
		a.ApprovedAt, note, a.SupersededByApprovalID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return notFound(domain.CodeValidationError, "approval %s not found", a.ApprovalID)
	}
	return nil
}

// --- ReviewTxStore (atomic approve / reopen in one pgx.Tx) -----------------------

func insertApprovalTx(ctx context.Context, tx pgx.Tx, a *domain.Approval) error {
	var note *string
	if a.ApprovalNote != "" {
		note = &a.ApprovalNote
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO approvals (`+approvalColumns+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		a.ApprovalID, a.JobID, a.ApprovedTranscriptVersionID,
		a.ApprovedBy, a.ApprovedAt, note, a.SupersededByApprovalID)
	return err
}

func (s *Store) ApproveJob(ctx context.Context, p store.ApproveJobParams) (*domain.Job, []uuid.UUID, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	j, err := scanJob(tx.QueryRow(ctx,
		`SELECT `+jobColumns+` FROM jobs WHERE job_id = $1 FOR UPDATE`, p.JobID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, notFound(domain.CodeJobNotFound, "job %s not found", p.JobID)
	}
	if err != nil {
		return nil, nil, err
	}
	if j.Status != domain.StatusInReview {
		return nil, nil, domain.ErrStatusConflict
	}
	if err := insertVersionTx(ctx, tx, p.ApprovedVersion, p.Segments); err != nil {
		return nil, nil, err
	}
	if err := insertApprovalTx(ctx, tx, p.Approval); err != nil {
		return nil, nil, err
	}
	rows, err := tx.Query(ctx, `
		UPDATE approvals SET superseded_by_approval_id = $1
		WHERE job_id = $2 AND superseded_by_approval_id IS NULL AND approval_id <> $1
		RETURNING approval_id`, p.Approval.ApprovalID, p.JobID)
	if err != nil {
		return nil, nil, err
	}
	var superseded []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, nil, err
		}
		superseded = append(superseded, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	// Re-approval supersedes every prior export inside the same transaction
	// (PRD 13.2 r5): they stay downloadable but are flagged.
	if _, err := tx.Exec(ctx, `
		UPDATE exports SET superseded = TRUE
		WHERE job_id = $1 AND superseded = FALSE`, p.JobID); err != nil {
		return nil, nil, err
	}
	j.Status = domain.StatusApproved
	j.LastError = nil
	j.ActionRequired = ""
	j.UpdatedAt = time.Now().UTC()
	if _, err := tx.Exec(ctx, updateJobSQL, updateJobArgs(j)...); err != nil {
		return nil, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return j, superseded, nil
}

func (s *Store) ReopenJob(ctx context.Context, p store.ReopenJobParams) (*domain.Job, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	j, err := scanJob(tx.QueryRow(ctx,
		`SELECT `+jobColumns+` FROM jobs WHERE job_id = $1 FOR UPDATE`, p.JobID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound(domain.CodeJobNotFound, "job %s not found", p.JobID)
	}
	if err != nil {
		return nil, err
	}
	if j.Status != domain.StatusApproved && j.Status != domain.StatusExported {
		return nil, domain.ErrStatusConflict
	}
	if err := insertVersionTx(ctx, tx, p.ReviewedVersion, p.Segments); err != nil {
		return nil, err
	}
	j.Status = domain.StatusInReview
	j.UpdatedAt = time.Now().UTC()
	if _, err := tx.Exec(ctx, updateJobSQL, updateJobArgs(j)...); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return j, nil
}

// --- AuditStore (append-only: no update/delete methods exist) --------------------

func (s *Store) Append(ctx context.Context, e *domain.AuditEvent) error {
	payload := e.EventPayload
	if payload == nil {
		payload = map[string]any{}
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO audit_events (audit_event_id, job_id, actor_type, actor_id,
			event_type, event_payload, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		e.AuditEventID, e.JobID, e.ActorType, e.ActorID, e.EventType, payloadJSON, e.CreatedAt)
	return err
}

func (s *Store) ListByJob(ctx context.Context, jobID uuid.UUID) ([]*domain.AuditEvent, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT audit_event_id, job_id, actor_type, actor_id, event_type, event_payload, created_at
		FROM audit_events WHERE job_id = $1 ORDER BY created_at ASC`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.AuditEvent
	for rows.Next() {
		var (
			e       domain.AuditEvent
			payload []byte
		)
		if err := rows.Scan(&e.AuditEventID, &e.JobID, &e.ActorType, &e.ActorID,
			&e.EventType, &payload, &e.CreatedAt); err != nil {
			return nil, err
		}
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &e.EventPayload); err != nil {
				return nil, fmt.Errorf("decode event_payload: %w", err)
			}
		}
		out = append(out, &e)
	}
	return out, rows.Err()
}

// --- ArtifactStore -----------------------------------------------------------------

func (s *Store) CreateArtifact(ctx context.Context, a *domain.MediaArtifact) error {
	// Staged uploads exist before any job; store NULL for the zero job id
	// (migration 0007 made media_artifacts.job_id nullable).
	var jobID *uuid.UUID
	if a.JobID != uuid.Nil {
		jobID = &a.JobID
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO media_artifacts (artifact_id, job_id, artifact_type, uri,
			mime_type, size_bytes, superseded, retention_until, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		a.ArtifactID, jobID, a.ArtifactType, a.URI,
		a.MimeType, a.SizeBytes, a.Superseded, a.RetentionUntil, a.CreatedAt)
	return err
}

func scanArtifact(row pgx.Row) (*domain.MediaArtifact, error) {
	var (
		a     domain.MediaArtifact
		jobID *uuid.UUID
	)
	err := row.Scan(&a.ArtifactID, &jobID, &a.ArtifactType, &a.URI,
		&a.MimeType, &a.SizeBytes, &a.Superseded, &a.RetentionUntil, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	if jobID != nil {
		a.JobID = *jobID
	}
	return &a, nil
}

const artifactColumns = `artifact_id, job_id, artifact_type, uri, mime_type,
	size_bytes, superseded, retention_until, created_at`

func (s *Store) GetArtifact(ctx context.Context, id uuid.UUID) (*domain.MediaArtifact, error) {
	a, err := scanArtifact(s.pool.QueryRow(ctx,
		`SELECT `+artifactColumns+` FROM media_artifacts WHERE artifact_id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound(domain.CodeMediaNotFound, "artifact %s not found", id)
	}
	return a, err
}

func (s *Store) ListArtifactsByJob(ctx context.Context, jobID uuid.UUID, artifactType string) ([]*domain.MediaArtifact, error) {
	q := `SELECT ` + artifactColumns + ` FROM media_artifacts WHERE job_id = $1`
	args := []any{jobID}
	if artifactType != "" {
		q += ` AND artifact_type = $2`
		args = append(args, artifactType)
	}
	q += ` ORDER BY created_at ASC`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.MediaArtifact
	for rows.Next() {
		a, err := scanArtifact(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) MarkArtifactsSuperseded(ctx context.Context, jobID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE media_artifacts SET superseded = TRUE WHERE job_id = $1`, jobID)
	return err
}

func (s *Store) ListExpiredArtifacts(ctx context.Context, cutoff time.Time, limit int) ([]*domain.MediaArtifact, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
		SELECT `+artifactColumns+` FROM media_artifacts
		WHERE retention_until IS NOT NULL AND retention_until < $1
		ORDER BY retention_until ASC LIMIT $2`, cutoff, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.MediaArtifact
	for rows.Next() {
		a, err := scanArtifact(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) DeleteArtifact(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM media_artifacts WHERE artifact_id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return notFound(domain.CodeMediaNotFound, "artifact %s not found", id)
	}
	return nil
}

func (s *Store) CreateExport(ctx context.Context, e *domain.ExportRecord) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO exports (export_id, job_id, approved_transcript_version_id,
			format, artifact_uri, validation_status, superseded, created_by, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		e.ExportID, e.JobID, e.ApprovedTranscriptVersionID,
		e.Format, e.ArtifactURI, e.ValidationStatus, e.Superseded, e.CreatedBy, e.CreatedAt)
	return err
}

const exportColumns = `export_id, job_id, approved_transcript_version_id,
	format, artifact_uri, validation_status, superseded, created_by, created_at`

func scanExport(row pgx.Row) (*domain.ExportRecord, error) {
	var e domain.ExportRecord
	err := row.Scan(&e.ExportID, &e.JobID, &e.ApprovedTranscriptVersionID,
		&e.Format, &e.ArtifactURI, &e.ValidationStatus, &e.Superseded, &e.CreatedBy, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (s *Store) GetExport(ctx context.Context, id uuid.UUID) (*domain.ExportRecord, error) {
	e, err := scanExport(s.pool.QueryRow(ctx,
		`SELECT `+exportColumns+` FROM exports WHERE export_id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound(domain.CodeExportNotFound, "export %s not found", id)
	}
	return e, err
}

func (s *Store) ListExportsByJob(ctx context.Context, jobID uuid.UUID) ([]*domain.ExportRecord, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+exportColumns+` FROM exports WHERE job_id = $1 ORDER BY created_at ASC`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.ExportRecord
	for rows.Next() {
		e, err := scanExport(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

var _ store.JobStore = (*Store)(nil)
var _ store.ReviewTxStore = (*Store)(nil)
var _ store.TranscriptStore = (*Store)(nil)
var _ store.SummaryStore = (*Store)(nil)
var _ store.QualityStore = (*Store)(nil)
var _ store.ApprovalStore = (*Store)(nil)
var _ store.AuditStore = (*Store)(nil)
var _ store.ArtifactStore = (*Store)(nil)
