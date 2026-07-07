// Package enums mirrors contracts/17_VRIA_Canonical_Schemas_and_Enums.md.
// Any change here requires a matching change in the canonical document.
package enums

type ValueState string

const (
	NotReady       ValueState = "NotReady"
	HypothesisOnly ValueState = "HypothesisOnly"
	BaselineReady  ValueState = "BaselineReady"
	OnTrack        ValueState = "OnTrack"
	AtRisk         ValueState = "AtRisk"
	Realized       ValueState = "Realized"
	NotRealized    ValueState = "NotRealized"
	Regressed      ValueState = "Regressed"
	Unproven       ValueState = "Unproven"
)

type Recommendation string

const (
	Build         Recommendation = "Build"
	ContinuePilot Recommendation = "ContinuePilot"
	Scale         Recommendation = "Scale"
	Fix           Recommendation = "Fix"
	Defer         Recommendation = "Defer"
	Rebaseline    Recommendation = "Rebaseline"
	Stop          Recommendation = "Stop"
	NeedsSponsor  Recommendation = "NeedsSponsor"
	NeedsEvidence Recommendation = "NeedsEvidence"
)

type Confidence string

const (
	High   Confidence = "High"
	Medium Confidence = "Medium"
	Low    Confidence = "Low"
)

type Freshness string

const (
	Fresh            Freshness = "Fresh"
	Aging            Freshness = "Aging"
	Stale            Freshness = "Stale"
	FreshnessUnknown Freshness = "Unknown"
)

type Authority string

const (
	Authoritative    Authority = "Authoritative"
	Secondary        Authority = "Secondary"
	AuthorityUnknown Authority = "Unknown"
)

type AttributionMethod string

const (
	DirectMeasurement  AttributionMethod = "DirectMeasurement"
	ABComparison       AttributionMethod = "A_BComparison"
	BeforeAfter        AttributionMethod = "BeforeAfter"
	MatchedComparison  AttributionMethod = "MatchedComparison"
	ExpertJudgement    AttributionMethod = "ExpertJudgement"
	ProxyMetric        AttributionMethod = "ProxyMetric"
	AttributionUnknown AttributionMethod = "Unknown"
)

type NetValueCheck string

const (
	NetPositive      NetValueCheck = "Positive"
	NetNegative      NetValueCheck = "Negative"
	NetNeutral       NetValueCheck = "Neutral"
	NetUnknown       NetValueCheck = "Unknown"
	NetNotApplicable NetValueCheck = "NotApplicable"
)

type SustainmentStatus string

const (
	SustainNotStarted SustainmentStatus = "NotStarted"
	SustainOk         SustainmentStatus = "Ok"
	SustainAtRisk     SustainmentStatus = "AtRisk"
	SustainRegressed  SustainmentStatus = "Regressed"
)

type ApprovalRequestState string

const (
	ReqDraft            ApprovalRequestState = "Draft"
	ReqSubmitted        ApprovalRequestState = "Submitted"
	ReqChangesRequested ApprovalRequestState = "ChangesRequested"
	ReqApproved         ApprovalRequestState = "Approved"
	ReqRejected         ApprovalRequestState = "Rejected"
	ReqWithdrawn        ApprovalRequestState = "Withdrawn"
)

type ArtifactState string

const (
	ArtDraft       ArtifactState = "Draft"
	ArtApproved    ArtifactState = "Approved"
	ArtPublished   ArtifactState = "Published"
	ArtSuperseded  ArtifactState = "Superseded"
	ArtInvalidated ArtifactState = "Invalidated"
)

type UseCaseTier string

const (
	TierTool         UseCaseTier = "Tool"
	TierAgent        UseCaseTier = "Agent"
	TierLayer        UseCaseTier = "Layer"
	TierUnclassified UseCaseTier = "Unclassified"
)

type DeliveryStatus string

const (
	DSDraft         DeliveryStatus = "Draft"
	DSDiscovery     DeliveryStatus = "Discovery"
	DSTraining      DeliveryStatus = "Training"
	DSPTBNotStarted DeliveryStatus = "PTB_NotStarted"
	DSPTBInProgress DeliveryStatus = "PTB_InProgress"
	DSPTBApproved   DeliveryStatus = "PTB_Approved"
	DSPTONotStarted DeliveryStatus = "PTO_NotStarted"
	DSPTOInProgress DeliveryStatus = "PTO_InProgress"
	DSPTOApproved   DeliveryStatus = "PTO_Approved"
	DSInProgress    DeliveryStatus = "InProgress"
	DSPilot         DeliveryStatus = "Pilot"
	DSProduction    DeliveryStatus = "Production"
	DSBlocked       DeliveryStatus = "Blocked"
	DSStopped       DeliveryStatus = "Stopped"
	DSUnknown       DeliveryStatus = "Unknown"
)

// FinancialBenefit reports whether a benefit type requires the net-value check
// (contracts/06 section 6: cost, productivity, and revenue claims).
func FinancialBenefit(benefitType string) bool {
	switch benefitType {
	case "Cost", "Productivity", "Revenue":
		return true
	}
	return false
}
