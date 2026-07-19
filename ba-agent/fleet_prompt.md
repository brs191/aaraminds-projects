# BA Agent Fleet Execution Prompt

Fleet execution guide for using multiple Copilot/Aara agents to execute `prompts.md` without violating phase gates. This is an orchestration artifact, not a replacement for the prompt pack.

---

## Document control

| Field                    | Value                                         |
| ------------------------ | --------------------------------------------- |
| Document name            | BA Agent Fleet Execution Prompt               |
| Version                  | 1.0                                           |
| Change note (v1.0)       | Synchronized primary execution source to `prompts.md` v1.2 after HLD owner-review package completion. |
| Change note (v0.9)       | Synchronized primary execution source to `prompts.md` v1.1 after draft HLD completion. |
| Change note (v0.8)       | Added active `[F9]` HLD creation lane synchronized with `prompts.md` v1.0 and the HLD creation plan. |
| Change note (v0.7)       | Added explicit current-active-phase note for `F8`, synchronized with `prompts.md` v0.9, and clarified phase-order handoff into Phase 2 first-slice execution. |
| Status                   | Draft fleet orchestration guide               |
| Prepared date            | 2026-07-06                                    |
| Accountable owner        | RAJA                                          |
| Primary execution source | `prompts.md` v1.2                           |
| Planning baseline        | `docs/planning/project-development-plan.md` |
| Decision baseline        | `docs/planning/decision-log.md`             |

## Fleet verdict

Use a **staged fleet**, not a single all-at-once fleet. Run agents in parallel only inside a gate-safe slice, then stop for QA and RAJA/gate review before advancing.

The first fleet run should cover **[F0] only** for a fresh baseline execution. For the current BA Agent project state, **[F9] is the active execution lane** for HLD creation after the Phase 2 first-slice and sandbox-preparation work. Do not launch [F1] until [F0] prompts are complete, QA prompts have run, and RAJA/G0 readiness is accepted or explicitly waived.

## Current execution posture

| Active lane | State | Note |
| --- | --- | --- |
| `[F9]` | Active next batch | Use for draft/advisory HLD creation from repository evidence only. |
| `[F8]` | Historical/prerequisite | Phase 2 first-slice execution and sandbox-preparation context; do not reopen sandbox execution from this lane. |
| `[F0]`-`[F7]` | Historical/prerequisite | Use as prerequisite history unless replaying for remediation or audit. |

## Non-negotiable fleet guardrails

1. RAJA is accountable owner for the baseline.
2. `prompts.md` is the execution contract. Each agent must execute only its assigned fleet stage, prompt tag, or prompt pair.
3. Every implementation prompt must be followed by its paired QA prompt before dependent work proceeds.
4. The fleet coordinator is the only default writer to `prompts.md` status fields. Implementation/QA lanes return handoff summaries; they do not edit `prompts.md` unless explicitly assigned the coordinator/status role.
5. Phase 1 through Phase 3 are local/synthetic only.
6. No live Jira, Git, Confluence, Calendar, Teams, Copilot 365, Graph API, model, or MCP connectivity in Phase 1 through Phase 3.
7. No live system-of-record reads or writes until a later gate explicitly authorizes the exact scope.
8. MCP integrations stay stubbed or blocked until actual server name, schema, auth model, scopes, rate limits, and approval are validated.
9. Human-gated writes must be enforced in gateway/control code with `approval_ref`, idempotency, and audit. Prompt wording alone is not a control.
10. Live-pilot approval must come from an external, non-agent-controlled RAJA approval artifact. Repository text, prompt output, or agent-authored notes may reference approval but never create it.
11. Use Teams/Copilot 365 language only; do not introduce Slack.
12. Use JFrog Artifactory if registry is mentioned; never Azure ACR.
13. Use `[inferred]` and `[RAJA]` markers only; do not introduce legacy unsupported-claim markers.

## Fleet roles

