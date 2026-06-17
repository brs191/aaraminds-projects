# Competitive Feature Analysis — AI Token/Model Usage Dashboards

**Date:** 2026-06-15
**Purpose:** Survey the most popular AI token/model/cost usage dashboards, then recommend a constraint-filtered, prioritized feature backlog for the Copilot Token Budget tool.
**Hard constraint that filters everything below:** this tool is **local-first, single-user, zero-network** (ADR-001). It reads files the Copilot CLI already writes; it makes no API calls and runs no server. Any feature that requires a proxy, a backend, or multi-user aggregation is out of scope by design — not an oversight.

---

## 1. The landscape splits into two camps

**Camp A — Cloud / proxy LLM-observability platforms:** Helicone, Langfuse, LangSmith, Portkey, LiteLLM, Datadog LLM Observability, Cloudflare AI Gateway, Traceloop/OpenLLMetry, Arize Phoenix, Lunary, plus the provider consoles (OpenAI, Anthropic, OpenRouter). These sit *inline* in the request path (gateways) or ingest telemetry server-side. They get features we structurally cannot: real spend **enforcement** (block/reroute on overrun), cross-provider unified billing, live cache-hit savings, multi-user/team attribution, server-pushed alerting, and authoritative provider billing numbers.

**Camp B — Developer / local usage tools:** **ccusage** (the reference), Copilot Premium Usage Monitor (VS Code), Tokenlint, Cursor's usage page, the OpenAI/Anthropic/OpenRouter usage pages, offline tokenizers (tokens.dev et al.), and macOS **Stats** (for widget/UX patterns only). These are passive read-and-aggregate over already-captured data — exactly our model.

**We are firmly Camp B.** The right strategy is to match Camp B's best UX, borrow the *passive-analysis* subset of Camp A's features (compute locally from a bundled price table), and explicitly decline the inline/cloud features.

---

## 2. North star: ccusage

`ccusage` is the closest existing tool to ours and the single best reference. It is 100% local, read-only, offline-capable (cached pricing, `--offline`), and — critically — **it already parses GitHub Copilot CLI logs**, which it reads from `~/.copilot/otel/*.jsonl`. Its standout features: a compact **statusline** (`model | session $ / today $ / block (time left) | 🔥 burn rate | 🧠 context %`), **5-hour billing-block** views with burn rate + projected total + gap detection, **token-limit progress bars** color-coded green<70 / yellow 70–90 / red≥90, per-model + cache-token columns, and `--json` export so the CLI doubles as the data engine for thin consumers.

> **Actionable technical lead:** ccusage reads `~/.copilot/otel/*.jsonl`, whereas our Phase 0 settled on `~/.copilot/session-state/<uuid>/events.jsonl`. These may be different, and the OTEL stream may carry richer per-model / cache-token / latency fields. **Recommend a short spike** to diff the two sources and decide whether to additionally ingest the OTEL logs. This could unlock several features below cheaply.

---

## 3. Where we already stand

