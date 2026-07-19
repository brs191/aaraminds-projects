# DIF Evaluation

Evaluation artifacts, fixtures, and golden sets live here.

Planned areas:

| Path | Purpose |
|---|---|
| `fixtures/` | Synthetic fixtures for parser, source-anchor, and RIF compatibility tests. |
| `golden/` | Golden corpus, golden queries, expected anchors, and expected caveats. |
| `p0-evaluation-plan.md` | P0 evaluation plan and release gates. |
| `source_anchor_roundtrip.py` | Stdlib-only P0 harness for golden source-anchor round-trip checks. |
| `json_caveat_checks.py` | Stdlib-only P0 harness for JSON caveat coverage and failure behavior checks. |
| `rif_compatibility_checks.py` | Stdlib-only P0 harness for ADR-016 RIF compatibility fixture checks. |
| `search_docs_checks.py` | Stdlib-only P0 harness for anchored `search_docs` contract checks. |
| `audit_usage_checks.py` | Stdlib-only P0 harness for audit/usage schema dimensions, separate write records, MCP call metering, and safe record content. |
| `degenerate_run_checks.py` | Stdlib-only P0 harness for ingestion-run promotion safety and degenerate-run guard behavior. |
| `run_p0.py` | Stdlib-only P0 golden evaluation runner that executes the Go component/full/build gates and all Python scaffold harnesses, then reports measured run metrics. |
| `path_checks.py` | Stdlib-only P0 harness for required path existence and CI baseline safety checks. |

Do not invent quality targets before baselines exist.

## Current validation commands

Run the full P0 golden evaluation gate from the repository root:

```bash
python3 evaluation/run_p0.py
```

Optional measured-result JSON can be written outside the repository or to a caller-chosen artifact path:

```bash
python3 evaluation/run_p0.py --json-output /tmp/dif-p0-results.json
```

Run source-anchor round-trip checks from the repository root:

```bash
python3 evaluation/source_anchor_roundtrip.py
```

Run JSON caveat checks from the repository root:

```bash
python3 evaluation/json_caveat_checks.py
```

Run RIF compatibility checks from the repository root:

```bash
python3 evaluation/rif_compatibility_checks.py
```

Run `search_docs` anchored retrieval contract checks from the repository root:

```bash
python3 evaluation/search_docs_checks.py
```

Run audit/usage write contract checks from the repository root:

```bash
python3 evaluation/audit_usage_checks.py
```

Run degenerate-run guard checks from the repository root:

```bash
python3 evaluation/degenerate_run_checks.py
```

Run path and CI baseline safety checks from the repository root:

```bash
python3 evaluation/path_checks.py
```
