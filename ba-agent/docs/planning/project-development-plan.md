# BA Agent Project Development Plan

Decision-grade, implementation-oriented planning baseline for the Business Analyst AI Agent. This plan is derived from the checked-in requirements and companion design documents; it is not a delivery commitment until scope, capacity, owners, and pilot approvals are confirmed.

---

## 1. Document control

| Field                        | Value                                                                                                                                  |
| ---------------------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| Document name                | BA Agent Project Development Plan                                                                                                      |
| Version                      | 0.3                                                                                                                                    |
| Change note (v0.3)           | Added pre-F1 technical baseline, clarified G2 no-write vs. G3 write-rejection proof, and tightened fleet sequencing assumptions.       |
| Change note (v0.2)           | Clarified decisions by gate, G0-to-G1 semantics, and Phase 1 fixture-placeholder ownership before fleet execution.                     |
| Status                       | Draft planning artifact for human review; not approved for delivery commitment                                                         |
| Prepared date                | 2026-07-02                                                                                                                             |
| Primary baseline             | `docs/requirements/business-analyst-agent-requirements.md` v0.4                                                                        |
| Companion inputs             | `ba_agent_runtime_architecture.md`, `ba_agent_mcp_tool_contracts.md`, `ba_agent_evaluation_harness.md`, `ba_agent_operations_model.md` |
| Task-framing input only      | `docs/requirements/ba-requirements-prompt.md` — used for workspace conventions and phase framing only, not product evidence            |
| Repository instruction input | `.github/copilot-instructions.md`                                                                                                      |
| Planning fixed constraint    | First thin-slice scope: Teams standup summary using synthetic Jira/Git fixtures, trace IDs, evidence refs, and no live writes          |
| Accountable owner baseline   | RAJA                                                                                                                                   |
| Variable constraints         | Calendar dates, team capacity, budgets, pilot team, and production approval path are [RAJA]                                             |

### Source register for this plan

| Plan source ID   | Document                                 | How this plan uses it                                                                                                                                                     |
| ---------------- | ---------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| PLAN-SRC-REQ     | `business-analyst-agent-requirements.md` | Primary scope, requirement IDs, MVP/Phase 2 split, user stories, acceptance criteria, risks, assumptions, dependencies, open questions                                    |
| PLAN-SRC-ARCH    | `ba_agent_runtime_architecture.md`       | Proposed runtime topology, Azure-primary stack, orchestrator/gateway separation, identity, CI/CD, observability, failure modes                                            |
| PLAN-SRC-MCP     | `ba_agent_mcp_tool_contracts.md`         | Proposed MCP tool conventions, read/write boundaries, approval refs, audit records, validation register                                                                   |
| PLAN-SRC-EVAL    | `ba_agent_evaluation_harness.md`         | Evaluation metrics, golden test sets, hard gates, sample thin-slice cases                                                                                                 |
| PLAN-SRC-OPS     | `ba_agent_operations_model.md`           | Proposed RACI, support tiers, incident response, release management, rollback model                                                                                       |
| PLAN-SRC-COPILOT | `.github/copilot-instructions.md`        | Repository conventions: docs-only state, evidence discipline, Teams surface, Azure-primary, JFrog Artifactory, GitHub Actions OIDC, Terraform AzureRM, human-gated writes |
| PLAN-SRC-FRAMING | `ba-requirements-prompt.md`              | Task framing only: phase separation, workspace conventions, quality-gate expectations                                                                                     |

### Evidence discipline

- Source-backed facts cite the checked-in document or requirement IDs that support them.
- Assumptions are explicitly listed and are not treated as approved decisions.
- Unsupported or owner-dependent details are marked `[inferred]` or `[RAJA]`.
- The prompt/framing document is not cited as product evidence; it is cited only for planning/task conventions.
- No delivery dates, budgets, named people, completed implementation work, or production readiness are claimed.

---

## 2. Planning assumptions and constraints

### Assumptions

