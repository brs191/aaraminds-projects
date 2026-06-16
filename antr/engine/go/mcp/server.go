// Command mcp is the Azure Network Topology Reviewer MCP server.
// It exposes three deterministic tools — get_topology, analyze_risks, and
// format_report — via the Model Context Protocol over stdio (default) or
// Streamable HTTP (when MCP_HTTP_PORT env-var is set).
//
// Critical wire-safety rule: under stdio transport, stdout is the JSON-RPC
// protocol wire. All logging MUST go to stderr. This file initialises slog
// with os.Stderr before any other output can occur.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aaraminds/azure-nettopo-engine/adapter"
	"github.com/aaraminds/azure-nettopo-engine/generator"
	"github.com/aaraminds/azure-nettopo-engine/internal/graph"
)

// azureTopologyFetcher wraps adapter.FetchFixture behind the TopologyFetcher
// interface so the tool handlers can be tested with a mock.
type azureTopologyFetcher struct {
	cred *azidentity.DefaultAzureCredential
}

func (f *azureTopologyFetcher) FetchFixture(
	ctx context.Context, subscriptionID string,
) (*graph.Fixture, error) {
	return adapter.FetchFixture(ctx, f.cred, subscriptionID)
}

func main() {
	// ── Logger: JSON to stderr (stdout is the MCP protocol wire). ────────────
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel(),
	}))
	slog.SetDefault(logger)

	// ── Azure credential (Managed Identity in prod, az login in dev). ────────
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		logger.Error("failed to create Azure credential", "err", err)
		os.Exit(1)
	}

	fetcher := &azureTopologyFetcher{cred: cred}

	// ── MCP server ────────────────────────────────────────────────────────────
	s := server.NewMCPServer(
		"azure-nettopo-reviewer",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	auditor := newAuditor(logger)
	registerTools(s, fetcher, logger, auditor)

	// ── Transport selection ───────────────────────────────────────────────────
	httpPort := os.Getenv("MCP_HTTP_PORT")
	if httpPort == "" {
		httpPort = os.Getenv("PORT") // Container Apps injects PORT
	}

	if httpPort != "" {
		// Streamable HTTP transport — used in Container Apps.
		addr := ":" + httpPort
		logger.Info("starting MCP server", "transport", "http", "addr", addr)
		httpSrv := server.NewStreamableHTTPServer(s)
		mux := http.NewServeMux()
		mux.Handle("/", httpSrv)
		if err := http.ListenAndServe(addr, mux); err != nil {
			logger.Error("HTTP server failed", "err", err)
			os.Exit(1)
		}
	} else {
		// stdio transport — default for local Copilot CLI / IDE integration.
		logger.Info("starting MCP server", "transport", "stdio")
		if err := server.ServeStdio(s); err != nil {
			logger.Error("stdio server failed", "err", err)
			os.Exit(1)
		}
	}
}

