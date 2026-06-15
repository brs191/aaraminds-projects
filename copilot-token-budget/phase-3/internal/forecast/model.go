// Package forecast provides month-end credit burn-rate modelling for the
// Copilot Token Budget alert engine.
package forecast

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aaraminds/copilot-session-manager/internal/budget"
	"github.com/aaraminds/copilot-session-manager/internal/session"
)

// DailyBurnRate returns average credits consumed per day across sessions.
// Returns 0 when daysElapsed <= 0 to guard against division by zero.
func DailyBurnRate(sessions []session.Session, daysElapsed int) float64 {
	if daysElapsed <= 0 {
		return 0
	}
	var totalNano int64
	for _, s := range sessions {
		totalNano += s.TotalNanoAIU
	}
	return budget.FromNanoAIU(totalNano) / float64(daysElapsed)
}

// MonthEndForecast returns the additional credits expected over the remaining days
// of the month (dailyBurn × daysRemaining). Returns 0 when daysRemaining <= 0.
//
// Deprecated for card display: this is remaining-burn only and collapses to 0 on the
// last day of the month, which hides the forecast. Use ProjectedMonthEndTotal for the
// projected month-end TOTAL that callers should surface.
func MonthEndForecast(dailyBurn float64, daysRemaining int) float64 {
	if daysRemaining <= 0 {
		return 0
	}
	return dailyBurn * float64(daysRemaining)
}

// ProjectedMonthEndTotal returns the projected total credits consumed by month end:
// credits already used plus the projected burn over the remaining days. On the last
// day (daysRemaining <= 0) it returns usedCredits, so the projection never vanishes.
func ProjectedMonthEndTotal(usedCredits, dailyBurn float64, daysRemaining int) float64 {
	if daysRemaining <= 0 {
		return usedCredits
	}
	return usedCredits + dailyBurn*float64(daysRemaining)
}

// ExceedsAllowance returns true when forecast exceeds the monthly allowance.
func ExceedsAllowance(forecast float64, allowance float64) bool {
	return forecast > allowance
}

// ModelRoutingRecommendation inspects per-model costs across sessions and flags
// any model whose cost per token exceeds 2× the overall average. Returns a sorted
// slice of human-readable recommendation strings (empty if none are warranted).
//
// avgCostPerToken is in credits-per-token. Pass 0 to skip the comparison
// (no recommendations will be returned).
func ModelRoutingRecommendation(sessions []session.Session, avgCostPerToken float64) []string {
	if avgCostPerToken <= 0 || len(sessions) == 0 {
		return nil
	}

	type modelStats struct {
		totalNanoAIU int64
		totalTokens  int64
	}
	stats := make(map[string]*modelStats)

	for _, s := range sessions {
		for _, m := range s.ModelMetrics {
			if _, ok := stats[m.Model]; !ok {
				stats[m.Model] = &modelStats{}
			}
			stats[m.Model].totalNanoAIU += m.NanoAIU
			stats[m.Model].totalTokens += m.InputTokens + m.OutputTokens
		}
	}

	var recs []string
	for modelName, ms := range stats {
		if ms.totalTokens == 0 {
			continue
		}
		costPerToken := budget.FromNanoAIU(ms.totalNanoAIU) / float64(ms.totalTokens)
		if costPerToken > 2*avgCostPerToken {
			alt := cheaperAlternative(modelName)
			if alt != "" {
				recs = append(recs, fmt.Sprintf(
					"%s (%.5f cr/token) → consider %s",
					modelName, costPerToken, alt,
				))
			}
		}
	}

	sort.Strings(recs) // deterministic output
	return recs
}

// cheaperAlternative returns a suggested cheaper model for the given model name,
// or empty string if no cheaper tier is known.
func cheaperAlternative(model string) string {
	lower := strings.ToLower(model)
	switch {
	case strings.Contains(lower, "opus"):
		return "claude-haiku (lower cost per token)"
	case strings.Contains(lower, "sonnet"):
		return "claude-haiku (if output quality allows)"
	default:
		return ""
	}
}
