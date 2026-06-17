// analytics/model.ts — passive, pure aggregations over a slice of sessions.
// TypeScript port of phase-1/session-manager/internal/analytics/analytics.go.
//
// Every function is pure: it takes the sessions (and, where cost is involved, a
// PricingConfig) and returns derived values without touching the file system, the
// clock, or any global state. Date bucketing always uses billingTime so spend lands
// in the month it settled, consistent with the rest of the tool. All bucketing is done
// in UTC (getUTC* + a UTC ISO-week port) so a near-midnight session lands in the same
// day/week/month as the Go CLI regardless of the machine's local timezone. Credits are
// computed via budget.fromNanoAIU so this module agrees with the budget package to the
// last nano.
//
// Zero npm runtime deps (ADR-003).

import { Session, billingTime, totalInputTokens, totalOutputTokens } from '../types';
import { fromNanoAIU } from '../budget/tracker';
import { PricingConfig, rateFor } from '../pricing/config';

// Bucket is one time slice (day, ISO week, or month) of aggregated usage.
export interface Bucket {
  key: string;        // human-stable label: "YYYY-MM-DD", "YYYY-Www", or "YYYY-MM"
  start: string;      // bucket lower time bound, ISO string (UTC-derived)
  sessions: number;   // count of sessions attributed to this bucket
  credits: number;    // total credits consumed in the bucket
  inputTokens: number;
  outputTokens: number;
  byModel: Record<string, number>; // model name -> credits consumed in the bucket
}

// Consumer is a ranked aggregate row (a session, model, or project).
export interface Consumer {
  name: string;        // display label (project / session id, model name, etc.)
  credits: number;     // total credit spend
  inputTokens: number;
  outputTokens: number;
  model: string;       // primary model, where one consumer maps to a single model
}

// sessionCredits returns a session's credit cost from its settled nanoAIU.
function sessionCredits(s: Session): number {
  return fromNanoAIU(s.totalNanoAIU);
}

// KeyFn maps a billing time to a (key, bucketStart) pair.
type KeyFn = (t: Date) => { key: string; start: Date };

// dailySeries returns one Bucket per calendar day that has data, keyed "YYYY-MM-DD"
// and sorted ascending by start. Mirrors Go DailySeries (UTC-day bucketing).
export function dailySeries(sessions: Session[]): Bucket[] {
  return series(sessions, t => {
    const day = utcMidnight(t);
    return { key: formatDay(day), start: day };
  });
}

// weeklySeries returns one Bucket per ISO week that has data, keyed "YYYY-Www"
// (ISO-year + ISO-week) and sorted ascending. start is the Monday of the ISO week.
// Mirrors Go WeeklySeries.
export function weeklySeries(sessions: Session[]): Bucket[] {
  return series(sessions, t => {
    const { year, week } = isoWeek(t);
    // Monday of the ISO week containing t, in UTC. getUTCDay(): Sunday == 0; ISO
    // treats Sunday as 7.
    let weekday = t.getUTCDay();
    if (weekday === 0) {
      weekday = 7;
    }
    const monday = utcMidnight(t);
    monday.setUTCDate(monday.getUTCDate() - (weekday - 1));
    return { key: isoWeekKey(year, week), start: monday };
  });
}

// monthlySeries returns one Bucket per calendar month that has data, keyed "YYYY-MM"
// and sorted ascending. Mirrors Go MonthlySeries.
export function monthlySeries(sessions: Session[]): Bucket[] {
  return series(sessions, t => {
    const month = new Date(Date.UTC(t.getUTCFullYear(), t.getUTCMonth(), 1, 0, 0, 0, 0));
    return { key: formatMonth(month), start: month };
  });
}

