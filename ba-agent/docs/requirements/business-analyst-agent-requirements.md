# Business Analyst AI Agent Requirements

Decision-quality draft requirements for the AaraMinds Business Analyst AI Agent, separating the Agile/Scrum MVP from Phase 2 enterprise BA capabilities.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | Business Analyst AI Agent Requirements |
| Version | 0.4 |
| Change note (v0.4) | Companion set completed: `ba_agent_runtime_architecture.md` (proposed topology and security design) and `ba_agent_operations_model.md` (proposed ownership, support, and release model) added; tool contracts v0.2 (validation register) and evaluation harness v0.2 (sample golden cases). |
| Change note (v0.3) | Added capability autonomy classification and authoritative source-of-record mapping; companion documents referenced: `ba_agent_mcp_tool_contracts.md` (proposed tool contracts) and `ba_agent_evaluation_harness.md` (evaluation gates and golden test sets). |
| Change note (v0.2) | Task-prompt (S4) citations removed from requirement evidence and reclassified as task framing; LangGraph moved from functional requirement to design constraint; ID scheme aligned to actual usage; literal Title heading removed. |
| Status | Draft for human review; not approved for delivery commitment |
| Prepared date | 2026-07-02 |
| Prepared by | `aara-business-analyst` drafting assistant |
| Primary scope baseline | MVP: Agile/Scrum BA Agent for standups, sprint planning, retrospectives, sprint health, Teams/Copilot 365, Jira/Git/Confluence/Calendar MCP tools, and human approval gates |
| Phase 2 scope baseline | Broader Enterprise BA capabilities from `ref-requirements.md` |
| Review status | Requires review by Product Owner, Scrum Master/BA SME, delivery lead, architect, QA, security/privacy, and affected tool owners |
| Classification note | Source context includes internal/proprietary classification material. One optional classification summary states AT&T PROPRIETARY (RESTRICTED). Final handling classification must be confirmed by the appropriate owner. |

### Source register

| Source ID | Source | Review status | Evidence used |
| --- | --- | --- | --- |
| S1 | `/Users/rb692q/projects/brs191/CC_UserCases/BA Agent/BusinessAnalystAgent.md` | Reviewed as text | Business problem, solution concept, MVP capabilities, Teams/Copilot 365, LangGraph, Jira/Git/Confluence/Calendar usage, human approval gate, sprint health monitoring |
| S2 | `/Users/rb692q/projects/brs191/CC_UserCases/BA Agent/BusinessAnalystAgent-workflow-diagram.md` | Reviewed as text | Workflow, LangGraph router, four specialized nodes, MCP tools, Teams Adaptive Cards, Confluence output, sprint planning approval, health escalation |
| S3 | `/Users/rb692q/projects/aaraminds-projects/ba-agent/ref-requirements.md` | Reviewed as text | Phase 2 Enterprise BA capabilities, operating principles, artifacts, traceability chain, candidate enterprise MCP tools |
| S4 | `/Users/rb692q/projects/aaraminds-projects/ba-agent/ba-requirements-prompt.md` | Reviewed as task framing | Required phase separation, Aara operating model, evidence rules, quality gates, workspace conventions |
| S5 | Optional information-classification PDFs in `/Users/rb692q/projects/brs191/CC_UserCases/BA Agent/` | Reviewed through text extraction summary in this run | Internal/proprietary and restricted classification context; selected data categories include process/operational information, security information, and business-critical source code |
| S6 | `BA_Agent_Architecture.pptx`, `BA_Agent_Workflow.pptx` | Available but not reviewed due to format/content extraction limitation | No requirement evidence taken from these decks |

### Evidence notation

- Source citations use source IDs and, where available, reviewed line ranges such as `[S1:L5-L8]`.
- `[inferred]` marks a conclusion that is reasonable from the reviewed sources but not directly stated.
- `[RAJA]` marks a statement that needs source confirmation before it can become a requirement.
- `S4` is the generation-task prompt. It is task framing, not product evidence: it may be referenced for scope separation, governance discipline, and quality-gate definitions, but it is never cited as evidence for a functional or non-functional requirement.
- This document is a draft synthesis. It does not approve scope, priority, cost, timeline, architecture, compliance obligations, or system-of-record updates.

---

## Executive summary

AT&T agile teams currently rely on Business Analysts and Scrum Masters to manually gather updates from Jira, Git, Confluence, and calendars; synthesize Scrum ceremony outputs; identify blockers; and communicate through Teams. The source notes state that this manual approach is time-consuming, inconsistent across teams, and creates bottlenecks when BAs support multiple squads. Delayed detection of stalled stories, scope creep, and resource conflicts can affect timelines and rework, while retrospective insights are inconsistently captured and not reliably translated into improvements [S1:L5-L8].

The proposed MVP is an AI-powered Agile/Scrum BA Agent using LangGraph orchestration and Microsoft Copilot 365 via Teams. It focuses on four capabilities: standup summarization, sprint planning recommendations with Scrum Master approval, data-driven retrospectives, and continuous sprint health monitoring [S1:L9-L21; S2:L28-L47]. The MVP uses Teams/Copilot 365 as the user surface and MCP tools for Jira, Git, Confluence, and Calendar [S2:L30-L37].

Phase 2 expands beyond Scrum support into Enterprise BA capabilities from `ref-requirements.md`: requirement discovery, user story generation, acceptance criteria, process mapping, gap analysis, stakeholder questions, impact analysis, traceability, BRD/FRD/PRD drafts, and test scenario inputs [S3:L3-L24; S3:L25-L148; S3:L202-L219].

Key decision points remain open: final product naming, business owner/sponsor, approved data classifications, sprint-health severity rules, allowed write actions, quantitative success targets, and Phase 2 prioritization.

---

## Business problem

### Facts from reviewed sources

| Fact ID | Fact | Evidence |
| --- | --- | --- |
| BA-FACT-001 | Agile development teams rely on BAs and Scrum Masters to facilitate daily standups, sprint planning, retrospectives, and sprint health monitoring. | [S1:L5-L8] |
| BA-FACT-002 | Current Scrum-support activities are predominantly manual and require gathering updates from Jira, Git, Confluence, and calendars. | [S1:L5-L8] |
| BA-FACT-003 | Manual synthesis creates inconsistent outputs and bottlenecks when BAs manage multiple squads. | [S1:L5-L8] |
| BA-FACT-004 | Delayed detection of risks such as stalled stories, scope creep, and resource conflicts can affect delivery timelines and rework. | [S1:L5-L8] |
| BA-FACT-005 | Retrospective insights are captured inconsistently and are not consistently translated into actionable improvements. | [S1:L5-L8] |
| BA-FACT-006 | The Enterprise BA reference goal is to convert unclear business input into structured delivery artifacts while not replacing the BA. | [S3:L3-L24] |

### Problem statement

Agile teams need a governed, Teams-based BA assistance capability that reduces manual ceremony preparation, improves consistency of Scrum outputs, detects sprint risks earlier, and preserves human decision authority. Enterprise BA teams also need a later-phase capability to convert ambiguous business input into traceable delivery artifacts.

### Ambiguity and conflict findings

| Finding ID | Finding | Impact | Required resolution |
| --- | --- | --- | --- |
| BA-CF-001 | S1 title is “AI Business Analyst Agent,” while S2 title is “AI Scrum Master (Business Analyst Agent) Workflow.” | Product positioning and stakeholder expectations may diverge. | Confirm final product name and whether MVP is positioned as BA support, Scrum Master support, or both. |
| BA-CF-002 | S3 describes a broad Enterprise BA Agent and includes a “best first version” for requirement discovery/user stories/gap analysis, while S4 requires this document’s MVP to be Agile/Scrum-focused. | Risk of mixing MVP and Phase 2 scope. | This draft follows S4 phase framing and assigns S3 broad BA capabilities to Phase 2 pending human confirmation. |
| BA-CF-003 | Owner, sponsor, primary business entity, and NDA fields in S1 are blank. | Accountability, review routing, and approval path are unresolved. | Identify named owners before delivery planning. |

