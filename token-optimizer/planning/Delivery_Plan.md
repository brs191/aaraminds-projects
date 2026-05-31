# AI Token Optimizer — Delivery Plan

**Status:** v0.1 — working draft
**Owner:** Raja (delivery owner)
**Date:** 2026-05-25
**Location:** `product-research/token-optimizer/planning/`
**Companion to:** `Roadmap.md`, `../product/AI_Token_Optimizer_Product_Brief_2026-05-24.md`, `../tracking/Status.md`

> Produced with the `AITO_Project_Planner` persona (composing `09_Project_Delivery_Planning_System`). The Roadmap defines *what* gets built and the gates; this plan adds the delivery layer — work breakdown to a binary Definition of Done, effort estimation, the critical path, the risk and assumption register, and the replan triggers. It deliberately does **not** commit a calendar date; see Commitment.

---

## How the planning gates resolved

The persona's gates were run against the token-optimizer source documents — the Roadmap, the spike plan, the build-vs-adopt analysis, the product brief, and the Module 5 systems review. Four are worth stating up front because they shape everything below.

**Plan Mode.** New plan — the first full delivery plan over the existing gate roadmap. Not a replan: nothing has slipped, the spike has not started.

**Fixed-Constraint Gate.** No constraint was supplied, so it was read from the source documents — and it is not uniform across the initiative:

- **M0 (the spike) is time-fixed by design.** The spike plan is explicit — "it is a time box, not a time target": 2–4 weeks, end early if decisive. Time is the anchor; *scope* (how much usage data, how large the fixture set) and *capacity* (how many developers route their work) flex within the box.
- **M1 (the gate) is a decision, not a build.** It carries no scope/time/capacity tradeoff — it consumes M0's evidence and produces a verdict in days.
- **M2 (the build) is gate-contingent and unscoped internally.** It exists only on a Green M1, and its internal phasing is "set on entry." Capacity for it is unspecified.

So this plan is detailed and committed for **M0 + M1**, and deliberately coarse and conditional for **M2**. Scheduling M2 now would be fiction twice over — conditional on a gate that has not run, and against a team that does not exist.

**Estimate Honesty Gate.** The spike is a measurement rig with no comparable past AITO delivery to anchor on. M0 is treated as a *structurally time-boxed* milestone — its job is a fast yes/no, not a scope to estimate to completion. Effort figures below are decomposition-based, deliberately wide, low-confidence. The M2 effort (~3–5 engineer-months) is carried verbatim from the Roadmap as `[VERIFY]` and is not refined here — it cannot be, until the gate clears and M2 is scoped.

**Critical Path Gate.** The milestone chain M0 → M1 → M2 is strict serial — the gates are deliberate decision points, not artefacts of resourcing. The critical path is the whole chain, plus a short pre-work run that sits *before* M0 Week 1 and is easy to miss (see Critical path).

---

## Outcome

The primary outcome — committed scope — is a **decision, not a product**: the AITO's engineering team has a measured, evidence-based answer to whether context compression meaningfully cuts its own AI coding-assistant token spend without degrading answer quality, and a recorded Green / Amber / Red verdict that either opens a product build, keeps the composition as internal tooling, or adopts off-the-shelf / shelves the initiative.

**For:** Raja as delivery owner and the AITO's engineering team who run the spike; and AITO's leadership, who receive the verdict and make the build call.

The conditional outcome — gate-contingent — is the **narrow Token Optimizer product** (Roadmap M2), scoped and planned only if M1 returns Green.

The outcome that matters is the *evidence*, not the activity. A spike that runs four weeks but cannot produce a trustworthy gate verdict — because the usage data was thin or synthetic, or the quality review was skipped — is not the outcome; it is the failure mode the time box and the code-heavy fixtures exist to prevent.

## Fixed constraint

**Time for M0** — fixed at a 2–4 week box; scope and capacity flex within it. For the initiative as a whole, **capacity** is the open variable that must be supplied before any date past the M0 box exists.

## Milestones

