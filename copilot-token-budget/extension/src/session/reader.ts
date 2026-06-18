// session/reader.ts — async JSONL session reader for the VS Code extension.
// TypeScript port of phase-1/session-manager/internal/session/reader.go.
// Uses only Node.js built-ins (fs, path, readline, os) — zero npm runtime deps (ADR-003).

import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';
import * as readline from 'readline';
import { Session, SessionSource, TokenBreakdown, billingTime } from '../types';

// sessionStateDir returns the platform-correct path to ~/.copilot/session-state.
// Uses os.homedir() — never hardcodes /Users/ or ~ (Windows-safe).
function sessionStateDir(): string {
  return path.join(os.homedir(), '.copilot', 'session-state');
}

// Collector is a source of sessions. Each implementation reads one upstream data
// source (Copilot CLI state, IDE usage, etc.) and returns sessions already stamped
// with their source. readSessions runs every registered collector and merges the
// results. A collector that fails returns []; the others still contribute. Mirrors
// the Go session.Collector interface.
interface Collector {
  // name returns the stable source identifier this collector stamps on sessions.
  name(): SessionSource;
  // collect reads and returns this source's sessions. A read failure is logged and
  // an empty slice returned — it never throws so a bad source cannot abort the merge.
  collect(): Promise<Session[]>;
}

// cliCollector reads GitHub Copilot CLI session-state via collectCliSessions. It is
// the only live source today.
const cliCollector: Collector = {
  name(): SessionSource {
    return 'copilot-cli';
  },
  collect(): Promise<Session[]> {
    return collectCliSessions();
  },
};

// ideCollector reads VS Code Copilot IDE sessions from standard VS Code user-data
// transcript roots (chatSessions/transcripts/emptyWindowChatSessions). This source
// is required when the user has IDE activity but little/no Copilot CLI usage.
const ideCollector: Collector = {
  name(): SessionSource {
    return 'copilot-ide';
  },
  async collect(): Promise<Session[]> {
    return collectIdeSessions();
  },
};

// collectors is the ordered set of sources readSessions merges. CLI first so that,
// all else equal, a CLI record is encountered before an IDE record with the same id.
const collectors: Collector[] = [cliCollector, ideCollector];

// readSessions runs every registered collector, concatenates their sessions,
// deduplicates by session id across all sources, and returns the survivors sorted by
// startTime descending (newest first). A collector that fails contributes nothing but
// does not abort the merge. For CLI-only data every id is unique, so the dedup is a
// no-op and existing behavior is preserved. Mirrors Go ReadAll.
export async function readSessions(): Promise<Session[]> {
  let merged: Session[] = [];
  for (const c of collectors) {
    let got: Session[];
    try {
      got = await c.collect();
    } catch (err) {
      console.error(`copilot-budget: collector ${c.name()} failed: ${err}`);
      continue;
    }
    // Stamp source defensively; collectors should set it, but never trust.
    for (const s of got) {
      if (s.source === undefined) {
        s.source = c.name();
      }
    }
    merged = merged.concat(got);
  }

  const deduped = dedupBySourceAndId(merged);

  // Enrich CLI sessions with IDE activity metadata if available.
  await enrichWithIdeMetadata(deduped);

  // Sort newest first — mirrors Go sort.Slice on StartTime.After.
  deduped.sort((a, b) => b.startTime.getTime() - a.startTime.getTime());
  return deduped;
}

// dedupBySourceAndId collapses sessions sharing the same {source}:{id} tuple to a
// single record per the readSessions dedup rule (final wins; else higher totalNanoAIU).
// Sessions with an empty id are passed through untouched (they cannot be keyed).
function dedupBySourceAndId(sessions: Session[]): Session[] {
  const best = new Map<string, Session>();
  const order: string[] = [];
  const unkeyed: Session[] = [];

  for (const s of sessions) {
    if (s.id === '') {
      unkeyed.push(s);
      continue;
    }
    const key = `${s.source}:${s.id}`;
    const prev = best.get(key);
    if (prev === undefined) {
      best.set(key, s);
      order.push(key);
      continue;
    }
    if (preferSession(prev, s)) {
      best.set(key, s);
    }
  }

  const out: Session[] = [];
  for (const key of order) {
    const s = best.get(key);
    if (s !== undefined) {
      out.push(s);
    }
  }
  return out.concat(unkeyed);
}

// preferSession reports whether candidate should replace current under the dedup
// rule: a final reading beats a non-final one; otherwise the higher totalNanoAIU
// wins. Ties keep the incumbent (deterministic). Mirrors Go preferSession.
function preferSession(current: Session, candidate: Session): boolean {
  if (candidate.isFinal !== current.isFinal) {
    return candidate.isFinal; // promote candidate only if it is the final one
  }
  return candidate.totalNanoAIU > current.totalNanoAIU;
}

