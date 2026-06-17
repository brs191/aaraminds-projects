package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/aaraminds/copilot-token-budget/internal/budget"
	"github.com/aaraminds/copilot-token-budget/internal/pricing"
	"github.com/aaraminds/copilot-token-budget/internal/session"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GetBudgetInput is the input schema for the get_budget_status tool.
type GetBudgetInput struct {
	WorkspacePath string `json:"workspacePath" jsonschema:"the absolute path to the workspace root"`
}

// GetBudgetOutput is the output schema for the get_budget_status tool.
// Fields match the cmd/analyze report exactly so integration tests can cross-check.
type GetBudgetOutput struct {
	Credits   float64 `json:"credits"`
	Pct       float64 `json:"pct"`
	Allowance int     `json:"allowance"`
	Status    string  `json:"status"`
	DaysLeft  int     `json:"daysLeft"`
	// Forecast is the projected month-end total credits: credits used so far plus
	// the linear projection of the current daily burn over the remaining days.
	Forecast float64 `json:"forecast"`
	// PremiumRequests is the total count of premium (paid-tier) requests made
	// across this month's settled sessions, from session.shutdown.totalPremiumRequests.
	PremiumRequests int64 `json:"premiumRequests"`
}

// GetBudgetStatus returns the current month's Copilot credit budget state plus
// a projected month-end total: credits used so far plus a linear projection of
// the daily burn rate (since day 1) over the remaining days of the month.
func GetBudgetStatus(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input GetBudgetInput,
) (*mcp.CallToolResult, GetBudgetOutput, error) {
	if err := validateWorkspacePath(input.WorkspacePath); err != nil {
		return nil, GetBudgetOutput{}, err
	}

	sessions, err := session.ReadThisMonth()
	if err != nil {
		return nil, GetBudgetOutput{}, fmt.Errorf("read sessions: %w", err)
	}

	// Load the effective pricing config so the allowance matches the CLI exactly.
	// Load never hard-fails on a missing/malformed file (it falls back to bundled
	// defaults); an error here means the config dir itself is unresolvable.
	cfg, err := pricing.Load()
	if err != nil {
		return nil, GetBudgetOutput{}, fmt.Errorf("load pricing: %w", err)
	}

	nanoAIUs := make([]int64, 0, len(sessions))
	burns := make([]sessionForBurn, 0, len(sessions))
	var premiumRequests int64
	for _, s := range sessions {
		premiumRequests += s.TotalPremiumRequests
		if s.TotalNanoAIU > 0 {
			nanoAIUs = append(nanoAIUs, s.TotalNanoAIU)
			burns = append(burns, sessionForBurn{nanoAIU: s.TotalNanoAIU})
		}
	}
	// Pass the configured allowance so this path matches the CLI's
	// budget.Calculate(nano, cfg.AllowanceCredits) rather than silently using the
	// hardcoded default of 7000.
	state := budget.Calculate(nanoAIUs, cfg.AllowanceCredits)

	// Use UTC for the day-of-month arithmetic so daysElapsed and lastDay are in
	// the same timezone as the analytics buckets (which bucket in UTC). Mixing a
	// local daysElapsed with a UTC lastDay skews daysLeft near month boundaries.
	today := time.Now().UTC()
	daysElapsed := today.Day()
	lastDay := time.Date(today.Year(), today.Month()+1, 0, 0, 0, 0, 0, time.UTC)
	daysLeft := lastDay.Day() - daysElapsed

	dailyBurn := dailyBurnRate(burns, daysElapsed)
	forecast := projectedMonthEndTotal(state.UsedCredits, dailyBurn, daysLeft)

	return nil, GetBudgetOutput{
		Credits:         state.UsedCredits,
		Pct:             state.UsedPct,
		Allowance:       state.AllowedCredits,
		Status:          string(state.Status),
		DaysLeft:        daysLeft,
		Forecast:        forecast,
		PremiumRequests: premiumRequests,
	}, nil
}