Drawn from the Roadmap and the `../tracking/milestones/` files, sharpened so each gate is a binary, demonstrable Definition of Done. The milestone chain is unchanged — this plan makes the Roadmap executable, it does not redesign it.

| # | Milestone | Definition of Done (binary) | Owner | Effort (low-confidence) | Depends on |
|---|---|---|---|---|---|
| Pre-work | Spike start gate | LiteLLM image pinned to a specific advisory-checked tag; M1 gate thresholds calibrated to AITO's economics and recorded; the spike runner(s) named and committed; a set of developers committed to routing daily work in Week 2 | Raja | ~1–3 days | — |
| M0 | Spike | Real-usage metrics collected over genuine daily work; an A/B run completed across a fixture set that includes code-heavy prompts; answer quality reviewed by a human; `summarize.py` aggregation run — **and no blocking gap forces the gate to rely on a `[VERIFY]`** | Raja (full-time, ~2 weeks) | ~8–15 engineer-days over a 2–4 week box | Pre-work complete |
| M1 | Decision gate | A **written verdict** — Green / Amber / Red — naming the measured numbers, the rationale, and (for Green) the leadership decision that AITO wants the niche as a product | Raja | ~1–3 days | M0 decision-ready |
| M2 | Conditional build | The narrow Option B product built and holding the Fidelity Floor and latency budget against the spike baseline. **Scoped and planned on entry — exists only on a Green M1.** | Raja (interim placeholder — real team named on entry) | ~3–5 engineer-months `[VERIFY]` | M1 = Green |

**M0 effort note.** The ~8–15 engineer-days is *marginal* effort — standup, fixture-set construction, the A/B run, quality review, analysis. It excludes Week 2's "real usage," which is near-zero marginal effort: developers route their normal daily work through the proxy, they do not do extra work. The 2–4 week *calendar* box is duration, not effort — Week 2 is mostly wait-time while real-usage data accumulates.

**M2 is not estimated here.** Its ~3–5 month figure is the Roadmap's `[VERIFY]` placeholder. M2's real delivery plan — milestones, the four mandatory systems-review fixes, the Fidelity Floor redesign, the IDE-plugin phasing — is produced on entry, after a Green gate. See Replan triggers.

**Capability readiness for M2.** A pass over the current `skills-pack/.claude/skills/` and agent roster (2026-05-25) shows two real skill gaps that sit on the build path and must be closed *before* M2 work begins, not during it. (1) **VS Code `.vsix` extension development** — no skill or agent today; `frontend-engineering` is explicitly React/Next.js and excludes IDE-plugin work. (2) **IntelliJ plugin development** — no skill or agent today; this gap hits the wedge directly, since IntelliJ parity is part of the product's defensible differentiation. One **partial** gap: the TLS-terminating interception proxy (Module 5 Finding 4) is touched by `mcp-go-server-building` but is not a focused capability — the proxy is the highest-assurance component in the design and warrants its own skill before build. Author the two missing IDE-plugin skills (and a focused proxy skill) via `skill-creator` as part of M2 entry pre-work, alongside the four Module 5 fixes and the Fidelity Floor redesign. The rest of the build — Go MCP servers, the Python compression sidecar, the AI evaluation harness, architecture, threat modeling, testing — is well covered by the current pack today.

**Ownership — concentration noted.** Pre-work, M0, and M1 are all owned by Raja (M0: full-time for ~2 weeks). The M2 build team remains Raja and only resolves on a Green M1. **The concentration is deliberate but worth flagging:** the planner persona's Output Discipline prefers separation between the engineer producing M0 evidence and the leader chairing the M1 verdict; consolidating both in one person unblocks the schedule but removes that check. The mitigation is structural — the M0 audit log is the canonical evidence, and the M1 verdict must cite measured numbers from it, so any reviewer can re-verify the verdict against the raw data. This concentration is tracked as a risk below.

## Critical path

The milestone chain is a **strict serial dependency**: `Pre-work → M0 → M1 → M2`. Each gate blocks the next; there is no parallelism to exploit between milestones.

