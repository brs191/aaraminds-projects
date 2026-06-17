// Package pricing externalizes per-model rate and allowance configuration so the
// tool's cost math is not hardcoded (ADR-008).
//
// The bundled defaults are authoritative for GitHub Copilot's Claude models and
// are derived from the GitHub Copilot "models and pricing" reference, using the
// convention 1 credit = $0.01. Users may override any of them by dropping a
// pricing.json file into the config directory (see Load and WriteDefaultIfAbsent);
// their file is merged over the bundled defaults rather than replacing them, so a
// partial file only needs to specify the fields it changes.
//
// budget.SonnetInputRate and friends remain for backward compatibility, but
// pricing.Config is the new source of truth for per-model rates.
package pricing

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aaraminds/copilot-token-budget/internal/platform"
)

// configFileName is the file users edit to override pricing, under platform.ConfigDir().
const configFileName = "pricing.json"

// ModelRate is the per-model pricing for one model family.
type ModelRate struct {
	// InputPerMillion is credits charged per one million input tokens.
	InputPerMillion float64 `json:"inputPerMillion"`
	// OutputPerMillion is credits charged per one million output tokens.
	OutputPerMillion float64 `json:"outputPerMillion"`
	// ContextWindowTokens is the model's usable context window in tokens.
	ContextWindowTokens int64 `json:"contextWindowTokens"`
}

// Config is the full pricing configuration: an allowance, a set of named model
// rates, and a Default rate used when a model name matches nothing.
type Config struct {
	// AllowanceCredits is the monthly credit allowance.
	AllowanceCredits int `json:"allowanceCredits"`
	// Models maps a canonical family key ("sonnet"/"opus"/"haiku") to its rate.
	Models map[string]ModelRate `json:"models"`
	// Default is the fallback rate for unmatched model names.
	Default ModelRate `json:"default"`
}

// Bundled defaults. Source: GitHub Copilot models-and-pricing reference, with
// 1 credit = $0.01. Per-million figures are in credits.
//
// Context windows reflect GitHub Copilot's default (non-extended) configuration
// for the Claude models, which is 200,000 tokens even though the underlying
// Claude API exposes a larger window via the extended/1M beta. Confirmed against
// the Copilot model reference (June 2026).
const defaultAllowanceCredits = 7_000

// defaults returns a fresh copy of the bundled configuration. A new map is
// allocated each call so callers can mutate the result without affecting others.
func defaults() Config {
	return Config{
		AllowanceCredits: defaultAllowanceCredits,
		Models: map[string]ModelRate{
			"sonnet": {InputPerMillion: 300, OutputPerMillion: 1500, ContextWindowTokens: 200000}, // [VERIFY] Claude context window
			"opus":   {InputPerMillion: 500, OutputPerMillion: 2500, ContextWindowTokens: 200000}, // [VERIFY] Claude context window
			"haiku":  {InputPerMillion: 100, OutputPerMillion: 500, ContextWindowTokens: 200000},  // [VERIFY] Claude context window
		},
		Default: ModelRate{InputPerMillion: 300, OutputPerMillion: 1500, ContextWindowTokens: 200000}, // [VERIFY] Claude context window — sonnet rates
	}
}

// Default returns the bundled default configuration with no file overrides.
func Default() Config { return defaults() }

// Load returns the effective pricing configuration. If a pricing.json exists in
// platform.ConfigDir() it is parsed and merged over the bundled defaults (the
// user's allowance and per-model rates win). Load never fails hard on a missing
// or malformed file: it logs to stderr and falls back to the bundled defaults,
// so first-run and corrupted-file cases both yield a usable Config.
//
// Load returns an error only when the config directory itself cannot be resolved.
func Load() (Config, error) {
	cfg := defaults()

	dir, err := platform.ConfigDir()
	if err != nil {
		return cfg, fmt.Errorf("pricing: cannot resolve config dir: %w", err)
	}
	path := filepath.Join(dir, configFileName)

	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "pricing: cannot read %s (%v); using bundled defaults\n", path, err)
		}
		return cfg, nil
	}

	var override Config
	if err := json.Unmarshal(data, &override); err != nil {
		fmt.Fprintf(os.Stderr, "pricing: malformed %s (%v); using bundled defaults\n", path, err)
		return cfg, nil
	}

	return mergeOver(cfg, override), nil
}

// mergeOver returns base with the non-zero fields of override applied on top.
// Zero values in override mean "not set, keep base" so a partial user file only
// overrides what it specifies. Per-model entries are merged field-by-field; a
// model present in override but not base is added.
func mergeOver(base, override Config) Config {
	out := base
	out.Models = make(map[string]ModelRate, len(base.Models))
	for k, v := range base.Models {
		out.Models[k] = v
	}

	if override.AllowanceCredits > 0 {
		out.AllowanceCredits = override.AllowanceCredits
	}
	out.Default = mergeRate(base.Default, override.Default)

	for k, ov := range override.Models {
		out.Models[strings.ToLower(k)] = mergeRate(out.Models[strings.ToLower(k)], ov)
	}
	return out
}

// mergeRate overlays the non-zero fields of override onto base.
func mergeRate(base, override ModelRate) ModelRate {
	out := base
	if override.InputPerMillion != 0 {
		out.InputPerMillion = override.InputPerMillion
	}
	if override.OutputPerMillion != 0 {
		out.OutputPerMillion = override.OutputPerMillion
	}
	if override.ContextWindowTokens != 0 {
		out.ContextWindowTokens = override.ContextWindowTokens
	}
	return out
}

// RateFor returns the rate for a model name using a case-insensitive substring
// match on the known family keys "opus", "sonnet", and "haiku" (checked in that
// order). Any name that matches none returns Default.
func (c Config) RateFor(model string) ModelRate {
	m := strings.ToLower(model)
	for _, key := range []string{"opus", "sonnet", "haiku"} {
		if strings.Contains(m, key) {
			if r, ok := c.Models[key]; ok {
				return r
			}
		}
	}
	return c.Default
}

// WriteDefaultIfAbsent writes the bundled defaults to platform.ConfigDir()/pricing.json
// (mode 0600) if no such file exists, giving users a starting point to edit. It is
// a no-op when the file is already present. Intended for a future `init` flow.
func WriteDefaultIfAbsent() error {
	dir, err := platform.ConfigDir()
	if err != nil {
		return fmt.Errorf("pricing: cannot resolve config dir: %w", err)
	}
	path := filepath.Join(dir, configFileName)

	if _, err := os.Stat(path); err == nil {
		return nil // already present
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("pricing: cannot stat %s: %w", path, err)
	}

	data, err := json.MarshalIndent(defaults(), "", "  ")
	if err != nil {
		return fmt.Errorf("pricing: cannot encode defaults: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("pricing: cannot write %s: %w", path, err)
	}
	return nil
}