| ID           | Assumption                                                                                                                                                                | Basis                                                                                   | Accountable owner |
| ------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------- | ----------------- |
| PLAN-ASM-001 | Planning starts from the v0.4 requirements baseline, which is draft-for-review and not an approved delivery commitment.                                                   | PLAN-SRC-REQ document control; PLAN-SRC-COPILOT source-of-truth notes                   | RAJA              |
| PLAN-ASM-002 | The first build increment should prove the smallest safe end-to-end path: Teams-facing standup summary from synthetic Jira/Git fixtures with trace IDs and evidence refs. | User task context; PLAN-SRC-REQ BA-US-MVP-001, BA-AC-MVP-002; PLAN-SRC-EVAL GTS-STANDUP | RAJA              |
| PLAN-ASM-003 | No live system-of-record writes are allowed in the first thin slice.                                                                                                      | PLAN-SRC-REQ BA-HIL-006, BA-AUT-001 through BA-AUT-005; PLAN-SRC-MCP write restriction  | RAJA              |
| PLAN-ASM-004 | Synthetic fixtures are acceptable for initial development and automated evaluation until classification and pilot data handling are approved.                             | PLAN-SRC-EVAL golden test set rule; PLAN-SRC-REQ BA-DSPC-002, BA-OQ-010                 | RAJA              |
| PLAN-ASM-005 | The proposed runtime stack is Azure-primary and should follow the companion architecture unless changed by recorded architect decision.                                   | PLAN-SRC-ARCH topology and CI/CD sections; PLAN-SRC-COPILOT conventions                 | RAJA              |
| PLAN-ASM-006 | MCP contracts are design proposals and cannot be treated as build-authoritative until validated against actual server implementations.                                    | PLAN-SRC-MCP status and validation register                                             | RAJA              |
| PLAN-ASM-007 | Phase 2 Enterprise BA capabilities remain out of MVP scope until an explicit Phase 2 readiness gate is passed.                                                            | PLAN-SRC-REQ scope overview, BA-QG-008, BA-RISK-002                                     | RAJA              |

### Constraints

| Constraint                                                              | Planning implication                                                                             | Source                                                                                                       |
| ----------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------ |
| Teams/Copilot 365 is the MVP user surface.                              | UX, output cards, approvals, and pilot workflow must be designed around Teams.                   | PLAN-SRC-REQ BA-MVP-FR-001; PLAN-SRC-ARCH source-fixed constraints; PLAN-SRC-COPILOT high-level architecture |
| LangGraph and MCP are source-fixed design constraints.                  | The build plan must include a router/capability graph and MCP-mediated tool access.              | PLAN-SRC-REQ design constraints; PLAN-SRC-ARCH source-fixed constraints                                      |
| Human-gated writes are mandatory.                                       | Tool write paths require recorded approval,`approval_ref`, idempotency, and audit.               | PLAN-SRC-REQ BA-HIL-006; PLAN-SRC-MCP contract conventions; PLAN-SRC-ARCH MCP gateway decision               |
| The MCP gateway is the enforcement boundary.                            | Approval validation, scope checks, rate limiting, and audit must be outside the model loop.      | PLAN-SRC-ARCH topology/security design; PLAN-SRC-MCP write restriction                                       |
| Repository is currently documentation-only.                             | First implementation work must create source structure, executable harness, and CI from scratch. | PLAN-SRC-COPILOT project state                                                                               |
| If CI/IaC is introduced, use GitHub Actions OIDC and Terraform AzureRM. | No stored cloud credentials; infra changes flow through reviewed IaC.                            | PLAN-SRC-ARCH environments and CI/CD; PLAN-SRC-COPILOT conventions                                           |
| If container registry is mentioned, use JFrog Artifactory.              | Image promotion and rollback references should use Artifactory tags.                             | PLAN-SRC-ARCH environments and CI/CD; PLAN-SRC-OPS rollback; PLAN-SRC-COPILOT conventions                    |
| Evaluation thresholds are mostly owner-set.                             | Do not invent numeric pass thresholds; use hard gates only where the harness defines them.       | PLAN-SRC-EVAL metric definitions and release gate procedure                                                  |

### Fixed constraint and trade-off statement

The fixed constraint for this plan is **thin-slice scope**, not date or capacity. The plan should not promise a calendar delivery date until named roles, capacity, pilot tenant access, tool-owner approvals, and security/privacy gates are confirmed. Quality is not a lever: approval-gate enforcement, evidence discipline, and phase separation are release floors.

---

## 3. Development strategy and MVP thin-slice rationale

### Strategy

Use a **synthetic-first, gate-first, evidence-first** delivery strategy:

1. **Synthetic-first:** build the first flow against synthetic Jira/Git fixtures before any pilot data access. This supports classification safety and deterministic evaluation (PLAN-SRC-EVAL golden test sets; PLAN-SRC-REQ BA-DSPC-002).
2. **Gate-first:** implement approval/audit/idempotency controls before any live write-capable workflow. The gateway must reject unapproved writes mechanically, not by prompt instruction (PLAN-SRC-MCP contract conventions; PLAN-SRC-ARCH MCP gateway decision).
3. **Evidence-first:** every factual output must carry evidence refs and a `trace_id`; unsupported statements are marked `[inferred]` or `[RAJA]` (PLAN-SRC-REQ BA-NFR-001, BA-NFR-006; PLAN-SRC-OPS support tiers).
4. **One capability first:** prove standup summary before adding sprint planning, retrospective, or health monitoring. This avoids mixing routing, writes, severity taxonomy, and Confluence publishing in the first build increment.
5. **Phase separation:** MVP backlog stays limited to standup, planning, retro, and health; Phase 2 enterprise requirements/story/process/traceability features are readiness work only until separately approved (PLAN-SRC-REQ scope overview, BA-QG-008).

### Why the first thin slice is Teams standup summary

| Rationale                                                                                                                                                          | Evidence / discipline                                                           |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------- |
| It exercises the primary Teams-facing interaction pattern without requiring live source-system mutation.                                                           | PLAN-SRC-REQ BA-MVP-FR-001, BA-MVP-FR-005, BA-AUT-001                           |
| It uses only read-side Jira/Git data, which is lower-risk than planning publish or Confluence publish flows.                                                       | PLAN-SRC-REQ BA-MVP-FR-004, BA-HIL-006; PLAN-SRC-MCP Jira/Git contracts         |
| It validates the core evidence loop: synthetic source data → routed intent → summary → blockers/risks → evidence refs → traceable output.                          | PLAN-SRC-EVAL GTS-STANDUP and BA-EM-002/006/007                                 |
| It creates reusable platform primitives: router, fixture loader, gateway audit record, Teams Adaptive Card payload, OpenTelemetry`trace_id`.                       | PLAN-SRC-ARCH topology and observability; PLAN-SRC-OPS support model            |
| It defers unresolved decisions that are not required for the first proof: sprint-scope publish semantics, severity taxonomy, Confluence posting, calendar privacy. | PLAN-SRC-REQ BA-OQ-005 through BA-OQ-009; PLAN-SRC-MCP cross-cutting open items |

---

## 4. Phase plan and gates

No calendar dates are assigned. Each phase exits only through its gate. A committed date requires confirmed capacity and dependency lead times.

| Phase                                         | Objective                                                                  | Key deliverables                                                                                                                                                                                                       | Gate / exit criteria                                                                                                                                                                    | Critical dependencies                                                                  |
| --------------------------------------------- | -------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| Phase 0 — Pre-work and decision closure       | Establish build authority and unblock implementation.                      | Approved planning baseline; RAJA accountable-owner assignment; MVP pilot boundary draft; security/classification decision path; tool-owner validation plan.                                                             | **G0: Build-start gate** — RAJA agrees this plan is the working baseline, or records explicit deviations.                                                                               | PLAN-SRC-REQ BA-OQ-001 through BA-OQ-015; PLAN-SRC-OPS RACI                            |
| Phase 1 — Engineering foundation              | Create runnable project skeleton and non-production development path.      | Phase 1 technical baseline; source tree; Python orchestrator service skeleton`[inferred]`; MCP gateway facade/fake skeleton `[inferred]`; synthetic fixture path/placeholder; local/dev config; local unit/typecheck/eval-placeholder commands; CI skeleton using GitHub Actions OIDC only when cloud deployment starts. | **G1: Skeleton gate** — repository has runnable unit/typecheck/evaluation-placeholder commands, no secrets in code, no live integrations, and documentation for local synthetic run.      | Docs-only repo state; PLAN-SRC-ARCH environments/CI/CD                                 |
| Phase 2 — First thin slice: synthetic standup | Prove end-to-end standup summary with synthetic Jira/Git data.             | Teams Adaptive Card payload; LangGraph standup route; Jira/Git fixture readers; evidence refs;`trace_id`; no live writes; GTS-STANDUP seed cases.                                                                      | **G2: Thin-slice demo gate** — STD-style cases pass; output cites synthetic Jira/Git refs; degraded Git case is honest; no write tool is invoked.                                       | PLAN-SRC-EVAL GTS-STANDUP samples; PLAN-SRC-REQ BA-AC-MVP-002                          |
| Phase 3 — Gateway and evaluation hardening    | Make control-plane behavior testable before live integrations.             | Gateway audit record; denied/degraded/throttled status handling; approval-ref rejection tests; trace propagation; initial golden harness execution record.                                                             | **G3: Control gate** — BA-EM-005 approval-gate bypass count is zero in adversarial tests; all tool calls produce audit records.                                                         | PLAN-SRC-MCP conventions; PLAN-SRC-EVAL GTS-GATE; PLAN-SRC-ARCH gateway design         |
| Phase 4 — Sandbox integration readiness       | Replace synthetic reads with validated sandbox MCP reads where approved.   | Tool-owner validation register updates; sandbox Jira/Git schemas matched; scope-denied and degraded cases; Teams sandbox/channel approval [RAJA].                                                                       | **G4: Sandbox gate** — only validated read tools are enabled; unvalidated tools remain stubbed or blocked; all deviations are documented.                                               | PLAN-SRC-MCP validation register; PLAN-SRC-REQ BA-QG-006                               |
| Phase 5 — MVP capability expansion            | Add planning, retro, and health capabilities after control path is proven. | Planning recommendation with approval request only; retro draft-only behavior; health checks with placeholder severity marked [RAJA]; expanded golden sets.                                                             | **G5: MVP candidate gate** — BA-EM-005 = 0, BA-EM-009 = 0, owner-set thresholds reviewed or explicitly waived.                                                                          | PLAN-SRC-REQ BA-MVP-FR-006 through BA-MVP-FR-011; PLAN-SRC-EVAL release gate procedure |
| Phase 6 — MVP pilot readiness and pilot       | Run a controlled pilot with approved live scopes only.                     | Pilot runbook; support/RACI; rollback plan; audit review cadence; release notes with harness run ID; approved pilot channels/projects/repos.                                                                           | **G6: Pilot authorization gate** — RAJA approves limited live use after security/privacy, tool, platform, and QA review lanes are satisfied.                                            | PLAN-SRC-OPS release management, incident response, support tiers                      |
| Phase 7 — Phase 2 readiness                   | Prepare, but do not implement, Enterprise BA capabilities.                 | Phase 2 prioritization brief; tool approval matrix; data/source classification plan; evaluation approach for GTS-P2-REQ.                                                                                               | **G7: Phase 2 readiness gate** — RAJA confirms first Phase 2 capability set and approves a separate plan.                                                                               | PLAN-SRC-REQ Phase 2 scope; PLAN-SRC-EVAL GTS-P2-REQ                                   |

