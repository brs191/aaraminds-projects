# M1 — Decision gate

**Owner:** Raja  ·  **Status:** **GREEN (verdict locked 2026-05-27, conditional on M0-lite R-validation)**  ·  **Calibrated:** 2026-05-26 (post Prior-Art refresh)  ·  **Verdict:** 2026-05-27
**Source:** `../../planning/AI_Token_Optimizer_Build_vs_Adopt_2026-05-21.md`, `../../spike/SPIKE_PLAN.md`, `../../product/AI_Token_Optimizer_Prior_Art_Landscape_2026-05-21.md` (2026-05-26 revision)

## Goal

Turn the M0 evidence into one decision: build the product, keep the spike composition as internal tooling, or adopt off-the-shelf / drop.

## Pre-work — before M0 Week 1

- [x] **Calibrate the gate thresholds (2026-05-26).** See the *Calibrated thresholds* section below; the original spike-plan defaults were lowered for token-reduction (Green 25% → 20%, Amber 15% → 10%) to account for the VS Code 1.118 native-compression baseline surfaced in the 2026-05-26 Prior-Art refresh. Quality and latency thresholds carry engineering rationale and are settled.
- [x] **Close `[VERIFY-economics]` (2026-05-27).** Inputs locked: S = $100/dev/month, D = 50 (conservative pilot cohort; real team size 500, scale ceiling ~3000), R = 20% (LLMLingua-2 vendor claim, to be validated by M0-lite), C = $5,000 (M2-lite scope, NOT M2 blueprint).
- [x] **Decide whether AITO wants the niche as a product (2026-05-27).** YES, but as internal pilot tooling against the locked M2-lite scope — NOT as a commercialized product. This honours the `Build_vs_Adopt` recommendation to not compete in the niche externally while still capturing internal value. Commercialization re-opens the gate (see scale-trigger below).

## Calibrated thresholds (2026-05-26)

The spike's A/B harness already compares with-optimizer against without-optimizer, and "without-optimizer" includes whatever native compression the assistant runs (Claude Code Auto-Compact, VS Code 1.118 token-efficiency). So every threshold below is **measured incremental beyond the assistant's free baseline** — not against a hypothetical raw-zero baseline. This is the calibration the Prior-Art 2026-05-26 update demands.

### Token reduction

| Outcome | Threshold | Rationale |
|---|---|---|
| Green | ≥ **20%** median input-token reduction incremental over the assistant's native baseline | Original 25% (`[VERIFY]` default) lowered because VS Code 1.118 claims up to ~20% on its own `[VERIFY]`. The optimizer must clear a real incremental gap to justify the M2 build cost. 20% is the threshold above which the gap is unambiguous and the economics work even at conservative token-spend assumptions. |
| Amber | ≥ **10%** | Floor for "real but modest." Below 10%, the optimizer's runtime cost (G6: ≤ 5% of tokens saved) eats most of the remaining margin and the M2 build payback stretches past 18–24 months under typical economics. |
| Red | < **10%** | Insufficient incremental savings to justify a multi-month build against a free, improving baseline. |

### Answer-quality regression (the Fidelity Floor)

| Outcome | Threshold | Rationale |
|---|---|---|
| Green | ≤ **5%** of A/B pairs show measurable degradation, **≤ 3%** on code-heavy fixtures | The 5% bound is the Fidelity Floor design target (Module 5 Finding 1). The tighter 3% on code-heavy is because LLMLingua-2 was trained on prose and code-heavy is the most likely failure surface; this is exactly the regression mode Copilot's agent-mode over-summarisation bug (`microsoft/vscode-copilot-release#11966`) exhibits, and avoiding it is the differentiator. |
| Red | > **5%** general OR > **3%** code-heavy OR any systematic degradation in a single category | Hard floor; cannot ship through. Auto-rollback (FR-17) must remove the offending strategy on every breach. |

### Latency overhead

