// ui/dashboardPanel.ts — singleton VS Code webview panel for the budget dashboard.
// Uses VS Code CSS variables so it respects the user's light/dark theme automatically.

// formatCreditsDisplay renders raw credits with thousands separators and up to two
// decimals — parity with the Go side (e.g. "8,554.03", "656.54"). Credits are already
// credits (nanoAIU / 1e9), so there is no further scaling and no "B"/billions unit.
function formatCreditsDisplay(credits: number): string {
  return credits.toLocaleString(undefined, { maximumFractionDigits: 2 });
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
  instructionFiles: InstructionFile[];
  forecast: SerializedForecast;
  trend: TrendPoint[]; // last 14 daily buckets, anomalies flagged
  topSessions: SerializedConsumer[];
  topModels: SerializedConsumer[];
  topProjects: SerializedConsumer[];
  modelCache: SerializedModelCache[]; // per-model prompt-cache tallies (Top Models)
  premiumRequests: number; // total premium requests across this month's settled sessions
  cliTotal: number; // total credits from CLI sessions — the real, tracked total
  ideTotal: number; // IDE credits: ALWAYS 0 today (no live IDE source); presented as "not tracked yet"
  ideTracked: boolean; // false until a live IDE source exists; gates the IDE figure's framing
  optimization: SerializedOptimizationSummary;
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

    // Compute source breakdown and premium-request total. cliTotal is the real,
    // tracked total. ideTotal is collected for completeness but is ALWAYS 0 today —
    // VS Code Chat is a separate, not-yet-implemented source (see reader.ts ideCollector
    // / ADR-007). It is presented as "not tracked yet (Phase 6)", NOT summed into a
    // combined total, so the dashboard never implies the IDE figure is a measured 0.
    let cliTotal = 0;
    let ideTotal = 0;
    let premiumRequests = 0;
    for (const s of sessions) {
      const credits = s.totalNanoAIU / 1_000_000_000;
      if (s.source === "copilot-cli") {
        cliTotal += credits;
      } else if (s.source === "copilot-ide") {
        ideTotal += credits;
      }
      premiumRequests += s.totalPremiumRequests;
    }
    // ideTracked stays false until a live IDE collector contributes sessions. Detecting
    // an actual IDE session (not the no-op stub) flips it on so the UI relabels itself
    // automatically when Phase 6 lands.
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

    const plan = buildOptimizationSummary(instructionFiles);
    const current = estimateInstructionCostPerSession(plan.alwaysLoadedTokens);
    const target = estimateInstructionCostPerSession(plan.targetTokens);

    const message: DashboardMessage = {
      sessions: sessions.map((s) => serialize(s, cfg)),
      budgetState,
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
      premiumRequests,
      cliTotal, // real tracked total
      ideTotal, // always 0 today — see ideTracked
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
  * { box-sizing: border-box; margin: 0; padding: 0; }

  body {
    font-family: var(--vscode-font-family);
    font-size: var(--vscode-font-size);
    color: var(--vscode-editor-foreground);
    background: var(--vscode-editor-background);
    padding: 16px;
  }

  h1 { font-size: 1.4em; margin-bottom: 4px; }
  h2 { font-size: 1.1em; margin: 20px 0 8px; border-bottom: 1px solid var(--vscode-panel-border); padding-bottom: 4px; }

  .subtitle { color: var(--vscode-descriptionForeground); font-size: 0.85em; margin-bottom: 20px; }

  /* Budget gauge */
  .gauge-row { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; }
  .gauge-bar { flex: 1; height: 16px; background: var(--vscode-progressBar-background, #333); border-radius: 4px; overflow: hidden; }
  .gauge-fill { height: 100%; border-radius: 4px; transition: width 0.4s; }
  .gauge-fill.ok       { background: #4caf50; }
  .gauge-fill.warning  { background: #ff9800; }
  .gauge-fill.critical { background: var(--vscode-statusBarItem-errorBackground, #c72e0f); }
  .gauge-label { min-width: 80px; text-align: right; font-weight: bold; }

  .budget-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; margin-bottom: 8px; }
  .stat-card {
    background: var(--vscode-sideBar-background);
    border: 1px solid var(--vscode-panel-border);
    border-radius: 4px;
    padding: 10px 14px;
  }
  .stat-label { font-size: 0.78em; color: var(--vscode-descriptionForeground); text-transform: uppercase; letter-spacing: 0.05em; }
  .stat-value { font-size: 1.3em; font-weight: bold; margin-top: 2px; }
  .stat-value.critical { color: var(--vscode-statusBarItem-errorBackground, #e05252); }
  .stat-value.warning  { color: #ff9800; }
  .stat-value.ok       { color: #4caf50; }
  /* "pending" marks a figure that is not measured yet (e.g. IDE source, Phase 6) so it
     is never read as a real value. */
  .stat-value.pending  { color: var(--vscode-descriptionForeground); font-style: italic; font-size: 1em; }
  .source-note { font-size: 0.8em; color: var(--vscode-descriptionForeground); margin: 4px 0 8px; }
  .cache-note  { font-size: 0.82em; color: var(--vscode-descriptionForeground); margin: 6px 0 0; }
  .cache-note .dominant { color: #ff9800; }

  /* Tables */
  table { width: 100%; border-collapse: collapse; font-size: 0.9em; }
  th { text-align: left; padding: 6px 8px; color: var(--vscode-descriptionForeground); font-weight: normal; font-size: 0.85em; text-transform: uppercase; border-bottom: 1px solid var(--vscode-panel-border); }
  td { padding: 6px 8px; border-bottom: 1px solid var(--vscode-panel-border, rgba(255,255,255,0.08)); }
  tr:last-child td { border-bottom: none; }
  tr:hover td { background: var(--vscode-list-hoverBackground); }

  .badge { display: inline-block; padding: 2px 6px; border-radius: 3px; font-size: 0.78em; font-weight: bold; }
  .badge.active   { background: #1a6b2a; color: #7ec891; }
  .badge.high     { background: #6b1a1a; color: #e09090; }
  .badge.medium   { background: #6b5a1a; color: #e0c890; }
  .badge.low      { background: #1a4b6b; color: #90c8e0; }
  .badge.ok       { background: #1a4b1a; color: #90e090; }

  .footer { margin-top: 24px; font-size: 0.78em; color: var(--vscode-descriptionForeground); }

  #no-data { color: var(--vscode-descriptionForeground); margin-top: 40px; text-align: center; }

  /* Usage trend (inline SVG bar chart) */
  h3 { font-size: 0.95em; margin: 4px 0 6px; color: var(--vscode-descriptionForeground); }
  .trend-chart { width: 100%; overflow-x: auto; }
  .trend-chart svg { display: block; max-width: 100%; height: auto; }
  .trend-bar        { fill: #4a9eff; }
  .trend-bar.anomaly { fill: #e0524f; }
  .trend-axis       { stroke: var(--vscode-panel-border); stroke-width: 1; }
  .trend-label      { fill: var(--vscode-descriptionForeground); font-size: 9px; }
  .trend-empty      { color: var(--vscode-descriptionForeground); font-size: 0.85em; padding: 12px 0; }
  .chart-legend     { font-size: 0.78em; color: var(--vscode-descriptionForeground); margin-top: 6px; }
  .legend-swatch    { display: inline-block; width: 10px; height: 10px; border-radius: 2px; margin: 0 4px 0 10px; vertical-align: middle; }
  .legend-swatch.normal  { background: #4a9eff; }
  .legend-swatch.anomaly { background: #e0524f; }

  /* Top consumers — three small tables side by side */
  .consumers-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; }
  .consumer-block table { font-size: 0.85em; }
  @media (max-width: 720px) { .consumers-grid { grid-template-columns: 1fr; } }
</style>
</head>
<body>
<h1>💰 Copilot Token Budget</h1>
<p class="subtitle">AT&T Enterprise · Local session data · Auto-refreshes every 30s</p>

<div id="content" style="display:none">
  <h2>Monthly Budget</h2>
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

<h2>Source Breakdown</h2>
<div class="budget-grid">
  <div class="stat-card">
    <div class="stat-label">CLI Sessions (tracked)</div>
    <div id="cli-total" class="stat-value">—</div>
  </div>
  <div class="stat-card">
    <div class="stat-label" id="ide-label">IDE Sessions</div>
    <div id="ide-total" class="stat-value pending">—</div>
  </div>
  <div class="stat-card">
    <div class="stat-label">Tracked Total</div>
    <div id="source-total" class="stat-value">—</div>
  </div>
</div>
<p id="ide-note" class="source-note">IDE (VS Code Chat) usage is not tracked yet — Phase 6. The tracked total reflects CLI sessions only.</p>

  <h2>Forecast</h2>
  <div class="budget-grid">
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
  <div class="consumers-grid">
    <div class="consumer-block">
      <h3>Top Sessions</h3>
      <table>
        <thead><tr><th>Project</th><th>Model</th><th>Credits</th></tr></thead>
        <tbody id="top-session-rows"></tbody>
      </table>
    </div>
    <div class="consumer-block">
      <h3>Top Models</h3>
      <table>
        <thead><tr><th>Model</th><th>In K</th><th>Out K</th><th>Credits</th></tr></thead>
        <tbody id="top-model-rows"></tbody>
      </table>
      <div id="model-cache-note"></div>
    </div>
    <div class="consumer-block">
      <h3>Top Projects</h3>
      <table>
        <thead><tr><th>Project</th><th>Model</th><th>Credits</th></tr></thead>
        <tbody id="top-project-rows"></tbody>
      </table>
    </div>
  </div>

  <h2>Sessions This Month</h2>
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

  <p class="footer">AT&T Copilot Enterprise promo — 7,000 cr/month until 2026-09-01</p>
</div>
<div id="no-data">Waiting for data…</div>

<script nonce="${nonce}">
  const vscode = acquireVsCodeApi();

  window.addEventListener('message', event => {
    const msg = event.data;
    render(msg);
  });

  function render(msg) {
    const bs = msg.budgetState;
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

    document.getElementById('allowed-credits').textContent = formatCreditsDisplay(bs.allowedCredits);

    const remEl = document.getElementById('remaining-credits');
    remEl.textContent = formatCreditsDisplay(bs.remainingCredits);
    remEl.className = 'stat-value ' + (bs.remainingCredits < 0 ? 'critical' : 'ok');

    // Source breakdown — CLI is the real tracked total. IDE is NOT a live source yet
    // (VS Code Chat is separate, Phase 6), so its figure is shown as "not tracked yet"
    // and is NEVER summed into the tracked total. ideTracked flips on automatically once
    // a live IDE collector contributes sessions, at which point it reads as a real value.
    const cliEl = document.getElementById('cli-total');
    cliEl.textContent = formatCreditsDisplay(msg.cliTotal);
    const ideEl = document.getElementById('ide-total');
    const ideNote = document.getElementById('ide-note');
    if (msg.ideTracked) {
      ideEl.textContent = formatCreditsDisplay(msg.ideTotal);
      ideEl.className = 'stat-value';
      ideNote.style.display = 'none';
      document.getElementById('source-total').textContent =
        formatCreditsDisplay(msg.cliTotal + msg.ideTotal);
    } else {
      // No live IDE source: do not present a measured "0". Label as pending and keep the
      // tracked total = CLI only so it is not implied to include IDE.
      ideEl.textContent = 'not tracked yet';
      ideEl.className = 'stat-value pending';
      ideNote.style.display = '';
      document.getElementById('source-total').textContent = formatCreditsDisplay(msg.cliTotal);
    }

    // Forecast block — daily burn rate, projected month-end total, over/under allowance.
    const fc = msg.forecast;
    const overClass = fc.exceedsAllowance ? 'critical' : 'ok';
    document.getElementById('forecast-burn').textContent = formatCreditsDisplay(fc.dailyBurn) + '/day';
    const totalEl = document.getElementById('forecast-total');
    totalEl.textContent = formatCreditsDisplay(fc.projectedMonthEndTotal);
    totalEl.className = 'stat-value ' + overClass;
    const verdictEl = document.getElementById('forecast-verdict');
    verdictEl.textContent = fc.exceedsAllowance
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
        <td>\${s.totalCredits.toFixed(2)}</td>
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

    // Per-model prompt-cache / reasoning tokens — parity with the Go report's
    // renderModelCacheReads. Only models with cache reads appear; empty otherwise.
    renderModelCache(msg.modelCache || []);

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
    savingsEl.textContent = (opt.potentialCreditsPerSession || 0).toFixed(2) + ' cr';
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
      const cls = d.anomaly ? 'trend-bar anomaly' : 'trend-bar';
      const day = d.key.slice(8); // DD from YYYY-MM-DD
      const title = esc(d.key) + ': ' + formatCreditsDisplay(d.credits) + (d.anomaly ? ' (anomalous)' : '');
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
               '</td><td>' + c.outputTokensK + '</td><td>' + c.credits.toFixed(2) + '</td></tr>';
      }
      return '<tr><td>' + esc(c.name) + '</td><td>' + esc(modelShort(c.model)) +
             '</td><td>' + c.credits.toFixed(2) + '</td></tr>';
    }).join('');
  }

  // renderModelCache lists prompt-cache read tokens (plus write/reasoning) per model
  // under the Top Models table — surfacing data captured on ModelMetric that was
  // otherwise never shown. Mirrors the Go renderModelCacheReads section: one line per
  // model with cache reads, flagged when cache reads dominate raw input. All text is
  // esc()'d; this whole script runs under the nonce, so it is CSP-safe.
  function renderModelCache(rows) {
    const host = document.getElementById('model-cache-note');
    if (!rows.length) {
      host.innerHTML = '';
      return;
    }
    const lines = rows.map(r => {
      const extras = [];
      if (r.cacheWriteTokens) extras.push(fmtK(r.cacheWriteTokens) + ' write');
      if (r.reasoningTokens)  extras.push(fmtK(r.reasoningTokens) + ' reasoning');
      const extra = extras.length ? ' (' + esc(extras.join(', ')) + ')' : '';
      const dom = r.dominatesInput
        ? ' <span class="dominant">cache reads dominate</span>'
        : '';
      return esc(modelShort(r.model)) + ': ' + esc(fmtK(r.cacheReadTokens)) +
             ' cache-read tokens' + extra + dom;
    });
    host.innerHTML = '<p class="cache-note">' + lines.join('<br/>') + '</p>';
  }

  // fmtK renders a token count compactly (e.g. 12,345 → "12.3K"); small counts stay raw.
  function fmtK(n) {
    n = Number(n) || 0;
    if (n >= 1000) {
      return (n / 1000).toLocaleString(undefined, { maximumFractionDigits: 1 }) + 'K';
    }
    return String(n);
  }

  // formatCreditsDisplay must be defined HERE, in webview (browser) scope — the inline
  // script runs in an isolated context and cannot see the extension-host module function
  // of the same name. Renders raw credits with thousands separators, up to two decimals
  // (parity with the Go side, e.g. "8,554.03"). No scaling, no "B" unit.
  function formatCreditsDisplay(credits) {
    return Number(credits).toLocaleString(undefined, { maximumFractionDigits: 2 });
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
</script>
</body>
</html>`;
}
