# BA Agent — Runtime Architecture and Security Design (Proposed)

Companion to `business-analyst-agent-requirements.md` (v0.4). Status: **proposed design for architect review — no reviewed source (S1–S6) defines deployment topology.** Sources fix three decisions: LangGraph orchestration, MCP tool access, and Teams/Copilot 365 as the surface. Everything else below is a concrete proposal on the AaraMinds pinned stack (Azure-primary; Entra ID; Key Vault; Container Apps/AKS; Grafana + Prometheus + OpenTelemetry; GitHub Actions OIDC; Terraform AzureRM) and is `[inferred]` unless cited.

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Runtime Architecture and Security Design |
| Version | 0.1 |
| Status | Proposed; requires architect, security, and platform-owner review |
| Prepared date | 2026-07-02 |
| Parent document | `business-analyst-agent-requirements.md` v0.4 |
| Sibling documents | `ba_agent_mcp_tool_contracts.md`, `ba_agent_evaluation_harness.md`, `ba_agent_operations_model.md` |
| Source-fixed constraints | LangGraph [S1:L11; S2:L30-L33]; MCP tools [S2:L34-L35]; Teams/Copilot 365 [S1:L11-L19; S2:L8-L9] |

## Topology

```
Teams / Copilot 365
      │  (user prompt / Adaptive Card actions)
      ▼
[Teams App / Bot channel registration]
      │
      ▼
┌─────────────────────────── Azure Container Apps environment ───────────────────────────┐
│                                                                                         │
│  orchestrator-svc (Python, LangGraph)          mcp-gateway-svc                          │
│  ── router + 4 capability graphs ─────────────▶ ── authN/authZ, scope enforcement,      │
│                                                    approval_ref validation, rate         │
│                                                    limiting, audit emission ──┐          │
│         │ state checkpoints                                                   │          │
│         ▼                                                                     ▼          │
│  [Postgres: LangGraph state,        [MCP servers: jira-mcp, git-mcp,   [audit events]    │
│   approval records]                  confluence-mcp, calendar-mcp,          │            │
│                                      teams-mcp — one container app each]    │            │
│                                                                             ▼            │
│  scheduler (Container Apps jobs) ──▶ health-monitor graph          Azure Event Hubs      │
│  Jira webhooks ──▶ ingress ─────────▶ health-monitor graph                 │             │
└─────────────────────────────────────────────────────────────────────────────┼───────────┘
                                                                              ▼
                                                    Postgres append-only audit store +
                                                    long-term archive (immutable blob)
```

Component decisions, with the tradeoff named:

- **Compute: Azure Container Apps, not AKS.** One team, ~7 services, event- and schedule-driven load. Container Apps gives KEDA-based scale-to-zero for the scheduler/webhook paths and removes cluster operations. Move to AKS only if the fleet grows past what Container Apps environments manage cleanly (many agents, service mesh needs). Deploy with `azurerm_container_app` under Terraform AzureRM.
- **Orchestrator: single `orchestrator-svc`** hosting the LangGraph router and the four capability graphs (standup, planning, retro, health) — one service, four graphs, matching the source's "four specialized nodes" [S2:L30-L33]. Do not split into four microservices; the graphs share state schema, identity, and release cadence, and splitting multiplies deployment surface for zero isolation benefit at MVP scale.
- **MCP gateway: separate `mcp-gateway-svc` in front of all MCP servers.** This is the control point that makes the contracts enforceable: scope checks, `approval_ref` validation against the approval store, rate limiting, and audit emission happen here — outside the LLM loop, so a compromised or confused agent cannot skip them. The gateway is the enforcement mechanism for BA-HIL-006 and BA-EM-005's zero-bypass gate.
- **State: Azure Database for PostgreSQL (Flexible Server).** LangGraph checkpoints (its Postgres checkpointer is first-party), approval records, and idempotency keys in one transactional store. Approval consumption is a single-row `UPDATE ... WHERE status='pending'` — atomic, no distributed-lock machinery. Cosmos DB is unnecessary here; nothing in this workload needs multi-region writes or schema-free documents.
- **Audit: append-only Postgres table + Event Hubs fan-out**, archived to immutable (WORM-configured) blob storage for retention. Audit writes are synchronous with the tool call at the gateway — a tool call whose audit write fails, fails.
- **Events: Azure Event Hubs** for Jira webhook ingestion buffering and audit fan-out. Webhook ingress validates Jira's signature before enqueueing.
- **Model access: Azure OpenAI via the orchestrator only.** MCP servers and the gateway never call the model. Model, prompt, and graph versions are stamped into every audit record.

## Identity and access (Entra ID)

| Principal | Type | Grants |
| --- | --- | --- |
| `ba-agent-orchestrator` | User-assigned managed identity | Call mcp-gateway; read Key Vault secrets scoped to orchestrator; Azure OpenAI inference; Postgres (state schema only). |
| `ba-agent-gateway` | User-assigned managed identity | Invoke MCP server apps; Postgres (approval + audit schemas); Event Hubs send. |
| `mcp-jira` / `mcp-git` / `mcp-confluence` / `mcp-calendar` / `mcp-teams` | One managed identity each | Only its own downstream credential from Key Vault; nothing else. Blast radius of any single credential = one system. |
| Teams bot registration | Entra app registration | Bot Framework channel only; approved channels per BA-OQ-003. |

