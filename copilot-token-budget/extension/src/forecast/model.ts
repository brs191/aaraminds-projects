// forecast/model.ts — month-end credit burn-rate modelling for the VS Code extension.
// Ported faithfully from phase-3/internal/forecast/model.go and the daysElapsed/
// daysRemaining computation in phase-3/cmd/alert/main.go.
//
// Pure arithmetic, zero npm runtime deps (ADR-003). The displayed forecast is the
// projected month-end TOTAL (used + dailyBurn × daysRemaining), never just the
// remaining-days portion — so it is never hidden on the last day of the month.

// MonthWindow holds the calendar position used to project month-end spend.
export interface MonthWindow {
  daysElapsed: number;   // calendar day-of-month (1-based), == Go today.Day()
  daysInMonth: number;   // total days in the current month
  daysRemaining: number; // daysInMonth - daysElapsed
}

// Forecast is the computed burn-rate projection surfaced in the UI.
export interface Forecast {
  dailyBurn: number;            // credits/day
  projectedMonthEndTotal: number; // used + dailyBurn × daysRemaining (cr)
  daysElapsed: number;
  daysRemaining: number;
  exceedsAllowance: boolean;    // projectedMonthEndTotal > allowance
}

// monthWindow computes daysElapsed / daysInMonth / daysRemaining for the given date.
// Mirrors Go: daysElapsed = today.Day(); lastDay = Date(year, month+1, 0); daysInMonth
// = lastDay.getDate(); daysRemaining = daysInMonth - daysElapsed.
export function monthWindow(today: Date = new Date()): MonthWindow {
  const daysElapsed = today.getDate();
  // new Date(year, month+1, 0) rolls back to the last day of the current month.
  const daysInMonth = new Date(today.getFullYear(), today.getMonth() + 1, 0).getDate();
  const daysRemaining = daysInMonth - daysElapsed;
  return { daysElapsed, daysInMonth, daysRemaining };
}

// dailyBurnRate returns average credits consumed per day.
// Returns 0 when daysElapsed <= 0 to guard against division by zero.
// Mirrors Go forecast.DailyBurnRate (caller passes already-summed usedCredits).
export function dailyBurnRate(usedCredits: number, daysElapsed: number): number {
  if (daysElapsed <= 0) {
    return 0;
  }
  return usedCredits / daysElapsed;
}

// projectedMonthEndTotal returns the projected total credits by month end:
// usedCredits + dailyBurn × daysRemaining. Mirrors the Go alert card's
// projectedTotal = state.UsedCredits + MonthEndForecast(dailyBurn, daysRemaining).
// daysRemaining <= 0 contributes nothing, so on the last day this equals usedCredits.
export function projectedMonthEndTotal(
  usedCredits: number,
  dailyBurn: number,
  daysRemaining: number
): number {
  const remainingForecast = daysRemaining > 0 ? dailyBurn * daysRemaining : 0;
  return usedCredits + remainingForecast;
}

// computeForecast bundles the full projection from used credits + allowance, using
// the current (or supplied) date for the month window.
export function computeForecast(
  usedCredits: number,
  allowance: number,
  today: Date = new Date()
): Forecast {
  const { daysElapsed, daysRemaining } = monthWindow(today);
  const dailyBurn = dailyBurnRate(usedCredits, daysElapsed);
  const total = projectedMonthEndTotal(usedCredits, dailyBurn, daysRemaining);
  return {
    dailyBurn,
    projectedMonthEndTotal: total,
    daysElapsed,
    daysRemaining,
    exceedsAllowance: total > allowance,
  };
}
