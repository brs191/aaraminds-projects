// Package config loads and validates the Ingestion Service configuration from
// environment variables. All runtime configuration is environment-driven —
// no values are hardcoded.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all runtime configuration for the Ingestion Service.
// Required fields cause [Load] to return an error if absent or empty.
// Optional fields fall back to the documented defaults.
type Config struct {
	// DatabaseURL is the PostgreSQL connection string.
	// Source: DATABASE_URL (required).
	DatabaseURL string

	// CloneDir is the base directory for git clone working trees.
	// Each run gets its own subdirectory: {CloneDir}/{runID}.
	// Source: CLONE_DIR (default: /tmp/rif-clones).
	CloneDir string

	// ExtractorJarPath is the absolute path to the shaded extractor JAR.
	// Source: EXTRACTOR_JAR_PATH (required).
	ExtractorJarPath string

	// ExtractorVersion is written into rif_meta.index_runs.extractor_version.
	// Source: EXTRACTOR_VERSION (default: 1.0.0-SNAPSHOT).
	ExtractorVersion string

	// Port is the HTTP listen port.
	// Source: PORT (default: 8080).
	Port string

	// LogLevel controls slog output verbosity.
	// Valid values: debug, info, warn, error.
	// Source: LOG_LEVEL (default: info).
	LogLevel string

	// AuditLogPath is the path for the GraphStore BlastRadius audit log.
	// Source: AUDIT_LOG_PATH (default: ./audit.log).
	AuditLogPath string

	// MinIndexNodeCount is the minimum number of nodes an extraction run must
	// produce before the version swap is allowed to proceed. A value of 0
	// disables this check (the hard zero-node guard always applies regardless).
	// Source: MIN_INDEX_NODE_COUNT (default: 0 — disabled).
	MinIndexNodeCount int

	// MinIndexEdgeCount is the minimum number of edges an extraction run must
	// produce before the version swap is allowed to proceed. A value of 0
	// disables this check.
	// Source: MIN_INDEX_EDGE_COUNT (default: 0 — disabled).
	MinIndexEdgeCount int

	// Phase2ExtractorsEnabled controls whether Phase 2 extractor JARs are invoked
	// after the Phase 1 extractor and merged into the same NDJSON stream before load.
	// Source: PHASE2_EXTRACTORS_ENABLED (default: false).
	Phase2ExtractorsEnabled bool

	// Phase2SourceRoot is resolved relative to the cloned repo directory and
	// passed to the Phase 2 extractors as --source-root.
	// Source: PHASE2_SOURCE_ROOT (default: src/main/java).
	Phase2SourceRoot string

	// Phase2DiJarPath is the path to the Phase 2 DI extractor shaded JAR.
	// Required when PHASE2_EXTRACTORS_ENABLED=true.
	Phase2DiJarPath string

	// Phase2AopJarPath is the path to the Phase 2 AOP extractor shaded JAR.
	// Required when PHASE2_EXTRACTORS_ENABLED=true.
	Phase2AopJarPath string

	// Phase2CrossServiceJarPath is the path to the Phase 2 cross-service extractor
	// shaded JAR. Required when PHASE2_EXTRACTORS_ENABLED=true.
	Phase2CrossServiceJarPath string

	// IncrementalEnabled enables Phase 5 queue worker + reconciliation loops.
	// Source: PHASE5_INCREMENTAL_ENABLED (default: true).
	IncrementalEnabled bool

	// EmbeddingServiceURL enables Phase 2 embedding enrichment during indexing.
	// When empty, embedding enrichment is skipped.
	// Source: EMBEDDING_SERVICE_URL (default: empty / disabled).
	EmbeddingServiceURL string

	// EmbeddingBatchSize controls how many nodes are sent per /embed request.
	// Source: EMBEDDING_BATCH_SIZE (default: 32).
	EmbeddingBatchSize int

	// GitHubWebhookSecret enables GitHub webhook request authenticity checks.
	// When set, /webhook/github requires X-Hub-Signature-256 HMAC validation.
	// Source: GITHUB_WEBHOOK_SECRET (default: empty / disabled).
	GitHubWebhookSecret string

	// APIToken enables bearer-token protection for state-changing REST endpoints.
	// Source: RIF_API_TOKEN (default: empty / disabled for local development).
	APIToken string

	// AllowedCloneHosts restricts accepted clone_url hosts for repo registration
	// and indexing. Source: ALLOWED_CLONE_HOSTS (default: github.com).
	AllowedCloneHosts []string
}

