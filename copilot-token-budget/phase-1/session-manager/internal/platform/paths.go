// Package platform provides cross-platform path helpers for the copilot-session-manager tool.
//
// All path construction uses filepath.Join and OS-provided home/config directory
// functions (os.UserHomeDir, os.UserConfigDir) so the same code compiles and runs
// correctly on macOS, Linux, and Windows without build tags or platform-specific files.
// runtime.GOOS is used at runtime for the single case that requires it (BinaryName).
package platform

import (
	"os"
	"path/filepath"
	"runtime"
)

// SessionStateDir returns the path to the Copilot CLI session-state directory.
//
// macOS/Linux: ~/.copilot/session-state
// Windows:     %USERPROFILE%\.copilot\session-state
func SessionStateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".copilot", "session-state"), nil
}

// ConfigDir returns the path to the copilot-token-budget config directory,
// creating it (mode 0700) if it does not exist.
//
// macOS/Linux: ~/.config/copilot-token-budget
// Windows:     %AppData%\copilot-token-budget
func ConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "copilot-token-budget")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

// BinaryName returns the platform-appropriate executable name for base.
// On Windows it appends ".exe"; on all other platforms it returns base unchanged.
func BinaryName(base string) string {
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}

// HomeDir returns the current user's home directory.
// Use this instead of os.UserHomeDir() directly so all home-dir lookups
// go through the platform package — consistent with SessionStateDir and ConfigDir.
func HomeDir() (string, error) {
	return os.UserHomeDir()
}

func WorkspaceInstructionsDir(workspaceRoot string) string {
	return filepath.Join(workspaceRoot, ".github", "instructions")
}
