package tools

import (
	"context"
	"fmt"

	"github.com/aaraminds/copilot-session-manager/internal/budget"
	"github.com/aaraminds/copilot-session-manager/internal/pricing"
	"github.com/aaraminds/copilot-session-manager/internal/session"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GetModelCostsInput is the input schema for the get_model_costs tool.
type GetModelCostsInput struct {
	WorkspacePath string `json:"workspacePath" jsonschema:"the absolute path to the workspace root"`
}

// ModelCost holds the billing summary for a single model.
type ModelCost struct {
	// InputRatePer1M and OutputRatePer1M are pricing in credits per million tokens.
	// The rates come from internal/pricing (pricing.Load().RateFor), which is the
	// single source of truth for per-model rate cards and honours any user
	// pricing.json override. No external API call.
	InputRatePer1M        float64 `json:"inputRatePer1M"`
	OutputRatePer1M       float64 `json:"outputRatePer1M"`
	TotalCreditsThisMonth float64 `json:"totalCreditsThisMonth"`
	SessionCount          int     `json:"sessionCount"`
}

// GetModelCostsOutput maps model name → billing summary.
type GetModelCostsOutput struct {
	Models map[string]ModelCost `json:"models"`
}

// GetModelCosts aggregates per-model billing across all sessions in the current
// month, annotated with input/output rate cards.
func GetModelCosts(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input GetModelCostsInput,
) (*mcp.CallToolResult, GetModelCostsOutput, error) {
	if err := validateWorkspacePath(input.WorkspacePath); err != nil {
		return nil, GetModelCostsOutput{}, err
	}

	sessions, err := session.ReadThisMonth()
	if err != nil {
		return nil, GetModelCostsOutput{}, fmt.Errorf("read sessions: %w", err)
	}

	// Load the effective pricing config once per call. Load never hard-fails on a
	// missing/malformed file (it falls back to bundled defaults), so an error here
	// means the config dir itself is unresolvable — surface it rather than guess.
	cfg, err := pricing.Load()
	if err != nil {
		return nil, GetModelCostsOutput{}, fmt.Errorf("load pricing: %w", err)
	}

	type stats struct {
		totalNanoAIU int64
		sessionCount int
	}
	modelStats := make(map[string]*stats)

	for _, s := range sessions {
		for _, m := range s.ModelMetrics {
			if _, ok := modelStats[m.Model]; !ok {
				modelStats[m.Model] = &stats{}
			}
			modelStats[m.Model].totalNanoAIU += m.NanoAIU
			modelStats[m.Model].sessionCount++
		}
	}

	result := make(map[string]ModelCost, len(modelStats))
	for model, st := range modelStats {
		rate := cfg.RateFor(model)
		result[model] = ModelCost{
			InputRatePer1M:        rate.InputPerMillion,
			OutputRatePer1M:       rate.OutputPerMillion,
			TotalCreditsThisMonth: budget.FromNanoAIU(st.totalNanoAIU),
			SessionCount:          st.sessionCount,
		}
	}

	return nil, GetModelCostsOutput{Models: result}, nil
}
