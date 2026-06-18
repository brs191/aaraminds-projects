package livebilling

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestRefresher_Disabled(t *testing.T) {
	// When live billing is disabled, Refresh should return nil immediately.
	cfg := Config{
		Enabled: false,
	}
	auth := AuthResolution{
		Disabled: true,
	}
	refresher := NewRefresher(cfg, auth)

	result := refresher.Refresh(context.Background())

	if result.Snapshot != nil {
		t.Errorf("expected nil snapshot when disabled, got %v", result.Snapshot)
	}
	if result.Error != "live billing disabled" {
		t.Errorf("expected 'live billing disabled' error, got %q", result.Error)
	}
	if result.Cached {
		t.Errorf("expected cached=false, got cached=true")
	}
}

func TestRefresher_NotReady(t *testing.T) {
	// When auth is not ready, Refresh should return nil with error message.
	cfg := Config{
		Enabled: true,
	}
	auth := AuthResolution{
		Ready:    false,
		Disabled: false,
		Mode:     "missing-token",
		Message:  "token not found",
	}
	refresher := NewRefresher(cfg, auth)

	result := refresher.Refresh(context.Background())

	if result.Snapshot != nil {
		t.Errorf("expected nil snapshot when not ready, got %v", result.Snapshot)
	}
	if result.Error == "" {
		t.Errorf("expected error message, got empty string")
	}
	if result.Cached {
		t.Errorf("expected cached=false, got cached=true")
	}
}

func TestRefresher_CacheHit(t *testing.T) {
	// When cache is fresh, Refresh should use it without fetching.
	tmpDir := t.TempDir()
	oldCfgDir := os.Getenv("HOME")
	oldConfigDir := os.Getenv("XDG_CONFIG_HOME")

	// Mock the config directory.
	os.Setenv("HOME", tmpDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	t.Cleanup(func() {
		os.Setenv("HOME", oldCfgDir)
		os.Setenv("XDG_CONFIG_HOME", oldConfigDir)
	})

	// Create a fresh cache entry.
	now := time.Now()
	snapshot := OrgBillingSnapshot{
		OrgSlug:         "my-org",
		Scope:           ScopeOrgAggregate,
		SourceLabel:     "(cached)",
		Availability:    AvailabilityAvailable,
		LastRefreshedAt: now.Add(-2 * time.Hour),
		AsOf:            now.Add(-2 * time.Hour),
		Credits:         35000,
	}
	ttl := 24 * time.Hour
	cacheEntry := NewCacheEntry(snapshot, nil, ttl, now)

	if err := SaveCache(cacheEntry); err != nil {
		t.Fatalf("cannot save cache: %v", err)
	}

	cfg := Config{
		Enabled:          true,
		OrgSlug:          "my-org",
		CacheMaxAgeHours: 24,
	}
	auth := AuthResolution{
		Ready:    true,
		Disabled: false,
		Token:    "dummy-token",
	}
	refresher := NewRefresher(cfg, auth)

	result := refresher.Refresh(context.Background())

	if result.Snapshot == nil {
		t.Fatalf("expected non-nil snapshot from cache, got nil")
	}
	if result.Snapshot.Credits != 35000 {
		t.Errorf("expected cached credits 35000, got %v", result.Snapshot.Credits)
	}
	if !result.Cached {
		t.Errorf("expected cached=true, got cached=false")
	}
	if result.Error != "" {
		t.Errorf("expected no error, got %q", result.Error)
	}
}

func TestRefresher_FreshFetch(t *testing.T) {
	// When cache is stale or missing, Refresh should fetch fresh data.
	tmpDir := t.TempDir()
	oldCfgDir := os.Getenv("HOME")
	oldConfigDir := os.Getenv("XDG_CONFIG_HOME")

	os.Setenv("HOME", tmpDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	t.Cleanup(func() {
		os.Setenv("HOME", oldCfgDir)
		os.Setenv("XDG_CONFIG_HOME", oldConfigDir)
	})

	// Mock GitHub API.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/graphql" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Parse request body.
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("cannot decode request: %v", err)
		}

		// Return success response with quota 35000.
		resp := EntitlementResponse{
			Data: struct {
				Viewer struct {
					Organization struct {
						CopilotQuota int `json:"copilotQuota"`
					} `json:"organization"`
				} `json:"viewer"`
			}{},
		}
		resp.Data.Viewer.Organization.CopilotQuota = 35000

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := Config{
		Enabled:            true,
		OrgSlug:            "my-org",
		TokenEnvVar:        "TEST_TOKEN",
		CacheMaxAgeHours:   24,
		RequestTimeoutSecs: 10,
		GitHubAPIURL:       server.URL,
		DryRun:             false,
	}

	auth := AuthResolution{
		Ready:    true,
		Disabled: false,
		Token:    "test-token",
		Config:   cfg,
	}

	refresher := NewRefresher(cfg, auth)
	result := refresher.Refresh(context.Background())

	if result.Snapshot == nil {
		t.Fatalf("expected non-nil snapshot from fetch, got nil")
	}
	if result.Snapshot.Credits != 35000 {
		t.Errorf("expected fetched credits 35000, got %v", result.Snapshot.Credits)
	}
	if result.Cached {
		t.Errorf("expected cached=false for fresh fetch, got cached=true")
	}
	if result.Error != "" {
		t.Errorf("expected no error, got %q", result.Error)
	}

	// Verify cache was saved.
	cached, err := LoadCache()
	if err != nil || cached.Snapshot.Credits == 0 {
		t.Fatalf("cache not saved: %v", err)
	}
	if cached.Snapshot.Credits != 35000 {
		t.Errorf("cached credits should be 35000, got %v", cached.Snapshot.Credits)
	}
}

func TestRefresher_NetworkError(t *testing.T) {
	// When GitHub API times out or network error occurs, Refresh should
	// log an error and return nil (graceful degradation).
	tmpDir := t.TempDir()
	oldCfgDir := os.Getenv("HOME")
	oldConfigDir := os.Getenv("XDG_CONFIG_HOME")

	os.Setenv("HOME", tmpDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	t.Cleanup(func() {
		os.Setenv("HOME", oldCfgDir)
		os.Setenv("XDG_CONFIG_HOME", oldConfigDir)
	})

	// Use a server that never responds (inducing timeout).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // Will timeout.
	}))
	defer server.Close()

	cfg := Config{
		Enabled:            true,
		OrgSlug:            "my-org",
		TokenEnvVar:        "TEST_TOKEN",
		CacheMaxAgeHours:   24,
		RequestTimeoutSecs: 1, // 1 second timeout
		GitHubAPIURL:       server.URL,
		DryRun:             false,
	}

	auth := AuthResolution{
		Ready:    true,
		Disabled: false,
		Token:    "test-token",
		Config:   cfg,
	}

	refresher := NewRefresher(cfg, auth)
	result := refresher.Refresh(context.Background())

	// On network error, snapshot should be nil.
	if result.Snapshot != nil {
		t.Errorf("expected nil snapshot on network error, got %v", result.Snapshot)
	}
	if result.Error == "" {
		t.Errorf("expected error message on network error, got empty string")
	}
	if result.Cached {
		t.Errorf("expected cached=false, got cached=true")
	}
}