// series is the shared bucketing engine. keyFn maps a billing time to a (key,
// bucketStart) pair; sessions sharing a key are aggregated and the result is sorted
// ascending by start. Only buckets with at least one session are returned. Mirrors
// Go series.
function series(sessions: Session[], keyFn: KeyFn): Bucket[] {
  const byKey = new Map<string, Bucket>();
  for (const s of sessions) {
    const { key, start } = keyFn(billingTime(s));
    let b = byKey.get(key);
    if (b === undefined) {
      b = {
        key,
        start: start.toISOString(),
        sessions: 0,
        credits: 0,
        inputTokens: 0,
        outputTokens: 0,
        byModel: {},
      };
      byKey.set(key, b);
    }
    b.sessions++;
    b.credits += sessionCredits(s);
    b.inputTokens += totalInputTokens(s);
    b.outputTokens += totalOutputTokens(s);
    for (const m of s.modelMetrics) {
      b.byModel[m.model] = (b.byModel[m.model] ?? 0) + fromNanoAIU(m.nanoAIU);
    }
  }

  const out = Array.from(byKey.values());
  out.sort((a, b) => new Date(a.start).getTime() - new Date(b.start).getTime());
  return out;
}

// topSessions returns up to n sessions ranked by credits descending. Each row's name
// is the project name, falling back to the session id when the project is unknown.
// Ties break by name ascending for determinism. Mirrors Go TopSessions.
export function topSessions(sessions: Session[], n: number): Consumer[] {
  const rows: Consumer[] = sessions.map(s => ({
    name: s.projectName !== '' ? s.projectName : s.id,
    credits: sessionCredits(s),
    inputTokens: totalInputTokens(s),
    outputTokens: totalOutputTokens(s),
    model: s.primaryModel,
  }));
  sortConsumers(rows);
  return topN(rows, n);
}

// topModels aggregates per-model credits across all sessions (summing each session's
// byModel contribution) and returns up to n models by credits descending. Mirrors
// Go TopModels.
export function topModels(sessions: Session[], n: number): Consumer[] {
  const byModel = new Map<string, Consumer>();
  for (const s of sessions) {
    for (const m of s.modelMetrics) {
      let a = byModel.get(m.model);
      if (a === undefined) {
        a = { name: m.model, credits: 0, inputTokens: 0, outputTokens: 0, model: m.model };
        byModel.set(m.model, a);
      }
      a.credits += fromNanoAIU(m.nanoAIU);
      a.inputTokens += m.inputTokens;
      a.outputTokens += m.outputTokens;
    }
  }
  const rows = Array.from(byModel.values());
  sortConsumers(rows);
  return topN(rows, n);
}

// topProjects aggregates credits and tokens by project name across all sessions and
// returns up to n projects by credits descending. Sessions with no project name
// aggregate under their id. Mirrors Go TopProjects.
export function topProjects(sessions: Session[], n: number): Consumer[] {
  const byProject = new Map<string, Consumer>();
  for (const s of sessions) {
    const name = s.projectName !== '' ? s.projectName : s.id;
    let a = byProject.get(name);
    if (a === undefined) {
      a = { name, credits: 0, inputTokens: 0, outputTokens: 0, model: '' };
      byProject.set(name, a);
    }
    a.credits += sessionCredits(s);
    a.inputTokens += totalInputTokens(s);
    a.outputTokens += totalOutputTokens(s);
    if (a.model === '') {
      a.model = s.primaryModel;
    }
  }
  const rows = Array.from(byProject.values());
  sortConsumers(rows);
  return topN(rows, n);
}

// sortConsumers orders by credits desc, breaking ties by name asc for deterministic
// output. Mirrors Go sortConsumers.
function sortConsumers(rows: Consumer[]): void {
  rows.sort((a, b) => {
    if (a.credits !== b.credits) {
      return b.credits - a.credits;
    }
    return a.name < b.name ? -1 : a.name > b.name ? 1 : 0;
  });
}

// topN returns the first n rows, or all of them when n <= 0 or n >= len. Mirrors Go topN.
function topN(rows: Consumer[], n: number): Consumer[] {
  if (n <= 0 || n >= rows.length) {
    return rows;
  }
  return rows.slice(0, n);
}

