// extension.ts — activation entry point for the Copilot Token Budget VS Code extension.
// Wires all UI components, commands, refresh loop, and configuration listener.

// formatCreditsDisplay renders raw credits with thousands separators and up to two
// decimals — parity with the Go side (e.g. "8,554.03", "656.54"). Credits are already
// credits (nanoAIU / 1e9), so there is no further scaling and no "B"/billions unit.
function formatCreditsDisplay(credits: number): string {
  return credits.toLocaleString(undefined, { maximumFractionDigits: 2 });
}

import * as vscode from 'vscode';
import { StatusBarManager } from './ui/statusBar';
import { BudgetTreeProvider } from './ui/sessionTree';
import { DashboardPanel } from './ui/dashboardPanel';
import { readThisMonth } from './session/reader';
import { calculate, MONTHLY_ALLOWANCE } from './budget/tracker';
import { scanWorkspace } from './instructions/analyzer';
import { BudgetState, Session, InstructionFile } from './types';
import { fireAlertIfNeeded } from './alerts/teamsAlert';
import { loadPricing } from './pricing/config';
import { reportToJson, sessionsToCsv } from './export/report';

// Latest refresh snapshot — kept module-level so the export command can serialize the
// same data the UI is currently showing, without re-reading from disk.
let lastSessions: Session[] = [];
let lastBudgetState: BudgetState = {
  usedCredits: 0,
  allowedCredits: MONTHLY_ALLOWANCE,
  usedPct: 0,
  remainingCredits: MONTHLY_ALLOWANCE,
  status: 'OK',
};

// Refresh timer handle — module-level so deactivate() can clear it.
let refreshTimer: NodeJS.Timeout | undefined;

// Status bar and tree provider are module-level so deactivate() can dispose them.
let statusBar: StatusBarManager | undefined;

// Last status we notified the user about. Used to fire an alert only on an
// upward transition (OK→WARNING, OK/WARNING→CRITICAL) and to re-arm after the
// status drops back down — unlike a never-cleared "shown" set, which suppressed
// re-alerts for the rest of the session even after dropping and re-crossing.
let lastNotifiedStatus: BudgetState['status'] = 'OK';

// Reentrancy guard for runRefresh. The refresh is fired from the interval timer,
// the refresh command, the config-change listener, and activation; without this
// guard, overlapping async refreshes could let a slow stale read finish last and
// clobber fresher data. While one refresh is in flight, others are skipped.
let refreshing = false;

export function activate(context: vscode.ExtensionContext): void {
  // ── UI components ──────────────────────────────────────────────────────────
  statusBar = new StatusBarManager(context);
  const treeProvider = new BudgetTreeProvider();

  const treeView = vscode.window.createTreeView('copilotBudget.view', {
    treeDataProvider: treeProvider,
    showCollapseAll: true,
  });
  context.subscriptions.push(treeView);
  // Dispose the tree provider's change EventEmitter on deactivate.
  context.subscriptions.push(treeProvider);

  statusBar.show();

  // ── Commands ───────────────────────────────────────────────────────────────
  context.subscriptions.push(
    vscode.commands.registerCommand('copilotBudget.showDashboard', () => {
      DashboardPanel.createOrShow(context);
    }),

    vscode.commands.registerCommand('copilotBudget.refresh', () => {
      void runRefresh(context, treeProvider);
    }),

    vscode.commands.registerCommand('copilotBudget.openSettings', () => {
      void vscode.commands.executeCommand(
        'workbench.action.openSettings',
        'copilotBudget'
      );
    }),

    vscode.commands.registerCommand('copilotBudget.exportUsage', () => {
      void exportUsage();
    }),
  );

  // ── Configuration change listener ──────────────────────────────────────────
  context.subscriptions.push(
    vscode.workspace.onDidChangeConfiguration(event => {
      if (event.affectsConfiguration('copilotBudget')) {
        void runRefresh(context, treeProvider);
        resetTimer(context, treeProvider);
      }
    })
  );

  // ── Initial refresh + periodic timer ──────────────────────────────────────
  void runRefresh(context, treeProvider);
  resetTimer(context, treeProvider);

  // Wrap the timer in a Disposable so VS Code auto-clears it on any shutdown path
  // (extension disable, hard crash) — not just the normal deactivate() call.
  context.subscriptions.push(
    new vscode.Disposable(() => {
      if (refreshTimer !== undefined) {
        clearInterval(refreshTimer);
        refreshTimer = undefined;
      }
    })
  );
}

