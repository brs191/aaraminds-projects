// livebilling/labels.ts — UI labels and snapshot selection for live billing.

import { LiveBillingSnapshot, Session } from '../types';

export function liveBillingLabel(snapshot?: LiveBillingSnapshot, now: Date = new Date()): string {
  if (snapshot === undefined) {
    return '(estimated)';
  }
  if (snapshot.availability === 'unavailable') {
    return '(unavailable)';
  }
  // If SourceLabel is already set (by the refresher), use it.
  if (snapshot.sourceLabel) {
    return snapshot.sourceLabel;
  }
  // Fallback for snapshots created without explicit SourceLabel.
  return `(org aggregate, ~${hoursAgo(snapshot.lastRefreshedAt, now)}h ago)`;
}

export function latestLiveBillingSnapshot(sessions: Session[]): LiveBillingSnapshot | undefined {
  let latest: LiveBillingSnapshot | undefined;
  for (const s of sessions) {
    const snap = s.orgBillingSnapshot;
    if (snap === undefined) {
      continue;
    }
    if (latest === undefined || snap.lastRefreshedAt.getTime() > latest.lastRefreshedAt.getTime()) {
      latest = snap;
    }
  }
  return latest;
}

function hoursAgo(t: Date, now: Date): number {
  const ms = now.getTime() - t.getTime();
  if (ms <= 0) {
    return 1;
  }
  return Math.max(1, Math.floor(ms / (60 * 60 * 1000)));
}
