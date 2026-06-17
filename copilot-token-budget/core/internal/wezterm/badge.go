// Package wezterm updates the WezTerm terminal tab title using OSC escape sequences.
// Engineers running cmd/dashboard see their live credit budget in the tab title.
package wezterm

import (
	"encoding/base64"
	"fmt"
)

// SetBadge writes two OSC escape sequences to stdout:
//   - OSC 0: sets the tab/window title (works in most terminals)
//   - OSC 1337 SetUserVar=badge: WezTerm-specific badge overlay
//
// OSC sequences are fire-and-forget — no error return.
func SetBadge(text string) {
	// OSC 0 — tab/window title (ANSI standard, broad compatibility)
	fmt.Printf("\033]0;%s\a", text)
	// OSC 1337 — WezTerm user var badge (base64-encoded value required by WezTerm)
	fmt.Printf("\033]1337;SetUserVar=badge=%s\a",
		base64.StdEncoding.EncodeToString([]byte(text)))
}

// BudgetBadgeText returns a formatted string suitable for SetBadge.
// Example: "💰 8315/7000 cr [CRITICAL]"
func BudgetBadgeText(usedCredits float64, allowance int, status string) string {
	return fmt.Sprintf("💰 %.0f/%d cr [%s]", usedCredits, allowance, status)
}
