package tools

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/aaraminds/copilot-session-manager/internal/budget"
	"github.com/aaraminds/copilot-session-manager/internal/instructions"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GetInstructionsInput is the input schema for the get_instruction_overhead tool.
type GetInstructionsInput struct {
	WorkspacePath string `json:"workspacePath" jsonschema:"the absolute path to the workspace root to audit"`
}

// InstructionInfo is the per-file data returned by get_instruction_overhead.
type InstructionInfo struct {
	Name                       string  `json:"name"`
	FilePath                   string  `json:"filePath"`
	Severity                   string  `json:"severity"`
	Tokens                     int64   `json:"tokens"`
	EstimatedCreditsPerSession float64 `json:"estimatedCreditsPerSession"`
}

// GetInstructionsOutput wraps the instruction file list.
type GetInstructionsOutput struct {
	Files []InstructionInfo `json:"files"`
}

// GetInstructionOverhead scans workspacePath for Copilot instruction files and
// returns their token counts, severities, and estimated credit costs per session.
// Results are sorted by token count descending (most expensive first).
func GetInstructionOverhead(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input GetInstructionsInput,
) (*mcp.CallToolResult, GetInstructionsOutput, error) {
	if err := validateWorkspacePath(input.WorkspacePath); err != nil {
		return nil, GetInstructionsOutput{}, err
	}

	files, err := instructions.ScanWorkspace(input.WorkspacePath)
	if err != nil {
		return nil, GetInstructionsOutput{}, fmt.Errorf("scan workspace: %w", err)
	}

	infos := make([]InstructionInfo, 0, len(files))
	for _, f := range files {
		credits, _ := budget.EstimateInstructionCostPerSession(f.EstimatedToks)
		infos = append(infos, InstructionInfo{
			Name:                       filepath.Base(f.Path),
			FilePath:                   f.Path,
			Severity:                   instructions.Severity(f.EstimatedToks),
			Tokens:                     f.EstimatedToks,
			EstimatedCreditsPerSession: credits,
		})
	}
	// ScanWorkspace already returns results sorted by EstimatedToks descending.
	return nil, GetInstructionsOutput{Files: infos}, nil
}
