# BA Agent All-Phase Execution Prompt Pack

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent All-Phase Execution Prompt Pack |
| Version | 1.2 |
| Change note (v1.2) | Marked [P9C]/[Q9C] complete after creating the HLD owner-review package and closing `HLD-G3` for RAJA review. |
| Change note (v1.1) | Marked [P9B]/[Q9B] complete after creating the draft/advisory BA Agent HLD and validating hard-gate/non-authorization posture. |
| Change note (v1.0) | Added `[F9]` HLD creation lane after RAJA scope-change directive, completed [P9A]/[Q9A] setup tracking, and added HLD draft/review prompt pairs. |
| Change note (v0.9) | Added full skill inventory routing, explicit current-active-phase note for `F8`, and synced phase-order wording to the implementation baseline. |
| Status | Draft prompt pack for GitHub Copilot / Aara coding-session execution |
| Prepared date | 2026-07-06 |
| Accountable owner | RAJA |
| Platform idiom | GitHub Copilot / Aara Markdown task prompts |
| Repository state at authoring | Documentation-only baseline unless a future prompt has already created source, tests, local commands, or harness code |
| Covered phases | Phase 0 through Phase 9 (including post-G7 Phase 2 first-slice execution governance and HLD creation) |
| Source context read | `.github/copilot-instructions.md`; `docs/planning/project-development-plan.md`; `docs/planning/decision-log.md`; `docs/planning/phase-2-implementation-plan.md`; `docs/requirements/business-analyst-agent-requirements.md`; `docs/requirements/ba_agent_runtime_architecture.md`; `docs/requirements/ba_agent_mcp_tool_contracts.md`; `docs/requirements/ba_agent_evaluation_harness.md` |
| Evidence marker policy | Use `[inferred]` for reasonable unsupported implementation choices and `[RAJA]` for owner-dependent decisions, thresholds, dates, names, scopes, or approvals |

---

## How to use this prompt pack

Use this file one prompt pair at a time in a coding session. Each implementation prompt is immediately followed by its QA/review prompt. Run the implementation prompt, run its Tests and Validations, then run the paired QA prompt before moving on.

After each prompt execution, update that prompt entry with:

- `Status`: `✅ Done`, `🟡 Partial`, or `❌ Blocked`.
- `Deliverable path`: the main path or paths changed.
- `Result`: one concise line describing what was delivered or why it is blocked.
- `Test evidence`: exact commands or manual checks run and their results.
- The icon in the prompt heading so table-of-contents views show progress at a glance.

In single-session execution, the active session may update its own prompt status. In fleet execution, prompt-status updates are coordinator-owned: implementation and QA lanes return a handoff, and the coordinator/status lane updates `prompts.md` serially to avoid merge conflicts.

Status icon legend:

| Icon | Status |
| --- | --- |
| ⏳ | Not started |
| 🔄 | In progress |
| ✅ | Completed |
| 🟡 | Partial |
| ❌ | Blocked |

Fleet execution tags identify the staged fleet batches from `fleet_prompt.md`. Use these tags in fleet-coordinator updates, batch summaries, and RAJA gate-review notes. Each fleet tag is fewer than 6 characters.

| Fleet tag | Fleet stage | Includes | Stop condition |
| --- | --- | --- | --- |
| [F0] | Phase 0 — Pre-work and decision closure | [P0A]/[Q0A] through [P0C]/[Q0C] | Stop after G0 readiness evidence is updated. |
| [F1] | Phase 1 — Engineering foundation | [P1T]/[Q1T], [P1A]/[Q1A] through [P1D]/[Q1D] | Stop at G1; do not begin Phase 2 without G1 readiness evidence. |
| [F2] | Phase 2 — Synthetic standup thin slice | [P2A]/[Q2A] through [P2E]/[Q2E] | Stop at G2; do not begin control hardening without thin-slice evidence. |
| [F3] | Phase 3 — Gateway and evaluation hardening | [P3A]/[Q3A] through [P3E]/[Q3E] | Stop at G3; BA-EM-005 must remain zero. |
| [F4] | Phase 4 — Sandbox integration readiness | [P4A]/[Q4A] through [P4E]/[Q4E] | Stop at G4; sandbox readiness does not authorize live pilot use. |
| [F5] | Phase 5 — MVP capability expansion | [P5A]/[Q5A] through [P5E]/[Q5E] | Stop at G5; BA-EM-005 and BA-EM-009 must remain zero. |
| [F6] | Phase 6 — MVP pilot readiness and pilot | [P6A]/[Q6A] through [P6G]/[Q6G] | Stop for explicit non-agent-controlled RAJA approval before [P6F]; production remains separate. |
| [F7] | Phase 7 — Phase 2 readiness | [P7A]/[Q7A] through [P7E]/[Q7E] | Stop with a separate Phase 2 plan for RAJA review. |
| [F8] | Phase 8 — Phase 2 first-slice execution | [P8A]/[Q8A] through [P8E]/[Q8E] | Stop at `P2-G5`; any sandbox/live/production path remains separately authorized. |
| [F9] | Phase 9 — HLD creation | [P9A]/[Q9A] through [P9C]/[Q9C] | Stop at `HLD-G3`; HLD is draft/advisory and repository-evidence-only unless RAJA separately approves more. |

## Current execution posture

The current active execution lane is **[F9] HLD creation**. Treat [F0] through [F8] as prerequisite/historical phases unless the coordinator is replaying them for remediation or audit.

## Internal Aara asset coverage contract (agents, skills, personas)

1. Every prompt pair must record: primary internal agent, secondary/review agent, invoked skill bundle, and persona lenses in coordinator handoff.
2. All internal agents and skills are treated as active coverage assets: direct lane assignment, conditional escalation, or explicit `N/A` with reason per batch.
3. Persona lenses are mandatory for QA/gate packages: BA SME, Product Owner, QA/Evaluation, Architect, Security/Privacy, Compliance/Legal, Platform/Tool Owner, and Delivery Lead; add Executive/FinOps lenses when scope requires.
4. Full internal asset catalog and trigger matrix is maintained in `fleet_prompt.md` and is authoritative for lane assignment.

## Internal skill coverage register (all internal skills)

Every skill family must be explicitly referenced by at least one prompt or coordinator instruction path. Skills may be assigned directly, conditionally escalated, or explicitly marked `N/A` with rationale in coordinator handoffs.

| Skill family | Skills | Default BA Agent trigger |
| --- | --- | --- |
| Core BA documentation/evaluation | `ai-technical-author`, `lsp-setup` | Decision-grade docs, prompt QA, and language-server setup for structured edits. |
| AI model and gateway | `azure-ai`, `azure-aigateway`, `microsoft-foundry`, `customize-cloud-agent` | AI orchestration, gateway policy, and model/governance design tasks. |
| App preparation/deployment/validation | `azure-prepare`, `azure-deploy`, `azure-validate`, `azure-upgrade`, `python-appservice-deploy` | Scaffold/deploy/validate workflows when explicitly in scope. |
| Infrastructure and compute | `azure-enterprise-infra-planner`, `azure-kubernetes`, `azure-compute`, `airunway-aks-setup`, `azure-quotas` | Compute/infrastructure planning or AKS/GPU scenarios. |
| Data/storage/observability | `azure-storage`, `azure-kusto`, `appinsights-instrumentation` | Data-plane, telemetry, and log-query design tasks. |
| Security/compliance/reliability | `azure-compliance`, `azure-rbac`, `azure-reliability`, `azure-diagnostics` | Security posture, RBAC, resilience, and diagnostics hardening. |
| Resource and visualization | `azure-resource-lookup`, `azure-resource-visualizer` | Inventory, discovery, and topology/resource visualization. |
| Ops/FinOps | `azure-ops`, `azure-cost` | CI/CD or cost-control workflows with explicit owner request. |
| Messaging/identity | `azure-messaging`, `entra-agent-id`, `entra-app-registration` | Messaging and identity integration troubleshooting/planning. |
| Migration and modernization | `azure-cloud-migrate`, `azure-hosted-copilot-sdk` | Cross-cloud migration or hosted Copilot SDK-specific modernization. |

### Phase-level internal routing baseline

| Fleet tag | Primary internal agents | Supporting internal agents | Skill bundles | Mandatory persona lenses |
| --- | --- | --- | --- | --- |
| [F0] | `aara-project-planner`, `aara-business-analyst` | `aara-project-reviewer`, `aara-ai-technical-author` | planning/docs quality bundles | BA SME, Product Owner, Delivery Lead |
| [F1] | `aara-python-ai-developer`, `aara-project-builder` | `aara-project-architect`, `aara-project-reviewer`, `aara-project-debugger` | scaffolding/type-safety bundles | Architect, QA/Evaluation, Platform/Tool Owner |
| [F2] | `aara-python-ai-developer`, `aara-business-analyst` | `aara-ai-evaluation-engineer`, `aara-project-reviewer` | synthetic fixture/eval bundles | BA SME, Product Owner, QA/Evaluation |
| [F3] | `aara-python-ai-developer`, `aara-ai-evaluation-engineer` | `aara-project-reviewer`, `aara-project-architect` | control-gate/evaluation bundles | Security/Privacy, Architect, QA/Evaluation |
| [F4] | `aara-project-architect`, `aara-business-analyst` | `aara-python-ai-developer`, `aara-project-reviewer` | tool-validation/data-safety bundles | Security/Privacy, Compliance/Legal, Platform/Tool Owner |
| [F5] | `aara-business-analyst`, `aara-project-builder` | `aara-ai-evaluation-engineer`, `aara-project-reviewer` | MVP expansion/eval bundles | BA SME, Product Owner, QA/Evaluation |
| [F6] | `aara-project-planner`, `aara-executive-narrative-advisor` | `aara-project-reviewer`, `aara-ai-technical-author` | pilot-readiness/control bundles | Delivery Lead, Security/Privacy, Compliance/Legal |
| [F7] | `aara-business-analyst`, `aara-ai-evaluation-engineer` | `aara-project-planner`, `aara-project-reviewer` | prioritization/tool-matrix/data-classification bundles | BA SME, Product Owner, Security/Privacy |
| [F8] | `aara-python-ai-developer`, `aara-business-analyst` | `aara-ai-evaluation-engineer`, `aara-project-architect`, `aara-project-reviewer` | first-slice execution/eval/gate bundles | BA SME, Product Owner, QA/Evaluation, Architect, Security/Privacy |
| [F9] | `aara-project-architect`, `aara-ai-technical-author` | `aara-business-analyst`, `aara-ai-application-architect`, `aara-project-reviewer`, `aara-ai-evaluation-engineer` | HLD architecture/docs/evidence-gate bundles | Architect, BA SME, Product Owner, QA/Evaluation, Security/Privacy, Platform/Tool Owner |

Specialist escalation roster (use when matching trigger conditions): `aara-agent-blueprint-advisor`, `aara-agent-engineer`, `aara-ai-application-architect`, `aara-azure-cost-reviewer`, `aara-business-strategist`, `aara-code-model-designer`, `aara-codebase-extraction-engineer`, `aara-content-strategist`, `aara-copilot-cost-reviewer`, `aara-data-tier-designer`, `aara-leadership-status-deck`, `aara-mcp-server-builder`, `aara-network-topology-reviewer`, `aara-next-bff-developer`, `aara-prompt-engineer`, `aara-senior-microservices-architect`, `aara-status-deck`, `aara-topology-visualizer`.

Command discipline:

- Before Phase 1 creates tooling, do not claim runnable commands exist.
- When a later prompt says "run the project test command," first inspect the repo for the command created by Phase 1, such as `python -m pytest`, `make test`, or another documented command.
- If the documented command is missing, stop and repair the command/documentation mismatch before relying on it.
- Do not skip a QA prompt. The QA prompt is the gate for moving to the next implementation prompt.
- QA prompts recommend readiness; RAJA or the named gate authority makes gate decisions. Do not treat a QA prompt as a human approval.
- Before running any implementation prompt except the first prompt in a phase, confirm all prior implementation+QA pairs in that phase are complete or explicitly marked non-blocking.
- If `rg` is unavailable, use IDE/search equivalent and record the substitution in Test evidence.
- Local `git` status/diff commands are allowed for worktree inspection; live Git provider/API access remains blocked until the relevant gate.

Canonical early deliverable paths:

| Fleet tag | Prompt | Expected deliverable path |
| --- | --- | --- |
| [F0] | [P0A] | `docs/planning/phase-0-baseline-review.md` |
| [F0] | [P0B] | `docs/planning/g0-readiness-package.md` |
| [F0] | [P0C] | `docs/planning/risk-open-question-triage.md` |
| [F1] | [P1T] | `docs/development/phase-1-technical-baseline.md` |
| [F1] | [P1A] | source/tooling paths selected by scaffold plus developer docs |
| [F1] | [P1B] | Python package skeleton under the selected source layout |
| [F1] | [P1C] | local command/test tooling paths selected by scaffold |
| [F1] | [P1D] | `docs/development/g1-readiness.md` |
| [F9] | [P9A] | `docs/planning/phase-2-hld-creation-plan.md` |
| [F9] | [P9B] | `docs/architecture/ba-agent-hld.md` |
| [F9] | [P9C] | `docs/development/phase-2-hld-review-package.md` |

---

## Global execution guardrails

- RAJA is the accountable owner for this planning baseline. Role-specific reviewers may be added later without changing RAJA accountability.
- G0 is clear only for synthetic-only engineering foundation work. It does not approve sandbox integration, live pilot use, production deployment, or Phase 2 capability build.
- Phase 1 through Phase 3 are local/synthetic only. Do not add live Jira, Git, Confluence, Calendar, Teams, Copilot 365, Graph API, model, or MCP connectivity in those phases.
- No live system-of-record reads or writes are allowed until a later gate explicitly authorizes the exact scope. G4 can prepare and validate sandbox read paths only for validated tools. G6 is the first gate that can authorize limited live pilot use.
- MCP integrations remain stubbed or blocked until their actual server name, schema, auth model, scopes, rate limits, and owner approval are validated and recorded.
- Human-gated writes are enforced in the gateway/control layer with approval records, `approval_ref`, idempotency, and audit. Prompt wording alone is not a control.
- Missing, mismatched, expired, replayed, cross-artifact, or otherwise invalid `approval_ref` values fail closed and are audited.
- Teams/Copilot 365 is the target user surface. Do not introduce Slack as an alternate surface.
- If CI/IaC is relevant, stay Azure-primary: GitHub Actions OIDC, Terraform AzureRM, managed identities, Key Vault, and no stored cloud credentials.
- If a container registry is mentioned, use JFrog Artifactory. Never use Azure ACR.
- Do not fabricate numeric quality thresholds. Hard gates are zero approval-gate bypasses and zero MVP/Phase 2 separation violations; owner-set thresholds remain `[RAJA]`.
- MVP scope is standup, planning, retrospective, and sprint health. Phase 2 Enterprise BA capabilities move from readiness to synthetic-first execution only after `P2-G0` acceptance in `docs/planning/phase-2-implementation-plan.md`.
- Every factual output from implemented code should carry evidence refs or be separated as assumptions/open questions with `[inferred]` or `[RAJA]` markers as appropriate.
- Search validations should inspect changed implementation/docs artifacts, not `prompts.md` or `fleet_prompt.md`, unless the prompt text itself is intentionally being edited. Use commands such as `rg -n "Slack|Azure ACR|acr\\.azurecr\\.io" <changed_paths>` and review hits; fail only when a hit enables, approves, stores, or broadens unsafe access.
- Phase-start prompts must stop if prior gate evidence is missing. Later phases may prepare planning artifacts, but must not execute gated behavior before the required gate is accepted by RAJA.

---

## Phase map

| Fleet tag | Phase | Gate | Prompt pairs | Execution focus |
| --- | --- | --- | --- | --- |
| [F0] | Phase 0 — Pre-work and decision closure | G0 | [P0A]/[Q0A], [P0B]/[Q0B], [P0C]/[Q0C] | Baseline review, G0 readiness, risk/open-question triage |
| [F1] | Phase 1 — Engineering foundation | G1 | [P1T]/[Q1T], [P1A]/[Q1A], [P1B]/[Q1B], [P1C]/[Q1C], [P1D]/[Q1D] | Technical baseline, repo layout, Python skeleton, local commands/tooling, docs-only-to-runnable transition |
| [F2] | Phase 2 — First thin slice: synthetic standup | G2 | [P2A]/[Q2A], [P2B]/[Q2B], [P2C]/[Q2C], [P2D]/[Q2D], [P2E]/[Q2E] | Synthetic fixtures, summary generation, Adaptive Card payloads, router/standup graph, local demo |
| [F3] | Phase 3 — Gateway and evaluation hardening | G3 | [P3A]/[Q3A], [P3B]/[Q3B], [P3C]/[Q3C], [P3D]/[Q3D], [P3E]/[Q3E] | Gateway/control layer, write fail-closed, `approval_ref`, audit, `trace_id`, GTS-GATE |
| [F4] | Phase 4 — Sandbox integration readiness | G4 | [P4A]/[Q4A], [P4B]/[Q4B], [P4C]/[Q4C], [P4D]/[Q4D], [P4E]/[Q4E] | Sandbox validation plan, actual MCP schema validation, read-only Jira/Git replacement, Teams approval readiness, blocked unvalidated tools |
| [F5] | Phase 5 — MVP capability expansion | G5 | [P5A]/[Q5A], [P5B]/[Q5B], [P5C]/[Q5C], [P5D]/[Q5D], [P5E]/[Q5E] | Planning recommendation, retro draft-only, health advisory monitoring, expanded golden sets, MVP candidate review |
| [F6] | Phase 6 — MVP pilot readiness and pilot | G6 | [P6A]/[Q6A], [P6B]/[Q6B], [P6C]/[Q6C], [P6D]/[Q6D], [P6E]/[Q6E], [P6F]/[Q6F], [P6G]/[Q6G] | Pilot runbook, RACI/support, release notes, rollback/kill switch, authorization package, explicit live pilot run, post-pilot assessment |
| [F7] | Phase 7 — Phase 2 readiness | G7 | [P7A]/[Q7A], [P7B]/[Q7B], [P7C]/[Q7C], [P7D]/[Q7D], [P7E]/[Q7E] | Phase 2 prioritization, tool matrix, data/classification plan, GTS-P2-REQ approach, separate plan readiness |
| [F8] | Phase 8 — Phase 2 first-slice execution | `P2-G1` through `P2-G5` | [P8A]/[Q8A], [P8B]/[Q8B], [P8C]/[Q8C], [P8D]/[Q8D], [P8E]/[Q8E] | First-slice scaffold, synthetic requirement-discovery execution, evaluation/control hardening, tool/data readiness decisions, candidate review stop |
| [F9] | Phase 9 — HLD creation | `HLD-G0` through `HLD-G3` | [P9A]/[Q9A], [P9B]/[Q9B], [P9C]/[Q9C] | HLD scope-change plan, draft/advisory HLD, HLD owner-review package |

---

## Full prompt entries

## ✅ [P0A] Baseline and decision-log review

Status: ✅ Done  
Deliverable path: `docs/planning/phase-0-baseline-review.md`  
Result: Created Phase 0 baseline review confirming RAJA ownership, G0 synthetic-only scope, DEC-001 through DEC-007 status, and no live access.  
Test evidence: Manual DEC/source cross-check; drift scan for legacy marker usage, forbidden surface/registry terms, live-authorization language, and unsafe broad-scope terms returned no blocking hits.

### Purpose

Review the checked-in planning baseline and decision log before implementation starts, and record any source-alignment gaps without changing scope or claiming approval.

### Execution prompt

You are working in the BA Agent repository. Read `.github/copilot-instructions.md`, `docs/planning/project-development-plan.md`, `docs/planning/decision-log.md`, and the requirements companion docs named in document control.

Produce or update `docs/planning/phase-0-baseline-review.md`. It must:

1. Confirm the current baseline: RAJA accountable owner, synthetic-only G0 build-start, Teams/Copilot 365 surface, LangGraph/MCP design constraints, and no live reads/writes.
2. Compare `docs/planning/decision-log.md` DEC-001 through DEC-007 with the G0 decisions in the development plan.
3. List any missing or divergent decision details as `[RAJA]` items, not as approved facts.
4. Preserve the MVP/Phase 2 separation: do not assume Phase 2 execution authorization without explicit `P2-G0` acceptance evidence.
5. Do not create source code, runtime commands, CI, IaC, or live integration configuration in this prompt.

### Tests

- No executable test command is expected for this docs-only Phase 0 prompt unless a docs lint command already exists.
- Manually cross-check DEC-001 through DEC-007 against the decision log and project plan.
- Manually cross-check that the review cites requirement IDs only where they are actually in the checked-in requirements.

### Validations

- Search changed docs for accidental live-authorization language and remove or mark it `[RAJA]`.
- Run `rg -n "Slack|Azure ACR|acr\\.azurecr\\.io" <changed_paths>` and confirm no forbidden surface or registry drift was introduced.
- Confirm the review does not use any evidence marker other than `[inferred]` or `[RAJA]`.

### Outcomes

- Baseline review is ready for G0 readiness packaging.
- Decision gaps are explicit and owner-routed.
- Immediately run QA prompt [Q0A].

---

## ✅ [Q0A] QA review for baseline and decision-log review

Status: ✅ Done  
Deliverable path: `docs/planning/phase-0-baseline-review.md`  
Result: QA verified source alignment, DEC accuracy, no Phase 1 implementation, and no coordinator gate-waiver authority after remediation.  
Test evidence: `aara-project-reviewer` F0 QA plus recheck; no blockers remaining.

### Purpose

Review [P0A] for source alignment, decision-log accuracy, and scope discipline.

### Execution prompt

