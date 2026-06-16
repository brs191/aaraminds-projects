package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetInstructionOverhead_IncludesOptimizationSummary(t *testing.T) {
	home := resolvedHomeDir(t)
	workspace, err := os.MkdirTemp(home, "instruction-summary-*")
	if err != nil {
		t.Skipf("cannot create workspace under home: %v", err)
	}
	defer os.RemoveAll(workspace)

	dir := filepath.Join(workspace, ".github", "instructions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir instructions dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "global.md"), []byte(strings.Repeat("x", 8000)), 0o644); err != nil {
		t.Fatalf("write global.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "small.md"), []byte(strings.Repeat("y", 1200)), 0o644); err != nil {
		t.Fatalf("write small.md: %v", err)
	}

	_, out, err := GetInstructionOverhead(context.TODO(), nil, GetInstructionsInput{WorkspacePath: workspace})
	if err != nil {
		t.Fatalf("GetInstructionOverhead: %v", err)
	}

	if len(out.Files) != 2 {
		t.Fatalf("files=%d want 2", len(out.Files))
	}
	if out.Optimization.AlwaysLoadedTokens != 2300 {
		t.Fatalf("alwaysLoadedTokens=%d want 2300", out.Optimization.AlwaysLoadedTokens)
	}
	if out.Optimization.TargetTokens != 1200 {
		t.Fatalf("targetTokens=%d want 1200", out.Optimization.TargetTokens)
	}
	if out.Optimization.ReducibleTokens != 1100 {
		t.Fatalf("reducibleTokens=%d want 1100", out.Optimization.ReducibleTokens)
	}
	if out.Optimization.PotentialCreditsPerSession <= 0 {
		t.Fatalf("potentialCreditsPerSession=%.4f want > 0", out.Optimization.PotentialCreditsPerSession)
	}
	if len(out.Optimization.Opportunities) == 0 {
		t.Fatal("expected at least one optimization opportunity")
	}
	if out.Optimization.Opportunities[0].ReducibleTokens <= 0 {
		t.Fatalf("first opportunity reducibleTokens=%d want > 0", out.Optimization.Opportunities[0].ReducibleTokens)
	}
}