### Critical path

The critical path is:

1. Build authority and RACI decisions.
2. Security/classification path for fixtures and later pilot data.
3. Source skeleton and executable evaluation harness.
4. Standup route + synthetic Jira/Git fixture model.
5. Gateway audit/approval rejection controls.
6. Tool-owner validation for sandbox Jira/Git reads.
7. Pilot approval and support model.

Adding capacity outside this chain will not shorten the pilot date unless it removes a dependency on this chain.

### Replan triggers

Replan if any of the following occurs:

- MVP scope expands beyond standup/planning/retro/health before G5.
- A critical-path owner or tool approval is unavailable after the build-start gate.
- Security/privacy blocks use of planned pilot data or Teams channel scope.
- MCP tool schemas differ materially from the proposed contracts.
- Any approval-gate bypass succeeds in GTS-GATE.
- Phase 2 capability work is requested before G7 without explicit change control.
- Planned team capacity drops below the confirmed baseline once capacity is known.

---

## 5. Work breakdown structure

RAJA is the accountable owner for all work packages. Role names below are execution/review lanes, not separate accountable owners.

| WBS ID | Work package                       | Deliverables                                                                                            | Dependencies                              | Accountable owner | Execution / review lanes                                      | Exit criteria                                                             |
| ------ | ---------------------------------- | ------------------------------------------------------------------------------------------------------- | ----------------------------------------- | ----------------- | ------------------------------------------------------------- | ------------------------------------------------------------------------- |
| WBS-00 | Baseline and decisions             | Accepted development plan; decision log; open-question triage                                           | PLAN-SRC-REQ BA-OQ list                   | RAJA              | Product Owner, Delivery Lead, BA SME, Architect               | G0 gate passed or deviations recorded                                     |
| WBS-01 | Repository and runtime skeleton    | Source layout; service entry points; configuration pattern; local synthetic run path                    | WBS-00                                    | RAJA              | Platform engineer, AI engineer                                | Project can run a no-op request locally with no secrets                   |
| WBS-02 | CI/evaluation foundation           | Unit/eval command; fixture validation; initial workflow; artifact retention plan [RAJA]                 | WBS-01                                    | RAJA              | Platform engineer, QA / AI evaluation reviewer                | Synthetic tests run in CI; no cloud credentials stored                    |
| WBS-03 | Teams interaction surface          | Teams Adaptive Card schema/payload builder; footer with`trace_id`; approved-channel abstraction         | WBS-01                                    | RAJA              | Frontend/Teams engineer, BA SME                               | Card payload validates and carries evidence refs                          |
| WBS-04 | LangGraph router and standup graph | Intent route for standup; unsupported request handling; graph version stamping                          | WBS-01                                    | RAJA              | AI engineer                                                   | Standup prompt routes correctly; unsupported/Phase 2 requests are flagged |
| WBS-05 | Synthetic Jira/Git fixtures        | Fixture schema; Jira story/status refs; Git commit/PR refs; degraded/denied cases                       | WBS-02                                    | RAJA              | QA, AI engineer, BA SME                                       | STD sample cases have deterministic fixture inputs and expected outputs   |
| WBS-06 | Standup summary generation         | Summary sections for status, blockers, risks, assumptions, open questions; no unsupported claims        | WBS-03, WBS-04, WBS-05                    | RAJA              | AI engineer, BA SME                                           | BA-AC-MVP-002 behavior met against synthetic fixtures                     |
| WBS-07 | MCP gateway control baseline       | Tool allowlist; audit record; denied/degraded/throttled statuses; write rejection path                  | WBS-01                                    | RAJA              | Platform engineer, Security engineer                          | GTS-GATE write attempts without valid approval are rejected and audited   |
| WBS-08 | Observability and traceability     | OpenTelemetry trace propagation; audit`trace_id`; model/prompt/graph version fields `[inferred]`        | WBS-07                                    | RAJA              | Platform engineer                                             | Any output can be traced to input fixtures and tool calls                 |
| WBS-09 | Evaluation harness seed            | GTS-STANDUP, GTS-ROUTER, GTS-GATE seed cases; BA-EM metric capture                                      | WBS-02, WBS-05, WBS-06, WBS-07            | RAJA              | QA / AI evaluation reviewer, BA SME                           | Hard gates measured; owner-set thresholds left [RAJA]                    |
| WBS-10 | Sandbox MCP validation             | Jira/Git actual schema validation; validation register updates; scope approval records                  | WBS-07, WBS-09                            | RAJA              | Tool owners, Platform engineer                                | Validated read tools can replace fixtures in sandbox only                 |
| WBS-11 | Sprint planning MVP                | Backlog/velocity/calendar reads; recommendation only; approval request flow; no publish unless approved | WBS-07, WBS-10, calendar privacy decision | RAJA              | AI engineer, Scrum Master, Tool owners                        | Planning output cannot publish without valid approval_ref                 |
| WBS-12 | Retrospective MVP                  | Jira metrics read; Confluence draft-only behavior; missing metric handling                              | WBS-07, WBS-10, Confluence owner decision | RAJA              | AI engineer, BA SME, Confluence owner                         | Metrics are never estimated; publish remains gated                        |
| WBS-13 | Sprint health MVP                  | Scheduled/webhook trigger path; severity taxonomy placeholder; advisory escalation                      | WBS-07, WBS-10, severity decision         | RAJA              | AI engineer, Scrum Master / PM                                | Health alerts are recommendations and cite evidence                       |
| WBS-14 | Operations readiness               | RACI; support tiers; incident response; rollback; release notes template                                | WBS-00, WBS-09                            | RAJA              | Delivery Lead, Platform engineer, Security owner              | Pilot runbook approved                                                    |
| WBS-15 | MVP pilot                          | Limited live pilot with approved scopes; monitoring; feedback capture; post-pilot assessment            | WBS-10 through WBS-14                     | RAJA              | Delivery Lead, Product Owner, Scrum Master, Platform engineer | Pilot exit report recommends continue/adjust/stop                         |
| WBS-16 | Phase 2 readiness                  | Prioritized Phase 2 scope; candidate integrations; GTS-P2-REQ plan                                      | WBS-15                                    | RAJA              | Product Owner, BA SME, Architect, Tool owners                 | Separate Phase 2 plan approved before build                               |

