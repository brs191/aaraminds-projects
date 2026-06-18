// extension/src/livebilling/refresher.ts — orchestrate cache-aware fetching of live billing data.
// Mirrors the Go refresher pattern for VS Code extension.

import { GitHubEntitlementFetcher } from './fetcher';
import {
  loadLiveBillingConfig,
  resolveLiveBillingAuth,
} from './config';
import {
  loadLiveBillingCache,
  saveLiveBillingCache,
  cacheIsFresh,
  newLiveBillingCacheEntry,
} from './cache';
import { LiveBillingSnapshot, LiveBillingCacheEntry } from '../types';

export interface RefreshResult {
  snapshot: LiveBillingSnapshot | undefined;
  error: string;
  cached: boolean;
}

export async function refreshLiveBilling(now: Date = new Date()): Promise<RefreshResult> {
  const config = loadLiveBillingConfig();
  const auth = resolveLiveBillingAuth(config);

  // If live billing is disabled, return nil immediately.
  if (auth.disabled) {
    return {
      snapshot: undefined,
      error: 'live billing disabled',
      cached: false,
    };
  }

  // If auth is not ready, return nil with error message.
  if (!auth.ready) {
    console.error(
      `copilot-budget: auth not ready (${auth.mode}); using estimated quota`
    );
    return {
      snapshot: undefined,
      error: auth.message,
      cached: false,
    };
  }

  // Try to load the existing cache.
  const cached = loadLiveBillingCache();
  if (cached && cacheIsFresh(cached, now)) {
    // Cache is fresh; return it without fetching.
    const snapshot = cached.snapshot;
    const h = hoursAgo(snapshot.lastRefreshedAt, now);
    snapshot.sourceLabel = `(authoritative, cached ~${h}h ago)`;
    return {
      snapshot,
      error: '',
      cached: true,
    };
  }

  // Cache is missing or stale. Fetch fresh data.
  console.error('copilot-budget: fetching live quota from GitHub...');

  const fetcher = new GitHubEntitlementFetcher(
    auth.config.gitHubAPIUrl || 'https://api.github.com',
    auth.token,
    auth.config.requestTimeoutSecs,
    auth.config.dryRun
  );

  let quota: number;
  try {
    quota = await fetcher.fetchEntitlements(config.orgSlug);
  } catch (err) {
    // Fetch failed. Log and return nil (graceful degradation).
    const errMsg = err instanceof Error ? err.message : String(err);
    if (errMsg.includes('timeout') || errMsg.includes('timed out')) {
      console.error('copilot-budget: GitHub API timed out; using estimated quota');
    } else if (errMsg.includes('401')) {
      console.error('copilot-budget: GitHub token invalid; using estimated quota');
    } else {
      console.error(
        `copilot-budget: fetch failed (${errMsg}); using estimated quota`
      );
    }
    return {
      snapshot: undefined,
      error: errMsg,
      cached: false,
    };
  }

  // Create a new snapshot with the fetched quota.
  const snapshot: LiveBillingSnapshot = {
    orgSlug: config.orgSlug,
    scope: 'org aggregate',
    sourceLabel: '(authoritative, live)',
    availability: 'available',
    lastRefreshedAt: now,
    asOf: now,
    credits: quota,
    error: undefined,
  };

  // Save to cache with TTL.
  const ttlHours = config.cacheMaxAgeHours;
  const cacheEntry = newLiveBillingCacheEntry(snapshot, undefined, ttlHours, now);
  try {
    saveLiveBillingCache(cacheEntry);
  } catch (err) {
    // Cache write failed, but we still have the fetched data, so log and continue.
    console.error(
      `copilot-budget: cannot save cache (${err instanceof Error ? err.message : err}); continuing without cache`
    );
  }

  console.error(`copilot-budget: fetched live quota ${quota} from GitHub`);

  return {
    snapshot,
    error: '',
    cached: false,
  };
}

// hoursAgo calculates the number of hours between time t and now.
// Returns at least 1 if t is less than 1 hour ago.
function hoursAgo(t: Date, now: Date): number {
  if (!t || t.getTime() === 0) {
    return 0;
  }
  const ms = now.getTime() - t.getTime();
  if (ms <= 0) {
    return 1;
  }
  return Math.max(1, Math.floor(ms / (60 * 60 * 1000)));
}