// Load reads environment variables and returns a fully populated [Config].
// Returns an error if any required variable is missing or empty.
func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:               os.Getenv("DATABASE_URL"),
		CloneDir:                  envWithDefault("CLONE_DIR", "/tmp/rif-clones"),
		ExtractorJarPath:          os.Getenv("EXTRACTOR_JAR_PATH"),
		ExtractorVersion:          envWithDefault("EXTRACTOR_VERSION", "1.0.0-SNAPSHOT"),
		Port:                      envWithDefault("PORT", "8080"),
		LogLevel:                  envWithDefault("LOG_LEVEL", "info"),
		AuditLogPath:              envWithDefault("AUDIT_LOG_PATH", "./audit.log"),
		MinIndexNodeCount:         envInt("MIN_INDEX_NODE_COUNT", 0),
		MinIndexEdgeCount:         envInt("MIN_INDEX_EDGE_COUNT", 0),
		Phase2ExtractorsEnabled:   envBool("PHASE2_EXTRACTORS_ENABLED", false),
		Phase2SourceRoot:          envWithDefault("PHASE2_SOURCE_ROOT", "src/main/java"),
		Phase2DiJarPath:           os.Getenv("PHASE2_DI_EXTRACTOR_JAR_PATH"),
		Phase2AopJarPath:          os.Getenv("PHASE2_AOP_EXTRACTOR_JAR_PATH"),
		Phase2CrossServiceJarPath: os.Getenv("PHASE2_CROSSSERVICE_EXTRACTOR_JAR_PATH"),
		IncrementalEnabled:        envBool("PHASE5_INCREMENTAL_ENABLED", true),
		EmbeddingServiceURL:       strings.TrimSpace(os.Getenv("EMBEDDING_SERVICE_URL")),
		EmbeddingBatchSize:        envInt("EMBEDDING_BATCH_SIZE", 32),
		GitHubWebhookSecret:       strings.TrimSpace(os.Getenv("GITHUB_WEBHOOK_SECRET")),
		APIToken:                  strings.TrimSpace(os.Getenv("RIF_API_TOKEN")),
		AllowedCloneHosts:         envCSV("ALLOWED_CLONE_HOSTS", "github.com"),
	}

	var errs []error
	if cfg.DatabaseURL == "" {
		errs = append(errs, fmt.Errorf("DATABASE_URL is required"))
	}

	if cfg.ExtractorJarPath == "" {
		errs = append(errs, fmt.Errorf("EXTRACTOR_JAR_PATH is required"))
	}
	if cfg.Phase2ExtractorsEnabled {
		if cfg.Phase2DiJarPath == "" {
			errs = append(errs, fmt.Errorf("PHASE2_DI_EXTRACTOR_JAR_PATH is required when PHASE2_EXTRACTORS_ENABLED=true"))
		}
		if cfg.Phase2AopJarPath == "" {
			errs = append(errs, fmt.Errorf("PHASE2_AOP_EXTRACTOR_JAR_PATH is required when PHASE2_EXTRACTORS_ENABLED=true"))
		}
		if cfg.Phase2CrossServiceJarPath == "" {
			errs = append(errs, fmt.Errorf("PHASE2_CROSSSERVICE_EXTRACTOR_JAR_PATH is required when PHASE2_EXTRACTORS_ENABLED=true"))
		}
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return cfg, nil
}

func envCSV(key, fallback string) []string {
	value := os.Getenv(key)
	if strings.TrimSpace(value) == "" {
		value = fallback
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// envWithDefault returns the value of the named environment variable, or
// fallback if the variable is unset or empty.
func envWithDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// envInt returns the integer value of the named environment variable,
// or fallback if the variable is unset, empty, or not a valid integer.
func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envBool(key string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return fallback
	}
	switch v {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}
