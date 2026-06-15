// ui/statusBar.ts — VS Code status bar manager for the Copilot Token Budget extension.
// Shows live credit usage in the status bar; colour reflects budget health. The
// hover tooltip is a richer MarkdownString breakdown (today, month, burn, forecast,
// newest-active context %).

import * as vscode from 'vscode';
import { BudgetState, Session, billingTime } from '../types';
import { statusBarText, fromNanoAIU } from '../budget/tracker';
import { computeForecast } from '../forecast/model';
import { PricingConfig } from '../pricing/config';
import { contextWindowPct } from '../analytics/model';

export class StatusBarManager {
  private readonly item: vscode.StatusBarItem;

  constructor(context: vscode.ExtensionContext) {
    this.item = vscode.window.createStatusBarItem(
      vscode.StatusBarAlignment.Left,
      100
    );
    this.item.tooltip = 'Copilot Budget — click to open dashboard';
    this.item.command = 'copilotBudget.showDashboard';
    context.subscriptions.push(this.item);
  }

  // update refreshes the badge label/colour and rebuilds the hover tooltip. sessions
  // are this month's sessions; cfg supplies per-model context-window sizes.
  update(state: BudgetState, sessions: Session[], cfg: PricingConfig): void {
    this.item.text = statusBarText(state);
    this.item.tooltip = buildTooltip(state, sessions, cfg);

    switch (state.status) {
      case 'CRITICAL':
        this.item.backgroundColor = new vscode.ThemeColor('statusBarItem.errorBackground');
        break;
      case 'WARNING':
        this.item.backgroundColor = new vscode.ThemeColor('statusBarItem.warningBackground');
        break;
      default:
        this.item.backgroundColor = undefined;
        break;
    }
  }

  show(): void {
    this.item.show();
  }

  hide(): void {
    this.item.hide();
  }

  dispose(): void {
    this.item.dispose();
  }
}

// buildTooltip composes the Markdown hover content. It is rendered by VS Code with the
// trusted-markdown renderer; all values here are numeric or sourced from the local
// session read, so no untrusted HTML is interpolated.
function buildTooltip(state: BudgetState, sessions: Session[], cfg: PricingConfig): vscode.MarkdownString {
  const now = new Date();

  // Today's credits: sum of sessions billed today.
  let todayCredits = 0;
  for (const s of sessions) {
    const bt = billingTime(s);
    if (
      bt.getFullYear() === now.getFullYear() &&
      bt.getMonth() === now.getMonth() &&
      bt.getDate() === now.getDate()
    ) {
      todayCredits += fromNanoAIU(s.totalNanoAIU);
    }
  }

  const f = computeForecast(state.usedCredits, state.allowedCredits);

  // Newest active session's context-window fullness, if any.
  const active = sessions
    .filter(s => s.isActive)
    .sort((a, b) => b.startTime.getTime() - a.startTime.getTime());
  const newestActive = active.length > 0 ? active[0] : undefined;

  const md = new vscode.MarkdownString();
  md.supportThemeIcons = true;
  md.appendMarkdown(`**Copilot Token Budget**\n\n`);
  md.appendMarkdown(`| Metric | Value |\n| --- | --- |\n`);
  md.appendMarkdown(`| Today | ${todayCredits.toFixed(1)} cr |\n`);
  md.appendMarkdown(
    `| Month | ${state.usedCredits.toFixed(0)} / ${state.allowedCredits} cr (${state.usedPct.toFixed(1)}%) |\n`
  );
  md.appendMarkdown(`| Daily burn | ${f.dailyBurn.toFixed(1)} cr/day |\n`);
  const verdict = f.exceedsAllowance ? ' ⚠ over allowance' : '';
  md.appendMarkdown(`| Projected month-end | ${f.projectedMonthEndTotal.toFixed(0)} cr${verdict} |\n`);
  if (newestActive !== undefined) {
    const ctx = contextWindowPct(newestActive, cfg);
    const label = newestActive.projectName !== '' ? newestActive.projectName : newestActive.id.slice(0, 8);
    md.appendMarkdown(`| Context (${escapeCell(label)}) | ${ctx.toFixed(1)}% |\n`);
  }
  md.appendMarkdown(`\n_Click to open the dashboard._`);

  return md;
}

// escapeCell escapes the markdown table cell separator so a project name containing a
// pipe cannot break the tooltip table layout.
function escapeCell(s: string): string {
  return s.replace(/\|/g, '\\|');
}