---

## Product vision

### MVP vision

Create a human-gated Agile/Scrum BA Agent that teams access through Copilot 365 in Teams. The agent uses LangGraph routing and approved MCP tools to automate data gathering, summarize standups, recommend sprint scope for human approval, generate retrospective reports, and monitor sprint health [S1:L9-L21; S2:L6-L26; S2:L28-L47].

### Phase 2 vision

Extend the product into an Enterprise BA drafting and analysis assistant that helps business users, product managers, architects, engineers, QA teams, and delivery leaders convert unclear business needs into structured, review-ready artifacts including requirements, user stories, acceptance criteria, process flows, gap analysis, impact analysis, traceability matrices, BRD/FRD/PRD drafts, and test scenario inputs [S3:L3-L24; S3:L202-L219].

### Product boundary

The agent drafts, summarizes, recommends, traces, and routes. It does not approve requirements, prioritize backlogs, accept sprint scope, make compliance commitments, or replace accountable humans [S3:L21-L24; S3:L221-L231].

---

## Goals and success criteria

Quantitative targets are not provided in the reviewed sources. Any numeric goals require owner confirmation before becoming acceptance criteria.

| Goal ID | Goal | Draft success criterion | Evidence | Status |
| --- | --- | --- | --- | --- |
| BA-G-001 | Reduce manual BA/Scrum Master overhead for Scrum ceremonies. | MVP can generate standup summaries and ceremony artifacts from approved tools without manual status collection for the in-scope team data. | [S1:L5-L8; S1:L13-L21] | Draft |
| BA-G-002 | Improve consistency of agile practice outputs across teams. | MVP outputs use repeatable templates and routing for standups, planning, retrospectives, and sprint health. | [S1:L21-L21; S2:L28-L47] | Draft |
| BA-G-003 | Surface sprint delivery risks earlier. | Sprint health monitoring identifies at-risk items and escalates high-severity blockers with suggested corrective actions. | [S1:L18-L18; S2:L37-L47] | Draft |
| BA-G-004 | Improve retrospective learning. | Retro reports include source sprint metrics and improvement recommendations and are posted to Confluence for organizational learning. | [S1:L17-L17; S2:L25-L25; S2:L45-L45] | Draft |
| BA-G-005 | Preserve human authority over planning and delivery decisions. | Sprint planning output cannot be published without Scrum Master review and approval. | [S1:L16-L16; S2:L36-L36] | Draft |
| BA-G-006 | Enable later Enterprise BA artifact generation. | Phase 2 can draft requirements, stories, acceptance criteria, process maps, gap/impact analysis, traceability, BRD/FRD/PRD, and test scenario inputs. | [S3:L3-L24; S3:L202-L219] | Phase 2 draft |

---

## Personas and stakeholders

### Source-supported personas and users

| Persona/stakeholder | Relationship to product | Evidence |
| --- | --- | --- |
| Business Analyst | Current facilitator and target beneficiary; the agent augments BA work and does not replace the BA. | [S1:L5-L8; S3:L21-L24] |
| Scrum Master | Reviews sprint planning recommendations and receives escalations. | [S1:L16-L18; S2:L36-L47] |
| Team member | Interacts through natural language in Teams. | [S1:L19-L19; S2:L41-L45] |
| Product manager / Product Owner | Phase 2 beneficiary for converting business needs into delivery artifacts; likely reviewer for scope and requirements [inferred]. | [S3:L7-L20; S3:L202-L219] |
| Business user | Phase 2 input provider and beneficiary. | [S3:L7-L20] |
| Engineering team / architect | Uses requirements, dependencies, touchpoints, and impact outputs for design and delivery. | [S3:L7-L20; S3:L202-L219] |
| QA team | Uses acceptance criteria and test scenario inputs. | [S3:L7-L20; S3:L19-L20; S3:L336-L336] |
| Delivery manager / PM | Receives sprint health escalations and uses delivery readiness artifacts. | [S2:L46-L46; S3:L7-L20] |
| Security/privacy reviewer | Reviews data classification, privacy, and security-sensitive processing before broader rollout [inferred]. | [S5] |
| Tool owners | Approve and govern MCP tool access for Jira, Git, Confluence, Calendar, Teams, and Phase 2 tools [inferred]. | [S2:L34-L35; S3:L322-L336] |

### Stakeholder fields requiring completion

S1 contains blank headings for use-case owner/editor/SME, sponsor, NDA, proposed solution type, third-party involvement, deployment/service prioritization, and primary business entity [S1:L23-L36]. These are unresolved and must be completed by humans before approval.

---

## Aara operating model

### Primary operating role

The primary drafting role for this work is `aara-business-analyst`. It operates as a trace-first, human-gated drafting assistant that extracts candidate requirements, separates facts from assumptions, identifies ambiguity, generates review-ready artifacts, and routes unresolved decisions to accountable humans (task framing [S4]).

### Applied review lenses

The primary `aara-business-analyst` agent was invoked for trace-first drafting. External persona-file review and optional supporting Aara agent invocations were not performed in this run; the following were applied as review lenses based on the task prompt descriptions:

| Lens | How it was applied | Evidence |
| --- | --- | --- |
| Layered base persona | Evidence-first, brownfield-first, decision-quality structure. | Task framing [S4] |
| Agent blueprint persona | Agent scope, tool boundaries, memory, controls, and governance. | Task framing [S4] |
| Delivery planning persona | Clear MVP/Phase 2 separation and dependency surfacing. | Task framing [S4] |
| Production review persona | Human controls, quality gates, and security/privacy escalation. | Task framing [S4]; S5 |
| AI application architecture skill lens | LangGraph/router/MCP topology captured only where source-supported. | [S1:L9-L21; S2:L28-L37] |
| AI evaluation harness skill lens | Evaluation gates, golden-input style review, and human quality gates included as draft controls [inferred]. | Task framing [S4] |
| Security and observability review lenses | Applied only to source-supported tool access, classification, monitoring, escalation, and traceability concerns [inferred]. | [S2:L37-L47; S5] |

### Human-only gates

The Aara operating model requires humans to approve requirements, resolve conflicts, set priority, accept sprint scope, approve change requests, and make compliance commitments. The agent may recommend, draft, route, and revise; it must not approve or publish authoritative decisions on its own.

---

## Traceability discipline

### Required discipline

Every requirement in this document must remain traceable to source evidence or be explicitly marked `[inferred]` or `[RAJA]`. Requirement IDs are stable and phase-aware:

- MVP functional requirements: `BA-MVP-FR-###`
- Phase 2 functional requirements: `BA-P2-FR-###`
- Non-functional requirements: `BA-NFR-###`
- Integration requirements: `BA-INT-###`
- Human control requirements: `BA-HIL-###`
- Data/security/privacy/compliance requirements: `BA-DSPC-###`
- User stories: `BA-US-MVP-###` / `BA-US-P2-###`
- Acceptance criteria: `BA-AC-PROD-###` / `BA-AC-MVP-###`
- Quality gates: `BA-QG-###`
- Goals: `BA-G-###`
- Facts: `BA-FACT-###`
- Conflict findings: `BA-CF-###`
- Risks: `BA-RISK-###`
- Assumptions: `BA-ASM-###`
- Dependencies: `BA-DEP-###`
- Open questions: `BA-OQ-###`

### Facts, assumptions, inferred points, and open questions

| Category | Handling rule |
| --- | --- |
| Facts | Stated directly in reviewed sources and cited. |
| Assumptions | Explicitly listed in the Assumptions section and not treated as approved. |
| Inferred points | Marked `[inferred]`, cited to the evidence that motivated the inference, and routed for review if material. |
| Open questions | Listed with IDs, owner needed, and decision impact. |

### Requirement lifecycle

1. Draft candidate requirement with source citation.
2. Mark unsupported details as `[RAJA]` or `[inferred]`.
3. Route to the proper human reviewer.
4. Update requirement version and rationale after feedback.
5. Maintain trace from source evidence to requirement, user story, acceptance criteria, and downstream delivery artifact.