M0's internal sequence is also serial — it is the spike plan's four weeks:

```
Pre-work → M0 Wk1 stand up → M0 Wk2 real usage → M0 Wk3 A/B + quality → M0 Wk4 analyse → M1
(1–3 days)  (2–4 dev-days)   (calendar wait +    (2–4 dev-days)        (1–2 dev-days)  (1–3 days)
                              fixture build 1–2d)
```

Three items sit *on* the path and are easy to miss:

- **Pre-work is on the critical path, not a warm-up.** Calibrating the gate thresholds, pinning the LiteLLM image, and naming the spike runner all gate Week 1. If pre-work is treated as something to do "while the spike starts," Week 1 runs against guessed thresholds and the M1 verdict is measured against the wrong bar. Time-box pre-work to ≤ 3 days and finish it before Week 1.
- **The fixture set is a convergence point.** The Week 3 A/B run cannot start until the fixture set exists, and the fixture set must be built from genuinely representative AITO prompts during Week 2. Build it *during* Week 2, in parallel with usage-data accumulation — discovered late, it pushes Week 3.
- **Week 2 is wait-time governed by adoption, not effort.** Real-usage data accumulates only as fast as developers actually route their work. Too few participants and Week 2 produces thin data, and the box expires without a trustworthy verdict — the dominant schedule risk, see Risks.

The critical path produces the M0 gate in **2–4 weeks of calendar time from the day pre-work completes** — closer to 2 if results are decisive early, the full 4 only if data is thin. M1 adds days. M2 is not on a calendar until it is scoped.

## Risks and assumptions

| Risk / Assumption | P | I | Response | Owner | Trigger signal |
|---|---|---|---|---|---|
| LLMLingua-2 is trained on prose, not code — compressing code blocks degrades answers | M | Critical (to the verdict) | **Accept** — this is the bet the spike exists to test cheaply; mitigation is the code-heavy fixture and the human quality review. A Red gate here is a *successful* spike. | `Raja` | Week 3 A/B shows quality regression concentrated on code-heavy prompts |
| Thin Week 2 adoption — too few developers route real work; data is sparse or synthetic | M | H | **Mitigate** — secure explicit commitment from a set of developers in the pre-work gate; monitor request volume from Week 2 day 2, not at Week 3. | Raja | `metrics/requests.jsonl` volume flat or low mid-Week 2 |
| Native auto-compact has already closed the gap — measured savings clear no threshold | M | H | **Accept and surface** — a market reality the spike measures, not a defect; an Amber/Red here is honest signal, not failure. | Raja | M0 median reduction lands below the Amber floor |
| LiteLLM 2026 supply-chain incident — running an unpatched image | M | H | **Mitigate** — pin to a specific advisory-checked tag in the pre-work gate; `[VERIFY]` the advisory before Week 1. | `Raja` | No advisory-clean tag exists at pin time |
| Gate thresholds never calibrated — verdict measured against guessed numbers | M | H | **Mitigate** — calibration is a pre-work exit condition; Week 1 does not start without it. | Raja | Week 1 begins with thresholds still at the `[VERIFY]` defaults |
| Quality scoring is subjective; LLM-as-judge is non-deterministic (systems review Finding 10) | M | M | **Mitigate** — human spot-checks pair with any judge score; no single number decides the gate. | `Raja` | An A/B verdict resting on one judge score with no human review |
| M2 capacity is a placeholder, not a commitment — Raja is named only to keep the row owned; the real team must be formed on entry | Certain | H (on the M2 date only) | **Accept and surface** — a 3–5 engineer-month build by one part-time owner is fiction; the placeholder unblocks paperwork but the M1 verdict (if Green) must include naming the actual M2 team. | Raja | M1 returns Green and no real M2 team is named within the gate verdict |
| Concentration — same person on M0 evidence and M1 verdict | Low | M | **Accept and mitigate** — Raja owns pre-work, M0, and M1; the planner persona's Output Discipline prefers separation. Mitigation: M0 audit log is the canonical evidence and M1 must cite measured numbers from it, so any reviewer can independently re-verify the verdict. | Raja | M1 verdict is published without traceable citations to the M0 audit log |
| **Assumption:** the spike kit runs correctly as built — its syntax and config were verified by manual review only; the sandbox build check could not be run | — | — | Tracked — run `python -m py_compile` on the hooks and a JSONL parse on the fixtures on first use; Week 1 standup is the verification. | `Raja` | Week 1 `docker compose up` or `measure.py` fails on first run |
| **Assumption:** AITO's developers' daily coding work is representative enough to measure | — | — | Tracked — the fixture set must be built from genuine prompts, not invented ones. | `Raja` | The fixture set is filled with synthetic prompts because real ones were not captured |
| **Assumption (M2 only):** the four mandatory Module 5 fixes, the Fidelity Floor redesign, and the missing VS Code / IntelliJ plugin skills are all in place before M2 build | — | — | Tracked — these are M2 *entry* conditions, not in-build work; the Fidelity Floor is unsound as designed (systems review Finding 1), and the two IDE-plugin skills do not exist in the pack today (see *Capability readiness for M2*). | Raja | M2 opens with the blueprint still at v0.1, or with the IDE-plugin skills unauthored |

