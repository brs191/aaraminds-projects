# AI Token Optimizer — Prior-Art Landscape

**Date:** 2026-05-21 · **Revised:** 2026-05-26 (Copilot June-2026 billing change + VS Code 1.118 token-efficiency work folded in) · **Author:** AITO product research · **Companion to:** `../design/AI_Token_Optimizer_Agent_Blueprint_v0.1.md`
**Method:** Web scan, broad-by-category. Vendor performance numbers are reported as vendor claims and marked `[VERIFY]` where not independently confirmed.

---

## Headline finding — read this first

**The AI Token Optimizer, as scoped in the v0.1 blueprint, is not a novel concept. The core product already exists and is shipping — including at least one YC-backed company.**

`Context Gateway` by **Compresr** (reported YC-backed `[VERIFY]`) is a proxy that sits between coding agents (Claude Code, Codex, OpenClaw) and the LLM API and compresses conversation history and tool output to cut token spend and latency — which is, almost line for line, the blueprint's value proposition. It is not alone: `OmniRoute` and `CCProxy` do the same thing, and the assistants themselves (Claude Code's built-in Auto-Compact) now compress context natively and by default.

This does not kill the idea, but it changes the decision. The honest framing: every individual capability in the blueprint — interception proxy, context compression, budgeting, telemetry, model routing — is **commoditized, open-source, or natively absorbed by the assistant vendors**. A from-scratch build of the blueprint as written would be re-implementing solved problems. The remaining question is whether AITO is targeting a genuinely unserved slice (Section: Gap Analysis) or should adopt-and-compose instead of build.

The good news buried in this: the interception proxy the blueprint called its "single largest delivery risk" is a **proven, well-trodden pattern** — three shipping products use exactly it. The risk is lower than the blueprint feared. The novelty is also lower.

### Update 2026-05-26 — the competitive picture sharpened in two ways since the original scan

**1. Copilot moves to usage-based billing on 1 June 2026.** GitHub announced the transition from premium-request counts to a token-based AI Credits model: usage is calculated against input, output, and cached tokens at the listed API rates, then converted to credits at $0.01 per credit. Copilot Pro carries $10/month in AI Credits, Pro+ carries $39, Business stays $19/user/month with $19 in credits. Code completions and Next Edit suggestions remain free of credit consumption. **What it means for the project:** the "AI coding-assistant spend scales with tokens" framing in the brief stops being abstract for Copilot users — they now see token cost directly. The token-optimizer's premise is correspondingly more concrete for any AITO developer on Copilot.

**2. VS Code is actively shipping native token-efficiency work.** VS Code 1.118 (April 2026) introduced a tool-search feature that splits agent toolsets into ~30 core tools plus a deferred set, claimed to deliver **up to 20% token savings** `[VERIFY]`, alongside optimised conversation-summarisation logic and an endpoint-aware token-budget validator. A community feature request — "Add a Context Compression Toggle to Save Token Usage in Copilot Chat" — is open against the Copilot repo, indicating developer demand but **no shipped user-facing third-party-style compression toggle yet**. An open bug also tracks **agent-mode over-summarisation creating loops and context loss** — concretely the failure mode the Token Optimizer's "Fidelity Floor" was designed to prevent.

**What this changes about the recommendation:** the prior-art scan's original "native auto-compact is the single biggest threat" finding is now sharper on both sides. The competitive bar rises (Microsoft is moving fast and is the platform owner, billing by token *and* optimising tokens natively). The differentiation also sharpens — Microsoft's native effort is opaque, server-side, and visibly experiencing the same quality-regression failure mode the Token Optimizer's Fidelity Floor is designed to address. **The M1 gate thresholds must be calibrated against current Copilot / VS Code 1.118 behaviour, not against a naive baseline**, or a Green verdict could be measuring a gap that Microsoft closes for free in the next release.

---

## Category 1 — Direct competitors: coding-agent context-compression proxies

