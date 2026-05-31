# M0-lite — Compressed measurement spike

**Owner:** Raja  ·  **Status:** Intermission as of 2026-05-31 — PoC window (5/28–5/29) elapsed, outcome unrecorded; Copilot routing-mode validation still 🔴 BLOCKING; measurement window 6/8–6/12  ·  **Replaces:** `M0-Spike.md` (deprecated 2026-05-27)
**Source:** `../../tracking/milestones/M1-Decision-Gate.md` (GREEN verdict, conditional on this measurement)  ·  `../../planning/M0-lite_Cohort_Recruitment.md`  ·  `../../spike/SPIKE_PLAN.md`

## Why this exists (and why it replaced M0)

The original M0 was a 2-4 week measurement spike sized for a $75k decision. At locked C=$5k (M2-lite scope), the gate decision is dominated by inputs, not by the spike's R measurement — so the spike's only job is to validate that R ≥ 10% incremental holds. A 2-4 week box is overkill for that question. M0-lite is the right-sized measurement.

## Protocol

**Phase 1 — 2-day PoC** · Thu 2026-05-28 → Fri 2026-05-29
- 7-dev cohort points coding agents at localhost LiteLLM proxy.
- Measure operational survival: do `docker-compose`, proxy, compression hook, and metrics pipeline survive contact with real machines?
- R signal at this point is **informative, not gating** — sample is too thin for a verdict.
- **PoC gate (operational):** ≥ 5 of 7 devs have the kit running with metrics being written by end of day Fri.

**Phase 2 — Intermission** · Sat 2026-05-30 → Sun 2026-06-07 (9 days)
- Fix any setup issues from PoC.
- Re-pin LiteLLM image if a fresher patch is released.
- Refine the daily check-in format based on what the PoC surfaced.

**Phase 3 — Full measurement** · Mon 2026-06-08 → Fri 2026-06-12 (5 working days)
- Same 7-dev cohort, full week of real coding work routed through proxy.
- End-of-week: aggregate `metrics/requests.jsonl` via `../../spike/summarize.py`.
- Run `../../spike/measure.py` over a representative fixture set built from week-1 prompts.
- Human quality review on code-heavy A/B pairs (the calibrated 3% regression cap from `M1-Decision-Gate.md`).
- **Measurement gate (statistical):** R ≥ **10%** median incremental token reduction on code-heavy prompts, vs. assistant's native baseline.

## Cohort

7 committed engineers (locked 2026-05-27): Namratha, Bharat, Mounika, Pranitha, Karthick, Rohit, Dhyan. Wait list of 6: Chansi, Pritam, Shankar, Saurabh, Ranjith, Shyla. Detailed table in `../../planning/M0-lite_Cohort_Recruitment.md`.

Backend/frontend role mapping per individual is TBD — to be filled in before PoC kickoff so the post-mortem can verify the composition matches the team's 70/30 split.

## Deliverables

- `../../spike/metrics/requests.jsonl` — combined real-usage data from PoC + measurement.
- `../../spike/results/ab_results.jsonl` — A/B compressed-vs-raw pairs on the fixture set.
- Aggregated R, quality regression %, and latency p95 from `summarize.py`.
- Per-dev exit interview notes.
- A written R verdict — pass/fail against the 10% threshold.

## Verdict outcomes

| R-lite | Quality | Action |
|---|---|---|
| ≥ 10% on code-heavy | ≤ 3% regression on code-heavy, ≤ 5% overall | **PASS** — green-light M2-lite build (task #17). M1 verdict stands as GREEN. |
| < 10% | — | **FAIL** — cancel M2-lite. M1 verdict reverts; write post-mortem in `../../evaluation/`. |
| ≥ 10% on code-heavy but quality regression > thresholds | — | **FAIL on quality** — cancel M2-lite; document Fidelity Floor failure mode for future reference. |

## Pre-flight tasks

- [ ] **🔴 BLOCKING — Validate Copilot routing mode for every cohort dev** (see `../../planning/validate_with_copilot.md`). All 7 devs run Copilot + Claude Opus 4.6; if any are in server-routed Mode A the proxy sees no traffic *and* the `S = $100/dev/mo` input behind the GREEN verdict is wrong. Settle before the 6/8 measurement window — it should have gated the 5/28 PoC. Mode A → STOP and re-open M1; Mode B → swap cohort to Claude Code / Cursor on `localhost:4000` and document the methodological caveat; Mode C → proceed.
- [x] Pin LiteLLM image (task #1 — DONE 2026-05-27 → `v1.83.14-stable.patch.3`)
- [x] Secure cohort (task #2 — 7 names committed 2026-05-27; per-dev role mapping still TBD)
- [ ] Send setup pack to each cohort member (see `../../planning/M0-lite_Cohort_Recruitment.md`)
- [ ] Per-dev confirmation: VS Code primary on the PoC + measurement machines
- [ ] Per-dev confirmation: code-heavy work during BOTH date windows (not pure CSS / docs / config)
- [ ] Per-dev role mapping: backend or frontend
- [ ] Daily check-in script + exit interview script drafted

## PoC tasks (Thu 5/28 – Fri 5/29)

- [ ] Day 1: each dev runs `docker-compose up --build`, configures agent to point at localhost:4000
- [ ] Day 1: confirm `metrics/requests.jsonl` is being written on each dev's machine
- [ ] Day 1 end: brief check-in — any setup issues?
- [ ] Day 2: real coding work through proxy; observe stability
- [ ] Day 2 end: pull `metrics/requests.jsonl` from each dev; preliminary R look (informative, not gating)
- [ ] PoC gate evaluation: did ≥ 5 of 7 devs run the kit with metrics? If yes → green-light intermission and Phase 3.

## Measurement tasks (Mon 6/8 – Fri 6/12)

- [ ] Day 1: re-confirm kit running on all 7 machines after intermission
- [ ] Days 1-5: real coding work routed through proxy; minimal touch
- [ ] Day 5: pull `metrics/requests.jsonl` from all devs
- [ ] Day 5: build representative fixture set from week's prompts
- [ ] Day 5 / weekend: run `measure.py` over fixtures → `results/ab_results.jsonl`
- [ ] Day 5 / weekend: human quality review on code-heavy A/B pairs
- [ ] Day 5 / weekend: aggregate via `summarize.py`; write verdict against R ≥ 10% threshold
- [ ] Hand