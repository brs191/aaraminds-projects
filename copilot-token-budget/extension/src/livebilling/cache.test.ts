import * as assert from 'assert';
import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';
import {
  cacheIsFresh,
  loadLiveBillingCache,
  newLiveBillingCacheEntry,
  saveLiveBillingCache,
} from './cache';
import { LiveBillingSnapshot } from '../types';

const snapshot: LiveBillingSnapshot = {
  orgSlug: 'att-enterprise',
  scope: 'org aggregate',
  sourceLabel: 'org aggregate, ~24h ago',
  availability: 'available',
  lastRefreshedAt: new Date('2026-06-17T10:00:00Z'),
  asOf: new Date('2026-06-16T10:00:00Z'),
  credits: 123.45,
  error: '',
};

class TestSuite {
  private tests: Array<{ name: string; fn: () => void | Promise<void> }> = [];

  test(name: string, fn: () => void | Promise<void>): void {
    this.tests.push({ name, fn });
  }

  async run(): Promise<void> {
    for (const test of this.tests) {
      await test.fn();
      console.log(`✓ ${test.name}`);
    }
  }
}

const suite = new TestSuite();

suite.test('cache freshness', () => {
  const entry = newLiveBillingCacheEntry(snapshot, { hello: 'world' }, 24, new Date('2026-06-17T10:00:00Z'));
  assert.strictEqual(cacheIsFresh(entry, new Date('2026-06-18T09:59:59Z')), true);
  assert.strictEqual(cacheIsFresh(entry, new Date('2026-06-18T10:00:01Z')), false);
});

suite.test('save and load cache entry', () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'copilot-livebilling-cache-'));
  const originalHome = process.env.HOME;
  const originalAppData = process.env.APPDATA;
  process.env.HOME = root;
  process.env.APPDATA = root;

  try {
    const entry = newLiveBillingCacheEntry(snapshot, { hello: 'world' }, 24, new Date('2026-06-17T10:00:00Z'));
    saveLiveBillingCache(entry);
    const loaded = loadLiveBillingCache();
    assert.ok(loaded);
    assert.strictEqual(loaded?.snapshot.orgSlug, 'att-enterprise');
    assert.strictEqual(loaded?.snapshot.sourceLabel, 'org aggregate, ~24h ago');
    assert.strictEqual(loaded?.snapshot.credits, 123.45);
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
