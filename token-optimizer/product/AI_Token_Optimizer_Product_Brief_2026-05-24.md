# AI Token Optimizer — Product Brief

**Date:** 2026-05-24 · **Revised:** 2026-05-25 · **Stage:** Pre-build — spike scoped, product gate-contingent
**Audience:** AITO's leadership (decision) and engineering (scope)
**Companion to:** `../planning/Roadmap.md`, `../design/AI_Token_Optimizer_Agent_Blueprint_v0.1.md`, `AI_Token_Optimizer_Prior_Art_Landscape_2026-05-21.md`, `../planning/AI_Token_Optimizer_Build_vs_Adopt_2026-05-21.md`, `../evaluation/AI_Token_Optimizer_Systems_Review_2026-05-21.md`, `../spike/SPIKE_PLAN.md`

> This is the stable "why" of the project — the problem, the market, the wedge, and the product decision. It does **not** carry the schedule: milestones, the decision gate, and sequencing live in `../planning/Roadmap.md`. The product is **conditional, not committed** — every product-side effort and savings figure is `[VERIFY]` until the spike produces measured numbers.

---

## Summary

AITO is evaluating a **local-first AI Token Optimizer** — a developer-desktop tool that compresses the context sent to AI coding assistants to cut token spend, with a measured guarantee that answer quality does not degrade.

The prior-art scan settled one thing hard: **the concept already ships.** Context Gateway (Compresr, reported YC-backed `[VERIFY]`), OmniRoute, and CCProxy already do proxy-plus-compression for coding agents, and the assistant vendors compress context natively for free (Claude Code Auto-Compact, on by default). Every component — interception proxy, compression, budgeting, telemetry — is commoditized or open-source. Building the v0.1 blueprint from scratch re-implements solved problems.

So this is not a build — it is a decision. The initiative is sequenced **spike → gate → conditional build**: a cheap measurement answers the one question no competitor's marketing can answer for AITO — how much does compression actually save on AITO's own coding usage, at what quality and latency cost — and only a clear result justifies a product.

**The decision this brief stages:** fund the spike now; commit to the product only at the gate.

---

## The problem

AI coding assistants — Claude Code, Cursor, Continue, Copilot — are now core to AITO's engineering, and their cost scales with tokens. Every request re-sends conversation history, tool output, and large system prompts; long sessions spend heavily on context that is largely redundant. There is no AITO baseline for this spend yet — establishing one is part of the spike.

The desired outcome: **lower token cost per developer, with no measurable loss of answer quality, and no source code leaving the machine.**

---

## Market reality — why this is hard

The prior-art scan is the uncomfortable centre of this brief:

| Capability | Status in the market |
|---|---|
| Proxy interception of coding-agent traffic | Shipping — Context Gateway, OmniRoute, CCProxy; a proven pattern |
| Context / prompt compression | Commodity — LLMLingua-2 (Microsoft, open-source) is free and mature |
| Budgets, virtual keys, spend tracking | Commodity gateway feature — LiteLLM, Portkey |
| Token telemetry / cost dashboards | A dozen tools — Langfuse, Helicone, LiteLLM |
| Native context management | **Free and built-in** — Claude Code Auto-Compact, on by default, improving each release |

The last row is the real threat: a third-party optimizer competes against a zero-cost, zero-install baseline that the assistant vendors improve without AITO. Anything built here has to beat that moving baseline, not a standstill.

---

## The wedge — what is actually unserved

Against that field, exactly three things are not already given away:

First, **local-first, zero-egress, single-developer.** Competitors are hosted gateways or self-hosted servers. A truly local tool — no new egress, source code never leaving the machine — is a genuine niche. It is also small, and it forfeits the team-pooled cache wins a server-side product gets for free.

Second, **IntelliJ parity.** Competitors are overwhelmingly CLI-agent proxies. A managed-install VS Code plus IntelliJ plugin pair is packaging differentiation — the underlying mechanism (a configurable base URL) is identical, so this is form factor, not capability.

Third, **the Fidelity Floor** — a *measured* no-degradation guarantee: a quality-regression loop that rolls back compression strategies which hurt answers. Few tools do this. It is the most genuine differentiator, and also the hardest part — the Module 5 systems review flagged it as not yet sound and requiring redesign.

The wedge is real but narrow. A product-grade build is justified only if the spike confirms savings are real **and** AITO decides it wants this niche as a product.