func TestRefresher_AuthError(t *testing.T) {
	// When GitHub API returns 401 (invalid token), Refresh should
	// log an auth error and return nil.
	tmpDir := t.TempDir()
	oldCfgDir := os.Getenv("HOME")
	oldConfigDir := os.Getenv("XDG_CONFIG_HOME")

	os.Setenv("HOME", tmpDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	t.Cleanup(func() {
		os.Setenv("HOME", oldCfgDir)
		os.Setenv("XDG_CONFIG_HOME", oldConfigDir)
	})

	// Mock GitHub API returning 401.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "Invalid token")
	}))
	defer server.Close()

	cfg := Config{
		Enabled:            true,
		OrgSlug:            "my-org",
		TokenEnvVar:        "TEST_TOKEN",
		CacheMaxAgeHours:   24,
		RequestTimeoutSecs: 10,
		GitHubAPIURL:       server.URL,
		DryRun:             false,
	}

	auth := AuthResolution{
		Ready:    true,
		Disabled: false,
		Token:    "bad-token",
		Config:   cfg,
	}

	refresher := NewRefresher(cfg, auth)
	result := refresher.Refresh(context.Background())

	// On auth error, snapshot should be nil.
	if result.Snapshot != nil {
		t.Errorf("expected nil snapshot on auth error, got %v", result.Snapshot)
	}
	if result.Error == "" {
		t.Errorf("expected error message on auth error, got empty string")
	}
	if result.Cached {
		t.Errorf("expected cached=false, got cached=true")
	}
}

func TestRefresher_OrgQuotaNotSet(t *testing.T) {
	// When GitHub API returns zero quota, Refresh should return nil.
	tmpDir := t.TempDir()
	oldCfgDir := os.Getenv("HOME")
	oldConfigDir := os.Getenv("XDG_CONFIG_HOME")

	os.Setenv("HOME", tmpDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	t.Cleanup(func() {
		os.Setenv("HOME", oldCfgDir)
		os.Setenv("XDG_CONFIG_HOME", oldConfigDir)
	})

	// Mock GitHub API returning zero quota.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := EntitlementResponse{
			Data: struct {
				Viewer struct {
					Organization struct {
						CopilotQuota int `json:"copilotQuota"`
					} `json:"organization"`
				} `json:"viewer"`
			}{},
		}
		resp.Data.Viewer.Organization.CopilotQuota = 0

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := Config{
		Enabled:            true,
		OrgSlug:            "my-org",
		TokenEnvVar:        "TEST_TOKEN",
		CacheMaxAgeHours:   24,
		RequestTimeoutSecs: 10,
		GitHubAPIURL:       server.URL,
		DryRun:             false,
	}

	auth := AuthResolution{
		Ready:    true,
		Disabled: false,
		Token:    "test-token",
		Config:   cfg,
	}

	refresher := NewRefresher(cfg, auth)
	result := refresher.Refresh(context.Background())

	// On zero quota, snapshot should be nil.
	if result.Snapshot != nil {
		t.Errorf("expected nil snapshot when quota is zero, got %v", result.Snapshot)
	}
	if result.Error == "" {
		t.Errorf("expected error message when quota is zero, got empty string")
	}
	if result.Cached {
		t.Errorf("expected cached=false, got cached=true")
	}
}

