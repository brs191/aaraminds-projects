# VRIA Operations Runbook

**Document set:** Value Realization Intelligence Agent  
**Version:** v1.2  
**Date:** 2026-07-05  
**Owner:** AaraMinds / Common Capabilities AI Portfolio  
**Status:** Implementation baseline  


---

## 1. Purpose

This runbook defines how VRIA is monitored, supported, released, and improved in production.

## 2. Operating Model

| Area | Owner |
|---|---|
| Product behavior | Product owner |
| Portfolio data | Portfolio lead |
| Runtime services | Engineering owner |
| Tool integrations | Tool/API owners |
| Security controls | Security/governance owner |
| Evidence sources | Data owners |
| Evaluation harness | AI quality/evaluation owner |
| Support | Platform/support team |

## 3. Production Metrics

| Metric | Purpose |
|---|---|
| Unsupported value claim rate | Safety and trust. |
| Approval bypass attempt rate | Governance control. |
| Tool failure rate | Runtime reliability. |
| Evidence coverage | Source-backed output quality. |
| Scorecard rejection rate | Output usefulness and quality. |
| Drift in state distribution | Detect score/model/data drift. |
| Cost per reliable insight | Cost-value discipline. |

## 4. Incident Types

- Unsupported claim published.
- Approval bypass.
- Wrong source-of-truth used.
- Metric corruption or stale data.
- Tool outage.
- Prompt injection escape.
- Model or scoring drift.

## 5. Rollback

Rollback must support prompt version, model version, scoring rules, tool contract, and scorecard publication. Published scorecards are superseded, not edited in place.