// collectCliSessions scans all session directories under session-state/ and returns
// the CLI sessions (unsorted; readSessions sorts after merging). A single unreadable
// session directory is logged and skipped — never throws.
async function collectCliSessions(): Promise<Session[]> {
  const stateDir = sessionStateDir();

  let entries: fs.Dirent[];
  try {
    entries = await fs.promises.readdir(stateDir, { withFileTypes: true });
  } catch (err) {
    console.error(`copilot-budget: cannot read session-state dir: ${err}`);
    return [];
  }

  const results: Session[] = [];

  for (const entry of entries) {
    if (!entry.isDirectory()) {
      continue;
    }
    const sessionDir = path.join(stateDir, entry.name);
    try {
      const session = await readOneSession(entry.name, sessionDir);
      results.push(session);
    } catch (err) {
      console.error(`copilot-budget: skipping session ${entry.name}: ${err}`);
    }
  }

  return results;
}

interface IdeModelBucket {
  inputTokens: number;
  outputTokens: number;
  cacheReadTokens: number;
  cacheWriteTokens: number;
  reasoningTokens: number;
}

interface IdeSessionAccumulator {
  id: string;
  workspaceDir: string;
  projectName: string;
  primaryModel: string;
  startTime: Date;
  endTime: Date;
  isFinal: boolean;
  hasFinalBilling: boolean;
  totalPremiumRequests: number;
  tokens: TokenBreakdown;
  totalTokens: {
    input: number;
    output: number;
    cacheRead: number;
    cacheWrite: number;
    reasoning: number;
  };
  modelBuckets: Map<string, IdeModelBucket>;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function ideUserDataRoots(): string[] {
  const home = os.homedir();
  if (process.platform === 'darwin') {
    return [
      path.join(home, 'Library', 'Application Support', 'Code', 'User'),
      path.join(home, 'Library', 'Application Support', 'Code - Insiders', 'User'),
    ];
  }
  if (process.platform === 'win32') {
    const appData = process.env.APPDATA ?? path.join(home, 'AppData', 'Roaming');
    return [
      path.join(appData, 'Code', 'User'),
      path.join(appData, 'Code - Insiders', 'User'),
    ];
  }
  return [
    path.join(home, '.config', 'Code', 'User'),
    path.join(home, '.config', 'Code - Insiders', 'User'),
  ];
}

async function collectIdeSessions(): Promise<Session[]> {
  const roots = ideUserDataRoots();
  const files = new Set<string>();

  for (const root of roots) {
    await collectIdeSessionFiles(root, files);
  }

  if (files.size === 0) {
    return [];
  }

  const accById = new Map<string, IdeSessionAccumulator>();
  for (const filePath of files) {
    const acc = await parseIdeSessionFile(filePath);
    if (acc == null) {
      continue;
    }
    const key = acc.id !== '' ? acc.id : filePath;
    const prev = accById.get(key);
    if (prev === undefined) {
      accById.set(key, acc);
    } else {
      mergeIdeAccumulators(prev, acc);
    }
  }

  if (accById.size === 0) {
    return [];
  }

  const sessions: Session[] = [];
  for (const acc of accById.values()) {
    sessions.push(buildIdeSession(acc));
  }
  return sessions;
}

async function collectIdeSessionFiles(root: string, out: Set<string>): Promise<void> {
  if (!await exists(root)) {
    return;
  }

  const stack = [root];
  while (stack.length > 0) {
    const dir = stack.pop();
    if (dir === undefined) {
      continue;
    }
    let entries: fs.Dirent[];
    try {
      entries = await fs.promises.readdir(dir, { withFileTypes: true });
    } catch {
      continue;
    }

    for (const entry of entries) {
      const full = path.join(dir, entry.name);
      if (entry.isDirectory()) {
        if (IDE_SESSION_DIR_NAMES.has(entry.name)) {
          await collectAllFiles(full, out);
        } else {
          stack.push(full);
        }
      }
    }
  }
}

async function collectAllFiles(dir: string, out: Set<string>): Promise<void> {
  let entries: fs.Dirent[];
  try {
    entries = await fs.promises.readdir(dir, { withFileTypes: true });
  } catch {
    return;
  }

  for (const entry of entries) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      await collectAllFiles(full, out);
      continue;
    }
    if (entry.isFile()) {
      out.add(full);
    }
  }
}

async function exists(p: string): Promise<boolean> {
  try {
    await fs.promises.access(p, fs.constants.F_OK);
    return true;
  } catch {
    return false;
  }
}

const IDE_SESSION_DIR_NAMES = new Set(['chatSessions', 'transcripts', 'emptyWindowChatSessions']);