Review the artifact created or updated by [P0A]. Make only focused corrections. Confirm it reflects the decision log: G0 clears synthetic-only engineering foundation work, live reads/writes remain blocked, and MCP tools remain stubbed/blocked until validated.

### Tests

- Re-run the manual DEC-001 through DEC-007 comparison.
- Re-check any requirement IDs cited in the artifact against `business-analyst-agent-requirements.md`.

### Validations

- Confirm no Phase 1 implementation work was started by the Phase 0 prompt.
- Confirm no owner-dependent item is presented as closed unless the decision log closes it.
- Confirm no forbidden surface, registry, or live-integration drift appears in changed paths.

### Outcomes

- Recommend [P0A] as ready for [P0B], or record the smallest remediation.
- Return tracking handoff for the coordinator to update [P0A] and [Q0A].

---

## ✅ [P0B] G0 readiness package

Status: ✅ Done  
Deliverable path: `docs/planning/g0-readiness-package.md`  
Result: Created G0 readiness package defining allowed synthetic-only Phase 1 work, blocked live/sandbox/pilot/production/Phase 2 scope, and first build target.  
Test evidence: Manual decision-log and project-plan checks; `rg` overclaim scan returned no blocking hits.

### Purpose

Create a G0 build-start readiness package that states what is authorized, what remains blocked, and what RAJA must explicitly approve or amend.

### Execution prompt

Using the Phase 0 baseline review and the checked-in decision log, create or update `docs/planning/g0-readiness-package.md`.

The package must:

1. State that G0 is clear only for synthetic-only engineering foundation work.
2. Record RAJA as accountable owner for the planning baseline.
3. Re-state the first build target: synthetic Teams standup summary using synthetic Jira/Git fixtures, evidence refs, `trace_id`, and no live writes.
4. List blocked work: sandbox integration, live pilot, production deployment, live system-of-record reads/writes, unvalidated MCP tools, and Phase 2 build.
5. Include a concise "allowed next work" list for Phase 1.
6. Include a "not authorized by G0" list to prevent scope drift.

### Tests

- No new executable command is expected.
- Manually check that every G0 claim is supported by `docs/planning/decision-log.md` lines for the G0 build-start assessment.
- Manually check the package against Phase 1 exit criteria in `docs/planning/project-development-plan.md`.

### Validations

- Confirm the package does not say G0 approves live system-of-record access or sandbox enablement.
- Confirm G0 language does not imply G1/G2/G3 have passed.
- Run `rg -n "production ready|live pilot approved|live reads enabled|live writes enabled" <changed_paths>` and remove or correct overclaims.

### Outcomes

- G0 readiness package is available for RAJA review.
- Phase 1 start boundaries are explicit.
- Immediately run QA prompt [Q0B].

---

## ✅ [Q0B] QA review for G0 readiness package

Status: ✅ Done  
Deliverable path: `docs/planning/g0-readiness-package.md`  
Result: QA verified G0 authority, blocked scope, RAJA accountability, first build target restatement, and no legacy marker usage.  
Test evidence: `aara-project-reviewer` F0 QA plus recheck; `rg` confirmed no forbidden/overclaim markers in F0 deliverables.

### Purpose

Review [P0B] for accurate G0 authority, clear blocked scope, and truthful gate language.

### Execution prompt

Review the G0 package from [P0B]. Preserve useful structure. Fix only language that overclaims approval, omits blockers, or conflicts with the decision log.

### Tests

- Re-check the package against the decision log G0 build-start assessment.
- Re-check the "allowed next work" list against Phase 1 deliverables in the project plan.

### Validations

- Confirm live reads/writes, sandbox enablement, production deployment, and Phase 2 build remain blocked.
- Confirm RAJA accountability is stated without inventing role delegates.
- Confirm evidence markers are limited to `[inferred]` and `[RAJA]`.

### Outcomes

- Recommend G0 package as ready for risk/open-question triage, or record targeted fixes.
- Return tracking handoff for the coordinator to update [P0B] and [Q0B].

---

## ✅ [P0C] Risk and open-question triage

Status: ✅ Done  
Deliverable path: `docs/planning/risk-open-question-triage.md`  
Result: Created risk/open-question triage covering BA-OQ-001 through BA-OQ-015, phase-gate impacts, and later-gate decisions.  
Test evidence: BA-OQ coverage scan found all 15 IDs; risk/gate manual reconciliation completed; unsafe wording scan returned no blocking hits.

### Purpose

Triage open questions, risks, and gate blockers so Phase 1 starts with a clear backlog and later phases do not silently absorb unresolved decisions.

### Execution prompt

Create or update `docs/planning/risk-open-question-triage.md`.

Include:

1. BA-OQ-001 through BA-OQ-015 from the requirements baseline, with phase/gate impact.
2. Planning risks that block or influence G4, G5, G6, and G7.
3. A short "can proceed now" section for synthetic-only Phase 1 work.
4. A "must decide before gate" section for G4 sandbox, G5 MVP candidate, G6 pilot, and G7 Phase 2 readiness.
5. Explicit entries for classification handling, tool-owner validation, pilot boundaries, severity taxonomy, planning publish semantics, Confluence retro behavior, audit retention, and Phase 2 prioritization.
6. No invented dates, names, thresholds, budgets, or approvals; mark owner-dependent fields `[RAJA]`.

### Tests

- No executable test command is expected.
- Manually reconcile all BA-OQ IDs against `business-analyst-agent-requirements.md`.
- Manually reconcile risk IDs against `docs/planning/project-development-plan.md` risk register.

### Validations

- Confirm the triage does not block Phase 1 synthetic-only work on decisions deferred to later gates.
- Confirm no later-phase item is accidentally marked approved.
- Run `rg -n "auto-approved|autonomous|unrestricted|all projects|all repos" <changed_paths>` and correct unsafe wording.

### Outcomes

- Phase-gated decision backlog is ready.
- Phase 1 can start without masking later blockers.
- Immediately run QA prompt [Q0C].

---

## ✅ [Q0C] QA review for risk and open-question triage

Status: ✅ Done  
Deliverable path: `docs/planning/risk-open-question-triage.md`  
Result: QA verified complete BA-OQ coverage, correct later-gate routing, synthetic-only Phase 1 boundary, and no later-phase approval overclaims.  
Test evidence: `aara-project-reviewer` F0 QA recheck returned OK to mark F0 complete.

### Purpose

Review [P0C] for complete open-question coverage and correct phase/gate routing.

### Execution prompt

Review the triage artifact from [P0C]. Make focused fixes for missing BA-OQ IDs, wrong gate placement, or unsafe assumptions.

### Tests

- Re-run the BA-OQ coverage check against the requirements baseline.
- Spot-check at least five risk/open-question entries against their source documents.

### Validations

- Confirm unresolved later-gate items are `[RAJA]`, not treated as closed.
- Confirm Phase 1 remains synthetic/local and not dependent on sandbox or pilot decisions.
- Confirm no forbidden surface, registry, or autonomous-write language was added.

### Outcomes

- Recommend Phase 0 readiness for RAJA/G0 review, or record blockers.
- Return tracking handoff for the coordinator to update [P0C] and [Q0C].

---

## ✅ [P1T] Phase 1 technical baseline

Status: ✅ Done  
Deliverable path: `docs/development/phase-1-technical-baseline.md`  
Result: Created Phase 1 technical baseline covering source layout, command contract, typed/Pydantic boundaries, safe defaults, gateway fake, and no-live guardrails.  
Test evidence: Manual baseline checks against G0/G1/RAJA constraints; `aara-python-ai-developer` QA found no blockers.

### Purpose

Create the technical baseline that constrains Phase 1 implementation lanes before any source scaffold is written.

### Execution prompt

Create `docs/development/phase-1-technical-baseline.md`. Stop if `[Q0C]` is not complete or if the G0 readiness package does not record that synthetic-only Phase 1 work is allowed.

The baseline must define:

1. Required source layout: `src/ba_agent/`, `tests/`, `tests/fixtures/`, and an evaluation/harness path to be created by the scaffold.
2. Python packaging choice and command contract: project manifest, dependency/lockfile approach, test command, typecheck command, no-live/safety check command, and local CLI/help command.
3. Type discipline: typed boundaries and Pydantic models for config, route decisions, graph state, gateway requests/responses, fixture records, eval cases, and Adaptive Card payloads; no untyped cross-module `dict[str, Any]` contracts unless justified as `[inferred]`.
4. Runtime mode defaults: local/synthetic mode by default, live integrations disabled, no required secrets, no network calls in tests.
5. LangGraph-compatible contracts [inferred]: `GraphState`, route decision, graph version stamping, and a placeholder state-transition interface even if LangGraph dependency is deferred.
6. Offline model/tool seams: `ModelClient` protocol/fake, gateway facade/fake, and explicit separation between orchestrator code and gateway/control code.
7. Gateway facade baseline: Phase 1 creates a local contract-test fake only, not the production MCP gateway; every write-like operation fails closed.
8. Minimum test expectations for Phase 1: import/smoke, CLI help, config default, live-mode rejection, no-network guard, typecheck, and placeholder eval/fixture-validation command.
9. Marker and evidence policy: `[inferred]` for unsupported implementation choices and `[RAJA]` for owner-dependent decisions.

Do not create source code in this prompt unless a tiny docs-only helper is required. This prompt is the contract for [P1A] through [P1D].

### Tests

- No executable project tests are expected unless a docs lint command already exists.
- Manually verify the baseline references G0, G1, RAJA accountability, and synthetic-only constraints.
- Manually verify every command named in the baseline is either explicitly future-created by [P1A]/[P1C] or marked `[RAJA]` / `[inferred]`.

### Validations

- Confirm the baseline does not authorize live integrations, model calls, cloud deployment, or system-of-record writes.
- Confirm the baseline is prescriptive enough for multiple implementation agents to avoid divergent layouts.
- Confirm no legacy marker is introduced.
- Confirm any `rg`/search validations in the baseline say to review hits rather than blindly fail on safe guardrail text.

### Outcomes

- Phase 1 has a canonical technical baseline.
- [P1A] can scaffold implementation against a stable contract.
- Immediately run QA prompt [Q1T].

---

## ✅ [Q1T] QA review for Phase 1 technical baseline

Status: ✅ Done  
Deliverable path: `docs/development/phase-1-technical-baseline.md`  
Result: QA verified local/synthetic defaults, no-network expectations, live-mode rejection, Pydantic boundary models, gateway facade separation, and no unauthorized scope.  
Test evidence: `aara-python-ai-developer` QA review returned OK to mark [P1T]/[Q1T] complete.

### Purpose

Review [P1T] for implementation clarity, Python service discipline, gate safety, and fleet suitability.

### Execution prompt

Review `docs/development/phase-1-technical-baseline.md`. Make focused corrections only; do not start source scaffolding.

### Tests

- No executable project tests are expected unless a docs lint command exists.
- Manually check the baseline against `.github/copilot-instructions.md`, `docs/planning/decision-log.md`, and [F1] prompt requirements.

### Validations

- Confirm the baseline mandates local/synthetic defaults, no-network tests, live-mode rejection, typed/Pydantic boundaries, and a clear orchestrator-to-gateway facade.
- Confirm it distinguishes the local gateway fake from the future production MCP gateway.
- Confirm no live integration, production deployment, Phase 2 capability, or legacy marker appears.

### Outcomes

- Recommend [P1T] as ready for [P1A], or record focused remediation.
- Return tracking handoff for the coordinator to update [P1T] and [Q1T].

---

## ✅ [P1A] Repository/source layout and tooling choice

Status: ✅ Done  
Deliverable path: `pyproject.toml`, `Makefile`, `src/ba_agent/`, `tests/`, `eval/README.md`, `docs/development/local-development.md`  
Result: Created Python-first local scaffold with source/test/eval placeholder layout and local-only command wrapper.  
Test evidence: `make check` passed: 15 tests, mypy success, config check, CLI help, synthetic help, and eval help.

### Purpose

Create the initial runnable repository foundation from the current docs-only state. Choose and document the source layout, Python packaging/test tooling, and Phase 1 scope boundaries without adding live integrations.

### Execution prompt

Implement the Phase 1 repository foundation. Stop if [Q1T] is not complete, if `docs/development/phase-1-technical-baseline.md` is missing, or if G0 readiness evidence does not allow synthetic-only Phase 1 work.

Requirements:

1. Follow the technical baseline from [P1T]; do not invent a competing layout or command contract.
2. Create the required Python-first source layout, including `src/ba_agent/`, `tests/`, `tests/fixtures/`, and evaluation/harness placeholder paths unless [P1T] records a justified `[inferred]` deviation.
3. Add a project manifest, packaging/test configuration, and command wrapper only as defined by [P1T].
4. Create or reserve typed/Pydantic boundary modules for config, route decisions, graph state, gateway contracts, fixture records, eval cases, and Adaptive Card payloads.
5. Add a local gateway facade/fake boundary distinct from the future production MCP gateway; the orchestrator must not directly call tool/system adapters.
6. Add no-network/live-mode rejection tests or placeholders exactly as the baseline defines.
7. Add a short developer note explaining that Phase 1 is synthetic-only, has no live Teams connectivity, and does not permit live system-of-record reads/writes.
8. Do not add Terraform, cloud deployment, container publishing, registry configuration, or stored secret assumptions.
9. If CI/IaC is mentioned, align to GitHub Actions OIDC and Terraform AzureRM only; if registry is mentioned, use JFrog Artifactory only.

### Tests

- After creating tooling, run the initial test command you created, such as `python -m pytest` or `make test`.
- Add at least one smoke test proving the package imports or a placeholder CLI responds locally without secrets or network calls.
- If a synthetic command placeholder is created, test controlled help/failure behavior.

### Validations

- Confirm no live endpoint, token, webhook URL, credential, cloud deployment, or registry config is introduced.
- Confirm docs say the repo was docs-only before this scaffold and that commands were created by this work.
- Run `rg -n "Slack|Azure ACR|acr\\.azurecr\\.io" <changed_paths>` and remove drift.
- Confirm the layout supports orchestrator/router, gateway/control, synthetic fixtures, evaluation harness, and tests.
- Confirm implementation matches `docs/development/phase-1-technical-baseline.md`.

### Outcomes

- A runnable local skeleton exists.
- The chosen tooling and local commands are documented.
- Phase 1 boundaries are explicit.
- Immediately run QA prompt [Q1A].

---

## ✅ [Q1A] QA review for repository/source layout and tooling

Status: ✅ Done  
Deliverable path: `pyproject.toml`, `Makefile`, `src/ba_agent/`, `tests/`, `eval/README.md`, `docs/development/local-development.md`  
Result: QA verified scaffold layout, command truthfulness after fix, no live clients/secrets, and no Phase 2 implementation.  
Test evidence: `aara-project-reviewer` recheck returned no blockers; `make check` passed.

### Purpose

Review [P1A] for correctness, minimality, scope control, and future compatibility with the synthetic standup thin slice.

### Execution prompt

Review the repository changes made by [P1A]. Do not rewrite the scaffold unless it is structurally incoherent. Apply the smallest fix needed.

### Tests

- Re-run the test command created in [P1A].
- Re-run the smoke/import/CLI help test created in [P1A].
- If a command wrapper was created, run it and confirm it delegates as documented.

### Validations

- Inspect changed files for no secrets, no live endpoints, no Slack, no Azure ACR, and no autonomous-write language.
- Confirm documentation does not imply G1 is passed unless the checklist is actually satisfied.
- Confirm any CI/IaC mention follows GitHub Actions OIDC / Terraform AzureRM constraints and avoids stored cloud credentials.

### Outcomes

- Recommend [P1A] as ready for [P1B], or record focused remediation.
- Return tracking handoff for the coordinator to update [P1A] and [Q1A].

---

## ✅ [P1B] Python package skeleton and safe defaults

Status: ✅ Done  
Deliverable path: `src/ba_agent/__init__.py`, `src/ba_agent/__main__.py`, `src/ba_agent/cli.py`, `src/ba_agent/config.py`, `src/ba_agent/models.py`, `src/ba_agent/gateway.py`, `src/ba_agent/orchestrator.py`, `src/ba_agent/py.typed`  
Result: Added typed/Pydantic package skeleton with local config defaults, CLI placeholders, gateway facade fake, and offline orchestrator/model seams.  
Test evidence: `make check` passed; package import, CLI, config, model, gateway, and no-network tests passed.

### Purpose

Create the initial Python package skeleton for the BA Agent orchestrator and control components [inferred], without live model calls, live MCP calls, or live Teams posting.

### Execution prompt

Build on the technical baseline from [P1T] and tooling created by [P1A].

Implement:

1. Package modules for orchestrator entry points, router/graph placeholders, gateway/control placeholders, config/settings, shared models/errors, and test utilities.
2. Defaults that force local/synthetic mode, such as `BA_AGENT_ENV=local` and `LIVE_INTEGRATIONS_ENABLED=false` or equivalent.
3. A CLI or module entry point that can run locally without network access. It may support `--help`, no-op health, or a safe synthetic placeholder.
4. A configuration pattern that refuses live mode unless a future validated setting and gate explicitly enable it.
5. Visible control-layer boundaries: the orchestrator must not directly mutate systems of record.
6. Minimal dependencies. If LangGraph is not added yet, expose a compatible interface [inferred] and document the deferral.

### Tests

- Add unit tests for package import, settings defaults, CLI help/no-op behavior, and live-mode rejection.
- Run the project test command created by [P1A].
- If a module entry point exists, run it locally, for example `python -m ba_agent --help` only if that entry point was created.

### Validations

- Confirm settings default to local/synthetic mode.
- Confirm no code imports live Jira, Git, Confluence, Calendar, Teams, Graph API, model, or cloud SDK clients for Phase 1 behavior.
- Confirm no secrets are required to run tests.
- Confirm orchestrator and gateway/control concepts are separated.

### Outcomes

- A Python package skeleton exists and can be imported/tested locally.
- Live integrations remain disabled by construction.
- The control-layer boundary is visible for later gateway enforcement.
- Immediately run QA prompt [Q1B].

---

## ✅ [Q1B] QA review for Python skeleton

Status: ✅ Done  
Deliverable path: `src/ba_agent/`, `tests/`  
Result: QA verified safe defaults, orchestrator/gateway separation, local gateway fake, Pydantic boundaries, and no live client dependencies.  
Test evidence: `aara-project-reviewer` recheck and `security-review` found no blockers; `make check` passed.

### Purpose

Review [P1B] for runnable correctness, safe defaults, and clean separation between orchestrator and control-layer responsibilities.

### Execution prompt

Review the Python skeleton created by [P1B]. Preserve the structure if it is coherent; make targeted fixes only.

### Tests

- Re-run all tests from [P1B].
- Run the local CLI/module command created in [P1B].
- Add one regression test if the review finds a missing safety assertion.

### Validations

- Inspect dependencies for premature cloud/live system clients.
- Inspect code paths for network calls, live URLs, token reads, or hidden writes.
- Confirm no Phase 2 requirement/story/process-map generation code was introduced.

### Outcomes

- Recommend the Python skeleton as ready for local command work, or record targeted fixes.
- Return tracking handoff for the coordinator to update [P1B] and [Q1B].

---

## ✅ [P1C] Local command and test tooling foundation

Status: ✅ Done  
Deliverable path: `Makefile`, `tests/test_cli.py`, `tests/test_config.py`, `tests/test_gateway.py`, `tests/test_imports.py`, `tests/test_makefile_commands.py`, `tests/test_models.py`, `tests/test_no_network.py`, `tests/conftest.py`  
Result: Added local test/typecheck/check commands, no-network test guard, Makefile command tests, live-mode rejection tests, and placeholder synthetic/eval command tests.  
Test evidence: `make check` passed: 15 tests, mypy success, no-live config, CLI help, synthetic help, and eval help.

### Purpose

Create local command patterns and test scaffolding that later prompts can rely on, while keeping behavior local/synthetic and not yet implementing the standup thin slice.

### Execution prompt

Add or refine local developer commands.

Requirements:

1. Document the project test command and ensure it runs from the repository root.
2. Add a safe local command namespace for future synthetic runs, eval runs, and config checks. Placeholder commands may return help or explicit "not implemented yet" messages.
3. Add tests for command discovery, help output, controlled failure, and no-network behavior.
4. Ensure commands do not require live credentials, tenant IDs, tokens, cloud subscriptions, or MCP servers.
5. Update developer docs with only commands that actually exist.
6. Leave standup fixture loading and Adaptive Card behavior for Phase 2 prompts unless already needed as a placeholder.

### Tests

- Run the documented project test command.
- Run each local command/help path created by this prompt.
- Add negative tests for unknown commands, unsafe live mode flags, and missing required local inputs.

### Validations

- Confirm command docs match actual command names and options.
- Confirm no command makes network calls or requires secrets.
- Run `rg -n "LIVE_INTEGRATIONS_ENABLED.*true|prod|production|tenant|token|secret" <changed_paths>` and confirm hits are blocked/test-only or corrected.

### Outcomes

- Later prompts have truthful local commands to build on.
- Test tooling is runnable and documented.
- Immediately run QA prompt [Q1C].

---

## ✅ [Q1C] QA review for local command and test tooling

Status: ✅ Done  
Deliverable path: `Makefile`, `tests/`, `docs/development/local-development.md`, `docs/development/g1-readiness.md`  
Result: QA verified command docs match actual Makefile/PYTHONPATH behavior and no-network/live-mode safeguards are tested.  
Test evidence: Command-contract mismatch was fixed; `aara-project-reviewer` recheck returned no blockers; `make check` passed.

### Purpose

Review [P1C] for command truthfulness, no-network behavior, and reliable test execution.

### Execution prompt

Review command and tooling changes from [P1C]. Make focused fixes for broken commands, misleading docs, or unsafe defaults.