Rules: no secrets in env vars or code — Key Vault references with managed identity only; downstream PATs/OAuth credentials (Jira, Git) are per-system, minimal-scope, and rotate on the cadence in `ba_agent_operations_model.md`; the requesting human's `user_id` rides in the audit record but the agent never impersonates users — approval authority is proven by the approval record, not by borrowed identity.

## RBAC matrix

| Role | Interact via Teams | Approve sprint plan | Approve Confluence publish | Modify tool scopes | Read audit log | Deploy |
| --- | --- | --- | --- | --- | --- | --- |
| Team member | ✓ | — | — | — | — | — |
| Scrum Master | ✓ | ✓ (BA-HIL-001) | ✓ | — | own-team records | — |
| BA SME / Product Owner | ✓ | — | ✓ | — | own-team records | — |
| Tool owner | — | — | — | ✓ (per system) | own-system records | — |
| Security/privacy owner | — | — | — | veto | full | — |
| Platform/delivery engineer | — | — | — | — | full | ✓ via pipeline only |

Role→person mapping is `[RAJA]` pending BA-OQ-002; enforcement is Entra group membership checked by the gateway on approval actions.

## Security design

- **Prompt-injection defenses (layered, because instruction-following filters alone fail):** (1) all tool-fetched content (ticket text, commit messages, page bodies) is wrapped in delimited data blocks and the system prompt instructs the model to treat it as data — necessary but not sufficient; (2) the gateway enforces a per-capability tool allowlist — the standup graph physically cannot call `publish_page` or `update_sprint_scope` regardless of model output; (3) writes require an `approval_ref` that only a human action in Teams can mint — injected text cannot fabricate one; (4) GTS-ROUTER/GTS-GATE adversarial cases regression-test all three layers per release.
- **Egress control:** Container Apps environment with restricted egress — allowlisted FQDNs only (Jira, Git provider, Atlassian, Graph API, Azure OpenAI). A prompt-injected exfiltration attempt has nowhere to send data.
- **Data handling:** classification rules per BA-OQ-010 gate what enters prompts; no restricted source material in prompts until sign-off (BA-DSPC-002). Calendar data is aggregated availability only — event subjects/attendees never leave the calendar MCP server (enforced in the server, not the prompt).
- **Secret redaction:** gateway scans tool responses for credential patterns before they reach the orchestrator; hits are redacted and logged as security events.
- **Retention:** audit records and prompts/completions retained per BA-OQ-014 `[RAJA]`; proposed default — audit 13 months hot + archive, completions 90 days — for security-owner review, not fact.
- **Tenancy:** single-tenant, single-team pilot. Multi-team rollout requires per-team scope partitioning in the gateway before onboarding a second team — do not defer this past the pilot.

## Environments and CI/CD

| Environment | Purpose | Data |
| --- | --- | --- |
| `dev` | Graph and contract development | Synthetic fixtures only |
| `staging` | Golden-set runs (release gate), sandbox Jira/Confluence projects | Sandbox tool tenants (per validation register) |
| `prod` | Pilot team | Live, approved scopes only |

GitHub Actions with OIDC federation to Entra (no stored cloud credentials); Terraform AzureRM (RBAC mode) for all infrastructure; container images in JFrog Artifactory; promotion `dev → staging → prod` gated by the evaluation harness run in staging (BA-EM-005 and BA-EM-009 hard gates block promotion). Prompt and graph changes ship through the same pipeline as code — no out-of-band prompt edits in prod.

## Observability

OpenTelemetry SDK in orchestrator, gateway, and MCP servers; traces span Teams request → router → graph → gateway → MCP server → source system, with `trace_id` written into audit records (satisfies BA-NFR-011's requirement that humans can review trigger → rationale → output). Prometheus metrics: request rate, routing distribution, tool latency/error rate by server, approval queue depth and age, gate-rejection count (any `approval_ref` rejection alerts — it is either an attack or a bug), token usage per capability. Grafana dashboards per audience: delivery (usage, approvals), platform (latency, errors, cost), security (gate rejections, redaction events, denied scopes). Alert routing and severities live in `ba_agent_operations_model.md`.

## Failure modes

| Failure | Behavior |
| --- | --- |
| Source system down (Jira/Git/etc.) | Gateway returns `degraded` per contract; agent reports partial data honestly; retry with exponential backoff + circuit breaker per MCP server. |
| Model unavailable / over quota | Request fails visibly in Teams with retry guidance; scheduled runs skip and log — never queue-and-replay stale summaries as fresh. |
| Approval store unreachable | All writes fail closed. Reads may continue. |
| Audit write failure | Tool call fails (audit is not best-effort). |
| Webhook flood | Event Hubs buffers; consumer rate-limits; duplicate events collapsed via idempotency keys. |
| Orchestrator crash mid-graph | LangGraph resumes from last Postgres checkpoint; side effects are safe to retry because writes are idempotent per contract. |

## Decisions the architect must confirm

Container Apps vs. AKS (fleet-growth assumption); single orchestrator service vs. split; Postgres for state + approvals (vs. any push for Cosmos); synchronous audit-write coupling; egress allowlist feasibility in the target tenant; Azure OpenAI model selection and region; retention defaults. Each is a named tradeoff above — reverse any of them with a recorded reason, not silently.
