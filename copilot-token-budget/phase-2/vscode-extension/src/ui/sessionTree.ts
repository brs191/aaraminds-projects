// ui/sessionTree.ts — VS Code tree view provider for the Budget Overview sidebar.
// Implements TreeDataProvider with 3 root nodes: Budget, Active Sessions, Instruction Files.

import * as vscode from 'vscode';
import * as path from 'path';
import { Session, BudgetState, InstructionFile } from '../types';
import { severity } from '../instructions/analyzer';
import { toDollars } from '../budget/tracker';
import { computeForecast } from '../forecast/model';
import { dailySeries, topSessions, topModels } from '../analytics/model';

export class BudgetTreeProvider implements vscode.TreeDataProvider<vscode.TreeItem> {
  private readonly _onDidChangeTreeData =
    new vscode.EventEmitter<vscode.TreeItem | undefined>();
  readonly onDidChangeTreeData = this._onDidChangeTreeData.event;

  private sessions: Session[] = [];
  private budgetState: BudgetState = {
    usedCredits: 0,
    allowedCredits: 7000,
    usedPct: 0,
    remainingCredits: 7000,
    status: 'OK',
  };
  private instructionFiles: InstructionFile[] = [];

  refresh(
    sessions: Session[],
    budgetState: BudgetState,
    instructionFiles: InstructionFile[]
  ): void {
    this.sessions = sessions;
    this.budgetState = budgetState;
    this.instructionFiles = instructionFiles;
    this._onDidChangeTreeData.fire(undefined);
  }

  getTreeItem(element: vscode.TreeItem): vscode.TreeItem {
    return element;
  }

  getChildren(element?: vscode.TreeItem): vscode.TreeItem[] {
    if (element === undefined) {
      return this.rootNodes();
    }
    const id = (element as LabeledItem).nodeId;
    switch (id) {
      case 'budget':      return this.budgetChildren();
      case 'forecast':    return this.forecastChildren();
      case 'trend':       return this.trendChildren();
      case 'consumers':   return this.consumerChildren();
      case 'active':      return this.activeSessionChildren();
      case 'instructions':return this.instructionChildren();
      default:            return [];
    }
  }

  private rootNodes(): vscode.TreeItem[] {
    const statusIcon = this.budgetState.status === 'CRITICAL'
      ? '$(circle-filled)'
      : this.budgetState.status === 'WARNING'
        ? '$(warning)'
        : '$(check)';

    return [
      node('budget',       `${statusIcon} Budget`,         vscode.TreeItemCollapsibleState.Expanded),
      node('forecast',     '$(graph) Forecast',            vscode.TreeItemCollapsibleState.Collapsed),
      node('trend',        '$(graph-line) Usage Trend',    vscode.TreeItemCollapsibleState.Collapsed),
      node('consumers',    '$(list-ordered) Top Consumers', vscode.TreeItemCollapsibleState.Collapsed),
      node('active',       '$(pulse) Active Sessions',     vscode.TreeItemCollapsibleState.Collapsed),
      node('instructions', '$(file-text) Instruction Files', vscode.TreeItemCollapsibleState.Collapsed),
    ];
  }

  // trendChildren shows the last 7 daily buckets (most recent first) as leaves.
  private trendChildren(): vscode.TreeItem[] {
    const daily = dailySeries(this.sessions);
    if (daily.length === 0) {
      return [leaf('No usage data')];
    }
    const last7 = daily.slice(Math.max(0, daily.length - 7)).reverse();
    return last7.map(b =>
      leaf(`${b.key}   ${b.credits.toFixed(2)} cr   (${b.sessions} sess)`)
    );
  }

  // consumerChildren shows the top 3 sessions and top 3 models by credits.
  private consumerChildren(): vscode.TreeItem[] {
    const sessions = topSessions(this.sessions, 3);
    const models = topModels(this.sessions, 3);
    if (sessions.length === 0 && models.length === 0) {
      return [leaf('No usage data')];
    }
    const items: vscode.TreeItem[] = [];
    items.push(leaf('— Sessions —'));
    if (sessions.length === 0) {
      items.push(leaf('  (none)'));
    } else {
      for (const c of sessions) {
        items.push(leaf(`  ${c.name}   ${c.credits.toFixed(2)} cr`));
      }
    }
    items.push(leaf('— Models —'));
    if (models.length === 0) {
      items.push(leaf('  (none)'));
    } else {
      for (const c of models) {
        items.push(leaf(`  ${c.model}   ${c.credits.toFixed(2)} cr`));
      }
    }
    return items;
  }

  private forecastChildren(): vscode.TreeItem[] {
    const f = computeForecast(
      this.budgetState.usedCredits,
      this.budgetState.allowedCredits
    );
    const verdict = f.exceedsAllowance
      ? `(exceeds ${this.budgetState.allowedCredits} cr allowance)`
      : `(within ${this.budgetState.allowedCredits} cr allowance)`;
    return [
      leaf(`Burn rate:       ${f.dailyBurn.toFixed(1)} cr/day`),
      leaf(`Projected total: ${f.projectedMonthEndTotal.toFixed(0)} cr ${verdict}`),
    ];
  }

  private budgetChildren(): vscode.TreeItem[] {
    const s = this.budgetState;
    return [
      leaf(`Used:      ${s.usedCredits.toFixed(2)} cr`),
      leaf(`Allowed:   ${s.allowedCredits} cr`),
      leaf(`Usage:     ${s.usedPct.toFixed(1)}%`),
      leaf(`Status:    ${s.status}`),
      leaf(`Remaining: ${s.remainingCredits.toFixed(2)} cr`),
      leaf(`Cost:      $${toDollars(s.usedCredits).toFixed(2)}`),
    ];
  }

  private activeSessionChildren(): vscode.TreeItem[] {
    const active = this.sessions.filter(s => s.isActive);
    if (active.length === 0) {
      return [leaf('No active sessions')];
    }
    return active.map(s => {
      const credits = (s.totalNanoAIU / 1_000_000_000).toFixed(2);
      return leaf(`${s.projectName}  ${credits} cr  [${s.primaryModel}]`);
    });
  }

  private instructionChildren(): vscode.TreeItem[] {
    if (this.instructionFiles.length === 0) {
      return [leaf('No instruction files found')];
    }
    return this.instructionFiles.map(f => {
      const sev = severity(f.estimatedTokens);
      const icon = sev === 'high' ? '$(warning)' : sev === 'medium' ? '$(info)' : '$(check)';
      const name = path.basename(f.path);
      return leaf(`${icon} ${name}  ~${f.estimatedTokens} tokens`);
    });
  }
}

// LabeledItem extends TreeItem with a stable nodeId for getChildren routing.
interface LabeledItem extends vscode.TreeItem {
  nodeId: string;
}

function node(
  nodeId: string,
  label: string,
  collapsibleState: vscode.TreeItemCollapsibleState
): LabeledItem {
  const item = new vscode.TreeItem(label, collapsibleState) as LabeledItem;
  item.nodeId = nodeId;
  return item;
}

function leaf(label: string): vscode.TreeItem {
  return new vscode.TreeItem(label, vscode.TreeItemCollapsibleState.None);
}
