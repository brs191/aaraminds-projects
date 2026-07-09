// Package memory is the full in-memory store implementation. It is the
// default when DATABASE_URL is not set and is used by the test suite.
package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/aaraminds/transcript-agent/internal/domain"
	"github.com/aaraminds/transcript-agent/internal/store"
)

// Store implements every interface in package store with mutex-guarded maps.
type Store struct {
	mu sync.RWMutex

	jobs      map[uuid.UUID]*domain.Job
	jobOrder  []uuid.UUID
	configs   map[uuid.UUID]*domain.JobConfig
	versions  map[uuid.UUID]*domain.TranscriptVersion
	verOrder  []uuid.UUID
	segments  map[uuid.UUID]*domain.Segment
	segByVer  map[uuid.UUID][]uuid.UUID
	summaries map[uuid.UUID]*domain.Summary
	sumOrder  []uuid.UUID
	reports   map[uuid.UUID]*domain.QualityReport
	repOrder  []uuid.UUID
	approvals map[uuid.UUID]*domain.Approval
	apprOrder []uuid.UUID
	audits    []*domain.AuditEvent
	artifacts map[uuid.UUID]*domain.MediaArtifact
	artOrder  []uuid.UUID
	exports   map[uuid.UUID]*domain.ExportRecord
	expOrder  []uuid.UUID
}

// New returns an empty in-memory store.
func New() *Store {
	return &Store{
		jobs:      map[uuid.UUID]*domain.Job{},
		configs:   map[uuid.UUID]*domain.JobConfig{},
		versions:  map[uuid.UUID]*domain.TranscriptVersion{},
		segments:  map[uuid.UUID]*domain.Segment{},
		segByVer:  map[uuid.UUID][]uuid.UUID{},
		summaries: map[uuid.UUID]*domain.Summary{},
		reports:   map[uuid.UUID]*domain.QualityReport{},
		approvals: map[uuid.UUID]*domain.Approval{},
		artifacts: map[uuid.UUID]*domain.MediaArtifact{},
		exports:   map[uuid.UUID]*domain.ExportRecord{},
	}
}

// Stores returns a store.Stores bundle backed by this single instance.
func (s *Store) Stores() store.Stores {
	return store.Stores{
		Jobs: s, Transcripts: s, Summaries: s, Quality: s,
		Approvals: s, Audit: s, Artifacts: s, Review: s,
	}
}

// --- helpers -----------------------------------------------------------

func copyJob(j *domain.Job) *domain.Job {
	c := *j
	if j.LastError != nil {
		le := *j.LastError
		c.LastError = &le
	}
	if j.JobConfigID != nil {
		id := *j.JobConfigID
		c.JobConfigID = &id
	}
	if j.CaptionReuse != nil {
		b := *j.CaptionReuse
		c.CaptionReuse = &b
	}
	return &c
}

func copySegment(sg *domain.Segment) *domain.Segment {
	c := *sg
	if sg.Confidence != nil {
		v := *sg.Confidence
		c.Confidence = &v
	}
	if sg.Flags != nil {
		f := make(map[string]bool, len(sg.Flags))
		for k, v := range sg.Flags {
			f[k] = v
		}
		c.Flags = f
	}
	return &c
}

func copyVersion(v *domain.TranscriptVersion) *domain.TranscriptVersion {
	c := *v
	if v.SourceVersionID != nil {
		id := *v.SourceVersionID
		c.SourceVersionID = &id
	}
	return &c
}

func copyApproval(a *domain.Approval) *domain.Approval {
	c := *a
	if a.SupersededByApprovalID != nil {
		id := *a.SupersededByApprovalID
		c.SupersededByApprovalID = &id
	}
	return &c
}

// --- JobStore ----------------------------------------------------------

func (s *Store) CreateJob(_ context.Context, j *domain.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[j.JobID] = copyJob(j)
	s.jobOrder = append(s.jobOrder, j.JobID)
	return nil
}

func (s *Store) GetJob(_ context.Context, id uuid.UUID) (*domain.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[id]
	if !ok {
		return nil, domain.E(domain.CodeJobNotFound, "job %s not found", id)
	}
	return copyJob(j), nil
}

func (s *Store) UpdateJob(_ context.Context, j *domain.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.jobs[j.JobID]; !ok {
		return domain.E(domain.CodeJobNotFound, "job %s not found", j.JobID)
	}
	s.jobs[j.JobID] = copyJob(j)
	return nil
}

