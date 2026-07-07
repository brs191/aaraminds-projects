package hypothesis

import (
	"errors"
	"testing"

	"github.com/aaraminds/vria/internal/enums"
	"github.com/aaraminds/vria/internal/scoring"
)

func newSvc() (*Service, *[]string) {
	var trail []string
	svc := NewService(func(action, tt, tid, actor string) {
		trail = append(trail, action)
	})
	return svc, &trail
}

func draftAndApprove(t *testing.T, svc *Service, uc string, changes map[string]interface{}) Hypothesis {
	t.Helper()
	d, err := svc.CreateDraft(uc, "owner", changes)
	if err != nil {
		t.Fatal(err)
	}
	req, err := svc.SubmitForApproval(d.DraftID, "owner")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.Decide(req.ApprovalID, "approve", "portfolio-lead"); err != nil {
		t.Fatal(err)
	}
	h, err := svc.Commit(d.DraftID, req.ApprovalID, "owner")
	if err != nil {
		t.Fatal(err)
	}
	return h
}

// Done-when: hypothesis CRUD round-trips through approval with versioning.
func TestRoundTripThroughApproval(t *testing.T) {
	svc, trail := newSvc()
	h := draftAndApprove(t, svc, "UC-1", map[string]interface{}{
		"value_owner": "owner", "expected_benefit": "less rework",
		"business_objective": "quality", "benefit_type": "Quality",
		"primary_metric_id": "M-1",
	})
	if h.RecordVersion != 1 || h.ApprovalState != enums.ArtApproved {
		t.Fatalf("v%d state=%s, want v1 Approved", h.RecordVersion, h.ApprovalState)
	}
	// Second cycle bumps the version.
	h = draftAndApprove(t, svc, "UC-1", map[string]interface{}{
		"baseline_value": 10.0, "target_value": 4.0,
	})
	if h.RecordVersion != 2 {
		t.Fatalf("version = %d, want 2", h.RecordVersion)
	}
	got, missing, err := svc.Get("UC-1")
	if err != nil || got.RecordVersion != 2 {
		t.Fatalf("get: %v v%d", err, got.RecordVersion)
	}
	if len(missing) != 0 {
		t.Fatalf("missing = %v, want none", missing)
	}
	want := []string{"hypothesis.draft_created", "approval.submitted", "approval.decided", "hypothesis.committed"}
	for _, w := range want {
		found := false
		for _, a := range *trail {
			if a == w {
				found = true
			}
		}
		if !found {
			t.Fatalf("audit trail missing %q: %v", w, *trail)
		}
	}
}

