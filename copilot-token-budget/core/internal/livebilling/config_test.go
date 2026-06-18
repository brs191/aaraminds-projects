package livebilling

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aaraminds/copilot-token-budget/internal/platform"
)

func useTempConfig(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("AppData", tmp)
	dir, err := platform.ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir: %v", err)
	}
	return dir
}

func TestDefault(t *testing.T) {
	c := Default()
	if c.Enabled {
		t.Fatal("Enabled = true, want false")
	}
	if c.TokenEnvVar != defaultTokenEnvVar {
		t.Fatalf("TokenEnvVar = %q, want %q", c.TokenEnvVar, defaultTokenEnvVar)
	}
	if c.CacheMaxAgeHours != defaultCacheMaxAge {
		t.Fatalf("CacheMaxAgeHours = %d, want %d", c.CacheMaxAgeHours, defaultCacheMaxAge)
	}
	if c.RequestTimeoutSecs != defaultRequestSecs {
		t.Fatalf("RequestTimeoutSecs = %d, want %d", c.RequestTimeoutSecs, defaultRequestSecs)
	}
}

func TestLoad_MergesOverDefaults(t *testing.T) {
	dir := useTempConfig(t)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}

	override := Config{
		Enabled:            true,
		OrgSlug:            "att-enterprise",
		TokenEnvVar:        "BILLING_TOKEN",
		CacheMaxAgeHours:   48,
		RequestTimeoutSecs: 20,
		DryRun:             true,
	}
	data, _ := json.Marshal(override)
	if err := os.WriteFile(filepath.Join(dir, configFileName), data, 0600); err != nil {
		t.Fatal(err)
	}

	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !c.Enabled || c.OrgSlug != "att-enterprise" {
		t.Fatalf("Load() = %+v, want merged override", c)
	}
	if c.TokenEnvVar != "BILLING_TOKEN" || c.CacheMaxAgeHours != 48 || c.RequestTimeoutSecs != 20 || !c.DryRun {
		t.Fatalf("Load() = %+v, want override values", c)
	}
}

func TestResolveAuth_Modes(t *testing.T) {
	disabled := ResolveAuth(Default(), os.LookupEnv)
	if disabled.Mode != "disabled" || !disabled.Disabled || disabled.Ready {
		t.Fatalf("disabled resolution = %+v", disabled)
	}

	missingOrg := ResolveAuth(Config{Enabled: true, TokenEnvVar: defaultTokenEnvVar, CacheMaxAgeHours: 24, RequestTimeoutSecs: 10}, os.LookupEnv)
	if missingOrg.Mode != "config-error" || missingOrg.Ready {
		t.Fatalf("missing org resolution = %+v", missingOrg)
	}

	dryRun := ResolveAuth(Config{
		Enabled:            true,
		OrgSlug:            "att-enterprise",
		TokenEnvVar:        defaultTokenEnvVar,
		CacheMaxAgeHours:   24,
		RequestTimeoutSecs: 10,
		DryRun:             true,
	}, os.LookupEnv)
	if dryRun.Mode != "dry-run" || dryRun.Ready || dryRun.HasToken {
		t.Fatalf("dry-run resolution = %+v", dryRun)
	}

	t.Setenv(defaultTokenEnvVar, "secret-token")
	ready := ResolveAuth(Config{
		Enabled:            true,
		OrgSlug:            "att-enterprise",
		TokenEnvVar:        defaultTokenEnvVar,
		CacheMaxAgeHours:   24,
		RequestTimeoutSecs: 10,
	}, os.LookupEnv)
	if ready.Mode != "ready" || !ready.Ready || !ready.HasToken || ready.Token != "secret-token" {
		t.Fatalf("ready resolution = %+v", ready)
	}
}
