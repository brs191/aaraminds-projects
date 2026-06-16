// types.ts — single source of truth for all Copilot Token Budget data types.
// Mirrors the Go data model in phase-1/session-manager/internal/session/reader.go exactly.
// All UI components import from here — never redefine these shapes elsewhere.

export interface TokenBreakdown {
  currentTokens: number;
  systemTokens: number;        // instruction file overhead — key budget metric
  conversationTokens: number;
  toolDefinitionsTokens: number;
}

export interface ModelMetric {
  model: string;
  inputTokens: number;
  outputTokens: number;
  nanoAIU: number;
  cacheReadTokens: number;      // phase 6: prompt caching reads
  cacheWriteTokens: number;     // phase 6: prompt caching writes
  reasoningTokens: number;      // phase 6: extended thinking tokens
}

// SessionSource identifies which collector produced a session.
//   'copilot-cli' — GitHub Copilot CLI session-state (the only live source today).
//   'copilot-ide' — VS Code IDE Copilot usage (Phase 6 stub; emits nothing yet).
// Mirrors Go session.Session.Source. Kept as a string union (not a free string)
// to stay strict-typed; the dedup step in the reader reasons about cross-source overlap.
export type SessionSource = 'copilot-cli' | 'copilot-ide';

export interface Session {
  id: string;
  workspaceDir: string;
  projectName: string;          // path.basename(workspaceDir)
  primaryModel: string;
  startTime: Date;
  endTime: Date;
  isActive: boolean;            // true when inuse.*.lock file present in session dir
  totalNanoAIU: number;
  tokens: TokenBreakdown;
  modelMetrics: ModelMetric[];

  // source identifies which Collector produced this session. Lets the dedup step
  // in readSessions reason about cross-source overlap. Mirrors Go Session.Source.
  source: SessionSource;

  // isFinal reports whether the billing/token figures are authoritative.
  // true  → a session.shutdown event has been applied (final, settled billing).
  // false → the figures are a partial/live reading from a running snapshot event
  //         (or still zero for an active session that has not emitted billing yet).
  // A partial snapshot must never overwrite a final reading. Mirrors Go Session.IsFinal.
  isFinal: boolean;
}

export interface InstructionFile {
  path: string;                 // absolute path
  scope: string;                // "workspace-root" or "project-scoped"
  project: string;              // basename of project dir (project-scoped only)
  estimatedTokens: number;      // rough estimate: UTF-8 byteLength(content) / 4 (matches Go)
}

export interface BudgetState {
  usedCredits: number;
  allowedCredits: number;
  usedPct: number;
  remainingCredits: number;
  status: 'OK' | 'WARNING' | 'CRITICAL';
}

// Helper functions — mirrors Go method receivers Session.TotalInputTokens / TotalOutputTokens.
// Implemented as free functions (not methods) for simpler import in UI components.

export function totalInputTokens(s: Session): number {
  return s.modelMetrics.reduce((sum, m) => sum + m.inputTokens, 0);
}

export function totalOutputTokens(s: Session): number {
  return s.modelMetrics.reduce((sum, m) => sum + m.outputTokens, 0);
}

// billingTime returns the time used to attribute a session to a calendar month.
// It is endTime (shutdown time) when set, otherwise startTime. Mirrors Go
// Session.BillingTime: spend is attributed to the month a session finalizes, not
// the month it started. Active sessions have no endTime and fall back to startTime.
export function billingTime(s: Session): Date {
  return s.endTime.getTime() !== 0 ? s.endTime : s.startTime;
}
