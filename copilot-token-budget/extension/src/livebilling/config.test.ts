import * as assert from 'assert';
import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';
import { defaults, loadLiveBillingConfig, resolveLiveBillingAuth } from './config';

class TestSuite {
  private tests: Array<{ name: string; fn: () => void | Promise<void> }> = [];

  test(name: string, fn: () => void | Promise<void>): void {
    this.tests.push({ name, fn });
  }

  async run(): Promise<void> {
    let failed = 0;
    for (const test of this.tests) {
      try {
        await test.fn();
        console.log(`✓ ${test.name}`);
      } catch (err) {
        failed += 1;
        console.error(`✗ ${test.name}: ${err}`);
      }
    }
    if (failed > 0) {
      process.exit(1);
    }
  }
}

const suite = new TestSuite();

suite.test('defaults are opt-in and token-env based', () => {
  const cfg = defaults();
  assert.strictEqual(cfg.enabled, false);
  assert.strictEqual(cfg.tokenEnvVar, 'COPILOT_BILLING_TOKEN');
  assert.strictEqual(cfg.cacheMaxAgeHours, 24);
  assert.strictEqual(cfg.requestTimeoutSecs, 10);
});

suite.test('loadLiveBillingConfig merges config.json over defaults', () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'copilot-livebilling-'));
  const originalHome = process.env.HOME;
  const originalAppData = process.env.APPDATA;
  process.env.HOME = root;
  process.env.APPDATA = root;

  try {
    const configDir = process.platform === 'win32'
      ? path.join(root, 'copilot-token-budget')
      : path.join(root, '.config', 'copilot-token-budget');
    fs.mkdirSync(configDir, { recursive: true });
    fs.writeFileSync(
      path.join(configDir, 'config.json'),
      JSON.stringify({
        enabled: true,
        orgSlug: 'att-enterprise',
        tokenEnvVar: 'BILLING_TOKEN',
        cacheMaxAgeHours: 48,
        requestTimeoutSecs: 20,
        dryRun: true,
      }),
    );

    const cfg = loadLiveBillingConfig();
    assert.strictEqual(cfg.enabled, true);
    assert.strictEqual(cfg.orgSlug, 'att-enterprise');
    assert.strictEqual(cfg.tokenEnvVar, 'BILLING_TOKEN');
    assert.strictEqual(cfg.cacheMaxAgeHours, 48);
    assert.strictEqual(cfg.requestTimeoutSecs, 20);
    assert.strictEqual(cfg.dryRun, true);
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

suite.test('resolveLiveBillingAuth reports disabled, dry-run, and ready modes', () => {
  const disabled = resolveLiveBillingAuth(defaults());
  assert.strictEqual(disabled.mode, 'disabled');
  assert.strictEqual(disabled.ready, false);
  assert.strictEqual(disabled.disabled, true);

  const dryRun = resolveLiveBillingAuth({
    enabled: true,
    orgSlug: 'att-enterprise',
    tokenEnvVar: 'COPILOT_BILLING_TOKEN',
    cacheMaxAgeHours: 24,
    requestTimeoutSecs: 10,
    dryRun: true,
  });
  assert.strictEqual(dryRun.mode, 'dry-run');
  assert.strictEqual(dryRun.ready, false);

  const ready = resolveLiveBillingAuth({
    enabled: true,
    orgSlug: 'att-enterprise',
    tokenEnvVar: 'COPILOT_BILLING_TOKEN',
    cacheMaxAgeHours: 24,
    requestTimeoutSecs: 10,
    dryRun: false,
  }, {
    COPILOT_BILLING_TOKEN: 'secret-token',
  });
  assert.strictEqual(ready.mode, 'ready');
  assert.strictEqual(ready.ready, true);
  assert.strictEqual(ready.token, 'secret-token');
});

void suite.run();