---

## 6. First thin-slice plan: Teams standup summary

### Scope

Build and validate a standup-summary flow that:

- Accepts a standup-style prompt from a Teams-oriented interaction adapter or approved dev-channel path [RAJA].
- Routes the request to the standup capability.
- Reads only synthetic Jira/Git fixtures.
- Produces a Teams Adaptive Card payload with status, blockers, risks, assumptions/open questions, evidence refs, and `trace_id`.
- Executes no live writes to Jira, Git, Confluence, Calendar, or Teams.
- Rejects any attempted write path at the gateway.

### Explicitly out of scope for the thin slice

| Out-of-scope item                                | Reason                                                                  |
| ------------------------------------------------ | ----------------------------------------------------------------------- |
| Live Jira/Git reads                              | Requires tool-owner scope validation and security review.               |
| Jira, Git, Confluence, Calendar, or Teams writes | First slice is no-live-write and read/evaluation focused.               |
| Sprint planning recommendations                  | Requires backlog, velocity, calendar privacy, and approval workflow.    |
| Retrospective reports                            | Requires Jira metric definitions and Confluence draft/publish decision. |
| Sprint health monitoring                         | Requires severity taxonomy and schedule/webhook decisions.              |
| Phase 2 requirement/story/process artifacts      | Explicitly separated by MVP/Phase 2 gates.                              |

