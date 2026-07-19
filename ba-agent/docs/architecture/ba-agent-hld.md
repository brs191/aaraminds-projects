# BA Agent High-Level Design

Draft/advisory HLD for the AaraMinds Business Analyst AI Agent. This document is repository-evidence-only and does not approve architecture, delivery commitment, sandbox execution, live integration, non-synthetic data use, production rollout, external publishing, autonomous approval, or write-like side effects.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent High-Level Design |
| Version | 0.3 |
| Change note (v0.3) | Recorded RAJA approval as the current draft baseline and synchronized status with the owner decision. |
| Change note (v0.2) | Linked HLD owner-review package and updated gate status after `HLD-G3` packaging. |
| Status | Approved draft baseline; non-authorizing reference architecture |
| Prepared date | 2026-07-13 |
| Accountable owner | RAJA |
| Execution lane | `[F9]` HLD creation |
| Gate status | `HLD-G3` owner-review package complete |
| Scope-change plan | `docs/planning/phase-2-hld-creation-plan.md` v0.4 |
| Owner-review package | `docs/development/phase-2-hld-review-package.md` v0.3 |
| Decision baseline | `docs/planning/decision-log.md` v2.9 |
| Requirement baseline | `docs/requirements/business-analyst-agent-requirements.md` v0.4 |
| Runtime architecture baseline | `docs/requirements/ba_agent_runtime_architecture.md` v0.1 |
| Tool contract baseline | `docs/requirements/ba_agent_mcp_tool_contracts.md` v0.3 |
| Evaluation baseline | `docs/requirements/ba_agent_evaluation_harness.md` v0.2 |

## 1) Executive architecture summary

BA Agent is a human-gated assistant for Scrum/BA workflows. The MVP surface is Teams/Copilot 365, the orchestration pattern is LangGraph, and source-system access is mediated through MCP tools. The Phase 2 scope extends into Enterprise BA drafting and traceability, but current HLD execution is limited to a draft/advisory repository-evidence-only architecture artifact.

The proposed target architecture has four major layers:

1. **User interaction layer:** Teams/Copilot 365 user surface for prompts, outputs, approval requests, and review routing.
2. **AI orchestration layer:** Python `orchestrator-svc` with a LangGraph router and capability graphs.
3. **Control and integration layer:** `mcp-gateway-svc` enforcing auth/scope/approval/idempotency/audit controls before MCP servers can reach downstream systems.
4. **State, audit, and evaluation layer:** Postgres-backed graph checkpoints/approval/audit records, Event Hubs fan-out, OpenTelemetry metrics/traces, and golden-test hard gates.

Evidence: MVP surface and LangGraph/MCP direction come from `business-analyst-agent-requirements.md` and `ba_agent_runtime_architecture.md`; tool-control conventions come from `ba_agent_mcp_tool_contracts.md`; hard gates come from `ba_agent_evaluation_harness.md`.

## 2) Scope and non-goals

### In scope for this HLD

| Area | HLD treatment |
| --- | --- |
| MVP capability framing | Standup summary, sprint planning recommendations, retrospective reports, and sprint-health monitoring. |
| Phase 2 context | Requirement-discovery and HLD lane context as draft/advisory follow-on work. |
| Runtime topology | Proposed Teams/Copilot 365 -> orchestrator -> MCP gateway -> MCP servers -> systems-of-record pattern. |
| Control model | Human-gated writes, `approval_ref`, idempotency, audit, read allowlists, blocked sandbox rows. |
| Evaluation | BA-EM-005 and BA-EM-009 hard gates plus synthetic golden-set posture. |
| Deployment direction | Azure-primary proposal, GitHub Actions OIDC, Terraform AzureRM, managed identities, Key Vault, JFrog Artifactory. |

### Explicit non-goals