async function parseIdeSessionFile(filePath: string): Promise<IdeSessionAccumulator | null> {
  let raw: string;
  try {
    raw = await fs.promises.readFile(filePath, 'utf8');
  } catch {
    return null;
  }

  const acc = makeIdeAccumulator(filePath);
  const records = parseIdePayload(raw);
  for (const record of records) {
    mergeIdeRecord(acc, record);
  }

  if (!hasUsefulIdeData(acc)) {
    return null;
  }

  if (acc.id === '') {
    acc.id = path.basename(filePath, path.extname(filePath));
  }
  return acc;
}

function hasUsefulIdeData(acc: IdeSessionAccumulator): boolean {
  return (
    acc.id !== '' ||
    acc.workspaceDir !== '' ||
    acc.projectName !== '' ||
    acc.primaryModel !== '' ||
    acc.startTime.getTime() !== 0 ||
    acc.endTime.getTime() !== 0 ||
    acc.totalPremiumRequests > 0 ||
    acc.totalTokens.input > 0 ||
    acc.totalTokens.output > 0 ||
    acc.totalTokens.cacheRead > 0 ||
    acc.totalTokens.cacheWrite > 0 ||
    acc.totalTokens.reasoning > 0 ||
    acc.modelBuckets.size > 0
  );
}

function parseIdePayload(raw: string): unknown[] {
  const trimmed = raw.trim();
  if (trimmed === '') {
    return [];
  }

  if (trimmed.startsWith('{') || trimmed.startsWith('[')) {
    try {
      return flattenIdePayload(JSON.parse(trimmed) as unknown);
    } catch {
      // fall through to JSONL parsing
    }
  }

  const records: unknown[] = [];
  for (const line of raw.split(/\r?\n/)) {
    const item = line.trim();
    if (item === '') {
      continue;
    }
    try {
      records.push(...flattenIdePayload(JSON.parse(item) as unknown));
    } catch {
      continue;
    }
  }
  return records;
}

function flattenIdePayload(value: unknown): unknown[] {
  if (Array.isArray(value)) {
    return value.flatMap(item => flattenIdePayload(item));
  }
  if (!isRecord(value)) {
    return [];
  }

  const nestedKeys = ['messages', 'items', 'turns', 'entries', 'records'];
  const nested: unknown[] = [];
  for (const key of nestedKeys) {
    const candidate = value[key];
    if (Array.isArray(candidate)) {
      nested.push(...candidate.flatMap(item => flattenIdePayload(item)));
    }
  }
  return nested.length > 0 ? nested : [value];
}

function makeIdeAccumulator(filePath: string): IdeSessionAccumulator {
  return {
    id: '',
    workspaceDir: '',
    projectName: '',
    primaryModel: '',
    startTime: new Date(0),
    endTime: new Date(0),
    isFinal: true,
    hasFinalBilling: false,
    totalPremiumRequests: 0,
    tokens: { currentTokens: 0, systemTokens: 0, conversationTokens: 0, toolDefinitionsTokens: 0 },
    totalTokens: { input: 0, output: 0, cacheRead: 0, cacheWrite: 0, reasoning: 0 },
    modelBuckets: new Map<string, IdeModelBucket>(),
  };
}

function mergeIdeAccumulators(target: IdeSessionAccumulator, source: IdeSessionAccumulator): void {
  if (source.id !== '' && target.id === '') {
    target.id = source.id;
  }
  if (source.workspaceDir !== '' && target.workspaceDir === '') {
    target.workspaceDir = source.workspaceDir;
  }
  if (source.projectName !== '' && target.projectName === '') {
    target.projectName = source.projectName;
  }
  if (source.primaryModel !== '' && target.primaryModel === '') {
    target.primaryModel = source.primaryModel;
  }
  if (target.startTime.getTime() === 0 || (source.startTime.getTime() !== 0 && source.startTime < target.startTime)) {
    target.startTime = source.startTime;
  }
  if (source.endTime.getTime() > target.endTime.getTime()) {
    target.endTime = source.endTime;
  }
  target.isFinal = target.isFinal || source.isFinal;
  target.totalPremiumRequests += source.totalPremiumRequests;
  target.totalTokens.input += source.totalTokens.input;
  target.totalTokens.output += source.totalTokens.output;
  target.totalTokens.cacheRead += source.totalTokens.cacheRead;
  target.totalTokens.cacheWrite += source.totalTokens.cacheWrite;
  target.totalTokens.reasoning += source.totalTokens.reasoning;

  for (const [model, bucket] of source.modelBuckets.entries()) {
    const current = target.modelBuckets.get(model);
    if (current === undefined) {
      target.modelBuckets.set(model, { ...bucket });
      continue;
    }
    current.inputTokens += bucket.inputTokens;
    current.outputTokens += bucket.outputTokens;
    current.cacheReadTokens += bucket.cacheReadTokens;
    current.cacheWriteTokens += bucket.cacheWriteTokens;
    current.reasoningTokens += bucket.reasoningTokens;
  }
}

