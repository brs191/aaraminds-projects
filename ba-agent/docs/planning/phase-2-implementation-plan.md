# BA Agent Phase 2 Implementation Plan

Decision-grade, implementation-oriented planning baseline for the first Phase 2 Enterprise BA slice. This plan follows the gate-first discipline of the Phase 1 project development plan, but it uses **Phase 2 gate names (`P2-G*`)**. It is not the earlier fleet tag `[F2]` for the MVP synthetic standup thin slice.

This document does not authorize live integrations, non-synthetic data use, production deployment, autonomous approval, or system-of-record updates.

---

## 1. Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Phase 2 Implementation Plan |
| Version | 0.4 |
| Change note (v0.4) | Recorded post-`P2-G5` scope-change addendum: HLD creation moves to a new draft/advisory `[F9]` lane without authorizing sandbox/live/non-synthetic paths. |
| Change note (v0.3) | Aligned gate dependencies and cross-doc wording: moved reviewer-delegate decision out of P2-G0 dependency, synchronized evaluation/governance references, and tightened execution-tracking coherence across Phase 2 artifacts. |
| Change note (v0.2) | Added execution-readiness sections for architecture changes, rollout, rollback, and documentation-control operations with traceability-matrix and decision-log update rules. |
| Status | Execution-ready baseline; first slice complete; HLD scope-change addendum active for `[F9]` |
| Prepared date | 2026-07-06 |
| Accountable owner | RAJA |
| Primary scope baseline | Phase 2 first slice: synthetic requirement discovery; post-`P2-G5` HLD creation addendum |
| Fixed constraint | First-slice scope, not date or capacity |
| Variable constraints | Calendar dates, team capacity, reviewers, tool scopes, non-synthetic data approval, pilot scope, and thresholds are `[RAJA]` |
| Execution authorization | This plan authorizes **planning/scaffold execution for `P2-G1` under synthetic-only, no-live, no-write guardrails** |
| `P2-G0` status | Accepted (2026-07-06) |
| `P2-G0` acceptance evidence | RAJA directive recorded on 2026-07-06: "go ahead an fill the gaps. make it execution-redy" |
| Non-authorization | No live integration, non-synthetic data, production deployment, autonomous approval, system-of-record update, or unapproved write-like side effect |
| Naming note | `P2-G0` through `P2-G5` are Phase 2 implementation-plan gates; they are not the prior `[F2]` fleet stage |

### Source register

| Plan source ID | Document | How this plan uses it |
| --- | --- | --- |
| P2-SRC-COPILOT | `.github/copilot-instructions.md` | Repository conventions: evidence discipline, `[inferred]`/`[RAJA]` markers, Teams/Copilot 365 surface, Azure-primary/JFrog/GitHub Actions OIDC/Terraform AzureRM conventions where relevant, no unapproved write-like behavior |
| P2-SRC-PDP | `docs/planning/project-development-plan.md` | Phase 1-style gate discipline, fixed-constraint planning, WBS pattern, critical-path and risk-register structure |
| P2-SRC-DEC | `docs/planning/decision-log.md` | RAJA accountability, synthetic-only build-start precedent, no-live/no-write default, approval-ref semantics baseline |
| P2-SRC-G7 | `docs/development/g7-readiness-review.md` | Historical G7 readiness prerequisite defining the required separate-plan sections and carry-forward guardrails prior to `P2-G0` acceptance |
| P2-SRC-PRIORITY | `docs/development/phase-2-prioritization-brief.md` | Recommended first capability set, Phase 2 capability ordering, stable project-context memory baseline, review lanes |
| P2-SRC-TOOLS | `docs/development/phase-2-tool-approval-matrix.md` | Default blocked tool posture, candidate Phase 2 tools, write-like action policy, approval evidence requirements |
| P2-SRC-DATA | `docs/development/phase-2-data-classification-plan.md` | Synthetic-only data posture, classification rules, source metadata expectations, redaction/retention open decisions |
| P2-SRC-GTS | `docs/development/gts-p2-req-evaluation-approach.md` | GTS-P2-REQ case format, minimum synthetic cases, expected output characteristics, BA-EM metric mapping |
| P2-SRC-REQ | `docs/requirements/business-analyst-agent-requirements.md` | Phase 2 functional requirements `BA-P2-FR-*`, human controls, data/security requirements, quality gates, risks, dependencies, open questions |
| P2-SRC-ARCH | `docs/requirements/ba_agent_runtime_architecture.md` | Proposed runtime control pattern, gateway boundary, Azure-primary deployment conventions where infrastructure is later relevant |
| P2-SRC-EVAL | `docs/requirements/ba_agent_evaluation_harness.md` | BA-EM metrics, GTS-P2-REQ baseline, hard gates for approval bypass and MVP/Phase 2 separation |
| P2-SRC-PROMPTS | `prompts.md` | Execution guardrails, completed Phase 2 readiness prompt lineage, no old evidence marker, staged-gate discipline |
| P2-SRC-FLEET | `fleet_prompt.md` | Fleet coordination model, gate-stop discipline, coordinator vs. lane ownership |

### Evidence discipline

- Source-backed claims cite the checked-in document or requirement IDs that support them.
- Reasonable implementation choices that are not directly supported are marked `[inferred]`.
- Owner-dependent values, thresholds, dates, scopes, reviewers, and approvals are marked `[RAJA]`.
- No numeric quality threshold is invented; owner thresholds remain `[RAJA]`.
- All Phase 2 output remains draft/advisory until human review and approval.

### G7 required-section coverage