---

## Scope overview: MVP, Phase 2, out of scope

### MVP scope

The MVP is limited to the Agile/Scrum BA Agent capabilities required by the task framing and supported by S1/S2:

- Teams/Copilot 365 natural language interaction.
- LangGraph routing for standup, sprint planning, retrospective, and sprint health intents.
- Standup summarization using Jira and Git data.
- Sprint planning recommendations using backlog priority, velocity history, and calendar availability.
- Scrum Master review/adjust/approval gate before publishing sprint planning output.
- Retrospective report generation using Jira metrics and Confluence output.
- Continuous sprint health monitoring on schedule or via Jira webhook events.
- MCP tools for Jira, Git, Confluence, Calendar, and Teams.
- Human approval gates for decisions and system-of-record changes.

### Phase 2 scope

Phase 2 adds broader Enterprise BA capabilities from S3:

- Requirement discovery from rough business input.
- Business problem/objective/stakeholder/current-state/future-state extraction.
- Business and functional requirement drafting.
- User story generation.
- Acceptance criteria generation.
- Process mapping.
- Gap analysis.
- Stakeholder question generation.
- Impact analysis.
- Traceability matrices.
- BRD/FRD/PRD drafts.
- Test scenario inputs.
- Candidate enterprise tool integrations, subject to security and owner approval.

### Out of scope

| Item | Rationale |
| --- | --- |
| Final approval of requirements, scope, priority, sprint commitment, or change requests | Human-only decision. |
| Autonomous publishing or mutation of Jira/Confluence/Calendar/Git records without approval | Human-gated operating model. |
| Compliance, legal, privacy, or regulatory sign-off | Requires accountable human owners. |
| Replacing BAs, Scrum Masters, Product Owners, architects, QA, or delivery leaders | S3 states the goal is not to replace the BA [S3:L21-L24]. |
| Non-Teams chat channels | MVP source specifies Teams/Copilot 365. |
| Architecture details not present in the reviewed text sources | Avoids fabricating implementation design. |
| Phase 2 enterprise capabilities inside MVP | S4 requires explicit phase separation. |

---

## Current-state context

### MVP current state

- Scrum ceremonies and sprint governance are heavily manual [S1:L5-L8].
- BAs collect updates across Jira, Git, Confluence, and calendars [S1:L5-L8].
- Status reporting and blocker synthesis are inconsistent across teams [S1:L5-L8].
- Delayed risk identification affects delivery timelines and rework [S1:L5-L8].
- Retrospective learnings are inconsistently captured and not reliably converted into action [S1:L5-L8].

### Phase 2 current state

S3 frames the broader BA challenge as converting rough, ambiguous business input into structured artifacts. Inputs may include meeting notes, business emails, customer requests, product ideas, process pain points, support tickets, and regulatory changes [S3:L27-L49]. The desired agent should clarify vague needs, extract requirements, produce stories and acceptance criteria, map processes, analyze impacts, and generate delivery-ready artifacts [S3:L202-L219].

### Security and classification context

Optional classification summaries indicate that supporting material may include internal/proprietary and restricted information, including process/operational information, security information, and business-critical source code [S5]. This creates a review requirement before broad data ingestion, prompt processing, or storage design is approved [inferred].

---

## Target-state capabilities

| Capability area | MVP target state | Phase 2 target state |
| --- | --- | --- |
| User surface | Teams/Copilot 365 natural language interface with Adaptive Card responses. | Teams plus approved enterprise BA work surfaces [inferred], subject to tool-owner approval. |
| Orchestration | LangGraph router dispatches to standup, sprint planning, retrospective, or health-monitor nodes. | Router may dispatch to Enterprise BA capabilities such as discovery, story generation, process mapping, and traceability [inferred]. |
| Ceremony support | Daily summaries, sprint planning recommendations, retro reports, health alerts. | Not primary Phase 2 focus unless connected to delivery artifact lifecycle. |
| Requirements support | MVP captures sprint-related insights and risks; formal requirement drafting is not MVP core except as needed for this document. | Full requirement discovery, requirements drafting, stories, acceptance criteria, BRD/FRD/PRD, traceability. |
| Analysis support | Sprint health and retrospective metrics. | Gap analysis, impact analysis, data/API/reporting/compliance impact analysis. |
| Human controls | Sprint planning approval gate and escalations to Scrum Master/PM. | Human review for requirements approval, unresolved decisions, prioritization, compliance, and release readiness. |
| Traceability | Sprint summaries, recommendations, and alerts should link to Jira/Git/Calendar/Confluence evidence [inferred]. | End-to-end trace from objective through release item and test case. |

---

## Functional Requirements with stable IDs

### Design constraints carried from source

The sources state solution decisions, not only needs. These are carried as design constraints for the architect to confirm — intentionally not written as functional requirements:

| Constraint | Source decision | Evidence |
| --- | --- | --- |
| Orchestration | LangGraph stateful routing across the four MVP capabilities. | [S1:L11-L11; S2:L30-L33] |
| Tool access mechanism | Jira, Git, Confluence, Teams, and Calendar reached through approved MCP tools. | [S2:L34-L35] |
| User surface | Copilot 365 in Teams with Adaptive Card responses. | [S1:L11-L19; S2:L8-L9; S2:L34-L35] |

### MVP Functional Requirements

| ID | Requirement | Evidence | Notes |
| --- | --- | --- | --- |
| BA-MVP-FR-001 | The MVP shall provide a user-facing natural language interaction surface through Copilot 365 in Teams. | [S1:L11-L19; S2:L8-L9; S2:L34-L35] | Teams is the MVP collaboration surface. |
| BA-MVP-FR-002 | The MVP shall accept both user-initiated and scheduled triggers for supported Scrum workflows. | [S2:L8-L8; S2:L41-L41] | Event-driven Jira webhook behavior is covered under sprint health. |
| BA-MVP-FR-003 | The MVP shall classify user intent and dispatch each request to the standup, sprint planning, retrospective, or sprint health capability. | [S1:L11-L19; S2:L9-L13; S2:L30-L33; S2:L42-L42] | Stable intent taxonomy for MVP. LangGraph is the source-stated orchestration choice — see Design constraints carried from source. |
| BA-MVP-FR-004 | The standup capability shall pull relevant data from Jira and Git, including commits, pull requests, and story status where available. | [S1:L15-L15; S2:L15-L15] | Exact Git provider is open. |
| BA-MVP-FR-005 | The standup capability shall generate concise daily summaries, detect blockers, surface risks, and deliver results as Teams Adaptive Cards. | [S1:L15-L15; S2:L20-L25; S2:L34-L35; S2:L45-L45] | Blocker/risk definitions require review. |
| BA-MVP-FR-006 | The sprint planning capability shall analyze backlog priority, team velocity history, and calendar availability to recommend sprint scope. | [S1:L16-L16; S2:L16-L16] | Recommendation only; not an approval decision. |
| BA-MVP-FR-007 | The sprint planning capability shall provide a human approval gate that lets the Scrum Master review and adjust the AI-recommended sprint scope before publishing. | [S1:L16-L16; S2:L36-L36] | Required Human-in-the-Loop control. |
| BA-MVP-FR-008 | The retrospective capability shall aggregate sprint metrics from Jira, including cycle time, carry-over, and defect rate, and generate structured retrospective reports with actionable improvement recommendations. | [S1:L17-L17; S2:L17-L17] | Metric definitions must be confirmed. |
| BA-MVP-FR-009 | The retrospective capability shall post or prepare retrospective outputs for Confluence to support organizational learning. | [S1:L17-L17; S2:L25-L25; S2:L45-L45] | Whether posting is automatic or approval-gated is open. |
| BA-MVP-FR-010 | The sprint health capability shall run on a schedule or through Jira webhook events to monitor at-risk sprint items. | [S1:L18-L18; S2:L37-L37; S2:L41-L41] | Schedule frequency and webhook events are open. |
| BA-MVP-FR-011 | The sprint health capability shall escalate high-severity blockers and at-risk items to the Scrum Master/PM with suggested corrective actions. | [S1:L18-L18; S2:L46-L46] | Severity model is open. |
| BA-MVP-FR-012 | The MVP shall expose Jira, Git, Confluence, Teams, and Calendar access through approved MCP tools callable by agent workflows. | [S2:L34-L35; S2:L43-L43] | Tool permissions require owner approval. |
| BA-MVP-FR-013 | The MVP shall output Teams Adaptive Cards and Confluence artifacts only for source-supported use cases: standup summaries, planning recommendations, retro reports, and health escalations. | [S1:L15-L19; S2:L25-L25; S2:L45-L45] | Prevents MVP scope drift. |
| BA-MVP-FR-014 | The MVP shall surface source-linked sprint risk categories including stalled stories, scope creep, and resource conflicts when tool data supports detection. | [S1:L7-L8] | Detection logic is `[inferred]`; source names the risk examples but not rules. |
| BA-MVP-FR-015 | The MVP shall preserve the distinction between AI recommendations and human decisions in all sprint-planning and corrective-action outputs. | [S1:L16-L18; S2:L36-L37] | Governance requirement derived from human-gated model. |

