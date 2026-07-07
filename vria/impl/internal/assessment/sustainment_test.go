package assessment

import (
	"testing"
	"time"

	"github.com/aaraminds/vria/internal/enums"
	"github.com/aaraminds/vria/internal/scoring"
)

type notification struct {
	useCaseID string
	owner     string
	status    enums.SustainmentStatus
}

func realizedService(t *testing.T) (*Service, *MemProvider, *[]notification) {
	t.Helper()
	hyp, prov, lookup := realizedFixture()
	svc := NewService(hyp, prov, lookup, nil)
	if _, err := svc.GenerateAssessment("UC-1", "lead"); err != nil {
		t.Fatalf("seed assessment: %v", err)
	}
	a, _ := svc.Latest("UC-1")
	if a.ValueState != enums.Realized {
		t.Fatalf("fixture value_state = %s, want Realized", a.ValueState)
	}
	notes := &[]notification{}
	return svc, prov, notes
}

func notifier(notes *[]notification) Notifier {
	return func(uc, owner string, status enums.SustainmentStatus) {
		*notes = append(*notes, notification{uc, owner, status})
	}
}

// Two consecutive failed checks regress the claim (contracts/20 §7, GE-006):
// a NEW assessment is generated with state Regressed and recommendation Fix.
func TestSchedulerTwoCycleRegression(t *testing.T) {
	svc, prov, notes := realizedService(t)
	sched := NewScheduler(svc, 0, notifier(notes)) // default 30-day window
	// Two checks below 80% of target (target 60, threshold 48).
	prov.QueueCheck("UC-1", scoring.SustainmentCheck{MeasuredBenefit: fptr(40), TargetValue: 60})
	prov.QueueCheck("UC-1", scoring.SustainmentCheck{MeasuredBenefit: fptr(40), TargetValue: 60})

	t0 := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	out := sched.RunDue(t0)
	if len(out) != 1 || out[0].Status != enums.SustainAtRisk || out[0].NewAssessmentID != "" {
		t.Fatalf("cycle 1 = %+v, want AtRisk with no new assessment", out)
	}
	if want := t0.Add(DefaultReportingWindow); !sched.NextCheckAt("UC-1").Equal(want) {
		t.Fatalf("next check = %v, want %v (last + reporting window)", sched.NextCheckAt("UC-1"), want)
	}
	// Not due yet: nothing runs.
	if out := sched.RunDue(t0.Add(time.Hour)); len(out) != 0 {
		t.Fatalf("early run executed %d checks, want 0", len(out))
	}
	// Second window: second consecutive failure regresses.
	out = sched.RunDue(t0.Add(DefaultReportingWindow))
	if len(out) != 1 || out[0].Status != enums.SustainRegressed {
		t.Fatalf("cycle 2 = %+v, want Regressed", out)
	}
	if out[0].NewAssessmentID == "" {
		t.Fatal("regression must generate a new assessment")
	}
	a, err := svc.Get(out[0].NewAssessmentID)
	if err != nil {
		t.Fatalf("get regressed assessment: %v", err)
	}
	if a.ValueState != enums.Regressed || a.Recommendation != enums.Fix {
		t.Fatalf("regressed assessment = %s/%s, want Regressed/Fix", a.ValueState, a.Recommendation)
	}
	if a.Version != 2 || a.SustainmentStatus != enums.SustainRegressed {
		t.Fatalf("version=%d sustainment=%s", a.Version, a.SustainmentStatus)
	}
	// The original Realized assessment is untouched (append-only).
	if first := svc.ListByUseCase("UC-1")[0]; first.ValueState != enums.Realized {
		t.Fatalf("original assessment mutated to %s", first.ValueState)
	}
	// Owner notified on both AtRisk and Regressed.
	if len(*notes) != 2 || (*notes)[0].status != enums.SustainAtRisk || (*notes)[1].status != enums.SustainRegressed {
		t.Fatalf("notifications = %+v", *notes)
	}
	if (*notes)[0].owner != "owner" {
		t.Fatalf("notified owner = %q, want value owner", (*notes)[0].owner)
	}
	// Latest is no longer Realized, so no further checks run.
	if out := sched.RunDue(t0.Add(2 * DefaultReportingWindow)); len(out) != 0 {
		t.Fatalf("post-regression run executed %d checks, want 0", len(out))
	}
}

// One failed check → AtRisk only: state stays Realized, no new assessment,
// and a subsequent passing check recovers to Ok.
func TestSchedulerSingleFailureIsAtRiskOnly(t *testing.T) {
	svc, prov, notes := realizedService(t)
	sched := NewScheduler(svc, DefaultReportingWindow, notifier(notes))
	prov.QueueCheck("UC-1", scoring.SustainmentCheck{MeasuredBenefit: fptr(40), TargetValue: 60})
	prov.QueueCheck("UC-1", scoring.SustainmentCheck{MeasuredBenefit: fptr(55), TargetValue: 60})

	t0 := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	out := sched.RunDue(t0)
	if len(out) != 1 || out[0].Status != enums.SustainAtRisk || out[0].NewAssessmentID != "" {
		t.Fatalf("cycle 1 = %+v, want AtRisk with no new assessment", out)
	}
	if n := len(svc.ListByUseCase("UC-1")); n != 1 {
		t.Fatalf("assessment count = %d, want 1 (no regeneration on single failure)", n)
	}
	out = sched.RunDue(t0.Add(DefaultReportingWindow))
	if len(out) != 1 || out[0].Status != enums.SustainOk {
		t.Fatalf("cycle 2 = %+v, want recovery to Ok", out)
	}
	if n := len(svc.ListByUseCase("UC-1")); n != 1 {
		t.Fatalf("assessment count = %d, want 1", n)
	}
	if len(*notes) != 1 || (*notes)[0].status != enums.SustainAtRisk {
		t.Fatalf("notifications = %+v, want single AtRisk", *notes)
	}
}

// A missing snapshot counts as a failed check (contracts/20 §7); two missing
// windows in a row regress the claim.
func TestSchedulerMissingSnapshotCountsAsFailed(t *testing.T) {
	svc, _, notes := realizedService(t)
	sched := NewScheduler(svc, DefaultReportingWindow, notifier(notes))
	// Nothing queued: the provider has no snapshot for either cycle.
	t0 := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	out := sched.RunDue(t0)
	if len(out) != 1 || !out[0].CheckFailed || out[0].Status != enums.SustainAtRisk {
		t.Fatalf("cycle 1 = %+v, want failed check → AtRisk", out)
	}
	out = sched.RunDue(t0.Add(DefaultReportingWindow))
	if len(out) != 1 || out[0].Status != enums.SustainRegressed || out[0].NewAssessmentID == "" {
		t.Fatalf("cycle 2 = %+v, want Regressed with new assessment", out)
	}
	if n := len(svc.CheckHistory("UC-1")); n != 2 {
		t.Fatalf("check history = %d entries, want 2", n)
	}
}