### Tests

- Re-run the documented test command.
- Re-run each CLI/help path created or changed.
- Re-run negative tests for unknown commands and unsafe live-mode flags.

### Validations

- Confirm command docs use only commands that exist.
- Confirm no hidden cloud, model, Graph API, or MCP calls occur.
- Confirm command failures are clear enough for later prompts to test.

### Outcomes

- Recommend local tooling as ready for docs-only-to-runnable review, or record remediation.
- Return tracking handoff for the coordinator to update [P1C] and [Q1C].

---

## ✅ [P1D] Docs-only-to-runnable transition and G1 readiness

Status: ✅ Done  
Deliverable path: `docs/development/local-development.md`, `docs/development/g1-readiness.md`  
Result: Documented local development commands and G1 readiness evidence without claiming Phase 2, sandbox, pilot, production, or live integration readiness.  
Test evidence: `make check` passed; drift scans found no blocking live-client/surface/registry issues in implementation paths.

### Purpose

Document the new runnable foundation and prepare a G1 Skeleton Gate readiness review without claiming thin-slice, sandbox, pilot, or production readiness.

### Execution prompt

Update developer-facing documentation after [P1T] through [P1C], including `docs/development/g1-readiness.md`.

Include:

1. Repository/source layout.
2. Tooling choice and why it is appropriate for Phase 1.
3. Local setup and local test command.
4. Safe local command namespace and placeholders.
5. Synthetic-only guardrails and no-secrets expectations.
6. G1 readiness checklist: source tree exists, local/dev config exists, unit command exists, no secrets in code, local commands are documented, live integrations disabled.
7. "What this does not do": no live Teams posting, no live Jira/Git/Confluence/Calendar reads, no live writes, no Phase 2 implementation, no production deployment.

### Tests

- Run the project test command.
- Run local command/help paths documented by Phase 1.
- If docs linting was added by Phase 1, run it; otherwise do not claim docs lint exists.

### Validations

- Cross-check docs against actual commands and paths.
- Confirm G1 readiness language is a review package, not an approval.
- Run `rg -n "Slack|Azure ACR|acr\\.azurecr\\.io|production ready|live integration enabled" <changed_paths>` and correct drift.
- Confirm no owner-set metric threshold is invented.

### Outcomes

- Documentation tells a future developer how to run and test the Phase 1 foundation.
- G1 readiness evidence is recorded without overstating approval.
- Immediately run QA prompt [Q1D].

---

## ✅ [Q1D] QA review for G1 readiness

Status: ✅ Done  
Deliverable path: `docs/development/g1-readiness.md`  
Result: Final QA verified G1 readiness evidence, command truthfulness, no-live boundaries, and no Phase 2/sandbox/pilot/production overclaim.  
Test evidence: `aara-project-reviewer` recheck returned OK to mark P1A-D/Q1A-D complete; `security-review` found no blockers; `make check` passed.

### Purpose

Perform the final Phase 1 QA pass for documentation accuracy, command truthfulness, and G1 gate readiness.

### Execution prompt

Review all documentation and readiness notes from [P1D]. Make only focused corrections.

### Tests

- Re-run the project test command.
- Re-run documented local command/help paths.
- If any command fails, fix the implementation/docs or mark the related prompt partial/blocked with a reason.

### Validations

- Cross-check paths in docs against the file system.
- Inspect readiness checklist against G1 exit criteria from the project plan.
- Confirm no live integration, production deployment, sandbox approval, or Phase 2 capability is claimed.

### Outcomes

- Phase 1 execution is ready for RAJA/G1 review, or blockers are explicitly recorded.
- Return tracking handoff for the coordinator to update [P1D] and [Q1D].

---

## ✅ [P2A] Synthetic standup fixture schema and loader

Status: ✅ Done  
Deliverable path: `src/ba_agent/models.py`, `src/ba_agent/fixtures.py`, `tests/fixtures/standup_cases.json`, `tests/test_fixtures.py`  
Result: Added versioned synthetic standup fixture schema/loader with content checksum, synthetic evidence validation, and normal/degraded/denied/throttled/empty/router seed cases.  
Test evidence: `make check` passed with 33 tests; fixture validation rejects non-synthetic refs and content/checksum tampering.

### Purpose

Create deterministic synthetic fixture schemas and a loader for the standup thin slice. Fixtures represent Jira/Git evidence, tool statuses, and evaluation metadata without using live or restricted data.

### Execution prompt

Implement synthetic fixture support for Phase 2. Stop if [Q1D] is not complete or if the Phase 1 G1 readiness evidence is missing.

Requirements:

1. Define fixture models for Jira sprint status, blockers/risks, Git activity, tool status, and evaluation case metadata.
2. Include `source_timestamp` and `retrieved_at` where tool-like fixture responses are modeled.
3. Require synthetic evidence refs such as `jira:synthetic:<project>/<issue-key>`, `git:synthetic:<repo>/<commit-or-pr>`, `tool:synthetic:<tool>/<case>`, and `eval:<case>`.
4. Reject real-looking or ambiguous source identifiers.
5. Seed normal, degraded Git, denied scope, throttled, empty sprint, and prompt-injection-in-data cases.
6. Add a fixture manifest with fixture-set version, case IDs, source file list, and deterministic hash/checksum [inferred] for reproducible eval runs.
7. Document that fixtures are synthetic placeholders and not evidence of completed project work.

### Tests

- Run the project test command after confirming the command exists.
- Add loader tests for valid fixtures.
- Add negative tests for missing evidence refs, non-synthetic identifiers, missing timestamps, unsupported tool statuses, and degraded Git data with absent commit activity.
- If a fixture CLI exists, run it against seeded fixtures.

### Validations

- Confirm no fixture contains real team names, project names, repository names, channel names, user names, or source data.
- Confirm every factual fixture item has an evidence ref.
- Confirm degraded/denied/throttled cases are explicit and not silently normalized.
- Run `rg -n "prod|production|tenant|token|secret|password" <fixture_paths>` and resolve unsafe hits.

### Outcomes

- Synthetic fixtures are loadable, validated, and reusable by standup, card, router, and eval prompts.
- Fixture seeds cover GTS-STANDUP and GTS-ROUTER edge cases.
- Immediately run QA prompt [Q2A].

---

## ✅ [Q2A] QA review for synthetic standup fixtures

Status: ✅ Done  
Deliverable path: `tests/fixtures/standup_cases.json`, `src/ba_agent/fixtures.py`, `tests/test_fixtures.py`  
Result: QA verified fixture safety, version/hash determinism, no real repo name, expanded router coverage, and synthetic-only evidence refs.  
Test evidence: `aara-project-reviewer` F2 recheck returned OK; fixture drift scan returned no blocking hits.

### Purpose

Review [P2A] for synthetic-only safety, schema validity, determinism, and evidence discipline.

### Execution prompt

Review the fixture schema, loader, and seeded fixtures. Make the smallest corrections needed.

### Tests

- Re-run all loader/schema tests.
- Add or adjust one negative test if a validation rule is only documented but not enforced.
- Run the fixture CLI path if one exists.

### Validations

- Inspect fixture files for real identifiers or accidental restricted/internal content.
- Confirm `source_timestamp` and `retrieved_at` are distinct where applicable.
- Confirm evidence refs use approved synthetic prefixes.

### Outcomes

- Recommend fixture foundation as ready for standup generation, or record targeted remediation.
- Return tracking handoff for the coordinator to update [P2A] and [Q2A].

---

## ✅ [P2B] Standup summary generation

Status: ✅ Done  
Deliverable path: `src/ba_agent/standup.py`, `tests/test_standup.py`  
Result: Added deterministic evidence-linked standup summaries with status snapshots, risks, degraded data honesty, open questions, and trace/fixture metadata.  
Test evidence: `make check` passed; standup tests cover normal, degraded Git, and empty sprint cases.

### Purpose

Generate source-linked standup summaries from synthetic Jira/Git fixtures, including degraded data honesty and no unsupported claims.

### Execution prompt

Implement deterministic local standup summary generation.

Requirements:

1. Consume only synthetic fixtures from [P2A].
2. Produce status snapshot, completed/in-progress/blocked items, risks, data-quality status, assumptions, open questions, evidence refs, and `trace_id`.
3. Surface blockers and risks only when fixture fields support them.
4. If Git data is degraded or absent, state that honestly and do not invent commit or PR activity.
5. Separate unsupported or owner-dependent statements as `[inferred]` or `[RAJA]`.
6. Do not call a live model, live MCP server, or live system of record.

### Tests

- Run the project test command after confirming it exists.
- Add summary tests for normal, degraded Git, denied scope, throttled, empty sprint, and stalled/flagged story cases.
- Add tests that every factual summary item includes or links to evidence refs.
- Add tests that degraded data does not produce fabricated activity.

### Validations

- Confirm summary output includes case ID, fixture version if available, `trace_id`, evidence refs, and data-quality status.
- Confirm no generated summary writes to files unless explicitly requested by a local command.
- Run `rg -n "requests\\.|httpx|GraphServiceClient|JIRA|Confluence|Calendar" <changed_source_paths>` and confirm no live clients were added.

### Outcomes

- Synthetic standup summary generation is deterministic and evidence-linked.
- Summary output is ready for Adaptive Card payload building.
- Immediately run QA prompt [Q2B].

---

## ✅ [Q2B] QA review for standup summary generation

Status: ✅ Done  
Deliverable path: `src/ba_agent/standup.py`, `tests/test_standup.py`  
Result: QA verified summary evidence coverage, degraded Git honesty, and no live model/MCP/system-of-record path.  
Test evidence: `aara-project-reviewer` F2 recheck returned OK; security review found no blockers.

### Purpose

Review [P2B] for summary correctness, evidence coverage, degraded-mode honesty, and no-live behavior.

### Execution prompt

Review standup generation. Keep the API stable unless a targeted safety or evidence fix is needed.

### Tests

- Re-run all standup summary tests.
- Add one regression test for any unsupported-claim or degraded-mode issue found.

### Validations

- Inspect sample outputs for facts/assumptions/open questions separation.
- Confirm no live model/MCP/system-of-record path exists.
- Confirm Phase 2 enterprise BA artifacts are not generated.

### Outcomes

- Recommend summary generation as ready for card payload work, or record targeted fixes.
- Return tracking handoff for the coordinator to update [P2B] and [Q2B].

---

## ✅ [P2C] Adaptive Card payload builder

Status: ✅ Done  
Deliverable path: `src/ba_agent/cards.py`, `tests/test_cards.py`  
Result: Added local Adaptive Card JSON builder with per-item evidence text, assumptions/open-questions sections, route metadata footer, and fail-closed send stub.  
Test evidence: `make check` passed; card tests verify required sections, route metadata, evidence refs, and send stub failure.

### Purpose

Build a Teams Adaptive Card payload generator for synthetic standup output without posting to Teams or requiring live Copilot 365 connectivity.

### Execution prompt

Implement an Adaptive Card payload builder.

The builder must:

1. Accept local standup summary output from [P2B].
2. Produce Teams/Copilot 365 Adaptive Card-compatible JSON.
3. Include title, status snapshot, completed/in-progress/blocked items, risks, data-quality section, assumptions/open questions, evidence refs, and footer with `trace_id`, graph/fixture version, and case ID.
4. Preserve evidence refs for every factual item.
5. Validate payload shape locally.
6. Do not implement live `send_adaptive_card`, Graph API, Bot Framework posting, or channel access. Any send interface must be a stub that fails closed.

### Tests

- Run the project test command after confirming it exists.
- Add card tests for normal, degraded Git, denied scope, throttled, and empty sprint cases.
- Add schema/shape tests for required fields and footer values.
- Add tests proving send/post paths are absent or fail closed.

### Validations

- Confirm card payloads contain no real Teams channel IDs or tenant-specific values.
- Confirm every factual card item carries or links to evidence refs.
- Confirm card copy uses Teams/Copilot 365 language and not Slack.
- Run `rg -n "send_adaptive_card|Graph API|Bot Framework|channel_id|tenant" <changed_source_paths>` and confirm hits are stubs/tests or corrected.

### Outcomes

- Adaptive Card JSON can be generated and validated locally from synthetic fixtures.
- No live Teams posting is implemented or implied.
- Immediately run QA prompt [Q2C].

---

## ✅ [Q2C] QA review for Adaptive Card payload builder

Status: ✅ Done  
Deliverable path: `src/ba_agent/cards.py`, `tests/test_cards.py`  
Result: QA verified card evidence preservation, assumptions section, route metadata, and no live Teams posting.  
Test evidence: `aara-project-reviewer` F2 recheck returned OK; security review found no blockers.

### Purpose

Review [P2C] for payload correctness, evidence discipline, no-posting enforcement, and Teams scope alignment.

### Execution prompt

Review the Adaptive Card builder. Preserve the builder API unless a targeted safety or schema fix is needed.

### Tests

- Re-run all card builder tests.
- Run one local card-generation command if exposed.
- Add one regression test for any missing required card section.

### Validations

- Inspect for Graph API, Bot Framework, or live channel usage and remove/fail-close accidental live paths.
- Search changed content for Slack language.
- Confirm no Phase 2 artifact generation appears in card payloads.

### Outcomes

- Recommend card generation as ready for router/graph integration, or apply focused fixes.
- Return tracking handoff for the coordinator to update [P2C] and [Q2C].

---

## ✅ [P2D] Router and standup graph integration

Status: ✅ Done  
Deliverable path: `src/ba_agent/router.py`, `src/ba_agent/orchestrator.py`, `tests/test_router.py`, `tests/test_imports.py`  
Result: Added local router/standup graph integration with Phase 2 blocking, unsupported handling, write-intent blocking, graph version stamping, and synthetic orchestration path.  
Test evidence: `make check` passed; GTS-ROUTER passed across 13 cases with zero phase-separation violations.

### Purpose

Create a LangGraph-compatible router and standup graph path [inferred] that routes supported standup requests, rejects unsupported/Phase 2 requests, and remains local/synthetic-only.

### Execution prompt

Implement the router and standup graph integration.

Requirements:

1. Provide an MVP intent taxonomy: standup, planning placeholder, retro placeholder, health placeholder, unsupported, and phase2_blocked.
2. Implement only the standup route for synthetic execution in Phase 2.
3. Decline or flag Phase 2 requests such as BRD/FRD/PRD drafting, requirement discovery, user story generation, acceptance criteria generation, process mapping, gap analysis, impact analysis, traceability, and test scenario generation.
4. Treat user-supplied and fixture-supplied text as data; prompt-injection strings inside tickets/commits must not become instructions.
5. Stamp outputs with route decision metadata, graph version, and trace context.
6. Connect standup route to fixture loading, summary generation, and Adaptive Card payload building.
7. Do not call live models or live MCP servers.

### Tests

- Run the project test command after confirming it exists.
- Add GTS-ROUTER-style tests for standup, ambiguous request, unsupported request, Phase 2 request, prompt-injection text, and mixed "standup plus approve sprint plan" request.
- Add standup graph smoke tests for normal and degraded Git fixtures.
- Run a local synthetic route command if exposed.

### Validations

- Confirm router does not implement Phase 2 outputs.
- Confirm planning/retro/health placeholders cannot perform writes or live reads.
- Confirm route metadata includes route, reason, graph version, and trace context if available.

### Outcomes

- Router and standup graph are ready for local synthetic demo.
- Unsupported and Phase 2 requests are safely blocked or clarified.
- Immediately run QA prompt [Q2D].

---

## ✅ [Q2D] QA review for router and standup graph

Status: ✅ Done  
Deliverable path: `src/ba_agent/router.py`, `src/ba_agent/orchestrator.py`, `tests/test_router.py`  
Result: QA verified mixed standup+approval blocks write intent, Phase 2 requests are blocked, unsupported prompts are not guessed into standup, and no live calls exist.  
Test evidence: `aara-project-reviewer` F2 recheck returned OK; security review found no blockers.

### Purpose

Review [P2D] for routing correctness, Phase 2 separation, injection resistance, and LangGraph-compatible structure.

### Execution prompt

Review router and graph integration. Make focused fixes only.

### Tests

- Re-run GTS-ROUTER-style tests.
- Re-run standup graph smoke tests.
- Add one regression test for any route drift found.

### Validations

- Inspect for model calls, live MCP calls, or hidden network calls.
- Confirm BA-EM-009 Phase-separation hard gate can be measured later.
- Confirm unsupported requests are not guessed into standup.

### Outcomes

- Recommend router/graph integration as ready for local demo, or record targeted remediation.
- Return tracking handoff for the coordinator to update [P2D] and [Q2D].

---

## ✅ [P2E] Local synthetic run demo and G2 readiness

Status: ✅ Done  
Deliverable path: `src/ba_agent/cli.py`, `src/ba_agent/evaluation.py`, `Makefile`, `eval/README.md`, `docs/development/local-development.md`, `docs/development/g2-readiness.md`, `tests/test_cli.py`, `tests/test_evaluation.py`  
Result: Added local synthetic demo command, GTS-STANDUP/GTS-ROUTER seed eval commands, G2 readiness evidence, and local docs updates.  
Test evidence: `make check` passed; normal/degraded demos ran; GTS-STANDUP passed across 7 cases and GTS-ROUTER passed across 13 cases.

### Purpose

Provide a local synthetic end-to-end demo for the standup thin slice and package G2 Thin-Slice Demo Gate evidence.

### Execution prompt

Implement or finalize a local synthetic demo command using [P2A] through [P2D].

The command should:

1. Accept a synthetic case ID such as `STD-001` or the equivalent from seeded fixtures.
2. Load synthetic Jira/Git/tool/eval fixtures.
3. Generate or propagate `trace_id`.
4. Route to the standup graph.
5. Produce structured summary output and optionally Adaptive Card JSON.
6. Return explicit degraded/denied/throttled data-quality sections.
7. Never post to Teams or call live Jira/Git/Confluence/Calendar/MCP/model services.
8. Produce G2 readiness notes showing STD-style cases pass, evidence refs are present, degraded Git is honest, and no write tool is invoked.
9. Include minimal executable GTS-STANDUP and GTS-ROUTER seed eval runs, or explicitly block G2 readiness until they exist.

### Tests

- Run the project test command after confirming it exists.
- Add command tests for normal, degraded Git, denied scope, empty sprint, prompt-injection-in-data, and unknown case ID.
- Run at least one manual local command for a normal case and one degraded Git case.
- Run GTS-STANDUP and GTS-ROUTER seed evaluations; if unavailable, record G2 as blocked rather than complete.

### Validations

- Confirm output includes `trace_id`, case ID, route metadata, fixture version if available, evidence refs, and data-quality status.
- Confirm no write tool is invoked.
- Run `rg -n "send_adaptive_card|update_sprint_scope|publish_page|live|production" <changed_source_paths>` and confirm hits are absent, blocked, or test-only.

### Outcomes

- Local synthetic standup demo is repeatable from a clean checkout.
- G2 readiness evidence is recorded without claiming sandbox or pilot readiness.
- Immediately run QA prompt [Q2E].

---

## ✅ [Q2E] QA review for local synthetic demo and G2 readiness

Status: ✅ Done  
Deliverable path: `docs/development/g2-readiness.md`, `src/ba_agent/`, `tests/`  
Result: QA verified local demo repeatability, seed evals, route metadata, no-write G2 boundary, and no sandbox/pilot/production overclaim.  
Test evidence: `aara-project-reviewer` F2 recheck returned OK; `security-review` found no blockers; targeted demo/eval checks passed.

### Purpose

Review [P2E] for end-to-end local behavior, deterministic demo evidence, and strict no-live-integration enforcement.

### Execution prompt

Review the local synthetic demo and G2 notes. Keep the command interface stable unless it is unsafe or unusable.

### Tests

- Re-run the command tests from [P2E].
- Run normal and degraded Git manual demo commands.
- Run minimal GTS-STANDUP and GTS-ROUTER seed evals; if unavailable, mark G2 readiness blocked.
- Add a deterministic `trace_id` test hook if nondeterminism blocks reliable tests.

### Validations

- Inspect command path for network imports/calls.
- Confirm outputs cite synthetic evidence only.
- Confirm G2 claims "no write tool invoked" and does not claim audited write-rejection proof; audited rejection proof belongs to G3.
- Confirm G2 readiness does not imply G3/G4/G5/G6 approval.

### Outcomes

- Recommend Phase 2 thin-slice demo readiness for RAJA/G2 review and control hardening, or record targeted fixes.
- Return tracking handoff for the coordinator to update [P2E] and [Q2E].

---

## ✅ [P3A] MCP gateway/control stub and allowlists

Status: ✅ Done  
Deliverable path: `src/ba_agent/gateway.py`, `tests/test_gateway.py`  
Result: Hardened local gateway/control fake with capability allowlists, blocked unvalidated tools, explicit statuses, and audit emission.  
Test evidence: `make check` passed with 40 tests; gateway allowlist/status tests passed.

### Purpose

Implement a local MCP gateway/control-layer stub that enforces capability allowlists and blocked/unvalidated tools outside the model/orchestrator loop.

### Execution prompt

Implement a local gateway/control stub. Stop if [Q2E] is not complete or if Phase 2 G2 thin-slice evidence is missing. This is not a live MCP server and must not call real systems.

Requirements:

1. Define tool categories: synthetic read stubs, blocked/unvalidated live read tools, and blocked write-like tools.
2. Enforce per-capability allowlists. The standup graph may call only synthetic Jira/Git read stubs needed for standup.
3. Represent statuses: `ok`, `degraded`, `denied`, `throttled`, `rejected`, and `blocked` or documented equivalents.
4. Return explicit denied/degraded/throttled responses rather than silently omitting data.
5. Keep enforcement in code; do not rely on prompt wording as the only protection.
6. Keep all Phase 3 behavior local/synthetic.

