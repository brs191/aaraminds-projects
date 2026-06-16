// budget/tracker.ts — credit conversion and budget state calculation.
// TypeScript port of phase-1/session-manager/internal/budget/tracker.go.
// Zero npm runtime dependencies — pure arithmetic (ADR-003).

import { Session, BudgetState } from '../types';

// formatCreditsDisplay renders raw credits with thousands separators and up to two
// decimals — parity with the Go side (e.g. "8,554.03", "656.54"). Credits are already
// credits (nanoAIU / 1e9), so there is no further scaling and no "B"/billions unit.
function formatCreditsDisplay(credits: number): string {
  return credits.toLocaleString(undefined, { maximumFractionDigits: 2 });
}

// Billing unit conversions — mirrors Go named constants exactly.
export const NANO_AIU_PER_CREDIT = 1_000_000_000;
export const DOLLARS_PER_CREDIT  = 0.01;
export const MONTHLY_ALLOWANCE   = 7_000;

// Sonnet token pricing (credits per million tokens).
export const SONNET_INPUT_RATE  = 300;
export const SONNET_OUTPUT_RATE = 1_500;

// fromNanoAIU converts raw nanoAIU billing units to credits.
export function fromNanoAIU(nanoAIU: number): number {
  return nanoAIU / NANO_AIU_PER_CREDIT;
}

// toDollars converts credits to US dollars.
export function toDollars(credits: number): number {
  return credits * DOLLARS_PER_CREDIT;
}

// calculate sums nanoAIU from sessions, converts to credits, and returns a BudgetState.
// allowance defaults to MONTHLY_ALLOWANCE when not provided or <= 0.
export function calculate(sessions: Session[], allowance?: number): BudgetState {
  const allowed = (allowance != null && allowance > 0) ? allowance : MONTHLY_ALLOWANCE;

  const totalNano = sessions.reduce((sum, s) => sum + s.totalNanoAIU, 0);
  const used = fromNanoAIU(totalNano);
  const usedPct = (used / allowed) * 100;
  const remainingCredits = allowed - used;

  return {
    usedCredits: used,
    allowedCredits: allowed,
    usedPct,
    remainingCredits,
    status: statusFor(usedPct),
  };
}

// estimateInstructionCostPerSession estimates the credit cost of always-loaded
// instruction tokens across a typical 50-turn session.
// Formula: (totalTokens * 50 * SONNET_INPUT_RATE) / 1_000_000
export function estimateInstructionCostPerSession(totalTokens: number): { credits: number; dollars: number } {
  const TURNS_PER_SESSION = 50;
  const credits = (totalTokens * TURNS_PER_SESSION * SONNET_INPUT_RATE) / 1_000_000;
  return { credits, dollars: toDollars(credits) };
}

// statusBarText returns the VS Code status bar label for a BudgetState.
// Uses VS Code ThemeIcon syntax for colour hints:
//   CRITICAL → $(circle-filled)  WARNING → $(warning)  OK → $(check)
export function statusBarText(state: BudgetState): string {
  const icon = state.status === 'CRITICAL'
    ? '$(circle-filled) '
    : state.status === 'WARNING'
      ? '$(warning) '
      : '$(check) ';
  return `${icon}💰 ${formatCreditsDisplay(state.usedCredits)} / ${formatCreditsDisplay(state.allowedCredits)}`;
}

// statusFor maps a usage percentage to a BudgetState status value.
function statusFor(pct: number): BudgetState['status'] {
  if (pct > 90) {
    return 'CRITICAL';
  }
  if (pct >= 60) {
    return 'WARNING';
  }
  return 'OK';
}
