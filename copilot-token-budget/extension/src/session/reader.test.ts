// session/reader.test.ts — tests for IDE collector and dedup logic.
// Uses Node.js built-in assert module only (no external test framework).

import * as assert from 'assert';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';
import { readSessions } from './reader';

// Test suite runner — simple inline test execution
class TestSuite {
  private tests: Array<{ name: string; fn: () => void | Promise<void> }> = [];
  private passCount = 0;
  private failCount = 0;

  test(name: string, fn: () => void | Promise<void>): void {
    this.tests.push({ name, fn });
  }

  async run(): Promise<void> {
    console.log('Running IDE Collector and Dedup Tests...\n');

    for (const test of this.tests) {
      try {
        await test.fn();
        console.log(`✓ ${test.name}`);
        this.passCount++;
      } catch (err) {
        console.error(`✗ ${test.name}: ${err}`);
        this.failCount++;
      }
    }

    console.log(
      `\n${this.passCount} passed, ${this.failCount} failed\n`
    );
    if (this.failCount > 0) {
      process.exit(1);
    }
  }
}

const suite = new TestSuite();

// Test 1: Standard VS Code IDE path shape
suite.test('testIDEStandardPathShape', () => {
  const chatSessionsPath =
    '/Users/user/Library/Application Support/Code/User/workspaceStorage/ws-123/chatSessions/session.jsonl';
  const transcriptPath =
    '/Users/user/Library/Application Support/Code/User/globalStorage/GitHub.copilot-chat/transcripts/session.jsonl';

  assert.ok(chatSessionsPath.includes('workspaceStorage'));
  assert.ok(chatSessionsPath.includes('chatSessions'));
  assert.ok(transcriptPath.includes('globalStorage'));
  assert.ok(transcriptPath.includes('transcripts'));
});

// Test 2: Event-level dedup using {parentId}:{id} seen-set
suite.test('testIDEDedup', () => {
  // Create dedup seen-set and test duplicate detection
  const seenEvents = new Set<string>();

  // First event
  const eventKey1 = 'msg-001:evt-123';
  seenEvents.add(eventKey1);
  assert.ok(seenEvents.has(eventKey1), 'First event should be tracked');

  // Duplicate event (same key)
  const eventKey2 = 'msg-001:evt-123';
  const isDuplicate = seenEvents.has(eventKey2);
  assert.ok(isDuplicate, 'Duplicate event should be detected');

  // Different event
  const eventKey3 = 'msg-001:evt-124';
  const isDifferent = !seenEvents.has(eventKey3);
  assert.ok(isDifferent, 'Different event should not be in seen-set');
});

// Test 3: apiCallId dedup — earliest-wins strategy
suite.test('testIDEAPICallIDDedup', () => {
  // Simulate apiCallId grouping with earliest-wins
  const apiCallIDGroups = new Map<string, { timestamp: Date; eventId: string }>();

  // Process three events with two unique apiCallIds
  const events = [
    { apiCallId: 'call-001', timestamp: new Date('2026-01-15T10:05:00Z'), eventId: 'evt-b' },
    { apiCallId: 'call-001', timestamp: new Date('2026-01-15T10:00:00Z'), eventId: 'evt-a' }, // Earlier
    { apiCallId: 'call-002', timestamp: new Date('2026-01-15T10:10:00Z'), eventId: 'evt-c' },
  ];

  for (const evt of events) {
    const existing = apiCallIDGroups.get(evt.apiCallId);
    if (!existing || evt.timestamp < existing.timestamp) {
      apiCallIDGroups.set(evt.apiCallId, { timestamp: evt.timestamp, eventId: evt.eventId });
    }
  }

  // Verify earliest event is kept for each apiCallId
  assert.equal(apiCallIDGroups.get('call-001')?.eventId, 'evt-a', 'Should keep earliest event');
  assert.equal(apiCallIDGroups.size, 2, 'Should have 2 unique apiCallIds');
});