## Contingency model

M0 has no calendar buffer to size — the 2–4 week box *is* the buffer: 2 weeks is the plan, the extra 2 weeks absorb thin data or an inconclusive first A/B pass. Beyond the box, the contingency is **structural, not temporal**, exactly as the Roadmap intends: M0 is the cheap, fast, fail-early test, and the M1 kill/continue gate is the contingency for the whole initiative. The project is designed to fail for a few engineer-weeks rather than a few engineer-months.

Once M1 returns Green and M2 is scoped, an explicit phase-boundary buffer is added before the first external-facing milestone (the IDE-plugin release), sized to the M2 findings. There is nothing to size before then.

## Commitment

**M0 is now dated; M2 remains deliberately undated.** With Raja named as the M0 spike runner at full-time allocation for ~2 weeks, the first half of the open capacity question is resolved; the M2 half remains open until a Green gate.

What can be stated honestly:

- **M0 duration:** a **2–4 week box** from the day pre-work completes. With Raja full-time, **plan date ~2 weeks**; **committed bound ~3 weeks** at high confidence (Week 2 is calendar wait-time for real-usage data and is not compressed by full-time effort, so the box may need its third week to clear thin data); the 4-week box is the hard cap. The genuine remaining uncertainty is the *start* date, which the pre-work gate governs.
- **M0 marginal effort:** roughly **8–15 engineer-days**, low confidence, decomposition basis — standup, fixture build, A/B, quality review, analysis. At full-time over 2 weeks (~10 working days) this fits with the lower end of the effort range; the upper end pushes into the third week.
- **M1:** days after M0 — a decision, not a build.
- **M2:** **deliberately undated.** Roadmap effort placeholder ~3–5 engineer-months `[VERIFY]`. It carries a calendar only after it is scoped on a Green gate, against a known team.

**The one focused question that remains:** *if M1 returns Green, who builds M2 and at what capacity?* That second half is needed only on a Green gate, and is better answered then.

## Replan triggers

This plan is replanned — not quietly absorbed — when any of these fire:

- **M1 returns Amber or Red** → the Roadmap's kill/pivot point, not a routine replan. Stop; M2 does not open. The plan's committed scope is complete.
- **M1 returns Green** → M2 opens and gets its own delivery plan (milestones, the four Module 5 fixes, the Fidelity Floor redesign, IDE-plugin phasing). This plan's M2 row is superseded by that plan.
- **The M0 box expires (4 weeks) without decision-ready evidence** → decide explicitly: extend the box with a stated reason, or take the M1 decision on the evidence in hand. Do not drift.
- **Week 2 adoption is too thin to trust** → the gate cannot be trusted on schedule; extend Week 2 within the box or re-scope which developers participate.
- **The spike kit fails on first run** (the verification assumption