This is the blueprint's exact niche. These products intercept a coding agent's traffic and compress it.

| Product | What it is | Notes |
|---|---|---|
| **Context Gateway** (Compresr) | Proxy between coding agents and LLM APIs; compresses conversation history + tool output while "preserving important context." Targets Claude Code, Codex, OpenClaw. "<1 minute setup." | The closest match to the blueprint. Reported YC-backed `[VERIFY]`. Hosted-gateway model. |
| **OmniRoute** | Open-source AI gateway; "RTK+Caveman stacked compression saves 15–95% eligible tokens per request" `[VERIFY]`; connects Claude Code, Codex, Cursor, Cline, Copilot; 160+ providers, multi-provider fallback, MCP/A2A. | Open-source, broad. Compression + routing + multimodal in one. |
| **CCProxy** | AI request proxy for Claude Code; multi-provider access (100+ models via OpenRouter), "zero configuration changes." | Routing/cost-arbitrage focus more than compression. |
| **Bifrost** | High-performance open-source AI gateway written in **Go**; unifies 20+ providers behind one OpenAI-compatible API; cost visibility as a first-class feature. | General-purpose gateway; relevant as a Go reference and as a telemetry baseline. |

**Implication:** the blueprint's proxy + compression core is already a product category with open-source and funded entrants. Building it again is a "me too" unless the form factor or guarantees differ materially.

---

## Category 2 — Prompt / context compression engines

The algorithmic core. The blueprint should treat compression as a dependency, not an invention.

- **LLMLingua / LongLLMLingua / LLMLingua-2** (Microsoft Research, open-source, EMNLP'23 / ACL'24). Uses a small model to identify and drop low-value tokens. Claims up to **20× compression with ~1.5-point accuracy drop** `[VERIFY]`; production workloads more typically see 4–10×. **LLMLingua-2** is task-agnostic, BERT-level encoder, 3–6× faster than v1, better on out-of-domain data. Model-agnostic (works ahead of GPT, Claude, Mistral, etc.).
- **Context-engineering toolset** — a March 2026 landscape mapped 15+ context-engineering tools including RTK, Headroom, LLMLingua, Edgee, and Portkey. The category ("compress, optimize, monitor LLM context") is now a recognized space with its own tooling map.

**Implication:** LLMLingua-2 is free, mature, and exactly the compression engine the blueprint needs. The blueprint's "Deterministic Optimizer Core" should wrap an existing engine, not build a new compressor.

---

## Category 3 — LLM gateways / proxies (caching, routing, budgets)

General-purpose gateways. They already deliver the budgeting and telemetry the blueprint scopes in.

- **LiteLLM** — open-source, self-hosted; virtual keys with **configurable per-team/service budget limits**, spend tracked against real provider cost; Redis-based caching. (Reported security compromise in 2026 `[VERIFY]` — see Source list.)
- **Portkey** — governance/guardrails first; **semantic caching** (fuzzy match of similar prompts) is its most defensible feature; PII filtering at the gateway.
- **OpenRouter** — 300+ models behind one API; Auto Router; prompt-caching pass-through.
- **Helicone** — drop-in observability/analytics for existing deployments. (Reported acquired in 2026 `[VERIFY]`.)
- **Cloudflare AI Gateway** — edge/geographic caching at scale.
- **Kong AI Gateway**, **agentgateway** — enterprise API-gateway vendors with LLM cost-tracking modules.

**Implication:** budgeting, virtual keys, spend tracking, and caching are commodity gateway features. The blueprint's "Budgeting & telemetry" pillar is fully commoditized.

---

## Category 4 — Model routing

The optimization lever the blueprint explicitly scoped *out* — but worth knowing it is also solved.