export function deactivate(): void {
  if (refreshTimer !== undefined) {
    clearInterval(refreshTimer);
    refreshTimer = undefined;
  }
  statusBar?.dispose();
}

// ── Export ───────────────────────────────────────────────────────────────────

// exportUsage shows a Save dialog and writes the latest snapshot to disk as JSON
// (the full report) or CSV (sessions), chosen by the saved file's extension. No
// network access — the bytes are serialized in-process and written with the VS Code
// workspace filesystem API.
async function exportUsage(): Promise<void> {
  const now = new Date();
  const stamp = `${now.getFullYear()}-${pad2(now.getMonth() + 1)}-${pad2(now.getDate())}`;
  const defaultUri = vscode.Uri.file(`copilot-usage-${stamp}.json`);

  const target = await vscode.window.showSaveDialog({
    defaultUri,
    saveLabel: 'Export Copilot Usage',
    filters: {
      JSON: ['json'],
      CSV: ['csv'],
    },
  });
  if (target === undefined) {
    return; // user cancelled
  }

  const lower = target.fsPath.toLowerCase();
  const isCsv = lower.endsWith('.csv');
  const content = isCsv
    ? sessionsToCsv(lastSessions)
    : reportToJson(lastSessions, lastBudgetState);

  try {
    await vscode.workspace.fs.writeFile(target, Buffer.from(content, 'utf8'));
    void vscode.window.showInformationMessage(
      `Copilot usage exported to ${target.fsPath} (${isCsv ? 'CSV' : 'JSON'}).`
    );
  } catch (err) {
    void vscode.window.showErrorMessage(`Copilot Budget: export failed — ${err}`);
  }
}

// pad2 zero-pads a one- or two-digit number for the default export filename.
function pad2(n: number): string {
  return n < 10 ? `0${n}` : String(n);
}

// ── Refresh logic ────────────────────────────────────────────────────────────

async function runRefresh(
  context: vscode.ExtensionContext,
  treeProvider: BudgetTreeProvider
): Promise<void> {
  // Skip if a refresh is already running — prevents overlapping reads from
  // clobbering fresh data with stale results.
  if (refreshing) {
    return;
  }
  refreshing = true;
  try {
    await doRefresh(context, treeProvider);
  } finally {
    refreshing = false;
  }
}

async function doRefresh(
  context: vscode.ExtensionContext,
  treeProvider: BudgetTreeProvider
): Promise<void> {
  const cfg = vscode.workspace.getConfiguration('copilotBudget');
  const workspaceSetting: string = cfg.get<string>('workspacePath') ?? '';

  // Allowance precedence (matches the Go CLI, which honors pricing.AllowanceCredits):
  //   1. An EXPLICITLY set `copilotBudget.monthlyAllowance` (user/workspace value) wins.
  //   2. Otherwise `loadPricing().allowanceCredits` (from pricing.json or bundled default).
  // We inspect the setting rather than reading its effective value, because get()
  // returns the package.json default (7000) even when the user never set it — which
  // would always shadow the pricing config and make a pricing.json allowance a no-op.
  const pricing = loadPricing();
  const allowance = resolveAllowance(cfg, pricing.allowanceCredits);

  const workspacePath =
    workspaceSetting !== ''
      ? workspaceSetting
      : vscode.workspace.workspaceFolders?.[0]?.uri.fsPath ?? '';

  let sessions: Session[] = [];
  let instrFiles: InstructionFile[] = [];

  try {
    sessions = await readThisMonth();
  } catch (err) {
    console.error(`copilot-budget: session read failed: ${err}`);
  }

  if (workspacePath !== '') {
    try {
      instrFiles = await scanWorkspace(workspacePath);
    } catch (err) {
      console.error(`copilot-budget: workspace scan failed: ${err}`);
    }
  }

  const budgetState = calculate(sessions, allowance);

  // Cache the snapshot for the export command.
  lastSessions = sessions;
  lastBudgetState = budgetState;

  // Update all UI surfaces. The status bar tooltip needs the sessions + pricing
  // config to compute today's spend and the newest-active context window %.
  // (pricing was already loaded above for allowance resolution.)
  statusBar?.update(budgetState, sessions, pricing);
  treeProvider.refresh(sessions, budgetState, instrFiles);

  // Update dashboard panel only if it is currently open.
  const panel = DashboardPanel.getInstance();
  if (panel) {
    panel.update(sessions, budgetState, instrFiles);
  }

  // Threshold alerts — shown at most once per threshold per VS Code session.
  maybeShowAlert(budgetState);

  // Teams alert — fire-and-forget; never blocks the refresh loop.
  fireAlertIfNeeded(context).catch(err =>
    console.error('Teams alert error:', err)
  );
}