// contextWindowPct returns how full a session's context window is, as a percent of the
// primary model's contextWindowTokens from cfg. Returns 0 when the window is
// unknown/zero to avoid a divide-by-zero. Mirrors Go ContextWindowPct.
export function contextWindowPct(s: Session, cfg: PricingConfig): number {
  const window = rateFor(cfg, s.primaryModel).contextWindowTokens;
  if (window <= 0) {
    return 0;
  }
  return (s.tokens.currentTokens / window) * 100;
}

// anomalousDays flags days whose credits exceed mean + 2*population-stddev of the
// supplied daily series. It returns the flagged buckets in input order. The result is
// empty when there are fewer than 3 data points (too few to define a distribution).
// Mirrors Go AnomalousDays exactly (population variance).
export function anomalousDays(daily: Bucket[]): Bucket[] {
  if (daily.length < 3) {
    return [];
  }
  const n = daily.length;
  let sum = 0;
  for (const b of daily) {
    sum += b.credits;
  }
  const mean = sum / n;

  let variance = 0;
  for (const b of daily) {
    const d = b.credits - mean;
    variance += d * d;
  }
  variance /= n; // population variance
  const threshold = mean + 2 * Math.sqrt(variance);

  const out: Bucket[] = [];
  for (const b of daily) {
    if (b.credits > threshold) {
      out.push(b);
    }
  }
  return out;
}

// utcMidnight returns 00:00:00.000 UTC of the day containing t.
function utcMidnight(t: Date): Date {
  return new Date(Date.UTC(t.getUTCFullYear(), t.getUTCMonth(), t.getUTCDate(), 0, 0, 0, 0));
}

// formatDay formats a date's UTC calendar day as "YYYY-MM-DD" (matches Go "2006-01-02").
function formatDay(d: Date): string {
  return `${pad4(d.getUTCFullYear())}-${pad2(d.getUTCMonth() + 1)}-${pad2(d.getUTCDate())}`;
}

// formatMonth formats a date's UTC calendar month as "YYYY-MM" (matches Go "2006-01").
function formatMonth(d: Date): string {
  return `${pad4(d.getUTCFullYear())}-${pad2(d.getUTCMonth() + 1)}`;
}

// isoWeekKey formats an ISO year/week pair as "YYYY-Www" with a zero-padded week.
// Mirrors Go isoWeekKey.
function isoWeekKey(year: number, week: number): string {
  return `${pad4(year)}-W${pad2(week)}`;
}

// isoWeek computes the ISO-8601 year and week number for a date's UTC calendar day,
// matching Go's time.Time.ISOWeek() (Go buckets in UTC). ISO weeks start on Monday and
// week 1 is the week containing the first Thursday of the year (equivalently, the week
// containing Jan 4). All arithmetic is in UTC so the result is timezone-independent.
function isoWeek(t: Date): { year: number; week: number } {
  // Work in UTC-date space (year/month/day) to match Go's UTC bucketing.
  const d = utcMidnight(t);
  // ISO weekday: Mon=1 .. Sun=7.
  let day = d.getUTCDay();
  if (day === 0) {
    day = 7;
  }
  // Shift to the Thursday of the current ISO week: the ISO year is the year of that
  // Thursday, and week 1 is the week containing it.
  const thursday = new Date(d);
  thursday.setUTCDate(d.getUTCDate() + (4 - day));
  const isoYear = thursday.getUTCFullYear();
  const jan1 = new Date(Date.UTC(isoYear, 0, 1));
  const week = Math.floor((thursday.getTime() - jan1.getTime()) / (7 * 24 * 60 * 60 * 1000)) + 1;
  return { year: isoYear, week };
}

function pad2(n: number): string {
  return n < 10 ? `0${n}` : String(n);
}

function pad4(n: number): string {
  const v = n < 0 ? 0 : n;
  return v.toString().padStart(4, '0');
}