### Synthetic fixture design

| Fixture type             | Minimum fields                                                                                                                              | Evidence refs                                  |
| ------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------- |
| Jira sprint status       | Synthetic project key, sprint ID, issue key, summary, status, assignee placeholder, story points, flagged status, last transition timestamp | `jira:synthetic:<project>/<issue-key>`         |
| Jira blocker/risk marker | Issue key, blocked/flagged field, blocker note, timestamp                                                                                   | `jira:synthetic:<project>/<issue-key>#blocker` |
| Git activity             | Synthetic repository, commit SHA-like ID, PR ID, author placeholder, title/message, timestamp                                               | `git:synthetic:<repo>/<commit-or-pr>`          |
| Tool status              | `ok`, `degraded`, `denied`, `throttled`; unavailable source list                                                                            | `tool:synthetic:<tool-name>/<case-id>`         |
| Evaluation case metadata | Case ID, input prompt, expected routing, expected output characteristics                                                                    | `eval:<case-id>`                               |

All fixture identifiers are synthetic placeholders. They are not evidence of completed project work.

### Output contract

The standup card must include:

1. Title: daily standup summary for the synthetic sprint.
2. Status snapshot: counts by status, each count backed by fixture refs.
3. Completed / in-progress / blocked items: each item cites issue key or PR/commit ref.
4. Risks: stalled or flagged items only when supported by fixture fields; detection rules marked [RAJA] until owner-approved.
5. Data quality section: source statuses, missing/degraded sources, retrieved time.
6. Assumptions/open questions: separate unsupported details instead of guessing.
7. Footer: `trace_id`, graph version, fixture set version, and evaluation case ID.

### Thin-slice acceptance criteria

| ID        | Acceptance criterion                                                                                                | Linked source concepts                                                               |
| --------- | ------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------ |
| TS-AC-001 | Standup prompt routes to the standup graph; unsupported and Phase 2 prompts do not route to standup.                | PLAN-SRC-REQ BA-AC-MVP-001; PLAN-SRC-EVAL GTS-ROUTER                                 |
| TS-AC-002 | Output uses Jira/Git fixture evidence only and marks any unsupported conclusion as `[inferred]` or `[RAJA]`.       | PLAN-SRC-REQ BA-NFR-001, BA-NFR-006; PLAN-SRC-EVAL BA-EM-002/003                     |
| TS-AC-003 | The Adaptive Card payload contains evidence refs and`trace_id` in a repeatable schema.                              | PLAN-SRC-MCP Teams contract; PLAN-SRC-ARCH observability; PLAN-SRC-OPS trace support |
| TS-AC-004 | If Git fixture status is`degraded`, the summary states Git data is unavailable and does not invent commit activity. | PLAN-SRC-MCP failure handling; PLAN-SRC-EVAL STD-002                                 |
| TS-AC-005 | The thin-slice path invokes no write-like tool. Approval-ref rejection and audited write-bypass proof are G3 control-gate criteria, not G2 completion criteria. | PLAN-SRC-MCP write restriction; PLAN-SRC-EVAL BA-EM-005/GTS-GATE |
| TS-AC-006 | No live system-of-record write occurs during the thin-slice run.                                                    | PLAN-SRC-REQ BA-HIL-006; user task context                                           |

---

## 7. Critical decisions by gate

DEC-001, DEC-002, DEC-003, and DEC-007 establish the current synthetic-only build authority. DEC-004, DEC-005, and DEC-006 are required before sandbox or non-synthetic work, but they do **not** block G1-G3 synthetic/local execution under the decision log.

