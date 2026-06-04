# Project Status & Completeness Audit

**Azure Network Topology Expert Reviewer** · 2026-06-03 · audit of everything produced to date

## Verdict

**Design, specification, and the test corpus are complete, and the deterministic analysis core is now built and verified — the surrounding product (live data, transport, higher tools) has not started.** The hard, previously-unproven part — the reachability/severity engine — exists twice: a stdlib Python reference (`engine/reference/analyze.py`) and a Go port (`engine/go/`) in the planned package layout. Both pass the fixture golden suite (5/5); the Go port compiles, `go vet`s clean, and was verified in place on Go 1.25 (2026-06-03). Still 0% built: the Azure adapter (live Resource Graph + Network Watcher), the MCP transport, the cost/simulation/generation tools, attack-paths/drift, RAG, and UI integration. Nothing is broken or lost. The APIM inconsistency is resolved; a handful of pack-sync steps remain.

## Inventory (what exists, and where)

- **Project folder** — `aaraminds-projects/azure-network-topology-reviewer/` (moved here 2026-06-03): the use-case (`NetworkTopologyAdvisor.md`), `…build-plan.md`, `…architecture.md` (+ inline diagram), `…engine-plan.md`, `…STATUS.md`, `…workflow-diagram.md`, the two source `.pptx`, and the `engine/` tree (Go production port + Python reference).
- **Skills (spec + consistency layer)** — staged in `aaraminds/skill-staging/`: `azure-network-topology-analysis` **v1.1.0** (4 refs, validated), `azure-network-cost-forecasting` **v0.1.0** (3 refs, one eval, known gap), `azure-network-iac-generation` **v0.1.0** (3 refs, unevaluated). Plus `apply-skill.py`, `apply-skill2.py`, `apply-skill3.py`.
- **Eval harness** — `skill-staging/eval/`: 6 fixtures + answer keys, 3 HTML review viewers, `benchmark.md`, `run-haiku-uplift-eval.py`. All JSON valid.

## Completeness by layer

| Layer | State |
|---|---|
| Design & planning | **Complete** (APIM inconsistency resolved 2026-06-03) |
| Skill 1 — analyzer spec / consistency | **Validated, v1.1.0** — but the pack still has stale **v0.1.0** |
| Skill 2 — cost spec | Drafted, one eval; the inter-VNet peering-cost gap is open |
| Skill 3 — IaC spec | Drafted, **never evaluated** |
| Eval harness | **Complete** (fixtures, keys, viewers, benchmark, pinned-model harness) |
| **Product engine — analysis core** | **Built + verified** (Python reference + Go port; 5/5 golden tests; Go compiles & `go vet`s clean) |
| **Product engine — adapter, transport, higher tools** | **Not started** (Azure adapter, MCP transport, simulate/forecast/generate, attack-paths, drift) |

## Gaps & loose ends (found in the audit)

1. **Pack is out of sync with staging (mechanical).** Installed skill 1 = `v0.1.0` — it is **missing the DNAT fix and the iteration-2 fixes** that are in staging `v1.1.0`. Skills 2 and 3 are **not installed**. `INDEX.md` has **0** network entries (`skill_audit.py --emit-index` was never run). `Ranking.md` has skill 1 only, and its row reads `v1.0.0` (now `v1.1.0`).
2. **APIM inconsistency — RESOLVED (2026-06-03).** The use-case, architecture, and build-plan docs showed APIM as the AI gateway, redundant under AskAT&T. Dropped from all three: the MCP ingress is now Container Apps built-in auth (Entra), and AskAT&T governs model access. No APIM anywhere in the design.
3. **Skill 2 unfinished.** The inter-VNet peering cost (the baseline caught it, the skill missed it) and the matching answer-key gap are on the iteration-2 list — not done.
4. **Skill 3 unvalidated.** No eval run at all; it is a `v0.1.0` draft.
5. **The analysis core is built; the rest of the engine is not.** Per the eval arc, the capability lives in the deterministic engine — and that core now exists as code (`engine/reference/analyze.py` + `engine/go/`), proven on the fixture corpus. Still a plan, not code: the Azure adapter, the MCP transport, and the simulate/forecast/generate tools.

## To complete the project — sequenced

**A. Pack sync (you run these — the `.claude/` tree is write-protected in-session):**

- `python apply-skill.py` — push skill 1 `v1.1.0` to the pack (replaces the stale `v0.1.0`).
- `python3 skills-pack/validation/tools/skill_audit.py --emit-index` — regenerate `INDEX.md`.
- Update the skill 1 `Ranking.md` row to `v1.1.0`; add rows for skills 2/3 when they're promoted.

**B. Decisions to make:**

- **APIM** — DONE (2026-06-03): dropped from all three docs; the ingress is Container Apps Entra auth and AskAT&T governs models.
- **Skills 2/3** — apply now as pre-eval drafts, or hold until validated.

**C. Finish the consistency layer (only if you keep investing in the skills as docs):**

- Skill 2: iteration-2 fix (the peering cost) + a cost eval round.
- Skill 3: an eval round (it has had none).

**D. Build the product (the remaining work — per `engine-plan.md`):**

- **Done:** the P1 analyzer core + golden tests from the fixtures (the keystone) — `engine/reference/analyze.py` + `engine/go/internal/analyze`, 5/5 passing.
- **Next:** P0 Azure adapter (Resource Graph + Network Watcher → `graph.Fixture`) and the MCP transport (`analyze_risks`/`get_topology` over `mcp-go`); then P2 `simulate_change`/`forecast_cost`; then P3 `generate_topology`. Forced sequence unchanged — the analyzer the rest calls is now built.

## Bottom line

We have a complete design, a complete specification (the three skills), a complete golden-test corpus (the eval fixtures), and now **the verified analysis core itself** — the deterministic reachability/severity engine the eval arc pointed at, passing 5/5 on the fixtures in both the Python reference and the Go port. "Complete the project" now means: close the small loose ends (pack sync), then build *around* the proven core — the Azure adapter and the MCP transport first, then the cost/simulation/generation tools. The keystone is no longer a plan; it's code.