| Required section from `P2-SRC-G7` | Coverage in this plan |
| --- | --- |
| Scope | Section 2 |
| Gates | Section 4 |
| Architecture changes | Section 10 |
| Tool validation | Sections 4, 9, and 10 |
| Data handling | Sections 2, 4, 9, and 11 |
| Evaluation | Section 9 |
| Support/RACI | Sections 5 and 13 |
| Rollout | Section 11 |
| Rollback | Section 12 |
| Documentation control | Section 13 |

---

## 2. Scope statement

This plan covers the **first Phase 2 implementation slice** only. `P2-G0` is accepted for synthetic-first execution, so this document is now the active execution baseline for `P2-G1` through `P2-G5` gate progression.

After `P2-G0`, the authorized execution boundary is **planning/scaffold and synthetic-only thin-slice work** for requirement discovery. The slice may define local schemas, local route/scaffold placeholders, synthetic fixtures, synthetic GTS-P2-REQ evaluation cases, and draft/advisory output structures. It must not enable live tools, process real data, publish artifacts, update systems of record, or imply production readiness.

### In scope for the first Phase 2 slice

| Capability | Requirement alignment | Scope note |
| --- | --- | --- |
| Requirement discovery from synthetic rough inputs | `BA-P2-FR-001`, `BA-P2-FR-002`, `BA-P2-FR-016` | Meeting-note, support-ticket, product-idea, regulatory-summary, and process-pain-point fixtures are fictional and minimal |
| Fact / assumption / `[inferred]` / open-question separation | `BA-NFR-001`, `BA-NFR-003`, `BA-AC-PROD-001` | Facts need evidence refs; missing rules become questions |
| Stakeholder clarification questions | `BA-P2-FR-009`, `BA-P2-FR-016` | Questions are routed to review lanes, not answered by the agent without evidence |
| Risk, dependency, conflict, and unresolved-decision surfacing | `BA-P2-FR-002`, `BA-P2-FR-016` | The output must expose uncertainty instead of smoothing it |
| Stable project-context memory schema | `BA-P2-FR-014`, `BA-DEP-010` | Unknown values remain `[RAJA]`; no hidden inference of business rules |
| Traceability skeleton | `BA-P2-FR-011` | Skeleton from input evidence to objective, draft requirement, and draft story candidate |
| Synthetic GTS-P2-REQ evaluation | `BA-QG-007`, `BA-EM-*` | Synthetic-only cases; owner thresholds remain `[RAJA]` |
| Human review routing | `BA-HIL-003`, `BA-HIL-004`, `BA-HIL-005` | BA SME, Product Owner, QA, architect, security/privacy, compliance/legal reviewers remain `[RAJA]` unless named later |

### Explicitly out of scope

| Out-of-scope item | Reason |
| --- | --- |
| Live Jira, Confluence, GitHub, Azure DevOps, SharePoint, Teams, SQL/Data, ServiceNow, Miro/Draw.io, or test-management integrations | All Phase 2 tools default blocked until owner, security/privacy, platform, scope, and schema validation are complete |
| Non-synthetic data | Classification handling, retention, residency, redaction, and allowed data classes are `[RAJA]` |
| Real meeting notes, emails, tickets, customer requests, source code, restricted documents, credentials, or production data | Data/classification plan blocks these until approval |
| BRD/FRD/PRD full generation | Later slice; first slice may create requirement-discovery summaries and trace skeletons only |
| Full user-story, acceptance-criteria, or test-case generation | Later slice; first slice may include draft story candidates only as trace skeleton nodes |
| Process maps, gap analysis, and impact analysis | Later slices; first slice may flag a process/gap/impact candidate without producing the artifact |
| HLD generation | Currently out of scope unless RAJA explicitly adds it later as Phase 2 scope |
| Production deployment or live pilot | Requires later readiness package and explicit RAJA approval |
| Autonomous approval or system-of-record update | Human-only decision; every external side effect is write-like and approval-gated |
| Teams sends, Confluence drafts/publishes, Jira updates, comments, approval-record creation, or webhook subscriptions | Treated as write-like side effects unless explicitly local/test-only and approved by gate |

### Post-`P2-G5` HLD scope-change addendum

RAJA explicitly moved HLD creation into the active focus on 2026-07-13. This creates a new `[F9]` HLD lane governed by `docs/planning/phase-2-hld-creation-plan.md`.

This addendum supersedes the HLD exclusion only for a **draft/advisory, repository-evidence-only HLD deliverable**. It does not authorize sandbox execution, live integrations, non-synthetic data use, production deployment, external publishing, autonomous approval, system-of-record updates, or write-like side effects.

The HLD lane must preserve the first-slice evidence discipline: source-backed claims cite repository evidence; unsupported architecture interpretations use `[inferred]`; owner-dependent decisions use `[RAJA]`.

---

## 3. Phase 2 first-slice strategy

Use a **synthetic-first, schema-first, evidence-first, gate-first** approach:

1. **Synthetic-first:** build and evaluate with fictional fixtures only. No non-synthetic input enters prompts, logs, fixtures, generated artifacts, or eval output.
2. **Schema-first:** define the requirement-discovery output contract, project-context memory schema, and traceability skeleton before broadening generation behavior.
3. **Evidence-first:** every factual claim carries an evidence ref and source metadata where available; unsupported material is separated as `[inferred]`, assumptions, open questions, conflicts, risks, or `[RAJA]` decisions.
4. **Human-gated:** requirement outputs are advisory/draft artifacts. The agent never approves requirements, resolves stakeholder conflicts, sets priority, accepts scope, or records system-of-record changes.
5. **Tool/data default deny:** all Phase 2 tools and data classes remain blocked until `P2-G4` or later approval evidence exists.
6. **MVP isolation:** Phase 2 routes, prompts, fixtures, and outputs must not leak into MVP standup/planning/retro/health behavior. BA-EM-009 remains a hard gate.
7. **Write-like side effects fail closed:** sends, publishes, comments, draft creation in external systems, approval records, subscriptions, and updates are write-like and approval-gated. BA-EM-005 remains a hard gate.