| Non-goal | Boundary |
| --- | --- |
| Architecture approval | This HLD is a review draft only; RAJA/architecture review is required. |
| Live or sandbox execution | No row in `mcp-validation-register.json` is validated; sandbox execution remains blocked. |
| Non-synthetic data processing | Real tickets, pages, repos, messages, meeting notes, and restricted data remain unauthorized. |
| Autonomous decisions | Agent output remains advisory; requirements, architecture, sprint scope, publishing, and compliance commitments remain human-owned. |
| External side effects | Writes, sends, publishes, comments, subscriptions, approval records, and external storage remain write-like and approval-gated. |

Evidence: `phase-2-hld-creation-plan.md`, `decision-log.md` `P2-DEC-016`/`P2-DEC-017`, `phase-2-implementation-plan.md`, and `ba_agent_mcp_tool_contracts.md`.

## 3) Source and decision traceability

| HLD claim | Source evidence |
| --- | --- |
| Teams/Copilot 365 is the user surface | `business-analyst-agent-requirements.md` Source register S1/S2; `ba_agent_runtime_architecture.md` source-fixed constraints |
| LangGraph is the orchestration model | `business-analyst-agent-requirements.md`; `ba_agent_runtime_architecture.md` source-fixed constraints |
| MCP tools mediate Jira/Git/Confluence/Calendar/Teams access | `business-analyst-agent-requirements.md`; `ba_agent_mcp_tool_contracts.md` |
| Writes require human approval and `approval_ref` | `ba_agent_mcp_tool_contracts.md`; `src/ba_agent/gateway.py` |
| Sandbox execution remains blocked | `decision-log.md` `P2-DEC-016`; `mcp-validation-register.json`; `src/ba_agent/phase2/sandbox_mcp.py` |
| HLD is now active scope | `decision-log.md` `P2-DEC-017`; `phase-2-hld-creation-plan.md`; `prompts.md` v1.0 |
| Hard gates are zero approval bypass and zero phase-separation violations | `ba_agent_evaluation_harness.md`; current `ba_agent eval GTS-GATE` / `GTS-ROUTER` commands |

## 4) Logical component architecture

```text
Teams / Copilot 365
        |
        v
User interaction and approval UI [RAJA]
        |
        v
orchestrator-svc (Python + LangGraph)
  - router
  - standup graph
  - planning graph
  - retro graph
  - health graph
  - Phase 2 requirement/HLD drafting lane [inferred]
        |
        v
mcp-gateway-svc
  - capability allowlists
  - auth/scope checks [inferred]
  - approval_ref validation
  - idempotency enforcement
  - audit emission
        |
        v
MCP server containers
  - Jira
  - Git
  - Confluence
  - Calendar
  - Teams/Copilot 365
        |
        v
Approved systems of record [blocked until validated]
```

### Component responsibilities

| Component | Responsibilities | Current status |
| --- | --- | --- |
| Teams/Copilot 365 surface | User prompts, advisory output display, human approval/review interactions. | Target surface; live channel/app details are `[RAJA]`. |
| `orchestrator-svc` | Route requests, execute LangGraph capability flows, call Azure OpenAI only from the orchestration tier, stamp model/prompt/graph metadata. | Proposed in runtime architecture; local source currently implements synthetic routes/control surfaces. |
| `mcp-gateway-svc` | Enforce allowlists, approval gates, idempotency, audit, and tool boundary controls outside the LLM loop. | Local gateway fake implements the core control semantics in `src/ba_agent/gateway.py`. |
| MCP server containers | Adapt each system's API into validated MCP tools with least-privilege credentials. | Proposed contracts exist; live/sandbox validation is blocked. |
| Postgres state/audit store | LangGraph checkpoints, approval records, idempotency keys, audit records. | Proposed; local implementation uses in-memory fakes for tests. |
| Event Hubs fan-out | Audit/webhook buffering and fan-out. | Proposed; not implemented in local Phase 2 slice. |
| Evaluation harness | Golden tests and hard gates for routing, phase separation, approval-gate bypass, evidence quality. | Executable local commands exist for hard gates and synthetic evals. |

## 5) Runtime flows

### 5.1 Local/synthetic read flow

