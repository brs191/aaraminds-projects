// ui/dashboardPanel.ts — singleton VS Code webview panel for the budget dashboard.
// Uses VS Code CSS variables so it respects the user's light/dark theme automatically.

// formatCreditsDisplay renders raw credits with thousands separators and no decimals.
// Credits are already credits (nanoAIU / 1e9), so there is no further scaling and no
// "B"/billions unit.
function formatCreditsDisplay(credits: number): string {
  return credits.toLocaleString(undefined, { maximumFractionDigits: 0 });
}

import * as vscode from "vscode";
import { Session, BudgetState, InstructionFile, SessionSource } from "../types";
import { computeForecast } from "../forecast/model";
import { totalInputTokens, totalOutputTokens } from "../types";
import { loadPricing, rateFor } from "../pricing/config";
import { estimateInstructionCostPerSession } from "../budget/tracker";
import { buildOptimizationSummary } from "../instructions/analyzer";
import {
  dailySeries,
  anomalousDays,
  topSessions,
  topModels,
  topProjects,
  contextWindowPct,
  Consumer,
} from "../analytics/model";
import { latestLiveBillingSnapshot, liveBillingLabel } from "../livebilling/labels";
import { LiveBillingSnapshot } from "../types";

// Forecast figures surfaced in the dashboard's Forecast block.
interface SerializedForecast {
  dailyBurn: number; // cr/day
  projectedMonthEndTotal: number; // cr
  exceedsAllowance: boolean;
}

// One daily usage-trend point. anomalous flags days above mean + 2*stddev.
interface TrendPoint {
  key: string; // "YYYY-MM-DD"
  credits: number;
  anomalous: boolean;
}

// One ranked consumer row for the Top Consumers tables.
interface SerializedConsumer {
  name: string;
  credits: number;
  inputTokensK: number;
  outputTokensK: number;
  model: string;
}

// One per-model prompt-cache tally surfaced alongside the Top Models table. Mirrors
// the Go renderModelCacheReads output: cache-read tokens per model, with a flag when
// cache reads dominate raw input for that model.
interface SerializedModelCache {
  model: string;
  cacheReadTokens: number;
  cacheWriteTokens: number;
  reasoningTokens: number;
  dominatesInput: boolean; // cacheReadTokens > inputTokens
}

interface SerializedModelSummary {
  model: string;
  credits: number;
  inputTokensK: number;
  outputTokensK: number;
}

interface SerializedOptimizationOpportunity {
  name: string;
  reducibleTokens: number;
  currentTokens: number;
  targetTokens: number;
  recommendation: string;
}

interface SerializedOptimizationSummary {
  alwaysLoadedTokens: number;
  targetTokens: number;
  reducibleTokens: number;
  currentCreditsPerSession: number;
  targetCreditsPerSession: number;
  potentialCreditsPerSession: number;
  opportunities: SerializedOptimizationOpportunity[];
}

// Message shape sent from the extension to the webview.
interface DashboardMessage {
  sessions: SerializedSession[];
  budgetState: BudgetState;
  orgBillingSnapshot?: LiveBillingSnapshot;
  instructionFiles: InstructionFile[];
  forecast: SerializedForecast;
  trend: TrendPoint[]; // last 14 daily buckets, anomalies flagged
  topSessions: SerializedConsumer[];
  topModels: SerializedConsumer[];
  topProjects: SerializedConsumer[];
  modelCache: SerializedModelCache[]; // per-model prompt-cache tallies (Top Models)
  modelSummary: SerializedModelSummary[]; // all-model consumption summary
  premiumRequests: number; // total premium requests across this month's settled sessions
  cliSessionCount: number; // number of CLI sessions discovered
  cliTotal: number; // total credits from CLI sessions — the real, tracked total
  ideSessionCount: number; // number of IDE sessions discovered
  ideTotal: number; // IDE credits from standard VS Code user-data transcripts; pending only if absent
  ideTracked: boolean; // true once a live IDE collector contributes sessions
  optimization: SerializedOptimizationSummary;
}

interface DashboardInboundMessage {
  type: "setAllowance";
  allowanceCredits: number;
}

// Sessions must be serialized (Dates → ISO strings) before posting to the webview.
interface SerializedSession {
  id: string;
  projectName: string;
  primaryModel: string;
  isActive: boolean;
  totalCredits: number;
  inputTokensK: number;
  outputTokensK: number;
  systemTokens: number;
  contextPct: number; // context-window fullness for the primary model
  startTime: string;
  source: SessionSource; // NEW: identify CLI vs IDE origin
}

export class DashboardPanel {
  private static instance: DashboardPanel | undefined;
  private readonly panel: vscode.WebviewPanel;
  private readonly context: vscode.ExtensionContext;

  private constructor(context: vscode.ExtensionContext) {
    this.context = context;
    this.panel = vscode.window.createWebviewPanel(
      "copilotBudgetDashboard",
      "Copilot Budget Dashboard",
      vscode.ViewColumn.One,
      {
        enableScripts: true,
        retainContextWhenHidden: true,
        // Defense-in-depth: the webview loads no local resources, so lock the roots
        // to the extension dir (paired with the CSP meta tag in buildHtml).
        localResourceRoots: [context.extensionUri],
      },
    );

    this.panel.webview.html = buildHtml(this.panel.webview);
    this.panel.webview.onDidReceiveMessage(
      async (raw: unknown) => {
        if (
          !raw ||
          typeof raw !== "object" ||
          (raw as { type?: unknown }).type !== "setAllowance"
        ) {
          return;
        }
        const msg = raw as DashboardInboundMessage;
        const next = Number(msg.allowanceCredits);
        if (!Number.isFinite(next) || next <= 0) {
          return;
        }
        const target =
          vscode.workspace.workspaceFolders &&
          vscode.workspace.workspaceFolders.length > 0
            ? vscode.ConfigurationTarget.Workspace
            : vscode.ConfigurationTarget.Global;
        await vscode
          .workspace
          .getConfiguration("copilotBudget")
          .update("monthlyAllowance", next, target);
      },
      null,
      context.subscriptions,
    );

    this.panel.onDidDispose(
      () => {
        DashboardPanel.instance = undefined;
      },
      null,
      context.subscriptions,
    );
  }