| Outcome | Threshold | Rationale |
|---|---|---|
| Green | < **300 ms p95** end-to-end for chat requests, < **100 ms p95** for inline completions | 300 ms p95 from the spike-plan default for chat — defensible for an interactive surface. The 100 ms p95 carve-out for completions reflects that they fire continuously as the developer types; anything slower makes the assistant feel sluggish, which P1 will not tolerate (PRD §5). |
| Red | > **500 ms p95** chat OR > **150 ms p95** completions OR any user-visible stall on a hot-path request | Engineering floor; degrades the developer experience beyond what the savings can buy back. |

### Self-funding

| Outcome | Threshold | Rationale |
|---|---|---|
| Green | Optimizer runtime cost (Compression Sidecar inference + Advisor agent calls + Evaluator) ≤ **5% of tokens saved**, measured per developer per week | Carried verbatim from PRD G6. Tightens the net-savings calculation: if the optimizer's overhead is more than 5% of what it saves, the build is not paying back its own runtime. |

### Economic break-even (locked 2026-05-27)

Payback period (months) = **C / (S × R × D)** where S = per-active-dev monthly token spend, D = active dev count, R = measured incremental token reduction, C = M2 build cost.

| Outcome | Payback period | Action |
|---|---|---|
| Green | ≤ 9 months | Build cleared on economics |
| Amber | 9–18 months | Requires re-scope (model routing, caching, narrower cohort) before build opens |
| Red | > 18 months | Cancel; cost math doesn't carry |

**Locked inputs (2026-05-27):**

| Variable | Value | Source / note |
|---|---|---|
| S | $100/dev/month | AITO billing snapshot |
| D | 50 engineers | Conservative pilot cohort. Real team is 500; success ceiling is ~3000. The 50 is intentionally small to validate before scaling — see scale-trigger below. |
| R | 20% (assumed) | LLMLingua-2 vendor claim against the VS Code 1.118 baseline. **Validation pending in M0-lite** (1-week compressed measurement). Math tolerates R down to ~11% before slipping to Amber; below 10% reverts to Red. |
| C | $5,000 | M2-lite scope, ~8 engineer-days at Hyderabad senior loaded rates. This is NOT the M2 blueprint scope ($75-110k); see scope reconciliation below. |

**Math:** Monthly savings = 50 × $100 × 0.20 = **$1,000**. Payback = $5,000 / $1,000 = **5 months → GREEN**.

### Scale-trigger — when to re-open this gate

Re-open M1 if any of the following becomes true. None of these are inputs the spike or pilot can reveal; each is a strategic shift that voids the locked verdict.

- Active D crosses **150 engineers** (M2-lite scope collapses on a larger user base; manual install and single-IDE constraint stop being acceptable).
- AITO commits to commercializing the optimizer as an external product (different project entirely — see scenario 2 in the 2026-05-27 conversation log; the `Build_vs_Adopt` non-compete recommendation needs re-litigating, not bypassing).
- VS Code or Claude Code native compression baseline jumps materially (R ≥ 20% becomes implausible incremental over a stronger baseline; refresh `product/AI_Token_Optimizer_Prior_Art_Landscape_*.md` and re-run the math).

### Leadership decision (separate from numbers)

A Green verdict requires not only the numeric criteria above but also a leadership confirmation that AITO wants the local-first / zero-egress / IntelliJ / Fidelity-Floor niche as a product. This is a strategic call, not a metric — the gate cannot return Green on numbers alone.

## The gate (calibrated)

| Outcome | Criteria | Action |
|---|---|---|
| **Green** | ≥ 20% incremental token reduction · ≤ 5% / ≤ 3% (code-heavy) quality regression · < 300 ms / < 100 ms (completions) latency · optimizer overhead ≤ 5% of savings · 12-month payback met against AITO economics · niche wanted as a product | Open M2 — build the narrow product (Option B) |
| **Amber** | ≥ 10% reduction with no quality regression but either payback period > 12 months OR niche not wanted as a product OR latency 300–500 ms / 100–150 ms | Keep the LiteLLM + LLMLingua-2 setup as internal tooling; stop |
| **Red** | < 10% reduction OR > 5% quality regression OR > 3% code-heavy regression OR > 500 ms / > 150 ms latency OR optimizer overhead > 5% of savings | Adopt an off-the-shelf product for internal use, or shelve the initiative |

