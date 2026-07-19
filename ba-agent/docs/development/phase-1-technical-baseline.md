# BA Agent Phase 1 Technical Baseline

This baseline constrains Phase 1 implementation before source scaffolding begins. It is the contract for [P1A] through [P1D] and remains local/synthetic only.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent Phase 1 Technical Baseline |
| Version | 0.1 |
| Status | Draft baseline for F1 execution |
| Prepared date | 2026-07-03 |
| Accountable owner | RAJA |
| Execution prompt | [P1T] |
| G0 evidence | `docs/planning/g0-readiness-package.md` |
| Planning baseline | `docs/planning/project-development-plan.md` v0.3 |

## Scope

Phase 1 creates a runnable local foundation only. It may create package structure, local commands, typed contracts, safe defaults, tests, and placeholder fixture/evaluation paths. It must not implement live Jira, Git, Confluence, Calendar, Teams, Copilot 365, Graph API, Azure OpenAI, or MCP connectivity.

## Required source layout

Use this layout unless a later prompt records a justified `[inferred]` deviation:

```text
src/
  ba_agent/
    __init__.py
    __main__.py
    cli.py
    config.py
    gateway.py
    models.py
    orchestrator.py
    py.typed
tests/
  fixtures/
  test_*.py
docs/
  development/
    phase-1-technical-baseline.md
    g1-readiness.md
eval/
  README.md
```

The package name is `ba_agent`. The CLI entry point is the module form:

```bash
PYTHONPATH=src python3 -m ba_agent --help
```

## Packaging and command contract

Use a simple `pyproject.toml` with setuptools [inferred]. Phase 1 may depend on `pydantic` for typed boundary models and dev tools `pytest` and `mypy`. No lockfile is required in Phase 1; a future lockfile decision remains `[RAJA]`.

Canonical commands:

| Purpose | Command | Required by |
| --- | --- | --- |
| Tests | `make test` (`PYTHONPATH=src python3 -m pytest`) | [P1A] |
| Typecheck | `make typecheck` (`PYTHONPATH=src python3 -m mypy src tests`) | [P1C] |
| Local CLI help | `make cli-help` (`PYTHONPATH=src python3 -m ba_agent --help`) | [P1B] |
| No-live config check | `make no-live` (`PYTHONPATH=src python3 -m ba_agent check-config`) | [P1C] |
| Synthetic placeholder | `make synthetic-help` (`PYTHONPATH=src python3 -m ba_agent synthetic --help`) | [P1C] |
| Evaluation placeholder | `make eval-help` (`PYTHONPATH=src python3 -m ba_agent eval --help`) | [P1C] |

If a `Makefile` is created, it must wrap only existing commands and must remain local-only:

```bash
make test
make typecheck
make check
```

## Type and model discipline

Use Pydantic models for boundary data. Required boundary models:

- `RuntimeSettings`
- `RouteDecision`
- `GraphState`
- `GatewayRequest`
- `GatewayResponse`
- `FixtureRecord`
- `EvalCase`
- `AdaptiveCardPayload`

Cross-module contracts should use typed models, protocols, enums, or explicitly typed collections. Avoid untyped cross-module `dict[str, Any]` contracts. If a generic mapping is unavoidable, record the reason as `[inferred]` in code comments or docs.

## Runtime defaults and safety

Defaults:

| Setting | Required default |
| --- | --- |
| `BA_AGENT_ENV` | `local` |
| `LIVE_INTEGRATIONS_ENABLED` | `false` |
| Network access in tests | Disabled by design; no test should require network |
| Secrets | No secrets required for tests or local commands |
| Live mode | Rejected in Phase 1 |

The config layer must fail closed if `LIVE_INTEGRATIONS_ENABLED=true` or an equivalent live-mode flag is requested.

## LangGraph-compatible contracts [inferred]

Do not require the LangGraph dependency in Phase 1. Expose compatible local contracts:

- `GraphState` with route, trace ID, graph version, evidence refs, and data-quality fields.
- `RouteDecision` with route, confidence/score placeholder [RAJA], reason, and blocked/unsupported flags.
- A placeholder transition interface that can be replaced by LangGraph nodes later.
- Graph version stamping for local outputs.

## Offline model and tool seams

Define protocols/fakes, not live clients:

- `ModelClient` protocol with a local fake implementation only.
- `GatewayFacade` protocol or class boundary.
- `LocalGatewayFake` for contract testing.
- No direct orchestrator mutation of systems of record.

## Gateway facade baseline

The Phase 1 gateway facade is a **local contract-test fake**, not the production MCP gateway. It must:

1. Keep read/write semantics explicit.
2. Deny all live tool calls.
3. Fail closed for write-like operations.
4. Return machine-readable local statuses such as `denied`, `degraded`, or `blocked`.
5. Preserve an interface that later Phase 3 control hardening can extend.

## Minimum Phase 1 tests

Phase 1 must include tests for:

1. Package import.
2. CLI help.
3. Config defaults.
4. Live-mode rejection.
5. Gateway facade write-fail-closed behavior.
6. No required secrets.
7. Placeholder synthetic/eval command help.
8. Typecheck command availability.

## Evidence and marker policy

- Use source citations for product or planning claims.
- Use `[inferred]` for reasonable unsupported implementation choices.
- Use `[RAJA]` for owner-dependent decisions, thresholds, dates, scopes, names, or approvals.
- Do not introduce the legacy marker.

## Explicit non-goals

- No live integrations.
- No model calls.
- No cloud deployment.
- No Terraform or container publishing.
- No system-of-record writes.
- No Teams posting.
- No Phase 2 Enterprise BA capability implementation.
