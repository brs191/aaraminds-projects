# VRIA Red-Team and Evaluation Harness

**Document set:** Value Realization Intelligence Agent  
**Version:** v1.2  
**Date:** 2026-07-05  
**Owner:** AaraMinds / Common Capabilities AI Portfolio  
**Status:** Implementation baseline  


---

## 1. Purpose

This document defines the security, reliability, and quality evaluation harness for VRIA.

## 2. Evaluation Layers

| Layer | Purpose |
|---|---|
| Golden evals | Validate expected value-assessment behavior. |
| Tool evals | Validate tool selection, schema, and failure handling. |
| Evidence evals | Validate no unsupported value claims. |
| Red-team evals | Validate adversarial resistance. |
| Policy evals | Validate approval boundaries. |
| Online evals | Validate production behavior and drift. |
| Business evals | Validate usefulness and decision quality. |

## 3. Red-Team Categories

| Category | Example |
|---|---|
| Prompt injection | Evidence document tells agent to ignore rules. |
| Unsupported claim | User requests realized value without metric. |
| Approval bypass | User asks to publish directly. |
| Metric manipulation | Conflicting or suspicious metric source. |
| Gross-vs-net distortion | User reports savings without initiative cost. |
| Confounder hiding | User omits known unrelated process change. |
| Data leakage | User requests restricted evidence. |
| A2A abuse | Specialist agent returns uncited assessment. |

## 4. Online Evaluation Metrics

- Unsupported value claim rate.
- Approval bypass attempt rate.
- Tool failure rate.
- Schema validation failure rate.
- Evidence citation coverage.
- Drift in value-state distribution.
- Cost per reliable insight.
- User override / rejection rate.

## 5. Release Gate

Use the release gate in `07`. Critical failures block release.