### Phase 2 Functional Requirements

| ID | Requirement | Evidence | Notes |
| --- | --- | --- | --- |
| BA-P2-FR-001 | Phase 2 shall support requirement discovery from rough inputs such as meeting notes, business emails, customer requests, product ideas, process pain points, support tickets, and regulatory changes. | [S3:L27-L40] | Regulatory content requires human compliance review. |
| BA-P2-FR-002 | Phase 2 shall extract a clear business problem, business objective, stakeholders, current-state issues, desired future state, open questions, assumptions, and risks. | [S3:L40-L49] | Facts/assumptions/questions must be separated. |
| BA-P2-FR-003 | Phase 2 shall convert business needs into business requirements and functional requirements. | [S3:L5-L20; S3:L202-L219] | Approval remains human-owned. |
| BA-P2-FR-004 | Phase 2 shall generate agile-ready user stories using the “As a / I want / so that” format. | [S3:L50-L59] | Stories remain draft until accepted. |
| BA-P2-FR-005 | Phase 2 shall generate practical, business-readable acceptance criteria using Given/When/Then style where appropriate. | [S3:L61-L76] | Edge case coverage requires SME review. |
| BA-P2-FR-006 | Phase 2 shall identify edge cases, dependencies, non-functional requirements, data needs, and API/system touchpoints for generated stories. | [S3:L61-L66] | Does not imply API design approval. |
| BA-P2-FR-007 | Phase 2 shall create structured current-state and future-state process maps with actor/system swimlanes, decision points, manual steps, automation opportunities, exceptions, and controls. | [S3:L77-L90] | Diagram tooling choice remains open. |
| BA-P2-FR-008 | Phase 2 shall perform gap analysis comparing current state and desired state, including gap and recommendation outputs. | [S3:L91-L100] | Recommendations require human review. |
| BA-P2-FR-009 | Phase 2 shall generate targeted stakeholder clarification questions for unresolved business rules, approvals, data ownership, reporting, and audit needs. | [S3:L101-L114; S3:L221-L231] | Questions should be routed before assumptions become requirements. |
| BA-P2-FR-010 | Phase 2 shall perform impact analysis across business process, users and roles, systems, data, APIs, reports, compliance, training, support operations, and downstream teams. | [S3:L115-L130] | Compliance impacts are analysis inputs, not approvals. |
| BA-P2-FR-011 | Phase 2 shall maintain traceability from business objective to business requirement, functional requirement, user story, acceptance criteria, test case, and release item. | [S3:L131-L148] | Core enterprise governance capability. |
| BA-P2-FR-012 | Phase 2 shall draft BRD, FRD, PRD, user-story, and traceability outputs in structured templates. | [S3:L17-L20; S3:L187-L195; S3:L242-L266] | Template selection requires owner confirmation. |
| BA-P2-FR-013 | Phase 2 shall generate test scenario inputs for QA/test-management use. | [S3:L19-L20; S3:L181-L185; S3:L336-L336] | Test-case approval remains with QA. |
| BA-P2-FR-014 | Phase 2 shall maintain stable project context memory such as project name, business domain, stakeholders, target users, systems, delivery methodology, known business rules, constraints, Definition of Ready, Definition of Done, Jira project key, and Confluence space. | [S3:L338-L353] | Known rules only; do not infer missing business rules. |
| BA-P2-FR-015 | Phase 2 shall support approved enterprise MCP integrations as candidates, including Jira, Confluence, GitHub, Azure DevOps, SharePoint, Teams, Miro/Draw.io, SQL/Data, ServiceNow, and test-management tools. | [S3:L322-L336] | Candidate list only; actual enablement requires security/tool-owner approval. |
| BA-P2-FR-016 | Phase 2 shall always highlight risks, dependencies, unresolved decisions, and open clarification questions in generated artifacts. | [S3:L210-L231] | Prevents false completion. |

---

## Non-Functional Requirements

| ID | Requirement | Evidence | Notes |
| --- | --- | --- | --- |
| BA-NFR-001 | Outputs shall separate facts, assumptions, inferred points, and open questions. | [S3:L221-L231] | Required traceability discipline. |
| BA-NFR-002 | Outputs shall use clear, business-readable language, structured headings, concise paragraphs, tables where useful, and practical examples only when helpful. | [S3:L221-L240] | Applies to MVP and Phase 2 outputs. |
| BA-NFR-003 | The agent shall not assume missing business rules; unclear rules shall become targeted open questions. | [S3:L221-L231] | Human decision required. |
| BA-NFR-004 | Generated recommendations shall remain distinguishable from approved decisions. | [S1:L16-L18; S2:L36-L47] | Prevents autonomous commitment. |
| BA-NFR-005 | Tool access and output publication shall be governed by human approval and tool-owner permissions [inferred]. | [S2:L34-L35; S2:L43-L45] | Exact permission model is open. |
| BA-NFR-006 | The system shall preserve source traceability from input evidence to generated output wherever source data is used [inferred]. | [S3:L131-L148] | MVP sprint outputs should link to Jira/Git/Calendar evidence where feasible. |
| BA-NFR-007 | The MVP shall be modular enough to keep four Scrum capabilities distinct while enabling Phase 2 Enterprise BA capabilities later [inferred]. | [S2:L30-L37; S3:L25-L148] | No architecture beyond source-supported LangGraph/MCP is specified. |
| BA-NFR-008 | The product shall be designed for use across multiple squads or portfolio contexts [inferred]. | [S1:L5-L8] | No scale target is provided. |
| BA-NFR-009 | Security-sensitive and classified source material shall be handled according to confirmed classification and approved review paths [inferred]. | [S5] | Requires security/privacy review before implementation. |
| BA-NFR-010 | Evaluation gates shall verify requirement traceability, output quality, routing correctness, human-gate enforcement, and absence of unsupported claims [inferred]. | Task framing [S4] | Quantitative thresholds are open. |
| BA-NFR-011 | Monitoring and escalation workflows shall be observable enough for humans to review trigger source, analysis rationale, and escalation output [inferred]. | [S1:L18-L18; S2:L37-L47] | Logging/audit design is open. |

---

## Integrations

### MVP integrations

| ID | Integration | Required MVP use | Evidence | Open decisions |
| --- | --- | --- | --- | --- |
| BA-INT-001 | Copilot 365 in Teams | User-facing natural language interface and Adaptive Card output. | [S1:L11-L19; S2:L8-L9; S2:L34-L35] | Tenant/app approval path, user groups, card templates. |
| BA-INT-002 | Jira MCP | Story status, backlog priority, sprint metrics, health monitoring, webhook events. | [S1:L15-L18; S2:L15-L18; S2:L37-L43] | Jira projects, fields, status mapping, webhook scope. |
| BA-INT-003 | Git MCP | Commits and pull requests for standup summaries. | [S1:L15-L15; S2:L15-L15] | Git provider and repository scope. |
| BA-INT-004 | Calendar MCP | Team availability for sprint planning recommendations. | [S1:L16-L16; S2:L16-L16] | Calendar source, privacy limits, availability granularity. |
| BA-INT-005 | Confluence MCP | Retrospective outputs and organizational learning artifacts. | [S1:L17-L17; S2:L25-L25; S2:L45-L45] | Space/page ownership and approval before posting. |
| BA-INT-006 | LangGraph | Stateful orchestration and conditional routing across MVP capabilities. | [S1:L11-L11; S2:L30-L33] | Runtime environment and operational ownership. |