// Done-when: rejected commits leave the original record untouched.
func TestRejectedCommitLeavesOriginalUntouched(t *testing.T) {
	svc, _ := newSvc()
	draftAndApprove(t, svc, "UC-2", map[string]interface{}{
		"value_owner": "owner", "expected_benefit": "original",
	})
	d, _ := svc.CreateDraft("UC-2", "owner", map[string]interface{}{
		"expected_benefit": "sneaky change",
	})
	req, _ := svc.SubmitForApproval(d.DraftID, "owner")
	if _, err := svc.Decide(req.ApprovalID, "reject", "portfolio-lead"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Commit(d.DraftID, req.ApprovalID, "owner"); !errors.Is(err, ErrApprovalRequired) {
		t.Fatalf("commit after rejection: %v, want ErrApprovalRequired", err)
	}
	h, _, _ := svc.Get("UC-2")
	if h.ExpectedBenefit != "original" || h.RecordVersion != 1 {
		t.Fatalf("original mutated: %+v", h)
	}
}

func TestCommitWithoutApprovalRefused(t *testing.T) {
	svc, _ := newSvc()
	d, _ := svc.CreateDraft("UC-3", "owner", map[string]interface{}{"value_owner": "o"})
	if _, err := svc.Commit(d.DraftID, "apr-bogus", "owner"); !errors.Is(err, ErrApprovalRequired) {
		t.Fatalf("commit without approval: %v", err)
	}
	// Submitted-but-undecided is also not enough.
	req, _ := svc.SubmitForApproval(d.DraftID, "owner")
	if _, err := svc.Commit(d.DraftID, req.ApprovalID, "owner"); !errors.Is(err, ErrApprovalRequired) {
		t.Fatalf("commit with pending approval: %v", err)
	}
}

// Disallowed fields reject the draft per contracts/09 §3.4.
func TestDisallowedFieldRejected(t *testing.T) {
	svc, _ := newSvc()
	d, _ := svc.CreateDraft("UC-4", "owner", map[string]interface{}{
		"realization_score": 100.0, // scores are computed, never written
	})
	if d.ValidationStatus != "Invalid" {
		t.Fatalf("draft with disallowed field must be Invalid: %+v", d)
	}
	if _, err := svc.SubmitForApproval(d.DraftID, "owner"); !errors.Is(err, ErrInvalidDraft) {
		t.Fatalf("invalid draft must not be submittable: %v", err)
	}
}

func TestMissingRequiredFieldsListed(t *testing.T) {
	svc, _ := newSvc()
	draftAndApprove(t, svc, "UC-5", map[string]interface{}{"value_owner": "o"})
	_, missing, _ := svc.Get("UC-5")
	for _, want := range []string{"expected_benefit", "primary_metric_id", "baseline_value", "target_value"} {
		found := false
		for _, m := range missing {
			if m == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing_required_fields lacks %q: %v", want, missing)
		}
	}
}

// GE-002 against the running workflow: hypothesis without baseline scores
// HypothesisOnly with no realized claim.
func TestGE002AgainstService(t *testing.T) {
	svc, _ := newSvc()
	h := draftAndApprove(t, svc, "UC-6", map[string]interface{}{
		"value_owner": "owner", "expected_benefit": "save cost",
		"business_objective": "cost", "benefit_type": "Cost",
		"primary_metric_id": "M-6",
	})
	r := scoring.Score(inputFrom(h))
	if r.ValueState != enums.HypothesisOnly {
		t.Fatalf("state = %s, want HypothesisOnly", r.ValueState)
	}
}

// GE-011 against the running workflow: expert judgement alone never yields
// Realized, even after hypothesis approval.
func TestGE011AgainstService(t *testing.T) {
	svc, _ := newSvc()
	h := draftAndApprove(t, svc, "UC-7", map[string]interface{}{
		"value_owner": "owner", "expected_benefit": "save cost",
		"business_objective": "cost", "benefit_type": "Cost",
		"primary_metric_id": "M-7", "baseline_value": 100.0, "target_value": 50.0,
		"attribution_method": "ExpertJudgement", "net_value_check": "Positive",
	})
	in := inputFrom(h)
	cur := 50.0
	in.CurrentValue = &cur // owner claims target achieved
	r := scoring.Score(in)
	if r.ValueState == enums.Realized {
		t.Fatal("ExpertJudgement-only claim must not be Realized")
	}
}

// inputFrom assembles a scoring input from the committed hypothesis, the way
// the assessment workflow will (metric snapshot fields left to the caller).
func inputFrom(h Hypothesis) scoring.Input {
	return scoring.Input{
		ValueOwner:               h.ValueOwner,
		BusinessObjective:        h.BusinessObjective,
		BenefitType:              h.BenefitType,
		PrimaryMetricID:          h.PrimaryMetricID,
		BaselineValue:            h.BaselineValue,
		BaselinePeriodDefined:    h.BaselineValue != nil,
		TargetValue:              h.TargetValue,
		EvidenceAuthority:        enums.Authoritative,
		EvidenceFreshness:        enums.Fresh,
		AllCitationsPresent:      true,
		HasEvidenceSource:        true,
		Attribution:              h.AttributionMethod,
		NetValue:                 h.NetValueCheck,
		Sustainment:              enums.SustainNotStarted,
		ApprovalBoundaryRecorded: true,
		ArtifactState:            h.ApprovalState,
	}
}
