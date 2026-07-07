package scoring

import (
	"testing"

	"github.com/aaraminds/vria/internal/enums"
)

func f(v float64) *float64 { return &v }

// strongInput mirrors worked example W1 in contracts/20 §3a.
func strongInput() Input {
	return Input{
		ValueOwner: "owner", Sponsor: "sponsor", BusinessObjective: "reduce MTTR",
		BenefitType: "Cost", Tier: enums.TierTool, PrimaryMetricID: "M-1",
		BaselineValue: f(100), BaselinePeriodDefined: true,
		CurrentValue: f(50), TargetValue: f(50),
		EvidenceAuthority: enums.Authoritative, EvidenceFreshness: enums.Fresh,
		AllCitationsPresent: true, HasEvidenceSource: true,
		Attribution:              enums.DirectMeasurement,
		NetValue:                 enums.NetPositive,
		Sustainment:              enums.SustainOk,
		ApprovalBoundaryRecorded: true,
		ArtifactState:            enums.ArtApproved,
	}
}

func TestWorkedExampleW1(t *testing.T) {
	r := Score(strongInput())
	if r.PreCapScore != 100 {
		t.Fatalf("W1 pre-cap = %d, want 100 (breakdown %+v)", r.PreCapScore, r.Breakdown)
	}
	if r.ValueState != enums.Realized {
		t.Fatalf("W1 state = %s, want Realized", r.ValueState)
	}
	if r.Recommendation != enums.Scale {
		t.Fatalf("W1 rec = %s, want Scale", r.Recommendation)
	}
	if r.Confidence != enums.High {
		t.Fatalf("W1 confidence = %s, want High", r.Confidence)
	}
}

func TestWorkedExampleW2(t *testing.T) {
	// Mid pilot, financial claim without cost data (W2 = 54 pre-cap).
	in := Input{
		ValueOwner: "owner", BusinessObjective: "reduce rework",
		BenefitType: "Productivity", Tier: enums.TierAgent, PrimaryMetricID: "M-2",
		BaselineValue: f(0), BaselinePeriodDefined: true,
		CurrentValue: f(50), TargetValue: f(100), // progress 0.5
		EvidenceAuthority: enums.Secondary, EvidenceFreshness: enums.Aging,
		AllCitationsPresent: true, HasEvidenceSource: true,
		Attribution: enums.BeforeAfter, ConfoundersDocumented: 2,
		NetValue:                 enums.NetUnknown,
		Sustainment:              enums.SustainNotStarted,
		ApprovalBoundaryRecorded: true,
	}
	r := Score(in)
	if r.PreCapScore != 56 {
		t.Fatalf("W2 pre-cap = %d, want 56 (breakdown %+v)", r.PreCapScore, r.Breakdown)
	}
	if r.ValueState != enums.AtRisk {
		t.Fatalf("W2 state = %s, want AtRisk (financial claim, net unknown)", r.ValueState)
	}
	if r.Recommendation != enums.NeedsEvidence {
		t.Fatalf("W2 rec = %s, want NeedsEvidence", r.Recommendation)
	}
}

func TestWorkedExampleW3(t *testing.T) {
	// Degenerate: no baseline (W3 = 45 pre-cap, HypothesisOnly).
	in := Input{
		ValueOwner: "owner", Sponsor: "sponsor", BusinessObjective: "risk reduction",
		BenefitType: "Risk", Tier: enums.TierTool, PrimaryMetricID: "M-3",
		EvidenceAuthority: enums.Authoritative, EvidenceFreshness: enums.Fresh,
		HasEvidenceSource: true,
		Attribution:       enums.AttributionUnknown,
		NetValue:          enums.NetNotApplicable, NetValueRationale: "non-financial",
		Sustainment:              enums.SustainNotStarted,
		ApprovalBoundaryRecorded: true,
	}
	r := Score(in)
	if r.PreCapScore != 45 {
		t.Fatalf("W3 pre-cap = %d, want 45 (breakdown %+v)", r.PreCapScore, r.Breakdown)
	}
	if r.ValueState != enums.HypothesisOnly {
		t.Fatalf("W3 state = %s, want HypothesisOnly", r.ValueState)
	}
}

