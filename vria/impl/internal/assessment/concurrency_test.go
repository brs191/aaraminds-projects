package assessment

import (
	"sync"
	"testing"
	"time"

	"github.com/aaraminds/vria/internal/enums"
	"github.com/aaraminds/vria/internal/hypothesis"
	"github.com/aaraminds/vria/internal/scoring"
)

func fp(v float64) *float64 { return &v }

// realizedSvc builds a service with one Realized use case, ready for
// sustainment checks.
func realizedSvc(t *testing.T) (*Service, *MemProvider) {
	t.Helper()
	hyp := hypothesis.NewService(nil)
	// Register a fully-formed hypothesis via the draft/approve path.
	d, err := hyp.CreateDraft("UC-1", "owner", map[string]interface{}{
		"value_owner": "owner", "expected_benefit": "cut cost",
		"business_objective": "cost", "benefit_type": "Cost",
		"primary_metric_id": "M-1", "baseline_value": 100.0, "target_value": 50.0,
		"attribution_method": "DirectMeasurement", "net_value_check": "Positive",
	})
	if err != nil {
		t.Fatal(err)
	}
	req, _ := hyp.SubmitForApproval(d.DraftID, "owner")
	hyp.Decide(req.ApprovalID, "approve", "lead")
	hyp.Commit(d.DraftID, req.ApprovalID, "lead")

	prov := NewMemProvider()
	prov.SetSnapshot("UC-1", MetricSnapshot{
		CurrentValue: fp(50), EvidenceAuthority: enums.Authoritative,
		EvidenceFreshness: enums.Fresh, AllCitationsPresent: true, HasEvidenceSource: true,
	})
	svc := NewService(hyp, prov,
		func(string) (UseCaseContext, bool) {
			return UseCaseContext{Tier: enums.TierTool, ApprovalBoundaryRecorded: true}, true
		}, nil)
	if _, err := svc.GenerateAssessment("UC-1", "lead"); err != nil {
		t.Fatal(err)
	}
	if a, _ := svc.Latest("UC-1"); a.ValueState != enums.Realized {
		t.Fatalf("setup: state = %s, want Realized", a.ValueState)
	}
	return svc, prov
}

// Fix #1: concurrent RunDue must not race the scheduler or corrupt check
// history. Run with -race; without the scheduler mutex this fails.
func TestConcurrentRunDueNoRace(t *testing.T) {
	svc, prov := realizedSvc(t)
	for i := 0; i < 20; i++ {
		prov.QueueCheck("UC-1", scoring.SustainmentCheck{MeasuredBenefit: fp(90), TargetValue: 100})
	}
	sch := NewScheduler(svc, time.Hour, nil)
	var wg sync.WaitGroup
	now := time.Now()
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); sch.RunDue(now) }()
	}
	wg.Wait()
	// Exactly one check ran for the single due window: overlapping calls must
	// not each append a check (which could fake "two consecutive failures").
	if n := len(svc.CheckHistory("UC-1")); n != 1 {
		t.Fatalf("check history = %d after concurrent RunDue, want 1", n)
	}
}

// Fix #8: a late RunDue anchors the next window to the prior schedule rather
// than drifting forward from "now".
func TestSustainmentScheduleDoesNotDrift(t *testing.T) {
	svc, prov := realizedSvc(t)
	prov.QueueCheck("UC-1", scoring.SustainmentCheck{MeasuredBenefit: fp(90), TargetValue: 100})
	sch := NewScheduler(svc, time.Hour, nil)
	t0 := time.Date(2026, 7, 7, 0, 0, 0, 0, time.UTC)
	sch.RunDue(t0)
	first := sch.NextCheckAt("UC-1") // t0 + 1h
	// A RunDue 3 hours late should not push next to late+1h; it anchors.
	prov.QueueCheck("UC-1", scoring.SustainmentCheck{MeasuredBenefit: fp(90), TargetValue: 100})
	sch.RunDue(t0.Add(3 * time.Hour))
	next := sch.NextCheckAt("UC-1")
	if !next.After(t0.Add(3 * time.Hour)) {
		t.Fatalf("next %v must be after now", next)
	}
	// Anchored: next is first + k*window, i.e. minutes align to :00, not :00 of late+1h drift.
	if next.Minute() != first.Minute() {
		t.Fatalf("schedule drifted: first=%v next=%v", first, next)
	}
}