// registerTools wires the three MCP tools to the server with the middleware
// chain applied to each handler.
func registerTools(
	s *server.MCPServer,
	fetcher TopologyFetcher,
	logger *slog.Logger,
	auditor *Auditor,
) {
	// ── Tool 1: get_topology ─────────────────────────────────────────────────
	topologyTool := mcpgo.NewTool("get_topology",
		mcpgo.WithDescription(
			"Fetches a complete Azure network topology snapshot for a subscription. "+
				"Returns the serialised graph.Fixture as JSON. "+
				"Use this when you need raw topology data before running analysis. "+
				"Does NOT perform any risk analysis — use analyze_risks for that."),
		mcpgo.WithString("subscription_id",
			mcpgo.Required(),
			mcpgo.Description("Azure subscription GUID (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)"),
		),
	)
	s.AddTool(topologyTool,
		withMiddleware(logger, "get_topology", false, getTopologyHandler(fetcher), auditor),
	)

	// ── Tool 2: analyze_risks ─────────────────────────────────────────────────
	analyzeRisksTool := mcpgo.NewTool("analyze_risks",
		mcpgo.WithDescription(
			"Fetches topology and runs the deterministic 13-rule analysis engine. "+
				"Returns findings as JSON with a severity summary. "+
				"Use this to identify security risks in an Azure subscription. "+
				"Does NOT generate a formatted report — use format_report for that."),
		mcpgo.WithString("subscription_id",
			mcpgo.Required(),
			mcpgo.Description("Azure subscription GUID"),
		),
		mcpgo.WithString("severity_filter",
			mcpgo.Description(
				"Optional severity to filter results to. "+
					"One of: Critical, High, Medium, Low, Informational. "+
					"Omit to return all findings."),
		),
	)
	s.AddTool(analyzeRisksTool,
		withMiddleware(logger, "analyze_risks", true, analyzeRisksHandler(fetcher), auditor),
	)

	// ── Tool 3: format_report ─────────────────────────────────────────────────
	formatReportTool := mcpgo.NewTool("format_report",
		mcpgo.WithDescription(
			"Fetches topology, runs analysis, and returns a formatted report. "+
				"format=markdown returns a structured Markdown report suitable for Confluence or GitHub. "+
				"format=drawio returns Draw.io mxGraph XML suitable for diagramming. "+
				"Use this when the user asks for a report or diagram."),
		mcpgo.WithString("subscription_id",
			mcpgo.Required(),
			mcpgo.Description("Azure subscription GUID"),
		),
		mcpgo.WithString("format",
			mcpgo.Required(),
			mcpgo.Description("Output format: markdown or drawio"),
		),
	)
	s.AddTool(formatReportTool,
		withMiddleware(logger, "format_report", true, formatReportHandler(fetcher), auditor),
	)

	// ── Tool 4: generate_topology ─────────────────────────────────────────────
	generateTool := mcpgo.NewTool("generate_topology",
		mcpgo.WithDescription(
			"Generate an Azure network topology from architect intent. "+
				"Produces validated Terraform and a GitHub PR for human approval. "+
				"The topology is validated by the same engine that analyzes live deployments — "+
				"zero Critical/High/Medium findings required before a PR is created (generated "+
				"infra is held to a stricter bar than estate review)."),
		mcpgo.WithString("intent",
			mcpgo.Required(),
			mcpgo.Description(
				"Natural language description of the desired network topology (20–2000 chars). "+
					"Example: 'Hub-spoke with Azure Firewall, VPN gateway, and isolated payment subnets.'"),
		),
		mcpgo.WithString("subscription_id",
			mcpgo.Required(),
			mcpgo.Description("Azure subscription GUID for AVNM/Firewall baseline context."),
		),
		mcpgo.WithString("region",
			mcpgo.Required(),
			mcpgo.Description("Primary Azure region slug (e.g. eastus2, westus3)."),
		),
		mcpgo.WithNumber("max_iterations",
			mcpgo.Description(
				"Maximum LLM refinement iterations (1–3). Default 2. "+
					"Each iteration re-generates the spec using the previous gate failures as feedback."),
		),
	)

	registry := generator.LoadDefaultRegistry()
	var ghClient generator.GitHubClient
	if os.Getenv("GENERATOR_MODE") == "stub" || os.Getenv("GITHUB_TOKEN") == "" {
		ghClient = &generator.StubGitHubClient{}
	} else {
		ghClient = &generator.RealGitHubClient{}
	}

	s.AddTool(generateTool,
		withMiddleware(logger, "generate_topology", true,
			generateTopologyHandler(&stubSpecProvider{}, registry, ghClient, fetcher, auditor),
			auditor),
	)

	// ── Tool 5: simulate_change ───────────────────────────────────────────────
	simulateTool := mcpgo.NewTool("simulate_change",
		mcpgo.WithDescription(
			"Apply a proposed topology delta in-memory and return the security "+
				"(reachability/severity) delta BEFORE deploying. The pre-deploy wedge. "+
				"Read-only — no Azure writes; both topologies analysed by the same engine."),
		mcpgo.WithString("subscription_id", mcpgo.Required(),
			mcpgo.Description("Azure subscription GUID")),
		mcpgo.WithString("delta",
			mcpgo.Description("TopologyDelta as JSON (addNsgRule/addPublicIp/modifyRoute/addPeering/...). Omit to no-op.")),
	)
	s.AddTool(simulateTool,
		withMiddleware(logger, "simulate_change", true, simulateChangeHandler(fetcher), auditor, "delta"),
	)

	// ── Tool 6: forecast_cost ─────────────────────────────────────────────────
	forecastTool := mcpgo.NewTool("forecast_cost",
		mcpgo.WithDescription(
			"Forecast fixed (SKU-exact) + variable (flow-log estimated) cost of the "+
				"estate or a proposed delta. Read-only."),
		mcpgo.WithString("subscription_id", mcpgo.Required(),
			mcpgo.Description("Azure subscription GUID")),
		mcpgo.WithString("delta",
			mcpgo.Description("Optional TopologyDelta as JSON")),
		mcpgo.WithString("region",
			mcpgo.Description("Azure region for pricing (default eastus)")),
	)
	s.AddTool(forecastTool,
		withMiddleware(logger, "forecast_cost", true, forecastCostHandler(fetcher), auditor, "delta"),
	)
}

// logLevel reads the LOG_LEVEL env-var and returns the corresponding slog.Level.
func logLevel() slog.Level {
	switch os.Getenv("LOG_LEVEL") {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// fmtErr is a helper used in tests and non-test code to create a tool error result.
func fmtErr(format string, args ...any) *mcpgo.CallToolResult {
	return mcpgo.NewToolResultError(fmt.Sprintf(format, args...))
}