| Decision ID | Decision                                                                                                           | Why it is critical                                                                                                 | Blocks                       | Accountable owner |
| ----------- | ------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------ | ---------------------------- | ----------------- |
| DEC-001     | Confirm product name and MVP positioning.                                                                          | Prevents confusion between BA support and Scrum Master support.                                                    | Stakeholder comms, UX labels | RAJA              |
| DEC-002     | Record RAJA as accountable owner across Product Owner, Delivery Lead, Architect, BA SME, QA, Security/privacy owner, Platform owner, and tool-owner responsibilities. | Gated plan needs a named accountable owner even when role-specific delegates are added later.                       | G0, approvals, RACI          | RAJA              |
| DEC-003     | Approve first thin-slice scope and no-live-write rule.                                                             | Keeps MVP start small and safe.                                                                                    | G0/G2                        | RAJA              |
| DEC-004     | Confirm pilot candidate boundaries: team, Jira project, repo, Teams channel, Confluence space, calendar scope.     | Defines access, evaluation relevance, and blast radius.                                                            | G4/G6                        | RAJA              |
| DEC-005     | Confirm classification handling for fixtures, prompts, outputs, logs, and later pilot data.                        | Required before any non-synthetic data use.                                                                        | G4/G6                        | RAJA              |
| DEC-006     | Validate actual MCP server names, schemas, auth model, rate limits, and scopes.                                    | Proposed contracts may differ from implementation reality.                                                         | G4                           | RAJA              |
| DEC-007     | Confirm approval-record model and`approval_ref` semantics.                                                         | Controls every write path and BA-EM-005 hard gate.                                                                 | G3/G5                        | RAJA              |
| DEC-008     | Decide what sprint-planning “publish” means.                                                                       | Determines whether Jira write tooling is needed.                                                                   | Sprint planning MVP          | RAJA              |
| DEC-009     | Define blocker/severity taxonomy.                                                                                  | Required for health monitoring labels and precision/recall evaluation.                                             | Sprint health MVP            | RAJA              |
| DEC-010     | Confirm Confluence retro behavior: draft-only, approval-gated publish, or other.                                   | Prevents accidental publication.                                                                                   | Retrospective MVP            | RAJA              |
| DEC-011     | Confirm Azure OpenAI model/region and retention expectations.                                                      | Affects security review, audit, and release reproducibility.                                                       | G6                           | RAJA              |
| DEC-012     | Confirm Container Apps vs. AKS and state/audit stores.                                                             | Companion architecture proposes Container Apps, Postgres, Event Hubs; architect must approve or record deviations. | Infrastructure build         | RAJA              |

---

## 8. Risk register

| Risk ID       | Risk                                                                                              | Likelihood | Impact   | Mitigation                                                                                                | Trigger                                                                              |
| ------------- | ------------------------------------------------------------------------------------------------- | ---------- | -------- | --------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------ |
| PLAN-RISK-001 | Role-specific delegates and review lanes remain unnamed even though RAJA is accountable.           | Medium     | Medium   | Keep RAJA as accountable owner; add delegates or reviewers when needed without changing accountability.     | A gate requires specialist review and no delegate/reviewer is identified.            |
| PLAN-RISK-002 | Tool contracts do not match actual MCP server schemas or permissions.                             | Medium     | High     | Run WBS-10 validation register before sandbox replacement; keep unvalidated tools stubbed/blocked.        | Schema diff or auth behavior differs from PLAN-SRC-MCP.                              |
| PLAN-RISK-003 | Classification/security review limits use of pilot data or collaboration outputs.                 | Medium     | High     | Keep G2 synthetic-only; require DEC-005 before non-synthetic data.                                        | Security owner rejects planned data class or retention model.                        |
| PLAN-RISK-004 | A write path bypasses human approval due to implementation bug.                                   | Low        | Critical | Gateway-enforced`approval_ref`, idempotency, audit; GTS-GATE hard gate; kill switch for write tools.      | BA-EM-005 count is greater than zero.                                                |
| PLAN-RISK-005 | Phase 2 enterprise BA features creep into MVP backlog.                                            | Medium     | Medium   | Enforce BA-QG-008 and G7; route Phase 2 requests to separate planning.                                    | MVP output or backlog includes BRD/user-story/process-map features without approval. |
| PLAN-RISK-006 | Sprint-health severity taxonomy is not defined, blocking reliable health alerts.                  | High       | Medium   | Do not start health-monitor labels until DEC-009; park ambiguous cases as [RAJA].                         | GTS-HEALTH cases cannot be labeled.                                                  |
| PLAN-RISK-007 | Jira fields and metrics vary by project, causing fabricated or inconsistent retro/health outputs. | Medium     | Medium   | Require missing fields to return`null`/`missing_fields[]`; validate project schema before MVP expansion.  | `get_sprint_metrics` lacks expected fields.                                          |
| PLAN-RISK-008 | Teams/Copilot approval path takes longer than expected.                                           | Medium     | High     | Start tenant/app/channel approval in Phase 0; keep card payload validation independent from live posting. | Approved dev/pilot channel unavailable by G4.                                        |
| PLAN-RISK-009 | Evaluation thresholds are not set by owners, delaying release decisions.                          | Medium     | Medium   | Use hard gates immediately; mark owner thresholds [RAJA]; schedule threshold review before G5.            | BA-EM metrics computed but no pass/fail owner threshold exists.                      |

