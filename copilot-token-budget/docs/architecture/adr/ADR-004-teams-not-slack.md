# ADR-004 — Microsoft Teams for alerts (not Slack)

**Status:** Accepted
**Date:** 2026-06-13

## Context

Budget threshold alerts need a push notification channel.

## Decision

Microsoft Teams via webhook. Not Slack.

## Rationale

AT&T engineers use Microsoft Teams as their primary communication tool. Slack is not available.
Teams supports incoming webhooks with Adaptive Card payloads — no SDK required, just an HTTPS POST.

## Consequences

- Teams webhook URL must be provisioned per-user or per-team
- Adaptive Card format is specific to Teams; not portable to Slack
- Phase 3 implementation: Go `net/http` POST to webhook URL; no Teams SDK needed
