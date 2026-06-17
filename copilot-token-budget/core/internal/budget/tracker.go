// Package budget converts raw nanoAIU billing values into credits, dollars, and
// budget-state summaries for the AT&T Copilot Token Budget tool.
package budget

// Billing unit conversions.
const (
	NanoAIUPerCredit        int64   = 1_000_000_000
	DollarsPerCredit        float64 = 0.01
	MonthlyAllowanceCredits int     = 7_000
)

// Sonnet token pricing (credits per million tokens).
const (
	SonnetInputRate  float64 = 300
	SonnetOutputRate float64 = 1_500
)

// BudgetStatus represents the health level of the current credit usage.
type BudgetStatus string

const (
	StatusOK       BudgetStatus = "OK"       // < 60% of allowance used
	StatusWarning  BudgetStatus = "WARNING"  // 60–90% of allowance used
	StatusCritical BudgetStatus = "CRITICAL" // > 90% of allowance used
)

// BudgetState holds a fully computed budget summary for a set of sessions.
//
// The json tags pin the wire shape to camelCase so it matches the TypeScript
// BudgetState exactly (note: RemainingCredit serializes as the plural
// "remainingCredits" to match the TS field). Without these tags Go would emit
// PascalCase and diverge from the extension's exports.
type BudgetState struct {
	UsedCredits     float64      `json:"usedCredits"`
	AllowedCredits  int          `json:"allowedCredits"`
	UsedPct         float64      `json:"usedPct"`
	RemainingCredit float64      `json:"remainingCredits"`
	Status          BudgetStatus `json:"status"`
}

// FromNanoAIU converts raw nanoAIU billing units to credits.
func FromNanoAIU(nanoAIU int64) float64 {
	return float64(nanoAIU) / float64(NanoAIUPerCredit)
}

// ToDollars converts credits to US dollars.
func ToDollars(credits float64) float64 {
	return credits * DollarsPerCredit
}

// Calculate sums nanoAIUValues, converts to credits, and computes the full BudgetState.
// If allowance is <= 0, MonthlyAllowanceCredits is used instead.
func Calculate(nanoAIUValues []int64, allowance int) BudgetState {
	if allowance <= 0 {
		allowance = MonthlyAllowanceCredits
	}

	var totalNano int64
	for _, v := range nanoAIUValues {
		totalNano += v
	}

	used := FromNanoAIU(totalNano)
	allowed := float64(allowance)
	pct := used / allowed * 100
	remaining := allowed - used

	return BudgetState{
		UsedCredits:     used,
		AllowedCredits:  allowance,
		UsedPct:         pct,
		RemainingCredit: remaining,
		Status:          statusFor(pct),
	}
}

// EstimateInstructionCostPerSession estimates the credit cost of always-loaded
// instruction file tokens across a typical 50-turn session.
//
// Formula: (totalTokens * 50 * SonnetInputRate) / 1_000_000
func EstimateInstructionCostPerSession(totalTokens int64) (credits float64, dollars float64) {
	const turnsPerSession = 50
	credits = (float64(totalTokens) * turnsPerSession * SonnetInputRate) / 1_000_000
	dollars = ToDollars(credits)
	return credits, dollars
}

// statusFor returns the BudgetStatus for a given usage percentage.
func statusFor(pct float64) BudgetStatus {
	switch {
	case pct > 90:
		return StatusCritical
	case pct >= 60:
		return StatusWarning
	default:
		return StatusOK
	}
}
