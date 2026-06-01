# Token Optimizer — Option C Spike Plan

**Date:** 2026-05-21 · **Type:** Time-boxed engineering spike · **Duration:** 2–4 weeks
**Companion to:** `../planning/AI_Token_Optimizer_Build_vs_Adopt_2026-05-21.md` (Option C)

---

## Objective

Retire the single biggest open question before any product build: **how much does context compression actually save on AITO's own coding-assistant usage, and at what quality and latency cost?**

Every number in the v0.1 blueprint that matters — the 25–45% token reduction, the <5% overhead, "no measurable quality degradation" — is currently `[VERIFY]`, i.e. guessed. This spike replaces those guesses with measured numbers from real usage, using only off-the-shelf parts (LiteLLM + LLMLingua-2), so the build-vs-adopt gate is decided on evidence.

This spike does **not** build a product. It builds a measurement rig.

---

## Scope

**In scope**
- Self-host LiteLLM as an OpenAI/Anthropic-compatible proxy in front of the team's LLM provider.
- Wrap LLMLingua-2 as a LiteLLM `async_pre_call_hook` that compresses eligible message content.
- Point AITO's own coding agents (Claude Code / Cursor / Continue / Copilot-via-proxy) at the proxy for daily work.
- Capture per-request metrics (token reduction, hook latency) and run a controlled A/B harness for answer-quality comparison.

**Out of scope**
- The bundled sidecar, Go MCP servers, the LangGraph agent, the IDE plugins — none of the v0.1 architecture.
- Budget *enforcement* (telemetry only; enforcement is a one-line LiteLLM feature added later if the gate passes).
- Model routing and caching.
- Any production hardening, multi-user setup, or polish.

---

## What gets measured

| Metric | How captured | Why it matters |
|---|---|---|
| **Input-token reduction** | `compression_hook.py` writes original vs compressed token counts per request to `metrics/requests.jsonl`; A/B harness confirms with provider `usage` data | The headline number — directly drives cost savings |
| **Answer-quality impact** | A/B harness sends each fixture through compressed + raw model aliases; both answers saved for human review and optional LLM-as-judge scoring | The Fidelity Floor question — savings are worthless if answers degrade |
| **Compression latency cost** | Hook records `hook_latency_ms` per request; A/B harness records end-to-end latency delta | Inline compression must not make the assistant feel slow |
| **Net savings after overhead** | Tokens saved × provider price, minus compression compute cost | Self-funding check — the optimizer must not cost more than it saves |
| **Compression failure rate** | Hook logs fail-open events (compression errored, request passed through raw) | Reliability signal for a production decision |

A/B is per-request: the proxy exposes two aliases for the same backend — a compressed one and a `-raw` one — so compression on/off is compared on identical prompts without restarts.

---

## Timeline

- **Week 1 — Stand up.** Run the kit (`README.md`), point one agent at the proxy, confirm requests flow and metrics are written. Tune `COMPRESSION_RATE` and the size threshold on a handful of prompts.
- **Week 2 — Real usage.** Whole team (or several developers) routes daily coding work through the proxy. `metrics/requests.jsonl` accumulates real-usage data. Build the fixture set from genuinely representative prompts (code-explain, refactor, debug, long-context).
- **Week 3 — A/B and quality.** Run `measure.py` over the fixture set. Review compressed-vs-raw answers for quality regression. Spot-check the worst reductions.
- **Week 4 (buffer) — Analyse and decide.** Aggregate metrics, write the gate decision, present.

If results are decisive earlier, end early — it is a time *box*, not a time *target*.

---

## Decision gate

> **⚠ Superseded by the calibrated gate (2026-05-26).** The thresholds below are the original `[VERIFY]` starting points. The **binding** gate — Green **≥ 20%** / Amber **≥ 10%** median reduction **incremental over the assistant's native baseline**, with a **≤ 3%** quality-regression cap on code-heavy fixtures — lives in `../tracking/milestones/M1-Decision-Gate.md` and is mirrored in `summarize.py`. Read those numbers, not the ones below, for any verdict. The values here are retained only as the historical pre-calibration baseline.

Thresholds are proposed starting points — adjust to AITO's economics before the spike starts, and treat them as `[VERIFY]`.

**Green — graduate to Option B (narrow product build).**
- Median input-token reduction **≥ 25%** on real usage, AND
- Answer-quality regression in **≤ 5%** of A/B pairs (no systematic degradation, especially on code-heavy prompts), AND
- Compression latency overhead **< 300 ms p95**, AND
- Net savings clearly positive after overhead, AND
- AITO wants the niche (local-first, zero-egress, IntelliJ, measured Fidelity Floor) as a product.
- → Re-scope the blueprint tightly around the wedge, fold in the Module 5 systems-review findings, then build — **Go for the gateway and MCP servers, with LLMLingua-2 reached through a Python compression sidecar** (language decided 2026-05-21; the spike stays Python as a disposable measurement rig).

**Amber — keep the Option C composition as internal tooling.**
- Savings are real (≥ 15%) but the niche is not wanted as a product, or quality/latency is acceptable-but-not-great.
- → Keep the LiteLLM + LLMLingua-2 setup running for the team. Do not start a product build.

**Red — adopt or drop.**
- Median reduction **< 15%**, or a measurable quality regression on normal coding prompts, or unacceptable latency.
- → Adopt an off-the-shelf product (Context Gateway / OmniRoute) for internal use, or shelve the initiative. The brainstormed alternative (a skill-leveraging internal agent) remains open.

---

## Known risks and how the spike handles them

- **LLMLingua-2 was trained on prose, not source code.** Compressing code blocks may degrade answers badly. The hook is conservative by default — it skips short messages and preserves the latest user message verbatim — and the A/B fixture set **must** include code-heavy prompts so the quality measurement catches this. This is the most likely cause of a Red outcome, and finding it now is the point.
- **Quality scoring is subjective.** LLM-as-judge is a starting signal, not a verdict; pair it with human spot-checks. Do not let a single number decide the gate.
- **Compression adds latency.** The hook runs LLMLingua-2 inference (a small model) per request. CPU is fine for a spike; the latency metric tells you if it is viable.
- **LiteLLM supply-chain.** A LiteLLM security incident was reported in 2026 `[VERIFY]`. Pin the image to a specific, advisory-checked tag before running (see `README.md`).
- **Compression must fail open.** The hook never breaks a request over a compression error — it logs and passes the request through raw. Verified in Week 1.
