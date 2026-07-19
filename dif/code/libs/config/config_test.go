package config

import (
	"log/slog"
	"strings"
	"testing"
)

func TestLoadFromMapRequiresAllFieldsExplicitly(t *testing.T) {
	t.Parallel()

	_, err := LoadFromMap(map[string]string{
		EnvProjectID: "project-a",
	})
	if err == nil {
		t.Fatal("expected explicit missing config error")
	}

	message := err.Error()
	for _, missing := range []string{
		EnvCorpusID,
		EnvDatabaseURL,
		EnvEnvironment,
		EnvLogLevel,
		EnvAuthMode,
	} {
		if !strings.Contains(message, missing) {
			t.Fatalf("expected missing config error to include %s, got %q", missing, message)
		}
	}
}

func TestLoadFromMapParsesTypedConfig(t *testing.T) {
	t.Parallel()

	cfg, err := LoadFromMap(validValues())
	if err != nil {
		t.Fatalf("expected config to load: %v", err)
	}

	if cfg.Scope.ProjectID != "project-a" {
		t.Fatalf("unexpected project ID %q", cfg.Scope.ProjectID)
	}
	if cfg.Scope.CorpusID != "engineering-docs" {
		t.Fatalf("unexpected corpus ID %q", cfg.Scope.CorpusID)
	}
	if cfg.Environment != EnvironmentTest {
		t.Fatalf("unexpected environment %q", cfg.Environment)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Fatalf("unexpected log level %v", cfg.LogLevel)
	}
	if cfg.AuthMode != AuthModeBearerToken {
		t.Fatalf("unexpected auth mode %q", cfg.AuthMode)
	}
}

func TestLoadFromMapRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		key   string
		value string
	}{
		"environment":  {key: EnvEnvironment, value: "pilot"},
		"log-level":    {key: EnvLogLevel, value: "verbose"},
		"auth-mode":    {key: EnvAuthMode, value: "implicit_trust"},
		"database-url": {key: EnvDatabaseURL, value: "sqlite://local.db"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			values := validValues()
			values[tc.key] = tc.value
			if _, err := LoadFromMap(values); err == nil {
				t.Fatalf("expected %s=%q to be rejected", tc.key, tc.value)
			}
		})
	}
}

func TestRedactedDatabaseURLDoesNotExposePassword(t *testing.T) {
	t.Parallel()

	cfg, err := LoadFromMap(validValues())
	if err != nil {
		t.Fatalf("expected config to load: %v", err)
	}

	redacted := cfg.RedactedDatabaseURL()
	if strings.Contains(redacted, "super-secret-password") {
		t.Fatalf("redacted database URL leaked password: %s", redacted)
	}
	if !strings.Contains(redacted, "dif_user") || !strings.Contains(redacted, "REDACTED") {
		t.Fatalf("redacted database URL lost useful context: %s", redacted)
	}
}

func validValues() map[string]string {
	return map[string]string{
		EnvProjectID:   "project-a",
		EnvCorpusID:    "engineering-docs",
		EnvDatabaseURL: "postgres://dif_user:super-secret-password@localhost:5432/rif_project?sslmode=disable",
		EnvEnvironment: "test",
		EnvLogLevel:    "info",
		EnvAuthMode:    "bearer_token",
	}
}