### Tests

- Run the project test command after confirming it exists.
- Add tests that standup can call allowed synthetic reads.
- Add tests that standup cannot call planning, retro, health, Teams send, Confluence publish, calendar write, Git write, or Jira write tools.
- Add denied/degraded/throttled status tests.

### Validations

- Confirm gateway/control code is separate from orchestrator decisions.
- Confirm no live MCP client/server or source-system client is called.
- Run `rg -n "requests\\.|httpx|GraphServiceClient|atlassian|jira|confluence|calendar" <changed_source_paths>` and confirm hits are local models/stubs/tests only.

### Outcomes

- Gateway/control allowlist behavior is testable locally.
- Unvalidated tools are blocked by code.
- Immediately run QA prompt [Q3A].

---

## ✅ [Q3A] QA review for gateway/control stub

Status: ✅ Done  
Deliverable path: `src/ba_agent/gateway.py`, `tests/test_gateway.py`  
Result: QA verified gateway boundary integrity, allowlist enforcement, local-only behavior, and blocked live/unvalidated tools.  
Test evidence: `aara-project-reviewer` F3 QA returned OK; security review found no blockers.

### Purpose

Review [P3A] for gateway boundary integrity, allowlist enforcement, and no-live behavior.

### Execution prompt

Review the gateway/control stub. Make only targeted fixes to enforce the boundary.

### Tests

- Re-run gateway allowlist and status tests.
- Add one adversarial test if an unvalidated tool can be reached.

### Validations

- Inspect for direct orchestrator calls to blocked tool functions.
- Confirm all live tools remain stubbed/blocked.
- Confirm local tests do not accidentally grant live capability.

### Outcomes

- Recommend gateway/control foundation as ready for write-fail-closed semantics, or record targeted remediation.
- Return tracking handoff for the coordinator to update [P3A] and [Q3A].

---

## ✅ [P3B] approval_ref and idempotency semantics

Status: ✅ Done  
Deliverable path: `src/ba_agent/gateway.py`, `src/ba_agent/models.py`, `tests/test_gateway.py`  
Result: Added local approval records, single-use approval_ref validation, idempotency duplicate rejection, write-like fail-closed behavior, and machine-readable rejections.  
Test evidence: `make check` passed; tests cover missing approval, wrong artifact, wrong action, replay, duplicate idempotency, audit failure, and valid-looking ref with live writes disabled.

### Purpose

Implement local `approval_ref` and idempotency semantics so write-like actions fail closed and can prove BA-EM-005 remains zero.

### Execution prompt

Extend the gateway/control layer.

Requirements:

1. Model approval records locally with artifact/action/actor scope, status, expiry or validity window [RAJA], and single-use behavior.
2. Require `idempotency_key` for every write-like action; require `approval_ref` for external state changes or user-visible sends except `request_approval`, which may only create a pending request and must never issue an `approval_ref`.
3. Reject and audit missing, mismatched, expired, replayed, cross-artifact, or cross-action approval refs.
4. For Phase 3, even valid-looking approval refs must not enable live writes; they may only authorize local test objects if needed.
5. Treat `subscribe_sprint_events`, `draft_page`, `update_sprint_scope`, `publish_page`, `send_adaptive_card`, `send_escalation`, calendar mutation, Git mutation, and approval-record creation as blocked/write-like paths unless explicitly local/test-only.
6. Model `record_human_approval` as non-agent-callable; repository text or agent-authored approval notes must never satisfy approval evidence.
7. Make rejection machine-readable for the evaluation harness.

### Tests

- Run the project test command after confirming it exists.
- Add GTS-GATE-style tests: no approval ref, wrong artifact, wrong action, replayed ref, duplicate idempotency key, instruction-in-data attempting to trigger a write, and standup graph attempting a write.
- Add tests that BA-EM-005 count remains zero when all write attempts are rejected.

### Validations

- Confirm write fail-closed behavior is enforced in code, not just docs.
- Confirm no test object is confused with live authorization.
- Run `rg -n "subscribe_sprint_events|draft_page|update_sprint_scope|publish_page|send_adaptive_card|send_escalation|request_approval|record_human_approval|approval_ref" <changed_source_paths>` and confirm every write-like hit is gated, blocked, non-agent-callable, or test-only.

### Outcomes

- `approval_ref` and idempotency behavior are locally testable.
- Unauthorized writes fail closed and are measurable by GTS-GATE.
- Immediately run QA prompt [Q3B].

---

## ✅ [Q3B] QA review for approval_ref and idempotency

Status: ✅ Done  
Deliverable path: `src/ba_agent/gateway.py`, `tests/test_gateway.py`  
Result: QA verified fail-closed approval/idempotency semantics, no self-approval path, auditable rejections, and no prompt-only control.  
Test evidence: `aara-project-reviewer` F3 QA returned OK; security review found no blockers.

### Purpose

Review [P3B] for fail-closed enforcement, approval semantics, idempotency handling, and GTS-GATE readiness.

### Execution prompt

Review approval and idempotency implementation. Make the smallest corrections needed to enforce the hard gate.

### Tests

- Re-run all GTS-GATE-style tests from [P3B].
- Add one adversarial regression test if any bypass path is discovered.
- Run the full project test command.

### Validations

- Inspect for write-like functions reachable without approval validation.
- Confirm rejection results are auditable and machine-readable.
- Confirm no prompt-only policy is used as the sole control.

### Outcomes

- Recommend fail-closed semantics as ready for audit/trace propagation, or record targeted remediation.
- Return tracking handoff for the coordinator to update [P3B] and [Q3B].

---

## ✅ [P3C] Audit records and trace_id propagation

Status: ✅ Done  
Deliverable path: `src/ba_agent/models.py`, `src/ba_agent/gateway.py`, `src/ba_agent/evaluation.py`, `docs/development/g3-readiness.md`  
Result: Added audit record model, gateway audit emission, audit write fail-closed behavior, and eval metadata with run IDs, trace IDs, fixture version, and graph version.  
Test evidence: `make check` passed; gateway audit tests and eval metadata output passed.

### Purpose

Define audit records and propagate `trace_id` across local CLI, router/graph, gateway, fixture reads, summary generation, Adaptive Card payloads, and evaluation records.

### Execution prompt

Implement trace and audit propagation.

Requirements:

1. Define trace context with `trace_id`, run ID, case ID, fixture version, graph version, route, and prompt/version placeholders [inferred] if used.
2. Define audit record fields aligned to MCP contracts: user ID placeholder, tool name, input hash, source system, timestamp, result status, and evidence refs.
3. Include local fields where useful: `trace_id`, route, capability, graph version, fixture version, and model version placeholder [inferred].
4. Emit an audit record for every gateway call, including ok, degraded, denied, throttled, rejected, blocked, and failed calls.
5. Ensure audit write failure fails the tool call in local behavior if audit writing is modeled.
6. Do not include secrets, live tokens, real user data, restricted content, or prompt contents with sensitive data in audit records.

### Tests

- Run the project test command after confirming it exists.
- Add tests proving a supplied `trace_id` is preserved end-to-end.
- Add tests proving a generated `trace_id` appears in audit records and final output/card.
- Add tests for required audit fields on ok/degraded/denied/rejected paths.
- Add tests for input hashing stability without exposing raw restricted data.

### Validations

- Confirm trace IDs are unique by default and injectable for deterministic tests.
- Confirm trace fields appear in Adaptive Card footer and evaluation records.
- Confirm audit records contain no secrets, live tokens, real user identifiers, or restricted source content.

### Outcomes

- `trace_id` propagation is testable end-to-end in local synthetic runs.
- Audit records are structured enough for G3 readiness review.
- Immediately run QA prompt [Q3C].

---

## ✅ [Q3C] QA review for audit and trace propagation

Status: ✅ Done  
Deliverable path: `src/ba_agent/models.py`, `src/ba_agent/gateway.py`, `src/ba_agent/evaluation.py`, `docs/development/g3-readiness.md`  
Result: QA verified audit fields, trace/eval metadata, no sensitive audit contents, and audit failure fail-closed behavior.  
Test evidence: `aara-project-reviewer` F3 QA returned OK; security review found no blockers.

### Purpose

Review [P3C] for trace completeness, audit safety, evidence linkage, and deterministic testability.

### Execution prompt

Review trace/audit implementation. Apply focused fixes only.

### Tests

- Re-run all trace/audit tests.
- Run one local synthetic command with an injected `trace_id` if supported.
- Add a regression test for any missing trace hop.

### Validations

- Inspect output samples for evidence refs and trace footer consistency.
- Confirm audit failures do not silently pass where fail-closed behavior is expected.
- Confirm no retention or production logging claim is made without `[RAJA]`.

### Outcomes

- Recommend trace/audit foundation as ready for evaluation hardening, or record targeted remediation.
- Return tracking handoff for the coordinator to update [P3C] and [Q3C].

---

## ✅ [P3D] GTS-GATE and evaluation hardening

Status: ✅ Done  
Deliverable path: `src/ba_agent/evaluation.py`, `src/ba_agent/cli.py`, `Makefile`, `tests/test_evaluation.py`, `tests/test_cli.py`, `eval/README.md`  
Result: Added GTS-GATE seed eval and command, hard-gate metrics for BA-EM-005 and BA-EM-009, and machine-readable eval metadata.  
Test evidence: `make check` passed; `make eval-gate` passed across 7 cases with approval_gate_bypass_count=0; GTS-ROUTER passed with phase_separation_violations=0.

### Purpose

Harden the local evaluation harness for GTS-GATE, GTS-STANDUP, and GTS-ROUTER so hard gates are measured before sandbox integration.

### Execution prompt

Implement or refine local evaluation harness support.

Requirements:

1. Define eval cases with case ID, input prompt/event, fixture refs, expected route, expected output characteristics, expected evidence refs, gate expectations, and expected audit behavior.
2. Seed GTS-GATE cases for missing approval ref, wrong artifact, replay, instruction-in-data, and disallowed write from standup.
3. Ensure BA-EM-005 approval-gate bypass count is computed with hard pass condition zero.
4. Ensure BA-EM-009 Phase-separation violations are computed with hard pass condition zero.
5. Compute owner-threshold metrics where feasible, but report them as measured/no-threshold unless RAJA has set a threshold.
6. Produce local machine-readable result output with run ID, case IDs, fixture version, graph version, and trace IDs.
7. Keep the harness local/synthetic and independent of live Teams/MCP/model connectivity.

### Tests

- Run the project test command after confirming it exists.
- Add eval case loading and validation tests.
- Add eval-runner tests for GTS-GATE rejection cases, GTS-ROUTER Phase 2 blocked cases, and GTS-STANDUP degraded cases.
- Run the eval command(s) you created for GTS-GATE, GTS-ROUTER, and GTS-STANDUP.

### Validations

- Confirm all eval fixtures are synthetic-only.
- Confirm no numeric owner thresholds are fabricated.
- Confirm hard gate results are visible in eval output.
- Confirm the harness does not require live connectivity.

### Outcomes

- Local evaluation hardening can measure BA-EM-005 and BA-EM-009.
- GTS-GATE evidence is ready for G3 review.
- Immediately run QA prompt [Q3D].

---

## ✅ [Q3D] QA review for GTS-GATE and evaluation hardening

Status: ✅ Done  
Deliverable path: `src/ba_agent/evaluation.py`, `tests/test_evaluation.py`, `docs/development/g3-readiness.md`  
Result: QA verified deterministic GTS-GATE coverage, no fabricated owner thresholds, BA-EM-005=0, and BA-EM-009=0.  
Test evidence: `aara-project-reviewer` F3 QA returned OK; manual eval evidence confirmed gate/router hard metrics.

### Purpose

Review [P3D] for harness correctness, deterministic coverage, metric honesty, and hard-gate enforcement.

### Execution prompt

Review evaluation hardening. Make targeted fixes only.

### Tests

- Re-run all eval tests.
- Run each eval set command created in [P3D].
- Add a unit test proving a simulated bypass would fail the hard gate.

### Validations

- Inspect seed cases for synthetic-only data and required evidence refs.
- Confirm owner-threshold metrics are reported without invented pass/fail thresholds.
- Confirm Phase 2 requests are only blocked/flagged, not implemented.

### Outcomes

- Recommend evaluation hardening as ready for G3 review, or record targeted remediation.
- Return tracking handoff for the coordinator to update [P3D] and [Q3D].

---

## ✅ [P3E] G3 control gate review

Status: ✅ Done  
Deliverable path: `docs/development/g3-readiness.md`  
Result: Created G3 readiness evidence with gateway boundary, allowlist summary, approval/idempotency semantics, audit schema/sample, eval results, and G4 prerequisites.  
Test evidence: `make check` passed; manual GTS-GATE/GTS-ROUTER/synthetic demo evidence recorded.

### Purpose

Package G3 Control Gate evidence: zero approval-gate bypasses in adversarial tests and audit records for all tool calls.

### Execution prompt

Create or update a G3 readiness review artifact.

Include:

1. Gateway/control boundary summary.
2. Tool allowlist summary.
3. `approval_ref` and idempotency semantics summary.
4. Audit record schema and sample redacted local audit records.
5. Trace propagation evidence from local synthetic runs.
6. GTS-GATE, GTS-ROUTER, and GTS-STANDUP run IDs/results.
7. Explicit statement that Phase 1-3 remain synthetic/local only and G3 does not authorize sandbox or live access.
8. Blockers for G4 sandbox readiness, especially actual MCP schema validation and tool-owner scopes.

### Tests

- Run the project test command after confirming it exists.
- Run GTS-GATE, GTS-ROUTER, and GTS-STANDUP eval commands.
- Run one local synthetic standup demo with trace output.

### Validations

- Confirm BA-EM-005 result is zero and BA-EM-009 result is zero in recorded evidence.
- Confirm all gateway calls in sampled runs have audit records.
- Run `rg -n "live|production|sandbox enabled|write enabled" <changed_paths>` and correct overclaims.

### Outcomes

- G3 readiness evidence is ready for RAJA/control review.
- Remaining G4 prerequisites are explicit.
- Immediately run QA prompt [Q3E].

---

## ✅ [Q3E] QA review for G3 control gate

Status: ✅ Done  
Deliverable path: `docs/development/g3-readiness.md`  
Result: QA verified G3 evidence completeness, hard-gate accuracy, no sandbox/pilot/production overclaim, and G4 prerequisites.  
Test evidence: `aara-project-reviewer` F3 QA returned OK; security review found no blockers.

### Purpose

Review [P3E] for gate evidence completeness, hard-gate accuracy, and no overclaiming beyond G3.

### Execution prompt

Review the G3 readiness artifact. Fix missing evidence, inaccurate results, or scope overclaims.

### Tests

- Re-run the test/eval commands cited in the G3 artifact.
- Re-run one traceable local demo.

### Validations

- Confirm G3 does not claim sandbox, pilot, production, or live write authorization.
- Confirm BA-EM-005 and BA-EM-009 evidence is directly traceable to eval output.
- Confirm audit examples are synthetic/redacted and contain no secrets.

### Outcomes

- Recommend Phase 3 readiness for RAJA/G3 review and G4 sandbox-readiness work, or record blockers.
- Return tracking handoff for the coordinator to update [P3E] and [Q3E].

---

## ✅ [P4A] Sandbox validation plan

Status: ✅ Done  
Deliverable path: `docs/development/sandbox-validation-plan.md`, `docs/development/mcp-validation-register.json`  
Result: Created sandbox validation plan and working validation register; no read tool is marked validated.  
Test evidence: `make check` passed with 47 tests; `make validate-mcp` reports no validated tools.

### Purpose

Create a sandbox validation plan that prepares for validated read-only MCP replacement without enabling unapproved tools.

### Execution prompt

Create or update a Phase 4 sandbox validation plan. Stop if [Q3E] is not complete or if RAJA/G3 review has not accepted control-gate evidence.

The plan must:

1. Identify candidate sandbox reads: Jira `get_sprint_status`, Git `get_recent_activity`, and other read tools only when needed.
2. State that actual tool schema, server name, auth model, scopes, rate limits, and owner approval must be validated before enablement.
3. Define a validation register working copy using `[RAJA]` placeholders for owners/scopes until confirmed.
4. Separate G4 sandbox read validation from G6 live pilot authorization.
5. Keep write tools blocked unless a later gate explicitly authorizes the exact approved action.
6. Include rollback to synthetic fixtures if any sandbox validation fails.

### Tests

- No code tests are required unless this prompt creates validation tooling.
- If validation tooling is created, run the project test command and the new validation-tool tests.
- Manually cross-check the plan against the MCP tool contracts validation register.

### Validations

- Confirm the plan does not treat proposed MCP contracts as build-authoritative truth.
- Confirm unvalidated tools remain stubbed/blocked.
- Run `rg -n "live pilot|production|write enabled|all projects|all repos" <changed_paths>` and correct unsafe wording.

### Outcomes

- Sandbox validation plan is ready for actual schema validation process work.
- G4 read-only boundaries are explicit.
- Immediately run QA prompt [Q4A].

---

## ✅ [Q4A] QA review for sandbox validation plan

Status: ✅ Done  
Deliverable path: `docs/development/sandbox-validation-plan.md`  
Result: QA verified sandbox readiness only, no live pilot/production/write authorization, and unvalidated tools remain blocked.  
Test evidence: `aara-project-reviewer` final F4 QA returned OK after validation hardening.

### Purpose

Review [P4A] for sandbox boundary clarity, MCP validation discipline, and no premature enablement.

### Execution prompt

Review the sandbox validation plan. Make focused fixes for unsafe scope, missing validation steps, or overclaims.

### Tests

- Re-run any validation-tool tests created in [P4A].
- Re-check the plan manually against MCP contract validation requirements.

### Validations

- Confirm G4 is sandbox readiness only, not live pilot approval.
- Confirm write tools remain blocked.
- Confirm owner/scopes are `[RAJA]` unless recorded elsewhere.

### Outcomes

- Recommend sandbox plan as ready for actual MCP schema validation, or record blockers.
- Return tracking handoff for the coordinator to update [P4A] and [Q4A].

---

## ✅ [P4B] Actual MCP schema validation process

Status: ✅ Done  
Deliverable path: `docs/development/mcp-schema-validation-process.md`, `src/ba_agent/validation.py`, `tests/test_validation.py`  
Result: Added MCP schema validation process and local register validation requiring sandbox env, read permission, owner/server, approved scopes, schema refs, auth/rate-limit refs, external approval evidence, timestamp, and no blockers.  
Test evidence: `make check` passed; tests cover unvalidated, incomplete validated, blank validated, and complete validated rows.

### Purpose

Define and, where safe, implement the process for validating actual MCP server schemas against the proposed contracts before any sandbox tool is enabled.

### Execution prompt

Create validation process docs and optional local tooling.

Requirements:

1. Define per-tool validation steps: owner named, actual server identified, schema captured, proposed-vs-actual diff recorded, auth/scopes/rate limits checked, sandbox scope approved, row marked validated with date.
2. Support read-only Jira/Git validation first.
3. Treat absent access as a blocker to be recorded, not a reason to mock approval.
4. If you create tooling, it must read local schema files or sandbox-approved metadata only; it must not call unapproved live endpoints.
5. Record deviations from proposed contracts with rationale and `[RAJA]` owner placeholders where needed.
6. Keep unvalidated tools blocked in config and code.

### Tests

- If tooling is created, run the project test command after confirming it exists.
- Add tests for schema-diff input validation, missing owner/scope rows, and blocked unvalidated tools.
- If no tooling is created, perform a manual checklist validation and record that no executable tooling was added.

### Validations

- Confirm validation docs distinguish proposed contract from actual validated schema.
- Confirm no sandbox secret, token, endpoint, or credential is committed.
- Run `rg -n "token|secret|password|client_secret|tenant" <changed_paths>` and resolve unsafe hits.

### Outcomes

- Actual MCP schema validation process is ready and auditable.
- Unvalidated MCP integrations remain blocked.
- Immediately run QA prompt [Q4B].

---

## ✅ [Q4B] QA review for MCP schema validation process

Status: ✅ Done  
Deliverable path: `docs/development/mcp-schema-validation-process.md`, `src/ba_agent/validation.py`, `tests/test_validation.py`  
Result: QA verified actual-schema rows cannot be marked enableable without complete validation evidence and no credentials/scope secrets are committed.  
Test evidence: Initial QA blocker on blank values fixed; final F4 QA returned OK.

### Purpose

Review [P4B] for validation-process completeness, safe tooling, and no credential leakage.

### Execution prompt

Review MCP validation process docs/tooling. Make targeted corrections for missing validation gates or unsafe assumptions.

### Tests

- Re-run validation-tool tests if tooling exists.
- Re-run the manual validation checklist on at least Jira and Git read tools.

### Validations

- Confirm actual-schema rows cannot be marked validated without owner/scope evidence.
- Confirm committed files contain no credentials or tenant-specific secrets.
- Confirm unvalidated tools remain blocked.

### Outcomes

- Recommend process as ready for read-only Jira/Git replacement work, or record blockers.
- Return tracking handoff for the coordinator to update [P4B] and [Q4B].

---

## ✅ [P4C] Read-only Jira/Git sandbox replacement path

Status: ✅ Done  
Deliverable path: `src/ba_agent/adapters.py`, `src/ba_agent/config.py`, `docs/development/read-only-sandbox-replacement.md`, `tests/test_validation.py`  
Result: Added synthetic-default read adapter boundary and sandbox_read mode that fails closed unless Jira/Git read tools are fully validated.  
Test evidence: `make check` passed; adapter tests verify synthetic default and sandbox-read fail-closed behavior.

### Purpose

Prepare the path to replace synthetic Jira/Git reads with validated sandbox MCP reads while preserving fallback to fixtures and blocking writes.