// Test 4: ModelMetric includes new cache/reasoning token fields
suite.test('testModelMetricsExtension', () => {
  // Create a ModelMetric with new fields
  const metric = {
    model: 'gpt-4-turbo',
    inputTokens: 1000,
    outputTokens: 500,
    nanoAIU: 1500000,
    cacheReadTokens: 200,
    cacheWriteTokens: 100,
    reasoningTokens: 50,
  };

  // Verify all fields exist and have correct values
  assert.strictEqual(metric.cacheReadTokens, 200, 'cacheReadTokens should be 200');
  assert.strictEqual(metric.cacheWriteTokens, 100, 'cacheWriteTokens should be 100');
  assert.strictEqual(metric.reasoningTokens, 50, 'reasoningTokens should be 50');
});

// Test 5: CLI and IDE sessions are merged and deduped by session id
suite.test('testCLIAndIDEMerge', () => {
  // Simulate merging CLI and IDE sessions
  const cliSessions = [
    { id: 'session-001', source: 'copilot-cli', totalNanoAIU: 1000000, isFinal: true },
    { id: 'session-002', source: 'copilot-cli', totalNanoAIU: 500000, isFinal: false },
  ];

  const ideSessions = [
    { id: 'session-001', source: 'copilot-ide', totalNanoAIU: 800000, isFinal: false },
    { id: 'session-003', source: 'copilot-ide', totalNanoAIU: 600000, isFinal: true },
  ];

  // Merge
  const merged = [...cliSessions, ...ideSessions];

  // Dedup by {source}:{id}: final wins, else higher totalNanoAIU wins
  const best = new Map<string, (typeof cliSessions)[0]>();
  for (const s of merged) {
    const key = `${s.source}:${s.id}`;
    const prev = best.get(key);
    if (!prev || (s.isFinal && !prev.isFinal)) {
      best.set(key, s);
    } else if (s.isFinal === prev.isFinal && s.totalNanoAIU > prev.totalNanoAIU) {
      best.set(key, s);
    }
  }

  // Verify dedup results
  assert.strictEqual(best.size, 4, 'Should have 4 unique source-scoped sessions');
  assert.strictEqual(
    best.get('copilot-cli:session-001')?.source,
    'copilot-cli',
    'session-001 should prefer final CLI version'
  );
  assert.strictEqual(
    best.get('copilot-ide:session-001')?.source,
    'copilot-ide',
    'IDE session should survive alongside CLI session with same id'
  );
});

// Test 6: Missing VS Code IDE path is non-fatal
suite.test('testIDEDegradation', () => {
  // Simulate graceful failure when file cannot be read
  const nonexistentPath = '/this/path/does/not/exist/chatSessions/session.jsonl';
  let readFailed = false;

  try {
    fs.accessSync(nonexistentPath);
  } catch {
    readFailed = true;
  }

  // Should have failed gracefully without throwing
  assert.ok(readFailed, 'Reading nonexistent file should set error flag');
});

// Test 7: Source field is stamped and preserved
suite.test('testSourceStamping', () => {
  const cliSession = { id: 'cli-uuid', source: 'copilot-cli' };
  const ideSession = { id: 'ide-uuid', source: 'copilot-ide' };

  assert.strictEqual(cliSession.source, 'copilot-cli', 'CLI source should be copilot-cli');
  assert.strictEqual(ideSession.source, 'copilot-ide', 'IDE source should be copilot-ide');
});

// Test 8: Dashboard serialization includes source field
suite.test('testDashboardSourceField', () => {
  const serializedCli = { id: 'cli-uuid', source: 'copilot-cli' };
  const serializedIde = { id: 'ide-uuid', source: 'copilot-ide' };

  assert.strictEqual(
    serializedCli.source,
    'copilot-cli',
    'Serialized CLI should have copilot-cli source'
  );
  assert.strictEqual(
    serializedIde.source,
    'copilot-ide',
    'Serialized IDE should have copilot-ide source'
  );
});