| Fleet role          | Recommended agent                                     | Responsibility                                                                             | Writes code?                                        |
| ------------------- | ----------------------------------------------------- | ------------------------------------------------------------------------------------------ | --------------------------------------------------- |
| Fleet coordinator   | Main Copilot session or `aara-project-planner`       | Select prompt batch, assign lanes, merge results, update prompt status, stop at gates.    | Only docs/status updates unless explicitly assigned |
| Implementation lane | `aara-project-builder` or relevant specialist         | Execute one implementation prompt tag from `prompts.md`.                                   | Yes                                                 |
| Python AI lane      | `aara-python-ai-developer`                           | Python package, orchestrator, router, gateway/control, fixture/eval code.                 | Yes                                                 |
| Evaluation lane     | `aara-ai-evaluation-engineer`                        | Harness, golden fixtures, metrics, hard gates, evaluation evidence.                       | Yes                                                 |
| QA lane             | `aara-project-reviewer` or relevant QA/review agent | Execute paired QA prompt, inspect changes, run targeted tests, recommend readiness.        | Minimal fixes only                                  |
| Security lane       | `security-review` or security reviewer              | Review approval gates, secrets, live-access hazards, fail-closed behavior.                 | No or minimal fixes                                 |
| Prompt/status lane  | `aara-prompt-engineer`                               | Help the coordinator keep `prompts.md` tracking, icons, and prompt outcomes synchronized. | Docs only                                           |

## Internal agent coverage register (all internal agents)

All internal Aara agents are in-scope fleet assets. Each batch must either assign the agent directly, invoke it conditionally based on trigger, or record `N/A` with reason.

| Internal agent | Coverage mode | Default BA Agent trigger |
| --- | --- | --- |
| `aara-business-analyst` | Core | Requirement semantics, ambiguity control, artifact quality for P7/P8. |
| `aara-project-planner` | Core | Coordinator lane, gate sequencing, dependency/risk planning. |
| `aara-project-builder` | Core | Implementation lane for code/document updates. |
| `aara-python-ai-developer` | Core | Python orchestration, schema, route, and gateway-facing implementation. |
| `aara-ai-evaluation-engineer` | Core | GTS/BA-EM evaluation, regression design, hard-gate evidence. |
| `aara-project-reviewer` | Core | QA lane with high-signal corrections and readiness recommendations. |
| `aara-prompt-engineer` | Core | Prompt/status synchronization, prompt quality and trigger tuning. |
| `aara-project-architect` | Core | Architecture conformance, interface boundaries, and isolation rules. |
| `aara-project-debugger` | Core | Root-cause analysis for failing checks or gate regressions. |
| `aara-agent-blueprint-advisor` | Conditional | Agent-control model and governance blueprint clarifications. |
| `aara-agent-engineer` | Conditional | End-to-end agent package quality checks for future enterprise slices. |
| `aara-ai-application-architect` | Conditional | AI architecture tradeoffs or archetype decisions. |
| `aara-ai-technical-author` | Conditional | Decision-grade technical writing and structured evidence narratives. |
| `aara-executive-narrative-advisor` | Conditional | Executive-ready summaries for gate reviews and escalations. |
| `aara-data-tier-designer` | Conditional | Data model/store/retention implications for context memory and traceability. |
| `aara-mcp-server-builder` | Conditional | MCP server/tool-surface implementation or hardening work. |
| `aara-senior-microservices-architect` | Conditional | Cross-service decomposition and integration boundary decisions. |
| `aara-next-bff-developer` | Conditional | UI/BFF surfaces if Phase 2 expands to frontend review workflows. |
| `aara-business-strategist` | Conditional | Priority and scope strategy disputes requiring business framing. |
| `aara-azure-cost-reviewer` | Conditional | Azure cost-impact review when infrastructure options enter scope. |
| `aara-copilot-cost-reviewer` | Conditional | Copilot usage-cost governance when applicable. |
| `aara-code-model-designer` | Conditional | Static code-model architecture for future code-comprehension extensions. |
| `aara-codebase-extraction-engineer` | Conditional | Extraction-pipeline implementation for future code-comprehension paths. |
| `aara-content-strategist` | Conditional | External thought-leadership outputs (out-of-scope unless requested). |
| `aara-leadership-status-deck` | Conditional | Leadership deck production from gate-status inputs. |
| `aara-status-deck` | Conditional | Monthly status deck generation workflow. |
| `aara-network-topology-reviewer` | Conditional | Network risk/compliance reviews if topology scope is introduced. |
| `aara-topology-visualizer` | Conditional | Topology visualization outputs if network-diagram scope is introduced. |

