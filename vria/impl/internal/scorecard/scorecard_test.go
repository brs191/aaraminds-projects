package scorecard

import (
	"errors"
	"testing"

	"github.com/aaraminds/vria/internal/enums"
)

// fakeAssessments: A-1 fully evidenced, A-2 has gaps.
func fakeAssessments(id string) (AssessmentInfo, bool) {
	switch id {
	case "A-1":
		return AssessmentInfo{AssessmentID: "A-1"}, true
	case "A-2":
		return AssessmentInfo{AssessmentID: "A-2", MissingEvidence: []string{"current_value"}}, true
	}
	return AssessmentInfo{}, false
}

func draft(t *testing.T, s *Service, ids ...string) Scorecard {
	t.Helper()
	c, err := s.CreateDraft("Q3 Portfolio", Period{Start: "2026-07-01", End: "2026-09-30"}, ids, "lead")
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}
	return c
}

func TestCreateDraftComputesEvidenceCoverage(t *testing.T) {
	s := NewService(fakeAssessments, nil)
	c := draft(t, s, "A-1", "A-2")
	if c.ArtifactState != enums.ArtDraft {
		t.Fatalf("state = %s, want Draft", c.ArtifactState)
	}
	cov := c.EvidenceCoverageSummary
	if cov.AssessmentsTotal != 2 || cov.WithCitations != 1 || cov.WithGaps != 1 {
		t.Fatalf("coverage = %+v, want total=2 citations=1 gaps=1", cov)
	}
	if _, err := s.CreateDraft("bad", Period{}, []string{"A-404"}, "lead"); !errors.Is(err, ErrUnknownAssessment) {
		t.Fatalf("unknown assessment err = %v", err)
	}
}

// GE-007: publication without an Approved request must not execute.
func TestPublishRequiresApprovedRequest(t *testing.T) {
	s := NewService(fakeAssessments, nil)
	c := draft(t, s, "A-1")

	// No approval at all.
	if _, err := s.Publish(c.ScorecardID, "", "lead"); !errors.Is(err, ErrApprovalRequired) {
		t.Fatalf("publish without approval: err = %v, want ErrApprovalRequired", err)
	}
	// Submitted-but-undecided request is not enough.
	req, err := s.SubmitForApproval(c.ScorecardID, ActionPublication, "lead")
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if _, err := s.Publish(c.ScorecardID, req.ApprovalID, "lead"); !errors.Is(err, ErrApprovalRequired) {
		t.Fatalf("publish with pending approval: err = %v, want ErrApprovalRequired", err)
	}
	// Rejected request is terminal and never unlocks publication.
	if _, err := s.Decide(req.ApprovalID, "reject", "sponsor"); err != nil {
		t.Fatalf("reject: %v", err)
	}
	if _, err := s.Publish(c.ScorecardID, req.ApprovalID, "lead"); !errors.Is(err, ErrApprovalRequired) {
		t.Fatalf("publish with rejected approval: err = %v, want ErrApprovalRequired", err)
	}
	got, _ := s.Get(c.ScorecardID)
	if got.ArtifactState != enums.ArtDraft || got.PublishedAt != nil {
		t.Fatalf("scorecard changed despite gate: %+v", got)
	}
	if n := len(s.DecisionRecords()); n != 0 {
		t.Fatalf("decision log has %d records after blocked publication, want 0", n)
	}
}

func TestPublishLifecycleWithDecisionLog(t *testing.T) {
	s := NewService(fakeAssessments, nil)
	c := draft(t, s, "A-1", "A-2")
	req, err := s.SubmitForApproval(c.ScorecardID, ActionPublication, "lead")
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	dec, err := s.Decide(req.ApprovalID, "approve", "sponsor")
	if err != nil || dec.State != enums.ReqApproved || dec.DecidedBy != "sponsor" {
		t.Fatalf("decide = %+v err=%v", dec, err)
	}
	// Approval moved the artifact Draft → Approved (contracts/18 §2.2).
	got, _ := s.Get(c.ScorecardID)
	if got.ArtifactState != enums.ArtApproved {
		t.Fatalf("state after approval = %s, want Approved", got.ArtifactState)
	}
	pub, err := s.Publish(c.ScorecardID, req.ApprovalID, "lead")
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if pub.ArtifactState != enums.ArtPublished || pub.PublishedAt == nil {
		t.Fatalf("published = %+v", pub)
	}
	recs := s.DecisionRecords()
	if len(recs) != 1 {
		t.Fatalf("decision log = %d records, want 1", len(recs))
	}
	r := recs[0]
	if r.DecisionType != ActionPublication || r.DecidedBy != "sponsor" ||
		r.ApprovalID != req.ApprovalID || r.TargetID != c.ScorecardID {
		t.Fatalf("decision record = %+v", r)
	}
	if pub.DecisionLogPointer != r.DecisionRecordID {
		t.Fatalf("decision_log_pointer = %s, want %s", pub.DecisionLogPointer, r.DecisionRecordID)
	}
}

