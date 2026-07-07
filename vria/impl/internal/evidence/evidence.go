// Package evidence implements freshness cadence, gap detection, and
// conflict resolution from gate-b-behavior/06 (§8 reporting windows,
// §9 conflict rules).
package evidence

import (
	"time"

	"github.com/aaraminds/vria/internal/enums"
)

// FreshnessFor classifies a snapshot against the metric's reporting window
// (gate-b-behavior/06 §8): within current window = Fresh, 1 missed window =
// Aging, 2+ = Stale.
func FreshnessFor(snapshotAt, now time.Time, reportingWindow time.Duration) enums.Freshness {
	if reportingWindow <= 0 || snapshotAt.IsZero() {
		return enums.FreshnessUnknown
	}
	age := now.Sub(snapshotAt)
	switch {
	case age <= reportingWindow:
		return enums.Fresh
	case age <= 2*reportingWindow:
		return enums.Aging
	}
	return enums.Stale
}

// NextSustainmentCheck is always computable: last check + reporting window.
func NextSustainmentCheck(lastCheck time.Time, reportingWindow time.Duration) time.Time {
	return lastCheck.Add(reportingWindow)
}

// Source is one evidence value for conflict resolution.
type Source struct {
	SourceID    string
	Value       float64
	Authority   enums.Authority
	RetrievedAt time.Time
}

// Conflict is a surfaced disagreement. Both values are always exposed —
// conflicting metrics are never averaged (06 §9).
type Conflict struct {
	Preferred Source
	Others    []Source
	Resolved  bool // false = no authoritative winner; human resolution required
}

// ResolveConflict prefers the authoritative system of record, then the newer
// source. When no single authoritative source exists, the conflict is
// surfaced unresolved.
func ResolveConflict(sources []Source) Conflict {
	if len(sources) == 0 {
		return Conflict{}
	}
	best := -1
	authCount := 0
	for i, s := range sources {
		if s.Authority == enums.Authoritative {
			authCount++
			if best == -1 || s.RetrievedAt.After(sources[best].RetrievedAt) {
				best = i
			}
		}
	}
	c := Conflict{}
	if authCount >= 1 {
		c.Preferred = sources[best]
		c.Resolved = true
		for i, s := range sources {
			if i != best {
				c.Others = append(c.Others, s)
			}
		}
		return c
	}
	// No authoritative source: prefer newest but mark unresolved.
	best = 0
	for i, s := range sources {
		if s.RetrievedAt.After(sources[best].RetrievedAt) {
			best = i
		}
	}
	c.Preferred = sources[best]
	c.Resolved = false
	for i, s := range sources {
		if i != best {
			c.Others = append(c.Others, s)
		}
	}
	return c
}