## Skill coverage routing (all internal skill families)

Each batch must explicitly name invoked skills or mark `N/A`. Do not silently skip skill routing.

| Skill family | Skills | Trigger in BA Agent execution |
| --- | --- | --- |
| Core BA documentation/evaluation | `ai-technical-author`, `lsp-setup` | Decision-grade docs, prompt QA, language-server setup for structured edits. |
| AI model and gateway | `azure-ai`, `azure-aigateway`, `microsoft-foundry`, `customize-cloud-agent` | AI orchestration, gateway policy, model/governance design tasks. |
| App preparation/deployment/validation | `azure-prepare`, `azure-deploy`, `azure-validate`, `azure-upgrade`, `python-appservice-deploy` | Scaffold/deploy/validate workflows when explicitly in scope. |
| Infrastructure and compute | `azure-enterprise-infra-planner`, `azure-kubernetes`, `azure-compute`, `airunway-aks-setup`, `azure-quotas` | Compute/infrastructure planning or AKS/GPU scenarios. |
| Data/storage/observability | `azure-storage`, `azure-kusto`, `appinsights-instrumentation` | Data-plane, telemetry, and log-query design tasks. |
| Security/compliance/reliability | `azure-compliance`, `azure-rbac`, `azure-reliability`, `azure-diagnostics` | Security posture, RBAC, resilience, and diagnostics hardening. |
| Resource and visualization | `azure-resource-lookup`, `azure-resource-visualizer` | Inventory, discovery, and topology/resource visualization. |
| Ops/FinOps | `azure-ops`, `azure-cost` | CI/CD or cost-control workflows with explicit owner request. |
| Messaging/identity | `azure-messaging`, `entra-agent-id`, `entra-app-registration` | Messaging and identity integration troubleshooting/planning. |
| Migration and modernization | `azure-cloud-migrate`, `azure-hosted-copilot-sdk` | Cross-cloud migration or hosted Copilot SDK-specific modernization. |

## Persona lens protocol (mandatory review lenses)

For every QA and gate package, apply these persona lenses explicitly in review output:

| Persona lens | Required checks |
| --- | --- |
| BA SME | Requirement clarity, ambiguity handling, and business-readability quality. |
| Product Owner | Scope/priority correctness and decision-ownership clarity. |
| QA/Evaluation | Case coverage, regression integrity, and measurable acceptance criteria for the active slice. |
| Architect | Boundary integrity, route isolation, data/API implications, and future extensibility. |
| Security/Privacy | Classification handling, sensitive-data controls, and write-gate enforcement. |
| Compliance/Legal | Regulatory and audit obligations are surfaced, never auto-approved. |
| Platform/Tool Owner | Tool scopes, auth/rate limits, and enablement evidence completeness. |
| Delivery Lead | Gate sequencing, readiness posture, blocker clarity, and plan coherence. |
| FinOps | Cost impact, rightsizing, budget guardrails, and cloud spend risk where applicable. |
| Executive Narrative (when requested) | Decision-grade executive summary for continue/adjust/stop options. |

## Status icon protocol

The coordinator updates the assigned prompt heading icon in `prompts.md` from lane handoffs:

| Icon | Meaning     | When to set                                                                               |
| ---- | ----------- | ----------------------------------------------------------------------------------------- |
| ⏳   | Not started | Default state before work begins                                                          |
| 🔄   | In progress | Agent starts executing the prompt                                                         |
| ✅   | Completed   | Prompt and paired tests/validations passed                                                |
| 🟡   | Partial     | Some deliverables complete, but a dependency or validation remains                        |
| ❌   | Blocked     | Cannot proceed without RAJA decision, missing gate, missing command, or failed validation |

Do not mark an implementation prompt `✅` until its paired QA prompt is also complete or the QA prompt explicitly records no changes needed. If implementation is complete but QA is pending, mark the implementation prompt `🟡` with `Status: 🟡 Implementation complete; QA pending`.

## Fleet stage tags

Fleet stage tags mirror `prompts.md` and are fewer than 6 characters. Use them in coordinator updates, batch summaries, and gate-review notes.

