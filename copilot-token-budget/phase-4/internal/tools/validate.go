// Package tools implements the six MCP tool handlers for the Copilot Token Budget
// server. Each handler is a pure function: it reads from the filesystem on every
// call and holds no mutable state, making concurrent tool calls race-free without
// explicit locking.
package tools

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aaraminds/copilot-session-manager/internal/platform"
)

// validateWorkspacePath enforces two security invariants:
//  1. The path must be absolute — rejects relative paths that could escape the
//     intended directory.
//  2. The path must be within the user's home directory — prevents path-traversal
//     attacks (e.g. workspacePath = "/etc") from exposing system files.
//
// Containment is checked on symlink-resolved paths: a path that is lexically
// inside home but symlinks outside it (e.g. ~/evil -> /etc) is rejected, closing
// a symlink-based path-traversal hole. Home is resolved too, since the home
// directory itself may be a symlink (e.g. macOS /var -> /private/var).
func validateWorkspacePath(workspacePath string) error {
	// Absolute-path check runs on the original input, before any resolution.
	if !filepath.IsAbs(workspacePath) {
		return fmt.Errorf("workspacePath must be absolute, got: %q", workspacePath)
	}

	home, err := platform.HomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	// Resolve symlinks on the workspace path. The path must exist to be scanned,
	// so an EvalSymlinks error means the target is missing or unresolvable.
	resolvedPath, err := filepath.EvalSymlinks(workspacePath)
	if err != nil {
		return fmt.Errorf("workspacePath does not exist or cannot be resolved: %w", err)
	}

	// Resolve home too; if it cannot be resolved, fall back to the unresolved home.
	resolvedHome, err := filepath.EvalSymlinks(home)
	if err != nil {
		resolvedHome = home
	}

	// filepath.Rel returns a path starting with ".." when resolvedPath is outside
	// home, which covers both lateral traversal and parent-directory attacks.
	rel, err := filepath.Rel(resolvedHome, resolvedPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("workspacePath must be within the user home directory for security")
	}
	return nil
}

// dailyBurnRate returns average credits consumed per day across sessions.
// Returns 0 when daysElapsed <= 0 to guard against division by zero.
// Inlined from phase-3/internal/forecast to avoid cross-module internal/ import.
func dailyBurnRate(sessions []sessionForBurn, daysElapsed int) float64 {
	if daysElapsed <= 0 {
		return 0
	}
	var totalNano int64
	for _, s := range sessions {
		totalNano += s.nanoAIU
	}
	return fromNanoAIU(totalNano) / float64(daysElapsed)
}

// projectedMonthEndTotal returns the projected month-end TOTAL credits:
// the credits already used plus the linear projection of additional credits
// over daysRemaining days. When daysRemaining <= 0 (month already over) the
// projection adds nothing and the already-used total is returned, so the
// forecast never collapses to 0 on the last day of the month.
func projectedMonthEndTotal(usedCredits, dailyBurn float64, daysRemaining int) float64 {
	if daysRemaining <= 0 {
		return usedCredits
	}
	return usedCredits + dailyBurn*float64(daysRemaining)
}

// fromNanoAIU converts raw nanoAIU to credits (1 credit = 1e9 nanoAIU).
func fromNanoAIU(nanoAIU int64) float64 {
	const nanoPerCredit = 1_000_000_000
	return float64(nanoAIU) / nanoPerCredit
}

// sessionForBurn is a minimal view of a session used by dailyBurnRate.
type sessionForBurn struct {
	nanoAIU int64
}
