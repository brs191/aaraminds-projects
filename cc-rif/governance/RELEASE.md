# Release Management

## Versioning

Use semantic versioning for platform releases:

- **MAJOR**: breaking API, schema, or contract changes
- **MINOR**: backward-compatible features
- **PATCH**: backward-compatible fixes

## Release Cadence

- Normal cadence: monthly platform release
- Out-of-band patch releases: security, production incidents, or severe regressions

## Release Readiness Gate

A release is eligible only when:

- changed services pass build/test gates
- schema migrations are reviewed and reversible
- deployment and rollback steps are documented
- security-impacting changes are explicitly reviewed

## Change Log Inputs

Every release note must include:

- included PRs and scope summary
- schema/config changes
- known risks and mitigations
- rollback trigger and rollback command path