1. User request or test case enters the router.
2. Router chooses a capability graph.
3. Graph reads synthetic fixtures or local placeholders.
4. Output is labeled draft/advisory and includes evidence references, assumptions, `[inferred]`, `[RAJA]`, and open questions where required.
5. Evaluation commands validate hard gates and output conformance.

Current evidence: Phase 2 first-slice synthetic artifacts and executable hard gates in `ba_agent eval GTS-GATE` and `ba_agent eval GTS-ROUTER`.

### 5.2 Future sandbox read flow

1. RAJA and owners complete row-level evidence for a specific tool row.
2. `mcp-validation-register.json` row moves to validated/ready with complete owner/scope/schema/auth/rate-limit/audit evidence.
3. Adapter construction checks `BA_AGENT_DATA_SOURCE_MODE=sandbox_read`, rejects live integrations, and requires a validated register row.
4. Adapter allowlists only approved upstream read tools before any upstream MCP call.
5. Tool response carries source timestamps and status; degraded/denied/throttled states remain visible.

Current evidence: `Phase2JiraReadOnlyMcpAdapter`, `Phase2ConfluenceReadOnlyMcpAdapter`, and `evaluate_phase2_sandbox_upstream_tool`. Current status remains blocked because no row validates.

### 5.3 Write-like action flow

1. Orchestrator creates an advisory artifact and may request human approval.
2. Human approval callback outside the model/tool loop issues a single-use `approval_ref` only after authenticated approval.
3. Gateway validates artifact/action/scope/expiry/idempotency and atomically consumes the `approval_ref`.
4. Gateway emits audit synchronously.
5. Only then may the write-like MCP tool execute.

Current evidence: `ba_agent_mcp_tool_contracts.md` and `src/ba_agent/gateway.py`. Current HLD lane does not authorize any write-like path.

## 6) Control-plane design

| Control | Design rule | Evidence |
| --- | --- | --- |
| Capability allowlist | A graph may only call tools allowed for its capability. | `src/ba_agent/gateway.py` `CAPABILITY_ALLOWLISTS`. |
| Write fail-closed | Write-like actions are rejected without valid approval. | `WRITE_LIKE_ACTIONS` and `LocalGatewayFake._reject_write_like` behavior. |
| Approval validation | `approval_ref` must match artifact/action, be unexpired, and be single-use. | `InMemoryApprovalStore.validate_and_consume`. |
| Sandbox adapter construction | Adapter cannot be constructed unless data mode, live flag, and register row are safe. | `src/ba_agent/phase2/sandbox_mcp.py`. |
| Upstream allowlist | Known write-like upstream tools are denied before upstream calls. | `PHASE2_KNOWN_WRITE_LIKE_UPSTREAM_TOOLS`. |
| Audit coupling | Tool calls emit audit records; audit failure fails the call in the local fake. | `InMemoryAuditSink` / gateway behavior. |

## 7) Data architecture and classification

| Data class | Current posture |
| --- | --- |
| Synthetic fixtures | Allowed for local development and evaluation. |
| Repository docs/source | Allowed as HLD evidence. |
| Sandbox tool responses | Blocked until row-level validation and RAJA execution approval. |
| Live system data | Blocked. |
| Restricted/source-code/customer data | Blocked unless future classification and handling approvals are recorded. |
| Prompt/completion retention | `[RAJA]`; runtime architecture proposes defaults for review, not fact. |
| Audit records | Required for all tools; retention/residency remain `[RAJA]`. |

Data must preserve `source_timestamp` separately from `retrieved_at` for tool responses, and outputs must not silently fabricate or omit degraded/denied/throttled source states.

## 8) Security, privacy, and safety design

1. **Human authority:** the agent drafts, recommends, and routes; it does not approve.
2. **Prompt-injection containment:** tool-fetched content is treated as data, capability allowlists constrain actions, and writes require non-agent approval.
3. **Least privilege:** each MCP server uses scoped credentials and a system-specific managed identity in the proposed architecture.
4. **Secret handling:** no secrets in code or prompts; proposed path is Key Vault with managed identities.
5. **No silent degradation:** unavailable or denied sources surface as degraded/denied/throttled rather than invented content.
6. **Auditability:** all tool calls, denials, approval failures, and control decisions must be traceable.
7. **Egress boundary:** runtime architecture proposes restricted egress allowlists; feasibility remains `[RAJA]`.