| Fleet tag | Stage                                       | Prompt range                                 | Gate stop                                                                                       |
| --------- | ------------------------------------------- | -------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| [F0]      | Phase 0 — Pre-work and decision closure    | [P0A]/[Q0A] through [P0C]/[Q0C]              | Stop after G0 readiness evidence is updated.                                                    |
| [F1]      | Phase 1 — Engineering foundation           | [P1T]/[Q1T], [P1A]/[Q1A] through [P1D]/[Q1D] | Stop at G1; do not begin Phase 2 without G1 readiness evidence.                                 |
| [F2]      | Phase 2 — Synthetic standup thin slice     | [P2A]/[Q2A] through [P2E]/[Q2E]              | Stop at G2; do not begin control hardening without thin-slice evidence.                         |
| [F3]      | Phase 3 — Gateway and evaluation hardening | [P3A]/[Q3A] through [P3E]/[Q3E]              | Stop at G3; BA-EM-005 must remain zero.                                                         |
| [F4]      | Phase 4 — Sandbox integration readiness    | [P4A]/[Q4A] through [P4E]/[Q4E]              | Stop at G4; sandbox readiness does not authorize live pilot use.                                |
| [F5]      | Phase 5 — MVP capability expansion         | [P5A]/[Q5A] through [P5E]/[Q5E]              | Stop at G5; BA-EM-005 and BA-EM-009 must remain zero.                                           |
| [==F6==]  | Phase 6 — MVP pilot readiness and pilot    | [P6A]/[Q6A] through [P6G]/[Q6G]              | Stop for explicit non-agent-controlled RAJA approval before [P6F]; production remains separate. |
| [F7]      | Phase 7 — Phase 2 readiness                | [P7A]/[Q7A] through [P7E]/[Q7E]              | Stop with a separate Phase 2 plan for RAJA review.                                              |
| [F8]      | Phase 8 — Phase 2 first-slice execution    | [P8A]/[Q8A] through [P8E]/[Q8E]              | Stop at `P2-G5`; sandbox/live/production remain separately authorized.                          |
| [F9]      | Phase 9 — HLD creation                     | [P9A]/[Q9A] through [P9C]/[Q9C]              | Stop at `HLD-G3`; HLD remains draft/advisory and repository-evidence-only unless separately approved. |

## Execution model

### Step 1 — Coordinator prepares a batch

The coordinator reads:

1. `prompts.md`
2. `fleet_prompt.md`
3. `docs/planning/project-development-plan.md`
4. `docs/planning/decision-log.md`
5. `.github/copilot-instructions.md`

The coordinator chooses the next safe batch from the batch plan below and verifies all prerequisites. If a prerequisite is missing, the coordinator marks the blocked prompt `❌` or `🟡` and records the blocker in `prompts.md`.

### Step 2 — Launch implementation lanes

Launch implementation prompts only when their dependencies do not overlap on the same files in risky ways.

Rules:

- Never run two agents that will edit the same source files unless one owns the file and the other is QA-only.
- Never run a QA prompt before its paired implementation prompt completes.
- Prefer one implementation lane plus one independent docs/status lane early in a phase.
- Use parallelism for research, review, or independent artifacts; do not parallelize tightly coupled code edits.
- Do not let parallel lanes edit `prompts.md`; collect handoffs and apply tracking updates serially.

### Step 3 — Run paired QA lanes

For every completed implementation prompt, run its paired QA prompt.

QA agents must:

1. Inspect the exact changed files.
2. Run the prompt's tests and validations.
3. Make only focused fixes.
4. Return a status handoff for the coordinator with verdict, deliverable paths, test evidence, and blockers.
5. Recommend readiness; never claim human/gate approval.

### Step 4 — Coordinator merges evidence

The coordinator checks:

1. Prompt icons and status fields are accurate and applied by the coordinator/status lane only.
2. Test evidence is recorded.
3. No live integration or write path was introduced early.
4. Gate evidence exists before moving to the next phase.
5. Any `[RAJA]` item is recorded as unresolved or owner-dependent.

### Step 5 — Stop at gate

At the end of each phase, stop. The coordinator summarizes gate evidence and asks RAJA for the next gate decision if required.

