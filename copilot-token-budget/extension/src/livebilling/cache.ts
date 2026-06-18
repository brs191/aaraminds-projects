// livebilling/cache.ts — local cache for live billing snapshots.
// The cache mirrors the Go side and stays under the VS Code config dir.

import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';
import { LiveBillingCacheEntry, LiveBillingSnapshot } from '../types';

const CACHE_FILE = 'live-billing-cache.json';

export function loadLiveBillingCache(): LiveBillingCacheEntry | undefined {
  const filePath = cachePath();
  let data: string;
  try {
    data = fs.readFileSync(filePath, 'utf8');
  } catch (err) {
    if ((err as NodeJS.ErrnoException).code !== 'ENOENT') {
      console.error(`copilot-budget: cannot read live billing cache ${filePath} (${err}); ignoring cache`);
    }
    return undefined;
  }

  try {
    return parseCacheEntry(JSON.parse(data));
  } catch (err) {
    console.error(`copilot-budget: malformed live billing cache ${filePath} (${err}); ignoring cache`);
    return undefined;
  }
}

export function saveLiveBillingCache(entry: LiveBillingCacheEntry): void {
  const filePath = cachePath();
  fs.mkdirSync(path.dirname(filePath), { recursive: true, mode: 0o700 });
  fs.writeFileSync(filePath, JSON.stringify(serializeCacheEntry(entry), null, 2), { mode: 0o600 });
}

export function cacheIsFresh(entry: LiveBillingCacheEntry, now: Date = new Date()): boolean {
  return entry.expiresAt.getTime() > now.getTime();
}

export function newLiveBillingCacheEntry(
  snapshot: LiveBillingSnapshot,
  payload: unknown,
  ttlHours: number,
  now: Date = new Date(),
): LiveBillingCacheEntry {
  const ttlMs = Math.max(1, ttlHours) * 60 * 60 * 1000;
  return {
    snapshot,
    payload,
    cachedAt: now,
    expiresAt: new Date(now.getTime() + ttlMs),
  };
}

function cachePath(): string {
  return path.join(configDir(), CACHE_FILE);
}

function configDir(): string {
  const home = os.homedir();
  if (process.platform === 'win32') {
    const appData = process.env.APPDATA ?? path.join(home, 'AppData', 'Roaming');
    return path.join(appData, 'copilot-token-budget');
  }
  return path.join(home, '.config', 'copilot-token-budget');
}

function parseCacheEntry(value: unknown): LiveBillingCacheEntry {
  if (!isRecord(value) || !isRecord(value.snapshot)) {
    throw new Error('cache entry missing snapshot');
  }
  return {
    snapshot: parseSnapshot(value.snapshot),
    payload: value.payload,
    cachedAt: new Date(stringifyDate(value.cachedAt)),
    expiresAt: new Date(stringifyDate(value.expiresAt)),
  };
}

function parseSnapshot(value: Record<string, unknown>): LiveBillingSnapshot {
  return {
    orgSlug: stringField(value.orgSlug),
    scope: stringField(value.scope) as LiveBillingSnapshot['scope'],
    sourceLabel: stringField(value.sourceLabel),
    availability: stringField(value.availability) as LiveBillingSnapshot['availability'],
    lastRefreshedAt: new Date(stringifyDate(value.lastRefreshedAt)),
    asOf: new Date(stringifyDate(value.asOf)),
    credits: numberField(value.credits),
    error: typeof value.error === 'string' ? value.error : undefined,
  };
}

function serializeCacheEntry(entry: LiveBillingCacheEntry): unknown {
  return {
    snapshot: serializeSnapshot(entry.snapshot),
    payload: entry.payload,
    cachedAt: entry.cachedAt.toISOString(),
    expiresAt: entry.expiresAt.toISOString(),
  };
}

function serializeSnapshot(snapshot: LiveBillingSnapshot): unknown {
  return {
    ...snapshot,
    lastRefreshedAt: snapshot.lastRefreshedAt.toISOString(),
    asOf: snapshot.asOf.toISOString(),
  };
}

function stringifyDate(value: unknown): string {
  if (typeof value === 'string') {
    return value;
  }
  throw new Error('expected date string');
}

function stringField(value: unknown): string {
  if (typeof value === 'string') {
    return value;
  }
  throw new Error('expected string field');
}

function numberField(value: unknown): number {
  if (typeof value === 'number') {
    return value;
  }
  return 0;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}
