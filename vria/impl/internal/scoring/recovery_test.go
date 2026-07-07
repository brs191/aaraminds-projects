package scoring

import (
	"testing"

	"github.com/aaraminds/vria/internal/enums"
)

// Fix #7: Regressed is not a permanent latch — a passing check after a
// regression recovers the claim.
func TestSustainmentRecoversAfterRegression(t *testing.T) {
	fail := SustainmentCheck{MeasuredBenefit: fptr(70), TargetValue: 100}
	pass := SustainmentCheck{MeasuredBenefit: fptr(95), TargetValue: 100}
	cases := []struct {
		name    string
		history []SustainmentCheck
		want    enums.SustainmentStatus
	}{
		{"two fails regress", []SustainmentCheck{fail, fail}, enums.SustainRegressed},
		{"pass after regress recovers", []SustainmentCheck{fail, fail, pass}, enums.SustainOk},
		{"regress then pass then single fail", []SustainmentCheck{fail, fail, pass, fail}, enums.SustainAtRisk},
	}
	for _, c := range cases {
		if got := EvaluateSustainment(c.history); got != c.want {
			t.Fatalf("%s: got %s want %s", c.name, got, c.want)
		}
	}
}

// Fix #14: an exactly-at-threshold measurement passes (no float-rounding
// spurious failure).
func TestSustainmentThresholdBoundary(t *testing.T) {
	c := SustainmentCheck{MeasuredBenefit: fptr(0.3), TargetValue: 3, Threshold: 0.1}
	if c.Failed() {
		t.Fatal("0.3 vs 10% of 3 (=0.3) must pass, not fail on float rounding")
	}
}

// Fix #11: non-finite metric inputs are not computable and never reach the
// score as a garbage value.
func TestProgressRejectsNonFinite(t *testing.T) {
	inf := 1.0
	for i := 0; i < 400; i++ {
		inf *= 10
	} // +Inf
	nan := inf - inf // NaN
	in := Input{BaselineValue: fptr(0), CurrentValue: &nan, TargetValue: fptr(100)}
	if _, ok := Progress(in); ok {
		t.Fatal("NaN current value must be non-computable")
	}
	in.CurrentValue = &inf
	if _, ok := Progress(in); ok {
		t.Fatal("Inf current value must be non-computable")
	}
	// And the full score stays in range.
	r := Score(Input{ValueOwner: "o", PrimaryMetricID: "m",
		BaselineValue: fptr(0), CurrentValue: &nan, TargetValue: fptr(100)})
	if r.Score < 0 || r.Score > 100 || r.Breakdown.MetricMovement != 0 {
		t.Fatalf("NaN leaked into score: %+v", r)
	}
}

func fptr(v float64) *float64 { return &v }

// Fix #6: the Gate A intake score exists and sums per contracts/20 §2.
func TestGateAScore(t *testing.T) {
	full := GateAInput{
		ValueOwnerNamed: true, DeliveryOwnerNamed: true, SponsorResolved: true,
		ScopeAndTierClear: true, ExpectedBenefitStated: true, PrimaryMetricNamed: true,
		BaselineVerified: true, TargetAndWindowDefined: true, EvidenceSourceNamed: true,
		ApprovalBoundaryLogged: true, DependenciesIdentified: true,
	}
	if got := GateAScore(full); got != 100 {
		t.Fatalf("full intake = %d, want 100", got)
	}
	// Planned baseline scores 8, not 15 (v1.2.1 split).
	planned := GateAInput{BaselineVerified: false, BaselinePlanApproved: true}
	if got := GateAScore(planned); got != 8 {
		t.Fatalf("planned baseline only = %d, want 8", got)
	}
	if GateAScore(GateAInput{}) != 0 {
		t.Fatal("empty intake must score 0")
	}
}