function mergeIdeRecord(acc: IdeSessionAccumulator, value: unknown): void {
  if (!isRecord(value)) {
    return;
  }

  const sessionId = findFirstString(value, ['sessionId', 'conversationId', 'chatSessionId', 'threadId', 'id']);
  if (sessionId !== null && sessionId !== '') {
    acc.id = sessionId;
  }

  const recordType = findFirstString(value, ['type']);
  if (recordType === 'session.shutdown') {
    acc.isFinal = true;
  }

  const workspaceDir =
    findFirstString(value, ['workspaceFolder', 'folderPath', 'workspacePath', 'cwd', 'workspaceDir', 'workspace']) ??
    null;
  if (workspaceDir !== null && workspaceDir !== '') {
    acc.workspaceDir = workspaceDir;
    acc.projectName = path.basename(workspaceDir);
  }

  const model = findFirstString(value, ['currentModel', 'model', 'modelId']);
  if (model !== null && model !== '' && acc.primaryModel === '') {
    acc.primaryModel = model;
  }

  const start = findFirstDate(value, ['startTime', 'created', 'timestamp', 'sessionStartTime']);
  if (start !== null && (acc.startTime.getTime() === 0 || start < acc.startTime)) {
    acc.startTime = start;
  }

  const end = findFirstDate(value, ['endTime', 'modified', 'updatedAt', 'timestamp']);
  if (end !== null && end > acc.endTime) {
    acc.endTime = end;
  }

  const premium = findFirstNumber(value, ['totalPremiumRequests', 'premiumRequests']);
  if (premium !== null) {
    acc.totalPremiumRequests += premium;
  }

  const currentTokens = findFirstNumber(value, ['currentTokens']);
  if (currentTokens !== null) {
    acc.tokens.currentTokens = Math.max(acc.tokens.currentTokens, currentTokens);
  }
  const systemTokens = findFirstNumber(value, ['systemTokens']);
  if (systemTokens !== null) {
    acc.tokens.systemTokens = Math.max(acc.tokens.systemTokens, systemTokens);
  }
  const conversationTokens = findFirstNumber(value, ['conversationTokens']);
  if (conversationTokens !== null) {
    acc.tokens.conversationTokens = Math.max(acc.tokens.conversationTokens, conversationTokens);
  }
  const toolTokens = findFirstNumber(value, ['toolDefinitionsTokens']);
  if (toolTokens !== null) {
    acc.tokens.toolDefinitionsTokens = Math.max(acc.tokens.toolDefinitionsTokens, toolTokens);
  }

  const modelMetrics = findFirstRecord(value, ['modelMetrics']);
  if (modelMetrics !== null) {
    acc.hasFinalBilling = true;
    acc.modelBuckets.clear();
    acc.totalTokens.input = 0;
    acc.totalTokens.output = 0;
    acc.totalTokens.cacheRead = 0;
    acc.totalTokens.cacheWrite = 0;
    acc.totalTokens.reasoning = 0;
    mergeIdeModelMetrics(acc, modelMetrics);
    return;
  }

  if (acc.hasFinalBilling) {
    return;
  }

  const inputTokens = findFirstNumber(value, ['inputTokens']);
  const outputTokens = findFirstNumber(value, ['outputTokens']);
  const cacheReadTokens = findFirstNumber(value, ['cacheReadTokens']);
  const cacheWriteTokens = findFirstNumber(value, ['cacheWriteTokens']);
  const reasoningTokens = findFirstNumber(value, ['reasoningTokens']);

  if (
    inputTokens !== null ||
    outputTokens !== null ||
    cacheReadTokens !== null ||
    cacheWriteTokens !== null ||
    reasoningTokens !== null
  ) {
    const bucket = bucketForModel(acc, model ?? acc.primaryModel);
    bucket.inputTokens += inputTokens ?? 0;
    bucket.outputTokens += outputTokens ?? 0;
    bucket.cacheReadTokens += cacheReadTokens ?? 0;
    bucket.cacheWriteTokens += cacheWriteTokens ?? 0;
    bucket.reasoningTokens += reasoningTokens ?? 0;
    acc.totalTokens.input += inputTokens ?? 0;
    acc.totalTokens.output += outputTokens ?? 0;
    acc.totalTokens.cacheRead += cacheReadTokens ?? 0;
    acc.totalTokens.cacheWrite += cacheWriteTokens ?? 0;
    acc.totalTokens.reasoning += reasoningTokens ?? 0;
  }
}