### Fixed constraint and trade-off statement

The fixed constraint is the first-slice capability set. Date and capacity are not fixed. If RAJA later fixes a date, then scope or capacity must move; quality, evidence discipline, data safety, and human-control gates are not levers.

### Critical path

The critical path is:

1. `P2-G0` RAJA plan acceptance and first-slice confirmation.
2. Technical baseline/scaffold that preserves MVP/Phase 2 separation.
3. Project-context memory schema and requirement-discovery output contract.
4. Synthetic fixture/case design for GTS-P2-REQ.
5. Requirement-discovery thin slice producing evidence-linked draft/advisory output.
6. Evaluation/control hardening for BA-EM metrics and hard gates.
7. Tool/data readiness decisions before any non-synthetic or external integration path.
8. Candidate review and pilot/readiness stop.

Adding capacity outside this chain will not shorten the Phase 2 readiness date unless it removes a dependency on this chain.

### Replan triggers

Replan if any of the following occurs:

- First-slice scope expands to BRD/FRD/PRD, process maps, gap/impact analysis, HLD, or full story/acceptance-criteria generation.
- RAJA fixes a date without changing scope or capacity.
- Any non-synthetic input is requested before classification approval.
- Any Phase 2 tool is requested before tool approval and schema validation.
- BA-EM-009 detects MVP/Phase 2 leakage.
- BA-EM-005 detects unapproved write/tool behavior.
- Review lanes or owner thresholds remain unavailable at the gate that needs them.

---

## 4. Phase 2 gates

No Phase 2 build may proceed without `P2-G0` acceptance by RAJA. This baseline includes that acceptance for synthetic-first execution. These gates are Phase 2 gates, not the earlier `[F2]` fleet tag.

| Gate | Objective | Key deliverables | Exit criteria | Critical dependencies |
| --- | --- | --- | --- | --- |
| **P2-G0 — Plan acceptance** | Establish Phase 2 build authority for planning/scaffold only. | Accepted or amended Phase 2 implementation plan; confirmed first slice; explicit non-authorization list; HLD scope decision recorded as out unless RAJA changes it. | RAJA accepts this plan or records deviations; no live integration, non-synthetic data, production, or write-like side effect is authorized. | P2-SRC-G7; P2-SRC-PRIORITY; RAJA decision |
| **P2-G1 — Technical baseline/scaffold** | Create/confirm a safe local scaffold for Phase 2 requirement-discovery work. | Phase 2 route/scaffold design `[inferred]`; local-only synthetic fixture paths; output schema draft; memory schema draft; no live clients; no credentials. | Local scaffold is isolated from MVP behavior; no live tools or external side effects; source/docs identify Phase 2 as draft/advisory. | `P2-G0`; existing MVP separation controls |
| **P2-G2 — Synthetic requirement-discovery thin slice** | Prove one end-to-end synthetic rough-input-to-discovery-output path. | GTS-P2-REQ seed cases; requirement-discovery output; facts/assumptions/`[inferred]`/open questions; stakeholder questions; risks/dependencies; trace skeleton. | Synthetic cases produce required sections; facts cite evidence refs; missing business rules become open questions; no BRD/FRD/PRD/process map/HLD generation. | `P2-G1`; P2-SRC-GTS; P2-SRC-DATA |
| **P2-G3 — Evaluation/control hardening** | Make Phase 2 quality and safety measurable. | BA-EM metric capture for P2 cases; regression cases for conflict/missing-rule/traceability; hard-gate tests for BA-EM-009 and BA-EM-005. | BA-EM-009 = 0; BA-EM-005 = 0; unmarked unsupported claims are escalated under the owner rule; owner thresholds remain `[RAJA]` unless set. | `P2-G2`; P2-SRC-EVAL; QA/AI evaluation lane `[RAJA]` |
| **P2-G4 — Tool/data readiness** | Decide whether any non-synthetic or external-tool path can be prepared later. | Tool approval matrix updates `[RAJA]`; classification/retention/residency decisions `[RAJA]`; schema validation plan; blocked-default register. | No Phase 2 tool enabled unless owner, security/privacy, platform, scope, rate-limit, and schema validation evidence exists; non-synthetic data remains blocked unless explicitly approved. | `P2-G3`; P2-SRC-TOOLS; P2-SRC-DATA |
| **P2-G5 — Candidate review and pilot/readiness stop** | Review the first-slice candidate and decide continue/adjust/stop. | Candidate review package; eval run summary; human review findings; risk/dependency updates; rollback/disable notes for Phase 2 route/scaffold. | RAJA decides continue, adjust, or stop. This gate does not authorize production. Any pilot, sandbox, or non-synthetic readiness requires explicit RAJA approval and satisfied `P2-G4` evidence. | `P2-G4`; BA SME/Product Owner/QA/security/privacy/architect review lanes `[RAJA]` |

---

## 5. Work breakdown structure

RAJA is the accountable owner for all work packages. Execution and review lanes are placeholders until RAJA names delegates.

