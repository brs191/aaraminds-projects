// Package config contains service-independent DIF runtime configuration.
//
// The package intentionally loads plain values only. It does not retrieve
// secrets from Azure, Key Vault, environment-specific services, or any live
// cloud dependency; production secret-store integration belongs to a later
// prompt after the auth and deployment posture are approved.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
)

const (
	EnvProjectID   = "DIF_PROJECT_ID"
	EnvCorpusID    = "DIF_CORPUS_ID"
	EnvDatabaseURL = "DIF_DATABASE_URL"
	EnvEnvironment = "DIF_ENVIRONMENT"
	EnvLogLevel    = "DIF_LOG_LEVEL"
	EnvAuthMode    = "DIF_AUTH_MODE"
)

// Environment is the deployment/runtime environment label used by DIF.
// It is intentionally generic and not tied to a customer, cloud subscription,
// service entry point, or hosting shape.
type Environment string

const (
	EnvironmentLocal       Environment = "local"
	EnvironmentTest        Environment = "test"
	EnvironmentDevelopment Environment = "development"
	EnvironmentStaging     Environment = "staging"
	EnvironmentProduction  Environment = "production"
)

// AuthMode is a placeholder for the auth posture selected by the caller.
// It does not approve an auth policy and does not retrieve credentials.
type AuthMode string

const (
	AuthModeBearerToken AuthMode = "bearer_token"
	AuthModeOAuthPKCE   AuthMode = "oauth_pkce"
)

// Scope identifies the project/corpus boundary for DIF operations.
type Scope struct {
	ProjectID string
	CorpusID  string
}

// Config is the service-independent P0 runtime baseline.
type Config struct {
	Scope       Scope
	DatabaseURL string
	Environment Environment
	LogLevel    slog.Level
	AuthMode    AuthMode
}

// LoadFromEnv loads and validates required DIF configuration from process
// environment variables. Missing required values are reported explicitly.
func LoadFromEnv() (Config, error) {
	return LoadFromLookup(os.LookupEnv)
}

// LoadFromMap loads configuration from a caller-provided map. It is useful for
// tests and for service bootstraps that already have an env-like source.
func LoadFromMap(values map[string]string) (Config, error) {
	return LoadFromLookup(func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	})
}

// LoadFromLookup loads and validates configuration using a lookup function.
func LoadFromLookup(lookup func(string) (string, bool)) (Config, error) {
	raw := map[string]string{}
	for _, key := range requiredEnvKeys() {
		value, ok := lookup(key)
		if ok {
			raw[key] = strings.TrimSpace(value)
		}
	}

	if err := requireNonEmpty(raw); err != nil {
		return Config{}, err
	}

	environment, err := ParseEnvironment(raw[EnvEnvironment])
	if err != nil {
		return Config{}, err
	}
	logLevel, err := ParseLogLevel(raw[EnvLogLevel])
	if err != nil {
		return Config{}, err
	}
	authMode, err := ParseAuthMode(raw[EnvAuthMode])
	if err != nil {
		return Config{}, err
	}
	if err := validateDatabaseURL(raw[EnvDatabaseURL]); err != nil {
		return Config{}, err
	}

	return Config{
		Scope: Scope{
			ProjectID: raw[EnvProjectID],
			CorpusID:  raw[EnvCorpusID],
		},
		DatabaseURL: raw[EnvDatabaseURL],
		Environment: environment,
		LogLevel:    logLevel,
		AuthMode:    authMode,
	}, nil
}

func requiredEnvKeys() []string {
	return []string{
		EnvProjectID,
		EnvCorpusID,
		EnvDatabaseURL,
		EnvEnvironment,
		EnvLogLevel,
		EnvAuthMode,
	}
}

func requireNonEmpty(values map[string]string) error {
	var missing []string
	for _, key := range requiredEnvKeys() {
		if values[key] == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("missing required DIF config: %s", strings.Join(missing, ", "))
}

// ParseEnvironment validates a DIF environment label.
func ParseEnvironment(value string) (Environment, error) {
	switch Environment(strings.ToLower(strings.TrimSpace(value))) {
	case EnvironmentLocal:
		return EnvironmentLocal, nil
	case EnvironmentTest:
		return EnvironmentTest, nil
	case EnvironmentDevelopment:
		return EnvironmentDevelopment, nil
	case EnvironmentStaging:
		return EnvironmentStaging, nil
	case EnvironmentProduction:
		return EnvironmentProduction, nil
	default:
		return "", fmt.Errorf("invalid %s %q", EnvEnvironment, value)
	}
}

// ParseLogLevel validates the configured log level.
func ParseLogLevel(value string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("invalid %s %q", EnvLogLevel, value)
	}
}

// ParseAuthMode validates the configured auth-mode placeholder.
func ParseAuthMode(value string) (AuthMode, error) {
	switch AuthMode(strings.ToLower(strings.TrimSpace(value))) {
	case AuthModeBearerToken:
		return AuthModeBearerToken, nil
	case AuthModeOAuthPKCE:
		return AuthModeOAuthPKCE, nil
	default:
		return "", fmt.Errorf("invalid %s %q", EnvAuthMode, value)
	}
}

func validateDatabaseURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("invalid %s: %w", EnvDatabaseURL, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("invalid %s: URL must include scheme and host", EnvDatabaseURL)
	}
	switch parsed.Scheme {
	case "postgres", "postgresql":
		return nil
	default:
		return fmt.Errorf("invalid %s: unsupported scheme %q", EnvDatabaseURL, parsed.Scheme)
	}
}

// RedactedDatabaseURL returns a log-safe database URL with credentials removed.
func (c Config) RedactedDatabaseURL() string {
	if c.DatabaseURL == "" {
		return ""
	}
	parsed, err := url.Parse(c.DatabaseURL)
	if err != nil {
		return "[REDACTED_INVALID_DATABASE_URL]"
	}
	if parsed.User != nil {
		if username := parsed.User.Username(); username != "" {
			parsed.User = url.UserPassword(username, "REDACTED")
		} else {
			parsed.User = nil
		}
	}
	return parsed.String()
}

// Validate re-checks an already constructed config.
func (c Config) Validate() error {
	var errs []error
	if strings.TrimSpace(c.Scope.ProjectID) == "" {
		errs = append(errs, errors.New("missing required DIF config: project ID"))
	}
	if strings.TrimSpace(c.Scope.CorpusID) == "" {
		errs = append(errs, errors.New("missing required DIF config: corpus ID"))
	}
	if strings.TrimSpace(c.DatabaseURL) == "" {
		errs = append(errs, errors.New("missing required DIF config: database URL"))
	} else if err := validateDatabaseURL(c.DatabaseURL); err != nil {
		errs = append(errs, err)
	}
	if _, err := ParseEnvironment(string(c.Environment)); err != nil {
		errs = append(errs, err)
	}
	if c.LogLevel < slog.LevelDebug || c.LogLevel > slog.LevelError {
		errs = append(errs, fmt.Errorf("invalid %s %d", EnvLogLevel, c.LogLevel))
	}
	if _, err := ParseAuthMode(string(c.AuthMode)); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