function mergeIdeModelMetrics(acc: IdeSessionAccumulator, modelMetrics: Record<string, unknown>): void {
  for (const [modelName, entry] of Object.entries(modelMetrics)) {
    if (!isRecord(entry)) {
      continue;
    }

    const usage = isRecord(entry.usage) ? entry.usage : undefined;
    const bucket = bucketForModel(acc, modelName);
    const inputTokens = numberFromValue(usage?.inputTokens);
    const outputTokens = numberFromValue(usage?.outputTokens);
    const cacheReadTokens = numberFromValue(usage?.cacheReadTokens);
    const cacheWriteTokens = numberFromValue(usage?.cacheWriteTokens);
    const reasoningTokens = numberFromValue(usage?.reasoningTokens);

    bucket.inputTokens += inputTokens;
    bucket.outputTokens += outputTokens;
    bucket.cacheReadTokens += cacheReadTokens;
    bucket.cacheWriteTokens += cacheWriteTokens;
    bucket.reasoningTokens += reasoningTokens;

    acc.totalTokens.input += inputTokens;
    acc.totalTokens.output += outputTokens;
    acc.totalTokens.cacheRead += cacheReadTokens;
    acc.totalTokens.cacheWrite += cacheWriteTokens;
    acc.totalTokens.reasoning += reasoningTokens;

    if (acc.primaryModel === '') {
      acc.primaryModel = modelName;
    }
  }
}

function bucketForModel(acc: IdeSessionAccumulator, model: string): IdeModelBucket {
  const key = model.trim() === '' ? 'unknown' : model;
  const existing = acc.modelBuckets.get(key);
  if (existing !== undefined) {
    return existing;
  }
  const bucket: IdeModelBucket = {
    inputTokens: 0,
    outputTokens: 0,
    cacheReadTokens: 0,
    cacheWriteTokens: 0,
    reasoningTokens: 0,
  };
  acc.modelBuckets.set(key, bucket);
  return bucket;
}

function buildIdeSession(acc: IdeSessionAccumulator): Session {
  const modelMetrics: Session['modelMetrics'] = [];
  let totalNanoAIU = 0;
  let primaryModel = acc.primaryModel;
  let bestCredits = -1;

  for (const [model, bucket] of acc.modelBuckets.entries()) {
    const credits = estimateIdeCredits(model, bucket.inputTokens, bucket.outputTokens);
    const nanoAIU = Math.round(credits * 1_000_000_000);
    totalNanoAIU += nanoAIU;
    modelMetrics.push({
      model,
      inputTokens: bucket.inputTokens,
      outputTokens: bucket.outputTokens,
      nanoAIU,
      cacheReadTokens: bucket.cacheReadTokens,
      cacheWriteTokens: bucket.cacheWriteTokens,
      reasoningTokens: bucket.reasoningTokens,
    });
    if (credits > bestCredits) {
      bestCredits = credits;
      primaryModel = model;
    }
  }

  if (modelMetrics.length === 0) {
    modelMetrics.push({
      model: primaryModel || 'unknown',
      inputTokens: 0,
      outputTokens: 0,
      nanoAIU: 0,
      cacheReadTokens: 0,
      cacheWriteTokens: 0,
      reasoningTokens: 0,
    });
  }

  const session: Session = {
    id: acc.id,
    workspaceDir: acc.workspaceDir,
    projectName: acc.projectName || (acc.workspaceDir !== '' ? path.basename(acc.workspaceDir) : ''),
    primaryModel,
    startTime: acc.startTime,
    endTime: acc.endTime,
    isActive: false,
    totalNanoAIU,
    totalPremiumRequests: acc.totalPremiumRequests,
    tokens: acc.tokens,
    modelMetrics,
    isFinal: acc.isFinal,
    source: 'copilot-ide',
  };

  if (session.primaryModel === '' && modelMetrics.length > 0) {
    session.primaryModel = modelMetrics[0].model;
  }

  return session;
}

const IDE_MODEL_RATES: Record<string, { inputPerMillion: number; outputPerMillion: number }> = {
  sonnet: { inputPerMillion: 300, outputPerMillion: 1500 },
  opus: { inputPerMillion: 500, outputPerMillion: 2500 },
  haiku: { inputPerMillion: 100, outputPerMillion: 500 },
};

function estimateIdeCredits(model: string, inputTokens: number, outputTokens: number): number {
  const rate = ideRateFor(model);
  return ((inputTokens * rate.inputPerMillion) + (outputTokens * rate.outputPerMillion)) / 1_000_000;
}

