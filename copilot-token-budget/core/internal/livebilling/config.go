// Package livebilling loads the optional Phase 8 live billing config and
// resolves the admin-provided auth token from the environment.
//
// The feature is opt-in and default-off. Missing or malformed config falls back
// to the bundled defaults so local-only behavior stays intact.
package livebilling

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aaraminds/copilot-token-budget/internal/platform"
)

const (
	configFileName      = "config.json"
	defaultTokenEnvVar  = "COPILOT_BILLING_TOKEN"
	defaultCacheMaxAge  = 24
	defaultRequestSecs  = 10
	defaultGitHubAPIURL = "https://api.github.com"
)

// Config is the live billing opt-in surface stored in config.json.
type Config struct {
	Enabled            bool   `json:"enabled"`
	OrgSlug            string `json:"orgSlug"`
	TokenEnvVar        string `json:"tokenEnvVar"`
	CacheMaxAgeHours   int    `json:"cacheMaxAgeHours"`
	RequestTimeoutSecs int    `json:"requestTimeoutSecs"`
	GitHubAPIURL       string `json:"gitHubAPIUrl"`
	DryRun             bool   `json:"dryRun"`
}

// AuthResolution is the resolved auth/config state for later fetch steps.
type AuthResolution struct {
	Config   Config
	Mode     string
	Token    string
	Ready    bool
	Message  string
	HasToken bool
	DryRun   bool
	Disabled bool
}

// Default returns the bundled default config.
func Default() Config {
	return Config{
		Enabled:            false,
		OrgSlug:            "",
		TokenEnvVar:        defaultTokenEnvVar,
		CacheMaxAgeHours:   defaultCacheMaxAge,
		RequestTimeoutSecs: defaultRequestSecs,
		GitHubAPIURL:       defaultGitHubAPIURL,
		DryRun:             false,
	}
}

// Load returns the effective live billing config from platform.ConfigDir()/config.json.
// Missing or malformed files fall back to defaults; only config-dir resolution can fail hard.
func Load() (Config, error) {
	cfg := Default()

	dir, err := platform.ConfigDir()
	if err != nil {
		return cfg, fmt.Errorf("livebilling: cannot resolve config dir: %w", err)
	}
	path := filepath.Join(dir, configFileName)

	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "livebilling: cannot read %s (%v); using bundled defaults\n", path, err)
		}
		return cfg, nil
	}

	var override Config
	if err := json.Unmarshal(data, &override); err != nil {
		fmt.Fprintf(os.Stderr, "livebilling: malformed %s (%v); using bundled defaults\n", path, err)
		return cfg, nil
	}

	return mergeOver(cfg, override), nil
}

// ResolveAuth evaluates the config against the environment and returns the
// readiness state for a later billing fetch.
func ResolveAuth(cfg Config, lookupEnv func(string) (string, bool)) AuthResolution {
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}

	out := AuthResolution{
		Config: cfg,
		DryRun: cfg.DryRun,
	}

	if !cfg.Enabled {
		out.Mode = "disabled"
		out.Disabled = true
		out.Message = "live billing disabled"
		return out
	}

	if strings.TrimSpace(cfg.OrgSlug) == "" {
		out.Mode = "config-error"
		out.Message = "live billing enabled but orgSlug is empty"
		return out
	}

	tokenEnvVar := cfg.TokenEnvVar
	if strings.TrimSpace(tokenEnvVar) == "" {
		tokenEnvVar = defaultTokenEnvVar
	}
	out.Config.TokenEnvVar = tokenEnvVar

	if cfg.DryRun {
		out.Mode = "dry-run"
		out.Message = "live billing dry-run; no HTTP requests will be made"
		return out
	}

	token, ok := lookupEnv(tokenEnvVar)
	if !ok || strings.TrimSpace(token) == "" {
		out.Mode = "missing-token"
		out.Message = fmt.Sprintf("live billing enabled but env var %s is not set", tokenEnvVar)
		return out
	}

	out.Mode = "ready"
	out.Token = token
	out.HasToken = true
	out.Ready = true
	out.Message = "live billing auth ready"
	return out
}

func mergeOver(base, override Config) Config {
	out := base

	if override.Enabled {
		out.Enabled = true
	}
	if strings.TrimSpace(override.OrgSlug) != "" {
		out.OrgSlug = override.OrgSlug
	}
	if strings.TrimSpace(override.TokenEnvVar) != "" {
		out.TokenEnvVar = override.TokenEnvVar
	}
	if override.CacheMaxAgeHours > 0 {
		out.CacheMaxAgeHours = clamp(override.CacheMaxAgeHours, 1, 72)
	}
	if override.RequestTimeoutSecs > 0 {
		out.RequestTimeoutSecs = clamp(override.RequestTimeoutSecs, 5, 60)
	}
	if strings.TrimSpace(override.GitHubAPIURL) != "" {
		out.GitHubAPIURL = override.GitHubAPIURL
	}
	if override.DryRun {
		out.DryRun = true
	}

	return out
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