---

## 9. Validation and evaluation plan

The repository currently has no executable harness checked in; `ba_agent_evaluation_harness.md` is a specification, not an implementation (PLAN-SRC-COPILOT). Therefore, implementation must create the harness before claiming executable validation.

### Evaluation scope by phase

| Phase gate           | Evaluation focus                                                                       | Harness concepts                                                                  |
| -------------------- | -------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| G1 Skeleton          | Basic run command, typecheck command, eval/fixture-validation placeholder command, fixture path/placeholder, no secrets, no live systems | Repository validation discipline; synthetic fixture readiness                     |
| G2 Thin slice        | Standup routing, Jira/Git evidence refs, degraded Git honesty, Adaptive Card structure | GTS-STANDUP, GTS-ROUTER, BA-EM-001/002/003/007                                    |
| G3 Control           | Approval-gate rejection, audit record, idempotency behavior                            | GTS-GATE, BA-EM-005 hard gate                                                     |
| G4 Sandbox           | Validated read tools, denied/degraded/throttled behavior, schema conformance           | BA-QG-006, MCP validation register                                                |
| G5 MVP candidate     | Full MVP regression across standup, planning, retro, health; phase separation          | GTS-STANDUP, GTS-PLANNING, GTS-RETRO, GTS-HEALTH, GTS-ROUTER, BA-EM-009 hard gate |
| G6 Pilot             | Staging run ID, sampled human review, support readiness, rollback readiness            | Release gate procedure; BA-EM-006; PLAN-SRC-OPS release management                |
| G7 Phase 2 readiness | Rough input → requirements discovery only in planning context                          | GTS-P2-REQ; BA-QG-008                                                             |

### Hard gates

| Gate                    | Required result                     | Source                                                   |
| ----------------------- | ----------------------------------- | -------------------------------------------------------- |
| Approval-gate bypass    | BA-EM-005 must be zero.             | PLAN-SRC-EVAL metric definitions and release procedure   |
| MVP/Phase 2 separation  | BA-EM-009 must be zero.             | PLAN-SRC-EVAL metric definitions; PLAN-SRC-REQ BA-QG-008 |
| Unsupported live writes | Zero live writes in thin-slice run. | PLAN-SRC-REQ BA-HIL-006; user task context               |

### Owner-threshold gates

The following are measured but not assigned numeric thresholds until owners set them: routing accuracy, evidence-link coverage, citation correctness, output-structure conformance, blocker-detection precision/recall, and regression coverage. These thresholds remain [RAJA] per PLAN-SRC-EVAL.

### Human review

Before pilot, BA SME and QA should sample outputs for:

- Evidence refs that actually support claims.
- Clear separation of facts, assumptions, inferences, and open questions.
- Draft/advisory labeling.
- No hidden sprint commitment or corrective-action commitment.
- Teams card readability for a Scrum Master or team member.

---

## 10. Suggested immediate next steps

1. Review and approve or amend this plan as the working delivery baseline.
2. Treat DEC-001, DEC-002, DEC-003, and DEC-007 as the synthetic-only build-start baseline; keep DEC-004, DEC-005, and DEC-006 open for their later gates.
3. Create a decision log and RACI using the role placeholders in this plan.
4. Start Teams/app/channel approval discovery and tool-owner validation planning in parallel.
5. Scaffold the repository for the local foundation, including fixture/evaluation placeholders only.
6. Build actual GTS-STANDUP, GTS-ROUTER, and GTS-GATE seed fixtures in the gated thin-slice/control phases before model/prompt tuning.
7. Demonstrate the first Adaptive Card payload from synthetic Jira/Git fixtures with `trace_id` and evidence refs.
8. Keep sprint planning, retrospective, health monitoring, and Phase 2 work behind their gates until the thin-slice and control gates pass.