function ideRateFor(model: string): { inputPerMillion: number; outputPerMillion: number } {
  const lower = model.toLowerCase();
  for (const key of ['opus', 'sonnet', 'haiku']) {
    if (lower.includes(key)) {
      return IDE_MODEL_RATES[key];
    }
  }
  return IDE_MODEL_RATES.sonnet;
}

function findFirstRecord(value: unknown, keys: string[]): Record<string, unknown> | null {
  if (!isRecord(value)) {
    return null;
  }
  for (const key of keys) {
    const found = value[key];
    if (isRecord(found)) {
      return found;
    }
  }
  for (const nested of Object.values(value)) {
    const found = findFirstRecord(nested, keys);
    if (found !== null) {
      return found;
    }
  }
  return null;
}

function findFirstString(value: unknown, keys: string[]): string | null {
  if (!isRecord(value)) {
    return null;
  }
  for (const key of keys) {
    const found = value[key];
    if (typeof found === 'string' && found !== '') {
      return found;
    }
  }
  for (const nested of Object.values(value)) {
    const found = findFirstString(nested, keys);
    if (found !== null) {
      return found;
    }
  }
  return null;
}

function findFirstNumber(value: unknown, keys: string[]): number | null {
  if (!isRecord(value)) {
    return null;
  }
  for (const key of keys) {
    const found = value[key];
    const num = numberFromValue(found);
    if (num !== null) {
      return num;
    }
  }
  for (const nested of Object.values(value)) {
    const found = findFirstNumber(nested, keys);
    if (found !== null) {
      return found;
    }
  }
  return null;
}

function findFirstDate(value: unknown, keys: string[]): Date | null {
  if (!isRecord(value)) {
    return null;
  }
  for (const key of keys) {
    const found = value[key];
    const parsed = parseIdeTimestamp(found);
    if (parsed !== null) {
      return parsed;
    }
  }
  for (const nested of Object.values(value)) {
    const found = findFirstDate(nested, keys);
    if (found !== null) {
      return found;
    }
  }
  return null;
}

function parseIdeTimestamp(value: unknown): Date | null {
  if (typeof value === 'number' && Number.isFinite(value) && value > 0) {
    return new Date(value);
  }
  if (typeof value === 'string' && value !== '') {
    const parsed = new Date(value);
    if (!isNaN(parsed.getTime())) {
      return parsed;
    }
  }
  return null;
}

function numberFromValue(value: unknown): number {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value;
  }
  return 0;
}

// readThisMonth returns sessions whose billingTime falls in the current calendar month.
// Billing is attributed to the month a session finalizes (endTime), falling back to
// startTime for active sessions — see types.billingTime. Both year AND month are checked
// to handle year boundaries correctly (Jan 1 edge case). Mirrors Go ReadThisMonth.
export async function readThisMonth(): Promise<Session[]> {
  const all = await readSessions();
  const now = new Date();
  return all.filter(s => {
    const bt = billingTime(s);
    return bt.getFullYear() === now.getFullYear() && bt.getMonth() === now.getMonth();
  });
}

// enrichWithIdeMetadata reads the IDE session metadata file and marks CLI sessions
// that also had IDE Chat activity. The metadata file is at ~/.copilot/vscode.session.metadata.cache.json
// and contains references to sessions that had IDE Chat usage.
async function enrichWithIdeMetadata(sessions: Session[]): Promise<void> {
  const metadataPath = path.join(os.homedir(), '.copilot', 'vscode.session.metadata.cache.json');
  
  let ideMetadata: Record<string, unknown>;
  try {
    const content = await fs.promises.readFile(metadataPath, 'utf8');
    ideMetadata = JSON.parse(content) as Record<string, unknown>;
  } catch {
    // Metadata file absent or unreadable — no IDE enrichment needed
    return;
  }
  
  const ideSessionIds = new Set(Object.keys(ideMetadata));
  
  // Mark all CLI sessions that are also in the IDE metadata
  for (const s of sessions) {
    if (s.source === 'copilot-cli' && ideSessionIds.has(s.id)) {
      s.hasIdeActivity = true;
      console.log(`[enrichWithIdeMetadata] Session ${s.id.slice(0, 8)} has IDE activity`);
    }
  }
}

