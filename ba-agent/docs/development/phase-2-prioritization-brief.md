# BA Agent Phase 2 Prioritization Brief

This brief prepares Phase 2 readiness planning for Enterprise BA capabilities. It does not authorize Phase 2 runtime implementation, live integrations, non-synthetic data use, or production deployment.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Phase 2 Prioritization Brief |
| Version | 0.1 |
| Status | Draft for RAJA/G7 readiness review |
| Prepared date | 2026-07-03 |
| Accountable owner | RAJA |
| Execution prompt | [P7A] |
| Requirement baseline | `docs/requirements/business-analyst-agent-requirements.md` v0.4 |
| MVP candidate evidence | `docs/development/g5-candidate-review.md` |
| G6 authorization status | `docs/development/g6-authorization-package.md` — live pilot blocked |

## Decision context

RAJA directed the project to park remaining MVP live-integration and pilot-execution items and move to Phase 2 readiness planning. Therefore, this brief uses G5/G6 readiness evidence and known blockers as inputs, not post-pilot findings. No live pilot findings exist because `[P6F]` and `[P6G]` remain blocked.

## Phase 2 scope boundary

Phase 2 is broader Enterprise BA readiness. It remains planning-only until RAJA approves a separate Phase 2 implementation plan. No runtime code, prompt behavior, integration, tool enablement, or generation capability is authorized by this brief.

## Candidate Phase 2 capabilities

| Capability | Requirement IDs | Description | Initial priority |
| --- | --- | --- | --- |
| Requirement discovery from rough inputs | BA-P2-FR-001, BA-P2-FR-002, BA-P2-FR-016 | Convert meeting notes, emails, tickets, customer requests, product ideas, process pain points, and regulatory changes into structured problem/objective/context/open-question outputs. | Recommended first slice [RAJA] |
| Business and functional requirement drafting | BA-P2-FR-003, BA-P2-FR-012, BA-P2-FR-016 | Draft business requirements, functional requirements, and BRD/FRD/PRD-style artifacts for human review. | Second slice [RAJA] |
| User stories and acceptance criteria | BA-P2-FR-004, BA-P2-FR-005, BA-P2-FR-006 | Generate agile-ready stories and Given/When/Then acceptance criteria with edge cases, dependencies, NFRs, data needs, and system touchpoints. | Second slice [RAJA] |
| Clarification questions and assumption control | BA-P2-FR-002, BA-P2-FR-009, BA-P2-FR-016 | Surface unresolved rules, approvals, data ownership, reporting, audit needs, risks, and dependencies before drafting final artifacts. | Recommended first slice [RAJA] |
| Traceability chain | BA-P2-FR-011, BA-P2-FR-016 | Maintain trace from objective to requirement, story, acceptance criteria, test case, and release item. | Recommended first slice [RAJA] |
| Stable project context memory | BA-P2-FR-014, BA-P2-FR-016 | Maintain approved project context such as project name, business domain, stakeholders, target users, systems, delivery methodology, known business rules, constraints, Definition of Ready, Definition of Done, Jira project key, and Confluence space. | Recommended first slice [RAJA] |
| Process mapping | BA-P2-FR-007 | Draft current/future process maps with actors, systems, decision points, manual steps, automation opportunities, exceptions, and controls. | Later slice [RAJA] |
| Gap analysis | BA-P2-FR-008 | Compare current and future state and generate gap/recommendation outputs for human review. | Later slice [RAJA] |
| Impact analysis | BA-P2-FR-010 | Analyze impacts across processes, roles, systems, data, APIs, reports, compliance, training, support, and downstream teams. | Later slice [RAJA] |
| Test scenario inputs | BA-P2-FR-013 | Produce QA/test-management scenario inputs that remain draft until QA review. | Later slice [RAJA] |
| Approved enterprise tool integrations | BA-P2-FR-015 | Candidate integrations include Jira, Confluence, GitHub, Azure DevOps, SharePoint, Teams, Miro/Draw.io, SQL/Data, ServiceNow, and test-management tools. | Approval-gated [RAJA] |

## Recommended first Phase 2 capability set

Recommended first slice [RAJA]:

1. Requirement discovery from synthetic rough inputs.
2. Fact / assumption / inference / open-question separation.
3. Stakeholder clarification questions.
4. Risks, dependencies, and unresolved-decision surfacing.
5. Traceability skeleton from objective to draft requirement and draft story.
6. Stable project context memory schema with all unknown values marked `[RAJA]`.
7. Synthetic-only GTS-P2-REQ evaluation cases.

