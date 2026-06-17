// Package instructions scans workspace and project-level Copilot instruction files,
// estimating their token overhead so engineers can trim bloated files and reduce
// per-message credit consumption.
package instructions

import (
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/aaraminds/copilot-session-manager/internal/platform"
)

// InstructionFile describes a single Copilot instruction markdown file.
type InstructionFile struct {
	Path          string // absolute path
	Scope         string // "workspace-root" or "project-scoped"
	EstimatedToks int64  // rough estimate: len(content) / 4
	Project       string // basename of the project dir (project-scoped only)
}

// OptimizationOpportunity captures one file-level token reduction candidate.
type OptimizationOpportunity struct {
	Path            string // absolute path
	Scope           string // "workspace-root" or "project-scoped"
	CurrentTokens   int64
	TargetTokens    int64
	ReducibleTokens int64
	Priority        string // "critical" | "high" | "medium"
	Recommendation  string
}

// OptimizationSummary captures total optimization potential across scanned files.
// Totals focus on workspace-root files because they are always loaded.
type OptimizationSummary struct {
	AlwaysLoadedTokens int64
	TargetTokens       int64
	ReducibleTokens    int64
	Opportunities      []OptimizationOpportunity
}

// SavingsRecommendation returns a human-readable recommendation for this file.
func (f InstructionFile) SavingsRecommendation() string {
	switch {
	case f.EstimatedToks >= 5000:
		return "CRITICAL — split or remove; >5K tokens loaded every message"
	case f.EstimatedToks >= 2000:
		return "HIGH — trim to <2K tokens"
	case f.EstimatedToks >= 500:
		return "MEDIUM — review for unnecessary content"
	default:
		return "OK"
	}
}

// Severity returns a lowercase severity label for the given token count.
// Used by the VS Code extension for coloring/icons — no emoji.
func Severity(toks int64) string {
	switch {
	case toks >= 2000:
		return "high"
	case toks >= 500:
		return "medium"
	default:
		return "low"
	}
}

// BuildOptimizationSummary derives a deterministic optimization plan from
// instruction token counts. It prioritizes workspace-root files because they are
// loaded for every prompt in every project.
func BuildOptimizationSummary(files []InstructionFile) OptimizationSummary {
	summary := OptimizationSummary{}
	opps := make([]OptimizationOpportunity, 0, len(files))

	for _, f := range files {
		target := targetTokens(f.EstimatedToks)
		reducible := f.EstimatedToks - target
		if reducible <= 0 {
			if f.Scope == "workspace-root" {
				summary.AlwaysLoadedTokens += f.EstimatedToks
				summary.TargetTokens += f.EstimatedToks
			}
			continue
		}

		opps = append(opps, OptimizationOpportunity{
			Path:            f.Path,
			Scope:           f.Scope,
			CurrentTokens:   f.EstimatedToks,
			TargetTokens:    target,
			ReducibleTokens: reducible,
			Priority:        Severity(f.EstimatedToks),
			Recommendation:  optimizationRecommendation(f.EstimatedToks, target),
		})

		if f.Scope == "workspace-root" {
			summary.AlwaysLoadedTokens += f.EstimatedToks
			summary.TargetTokens += target
			summary.ReducibleTokens += reducible
		}
	}

	sort.Slice(opps, func(i, j int) bool {
		return opps[i].ReducibleTokens > opps[j].ReducibleTokens
	})
	summary.Opportunities = opps
	return summary
}

// ScanWorkspace scans workspaceRoot for Copilot instruction files at two levels:
//  1. <workspaceRoot>/.github/instructions/*.md  → Scope "workspace-root"
//  2. <workspaceRoot>/<subdir>/.github/instructions/*.md → Scope "project-scoped"
//
// Duplicate physical files (e.g. symlinked repos) are deduplicated via
// filepath.EvalSymlinks. Results are sorted by EstimatedToks descending.
func ScanWorkspace(workspaceRoot string) ([]InstructionFile, error) {
	absRoot, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}) // keyed by real (symlink-resolved) path
	var results []InstructionFile

	// Level 1: workspace-root instruction files.
	wsInstructionsDir := platform.WorkspaceInstructionsDir(absRoot)
	scanDir(wsInstructionsDir, "workspace-root", "", seen, &results)

	// Level 2: one level of subdirectories — each may have its own .github/instructions/.
	entries, err := os.ReadDir(absRoot)
	if err != nil {
		return results, nil // workspace root unreadable at subdir level — return what we have
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subdir := filepath.Join(absRoot, entry.Name())
		projInstructionsDir := platform.WorkspaceInstructionsDir(subdir)
		scanDir(projInstructionsDir, "project-scoped", entry.Name(), seen, &results)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].EstimatedToks > results[j].EstimatedToks
	})

	return results, nil
}

func targetTokens(toks int64) int64 {
	switch {
	case toks >= 5000:
		return 1200
	case toks >= 2000:
		return 900
	case toks >= 500:
		return 400
	default:
		return toks
	}
}

func optimizationRecommendation(current, target int64) string {
	switch {
	case current >= 5000:
		return "split into scoped files and keep shared core concise"
	case current >= 2000:
		return "compress rules to bullets and remove duplicated guidance"
	default:
		_ = target
		return "trim examples and move rarely used guidance to on-demand docs"
	}
}

// scanDir scans dir for *.md files and appends non-duplicate InstructionFiles to results.
func scanDir(dir, scope, project string, seen map[string]struct{}, results *[]InstructionFile) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return // directory absent or unreadable — silently skip
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".md" {
			continue
		}

		absPath := filepath.Join(dir, entry.Name())

		// Resolve symlinks for deduplication.
		realPath, err := filepath.EvalSymlinks(absPath)
		if err != nil {
			log.Printf("instructions: cannot resolve symlink %q: %v", absPath, err)
			continue
		}
		if _, dup := seen[realPath]; dup {
			continue
		}
		seen[realPath] = struct{}{}

		content, err := os.ReadFile(absPath)
		if err != nil {
			log.Printf("instructions: cannot read %q: %v", absPath, err)
			continue
		}

		*results = append(*results, InstructionFile{
			Path:          absPath,
			Scope:         scope,
			EstimatedToks: int64(len(content)) / 4,
			Project:       project,
		})
	}
}