// readOneSession parses a single session directory into a Session object.
async function readOneSession(uuid: string, sessionDir: string): Promise<Session> {
  const session: Session = {
    id: uuid,
    workspaceDir: '',
    projectName: '',
    primaryModel: '',
    startTime: new Date(0),
    endTime: new Date(0),
    isActive: false,
    totalNanoAIU: 0,
    totalPremiumRequests: 0,
    tokens: { currentTokens: 0, systemTokens: 0, conversationTokens: 0, toolDefinitionsTokens: 0 },
    modelMetrics: [],
    isFinal: false,
    source: 'copilot-cli',
  };

  // Detect active session: any inuse.*.lock file present.
  session.isActive = await hasLockFile(sessionDir);

  // workspace.yaml provides workspaceDir without JSONL parsing.
  session.workspaceDir = await readWorkspaceCWD(sessionDir);

  // Parse events.jsonl for billing and timing fields. A missing/unreadable events.jsonl
  // is fatal for the session — Go's readSession returns the open error and readAll skips
  // the session. We mirror that: a session dir with workspace.yaml but no events.jsonl is
  // dropped (throws here → caller logs and skips), not kept with zeros.
  await parseEventsFile(sessionDir, session);

  if (session.workspaceDir !== '') {
    session.projectName = path.basename(session.workspaceDir);
  }

  return session;
}

// hasLockFile checks for the presence of any inuse.*.lock file in a session dir.
async function hasLockFile(sessionDir: string): Promise<boolean> {
  try {
    const files = await fs.promises.readdir(sessionDir);
    return files.some(f => f.startsWith('inuse.') && f.endsWith('.lock'));
  } catch {
    return false;
  }
}

// readWorkspaceCWD reads the cwd: field from workspace.yaml using async readline.
// Returns empty string if the file is absent or the field is not found.
async function readWorkspaceCWD(sessionDir: string): Promise<string> {
  const yamlPath = path.join(sessionDir, 'workspace.yaml');
  let fileStream: fs.ReadStream | undefined;
  let rl: readline.Interface | undefined;
  try {
    fileStream = fs.createReadStream(yamlPath, { encoding: 'utf8' });
    rl = readline.createInterface({ input: fileStream, crlfDelay: Infinity });
    for await (const line of rl) {
      if (line.startsWith('cwd:')) {
        return line.slice('cwd:'.length).trim();
      }
    }
  } catch {
    // File absent or unreadable — return empty string, not an error.
  } finally {
    // Close the readline interface and destroy the stream on every path
    // (match found, EOF, or error) to avoid leaking file handles.
    rl?.close();
    fileStream?.destroy();
  }
  return '';
}

// Envelope shape for the top-level JSON in each events.jsonl line.
interface EventEnvelope {
  type: string;
  timestamp?: string;
  data?: unknown;
}

// Shape of session.start data.
interface StartEventData {
  startTime?: string;
  context?: { cwd?: string };
}

// Shape of session.shutdown data.
interface ShutdownEventData {
  totalNanoAiu?: number;
  totalPremiumRequests?: number; // only on session.shutdown — premium request count
  sessionStartTime?: number;   // Unix epoch milliseconds — fallback for startTime
  currentModel?: string;
  currentTokens?: number;
  systemTokens?: number;
  conversationTokens?: number;
  toolDefinitionsTokens?: number;
  modelMetrics?: Record<string, ModelMetricsEntry>;
}

interface ModelMetricsEntry {
  totalNanoAiu?: number;
  // All token counts live under usage.* — matches the real schema and the Go side,
  // which read inputTokens/outputTokens/cacheReadTokens/cacheWriteTokens/reasoningTokens
  // from modelMetrics.<model>.usage, not the metrics entry's top level.
  usage?: {
    inputTokens?: number;
    outputTokens?: number;
    cacheReadTokens?: number;   // phase 6: prompt caching reads
    cacheWriteTokens?: number;  // phase 6: prompt caching writes
    reasoningTokens?: number;   // phase 6: extended thinking tokens
  };
}

// parseEventsFile reads events.jsonl line by line using readline (async, no readFileSync).
async function parseEventsFile(sessionDir: string, session: Session): Promise<void> {
  const eventsPath = path.join(sessionDir, 'events.jsonl');
  const fileStream = fs.createReadStream(eventsPath, { encoding: 'utf8' });
  const rl = readline.createInterface({ input: fileStream, crlfDelay: Infinity });

  try {
    for await (const line of rl) {
      const trimmed = line.trim();
      if (trimmed === '') {
        continue;
      }

      let envelope: EventEnvelope;
      try {
        envelope = JSON.parse(trimmed) as EventEnvelope;
      } catch {
        continue; // skip malformed lines — mirrors Go behaviour
      }

      if (envelope.type === 'session.start') {
        applyStartEvent(envelope.data as StartEventData, session);
      } else if (envelope.type === 'session.shutdown') {
        applyShutdownEvent(
          envelope.data as ShutdownEventData,
          envelope.timestamp ?? '',
          session
        );
      } else {
        // Any other event may carry a running billing/token snapshot. Apply it as the
        // latest live reading so active sessions display in-progress spend instead of
        // zeros. Events are appended chronologically (last-write-wins). A partial snapshot
        // must never overwrite a final (shutdown) reading. Mirrors Go applySnapshotEvent.
        applySnapshotEvent(envelope.data as ShutdownEventData, session);
      }
    }
  } finally {
    // Close the readline interface and destroy the stream on every path (EOF, error,
    // or early throw) to avoid leaking file handles — mirrors readWorkspaceCWD.
    rl.close();
    fileStream.destroy();
  }
}

