# BA Agent G2 Readiness Evidence

This document records the Phase 2 synthetic standup thin-slice evidence. It recommends readiness for RAJA/G2 review; it does not approve gateway control hardening, sandbox integration, live pilot use, production deployment, or live system-of-record access.

---

## Document control

| Field | Value |
| --- | --- |
| Document name | BA Agent G2 Readiness Evidence |
| Version | 0.1 |
| Status | Draft for RAJA/G2 review |
| Prepared date | 2026-07-03 |
| Accountable owner | RAJA |
| Execution prompts | [P2A], [P2B], [P2C], [P2D], [P2E] |
| G1 evidence | `docs/development/g1-readiness.md` |

## G2 readiness verdict

The synthetic standup thin slice is implemented locally and is ready for RAJA/G2 review. The path loads synthetic Jira/Git/tool/eval fixtures, routes supported standup prompts, generates source-linked summaries, builds Adaptive Card JSON locally, and runs minimal GTS-STANDUP and GTS-ROUTER seed evals.

G2 verifies that no write-like tool is invoked. Audited write-rejection proof remains a G3 control-gate responsibility.

## Deliverables

| Area | Delivered path |
| --- | --- |
| Fixture models | `src/ba_agent/models.py` |
| Fixture loader | `src/ba_agent/fixtures.py` |
| Fixture seeds | `tests/fixtures/standup_cases.json` |
| Standup summary | `src/ba_agent/standup.py` |
| Adaptive Card builder | `src/ba_agent/cards.py` |
| Router | `src/ba_agent/router.py` |
| Local orchestration path | `src/ba_agent/orchestrator.py` |
| Seed evaluation runner | `src/ba_agent/evaluation.py` |
| CLI integration | `src/ba_agent/cli.py` |
| Tests | `tests/test_fixtures.py`, `tests/test_standup.py`, `tests/test_cards.py`, `tests/test_router.py`, `tests/test_evaluation.py`, `tests/test_cli.py` |

## Fixture manifest

| Field | Value |
| --- | --- |
| Fixture version | `synthetic-standup-v1` |
| Case IDs | `STD-001`, `STD-002`, `STD-003`, `STD-004`, `STD-005`, `RTR-002`, `RTR-003`, `RTR-004`, `RTR-005`, `PLN-001`, `RET-001`, `HLT-001`, `AMB-001` |
| Source file | `tests/fixtures/standup_cases.json` |
| Checksum | `sha256:3dda618a9c198fb30864ea7588b9d8c72da63fbc22586d127e5ca52765ef7d05` |

## Command evidence

| Check | Command | Result |
| --- | --- | --- |
| Unit tests | `make test` / `PYTHONPATH=src python3 -m pytest` | 33 passed |
| Typecheck | `make typecheck` / `PYTHONPATH=src python3 -m mypy src tests` | Success: no issues |
| Full check | `make check` | Passed |
| Normal demo | `make synthetic-demo` / `PYTHONPATH=src python3 -m ba_agent synthetic STD-001` | Adaptive Card JSON produced with `trace_id`, evidence refs, fixture version, and case ID |
| Degraded Git demo | `PYTHONPATH=src python3 -m ba_agent synthetic STD-002` | Output states Git data is degraded and does not invent commit/PR activity |
| Standup seed eval | `make eval-standup` | `GTS-STANDUP` passed across 7 standup cases |
| Router seed eval | `make eval-router` | `GTS-ROUTER` passed across 13 router cases with zero phase-separation violations |

## Acceptance evidence

| Criterion | Evidence |
| --- | --- |
| Standup prompt routes to standup graph | `GTS-ROUTER` seed eval passes; router tests cover standup prompts. |
| Unsupported/Phase 2 prompts are blocked | Router tests cover unsupported and Phase 2 requests; `GTS-ROUTER` reports zero phase-separation violations. |
| Output uses fixture evidence only | Summary/card tests assert evidence refs; fixture validation rejects non-synthetic refs. |
| Degraded Git honesty | `STD-002` has degraded Git status and no git activity; summary/card output reflects degradation. |
| Adaptive Card payload is local only | Card builder returns JSON and `send_adaptive_card_stub` fails closed. |
| No live writes | G2 path invokes no write-like tool; write-rejection/audit proof remains G3. |

## Explicit non-goals still in force

- No live Jira, Git, Confluence, Calendar, Teams, Copilot 365, Graph API, model, or MCP connectivity.
- No Teams posting.
- No system-of-record writes.
- No sandbox integration.
- No production deployment.
- No Phase 2 Enterprise BA capability implementation.
- No GTS-GATE hardening; that is owned by G3.

## RAJA review note

G2 readiness means the local synthetic standup thin slice is repeatable and evidence-linked. It does not authorize G3, sandbox, pilot, production, or live integrations.