### Authoritative source-of-record mapping

System of record per data domain [inferred from source usage; tool owners to confirm]. The agent treats the system of record as evidence ground truth; derived or cached data must cite its origin and timestamp.

| Domain | System of record | Agent access | Allowed agent action | Evidence requirement |
| --- | --- | --- | --- | --- |
| Stories, backlog, sprint status, sprint metrics | Jira | Read; write approval-gated (BA-HIL-006) | Read for summaries/planning/retro/health; no unapproved status or field changes. | Cite issue keys and query timestamp. |
| Commits, pull requests | Git provider (BA-OQ-008) | Read-only | Read for standup evidence. | Cite repo, commit/PR identifiers. |
| Team availability | Calendar (BA-OQ-009) | Read-only, privacy-limited | Aggregate availability for planning; no event creation or detail exposure. | Cite availability window queried, not event contents. |
| Retro reports, organizational learning artifacts | Confluence | Write approval-gated (BA-OQ-007) | Draft/post retro artifacts per approved permissions. | Link report to source Jira metrics. |
| User interaction, notifications, escalations | Teams / Copilot 365 | Read/respond in approved channels | Adaptive Card output; escalation delivery. | Escalations link to source-system evidence. |
| Requirements and delivery artifacts (Phase 2) | To be designated (BA-OQ-013) | Draft-only | Draft artifacts routed for human review. | Trace per BA-P2-FR-011. |

### Phase 2 candidate integrations

S3 lists enterprise MCP tools that the BA Agent can connect to: Jira, Confluence, GitHub, Azure DevOps, SharePoint, Teams, Miro/Draw.io, SQL/Data, ServiceNow, and TestRail/QA [S3:L322-L336]. These are candidate Phase 2 integrations only; each requires separate approval, security review, and delivery planning.

---

## Human-in-the-Loop controls

| ID | Control | Requirement | Evidence | Human owner needed |
| --- | --- | --- | --- | --- |
| BA-HIL-001 | Sprint planning approval | AI-recommended sprint scope must be reviewed, adjusted if needed, and approved by the Scrum Master before publishing. | [S1:L16-L16; S2:L36-L36] | Scrum Master |
| BA-HIL-002 | Sprint health escalation review | Suggested corrective actions for blockers and at-risk items must be presented as recommendations, not automatic commitments. | [S1:L18-L18; S2:L46-L46] | Scrum Master / PM |
| BA-HIL-003 | Requirement approval | Generated business/functional requirements, user stories, acceptance criteria, and artifacts remain drafts until approved by accountable humans. | [S3:L21-L24; S3:L221-L231] | Product Owner / BA SME |
| BA-HIL-004 | Conflict resolution | Conflicting stakeholder statements and source conflicts must be surfaced, not silently resolved. | [S3:L221-L231] | Product Owner / SME |
| BA-HIL-005 | Compliance/security routing | Security, privacy, restricted-data, or regulatory-impact findings require review by appropriate owners before implementation or publication. | [S5] | Security/privacy/compliance owner |
| BA-HIL-006 | System-of-record write gate | Updates to Jira, Confluence, Calendar, Git, or test-management systems shall require explicit human approval unless separately authorized by policy [inferred]. | [S2:L43-L45] | Tool owner / delivery lead |

### Capability autonomy classification

Autonomy level per capability. Levels: **Advisory** (agent output is information only), **Drafting** (agent produces an artifact that remains a draft until human review), **Approval-gated** (agent may act only after explicit human approval). No MVP or Phase 2 capability is autonomous.

| ID | Capability | Autonomy level | Evidence | Notes |
| --- | --- | --- | --- | --- |
| BA-AUT-001 | Standup summary | Advisory / drafting | [S1:L15-L15; S2:L20-L25] | Summary is informational; no system-of-record write. |
| BA-AUT-002 | Sprint planning recommendation | Approval-gated | [S1:L16-L16; S2:L36-L36] | Publishing requires Scrum Master review/adjust/approval (BA-HIL-001). |
| BA-AUT-003 | Retrospective report generation | Drafting; approval before posting | [S1:L17-L17; S2:L25-L25] | Whether Confluence posting is auto or gated is open (BA-OQ-007); default is gated [inferred]. |
| BA-AUT-004 | Sprint health alert | Advisory escalation | [S1:L18-L18; S2:L46-L46] | Corrective actions are suggestions, not commitments (BA-HIL-002). |
| BA-AUT-005 | Jira / Confluence / Calendar / Git writes | Approval-gated only | [S2:L43-L45] | Governed by BA-HIL-006 [inferred]. |
| BA-AUT-006 | Phase 2 requirement, story, and artifact generation | Drafting only | [S3:L21-L24; S3:L221-L231] | Approval remains with accountable humans (BA-HIL-003). |

---

## Data, security, privacy, and compliance requirements

| ID | Requirement | Evidence | Notes |
| --- | --- | --- | --- |
| BA-DSPC-001 | The agent shall preserve source metadata where available, including source system/document, date, owner, and confidentiality/classification. | [S5] | Supports traceability and review. |
| BA-DSPC-002 | Data ingestion for internal/proprietary, restricted, security-sensitive, or source-code material shall be reviewed against approved classification guidance before use [inferred]. | [S5] | Final classification owner required. |
| BA-DSPC-003 | The agent shall not infer or approve regulatory, legal, privacy, or audit obligations; it shall flag them for owner review. | [S3:L101-L114; S3:L115-L130] | Compliance impact analysis is not sign-off. |
| BA-DSPC-004 | Secrets, credentials, and highly sensitive security material shall not be intentionally included in prompts, generated documents, or collaboration posts [inferred]. | [S5] | Exact detection/redaction control is open. |
| BA-DSPC-005 | Access to MCP tools shall follow least-privilege permissions and approved tool-owner scopes [inferred]. | [S2:L34-L35; S2:L43-L45; S5] | Authentication/authorization design is open. |
| BA-DSPC-006 | Teams and Confluence outputs shall honor the classification of source data and audience authorization [inferred]. | [S1:L15-L19; S2:L25-L45; S5] | Requires security/privacy review. |
| BA-DSPC-007 | Retention, audit logging, and data residency expectations are `[RAJA]` until confirmed by security, privacy, and platform owners. | [S5] | Open question, not an approved requirement. |

---

## Reporting and analytics

### MVP reporting outputs

| Output | Description | Evidence |
| --- | --- | --- |
| Daily standup summary | Concise summary from Jira/Git including blockers and risks, delivered through Teams Adaptive Cards. | [S1:L15-L15; S2:L25-L25; S2:L45-L45] |
| Sprint planning recommendation | Recommended sprint scope based on backlog priority, velocity history, and calendar availability, routed through approval. | [S1:L16-L16; S2:L36-L36] |
| Retrospective report | Structured report using Jira metrics such as cycle time, carry-over, and defect rate, with improvement recommendations. | [S1:L17-L17] |
| Sprint health alert | Escalation of high-severity blockers and at-risk items with suggested corrective actions. | [S1:L18-L18; S2:L46-L46] |
| Confluence learning artifact | Posted or prepared retrospective output for organizational learning. | [S1:L17-L17; S2:L25-L45] |

### Phase 2 reporting and analysis outputs

