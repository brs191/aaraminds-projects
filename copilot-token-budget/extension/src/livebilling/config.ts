// livebilling/config.ts — optional Phase 8 live billing config + auth resolver.
// Reads the same platform config dir shape as the Go side, but stays disabled by
// default and never writes secrets to disk. Supports out-of-the-box auto-detection
// of org slug and token from GitHub CLI.

import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';
import { execSync } from 'child_process';

export interface LiveBillingConfig {
  enabled: boolean;
  orgSlug: string;
  tokenEnvVar: string;
  cacheMaxAgeHours: number;
  requestTimeoutSecs: number;
  gitHubAPIUrl?: string;
  dryRun: boolean;
}

export interface LiveBillingAuthResolution {
  config: LiveBillingConfig;
  mode: 'disabled' | 'config-error' | 'dry-run' | 'missing-token' | 'ready' | 'auto-detected';
  token: string;
  ready: boolean;
  hasToken: boolean;
  dryRun: boolean;
  disabled: boolean;
  autoDetected: boolean;
  message: string;
}

const DEFAULT_TOKEN_ENV_VAR = 'COPILOT_BILLING_TOKEN';

export function defaults(): LiveBillingConfig {
  return {
    enabled: false,
    orgSlug: '',
    tokenEnvVar: DEFAULT_TOKEN_ENV_VAR,
    cacheMaxAgeHours: 24,
    requestTimeoutSecs: 10,
    dryRun: false,
  };
}

export function loadLiveBillingConfig(): LiveBillingConfig {
  const cfg = defaults();
  const pathToConfig = configPath();

  let data: string;
  try {
    data = fs.readFileSync(pathToConfig, 'utf8');
  } catch (err) {
    if ((err as NodeJS.ErrnoException).code !== 'ENOENT') {
      console.error(`copilot-budget: cannot read live billing config ${pathToConfig} (${err}); using defaults`);
    }
    return cfg;
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(data);
  } catch (err) {
    console.error(`copilot-budget: malformed live billing config ${pathToConfig} (${err}); using defaults`);
    return cfg;
  }

  return mergeOver(cfg, parsed);
}

export function resolveLiveBillingAuth(
  cfg: LiveBillingConfig,
  env: NodeJS.ProcessEnv = process.env,
  autoDetect: boolean = true,
): LiveBillingAuthResolution {
  const tokenEnvVar = cfg.tokenEnvVar.trim() === '' ? DEFAULT_TOKEN_ENV_VAR : cfg.tokenEnvVar;
  const out: LiveBillingAuthResolution = {
    config: { ...cfg, tokenEnvVar },
    mode: 'disabled',
    token: '',
    ready: false,
    hasToken: false,
    dryRun: cfg.dryRun,
    disabled: false,
    autoDetected: false,
    message: '',
  };

  if (!cfg.enabled) {
    out.disabled = true;
    out.message = 'live billing disabled';
    return out;
  }

  // If orgSlug is missing, try auto-detect from GitHub CLI.
  let orgSlug = cfg.orgSlug.trim();
  if (orgSlug === '' && autoDetect) {
    try {
      const detected = autoDetectOrgSlug();
      if (detected) {
        orgSlug = detected;
        out.autoDetected = true;
        out.mode = 'auto-detected';
      }
    } catch (err) {
      console.error(`copilot-budget: auto-detect org failed (${err}); skipping`);
    }
  }

  if (orgSlug === '') {
    out.mode = 'config-error';
    out.message = 'live billing enabled but orgSlug is empty (auto-detect failed)';
    return out;
  }

  // Update config with detected orgSlug.
  out.config.orgSlug = orgSlug;

  if (cfg.dryRun) {
    out.mode = 'dry-run';
    out.message = 'live billing dry-run; no HTTP requests will be made';
    return out;
  }

  const token = env[tokenEnvVar]?.trim() ?? '';
  if (token === '') {
    out.mode = 'missing-token';
    out.message = `live billing enabled but env var ${tokenEnvVar} is not set`;
    return out;
  }

  out.mode = out.autoDetected ? 'auto-detected' : 'ready';
  out.token = token;
  out.ready = true;
  out.hasToken = true;
  out.message = out.autoDetected
    ? `live billing auto-detected (org: ${orgSlug})`
    : 'live billing auth ready';
  return out;
}

// autoDetectOrgSlug tries to fetch the organization slug from `gh` CLI.
// Returns the org slug on success, empty string on failure.
export function autoDetectOrgSlug(): string {
  try {
    const result = execSync('gh api user --jq .login', { encoding: 'utf8', timeout: 5000 });
    return result.trim();
  } catch (err) {
    return '';
  }
}

function configPath(): string {
  return path.join(configDir(), 'config.json');
}

function configDir(): string {
  const home = os.homedir();
  if (process.platform === 'win32') {
    const appData = process.env.APPDATA ?? path.join(home, 'AppData', 'Roaming');
    return path.join(appData, 'copilot-token-budget');
  }
  return path.join(home, '.config', 'copilot-token-budget');
}

function mergeOver(base: LiveBillingConfig, override: unknown): LiveBillingConfig {
  if (!isRecord(override)) {
    return base;
  }

  const out: LiveBillingConfig = { ...base };
  if (override.enabled === true) {
    out.enabled = true;
  }
  if (typeof override.orgSlug === 'string' && override.orgSlug.trim() !== '') {
    out.orgSlug = override.orgSlug;
  }
  if (typeof override.tokenEnvVar === 'string' && override.tokenEnvVar.trim() !== '') {
    out.tokenEnvVar = override.tokenEnvVar;
  }
  if (typeof override.cacheMaxAgeHours === 'number' && override.cacheMaxAgeHours > 0) {
    out.cacheMaxAgeHours = clamp(Math.trunc(override.cacheMaxAgeHours), 1, 72);
  }
  if (typeof override.requestTimeoutSecs === 'number' && override.requestTimeoutSecs > 0) {
    out.requestTimeoutSecs = Math.trunc(override.requestTimeoutSecs);
  }
  if (override.dryRun === true) {
    out.dryRun = true;
  }
  return out;
}

function clamp(value: number, min: number, max: number): number {
  if (value < min) {
    return min;
  }
  if (value > max) {
    return max;
  }
  return value;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}
