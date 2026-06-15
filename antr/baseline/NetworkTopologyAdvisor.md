## Title

Azure Network Topology Expert Reviewer — AI-Driven Continuous Design Validation

## Shape

Agent — automates the network-design review workflow. Multi-step (fetch topology → analyze → check security → identify risks → recommend → report) but bounded to a single workflow with known steps. Promote to a Layer only when it absorbs a second domain (cost forecasting, compliance attestation) end to end.

## Business Problem

Azure environments at AT&T are highly distributed, security-sensitive, and cost-sensitive, and they change continuously as teams deploy and refactor workloads. Network topology — subnets, VNet peering, route tables, NSGs, firewalls, gateways — is one of the most complex and risky layers in the architecture, and manual reviews do not scale. The current practice presents several critical challenges:

1. **No Continuous Design Validation** — Topology reviews happen at project gates and during incidents, not continuously. Drift between the approved design and the deployed reality goes undetected until it surfaces as an outage, exposure, or audit finding.

2. **Late, Reactive Security Detection** — Misconfigured NSGs, over-permissive routes, transitive peering exposures, and orphaned public endpoints are typically found by security audits or red-team exercises, not by the teams that introduced them. Time-to-detect is measured in weeks.

3. **Cost Invisibility at Design Time** — Architects make topology decisions (hub-and-spoke vs full mesh, gateway SKU, firewall placement, private link footprint) without an automated cost projection. Expensive choices are committed before they are visible.

4. **Specialist Bottleneck for Reviews** — Network-architecture review depends on a small number of senior architects. Their backlog determines how fast new workloads can land in production, and routine reviews crowd out the high-value design work they should be doing.

5. **No Reusable Interface for AI Agents** — There is no programmatic way for AI assistants, the Azure Cost Optimizer, or CI/CD pipelines to query topology, simulate a proposed change, or get a structured risk verdict. Topology knowledge stays in slides and one-off scripts.

### Impact

These issues result in undetected security exposure, late-stage architecture rework, avoidable cloud spend on the wrong gateway and peering choices, and a review bottleneck that does not scale with the rate at which new Azure workloads are landing.

## Problem Solution

Build the **Azure Network Topology Expert Reviewer** — an AI-driven agent that fetches Azure network topology data, analyzes it across structure, security, and cost dimensions, simulates proposed changes, and emits structured recommendations on a continuous cadence and on demand. The agent runs against deployed environments and against pre-deployment topology proposals so risks are caught before they ship.

### 1. Topology Ingestion and Map Construction

Fetch network topology data through Azure Resource Manager and Network Watcher across in-scope subscriptions and assemble a comprehensive map: subnet structure and address-space allocation, VNet peering and transitivity, route tables and effective routes, NSG associations, firewall and gateway configurations, and connectivity between critical workloads. The map is the substrate every downstream analysis runs on.

### 2. Continuous Security and Risk Analysis

Run the topology map through a security-and-risk analysis stage that flags misconfigured NSG rules, over-permissive routes, transitive peering exposures, orphaned public endpoints, and missing segmentation between workload tiers. Each finding is tagged with severity and the owning resource group so remediation can be routed without a triage call.

### 3. AI-Driven Design Recommendations (RAG)

A Retrieval-Augmented Generation layer grounded on AT&T Azure architecture standards (via **Ask Docs**) produces design recommendations against the deployed topology — alternative peering patterns, gateway SKU choices, firewall placement, private-link adoption — with explicit rationale and links to the policy document each recommendation derives from. No bare LLM speculation.

### 4. Cost-Aware Simulation and Forecast

Before a topology change is approved, the agent simulates the change against the live map and produces a forecast: cost delta, security posture delta, blast-radius delta. Cost numbers are sourced through the same Azure Cost MCP Server used by the Azure Cost Optimizer so the two agents share one source of truth.

### 5. Stakeholder Reporting and Escalation

Emit a structured report with prioritized findings, recommended actions, and forecasted outcomes. High-severity findings auto-escalate to the network architecture team; medium and low findings ticket to the owning resource-group team via the existing workflow.

### Technology Stack

| Component        | Technology                                            |
| ---------------- | ----------------------------------------------------- |
| LLM / GenAI      | AskAT&T (enterprise GenAI platform)                                 |
| Orchestration    | LangChain / LangGraph                                 |
| Topology Data    | Azure Resource Manager, Network Watcher               |
| Cost Forecast    | Azure Cost MCP Server (shared with Cost Optimizer)    |
| RAG / Knowledge  | Azure AI Search (Ask Docs — AT&T architecture standards) |
| Authentication   | Azure Managed Identity + Azure RBAC                   |
| Governance       | Entra auth (Container Apps ingress); AskAT&T governs models |
| Observability    | Azure Monitor + Application Insights                  |
| User Interface   | AskAT&T Workflows                                     |

### Expected Outcomes

_Targets are pre-baseline estimates; each requires a measured baseline before submission._

- **Earlier security detection** — topology misconfigurations surfaced at design time and continuously post-deployment, not at audit time
- **Cost visibility at design time** — cost-delta forecast on every proposed topology change, before commit
- **Reduced review backlog** — senior architects spend their time on novel design, not on routine reviews
- **Consistent design enforcement** — recommendations grounded on a single, version-controlled set of architecture standards (RAG sources)
- **Reusable topology interface** — a structured topology+risk API that the Cost Optimizer, architecture reviews, and CI/CD pipelines can call

## Use Case Owner/Use Case Editor/Tech SME

## Use Case/Technology Solution Sponsor

## NDA?

## Proposed Solution Type

## Vendor/Third-Party Involvement: Do you plan to incorporate any vendor/third-party technologies, resources (including personnel), and/or services in the development or usage of this use case?

## Network Deployment/Service Prioritization: Will the proposed solution help determine where to deploy the network, services or repair prioritization?

## Primary Business Entity
