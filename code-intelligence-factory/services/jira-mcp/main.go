// Command jira-mcp is the Code Intelligence Factory's Jira MCP server.
//
// It exposes a CIF-scoped Jira tool surface over stdio (so the Go control
// plane can launch it as a subprocess) using the official MCP Go SDK. The hosted
// Atlassian server is intentionally not used — CIF owns this server so the tool
// surface is exactly what the factory needs (see ../../docs/ARCHITECTURE.md §12).
package main

import (
	"context"
	"log"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/aaraminds/code-intelligence-factory/services/jira-mcp/internal/jira"
	"github.com/aaraminds/code-intelligence-factory/services/jira-mcp/internal/tools"
)

func main() {
	// Jira Cloud auth: account email + API token (Basic). For production prefer
	// OAuth 2.0 (3LO); see internal/jira/client.go do() for the swap point.
	baseURL := mustEnv("JIRA_BASE_URL") // e.g. https://your-site.atlassian.net
	email := mustEnv("JIRA_EMAIL")
	token := mustEnv("JIRA_API_TOKEN")

	client := jira.NewClient(baseURL, email, token)

	// Instance-specific custom-field IDs that hold CIF traceability values.
	// Resolve these per Jira project (GET /rest/api/3/field) and inject via env.
	cfg := tools.Config{
		FieldUSID:   os.Getenv("JIRA_FIELD_US_ID"),   // e.g. customfield_10031 -> US- id
		FieldBRLink: os.Getenv("JIRA_FIELD_BR_LINK"), // e.g. customfield_10032 -> BR- id
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "cif-jira",
		Version: "0.1.0",
	}, nil)

	tools.Register(server, client, cfg)

	// stdio transport: control plane launches this binary and speaks MCP over
	// stdin/stdout. For a remote deployment, swap to the SDK's streamable-HTTP
	// handler and run behind the control plane's auth.
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required environment variable %s is not set", key)
	}
	return v
}
