# cc-rif — Internal Review Feedback

> **Remediation status (2026-07-08):** C1, C2 fixed (with the C1 integration test now using the production mux). Embedding service hardened (batch cap, encode semaphore, per-batch persistence with retries in batch_cli). Agent + embedding services: logging added, exception details no longer leaked to clients (M13, M7). Rebrand complete: `com.att`/`github.com/att` → `com.aaraminds`/`github.com/aaraminds`, Java package dirs moved, `attoss.jfrog.io` + AT&T comments scrubbed, client handle removed from SECURITY.md, CODEOWNERS rewritten at repo root for the current layout ([VERIFY] placeholder handle). `.DS_Store` files and `dependency-reduced-pom.xml` deleted. Still open: H1–H10 (auth, shutdown/timeouts, CI relocation, git init), doc triage, remaining Medium/Low items.

**Date:** 2026-07-07
**Scope:** Full repo — services (Go/Python), extractors (Java), docs, governance, repo hygiene
**Method:** Four parallel review tracks; both Critical findings verified directly against source.

---

## Verdict

The engineering core is real and above average: deterministic extractors, parameterized SQL throughout, constant-time auth compares, provenance gates that refuse version swaps on degenerate runs. But the repo is not shippable. It carries two verified critical bugs, zero authentication on any network surface, AT&T client contamination throughout, dead CI, and a documentation set whose evidence system cites a repo layout that no longer exists. The migration moved the code but not the identity, governance, or doc integrity.

Nothing requires a redesign. This is identity, wiring, and hardening debt on a sound core.

---

## Critical (verified in code)

### C1. MCP server `/mcp` endpoint is broken for real MCP clients
`services/mcp-server/main.go:43-49` — every JSON POST is first decoded by `serveRawToolCall` (main.go:82-91), which consumes the request body. When `req.Method != "tools/call"` (i.e., `initialize`, `tools/list` — the first calls every MCP SDK client makes), it returns `false` and falls through to `streamableHandler.ServeHTTP` with an already-drained body. Standard MCP session setup fails.

The integration test (`app_test.go:181-220`) builds its own mux **without** the shim, so this path is untested and the bug is invisible to CI.

**Fix:** buffer the body (`io.ReadAll` + `r.Body = io.NopCloser(bytes.NewReader(buf))`) before sniffing, or route the raw shim to a separate path. Add an integration test against the production mux.

### C2. Agent narration silently degrades to a canned template
`services/agent-service/narrator.py:37-42` — the grounding check `if not any(ref in text for ref in citations)` requires the LLM to reproduce a full citation string verbatim (e.g., `search_code: apm0045942@sha:src/A.java:10 (exact)`). Combined with: default model `ollama/llama3.1:8b` (`config.py:11`) absent in most deployments, and a bare `except` that swallows every failure with no logging (the service has no `logging` import at all). Net effect: the service's core feature returns `f"{prompt} Citations: {top}."` and every test still passes — the LLM path has never been exercised by a passing test.

**Fix:** match grounding on `result_excerpt` substrings; instruct the citation format explicitly in the prompt; log every fallback event; fail loud on missing model config.

---

## High

### Security / exposure

- **H1. No authentication on any endpoint, any service.** MCP server `/mcp` (main.go), embedding `/embed` (`app.py:294-309`), agent `/explain` and `/investigate_impact` (`app.py:39-61`). Agent Dockerfile binds `0.0.0.0`. `/explain` triggers paid LLM calls per request — an exposed instance is an open spend amplifier plus repo-intelligence exfiltration path.
- **H2. Webhook reads an unbounded body.** `services/ingestion/handler/webhook.go:55` uses `io.ReadAll` with no `http.MaxBytesReader`, while `decodeBody` (helpers.go:30) correctly caps other endpoints at 1 MiB. And `GITHUB_WEBHOOK_SECRET` may be empty with only a startup warning (main.go:112-114) — the endpoint is then fully unauthenticated.
- **H3. MCP server has no HTTP timeouts, no graceful shutdown.** `main.go:53` bare `http.ListenAndServe` — Slowloris-exposed, no SIGTERM handling.