- **RouteLLM** (lm-sys, open-source) — router framework; drop-in OpenAI-client replacement; trained routers; vendor claim **up to 85% cost reduction at ~95% GPT-4 performance** `[VERIFY]`.
- **NotDiamond** — model router that picks the best LLM per query; powers OpenRouter's Auto Router (33-model curated pool).
- **OpenRouter Auto Router** — routing with no fee beyond the selected model's rate.

**Implication:** if model routing ever re-enters scope (blueprint Section 4 deviation note), RouteLLM is the off-the-shelf answer.

---

## Category 5 — Semantic caching

- **GPTCache** — earliest dedicated open-source semantic-caching library; Python; plugs into LangChain / OpenAI SDK.
- **Portkey semantic cache**, **Cloudflare** geographic cache — gateway-integrated caching.

**Implication:** caching (the blueprint's other scoped-out lever) is also off-the-shelf.

---

## Category 6 — Native context management inside the assistants themselves

This is the most strategically important competitor, because it is free, built-in, and improving without AITO.

- **Claude Code Auto-Compact** — Anthropic's built-in context compression; **on by default**; summarises/compresses conversation history near the context limit. Anthropic improved its preservation logic (Mar 2025) and trigger timing (late 2025); `CLAUDE_AUTOCOMPACT_PCT_OVERRIDE` lets users tune it.
- **VS Code Prompt TSX** — declares a prompt as a priority-tagged component tree; **automatically prunes the lowest-priority elements** when over token budget. Built into the VS Code AI extension API.
- **VS Code 1.118 token-efficiency work** (April 2026) — agent toolset split into a ~30-tool core plus a deferred set, claimed up to **20% token savings** `[VERIFY]`, optimised conversation-summarisation logic, endpoint-aware token-budget validation. Ships ahead of the 1 June 2026 Copilot billing change.
- **Copilot usage-based billing** (effective 1 June 2026) — Copilot Pro/Pro+/Business move to AI Credits priced against token usage at API rates ($0.01/credit). This **makes per-token cost visible to the developer** — completing the economic loop that gives a third-party optimizer something measurable to save against.
- **Provider prompt caching** (Anthropic / OpenAI) — cached prompt tokens are billed at a steep discount automatically.
- **An open community feature request** — "Add a Context Compression Toggle to Save Token Usage in Copilot Chat" (`microsoft/vscode#284712`) — confirms developer demand and confirms no shipped user-facing third-party-style compression toggle exists yet.
- **An open agent-mode over-summarisation bug** — Copilot's native summariser produces loops and context loss under certain conditions (`microsoft/vscode-copilot-release#11966`). This is the failure mode the Token Optimizer's **Fidelity Floor** is specifically designed to detect and roll back.

**Implication (revised 2026-05-26):** the assistant vendors are compressing context themselves, for free, and getting better each release — and they have started billing by token. The third-party optimizer competes against a moving, zero-cost, zero-install baseline that is now well-incentivised to keep moving. This remains the single biggest threat to the blueprint's premise, and it has sharpened: the M1 gate's "savings ≥ X%" verdict must be measured **against current Copilot / VS Code 1.118 behaviour**, or it risks measuring a gap Microsoft closes for free within a release.

---

## Category 7 — Token telemetry / cost dashboards

- **Langfuse, Braintrust, Traceloop, OpenObserve, Helicone**, plus **LiteLLM** spend tracking — real-time dashboards, per-user / per-feature / per-team cost attribution, budget alerts.

**Implication:** thoroughly commoditized. There is no telemetry product to invent here.

---

## Category 8 — VS Code proxy / interception extensions

Confirms the blueprint's interception approach is standard, and that the plumbing already exists.

- **LM Proxy** — exposes VS Code Copilot via OpenAI/Anthropic/Claude-Code-compatible REST APIs.
- **VS Code Copilot Proxy** — proxies external agents through VS Code Copilot models.
- **LLM Proxy** (Alorse) — lets BYOK editors (Cursor, Continue) connect to any OpenAI-compatible LLM via aliasing.
- **Microsoft Dev Proxy** — simulates token-based throttling, captures AI telemetry; ships with a VS Code toolkit extension.

**Implication:** the localhost-proxy + configurable-base-URL pattern the blueprint settled on is the established interception mechanism for VS Code. The blueprint's "largest delivery risk" is a solved problem.

---

## Gap Analysis — what (if anything) is still unserved

Against the crowded field above, the blueprint retains only a thin band of genuine differentiation:

1. **Local-first, zero-egress, single-developer desktop tool.** Almost every product above is a hosted gateway or a self-hosted *server*. A truly local, no-new-egress, single-developer optimizer is a real niche — but a small one, and it trades away the easiest compression wins (a local tool can't pool cache across a team).
2. **IDE-plugin form factor with IntelliJ parity.** Competitors are overwhelmingly CLI-agent proxies (point Claude Code / Codex at an endpoint). A true VS Code `.vsix` + IntelliJ plugin with managed install is less common — though functionally a configurable-base-URL proxy is the same thing, so this is packaging differentiation, not capability differentiation.
3. **The "Fidelity Floor" — measured no-degradation guarantee.** Most compression tools simply compress; few run a measured quality-regression feedback loop that rolls back strategies that hurt answers. This is the blueprint's most genuine potential differentiator — and also its hardest, unsolved-by-others part (and the one Module 5 flagged as not yet sound).

What is **not** differentiated and should not be built from scratch: the compression engine (use LLMLingua-2), the proxy mechanics (proven by Context Gateway / OmniRoute / CCProxy), telemetry/dashboards (a dozen tools), model routing (RouteLLM), and caching (GPTCache / Portkey).

---

## Implications for the blueprint

The prior-art scan does what a prior-art scan is for: it surfaces a serious problem with the premise *before* build. Three honest paths:

- **Adopt-and-compose, don't build.** The fastest route to the blueprint's outcome is to compose existing parts — LLMLingua-2 for compression, LiteLLM or an existing proxy for the gateway/budgets — or simply adopt Context Gateway / OmniRoute. This likely delivers 80% of the value for ~10% of the effort, and is worth seriously costing out before any build decision.
- **Narrow to the unserved slice.** If AITO builds, it must build only what the field does not already give away: local-first + zero-egress + IntelliJ-first + the measured Fidelity Floor. That is a defensible *niche* product, not the broad optimizer v0.1 describes. It needs the blueprint re-scoped around that wedge.
- **Reconsider the use case.** If neither the niche nor the compose-path is compelling, the Token Optimizer may not be the right first product — the brainstorming options (a skill-leveraging internal agent) remain open.

Recommended next step: a short build-vs-adopt comparison — effort and cost of composing LLMLingua-2 + LiteLLM (or adopting Context Gateway) versus building the v0.1 blueprint — so the decision is made on evidence, not on sunk design effort.

---

## Sources

- [LLMLingua — Microsoft Research blog](https://www.microsoft.com/en-us/research/blog/llmlingua-innovating-llm-efficiency-with-prompt-compression/)
- [microsoft/LLMLingua — GitHub](https://github.com/microsoft/LLMLingua)
- [LLMLingua paper (arXiv 2310.05736)](https://arxiv.org/pdf/2310.05736)
- [LongLLMLingua paper (arXiv 2310.06839)](https://arxiv.org/pdf/2310.06839)
- [Context Engineering Tools 2026](https://cc.bruniaux.com/context-engineering/)
- [LLMLingua 2026 — TokenMix blog](https://tokenmix.ai/blog/llmlingua-prompt-compression-2026)
- [Context Gateway — Product Hunt](https://www.producthunt.com/products/context-gateway)
- [Context Gateway — MF8.BIZ](https://www.mf8.biz/en/product/context-gateway)
- [OmniRoute — GitHub](https://github.com/diegosouzapw/OmniRoute)
- [CCProxy — AI Request Proxy for Claude Code](https://ccproxy.orchestre.dev/)
- [Best AI Gateway to Manage Claude Code Cost in 2026 — Maxim](https://www.getmaxim.ai/articles/best-ai-gateway-to-manage-claude-code-cost-in-2026/)
- [Portkey vs LiteLLM vs OpenRouter — PkgPulse](https://www.pkgpulse.com/guides/portkey-vs-litellm-vs-openrouter-llm-gateway-2026)
- [The LLM proxy landscape in 2026 (Helicone acquired, LiteLLM compromised)](https://dev.to/stockyarddev/the-llm-proxy-landscape-in-2026-helicone-acquired-litellm-compromised-and-whats-next-3oon)
- [Top 5 LLM Gateways in 2026 — DEV](https://dev.to/varshithvhegde/top-5-llm-gateways-in-2026-a-deep-dive-comparison-for-production-teams-34d2)
- [lm-sys/RouteLLM — GitHub](https://github.com/lm-sys/routellm)
- [Not-Diamond/awesome-ai-model-routing — GitHub](https://github.com/Not-Diamond/awesome-ai-model-routing)
- [OpenRouter prompt caching docs](https://openrouter.ai/docs/guides/best-practices/prompt-caching)
- [Best Open Source Semantic Caching Tools — DEV](https://dev.to/debmckinney/best-open-source-semantic-caching-tools-for-smart-llm-routing-3bna)
- [What Is Auto Compact in Claude Code — CometAPI](https://www.cometapi.com/what-is-auto-compact-in-claude-code/)
- [Claude Code Compaction explained](https://okhlopkov.com/claude-code-compaction-explained/)
- [AI extensibility in VS Code — VS Code docs](https://code.visualstudio.com/api/extension-guides/ai/ai-extensibility-overview)
- [LM Proxy — VS Code Marketplace](https://marketplace.visualstudio.com/items?itemName=ryonakae.vscode-lm-proxy)
- [LLM Proxy — VS Code Marketplace](https://marketplace.visualstudio.com/items?itemName=Alorse.llm-proxy)
- [Dev Proxy v2.0 — Microsoft 365 Developer Blog](https://devblogs.microsoft.com/microsoft365dev/dev-proxy-v2-0-with-improved-ai-telemetry-and-small-breaking-changes/)
- [The Real Cost of AI Coding in 2026 — Morph](https://www.morphllm.com/ai-coding-costs)
- [Best tools for monitoring LLM costs 2026 — Braintrust](https://www.braintrust.dev/articles/best-llm-monitoring-tools-2026)
- [Token Optimisation 101 — DEV Community](https://dev.to/stevengonsalvez/token-optimisation-101-stop-burning-money-on-ai-coding-agents-4mce)

**Added in the 2026-05-26 revision:**

- [GitHub Copilot is moving to usage-based billing — The GitHub Blog](https://github.blog/news-insights/company-news/github-copilot-is-moving-to-usage-based-billing/)
- [Models and pricing for GitHub Copilot — GitHub Docs](https://docs.github.com/en/copilot/reference/copilot-billing/models-and-pricing)
- [VS Code Curbs Token Use Ahead of Copilot's Usage-Based Billing Switch — Visual Studio Magazine](https://visualstudiomagazine.com/articles/2026/04/30/vs-code-curbs-token-use-ahead-of-copilots-controversial-usage-based-billing-switch.aspx)
- [Feature Request: Add 'Context Compression' Toggle to Copilot Chat — microsoft/vscode #284712](https://github.com/microsoft/vscode/issues/284712)
- [Over-summarising in Copilot Agent mode creates loops and loss of context — microsoft/vscode-copilot-release #11966](https://github.com/microsoft/vscode-copilot-release/issues/11966)
- [Optimise conversation summarisation logic and caching — microsoft/vscode-copilot-chat #1846](https://github.com/microsoft/vscode-copilot-chat/pull/1846)