| WBS ID | Work package | Deliverables | Dependencies | Accountable owner | Execution / review lanes | Exit criteria |
| --- | --- | --- | --- | --- | --- | --- |
| P2-WBS-00 | Plan acceptance and decision setup | Accepted/amended plan; Phase 2 decision register entries `[RAJA]`; first-slice confirmation | None | RAJA | Delivery lead, Product Owner, BA SME | `P2-G0` accepted or deviations recorded |
| P2-WBS-01 | Technical baseline/scaffold | Local-only Phase 2 scaffold `[inferred]`; route names; fixture directories; no-live/no-write guardrails | P2-WBS-00 | RAJA | AI engineer, platform engineer, architect | No live clients, credentials, external sends, or MVP route leakage |
| P2-WBS-02 | Synthetic fixture and case design | P2REQ synthetic cases; source metadata; expected output characteristics | P2-WBS-01 | RAJA | QA / AI evaluation reviewer, BA SME | Minimum GTS-P2-REQ cases are reviewable and fictional |
| P2-WBS-03 | Requirement-discovery output contract | Structured draft/advisory template with facts, assumptions, `[inferred]`, open questions, risks, dependencies, conflicts, trace links | P2-WBS-01 | RAJA | BA SME, Product Owner, AI engineer | Required sections are explicit and machine-checkable where practical |
| P2-WBS-04 | Project-context memory schema | Stable context schema with `[RAJA]` unknowns; no inferred business rules | P2-WBS-03 | RAJA | BA SME, architect, security/privacy | Unknowns stay `[RAJA]`; context source/owner fields present |
| P2-WBS-05 | Traceability skeleton | Evidence → objective → draft requirement → draft story candidate links | P2-WBS-03, P2-WBS-04 | RAJA | BA SME, QA, architect | Trace IDs and evidence refs appear in every output |
| P2-WBS-06 | Clarification and uncertainty handling | Stakeholder questions, unresolved decisions, conflict handling, risk/dependency surfacing | P2-WBS-03 | RAJA | BA SME, Product Owner, compliance/legal `[RAJA]` | Missing rules become questions; conflicts are preserved, not resolved by the agent |
| P2-WBS-07 | Synthetic thin-slice execution | Synthetic rough input processed into requirement-discovery output | P2-WBS-02 through P2-WBS-06 | RAJA | AI engineer, BA SME | `P2-G2` exit criteria met |
| P2-WBS-08 | Evaluation harness / metrics | BA-EM mapping; GTS-P2-REQ run records; structure/evidence/citation checks `[inferred]` | P2-WBS-07 | RAJA | QA / AI evaluation reviewer | `P2-G3` exit criteria met; thresholds `[RAJA]` unless set |
| P2-WBS-09 | Control and write-safety review | Proof that no external write-like action is reachable; Phase 2 route does not affect MVP | P2-WBS-07, P2-WBS-08 | RAJA | Security reviewer, QA, architect | BA-EM-005 = 0 and BA-EM-009 = 0 |
| P2-WBS-10 | Tool/data readiness review | Tool matrix updates; data/classification decisions; schema validation plan | P2-WBS-09 | RAJA | Tool owners, security/privacy, platform | No tool/data enablement without approval evidence |
| P2-WBS-11 | Candidate review package | Summary of scope delivered, eval evidence, risks, decisions, non-authorizations, continue/adjust/stop recommendation | P2-WBS-08 through P2-WBS-10 | RAJA | Delivery lead, Product Owner, BA SME, QA, architect, security/privacy | `P2-G5` review ready |
| P2-WBS-12 | Architecture delta implementation notes | Route/graph/schema/gateway delta notes aligned to Section 10 | P2-WBS-01 through P2-WBS-05 | RAJA | Architect, AI engineer, platform engineer | Section 10 checklist satisfied and reviewed |
| P2-WBS-13 | Rollout readiness package | Staged rollout evidence for R0/R1/R2 boundaries and blocked actions | P2-WBS-09, P2-WBS-10 | RAJA | Delivery lead, QA, security/privacy, platform | Rollout stage criteria documented with gate evidence |
| P2-WBS-14 | Rollback drill and disable readiness | Trigger coverage and rollback runbook evidence aligned to Section 12 | P2-WBS-09 | RAJA | QA, architect, platform engineer | Rollback triggers and procedure reviewed and executable |
| P2-WBS-15 | Documentation-control operations | Traceability matrix initialization and decision-log update discipline | P2-WBS-00 and ongoing | RAJA | Delivery lead, BA SME, QA | Section 13 update rules are active and current |

---

## 6. First Phase 2 thin-slice plan: synthetic requirement discovery

### Objective

Process a fictional rough business input into a **draft/advisory requirement-discovery summary** that separates evidence-backed facts from assumptions, `[inferred]` items, open questions, conflicts, risks, dependencies, and unresolved decisions. The output includes a stable project-context memory object and traceability skeleton, but it does not produce full BRD/FRD/PRD, final stories, final acceptance criteria, process maps, gap analysis, impact analysis, or HLD.

### Expected synthetic inputs

| Input type | Example shape | Default handling |
| --- | --- | --- |
| Synthetic meeting notes | Vague stakeholder discussion with goals, concerns, and missing rules | Extract supported facts; preserve ambiguity |
| Synthetic support-ticket cluster | Fictional operational pain points and impacted users | Surface process issue, risk, dependency, and questions |
| Synthetic conflicting stakeholder statements | Two supported but inconsistent statements | Preserve conflict; route to Product Owner/BA SME/compliance as needed |
| Synthetic missing business rules | Request with absent eligibility, approval, audit, or data rules | Ask clarification questions; do not invent rules |
| Synthetic regulatory-change summary | Fictional compliance-triggering change | Flag legal/privacy/audit review lane; do not approve obligations |
| Synthetic product idea | Outcome hypothesis with partial stakeholder context | Draft objective, draft requirement candidate, and trace skeleton |
| Synthetic process pain point | Current-state issue with possible future-state need | Flag as later process/gap candidate; do not generate a process map |
| Synthetic tool-origin evidence metadata | Fictional Jira/Confluence/Teams-style refs with source owner/timestamps/classification | Preserve metadata and identify source conflicts/staleness |

