# Azure Network Topology Reviewer — Target Architecture

**Date:** 2026-06-02 · **See also:** `../NetworkTopologyReviewer-architecture.md` (Mermaid diagram + decisions)

---

## One-sentence design

A **deterministic graph engine** at the core — reachability, rules, severity — with the **LLM at the edges**
(explain, recommend, intent→spec), exposed as a **single MCP server** so the agent UI, CI/CD, and the
Azure Cost Optimizer peer all consume the same interface.

---

## Components

### MCP Server — the reusable interface

| Tool | Phase | Description |
|---|---|---|
| `get_topology` | v1 | Materialise `graph.Fixture` from live Azure sources |
| `analyze_risks` | v1 | Run deterministic engine + LLM explain layer; return structured report |
| `simulate_change` | v2 | Apply delta to in-memory graph; return security + cost delta |
| `forecast_cost` | v2 | Fixed (SKU exact) + variable (flow-log estimated) cost forecast |
| `generate_topology` | v3 | Intent → spec → modules → Terraform PR (validated before emit) |

### Deterministic Graph Engine (core — never the LLM)

- **Graph model** (`engine/go/internal/graph/`) — `Fixture` type: VNets, subnets, NSGs (effective rules),
  route tables (effective routes), NICs, public IPs, AVNM admin rules, Azure Firewall NAT
- **Analysis engine** (`engine/go/internal/analyze/`) — `Analyze()`: 4-gate reachability (AVNM → NSG → route → PIP),
  CIDR overlap, orphaned endpoints, tier segmentation
- **Simulation** (Phase 2) — apply delta to fixture, re-run `Analyze()`
- **Generation** (Phase 3) — spec → Terraform modules → validate via `Analyze()` before emit

### Azure data sources (read-only)

| Source | Used for |
|---|---|
| Azure Resource Graph (KQL) | Fast inventory: VNets, NSGs, route tables, PIPs, NICs |
| Network Watcher — Effective Security Rules | Evaluated (not declared) NSG rules per NIC |
| Network Watcher — Effective Routes | Evaluated routing table per NIC |
| Network Watcher — Topology API | Topology graph for hub/spoke discovery |
| Microsoft Defender for Cloud | Attack-path / internet-exposure signals (consume, don't reimplement) |
| Azure Retail Prices API | Fixed-cost SKU pricing (Phase 2) |
| VNet Flow Logs + Traffic Analytics | Variable-cost estimation (Phase 2; NOT NSG flow logs — deprecated) |
| Azure Cost MCP | Actuals reconciliation (Phase 2) |

### LLM at the edges (thin, never authoritative)

- **LangGraph orchestrator** (Python) — routes findings to AskAT&T for explanation + RAG synthesis
- **AskAT&T GenAI** — explain findings in natural language; synthesize grounded recommendations
- **Azure AI Search** — RAG on AT&T architecture standards; every recommendation cites a source clause
- LLM **never** decides severity, computes reachability, or authors Terraform

### Governance

- **Ingress:** Container Apps built-in auth (Entra ID) — no APIM
- **Identity:** Managed Identity with Reader + data-plane Network read (no write)
- **Model auth:** AskAT&T via JWT bearer; token acquired via Managed Identity or AskAT&T token service;
  secret in Azure Key Vault; never logged
- **Container registry:** JFrog Artifactory (AT&T standard)
- **Observability:** Azure Monitor + Application Insights on MCP server + engine

---

## Two modes, one engine

| Mode | Phases | Path |
|---|---|---|
| **Review** | v1 + v2 | Read-only: fetch → analyze → explain → report + escalation |
| **Generate** | v3 | Intent → spec → modules → render Terraform → **validate via analyzer** → PR |

The generator validates through the same `Analyze()` call the reviewer uses — they cannot diverge.

---

## What this is deliberately not

- Not rebuilding Microsoft Defender for Cloud — consume its signals where they overlap
- Not letting the LLM compute reachability, severity, or author Terraform
- Not granting any apply/write permission — every change leaves as a PR
- Not shipping unbaselined outcome metrics — measure before claiming numbers
