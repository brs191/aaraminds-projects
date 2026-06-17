# ADR-005 — JFrog Artifactory for distribution (not Azure ACR)

**Status:** Accepted
**Date:** 2026-06-13

## Context

The Go binary and VS Code `.vsix` package need to be distributed internally to AT&T engineers.

## Decision

JFrog Artifactory for all artifact distribution. Azure ACR is NOT used.

## Rationale

Azure ACR is an anti-pattern for AT&T use cases. AT&T's standard artifact repository is JFrog
Artifactory. This applies to:
- Go binary distribution
- VS Code `.vsix` package hosting
- Any Docker images (if the tool is ever containerized)

## Consequences

- Phase 5 CI/CD uses `jf` CLI (JFrog) not `az acr` commands
- GitHub Actions workflow uses `JFROG_ACCESS_TOKEN` secret, not Azure credentials
- Engineers install the tool from Artifactory, not from a public marketplace
