# BA Agent Local Development

This repository now has a Phase 1 local/synthetic Python foundation. The project was documentation-only before this scaffold.

## Scope

- Local/synthetic only.
- No live Teams/Copilot 365 connectivity.
- No live Jira, Git, Confluence, Calendar, Graph API, model, or MCP calls.
- No system-of-record reads or writes.
- No cloud deployment, Terraform, container publishing, or registry configuration.

## Setup

The source layout uses `src/ba_agent`. You can run commands through `make`, which sets `PYTHONPATH=src`.

## Commands

| Purpose | Command |
| --- | --- |
| Run tests | `make test` |
| Typecheck | `make typecheck` |
| Full local check | `make check` |
| Validate local config | `make no-live` |
| CLI help | `make cli-help` |
| Synthetic placeholder help | `make synthetic-help` |
| Eval placeholder help | `make eval-help` |
| Synthetic standup demo | `make synthetic-demo` |
| Standup seed eval | `make eval-standup` |
| Router seed eval | `make eval-router` |
| Gate seed eval | `make eval-gate` |
| Planning seed eval | `make eval-planning` |
| Retro seed eval | `make eval-retro` |
| Health seed eval | `make eval-health` |
| Full MVP eval | `make eval-mvp` |

The underlying Python commands are:

```bash
PYTHONPATH=src python3 -m pytest
PYTHONPATH=src python3 -m mypy src tests
PYTHONPATH=src python3 -m ba_agent --help
PYTHONPATH=src python3 -m ba_agent check-config
PYTHONPATH=src python3 -m ba_agent synthetic --help
PYTHONPATH=src python3 -m ba_agent eval --help
PYTHONPATH=src python3 -m ba_agent synthetic STD-001
PYTHONPATH=src python3 -m ba_agent eval GTS-STANDUP
PYTHONPATH=src python3 -m ba_agent eval GTS-ROUTER
PYTHONPATH=src python3 -m ba_agent eval GTS-GATE
PYTHONPATH=src python3 -m ba_agent eval GTS-PLANNING
PYTHONPATH=src python3 -m ba_agent eval GTS-RETRO
PYTHONPATH=src python3 -m ba_agent eval GTS-HEALTH
PYTHONPATH=src python3 -m ba_agent eval GTS-MVP
```

## Safety defaults

`BA_AGENT_ENV` defaults to `local`, and `LIVE_INTEGRATIONS_ENABLED` defaults to `false`. Phase 1 rejects live mode.
