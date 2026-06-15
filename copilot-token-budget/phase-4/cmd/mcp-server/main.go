// Command mcp-server is the Copilot Token Budget MCP server.
//
// It exposes six tools over stdio transport so Copilot CLI (or any MCP client)
// can query budget status, monthly sessions, instruction overhead, model costs,
// a usage timeseries, and top credit consumers.
//
// Usage:
//
//	copilot-budget-mcp [--debug]
//
// The version is injected at build time:
//
//	go build -ldflags "-X main.Version=v1.0.0" ./cmd/mcp-server
//
// IMPORTANT: stdout is the MCP protocol channel — never write to it directly.
// All diagnostic output goes to stderr (gated by --debug).
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/aaraminds/copilot-session-manager/phase4/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Version is set at build time via -ldflags "-X main.Version=<tag>".
var Version = "dev"

func main() {
	debug := flag.Bool("debug", false, "write diagnostic logs to stderr")
	flag.Parse()

	// stdout is the MCP protocol channel — never log there.
	if *debug {
		log.SetOutput(os.Stderr)
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.Printf("copilot-budget-mcp %s starting", Version)
	} else {
		log.SetOutput(io.Discard)
	}

	server := mcp.NewServer(
		&mcp.Implementation{Name: "copilot-token-budget", Version: Version},
		nil,
	)

	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "get_budget_status",
			Description: "Get current Copilot credit usage, percentage of monthly allowance, and month-end forecast. All data is read from local session files — no network call.",
		},
		tools.GetBudgetStatus,
	)

	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "get_sessions",
			Description: "Returns all Copilot sessions for the current calendar month with per-session credits and an isActive flag, sorted by credit consumption descending.",
		},
		tools.GetSessions,
	)

	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "get_instruction_overhead",
			Description: "Audit .github/instructions/ files in a workspace. Returns token counts, severity ratings, and estimated credit cost per 50-turn session.",
		},
		tools.GetInstructionOverhead,
	)

	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "get_model_costs",
			Description: "Break down Copilot credit costs by model for the current month. Returns total credits and estimated rate cards per model.",
		},
		tools.GetModelCosts,
	)

	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "get_usage_timeseries",
			Description: "Return Copilot credit and token usage bucketed over time. granularity is daily (default, current month), weekly, or monthly (weekly/monthly span full history). Each bucket has key, start (RFC3339), sessions, credits, inputTokens, outputTokens.",
		},
		tools.GetUsageTimeseries,
	)

	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "get_top_consumers",
			Description: "Return the top-N sessions, models, and projects by credit consumption for the current month (default n=5). Each row has name, credits, inputTokens, outputTokens, and model, sorted by credits descending.",
		},
		tools.GetTopConsumers,
	)

	// Run over stdio — blocks until the client disconnects.
	// Any startup work that would delay this (file scanning, etc.) is deferred
	// to the individual tool handlers to keep startup ≤ 100ms.
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		fmt.Fprintf(os.Stderr, "copilot-budget-mcp: %v\n", err)
		os.Exit(1)
	}
}