// TransitionJob implements the compare-and-swap status primitive under the
// store lock: verify current status == from, apply, persist — atomically.
func (s *Store) TransitionJob(_ context.Context, jobID uuid.UUID, from domain.Status, apply func(*domain.Job) error) (*domain.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.jobs[jobID]
	if !ok {
		return nil, domain.E(domain.CodeJobNotFound, "job %s not found", jobID)
	}
	if cur.Status != from {
		return nil, domain.ErrStatusConflict
	}
	next := copyJob(cur)
	if err := apply(next); err != nil {
		return nil, err
	}
	s.jobs[jobID] = copyJob(next)
	return next, nil
}

func (s *Store) ListJobs(_ context.Context) ([]*domain.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*domain.Job, 0, len(s.jobOrder))
	for _, id := range s.jobOrder {
		out = append(out, copyJob(s.jobs[id]))
	}
	// Newest first for API listing.
	sort.SliceStable(out, func(i, k int) bool { return out[i].CreatedAt.After(out[k].CreatedAt) })
	return out, nil
}

func (s *Store) ListJobsByStatus(_ context.Context, statuses ...domain.Status) ([]*domain.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	want := map[domain.Status]bool{}
	for _, st := range statuses {
		want[st] = true
	}
	var out []*domain.Job
	for _, id := range s.jobOrder { // oldest first for FIFO processing
		if want[s.jobs[id].Status] {
			out = append(out, copyJob(s.jobs[id]))
		}
	}
	return out, nil
}

func (s *Store) CreateJobConfig(_ context.Context, c *domain.JobConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cc := *c
	s.configs[c.JobConfigID] = &cc
	return nil
}

func (s *Store) GetJobConfig(_ context.Context, id uuid.UUID) (*domain.JobConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.configs[id]
	if !ok {
		return nil, domain.E(domain.CodeValidationError, "job_config %s not found", id)
	}
	cc := *c
	return &cc, nil
}

// --- TranscriptStore ---------------------------------------------------

func (s *Store) CreateVersion(_ context.Context, v *domain.TranscriptVersion, segments []*domain.Segment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.versions[v.TranscriptVersionID] = copyVersion(v)
	s.verOrder = append(s.verOrder, v.TranscriptVersionID)
	for _, sg := range segments {
		s.segments[sg.SegmentID] = copySegment(sg)
		s.segByVer[v.TranscriptVersionID] = append(s.segByVer[v.TranscriptVersionID], sg.SegmentID)
	}
	return nil
}

func (s *Store) GetVersion(_ context.Context, id uuid.UUID) (*domain.TranscriptVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.versions[id]
	if !ok {
		return nil, domain.E(domain.CodeTranscriptNotFound, "transcript version %s not found", id)
	}
	return copyVersion(v), nil
}

func (s *Store) ListVersions(_ context.Context, jobID uuid.UUID) ([]*domain.TranscriptVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*domain.TranscriptVersion
	for _, id := range s.verOrder {
		if s.versions[id].JobID == jobID {
			out = append(out, copyVersion(s.versions[id]))
		}
	}
	return out, nil
}

func (s *Store) LatestVersion(_ context.Context, jobID uuid.UUID, versionType string) (*domain.TranscriptVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var latest *domain.TranscriptVersion
	for _, id := range s.verOrder {
		v := s.versions[id]
		if v.JobID == jobID && v.VersionType == versionType {
			latest = v // verOrder is insertion order; last wins
		}
	}
	if latest == nil {
		return nil, nil
	}
	return copyVersion(latest), nil
}

func (s *Store) ListSegments(_ context.Context, versionID uuid.UUID) ([]*domain.Segment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := s.segByVer[versionID]
	out := make([]*domain.Segment, 0, len(ids))
	for _, id := range ids {
		out = append(out, copySegment(s.segments[id]))
	}
	sort.SliceStable(out, func(i, k int) bool { return out[i].StartMS < out[k].StartMS })
	return out, nil
}

func (s *Store) GetSegment(_ context.Context, segmentID uuid.UUID) (*domain.Segment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sg, ok := s.segments[segmentID]
	if !ok {
		return nil, domain.E(domain.CodeSegmentNotFound, "segment %s not found", segmentID)
	}
	return copySegment(sg), nil
}

func (s *Store) UpdateSegment(_ context.Context, sg *domain.Segment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.segments[sg.SegmentID]; !ok {
		return domain.E(domain.CodeSegmentNotFound, "segment %s not found", sg.SegmentID)
	}
	s.segments[sg.SegmentID] = copySegment(sg)
	return nil
}

// --- SummaryStore ------------------------------------------------------

