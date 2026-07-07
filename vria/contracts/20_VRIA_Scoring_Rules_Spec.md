# VRIA Scoring Rules Specification

**Document set:** Value Realization Intelligence Agent  
**Version:** v1.2.1  
**Date:** 2026-07-05  
**Owner:** AaraMinds / Common Capabilities AI Portfolio  
**Status:** Implementation baseline  


---

## 1. Purpose

This document defines executable scoring logic for VRIA. Scores are rule-governed; the model may explain, but must not invent or override the scoring rules.

## 2. Gate A Intake Readiness Score

| Component | Points |
|---|---:|
| Named value owner | 10 |
| Named delivery owner | 5 |
| Sponsor identified or sponsor gap declared | 5 |
| Clear scope and tier | 10 |
| Expected benefit stated | 10 |
| Primary metric identified | 15 |
| Baseline verified and available | 15 |
| (or) Baseline establishment plan approved, not yet verified | 8 |
| Target and time window defined | 10 |
| Evidence source identified | 10 |
| Approval boundary recorded | 5 |
| Dependencies identified | 5 |

**Total:** 100.

## 3. Gate B+ Realization Score

| Component | Points |
|---|---:|
| Strategic alignment | 10 |
| Baseline quality | 15 |
| Evidence quality | 20 |
| Metric movement | 20 |
| Attribution confidence | 10 |
| Net value | 10 |
| Sustainment | 10 |
| Governance readiness | 5 |

**Total:** 100.

## 3a. Component Formulas (executable)

Each component is a deterministic function of canonical schema fields (`contracts/17`). No model judgment inside a formula. Inputs: `ValueHypothesis` (VH), `MetricSnapshot` (MS), linked `EvidenceSource` records (ES), assessment sustainment fields.

| Component | Max | Formula |
|---|---:|---|
| strategic_alignment | 10 | `VH.business_objective` non-empty AND `sponsor` named = 10; `business_objective` non-empty only = 6; otherwise = 0. `[DECISION NEEDED]` Richer rubric option: map `benefit_type` to a portfolio-priority table maintained by the portfolio lead (10/6/3 by priority band). Adopt at first quarterly review. |
| baseline_quality | 15 | Baseline value present + period defined + source `Authoritative` = 15; value + period, source `Secondary` = 10; value without period = 6; approved baseline plan only (no value) = 4; none = 0. |
| evidence_quality | 20 | Sum of: authority (`Authoritative`=8, `Secondary`=4, `Unknown`=0) + freshness (`Fresh`=8, `Aging`=5, `Stale`=2, `Unknown`=0) + citation pointer present on all linked ES (4, else 0). |
| metric_movement | 20 | `progress = clamp((MS.current_value − MS.baseline_value) / (MS.target_value − MS.baseline_value), 0, 1)` (invert numerator and denominator sign for lower-is-better metrics). Points = `round(20 × progress)`. Any of current/baseline/target missing, or `target = baseline` → 0. |
| attribution_confidence | 10 | `DirectMeasurement`/`A_BComparison`=10; `MatchedComparison`=7; `BeforeAfter` with ≥1 documented confounder=6, without=4; `ExpertJudgement`/`ProxyMetric`=3; `Unknown`=0. |
| net_value | 10 | `Positive`=10; `NotApplicable` with rationale=8; `Neutral`=5; `Unknown` or `Negative`=0. |
| sustainment | 10 | `sustainment_status`: `Ok`=10; `NotStarted` (not yet Realized, no negative signal)=6; `AtRisk`=4; `Regressed`=0. |
| governance_readiness | 5 | Approval boundary recorded (`approval_required_for` non-empty)=3 + no unresolved policy/injection flags=2. |

### Worked Examples

**W1 — strong Realized case:** sponsor + objective (10) + verified authoritative baseline (15) + evidence 8+8+4 (20) + current = target → progress 1.0 (20) + DirectMeasurement (10) + net Positive (10) + sustainment Ok (10) + governance 3+2 (5) = **100**.

