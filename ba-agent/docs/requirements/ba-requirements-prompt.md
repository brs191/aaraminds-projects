# Goal

Create a decision-quality requirements document for the Business Analyst AI Agent and save it under:

`/Users/rb692q/projects/aaraminds-projects/ba-agent/business-analyst-agent-requirements.md`

This is a docs-only generation task. Do not edit source documents.

# Source documents to read first

1. `/Users/rb692q/projects/brs191/CC_UserCases/BA Agent/BusinessAnalystAgent.md`
2. `/Users/rb692q/projects/brs191/CC_UserCases/BA Agent/BusinessAnalystAgent-workflow-diagram.md`
3. `/Users/rb692q/projects/aaraminds-projects/ba-agent/ref-requirements.md`

Also inspect optional supporting decks/PDFs in:

`/Users/rb692q/projects/brs191/CC_UserCases/BA Agent/`

If supporting files are not readable as text, mention them as "available but not reviewed due to format/tooling limitation."

# Aara routing and operating model

If Aara agents are available, use this routing model:

- Primary agent: `aara-business-analyst`
  - Use for trace-first requirements drafting, stakeholder intent extraction, requirement IDs, user stories, acceptance criteria, open questions, and impact notes.
  - Treat the agent as a human-gated drafting assistant, not an autonomous approver.

Optional supporting agents, only for their relevant sections:

- `aara-agent-blueprint-advisor`: agent operating model, tools, controls, memory, governance, and lifecycle boundaries.
- `aara-ai-application-architect`: AI/LLM application topology and orchestration fit, only if architecture-level requirements are needed.
- `aara-ai-evaluation-engineer`: evaluation gates, rubrics, golden datasets, acceptance criteria, and quality controls.
- `aara-project-architect`: system architecture and integration implications, only where supported by source evidence.
- `aara-project-planner`: phased delivery, MVP/Phase 2 sequencing, dependencies, and delivery gates.
- `aara-executive-narrative-advisor`: executive-summary polish only if needed.

If agent invocation is not available in this environment, apply the routing descriptions above as review lenses and state that agent routing was not executable in the final execution report.

# Persona lenses

Apply these Aara persona lenses where relevant:

- Base persona: `01_Layered_Base_System_v1.1.md` — verdict-first, brownfield-first, stack-pinned, evidence-based reasoning.
- Agent blueprint persona: `08_AI_Agent_Blueprint_System_v1.1.md` — agent scope, tools, controls, memory, evaluation, and governance.
- Delivery planning persona: `09_Project_Delivery_Planning_System_v1.0.md` — sequencing, milestones, delivery gates, and dependencies.
- Production review persona: `05_AI_Systems_Review_System_v1.2.md` — production-readiness and governance checks, only where needed.
- Optional design lens: `AaraMinds_AI_Agent_Blueprint_Advisor_v1.1.md`.

# Existing skills to apply

Use only existing skills. Do not fabricate BA-specific skills.

Relevant skills:

- `ai-application-architecture` for AI application topology and orchestration requirements.
- `ai-evaluation-harness` for evaluation gates, acceptance rubrics, golden datasets, and measurable quality checks.
- `microservices-architecture-design`, `microservices-api-design`, `azure-service-mapping`, `azure-microservices-security`, and `azure-microservices-observability` only if architecture, API, Azure, security, or observability sections are explicitly in scope.
- `prompt-engineering` only for keeping this task scoped and output-compliant; do not add a prompt-engineering section to the requirements document.

If a BA-specific skill would be useful, list it only as a proposed future enablement item, not as an existing dependency.

# Scope framing

Separate MVP and Phase 2 clearly.

- MVP scope: Agile/Scrum BA Agent for standups, sprint planning, retrospectives, sprint health, Teams/Copilot 365, Jira/Git/Confluence/Calendar MCP tools, and human approval gates.
- Phase 2 scope: broader Enterprise BA capabilities from `ref-requirements.md`, including requirement discovery, user stories, acceptance criteria, process mapping, gap analysis, impact analysis, traceability, BRD/FRD/PRD, and test scenario outputs.
- Do not merge MVP and Phase 2 into one undifferentiated scope.

# Workspace conventions

Follow AaraMinds conventions:

- Design-first and brownfield-first.
- Teams, not Slack.
- Azure is the primary cloud.
- Use JFrog Artifactory, not Azure ACR, if artifact registry is mentioned.
- Use GitHub Actions OIDC and Terraform AzureRM only if CI/CD or IaC is in scope.
- Mark unsupported conclusions as `[inferred]`.
- Do not fabricate business rules, metrics, compliance obligations, implementation details, agents, personas, or skills.

# Required document sections

Create a formal Markdown requirements document with these sections:

1. Document title (a single H1 heading; do not emit a literal "Title" section heading)
2. Document control
3. Executive summary
4. Business problem
5. Product vision
6. Goals and success criteria
7. Personas and stakeholders
8. Aara operating model
9. Traceability discipline
10. Scope overview: MVP, Phase 2, out of scope
11. Current-state context
12. Target-state capabilities
13. Functional requirements with stable IDs, e.g. `BA-MVP-FR-001`
14. Non-functional requirements
15. Integrations
16. Human-in-the-loop controls
17. Data, security, privacy, and compliance requirements
18. Reporting and analytics
19. User stories with acceptance criteria
20. Product-level and MVP acceptance criteria
21. Evaluation and quality gates
22. Risks
23. Assumptions
24. Dependencies
25. Open questions
26. Traceability matrix mapping source evidence to requirement IDs
27. Recommended next steps

# Evidence and traceability rules

- Use the CC BA Agent files as the MVP source of truth.
- Use `ref-requirements.md` for structure, style, and Phase 2 enterprise BA capabilities.
- Every functional requirement should have a stable ID.
- Each requirement should cite source evidence where possible.
- Use `[inferred]` for conclusions not directly supported by source text.
- Put uncertainty into Open Questions instead of guessing.
- The traceability matrix should map source documents or source sections to requirement IDs.
- Do not cite this prompt as evidence for any functional or non-functional requirement. It may be referenced only as task framing (scope separation, operating model, quality gates), and such references must be labeled as task framing.

# Quality gates

After creating the file, verify that the Markdown clearly contains:

- `Aara operating model`
- `aara-business-analyst`
- `Traceability discipline`
- `MVP`
- `Phase 2`
- `Functional Requirements`
- `Non-Functional Requirements`
- `Human-in-the-Loop`
- `Open Questions`
- `Traceability Matrix`
- `Acceptance Criteria`
- `Teams`

Also verify it does not contain:

- `Slack`
- `Azure ACR`

# Final response

At the end, report only:

- saved file path
- source files reviewed
- Aara agents/persona lenses/skills applied or unavailable
- verification result
- any source files that could not be reviewed due to format/tooling limitations
