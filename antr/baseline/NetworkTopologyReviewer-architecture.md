---
title: Azure Network Topology Expert Reviewer — Architecture
description: Component architecture. Deterministic graph engine at the core, LLM at the edges, exposed via an MCP server, with review and generate modes.
date: 2026-06-02
---

## How to read this

The **deterministic graph engine** is the center of gravity — graph model, analysis (reachability, rules, severity), simulation/forecast, and generation all operate on one cloud-neutral graph. The **LLM is at the edges**: it explains findings, synthesizes RAG-grounded recommendations, and translates architect intent into a spec. It never computes reachability or severity, and never authors network Terraform. Everything is reachable through one **MCP server**, so the agent UI, CI/CD, and the Cost Optimizer all consume the same interface. Two modes share the same core: **Review** (read-only, deployed topology) and **Generate** (architect intent → validated Terraform PR).

```mermaid
flowchart TB
    subgraph CONS["Consumers"]
        UI["AskAT&T Workflows (UI)"]
        CICD["CI/CD Pipelines"]
        COST["Azure Cost Optimizer (peer agent)"]
    end

    subgraph GOV["Governance edge"]
        INGRESS["Ingress auth — Entra (Container Apps)"]
        MI["Managed Identity + RBAC (read-only)"]
    end

    subgraph IFACE["MCP Server — reusable interface"]
        T1["get_topology"]
        T2["analyze_risks"]
        T3["simulate_change"]
        T4["forecast_cost"]
        T5["generate_topology"]
    end

    subgraph CORE["Deterministic Graph Engine (core)"]
        GRAPH["Cloud-neutral Topology Graph"]
        ANALYZE["Analysis Engine — reachability, rules, severity"]
        SIM["Simulation + Cost Forecast"]
        GEN["Design Generation — spec to modules to Terraform to validate"]
    end

    subgraph EDGE["LLM at the edges (thin)"]
        ORCH["LangGraph Orchestrator"]
        LLM["AskAT&T GenAI — explain, synthesize, intent to spec"]
        RAG["Azure AI Search — RAG on AT&T standards"]
    end

    subgraph SRC["Azure data sources — read-only adapters"]
        ARG["Azure Resource Graph (inventory)"]
        NW["Network Watcher — effective rules/routes, topology"]
        DEF["Defender for Cloud — attack-path signals"]
        PRICE["Retail Prices API"]
        FLOW["VNet Flow Logs + Traffic Analytics"]
        CMCP["Azure Cost MCP (actuals)"]
    end

    subgraph OUT["Outputs"]
        REP["Structured Report"]
        ESC["Escalation / Tickets"]
        PR["Terraform PR — GitHub Actions + OIDC"]
        AVNM["Azure Virtual Network Manager — connectivity + security admin rules"]
    end

    OBS["Azure Monitor + App Insights (cross-cutting)"]

    %% consumer to interface
    UI --> INGRESS
    CICD --> INGRESS
    COST --> INGRESS
    INGRESS --> IFACE
    MI -. guards .-> IFACE

    %% interface to core
    T1 --> GRAPH
    T2 --> ANALYZE
    T3 --> SIM
    T4 --> SIM
    T5 --> GEN

    %% core internal
    GRAPH --> ANALYZE
    ANALYZE --> SIM
    ANALYZE --> GEN
    GEN -. validate before emit .-> ANALYZE

    %% data sources feed core (read)
    ARG --> GRAPH
    NW --> GRAPH
    DEF --> ANALYZE
    PRICE --> SIM
    FLOW --> SIM
    CMCP --> SIM

    %% llm at edges
    ANALYZE -. uses .-> ORCH
    GEN -. uses .-> ORCH
    ORCH --> LLM
    ORCH --> RAG
    RAG --> LLM

    %% outputs
    ANALYZE --> REP
    SIM --> REP
    REP --> ESC
    GEN --> PR
    PR --> AVNM

    %% observability
    OBS -. monitors .-> IFACE
    OBS -. monitors .-> CORE

    classDef core fill:#0a4d68,stroke:#063b50,color:#ffffff;
    classDef edge fill:#f4a261,stroke:#c97f3f,color:#1a1a1a;
    classDef data fill:#e9f2f4,stroke:#0a4d68,color:#1a1a1a;
    classDef iface fill:#2a9d8f,stroke:#1d6e64,color:#ffffff;
    class GRAPH,ANALYZE,SIM,GEN core;
    class ORCH,LLM,RAG edge;
    class ARG,NW,DEF,PRICE,FLOW,CMCP data;
    class T1,T2,T3,T4,T5 iface;
```

## Legend / key decisions encoded in the diagram

- **Core (dark teal):** the deterministic engine. Reachability and severity are computed here, not by the model.
- **Edges (orange):** the only places the LLM runs — explanation, RAG-grounded recommendation, intent→spec. RAG always grounds the model on a versioned AT&T standard.
- **Read-only adapters (light):** all data ingress is read. The identity that runs review holds no write permission.
- **Interface (green):** every capability is an MCP tool, so review, cost, and generation are reusable by the UI, CI/CD, and the Cost Optimizer alike.
- **Write path is PR-only:** generation emits a Terraform PR through GitHub Actions + OIDC and targets Azure Virtual Network Manager for enforcement. The agent never applies a change.

## Phasing overlay

`get_topology` + `analyze_risks` ship in **v1** (review). `simulate_change` + `forecast_cost` ship in **v2** (cost-aware simulation). `generate_topology` ships in **v3** (design generation). See `NetworkTopologyReviewer-build-plan.md` for the forced sequence and exit criteria.