### Expected outputs

Every output is labeled **synthetic**, **draft**, and **advisory** and includes:

1. `trace_id`, case ID, fixture version, generated artifact version, and route.
2. Source register for the synthetic case.
3. Source metadata: source system/document, source owner where available, source timestamp/retrieved timestamp, and classification label where available.
4. Business problem and objective, only where supported.
5. Stakeholders and target users, only where supported or `[RAJA]`.
6. Current-state issues and desired future state, only where supported.
7. Facts with evidence refs.
8. Assumptions separated from facts.
9. `[inferred]` items separated from facts and assumptions.
10. Conflicts and unresolved decisions.
11. Open stakeholder clarification questions.
12. Risks and dependencies.
13. Draft requirement candidates.
14. Draft story candidate skeletons where useful for traceability.
15. Traceability skeleton.
16. Human review lanes.
17. Explicit non-approval statement.

### Source and evidence discipline

| Output category | Required evidence behavior |
| --- | --- |
| Facts | Must cite synthetic evidence refs such as `eval:P2REQ-001` and source metadata refs |
| Assumptions | Must be labeled as assumptions and routed for review if material |
| `[inferred]` items | Must be explicitly marked `[inferred]` and not treated as accepted truth |
| Open questions | Must identify the decision owner where known, otherwise `[RAJA]` |
| Conflicts | Must list conflicting source statements without deciding which is correct |
| Risks/dependencies | Must name the source signal or mark unsupported detail `[inferred]` |
| Draft requirements/stories | Must remain draft/advisory and trace to objective/evidence |

### Project-context memory schema

The first slice defines the schema, not a live persistent enterprise memory. Unknown values remain `[RAJA]`.

| Field | Initial handling |
| --- | --- |
| `project_name` | Synthetic case value or `BA Agent [RAJA if renamed]` |
| `business_domain` | `[RAJA]` unless present in synthetic case |
| `stakeholders` | Synthetic actors only; real people blocked |
| `target_users` | Synthetic actors only or `[RAJA]` |
| `source_systems` | Synthetic source names only |
| `delivery_methodology` | `[RAJA]` unless synthetic case states it |
| `known_business_rules` | Only source-supported rules; never inferred |
| `constraints` | Source-supported constraints plus `[RAJA]` unknowns |
| `definition_of_ready` | `[RAJA]` |
| `definition_of_done` | `[RAJA]` |
| `jira_project_key` | Synthetic placeholder only; real key blocked |
| `confluence_space` | Synthetic placeholder only; real space blocked |
| `approved_artifact_templates` | `[RAJA]` |
| `classification_label` | Synthetic label or `[RAJA]`; non-synthetic labels require owner approval |
| `retention_rule` | `[RAJA]` |
| `context_owner` | `[RAJA]` |
| `last_reviewed_by` | `[RAJA]` |

### Traceability skeleton

The first slice maintains draft trace links only:

| Trace node | Example ID pattern | Meaning |
| --- | --- | --- |
| Input evidence | `p2-input:P2REQ-001:e1` | Synthetic evidence item |
| Business objective | `p2-obj:P2REQ-001:001` | Draft objective derived from evidence |
| Draft requirement candidate | `p2-req-draft:P2REQ-001:001` | Review-ready requirement candidate, not approved |
| Draft story candidate | `p2-story-draft:P2REQ-001:001` | Optional story skeleton, not accepted backlog scope |
| Open question | `p2-question:P2REQ-001:001` | Stakeholder clarification needed |
| Risk/dependency | `p2-risk:P2REQ-001:001` | Delivery or analysis risk/dependency |

No trace node is a system-of-record update. External publication or storage is out of scope until approved.

---

## 7. Critical decisions required by gate

Not every decision is required before the first scaffold. Decisions are due at the gate that depends on them.

| Decision ID | Decision | Current baseline | Needed by | Accountable owner |
| --- | --- | --- | --- | --- |
| P2-DEC-001 | Accept or amend this Phase 2 plan. | Accepted for execution-readiness update v0.3. | `P2-G0` | RAJA |
| P2-DEC-002 | Confirm the first Phase 2 capability set. | First slice confirmed as synthetic requirement discovery. | `P2-G0` | RAJA |
| P2-DEC-003 | Confirm HLD generation scope. | HLD generation remains out of first-slice scope. | `P2-G0`; revisit only by change decision | RAJA |
| P2-DEC-004 | Name review delegates for BA SME, Product Owner, QA, architect, security/privacy, compliance/legal, platform, and tool owners. | RAJA remains accountable; delegates `[RAJA]`. | `P2-G3`/`P2-G4` | RAJA |
| P2-DEC-005 | Approve project-context memory fields and ownership. | Schema proposed; values `[RAJA]`. | `P2-G1` / `P2-G2` | RAJA |
| P2-DEC-006 | Approve first-slice output contract and labels. | Draft/advisory synthetic output contract proposed. | `P2-G2` | BA SME / Product Owner `[RAJA]` |
| P2-DEC-007 | Approve GTS-P2-REQ case set and labeling RACI. | Minimum cases from P2-SRC-GTS. | `P2-G2` / `P2-G3` | QA / AI evaluation reviewer `[RAJA]` |
| P2-DEC-008 | Set owner thresholds for BA-EM metrics. | Thresholds remain `[RAJA]`; hard gates remain zero for BA-EM-005 and BA-EM-009. | `P2-G3` | RAJA and metric owners |
| P2-DEC-009 | Confirm Phase 2 tool priorities, scopes, and validation evidence. | All tools blocked by default. | `P2-G4` | Tool owners `[RAJA]` |
| P2-DEC-010 | Confirm classification, redaction, retention, and residency for any non-synthetic path. | Synthetic-only. | `P2-G4` | Security/privacy/platform `[RAJA]` |
| P2-DEC-011 | Decide whether any sandbox/pilot readiness should follow the candidate review. | Not authorized by this plan. | `P2-G5` | RAJA |
| P2-DEC-012 | Approve artifact storage/publishing policy. | No external storage/publish. | Before any write-like side effect | RAJA / tool owners `[RAJA]` |
| P2-DEC-013 | Approve architecture-change delta for first-slice route/graph/schema/gateway boundaries. | Architecture delta defined in Section 10. | `P2-G1` | RAJA / architect `[RAJA]` |
| P2-DEC-014 | Approve staged rollout boundary (R0/R1/R2) and sandbox-readiness criteria. | Synthetic-first rollout defined in Section 11. | `P2-G4` / `P2-G5` | RAJA |
| P2-DEC-015 | Approve rollback trigger handling and documentation-control operating rules. | Rollback and doc-control rules defined in Sections 12 and 13. | `P2-G3` onward | RAJA |

