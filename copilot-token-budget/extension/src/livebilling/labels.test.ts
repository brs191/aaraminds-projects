import * as assert from 'assert';
import { latestLiveBillingSnapshot, liveBillingLabel } from './labels';
import { Session } from '../types';

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

suite.test('label states', () => {
  assert.strictEqual(liveBillingLabel(undefined, new Date('2026-06-17T12:00:00Z')), '(estimated)');
  assert.strictEqual(
    liveBillingLabel({
      orgSlug: 'att-enterprise',
      scope: 'org aggregate',
      sourceLabel: 'org aggregate, ~2h ago',
      availability: 'unavailable',
      lastRefreshedAt: new Date('2026-06-17T10:00:00Z'),
      asOf: new Date('2026-06-17T10:00:00Z'),
      credits: 1,
    }, new Date('2026-06-17T12:00:00Z')),
    '(unavailable)',
  );
  assert.strictEqual(
    liveBillingLabel({
      orgSlug: 'att-enterprise',
      scope: 'org aggregate',
      sourceLabel: 'org aggregate, ~2h ago',
      availability: 'available',
      lastRefreshedAt: new Date('2026-06-17T10:00:00Z'),
      asOf: new Date('2026-06-17T10:00:00Z'),
      credits: 1,
    }, new Date('2026-06-17T12:00:00Z')),
    '(org aggregate, ~2h ago)',
  );
});

suite.test('latest snapshot selection', () => {
  const sessions: Session[] = [
    {
      id: 'a',
      workspaceDir: '',
      projectName: '',
      primaryModel: '',
      startTime: new Date('2026-06-17T10:00:00Z'),
      endTime: new Date(0),
      isActive: false,
      totalNanoAIU: 0,
      totalPremiumRequests: 0,
      tokens: { currentTokens: 0, systemTokens: 0, conversationTokens: 0, toolDefinitionsTokens: 0 },
      modelMetrics: [],
      source: 'copilot-cli',
      isFinal: true,
      orgBillingSnapshot: {
        orgSlug: 'att-enterprise',
        scope: 'org aggregate',
        sourceLabel: 'org aggregate, ~3h ago',
        availability: 'available',
        lastRefreshedAt: new Date('2026-06-17T09:00:00Z'),
        asOf: new Date('2026-06-17T09:00:00Z'),
        credits: 1,
      },
    },
    {
      id: 'b',
      workspaceDir: '',
      projectName: '',
      primaryModel: '',
      startTime: new Date('2026-06-17T10:00:00Z'),
      endTime: new Date(0),
      isActive: false,
      totalNanoAIU: 0,
      totalPremiumRequests: 0,
      tokens: { currentTokens: 0, systemTokens: 0, conversationTokens: 0, toolDefinitionsTokens: 0 },
      modelMetrics: [],
      source: 'copilot-cli',
      isFinal: true,
      orgBillingSnapshot: {
        orgSlug: 'att-enterprise',
        scope: 'org aggregate',
        sourceLabel: 'org aggregate, ~1h ago',
        availability: 'available',
        lastRefreshedAt: new Date('2026-06-17T11:00:00Z'),
        asOf: new Date('2026-06-17T11:00:00Z'),
        credits: 2,
      },
    },
  ];
  assert.strictEqual(latestLiveBillingSnapshot(sessions)?.credits, 2);
});

void suite.run();