func TestRefresher_CacheStaleness(t *testing.T) {
	// When cache is older than CacheMaxAgeHours, it should be considered
	// stale and trigger a fresh fetch.
	tmpDir := t.TempDir()
	oldCfgDir := os.Getenv("HOME")
	oldConfigDir := os.Getenv("XDG_CONFIG_HOME")

	os.Setenv("HOME", tmpDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	t.Cleanup(func() {
		os.Setenv("HOME", oldCfgDir)
		os.Setenv("XDG_CONFIG_HOME", oldConfigDir)
	})

	// Create a stale cache entry (older than TTL).
	now := time.Now()
	oldTime := now.Add(-48 * time.Hour) // 48 hours old
	snapshot := OrgBillingSnapshot{
		OrgSlug:         "my-org",
		Scope:           ScopeOrgAggregate,
		Availability:    AvailabilityAvailable,
		LastRefreshedAt: oldTime,
		AsOf:            oldTime,
		Credits:         30000, // Old value
	}
	ttl := 24 * time.Hour
	cacheEntry := NewCacheEntry(snapshot, nil, ttl, oldTime)

	if err := SaveCache(cacheEntry); err != nil {
		t.Fatalf("cannot save cache: %v", err)
	}

	// Mock GitHub API to return new quota.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := EntitlementResponse{
			Data: struct {
				Viewer struct {
					Organization struct {
						CopilotQuota int `json:"copilotQuota"`
					} `json:"organization"`
				} `json:"viewer"`
			}{},
		}
		resp.Data.Viewer.Organization.CopilotQuota = 40000

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := Config{
		Enabled:            true,
		OrgSlug:            "my-org",
		CacheMaxAgeHours:   24,
		RequestTimeoutSecs: 10,
		GitHubAPIURL:       server.URL,
		DryRun:             false,
	}

	auth := AuthResolution{
		Ready:    true,
		Disabled: false,
		Token:    "test-token",
		Config:   cfg,
	}

	refresher := NewRefresher(cfg, auth)
	result := refresher.Refresh(context.Background())

	// Should fetch fresh data, not use stale cache.
	if result.Snapshot == nil {
		t.Fatalf("expected non-nil snapshot, got nil")
	}
	if result.Snapshot.Credits != 40000 {
		t.Errorf("expected fresh credentials 40000, got %v", result.Snapshot.Credits)
	}
	if result.Cached {
		t.Errorf("expected cached=false for stale cache, got cached=true")
	}
}

// BenchmarkRefresher_CacheHit measures the performance of cache hits.
func BenchmarkRefresher_CacheHit(b *testing.B) {
	tmpDir := b.TempDir()
	oldCfgDir := os.Getenv("HOME")
	oldConfigDir := os.Getenv("XDG_CONFIG_HOME")

	os.Setenv("HOME", tmpDir)
	os.Setenv("XDG_CONFIG_HOME", "")
	b.Cleanup(func() {
		os.Setenv("HOME", oldCfgDir)
		os.Setenv("XDG_CONFIG_HOME", oldConfigDir)
	})

	// Create a fresh cache entry.
	now := time.Now()
	snapshot := OrgBillingSnapshot{
		OrgSlug:         "my-org",
		Scope:           ScopeOrgAggregate,
		Availability:    AvailabilityAvailable,
		LastRefreshedAt: now,
		AsOf:            now,
		Credits:         35000,
	}
	ttl := 24 * time.Hour
	cacheEntry := NewCacheEntry(snapshot, nil, ttl, now)

	if err := SaveCache(cacheEntry); err != nil {
		b.Fatalf("cannot save cache: %v", err)
	}

	cfg := Config{
		Enabled:          true,
		OrgSlug:          "my-org",
		CacheMaxAgeHours: 24,
	}
	auth := AuthResolution{
		Ready:    true,
		Disabled: false,
		Token:    "dummy-token",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		refresher := NewRefresher(cfg, auth)
		refresher.Refresh(context.Background())
	}
}