| Output | Description | Evidence |
| --- | --- | --- |
| Requirements package | Business requirements, functional requirements, user stories, acceptance criteria. | [S3:L3-L20; S3:L202-L219] |
| Process map | Current-state and future-state flows with swimlanes, decisions, exceptions, and controls. | [S3:L77-L90] |
| Gap analysis table | Current state, future state, gap, and recommendation. | [S3:L91-L100] |
| Impact analysis | Business, technology, operations, users, data, APIs, reports, compliance, training, support, downstream teams. | [S3:L115-L130] |
| Traceability matrix | Objective-to-release trace across requirements, stories, acceptance criteria, test cases, and release items. | [S3:L131-L148] |
| BRD/FRD/PRD drafts | Structured document outputs for human review. | [S3:L17-L20; S3:L187-L195; S3:L242-L266] |
| Test scenario inputs | QA-ready scenario inputs derived from requirements and acceptance criteria. | [S3:L19-L20; S3:L181-L185; S3:L336-L336] |

---

## User stories with Acceptance Criteria

### MVP user stories

| Story ID | User story | Linked requirements | Acceptance Criteria |
| --- | --- | --- | --- |
| BA-US-MVP-001 | As a team member, I want to ask for a daily standup summary in Teams so that I can understand status, blockers, and risks without manual status collection. | BA-MVP-FR-001, BA-MVP-FR-003, BA-MVP-FR-004, BA-MVP-FR-005, BA-MVP-FR-012 | Given the user requests a standup summary in Teams, when the intent is routed to the standup capability, then the agent retrieves available Jira/Git evidence and returns a Teams Adaptive Card with summary, blockers, and risks. |
| BA-US-MVP-002 | As a Scrum Master, I want AI-assisted sprint planning recommendations so that I can review candidate scope before committing the sprint. | BA-MVP-FR-006, BA-MVP-FR-007, BA-MVP-FR-015, BA-HIL-001 | Given backlog, velocity history, and calendar availability are available, when sprint planning is requested, then the agent presents recommended scope for Scrum Master review and does not publish until approved. |
| BA-US-MVP-003 | As a BA or Scrum Master, I want a structured retrospective report so that sprint learnings are captured consistently and shared in Confluence. | BA-MVP-FR-008, BA-MVP-FR-009, BA-INT-005 | Given a sprint has completed and Jira metrics are available, when the retrospective capability runs, then the agent generates a structured retro report with cycle time, carry-over, defect-rate evidence, and improvement recommendations for Confluence review/posting. |
| BA-US-MVP-004 | As a Scrum Master or PM, I want sprint health alerts so that high-severity blockers and at-risk work are escalated before they impact delivery. | BA-MVP-FR-010, BA-MVP-FR-011, BA-MVP-FR-014, BA-HIL-002 | Given the monitor runs on schedule or Jira event, when high-severity blockers or at-risk items are detected, then the agent sends an escalation with source-linked rationale and suggested corrective actions. |
| BA-US-MVP-005 | As a team member, I want one Teams entry point for standup, planning, retro, and health-check questions so that I do not switch tools to find sprint insights. | BA-MVP-FR-001, BA-MVP-FR-002, BA-MVP-FR-003, BA-MVP-FR-013 | Given a supported request is entered in Teams, when the router classifies the intent, then the request is dispatched to the correct MVP capability and unsupported requests are flagged for clarification. |

### Phase 2 user stories

| Story ID | User story | Linked requirements | Acceptance Criteria |
| --- | --- | --- | --- |
| BA-US-P2-001 | As a business user, I want rough input converted into a structured requirement-discovery summary so that workshops start with shared clarity. | BA-P2-FR-001, BA-P2-FR-002, BA-P2-FR-016 | Given rough input is provided, when discovery runs, then the output separates problem, objective, stakeholders, current state, future state, assumptions, risks, and open questions. |
| BA-US-P2-002 | As a Product Owner or BA, I want draft user stories and acceptance criteria so that backlog refinement starts from a consistent artifact. | BA-P2-FR-003, BA-P2-FR-004, BA-P2-FR-005, BA-P2-FR-006 | Given approved requirement input, when story generation runs, then each story follows the source-supported format and includes draft acceptance criteria, dependencies, and unresolved questions. |
| BA-US-P2-003 | As a BA, I want process maps and gap analysis so that stakeholders can compare current and desired states before deciding scope. | BA-P2-FR-007, BA-P2-FR-008 | Given current-state and desired-state input, when process/gap analysis runs, then the output identifies flow steps, actors/systems, decision points, gaps, and recommendations for review. |
| BA-US-P2-004 | As an architect or delivery lead, I want impact analysis and traceability so that design and planning can see affected systems, data, APIs, reports, operations, and release items. | BA-P2-FR-010, BA-P2-FR-011 | Given a draft change or requirement set, when impact analysis runs, then the agent produces source-linked impacts and traceability relationships without approving scope. |
| BA-US-P2-005 | As a QA lead, I want test scenario inputs derived from requirements and acceptance criteria so that test planning can begin with traceable coverage. | BA-P2-FR-005, BA-P2-FR-011, BA-P2-FR-013 | Given approved or review-ready requirements, when test scenario generation runs, then each scenario input traces back to a requirement and acceptance criterion. |

---

## Product-level and MVP Acceptance Criteria

### Product-level Acceptance Criteria

| ID | Acceptance criterion | Linked requirements |
| --- | --- | --- |
| BA-AC-PROD-001 | All generated requirements and artifacts identify whether statements are facts, assumptions, inferred points, or open questions. | BA-NFR-001, BA-NFR-003 |
| BA-AC-PROD-002 | Every functional requirement links to source evidence or is explicitly marked `[inferred]` or `[RAJA]`. | BA-NFR-006, BA-P2-FR-011 |
| BA-AC-PROD-003 | Human-only decisions are never presented as agent-approved outcomes. | BA-NFR-004, BA-HIL-003, BA-HIL-004 |
| BA-AC-PROD-004 | MVP and Phase 2 capabilities remain visibly separated in requirements, roadmap discussions, and acceptance reviews. | BA-NFR-007 |
| BA-AC-PROD-005 | Security/privacy/compliance impacts are routed for owner review and are not treated as approved obligations by the agent. | BA-DSPC-002, BA-DSPC-003, BA-HIL-005 |

### MVP Acceptance Criteria

| ID | Acceptance criterion | Linked requirements |
| --- | --- | --- |
| BA-AC-MVP-001 | A Teams/Copilot 365 request for each MVP intent routes to the correct capability: standup, sprint planning, retrospective, or sprint health. | BA-MVP-FR-001, BA-MVP-FR-003 |
| BA-AC-MVP-002 | Standup output uses Jira and Git evidence and returns a concise Teams Adaptive Card with status, blockers, and risks. | BA-MVP-FR-004, BA-MVP-FR-005 |
| BA-AC-MVP-003 | Sprint planning output shows recommended scope and requires Scrum Master review/adjust/approval before publishing. | BA-MVP-FR-006, BA-MVP-FR-007, BA-HIL-001 |
| BA-AC-MVP-004 | Retrospective output includes Jira sprint metrics named in source evidence and prepares or posts the report to Confluence according to approved permissions. | BA-MVP-FR-008, BA-MVP-FR-009, BA-INT-005 |
| BA-AC-MVP-005 | Sprint health monitoring can run on schedule or Jira event and escalate at-risk items with suggested corrective actions. | BA-MVP-FR-010, BA-MVP-FR-011 |
| BA-AC-MVP-006 | No MVP workflow performs an unapproved system-of-record write or delivery commitment. | BA-MVP-FR-015, BA-HIL-006 |

---

## Evaluation and quality gates

Gate metric definitions, golden test sets, and pass thresholds are specified in the companion document `ba_agent_evaluation_harness.md`. Thresholds are `[RAJA]` until set by the accountable owner. Proposed MCP tool contracts (inputs, outputs, permissions, failure handling, audit records) are specified in `ba_agent_mcp_tool_contracts.md` for architect confirmation. Proposed runtime topology and security design are in `ba_agent_runtime_architecture.md`; proposed ownership, support tiers, incident response, and release management are in `ba_agent_operations_model.md`. All companion documents are design proposals subordinate to this requirements baseline.

