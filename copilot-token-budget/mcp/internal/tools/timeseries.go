package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/aaraminds/copilot-session-manager/internal/analytics"
	"github.com/aaraminds/copilot-session-manager/internal/session"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GetUsageTimeseriesInput is the input schema for the get_usage_timeseries tool.
type GetUsageTimeseriesInput struct {
	WorkspacePath string `json:"workspacePath" jsonschema:"the absolute path to the workspace root"`
	// Granularity selects the bucket size: "daily" (default), "weekly", or "monthly".
	Granularity string `json:"granularity,omitempty" jsonschema:"bucket size: daily (default), weekly, or monthly"`
}

// TimeseriesBucket is one aggregated time slice in the usage timeseries.
type TimeseriesBucket struct {
	// Key is the human-stable bucket label ("2006-01-02", "2006-W01", "2006-01").
	Key string `json:"key"`
	// Start is the bucket's lower time bound in RFC3339.
	Start string `json:"start"`
	// Sessions is the count of sessions attributed to the bucket.
	Sessions int `json:"sessions"`
	// Credits is total credits consumed in the bucket.
	Credits float64 `json:"credits"`
	// InputTokens / OutputTokens are token totals across the bucket's sessions.
	InputTokens  int64 `json:"inputTokens"`
	OutputTokens int64 `json:"outputTokens"`
}

// GetUsageTimeseriesOutput wraps the ordered list of buckets.
type GetUsageTimeseriesOutput struct {
	Buckets []TimeseriesBucket `json:"buckets"`
}

// GetUsageTimeseries returns Copilot credit/token usage bucketed over time using
// internal/analytics. Daily granularity covers the current calendar month;
// weekly and monthly granularities span the full available session history so
// real trends are visible across months. Buckets are sorted ascending by start.
func GetUsageTimeseries(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input GetUsageTimeseriesInput,
) (*mcp.CallToolResult, GetUsageTimeseriesOutput, error) {
	if err := validateWorkspacePath(input.WorkspacePath); err != nil {
		return nil, GetUsageTimeseriesOutput{}, err
	}

	granularity := input.Granularity
	if granularity == "" {
		granularity = "daily"
	}

	var buckets []analytics.Bucket
	switch granularity {
	case "daily":
		// Daily series is scoped to the current month — that is the budget window
		// users reason about day-to-day.
		sessions, readErr := session.ReadThisMonth()
		if readErr != nil {
			return nil, GetUsageTimeseriesOutput{}, fmt.Errorf("read sessions: %w", readErr)
		}
		buckets = analytics.DailySeries(sessions)
	case "weekly":
		// Weekly/monthly use full history so multi-month trends are real.
		sessions, readErr := session.ReadAll()
		if readErr != nil {
			return nil, GetUsageTimeseriesOutput{}, fmt.Errorf("read sessions: %w", readErr)
		}
		buckets = analytics.WeeklySeries(sessions)
	case "monthly":
		sessions, readErr := session.ReadAll()
		if readErr != nil {
			return nil, GetUsageTimeseriesOutput{}, fmt.Errorf("read sessions: %w", readErr)
		}
		buckets = analytics.MonthlySeries(sessions)
	default:
		return nil, GetUsageTimeseriesOutput{}, fmt.Errorf("invalid granularity %q: want daily, weekly, or monthly", granularity)
	}

	out := make([]TimeseriesBucket, 0, len(buckets))
	for _, b := range buckets {
		out = append(out, TimeseriesBucket{
			Key:          b.Key,
			Start:        b.Start.Format(time.RFC3339),
			Sessions:     b.Sessions,
			Credits:      b.Credits,
			InputTokens:  b.InputTokens,
			OutputTokens: b.OutputTokens,
		})
	}

	return nil, GetUsageTimeseriesOutput{Buckets: out}, nil
}
