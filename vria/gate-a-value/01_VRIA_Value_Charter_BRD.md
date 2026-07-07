# VRIA Value Charter / BRD

**Document set:** Value Realization Intelligence Agent  
**Version:** v1.2  
**Date:** 2026-07-05  
**Owner:** AaraMinds / Common Capabilities AI Portfolio  
**Status:** Implementation baseline  


---

## 1. Executive Summary

The **Value Realization Intelligence Agent (VRIA)** is a portfolio governance and measurement agent for AI initiatives. It tracks AI use cases from intake through pilot and production, maps each initiative to value hypotheses and measurable evidence, and produces leadership-ready recommendations on whether to **build, defer, scale, fix, re-baseline, continue, or stop**.

VRIA is not an agent that builds the individual AI use cases. It is the **value-control layer** for the AI portfolio.

## 2. Business Problem

The AI portfolio contains multiple initiatives across tools, agents, and domain layers. Innovation activity is visible, but value evidence is not consistently normalized, measured, attributed, or sustained.

Common issues:

- Use cases are tracked in spreadsheets, decks, BRDs, PRDs, PTB/PTO status trackers, and team updates.
- PTB/PTO/training status exists, but value readiness is not consistently visible.
- Some initiatives have strong narratives but weak baselines, weak attribution, or no net-value check.
- Leadership needs clarity on what to fund, pilot, scale, fix, defer, stop, or re-baseline.
- Value ownership is often weaker than delivery ownership.
- Benefits can regress after pilot unless sustainment is monitored.

## 3. Business Outcomes

| Outcome | Metric | Target for MVP |
|---|---|---|
| Portfolio clarity | % use cases normalized in registry | 100% of initial inventory |
| Value discipline | % pilot use cases with value hypothesis | 100% of pilot set |
| Evidence readiness | % pilot use cases with baseline/current/target status known | >= 80% |
| Leadership decision quality | % recommendations with evidence and confidence | 100% |
| Governance quality | % high-risk actions routed through approval workflow | 100% |
| Waste reduction | # weak/unmeasurable use cases deferred or re-baselined | Tracked, not pre-claimed |
| Sustainment | % realized cases rechecked after reporting window | 100% of realized claims |

## 4. Scope

### In Scope

- Portfolio intake and normalization.
- Tool / Agent / Layer classification.
- Value hypothesis capture.
- Baseline and evidence readiness assessment.
- Value realization scoring.
- Evidence-gap detection.
- Net-value, attribution, and confidence checks.
- Leadership scorecard drafting.
- Recommendation drafting: build, continue, scale, fix, defer, re-baseline, stop, needs sponsor, needs evidence.
- Approval-gated publication and status updates.
- Decision audit and ValueOps feedback loop.

### Out of Scope

- Autonomous funding decisions.
- Autonomous PTB/PTO or official status changes.
- Official finance booking of savings.
- Replacing product, architecture, finance, security, or governance approvals.
- Building the individual AI use cases inside the portfolio.

## 5. Stakeholders

| Stakeholder | Role |
|---|---|
| Executive sponsor | Owns portfolio value mandate and funding escalation. |
| Portfolio lead | Owns registry, cadence, scorecard, and decision process. |
| Use-case owner | Owns value hypothesis, evidence, and response to gaps. |
| Delivery owner | Owns execution status and implementation progress. |
| Finance / FinOps owner | Validates financial baselines, costs, savings, and net value when relevant. |
| Architecture owner | Reviews runtime, tool, and governance architecture. |
| Security/governance owner | Owns policy tiers, approvals, audit, and re-review triggers. |
| Engineering owner | Owns runtime build, integrations, observability, and operations. |
| Evaluation owner | Owns golden tests, red-team harness, online evals, and regressions. |

## 6. Autonomy Level

**MVP autonomy:** Level 2 — Drafting.

VRIA may analyze, score, summarize, and draft recommendations. It must not approve funding, declare official benefit realization, publish scorecards, or update official status without human approval.

Approval-gated execution is allowed only for low-risk administrative actions after explicit approval, such as creating a follow-up task or publishing an approved scorecard.

## 7. Success Criteria

The MVP is successful when:

1. All portfolio use cases are normalized into the registry.
2. 5–6 pilot use cases have value hypotheses and baseline readiness assessed.
3. The agent generates a leadership scorecard with evidence-backed classifications.
4. Every recommendation has score, confidence, evidence gaps, and owner action.
5. Unsupported value claims are marked **Unproven**, not reported as achieved.
6. Financial or productivity value claims include attribution and net-value checks.
7. Approval workflow is tested for publishing, status changes, and follow-up actions.
8. Golden eval pass rate meets the release gate defined in `07` and `11`.

## 8. Kill / Defer Conditions

Defer or stop MVP expansion if:

- No sponsor is identified.
- Pilot owners cannot provide baselines or evidence sources.
- The agent produces unsupported value claims in release-gate tests.
- Approval workflow is bypassable.
- Tool contracts cannot be implemented with least privilege.
- Scorecards cannot be audited back to evidence.

## 9. Recommendation

Build VRIA as a narrow MVP around the initial AI use-case portfolio, using the pilot set defined in `gate-d-operations/13_VRIA_Pilot_Plan.md`.

Do not position it as a generic dashboard. Position it as the **evidence-backed ValueOps layer** for the AI portfolio.
