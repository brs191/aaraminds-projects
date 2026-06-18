import * as assert from 'assert';
import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';
import { refreshLiveBilling } from './refresher';
import * as cache from './cache';

// Custom test suite (same pattern as cache.test.ts)
class TestSuite {
  private tests: Array<{ name: string; fn: () => void | Promise<void> }> = [];

  test(name: string, fn: () => void | Promise<void>): void {
    this.tests.push({ name, fn });
  }

  async run(): Promise<void> {
    for (const test of this.tests) {
      try {
        await test.fn();
        console.log(`✓ ${test.name}`);
      } catch (err) {
        console.error(`✗ ${test.name}: ${err}`);
        throw err;
      }
    }
  }
}

const suite = new TestSuite();

suite.test('disabled: returns gracefully when live billing disabled', async () => {
  // With default config (disabled), should return nil gracefully.
  const now = new Date();
  const result = await refreshLiveBilling(now);
  // Check that we get a proper RefreshResult, not an error
  assert.ok(result);
  assert.ok('snapshot' in result);
  assert.ok('error' in result);
  assert.ok('cached' in result);
});

suite.test('cache freshness logic', async () => {
  // Create a temporary cache and verify freshness checks work.
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'copilot-livebilling-refresher-'));
  const originalHome = process.env.HOME;
  const originalAppData = process.env.APPDATA;
  process.env.HOME = root;
  process.env.APPDATA = root;

  try {
    const now = new Date();
    const snapshot = {
      orgSlug: 'test-org',
      scope: 'org aggregate' as const,
      sourceLabel: 'test',
      availability: 'available' as const,
      lastRefreshedAt: now,
      asOf: now,
      credits: 35000,
    };
    const ttlHours = 24;
    const cacheEntry = cache.newLiveBillingCacheEntry(snapshot, undefined, ttlHours, now);
    cache.saveLiveBillingCache(cacheEntry);

    // Load and verify cache is fresh
    const loaded = cache.loadLiveBillingCache();
    assert.ok(loaded);
    assert.strictEqual(cache.cacheIsFresh(loaded), true);

    // Verify after expiry, cache is stale
    const futureDate = new Date(now.getTime() + 25 * 60 * 60 * 1000);
    assert.strictEqual(cache.cacheIsFresh(loaded, futureDate), false);
  } finally {
    if (originalHome === undefined) {
      delete process.env.HOME;
    } else {
      process.env.HOME = originalHome;
    }
    if (originalAppData === undefined) {
      delete process.env.APPDATA;
    } else {
      process.env.APPDATA = originalAppData;
    }
  }
});

suite.test('no panic on errors', async () => {
  // Verify that refreshLiveBilling never throws, even in error conditions.
  // With default config (disabled), it should return gracefully.
  try {
    const result = await refreshLiveBilling();
    assert.ok(result !== undefined);
    assert.ok('snapshot' in result);
    assert.ok('error' in result);
    assert.ok('cached' in result);
  } catch (err) {
    assert.fail(`refreshLiveBilling should never throw: ${err}`);
  }
});

suite.test('snapshot shape is correct when snapshot exists', async () => {
  // Test cache save/load to verify snapshot shape.
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'copilot-livebilling-refresher-shape-'));
  const originalHome = process.env.HOME;
  const originalAppData = process.env.APPDATA;
  process.env.HOME = root;
  process.env.APPDATA = root;

  try {
    const now = new Date();
    const snapshot = {
      orgSlug: 'test-org',
      scope: 'org aggregate' as const,
      sourceLabel: 'test',
      availability: 'available' as const,
      lastRefreshedAt: now,
      asOf: now,
      credits: 35000,
    };
    const ttlHours = 24;
    const cacheEntry = cache.newLiveBillingCacheEntry(snapshot, undefined, ttlHours, now);
    cache.saveLiveBillingCache(cacheEntry);

    const loaded = cache.loadLiveBillingCache();
    if (loaded && loaded.snapshot) {
      const s = loaded.snapshot;
      assert.ok('orgSlug' in s);
      assert.ok('scope' in s);
      assert.ok('sourceLabel' in s);
      assert.ok('availability' in s);
      assert.ok('lastRefreshedAt' in s);
      assert.ok('asOf' in s);
      assert.ok('credits' in s);
    }
  } finally {
    if (originalHome === undefined) {
      delete process.env.HOME;
    } else {
      process.env.HOME = originalHome;
    }
    if (originalAppData === undefined) {
      delete process.env.APPDATA;
    } else {
      process.env.APPDATA = originalAppData;
    }
  }
});

void suite.run();
