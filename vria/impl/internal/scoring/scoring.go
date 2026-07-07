// Package scoring implements contracts/20_VRIA_Scoring_Rules_Spec.md.
// Pure and deterministic: no I/O, no LLM calls, no clock reads.
package scoring

import (
	"math"

	"github.com/aaraminds/vria/internal/enums"
)

// Input is the assessment context assembled from ValueHypothesis,
// MetricSnapshot, and EvidenceSource records (contracts/17).
type Input struct {
	// Ownership and framing
	ValueOwner        string
	Sponsor           string
	BusinessObjective string
	BenefitType       string
	Tier              enums.UseCaseTier
	PrimaryMetricID   string

	// Baseline (contracts/20 §3a baseline_quality)
	BaselineValue         *float64
	BaselinePeriodDefined bool
	BaselinePlanApproved  bool // approved plan, value not yet verified

	// Metric snapshot
	CurrentValue  *float64
	TargetValue   *float64
	LowerIsBetter bool

	// Evidence
	EvidenceAuthority   enums.Authority
	EvidenceFreshness   enums.Freshness
	AllCitationsPresent bool
	HasEvidenceSource   bool // false when the claimed metric source is unavailable (GE-005)
	OwnerAcceptedStale  bool

	// Attribution
	Attribution                     enums.AttributionMethod
	ConfoundersDocumented           int
	MaterialConfoundersUndocumented bool

	// Net value
	NetValue          enums.NetValueCheck
	NetValueRationale string // required for NotApplicable

	// Sustainment
	Sustainment enums.SustainmentStatus

	// Governance
	ApprovalBoundaryRecorded bool
	PolicyIssueUnresolved    bool // unresolved prompt-injection or policy finding
	ArtifactState            enums.ArtifactState

	// Claim context
	HasValueClaim    bool // a realized-value claim is being made or implied
	DeliveryComplete bool // PTB/PTO or production complete
}

// Breakdown holds per-component points (contracts/17 §7 score_breakdown).
type Breakdown struct {
	StrategicAlignment    int `json:"strategic_alignment"`
	BaselineQuality       int `json:"baseline_quality"`
	EvidenceQuality       int `json:"evidence_quality"`
	MetricMovement        int `json:"metric_movement"`
	AttributionConfidence int `json:"attribution_confidence"`
	NetValue              int `json:"net_value"`
	Sustainment           int `json:"sustainment"`
	GovernanceReadiness   int `json:"governance_readiness"`
}

func (b Breakdown) Total() int {
	return b.StrategicAlignment + b.BaselineQuality + b.EvidenceQuality +
		b.MetricMovement + b.AttributionConfidence + b.NetValue +
		b.Sustainment + b.GovernanceReadiness
}

// Result mirrors the score_value_realization output (contracts/09 §3.7).
type Result struct {
	PreCapScore      int
	Score            int // post-cap evidential score
	PublicationScore int // after publication-readiness cap (contracts/20 §4)
	Breakdown        Breakdown
	AppliedCaps      []string
	ValueState       enums.ValueState
	Recommendation   enums.Recommendation
	Confidence       enums.Confidence
	MissingEvidence  []string
	SecurityEvent    bool // policy/injection finding must be logged (GE-010)
}

// Score applies contracts/20 §2–§6 exactly. The model may explain a Result;
// it must never produce one.
func Score(in Input) Result {
	bd := Breakdown{
		StrategicAlignment:    strategicAlignment(in),
		BaselineQuality:       baselineQuality(in),
		EvidenceQuality:       evidenceQuality(in),
		MetricMovement:        metricMovement(in),
		AttributionConfidence: attributionConfidence(in),
		NetValue:              netValue(in),
		Sustainment:           sustainment(in),
		GovernanceReadiness:   governanceReadiness(in),
	}
	r := Result{PreCapScore: bd.Total(), Breakdown: bd}
	r.MissingEvidence = missingEvidence(in)

	// Caps (contracts/20 §4) — lowest applicable cap wins.
	capScore := r.PreCapScore
	apply := func(name string, max int) {
		r.AppliedCaps = append(r.AppliedCaps, name)
		if capScore > max {
			capScore = max
		}
	}
	if in.PolicyIssueUnresolved {
		apply("PolicyIssueUnresolved", 0)
		r.SecurityEvent = true
	}
	if in.ValueOwner == "" {
		apply("NoValueOwner", 29)
	}
	if in.PrimaryMetricID == "" {
		apply("NoPrimaryMetric", 39)
	}
	if in.BaselineValue == nil {
		apply("NoBaseline", 49)
	}
	if in.CurrentValue == nil {
		apply("NoCurrentValue", 59)
	}
	if in.EvidenceAuthority != enums.Authoritative {
		apply("EvidenceNotAuthoritative", 64)
	}
	if in.Attribution == enums.AttributionUnknown {
		apply("AttributionUnknown", 69)
	}
	if in.MaterialConfoundersUndocumented {
		apply("MaterialConfoundersUndocumented", 74)
	}
	if in.NetValue == enums.NetUnknown && enums.FinancialBenefit(in.BenefitType) {
		apply("NetValueUnknownFinancialClaim", 74)
	}
	if in.EvidenceFreshness == enums.Stale && !in.OwnerAcceptedStale {
		apply("EvidenceStale", 79)
	}
	r.Score = capScore

	// Publication-readiness cap gates publication only, never evidential trending.
	r.PublicationScore = r.Score
	if in.ArtifactState != enums.ArtApproved && in.ArtifactState != enums.ArtPublished {
		if r.PublicationScore > 89 {
			r.PublicationScore = 89
		}
	}

	r.ValueState = valueState(in)
	r.Confidence = confidence(in)
	r.Recommendation = recommend(in, r)
	return r
}