### Reliability

- **H4. Ingestion graceful shutdown deadlocks (Phase 5 on by default).** `services/ingestion/main.go:158-195` — on SIGTERM the signal goroutine calls `srv.Shutdown` and returns nil; errgroup context only cancels on non-nil error, so `queueWorker.Run` / `reconciler.Run` never stop → `g.Wait()` blocks → SIGKILL. Inverse bug at main.go:181-184: fatal worker error never calls `srv.Shutdown`, so HTTP serves forever with a dead worker. **Fix:** `signal.NotifyContext` parent, cancel in both paths before `Shutdown`.
- **H5. Pipeline runs on `context.Background()` with no timeout.** `service/index_service.go:87`, `service/incremental_service.go:60` — a hung `git clone` / `git fetch` / `java -jar` leaks the subprocess and wedges the repo permanently in `running` (blocked by `ErrIndexRunInProgress`, `store/run_store.go:155-167`). No stale-run reaper exists. **Fix:** per-stage or per-run `context.WithTimeout`.
- **H6. `explain_architecture` drops ctx, no timeout.** `services/mcp-server/app.go:325` — `http.Post` with `http.DefaultClient`; a hung agent-service call blocks the tool forever, uncancellable.
- **H7. `batch_cli.py` loses all completed work on one failure.** `batch_cli.py:117,147-149` — `asyncio.gather` (no `return_exceptions`) over all batches before any DB write; one failed `/embed` cancels siblings and writes nothing. All vectors held in memory simultaneously. **Fix:** write per completed batch (`asyncio.as_completed`), retry individual batches.

### Build / supply chain

- **H8. Agent Dockerfile ignores the lockfile and bakes the whole dir into the image.** `services/agent-service/Dockerfile:9-10` — `COPY services/agent-service /app` (any local `.env` → image layers; no `.dockerignore` exists anywhere in the repo), then `pip install .` resolves `>=` floors, not `uv.lock`. Non-reproducible builds; runs as root; no `HEALTHCHECK`. No Dockerfile at all for the embedding service — the one with the heavyweight model dependency.
- **H9. `/embed` accepts an unbounded batch.** `app.py:301-309` forwards the entire client payload upstream in one call; `Settings.batch_size` (`app.py:32`) is dead config. **Fix:** reject oversized batches (413/422), chunk upstream calls.
- **H10. Dead CI + no version control.** Workflows parked at `platform/ci/*.yml` — GitHub Actions only reads `.github/workflows/`. No `.github/`, no `.gitignore`, and cc-rif is not its own git repo (resolves to the parent monorepo) while every ops doc assumes a standalone remote. **Fix:** `git init`, root `.gitignore` (target/, .venv, __pycache__, .DS_Store, *.ndjson), relocate workflows.

---

## Client-branding contamination (fix before anything else)

152 `com.att` occurrences across `extractors/spring-java/**` and `extractors/core-java/**`; Go modules declared as `module github.com/att/rif/{ingestion,retriever,mcp-server}`; `@rb692q_ATT` as sole owner in `governance/CODEOWNERS` and security contact in `governance/SECURITY.md`; `platform/deploy/deploy-ingestion.yml` references `attoss.jfrog.io` (client-internal Artifactory) and an AT&T-specific ACR comment; `docs/architecture/TELECOM_STANDARDS_MAPPING_APPENDIX.md` is a pure client-stakeholder artifact.

The sibling `repo-intelligence-factory` already rebranded its extractor to `com.aaraminds.repointel` — cc-rif copied the **pre-rebrand** tree. If this repo is intended as AaraMinds IP, provenance is a legal question no document currently raises. One coordinated rename pass resolves this; also converge the four names in circulation (cc-rif / rif / repo-intelligence-factory / repointel) in the same pass.

---

## Medium

### Go services

