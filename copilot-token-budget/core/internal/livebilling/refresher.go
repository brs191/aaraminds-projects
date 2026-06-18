// Package livebilling/refresher orchestrates fetching live billing data,
// checking cache freshness, updating cache, and returning snapshots.
package livebilling

import (
	"context"
	"fmt"
	"os"
	"time"
)

// RefreshSnapshot wraps the result of a refresh attempt.
type RefreshSnapshot struct {
	Snapshot *OrgBillingSnapshot
	Error    string
	Cached   bool
}

// Refresher orchestrates cache-aware fetching of live billing data.
type Refresher struct {
	cfg    Config
	auth   AuthResolution
	cache  CacheEntry
	cached bool
}

// NewRefresher constructs a Refresher from config and auth resolution.
func NewRefresher(cfg Config, auth AuthResolution) *Refresher {
	return &Refresher{
		cfg:  cfg,
		auth: auth,
	}
}

// Refresh attempts to fetch live billing data, respecting cache freshness.
// On success, it:
//   1. Checks if cached data is still fresh; if so, returns it.
//   2. Otherwise calls Fetcher.FetchEntitlements() and updates the cache.
//   3. Returns a snapshot with source label reflecting the outcome.
//
// On error (network, auth, or config), it logs to stderr and returns
// a nil snapshot (graceful degradation to estimated mode).
func (r *Refresher) Refresh(ctx context.Context) RefreshSnapshot {
	now := time.Now()

	// If live billing is disabled or not ready, return nil immediately.
	if r.auth.Disabled {
		return RefreshSnapshot{
			Snapshot: nil,
			Error:    "live billing disabled",
			Cached:   false,
		}
	}

	if !r.auth.Ready {
		fmt.Fprintf(os.Stderr, "livebilling: auth not ready (%s); using estimated quota\n", r.auth.Mode)
		return RefreshSnapshot{
			Snapshot: nil,
			Error:    r.auth.Message,
			Cached:   false,
		}
	}

	// Try to load the existing cache.
	cached, err := LoadCache()
	if err == nil && cached.IsFresh(now) {
		// Cache is fresh; return it without fetching.
		snapshot := cached.Snapshot
		h := hoursAgo(snapshot.LastRefreshedAt, now)
		snapshot.SourceLabel = fmt.Sprintf("(authoritative, cached ~%dh ago)", h)
		return RefreshSnapshot{
			Snapshot: &snapshot,
			Error:    "",
			Cached:   true,
		}
	}

	// Cache is missing or stale. Fetch fresh data.
	fmt.Fprintf(os.Stderr, "livebilling: fetching live quota from GitHub...\n")

	fetcher := NewFetcher(r.cfg, r.auth.Token)
	quota, fetchErr := fetcher.FetchEntitlements(ctx, r.cfg.OrgSlug)

	if fetchErr != nil {
		// Fetch failed. Log and return nil (graceful degradation).
		if _, ok := fetchErr.(interface{ Timeout() bool }); ok {
			fmt.Fprintf(os.Stderr, "livebilling: GitHub API timed out; using estimated quota\n")
		} else {
			switch fetchErr.Error() {
			case fmt.Sprintf("fetcher: GitHub API returned 401"):
				fmt.Fprintf(os.Stderr, "livebilling: GitHub token invalid; using estimated quota\n")
			default:
				fmt.Fprintf(os.Stderr, "livebilling: fetch failed (%v); using estimated quota\n", fetchErr)
			}
		}
		return RefreshSnapshot{
			Snapshot: nil,
			Error:    fetchErr.Error(),
			Cached:   false,
		}
	}

	// Create a new snapshot with the fetched quota.
	snapshot := OrgBillingSnapshot{
		OrgSlug:         r.cfg.OrgSlug,
		Scope:           ScopeOrgAggregate,
		SourceLabel:     "(authoritative, live)",
		Availability:    AvailabilityAvailable,
		LastRefreshedAt: now,
		AsOf:            now,
		Credits:         float64(quota),
		Error:           "",
	}

	// Save to cache with TTL.
	ttl := time.Duration(r.cfg.CacheMaxAgeHours) * time.Hour
	cacheEntry := NewCacheEntry(snapshot, nil, ttl, now)
	if err := SaveCache(cacheEntry); err != nil {
		// Cache write failed, but we still have the fetched data, so log and continue.
		fmt.Fprintf(os.Stderr, "livebilling: cannot save cache (%v); continuing without cache\n", err)
	}

	fmt.Fprintf(os.Stderr, "livebilling: fetched live quota %d from GitHub\n", quota)

	return RefreshSnapshot{
		Snapshot: &snapshot,
		Error:    "",
		Cached:   false,
	}
}
