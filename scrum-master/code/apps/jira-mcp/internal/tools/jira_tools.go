// Package tools registers the Jira read tools on the MCP server.
//
// P0 STUB: every handler returns embedded fixture JSON (fixtures/*.json) so the
// orchestrator runs with no Jira credentials. The tool *contract* (names, args,
// shape of the returned JSON) is the real contract — when OAuth 3LO lands, only
// the body of each handler changes (fixture read -> Jira REST call). Keep the
// returned JSON shape stable so the orchestrator is unaffected.
package tools

import (
	"context"
	"embed"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

//go:embed fixtures/*.json
var fixtures embed.FS

// Register adds all Jira read tools to the MCP server.
func Register(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("get_active_sprint",
			mcp.WithDescription("Return the active sprint for a Jira board (id, name, state, start/end dates, goal). STUB: returns fixture data."),
			mcp.WithString("board_id",
				mcp.Required(),
				mcp.Description("Jira board id"),
			),
		),
		getActiveSprint,
	)

	s.AddTool(
		mcp.NewTool("get_sprint_issues",
			mcp.WithDescription("Return issues for a sprint with status, assignee, blocked flag, time-tracking fields (timeoriginalestimate/timeestimate/timespent, seconds) and days-in-status. STUB: returns fixture data."),
			mcp.WithString("sprint_id",
				mcp.Required(),
				mcp.Description("Jira sprint id"),
			),
		),
		getSprintIssues,
	)
}

func getActiveSprint(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if _, err := req.RequireString("board_id"); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return fixtureResult("fixtures/active_sprint.json")
}

func getSprintIssues(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if _, err := req.RequireString("sprint_id"); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return fixtureResult("fixtures/sprint_issues.json")
}

func fixtureResult(path string) (*mcp.CallToolResult, error) {
	data, err := fixtures.ReadFile(path)
	if err != nil {
		return mcp.NewToolResultError("fixture not found: " + path), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
