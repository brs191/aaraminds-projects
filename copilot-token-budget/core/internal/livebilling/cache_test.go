package livebilling

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCacheEntryFreshness(t *testing.T) {
	entry := NewCacheEntry(OrgBillingSnapshot{
		OrgSlug:         "att-enterprise",
		Scope:           ScopeOrgAggregate,
		SourceLabel:     "org aggregate, ~24h ago",
		Availability:    AvailabilityAvailable,
		LastRefreshedAt: time.Date(2026, 6, 17, 10, 0, 0, 0, time.UTC),
		AsOf:            time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC),
		Credits:         123.45,
	}, json.RawMessage(`{"billing":true}`), 24*time.Hour, time.Date(2026, 6, 17, 10, 0, 0, 0, time.UTC))

	if !entry.IsFresh(time.Date(2026, 6, 18, 9, 59, 59, 0, time.UTC)) {
		t.Fatal("expected cache entry to be fresh")
	}
	if entry.IsFresh(time.Date(2026, 6, 18, 10, 0, 1, 0, time.UTC)) {
		t.Fatal("expected cache entry to be stale")
	}
}

func TestSaveAndLoadCache(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("AppData", tmp)

	entry := NewCacheEntry(OrgBillingSnapshot{
		OrgSlug:         "att-enterprise",
		Scope:           ScopeOrgAggregate,
		SourceLabel:     "org aggregate, ~24h ago",
		Availability:    AvailabilityAvailable,
		LastRefreshedAt: time.Date(2026, 6, 17, 10, 0, 0, 0, time.UTC),
		AsOf:            time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC),
		Credits:         123.45,
	}, json.RawMessage(`{"billing":true}`), 24*time.Hour, time.Date(2026, 6, 17, 10, 0, 0, 0, time.UTC))
	if err := SaveCache(entry); err != nil {
		t.Fatalf("SaveCache: %v", err)
	}

	path, err := CachePath()
	if err != nil {
		t.Fatalf("CachePath: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected cache file at %s: %v", filepath.Clean(path), err)
	}

	loaded, err := LoadCache()
	if err != nil {
		t.Fatalf("LoadCache: %v", err)
	}
	if loaded.Snapshot.OrgSlug != "att-enterprise" {
		t.Fatalf("loaded snapshot = %+v", loaded.Snapshot)
	}
	if loaded.Snapshot.Scope != ScopeOrgAggregate {
		t.Fatalf("loaded scope = %q", loaded.Snapshot.Scope)
	}
	var payload map[string]any
	if err := json.Unmarshal(loaded.Payload, &payload); err != nil {
		t.Fatalf("payload unmarshal: %v", err)
	}
	if v, ok := payload["billing"].(bool); !ok || !v {
		t.Fatalf("loaded payload = %#v", payload)
	}
}
