package scoring

// GateAInput carries the intake-readiness signals for the Gate A score
// (contracts/20 §2). These are explicit booleans because Gate A measures
// whether information is *present and declared*, not its numeric value.
type GateAInput struct {
	ValueOwnerNamed        bool
	DeliveryOwnerNamed     bool
	SponsorResolved        bool // named sponsor OR an explicitly declared gap
	ScopeAndTierClear      bool
	ExpectedBenefitStated  bool
	PrimaryMetricNamed     bool
	BaselineVerified       bool // verified and available
	BaselinePlanApproved   bool // approved plan, not yet verified
	TargetAndWindowDefined bool
	EvidenceSourceNamed    bool
	ApprovalBoundaryLogged bool
	DependenciesIdentified bool
}

// GateAScore computes the intake readiness score (contracts/20 §2, total 100).
// Baseline is scored as verified (15) OR planned-only (8), never both — the
// v1.2.1 split that separated a real baseline from an approved plan.
func GateAScore(in GateAInput) int {
	score := 0
	if in.ValueOwnerNamed {
		score += 10
	}
	if in.DeliveryOwnerNamed {
		score += 5
	}
	if in.SponsorResolved {
		score += 5
	}
	if in.ScopeAndTierClear {
		score += 10
	}
	if in.ExpectedBenefitStated {
		score += 10
	}
	if in.PrimaryMetricNamed {
		score += 15
	}
	switch {
	case in.BaselineVerified:
		score += 15
	case in.BaselinePlanApproved:
		score += 8
	}
	if in.TargetAndWindowDefined {
		score += 10
	}
	if in.EvidenceSourceNamed {
		score += 10
	}
	if in.ApprovalBoundaryLogged {
		score += 5
	}
	if in.DependenciesIdentified {
		score += 5
	}
	return score
}
