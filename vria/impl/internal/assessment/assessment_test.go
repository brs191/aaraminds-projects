package assessment

import (
	"errors"
	"testing"

	"github.com/aaraminds/vria/internal/enums"
	"github.com/aaraminds/vria/internal/hypothesis"
)

// fakeHyp is an in-memory HypothesisSource.
type fakeHyp struct {
	byUC map[string]hypothesis.Hypothesis
}

func (f *fakeHyp) Get(useCaseID string) (hypothesis.Hypothesis, []string, error) {
	h, ok := f.byUC[useCaseID]
	if !ok {
		return hypothesis.Hypothesis{}, nil, hypothesis.ErrNotFound
	}
	return h, nil, nil
}

func fptr(v float64) *float64 { return &v }

// realizedFixture wires a hypothesis + snapshot that satisfies every
// Realized condition (gate-a-value/03 §5).
func realizedFixture() (*fakeHyp, *MemProvider, UseCaseLookup) {
	hyp := &fakeHyp{byUC: map[string]hypothesis.Hypothesis{
		"UC-1": {
			ValueHypothesisID: "vh-1", UseCaseID: "UC-1",
			BusinessObjective: "reduce MTTR", ExpectedBenefit: "faster triage",
			BenefitType: "CycleTime", PrimaryMetricID: "M-1",
			BaselineValue: fptr(120), TargetValue: fptr(60),
			AttributionMethod: enums.DirectMeasurement,
			NetValueCheck:     enums.NetPositive,
			ValueOwner:        "owner",
			ApprovalState:     enums.ArtApproved,
		},
	}}
	prov := NewMemProvider()
	prov.SetSnapshot("UC-1", MetricSnapshot{
		CurrentValue:          fptr(60),
		LowerIsBetter:         true,
		BaselinePeriodDefined: true,
		EvidenceAuthority:     enums.Authoritative,
		EvidenceFreshness:     enums.Fresh,
		AllCitationsPresent:   true,
		HasEvidenceSource:     true,
		HasValueClaim:         true,
	})
	lookup := func(id string) (UseCaseContext, bool) {
		if id != "UC-1" {
			return UseCaseContext{}, false
		}
		return UseCaseContext{
			Sponsor: "sponsor", Tier: enums.TierAgent,
			DeliveryComplete: true, ApprovalBoundaryRecorded: true,
		}, true
	}
	return hyp, prov, lookup
}

func TestGenerateAssessmentPersistsImmutableDraft(t *testing.T) {
	hyp, prov, lookup := realizedFixture()
	var audited []string
	svc := NewService(hyp, prov, lookup, func(action, tt, tid, actor string) {
		audited = append(audited, action)
	})

	a, err := svc.GenerateAssessment("UC-1", "portfolio-lead")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if a.ApprovalState != enums.ArtDraft {
		t.Fatalf("approval_state = %s, want Draft", a.ApprovalState)
	}
	if a.ScoringRuleVersion != ScoringRuleVersion {
		t.Fatalf("scoring_rule_version = %s, want %s", a.ScoringRuleVersion, ScoringRuleVersion)
	}
	if a.ValueState != enums.Realized {
		t.Fatalf("value_state = %s, want Realized", a.ValueState)
	}
	// Deterministic total (contracts/20 §3a): 10+15+20+20+10+10+6+5.
	if a.RealizationScore != 96 || a.PreCapScore != 96 {
		t.Fatalf("score = %d pre-cap %d, want 96/96", a.RealizationScore, a.PreCapScore)
	}
	if a.Recommendation != enums.Scale || a.Confidence != enums.High {
		t.Fatalf("rec/conf = %s/%s, want Scale/High", a.Recommendation, a.Confidence)
	}
	if len(a.MissingEvidence) != 0 || len(a.AppliedCaps) != 0 {
		t.Fatalf("unexpected gaps %v caps %v", a.MissingEvidence, a.AppliedCaps)
	}
	if a.Version != 1 || a.CreatedAt.IsZero() {
		t.Fatalf("version=%d created_at=%v", a.Version, a.CreatedAt)
	}
	if len(audited) != 1 || audited[0] != "assessment.generated" {
		t.Fatalf("audit = %v", audited)
	}

	// Append-only: regeneration creates a new version and leaves v1 intact.
	b, err := svc.GenerateAssessment("UC-1", "portfolio-lead")
	if err != nil {
		t.Fatalf("regenerate: %v", err)
	}
	if b.AssessmentID == a.AssessmentID || b.Version != 2 {
		t.Fatalf("regeneration must create a new record: %s v%d", b.AssessmentID, b.Version)
	}
	got, err := svc.Get(a.AssessmentID)
	if err != nil || got.Version != 1 || got.ValueState != enums.Realized {
		t.Fatalf("v1 mutated: %+v err=%v", got, err)
	}
	if n := len(svc.ListByUseCase("UC-1")); n != 2 {
		t.Fatalf("history has %d assessments, want 2", n)
	}
}

func TestGenerateAssessmentMissingMetricDataStaysHonest(t *testing.T) {
	hyp := &fakeHyp{byUC: map[string]hypothesis.Hypothesis{
		"UC-2": {
			ValueHypothesisID: "vh-2", UseCaseID: "UC-2",
			BusinessObjective: "cut toil", ValueOwner: "owner",
			AttributionMethod: enums.AttributionUnknown,
			NetValueCheck:     enums.NetUnknown,
			ApprovalState:     enums.ArtDraft,
		},
	}}
	svc := NewService(hyp, NewMemProvider(), nil, nil)
	a, err := svc.GenerateAssessment("UC-2", "lead")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if a.ValueState != enums.HypothesisOnly {
		t.Fatalf("value_state = %s, want HypothesisOnly", a.ValueState)
	}
	want := map[string]bool{
		"primary_metric_id": true, "baseline_value": true, "current_value": true,
		"target_value": true, "evidence_source": true, "attribution_method": true,
	}
	for _, m := range a.MissingEvidence {
		delete(want, m)
	}
	if len(want) != 0 {
		t.Fatalf("missing_evidence lacks %v (got %v)", want, a.MissingEvidence)
	}
	// NoPrimaryMetric cap (39) must apply — no inferred values (GE-013).
	if a.RealizationScore > 39 {
		t.Fatalf("score = %d, want <= 39 (NoPrimaryMetric cap)", a.RealizationScore)
	}
	if a.Recommendation != enums.NeedsEvidence {
		t.Fatalf("recommendation = %s, want NeedsEvidence", a.Recommendation)
	}
}

func TestGenerateAssessmentRegistryOnlyUseCase(t *testing.T) {
	hyp := &fakeHyp{byUC: map[string]hypothesis.Hypothesis{}}
	lookup := func(id string) (UseCaseContext, bool) {
		return UseCaseContext{Tier: enums.TierTool}, id == "UC-3"
	}
	svc := NewService(hyp, NewMemProvider(), lookup, nil)
	a, err := svc.GenerateAssessment("UC-3", "lead")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	// No hypothesis means no value owner: NotReady, never invented fields.
	if a.ValueState != enums.NotReady {
		t.Fatalf("value_state = %s, want NotReady", a.ValueState)
	}
}

func TestGenerateAssessmentUnknownUseCase(t *testing.T) {
	svc := NewService(&fakeHyp{byUC: map[string]hypothesis.Hypothesis{}}, nil, nil, nil)
	_, err := svc.GenerateAssessment("UC-404", "lead")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