// --- Component formulas (contracts/20 §3a) ---

func strategicAlignment(in Input) int {
	switch {
	case in.BusinessObjective != "" && in.Sponsor != "":
		return 10
	case in.BusinessObjective != "":
		return 6
	}
	return 0
}

func baselineQuality(in Input) int {
	switch {
	case in.BaselineValue != nil && in.BaselinePeriodDefined && in.EvidenceAuthority == enums.Authoritative:
		return 15
	case in.BaselineValue != nil && in.BaselinePeriodDefined:
		return 10
	case in.BaselineValue != nil:
		return 6
	case in.BaselinePlanApproved:
		return 4
	}
	return 0
}

func evidenceQuality(in Input) int {
	pts := 0
	switch in.EvidenceAuthority {
	case enums.Authoritative:
		pts += 8
	case enums.Secondary:
		pts += 4
	}
	switch in.EvidenceFreshness {
	case enums.Fresh:
		pts += 8
	case enums.Aging:
		pts += 5
	case enums.Stale:
		pts += 2
	}
	if in.AllCitationsPresent {
		pts += 4
	}
	return pts
}

// Progress reports target progress in [0,1]; ok=false when not computable.
func Progress(in Input) (float64, bool) {
	if in.BaselineValue == nil || in.CurrentValue == nil || in.TargetValue == nil {
		return 0, false
	}
	den := *in.TargetValue - *in.BaselineValue
	if den == 0 {
		return 0, false
	}
	p := (*in.CurrentValue - *in.BaselineValue) / den
	if p < 0 {
		p = 0
	}
	if p > 1 {
		p = 1
	}
	return p, true
}

func metricMovement(in Input) int {
	p, ok := Progress(in)
	if !ok {
		return 0 // missing data yields zero, never an inferred value (GE-013)
	}
	return int(math.Round(20 * p))
}

func attributionConfidence(in Input) int {
	switch in.Attribution {
	case enums.DirectMeasurement, enums.ABComparison:
		return 10
	case enums.MatchedComparison:
		return 7
	case enums.BeforeAfter:
		if in.ConfoundersDocumented >= 1 {
			return 6
		}
		return 4
	case enums.ExpertJudgement, enums.ProxyMetric:
		return 3
	}
	return 0
}

func netValue(in Input) int {
	switch in.NetValue {
	case enums.NetPositive:
		return 10
	case enums.NetNotApplicable:
		if in.NetValueRationale != "" {
			return 8
		}
		return 0
	case enums.NetNeutral:
		return 5
	}
	return 0 // Unknown, Negative
}

func sustainment(in Input) int {
	switch in.Sustainment {
	case enums.SustainOk:
		return 10
	case enums.SustainNotStarted:
		return 6
	case enums.SustainAtRisk:
		return 4
	}
	return 0 // Regressed
}

func governanceReadiness(in Input) int {
	pts := 0
	if in.ApprovalBoundaryRecorded {
		pts += 3
	}
	if !in.PolicyIssueUnresolved {
		pts += 2
	}
	return pts
}

// --- Value state mapping (contracts/20 §5) ---

