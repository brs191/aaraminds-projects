package instructions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeInstructionsDir creates <root>/.github/instructions/ and writes named files.
func makeInstructionsDir(t *testing.T, root string, files map[string]string) {
	t.Helper()
	dir := filepath.Join(root, ".github", "instructions")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("makeInstructionsDir: %v", err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatalf("write %q: %v", name, err)
		}
	}
}

func TestScanWorkspace_WorkspaceRootFiles(t *testing.T) {
	root := t.TempDir()
	content := strings.Repeat("x", 4000) // 4000 bytes → 1000 estimated tokens
	makeInstructionsDir(t, root, map[string]string{
		"global.instructions.md": content,
	})

	files, err := ScanWorkspace(root)
	if err != nil {
		t.Fatalf("ScanWorkspace: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	f := files[0]
	if f.Scope != "workspace-root" {
		t.Errorf("Scope = %q, want workspace-root", f.Scope)
	}
	if f.EstimatedToks != 1000 {
		t.Errorf("EstimatedToks = %d, want 1000", f.EstimatedToks)
	}
	if f.Project != "" {
		t.Errorf("Project = %q, want empty for workspace-root", f.Project)
	}
}

func TestScanWorkspace_ProjectScopedFiles(t *testing.T) {
	root := t.TempDir()
	// Create a subdirectory project with its own .github/instructions/
	projDir := filepath.Join(root, "myservice")
	makeInstructionsDir(t, projDir, map[string]string{
		"service.instructions.md": strings.Repeat("y", 8000), // 2000 tokens
	})

	files, err := ScanWorkspace(root)
	if err != nil {
		t.Fatalf("ScanWorkspace: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	f := files[0]
	if f.Scope != "project-scoped" {
		t.Errorf("Scope = %q, want project-scoped", f.Scope)
	}
	if f.Project != "myservice" {
		t.Errorf("Project = %q, want myservice", f.Project)
	}
	if f.EstimatedToks != 2000 {
		t.Errorf("EstimatedToks = %d, want 2000", f.EstimatedToks)
	}
}

func TestScanWorkspace_SortedByTokensDesc(t *testing.T) {
	root := t.TempDir()
	// workspace-root: 500 tokens (2000 bytes)
	makeInstructionsDir(t, root, map[string]string{
		"small.md": strings.Repeat("a", 2000),
	})
	// project: 3000 tokens (12000 bytes)
	makeInstructionsDir(t, filepath.Join(root, "bigproject"), map[string]string{
		"big.md": strings.Repeat("b", 12000),
	})

	files, err := ScanWorkspace(root)
	if err != nil {
		t.Fatalf("ScanWorkspace: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	if files[0].EstimatedToks <= files[1].EstimatedToks {
		t.Errorf("not sorted desc: [0]=%d [1]=%d", files[0].EstimatedToks, files[1].EstimatedToks)
	}
}

func TestScanWorkspace_Deduplication(t *testing.T) {
	root := t.TempDir()
	// Create a real file
	makeInstructionsDir(t, root, map[string]string{
		"dedup.md": strings.Repeat("z", 400),
	})
	realFile := filepath.Join(root, ".github", "instructions", "dedup.md")

	// Create a subdir whose .github/instructions is a symlink to the workspace-root dir
	subDir := filepath.Join(root, "linked")
	linkedInstructions := filepath.Join(subDir, ".github", "instructions")
	if err := os.MkdirAll(filepath.Dir(linkedInstructions), 0755); err != nil {
		t.Fatal(err)
	}
	targetDir := filepath.Join(root, ".github", "instructions")
	if err := os.Symlink(targetDir, linkedInstructions); err != nil {
		t.Skipf("symlink not supported on this platform: %v", err)
	}
	_ = realFile

	files, err := ScanWorkspace(root)
	if err != nil {
		t.Fatalf("ScanWorkspace: %v", err)
	}
	// Should have exactly 1 file — the symlinked duplicate must be suppressed
	if len(files) != 1 {
		t.Errorf("expected 1 file after dedup, got %d", len(files))
		for _, f := range files {
			t.Logf("  %s (%s)", f.Path, f.Scope)
		}
	}
}

func TestScanWorkspace_EmptyRoot(t *testing.T) {
	root := t.TempDir()
	files, err := ScanWorkspace(root)
	if err != nil {
		t.Fatalf("ScanWorkspace on empty root: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files on empty root, got %d", len(files))
	}
}

func TestScanWorkspace_NonMdFilesIgnored(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".github", "instructions")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	// Write .txt and .json — should be ignored
	_ = os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("ignored"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "config.json"), []byte("{}"), 0644)
	// One valid .md
	_ = os.WriteFile(filepath.Join(dir, "valid.md"), []byte(strings.Repeat("v", 800)), 0644)

	files, err := ScanWorkspace(root)
	if err != nil {
		t.Fatalf("ScanWorkspace: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 .md file, got %d", len(files))
	}
}

func TestSavingsRecommendation(t *testing.T) {
	cases := []struct {
		toks int64
		want string
	}{
		{6000, "CRITICAL"},
		{5000, "CRITICAL"},
		{2500, "HIGH"},
		{2000, "HIGH"},
		{800, "MEDIUM"},
		{500, "MEDIUM"},
		{499, "OK"},
		{0, "OK"},
	}
	for _, c := range cases {
		f := InstructionFile{EstimatedToks: c.toks}
		got := f.SavingsRecommendation()
		if !strings.Contains(got, c.want) {
			t.Errorf("toks=%d: SavingsRecommendation()=%q, want to contain %q", c.toks, got, c.want)
		}
	}
}

func TestSeverity(t *testing.T) {
	cases := []struct {
		toks int64
		want string
	}{
		{3000, "high"},
		{2000, "high"},
		{1000, "medium"},
		{500, "medium"},
		{499, "low"},
		{0, "low"},
	}
	for _, c := range cases {
		got := Severity(c.toks)
		if got != c.want {
			t.Errorf("Severity(%d) = %q, want %q", c.toks, got, c.want)
		}
	}
}
