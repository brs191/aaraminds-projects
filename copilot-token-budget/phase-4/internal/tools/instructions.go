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

// OptimizationOpportunity summarizes one file-level reduction candidate.
type OptimizationOpportunity struct {
	Name                       string  `json:"name"`
	FilePath                   string  `json:"filePath"`
	Scope                      string  `json:"scope"`
	CurrentTokens              int64   `json:"currentTokens"`
	TargetTokens               int64   `json:"targetTokens"`
	ReducibleTokens            int64   `json:"reducibleTokens"`
	PotentialCreditsPerSession float64 `json:"potentialCreditsPerSession"`
	Priority                   string  `json:"priority"`
	Recommendation             string  `json:"recommendation"`
}

// OptimizationSummary reports always-loaded optimization potential.
type OptimizationSummary struct {
	AlwaysLoadedTokens         int64                     `json:"alwaysLoadedTokens"`
	TargetTokens               int64                     `json:"targetTokens"`
	ReducibleTokens            int64                     `json:"reducibleTokens"`
	CurrentCreditsPerSession   float64                   `json:"currentCreditsPerSession"`
	TargetCreditsPerSession    float64                   `json:"targetCreditsPerSession"`
	PotentialCreditsPerSession float64                   `json:"potentialCreditsPerSession"`
	Opportunities              []OptimizationOpportunity `json:"opportunities"`
}

// GetInstructionsOutput wraps the instruction file list.
type GetInstructionsOutput struct {
	Files        []InstructionInfo   `json:"files"`
	Optimization OptimizationSummary `json:"optimization"`
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

	plan := instructions.BuildOptimizationSummary(files)
	currentCredits, _ := budget.EstimateInstructionCostPerSession(plan.AlwaysLoadedTokens)
	targetCredits, _ := budget.EstimateInstructionCostPerSession(plan.TargetTokens)
	opps := make([]OptimizationOpportunity, 0, len(plan.Opportunities))
	for _, o := range plan.Opportunities {
		savedCredits, _ := budget.EstimateInstructionCostPerSession(o.ReducibleTokens)
		opps = append(opps, OptimizationOpportunity{
			Name:                       filepath.Base(o.Path),
			FilePath:                   o.Path,
			Scope:                      o.Scope,
			CurrentTokens:              o.CurrentTokens,
			TargetTokens:               o.TargetTokens,
			ReducibleTokens:            o.ReducibleTokens,
			PotentialCreditsPerSession: savedCredits,
			Priority:                   o.Priority,
			Recommendation:             o.Recommendation,
		})
	}
	// ScanWorkspace already returns results sorted by EstimatedToks descending.
	return nil, GetInstructionsOutput{
		Files: infos,
		Optimization: OptimizationSummary{
			AlwaysLoadedTokens:         plan.AlwaysLoadedTokens,
			TargetTokens:               plan.TargetTokens,
			ReducibleTokens:            plan.ReducibleTokens,
			CurrentCreditsPerSession:   currentCredits,
			TargetCreditsPerSession:    targetCredits,
			PotentialCreditsPerSession: currentCredits - targetCredits,
			Opportunities:              opps,
		},
	}, nil
}