  static createOrShow(context: vscode.ExtensionContext): DashboardPanel {
    if (DashboardPanel.instance) {
      DashboardPanel.instance.panel.reveal(vscode.ViewColumn.One);
      return DashboardPanel.instance;
    }
    DashboardPanel.instance = new DashboardPanel(context);
    return DashboardPanel.instance;
  }

  // getInstance returns the open panel or undefined — used by the refresh loop
  // to update the dashboard only when it is currently visible.
  static getInstance(): DashboardPanel | undefined {
    return DashboardPanel.instance;
  }

  update(
    sessions: Session[],
    budgetState: BudgetState,
    instructionFiles: InstructionFile[],
  ): void {
    const f = computeForecast(
      budgetState.usedCredits,
      budgetState.allowedCredits,
    );
    const cfg = loadPricing();

    // Compute source breakdown and premium-request total. CLI gets a separate session
    // count + credit total; IDE gets the same treatment when discovered.
    let cliSessionCount = 0;
    let cliTotal = 0;
    let ideSessionCount = 0;
    let ideTotal = 0;
    let premiumRequests = 0;
    for (const s of sessions) {
      const credits = s.totalNanoAIU / 1_000_000_000;
      if (s.source === "copilot-cli") {
        cliSessionCount += 1;
        cliTotal += credits;
      } else if (s.source === "copilot-ide") {
        ideSessionCount += 1;
        ideTotal += credits;
      }
      premiumRequests += s.totalPremiumRequests;
    }
    // ideTracked flips on automatically when the IDE collector contributes sessions.
    const ideTracked = sessions.some((s) => s.source === "copilot-ide");

    // Per-model prompt-cache tallies for the Top Models area — mirrors Go
    // renderModelCacheReads. Aggregate cache/reasoning tokens and raw input per model.
    const cacheByModel = new Map<
      string,
      {
        cacheRead: number;
        cacheWrite: number;
        reasoning: number;
        input: number;
      }
    >();
    const summaryByModel = new Map<
      string,
      {
        credits: number;
        inputTokens: number;
        outputTokens: number;
      }
    >();
    for (const s of sessions) {
      for (const m of s.modelMetrics) {
        let t = cacheByModel.get(m.model);
        if (t === undefined) {
          t = { cacheRead: 0, cacheWrite: 0, reasoning: 0, input: 0 };
          cacheByModel.set(m.model, t);
        }
        t.cacheRead += m.cacheReadTokens;
        t.cacheWrite += m.cacheWriteTokens;
        t.reasoning += m.reasoningTokens;
        t.input += m.inputTokens;

        let sum = summaryByModel.get(m.model);
        if (sum === undefined) {
          sum = { credits: 0, inputTokens: 0, outputTokens: 0 };
          summaryByModel.set(m.model, sum);
        }
        sum.credits += m.nanoAIU / 1_000_000_000;
        sum.inputTokens += m.inputTokens;
        sum.outputTokens += m.outputTokens;
      }
    }

    // Usage trend: full daily series, anomalies flagged, then the last 14 days.
    const daily = dailySeries(sessions);
    const anomalousKeys = new Set(anomalousDays(daily).map((b) => b.key));
    const trend: TrendPoint[] = daily
      .slice(Math.max(0, daily.length - 14))
      .map((b) => ({
        key: b.key,
        credits: b.credits,
        anomalous: anomalousKeys.has(b.key),
      }));

    // Mirror Go: only surface models that actually have cache reads, in Top-Models rank
    // order, so the section stays compact (empty when no model has cache reads).
    const rankedModels = topModels(sessions, 5).map(serializeConsumer);
    const modelCache: SerializedModelCache[] = [];
    for (const c of rankedModels) {
      const t = cacheByModel.get(c.model);
      if (t === undefined || t.cacheRead === 0) {
        continue;
      }
      modelCache.push({
        model: c.model,
        cacheReadTokens: t.cacheRead,
        cacheWriteTokens: t.cacheWrite,
        reasoningTokens: t.reasoning,
        dominatesInput: t.cacheRead > t.input,
      });
    }
    const modelSummary: SerializedModelSummary[] = Array
      .from(summaryByModel.entries())
      .map(([model, s]) => ({
        model,
        credits: s.credits,
        inputTokensK: Math.floor(s.inputTokens / 1000),
        outputTokensK: Math.floor(s.outputTokens / 1000),
      }))
      .sort((a, b) => b.credits - a.credits);

    const plan = buildOptimizationSummary(instructionFiles);
    const current = estimateInstructionCostPerSession(plan.alwaysLoadedTokens);
    const target = estimateInstructionCostPerSession(plan.targetTokens);
    const orgBillingSnapshot = latestLiveBillingSnapshot(sessions);

    const message: DashboardMessage = {
      sessions: sessions.map((s) => serialize(s, cfg)),
      budgetState,
      orgBillingSnapshot,
      instructionFiles,
      forecast: {
        dailyBurn: f.dailyBurn,
        projectedMonthEndTotal: f.projectedMonthEndTotal,
        exceedsAllowance: f.exceedsAllowance,
      },
      trend,
      topSessions: topSessions(sessions, 5).map(serializeConsumer),
      topModels: rankedModels,
      topProjects: topProjects(sessions, 5).map(serializeConsumer),
      modelCache,
      modelSummary,
      premiumRequests,
      cliSessionCount,
      cliTotal, // real tracked total
      ideSessionCount,
      ideTotal, // IDE credits total for discovered IDE sessions
      ideTracked,
      optimization: {
        alwaysLoadedTokens: plan.alwaysLoadedTokens,
        targetTokens: plan.targetTokens,
        reducibleTokens: plan.reducibleTokens,
        currentCreditsPerSession: current.credits,
        targetCreditsPerSession: target.credits,
        potentialCreditsPerSession: Math.max(
          0,
          current.credits - target.credits,
        ),
        opportunities: plan.opportunities.slice(0, 5).map((o) => ({
          name: o.path.split(/[\\/]/).pop() || o.path,
          reducibleTokens: o.reducibleTokens,
          currentTokens: o.currentTokens,
          targetTokens: o.targetTokens,
          recommendation: o.recommendation,
        })),
      },
    };
    console.log(`[DashboardPanel.update] Posting message with ${message.sessions.length} sessions, budget=${message.budgetState.usedCredits}/${message.budgetState.allowedCredits}`);
    void this.panel.webview.postMessage(message);
  }