**W2 — mid pilot, financial claim without cost:** objective only (6) + secondary baseline with period (10) + evidence 4+5+4 (13) + progress 0.5 (10) + BeforeAfter with confounders (6) + net Unknown (0) + NotStarted (6) + governance 3+2 (5) = **56** pre-cap. Cap "Net value Unknown for financial claim" (74) does not bind at 56. State: OnTrack blocked by net-value rule → AtRisk; recommendation NeedsEvidence.

**W3 — degenerate, no baseline:** objective + sponsor (10) + baseline none (0) + evidence 8+8+0 (16) + movement missing baseline (0) + Unknown attribution (0) + net NotApplicable (8) + NotStarted (6) + governance (5) = **45** pre-cap. Cap "No baseline" (49) does not bind; cap "Attribution Unknown" (69) does not bind. State: HypothesisOnly.

## 4. Score Caps

Caps are applied after raw score calculation. The lowest applicable cap wins.

| Condition | Maximum Score | State Impact |
|---|---:|---|
| No value owner | 29 | NotReady |
| No primary metric | 39 | HypothesisOnly |
| No baseline | 49 | HypothesisOnly |
| No current value | 59 | BaselineReady maximum |
| Evidence source not authoritative | 64 | Unproven / AtRisk |
| Attribution method Unknown | 69 | Cannot be Realized |
| Material confounders undocumented | 74 | Cannot be High confidence |
| Net value Unknown for financial/productivity claim | 74 | Cannot be Realized |
| Evidence stale | 79 | Cannot be Realized without owner exception |
| Publication-readiness cap: approval state not Approved | 89 | Draft only. **This cap gates publication, not evidential quality** - dashboards must plot the pre-cap realization score for trends and apply this cap only to publication eligibility. |
| Prompt-injection or policy issue unresolved | 0 | Blocked until resolved |

## 5. Value State Mapping

| Conditions | Value State |
|---|---|
| Missing owner/scope/metric | NotReady |
| Expected benefit exists but no baseline | HypothesisOnly |
| Baseline and target exist but no current value | BaselineReady |
| Current value trending positively but target not yet achieved | OnTrack |
| Delivery/evidence/governance risk threatens outcome | AtRisk |
| Target achieved with fresh authoritative evidence, attribution, net value, and approval | Realized |
| Target missed with evidence | NotRealized |
| Previously Realized but two consecutive sustainment checks failed (section 7) | Regressed |
| Claim cannot be substantiated | Unproven |

## 6. Recommendation Mapping

| Situation | Recommendation |
|---|---|
| Score >= 85 and Realized | Scale or ContinuePilot depending on maturity. |
| Score 70-84 and OnTrack | ContinuePilot or Build. |
| Score 50-69 with fixable evidence gaps | Fix or NeedsEvidence. |
| No sponsor for high-commitment Layer | NeedsSponsor. |
| No metric or baseline | Rebaseline or NeedsEvidence. |
| Negative net value | Stop, Fix, or Rebaseline. |
| Regressed | Fix or Rebaseline. |
| Policy/approval issue | Defer until control is resolved. |

## 7. Sustainment Threshold

Referenced by GE-006, the PRD value states, and the state mapping above. Definition:

- After a use case reaches `Realized`, VRIA runs a **sustainment check once per metric `reporting_window`** (cadence defined in `gate-b-behavior/06` section 8; default monthly).
- A sustainment check **fails** when the measured benefit for the period is below the **sustainment threshold: 80% of target value** (default; the value owner may set a different threshold at approval time, recorded on the assessment).
- **First failed check:** state stays `Realized`; owner is notified; assessment records `sustainment: at_risk`.
- **Two consecutive failed checks:** state moves to `Regressed`; recommendation is `Fix` or `Rebaseline`; owner review is required.
- A stale or missing metric snapshot counts as a failed sustainment check.

## 8. Confidence Rules

- High requires authoritative fresh evidence, baseline/current/target, attribution, and no material unresolved confounders.
- Medium allows some uncertainty but must disclose caveats.
- Low is required when source authority, attribution, freshness, or cost data is weak.

## 9. Tier-Specific Notes

| Tier | Scoring Note |
|---|---|
| Tool | Prefer operational metrics: MTTR, query time, manual effort avoided. |
| Agent | Prefer workflow metrics: cycle time, rework, quality, approval rate. |
| Layer | Requires stronger sponsor, data ownership, attribution, and sustainment checks. |