### Execution prompt

Implement or document the read-only replacement path.

Requirements:

1. Add a config-controlled data-source mode: synthetic by default; sandbox read mode only when validation register rows are marked validated.
2. Add adapter interfaces for Jira sprint status and Git recent activity that can use synthetic fixtures or validated sandbox reads.
3. Keep Jira/Git write tools absent or blocked.
4. Preserve `source_timestamp`, `retrieved_at`, evidence refs, `trace_id`, denied/degraded/throttled statuses, and audit records.
5. Ensure sandbox reads are scoped to approved projects/repos only.
6. If actual sandbox access is unavailable, implement only the adapter boundary and record the blocker.

### Tests

- Run the project test command after confirming it exists.
- Add tests that default mode is synthetic.
- Add tests that sandbox mode fails closed unless validation rows and approved scopes are present.
- Add tests that read adapters preserve evidence refs and degraded/denied statuses.
- Add tests that write attempts remain rejected.

### Validations

- Confirm no live or production mode is enabled by default.
- Confirm any sandbox endpoint/config is not committed as a secret.
- Run `rg -n "update_sprint_scope|commit|push|write|publish" <changed_source_paths>` and confirm hits are blocked/test-only where relevant.

### Outcomes

- Read-only Jira/Git replacement path is ready for validated sandbox use.
- Synthetic fallback remains default and safe.
- Immediately run QA prompt [Q4C].

---

## ✅ [Q4C] QA review for read-only Jira/Git replacement

Status: ✅ Done  
Deliverable path: `src/ba_agent/adapters.py`, `src/ba_agent/config.py`, `docs/development/read-only-sandbox-replacement.md`  
Result: QA verified synthetic fallback remains default, sandbox reads do not imply pilot approval, and write/mutation paths remain blocked.  
Test evidence: `aara-project-reviewer` final F4 QA returned OK; security review found no blockers.

### Purpose

Review [P4C] for safe default behavior, validated-read gating, evidence preservation, and write blocking.

### Execution prompt

Review the read-only replacement path. Apply focused fixes for unsafe defaults or schema drift.

### Tests

- Re-run adapter/config tests.
- Re-run standup synthetic tests to prove fallback still works.
- Add a regression test if sandbox mode can activate without validation.

### Validations

- Confirm validated sandbox reads do not imply live pilot approval.
- Confirm Jira/Git mutations remain blocked.
- Confirm evidence refs and timestamps survive adapter translation.

### Outcomes

- Recommend read-only replacement path as ready for Teams sandbox readiness, or record blockers.
- Return tracking handoff for the coordinator to update [P4C] and [Q4C].

---

## ✅ [P4D] Teams sandbox and channel approval readiness

Status: ✅ Done  
Deliverable path: `docs/development/teams-sandbox-readiness.md`  
Result: Documented Teams sandbox approval requirements and fallback local card JSON review; no send adapter or live posting enabled.  
Test evidence: `make check` passed; card send remains a fail-closed stub; no tenant/channel/secret values committed.

### Purpose

Prepare Teams/Copilot 365 sandbox and channel approval readiness without sending live messages before approval.

### Execution prompt

Create or update Teams sandbox readiness materials and optional local validation.

Requirements:

1. Document required tenant/app/channel approvals, approved channel scope, and audience/classification checks as `[RAJA]` until confirmed.
2. Keep Teams Adaptive Card payload validation independent from live posting.
3. If a sandbox send adapter is represented, keep it disabled until the Teams tool row is validated and channel approval is recorded.
4. Ensure card payloads carry evidence refs, `trace_id`, data-quality status, and advisory/synthetic labels where appropriate.
5. Do not add Slack, alternate chat channels, or unapproved Microsoft Graph permissions.
6. Document the fallback: local card JSON review when Teams sandbox approval is unavailable.

### Tests

- Run the project test command after confirming it exists.
- Re-run Adaptive Card payload tests.
- Add tests that Teams send is blocked until validation and approved channel config are present.
- If local schema validation exists, run it on normal and degraded card payloads.

### Validations

- Confirm no real channel ID, tenant ID, bot secret, or Graph permission is committed.
- Confirm card copy targets Teams/Copilot 365 only.
- Run `rg -n "Slack|Bot Framework|GraphServiceClient|channel_id|tenant|client_secret" <changed_paths>` and confirm hits are approved placeholders, stubs, or removed.

### Outcomes

- Teams sandbox/channel readiness is documented.
- Live posting remains blocked until validated and approved.
- Immediately run QA prompt [Q4D].

---

## ✅ [Q4D] QA review for Teams sandbox readiness

Status: ✅ Done  
Deliverable path: `docs/development/teams-sandbox-readiness.md`  
Result: QA verified Teams/Copilot-only readiness language, no Slack/alternate chat channel, no real tenant/channel/secret values, and posting remains disabled.  
Test evidence: `aara-project-reviewer` final F4 QA returned OK; security review found no blockers.

### Purpose

Review [P4D] for Teams/Copilot 365 alignment, no-posting enforcement, and approval readiness.

### Execution prompt

Review Teams sandbox readiness materials/tooling. Fix only approval, scope, or no-posting issues.

### Tests

- Re-run card payload tests.
- Re-run blocked-send tests.
- Run local schema validation if available.

### Validations

- Confirm no Slack or alternate chat channel language is introduced.
- Confirm no real tenant/channel/secret values are committed.
- Confirm Teams posting is disabled unless validation and approval are recorded.

### Outcomes

- Recommend Teams readiness as ready for G4 blocked-tool review, or record blockers.
- Return tracking handoff for the coordinator to update [P4D] and [Q4D].

---

## ✅ [P4E] Block unvalidated tools and G4 review

Status: ✅ Done  
Deliverable path: `docs/development/g4-readiness.md`, `docs/development/mcp-validation-register.json`, `src/ba_agent/validation.py`, `src/ba_agent/adapters.py`  
Result: Created G4 readiness evidence showing no validated tools, blocked Jira/Git/Teams tool rows, synthetic default, sandbox_read fail-closed, and no live/pilot/production authorization.  
Test evidence: `make check` passed with 47 tests; `make validate-mcp` reports validated=[] and blocked get_sprint_status/get_recent_activity/send_adaptive_card.

### Purpose

Prove unvalidated tools remain blocked and package G4 Sandbox Gate readiness evidence.

### Execution prompt

Create or update the G4 readiness review.

Include:

1. Validation register status for Jira/Git reads and any other considered tool.
2. Evidence that only validated read tools can be enabled.
3. Evidence that unvalidated tools remain stubbed/blocked.
4. Evidence that write tools remain approval-gated and blocked unless explicitly authorized later.
5. Denied/degraded/throttled behavior tests for sandbox-read adapters.
6. Teams sandbox/channel approval readiness status.
7. Deviations, blockers, and fallback to synthetic fixtures.

### Tests

- Run the project test command after confirming it exists.
- Run adapter gating tests, blocked-send tests, and GTS-GATE tests.
- Run standup evals in synthetic mode and sandbox-read mode only if validation rows and approved sandbox config exist.

### Validations

- Confirm no unvalidated tool can be enabled by config alone.
- Confirm G4 does not authorize live pilot or production use.
- Run `rg -n "LIVE_INTEGRATIONS_ENABLED.*true|production|prod|write enabled|unvalidated.*enabled" <changed_paths>` and correct unsafe wording or code.

### Outcomes

- G4 readiness evidence is ready for RAJA/tool-owner review.
- Unvalidated tools are provably blocked.
- Immediately run QA prompt [Q4E].

---

## ✅ [Q4E] QA review for blocked tools and G4 readiness

Status: ✅ Done  
Deliverable path: `docs/development/g4-readiness.md`  
Result: QA verified G4 readiness evidence, unvalidated tools blocked, no sandbox/live/pilot/production/write authorization, and complete validation-evidence requirements.  
Test evidence: `aara-project-reviewer` final F4 QA returned OK; `security-review` found no blockers.

### Purpose

Review [P4E] for G4 gate accuracy, unvalidated-tool blocking, and no live/pilot overclaims.

### Execution prompt

Review G4 readiness evidence. Make focused corrections for missing test proof, invalid validation rows, or unsafe claims.

### Tests

- Re-run tests/evals cited in the G4 artifact.
- Add a regression test if any unvalidated tool can be enabled.

### Validations

- Confirm only validated read tools can replace fixtures in sandbox.
- Confirm write tools remain blocked/gated.
- Confirm G4 readiness does not claim G5/G6 approval.

### Outcomes

- Recommend Phase 4 readiness for RAJA/G4 review and MVP capability expansion, or record blockers.
- Return tracking handoff for the coordinator to update [P4E] and [Q4E].

---

## ✅ [P5A] Sprint planning recommendation flow

Status: ✅ Done  
Deliverable path: `src/ba_agent/mvp.py`, `src/ba_agent/evaluation.py`, `tests/test_mvp.py`, `tests/test_evaluation.py`  
Result: Added planning recommendation flow with draft/advisory output, missing velocity/availability open questions, request-approval-only behavior, and no sprint-scope publish/update call.  
Test evidence: `make check` passed with 55 tests; `GTS-PLANNING` passed across 5 cases with publish_bypass_count=0.

### Purpose

Implement MVP sprint planning recommendations using backlog, velocity, and calendar availability, with approval request only and no sprint-scope publishing.

### Execution prompt

Implement the planning capability after G3/G4 controls are in place. Stop if [Q4E] is not complete or if RAJA/G4 review has not accepted sandbox-readiness evidence. Do not expand MVP capabilities on unvalidated tools.

Requirements:

1. Use synthetic fixtures or validated sandbox reads only; default to synthetic if validation is incomplete.
2. Analyze backlog priority, velocity history, and aggregate calendar availability.
3. Produce a recommendation labeled as draft/advisory, not approved scope.
4. Route approval through a local or validated `request_approval` control path only; do not call `update_sprint_scope`.
5. Preserve evidence refs and data-quality status for backlog, velocity, and availability.
6. If velocity or availability data is missing, ask for input or mark unavailable; do not invent capacity.
7. Keep `approval_ref` semantics in the gateway/control layer.

### Tests

- Run the project test command after confirming it exists.
- Add GTS-PLANNING tests for normal, low availability, missing velocity, oversized backlog, and rejected approval.
- Add tests that planning never publishes or mutates Jira without an approved, validated later gate.
- Run GTS-GATE tests.

### Validations

- Confirm recommendations are labeled draft/advisory.
- Confirm calendar details are aggregate-only and do not expose event subjects or attendees.
- Run `rg -n "update_sprint_scope|publish|commit sprint|approved scope" <changed_source_paths>` and confirm hits are blocked, test-only, or advisory.

### Outcomes

- Planning recommendation flow exists without publishing sprint scope.
- Approval request behavior is controlled by gateway semantics.
- Immediately run QA prompt [Q5A].

---

## ✅ [Q5A] QA review for sprint planning flow

Status: ✅ Done  
Deliverable path: `src/ba_agent/mvp.py`, `tests/test_mvp.py`  
Result: QA verified recommendation-only planning, no `update_sprint_scope` call in planning, missing availability/velocity not estimated, and approval request does not issue `approval_ref`.  
Test evidence: Initial QA blockers fixed; final F5 QA returned OK and security review found no blockers.

### Purpose

Review [P5A] for recommendation-only behavior, evidence discipline, privacy, and approval-gate enforcement.

### Execution prompt

Review the planning flow. Fix only targeted issues that could cause overcommitment, fabricated capacity, or write bypass.

### Tests

- Re-run GTS-PLANNING tests.
- Re-run GTS-GATE tests.
- Add a regression test if any publish/write path is reachable.

### Validations

- Confirm no sprint scope is published or mutated.
- Confirm recommendations remain distinguishable from human decisions.
- Confirm missing metrics/data are not estimated.

### Outcomes

- Recommend planning flow as ready for retro work, or record remediation.
- Return tracking handoff for the coordinator to update [P5A] and [Q5A].

---

## ✅ [P5B] Retrospective draft-only flow

Status: ✅ Done  
Deliverable path: `src/ba_agent/mvp.py`, `src/ba_agent/evaluation.py`, `tests/test_mvp.py`, `tests/test_evaluation.py`  
Result: Added retrospective draft-only report generation with complete/partial/zero metric cases, missing metric honesty, and publish blocked by gateway.  
Test evidence: `make check` passed; `GTS-RETRO` passed across 3 cases with publish_bypass_count=0.

### Purpose

Implement retrospective report generation as draft-only, with missing metric honesty and no Confluence publish.

### Execution prompt

Implement the retro capability.

Requirements:

1. Use synthetic fixtures or validated sandbox Jira metrics reads only.
2. Generate a structured retrospective report with cycle time, carry-over, defect rate, improvements, assumptions/open questions, evidence refs, and `trace_id`.
3. If a metric is unavailable, represent it as unavailable/null with `missing_fields`; do not estimate.
4. Prepare local draft output or validated sandbox draft-only behavior only when allowed by the validation register.
5. Keep `publish_page` blocked unless a later explicit gate authorizes it with `approval_ref`.
6. Label all outputs draft-only and not approved for publication.

### Tests

- Run the project test command after confirming it exists.
- Add GTS-RETRO tests for complete metrics, partial metrics, zero-defect/zero-carry-over sprint, and publish attempt rejected.
- Run GTS-GATE tests.

### Validations

- Confirm no Confluence publish path is enabled.
- Confirm missing metrics are not fabricated.
- Run `rg -n "publish_page|Confluence.*publish|auto-post|estimate" <changed_source_paths>` and confirm hits are blocked/test-only or corrected.

### Outcomes

- Retro draft-only flow exists with evidence-linked metrics.
- Publish remains blocked/gated.
- Immediately run QA prompt [Q5B].

---

## ✅ [Q5B] QA review for retrospective draft-only flow

Status: ✅ Done  
Deliverable path: `src/ba_agent/mvp.py`, `tests/test_mvp.py`  
Result: QA verified draft-only retro output, no Confluence publish enablement, evidence-linked metrics, and no estimated missing metrics.  
Test evidence: Final F5 QA returned OK and security review found no blockers.

### Purpose

Review [P5B] for draft-only behavior, metric fidelity, evidence refs, and no publish bypass.

### Execution prompt

Review the retro flow. Make focused fixes for fabricated metrics, publish drift, or missing evidence.

### Tests

- Re-run GTS-RETRO tests.
- Re-run GTS-GATE tests.
- Add a regression test if publish behavior is ambiguous.

### Validations

- Confirm retro outputs are draft-only.
- Confirm every metric traces to source/fixture evidence or is marked unavailable.
- Confirm no approved Confluence space is invented.

### Outcomes

- Recommend retro flow as ready for health monitoring work, or record remediation.
- Return tracking handoff for the coordinator to update [P5B] and [Q5B].

---

## ✅ [P5C] Sprint health advisory monitoring

Status: ✅ Done  
Deliverable path: `src/ba_agent/mvp.py`, `src/ba_agent/evaluation.py`, `tests/test_mvp.py`, `tests/test_evaluation.py`  
Result: Added health advisory reports for healthy, blocker, scope creep, resource conflict, and stalled story cases with [RAJA] severity and blocked escalation send.  
Test evidence: `make check` passed; `GTS-HEALTH` passed across 5 cases with escalation_bypass_count=0.

### Purpose

Implement sprint health monitoring as advisory-only, with scheduled/webhook trigger abstractions, source-linked findings, and placeholder severity rules marked `[RAJA]`.

### Execution prompt

Implement the health capability.

Requirements:

1. Use synthetic fixtures or validated sandbox read data only.
2. Model schedule/webhook triggers locally or through validated sandbox metadata; do not register unapproved webhooks.
3. Detect stalled stories, scope creep, blockers, and resource conflicts only when supported by data.
4. Mark severity taxonomy thresholds `[RAJA]` until defined.
5. Produce advisory escalations with evidence refs and suggested actions labeled recommendations.
6. Keep `send_escalation` blocked unless validated and approved; local output is acceptable.
7. Preserve `trace_id` and audit records for checks.

### Tests

- Run the project test command after confirming it exists.
- Add GTS-HEALTH tests for healthy sprint, stalled story, scope creep, resource conflict, ambiguous severity, and blocked send escalation.
- Run GTS-GATE tests and BA-EM-009 Phase-separation checks.

### Validations

- Confirm health alerts are recommendations, not corrective-action commitments.
- Confirm webhook registration is not enabled without validation/approval.
- Run `rg -n "subscribe_sprint_events|send_escalation|webhook|severity" <changed_source_paths>` and confirm hits are validated, local, blocked, or `[RAJA]`.

### Outcomes

- Sprint health advisory monitoring exists without autonomous escalation.
- Severity and webhook decisions remain owner-gated.
- Immediately run QA prompt [Q5C].

---

## ✅ [Q5C] QA review for sprint health advisory monitoring

Status: ✅ Done  
Deliverable path: `src/ba_agent/mvp.py`, `tests/test_mvp.py`  
Result: QA verified advisory-only health monitoring, no webhook registration, blocked send escalation, [RAJA] severity, and no autonomous corrective action.  
Test evidence: Final F5 QA returned OK and security review found no blockers.

### Purpose

Review [P5C] for advisory-only behavior, evidence linkage, severity honesty, and blocked sends/webhooks.

### Execution prompt

Review health monitoring. Apply focused fixes for autonomous action, fabricated severity, or unapproved webhook/send behavior.

### Tests

- Re-run GTS-HEALTH tests.
- Re-run GTS-GATE and Phase-separation tests.
- Add a regression test if a send/webhook path is reachable without validation.

### Validations

- Confirm ambiguous severity is parked as `[RAJA]`, not guessed.
- Confirm suggested actions are not commitments.
- Confirm no live webhook is registered.

### Outcomes

- Recommend health monitoring as ready for expanded golden-set work, or record remediation.
- Return tracking handoff for the coordinator to update [P5C] and [Q5C].

---

## ✅ [P5D] Expanded MVP golden sets

Status: ✅ Done  
Deliverable path: `src/ba_agent/evaluation.py`, `src/ba_agent/cli.py`, `Makefile`, `tests/test_evaluation.py`, `tests/test_cli.py`, `eval/README.md`  
Result: Expanded local MVP evals to GTS-PLANNING, GTS-RETRO, GTS-HEALTH, and GTS-MVP with hard-gate metrics and no fabricated owner-threshold pass/fail.  
Test evidence: `make check` passed; `GTS-MVP` passed across 40 synthetic cases with approval_gate_bypass_count=0, phase_separation_violations=0, and owner_threshold_metrics_with_fabricated_threshold=0.

### Purpose

Expand the evaluation harness across all MVP capabilities and hard gates without inventing owner-set numeric thresholds.

### Execution prompt

Expand golden sets and metrics.

Requirements:

1. Add or complete GTS-STANDUP, GTS-PLANNING, GTS-RETRO, GTS-HEALTH, GTS-ROUTER, and GTS-GATE cases.
2. Include normal, degraded, denied, throttled, missing-data, adversarial prompt-injection, and Phase 2 leakage cases.
3. Compute BA-EM-001 through BA-EM-009 where feasible.
4. Enforce hard gates BA-EM-005 = 0 and BA-EM-009 = 0.
5. Report owner-threshold metrics as measured/no-threshold unless RAJA has set values.
6. Record run ID, case IDs, fixture versions, graph/prompt versions, and trace IDs.
7. Keep all golden data synthetic unless an approved sandbox path is explicitly validated for the specific read.

### Tests

- Run the project test command after confirming it exists.
- Run all MVP eval sets.
- Add tests for eval output structure, run ID presence, hard-gate failure behavior, and no threshold fabrication.

### Validations

- Confirm each golden set uses synthetic data or validated sandbox reads only.
- Confirm Phase 2 requests are blocked in MVP runs.
- Run `rg -n "threshold.*[0-9]|pass.*[0-9]+%" <changed_paths>` and confirm any numeric threshold is documented as RAJA-set or removed.

### Outcomes

- Expanded MVP golden sets are ready for G5 candidate review.
- Hard gates and owner-threshold metrics are reported honestly.
- Immediately run QA prompt [Q5D].

---

## ✅ [Q5D] QA review for expanded MVP golden sets

Status: ✅ Done  
Deliverable path: `src/ba_agent/evaluation.py`, `tests/test_evaluation.py`, `docs/development/g5-candidate-review.md`  
Result: QA verified expanded golden-set coverage, BA-EM-005=0, BA-EM-009=0, run IDs/versions, and no fabricated owner thresholds.  
Test evidence: Final F5 QA returned OK; manual MVP eval confirmed expected metrics.

### Purpose

Review [P5D] for golden-set coverage, metric honesty, hard-gate enforcement, and synthetic/sandbox data discipline.

### Execution prompt

Review expanded eval harness changes. Make targeted fixes for missing coverage or metric overclaims.

### Tests

- Re-run all MVP eval sets.
- Re-run hard-gate failure tests.
- Add a case if one required capability lacks degraded/denied/missing-data coverage.

### Validations

- Confirm BA-EM-005 and BA-EM-009 block candidate readiness on failure.
- Confirm owner-set thresholds are not fabricated.
- Confirm eval outputs include run ID and versions.

### Outcomes

- Recommend expanded evals as ready for MVP candidate gate review, or record blockers.
- Return tracking handoff for the coordinator to update [P5D] and [Q5D].

---

## ✅ [P5E] MVP candidate gate review

Status: ✅ Done  
Deliverable path: `docs/development/g5-candidate-review.md`  
Result: Created G5 MVP candidate review with capability status, eval evidence, hard-gate results, human review checklist, and G6 blockers.  
Test evidence: `make check` passed; planning/retro/health/MVP evals passed; no pilot/production overclaim found.

### Purpose

