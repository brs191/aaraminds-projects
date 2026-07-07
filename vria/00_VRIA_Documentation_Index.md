# VRIA Documentation Index

**Document set:** Value Realization Intelligence Agent  
**Version:** v1.3  
**Date:** 2026-07-07  
**Owner:** AaraMinds / Common Capabilities AI Portfolio  
**Status:** Implementation baseline  


---

## Purpose

This documentation pack defines the enterprise baseline to design, build, pilot, govern, and operate the **Value Realization Intelligence Agent (VRIA)**.

The agent answers one leadership question:

> Which AI use cases are creating measurable value, which are ready to scale, which are blocked, and which should be fixed, deferred, re-baselined, or stopped?

## v1.2.1 Fix Summary

| Fix | Resolution |
|---|---|
| Sustainment threshold undefined | Defined in `20` section 7: check every freshness cycle after Realized; fail below 80% of target (owner-adjustable); two consecutive failures move state to Regressed. |
| Volume dataset dropped in v1.2 | Reinstated in `07` section 4: percentage gates bind to a >= 50-record labeled dataset; golden tests tagged critical (100% pass) vs non-critical. |
| Approval cap conflated with quality | Renamed publication-readiness cap in `20`; dashboards trend the pre-cap realization score. |
| Intake baseline scoring too generous | Split: 15 points verified baseline, 8 points approved plan only. |

## v1.2 Fix Summary

v1.2 closes the implementation-readiness gaps found in v1.1.

| Fix Area | v1.2 Resolution |
|---|---|
| Version consistency | All reusable documents are now v1.2 and marked implementation baseline. |
| Canonical schemas | Added `17_VRIA_Canonical_Schemas_and_Enums.md`; downstream docs reference it as authoritative. |
| Approval workflow | Added `18_VRIA_Approval_Workflow_Spec.md`; approval is now a first-class lifecycle, not a note. |
| Physical schema | Added `19_VRIA_Physical_Data_Model.md` with PostgreSQL table structure, versioning, audit, and RLS guidance. |
| Scoring logic | Added `20_VRIA_Scoring_Rules_Spec.md` with executable gates, caps, value states, and recommendation rules. |
| API/events | Added `21_VRIA_API_and_Event_Contracts.md` with REST and event contracts. |
| Tool contracts | Rewritten with strict input/output contracts, approval tools, failure behavior, policy tier, timeout, audit, and A2A provenance. |
| Evidence discipline | Propagated `Regressed`, `net_value_check`, `initiative_cost_period`, `attribution_method`, known confounders, and sustainment checks. |

## Blueprint Alignment

The documentation follows the **AaraMinds Value Realization Intelligence Agent Blueprint**:

1. Portfolio Value & Intake Triage
2. Eval-First Product & Agent Behavior
3. Runtime, Tools & Governance Design
4. Pilot, ValueOps & Production Use

## Folder Structure

```text
vria/
├── 00_VRIA_Documentation_Index.md   # This index
├── CHANGELOG_v1_2.md                # Version history
├── gate-a-value/                    # 01-03: charter, intake, hypothesis
├── gate-b-behavior/                 # 04-07: PRD, agent design, evidence, golden evals
├── gate-c-runtime/                  # 08-11: architecture, tools, security, red team
├── gate-d-operations/               # 12-16: backlog, pilot, runbook, dashboard, readiness
├── contracts/                       # 17-21: schemas, approval, data model, scoring, API
├── internal/                        # 99: source inventory — do not distribute
└── impl/                            # Go implementation: scoring, approval, registry, golden evals
```

## Documentation Map