Rationale:

- It directly covers BA-P2-FR-001, BA-P2-FR-002, BA-P2-FR-009, BA-P2-FR-011, BA-P2-FR-014, and BA-P2-FR-016.
- It establishes evidence and trace discipline before creating larger BRD/FRD/PRD artifacts.
- It can be evaluated with synthetic data only.
- It avoids early tool-integration risk.
- It preserves human approval and review boundaries.

## Inputs from MVP readiness

| Input | Phase 2 implication |
| --- | --- |
| MVP local/synthetic implementation passed hard gates. | Keep hard gates for approval bypass and phase separation in Phase 2. |
| MVP live pilot is parked. | Do not depend on live pilot telemetry or production feedback for first Phase 2 readiness planning. |
| Tool validation remains incomplete. | Start Phase 2 with synthetic inputs and local artifacts; do not enable enterprise tools by default. |
| GTS-GATE and route blocking are in place. | Reuse the control discipline so Phase 2 generation cannot leak into MVP routes. |
| Evidence refs and trace IDs are established patterns. | Preserve evidence refs and trace IDs in all Phase 2 outputs. |

## Dependencies

| Dependency | Required before |
| --- | --- |
| RAJA confirms first Phase 2 slice. | Any Phase 2 implementation prompt. |
| Security/privacy confirms classification handling. | Any non-synthetic Phase 2 input. |
| Tool owners approve integrations and scopes. | Any Phase 2 MCP access. |
| BA SME / Product Owner review rubric. | Requirement/story/acceptance-criteria output review. |
| GTS-P2-REQ evaluation approach. | Any generated Phase 2 artifact considered gate-ready. |

## Stable project context memory baseline

The first Phase 2 slice should define a project context object before generating requirements or stories. Unknown values remain `[RAJA]`; the agent must not infer missing business rules.

| Context field | Initial value |
| --- | --- |
| Project name | BA Agent [RAJA if renamed] |
| Business domain | [RAJA] |
| Stakeholders | [RAJA] |
| Target users | [RAJA] |
| Source systems | [RAJA] |
| Delivery methodology | [RAJA] |
| Known business rules | [RAJA] |
| Constraints | [RAJA] |
| Definition of Ready | [RAJA] |
| Definition of Done | [RAJA] |
| Jira project key | [RAJA] |
| Confluence space | [RAJA] |
| Approved artifact templates | [RAJA] |

## Risks

| Risk | Mitigation |
| --- | --- |
| Phase 2 scope expands too broadly. | Start with requirement-discovery readiness only; keep BRD/FRD/PRD and process maps behind later slices. |
| Generated artifacts look approved. | Label every output as draft/advisory until human approval. |
| Missing business rules are inferred. | Turn missing rules into open questions; use `[inferred]` only for explicitly unsupported reasoning. |
| Real stakeholder data leaks into prompts/evals. | Use synthetic data until classification handling is approved. |
| Tool approvals are assumed from MVP. | Require separate Phase 2 tool approval matrix. |

## Human review lanes

| Review lane | Purpose |
| --- | --- |
| RAJA | Accountable owner and phase-gate decision maker. |
| BA SME [RAJA] | Reviews requirement quality, ambiguity handling, and artifact usefulness. |
| Product Owner [RAJA] | Reviews scope, prioritization, and business objective framing. |
| QA / AI evaluation reviewer [RAJA] | Reviews GTS-P2-REQ cases and quality metrics. |
| Security/privacy owner [RAJA] | Reviews classification, retention, redaction, and non-synthetic input rules. |
| Architect [RAJA] | Reviews traceability, integration implications, and tool boundary impacts. |

## Explicit non-authorization

This brief does not authorize:

- Phase 2 runtime implementation.
- Requirement/story/BRD/FRD/PRD generation in production.
- Any live enterprise integration.
- Any non-synthetic data processing.
- Any autonomous approval or system-of-record update.
- HLD generation as a BA Agent capability.

## Next Phase 2 readiness artifacts

1. Phase 2 tool approval matrix.
2. Phase 2 data/classification plan.
3. GTS-P2-REQ evaluation approach.
4. Separate Phase 2 implementation plan readiness review.