// applySnapshotEvent applies billing/token fields from a non-shutdown event as a
// running (non-final) snapshot. It only acts when the event carries a billing signal
// (totalNanoAiu > 0 OR currentTokens > 0) and the session is not yet final. The shape
// reuses ShutdownEventData because running snapshots carry the same billing fields.
function applySnapshotEvent(data: ShutdownEventData | undefined, session: Session): void {
  if (data == null || session.isFinal) {
    return;
  }

  const nano = data.totalNanoAiu ?? 0;
  const current = data.currentTokens ?? 0;
  if (nano <= 0 && current <= 0) {
    return; // no billing signal — nothing to apply
  }

  applyBillingFields(data, session);
  // isFinal stays false — this is a live partial, not a settled reading.
}

// applyBillingFields populates totalNanoAIU, tokens, modelMetrics, and primaryModel
// from a shutdown- or snapshot-shaped payload. Shared by applyShutdownEvent and
// applySnapshotEvent so partial and final readings stay in sync.
function applyBillingFields(data: ShutdownEventData, session: Session): void {
  session.totalNanoAIU = data.totalNanoAiu ?? 0;

  session.tokens = {
    currentTokens: data.currentTokens ?? 0,
    systemTokens: data.systemTokens ?? 0,
    conversationTokens: data.conversationTokens ?? 0,
    toolDefinitionsTokens: data.toolDefinitionsTokens ?? 0,
  };

  // Rebuild modelMetrics fresh each time; derive primaryModel as the model with
  // highest nanoAIU (falling back to currentModel).
  session.modelMetrics = [];
  session.primaryModel = data.currentModel ?? '';
  let bestNano = 0;

  for (const [modelName, entry] of Object.entries(data.modelMetrics ?? {})) {
    const nanoAIU = entry.totalNanoAiu ?? 0;
    session.modelMetrics.push({
      model: modelName,
      inputTokens: entry.usage?.inputTokens ?? 0,
      outputTokens: entry.usage?.outputTokens ?? 0,
      nanoAIU,
      cacheReadTokens: entry.usage?.cacheReadTokens ?? 0,
      cacheWriteTokens: entry.usage?.cacheWriteTokens ?? 0,
      reasoningTokens: entry.usage?.reasoningTokens ?? 0,
    });

    if (nanoAIU > bestNano) {
      bestNano = nanoAIU;
      session.primaryModel = modelName;
    }
  }
}

// applyStartEvent populates startTime and workspaceDir from a session.start event.
function applyStartEvent(data: StartEventData | undefined, session: Session): void {
  if (data == null) {
    return;
  }
  if (session.startTime.getTime() === 0 && data.startTime) {
    const t = parseTimestamp(data.startTime);
    if (t !== null) {
      session.startTime = t;
    }
  }
  if (session.workspaceDir === '' && data.context?.cwd) {
    session.workspaceDir = data.context.cwd;
  }
}

// applyShutdownEvent populates billing fields from a session.shutdown event.
function applyShutdownEvent(
  data: ShutdownEventData | undefined,
  topTimestamp: string,
  session: Session
): void {
  if (data == null) {
    return;
  }

  // Shutdown is authoritative: it always overwrites any prior running snapshot and
  // marks the session final so later (out-of-order) partials cannot clobber it.
  applyBillingFields(data, session);
  // totalPremiumRequests is only carried by the shutdown event, so capture it here
  // rather than in the shared applyBillingFields (snapshot events do not have it).
  session.totalPremiumRequests = data.totalPremiumRequests ?? 0;
  session.isFinal = true;

  // endTime from the top-level event timestamp.
  const endT = parseTimestamp(topTimestamp);
  if (endT !== null) {
    session.endTime = endT;
  }

  // Fallback startTime: sessionStartTime is Unix epoch milliseconds.
  if (session.startTime.getTime() === 0 && data.sessionStartTime && data.sessionStartTime > 0) {
    session.startTime = new Date(data.sessionStartTime);
  }
}

// parseTimestamp parses an ISO 8601 / RFC 3339 string into a Date.
// Returns null on failure — callers check for null or getTime() === 0.
function parseTimestamp(s: string): Date | null {
  if (!s) {
    return null;
  }
  const d = new Date(s);
  return isNaN(d.getTime()) ? null : d;
}