## 9) Evaluation and quality gates

| Gate/metric | HLD design requirement |
| --- | --- |
| BA-EM-005 approval-gate bypass count | Must remain zero; any write-like success without valid approval blocks release. |
| BA-EM-009 phase-separation violations | Must remain zero; MVP outputs cannot expose Phase 2 capabilities. |
| BA-EM-002/003/006 evidence quality | Source-backed claims must carry references; unsupported claims must be marked `[inferred]` or `[RAJA]`. |
| GTS-GATE | Regression set for approval bypass attempts. |
| GTS-ROUTER | Regression set for route separation, unsupported requests, and phase boundaries. |
| GTS-P2-REQ | Phase 2 requirement-discovery synthetic set; supports Phase 2 evidence discipline. |

Current local gate evidence from the HLD lane: `validate-mcp` shows all sandbox rows blocked; `GTS-GATE` passes with `approval_gate_bypass_count=0`; `GTS-ROUTER` passes with `phase_separation_violations=0`.

## 10) Deployment and operations direction

The proposed deployment direction is Azure-primary, not approved for production by this HLD:

| Area | Proposed direction | Status |
| --- | --- | --- |
| Compute | Azure Container Apps for orchestrator, gateway, MCP servers, scheduler jobs | Proposed in runtime architecture; `[RAJA]` for final platform decision |
| Identity | Entra ID app registration and user-assigned managed identities | Proposed; tenant/app details `[RAJA]` |
| Secrets | Key Vault references with managed identities | Proposed |
| Infrastructure as code | Terraform AzureRM | Proposed |
| CI/CD | GitHub Actions OIDC, no stored cloud credentials | Proposed |
| Container registry | JFrog Artifactory | Workspace convention |
| Observability | OpenTelemetry, Prometheus, Grafana, audit traces | Proposed |
| Eventing | Event Hubs for audit/webhook fan-out | Proposed |
| Model access | Azure OpenAI called only by orchestrator | Proposed; model/region `[RAJA]` |

## 11) Key risks and mitigations

| Risk | Impact | Mitigation |
| --- | --- | --- |
| HLD overclaims implementation or approval | Architecture could be misread as authorized | Keep draft/advisory labels, cite sources, route decisions to RAJA. |
| Sandbox/live scope creep | Unauthorized data/tool calls | Keep register rows blocked until complete evidence and explicit execution approval. |
| Prompt injection triggers side effects | Unauthorized writes or exfiltration | Gateway allowlists, approval refs, idempotency, audit, egress controls. |
| Tool schema drift | Adapters mismatch actual MCP tools | Validation register and owner evidence before enablement. |
| Evidence hallucination in artifacts | Unsupported facts appear authoritative | Require evidence refs, `[inferred]`, `[RAJA]`, QA review. |
| Phase leakage | MVP/Phase 2 separation breaks | GTS-ROUTER and BA-EM-009 hard gate. |

## 12) Open decisions for RAJA/reviewers

| Decision | Owner |
| --- | --- |
| Approve, amend, or defer this HLD as the draft architecture baseline | RAJA / architect |
| Confirm target environment and deployment topology | RAJA / platform owner |
| Confirm model, region, retention, and residency | RAJA / security/privacy/platform owners |
| Name reviewer delegates for BA SME, Product Owner, QA, architecture, security/privacy, compliance/legal, platform, and tool owners | RAJA |
| Decide whether HLD output should become a maintained architecture baseline or remain a one-time draft | RAJA |
| Decide next executable HLD follow-up after review package | RAJA |

## 13) HLD verdict

This HLD is ready for QA as a draft/advisory architecture artifact. It establishes a coherent target design from existing repository evidence while preserving all current blocks: no sandbox execution, no live integration, no non-synthetic data use, no production deployment, no autonomous approval, and no external write-like side effects.
