# Security Policy

## Reporting a Vulnerability

Do not open public issues for suspected vulnerabilities. Report findings privately to the repository owner (`@rb692q_ATT`) with:

- affected component and version/commit
- impact summary
- reproduction steps or proof of concept
- suggested mitigation (if available)

Acknowledgement target: within 2 business days.  
Initial triage target: within 5 business days.

## Supported Security Baseline

The following controls are required for production deployments:

- webhook authenticity validation for GitHub events (`X-Hub-Signature-256`)
- no long-lived cloud credentials in CI (OIDC for Azure, short-lived JFrog tokens)
- container images published to JFrog Artifactory only
- dependency and code scanning in CI before production rollout
- audit logging enabled for ingestion and change workflows

If a deployment cannot meet these controls, treat it as non-production.