---

## 8. Risk register

| Risk ID | Risk | Likelihood | Impact | Mitigation | Trigger |
| --- | --- | --- | --- | --- | --- |
| P2-RISK-001 | Phase 2 scope expands into BRD/FRD/PRD, process maps, gap/impact analysis, HLD, or full story/acceptance-criteria generation before the first slice is proven. | Medium | High | Keep first slice limited to requirement discovery, trace skeleton, and GTS-P2-REQ; route scope additions to replan. | New work item asks for later-slice artifact generation before `P2-G5`. |
| P2-RISK-002 | Non-synthetic data is introduced through examples, fixtures, prompts, logs, or generated artifacts. | Medium | High | Use synthetic-only fixtures; block real notes/emails/tickets/source docs until classification approval. | Any input contains real names, real project keys, real tickets, credentials, customer identifiers, or production content. |
| P2-RISK-003 | Missing business rules are silently inferred as requirements. | Medium | High | Enforce facts/assumptions/`[inferred]`/open-question separation; make missing rules open questions. | Output states eligibility, approval, audit, privacy, or business policy without evidence. |
| P2-RISK-004 | Draft/advisory Phase 2 outputs are mistaken for approved requirements or backlog commitments. | Medium | High | Put draft/advisory and non-approval labels in every output; require human review lanes. | A generated requirement/story is copied into delivery planning as approved without review evidence. |
| P2-RISK-005 | Phase 2 behavior leaks into MVP routes or MVP release notes. | Medium | High | Maintain route isolation and regression tests; BA-EM-009 hard gate remains zero. | MVP standup/planning/retro/health behavior generates Phase 2 artifacts. |
| P2-RISK-006 | A tool call, Teams send, external draft, publish, comment, approval record, or subscription occurs without approval. | Low | Critical | Treat every external side effect as write-like; gateway/control fail-closed; BA-EM-005 hard gate remains zero. | Any write-like action succeeds without approved gate evidence. |
| P2-RISK-007 | Tool approvals are assumed from MVP readiness. | Medium | High | Require separate Phase 2 tool approval matrix and schema/scope validation. | A Phase 2 tool path is enabled because an MVP tool was previously validated. |
| P2-RISK-008 | Evaluation thresholds and reviewers remain unset, delaying candidate review. | Medium | Medium | Use hard gates immediately; mark owner thresholds `[RAJA]`; schedule threshold/reviewer decisions by `P2-G3`. | BA-EM metrics are computed but cannot be interpreted by owners. |
| P2-RISK-009 | Synthetic cases are too narrow and overfit the first output. | Medium | Medium | Use the minimum GTS-P2-REQ spread: basic, operational, conflict, missing-rule, regulatory, product idea, process pain, tool-origin metadata. | Candidate passes one case but fails conflict/missing-rule/source-metadata cases. |
| P2-RISK-010 | Rollback path is defined but not operationally exercised before a hard-gate breach. | Low | High | Keep rollback triggers/procedure in Section 12 and require rollback-readiness review in P2-WBS-14. | A gate breach occurs and owners cannot execute rollback in one pass. |
| P2-RISK-011 | Documentation drift breaks traceability between requirement IDs, outputs, eval cases, and decisions. | Medium | High | Enforce Section 13 documentation-control update rules and keep the Phase 2 traceability matrix current. | Gate review finds stale mappings, missing decision references, or unverifiable output lineage. |

---

## 9. Validation and evaluation plan

Phase 2 validation uses GTS-P2-REQ and BA-EM metrics. Owner-set thresholds remain `[RAJA]`. Hard gates are non-negotiable where defined.

### Evaluation by gate

| Gate | Evaluation focus | Evidence expected |
| --- | --- | --- |
| `P2-G1` | Scaffold safety and separation | No live clients, no credentials, no external sends, no MVP behavior changes, no unapproved tool paths |
| `P2-G2` | Synthetic requirement-discovery output | GTS-P2-REQ cases produce required draft/advisory sections with evidence refs and trace IDs |
| `P2-G3` | Metric capture and hard controls | BA-EM mapping, BA-EM-005 = 0, BA-EM-009 = 0, unsupported-claim review, output-structure checks |
| `P2-G4` | Tool/data readiness | Tool approvals remain blocked unless evidence exists; non-synthetic data remains blocked unless classification decisions exist |
| `P2-G5` | Candidate review | Eval summary, human review findings, risks/dependencies, continue/adjust/stop recommendation |