Package G5 MVP Candidate Gate evidence across standup, planning, retro, health, router, and gateway controls.

### Execution prompt

Create or update the G5 candidate review package.

Include:

1. Capability status for standup, planning, retro, and health.
2. Eval run IDs and results across all MVP golden sets.
3. BA-EM-005 and BA-EM-009 hard-gate results.
4. Owner-threshold metrics reported as measured/no-threshold or RAJA-waived where applicable.
5. Human review checklist for BA SME/QA sampling.
6. Remaining blockers for G6 pilot: pilot boundaries, classification handling, tool scopes, Teams channel approval, support/RACI, rollback drill, release notes.
7. Explicit statement that G5 candidate readiness does not authorize live pilot without G6.

### Tests

- Run the project test command after confirming it exists.
- Run all MVP eval sets and capture run IDs.
- Run at least one local/sandbox-approved demo per implemented capability.

### Validations

- Confirm BA-EM-005 = 0 and BA-EM-009 = 0 in recorded evidence.
- Confirm no unreviewed owner-threshold metric is converted into pass/fail.
- Run `rg -n "pilot approved|production ready|live enabled" <changed_paths>` and correct overclaims.

### Outcomes

- G5 candidate package is ready for RAJA and reviewer decision.
- G6 readiness blockers are explicit.
- Immediately run QA prompt [Q5E].

---

## ✅ [Q5E] QA review for MVP candidate gate

Status: ✅ Done  
Deliverable path: `docs/development/g5-candidate-review.md`  
Result: QA verified G5 package completeness, hard-gate evidence, G6 blockers, and no live pilot/production/Phase 2 authorization.  
Test evidence: Final F5 QA returned OK and security review found no blockers.

### Purpose

Review [P5E] for candidate-gate completeness, hard-gate evidence, and no premature pilot authorization.

### Execution prompt

Review the G5 package. Correct missing evidence, inaccurate gate status, or overclaiming.

### Tests

- Re-run all tests/evals cited by the G5 package.
- Spot-check one output per MVP capability for evidence refs and advisory/draft labeling.

### Validations

- Confirm G5 does not authorize G6 live pilot.
- Confirm G6 blockers are listed with RAJA/tool-owner/security/platform review lanes.
- Confirm no Phase 2 capability is included in MVP release notes/backlog.

### Outcomes

- Recommend Phase 5 readiness for RAJA/G5 review and pilot-readiness work, or record blockers.
- Return tracking handoff for the coordinator to update [P5E] and [Q5E].

---

## ✅ [P6A] Pilot runbook and scope package

Status: ✅ Done  
Deliverable path: `docs/development/pilot-runbook.md`  
Result: Created limited-scope MVP pilot runbook with [RAJA] scopes, entry criteria, stop conditions, data/evidence expectations, and no live-use authorization.  
Test evidence: `make check` passed; F6 QA verified limited scope and no self-authorization.

### Purpose

Create a controlled MVP pilot runbook and scope package for approved live use only after G6 authorization.

### Execution prompt

Create or update a pilot runbook. Stop if [Q5E] is not complete or if RAJA/G5 review has not accepted MVP candidate evidence. This prompt prepares pilot readiness only; it does not authorize live use.

Include:

1. Pilot objective and limited scope.
2. Candidate team, Jira project, repository, Teams channel, Confluence space, calendar scope, and approvers as `[RAJA]` until confirmed.
3. Entry criteria: G5 passed, classification handling approved, tool scopes validated, Teams/channel approved, support/RACI reviewed, rollback/kill switch tested, release notes prepared.
4. In-scope capabilities and explicit out-of-scope Phase 2 capabilities.
5. Data-handling and evidence/audit expectations.
6. Daily/weekly pilot operating cadence as `[RAJA]` unless set.
7. Stop conditions: gate bypass, data exposure, unauthorized write, severe output regression, or scope drift.

### Tests

- No code tests are required unless the runbook adds tooling.
- If tooling or scripts are added, run the project test command after confirming it exists.
- Manually cross-check runbook entry criteria against G5/G6 plan requirements.

### Validations

- Confirm the runbook does not enable live use by itself.
- Confirm all unconfirmed pilot scopes are `[RAJA]`.
- Run `rg -n "all projects|all repos|all channels|unrestricted|production ready" <changed_paths>` and correct unsafe wording.

### Outcomes

- Pilot runbook and scope package are ready for support/RACI review.
- G6 entry criteria are visible.
- Immediately run QA prompt [Q6A].

---

## ✅ [Q6A] QA review for pilot runbook

Status: ✅ Done  
Deliverable path: `docs/development/pilot-runbook.md`  
Result: QA verified pilot runbook entry criteria, stop conditions, least-privilege scope placeholders, and no Phase 2 pilot scope.  
Test evidence: F6 QA returned OK for P6A/Q6A.

### Purpose

Review [P6A] for limited-scope pilot discipline, entry criteria, stop conditions, and no self-authorization.

### Execution prompt

Review the pilot runbook. Make targeted fixes for unsafe scope or missing gate criteria.

### Tests

- Re-check runbook entry criteria against G6 requirements in the project plan.
- Re-run any tooling tests added by [P6A].

### Validations

- Confirm the runbook requires explicit RAJA authorization before limited live use.
- Confirm pilot scope is least-privilege and bounded.
- Confirm Phase 2 remains out of pilot scope.

### Outcomes

- Recommend pilot runbook as ready for support/RACI review, or record blockers.
- Return tracking handoff for the coordinator to update [P6A] and [Q6A].

---

## ✅ [P6B] Support/RACI under RAJA accountability

Status: ✅ Done  
Deliverable path: `docs/development/support-raci.md`  
Result: Created pilot support/RACI with RAJA accountability, role lanes as [RAJA], support tiers, incident triggers, and trace/audit support path.  
Test evidence: F6 QA verified no invented SLAs/named staff and support paths align to trace/audit evidence.

### Purpose

Review and adapt support/RACI responsibilities for the pilot while preserving RAJA as accountable owner for this baseline.

### Execution prompt

Create or update support/RACI readiness materials.

Requirements:

1. Preserve RAJA accountable-owner baseline.
2. Define responsible/review lanes for Product Owner, BA SME, Scrum Master, QA, Security/privacy, Platform, Tool owners, and Delivery lead as `[RAJA]` until named.
3. Document support tiers for output questions, platform/tool errors, security incidents, and gate bypasses.
4. Define incident triggers and escalation paths, especially approval-gate bypass and data exposure.
5. Link support steps to `trace_id`, audit records, eval run IDs, and release versions.
6. Do not invent support SLAs, cadence, or named on-call staff; mark owner-dependent values `[RAJA]`.

### Tests

- No code tests are required unless support tooling is added.
- If tooling is added, run the project test command after confirming it exists.
- Manually trace one sample issue from Teams output `trace_id` to audit/eval evidence using available local artifacts.

### Validations

- Confirm RAJA accountability remains explicit.
- Confirm support model does not claim 24x7, SLA, or named staff without `[RAJA]`.
- Confirm Sev1-like gate bypass behavior disables writes at gateway/kill switch.

### Outcomes

- Support/RACI readiness is documented for G6 review.
- Incident paths are traceable and owner-routed.
- Immediately run QA prompt [Q6B].

---

## ✅ [Q6B] QA review for support/RACI

Status: ✅ Done  
Deliverable path: `docs/development/support-raci.md`  
Result: QA verified RAJA accountability, support routing, escalation triggers, and no fabricated operational commitments.  
Test evidence: F6 QA returned OK for P6B/Q6B.

### Purpose

Review [P6B] for RAJA accountability, realistic support routing, and no fabricated operational commitments.

### Execution prompt

Review support/RACI materials. Make focused corrections for accountability drift, missing escalation paths, or invented commitments.

### Tests

- Re-run any support-tool tests.
- Manually validate one trace-to-audit support scenario.

### Validations

- Confirm all unnamed roles/cadences are `[RAJA]`.
- Confirm gate bypass routes to immediate disable/kill-switch behavior.
- Confirm support model aligns with audit and trace evidence.

### Outcomes

- Recommend support/RACI package as ready for release-note work, or record blockers.
- Return tracking handoff for the coordinator to update [P6B] and [Q6B].

---

## ✅ [P6C] Release notes and harness run ID

Status: ✅ Done  
Deliverable path: `docs/development/pilot-release-notes.md`  
Result: Created pilot release notes with package/prompt/fixture/graph versions, MVP harness run IDs, hard-gate evidence, limitations, blocked tools, and rollback references.  
Test evidence: `make check` passed; F6 QA verified actual run IDs and no registry/cloud/production overclaim.

### Purpose

Prepare pilot release notes that record versions, harness run ID, gate evidence, known limitations, and rollback reference.

### Execution prompt

Create or update release-note materials for the MVP pilot candidate.

Include:

1. Version identifiers for code, prompt/graph, fixtures, eval harness, and model/config where applicable.
2. Harness run ID and results for all MVP golden sets.
3. BA-EM-005 and BA-EM-009 hard-gate evidence.
4. Owner-threshold metrics as measured/no-threshold or explicitly RAJA-waived.
5. Validated tool scopes and blocked tool list.
6. Known limitations and unresolved `[RAJA]` items.
7. Rollback references: previous code/prompt/graph version, kill-switch flags, and Artifactory tag only if containers are actually in use.
8. No out-of-band production prompt edits.

### Tests

- Run the project test command after confirming it exists.
- Run all MVP eval sets and capture the run ID used in release notes.
- If packaging or container steps exist, run only the existing non-production validation commands and do not introduce registry publishing in this prompt.

### Validations

- Confirm release notes cite actual run IDs and versions.
- Confirm no Azure ACR reference appears if registry is mentioned; use JFrog Artifactory only.
- Run `rg -n "Azure ACR|acr\\.azurecr\\.io|out-of-band|manual prod prompt" <changed_paths>` and correct drift.

### Outcomes

- Pilot release notes include harness run ID and gate evidence.
- Rollback references are ready for drill validation.
- Immediately run QA prompt [Q6C].

---

## ✅ [Q6C] QA review for release notes

Status: ✅ Done  
Deliverable path: `docs/development/pilot-release-notes.md`  
Result: QA verified release-note accuracy, BA-EM-005/BA-EM-009 evidence, no Azure ACR/cloud credential path, and live pilot remains pending G6 authorization.  
Test evidence: F6 QA returned OK for P6C/Q6C.

### Purpose

Review [P6C] for release-note accuracy, version/run evidence, registry alignment, and no production overclaims.

### Execution prompt

Review release notes. Correct missing run IDs, unsupported version claims, or platform drift.

### Tests

- Re-run the eval command that produced the cited harness run ID, or confirm the saved result exists.
- Re-run project tests if release-note generation tooling changed.

### Validations

- Confirm BA-EM-005 and BA-EM-009 evidence is cited.
- Confirm no Azure ACR, stored cloud credential, or out-of-band prompt-change path is introduced.
- Confirm live pilot is still pending G6 authorization.

### Outcomes

- Recommend release notes as ready for rollback/kill-switch drill, or record blockers.
- Return tracking handoff for the coordinator to update [P6C] and [Q6C].

---

## ✅ [P6D] Rollback and kill-switch drill

Status: ✅ Done  
Deliverable path: `docs/development/rollback-kill-switch-drill.md`  
Result: Created local/non-production rollback and kill-switch drill evidence covering write disablement, route/capability boundaries, synthetic fallback, live-mode rejection, and sandbox-read blocking.  
Test evidence: `make check` passed; GTS-GATE and validate-mcp evidence recorded.

### Purpose

Drill rollback and kill-switch behavior before limited live pilot authorization, proving writes/capabilities can be disabled quickly and safely.

### Execution prompt

Implement or document a non-production rollback/kill-switch drill.

Requirements:

1. Define kill switches for write tools, per-capability router disablement, sandbox/live tool modes, and model/prompt/graph rollback where applicable.
2. Demonstrate write tools disabled at the gateway without code changes if feature flags/config support it.
3. Demonstrate capability disablement for planning/retro/health while standup remains available, if supported.
4. Record rollback path for code, prompt/graph, and model/config versions.
5. Use non-production or local mode only for the drill.
6. Do not publish containers or modify cloud infrastructure unless that pipeline already exists and is approved for non-production.

### Tests

- Run the project test command after confirming it exists.
- Add tests for write kill switch, capability disablement, and fallback to synthetic fixtures.
- Run GTS-GATE after toggling write disablement.
- Run one smoke demo after restoring normal non-production config.

### Validations

- Confirm kill switches fail closed.
- Confirm no live user/channel/project is affected by the drill.
- Run `rg -n "LIVE_INTEGRATIONS_ENABLED.*true|write.*enabled|production" <changed_paths>` and correct unsafe defaults.

### Outcomes

- Rollback/kill-switch evidence is ready for G6 review.
- Write disablement and capability fallback are proven in non-production/local mode.
- Immediately run QA prompt [Q6D].

---

## ✅ [Q6D] QA review for rollback and kill-switch drill

Status: ✅ Done  
Deliverable path: `docs/development/rollback-kill-switch-drill.md`  
Result: QA verified local drill scope, fail-closed controls, no live users/channels/projects/repos affected, and rollback paths cite current artifacts only.  
Test evidence: F6 QA returned OK for P6D/Q6D.

### Purpose

Review [P6D] for fail-closed kill switches, safe drill scope, and rollback evidence.

### Execution prompt

Review rollback/kill-switch changes and drill records. Make focused fixes for unsafe toggles or incomplete proof.

### Tests

- Re-run kill-switch tests.
- Re-run GTS-GATE with write disablement.
- Re-run smoke demo after config restoration.

### Validations

- Confirm drill did not touch live users, channels, projects, or repos.
- Confirm write tools remain disabled when kill switch is active.
- Confirm rollback paths cite existing versions/artifacts only.

### Outcomes

- Recommend rollback/kill-switch readiness for RAJA pilot authorization review, or record blockers.
- Return tracking handoff for the coordinator to update [P6D] and [Q6D].

---

## ✅ [P6E] G6 authorization package only

Status: ✅ Done  
Deliverable path: `docs/development/g6-authorization-package.md`  
Result: Created G6 authorization package showing readiness inputs and explicitly blocking live pilot execution pending external non-agent-controlled RAJA approval and missing review artifacts.  
Test evidence: `make check` passed; F6 QA verified no live enablement and missing approval/review evidence is recorded.

### Purpose

Prepare the G6 authorization gate package. This prompt does not run the live pilot and must not enable live use.

### Execution prompt

Create the G6 pilot authorization package. Stop if G5 candidate evidence, rollback/kill-switch evidence, required review evidence, or external non-agent-controlled RAJA approval evidence is missing. Do not enable live use, change live configuration, or run a pilot in this prompt.

Gate package must include:

1. G5 candidate evidence and release notes with harness run ID.
2. Security/privacy/classification approval for pilot data.
3. Tool-owner approvals for exact Jira project, repo, Teams channel, Confluence space, calendar scope, and any write permissions.
4. Teams tenant/app/channel approval.
5. Support/RACI sign-off under RAJA accountability.
6. Rollback/kill-switch drill evidence.
7. Limited pilot scope, start/stop criteria, monitoring plan, and feedback capture.
8. External approval artifact reference from a non-agent-controlled source; repo text may reference it but must not create it.

The package may state that live execution is ready for RAJA review only if every required approval artifact is present. It must not say that live use has started.

### Tests

- Run the project test command and all MVP eval sets before preparing a readiness recommendation.
- Run GTS-GATE and confirm BA-EM-005 remains zero.
- Run a pre-pilot smoke test in sandbox/validated mode.
- Confirm rollback/kill-switch drill evidence is attached or referenced.

### Validations

- Confirm the package requires explicit RAJA approval from a non-agent-controlled source before live pilot execution.
- Confirm no live pilot execution or live enablement occurs in this prompt.
- Confirm unapproved writes remain blocked.
- Confirm no Phase 2 capability is exposed.
- Run `rg -n "all projects|all repos|all channels|Phase 2 enabled|write bypass" <changed_paths>` and correct unsafe wording.

### Outcomes

- G6 authorization package is ready for RAJA review, or blockers are recorded without enabling live use.
- If explicit non-agent-controlled RAJA approval is recorded, proceed to [P6F] for live pilot execution.
- Immediately run QA prompt [Q6E].

---

## ✅ [Q6E] QA review for G6 authorization package

Status: ✅ Done  
Deliverable path: `docs/development/g6-authorization-package.md`  
Result: QA verified authorization package completeness for readiness review and confirmed pilot execution remains blocked without external non-agent-controlled RAJA approval.  
Test evidence: F6 QA returned OK to mark P6E/Q6E complete as blocked/readiness-only.

### Purpose

Review [P6E] for authorization-package completeness, least-privilege scope, and no premature live enablement.

### Execution prompt

Review the G6 authorization package. If explicit non-agent-controlled RAJA approval is missing or incomplete, ensure no live enablement occurred and record blockers.

### Tests

- Re-run pre-pilot tests/evals cited in the gate package where possible.
- Re-run GTS-GATE after any pilot configuration change.
- Review rollback/kill-switch evidence.

### Validations

- Confirm required review evidence is present or explicitly blocked.
- Confirm pilot scope is narrow and names approved projects/repos/channels/spaces/calendars only as `[RAJA]` until recorded.
- Confirm no live run, unauthorized read/write, or Phase 2 capability exposure occurred.

### Outcomes

- Recommend the authorization package for RAJA/G6 review, or record blockers.
- If explicit non-agent-controlled RAJA approval is recorded, [P6F] may execute the limited pilot. Otherwise stop.
- Return tracking handoff for the coordinator to update [P6E] and [Q6E].

---

## ❌ [P6F] Limited live pilot execution after explicit approval

Status: ❌ Blocked  
Deliverable path: `docs/development/pilot-execution-blocked.md`  
Result: Live pilot execution did not run because no external non-agent-controlled RAJA approval artifact exists for exact pilot scope.  
Test evidence: F6 QA verified no live enablement occurred; security review found no blockers.

### Purpose

Execute the limited live pilot only after explicit recorded RAJA approval from the G6 authorization package.

### Execution prompt

Stop if [Q6E] is not complete or if explicit RAJA approval for the exact pilot scope is missing from an external, non-agent-controlled approval artifact. Do not infer approval from readiness language or repo text.

If approval exists, run the limited live pilot according to the runbook:

1. Enable only the approved scopes, channels, tools, and capabilities.
2. Keep all unapproved writes blocked.
3. Keep Phase 2 capabilities unavailable.
4. Capture `trace_id`, audit records, harness run ID, release version, user feedback, and any incidents.
5. Stop immediately if a write bypass, data exposure, unapproved scope, unsupported live tool, or Phase 2 exposure occurs.
6. Restore safe defaults after the pilot window if the runbook requires it.

### Tests

- Before live execution, re-run the project test command, MVP eval sets, and GTS-GATE.
- During execution, sample approved interactions and verify audit/trace records exist.
- After execution or stop, re-run GTS-GATE and any smoke test affected by configuration.

### Validations

- Confirm explicit RAJA approval from the non-agent-controlled artifact matches the exact live scope used.
- Confirm no broad access such as all projects, all repos, or all channels is enabled.
- Confirm unapproved writes remain blocked and BA-EM-005 remains zero.
- Confirm no Phase 2 capability is exposed and BA-EM-009 remains zero.
- Review search hits from `rg -n "all projects|all repos|all channels|Phase 2 enabled|write bypass" <changed_paths>` rather than assuming any hit is a failure.

### Outcomes

- Limited pilot execution evidence is recorded, or execution is blocked without enabling live use.
- Pilot evidence is ready for post-pilot assessment.
- Immediately run QA prompt [Q6F].

---

## ❌ [Q6F] QA review for limited live pilot execution

Status: ❌ Blocked  
Deliverable path: `docs/development/pilot-execution-blocked.md`  
Result: QA for live pilot execution is blocked because P6F did not run and no live pilot evidence exists.  
Test evidence: F6 QA verified P6F remains blocked by missing external RAJA approval artifact.

### Purpose

Review [P6F] for exact-scope execution, trace/audit evidence, hard-gate preservation, and safe shutdown.

### Execution prompt

Review the live pilot execution evidence. If explicit approval was missing, verify that [P6F] stopped without enabling live use. Apply only documentation/evidence fixes unless a safety defect requires immediate rollback or kill-switch action.

### Tests

- Re-run GTS-GATE after pilot configuration changes.
- Re-run the project test command.
- Review sampled pilot trace IDs and audit records.

### Validations

- Confirm the executed scope exactly matches explicit RAJA approval from the non-agent-controlled artifact.
- Confirm no unauthorized read/write, broad scope, or Phase 2 capability exposure occurred.
- Confirm rollback/kill-switch behavior remains available after the pilot.

### Outcomes

- Recommend pilot execution evidence for post-pilot assessment, or record authorization/execution blockers.
- Return tracking handoff for the coordinator to update [P6F] and [Q6F].

---

## ❌ [P6G] Post-pilot assessment

Status: ❌ Blocked  
Deliverable path: `docs/development/pilot-execution-blocked.md`  
Result: Post-pilot assessment did not run because no pilot was executed.  
Test evidence: F6 QA verified P6G should remain blocked until a pilot runs.

### Purpose

Assess the MVP pilot and recommend continue, adjust, or stop without turning pilot findings into production approval.

### Execution prompt

Create a post-pilot assessment.

Include:

1. Pilot scope actually used.
2. Harness run IDs and release versions.
3. Usage summary with no fabricated metrics; owner-dependent targets remain `[RAJA]`.
4. Sampled output review: evidence correctness, data-quality honesty, advisory/draft labeling, traceability.
5. Incident/support summary with trace IDs and audit refs.
6. Gate bypass check: BA-EM-005 must remain zero.
7. Phase separation check: BA-EM-009 must remain zero.
8. User feedback and open issues.
9. Recommendation: continue, adjust, or stop, with rationale and required next decisions.
10. Explicit statement that production rollout requires a separate authorization path.

