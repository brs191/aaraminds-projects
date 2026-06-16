// ui/sessionTree.ts — VS Code tree view provider for the Budget Overview sidebar.
// Implements TreeDataProvider with 3 root nodes: Budget, Active Sessions, Instruction Files.

// formatCreditsDisplay renders raw credits with thousands separators and up to two
// decimals — parity with the Go side (e.g. "8,554.03", "656.54"). Credits are already
// credits (nanoAIU / 1e9), so there is no further scaling and no "B"/billions unit.
function formatCreditsDisplay(credits: number): string {
  return credits.toLocaleString(undefined, { maximumFractionDigits: 2 });
}

import * as vscode from "vscode";
import * as path from "path";
import { Session, BudgetState, InstructionFile } from "../types";
import { buildOptimizationSummary, severity } from "../instructions/analyzer";
import { estimateInstructionCostPerSession } from "../budget/tracker";
import { toDollars } from "../budget/tracker";
import { computeForecast } from "../forecast/model";
import { dailySeries, topSessions, topModels } from "../analytics/model";

export class BudgetTreeProvider implements vscode.TreeDataProvider<vscode.TreeItem> {
  private readonly _onDidChangeTreeData = new vscode.EventEmitter<
    vscode.TreeItem | undefined
  >();
  readonly onDidChangeTreeData = this._onDidChangeTreeData.event;

  private sessions: Session[] = [];
  private budgetState: BudgetState = {
    usedCredits: 0,
    allowedCredits: 7000,
    usedPct: 0,
    remainingCredits: 7000,
    status: "OK",
  };
  private instructionFiles: InstructionFile[] = [];

