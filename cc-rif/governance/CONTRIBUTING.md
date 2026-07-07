# Contributing Guide

## Workflow

1. Create a feature branch from `main`.
2. Keep changes scoped to one capability or fix.
3. Run targeted tests for changed modules before opening a PR.
4. Open a PR with risk, rollback, and validation notes.

## Pull Request Requirements

- clear problem statement and scope
- evidence of tests/build checks for touched modules
- migration notes for schema/config/API changes
- backward-compatibility impact called out explicitly

## Repository Conventions

- Use JFrog Artifactory for container images (no Azure ACR for this repo).
- Keep deterministic extraction and graph-fact generation as the source of truth.
- Do not commit build artifacts or binaries.
- For phase playbook execution, update the relevant `PHASE_N_AGENTS.md` result status.
