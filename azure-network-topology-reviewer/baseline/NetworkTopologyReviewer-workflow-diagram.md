---
title: Azure Network Topology Reviewer Workflow
description: Flowchart for the Azure Network Topology Reviewer pipeline

## Analyze Phase Details
The Analyze phase inspects:
- Subnet structure and address space allocation
- VNet peering relationships and transitivity
- Route tables and effective routes
- Network security group (NSG) associations
- Firewall and gateway configurations
- Connectivity between critical workloads
This phase builds a comprehensive map of the network, highlights segmentation, and flags potential exposure points before security checks.
---

```mermaid
flowchart TD
    A[Start: Trigger Review] --> B[Fetch Network Topology Data]
    B --> C[Analyze Subnets, Peering, Routes, NSGs, Firewalls, Connectivity]
    C --> D[Check Security Groups & Rules]
    D --> E[Identify Misconfigurations & Risks]
    E --> F[Generate Recommendations]
    F --> G[Summarize & Report to Stakeholders]
    G --> H[End: Output Results]

    subgraph Optional AI Enhancements
        X[AI-driven Risk Assessment]
    end
    E -- Optional --> X
    X -- Results --> F
```