  refresh(
    sessions: Session[],
    budgetState: BudgetState,
    instructionFiles: InstructionFile[],
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
      case "budget":
        return this.budgetChildren();
      case "forecast":
        return this.forecastChildren();
      case "trend":
        return this.trendChildren();
      case "consumers":
        return this.consumerChildren();
      case "active":
        return this.activeSessionChildren();
      case "instructions":
        return this.instructionChildren();
      default:
        return [];
    }
  }

  private rootNodes(): vscode.TreeItem[] {
    const statusIcon =
      this.budgetState.status === "CRITICAL"
        ? "$(circle-filled)"
        : this.budgetState.status === "WARNING"
          ? "$(warning)"
          : "$(check)";

    return [
      node(
        "budget",
        `${statusIcon} Budget`,
        vscode.TreeItemCollapsibleState.Expanded,
      ),
      node(
        "forecast",
        "$(graph) Forecast",
        vscode.TreeItemCollapsibleState.Collapsed,
      ),
      node(
        "trend",
        "$(graph-line) Usage Trend",
        vscode.TreeItemCollapsibleState.Collapsed,
      ),
      node(
        "consumers",
        "$(list-ordered) Top Consumers",
        vscode.TreeItemCollapsibleState.Collapsed,
      ),
      node(
        "active",
        "$(pulse) Active Sessions",
        vscode.TreeItemCollapsibleState.Collapsed,
      ),
      node(
        "instructions",
        "$(file-text) Instruction Files",
        vscode.TreeItemCollapsibleState.Collapsed,
      ),
    ];
  }

  // trendChildren shows the last 7 daily buckets (most recent first) as leaves.
  private trendChildren(): vscode.TreeItem[] {
    const daily = dailySeries(this.sessions);
    if (daily.length === 0) {
      return [leaf("No usage data")];
    }
    const last7 = daily.slice(Math.max(0, daily.length - 7)).reverse();
    return last7.map((b) =>
      leaf(
        `${b.key}   ${formatCreditsDisplay(b.credits)}   (${b.sessions} sess)`,
      ),
    );
  }

  // consumerChildren shows the top 3 sessions and top 3 models by credits.
  private consumerChildren(): vscode.TreeItem[] {
    const sessions = topSessions(this.sessions, 3);
    const models = topModels(this.sessions, 3);
    if (sessions.length === 0 && models.length === 0) {
      return [leaf("No usage data")];
    }
    const items: vscode.TreeItem[] = [];
    items.push(leaf("— Sessions —"));
    if (sessions.length === 0) {
      items.push(leaf("  (none)"));
    } else {
      for (const c of sessions) {
        items.push(leaf(`  ${c.name}   ${formatCreditsDisplay(c.credits)}`));
      }
    }
    items.push(leaf("— Models —"));
    if (models.length === 0) {
      items.push(leaf("  (none)"));
    } else {
      for (const c of models) {
        items.push(leaf(`  ${c.model}   ${formatCreditsDisplay(c.credits)}`));
      }
    }
    return items;
  }

  private forecastChildren(): vscode.TreeItem[] {
    const f = computeForecast(
      this.budgetState.usedCredits,
      this.budgetState.allowedCredits,
    );
    const verdict = f.exceedsAllowance
      ? `(exceeds ${formatCreditsDisplay(this.budgetState.allowedCredits)} allowance)`
      : `(within ${formatCreditsDisplay(this.budgetState.allowedCredits)} allowance)`;
    return [
      leaf(`Burn rate:       ${formatCreditsDisplay(f.dailyBurn)}/day`),
      leaf(
        `Projected total: ${formatCreditsDisplay(f.projectedMonthEndTotal)} ${verdict}`,
      ),
    ];
  }

  private budgetChildren(): vscode.TreeItem[] {
    const s = this.budgetState;
    return [
      leaf(`Used:      ${formatCreditsDisplay(s.usedCredits)}`),
      leaf(`Allowed:   ${formatCreditsDisplay(s.allowedCredits)}`),
      leaf(`Usage:     ${s.usedPct.toFixed(1)}%`),
      leaf(`Status:    ${s.status}`),
      leaf(`Remaining: ${formatCreditsDisplay(s.remainingCredits)}`),
      leaf(`Cost:      $${toDollars(s.usedCredits).toFixed(2)}`),
    ];
  }

  private activeSessionChildren(): vscode.TreeItem[] {
    const active = this.sessions.filter((s) => s.isActive);
    if (active.length === 0) {
      return [leaf("No active sessions")];
    }
    return active.map((s) => {
      const credits = s.totalNanoAIU / 1_000_000_000;
      return leaf(
        `${s.projectName}  ${formatCreditsDisplay(credits)}  [${s.primaryModel}]`,
      );
    });
  }

  private instructionChildren(): vscode.TreeItem[] {
    if (this.instructionFiles.length === 0) {
      return [leaf("No instruction files found")];
    }

    const plan = buildOptimizationSummary(this.instructionFiles);
    const current = estimateInstructionCostPerSession(plan.alwaysLoadedTokens);
    const target = estimateInstructionCostPerSession(plan.targetTokens);
    const savings = Math.max(0, current.credits - target.credits);

    const nodes: vscode.TreeItem[] = [
      leaf(`Always-loaded: ~${plan.alwaysLoadedTokens} tokens`),
      leaf(`Target: ~${plan.targetTokens} tokens`),
      leaf(`Potential savings: ${formatCreditsDisplay(savings)} / session`),
    ];

    const top = plan.opportunities.slice(0, 3);
    if (top.length > 0) {
      nodes.push(leaf("— Top optimization candidates —"));
      for (const o of top) {
        const name = path.basename(o.path);
        nodes.push(leaf(`$(arrow-down) ${name}  -${o.reducibleTokens} tokens`));
      }
    }

    for (const f of this.instructionFiles) {
      const sev = severity(f.estimatedTokens);
      const icon =
        sev === "high"
          ? "$(warning)"
          : sev === "medium"
            ? "$(info)"
            : "$(check)";
      const name = path.basename(f.path);
      nodes.push(leaf(`${icon} ${name}  ~${f.estimatedTokens} tokens`));
    }
    return nodes;
  }
}

// LabeledItem extends TreeItem with a stable nodeId for getChildren routing.
interface LabeledItem extends vscode.TreeItem {
  nodeId: string;
}

function node(
  nodeId: string,
  label: string,
  collapsibleState: vscode.TreeItemCollapsibleState,
): LabeledItem {
  const item = new vscode.TreeItem(label, collapsibleState) as LabeledItem;
  item.nodeId = nodeId;
  return item;
}

function leaf(label: string): vscode.TreeItem {
  return new vscode.TreeItem(label, vscode.TreeItemCollapsibleState.None);
}
