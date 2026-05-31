// Command jira-mcp is the Go MCP server that exposes Jira Cloud read tools to the
// LangGraph orchestrator. For P0 it serves STUBBED fixture data so the rest of the
// system can run end-to-end with zero credentials. When OAuth 3LO is wired (see the
// project brain at scrum-master/planning/Open_Questions.md), the handlers in internal/tools
// are swapped for real Jira REST v3 / Agile API calls — the tool contract is unchanged.
package main

import (
	"log"
	"os"

	"github.com/aaraminds/scrum-master-agent/jira-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	s := server.NewMCPServer("jira-mcp", "0.1.0")
	tools.Register(s)

	addr := ":" + getenv("PORT", "8080")
	httpSrv := server.NewStreamableHTTPServer(s)
	log.Printf("jira-mcp listening on %s (MCP endpoint: %s/mcp)", addr, addr)
	if err := httpSrv.Start(addr); err != nil {
		log.Fatalf("jira-mcp server error: %v", err)
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	r