### Tests

- Run post-pilot regression evals using the release candidate configuration.
- Run the project test command after confirming it exists.
- Re-run GTS-GATE and Phase-separation checks.

### Validations

- Confirm no pilot metric is fabricated or overstated.
- Confirm no production readiness claim is made without separate approval.
- Confirm all incidents or complaints are traceable through `trace_id` and audit evidence.

### Outcomes

- Post-pilot assessment is ready for RAJA decision.
- Phase 7 readiness inputs and MVP follow-up blockers are clear.
- Immediately run QA prompt [Q6G].

---

## ❌ [Q6G] QA review for post-pilot assessment

Status: ❌ Blocked  
Deliverable path: `docs/development/pilot-execution-blocked.md`  
Result: QA for post-pilot assessment is blocked because P6G did not run and no pilot assessment exists.  
Test evidence: F6 QA verified Q6G should remain blocked until post-pilot evidence exists.

### Purpose

Review [P6G] for truthful pilot findings, gate evidence, traceability, and no production overclaiming.

### Execution prompt

Review the post-pilot assessment. Correct unsupported metrics, missing evidence, or premature rollout claims.

### Tests

- Re-run post-pilot evals cited in the assessment.
- Re-run GTS-GATE and Phase-separation checks.
- Spot-check sampled outputs against trace/audit evidence.

### Validations

- Confirm recommendation is evidence-based and bounded.
- Confirm production rollout is not implied by pilot completion.
- Confirm Phase 2 readiness inputs are planning-only.

### Outcomes

- Recommend post-pilot assessment readiness for RAJA review before Phase 7 readiness work, or record blockers.
- Return tracking handoff for the coordinator to update [P6G] and [Q6G].

---

## ✅ [P7A] Phase 2 prioritization brief

Status: ✅ Done  
Deliverable path: `docs/development/phase-2-prioritization-brief.md`  
Result: Created Phase 2 prioritization brief covering all BA-P2-FR-001 through BA-P2-FR-016, recommended first slice [RAJA], dependencies, risks, data/tool approvals, and planning-only boundary.  
Test evidence: Manual BA-P2-FR-001 through BA-P2-FR-016 mapping; Q7A review passed after adding stable project context memory coverage.

### Purpose

Prepare a Phase 2 prioritization brief without implementing Enterprise BA capabilities.

### Execution prompt

Create or update a Phase 2 prioritization brief. Stop if [Q6G] is not complete or if RAJA has not accepted the post-pilot assessment as an input to Phase 2 readiness.

Include:

1. Candidate Phase 2 capabilities: requirement discovery, business/functional requirements, user stories, acceptance criteria, process mapping, gap analysis, stakeholder questions, impact analysis, traceability, BRD/FRD/PRD drafts, and test scenario inputs.
2. Inputs from post-pilot findings and unresolved BA-OQ-012/BA-OQ-013.
3. Recommended first Phase 2 capability set as `[RAJA]` until RAJA confirms.
4. Dependencies, risks, data classifications, tool approvals, and human review lanes.
5. Explicit statement that this is planning/readiness only and no Phase 2 build is authorized by the brief.

### Tests

- No executable tests are expected unless planning tooling is added.
- Manually cross-check candidate capabilities against Phase 2 requirements BA-P2-FR-001 through BA-P2-FR-016.
- Manually cross-check MVP/Phase 2 separation against BA-QG-008.

### Validations

- Confirm no Phase 2 code, prompts, or runtime behavior is implemented.
- Confirm prioritization values are `[RAJA]` unless explicitly decided.
- Run `rg -n "Phase 2 enabled|BRD generator implemented|user story generator implemented" <changed_paths>` and correct overclaims.

### Outcomes

- Phase 2 prioritization brief is ready for tool/data/eval planning.
- Phase 2 remains unbuilt pending G7 and separate plan approval.
- Immediately run QA prompt [Q7A].

---

## ✅ [Q7A] QA review for Phase 2 prioritization brief

Status: ✅ Done  
Deliverable path: `docs/development/phase-2-prioritization-brief.md`  
Result: QA verified Phase 2 scope accuracy, BA-P2-FR-014 coverage, [RAJA] priorities, planning/readiness-only language, no runtime implementation, and no HLD overclaim.  
Test evidence: `aara-business-analyst` Q7A recheck returned OK to mark P7A/Q7A complete.

### Purpose

Review [P7A] for Phase 2 scope accuracy, prioritization honesty, and no implementation drift.

### Execution prompt

Review the Phase 2 brief. Make focused fixes for missing capabilities, unsupported priorities, or implementation overclaims.

### Tests

- Re-run manual mapping to BA-P2-FR-001 through BA-P2-FR-016.
- Re-check BA-QG-008 separation language.

### Validations

- Confirm no Phase 2 runtime code or enabled prompt behavior was added.
- Confirm first capability set remains `[RAJA]` until decided.
- Confirm MVP backlog/release notes remain Phase 2-free.

### Outcomes

- Recommend prioritization brief as ready for tool approval matrix work, or record blockers.
- Return tracking handoff for the coordinator to update [P7A] and [Q7A].

---

## ✅ [P7B] Phase 2 tool approval matrix

Status: ✅ Done  
Deliverable path: `docs/development/phase-2-tool-approval-matrix.md`  
Result: Created Phase 2 tool approval matrix covering all required candidate integrations, data classes, read/write intent, owners/scopes as [RAJA], validation status, blockers, and blocked defaults.  
Test evidence: Manual cross-check against BA-P2-FR-015 and BA-DEP-009; QA confirmed all candidate tools are present and blocked by default.

### Purpose

Create a tool approval matrix for candidate Phase 2 integrations without enabling any unapproved integration.

### Execution prompt

Create or update a Phase 2 tool approval matrix.

Include:

1. Candidate tools from the requirements: Jira, Confluence, GitHub, Azure DevOps, SharePoint, Teams, Miro/Draw.io, SQL/Data, ServiceNow, and test-management tools.
2. Intended capability use, data classes, read/write intent, owner, security/privacy review, scopes, validation status, and blockers.
3. Default action for every tool: blocked until owner/security/platform approval and actual schema validation.
4. Write policy: draft-only or human-gated; no autonomous writes.
5. Relationship to MVP tools without assuming MVP approvals transfer to Phase 2.

### Tests

- No executable tests are expected unless matrix validation tooling is added.
- If tooling is added, run the project test command after confirming it exists.
- Manually cross-check tools against BA-P2-FR-015 and BA-DEP-009.

### Validations

- Confirm no tool row is marked enabled without approval evidence.
- Confirm no credentials, endpoints, or real scopes are committed.
- Run `rg -n "enabled|approved|write" <changed_paths>` and verify each hit has evidence or is blocked/gated.

### Outcomes

- Phase 2 tool approval matrix is ready for data/classification planning.
- All candidate integrations remain blocked by default.
- Immediately run QA prompt [Q7B].

---

## ✅ [Q7B] QA review for Phase 2 tool matrix

Status: ✅ Done  
Deliverable path: `docs/development/phase-2-tool-approval-matrix.md`  
Result: QA verified candidate-tool completeness, blocked defaults, draft/human-gated write policy, no MVP approval reuse, and no credential/endpoint/scope leakage.  
Test evidence: `aara-business-analyst` QA returned OK to mark P7B/Q7B complete.

### Purpose

Review [P7B] for candidate-tool completeness, blocked defaults, and no approval leakage.

### Execution prompt

Review the tool matrix. Fix missing tools, unsafe enablement, or unclear ownership.

### Tests

- Re-run manual cross-check against BA-P2-FR-015 and BA-DEP-009.
- Re-run tooling tests if any were created.

### Validations

- Confirm every tool defaults to blocked until approved/validated.
- Confirm write intent is draft-only or human-gated.
- Confirm MVP approvals are not reused without review.

### Outcomes

- Recommend tool matrix as ready for data/classification planning, or record blockers.
- Return tracking handoff for the coordinator to update [P7B] and [Q7B].

---

## ✅ [P7C] Phase 2 data and classification plan

Status: ✅ Done  
Deliverable path: `docs/development/phase-2-data-classification-plan.md`  
Result: Created Phase 2 data/classification plan covering candidate inputs/outputs, DSPC rules, minimization/redaction, retention/residency [RAJA], evidence metadata, and human review lanes.  
Test evidence: Manual cross-check against BA-DSPC-001 through BA-DSPC-007 and BA-OQ-010/014; QA found no blockers.

### Purpose

Prepare a Phase 2 data/classification plan for Enterprise BA inputs and artifacts before any non-synthetic data is processed.

### Execution prompt

Create or update a Phase 2 data/classification plan.

Include:

1. Candidate input categories: meeting notes, business emails, customer requests, product ideas, process pain points, support tickets, regulatory changes, source documents, and approved tool data.
2. Output categories: requirements, user stories, acceptance criteria, process maps, gap/impact analysis, traceability, BRD/FRD/PRD drafts, and test scenario inputs.
3. Classification handling rules to be confirmed by security/privacy owner as `[RAJA]`.
4. Data minimization, prompt/input redaction, retention/audit expectations, and evidence/source metadata expectations.
5. Rules that restricted, internal, source-code, or security-sensitive data must not enter prompts/logs/evals until approved handling is recorded.
6. Human review lanes for compliance/security/privacy findings.

### Tests

- No executable tests are expected unless validation tooling is added.
- If tooling is added, run the project test command after confirming it exists.
- Manually cross-check the plan against BA-DSPC-001 through BA-DSPC-007 and BA-OQ-010/014.

### Validations

- Confirm no real restricted/internal/source-code data is added as examples.
- Confirm retention/residency values are `[RAJA]` unless decided.
- Run `rg -n "password|secret|token|restricted sample|source code sample" <changed_paths>` and remove unsafe content.

### Outcomes

- Phase 2 data/classification plan is ready for evaluation approach work.
- Non-synthetic Phase 2 data remains blocked pending approval.
- Immediately run QA prompt [Q7C].

---

## ✅ [Q7C] QA review for Phase 2 data/classification plan

Status: ✅ Done  
Deliverable path: `docs/development/phase-2-data-classification-plan.md`  
Result: QA verified data-classification completeness, safe examples, non-synthetic data blocking, [RAJA] retention/residency, and no sensitive sample data.  
Test evidence: `aara-business-analyst` QA returned OK to mark P7C/Q7C complete; added source-owner metadata note.

### Purpose

Review [P7C] for data-classification completeness, safe examples, and non-synthetic data blocking.

### Execution prompt

Review data/classification plan. Correct unsafe examples, missing controls, or unsupported retention claims.

### Tests

- Re-run manual cross-check against BA-DSPC requirements and BA-OQ-010/014.
- Re-run tooling tests if created.

### Validations

- Confirm no sensitive sample data is committed.
- Confirm all classification and retention decisions are owner-routed.
- Confirm Phase 2 evals remain synthetic until handling is approved.

### Outcomes

- Recommend data/classification plan as ready for GTS-P2-REQ approach, or record blockers.
- Return tracking handoff for the coordinator to update [P7C] and [Q7C].

---

## ✅ [P7D] GTS-P2-REQ evaluation approach

Status: ✅ Done  
Deliverable path: `docs/development/gts-p2-req-evaluation-approach.md`  
Result: Created synthetic GTS-P2-REQ evaluation approach with case format, expected outputs, review rubric, BA-EM mapping, hard BA-EM-009 boundary, and planning-only/no-runtime scope.  
Test evidence: Manual validation against evaluation harness, prioritization brief, and data/classification plan; QA passed after adding fixture data, expected routing, labeled ground truth, trace/artifact metadata, and compliance/legal review lane.

### Purpose

Design the Phase 2 GTS-P2-REQ evaluation approach for requirement discovery readiness without building Phase 2 capability runtime.

### Execution prompt

Create or update a GTS-P2-REQ evaluation approach.

Include:

1. Synthetic-only case format for rough business inputs, meeting notes, tickets, conflicting stakeholder statements, and missing business rules.
2. Expected output characteristics: fact/assumption separation, conflict surfacing, open questions, risks/dependencies, evidence refs, trace discipline, and draft-only labeling.
3. Human review rubric for BA SME, Product Owner, QA, security/privacy, and architect lanes.
4. BA-EM metric mapping, including BA-EM-009 to prevent Phase 2 leakage into MVP.
5. Owner-threshold metrics as measured/no-threshold until RAJA sets values.
6. No live enterprise tools, no non-synthetic data, and no Phase 2 runtime implementation.

### Tests

- No runtime tests are required unless eval-case validation tooling is added.
- If tooling is added, run the project test command after confirming it exists.
- Add or manually validate sample synthetic GTS-P2-REQ cases for conflict, missing rule, and traceability coverage.

### Validations

- Confirm sample cases are synthetic and contain no real business data.
- Confirm approach does not create runnable Phase 2 generation behavior.
- Run `rg -n "threshold.*[0-9]|real customer|production data|Phase 2 enabled" <changed_paths>` and correct unsafe content.

### Outcomes

- GTS-P2-REQ evaluation approach is ready for separate Phase 2 plan review.
- Phase 2 remains planning-only.
- Immediately run QA prompt [Q7D].

---

## ✅ [Q7D] QA review for GTS-P2-REQ approach

Status: ✅ Done  
Deliverable path: `docs/development/gts-p2-req-evaluation-approach.md`  
Result: QA verified synthetic eval discipline, fixture/mock response fields, expected routing, expected evidence refs, labeled ground truth, trace/version metadata, compliance/legal lane, [RAJA] thresholds, and no runtime implementation.  
Test evidence: `aara-ai-evaluation-engineer` QA recheck returned OK to mark P7D/Q7D complete.

### Purpose

Review [P7D] for synthetic eval discipline, rubric completeness, metric honesty, and no Phase 2 runtime build.

### Execution prompt

Review GTS-P2-REQ approach. Make focused fixes for unsafe data, missing review criteria, or hidden implementation.

### Tests

- Re-run eval-case validation tooling if created.
- Manually validate synthetic sample coverage for conflict, missing rule, and traceability.

### Validations

- Confirm no real business, customer, or restricted data appears.
- Confirm no owner threshold is fabricated.
- Confirm Phase 2 runtime behavior remains unimplemented.

### Outcomes

- Recommend GTS-P2-REQ approach as ready for G7 readiness review, or record blockers.
- Return tracking handoff for the coordinator to update [P7D] and [Q7D].

---

## ✅ [P7E] Separate Phase 2 plan readiness review

Status: ✅ Done  
Deliverable path: `docs/development/g7-readiness-review.md`  
Result: Created G7 readiness package summarizing Phase 2 readiness artifacts and required separate Phase 2 implementation plan sections, with explicit RAJA decisions before build.  
Test evidence: Manual cross-check against Phase 2 requirements, BA-QG-008, GTS-P2-REQ approach, and readiness artifacts; QA found no blockers.

### Purpose

Package G7 Phase 2 readiness evidence and define what a separate Phase 2 plan must contain before Enterprise BA capability build can start.

### Execution prompt

Create or update the G7 readiness review.

Include:

1. Phase 2 prioritization brief status.
2. Tool approval matrix status.
3. Data/classification plan status.
4. GTS-P2-REQ evaluation approach status.
5. Required separate Phase 2 plan sections: scope, gates, architecture changes, tool validation, data handling, evaluation, support/RACI, rollout, and rollback.
6. Explicit RAJA decision needed: first Phase 2 capability set and approval to create a separate plan.
7. Explicit statement that G7 readiness does not authorize implementation until the separate plan is approved.
8. Carry-forward MVP guardrails for Teams/Copilot 365, human-gated writes, evidence discipline, Azure-primary stack, and blocked unvalidated tools.

### Tests

- No executable tests are expected unless readiness tooling is added.
- If tooling is added, run the project test command after confirming it exists.
- Manually cross-check G7 package against Phase 2 requirements, BA-QG-008, and GTS-P2-REQ spec.

### Validations

- Confirm no Phase 2 implementation prompt is executed as part of G7 readiness.
- Confirm separate-plan approval is required before build.
- Run `rg -n "Phase 2 build approved|implementation started|enabled" <changed_paths>` and correct overclaims.

### Outcomes

- G7 readiness package is ready for RAJA decision.
- Separate Phase 2 plan requirements are explicit.
- Immediately run QA prompt [Q7E].

---

## ✅ [Q7E] QA review for G7 readiness

Status: ✅ Done  
Deliverable path: `docs/development/g7-readiness-review.md`  
Result: QA verified complete G7 readiness evidence, separate-plan requirement, carry-forward guardrails, RAJA decisions, and no unauthorized Phase 2 build.  
Test evidence: `aara-project-planner` QA returned OK to mark P7E/Q7E complete.

### Purpose

Review [P7E] for complete G7 readiness evidence, clear separate-plan requirement, and no unauthorized Phase 2 build.

### Execution prompt

Review the G7 readiness package. Make focused corrections for missing evidence, approval ambiguity, or implementation drift.

### Tests

- Re-run any readiness-tool tests.
- Re-run manual cross-check against Phase 2 requirements, BA-QG-008, and GTS-P2-REQ.

### Validations

- Confirm RAJA must approve a separate Phase 2 plan before implementation.
- Confirm candidate tools/data remain blocked pending approval.
- Confirm MVP guardrails carry forward.

### Outcomes

- The all-phase execution prompt pack ends with a clear G7 decision package.
- Return tracking handoff for the coordinator to update [P7E] and [Q7E].

---

## ✅ [P8A] Phase 2 `P2-G1` technical baseline/scaffold

Status: ✅ Complete  
Deliverable path: `docs/development/p2-g1-technical-baseline.md`  
Result: P2-G1 technical baseline created (v0.1). Defines route isolation (`phase2_requirement_discovery`), scaffold structure (`src/ba_agent/phase2/` + `tests/phase2/`), output contract (Section 4), project-context memory schema (Section 5), gateway/control carry-forward (Section 6), synthetic-only discipline (Section 7), minimum test expectations (Section 8), P2-G1 exit criteria checklist (Section 9), and P2-DEC-013 delta review artifact. All exit criteria items are defined and evidenced; scaffold stub implementation completed by Q8A.  
Test evidence: Manual cross-check against P2-G1 exit criteria complete (Section 9 all 7 items evidenced in document). No existing MVP files modified. Stub creation and `make test` (93 passed, 0 failed) confirmed at Q8A.

### Purpose

Start Phase 2 first-slice execution under the accepted `P2-G0` baseline by defining the technical scaffold and route isolation controls for requirement discovery.

### Execution prompt

Create or update `docs/development/p2-g1-technical-baseline.md` and related scaffold notes. Confirm:

1. Phase 2 route/scaffold boundaries are isolated from MVP routes.
2. Synthetic-only fixture paths are defined.
3. Output contract and project-context memory schema references are linked to `docs/planning/phase-2-implementation-plan.md`.
4. No live clients, credentials, or write-like side effects are enabled.
5. Required `P2-DEC-*` dependencies for `P2-G1` are explicit.

### Tests

- Run only existing project checks if scaffold/code changes are introduced.
- If this is docs-only, perform manual cross-check against `P2-G1` exit criteria.

### Validations

- Confirm no live integration enablement language appears.
- Confirm MVP route behavior is unchanged.

### Outcomes

- `P2-G1` readiness evidence is documented.
- Immediately run QA prompt [Q8A].

---

## ✅ [Q8A] QA review for `P2-G1` technical baseline/scaffold

Status: ✅ Complete  
Deliverable path: `docs/development/p2-g1-technical-baseline.md`  
Result: Conditional Pass. Document is structurally complete and all 7 exit criteria are evidenced. One soft finding: Section 9 exit criterion checkboxes remain in "pending" state (by design — awaiting RAJA Closed sign-off on P2-DEC-013). All hard gates confirmed: BA-EM-005 = 0 and BA-EM-009 = 0 (no write-like side effects, no MVP/Phase 2 leakage). P2-DEC-013 updated to Conditional/partial in decision log.  
Test evidence: `make test` — 93 passed, 0 failed. 38 new Phase 2 tests (test_discovery.py + test_separation.py) all pass. All 55 MVP tests pass (BA-EM-009 baseline unaffected). Stub files created: `src/ba_agent/phase2/__init__.py`, `router.py`, `discovery.py`, `models.py`, `context_memory.py`, `traceability.py`; `tests/phase2/__init__.py`, `fixtures/.gitkeep`, `fixtures/P2REQ-STUB-001.json`, `test_discovery.py`, `test_separation.py`.

### Purpose

Review [P8A] for route isolation, synthetic-only discipline, and no-live/no-write conformance.

### Execution prompt

Review the `P2-G1` baseline artifact and make focused corrections only. Verify `P2-G1` exit criteria are fully evidenced.

### Tests

- Re-run any existing checks introduced by [P8A].
- Re-run manual checklist comparison to `P2-G1` criteria.

### Validations

- Confirm no live tool/data path is enabled.
- Confirm MVP/Phase 2 separation statements are explicit.

### Outcomes

- `P2-G1` recommendation is ready (or blocker is explicit).
- Return tracking handoff for coordinator update.

---

## ✅ [P8B] Phase 2 `P2-G2` synthetic requirement-discovery thin slice

Status: ✅ Complete  
Deliverable path: `docs/development/p2-g2-synthetic-thin-slice.md`  
Result: Completed synthetic thin-slice implementation and QA evidence package.  
Test evidence: `PYTHONPATH=src python3 -m pytest tests/phase2/ -q` passed (44 passed); `make test` passed (99 passed).