func (s *Store) CreateSummary(_ context.Context, sm *domain.Summary) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := *sm
	s.summaries[sm.SummaryID] = &c
	s.sumOrder = append(s.sumOrder, sm.SummaryID)
	return nil
}

func (s *Store) GetSummary(_ context.Context, id uuid.UUID) (*domain.Summary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sm, ok := s.summaries[id]
	if !ok {
		return nil, domain.E(domain.CodeSummaryNotFound, "summary %s not found", id)
	}
	c := *sm
	return &c, nil
}

func (s *Store) LatestSummaryByJob(_ context.Context, jobID uuid.UUID) (*domain.Summary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var latest *domain.Summary
	for _, id := range s.sumOrder {
		if s.summaries[id].JobID == jobID {
			latest = s.summaries[id]
		}
	}
	if latest == nil {
		return nil, domain.E(domain.CodeSummaryNotFound, "no summary for job %s", jobID)
	}
	c := *latest
	return &c, nil
}

func (s *Store) UpdateSummary(_ context.Context, sm *domain.Summary) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.summaries[sm.SummaryID]; !ok {
		return domain.E(domain.CodeSummaryNotFound, "summary %s not found", sm.SummaryID)
	}
	c := *sm
	s.summaries[sm.SummaryID] = &c
	return nil
}

// --- QualityStore ------------------------------------------------------

func (s *Store) CreateReport(_ context.Context, r *domain.QualityReport) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := *r
	c.Issues = append([]domain.QualityIssue(nil), r.Issues...)
	s.reports[r.QualityReportID] = &c
	s.repOrder = append(s.repOrder, r.QualityReportID)
	return nil
}

func (s *Store) LatestReportByJob(_ context.Context, jobID uuid.UUID) (*domain.QualityReport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var latest *domain.QualityReport
	for _, id := range s.repOrder {
		if s.reports[id].JobID == jobID {
			latest = s.reports[id]
		}
	}
	if latest == nil {
		return nil, domain.E(domain.CodeQualityReportNotFound, "no quality report for job %s", jobID)
	}
	c := *latest
	c.Issues = append([]domain.QualityIssue(nil), latest.Issues...)
	return &c, nil
}

// --- ApprovalStore -----------------------------------------------------

func (s *Store) CreateApproval(_ context.Context, a *domain.Approval) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.approvals[a.ApprovalID] = copyApproval(a)
	s.apprOrder = append(s.apprOrder, a.ApprovalID)
	return nil
}

func (s *Store) GetApproval(_ context.Context, id uuid.UUID) (*domain.Approval, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.approvals[id]
	if !ok {
		return nil, domain.E(domain.CodeValidationError, "approval %s not found", id)
	}
	return copyApproval(a), nil
}

func (s *Store) ListApprovalsByJob(_ context.Context, jobID uuid.UUID) ([]*domain.Approval, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*domain.Approval
	for _, id := range s.apprOrder {
		if s.approvals[id].JobID == jobID {
			out = append(out, copyApproval(s.approvals[id]))
		}
	}
	return out, nil
}

func (s *Store) CurrentApproval(_ context.Context, jobID uuid.UUID) (*domain.Approval, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var latest *domain.Approval
	for _, id := range s.apprOrder {
		a := s.approvals[id]
		if a.JobID == jobID && a.SupersededByApprovalID == nil {
			latest = a
		}
	}
	if latest == nil {
		return nil, nil
	}
	return copyApproval(latest), nil
}

func (s *Store) UpdateApproval(_ context.Context, a *domain.Approval) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.approvals[a.ApprovalID]; !ok {
		return domain.E(domain.CodeValidationError, "approval %s not found", a.ApprovalID)
	}
	s.approvals[a.ApprovalID] = copyApproval(a)
	return nil
}

// --- AuditStore (append-only) ------------------------------------------

func (s *Store) Append(_ context.Context, e *domain.AuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := *e
	if e.JobID != nil {
		id := *e.JobID
		c.JobID = &id
	}
	if e.EventPayload != nil {
		p := make(map[string]any, len(e.EventPayload))
		for k, v := range e.EventPayload {
			p[k] = v
		}
		c.EventPayload = p
	}
	s.audits = append(s.audits, &c)
	return nil
}

func (s *Store) ListByJob(_ context.Context, jobID uuid.UUID) ([]*domain.AuditEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*domain.AuditEvent
	for _, e := range s.audits {
		if e.JobID != nil && *e.JobID == jobID {
			c := *e
			out = append(out, &c)
		}
	}
	return out, nil
}

// --- ArtifactStore -----------------------------------------------------

