// pricing/config.ts — overridable per-model pricing for the VS Code extension.
// TypeScript port of phase-1/session-manager/internal/pricing/pricing.go (ADR-008).
//
// The bundled defaults are authoritative for GitHub Copilot's Claude models and use
// the convention 1 credit = $0.01. Per-million figures are in credits. Users may
// override any of them by pointing the `copilotBudget.pricingPath` setting at a
// pricing.json; that file is merged OVER the bundled defaults (a partial file only
// needs to specify the fields it changes), and a missing/malformed file never throws.
//
// Uses only Node.js built-ins (fs) — zero npm runtime deps (ADR-003).

import * as fs from 'fs';
import * as vscode from 'vscode';

// ModelRate is the per-model pricing for one model family.
export interface ModelRate {
  inputPerMillion: number;     // credits charged per one million input tokens
  outputPerMillion: number;    // credits charged per one million output tokens
  contextWindowTokens: number; // the model's usable context window in tokens
}

// PricingConfig is the full pricing configuration: an allowance, named model rates,
// and a default rate used when a model name matches nothing.
export interface PricingConfig {
  allowanceCredits: number;            // monthly credit allowance
  models: Record<string, ModelRate>;   // canonical family key -> rate
  default: ModelRate;                  // fallback rate for unmatched model names
}

// Bundled defaults. Source: GitHub Copilot models-and-pricing reference, with
// 1 credit = $0.01. Context windows reflect Copilot's default (non-extended)
// 200,000-token configuration for the Claude models. Identical to the Go defaults().
const DEFAULT_ALLOWANCE_CREDITS = 7_000;

// defaults returns a fresh copy of the bundled configuration each call so callers
// can mutate the result without affecting others.
export function defaults(): PricingConfig {
  return {
    allowanceCredits: DEFAULT_ALLOWANCE_CREDITS,
    models: {
      sonnet: { inputPerMillion: 300, outputPerMillion: 1500, contextWindowTokens: 200000 }, // [VERIFY] Claude context window
      opus:   { inputPerMillion: 500, outputPerMillion: 2500, contextWindowTokens: 200000 }, // [VERIFY] Claude context window
      haiku:  { inputPerMillion: 100, outputPerMillion: 500, contextWindowTokens: 200000 },  // [VERIFY] Claude context window
    },
    default: { inputPerMillion: 300, outputPerMillion: 1500, contextWindowTokens: 200000 },  // [VERIFY] Claude context window — sonnet rates
  };
}

// Shape of an override file. Every field is optional so a partial file merges cleanly.
interface PricingOverride {
  allowanceCredits?: number;
  models?: Record<string, Partial<ModelRate>>;
  default?: Partial<ModelRate>;
}

// loadPricing returns the effective pricing configuration. If the
// `copilotBudget.pricingPath` setting points at a readable pricing.json it is parsed
// and merged over the bundled defaults (the user's allowance and per-model rates win).
// loadPricing never throws on a missing or malformed file: it logs to the console and
// falls back to the bundled defaults, so first-run and corrupted-file cases both yield
// a usable config. Mirrors Go pricing.Load.
export function loadPricing(): PricingConfig {
  const cfg = defaults();

  const pricingPath = vscode.workspace
    .getConfiguration('copilotBudget')
    .get<string>('pricingPath') ?? '';
  if (pricingPath === '') {
    return cfg;
  }

  let data: string;
  try {
    data = fs.readFileSync(pricingPath, 'utf8');
  } catch (err) {
    console.error(`copilot-budget: cannot read pricing file ${pricingPath} (${err}); using bundled defaults`);
    return cfg;
  }

  let override: PricingOverride;
  try {
    override = JSON.parse(data) as PricingOverride;
  } catch (err) {
    console.error(`copilot-budget: malformed pricing file ${pricingPath} (${err}); using bundled defaults`);
    return cfg;
  }

  return mergeOver(cfg, override);
}

// mergeOver returns base with the non-zero fields of override applied on top. Zero
// values in override mean "not set, keep base" so a partial file only overrides what
// it specifies. Per-model entries merge field-by-field; a model present in override
// but not base is added (keyed lowercase). Mirrors Go mergeOver.
function mergeOver(base: PricingConfig, override: PricingOverride): PricingConfig {
  const out: PricingConfig = {
    allowanceCredits: base.allowanceCredits,
    models: { ...base.models },
    default: { ...base.default },
  };

  if (override.allowanceCredits != null && override.allowanceCredits > 0) {
    out.allowanceCredits = override.allowanceCredits;
  }
  out.default = mergeRate(base.default, override.default);

  for (const [k, ov] of Object.entries(override.models ?? {})) {
    const key = k.toLowerCase();
    out.models[key] = mergeRate(out.models[key] ?? zeroRate(), ov);
  }
  return out;
}

// mergeRate overlays the non-zero fields of override onto base. Mirrors Go mergeRate.
function mergeRate(base: ModelRate, override: Partial<ModelRate> | undefined): ModelRate {
  const out: ModelRate = { ...base };
  if (override == null) {
    return out;
  }
  if (override.inputPerMillion != null && override.inputPerMillion !== 0) {
    out.inputPerMillion = override.inputPerMillion;
  }
  if (override.outputPerMillion != null && override.outputPerMillion !== 0) {
    out.outputPerMillion = override.outputPerMillion;
  }
  if (override.contextWindowTokens != null && override.contextWindowTokens !== 0) {
    out.contextWindowTokens = override.contextWindowTokens;
  }
  return out;
}

// zeroRate is the all-zero rate used as the base when an override introduces a model
// family that the defaults do not contain.
function zeroRate(): ModelRate {
  return { inputPerMillion: 0, outputPerMillion: 0, contextWindowTokens: 0 };
}

// rateFor returns the rate for a model name using a case-insensitive substring match
// on the known family keys "opus", "sonnet", and "haiku" (checked in that order). Any
// name that matches none returns the default rate. Mirrors Go Config.RateFor.
export function rateFor(cfg: PricingConfig, model: string): ModelRate {
  const m = model.toLowerCase();
  for (const key of ['opus', 'sonnet', 'haiku']) {
    if (m.includes(key)) {
      const r = cfg.models[key];
      if (r !== undefined) {
        return r;
      }
    }
  }
  return cfg.default;
}
