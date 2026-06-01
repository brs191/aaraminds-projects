# Token Optimizer — Status

**Updated:** 2026-05-31  ·  **Owner:** Raja  ·  **Active milestone:** M0-lite — intermission (PoC window 5/28–5/29 elapsed, outcome unrecorded; measurement 6/8–6/12)

Start here each working session. This dashboard rolls up the per-milestone files in `milestones/`.

## ⚠ Open blockers (added 2026-05-31) — resolve before the 6/8 measurement window

Two items gate a trustworthy M0-lite measurement and are **not yet recorded**:

1. **Copilot routing-mode validation — UNRESOLVED, highest risk.** Cohort screening (2026-05-27) found all 7 devs run GitHub Copilot + Claude Opus 4.6 as primary. If Copilot routes server-side (Mode A), the localhost proxy sees **zero** traffic *and* the `S = $100/dev/mo` per-token assumption behind the GREEN verdict is wrong — the economic math collapses. Settle per `../planning/validate_with_copilot.md` before 6/8; it should have gated the 5/28 PoC. **The verdict's real exposure is S, not the R it is officially conditioned on.**
2. **PoC outcome (5/28–5/29) — UNRECORDED.** The 2-day operational PoC window has elapsed; its gate result (≥ 5 of 7 devs running with metrics) is not captured below. Record it, or note that the PoC was deferred pending blocker #1.

## Milestones (post-2026-05-27 verdict)

| Milestone | What it produces | Gate | Owner | State |
|---|---|---|---|---|
| Pre-work | Pinned LiteLLM image · 7 dev pilot cohort secured for M0-lite | Both items committed | Raja | **DONE 2026-05-27** (LiteLLM pinned to v1.83.14-stable.patch.3; cohort = Namratha, Bharat, Mounika, Pranitha, Karthick, Rohit, Dhyan) |
| M0-lite | 2-day PoC (Thu 5/28 – Fri 5/29) + 5-day measurement (Mon 6/8 – Fri 6/12) — validate R ≥ 10% incremental on code-heavy prompts | R-lite ≥ 10% on real AITO usage at end of measurement week | Raja | Pre-flight — see `milestones/M0-lite.md` |
| M1 — Decision gate | Documented Green / Amber / Red verdict | Verdict recorded with locked inputs, math, and scope reconciliation | Raja | **GREEN — locked 2026-05-27 (conditional on M0-lite R-validation)** |
| M2-lite — Conditional build | The narrow $5k internal pilot product — VS Code only, manual install, one-time Fidelity Floor | Built to frozen scope, deployed to pilot D=50 | Raja | Pending M0-lite |
| Pilot rollout + monitor | 30-day observation of D=50 cohort against locked inputs | Pilot data matches spike measurement materially | Raja | Pending M2-lite |
| ~~M0 — Spike~~ | ~~2-4 week measurement~~ | — | — | **CANCELLED 2026-05-27** — gate decided on inputs at $5k C; replaced by M0-lite |
| ~~M2 — Conditional build (blueprint)~~ | ~~Full blueprint with IntelliJ + Go sidecar + Fidelity Floor productized + 4 Required Fixes~~ | — | — | **CLOSED 2026-05-27** — not authorized by GREEN verdict at locked inputs. Replaced by M2-lite. Re-opens only on scale-trigger (see `M1-Decision-Gate.md`). |

## Gate states

- **Pre-work gate** — **CLEARED 2026-05-27**. Status of the post-verdict punch list:
  - Pin LiteLLM image — **DONE 2026-05-27** (`v1.83.14-stable.patch.3`).
  - Secure 7-dev pilot cohort for M0-lite — **DONE 2026-05-27** (Namratha, Bharat, Mounika, Pranitha, Karthick, Rohit, Dhyan; 6 wait-listed). Per-dev role mapping (backend/frontend) still to be filled in `planning/M0-lite_Cohort_Recruitment.md` before PoC kickoff.
  - Close `[VERIFY-economics]` — **DONE 2026-05-27**. Inputs: S=$100, D=50, R=20%, C=$5k.
  - Calibrate M1 thresholds — **DONE 2026-05-26**.
  - Refresh Prior-Art for Copilot finding — **DONE 2026-05-26** (revision in `product/AI_Token_Optimizer_Prior_Art_Landscape_2026-05-21.md`).
- **Copilot routing-mode gate (pre-PoC)** — **🔴 UNRESOLVED (added 2026-05-31).** Per `../planning/validate_with_copilot.md`, the proxy can only interpose in Mode B (with an assistant swap to Claude Code / Cursor) or Mode C (custom endpoint). Mode A halts M0-lite and re-opens the `S` input. No result recorded.
- **M0-lite PoC gate** — **outcome unrecorded as of 2026-05-31.** The 2-day operational PoC window (Thu 5/28 → Fri 5/29) has elapsed; record the result here (pass criterion: ≥ 5 of 7 devs had the kit running with metrics by end of Fri 5/29), or note whether it was deferred pending the Copilot routing-mode validation above.
- **M0-lite measurement gate** — not reached. 5-day measurement window Mon 2026-06-08 → Fri 2026-06-12. Pass criterion: R ≥ 10% incremental on code-heavy prompts.
- **M1 gate** — **CLEARED 2026-05-27, GREEN (conditional)**. See `milestones/M1-Decision-Gate.md` for inputs, math, M2-lite scope contract, and scale-trigger.
- **M2-lite gate** — not reached; opens on M0-lite R-validation ≥ 10%.
- **Pilot gate** — not reached.

## Scale-trigger watch

The locked verdict assumes D=50 and the M2-lite scope. Re-open M1 if **any** of these becomes true (full criteria in `milestones/M1-Decision-Gate.md`):

- Active D crosses 150 engineers.
- AITO commits to commercializing the optimizer externally.
- VS Code or Claude Code native compression baseline shifts materially.

## Progress

Product definition is complete (Brief, Prior-Art, Build-vs-Adopt, Systems Review). M1 gate has cleared GREEN as of 2026-05-27 based on the economic math at locked inputs — the verdict pivots on C=$5k buying M2-lite scope, not the M2 blueprint. The original 2-4 week M0 box is cancelled and replaced by a 1-week M0-lite whose only job is to validate that R ≥ 10% incremental holds on AITO's code-heavy prompts.

Active path forward: pin LiteLLM image → secure 3-5 dev pilot cohort → run M0-lite (1 week) → freeze M2-lite scope contract in `planning/` → build M2-lite (~8 eng-days) → roll out to D=50 + 30-day monitoring.

## Planning vs. execution split

`planning/` is the durable plan — strategy, scope contracts, cost estimates for variants we might build later. `tracking/` is volatile execution state — this file plus per-milestone files. The M2-lite scope cut list lives in both: as the verdict's contract in `M1-Decision-Gate.md`, and as a standalone scope-freeze doc in `planning/` (task #16) so engineers building see it.
