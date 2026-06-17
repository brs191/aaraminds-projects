package tools

import (
	"context"
	"fmt"

	"github.com/aaraminds/copilot-token-budget/internal/analytics"
	"github.com/aaraminds/copilot-token-budget/internal/session"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// defaultTopN is the number of rows returned per ranking when n is unset.
const defaultTopN = 5

// GetTopConsumersInput is the input schema for the get_top_consumers tool.
type GetTopConsumersInput struct {
	WorkspacePath string `json:"workspacePath" jsonschema:"the absolute path to the workspace root"`
	// N is how many rows to return per ranking (default 5).
	N int `json:"n,omitempty" jsonschema:"number of rows per ranking (default 5)"`
}

// ConsumerRow is one ranked consumer (a session, model, or project).
type ConsumerRow struct {
	Name         string  `json:"name"`
	Credits      float64 `json:"credits"`
	InputTokens  int64   `json:"inputTokens"`
	OutputTokens int64   `json:"outputTokens"`
	Model        string  `json:"model"`
}

// GetTopConsumersOutput holds the three ranked lists for the current month.
type GetTopConsumersOutput struct {
	TopSessions []ConsumerRow `json:"topSessions"`
	TopModels   []ConsumerRow `json:"topModels"`
	TopProjects []ConsumerRow `json:"topProjects"`
}

// GetTopConsumers returns the top-N sessions, models, and projects by credit
// consumption for the current calendar month, using internal/analytics. Each
// list is sorted by credits descending (ties broken by name) for deterministic
// output.
func GetTopConsumers(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input GetTopConsumersInput,
) (*mcp.CallToolResult, GetTopConsumersOutput, error) {
	if err := validateWorkspacePath(input.WorkspacePath); err != nil {
		return nil, GetTopConsumersOutput{}, err
	}

	n := input.N
	if n <= 0 {
		n = defaultTopN
	}

	sessions, err := session.ReadThisMonth()
	if err != nil {
		return nil, GetTopConsumersOutput{}, fmt.Errorf("read sessions: %w", err)
	}

	return nil, GetTopConsumersOutput{
		TopSessions: toConsumerRows(analytics.TopSessions(sessions, n)),
		TopModels:   toConsumerRows(analytics.TopModels(sessions, n)),
		TopProjects: toConsumerRows(analytics.TopProjects(sessions, n)),
	}, nil
}

// toConsumerRows maps analytics.Consumer values to the tool's JSON row shape.
func toConsumerRows(in []analytics.Consumer) []ConsumerRow {
	out := make([]ConsumerRow, 0, len(in))
	for _, c := range in {
		out = append(out, ConsumerRow{
			Name:         c.Name,
			Credits:      c.Credits,
			InputTokens:  c.InputTokens,
			OutputTokens: c.OutputTokens,
			Model:        c.Model,
		})
	}
	return out
}