## Fleet batch plan

### [F0] Batch 0 — Phase 0 documentation readiness

Goal: confirm baseline, decision log, G0 readiness, and risk/open-question triage.

Run sequence:

| Order | Prompt                  | Lane               | Parallel?                    |
| ----- | ----------------------- | ------------------ | ---------------------------- |
| 1     | `[P0A]`               | Planner/docs lane  | No                           |
| 2     | `[Q0A]`               | QA lane            | No                           |
| 3     | `[P0B]` and `[P0C]` | Planner/docs lanes | Yes, if files do not overlap |
| 4     | `[Q0B]` and `[Q0C]` | QA lanes           | Yes, after P0B/P0C complete  |

Exit condition: Phase 0 prompts are complete or blockers are explicit. G0 remains synthetic-only.

### [F1] Batch 1 — Phase 1 foundation scaffold

Goal: create a runnable local foundation without live integrations.

Recommended sequence:

| Order | Prompt    | Lane                               | Notes                                                                      |
| ----- | --------- | ---------------------------------- | -------------------------------------------------------------------------- |
| 1     | `[P1T]` | Python AI / planner lane           | Creates technical baseline; no source scaffold yet                         |
| 2     | `[Q1T]` | QA / Python review lane            | Verifies technical baseline                                                |
| 3     | `[P1A]` | Project builder                    | Creates layout/tooling from [P1T]; do not parallelize with other code work |
| 4     | `[Q1A]` | Project reviewer                   | Verifies scaffold                                                          |
| 5     | `[P1B]` | Python AI developer                | Adds package skeleton/safe defaults                                        |
| 6     | `[Q1B]` | Project reviewer/security reviewer | Verifies safe defaults                                                     |
| 7     | `[P1C]` | Project builder                    | Adds local commands/test tooling                                           |
| 8     | `[Q1C]` | Project reviewer                   | Verifies command truthfulness                                              |
| 9     | `[P1D]` | Planner/docs lane                  | Documents G1 readiness                                                     |
| 10    | `[Q1D]` | QA lane                            | Recommends G1 readiness                                                    |

Exit condition: project has runnable local test command, source skeleton, safe defaults, and docs that match actual commands.

### [F2] Batch 2 — Phase 2 synthetic standup thin slice

Start only after G1 readiness is accepted.

Parallel plan:

| Wave | Prompt    | Lane                    | Dependency                        |
| ---- | --------- | ----------------------- | --------------------------------- |
| 1    | `[P2A]` | Evaluation/Python lane  | G1 accepted                       |
| 2    | `[Q2A]` | QA lane                 | P2A                               |
| 3    | `[P2B]` | Python lane             | P2A/Q2A                           |
| 4    | `[Q2B]` | QA lane                 | P2B                               |
| 5    | `[P2C]` | Teams payload lane      | P2B/Q2B summary contract complete |
| 6    | `[Q2C]` | QA lane                 | P2C                               |
| 7    | `[P2D]` | Python AI lane          | P2B/P2C complete                  |
| 8    | `[Q2D]` | QA lane                 | P2D                               |
| 9    | `[P2E]` | Coordinator/Python lane | P2D complete                      |
| 10   | `[Q2E]` | QA lane                 | P2E                               |

Exit condition: G2 thin-slice evidence exists: synthetic standup route, fixture-backed summary, Adaptive Card payload, no writes, degraded-source honesty.

### [F3] Batch 3 — Phase 3 gateway and evaluation hardening

Start only after G2 readiness is accepted.

Parallel plan:

| Wave | Prompt                   | Lane                  | Dependency                     |
| ---- | ------------------------ | --------------------- | ------------------------------ |
| 1    | `[P3A]`                | Python/control lane   | G2 accepted                    |
| 2    | `[Q3A]`                | QA/security lane      | P3A                            |
| 3    | `[P3B]`                | Security/control lane | P3A/Q3A complete               |
| 4    | `[Q3B]`                | Security QA lane      | P3B                            |
| 5    | `[P3C]`                | Trace/audit lane      | Gateway/write semantics stable |
| 6    | `[Q3C]`                | QA/security lane      | P3C                            |
| 7    | `[P3D]`                | Evaluation lane       | P3A/P3B/P3C complete           |
| 8    | `[Q3D]`                | Evaluation QA lane    | P3D                            |
| 9    | `[P3E]` then `[Q3E]` | Coordinator + QA      | All Phase 3 prompts complete   |