| Gate ID | Gate | Draft pass condition | Reviewer |
| --- | --- | --- | --- |
| BA-QG-001 | Source traceability gate | Each functional requirement, user story, and acceptance criterion maps to at least one source-backed requirement or is marked `[inferred]`/`[RAJA]`. | BA SME / Product Owner |
| BA-QG-002 | MVP intent-routing gate | Test prompts for standup, sprint planning, retrospective, and sprint health route to the correct capability node [inferred]. | Architect / BA SME |
| BA-QG-003 | Human-gate enforcement | Sprint planning recommendations cannot be published without Scrum Master approval. | Scrum Master / QA |
| BA-QG-004 | Output quality review | Outputs are clear, business-readable, structured, and separate facts from assumptions/questions. | BA SME / Product Owner |
| BA-QG-005 | Security/classification review | Data sources, Teams outputs, and Confluence artifacts are reviewed against confirmed classification and access rules before pilot use. | Security/privacy owner |
| BA-QG-006 | Integration sandbox validation | Jira, Git, Confluence, Calendar, and Teams MCP tool access is validated in an approved non-production or controlled pilot context [inferred]. | Tool owners / QA |
| BA-QG-007 | Evaluation harness readiness | A reviewed set of representative prompts and expected output characteristics exists for regression review before release [inferred]. | AI evaluation reviewer / BA SME |
| BA-QG-008 | Phase separation review | MVP release notes and backlog do not include Phase 2 enterprise BA capabilities unless explicitly approved as scope changes. | Product Owner / Delivery lead |

---

## Risks

| Risk ID | Risk | Evidence | Mitigation / routing |
| --- | --- | --- | --- |
| BA-RISK-001 | MVP identity may be unclear because sources use both Business Analyst Agent and AI Scrum Master wording. | [S1:L1-L3; S2:L1-L4] | Confirm product name and role positioning. |
| BA-RISK-002 | Phase 2 capabilities could be pulled into MVP, expanding scope beyond Scrum ceremonies. | [S3:L355-L371] | Enforce phase gates and change-control review. |
| BA-RISK-003 | Sprint-health severity and escalation rules are not defined. | [S1:L18-L18; S2:L37-L47] | Define severity taxonomy, recipients, and response expectations. |
| BA-RISK-004 | Data classification may restrict which documents, code, or security information the agent can process. | [S5] | Complete security/privacy review before implementation. |
| BA-RISK-005 | Tool permissions could accidentally allow unapproved updates to systems of record. | [S2:L43-L45] | Implement explicit approval gates and least-privilege scopes [inferred]. |
| BA-RISK-006 | Metrics named for retrospectives may not be consistently available across Jira projects. | [S1:L17-L17] | Confirm Jira field standards and fallback behavior. |
| BA-RISK-007 | Architecture and workflow decks were available but not text-reviewable in this run. | [S6] | Request readable exports or source diagrams before architecture sign-off. |
| BA-RISK-008 | AI-generated recommendations may be mistaken for approved decisions. | [S1:L16-L18; S2:L36-L47] | Label outputs as draft/recommendation and require human approval. |
| BA-RISK-009 | Quantitative business success targets are absent. | [S1:L21-L21] | Product Owner to define measurable targets if needed for business case. |

---

## Assumptions

| Assumption ID | Assumption | Evidence / rationale | Review owner |
| --- | --- | --- | --- |
| BA-ASM-001 | The task framing in S4 is authoritative for separating MVP from Phase 2 in this document. | Task framing [S4] explicitly defines scope separation. | Product Owner |
| BA-ASM-002 | The MVP is a drafting/recommendation assistant and will not autonomously commit sprint scope. | Human approval gate in S1/S2 [S1:L16-L16; S2:L36-L36]. | Scrum Master / Delivery lead |
| BA-ASM-003 | Teams/Copilot 365 is available as the user-facing surface for the MVP. | S1/S2 specify Teams/Copilot 365 [S1:L11-L19; S2:L8-L9]. | Platform owner |
| BA-ASM-004 | Required Jira, Git, Confluence, and Calendar data can be accessed through approved MCP tools. | S2 lists MCP tools [S2:L34-L35]. | Tool owners |
| BA-ASM-005 | Phase 2 enterprise BA capabilities will be prioritized after MVP validation rather than delivered together with MVP. | Required by task framing [S4]. | Product Owner / Planner |
| BA-ASM-006 | Data classification and approved handling rules will be confirmed before any pilot processes restricted or security-sensitive material. | Classification context [S5]. | Security/privacy owner |

---

## Dependencies

| Dependency ID | Dependency | Phase | Evidence | Decision needed |
| --- | --- | --- | --- | --- |
| BA-DEP-001 | Teams/Copilot 365 access and app/plugin approval. | MVP | [S1:L11-L19; S2:L8-L9; S2:L34-L35] | Tenant/platform approval path. |
| BA-DEP-002 | LangGraph runtime and orchestration ownership. | MVP | [S1:L11-L11; S2:L30-L33] | Runtime environment and support team. |
| BA-DEP-003 | Jira project access, fields, backlog, sprint metrics, and webhook availability. | MVP | [S1:L15-L18; S2:L15-L18; S2:L37-L43] | Field mapping, project scope, webhook scope. |
| BA-DEP-004 | Git access for commits and pull requests. | MVP | [S1:L15-L15; S2:L15-L15] | Repository and provider scope. |
| BA-DEP-005 | Calendar availability access for sprint planning. | MVP | [S1:L16-L16; S2:L16-L16] | Privacy limits and aggregation rules. |
| BA-DEP-006 | Confluence space/page ownership for retrospective reports. | MVP | [S1:L17-L17; S2:L25-L45] | Posting permissions and approval gate. |
| BA-DEP-007 | Human reviewers for Scrum, product, BA, QA, architecture, security/privacy, and tool ownership. | MVP and Phase 2 | [S3:L7-L20; S5] | Named reviewers and RACI. |
| BA-DEP-008 | Representative prompt/test set for evaluation gates. | MVP and Phase 2 | Task framing [S4] | Golden examples and review rubric. |
| BA-DEP-009 | Phase 2 enterprise tool approvals for SharePoint, Miro/Draw.io, SQL/Data, ServiceNow, TestRail/QA, GitHub, and Azure DevOps where selected. | Phase 2 | [S3:L322-L336] | Prioritized integration roadmap. |
| BA-DEP-010 | Stable project context values for Phase 2 memory. | Phase 2 | [S3:L338-L353] | Approved source of truth for business rules and constraints. |

---

## Open Questions

| ID | Open question | Why it matters | Suggested owner |
| --- | --- | --- | --- |
| BA-OQ-001 | What is the approved product name and positioning: Business Analyst Agent, AI Scrum Master, or another name? | Avoids stakeholder confusion and scope drift. | Product Owner / Sponsor |
| BA-OQ-002 | Who are the use-case owner, editor, technical SME, sponsor, primary business entity, and approval reviewers? | S1 leaves these fields blank. | Sponsor / Delivery lead |
| BA-OQ-003 | Which squads, Jira projects, repositories, Confluence spaces, and calendars are in the MVP pilot? | Defines data scope and access permissions. | Scrum Master / Tool owners |
| BA-OQ-004 | What are the quantitative success targets, if any, for manual-effort reduction, risk-detection timeliness, or output quality? | Sources state desired outcomes but not metrics. | Product Owner |
| BA-OQ-005 | What rules define blocker severity, at-risk items, stalled stories, scope creep, and resource conflicts? | Required for sprint health monitoring. | Scrum Master / PM |
| BA-OQ-006 | What does “publish” mean for sprint planning: update Jira, post to Teams, create Confluence notes, or another action? | Determines human-gate enforcement. | Scrum Master / Tool owners |
| BA-OQ-007 | Should Confluence retro reports be auto-posted after review, posted as drafts, or only generated for manual posting? | Balances automation with governance. | BA / Confluence owner |
| BA-OQ-008 | Which Git provider and pull request metadata are in scope? | S1/S2 name Git generically. | Engineering/tool owner |
| BA-OQ-009 | What calendar availability detail may the agent use without exposing sensitive personal schedule content? | Required for privacy-aware planning. | Security/privacy owner / Scrum Master |
| BA-OQ-010 | What classification handling rules apply to internal, restricted, security-sensitive, or source-code data used by the agent? | Required before pilot data ingestion. | Security/privacy owner |
| BA-OQ-011 | What are the approved write permissions for Jira, Confluence, Calendar, Git, and later Phase 2 tools? | Prevents unauthorized system-of-record updates. | Tool owners |
| BA-OQ-012 | Which Phase 2 capabilities should be prioritized first after MVP validation? | S3 contains a broad capability set. | Product Owner / Planner |
| BA-OQ-013 | Which Phase 2 enterprise integrations are approved and necessary? | Candidate list is broad and requires security/tool review. | Product Owner / Tool owners |
| BA-OQ-014 | What retention, audit, and data residency requirements apply to prompts, outputs, Teams messages, and Confluence artifacts? | Required for production readiness. | Security/privacy/platform owners |
| BA-OQ-015 | What evaluation rubric and representative test set will be used to approve output quality? | Required for consistent release gates. | BA SME / AI evaluation reviewer |

