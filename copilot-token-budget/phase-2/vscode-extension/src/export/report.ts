// export/report.ts — deterministic JSON / CSV serialization of the tool's computed
// state for the `copilotBudget.exportUsage` command. TypeScript port of the shapes in
// phase-1/session-manager/internal/export/export.go (stable field names and column
// order so saved reports diff cleanly across runs). Pure with respect to its inputs —
// it never touches the file system (the command layer writes the returned strings).
//
// Zero npm runtime deps (ADR-003).

import { Session, billingTime, totalInputTokens, totalOutputTokens } from '../types';
import { BudgetState } from '../types';
import { fromNanoAIU } from '../budget/tracker';
import {
  Bucket,
  Consumer,
  dailySeries,
  topSessions,
  topModels,
  topProjects,
} from '../analytics/model';

// SessionView is the flattened, serialization-friendly projection of a session.
// Mirrors Go export.SessionView (field names and order).
interface SessionView {
  id: string;
  source: string;
  project: string;
  model: string;
  billingDate: string; // "YYYY-MM-DD"
  credits: number;
  inputTokens: number;
  outputTokens: number;
  systemTokens: number;
  isActive: boolean;
  isFinal: boolean;
  startTime: string;
  endTime?: string;
}

// Report is the top-level aggregate serialized by reportToJson. Mirrors Go export.Report.
interface Report {
  generatedAt: string;
  budgetState: BudgetState;
  daily: Bucket[];
  topSessions: Consumer[];
  topModels: Consumer[];
  topProjects: Consumer[];
  sessions: SessionView[];
}

// buildReport assembles the full report from this month's sessions and the budget state.
export function buildReport(sessions: Session[], budgetState: BudgetState): Report {
  return {
    generatedAt: new Date().toISOString(),
    budgetState,
    daily: dailySeries(sessions),
    topSessions: topSessions(sessions, 0),
    topModels: topModels(sessions, 0),
    topProjects: topProjects(sessions, 0),
    sessions: sessions.map(sessionView),
  };
}

// reportToJson serializes a report as indented, deterministic JSON. Determinism comes
// from the analytics slices already being sorted and JSON.stringify preserving order.
export function reportToJson(sessions: Session[], budgetState: BudgetState): string {
  return JSON.stringify(buildReport(sessions, budgetState), null, 2);
}

// sessionsToCsv writes one row per session with a stable header. Columns mirror Go
// export.SessionsToCSV: date,project,model,source,credits,inputTokens,outputTokens,
// systemTokens,isActive,isFinal.
export function sessionsToCsv(sessions: Session[]): string {
  const header = [
    'date', 'project', 'model', 'source',
    'credits', 'inputTokens', 'outputTokens', 'systemTokens',
    'isActive', 'isFinal',
  ];
  const lines = [header.join(',')];
  for (const s of sessions) {
    const row = [
      formatDay(billingTime(s)),
      s.projectName,
      s.primaryModel,
      s.source,
      formatCredits(fromNanoAIU(s.totalNanoAIU)),
      String(totalInputTokens(s)),
      String(totalOutputTokens(s)),
      String(s.tokens.systemTokens),
      String(s.isActive),
      String(s.isFinal),
    ].map(csvField);
    lines.push(row.join(','));
  }
  return lines.join('\n') + '\n';
}

// sessionView builds the flattened view for one session. Mirrors Go NewSessionView.
function sessionView(s: Session): SessionView {
  const view: SessionView = {
    id: s.id,
    source: s.source,
    project: s.projectName,
    model: s.primaryModel,
    billingDate: formatDay(billingTime(s)),
    credits: fromNanoAIU(s.totalNanoAIU),
    inputTokens: totalInputTokens(s),
    outputTokens: totalOutputTokens(s),
    systemTokens: s.tokens.systemTokens,
    isActive: s.isActive,
    isFinal: s.isFinal,
    startTime: s.startTime.toISOString(),
  };
  // endTime is omitted when unset (epoch 0), mirroring Go's `omitempty`.
  if (s.endTime.getTime() !== 0) {
    view.endTime = s.endTime.toISOString();
  }
  return view;
}

// csvField quotes a field when it contains a comma, quote, or newline (RFC 4180).
function csvField(value: string): string {
  if (/[",\n\r]/.test(value)) {
    return '"' + value.replace(/"/g, '""') + '"';
  }
  return value;
}

// formatCredits renders a credit value compactly, trimming trailing zeros (Number's
// default string form does this) and never using exponential notation — matching Go's
// strconv.FormatFloat(c, 'f', -1, 64). String(c) is already shortest-round-trip and
// non-exponential for all realistic credit magnitudes (roughly 1e-6 .. 1e21); the guard
// expands the rare exponential form so the CSV stays plain-decimal like the Go output.
function formatCredits(c: number): string {
  const s = String(c);
  if (!/[eE]/.test(s)) {
    return s;
  }
  // Expand exponential notation to a plain decimal string without trailing zeros.
  // toFixed(20) is enough precision for any double; trim the trailing zeros/dot after.
  return c.toFixed(20).replace(/\.?0+$/, '');
}

// formatDay formats a local date as "YYYY-MM-DD" to match the Go billing-date format.
function formatDay(d: Date): string {
  const y = d.getFullYear().toString().padStart(4, '0');
  const m = (d.getMonth() + 1).toString().padStart(2, '0');
  const day = d.getDate().toString().padStart(2, '0');
  return `${y}-${m}-${day}`;
}
