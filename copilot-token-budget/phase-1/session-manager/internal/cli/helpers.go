// Package cli provides shared helper utilities for the analyze and dashboard commands.
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aaraminds/copilot-session-manager/internal/session"
)

const (
	AnsiReset = "\033[0m"
	AnsiRed   = "\033[31m"
)

// FilterThisMonth returns sessions whose billing time falls in the current calendar month.
// Billing is attributed to the month a session finalizes (EndTime), falling back to
// StartTime for active sessions — see session.Session.BillingTime. Both Year and Month
// are checked to avoid a false match on the same month of a prior year.
func FilterThisMonth(sessions []session.Session) []session.Session {
	// Compare in UTC to match the analytics bucketing (which normalizes
	// BillingTime to UTC) and session.ReadThisMonth, so a session near a month
	// boundary is attributed to the same month regardless of the host timezone.
	now := time.Now().UTC()
	var result []session.Session
	for _, s := range sessions {
		bt := s.BillingTime().UTC()
		if bt.Year() == now.Year() && bt.Month() == now.Month() {
			result = append(result, s)
		}
	}
	return result
}

// ResolveWorkspaceRoot returns the absolute path for the workspace root.
// If os.Args[1] is provided it is used; otherwise os.Getwd() is the default.
func ResolveWorkspaceRoot() (string, error) {
	var raw string
	if len(os.Args) > 1 {
		raw = os.Args[1]
	} else {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		raw = wd
	}
	return filepath.Abs(raw)
}

// ResolveWorkspaceRootFrom returns the absolute workspace root from an explicit
// argument list (typically flag.Args() after flag parsing). The first positional
// argument is used when present; otherwise os.Getwd() is the default. This is the
// flag-aware sibling of ResolveWorkspaceRoot, used by commands that also accept
// flags so a flag value is never mistaken for the workspace path.
func ResolveWorkspaceRootFrom(args []string) (string, error) {
	var raw string
	if len(args) > 0 {
		raw = args[0]
	} else {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		raw = wd
	}
	return filepath.Abs(raw)
}

// Fatalf prints an error message to stderr and exits with code 1.
func Fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, AnsiRed+"ERROR: "+AnsiReset+format+"\n", args...)
	os.Exit(1)
}