---

## Traceability Matrix

### Source-to-requirement trace

| Source evidence | Summary | Requirement IDs |
| --- | --- | --- |
| S1:L5-L8 | Manual BA/Scrum Master ceremonies, multi-tool status gathering, inconsistent reporting, delayed risk detection, retro inconsistency. | BA-MVP-FR-004, BA-MVP-FR-005, BA-MVP-FR-014, BA-NFR-008, BA-RISK-009 |
| S1:L9-L21 | AI-powered BA Agent using LangGraph, Copilot 365 via Teams, standup, planning, retrospective, health monitoring, natural language interaction, reduced overhead and consistent practices. | BA-MVP-FR-001 through BA-MVP-FR-015, BA-INT-001 through BA-INT-006, BA-G-001 through BA-G-005 |
| S1:L23-L36 | Blank owner/sponsor/entity fields. | BA-CF-003, BA-OQ-002 |
| S2:L6-L26 | Workflow diagram: User/Schedule to Copilot 365 Teams, LangGraph router, four agents, MCP tools, LLM summarization/risk detection, Teams/Confluence output. | BA-MVP-FR-001, BA-MVP-FR-002, BA-MVP-FR-003, BA-MVP-FR-004, BA-MVP-FR-006, BA-MVP-FR-008, BA-MVP-FR-010, BA-MVP-FR-012, BA-MVP-FR-013 |
| S2:L28-L37 | Key design decisions: LangGraph, four nodes, MCP tools, Copilot 365, approval gate, continuous monitor. | BA-MVP-FR-003, BA-MVP-FR-007, BA-MVP-FR-010, BA-MVP-FR-012, BA-HIL-001, BA-NFR-007 |
| S2:L39-L47 | Agent flow summary: schedule/user prompt, routing, tool execution, LLM reasoning, Teams/Confluence output, escalation. | BA-MVP-FR-002, BA-MVP-FR-003, BA-MVP-FR-011, BA-MVP-FR-013, BA-HIL-002 |
| S3:L3-L24 | Enterprise BA purpose and artifact outputs; not replacing BA. | BA-P2-FR-003, BA-P2-FR-012, BA-P2-FR-013, BA-HIL-003, BA-G-006 |
| S3:L27-L49 | Requirement discovery inputs and outputs. | BA-P2-FR-001, BA-P2-FR-002 |
| S3:L50-L76 | User story and acceptance criteria generation. | BA-P2-FR-004, BA-P2-FR-005, BA-P2-FR-006, BA-US-P2-002 |
| S3:L77-L100 | Process mapping and gap analysis. | BA-P2-FR-007, BA-P2-FR-008, BA-US-P2-003 |
| S3:L101-L130 | Stakeholder questions and impact analysis areas. | BA-P2-FR-009, BA-P2-FR-010, BA-DSPC-003 |
| S3:L131-L148 | Traceability chain from objective to release item. | BA-P2-FR-011, BA-NFR-006, BA-US-P2-004 |
| S3:L150-L185 | Recommended folder/skills/artifact capabilities. | BA-P2-FR-006, BA-P2-FR-012, BA-P2-FR-013 |
| S3:L202-L219 | Core responsibilities for Enterprise BA Agent. | BA-P2-FR-002, BA-P2-FR-003, BA-P2-FR-010, BA-P2-FR-016 |
| S3:L221-L240 | Operating principles and output style. | BA-NFR-001, BA-NFR-002, BA-NFR-003, BA-HIL-004 |
| S3:L242-L266 | Enterprise BA workflow from input to Jira-ready output. | BA-P2-FR-012, BA-P2-FR-016 |
| S3:L322-L336 | Candidate enterprise MCP tools. | BA-P2-FR-015, BA-DEP-009 |
| S3:L338-L353 | Agent memory stable project context. | BA-P2-FR-014, BA-DEP-010 |
| S3:L355-L371 | Reference “best first version” and later adds. | BA-CF-002, BA-RISK-002, BA-OQ-012 |
| S4 (task framing — not requirement evidence) | Phase separation, Aara operating model, evidence discipline, and quality gates. Governs document structure only; framing-derived items listed. | BA-ASM-001, BA-ASM-005, BA-NFR-010, BA-DEP-008, BA-QG-001, BA-QG-007, BA-QG-008 |
| S5 | Optional classification summaries. | BA-DSPC-001 through BA-DSPC-007, BA-HIL-005, BA-NFR-009, BA-RISK-004 |
| S6 | PPTX decks unavailable for text review. | BA-RISK-007 |

### Requirement-to-story trace

| Requirement IDs | User story IDs |
| --- | --- |
| BA-MVP-FR-001, BA-MVP-FR-002, BA-MVP-FR-003 | BA-US-MVP-001, BA-US-MVP-005 |
| BA-MVP-FR-004, BA-MVP-FR-005 | BA-US-MVP-001 |
| BA-MVP-FR-006, BA-MVP-FR-007, BA-MVP-FR-015 | BA-US-MVP-002 |
| BA-MVP-FR-008, BA-MVP-FR-009 | BA-US-MVP-003 |
| BA-MVP-FR-010, BA-MVP-FR-011, BA-MVP-FR-014 | BA-US-MVP-004 |
| BA-P2-FR-001, BA-P2-FR-002, BA-P2-FR-016 | BA-US-P2-001 |
| BA-P2-FR-003, BA-P2-FR-004, BA-P2-FR-005, BA-P2-FR-006 | BA-US-P2-002 |
| BA-P2-FR-007, BA-P2-FR-008 | BA-US-P2-003 |
| BA-P2-FR-010, BA-P2-FR-011 | BA-US-P2-004 |
| BA-P2-FR-013 | BA-US-P2-005 |

---

## Recommended next steps

1. Confirm product naming and MVP positioning with sponsor/Product Owner.
2. Complete missing ownership fields from S1 and establish a RACI for product, BA, Scrum, architecture, QA, security/privacy, and tool ownership.
3. Validate source classification and approved handling rules before any pilot data is processed.
4. Define MVP pilot boundaries: teams, Jira projects, repositories, Confluence spaces, calendars, and Teams channels.
5. Define sprint-health severity taxonomy, escalation recipients, and response expectations.
6. Decide which output actions are draft-only, approval-gated, or allowed as approved writes.
7. Create a small reviewed evaluation set for standup, planning, retrospective, and health-monitor prompts.
8. Request readable exports of the architecture and workflow decks before architecture sign-off.
9. Route this requirements draft to Product Owner, Scrum Master/BA SME, architect, QA, security/privacy, and tool owners for review.
10. After MVP review, run a separate Phase 2 prioritization workshop for Enterprise BA capabilities and integrations.