---

## What the product would be

If the gate clears Green, the product is Option B — build narrow: only the wedge, wrapping commodity parts rather than reinventing them. The architecture decisions are already locked in the v0.1 blueprint:

- **Bundled local sidecar** — a Go core plus Go MCP servers, supervised as `127.0.0.1` child processes.
- **Localhost loopback proxy** for interception — the established, low-risk mechanism (blueprint Section 11).
- **LLMLingua-2 reached through a Python compression sidecar** — the product is Go; compression stays Python because LLMLingua-2 is a Python library (language decided 2026-05-21).
- **Metadata-only agent egress** — raw source code is never sent to the agent's LLM; the agent receives `get_context_metadata` only (blueprint Section 5).
- **VS Code `.vsix` plus IntelliJ plugin**, manual local install.
- **The Fidelity Floor**, re-designed to resolve the Module 5 systems-review findings before it is relied on.

**Explicitly out of scope:** model routing, semantic caching, multi-user/team setup, and the broad v0.1 optimizer. Budget *enforcement* is a later one-line LiteLLM add, not a launch feature.

Effort if built `[VERIFY]`: ~3–5 engineer-months. The how and when are in `../planning/Roadmap.md`.

---

## Decision posture

Three rules govern how AITO spends against this opportunity:

The build-vs-adopt analysis ruled out building the v0.1 blueprint as written (Option A: ~6–9+ engineer-months `[VERIFY]`, most of it re-implementing commodity). **Do not build Option A.**

The core unknown — real token savings on AITO's own usage — is unmeasured. A 2–4 week spike answers it for a few engineer-weeks. **Do not start a product build before the spike has produced real numbers.**

A product build is justified only on two conditions together: the spike shows real savings with no quality regression, *and* AITO wants the niche as a product. The second is a leadership call, made independently of the savings number. **A Green gate needs both.**

---

## Success criteria

For the spike, success is a decisive gate verdict — the gate criteria in the roadmap are the metrics. For the product, if built: measured per-developer token-cost reduction against the spike-established baseline, the Fidelity Floor holding (quality regression staying within the gate's bound in ongoing use), and compression latency staying within the gate's p95 budget. No metric ships without a baseline, time window, and source.

---

## Risks

**LLMLingua-2 was trained on prose, not source code.** Compressing code blocks may degrade answers badly. This is the most likely cause of a Red gate — and surfacing it cheaply is the point of the spike's code-heavy A/B fixtures.

**Native auto-compact erodes the premise.** Claude Code and others compress context for free and improve each release. If the free baseline is already good, measured savings may not clear the gate.

**LiteLLM supply chain.** A 2026 LiteLLM security incident was reported `[VERIFY]`. The spike pins the image to a specific, advisory-checked tag; a product would need the same discipline.

**The Fidelity Floor is unsolved.** It is the differentiator and the hardest part; Module 5 flagged it as not yet sound. A build cannot ship it as designed without a redesign.

**The niche may be too small.** Local-first, single-developer, zero-egress is a real but narrow slice, and it forfeits team-pooled compression wins. Whether it is wanted as a product is a leadership call, not a spike output.

**No baseline exists yet.** AITO has no measured current token spend — every savings claim is unverified until the spike establishes one.

---

## Non-goals

This project will not build the broad v0.1 optimizer, will not add model routing or semantic caching, will not target team or multi-tenant deployment, and will not start a product build before the spike has measured real savings. Each is a deliberate exclusion, not an oversight.

---

## Related documents

- `../planning/Roadmap.md` — the durable plan: milestones, the decision gate, sequencing
- `../design/AI_Token_Optimizer_Agent_Blueprint_v0.1.md` — the Module 8 blueprint and locked architecture decisions
- `../evaluation/AI_Token_Optimizer_Systems_Review_2026-05-21.md` — Module 5 conformance review; the findings a build must fold in
- `AI_Token_Optimizer_Prior_Art_Landscape_2026-05-21.md` — the prior-art scan behind the market-reality section
- `../planning/AI_Token_Optimizer_Build_vs_Adopt_2026-05-21.md` — the four options and the spike-then-gate recommendation
- `../spike/SPIKE_PLAN.md` and `../spike/README.md` — the Phase 1 runbook and runnable kit
- `AI_Token_Optimizer_Product_Brief_Infographic.html` — the one-page visual summary of this brief
