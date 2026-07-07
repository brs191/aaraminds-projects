# VRIA Technical Solution Architecture

**Document set:** Value Realization Intelligence Agent  
**Version:** v1.2  
**Date:** 2026-07-05  
**Owner:** AaraMinds / Common Capabilities AI Portfolio  
**Status:** Implementation baseline  


---

## 1. Architecture Summary

VRIA is a governed portfolio intelligence system with a registry, evidence model, scoring workflow, dashboard, MCP/A2A integrations, approval workflow, evaluation harness, and ValueOps runtime.

| Layer | Decision |
|---|---|
| Channel | React ValueOps dashboard (ADR-03). Teams entry point post-MVP. |
| Orchestration | Deterministic Go workflow engine for intake, assessment, approval, and scorecard generation; LLM calls only in drafting/recommendation steps. |
| Backend | Go on Azure Container Apps (ADR-01). |
| Data | Azure Database for PostgreSQL Flexible Server — registry, hypotheses, assessments, approvals, decisions, audit. |
| Search/RAG | pgvector on the same PostgreSQL instance (ADR-02). |
| Tools/MCP | Registry, Jira/ADO, metrics, cost, document evidence, dashboard/reporting — Go MCP servers per `09`. |
| A2A | Post-MVP (ADR-04); adapter interface stubbed behind the tool layer. |
| Observability | OpenTelemetry → Prometheus / Grafana; traces correlated by `trace_id` from audit events. |
| Governance | Entra ID RBAC, workload identity per agent, policy gateway, audit logs, approval gates. |

## 2. Logical Architecture

```text
Users / Channels
  ├─ React ValueOps Dashboard
  └─ Microsoft Teams / Copilot entry point

Application Layer
  ├─ VRIA API Service
  ├─ Approval Workflow Service
  ├─ Scorecard Service
  └─ Admin / Configuration Service

Agent Orchestration
  ├─ Intake Workflow
  ├─ Evidence Retrieval Workflow
  ├─ Scoring Workflow
  ├─ Recommendation Drafting Workflow
  ├─ Approval Submission Workflow
  └─ ValueOps Feedback Workflow

Tool / MCP / A2A Layer
  ├─ Use-case Registry MCP
  ├─ Jira / ADO Status MCP
  ├─ Metric Snapshot MCP
  ├─ Cost / FinOps MCP
  ├─ Document Evidence MCP
  ├─ Dashboard Publishing MCP
  └─ Specialist Agent A2A Adapter
```

## 3. Deployment Architecture

```text
Azure subscription (per environment: dev / staging / prod)
  ├─ Resource group: rg-vria-<env>
  │   ├─ Azure Container Apps environment
  │   │   ├─ vria-api            (Go; HTTP API, orchestration)
  │   │   ├─ vria-scoring        (Go; deterministic scoring engine)
  │   │   ├─ vria-mcp-metrics    (Go MCP server, Tier 2)
  │   │   ├─ vria-mcp-evidence   (Go MCP server, Tier 2)
  │   │   └─ vria-dashboard      (React static + BFF)
  │   ├─ Azure Database for PostgreSQL Flexible Server (+ pgvector)
  │   ├─ Azure Key Vault          (secrets via managed identity; no connection strings in env)
  │   ├─ Azure Service Bus        (event contracts from `21`)
  │   └─ Azure Monitor workspace  (OTel collector → Prometheus / Grafana)
  └─ Identity
      ├─ GitHub Actions → OIDC federated credential → deploy (no PATs)
      ├─ Each Container App → user-assigned managed identity, least privilege
      └─ Users → Entra ID; roles from `10` section 3 as app roles
Terraform AzureRM (RBAC mode) provisions everything; state in azurerm backend.
```

### Architecture Decision Records

**ADR-01 — Go backend on Azure Container Apps.** Accepted. Scoring must be deterministic and auditable — a compiled, single-binary service with table-driven tests fits; the MCP servers are Go per the skills-pack standard, keeping one language across runtime. Rejected: Python (second runtime, weaker concurrency story for schedulers); AKS (cluster overhead unjustified at 10–50 users).

**ADR-02 — pgvector over Azure AI Search.** Accepted. Evidence corpus is small (hundreds of documents); pgvector keeps evidence, citations, and registry in one transactional store — citation pointers join directly to `evidence_sources`. One less service, one less identity boundary. Rejected: Azure AI Search (better at >100k docs and hybrid ranking; adopt only if corpus growth demands it — revisit trigger: >50k documents or >500ms p95 retrieval).

**ADR-03 — React dashboard first, Teams post-MVP.** Accepted. The seven views in `gate-d-operations/15` need approval queues and evidence drill-downs that a Teams card cannot carry; Teams becomes a notification/entry surface after pilot. Rejected: Teams-first (would force the approval UX into adaptive cards).

**ADR-04 — A2A post-MVP.** Accepted. No pilot exit criterion in `gate-d-operations/13` requires specialist agents; the A2A envelope in `09` section 4 is frozen as the contract, and a stub adapter satisfies the interface. Rejected: A2A in MVP (adds allowlist governance and provenance validation work with zero pilot value). Owner: engineering owner; revisit at pilot go/no-go.

## 4. Dependencies

- Physical data model: `19`.
- API/events: `21`.
- Tool contracts: `09`.
- Approval workflow: `18`.
- Scoring rules: `20`.