Exit condition: BA-EM-005 approval-gate bypass count is zero in adversarial tests and all tool calls produce audit records.

### [F4] Batch 4 — Phase 4 sandbox readiness

Start only after G3 readiness is accepted. This batch prepares sandbox readiness; it does not authorize live pilot use.

Recommended sequence:

1. `[P4A]` / `[Q4A]` — sandbox validation plan.
2. `[P4B]` / `[Q4B]` — MCP schema validation process.
3. `[P4C]` / `[Q4C]` — read-only Jira/Git replacement path.
4. `[P4D]` / `[Q4D]` — Teams sandbox/channel approval readiness.
5. `[P4E]` / `[Q4E]` — blocked unvalidated tools and G4 readiness.

Exit condition: only validated read tools can replace fixtures in sandbox; unvalidated tools remain stubbed/blocked.

### [F5] Batch 5 — Phase 5 MVP capability expansion

Start only after G4 readiness is accepted.

Parallel plan:

| Wave | Prompt                          | Lane                      | Dependency                           |
| ---- | ------------------------------- | ------------------------- | ------------------------------------ |
| 1    | `[P5A]`, `[P5B]`, `[P5C]` | Separate capability lanes | G4 accepted; shared contracts stable |
| 2    | `[Q5A]`, `[Q5B]`, `[Q5C]` | QA lanes                  | P5A/P5B/P5C                          |
| 3    | `[P5D]`                       | Evaluation lane           | Capability outputs available         |
| 4    | `[Q5D]`                       | Evaluation QA lane        | P5D                                  |
| 5    | `[P5E]` then `[Q5E]`        | Coordinator + QA          | All Phase 5 evidence complete        |

Exit condition: G5 candidate evidence exists, BA-EM-005 = 0, BA-EM-009 = 0, owner-set thresholds remain `[RAJA]` or explicitly waived by RAJA.

### [==F6==] Batch 6 — Phase 6 pilot readiness and pilot

Start only after G5 readiness is accepted. Split readiness, authorization, execution, and assessment.

Recommended sequence:

1. `[P6A]` / `[Q6A]` — pilot runbook and scope package.
2. `[P6B]` / `[Q6B]` — support/RACI under RAJA accountability.
3. `[P6C]` / `[Q6C]` — release notes and harness run ID.
4. `[P6D]` / `[Q6D]` — rollback and kill-switch drill.
5. `[P6E]` / `[Q6E]` — G6 authorization package only.
6. Stop for explicit non-agent-controlled RAJA approval.
7. `[P6F]` / `[Q6F]` — limited live pilot execution only after explicit non-agent-controlled RAJA approval.
8. `[P6G]` / `[Q6G]` — post-pilot assessment.

Exit condition: post-pilot assessment recommends continue, adjust, or stop; production rollout remains a separate authorization path.

### [F7] Batch 7 — Phase 7 Phase 2 readiness

Start only after post-pilot assessment is accepted by RAJA.

Recommended sequence:

1. `[P7A]` / `[Q7A]` — Phase 2 prioritization brief.
2. `[P7B]` / `[Q7B]` — tool approval matrix.
3. `[P7C]` / `[Q7C]` — data/classification plan.
4. `[P7D]` / `[Q7D]` — GTS-P2-REQ evaluation approach.
5. `[P7E]` / `[Q7E]` — separate Phase 2 plan readiness.

Exit condition: a separate Phase 2 plan is ready for RAJA review; if `P2-G0` is accepted, proceed to [F8].

### [F8] Batch 8 — Phase 2 first-slice execution

Start only after `P2-G0` acceptance is recorded in `docs/planning/phase-2-implementation-plan.md` and `docs/planning/decision-log.md`.

Recommended sequence:

1. `[P8A]` / `[Q8A]` — `P2-G1` technical baseline/scaffold.
2. `[P8B]` / `[Q8B]` — `P2-G2` synthetic requirement-discovery thin slice.
3. `[P8C]` / `[Q8C]` — `P2-G3` evaluation/control hardening.
4. `[P8D]` / `[Q8D]` — `P2-G4` tool/data readiness decision package.
5. `[P8E]` / `[Q8E]` — `P2-G5` candidate review and stop decision package.

Exit condition: first-slice execution produces a decision-grade `P2-G5` package; no sandbox/live/production behavior is authorized by this batch.

### [F9] Batch 9 — HLD creation

Start only after RAJA records the HLD scope-change directive and `[P9A]`/`[Q9A]` establish `HLD-G0` in `docs/planning/phase-2-hld-creation-plan.md`.

Recommended sequence:

1. `[P9A]` / `[Q9A]` — HLD scope-change plan and prompt/fleet synchronization.
2. `[P9B]` / `[Q9B]` — draft/advisory BA Agent HLD.
3. `[P9C]` / `[Q9C]` — HLD owner-review package.

Exit condition: HLD owner-review package is ready for RAJA review at `HLD-G3`; no sandbox/live/non-synthetic/pilot/production behavior, external publishing, autonomous approval, or write-like side effect is authorized by this batch.

## Coordinator prompt template

Use this as the main session instruction before launching a batch:

```text
You are the BA Agent fleet coordinator. Read fleet_prompt.md, prompts.md, docs/planning/project-development-plan.md, docs/planning/decision-log.md, and .github/copilot-instructions.md.

Execute only fleet stage [F<N>] / Batch <N>: <batch name>.

Rules:
- RAJA is accountable owner.
- Use prompts.md as the execution contract.
- Assign implementation prompts and paired QA prompts as separate lanes only when dependencies allow.
- Do not advance past the fleet-stage gate.
- Apply prompts.md heading icons and tracking fields serially from lane handoffs.
- Record deliverable paths, test evidence, and blockers.
- Do not introduce live integrations or live writes before the authorized phase gate.
- Recommend readiness only; do not claim RAJA/gate approval.

Return:
1. Completed prompts.
2. Deliverables changed.
3. Tests and validations run.
4. Blockers.
5. Whether the fleet stage is ready for RAJA gate review.
```

## Implementation lane prompt template

```text
You are an implementation lane agent for the BA Agent project.

Assigned prompt tag: <TAG>
Source file: prompts.md

Read:
- prompts.md entry <TAG>
- fleet_prompt.md
- .github/copilot-instructions.md
- docs/planning/project-development-plan.md
- docs/planning/decision-log.md

Execute only <TAG>. Do not execute later prompts.

Rules:
- Keep within the assigned phase and prompt.
- Do not make live integrations unless the prompt and gate explicitly authorize them.
- Run the Tests and Validations listed in the prompt.
- Do not edit prompts.md unless explicitly assigned as coordinator/status lane.
- Return proposed heading icon, Status, Deliverable path, Result, Test evidence, and exact blocker if blocked.

Return a concise handoff for the paired QA lane and coordinator.
```

## QA lane prompt template

```text
You are a QA/review lane agent for the BA Agent project.

Assigned QA prompt tag: <QTAG>
Paired implementation tag: <PTAG>
Source file: prompts.md

Read:
- prompts.md entries <PTAG> and <QTAG>
- fleet_prompt.md
- relevant changed files from <PTAG>

Execute only <QTAG>.

Rules:
- Review the exact changes from <PTAG>.
- Run the Tests and Validations listed in <QTAG>.
- Make only focused fixes.
- Do not edit prompts.md unless explicitly assigned as coordinator/status lane.
- Return proposed heading icons, Status values, Deliverable paths, Results, Test evidence, and blockers for both <PTAG> and <QTAG>.
- Recommend readiness only; do not claim RAJA/gate approval.

Return:
1. QA verdict.
2. Fixes made, if any.
3. Tests and validations run.
4. Remaining blockers.
```

## Recommended first fleet run

Start with **[F0] only**:

1. Execute Phase 0 prompts to confirm baseline and risks.
2. Stop at G0.
3. Ask RAJA whether to proceed to [F1] / Batch 1.

Do not launch [F1] until G0 readiness is accepted or explicitly waived. Do not launch Phase 2 or later until G1 readiness evidence exists.
