# AI Token Optimizer — Roadmap

**Date:** 2026-05-25 · **Owner:** Raja · **Horizon:** spike now; the build is gate-contingent
**Companion to:** `AI_Token_Optimizer_Build_vs_Adopt_2026-05-21.md`, `../product/AI_Token_Optimizer_Product_Brief_2026-05-24.md`

> This is the durable plan — it changes only when the strategy changes. The volatile execution layer is `../tracking/`: `Status.md` is the live dashboard and `../tracking/milestones/` holds the working checklist for each milestone. Gates, not task counts, govern progress.

---

## Shape of the plan

The initiative is sequenced **spike → gate → conditional build**. Three milestones, each ending in a kill/continue gate. M2 is conditional — it exists only if M1 returns Green.

```
M0 — Spike          M1 — Decision gate         M2 — Conditional build
measure real    →   Green / Amber / Red    →   the narrow product
savings             the formal kill point      only on Green
2–4 weeks           days                       ~3–5 eng-months [VERIFY]
```

The reasoning behind this sequencing — and why building the v0.1 blueprint outright (Option A) was rejected — is in `AI_Token_Optimizer_Build_vs_Adopt_2026-05-21.md`.

---

## Milestones

| Milestone | Horizon | Delivers | Gate — kill / continue |
|---|---|---|---|
| **M0 — Spike** | 2–4 weeks, now | Measured token reduction, answer-quality, and latency on real AITO usage | Decision-ready: real-usage and A/B data collected, quality reviewed → continue to M1 |
| **M1 — Decision gate** | Days, after M0 | A documented Green / Amber / Red verdict | The formal kill point. Green → open M2. Amber → keep as internal tooling, stop. Red → adopt off-the-shelf or shelve. |
| **M2 — Conditional build** | ~3–5 eng-months `[VERIFY]` | The narrow Option B product | Set on entry; at minimum the product holds the Fidelity Floor and the latency budget |

---

## M0 — Spike

**Goal.** Retire the biggest open question before any build: how much does context compression actually save on AITO's own coding-assistant usage, and at what quality and latency cost? Replace every `[VERIFY]` savings guess with measured numbers.

**Delivers.** A LiteLLM proxy plus an LLMLingua-2 compression hook running locally; AITO coding agents routed through it for daily work; real-usage metrics (`metrics/requests.jsonl`); A/B compressed-vs-raw results (`results/ab_results.jsonl`); aggregated numbers and a human quality review.

**Shape of the work.** Time-boxed at 2–4 weeks: Week 1 stand up, Week 2 real-usage data, Week 3 A/B and quality review, Week 4 analyse. End early if results are decisive. The full runbook and runnable kit are in `../spike/SPIKE_PLAN.md` and `../spike/`.

**Gate — continue to M1.** Enough evidence to make the M1 call without guessing: real-usage metrics over genuine daily work, an A/B run including code-heavy prompts, a human quality review, and no blocking gap that forces the decision to lean on a `[VERIFY]`.

**Pre-work.** Calibrate the M1 thresholds below to AITO's token economics *before* Week 1.

---

## M1 — Decision gate

**Goal.** Turn the M0 evidence into one decision. This is the formal kill/continue point for the whole initiative.

**Calibrated 2026-05-26** against the post-VS Code-1.118 baseline (see `../product/AI_Token_Optimizer_Prior_Art_Landscape_2026-05-21.md` 2026-05-26 revision). Canonical source for the calibration rationale: `../tracking/milestones/M1-Decision-Gate.md`.

| Outcome | Criteria | Action |
|---|---|---|
| **Green** | Median input-token reduction **≥ 20% incremental** over the assistant's native baseline · quality regression **≤ 5%** of A/B pairs (**≤ 3%** code-heavy) · latency overhead **< 300 ms p95** chat, **< 100 ms p95** completions · optimizer overhead **≤ 5%** of tokens saved · payback ≤ 12 months on AITO economics `[VERIFY-economics]` · niche wanted as a product | Open M2 — build the narrow product |
| **Amber** | ≥ 10% reduction with no quality regression but either payback > 12 months OR niche not wanted as a product OR latency 300–500 ms / 100–150 ms | Keep the LiteLLM + LLMLingua-2 setup as internal tooling; stop |
| **Red** | < 10% reduction OR > 5% quality regression OR > 3% code-heavy regression OR > 500 ms / > 150 ms latency OR optimizer overhead > 5% of savings | Adopt an off-the-shelf product for internal use, or shelve the initiative |

A Green verdict needs the answer-quality criterion as well as the token number — a human judgement on the A/B output, not a single metric — **and** a leadership decision that AITO wants the niche as a product. Both, or it is not Green.

**Gate — verdict recorded.** M1 is done when a written verdict exists naming the outcome, the measured numbers behind it, and the rationale.

---

## M2 — Conditional build

**Exists only on a Green M1.** If M1 is Amber or Red, M2 never opens.

**Goal.** Build Option B — the narrow product: only the unserved wedge (local-first, zero-egress, IntelliJ parity, the measured Fidelity Floor), wrapping commodity parts rather than reinventing them. The product definition is in `../product/AI_Token_Optimizer_Product_Brief_2026-05-24.md`; the locked architecture is in `../design/AI_Token_Optimizer_Agent_Blueprint_v0.1.md`.

**Entry pre-work.** Re-scope the v0.1 blueprint tightly around the wedge (bump to v0.2), fold in the four Required Fixes from the Module 5 systems review, and redesign the Fidelity Floor to resolve the systems-review findings before relying on it.

**Internal phasing.** Set on entry. The provisional basis is the v0.1 blueprint's phased build — gateway and Go MCP core first, then the Python compression sidecar and the Fidelity Floor, then the VS Code plugin, then IntelliJ. Each phase carries its own conformance re-check against the systems-review baseline.

**Gate.** Defined when M2 opens. At minimum: the product holds the Fidelity Floor (quality regression within the gate bound in ongoing use) and compression latency stays within the p95 budget, both measured against the spike-established baseline.

---

## Component → milestone mapping

Which milestone builds which part of the architecture. Note that nothing in M0 is product code — the spike is a disposable measurement rig.

| Component | Milestone | Notes |
|---|---|---|
| LiteLLM proxy (off-the-shelf) | M0 | Spike rig only — disposable |
| LLMLingua-2 compression hook (Python) | M0 | Spike rig; the product re-homes LLMLingua-2 in a Python sidecar |
| A/B measurement harness | M0 | Disposable measurement code |
| Go gateway + localhost loopback proxy | M2 | Product interception layer |
| Go MCP servers | M2 | Part of the bundled sidecar |
| Python compression sidecar (LLMLingua-2) | M2 | Product compression engine |
| Fidelity Floor (quality-regression loop) | M2 | Needs redesign first — Module 5 finding |
| VS Code `.vsix` + IntelliJ plugins | M2 | Product form factor |
| Budget enforcement | Post-M2 | A later one-line LiteLLM add, not a launch feature |

---

## Gate discipline

A milestone is done only when its **gate** passes — not when its tasks are all ticked. M1 is the formal kill point for the initiative: a non-Green verdict ends the build path. This roadmap deliberately spends a few engineer-weeks (M0) to decide whether a multi-month build (M2) is justified at all — sequencing spend against evidence, not against sunk design effort.

## What changes this roadmap

Only a strategy change: a different decision at M1, a re-scoping of the wedge, or a material shift in the market (for example, native auto-compact closing the gap entirely). Day-to-day execution state churns in `../tracking/` and does not touch this file.