### GTS-P2-REQ minimum coverage

| Case | Purpose | Required behavior |
| --- | --- | --- |
| P2REQ-001 | Basic requirement discovery from vague synthetic meeting notes | Extract supported problem/objective/context and separate facts from assumptions |
| P2REQ-002 | Synthetic support-ticket cluster | Surface process issue, impacted users, open questions, risks, dependencies, and evidence refs |
| P2REQ-003 | Conflicting stakeholder statements | Preserve conflict; do not decide; route to Product Owner/BA SME/compliance where needed |
| P2REQ-004 | Missing business rules | Generate targeted clarification questions instead of inventing rules |
| P2REQ-005 | Regulatory-change summary | Flag legal/privacy/audit obligations for owner review; do not approve obligations |
| P2REQ-006 | Product idea | Produce draft objective, draft requirement, draft story skeleton, and trace links |
| P2REQ-007 | Process pain point | Identify process/gap candidate without generating final process map or gap analysis |
| P2REQ-008 | Tool-origin synthetic evidence | Preserve source system, owner, timestamps, classification, staleness/conflict, and evidence refs |

### BA-EM metric mapping

| Metric | Phase 2 use | Threshold / gate |
| --- | --- | --- |
| BA-EM-001 Routing accuracy | Requirement-discovery inputs route only to approved Phase 2 readiness/scaffold path after `P2-G0`. | `[RAJA]` |
| BA-EM-002 Evidence-link coverage | Factual claims in discovery outputs carry source/evidence refs. | `[RAJA]` |
| BA-EM-003 Unsupported-claim rate | Unmarked unsupported claims are identified through automated and sampled review. | `[RAJA]`; unmarked unsupported claims must be surfaced for owner review |
| BA-EM-005 Approval-gate bypass count | Any write-like tool behavior without valid approval evidence. | Hard gate = 0 |
| BA-EM-006 Citation correctness | Sampled evidence refs support the claims they annotate. | `[RAJA]` |
| BA-EM-007 Output-structure conformance | Required sections appear and are clearly separated. | `[RAJA]` |
| BA-EM-008 Regression coverage | GTS-P2-REQ cases executed on relevant changes. | `[RAJA]` |
| BA-EM-009 Phase-separation violations | MVP routes must not expose Phase 2 behavior before approval, and Phase 2 must remain isolated from MVP. | Hard gate = 0 |

### Hard gates

| Hard gate | Required result |
| --- | --- |
| MVP/Phase 2 separation | BA-EM-009 = 0 |
| Unapproved tool/write behavior | BA-EM-005 = 0 |
| Data safety | No non-synthetic input or output content until classification approval |
| Tool safety | No Phase 2 tool enabled without approval and validation evidence |
| Human approval | No generated artifact is treated as approved without human review |
| System-of-record safety | No system-of-record update, external draft, publish, send, comment, approval-record creation, webhook subscription, or other side effect without explicit approval path |

### Human review lanes

| Review lane | Review criteria |
| --- | --- |
| BA SME `[RAJA]` | Requirement clarity, ambiguity handling, business readability, fact/question separation |
| Product Owner `[RAJA]` | Business objective, scope, priority, stakeholder intent, decision ownership |
| QA / AI evaluation reviewer `[RAJA]` | GTS-P2-REQ case coverage, structure conformance, regression evidence |
| Architect `[RAJA]` | Trace chain, system/API/data implication flags, route isolation, future integration implications |
| Security/privacy owner `[RAJA]` | Classification handling, redaction, sensitive data blocking, retention/residency decisions |
| Compliance/legal owner `[RAJA]` | Regulatory/legal/audit flags; no agent-approved obligations |
| Tool owners `[RAJA]` | Tool scope, permissions, schema validation, rate limits, write policy |

---

## 10. Architecture changes and implementation deltas

This section defines the **first-slice technical delta** from the current MVP synthetic baseline. It is constrained to local synthetic behavior and must preserve MVP/Phase 2 isolation.

| Surface | Current baseline | Phase 2 first-slice delta | Gate ownership |
| --- | --- | --- | --- |
| Routing | MVP routes for standup/planning/retro/health only | Add a separate Phase 2 requirement-discovery route with explicit draft/advisory labeling and hard route isolation from MVP flows | `P2-G1`, `P2-G3` |
| Orchestration graph | MVP-oriented orchestration path | Add a requirement-discovery flow `[inferred]`: intake → evidence extraction → fact/assumption/`[inferred]` separation → conflict/open-question surfacing → trace assembly → draft packaging | `P2-G1`, `P2-G2` |
| Output contract | MVP output schemas/cards | Add a structured requirement-discovery output contract from Section 6, including evidence refs, source metadata, trace nodes, and non-approval statement | `P2-G1`, `P2-G2` |
| Project context memory | MVP context handling | Add first-slice project-context memory schema (Section 6), with unknowns marked `[RAJA]` and no inferred business rules | `P2-G1`, `P2-G2` |
| Gateway and control layer | Local synthetic gateway controls | Keep default-deny for all Phase 2 tools; no live adapters enabled; preserve approval-gate fail-closed behavior for any write-like action | `P2-G1`, `P2-G4` |
| Evaluation harness | MVP synthetic eval coverage | Add GTS-P2-REQ synthetic cases and BA-EM metric mapping for Phase 2 structure, citation discipline, and separation controls | `P2-G2`, `P2-G3` |
| Artifact templates | MVP-oriented artifacts | Add draft/advisory template for requirement-discovery summary; BRD/FRD/PRD/process map/HLD templates remain out of first-slice scope | `P2-G2`, `P2-G5` |

### Architecture-change acceptance checklist

