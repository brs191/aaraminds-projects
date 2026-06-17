package platform_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aaraminds/copilot-token-budget/internal/platform"
)

func TestSessionStateDir(t *testing.T) {
	got, err := platform.SessionStateDir()
	if err != nil {
		t.Fatalf("SessionStateDir() error: %v", err)
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".copilot", "session-state")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	// Must not contain hardcoded separators from string concatenation
	if strings.Contains(got, "//") || strings.Contains(got, `\\`) {
		t.Errorf("path contains double separator: %q", got)
	}
}

func TestConfigDir(t *testing.T) {
	got, err := platform.ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() error: %v", err)
	}
	if got == "" {
		t.Fatal("ConfigDir() returned empty string")
	}
	// Directory must exist after the call
	info, err := os.Stat(got)
	if err != nil {
		t.Fatalf("ConfigDir() directory does not exist: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("ConfigDir() path is not a directory: %q", got)
	}
	// Must end with our app name
	if filepath.Base(got) != "copilot-token-budget" {
		t.Errorf("ConfigDir() base name = %q, want %q", filepath.Base(got), "copilot-token-budget")
	}
}

func TestBinaryName(t *testing.T) {
	name := platform.BinaryName("analyze")
	if name == "" {
		t.Fatal("BinaryName() returned empty string")
	}
	// On any platform the base name must be present
	if !strings.HasPrefix(name, "analyze") {
		t.Errorf("BinaryName(%q) = %q, want prefix %q", "analyze", name, "analyze")
	}
}

func TestWorkspaceInstructionsDir(t *testing.T) {
	root := filepath.Join("home", "user", "myproject")
	got := platform.WorkspaceInstructionsDir(root)
	want := filepath.Join(root, ".github", "instructions")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