- **M1.** `X-RIF-Repo-ID` header trusted but not HMAC-covered (`webhook.go:71`) — a replayed validly-signed payload can re-route jobs to an arbitrary repo_id.
- **M2.** Missing `X-GitHub-Event` header treated as a push (`webhook.go:45-46`) — require it to equal `push`.
- **M3.** Schema/impl mismatch: `tools.schema.json` marks `qualified_name`, `changed_entity`, `entity`, `component` required, but only `search_code` enforces non-empty (`app.go:208`). Empty `entity` reaches `nodeIDByQualifiedName` (`app.go:406-436`) where `ILIKE '%' || $2 || '%'` matches everything → silent garbage. Unescaped `%`/`_` act as wildcards. Schema is hand-maintained → drift guaranteed.
- **M4.** Audit-write failure fails the whole tool call (`app.go:237,263,298,337,373`) — log-and-continue for read-only tools.
- **M5.** `VectorSearch` can't use a pgvector index (`backend_pg.go:26-39`): `UNION ALL` with outer-only `ORDER BY distance` forces sequential scans. Push `ORDER BY embedding <=> $2 LIMIT $3` into each branch.
- **M6.** Rate limiting runs *after* the DB existence check (`app.go:379-395`) — swap the order.
- **M7.** `isDuplicateKeyError` string-matches "23505" (`run_store.go:567-569`) — use `errors.As(&pgconn.PgError{})`.
- **M8.** Ingestion Dockerfile runs as root; "No shell in production" comment (line 59) is false on alpine.
- **M9.** Dependency CVEs compiled into the binary: `go-git v5.12.0` (CVE-2025-21613/21614, fixed v5.13.0), `golang.org/x/crypto v0.31.0` (CVE-2025-22869, fixed v0.35.0) — via phase5 indirects. Run `govulncheck`, bump.
- **M10.** MCP `/health` is static (`main.go:39-42`) — never pings Postgres. Ingestion's `handler/health.go` does it right; mirror it.

### Python services

- **M11.** No concurrency limit on local embedding model (`app.py:243`) — anyio default allows 40 threads into one `SentenceTransformer.encode`; add `asyncio.Semaphore(1-2)`.
- **M12.** LangGraph usage adds risk without value (`agents.py:202-220`): try/except covers only the import; a runtime failure 500s instead of using the working sequential fallback. `pyproject.toml` permits `langgraph>=0.2.0` while lock pins 1.2.7 — three majors of drift allowed. The graph is a fixed linear 3-step chain (unused `"plan"` key at `agents.py:76,146`). Delete it or use it for real.
- **M13.** Internal exception text leaked to clients (embedding `app.py:309`, agent `app.py:48-60`, `mcp_client.py:52` embeds up to 400 chars of upstream body). Log internally, return generic detail.
- **M14.** `mcp_client.py:41-47` "Event loop is closed" catch is a test-harness hack in production code — root cause is module-level `app = create_app()` sharing one `AsyncClient` across loops. Fix lifecycle via lifespan. Also: zero retries/backoff on MCP calls.
- **M15.** Prompt-injection surface unfenced (`narrator.py:30`; `agents.py:120-124,189-191`): repo-derived excerpts and user input interpolated directly into the prompt. Currently masked by C2 (LLM output usually discarded); fixing C2 raises this severity. Fence citations as data.
- **M16.** embedding-service `pyproject.toml` likely not installable as declared: `[project.scripts]` with no `[build-system]`/`py-modules`; flat-layout auto-discovery fails on multiple top-level modules. [VERIFY with `pip install .`]
- **M17.** Zero logging in agent service; deprecated `@app.on_event("startup")` at embedding `app.py:290`; `trust_remote_code=True` (`app.py:85`) — pin the model revision.

### Extractors (Java)