1. Phase 2 route is isolated from MVP routes.
2. Requirement-discovery output is schema-defined and machine-checkable where practical.
3. No live tool client is enabled in first-slice code paths.
4. Evidence refs and trace IDs are required in first-slice outputs.
5. BA-EM-005 and BA-EM-009 hard gates remain enforceable.

## 11. Rollout plan (synthetic-first)

Rollout is staged and gated. Production rollout is explicitly excluded from this plan.

| Stage | Entry criteria | Allowed actions | Blocked actions | Exit artifacts |
| --- | --- | --- | --- | --- |
| R0 — Local synthetic execution | `P2-G0` accepted | Implement and run Phase 2 first-slice scaffold and synthetic evals | Any live integration, non-synthetic input, external side effects | `P2-G1` to `P2-G3` evidence package |
| R1 — Sandbox-readiness decision | `P2-G3` complete and `P2-G4` prep started | Produce tool-validation and classification decisions; update blocked-default register | Enabling tools/data without owner/security/platform evidence | `P2-G4` decision package |
| R2 — Sandbox pilot consideration | `P2-G4` approved by RAJA | Prepare a separate sandbox-readiness/pilot authorization package | Direct sandbox execution without explicit RAJA authorization | Candidate review package at `P2-G5` plus sandbox authorization decision `[RAJA]` |
| R3 — Production path | Not in scope for this plan | None | Any production deployment or live pilot execution | Separate post-Phase 2 decision artifact `[RAJA]` |

### Data-handling controls during rollout

1. Synthetic-only remains mandatory through R0 and R1.
2. Any non-synthetic path requires approved classification, retention, residency, and redaction decisions at `P2-G4`.
3. Any external side effect remains write-like and approval-gated even in sandbox-preparation stages.

## 12. Rollback and disable plan

Rollback applies to first-slice Phase 2 behavior only and must preserve audit/evidence history.

### Rollback triggers

| Trigger ID | Trigger condition | Required response |
| --- | --- | --- |
| P2-RB-001 | BA-EM-005 > 0 or any unapproved write-like behavior | Immediate Phase 2 route disable, incident review, gate stop |
| P2-RB-002 | BA-EM-009 > 0 or MVP/Phase 2 leakage | Immediate route isolation rollback and regression recheck |
| P2-RB-003 | Non-synthetic data found in first-slice path | Purge/contain artifact per approved policy `[RAJA]`, stop Phase 2 execution |
| P2-RB-004 | Material citation/traceability failure trend | Revert to last accepted synthetic baseline and reopen evaluation review |

### Rollback procedure

1. Disable Phase 2 route/scaffold entrypoint(s) and keep MVP routes active.
2. Revert Phase 2 prompt/graph changes to the last accepted synthetic-only baseline.
3. Keep Phase 2 tools in blocked-default state and re-assert no-live/no-write controls.
4. Preserve audit logs, eval results, and failing fixtures as investigation evidence.
5. Record rollback reason, scope, and owner action in decision-log updates (Section 13).

## 13. Support, RACI, and documentation control operations

### Support and review RACI (first slice)

| Lane | Responsibility | Escalation output |
| --- | --- | --- |
| RAJA (accountable owner) | Final gate decisions, scope control, threshold ownership, authorization boundaries | Gate decision record and decision-log updates |
| BA SME / Product Owner `[RAJA]` | Requirement quality, ambiguity handling, question quality, business intent checks | Review findings attached to `P2-G2`/`P2-G5` package |
| QA / AI evaluation reviewer `[RAJA]` | GTS-P2-REQ coverage, BA-EM metrics, regression outcomes | Evaluation summary and hard-gate verdict |
| Architect `[RAJA]` | Route isolation, trace chain integrity, architecture-change conformance | Architecture conformance note for `P2-G1`/`P2-G3` |
| Security/privacy/compliance lanes `[RAJA]` | Classification, redaction, retention/residency, obligation flags | `P2-G4` data/tool readiness findings |
| Platform/tool owners `[RAJA]` | Tool scope, auth, schema/rate-limit validation, blocked-default enforcement | Tool-validation record and enable/deny decision evidence |

### Documentation control operations

| Artifact | Required update trigger | Required update |
| --- | --- | --- |
| `docs/planning/phase-2-implementation-plan.md` | Gate decision, scope change, or control-model change | Bump version, add change note, update affected sections and gate criteria |
| `docs/planning/phase-2-traceability-matrix.md` | Requirement/output/eval case changes | Update requirement-to-output-to-eval mapping and change log |
| `docs/planning/decision-log.md` | Any `P2-DEC-*` closure, deferment, or rollback | Add/update Phase 2 decision entries with status and evidence |
| `prompts.md` | Prompt execution outcomes for Phase 2 tracks | Update prompt result status and one-line outcome summary |

### Decision-log update rule

For every closed, conditional, deferred, or rolled-back `P2-DEC-*` item, update `docs/planning/decision-log.md` in the same change set with: decision ID, outcome, status, gate, and evidence reference.

## 14. Immediate next steps

1. Start `P2-G1` using the architecture delta in Section 10 and maintain synthetic-only controls.
2. Keep `docs/planning/phase-2-traceability-matrix.md` current for first-slice requirement/eval mapping.
3. Run `P2-G2` synthetic thin-slice implementation with the Section 6 output contract.
4. Run `P2-G3` evaluation/control hardening and confirm BA-EM-005 = 0 and BA-EM-009 = 0.
5. Keep all tools/data blocked until `P2-G4` decision evidence is recorded.
6. Use Section 12 rollback procedure immediately on any hard-gate breach.
7. Record all gate and decision outcomes in `docs/planning/decision-log.md` as they are resolved.
