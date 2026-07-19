# BA Agent G1 Readiness Evidence

This document records the Phase 1 engineering foundation evidence. It recommends readiness for RAJA/G1 review; it does not approve Phase 2, sandbox integration, live pilot use, production deployment, or live system-of-record access.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent G1 Readiness Evidence |
| Version | 0.1 |
| Status | Draft for RAJA/G1 review |
| Prepared date | 2026-07-03 |
| Accountable owner | RAJA |
| Execution prompts | [P1A], [P1B], [P1C], [P1D] |
| Technical baseline | `docs/development/phase-1-technical-baseline.md` |
| G0 package | `docs/planning/g0-readiness-package.md` |

## G1 readiness verdict

The Phase 1 foundation is ready for RAJA/G1 review. The repository now has a local Python package skeleton, safe local command surface, unit tests, typecheck command, no-live configuration check, gateway facade fake, and placeholder synthetic/eval command paths.

## Deliverables

| Area | Delivered path |
| --- | --- |
| Python package manifest | `pyproject.toml` |
| Local command wrapper | `Makefile` |
| Package source | `src/ba_agent/` |
| Tests | `tests/` |
| Fixture placeholder path | `tests/fixtures/.gitkeep` |
| Evaluation placeholder path | `eval/README.md` |
| Local development docs | `docs/development/local-development.md` |
| Technical baseline | `docs/development/phase-1-technical-baseline.md` |

## Command evidence

| Check | Command | Result |
| --- | --- | --- |
| Unit tests | `make test` / `PYTHONPATH=src python3 -m pytest` | 15 passed |
| Typecheck | `make typecheck` / `PYTHONPATH=src python3 -m mypy src tests` | Success: no issues |
| No-live config | `make no-live` / `PYTHONPATH=src python3 -m ba_agent check-config` | Local config accepted; live integrations disabled |
| CLI help | `make cli-help` / `PYTHONPATH=src python3 -m ba_agent --help` | Help printed successfully |
| Synthetic placeholder | `make synthetic-help` / `PYTHONPATH=src python3 -m ba_agent synthetic --help` | Help printed successfully |
| Eval placeholder | `make eval-help` / `PYTHONPATH=src python3 -m ba_agent eval --help` | Help printed successfully |
| Full local check | `make check` | Passed |

## Safety evidence

| Requirement | Evidence |
| --- | --- |
| Local/synthetic default | `RuntimeSettings.from_env({})` defaults to `environment="local"` and `live_integrations_enabled=false`. |
| Live-mode rejection | Tests reject `LIVE_INTEGRATIONS_ENABLED=true` and non-local environment. |
| No-network tests | Autouse pytest fixture blocks socket connections. |
| Gateway facade | `LocalGatewayFake` is explicitly a local contract-test fake, not production MCP gateway. |
| Write-like fail-closed | `LocalGatewayFake` denies write-like actions including sprint updates, publishing, Teams send/escalation, approval actions, calendar mutation, and Git mutation. |
| Typed boundaries | Pydantic models exist for settings, route decisions, graph state, gateway request/response, fixtures, eval cases, and Adaptive Card payloads. |
| Orchestrator boundary | `orchestrator.py` exposes offline model and graph-state placeholders only. |

## Explicit non-goals still in force

- No live Jira, Git, Confluence, Calendar, Teams, Copilot 365, Graph API, model, or MCP connectivity.
- No live system-of-record reads or writes.
- No Teams posting or escalation delivery.
- No sandbox integration.
- No cloud deployment, Terraform, container publishing, or registry configuration.
- No Phase 2 Enterprise BA capability implementation.

## Known follow-on work

| Next gate | Work |
| --- | --- |
| G2 / [F2] | Actual synthetic fixture schema/loading, standup summary, Adaptive Card payload, router integration, local synthetic demo, and minimal GTS-STANDUP/GTS-ROUTER seed evals. |
| G3 / [F3] | Gateway control hardening, `approval_ref` semantics, idempotency, audit records, trace propagation, and GTS-GATE hardening. |

## RAJA review note

G1 readiness means the local foundation exists and passes the documented local checks. It does not authorize Phase 2, sandbox, pilot, production, or live integrations.
