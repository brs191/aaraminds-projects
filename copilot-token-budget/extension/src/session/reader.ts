// session/reader.ts — async JSONL session reader for the VS Code extension.
// TypeScript port of phase-1/session-manager/internal/session/reader.go.
// Uses only Node.js built-ins (fs, path, readline, os) — zero npm runtime deps (ADR-003).

import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';
import * as readline from 'readline';
import { Session, SessionSource, billingTime } from '../types';

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

// ideCollector is intentionally a no-op stub. It previously read ~/.copilot/session-state
// dirs carrying a vscode.metadata.json marker, but that source is mis-pointed: VS Code
// Copilot Chat is a SEPARATE source (chatSessions/transcripts under VS Code user data,
// NOT ~/.copilot) and needs a real rewrite. Mirrors the Go ideCollector revert.
// TODO: VS Code Chat is a separate source — chatSessions/transcripts under VS Code user
// data, not ~/.copilot; pending real implementation, see ADR-007.
const ideCollector: Collector = {
  name(): SessionSource {
    return 'copilot-ide';
  },
  async collect(): Promise<Session[]> {
    return [];
  },
};

// collectors is the ordered set of sources readSessions merges. CLI first so that,
// all else equal, a CLI record is encountered before an IDE record for the same id.
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

  const deduped = dedupById(merged);

  // Sort newest first — mirrors Go sort.Slice on StartTime.After.
  deduped.sort((a, b) => b.startTime.getTime() - a.startTime.getTime());
  return deduped;
}

// dedupById collapses sessions sharing an id to a single record per the readSessions
// dedup rule (final wins; else higher totalNanoAIU). Sessions with an empty id are
// passed through untouched (they cannot be keyed). Mirrors Go dedupByID.
function dedupById(sessions: Session[]): Session[] {
  const best = new Map<string, Session>();
  const order: string[] = [];
  const unkeyed: Session[] = [];

  for (const s of sessions) {
    if (s.id === '') {
      unkeyed.push(s);
      continue;
    }
    const prev = best.get(s.id);
    if (prev === undefined) {
      best.set(s.id, s);
      order.push(s.id);
      continue;
    }
    if (preferSession(prev, s)) {
      best.set(s.id, s);
    }
  }

  const out: Session[] = [];
  for (const id of order) {
    const s = best.get(id);
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

