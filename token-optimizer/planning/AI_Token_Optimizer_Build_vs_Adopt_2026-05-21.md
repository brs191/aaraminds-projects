# AI Token Optimizer — Build vs. Adopt

**Date:** 2026-05-21 · **Companion to:** `../product/AI_Token_Optimizer_Prior_Art_Landscape_2026-05-21.md`, `../design/AI_Token_Optimizer_Agent_Blueprint_v0.1.md`
**Purpose:** Decide how AITO should obtain the Token Optimizer outcome, given that the prior-art scan found the concept already shipping.

> Effort and cost figures are rough order-of-magnitude estimates marked `[VERIFY]`. They exist to compare options against each other, not as commitments. No AITO baseline exists yet — establishing one is itself part of the recommendation.

---

## The four options

Not a binary. The landscape scan exposed a spectrum from full custom build to pure adoption:

- **A — Build full.** Implement the v0.1 blueprint as written: bundled sidecar, Go core + Go MCP servers, LangGraph agent, VS Code + IntelliJ plugins, eval harness, Fidelity Floor.
- **B — Build narrow.** Build only the unserved wedge: local-first, zero-egress, IntelliJ-included, measured Fidelity Floor — but wrap LLMLingua-2 as the compression engine rather than inventing one, and drop the inline LLM agent.
- **C — Compose.** Integrate existing open-source parts behind thin glue: LLMLingua-2 for compression + self-hosted LiteLLM for the proxy, budgets, and telemetry. Minimal custom code.
- **D — Adopt.** Use a shipping product directly — Context Gateway or self-hosted OmniRoute — and configure it for AITO's agents.

---

## Comparison table

| Dimension | A — Build full | B — Build narrow | C — Compose | D — Adopt |
|---|---|---|---|---|
| **Effort** `[VERIFY]` | ~6–9+ eng-months | ~3–5 eng-months | ~2–6 eng-weeks | ~1–5 days |
| **Time to first value** | Months | Months | Weeks | Same day |
| **Eng cost** | Highest | High | Low | Minimal |
| **Differentiation / IP owned** | High in theory — but re-builds solved problems, so low in practice | Genuine: the niche wedge is the only unserved slice | Integration know-how only; no product IP | None |
| **Control & customization** | Total | High | Moderate (bounded by LiteLLM + LLMLingua-2 APIs) | Low — vendor roadmap |
| **Maintenance burden** | Heavy — full custom stack, two IDE plugins, agent | Moderate | Low — upstream maintains the hard parts | None (hosted D); low (self-host) |
| **Local-first / zero-egress** | Achievable | Achievable — core of the wedge | Achievable if both parts self-hosted | Context Gateway: no (hosted). OmniRoute: yes if self-hosted |
| **IntelliJ support** | Yes (built) | Yes — part of the wedge | DIY thin client | Not standard — CLI-agent oriented |
| **Fidelity Floor (measured no-degradation)** | Yes (designed) — but Module 5 flagged it not yet sound | Yes — the headline differentiator | Not out-of-box; custom add-on | Not offered |
| **Vendor / dependency risk** | Low external; high execution risk | Low external; moderate execution | Tied to LLMLingua-2 + LiteLLM health (note: LiteLLM had a reported 2026 security incident `[VERIFY]`) | High — single third party; hosted option adds data-egress risk |
| **Validates the core unknown** (real token savings on AITO usage) | Only after months of build | Only after months | **Yes — within weeks** | **Yes — within days** |
| **Strategic fit for AITO** | Weak — most effort spent re-implementing commodity | Strong *if* the niche is wanted as a product | Strong as a learning/decision step | Weak as a product; fine as internal tooling |

---

## Reading the table

**Option A is the weakest path.** It is the most expensive option and, because the prior-art scan showed every component is commoditized, most of that spend re-implements solved problems. The v0.1 blueprint should not be built as-is.

**The core unknown is the same for every option:** how much does compression actually save on AITO *own* coding-assistant usage? The blueprint's headline 25–45% is `[VERIFY]` — unmeasured. Options C and D answer that question in days-to-weeks; A and B only answer it after months of sunk build effort. Spending months to discover the premise was thin is the exact failure the prior-art scan exists to prevent.

**Differentiation lives only in Option B's wedge** — local-first, zero-egress, IntelliJ, measured Fidelity Floor. Everything else is a commodity. So a product-grade build is only justified if AITO specifically wants that niche, *and* the savings measurement confirms the value is real.

---

## Recommendation

**Run Option C as a 2–4 week time-boxed spike, then decide at a gate.**

1. **Spike (Option C).** Self-host LiteLLM as the proxy + budget/telemetry layer; wrap LLMLingua-2 for compression. Point AITO's own coding agents at it. Cost: a few engineer-weeks, mostly integration.
2. **Measure.** Capture the real numbers the blueprint marked `[VERIFY]`: actual token reduction, answer-quality impact, latency cost, net savings after overhead — on real AITO usage. This retires the single biggest open question.
3. **Decision gate.**
   - *Savings large + the niche wanted as a product* → graduate to **Option B**, re-scoped tightly around the wedge, with the Module 5 findings folded in.
   - *Savings large but no product ambition* → keep the Option C composition as **internal tooling**; stop.
   - *Savings modest* → **adopt Option D** for internal use, or drop the initiative — and the brainstacked alternatives (a skill-leveraging internal agent) remain open.

This sequences spend against evidence: a few weeks buys the measurement that decides whether a multi-month build is justified at all. **Do not start with Option A. Do not start a product build (B) before the spike (C) has produced real savings numbers.**

---

## Sources

- [Context Gateway — Product Hunt](https://www.producthunt.com/products/context-gateway)
- [OmniRoute — GitHub](https://github.com/diegosouzapw/OmniRoute)
- [microsoft/LLMLingua — GitHub](https://github.com/microsoft/LLMLingua)
- [LiteLLM spend tracking — docs](https://docs.litellm.ai/docs/proxy/cost_tracking)
- [The LLM proxy landscape in 2026 — DEV](https://dev.to/stockyarddev/the-llm-proxy-landscape-in-2026-helicone-acquired-litellm-compromised-and-whats-next-3oon)
