package livebilling

import (
	"encoding/json"
	"time"
)

// Scope identifies the billing scope represented by a live billing snapshot.
type Scope string

const (
	// ScopeOrgAggregate marks the org-aggregate Copilot billing surface.
	ScopeOrgAggregate Scope = "org aggregate"
)

// Availability describes whether a live billing snapshot is current enough to use.
type Availability string

const (
	AvailabilityAvailable   Availability = "available"
	AvailabilityStale       Availability = "stale"
	AvailabilityUnavailable Availability = "unavailable"
)

// OrgBillingSnapshot is the org-level live billing metadata carried through the model.
type OrgBillingSnapshot struct {
	OrgSlug         string       `json:"orgSlug"`
	Scope           Scope        `json:"scope"`
	SourceLabel     string       `json:"sourceLabel"`
	Availability    Availability `json:"availability"`
	LastRefreshedAt time.Time    `json:"lastRefreshedAt"`
	AsOf            time.Time    `json:"asOf"`
	Credits         float64      `json:"credits,omitempty"`
	Error           string       `json:"error,omitempty"`
}

// CacheEntry persists a live billing snapshot plus the raw payload it was derived from.
type CacheEntry struct {
	Snapshot  OrgBillingSnapshot `json:"snapshot"`
	Payload   json.RawMessage    `json:"payload,omitempty"`
	CachedAt  time.Time          `json:"cachedAt"`
	ExpiresAt time.Time          `json:"expiresAt"`
}

// IsFresh reports whether the cache entry is still within its TTL.
func (e CacheEntry) IsFresh(now time.Time) bool {
	return !e.ExpiresAt.IsZero() && now.Before(e.ExpiresAt)
}
