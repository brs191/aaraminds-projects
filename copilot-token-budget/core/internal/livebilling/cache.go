package livebilling

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aaraminds/copilot-token-budget/internal/platform"
)

const cacheFileName = "live-billing-cache.json"

// CachePath returns the location of the live billing cache file.
func CachePath() (string, error) {
	dir, err := platform.ConfigDir()
	if err != nil {
		return "", fmt.Errorf("livebilling: cannot resolve config dir: %w", err)
	}
	return filepath.Join(dir, cacheFileName), nil
}

// LoadCache reads the cached live billing payload if present.
// Missing or malformed files fall back to a zero-value entry.
func LoadCache() (CacheEntry, error) {
	path, err := CachePath()
	if err != nil {
		return CacheEntry{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "livebilling: cannot read %s (%v); ignoring cache\n", path, err)
		}
		return CacheEntry{}, nil
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		fmt.Fprintf(os.Stderr, "livebilling: malformed %s (%v); ignoring cache\n", path, err)
		return CacheEntry{}, nil
	}
	return entry, nil
}

// SaveCache writes the cache entry with restrictive permissions.
func SaveCache(entry CacheEntry) error {
	path, err := CachePath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("livebilling: cannot create cache dir: %w", err)
	}
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("livebilling: cannot encode cache: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("livebilling: cannot write cache: %w", err)
	}
	return nil
}

// NewCacheEntry builds a cache entry with an absolute expiration derived from ttl.
func NewCacheEntry(snapshot OrgBillingSnapshot, payload json.RawMessage, ttl time.Duration, now time.Time) CacheEntry {
	return CacheEntry{
		Snapshot:  snapshot,
		Payload:   payload,
		CachedAt:  now.UTC(),
		ExpiresAt: now.UTC().Add(ttl),
	}
}
