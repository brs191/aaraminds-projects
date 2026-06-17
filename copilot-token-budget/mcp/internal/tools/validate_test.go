package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aaraminds/copilot-token-budget/internal/platform"
)

// resolvedHomeDir returns the symlink-resolved home directory used by
// validateWorkspacePath, so tests build paths that are genuinely inside the
// containment boundary regardless of how home is symlinked on the host.
func resolvedHomeDir(t *testing.T) string {
	t.Helper()
	home, err := platform.HomeDir()
	if err != nil {
		t.Skipf("cannot determine home dir: %v", err)
	}
	if resolved, err := filepath.EvalSymlinks(home); err == nil {
		return resolved
	}
	return home
}

// TestValidateWorkspacePath_SymlinkEscapeRejected covers the path-traversal
// hole: a path lexically inside home that symlinks to a location outside home
// must be rejected once symlinks are resolved.
func TestValidateWorkspacePath_SymlinkEscapeRejected(t *testing.T) {
	home := resolvedHomeDir(t)

	// Create a directory inside home and a symlink (also inside home) that points
	// outside home (to /etc, which exists on Linux/macOS CI).
	base, err := os.MkdirTemp(home, "validate-symlink-*")
	if err != nil {
		t.Skipf("cannot create temp dir in home: %v", err)
	}
	defer os.RemoveAll(base)

	link := filepath.Join(base, "escape")
	if err := os.Symlink("/etc", link); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	if err := validateWorkspacePath(link); err == nil {
		t.Fatalf("expected symlink escaping home (%s -> /etc) to be rejected", link)
	}
}

// TestValidateWorkspacePath_NormalSubdirAccepted verifies a genuine subdirectory
// of home passes validation.
func TestValidateWorkspacePath_NormalSubdirAccepted(t *testing.T) {
	home := resolvedHomeDir(t)

	dir, err := os.MkdirTemp(home, "validate-subdir-*")
	if err != nil {
		t.Skipf("cannot create temp dir in home: %v", err)
	}
	defer os.RemoveAll(dir)

	if err := validateWorkspacePath(dir); err != nil {
		t.Fatalf("expected normal home subdir %q to be accepted, got: %v", dir, err)
	}
}

// TestValidateWorkspacePath_EtcRejected verifies an absolute path outside home
// is rejected.
func TestValidateWorkspacePath_EtcRejected(t *testing.T) {
	if err := validateWorkspacePath("/etc"); err == nil {
		t.Fatal("expected /etc (outside home) to be rejected")
	}
}

// TestValidateWorkspacePath_RelativeRejected verifies relative paths are rejected
// before any symlink resolution.
func TestValidateWorkspacePath_RelativeRejected(t *testing.T) {
	err := validateWorkspacePath("relative/path")
	if err == nil {
		t.Fatal("expected relative path to be rejected")
	}
	if !strings.Contains(err.Error(), "must be absolute") {
		t.Errorf("expected absolute-path error, got: %v", err)
	}
}

// TestValidateWorkspacePath_SiblingPrefixRejected verifies the classic
// /home/user vs /home/user-evil sibling case: a sibling whose path shares a
// string prefix with home must not pass the containment check.
func TestValidateWorkspacePath_SiblingPrefixRejected(t *testing.T) {
	home := resolvedHomeDir(t)

	// Build a sibling directory next to home with the same name plus "-evil".
	parent := filepath.Dir(home)
	sibling := filepath.Join(parent, filepath.Base(home)+"-evil")

	if err := os.Mkdir(sibling, 0o755); err != nil {
		t.Skipf("cannot create sibling dir %q: %v", sibling, err)
	}
	defer os.RemoveAll(sibling)

	if err := validateWorkspacePath(sibling); err == nil {
		t.Fatalf("expected sibling %q (prefix of home %q) to be rejected", sibling, home)
	}
}

// TestValidateWorkspacePath_NonexistentRejected verifies a path that cannot be
// symlink-resolved (does not exist) is rejected with a clear error.
func TestValidateWorkspacePath_NonexistentRejected(t *testing.T) {
	home := resolvedHomeDir(t)
	missing := filepath.Join(home, "definitely-not-a-real-dir-xyz-123")

	err := validateWorkspacePath(missing)
	if err == nil {
		t.Fatalf("expected nonexistent path %q to be rejected", missing)
	}
	if !strings.Contains(err.Error(), "does not exist or cannot be resolved") {
		t.Errorf("expected resolution error, got: %v", err)
	}
}