func valueState(in Input) enums.ValueState {
	if in.Sustainment == enums.SustainRegressed {
		return enums.Regressed
	}
	if in.ValueOwner == "" {
		return enums.NotReady
	}
	// A claim without a substantiating evidence source is Unproven (GE-005);
	// completed delivery without any metric is delivery progress, not value (GE-008).
	if in.HasValueClaim && !in.HasEvidenceSource {
		return enums.Unproven
	}
	if in.PrimaryMetricID == "" {
		if in.DeliveryComplete {
			return enums.Unproven
		}
		return enums.HypothesisOnly
	}
	if in.BaselineValue == nil {
		return enums.HypothesisOnly
	}
	if in.CurrentValue == nil || in.TargetValue == nil {
		return enums.BaselineReady
	}
	p, _ := Progress(in)
	if p >= 1 {
		if realizedEligible(in) {
			return enums.Realized
		}
		if in.NetValue == enums.NetNegative {
			return enums.NotRealized // gross target hit, net value negative (GE-015)
		}
		return enums.AtRisk // target reached but evidential conditions unmet
	}
	if in.NetValue == enums.NetNegative {
		return enums.AtRisk
	}
	if enums.FinancialBenefit(in.BenefitType) && in.NetValue == enums.NetUnknown {
		return enums.AtRisk // financial claim without net-value check cannot be OnTrack
	}
	if p > 0 {
		return enums.OnTrack
	}
	return enums.AtRisk
}

// realizedEligible enforces gate-a-value/03 §5: every condition, no exception.
func realizedEligible(in Input) bool {
	if in.BaselineValue == nil || in.CurrentValue == nil {
		return false
	}
	if in.EvidenceAuthority != enums.Authoritative {
		return false
	}
	if in.EvidenceFreshness != enums.Fresh && !(in.EvidenceFreshness == enums.Aging && in.OwnerAcceptedStale) {
		return false
	}
	if in.Attribution == enums.AttributionUnknown {
		return false
	}
	// ExpertJudgement alone is never enough for Realized (contracts/06 §5, GE-011).
	if in.Attribution == enums.ExpertJudgement {
		return false
	}
	if in.MaterialConfoundersUndocumented {
		return false
	}
	if in.NetValue != enums.NetPositive && in.NetValue != enums.NetNotApplicable {
		return false
	}
	if enums.FinancialBenefit(in.BenefitType) && in.NetValue == enums.NetNotApplicable {
		return false // financial claims need a real net-value check
	}
	if in.ArtifactState != enums.ArtApproved && in.ArtifactState != enums.ArtPublished {
		return false // no approval, no realized claim
	}
	if in.Sustainment == enums.SustainRegressed {
		return false
	}
	return true
}

// --- Confidence (contracts/20 §8) ---

func confidence(in Input) enums.Confidence {
	weak := in.EvidenceAuthority != enums.Authoritative ||
		in.Attribution == enums.AttributionUnknown ||
		in.Attribution == enums.ExpertJudgement ||
		in.Attribution == enums.ProxyMetric ||
		in.EvidenceFreshness == enums.Stale ||
		in.EvidenceFreshness == enums.FreshnessUnknown ||
		(enums.FinancialBenefit(in.BenefitType) && in.NetValue == enums.NetUnknown)
	if weak {
		return enums.Low
	}
	high := in.EvidenceAuthority == enums.Authoritative &&
		in.EvidenceFreshness == enums.Fresh &&
		in.BaselineValue != nil && in.CurrentValue != nil && in.TargetValue != nil &&
		!in.MaterialConfoundersUndocumented
	if high {
		return enums.High
	}
	return enums.Medium
}

// --- Recommendation mapping (contracts/20 §6), precedence top-down ---

func recommend(in Input, r Result) enums.Recommendation {
	switch {
	case in.PolicyIssueUnresolved:
		return enums.Defer
	case r.ValueState == enums.Regressed:
		return enums.Fix
	case in.NetValue == enums.NetNegative:
		return enums.Fix // never Scale on gross benefit alone (GE-015)
	case in.Sponsor == "" && in.Tier == enums.TierLayer:
		return enums.NeedsSponsor
	case in.PrimaryMetricID == "" || in.BaselineValue == nil:
		return enums.NeedsEvidence
	case r.ValueState == enums.Unproven:
		return enums.NeedsEvidence
	case r.Score >= 85 && r.ValueState == enums.Realized:
		return enums.Scale
	case r.Score >= 70 && r.ValueState == enums.OnTrack:
		return enums.ContinuePilot
	case r.Score >= 50:
		if len(r.MissingEvidence) > 0 {
			return enums.NeedsEvidence
		}
		return enums.Fix
	}
	return enums.NeedsEvidence
}

func missingEvidence(in Input) []string {
	var m []string
	if in.PrimaryMetricID == "" {
		m = append(m, "primary_metric_id")
	}
	if in.BaselineValue == nil {
		m = append(m, "baseline_value")
	}
	if in.CurrentValue == nil {
		m = append(m, "current_value")
	}
	if in.TargetValue == nil {
		m = append(m, "target_value")
	}
	if !in.HasEvidenceSource {
		m = append(m, "evidence_source")
	}
	if in.Attribution == enums.AttributionUnknown {
		m = append(m, "attribution_method")
	}
	if enums.FinancialBenefit(in.BenefitType) && in.NetValue == enums.NetUnknown {
		m = append(m, "initiative_cost_period")
	}
	return m
}