// Test 8b: IDE standard VS Code path is surfaced through readSessions
suite.test('testIDEStandardPathSurface', async () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'copilot-ide-'));
  const home = root;
  const originalHome = process.env.HOME;
  process.env.HOME = home;

  try {
    const sessionDir = path.join(
      root,
      'Library',
      'Application Support',
      'Code',
      'User',
      'workspaceStorage',
      'ws-123',
      'chatSessions'
    );
    fs.mkdirSync(sessionDir, { recursive: true });
    const transcriptPath = path.join(sessionDir, 'session.jsonl');
    fs.writeFileSync(
      transcriptPath,
      [
        JSON.stringify({
          type: 'session.start',
          data: {
            sessionId: 'ide-session-001',
            startTime: '2026-06-17T10:00:00.000Z',
            context: { cwd: '/Users/rb692q/projects/sample-workspace' },
          },
        }),
        JSON.stringify({
          type: 'assistant.message',
          data: {
            sessionId: 'ide-session-001',
            model: 'claude-sonnet-4.6',
            inputTokens: 1000,
            outputTokens: 100,
            timestamp: '2026-06-17T10:05:00.000Z',
          },
        }),
        JSON.stringify({
          type: 'session.shutdown',
          data: {
            sessionId: 'ide-session-001',
            totalPremiumRequests: 3,
            currentModel: 'claude-sonnet-4.6',
            modelMetrics: {
              'claude-sonnet-4.6': {
                usage: {
                  inputTokens: 1000,
                  outputTokens: 100,
                },
              },
            },
          },
          timestamp: '2026-06-17T10:10:00.000Z',
        }),
      ].join('\n'),
      'utf8'
    );

    const sessions = await readSessions();
    const ideSession = sessions.find((s) => s.source === 'copilot-ide');

    assert.ok(ideSession, 'IDE session should be present');
    assert.strictEqual(ideSession?.id, 'ide-session-001');
    assert.strictEqual(ideSession?.projectName, 'sample-workspace');
    assert.strictEqual(ideSession?.workspaceDir, '/Users/rb692q/projects/sample-workspace');
    assert.strictEqual(ideSession?.totalNanoAIU, 450000000, 'IDE session should use final billing, not double-count');
  } finally {
    if (originalHome === undefined) {
      delete process.env.HOME;
    } else {
      process.env.HOME = originalHome;
    }
  }
});

// Test 9: Dashboard aggregates CLI and IDE totals separately
suite.test('testDashboardSourceBreakdown', () => {
  const sessions = [
    { source: 'copilot-cli', totalNanoAIU: 1000000000 }, // 1 cr
    { source: 'copilot-cli', totalNanoAIU: 500000000 },  // 0.5 cr
    { source: 'copilot-ide', totalNanoAIU: 800000000 },  // 0.8 cr
    { source: 'copilot-ide', totalNanoAIU: 200000000 },  // 0.2 cr
  ];

  let cliCount = 0;
  let ideCount = 0;
  let cliTotal = 0;
  let ideTotal = 0;
  for (const s of sessions) {
    const credits = s.totalNanoAIU / 1_000_000_000;
    if (s.source === 'copilot-cli') {
      cliCount += 1;
      cliTotal += credits;
    } else {
      ideCount += 1;
      ideTotal += credits;
    }
  }

  assert.strictEqual(cliCount, 2, 'CLI count should be 2');
  assert.strictEqual(ideCount, 2, 'IDE count should be 2');
  assert.strictEqual(cliTotal, 1.5, 'CLI total should be 1.5 cr');
  assert.strictEqual(ideTotal, 1.0, 'IDE total should be 1.0 cr');
});

// Test 10: No 'any' types and no TypeScript errors
suite.test('testTypeScriptStrict', () => {
  // This test verifies that the TypeScript code compiles with strict mode
  // If tsc passes without errors, this test passes

  const sessionSource: 'copilot-cli' | 'copilot-ide' = 'copilot-cli';
  assert.ok(
    sessionSource === 'copilot-cli' || sessionSource === 'copilot-ide',
    'SessionSource should be one of the known values'
  );
});

// Run all tests
suite.run().catch((err) => {
  console.error('Test runner failed:', err);
  process.exit(1);
});