- **M18.** `NodeIdComputer` duplicated between `spring-java/common` and `core-java`, held together by a Javadoc promise ("matches core-java exactly"). Divergence silently breaks Phase-2→Phase-1 edge joins. Extract a shared module or add a cross-module contract test.
- **M19.** Duplicate edge_ids emitted: `computeEdgeId(from,label,to)` has no injection-point discriminator and `EmitHelper.emit()` doesn't dedupe — two `@Autowired` fields of the same type produce identical edge_ids. Dedupe or include source line.
- **M20.** `resolveTypeFqn` fallback appends `"?"` to unresolved names (`SpringDiExtractor.java:349`) and mints node IDs no extractor will emit → dangling edges. Flag as `unresolved:true` instead of silently minting.
- **M21.** pom hygiene: no `project.build.outputTimestamp` (non-reproducible shaded jars — undercuts the project's own determinism story), no enforcer plugin, no aggregator parent (duplicate version pins across two poms; `run.sh` invokes mvn twice as workaround), `logback-classic 1.5.6` carries CVE-2024-12798/12801 (fixed 1.5.13+) — would trip the repo's own Anchore `high` gate. `extractors/core-java/dependency-reduced-pom.xml` is a committed build artifact — delete.
- **M22.** `StaticJavaParser.setConfiguration()` global mutable state in Di/Aop/CrossService extractors — breaks in-process concurrent use; switch to instance `JavaParser`.
- **M23.** `extractors/core-java/` and sibling `repo-intelligence-factory/phase-1/extractor` are diverging both ways (sibling rebranded, cc-rif added tests/resources). Declare cc-rif canonical, backport the rebrand, freeze the sibling.

### Documentation & governance

- **M24.** The architecture docs' evidence system is structurally unverifiable: 300+ citations of the form `(source: phase-1/...#Lxx)` point at the dead legacy layout. Counts: `TECHNICAL_DOCUMENTATION_CONSOLIDATED.md` 127, `rif_technical_document.md` 125, `ARCHITECTURE_DEEP_DIVE.md` 22, `SYSTEM_OVERVIEW.md` 14, `DOCUMENT_CONSISTENCY_MATRIX.md` 11 (which marks claims "Verified" against nonexistent paths), `PHASE_IMPLEMENTATION_STATUS.md` 9. Only `API_AND_TOOLING_REFERENCE.md` and `OPERATIONS_RUNBOOK.md` were re-pointed — use them as the pattern. `KNOWN_GAPS_AND_RISKS.md` rec #3 (doc CI path check) predicted exactly this failure and was never implemented — implement it.
- **M25.** Two near-identical "Consolidated" technical docs (722 and 415 lines, both v1.0, both 2026-07-01). Delete one.
- **M26.** Contradiction: `cutover-and-rollback-plan.md` (2026-07-01) says schema idempotency "Blocked"; `risk-register.md` R-003 and `compatibility-report.md` §4 (2026-07-02) record a full pass. The plan would fail its own gate check — refresh it.
- **M27.** CODEOWNERS triply broken: wrong location (`governance/` — GitHub reads root, `docs/`, or `.github/` only), rules target the dead phase layout, sole owner is a client handle. SECURITY.md routes vuln reports to the same client handle and mandates client-environment policy (JFrog-only images, GHE assumptions). CONTRIBUTING.md references nonexistent `PHASE_N_AGENTS.md` files.
- **M28.** False-assurance scaffolding: empty `tests/{unit,integration,e2e,perf,security,fixtures}` pyramid, empty `docs/runbooks/`, dead CI, unenforced CODEOWNERS — the repo looks more governed than it is.

---

## Low

- `dedupeStrings` O(n²) via `slices.Contains` (`webhook.go:233-248`) — use a map.
- `sanitizeQuery` strip-based tag filtering (`app.go:30,454`) is bypassable — its own test (`app_test.go:97-102`) shows residual `</tool>`. Treat outputs as data; drop the theater.
- Double embedding health check per run (`index_service.go:98-110`, :268).
- `envInt`/`envBool` silently swallow malformed values (`config/config.go:187-212`) — fail loud.
- Impact `Confidence` populated with `item.Tier` (`app.go:288-289`) — conflates concepts.
- `agents.py:84,148`: `top_k=self.max_hops` conflates hop depth with result count (silently caps at 3).
- `agents.py:116,184`: component-not-found raises `RuntimeError` → 500; should be 404.
- Embedding truncation is char-based (512 chars) vs the model's 512-token window (`app.py:237-239`) — wastes ~75% of context.
- `batch_cli.py`: row-by-row UPDATE instead of `executemany`; client-side `limit` slicing instead of SQL `LIMIT`; plain list passed to vector column — wrap in `pgvector.Vector`.
- `test_e2e.py:80-81` deletes the Go binary each run (full rebuild) and errors instead of skipping when `go` is absent.
- Agent `/health` static — never pings the MCP server it depends on.
- Five `.DS_Store` files (`docs/`, `extractors/`, `extractors/core-java/`, `data/`, `libs/`) — ironic given `scripts/repo_hygiene_check.sh` exists.
- `docs/ops/` migration debris: two literal agent-prompt files (`rif-migration-source.md`, `fleet-ready-migration-source.md`), a stale 281-line `final-tree.txt`, `pr-package.md` duplicating the cutover blocker table.
- `LombokUtil.isLombokGeneratedField` false-positives on explicit `log` fields in `@Slf4j` classes.
- Extractor tests are happy-path only: no constructor/setter injection, `@Inject`, Lombok-skip, or unresolved-type cases; `NodeIdComputerTest` is near-tautological; zero tests for `EmitHelper`, `SourceRefBuilder`, `LombokUtil`, `backend_pg.go`, ingestion handlers/pipeline, `batch_cli.py`.
- Bench (`retriever_bench_test.go`) measures in-memory fuse/BFS against static fakes — do not quote its numbers as service latency.

---

## What is genuinely well done (keep these patterns)

- Parameterized SQL everywhere, including pgvector literals as `$n::vector` params (`run_store.go:337`, `backend_pg.go:41`).
- `subtle.ConstantTimeCompare` for bearer tokens; `hmac.Equal` for webhook signatures. Secrets scan clean — no hardcoded credentials; `.env.example` placeholders only.
- B1 provenance gate and B2 degenerate-run guard (`index_service.go:211-230, 677-748`) — refusing a version swap on empty extraction is the right failure mode.
- `AtomicVersionSwap` with optimistic locking (`run_store.go:279-316`); `InsertRun` serialization via `SELECT ... FOR UPDATE`.
- Incremental fallback-to-full-reindex with explicit reasons (`incremental_service.go:201-222`); embedding client retry with ctx-aware backoff (`embedding_client.go:75-132`).
- Citation gates that fail closed (`agents.py:116,184`) and `min_length=1` on `source_refs` in response models — "no ungrounded answers" enforced structurally.
- `LiteLLMEmbedder` response validation (`app.py:198-229`) including the bool-is-not-float trap; `asyncio.to_thread` for blocking encode; deterministic `HashEmbedder` test fallback.
- Cross-language e2e harness spawning the real Go MCP binary with health-wait and SIGTERM cleanup (`test_e2e.py`).
- Deterministic extractors: sorted file walks, content-addressed IDs, stable JSON key order, honest completeness caveats on every edge.
- `docs/ops/move-log.md` and the self-honest No-Go cutover gating — above-average migration hygiene.
- `impact_test.go` / `rrf_test.go` — real behavior tests (hub damping, RRF ordering over a constructed graph). Embedding `test_embed_api.py` via `httpx.MockTransport` — behavioral, not mock-assertion theater.

---

## Priority order

1. **Rebrand** — `com.att` / `github.com/att` → `com.aaraminds.*`; scrub `attoss.jfrog.io` and AT&T comments; replace `@rb692q_ATT` in CODEOWNERS/SECURITY; move CODEOWNERS to repo root. One coordinated pass; converge naming while at it.
2. **Fix C1 and C2** — both are one-day fixes.
3. **Version control + CI** — `git init`, root `.gitignore`, move `platform/ci/*.yml` → `.github/workflows/`; delete `.DS_Store` files and `dependency-reduced-pom.xml`.
4. **Auth + lifecycle hardening** — API auth on all three services (H1), webhook body cap (H2), shutdown/timeout fixes (H3-H6), batch resilience (H7-H9).
5. **Doc triage** — delete one consolidated doc + migration prompt files; rewrite or delete the six legacy-cited architecture docs; refresh the cutover plan; add the doc-path CI check.
6. **Structural debt** — unify `NodeIdComputer` (M18), edge-id dedupe (M19), bump `go-git`/`x/crypto`/`logback` (M9, M21), fill the named test gaps.

Items marked [VERIFY] were not empirically confirmed in the review environment.