func (s *Store) CreateArtifact(_ context.Context, a *domain.MediaArtifact) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := *a
	s.artifacts[a.ArtifactID] = &c
	s.artOrder = append(s.artOrder, a.ArtifactID)
	return nil
}

func (s *Store) GetArtifact(_ context.Context, id uuid.UUID) (*domain.MediaArtifact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.artifacts[id]
	if !ok {
		return nil, domain.E(domain.CodeMediaNotFound, "artifact %s not found", id)
	}
	c := *a
	return &c, nil
}

func (s *Store) ListArtifactsByJob(_ context.Context, jobID uuid.UUID, artifactType string) ([]*domain.MediaArtifact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*domain.MediaArtifact
	for _, id := range s.artOrder {
		a := s.artifacts[id]
		if a.JobID == jobID && (artifactType == "" || a.ArtifactType == artifactType) {
			c := *a
			out = append(out, &c)
		}
	}
	return out, nil
}

func (s *Store) MarkArtifactsSuperseded(_ context.Context, jobID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range s.artOrder {
		if s.artifacts[id].JobID == jobID {
			s.artifacts[id].Superseded = true
		}
	}
	return nil
}

func (s *Store) CreateExport(_ context.Context, e *domain.ExportRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := *e
	s.exports[e.ExportID] = &c
	s.expOrder = append(s.expOrder, e.ExportID)
	return nil
}

func (s *Store) GetExport(_ context.Context, id uuid.UUID) (*domain.ExportRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.exports[id]
	if !ok {
		return nil, domain.E(domain.CodeExportNotFound, "export %s not found", id)
	}
	c := *e
	return &c, nil
}

// --- ReviewTxStore (atomic approve / reopen under one lock hold) ---------

// insertVersionLocked stores a version and its segments. Caller holds mu.
func (s *Store) insertVersionLocked(v *domain.TranscriptVersion, segments []*domain.Segment) {
	s.versions[v.TranscriptVersionID] = copyVersion(v)
	s.verOrder = append(s.verOrder, v.TranscriptVersionID)
	for _, sg := range segments {
		s.segments[sg.SegmentID] = copySegment(sg)
		s.segByVer[v.TranscriptVersionID] = append(s.segByVer[v.TranscriptVersionID], sg.SegmentID)
	}
}

func (s *Store) ApproveJob(_ context.Context, p store.ApproveJobParams) (*domain.Job, []uuid.UUID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.jobs[p.JobID]
	if !ok {
		return nil, nil, domain.E(domain.CodeJobNotFound, "job %s not found", p.JobID)
	}
	if cur.Status != domain.StatusInReview {
		return nil, nil, domain.ErrStatusConflict
	}
	s.insertVersionLocked(p.ApprovedVersion, p.Segments)
	s.approvals[p.Approval.ApprovalID] = copyApproval(p.Approval)
	s.apprOrder = append(s.apprOrder, p.Approval.ApprovalID)
	var superseded []uuid.UUID
	for _, id := range s.apprOrder {
		a := s.approvals[id]
		if a.JobID == p.JobID && a.SupersededByApprovalID == nil && a.ApprovalID != p.Approval.ApprovalID {
			by := p.Approval.ApprovalID
			a.SupersededByApprovalID = &by
			superseded = append(superseded, a.ApprovalID)
		}
	}
	next := copyJob(cur)
	next.Status = domain.StatusApproved
	next.LastError = nil
	next.ActionRequired = ""
	next.UpdatedAt = time.Now().UTC()
	s.jobs[p.JobID] = copyJob(next)
	return next, superseded, nil
}

func (s *Store) ReopenJob(_ context.Context, p store.ReopenJobParams) (*domain.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.jobs[p.JobID]
	if !ok {
		return nil, domain.E(domain.CodeJobNotFound, "job %s not found", p.JobID)
	}
	if cur.Status != domain.StatusApproved && cur.Status != domain.StatusExported {
		return nil, domain.ErrStatusConflict
	}
	s.insertVersionLocked(p.ReviewedVersion, p.Segments)
	next := copyJob(cur)
	next.Status = domain.StatusInReview
	next.UpdatedAt = time.Now().UTC()
	s.jobs[p.JobID] = copyJob(next)
	return next, nil
}

func (s *Store) ListExportsByJob(_ context.Context, jobID uuid.UUID) ([]*domain.ExportRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*domain.ExportRecord
	for _, id := range s.expOrder {
		if s.exports[id].JobID == jobID {
			c := *s.exports[id]
			out = append(out, &c)
		}
	}
	return out, nil
}