// Property: score is always within [0,100] and post-cap never exceeds pre-cap.
func TestScoreBounds(t *testing.T) {
	inputs := []Input{{}, strongInput()}
	// permutation sweep over key dimensions
	for _, auth := range []enums.Authority{enums.Authoritative, enums.Secondary, enums.AuthorityUnknown} {
		for _, fr := range []enums.Freshness{enums.Fresh, enums.Aging, enums.Stale, enums.FreshnessUnknown} {
			for _, att := range []enums.AttributionMethod{enums.DirectMeasurement, enums.BeforeAfter, enums.ExpertJudgement, enums.AttributionUnknown} {
				for _, nv := range []enums.NetValueCheck{enums.NetPositive, enums.NetNegative, enums.NetUnknown, enums.NetNotApplicable} {
					in := strongInput()
					in.EvidenceAuthority = auth
					in.EvidenceFreshness = fr
					in.Attribution = att
					in.NetValue = nv
					inputs = append(inputs, in)
				}
			}
		}
	}
	for i, in := range inputs {
		r := Score(in)
		if r.PreCapScore < 0 || r.PreCapScore > 100 || r.Score < 0 || r.Score > 100 {
			t.Fatalf("case %d: scores out of range: pre=%d post=%d", i, r.PreCapScore, r.Score)
		}
		if r.Score > r.PreCapScore {
			t.Fatalf("case %d: post-cap %d exceeds pre-cap %d", i, r.Score, r.PreCapScore)
		}
		if r.ValueState == "" {
			t.Fatalf("case %d: state mapping not total", i)
		}
	}
}

// Publication cap gates publication eligibility only; evidential score is unaffected.
func TestPublicationCapSeparation(t *testing.T) {
	in := strongInput()
	in.ArtifactState = enums.ArtDraft
	r := Score(in)
	if r.Score != 100 {
		t.Fatalf("evidential score = %d, want 100 (publication cap must not bind it)", r.Score)
	}
	if r.PublicationScore != 89 {
		t.Fatalf("publication score = %d, want 89", r.PublicationScore)
	}
}

func TestLowerIsBetterProgress(t *testing.T) {
	in := strongInput() // baseline 100 → target 50, current 50: progress 1.0
	p, ok := Progress(in)
	if !ok || p != 1.0 {
		t.Fatalf("progress = %v ok=%v, want 1.0", p, ok)
	}
}

func TestSustainmentEvaluation(t *testing.T) {
	pass := SustainmentCheck{MeasuredBenefit: f(90), TargetValue: 100}
	fail := SustainmentCheck{MeasuredBenefit: f(70), TargetValue: 100} // below 80%
	missing := SustainmentCheck{TargetValue: 100}                      // missing snapshot = failed

	cases := []struct {
		history []SustainmentCheck
		want    enums.SustainmentStatus
	}{
		{nil, enums.SustainNotStarted},
		{[]SustainmentCheck{pass, pass}, enums.SustainOk},
		{[]SustainmentCheck{pass, fail}, enums.SustainAtRisk},
		{[]SustainmentCheck{fail, fail}, enums.SustainRegressed},
		{[]SustainmentCheck{fail, pass, fail}, enums.SustainAtRisk}, // non-consecutive
		{[]SustainmentCheck{pass, fail, missing}, enums.SustainRegressed},
	}
	for i, c := range cases {
		if got := EvaluateSustainment(c.history); got != c.want {
			t.Fatalf("case %d: got %s want %s", i, got, c.want)
		}
	}
}

func TestOwnerAdjustedThreshold(t *testing.T) {
	c := SustainmentCheck{MeasuredBenefit: f(70), TargetValue: 100, Threshold: 0.65}
	if c.Failed() {
		t.Fatal("70 vs owner threshold 65% of 100 should pass")
	}
}
