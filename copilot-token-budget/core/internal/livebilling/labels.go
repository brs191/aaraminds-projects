package livebilling

import (
	"fmt"
	"time"
)

// DisplayLabel returns the UI label for a live billing snapshot.
// The visible contract stays deliberately small:
//   - nil or missing snapshot => (estimated)
//   - unavailable snapshot    => (unavailable)
//   - snapshot with SourceLabel => return the SourceLabel as-is
//   - otherwise               => (org aggregate, ~Xh ago)
func DisplayLabel(snapshot *OrgBillingSnapshot, now time.Time) string {
	if snapshot == nil {
		return "(estimated)"
	}
	if snapshot.Availability == AvailabilityUnavailable {
		return "(unavailable)"
	}
	// If SourceLabel is already set (by the refresher), use it.
	if snapshot.SourceLabel != "" {
		return snapshot.SourceLabel
	}
	// Fallback for snapshots created without explicit SourceLabel.
	hours := hoursAgo(snapshot.LastRefreshedAt, now)
	return fmt.Sprintf("(org aggregate, ~%dh ago)", hours)
}

func hoursAgo(t time.Time, now time.Time) int {
	if t.IsZero() {
		return 0
	}
	d := now.UTC().Sub(t.UTC())
	if d < 0 {
		d = 0
	}
	h := int(d.Hours())
	if h < 1 {
		return 1
	}
	return h
}
