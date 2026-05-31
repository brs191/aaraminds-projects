# M0 — Spike (DEPRECATED)

> **DEPRECATED 2026-05-27.** Replaced by `M0-lite.md` after the 2026-05-27 GREEN verdict locked C = $5k (M2-lite scope). The 2-4 week M0 box was sized for a $75k decision; at $5k the gate decision is dominated by inputs and only needs a 2-day PoC + 5-day measurement to validate R ≥ 10% incremental. See `M0-lite.md` for the active plan. This file is preserved as historical record of the original sizing.

**Owner:** Raja (full-time, ~2 weeks)  ·  **Status:** **DEPRECATED — see `M0-lite.md`**  ·  **Original duration:** 2–4 week box; plan ~2 weeks, committed bound ~3 weeks at full-time allocation
**Runbook:** `../../spike/SPIKE_PLAN.md`  ·  **Kit:** `../../spike/README.md`

## Goal

Retire the biggest open question before any build: how much does context compression actually save on AITO's own coding-assistant usage, and at what quality and latency cost? Replace every `[VERIFY]` savings guess with measured numbers from real usage.

## Deliverables

- LiteLLM proxy plus the LLMLingua-2 compression hook running locally, with fail-open behaviour confirmed.
- AITO coding agents routed through the proxy for daily work.
- `metrics/requests.jsonl` — real-usage token-reduction and latency data.
- `results/ab_results.jsonl` — A/B compressed-vs-raw results on a representative fixture set.
- Aggregated metrics from `summarize.py`, plus a human quality review of the A/B answers.

## Gate — decision-ready

M0 is done when there is enough evidence to make the M1 call without guessing:

- Real-usage metrics collected over genuine daily work, not synthetic prompts.
- An A/B run across a fixture set that includes code-heavy prompts.
- Answer quality reviewed by a human, not just a token-count number.
- No blocking gap that would force the gate decision to lean on a `[VERIFY]`.

## Tasks

Week 1 — stand up

- [ ] Pin the LiteLLM image to a specific, advisory-checked tag (a 2026 supply-chain incident was reported)
- [ ] Run the kit, point one agent at the proxy, confirm requests flow and metrics are written
- [ ] Confirm compression fails open — a compression error passes the request through raw
- [ ] Tune `COMPRESSION_RATE` and the size threshold on a handful of prompts

Week 2 — real usage

- [ ] Route daily coding work (whole team, or several developers) through the proxy
- [ ] Build the fixture set from genuinely representative AITO prompts

Week 3 — A/B and quality

- [ ] Run `measure.py` over the fixture set
- [ ] Review compressed-vs-raw answers for quality regression, especially the code-heavy fixture
- [ ] Spot-check the worst token reductions

Week 4 — analyse

- [ ] Aggregate metrics with `summarize.py`
- [ ] Hand the evidence to M1

## Notes

Calibrate the M1 gate thresholds to AITO's economics **before** Week 1 — see