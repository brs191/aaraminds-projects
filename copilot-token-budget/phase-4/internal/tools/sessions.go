package tools

import (
	"context"
	"fmt"
	"sort"

	"github.com/aaraminds/copilot-session-manager/internal/budget"
	"github.com/aaraminds/copilot-session-manager/internal/session"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GetSessionsInput is the input schema for the get_sessions tool.
type GetSessionsInput struct {
	WorkspacePath string `json:"workspacePath" jsonschema:"the absolute path to the workspace root"`
}

// SessionInfo is the per-session data returned by get_sessions.
type SessionInfo struct {
	Name          string  `json:"name"`
	Model         string  `json:"model"`
	Credits       float64 `json:"credits"`
	ContextTokens int64   `json:"contextTokens"`
	IsActive      bool    `json:"isActive"`
}

// GetSessionsOutput wraps the session list for consistent JSON schema.
type GetSessionsOutput struct {
	Sessions []SessionInfo `json:"sessions"`
}

// GetSessions returns all sessions for the current calendar month, each tagged
// with an isActive flag, sorted by credit consumption descending.
func GetSessions(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input GetSessionsInput,
) (*mcp.CallToolResult, GetSessionsOutput, error) {
	if err := validateWorkspacePath(input.WorkspacePath); err != nil {
		return nil, GetSessionsOutput{}, err
	}

	sessions, err := session.ReadThisMonth()
	if err != nil {
		return nil, GetSessionsOutput{}, fmt.Errorf("read sessions: %w", err)
	}

	infos := make([]SessionInfo, 0, len(sessions))
	for _, s := range sessions {
		infos = append(infos, SessionInfo{
			Name:          s.ProjectName,
			Model:         s.PrimaryModel,
			Credits:       budget.FromNanoAIU(s.TotalNanoAIU),
			ContextTokens: s.Tokens.CurrentTokens,
			IsActive:      s.IsActive,
		})
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Credits > infos[j].Credits
	})

	return nil, GetSessionsOutput{Sessions: infos}, nil
}