// resolveAllowance implements the allowance precedence: an explicitly set
// `copilotBudget.monthlyAllowance` (a global or workspace value, as opposed to the
// package.json default) overrides the pricing config; otherwise the pricing config's
// allowanceCredits is used. This keeps the extension consistent with the Go CLI,
// which honors pricing.AllowanceCredits.
function resolveAllowance(
  cfg: vscode.WorkspaceConfiguration,
  pricingAllowance: number
): number {
  const info = cfg.inspect<number>('monthlyAllowance');
  const explicit =
    info?.workspaceFolderValue ??
    info?.workspaceValue ??
    info?.globalValue;
  if (typeof explicit === 'number' && Number.isFinite(explicit) && explicit > 0) {
    return explicit;
  }
  return pricingAllowance > 0 ? pricingAllowance : MONTHLY_ALLOWANCE;
}

// resetTimer clears any existing interval and starts a fresh one.
// Clear + reset prevents timer stacking when settings change.
function resetTimer(
  context: vscode.ExtensionContext,
  treeProvider: BudgetTreeProvider
): void {
  if (refreshTimer !== undefined) {
    clearInterval(refreshTimer);
    refreshTimer = undefined;
  }

  const cfg = vscode.workspace.getConfiguration('copilotBudget');
  const intervalSec: number = cfg.get<number>('refreshIntervalSec') ?? 30;
  const intervalMs = Math.max(intervalSec, 10) * 1000; // floor at 10s for safety

  refreshTimer = setInterval(() => {
    void runRefresh(context, treeProvider);
  }, intervalMs);
}

// maybeShowAlert fires a VS Code notification on an UPWARD status transition:
//   OK → WARNING, and {OK, WARNING} → CRITICAL.
// It tracks the last-notified status rather than a never-cleared "shown" set, so
// after usage drops (e.g. into a new month) the alert re-arms and fires again the
// next time the threshold is re-crossed. No alert fires on equal or downward moves.
function maybeShowAlert(state: BudgetState): void {
  const prev = lastNotifiedStatus;
  const key = state.status; // 'OK' | 'WARNING' | 'CRITICAL'

  // Always record the current status so a later re-crossing is detected correctly,
  // including downward moves (CRITICAL → OK re-arms both WARNING and CRITICAL).
  lastNotifiedStatus = key;

  if (key === 'OK') {
    return;
  }
  // Fire only when the status actually escalated above the previous level.
  const rank: Record<BudgetState['status'], number> = { OK: 0, WARNING: 1, CRITICAL: 2 };
  if (rank[key] <= rank[prev]) {
    return;
  }

  const msg = key === 'CRITICAL'
    ? `⚠️ Copilot Budget CRITICAL: ${formatCreditsDisplay(state.usedCredits)} / ${formatCreditsDisplay(state.allowedCredits)} used (${state.usedPct.toFixed(1)}%). AT&T monthly allowance exceeded.`
    : `Copilot Budget WARNING: ${formatCreditsDisplay(state.usedCredits)} / ${formatCreditsDisplay(state.allowedCredits)} used (${state.usedPct.toFixed(1)}%).`;

  if (key === 'CRITICAL') {
    void vscode.window.showWarningMessage(msg, 'Open Dashboard').then(choice => {
      if (choice === 'Open Dashboard') {
        void vscode.commands.executeCommand('copilotBudget.showDashboard');
      }
    });
  } else {
    void vscode.window.showInformationMessage(msg);
  }
}