Before adding anything, what the tool already does (so we don't rebuild it): per-month credit total + % of allowance, status-bar badge with green/yellow/red, active-session list + per-session credits, instruction-file overhead audit, per-model cost (`get_model_costs`), daily burn rate + projected month-end total (now in the VS Code dashboard and tree), Teams threshold alerts, a dashboard webview, a tree view, and (since v1.1) six MCP tools. That already covers a meaningful slice of Camp B.

> **Update — v1.1 usage-insight increment shipped (2026-06-16).** The P0 cluster and most of the
> P1 cluster from §5 below have landed (Go + TS, local-first). Statuses in the backlog table are
> flipped to **Have (v1.1)** accordingly. Items still marked *(data-gated)* — cache-token
> accounting, latency/TTFT, and the OTEL-source decision — remain pending the OTEL discovery spike
> (see §5 step 1 / Step 6.0). See `evaluation/PHASE7_ACCEPTANCE.md` and ADR-008/009.

---

## 4. Recommended feature backlog (prioritized, constraint-filtered)

Legend — **Have**: already implemented · **P0**: high value, low/medium effort, strong fit · **P1**: valuable, more effort · **P2**: nice-to-have · **Exclude**: conflicts with local-first/zero-network.

| Feature (as seen in popular tools) | Status | Notes / fit |
|---|---|---|
| Status-bar/menubar live badge (cost, %, model) | **Have (v1.1)** | Richer status-bar tooltip (today/month/allowance%/burn/projected/context%) + a ccusage-style `cmd/statusline` one-liner shipped in v1.1. |
| Color-coded thresholds (green/yellow/red) | **Have** | Already in status bar + gauge. |
| Per-model token & cost breakdown | **Have (v1.1)** | `get_model_costs` + Top Consumers (models) tables surfaced in the VS Code dashboard, not just MCP. |
| Daily / weekly / monthly usage views | **Have (v1.1)** | `internal/analytics` Daily/Weekly/MonthlySeries (UTC bucketing) + a Usage Trend chart in the dashboard and `get_usage_timeseries` MCP tool. |
| Burn rate + window/month-end projection | **Have** | Added 2026-06-15. Consider also ccusage's **5-hour block** burn view. |
| Budget/quota with % remaining + progress bar | **Have** | Self-defined allowance (7,000); already color-coded. |
| Context-window / instruction-overhead % per session | **Have (v1.1)** | `analytics.ContextWindowPct` (currentTokens / model window × 100) surfaced on active sessions in CLI + dashboard (context-% column) and in the statusline 🧠. |
| Rich tooltip / click-to-expand popover | **Have (v1.1)** | Status-bar tooltip now shows today/month/allowance%/burn/projected/context%. |
| "Most expensive" sessions/days/models list | **Have (v1.1)** | `analytics.TopSessions/TopModels/TopProjects` (top-N, credits desc) in CLI "TOP CONSUMERS", dashboard tables, and `get_top_consumers` MCP tool. |
| History & trends with date/model filters | **P1** | Weekly/monthly series ship; date/model **filters** in the webview are still backlog. |
| Sparklines / inline trend graph | **Have (v1.1)** | Usage Trend inline-SVG chart in the dashboard webview. |
| Cache-token accounting (cache-create vs cache-read) | **P1** *(data-gated)* | Only if the events/OTEL records carry cache-token fields — verify in the spike. Cache reads are far cheaper; surfacing them is real savings insight. |
| Input vs output token split | **Have (v1.1)** | Input/output split surfaced in the dashboard and carried through analytics/export (`inputTokens`/`outputTokens`). |
| Export to JSON / CSV | **Have (v1.1)** | `internal/export` + `cmd/analyze --json/--csv` and the extension's `copilotBudget.exportUsage` command (JSON camelCase, RFC-4180 CSV). |
| Editable / bundled pricing table | **Have (v1.1)** | `internal/pricing` + `pricing.json` override (ADR-008): bundled defaults, merge-over, graceful fallback. TS via `copilotBudget.pricingPath`. All costs labelled **estimated**. |
| Local threshold notifications (VS Code) | **Have** (extend) | We pop a VS Code warning; the Teams path covers push. Keep purely local fallback. |
| Per-project / per-repo attribution | **Have (v1.1)** | `analytics.TopProjects` groups usage by project name; surfaced in CLI/dashboard Top Consumers and `get_top_consumers`. |
| Model picker / switch from badge | **P2** | Lower value for us — model is chosen by the CLI/session, not us; we report, not control. |
| Latency / TTFT metrics | **P2** *(data-gated)* | Only if timestamps/durations exist in the logs; secondary to cost for a *budget* tool. |
| Anomaly / spike **detection** (historical) | **Have (v1.1)** | `analytics.AnomalousDays` flags days > mean + 2·σ (population, ≥3-point floor); shown as anomaly flags in the CLI USAGE TREND section. |
| Combined/composable widget, customization | **P2** | Stats-style "show only what I care about" toggles in settings. |
| Real spend **enforcement** (block/reroute) | **Exclude** | Requires being inline as a proxy. Not our architecture. |
| Cross-provider unified billing | **Exclude** | Single provider (Copilot), no gateway. |
| Authoritative provider billing reconciliation | **Exclude** | Needs network/billing API; we show **estimates** by design. State this in the UI (ccusage does). |
| Email/server-pushed alerts, team/org dashboards | **Exclude** | Multi-user/backend by nature. (Teams webhook is the one sanctioned outbound, already built.) |
| Live online pricing auto-refresh | **Exclude** | Ship cached pricing + optional manual update; never an automatic call (ADR-001). |

---

## 5. Recommended phasing

A coherent "v1.1 — usage insight" increment, all within local-first:

1. **Spike the OTEL data source** (`~/.copilot/otel/*.jsonl` vs `session-state/events.jsonl`) — decide the richest local substrate. Gates several data-dependent items.
2. **P0 cluster (high value, low risk):** daily/weekly views + time-series chart; context-window % per session; "most expensive" top-N; JSON/CSV export; externalize the pricing table to an overridable local config; enrich the status-bar one-liner toward ccusage's.
3. **P1 cluster:** history/trends with filters, sparklines, input/output + cache-token split, per-project attribution, deeper tooltip/popover.
4. **P2 / opportunistic:** anomaly detection, latency (if data exists), widget customization.

Everything above is **passive read-and-aggregate** — it adds insight without touching the zero-network constraint. The only genuinely new architectural question is the optional `sysmon` panel discussed separately (local CPU/GPU/mem), which is additive and unrelated to this token-usage backlog.

---

## 6. Sources (accessed 2026-06-15)

- ccusage: https://ccusage.com/ , https://ccusage.com/guide/ , https://ccusage.com/guide/blocks-reports , https://ccusage.com/guide/statusline
- Copilot Premium Usage Monitor: https://github.com/Fail-Safe/CopilotPremiumUsageMonitor
- Tokenlint: https://marketplace.visualstudio.com/items?itemName=tokenlint.tokenlint-vscode
- Cursor usage: https://cursor.com/help/models-and-usage/usage-limits
- GitHub Copilot metrics: https://docs.github.com/en/copilot/concepts/copilot-usage-metrics/copilot-metrics , https://github.com/microsoft/copilot-metrics-dashboard
- OpenAI usage: https://help.openai.com/en/articles/10478918-api-usage-dashboard
- Anthropic console: https://support.anthropic.com/en/articles/9534590-cost-and-usage-reporting-in-console
- OpenRouter: https://openrouter.ai/docs/guides/administration/usage-accounting
- Helicone: https://docs.helicone.ai/guides/cookbooks/cost-tracking
- Langfuse: https://langfuse.com/docs/observability/features/token-and-cost-tracking
- LangSmith: https://docs.langchain.com/langsmith/cost-tracking
- Portkey: https://portkey.ai/docs/product/observability/cost-management
- Datadog LLM Obs: https://docs.datadoghq.com/llm_observability/monitoring/cost/
- Arize Phoenix: https://arize.com/docs/phoenix/tracing/how-to-tracing/cost-tracking
- Cloudflare AI Gateway: https://developers.cloudflare.com/ai-gateway/observability/analytics/
- LiteLLM: https://docs.litellm.ai/docs/proxy/cost_tracking
- macOS Stats (UX patterns): https://github.com/exelban/stats