func TestSupersedeIsApprovalGatedAndLinksReplacement(t *testing.T) {
	s := NewService(fakeAssessments, nil)
	old := draft(t, s, "A-1")
	// Publish the original first.
	req, _ := s.SubmitForApproval(old.ScorecardID, ActionPublication, "lead")
	s.Decide(req.ApprovalID, "approve", "sponsor")
	published, err := s.Publish(old.ScorecardID, req.ApprovalID, "lead")
	if err != nil {
		t.Fatalf("publish original: %v", err)
	}
	repl := draft(t, s, "A-1", "A-2")

	// Without an approved supersession request: gate holds.
	if _, err := s.Supersede(old.ScorecardID, repl.ScorecardID, req.ApprovalID, "lead"); !errors.Is(err, ErrApprovalRequired) {
		t.Fatalf("supersede with publication approval reused: err = %v, want ErrApprovalRequired", err)
	}
	sreq, err := s.SubmitForApproval(old.ScorecardID, ActionSupersession, "lead")
	if err != nil {
		t.Fatalf("submit supersession: %v", err)
	}
	if _, err := s.Decide(sreq.ApprovalID, "approve", "portfolio-lead"); err != nil {
		t.Fatalf("approve supersession: %v", err)
	}
	got, err := s.Supersede(old.ScorecardID, repl.ScorecardID, sreq.ApprovalID, "lead")
	if err != nil {
		t.Fatalf("supersede: %v", err)
	}
	if got.SupersedesScorecardID != old.ScorecardID {
		t.Fatalf("supersedes_scorecard_id = %s, want %s", got.SupersedesScorecardID, old.ScorecardID)
	}
	oldNow, _ := s.Get(old.ScorecardID)
	if oldNow.ArtifactState != enums.ArtSuperseded {
		t.Fatalf("old state = %s, want Superseded", oldNow.ArtifactState)
	}
	// Published content is never edited in place: only the lifecycle moved.
	if oldNow.PublishedAt == nil || !oldNow.PublishedAt.Equal(*published.PublishedAt) ||
		oldNow.Title != published.Title || len(oldNow.AssessmentIDs) != len(published.AssessmentIDs) {
		t.Fatalf("published scorecard content mutated: %+v", oldNow)
	}
	if n := len(s.DecisionRecords()); n != 2 {
		t.Fatalf("decision log = %d records, want 2 (publish + supersede)", n)
	}
}

func TestInvalidateRequiresReasonAndApproval(t *testing.T) {
	s := NewService(fakeAssessments, nil)
	c := draft(t, s, "A-1")
	req, _ := s.SubmitForApproval(c.ScorecardID, ActionPublication, "lead")
	s.Decide(req.ApprovalID, "approve", "sponsor")
	if _, err := s.Publish(c.ScorecardID, req.ApprovalID, "lead"); err != nil {
		t.Fatalf("publish: %v", err)
	}
	ireq, _ := s.SubmitForApproval(c.ScorecardID, ActionInvalidation, "lead")
	s.Decide(ireq.ApprovalID, "approve", "governance")
	if _, err := s.Invalidate(c.ScorecardID, ireq.ApprovalID, "", "lead"); !errors.Is(err, ErrReasonRequired) {
		t.Fatalf("invalidate without reason: err = %v", err)
	}
	got, err := s.Invalidate(c.ScorecardID, ireq.ApprovalID, "metric source retracted", "lead")
	if err != nil || got.ArtifactState != enums.ArtInvalidated {
		t.Fatalf("invalidate = %+v err=%v", got, err)
	}
	recs := s.DecisionRecords()
	if last := recs[len(recs)-1]; last.Rationale != "metric source retracted" {
		t.Fatalf("invalidation rationale not preserved: %+v", last)
	}
}