  dispose(): void {
    this.panel.dispose();
  }
}

function serialize(
  s: Session,
  cfg: ReturnType<typeof loadPricing>,
): SerializedSession {
  return {
    id: s.id,
    projectName: s.projectName || s.workspaceDir || s.id.slice(0, 8),
    primaryModel: s.primaryModel,
    isActive: s.isActive,
    totalCredits: s.totalNanoAIU / 1_000_000_000,
    inputTokensK: Math.floor(totalInputTokens(s) / 1000),
    outputTokensK: Math.floor(totalOutputTokens(s) / 1000),
    systemTokens: s.tokens.systemTokens,
    contextPct: contextWindowPct(s, cfg),
    startTime: s.startTime.toISOString(),
    source: s.source, // NEW: identify CLI vs IDE origin
  };
}

// serializeConsumer projects an analytics Consumer to the webview wire shape,
// pre-rounding token totals to thousands for compact display.
function serializeConsumer(c: Consumer): SerializedConsumer {
  return {
    name: c.name,
    credits: c.credits,
    inputTokensK: Math.floor(c.inputTokens / 1000),
    outputTokensK: Math.floor(c.outputTokens / 1000),
    model: c.model,
  };
}

// nonce returns a random base64-ish token for the CSP script-src allowlist.
function makeNonce(): string {
  const chars =
    "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
  let out = "";
  for (let i = 0; i < 32; i++) {
    out += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return out;
}

// buildHtml returns the full dashboard HTML.
// Uses VS Code CSS variables for full light/dark theme support. A strict
// Content-Security-Policy (default-src 'none'; inline styles allowed; scripts
// only via the per-load nonce) is defense-in-depth: even if a future attribute
// interpolation slipped past esc(), no untrusted script could execute.
function buildHtml(webview: vscode.Webview): string {
  const nonce = makeNonce();
  const csp =
    `default-src 'none'; ` +
    `style-src 'unsafe-inline'; ` +
    `script-src 'nonce-${nonce}'; ` +
    `img-src ${webview.cspSource};`;
  return /* html */ `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8" />
<meta http-equiv="Content-Security-Policy" content="${csp}" />
<meta name="viewport" content="width=device-width, initial-scale=1.0" />
<title>Copilot Budget Dashboard</title>
<style>
  :root {
    --tn-bg: #1a1b26;
    --tn-bg-alt: #1f2335;
    --tn-surface: #24283b;
    --tn-border: #3b4261;
    --tn-fg: #c0caf5;
    --tn-muted: #a9b1d6;
    --tn-accent: #7aa2f7;
    --tn-green: #9ece6a;
    --tn-yellow: #e0af68;
    --tn-red: #f7768e;
    --tn-cyan: #7dcfff;
  }
  * { box-sizing: border-box; margin: 0; padding: 0; }

  body {
    font-family: var(--vscode-font-family, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif);
    font-size: var(--vscode-font-size, 13px);
    color: var(--tn-fg);
    background: var(--tn-bg);
    padding: 16px;
  }

  h1 { font-size: 1.4em; margin-bottom: 4px; }
  h2 { font-size: 1.1em; margin: 20px 0 8px; border-bottom: 1px solid var(--tn-border); padding-bottom: 4px; }
  .section-header {
    margin: 20px 0 8px;
    border-bottom: 1px solid var(--tn-border);
    padding-bottom: 4px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
  }
  .section-header h2 {
    margin: 0;
    border-bottom: none;
    padding-bottom: 0;
  }

  .subtitle { color: var(--tn-muted); font-size: 0.85em; margin-bottom: 20px; }

  /* Budget gauge */
  .gauge-row { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; }
  .gauge-bar { flex: 1; height: 16px; background: var(--tn-bg-alt); border-radius: 4px; overflow: hidden; }
  .gauge-fill { height: 100%; border-radius: 4px; transition: width 0.4s; }
  .gauge-fill.ok       { background: var(--tn-green); }
  .gauge-fill.warning  { background: var(--tn-yellow); }
  .gauge-fill.critical { background: var(--tn-red); }
  .gauge-label { min-width: 80px; text-align: right; font-weight: bold; }

  .budget-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; margin-bottom: 8px; }
  .source-grid { display: grid; grid-template-columns: repeat(5, minmax(0, 1fr)); gap: 12px; margin-bottom: 8px; }
  .forecast-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px; margin-bottom: 8px; }
  .stat-card {
    background: var(--tn-surface);
    border: 1px solid var(--tn-border);
    border-radius: 4px;
    padding: 10px 14px;
  }
  .stat-label { font-size: 0.78em; color: var(--tn-muted); text-transform: uppercase; letter-spacing: 0.05em; }
  .stat-value { font-size: 1.3em; font-weight: bold; margin-top: 2px; }
  .stat-value.critical { color: var(--tn-red); }
  .stat-value.warning  { color: var(--tn-yellow); }
  .stat-value.ok       { color: var(--tn-green); }
  /* "pending" marks a figure that is not measured yet (e.g. no IDE transcripts found yet) so it
     is never read as a real value. */
  .stat-value.pending  { color: var(--tn-muted); font-style: italic; font-size: 1em; }
  .source-note { font-size: 0.8em; color: var(--tn-muted); margin: 4px 0 8px; }
  .cache-title { font-size: 0.8em; color: var(--tn-muted); margin: 0 0 4px; }
  .cache-table { width: 100%; border-collapse: collapse; font-size: 0.8em; }
  .cache-table th, .cache-table td { padding: 4px 6px; border-bottom: 1px solid var(--tn-border); }
  .cache-table th { color: var(--tn-muted); text-transform: uppercase; font-weight: normal; }
  .cache-table tr:last-child td { border-bottom: none; }

  /* Tables */
  table { width: 100%; border-collapse: collapse; font-size: 0.9em; }
  th { text-align: left; padding: 6px 8px; color: var(--tn-muted); font-weight: normal; font-size: 0.85em; text-transform: uppercase; border-bottom: 1px solid var(--tn-border); }
  td { padding: 6px 8px; border-bottom: 1px solid var(--tn-border); }
  tr:last-child td { border-bottom: none; }
  tr:hover td { background: var(--tn-bg-alt); }

  .badge { display: inline-block; padding: 2px 6px; border-radius: 3px; font-size: 0.78em; font-weight: bold; }
  .badge.active   { background: #1a6b2a; color: #7ec891; }
  .badge.high     { background: #6b1a1a; color: #e09090; }
  .badge.medium   { background: #6b5a1a; color: #e0c890; }
  .badge.low      { background: #1a4b6b; color: #90c8e0; }
  .badge.ok       { background: #1a4b1a; color: #90e090; }

  .footer { margin-top: 24px; font-size: 0.78em; color: var(--tn-muted); }
  .allowance-inline {
    display: flex;
    gap: 8px;
    align-items: center;
    flex-wrap: wrap;
  }
  .allowance-inline input {
    width: 160px;
    background: var(--tn-bg-alt);
    color: var(--tn-fg);
    border: 1px solid var(--tn-border);
    border-radius: 4px;
    padding: 5px 8px;
    font-family: var(--vscode-font-family);
    font-size: 0.95em;
  }
  .allowance-inline button {
    border: 1px solid var(--tn-border);
    border-radius: 4px;
    padding: 5px 10px;
    cursor: pointer;
    background: var(--tn-accent);
    color: var(--tn-bg);
    font-family: var(--vscode-font-family);
    font-size: 0.95em;
  }
  .allowance-inline button:hover {
    background: var(--tn-cyan);
  }
  .allowance-error {
    font-size: 0.8em;
    color: var(--tn-red);
  }

  #no-data { color: var(--tn-muted); margin-top: 40px; text-align: center; }

  /* Usage trend (inline SVG bar chart) */
  h3 { font-size: 0.95em; margin: 4px 0 6px; color: var(--tn-muted); }
  .heading-sessions { color: #73daca; }
  .heading-models { color: #7aa2f7; }
  .heading-projects { color: #bb9af7; }
  .heading-summary { color: #e0af68; }
  .heading-cache { color: #f7768e; }
  .trend-chart { width: 100%; overflow-x: auto; }
  .trend-chart svg { display: block; max-width: 100%; height: auto; }
  .trend-bar        { fill: #4a9eff; }
  .trend-bar.anomaly { fill: #e0524f; }
  .trend-axis       { stroke: var(--tn-border); stroke-width: 1; }
  .trend-label      { fill: var(--tn-muted); font-size: 9px; }
  .trend-empty      { color: var(--tn-muted); font-size: 0.85em; padding: 12px 0; }
  .chart-legend     { font-size: 0.78em; color: var(--tn-muted); margin-top: 6px; }
  .legend-swatch    { display: inline-block; width: 10px; height: 10px; border-radius: 2px; margin: 0 4px 0 10px; vertical-align: middle; }
  .legend-swatch.normal  { background: #4a9eff; }
  .legend-swatch.anomaly { background: #e0524f; }

  /* Top consumers layout: first row (sessions+models), second row (projects+summaries). */
  .consumers-row { display: grid; gap: 16px; margin-bottom: 12px; }
  .consumers-row.primary { grid-template-columns: repeat(2, 1fr); }
  .consumers-row.secondary { grid-template-columns: repeat(3, 1fr); }
  .consumer-block table { font-size: 0.85em; }
  @media (max-width: 720px) {
    .consumers-row.primary,
    .consumers-row.secondary { grid-template-columns: 1fr; }
  }
  details.collapsible-section {
    margin-top: 20px;
  }
  details.collapsible-section summary {
    cursor: pointer;
    font-size: 1.1em;
    border-bottom: 1px solid var(--tn-border);
    padding-bottom: 4px;
    margin-bottom: 8px;
    list-style: none;
  }
  details.collapsible-section summary::-webkit-details-marker {
    display: none;
  }
  details.collapsible-section summary::before {
    content: '▸ ';
    color: var(--tn-muted);
  }
  details.collapsible-section[open] summary::before {
    content: '▾ ';
  }
</style>
</head>
<body>
<h1>💰 Copilot Token Budget</h1>
<p class="subtitle">AT&T Enterprise · Local session data · Auto-refreshes every 30s</p>

<div id="content" style="display:none">
  <div class="section-header">
    <h2>Monthly Budget</h2>
    <div class="allowance-inline">
      <input id="allowance-input" type="number" min="1" step="1" placeholder="Allowance" />
      <button id="allowance-apply">Apply</button>
      <span id="allowance-error" class="allowance-error"></span>
    </div>
  </div>
  <div class="gauge-row">
    <div class="gauge-bar"><div id="gauge-fill" class="gauge-fill"></div></div>
    <div class="gauge-label"><span id="gauge-pct">—</span></div>
  </div>
  <div class="budget-grid">
    <div class="stat-card">
      <div class="stat-label">Used</div>
      <div id="used-credits" class="stat-value">—</div>
    </div>
    <div class="stat-card">
      <div class="stat-label">Allowance</div>
      <div id="allowed-credits" class="stat-value">—</div>
    </div>
    <div class="stat-card">
      <div class="stat-label">Remaining</div>
      <div id="remaining-credits" class="stat-value">—</div>
    </div>
  </div>
  <p id="billing-note" class="source-note">Live billing source: estimated</p>

<h2>Source Breakdown</h2>
<div class="source-grid">
  <div class="stat-card">
    <div class="stat-label">CLI Sessions:</div>
    <div id="cli-sessions" class="stat-value">—</div>
  </div>
  <div class="stat-card">
    <div class="stat-label">CLI Credits:</div>
    <div id="cli-credits" class="stat-value">—</div>
  </div>
  <div class="stat-card">
    <div class="stat-label" id="ide-sessions-label">IDE Sessions:</div>
    <div id="ide-sessions" class="stat-value pending">—</div>
  </div>
  <div class="stat-card">
    <div class="stat-label" id="ide-credits-label">IDE Credits:</div>
    <div id="ide-credits" class="stat-value pending">—</div>
  </div>
  <div class="stat-card">
    <div class="stat-label">Tracked Total</div>
    <div id="source-total" class="stat-value">—</div>
  </div>
</div>
<p id="ide-note" class="source-note">No IDE sessions were discovered under the standard VS Code user-data paths yet, so the dashboard is showing CLI only.</p>

  <h2>Forecast</h2>
  <div class="forecast-grid">
    <div class="stat-card">
      <div class="stat-label">Daily Burn Rate</div>
      <div id="forecast-burn" class="stat-value">—</div>
    </div>
    <div class="stat-card">
      <div class="stat-label">Projected Month-End</div>
      <div id="forecast-total" class="stat-value">—</div>
    </div>
    <div class="stat-card">
      <div class="stat-label">vs Allowance</div>
      <div id="forecast-verdict" class="stat-value">—</div>
    </div>
    <div class="stat-card">
      <div class="stat-label">Premium Requests</div>
      <div id="premium-requests" class="stat-value">—</div>
    </div>
  </div>

  <h2>Usage Trend (last 14 days)</h2>
  <div id="trend-chart" class="trend-chart"></div>
  <p class="chart-legend">
    <span class="legend-swatch normal"></span> daily credits
    <span class="legend-swatch anomaly"></span> anomalous day (&gt; mean + 2σ)
  </p>

  <h2>Top Consumers</h2>
  <div class="consumers-row primary">
    <div class="consumer-block">
      <h3 class="heading-sessions">Top Sessions</h3>
      <table>
        <thead><tr><th>Project</th><th>Model</th><th>Credits</th></tr></thead>
        <tbody id="top-session-rows"></tbody>
      </table>
    </div>
    <div class="consumer-block">
      <h3 class="heading-models">Top Models</h3>
      <table>
        <thead><tr><th>Model</th><th>In K</th><th>Out K</th><th>Credits</th></tr></thead>
        <tbody id="top-model-rows"></tbody>
      </table>
    </div>
  </div>
  <div class="consumers-row secondary">
    <div class="consumer-block">
      <h3 class="heading-projects">Top Projects</h3>
      <table>
        <thead><tr><th>Project</th><th>Model</th><th>Credits</th></tr></thead>
        <tbody id="top-project-rows"></tbody>
      </table>
    </div>
    <div class="consumer-block">
      <h3 class="heading-summary">Model Consumption Summary</h3>
      <table class="cache-table">
        <thead><tr><th>Model</th><th>Credits</th><th>In K</th><th>Out K</th></tr></thead>
        <tbody id="model-summary-rows"></tbody>
      </table>
    </div>
    <div class="consumer-block">
      <h3 class="heading-cache">Prompt Cache Tokens</h3>
      <table class="cache-table">
        <thead><tr><th>Model</th><th>Read</th><th>Write</th><th>Reasoning</th></tr></thead>
        <tbody id="prompt-cache-rows"></tbody>
      </table>
    </div>
  </div>

  <h2>Instruction File Overhead</h2>
  <table>
    <thead>
      <tr><th>File</th><th>~Tokens</th><th>Scope</th><th>Recommendation</th></tr>
    </thead>
    <tbody id="instr-rows"></tbody>
  </table>

  <h2>Token Optimization Plan</h2>
  <div class="budget-grid">
    <div class="stat-card">
      <div class="stat-label">Always Loaded Tokens</div>
      <div id="opt-always" class="stat-value">—</div>
    </div>
    <div class="stat-card">
      <div class="stat-label">Target Tokens</div>
      <div id="opt-target" class="stat-value">—</div>
    </div>
    <div class="stat-card">
      <div class="stat-label">Savings / Session</div>
      <div id="opt-savings" class="stat-value">—</div>
    </div>
  </div>
  <table>
    <thead>
      <tr><th>File</th><th>Current</th><th>Target</th><th>Reduce</th><th>Action</th></tr>
    </thead>
    <tbody id="opt-rows"></tbody>
  </table>

  <details class="collapsible-section">
    <summary>Sessions This Month</summary>
    <table>
      <thead>
        <tr>
          <th>Project</th><th>Model</th>
          <th>Input K</th><th>Output K</th>
          <th>Context %</th>
          <th>Credits</th><th>Source</th><th>Status</th>
        </tr>
      </thead>
      <tbody id="session-rows"></tbody>
    </table>
  </details>

  <p class="footer">AT&T Copilot Enterprise · Local-only telemetry</p>
</div>
<div id="no-data">Waiting for data…</div>

<script nonce="${nonce}">
  const vscode = acquireVsCodeApi();
  let latestMessage = null;
  let allowanceOverride = null;

  window.addEventListener('message', event => {
    const msg = event.data;
    latestMessage = msg;
    render(msg);
  });

  function render(msg) {
    const allowanceInput = document.getElementById('allowance-input');
    const allowance =
      allowanceOverride != null
        ? allowanceOverride
        : msg.budgetState.allowedCredits;
    const bs = computeBudgetState(msg.budgetState.usedCredits, allowance);
    const pct = Math.min(bs.usedPct, 100);
    const statusClass = bs.status.toLowerCase();

    document.getElementById('content').style.display = '';
    document.getElementById('no-data').style.display = 'none';

    // Gauge
    const fill = document.getElementById('gauge-fill');
    fill.style.width = pct + '%';
    fill.className = 'gauge-fill ' + statusClass;
    document.getElementById('gauge-pct').textContent = bs.usedPct.toFixed(1) + '%';

    // Stat cards
    const usedEl = document.getElementById('used-credits');
    usedEl.textContent = formatCreditsDisplay(bs.usedCredits);
    usedEl.className = 'stat-value ' + statusClass;

    if (document.activeElement !== allowanceInput) {
      allowanceInput.value = String(Math.round(bs.allowedCredits));
    }
    document.getElementById('allowed-credits').textContent = formatCreditsDisplay(bs.allowedCredits);

    const remEl = document.getElementById('remaining-credits');
    remEl.textContent = formatCreditsDisplay(bs.remainingCredits);
    remEl.className = 'stat-value ' + (bs.remainingCredits < 0 ? 'critical' : 'ok');
    const billingNote = document.getElementById('billing-note');
    billingNote.textContent = 'Live billing source: ' + liveBillingLabel(msg.orgBillingSnapshot);

    // Source breakdown — CLI is the authoritative tracked total. IDE is only shown as
    // pending when the collector found no local IDE sessions, and it is not summed into
    // the tracked total until that source is present.
    const cliSessionsEl = document.getElementById('cli-sessions');
    const cliCreditsEl = document.getElementById('cli-credits');
    cliSessionsEl.textContent = Number(msg.cliSessionCount || 0).toLocaleString();
    cliCreditsEl.textContent = formatCreditsDisplay(msg.cliTotal);
    const ideSessionsEl = document.getElementById('ide-sessions');
    const ideCreditsEl = document.getElementById('ide-credits');
    const ideNote = document.getElementById('ide-note');
    if (msg.ideTracked) {
      ideSessionsEl.textContent = Number(msg.ideSessionCount || 0).toLocaleString();
      ideSessionsEl.className = 'stat-value';
      ideCreditsEl.textContent = formatCreditsDisplay(msg.ideTotal);
      ideCreditsEl.className = 'stat-value';
      ideNote.style.display = 'none';
      document.getElementById('source-total').textContent =
        formatCreditsDisplay(msg.cliTotal + msg.ideTotal);
    } else {
      // No IDE source found: do not present a measured "0". Label as pending and keep the
      // tracked total = CLI only so it is not implied to include IDE.
      ideSessionsEl.textContent = 'not tracked yet';
      ideSessionsEl.className = 'stat-value pending';
      ideCreditsEl.textContent = 'not tracked yet';
      ideCreditsEl.className = 'stat-value pending';
      ideNote.style.display = '';
      document.getElementById('source-total').textContent = formatCreditsDisplay(msg.cliTotal);
    }

    // Forecast block — daily burn rate, projected month-end total, over/under allowance.
    const fc = msg.forecast;
    const exceedsAllowance = fc.projectedMonthEndTotal > bs.allowedCredits;
    const overClass = exceedsAllowance ? 'critical' : 'ok';
    document.getElementById('forecast-burn').textContent = formatCreditsDisplay(fc.dailyBurn) + '/day';
    const totalEl = document.getElementById('forecast-total');
    totalEl.textContent = formatCreditsDisplay(fc.projectedMonthEndTotal);
    totalEl.className = 'stat-value ' + overClass;
    const verdictEl = document.getElementById('forecast-verdict');
    verdictEl.textContent = exceedsAllowance
      ? 'OVER ' + formatCreditsDisplay(fc.projectedMonthEndTotal - bs.allowedCredits)
      : 'within ' + formatCreditsDisplay(bs.allowedCredits - fc.projectedMonthEndTotal);
    verdictEl.className = 'stat-value ' + overClass;

    // Premium requests this month — count of paid-tier requests across settled
    // sessions (parity with the Go budget block). Only finalized sessions carry it.
    document.getElementById('premium-requests').textContent =
      Number(msg.premiumRequests || 0).toLocaleString();

    // Session rows
    const tbody = document.getElementById('session-rows');
    tbody.innerHTML = msg.sessions.map(s => {
      const date = new Date(s.startTime).toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
      const active = s.isActive ? '<span class="badge active">● ACTIVE</span>' : date;
      const sourceLabel = s.source === 'copilot-cli' ? 'CLI' : 'IDE';
      return \`<tr>
        <td>\${esc(s.projectName)}</td>
        <td>\${esc(modelShort(s.primaryModel))}</td>
        <td>\${s.inputTokensK}</td>
        <td>\${s.outputTokensK}</td>
        <td>\${s.contextPct.toFixed(1)}%</td>
        <td>\${formatCreditsDisplay(s.totalCredits)}</td>
        <td>\${sourceLabel}</td>
        <td>\${active}</td>
      </tr>\`;
    }).join('');

    // Usage trend — inline SVG bar chart, anomalies highlighted.
    renderTrend(msg.trend || []);

    // Top consumers — three small tables.
    renderConsumers('top-session-rows', msg.topSessions || [], 'session');
    renderConsumers('top-model-rows',   msg.topModels   || [], 'model');
    renderConsumers('top-project-rows', msg.topProjects || [], 'project');

    renderModelSummary('model-summary-rows', msg.modelSummary || []);
    renderPromptCache('prompt-cache-rows', msg.modelCache || []);

    // Instruction rows
    const itbody = document.getElementById('instr-rows');
    itbody.innerHTML = msg.instructionFiles.map(f => {
      const name = f.path.split(/[\\/]/).pop() || f.path;
      const sev = severityClass(f.estimatedTokens);
      const rec = savingsRec(f.estimatedTokens);
      return \`<tr>
        <td>\${esc(name)}</td>
        <td>\${f.estimatedTokens}</td>
        <td><span class="badge \${sev}">\${esc(f.scope)}</span></td>
        <td>\${esc(rec)}</td>
      </tr>\`;
    }).join('');

    // Token optimization plan.
    const opt = msg.optimization || {
      alwaysLoadedTokens: 0,
      targetTokens: 0,
      reducibleTokens: 0,
      currentCreditsPerSession: 0,
      targetCreditsPerSession: 0,
      potentialCreditsPerSession: 0,
      opportunities: [],
    };
    document.getElementById('opt-always').textContent = String(opt.alwaysLoadedTokens || 0);
    document.getElementById('opt-target').textContent = String(opt.targetTokens || 0);
    const savingsEl = document.getElementById('opt-savings');
    savingsEl.textContent = formatCreditsDisplay(opt.potentialCreditsPerSession || 0) + ' cr';
    savingsEl.className = 'stat-value ' + ((opt.potentialCreditsPerSession || 0) > 0 ? 'warning' : 'ok');

    const optRows = document.getElementById('opt-rows');
    const opportunities = opt.opportunities || [];
    if (!opportunities.length) {
      optRows.innerHTML = '<tr><td colspan="5">No optimization opportunities detected.</td></tr>';
    } else {
      optRows.innerHTML = opportunities.map(o =>
        '<tr>' +
          '<td>' + esc(o.name) + '</td>' +
          '<td>' + o.currentTokens + '</td>' +
          '<td>' + o.targetTokens + '</td>' +
          '<td>-' + o.reducibleTokens + '</td>' +
          '<td>' + esc(o.recommendation) + '</td>' +
        '</tr>'
      ).join('');
    }
  }

  // renderTrend draws an inline SVG bar chart of daily credits. Anomalous days use a
  // distinct colour. Built entirely from posted numeric/string data — no external libs,
  // CSP-safe (this whole script runs under the nonce). Text is esc()'d defensively.
  function renderTrend(trend) {
    const host = document.getElementById('trend-chart');
    if (!trend.length) {
      host.innerHTML = '<p class="trend-empty">No usage data yet for the trend window.</p>';
      return;
    }

    const W = 640, H = 160, padL = 40, padB = 28, padT = 10, padR = 8;
    const plotW = W - padL - padR;
    const plotH = H - padT - padB;
    const maxCredits = Math.max(1, ...trend.map(d => d.credits));
    const n = trend.length;
    const slot = plotW / n;
    const barW = Math.max(2, slot * 0.7);

    let bars = '';
    for (let i = 0; i < n; i++) {
      const d = trend[i];
      const h = (d.credits / maxCredits) * plotH;
      const x = padL + i * slot + (slot - barW) / 2;
      const y = padT + (plotH - h);
      const cls = d.anomalous ? 'trend-bar anomaly' : 'trend-bar';
      const day = d.key.slice(8); // DD from YYYY-MM-DD
      const title = esc(d.key) + ': ' + formatCreditsDisplay(d.credits) + (d.anomalous ? ' (anomalous)' : '');
      bars += '<rect class="' + cls + '" x="' + x.toFixed(1) + '" y="' + y.toFixed(1) +
              '" width="' + barW.toFixed(1) + '" height="' + Math.max(0, h).toFixed(1) + '">' +
              '<title>' + title + '</title></rect>';
      // Show a tick label every other bar to avoid crowding.
      if (n <= 14 || i % 2 === 0) {
        bars += '<text class="trend-label" x="' + (x + barW / 2).toFixed(1) +
                '" y="' + (H - padB + 12) + '" text-anchor="middle">' + esc(day) + '</text>';
      }
    }

    // Y axis max label + baseline.
    const axis =
      '<line class="trend-axis" x1="' + padL + '" y1="' + padT + '" x2="' + padL + '" y2="' + (padT + plotH) + '" />' +
      '<line class="trend-axis" x1="' + padL + '" y1="' + (padT + plotH) + '" x2="' + (W - padR) + '" y2="' + (padT + plotH) + '" />' +
      '<text class="trend-label" x="' + (padL - 4) + '" y="' + (padT + 8) + '" text-anchor="end">' + maxCredits.toFixed(0) + '</text>' +
      '<text class="trend-label" x="' + (padL - 4) + '" y="' + (padT + plotH) + '" text-anchor="end">0</text>';

    host.innerHTML =
      '<svg viewBox="0 0 ' + W + ' ' + H + '" preserveAspectRatio="xMidYMid meet" role="img" aria-label="Daily credit usage">' +
      axis + bars + '</svg>';
  }

  // renderConsumers fills a top-consumers table body. The column layout varies by kind
  // so each row matches its table header.
  function renderConsumers(tbodyId, rows, kind) {
    const tbody = document.getElementById(tbodyId);
    if (!rows.length) {
      const cols = kind === 'model' ? 4 : 3;
      tbody.innerHTML = '<tr><td colspan="' + cols + '">No data</td></tr>';
      return;
    }
    tbody.innerHTML = rows.map(c => {
      if (kind === 'model') {
        return '<tr><td>' + esc(modelShort(c.model)) + '</td><td>' + c.inputTokensK +
               '</td><td>' + c.outputTokensK + '</td><td>' + formatCreditsDisplay(c.credits) + '</td></tr>';
      }
      return '<tr><td>' + esc(c.name) + '</td><td>' + esc(modelShort(c.model)) +
               '</td><td>' + formatCreditsDisplay(c.credits) + '</td></tr>';
    }).join('');
  }

  function renderModelSummary(tbodyId, rows) {
    const tbody = document.getElementById(tbodyId);
    if (!rows.length) {
      tbody.innerHTML = '<tr><td colspan="4">No model usage data</td></tr>';
      return;
    }
    tbody.innerHTML = rows.map(r =>
      '<tr>' +
        '<td>' + esc(modelShort(r.model)) + '</td>' +
        '<td>' + esc(formatCreditsDisplay(r.credits)) + '</td>' +
        '<td>' + esc(String(r.inputTokensK)) + '</td>' +
        '<td>' + esc(String(r.outputTokensK)) + '</td>' +
      '</tr>'
    ).join('');
  }

  function renderPromptCache(tbodyId, rows) {
    const tbody = document.getElementById(tbodyId);
    if (!rows.length) {
      tbody.innerHTML = '<tr><td colspan="4">No cache token data</td></tr>';
      return;
    }
    tbody.innerHTML = rows.map(r =>
      '<tr>' +
        '<td>' + esc(modelShort(r.model)) + '</td>' +
        '<td>' + esc(fmtTokenCompact(r.cacheReadTokens)) + '</td>' +
        '<td>' + esc(fmtTokenCompact(r.cacheWriteTokens)) + '</td>' +
        '<td>' + esc(fmtTokenCompact(r.reasoningTokens)) + '</td>' +
      '</tr>'
    ).join('');
  }

  // fmtTokenCompact renders token counts as M/K for readability.
  function fmtTokenCompact(n) {
    n = Number(n) || 0;
    if (n >= 1000000) {
      return (n / 1000000).toLocaleString(undefined, { maximumFractionDigits: 1 }) + 'M';
    }
    if (n >= 1000) {
      return (n / 1000).toLocaleString(undefined, { maximumFractionDigits: 1 }) + 'K';
    }
    return String(n);
  }

  // formatCreditsDisplay must be defined HERE, in webview (browser) scope — the inline
  // script runs in an isolated context and cannot see the extension-host module function
  // of the same name. Renders raw credits with thousands separators and no decimals.
  function formatCreditsDisplay(credits) {
    return Number(credits).toLocaleString(undefined, { maximumFractionDigits: 0 });
  }

  // Webview-local copy of the live-billing label formatter. The webview script runs
  // in an isolated context and cannot call extension-host TypeScript helpers directly.
  function liveBillingLabel(snapshot) {
    if (!snapshot) return '(estimated)';
    if (snapshot.availability === 'unavailable') return '(unavailable)';
    if (snapshot.sourceLabel) return snapshot.sourceLabel;
    const refreshed = new Date(snapshot.lastRefreshedAt || 0);
    const ms = Date.now() - refreshed.getTime();
    const h = ms > 0 ? Math.max(1, Math.floor(ms / (60 * 60 * 1000))) : 1;
    return '(org aggregate, ~' + h + 'h ago)';
  }

  function modelShort(m) {
    return (m || '').replace('claude-', '').replace('gpt-', '');
  }

  function severityClass(t) {
    if (t >= 2000) return 'high';
    if (t >= 500)  return 'medium';
    return 'low';
  }

  function savingsRec(t) {
    if (t >= 5000) return 'CRITICAL — split or remove';
    if (t >= 2000) return 'HIGH — trim to <2K tokens';
    if (t >= 500)  return 'MEDIUM — review content';
    return 'OK';
  }

  function esc(s) {
    return String(s)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;');
  }

  function computeBudgetState(usedCredits, allowedCredits) {
    const allowed = Number(allowedCredits);
    const safeAllowed = Number.isFinite(allowed) && allowed > 0 ? allowed : 1;
    const usedPct = (usedCredits / safeAllowed) * 100;
    const remainingCredits = safeAllowed - usedCredits;
    return {
      usedCredits,
      allowedCredits: safeAllowed,
      usedPct,
      remainingCredits,
      status: budgetStatusFor(usedPct),
    };
  }

  function budgetStatusFor(pct) {
    if (pct > 90) return 'CRITICAL';
    if (pct >= 60) return 'WARNING';
    return 'OK';
  }

  function readAllowanceInput() {
    const input = document.getElementById('allowance-input');
    const value = Number(input.value);
    if (!Number.isFinite(value) || value <= 0) {
      return null;
    }
    return value;
  }

  function showAllowanceError(message) {
    document.getElementById('allowance-error').textContent = message || '';
  }

  function applyAllowance(updateSetting) {
    const value = readAllowanceInput();
    if (value === null) {
      showAllowanceError('Enter a positive allowance value.');
      return;
    }
    showAllowanceError('');
    allowanceOverride = value;
    if (latestMessage) {
      render(latestMessage);
    }
    if (updateSetting) {
      vscode.postMessage({ type: 'setAllowance', allowanceCredits: value });
    }
  }

  document.getElementById('allowance-input').addEventListener('input', () => {
    applyAllowance(false);
  });
  document.getElementById('allowance-apply').addEventListener('click', () => {
    applyAllowance(true);
  });
  document.getElementById('allowance-input').addEventListener('keydown', (e) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      applyAllowance(true);
    }
  });
</script>
</body>
</html>`;
}
