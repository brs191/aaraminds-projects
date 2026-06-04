# Test Strategy — Scrum Master Agent

**Owner:** Raja · **Produced via:** `engineering:testing-strategy` skill · **Source:** code at `../code/`

The product's safety-critical invariant is the DOC — *no write without a recorded approval* ([Agent_Blueprint.md](../design/Agent_Blueprint.md) §6). Testing is weighted toward proving that invariant and the analysis correctness, not toward coverage vanity.

## Pyramid (target shape)

```
        /   E2E    \      1–2: docker compose up → brief → gate → audit row
       / Integration \    ~6: MCP contract, checkpointer pause/resume, audit chain
      /   Unit Tests  \   ~20: brief/report builders, status bucketing, time math
```

## By component

| Component | Test type | Focus |
|-----------|-----------|-------|
| `brief.py` / `build_report` | Unit (pure, no I/O) | Blocker/stale detection, time-based totals, missing-estimate flag, **every status bucketed** (incl. `Blocked`), TOC in report — `test_brief.py` |
| `gate.py` (DOC choke point) | Unit (pure, fakes) | `coerce_approved` fail-closed; `execute_decision`: approved→action, rejected→no action, delivery-failure→`failed` — `test_gate.py` (runs with nothing but pytest) |
| `jira-mcp` (Go) | Unit + contract | Tool schema (names/args), fixture JSON validity, time-tracking field present; contract = returned shape the orchestrator parses |
| `teams-adapter` (Go) | Unit | Stub mode (no webhook → not delivered); Adaptive Card envelope shape; non-2xx → error |
| `graph.py` | Integration (in-memory) | Real `interrupt()`/resume: pause→no write; resume approved→post; resume rejected/empty→**no write**; delivery failure audited; idempotent single recommendation — `test_doc_invariant.py` (MemorySaver + fake ports, no Postgres/HTTP) |
| `audit.py` | Integration (live PG) | `recommendation→approval→action_audit` rows; `fetchone()` None-guard; `fetch_pending` queue — **gap: needs a Postgres-backed run** |
| End-to-end | Smoke | `docker compose up` → brief logged → one `recommendation` + one `approval` + one `action_audit` row |

## Coverage targets

- **DOC paths: 100%** — approved-write, rejected-no-write, and delivery-failure-records-failed are all asserted. Non-negotiable.
- Analysis logic (`brief.py`): ~90% — it's pure and cheap.
- Adapters/glue: happy path + one failure path each. No coverage theater on framework code.

## Priority example cases

Status: ✅ = implemented and passing, ◻ = still to write.

1. ✅ **DOC — rejection writes nothing:** resume `{approved:false}` ⇒ no `action_audit` row; `approval.decision = rejected`. (`test_gate.py`, `test_doc_invariant.py`)
2. ✅ **DOC — delivery failure is audited, not silent:** publisher raises ⇒ `action_audit.result = failed`, run doesn't crash mid-chain. (both files)
3. ✅ **Idempotent resume:** the two-phase `ainvoke` (interrupt then resume) creates exactly **one** `recommendation` row (checkpointer replays completed nodes once). (`test_doc_invariant.py`)
4. ✅ **Status completeness:** a `Blocked` issue appears in the brief body, not only in the Blockers section. (`test_brief.py`)
5. ✅ **Fail-closed gate:** resume `{}` or `{approved:null}` ⇒ treated as reject. (`test_gate.py`, `test_doc_invariant.py`)

## Gaps to close (P1)

- **Live-Postgres E2E.** The DOC tests use an in-memory checkpointer + fake ports; add a `docker compose`-backed run that asserts the real `recommendation→approval→action_audit` rows and `fetch_pending` against Postgres (interrupt/resume semantics are checkpointer-agnostic, so this is about the SQL layer, not the gate).
- No load/rate-limit test against the Jira points-based model — add before real Jira wiring.
- No contract test pinning the MCP tool JSON shape across a mcp-go upgrade — add when bumping past v0.43.0.
- Auth/OAuth-3LO flow untested (parked with auth).