### Purpose

Execute the first synthetic requirement-discovery thin slice and produce draft/advisory outputs aligned to Section 6 of the Phase 2 implementation plan.

### Execution prompt

Implement or document the `P2-G2` thin slice so that synthetic cases produce:

1. Facts/assumptions/`[inferred]`/open-question separation.
2. Risk/dependency/conflict surfacing.
3. Traceability skeleton outputs.
4. Draft/advisory non-approval labeling.
5. No BRD/FRD/PRD/process-map/HLD generation.

### Tests

- Run targeted existing tests/checks for thin-slice behavior.
- Execute synthetic case checks for minimum P2REQ coverage where available.

### Validations

- Confirm no non-synthetic input is used.
- Confirm output contract sections are present and separated.

### Outcomes

- `P2-G2` evidence package is produced.
- Immediately run QA prompt [Q8B].

---

## ✅ [Q8B] QA review for `P2-G2` synthetic thin slice

Status: ✅ Complete  
Deliverable path: `docs/development/p2-g2-synthetic-thin-slice.md`  
Result: Conditional Pass — the thin slice is coherent and all tests pass, but the synthetic guard accepts arbitrary text that merely contains “synthetic,” `build_trace_skeleton` can duplicate `p2-input` IDs when input candidates are supplied, and `P2-DEC-013` still needs a close action in the decision log.  
Test evidence: `PYTHONPATH=src python3 -m pytest tests/phase2/ -q` passed (44 passed); `make test` passed (99 passed).

### Purpose

Review [P8B] for first-slice scope discipline, output-structure correctness, and evidence integrity.

### Execution prompt

Review the `P2-G2` artifact and correct only misalignments with first-slice scope and output contract requirements.

### Tests

- Re-run relevant synthetic-case checks and structural validations.

### Validations

- Confirm acceptance-criteria/test-case generation was not introduced in first slice.
- Confirm all outputs are draft/advisory and trace-linked.

### Outcomes

- `P2-G2` recommendation is ready (or blocker is explicit).
- Return tracking handoff for coordinator update.

---

## ✅ [P8C] Phase 2 `P2-G3` evaluation/control hardening

Status: ✅ Done  
Deliverable path: `docs/development/p2-g3-evaluation-control-hardening.md`  
Result: Delivered `P2-G3` evidence package with BA-EM mapping, explicit hard-gate zero evidence (`BA-EM-005`, `BA-EM-009`), regression coverage for conflict/missing-rule/traceability, and unsupported-claim review/routing method.  
Test evidence: `PYTHONPATH=src python3 -m pytest tests/phase2/test_thin_slice.py tests/phase2/test_discovery.py tests/phase2/test_separation.py tests/test_router.py tests/test_evaluation.py -q` (65 passed); `PYTHONPATH=src python3 -m ba_agent eval GTS-GATE` (approval_gate_bypass_count=0); `PYTHONPATH=src python3 -m ba_agent eval GTS-ROUTER` (phase_separation_violations=0).

### Purpose

Harden evaluation/control coverage for the Phase 2 first slice with explicit BA-EM hard-gate evidence.

### Execution prompt

Produce/update `P2-G3` evidence:

1. BA-EM mapping and result capture for first-slice synthetic cases.
2. Hard-gate evidence for BA-EM-005 = 0 and BA-EM-009 = 0.
3. Regression checks for conflict/missing-rule/traceability behaviors.
4. Unsupported-claim review method and findings routing.

### Tests

- Run existing targeted eval/test commands for Phase 2 first-slice behaviors.

### Validations

- Confirm hard-gate evidence is explicit and reproducible.
- Confirm owner thresholds remain `[RAJA]` unless explicitly set.

### Outcomes

- `P2-G3` package is ready for decision review.
- Immediately run QA prompt [Q8C].

---

## ✅ [Q8C] QA review for `P2-G3` evaluation/control hardening

Status: ✅ Done  
Deliverable path: `docs/development/p2-g3-evaluation-control-hardening.md`  
Result: QA confirmed hard-gate reporting integrity, reproducible command evidence, and no live/non-synthetic authorization drift in the `P2-G3` package.  
Test evidence: Re-ran targeted `P2-G3` checks: `PYTHONPATH=src python3 -m pytest tests/phase2/test_thin_slice.py tests/phase2/test_discovery.py tests/phase2/test_separation.py tests/test_router.py tests/test_evaluation.py -q` (65 passed), `PYTHONPATH=src python3 -m ba_agent eval GTS-GATE`, and `PYTHONPATH=src python3 -m ba_agent eval GTS-ROUTER`.

### Purpose

Review [P8C] for hard-gate correctness and metric/reporting integrity.

### Execution prompt

Review `P2-G3` outputs and make focused corrections for metric drift, missing hard-gate evidence, or overclaiming.

### Tests

- Re-run targeted checks used to produce BA-EM evidence.

### Validations

- Confirm BA-EM-005 and BA-EM-009 remain hard-gate zero.
- Confirm no live/non-synthetic behavior was introduced.

### Outcomes

- `P2-G3` recommendation is ready (or blocker is explicit).
- Return tracking handoff for coordinator update.

---

## ✅ [P8D] Phase 2 `P2-G4` tool/data readiness decision package

Status: ✅ Done  
Deliverable path: `docs/development/p2-g4-tool-data-readiness.md`  
Result: QA confirmed `P2-G4` keeps hard non-authorization boundaries, blocked-by-default tool/data posture, and `[RAJA]` owner-decision markers without enabling live/sandbox/production behavior.  
Test evidence: `PYTHONPATH=src python3 -m ba_agent validate-mcp --register docs/development/mcp-validation-register.json` => `validated: []`, `blocked: [get_sprint_status, get_recent_activity, send_adaptive_card]`; `rg -n -i "Slack|Azure ACR|acr\\.azurecr\\.io" docs/development/p2-g4-tool-data-readiness.md docs/planning/phase-2-implementation-plan.md docs/planning/decision-log.md docs/development/phase-2-tool-approval-matrix.md docs/development/phase-2-data-classification-plan.md` => only boundary reminder hit in `p2-g4-tool-data-readiness.md` (`no Slack channel expansion`, `not Azure ACR`), no `acr.azurecr.io`; `rg -n -i "authorize|authorization|sandbox|production|live" docs/development/p2-g4-tool-data-readiness.md` => explicit non-authorization/fail-closed boundary statements only.

### Purpose

Prepare the `P2-G4` decision package for tool/data readiness without enabling any tool or non-synthetic data path.

### Execution prompt

Create or update a `P2-G4` package that includes:

1. Tool matrix deltas and approval evidence status.
2. Classification/redaction/retention/residency decision state.
3. Blocked-default register updates.
4. Explicit statement of what remains blocked and why.

### Tests

- Run existing matrix/validation checks if present.
- Otherwise perform documented manual cross-checks.

### Validations

- Confirm no tool is enabled without required evidence.
- Confirm non-synthetic input remains blocked unless approved.

### Outcomes

- `P2-G4` decision package is ready for RAJA review.
- Immediately run QA prompt [Q8D].

---

## ✅ [Q8D] QA review for `P2-G4` tool/data readiness

Status: ✅ Done  
Deliverable path: `docs/development/p2-g4-tool-data-readiness.md`  
Result: QA review found no corrective edits required; approval-evidence gating remains explicit, default blocked behavior is intact, and write-like side effects remain fail-closed pending separate authorization `[RAJA]`.  
Test evidence: Re-ran [P8D] evidence checks with identical outcomes: validate-mcp blocked all listed tools (`validated: []`), Slack/ACR drift scan showed only explicit prohibition wording, and authorization-language scan confirmed no enablement language.

### Purpose

Review [P8D] for approval-evidence completeness and blocked-default integrity.

### Execution prompt

Review `P2-G4` artifacts and correct missing approval evidence links, ambiguous scope, or unsafe enablement language.

### Tests

- Re-run any existing checks used in [P8D].

### Validations

- Confirm default action remains blocked for unapproved tools/data.
- Confirm write-like side effects remain fail-closed.

### Outcomes

- `P2-G4` recommendation is ready (or blocker is explicit).
- Return tracking handoff for coordinator update.

---

## ✅ [P8E] Phase 2 `P2-G5` candidate review and stop decision package

Status: ✅ Done  
Deliverable path: `docs/development/p2-g5-candidate-review.md`  
Result: `P2-G5` candidate review package is decision-ready with explicit advisory-only continue/adjust/stop options, hard-gate traceability, and non-authorization boundaries that keep sandbox/live/production separately owner-authorized (`[RAJA]`).  
Test evidence: `PYTHONPATH=src python3 -m ba_agent eval GTS-GATE` (passed; `approval_gate_bypass_count=0`), `PYTHONPATH=src python3 -m ba_agent eval GTS-ROUTER` (passed; `phase_separation_violations=0`), `PYTHONPATH=src python3 -m ba_agent validate-mcp --register docs/development/mcp-validation-register.json` (`validated=[]`; blocked=`get_sprint_status,get_recent_activity,send_adaptive_card`), `rg "Continue|Adjust|Stop|advisory|RAJA|not authorize|does \\*\\*not\\*\\* authorize|separate owner-approved|P2-DEC-011|P2-DEC-014" docs/development/p2-g5-candidate-review.md -n` (confirms advisory + non-authorization wording), `rg "BA-EM-005|BA-EM-009|approval_gate_bypass_count|phase_separation_violations|GTS-GATE|GTS-ROUTER|validate-mcp" docs/development/p2-g5-candidate-review.md docs/planning -n` (confirms hard-gate/evidence traceability).

### Purpose

Package first-slice candidate evidence for `P2-G5` continue/adjust/stop decision without authorizing sandbox/live/production rollout.

### Execution prompt

Create/update `P2-G5` candidate review package with:

1. Scope delivered vs first-slice scope.
2. Eval/control summary and hard-gate outcomes.
3. Open risks/dependencies/decisions.
4. Rollback readiness and unresolved blockers.
5. Recommendation options: continue, adjust, or stop.

### Tests

- Re-run existing targeted checks that support the package findings.

### Validations

- Confirm no production authorization language is introduced.
- Confirm any sandbox path is explicitly separate and owner-approved only.

### Outcomes

- `P2-G5` package is ready for RAJA decision.
- Immediately run QA prompt [Q8E].

---

## ✅ [Q8E] QA review for `P2-G5` candidate review

Status: ✅ Done  
Deliverable path: `docs/development/p2-g5-candidate-review.md`  
Result: QA confirms decision clarity, evidence traceability to BA-EM hard gates, safe non-authorization boundaries, and explicit statement that any sandbox/live/production progression requires separate owner authorization (`[RAJA]`). No corrective edit required to the candidate package.  
Test evidence: `PYTHONPATH=src python3 -m ba_agent eval GTS-GATE` (passed; `approval_gate_bypass_count=0`), `PYTHONPATH=src python3 -m ba_agent eval GTS-ROUTER` (passed; `phase_separation_violations=0`), `PYTHONPATH=src python3 -m ba_agent validate-mcp --register docs/development/mcp-validation-register.json` (`validated=[]`; blocked list only), `rg "\\[P8E\\]|\\[Q8E\\]|P2-G5" docs/development docs/planning` (confirms P2-G5 document linkage in development/planning set and tracking references).

### Purpose

Review [P8E] for decision clarity, evidence traceability, and safe non-authorization boundaries.

### Execution prompt

Review the `P2-G5` package and make focused corrections for missing evidence links, decision ambiguity, or overclaims.

### Tests

- Re-run any checks referenced in the package.
- Re-run manual cross-check against `docs/planning/phase-2-implementation-plan.md` Sections 9 through 14.

### Validations

- Confirm recommendations are advisory and owner-routed.
- Confirm stop condition remains: no sandbox/live/production path without separate explicit authorization.

### Outcomes

- Phase 8 package closes with a decision-grade `P2-G5` artifact and explicit next-step options for RAJA.
- Return tracking handoff for the coordinator to update [P8E] and [Q8E].

---

## ✅ [P9A] HLD scope-change plan and prompt/fleet synchronization

Status: ✅ Done  
Deliverable path: `docs/planning/phase-2-hld-creation-plan.md`; `docs/planning/decision-log.md`; `docs/planning/phase-2-implementation-plan.md`; `docs/planning/phase-2-traceability-matrix.md`; `prompts.md`; `fleet_prompt.md`  
Result: Created the `[F9]` HLD creation lane, recorded RAJA's HLD scope-change decision, and preserved repository-evidence-only, draft/advisory, non-authorizing boundaries for the future HLD.  
Test evidence: `PYTHONPATH=src python3 -m ba_agent validate-mcp` => `validated=[]` and all sandbox rows blocked; `PYTHONPATH=src python3 -m ba_agent eval GTS-GATE` passed with `approval_gate_bypass_count=0`; `PYTHONPATH=src python3 -m ba_agent eval GTS-ROUTER` passed with `phase_separation_violations=0`; changed-doc drift scan showed only explicit prohibition wording for blocked surface/registry terms and no registry endpoint.

### Purpose

Move HLD creation into an explicit Phase 2 follow-on lane without weakening sandbox/live/non-synthetic/write-safety controls.

### Execution prompt

Create/update the HLD scope-change planning package with:

1. RAJA decision trace for the HLD focus change.
2. Active `[F9]` prompt/fleet lane.
3. Allowed repository-only inputs and blocked non-synthetic/live inputs.
4. HLD gate definitions and validation expectations.
5. Traceability updates tying the HLD lane to decision and planning baselines.

### Tests

- Cross-check document-control versions and companion references.
- Scan changed docs for forbidden drift markers and terms.

### Validations

- Confirm no sandbox/live/non-synthetic/pilot/production path is authorized.
- Confirm HLD is draft/advisory and owner-routed.
- Confirm `[RAJA]` and `[inferred]` remain the only unsupported/owner-dependent markers.

### Outcomes

- `HLD-G0` setup is complete.
- Immediately run QA prompt [Q9A].

---

## ✅ [Q9A] QA review for HLD scope-change setup

Status: ✅ Done  
Deliverable path: `docs/planning/phase-2-hld-creation-plan.md`; `docs/planning/decision-log.md`; `docs/planning/phase-2-implementation-plan.md`; `docs/planning/phase-2-traceability-matrix.md`; `prompts.md`; `fleet_prompt.md`  
Result: QA confirmed the HLD lane is explicitly bounded as draft/advisory, repository-evidence-only, and non-authorizing; no sandbox/live/non-synthetic/write-like enablement was introduced.  
Test evidence: Manual cross-check against prior Phase 2 HLD exclusion language, `P2-DEC-016` sandbox blocks, and new `P2-DEC-017` scope-change wording; gate commands passed and changed-doc drift scan showed only explicit prohibition wording for blocked surface/registry terms and no registry endpoint.

### Purpose

Review [P9A] for scope-change correctness, prompt/fleet consistency, and control-boundary preservation.

### Execution prompt

Review the HLD setup artifacts and correct any missing decision trace, stale active-lane reference, ambiguous authorization language, or companion-document mismatch.

### Tests

- Re-run changed-document cross-checks from [P9A].
- Inspect `P2-DEC-016`, `P2-DEC-017`, and the HLD plan for contradictory authorization wording.

### Validations

- Confirm prior HLD exclusion is superseded only for the new HLD lane.
- Confirm [F9] does not authorize sandbox execution, live integrations, non-synthetic data, production, external publishing, autonomous approval, or write-like side effects.

### Outcomes

- [P9A]/[Q9A] close `HLD-G0`.
- Next implementation prompt is [P9B].

---

## ✅ [P9B] Draft BA Agent HLD

Status: ✅ Done  
Deliverable path: `docs/architecture/ba-agent-hld.md`  
Result: Created the draft/advisory BA Agent HLD from checked-in repository evidence only, preserving non-authorization boundaries and `[inferred]`/`[RAJA]` evidence discipline.  
Test evidence: `PYTHONPATH=src python3 -m ba_agent validate-mcp` => all sandbox rows blocked; `PYTHONPATH=src python3 -m ba_agent eval GTS-GATE` passed with `approval_gate_bypass_count=0`; `PYTHONPATH=src python3 -m ba_agent eval GTS-ROUTER` passed with `phase_separation_violations=0`; HLD drift scan showed no forbidden runtime surface/registry enablement or registry endpoint.

### Purpose

Create the draft/advisory BA Agent HLD from checked-in repository evidence only.

### Execution prompt

Create `docs/architecture/ba-agent-hld.md` with:

1. Purpose, audience, status, and explicit non-approval notice.
2. Source and decision traceability.
3. Scope, non-goals, and blocked paths.
4. Logical component architecture.
5. Runtime flow for local/synthetic execution.
6. Gateway/control-plane and approval-gate design.
7. Data classification, evidence discipline, and memory/traceability posture.
8. Evaluation and hard-gate strategy.
9. Observability/audit posture.
10. Deployment direction as proposal only, using `[RAJA]` where owner decisions are required.
11. Risks, assumptions, dependencies, and open decisions.

### Tests

- Run existing hard-gate evals when the HLD cites BA-EM-005 or BA-EM-009.
- Run `validate-mcp` if the HLD cites sandbox validation posture.
- Scan the HLD for forbidden drift terms and unsupported approval language.

### Validations

- Confirm every factual architecture claim is source-backed or marked `[inferred]`.
- Confirm owner-dependent values and approvals are marked `[RAJA]`.
- Confirm the HLD does not claim sandbox/live/pilot/production readiness.

### Outcomes

- Draft HLD is ready for QA.
- Immediately run QA prompt [Q9B].

---

## ✅ [Q9B] QA review for draft BA Agent HLD

Status: ✅ Done  
Deliverable path: `docs/architecture/ba-agent-hld.md`  
Result: QA confirmed the HLD is draft/advisory, source-traced to repository evidence, explicit about open `[RAJA]` decisions, and does not authorize sandbox/live/non-synthetic/write-like paths.  
Test evidence: Re-ran [P9B] gate checks; manually cross-checked HLD scope, non-goals, control-plane wording, hard-gate references, and open-decision routing against `phase-2-hld-creation-plan.md`, `decision-log.md`, runtime architecture, MCP contracts, evaluation harness, and current gateway/sandbox adapter code.

### Purpose

Review the HLD for architecture consistency, evidence discipline, and control-boundary safety.

### Execution prompt

Review `docs/architecture/ba-agent-hld.md` and make focused corrections for:

1. Missing source citations or unmarked inference.
2. Stale or contradictory decisions.
3. Unauthorized sandbox/live/non-synthetic/write-like claims.
4. Missing `[RAJA]` markers on owner-dependent decisions.
5. Architecture gaps that should be explicit open decisions rather than invented facts.

### Tests

- Re-run the checks used in [P9B].
- Cross-check against `docs/planning/phase-2-hld-creation-plan.md` and `docs/planning/decision-log.md`.

### Validations

- Confirm HLD remains draft/advisory.
- Confirm hard-gate and approval-gate wording is consistent with BA-EM-005 and BA-EM-009.

### Outcomes

- HLD is ready for owner-review packaging.
- Next implementation prompt is [P9C].

---

## ✅ [P9C] HLD owner-review package

Status: ✅ Done  
Deliverable path: `docs/development/phase-2-hld-review-package.md`  
Result: Created the HLD owner-review package with RAJA action options, persona-lens checklist, open `[RAJA]` decisions, risks/dependencies, evidence summary, and explicit non-approval boundaries.  
Test evidence: `PYTHONPATH=src python3 -m ba_agent validate-mcp` => all sandbox rows blocked; `PYTHONPATH=src python3 -m ba_agent eval GTS-GATE` passed with `approval_gate_bypass_count=0`; `PYTHONPATH=src python3 -m ba_agent eval GTS-ROUTER` passed with `phase_separation_violations=0`; drift scan across HLD/review/planning/prompt files showed no registry endpoint or legacy unsupported-marker drift.

### Purpose

Package the HLD for RAJA review without self-approving the architecture.

### Execution prompt

Create `docs/development/phase-2-hld-review-package.md` with:

1. HLD scope and source summary.
2. Review checklist by persona lens.
3. Open `[RAJA]` decisions.
4. Key risks, dependencies, and unresolved assumptions.
5. Recommended owner actions: approve as draft baseline, amend, or defer.
6. Explicit statement that the package does not authorize sandbox/live/pilot/production or external side effects.

### Tests

- Cross-check package decisions against the HLD and decision log.
- Scan changed docs for forbidden drift terms and approval overclaim.

### Validations

- Confirm review options are advisory and owner-routed.
- Confirm no agent-authored approval is created.

### Outcomes

- `HLD-G3` package is ready for RAJA review.
- Immediately run QA prompt [Q9C].

---

## ✅ [Q9C] QA review for HLD owner-review package

Status: ✅ Done  
Deliverable path: `docs/development/phase-2-hld-review-package.md`  
Result: QA confirmed the owner-review package routes decisions to RAJA, presents advisory approve/amend/defer options, and does not approve the HLD, sandbox/live access, production rollout, or write-like side effects.  
Test evidence: Re-ran [P9C] gate checks; manually cross-checked the package against `ba-agent-hld.md`, `phase-2-hld-creation-plan.md`, `decision-log.md`, `prompts.md`, and `fleet_prompt.md`.

### Purpose

Review [P9C] for owner-review clarity and non-approval safety.

### Execution prompt

Review the HLD owner-review package and correct missing review asks, unsupported recommendations, stale references, or unsafe approval language.

### Tests

- Re-run the checks used in [P9C].

### Validations

- Confirm the package routes decisions to RAJA.
- Confirm the package does not approve the HLD, sandbox/live access, production rollout, or write-like side effects.

### Outcomes

- HLD lane is ready for RAJA review at `HLD-G3`.
- Return tracking handoff for coordinator update.