| Gate | Document | Purpose |
|---|---|---|
| Foundation | `00_VRIA_Documentation_Index.md` | Pack index and change baseline. |
| Gate A | `gate-a-value/01_VRIA_Value_Charter_BRD.md` | Business problem, outcomes, scope, stakeholders, success metrics. |
| Gate A | `gate-a-value/02_VRIA_Portfolio_Intake_Model.md` | Use-case intake, normalization, tiering, and prioritization model. |
| Gate A | `gate-a-value/03_VRIA_Value_Hypothesis_Template.md` | Standard value hypothesis template. |
| Gate B | `gate-b-behavior/04_VRIA_PRD.md` | Product requirements, personas, journeys, functional and non-functional requirements. |
| Gate B | `gate-b-behavior/05_VRIA_Agent_Design_Spec.md` | Agent role, autonomy, behavior rules, memory/context policy, approval boundaries. |
| Gate B | `gate-b-behavior/06_VRIA_Value_Evidence_Model.md` | Evidence, attribution, net value, freshness, lineage, and confidence model. |
| Gate B | `gate-b-behavior/07_VRIA_Golden_Eval_Set.md` | Continuous golden eval set for normal, edge, and regression behavior. |
| Gate C | `gate-c-runtime/08_VRIA_Technical_Solution_Architecture.md` | Logical and deployment architecture. |
| Gate C | `gate-c-runtime/09_VRIA_MCP_A2A_Tool_Contracts.md` | Strict MCP/A2A tool contracts and approval tools. |
| Gate C | `gate-c-runtime/10_VRIA_Security_Governance_Model.md` | RBAC, policy tiers, prompt-injection defense, audit, and re-review triggers. |
| Gate C | `gate-c-runtime/11_VRIA_Red_Team_and_Evaluation_Harness.md` | Security, policy, adversarial, and production eval harness. |
| Gate D | `gate-d-operations/12_VRIA_Implementation_Backlog.md` | Epics, stories, milestones, dependencies, and DoD. |
| Gate D | `gate-d-operations/13_VRIA_Pilot_Plan.md` | Pilot scope, cadence, acceptance criteria, and go/no-go model. |
| Gate D | `gate-d-operations/14_VRIA_Operations_Runbook.md` | Production operations, incidents, rollback, releases, and support. |
| Gate D | `gate-d-operations/15_VRIA_ValueOps_Dashboard_Spec.md` | Dashboard views, scorecards, evidence gaps, decisions, and eval health. |
| Gate D | `gate-d-operations/16_VRIA_Production_Readiness_Checklist.md` | Final production readiness checklist. |
| Cross-cutting | `contracts/17_VRIA_Canonical_Schemas_and_Enums.md` | Authoritative enums, JSON schemas, and payload definitions. |
| Cross-cutting | `contracts/18_VRIA_Approval_Workflow_Spec.md` | First-class approval workflow and tool contracts. |
| Cross-cutting | `contracts/19_VRIA_Physical_Data_Model.md` | PostgreSQL physical data model, audit, RLS, indexes, and migrations. |
| Cross-cutting | `contracts/20_VRIA_Scoring_Rules_Spec.md` | Executable scoring formulas, caps, state mapping, and recommendations. |
| Cross-cutting | `contracts/21_VRIA_API_and_Event_Contracts.md` | REST APIs, event contracts, and integration payloads. |
| Internal-only | `internal/99_Source_AI_Use_Case_Inventory.md` | Source inventory snapshot. Keep internal. |

## Authoritative Document Rules

| Topic | Authoritative Document |
|---|---|
| Enums and payload schemas | `contracts/17_VRIA_Canonical_Schemas_and_Enums.md` |
| Approval states and transitions | `contracts/18_VRIA_Approval_Workflow_Spec.md` |
| Tables, keys, indexes, and audit persistence | `contracts/19_VRIA_Physical_Data_Model.md` |
| Score calculations and recommendation logic | `contracts/20_VRIA_Scoring_Rules_Spec.md` |
| REST APIs and events | `contracts/21_VRIA_API_and_Event_Contracts.md` |
| Tool and A2A contracts | `gate-c-runtime/09_VRIA_MCP_A2A_Tool_Contracts.md` |
| Evidence model and value claims | `gate-b-behavior/06_VRIA_Value_Evidence_Model.md` |

When a conflict exists, the authoritative document wins.

## Enterprise Acceptance Bar

No duplicate enums. No vague `{}` schemas. No approval action without workflow state. No tool without strict contract. No score without executable logic. No value claim without evidence, attribution, net-value check, and approval state.
