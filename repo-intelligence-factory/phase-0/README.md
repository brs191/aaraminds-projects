# Phase 0 — Spike (runbook)

Working folder for executing Phase 0. The frozen plan is in `../baseline/PHASE_0_SPIKE_PLAN.md`; this is how to actually run it.

**Target repo:** `../clear/apm0045942-credit-routing-service` — Java 17 / Spring Boot, ~117k LOC, 2,705 commits.
**Goal:** answer two questions with real numbers before any platform code — *is it worth building?* and *does AGE hold?* — then lock the remaining decisions.

```
phase-0/
├── README.md                     ← you are here
├── evalset/
│   ├── understanding-goldset.csv ← fill: Q/A about the repo, tagged by type
│   └── impact-goldset.csv        ← fill: real changes + true downstream set (mine git)
└── age-benchmark/
    ├── provision.sh              ← az: dev Postgres Flexible Server (PG16) + AGE
    ├── setup.sql                 ← enable AGE + create the graph
    └── benchmark.py              ← run the traversal queries, report p50/p95
```

Prereqs: Docker, Git, Python 3.11+ with `uv`, `az` CLI (logged in), `psql`.

---

## Workstream A — Capability spike (potpie)  ·  build-vs-buy read

Stand up potpie on the target repo and see whether repo intelligence actually helps on *this* code — and whether potpie-as-is is already good enough.

```bash
git clone --recurse-submodules https://github.com/potpie-ai/potpie.git
cd potpie
cp .env.template .env
```

In `.env`, **keep the client code local** — use Ollama, not a cloud LLM (this is the governance-aligned path; nothing egresses):

```bash
LLM_PROVIDER=ollama
CHAT_MODEL=ollama_chat/qwen2.5-coder:7b
INFERENCE_MODEL=ollama_chat/qwen2.5-coder:7b
```

Then:

```bash
curl -LsSf https://astral.sh/uv/install.sh | sh
uv sync
chmod +x scripts/start.sh && ./scripts/start.sh      # docker + migrations + FastAPI(:8001) + Celery
curl -s http://localhost:8001/health
```

Parse the credit-routing-service (it's local and private — keep it local; see potpie docs for the local-repo parse call), then watch parsing finish:

```bash
curl -s 'http://localhost:8001/api/v1/parsing-status/<project-id>'
```

Run the eval set (Workstream B) through the **Codebase Q&A** and **Debugging / blast-radius** agents. **Record:** indexing time; answer correctness per question type; blast-radius quality on the historical changes. **Then write the build-vs-buy verdict:** where potpie is good enough as-is vs where it falls short — that gap is what you'd actually build. Stop with `./scripts/stop.sh`.

---

## Workstream B — Eval set (the measurement spine)

The reusable asset that makes every later phase testable. Start it immediately — it needs no infra.

- `evalset/understanding-goldset.csv` — aim for 50–100 human-authored Q/A, tagged `usage | dataflow | cross-file | cross-service`. Seeded with credit-routing examples; replace the `REPLACE:` cells with real answers + `repo@sha:path:line`.
- `evalset/impact-goldset.csv` — 15–25 real changes mined from the 2,705-commit history where follow-up commits / CI failures reveal the *true* downstream set. The seeded rows show the tiers that matter here: `static`, `cross-service` (SOAP), `inferred-aop` (AOP), `inferred-di` (Spring). These are exactly where naive call graphs fail.

**Scoring:** LLM-judge, but validate it first — on a 20-item calibration subset, require ≥90% agreement with two human raters before trusting automated scores. Report **per question-type**; don't average away the weak `how`/`where` cases. (Ask me to generate the scoring harness when the gold sets are filled.)

> Reality check: rigorous SOTA on repo-understanding (SWE-QA) is ~48%. A number near that on hard cross-file/impact questions is a real result, not a failure. Set your pass bar from a measured baseline, not aspiration.

---

## Workstream C — AGE traversal benchmark (the go/no-go)

The one test that decides the production graph store.

```bash
cd age-benchmark
export ADMIN_PASS='<strong-password>'
export MY_IP="$(curl -s ifconfig.me)"
./provision.sh                      # creates a throwaway PG16 + AGE server, prints the connect string

psql "host=<srv>.postgres.database.azure.com port=5432 dbname=repointel user=pgadmin sslmode=require" -f setup.sql

export PGCONN='host=<srv>.postgres.database.azure.com port=5432 dbname=repointel user=pgadmin password=*** sslmode=require'
pip install "psycopg[binary]"
python benchmark.py --generate --nodes 2500 --avg-degree 4    # repo-scale synthetic graph
python benchmark.py --iterations 50                           # p50 / p95 per query
```

For a faithful result, replace the synthetic graph with the real deterministic graph exported from Workstream A. **Measure** p50/p95 for callers, transitive dependents at depth 1/2/3, and blast-radius. Tear down when done: `az group delete -n rg-repo-intel-phase0 --yes --no-wait`.

---

## Pass / fail — set BEFORE running (no moving goalposts)

| Gate | Pass | Fail → action |
|------|------|---------------|
| **AGE** | impact queries within your interactive budget (e.g. `p95 < 1–2 s` [calibrate]) at depth ≤ 3 | over budget on a real-shaped graph → Cosmos Gremlin (strict managed) or FalkorDB on Container Apps |
| **Capability** | ≥ `X%` [set from baseline] of the gold set correct *with citations*; blast-radius useful on ≥ `Y` historical changes | weak → adopt potpie as-is, or narrow the wedge |
| **Build-vs-buy** | explicit verdict: adopt potpie vs build, and the size of the gap | — |

## Exit — the findings memo (one page)

Phase 0 is done when you can fill this in:

1. **AGE:** go / no-go, with the p50/p95 table.
2. **Capability:** per-type accuracy + impact precision/recall, and the build-vs-buy verdict.
3. **Decisions locked:** AGE vs fallback · embedding model (default self-hosted `jina-code-embeddings-1.5b`) · `scip-java` confirmed · build vs extend-potpie.

That memo unlocks Phase 1.