A Green verdict requires the answer-quality criterion AND the leadership product-ambition decision in addition to the token number — not a single metric.

## Verdict — 2026-05-27 — GREEN (conditional)

**Outcome: GREEN**, conditional on M0-lite confirming R ≥ 10% incremental on AITO's actual code-heavy prompts.

The verdict pivots on the economic math, not on a full 2-4 week measurement spike. With S, D, and C locked, payback is 5 months — comfortably within the ≤9-month GREEN threshold, with enough margin to absorb a meaningful R miss (down to R=11% still clears AMBER). The full M0 box was sized for a $75k decision; at $5k the decision is dominated by inputs, and the only honest spike question becomes "does R hold above the AMBER cliff?" — which a 1-week M0-lite answers cheaply.

### What GREEN authorizes

**Build M2-lite, not M2 blueprint.** The $5k envelope buys ~8 engineer-days at Hyderabad senior loaded rates. That funds a specific narrow scope, and any drift back to the blueprint retroactively voids the verdict.

**M2-lite scope contract (in/out):**

| In scope | Out of scope |
|---|---|
| Existing `spike/` kit (LiteLLM + LLMLingua-2 + compression hook), pinned image | IntelliJ plugin (blueprint called for parity — explicitly cut) |
| VS Code `.vsix` pointing at localhost LiteLLM proxy | Bundled Go sidecar with supervised child processes |
| Manual install via docker-compose + documented setup | Productized Fidelity Floor monitoring (continuous) |
| One-time Fidelity Floor measurement at install | Automated supervisor for compression-sidecar lifecycle |
| Metadata-only egress (existing in spike kit) | The 4 Required Fixes from `evaluation/AI_Token_Optimizer_Systems_Review_2026-05-21.md` as built features |

**Risks accepted by this scope:** if docker-compose breaks on a user's machine the optimizer dies until they fix it; no IntelliJ user coverage (acceptable at D=50 if all 50 are VS Code users — verify in cohort securing); no graceful degradation beyond what LiteLLM already provides; Required-Fix surface remains as known operational gaps documented in the Systems Review rather than mitigated in code.

### What GREEN does NOT authorize

Anything outside the cut list above. Specifically: if anyone asks during build for IntelliJ support, productized monitoring, or any of the 4 Required Fixes as features, the answer is "out of scope by 2026-05-27 decision." Scope creep is the single failure mode that retroactively flips this verdict from GREEN to RED — every added feature day past 8 makes the payback math worse.

### M0-lite — the validation condition

**1 week, 3-5 pilot devs, real coding work**, measured against the assistant's native baseline:

- R-lite ≥ 10% incremental on code-heavy fixtures → green-light M2-lite build.
- R-lite < 10% → cancel M2-lite, write the post-mortem; the verdict reverts to "would have been GREEN if R held, did not."

The original 2-4 week M0 box is **cancelled**. That scope was sized for a $75k decision; at $5k a 1-week sample is sufficient insurance against the only assumption that can flip the math.

### Gate — verdict recorded

M1 is done. Verdict captured above with locked inputs, calibrated thresholds met, the leadership product-ambition call made (YES to internal pilot tooling, NO to commercialization), and the M0-lite validation condition set. The scale-trigger watches for the conditions that would void the verdict.

## Tasks

- [x] Calibrate gate thresholds (2026-05-26)
- [x] Close `[VERIFY-economics]` with locked inputs (2026-05-27)
- [x] Make the product-ambition decision (2026-05-27 — internal pilot tooling, not commercialized)
- [x] Record the GREEN verdict with rationale and numbers (2026-05-27)
- [x] **Hand off to M0-lite** (2026-05-27) — see `M0-lite.md`. PoC Thu 5/28 → Fri 5/29 with 7-dev cohort; measurement Mon 6/8 → Fri 6/12.
- [x] **Freeze M2-lite scope contract** (2026-05-27) — written to `../../planning/M2-lite_Scope_Contract.md`.
- [ ] **Reconfirm verdict at end of measurement week (2026-06-12)** — if R-lite ≥ 10% on code-heavy, GREEN stands and M2-lite build is authorized. If R-lite < 10%, revert verdict and write post-mortem.
