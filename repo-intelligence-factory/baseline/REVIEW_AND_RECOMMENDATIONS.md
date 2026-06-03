# Repository Intelligence Factory — Critical Review

**Reviewer:** Claude (principal-engineer review, AaraMinds voice)
**Date:** 2026-06-02
**Inputs reviewed:** `requirements-kickoff/{PRD, ARCHITECTURE, REPOSITORY_INTELLIGENCE_BLUEPRINT, IMPLEMENTATION_ROADMAP}.md`
**Method:** Doc assessment + three parallel 2026 research streams (technology state-of-the-art, competitive landscape, hard-problem/eval pressure-test). Sources at the end.

---

## Verdict

The *shape* of the plan is right: `ingest → parse → resolve symbols → embed → graph → hybrid-retrieve → agents → MCP` is the proven pattern, and real 2026 code systems are built this way. But four of the named technology choices are wrong or off-stack, one of them dangerously over-engineered; the success criteria are unfalsifiable and arithmetically suspicious; and — the hard part — **the platform as scoped is a commoditized, open-source-solved problem with no moat.** The exact stack you specced already ships as a 5.4k-star Apache-2.0 project (potpie), and free tools (DeepWiki) give away "ask questions about a repo + see its architecture" for $0.

So the real decision isn't *how to build it* — it's **what this factory is for**, which forks the whole strategy:

- **If it's internal leverage** (a machine that lets AaraMinds onboard onto inherited client brownfield codebases in hours instead of weeks): build it, but *assemble from OSS* and spend your effort on the Azure/brownfield/impact-analysis layer that's specific to your delivery work. This is the strong, defensible use of the idea and it fits the company's existing skills-pack (Azure microservices, PR review, SOC 2/ISO 27001).
- **If it's a product** (sell repo intelligence to others): don't build the general platform. It's commoditized. The only fundable wedge is **explainable, auditable change-impact analysis for large brownfield monorepos in self-hosted/air-gapped regulated environments** — and even then you stand on OSS plumbing and put 80% of effort into accuracy + audit, not into rebuilding Tree-sitter→graph.

The kickoff docs are at roughly 5% fidelity — section headers, not a plan. That's fine for a kickoff; this review is the pressure test before any code gets written.

---

## Decisions locked (2026-06-02)

Resolved with Raja in review:

- **Purpose: internal delivery leverage.** This factory accelerates onboarding onto AaraMinds' own and client brownfield codebases — it is not (for now) a product to sell. Optimize for least ops and stack-fit, not multi-tenant productization.
- **Agent framework: keep LangGraph.** A deliberate, blessed exception to the Go/Java-only backend rule. Isolate it as a Python microservice behind the MCP/HTTP boundary so Python doesn't leak into the core services. Bonus: LangGraph + a deterministic graph is the potpie stack, lowering the cost of assembling from OSS.
- **Graph DB: Apache AGE on Azure Database for PostgreSQL Flexible Server (managed, GA).** Both inputs converge: internal leverage + no-AKS/managed-only. AGE is GA and first-party managed on Azure (PG13–16), gives you openCypher, and co-locates graph + pgvector + metadata in one managed instance. Neo4j is now spike-only, not production.
- **Retrieval: hybrid, hand-rolled over AGE + pgvector — no GraphRAG framework.** Graph is built deterministically from AST/SCIP (not LLM-constructed). Query-time retrieval fuses three signals — pgvector (semantic) + Postgres FTS/BM25 (exact identifiers) + AGE Cypher traversal (structural, bounded-depth) — then reranks. All in one Postgres. Optional LLM-derived semantic edges are a later enrichment layer, not v1.
- **Compute: Azure Container Apps, not AKS.** "No AKS" means the Go MCP server, parsing/embedding workers, and the LangGraph service run on Container Apps (on-stack, serverless containers). If "no AKS" actually means "no containers at all / pure PaaS," flag it — a custom parsing/graph pipeline can't really live without a container runtime, and we'd need to talk.
- **Spike vehicle: potpie (Neo4j in local Docker), disposable.** Fastest path to real numbers; the Neo4j here is a throwaway dev-box container, not infra you operate. Production migrates to AGE behind a `GraphStore` interface.

**One open validation gate:** AGE's known weakness is unbounded variable-length traversal — exactly the impact-analysis "full blast radius" access pattern. Benchmark your top 5 real traversal queries against AGE at representative graph size. If AGE holds → done. If it chokes, the fallback *under the no-self-managed constraint* is **Cosmos DB Gremlin** (managed, no container; you lose Cypher, pay RU on multi-hop) — or, if you relax to Container Apps, **FalkorDB** (Cypher, code-graph-native, lighter than Neo4j). The fallback depends on what "no AKS" precisely rules out.

---

## What the docs get right

- **The pipeline topology.** Parse → symbols → embed → graph → retrieve → agents → MCP is exactly how CodeGraph, Cognee, and FalkorDB's code-graph are built in 2026. You didn't invent a weird architecture.
- **Tree-sitter for parsing** — correct and current (v0.26.8, May 2026). Fast, incremental, error-tolerant, every mainstream grammar is solid.
- **pgvector for embeddings** — correct, on-stack, and co-locatable with the graph (see AGE below).
- **An MCP server** — correct, and the single strongest decision in the stack. The official **Go MCP SDK** (`modelcontextprotocol/go-sdk`, v1.6.0, co-maintained with Google) is stable and on-stack. Build it in Go.
- **Static analysis creates facts; vectors add semantics; graph stores relationships** (Blueprint) — the right instinct. The execution details are where it goes wrong.

---

## The four corrections

### 1. Kill full GraphRAG. The code graph must be deterministic (AST/SCIP), not LLM-inferred.

This is the most important correction in the review. GraphRAG's whole value is *inferring* an entity/relationship graph from unstructured prose using LLM calls. **Code already gives you a precise call/import/symbol graph for free** from the compiler and indexers — paying an LLM to re-derive it is slower, more expensive, *and less correct*. The Jan-2026 head-to-head on Java codebases (arXiv 2601.08773) is decisive:

| Metric | AST-derived graph | LLM-extracted graph |
|---|---|---|
| Correct answers (of 45) | **43** (0 hallucinations) | 38 (5 wrong) |
| Graph build time | **2.8s / 13.8s** | 200s / 884s (~70–100× slower) |
| End-to-end cost | **2.1×** | **45.6×** |
| Node/chunk coverage | **0.90** | 0.64–0.73 (stochastic drops ~30%) |

Full Microsoft GraphRAG is research-grade for production economics: a corpus that costs <$5 to embed runs $50–200+ through GraphRAG, with four-figure indexing bills on large repos — and it goes stale, with painful incremental updates. Microsoft itself shipped *LazyGraphRAG* (~0.1% of the indexing cost) as the tell that full GraphRAG wasn't worth it.

**Do instead:** build the graph deterministically from AST + static analysis with typed edges (`calls`, `imports`, `extends`, `implements`, `injects`). Use **hybrid retrieval** — pgvector (semantic) + BM25/Postgres FTS (exact identifiers like function names and error codes, where pure vector fails) + deterministic graph traversal + a reranker. Use the LLM at *query time* for synthesis and to *enrich* nodes with summaries/intent — never to build the structural graph. If you later want semantic links the compiler can't see (code↔docs↔tickets), reach for LightRAG/LazyGraphRAG, not full GraphRAG.

### 2. Drop Semgrep from the extraction path. It's the wrong tool.

Semgrep is a security/lint pattern-matcher. Its Community Edition does **single-file, single-function** analysis only — it cannot trace calls across files — and cross-function taint moved behind the commercial platform in late 2024. Using it to populate a code knowledge graph is re-implementing a worse version of what proper indexers already produce.

**Do instead:**
- **SCIP indexers** (`scip-java`, `scip-go`, `scip-typescript`, `scip-python`) for compiler-accurate defs/refs/call edges. SCIP is the de facto 2026 standard for cross-repo code intelligence (it replaced LSIF). Note: GitHub's **Stack Graphs is archived (Sept 2025)** — don't build on it despite older blog advice.
- **Native package tooling** for module-level dependency edges (`go mod graph`, Maven/Gradle dependency tree, `uv`/pip resolution) — far better than parsing for `DEPENDS_ON`.
- **Tree-sitter heuristics** only as the fallback for languages without a SCIP indexer.
- Keep Semgrep/OpenGrep **only** if you separately want a security-findings feature — that's its actual job.

### 3. Swap Neo4j/Memgraph for Apache AGE (primary) / FalkorDB (escape hatch). Honor the Azure stack.

The Blueprint names Neo4j/Memgraph. Both are **off the AaraMinds approved stack** (Postgres+pgvector, MongoDB, Cosmos DB), and Memgraph is BSL-licensed (~$25k/yr), memory-bound, with documented stability concerns at scale.

- **Apache AGE (PostgreSQL extension) — primary.** Adds openCypher to Postgres. The big win: **graph + pgvector + metadata in one store**, so hybrid retrieval (correction #1) lives in a single operational surface and you can JOIN graph results against relational tables. Real 2026 production use exists (Trendyol migrated graph reads to AGE). **The catch:** unbounded variable-length traversals bypass Postgres indexes and degrade — which is exactly the "full blast radius of this change" access pattern. Design around it with bounded fixed-depth iterative queries.
- **FalkorDB — documented escape hatch.** Purpose-built for GraphRAG, Cypher-compatible, ships a code-graph product, far lighter on AKS than Neo4j. Source-available (needs license sign-off). Move here if deep-traversal performance becomes the bottleneck.
- **Cosmos DB Gremlin — fallback** only if mandated to first-party-managed-only. You trade Cypher for Gremlin, pay RU on multi-hop traversals, and adopt a service Microsoft is visibly steering away from (toward Graph in Fabric).

**Update (2026-06-02): AGE is GA and first-party *managed* on Azure Database for PostgreSQL Flexible Server (PG13–16)** — so the earlier "is AGE even managed on Azure?" caveat resolves in its favor, and given the locked inputs (internal leverage + no-AKS) **AGE is the production pick.** The benchmark is no longer AGE-vs-FalkorDB but a **go/no-go validation gate**: run your top 5 real traversal queries against AGE at representative scale. If it holds, done; if unbounded deep traversal chokes, fall back to Cosmos Gremlin (strict no-self-managed) or FalkorDB on Container Apps (if you relax that constraint). One operational note: AGE is excluded from Azure's in-place major-version upgrade path, so plan major PG jumps as dump/restore.

### 4. Keep LangGraph — as a deliberate, isolated exception.

Decided: keep LangGraph (and thus Python), overriding the Go/Java-only backend rule. It's the most mature agent runtime and a reasonable deviation. Two conditions keep it clean: (1) isolate it as a **separate Python microservice behind the MCP/HTTP boundary** so Python never leaks into the Go MCP server or Java services; (2) record the exception in `aaramind/.claude/CLAUDE.md` so it's blessed drift, not silent drift. Upside: LangGraph + a deterministic graph *is* the potpie stack, so this lowers the cost of forking/assembling from OSS rather than raising it.

---

## Net stack delta

| Layer | Specced | Recommendation |
|---|---|---|
| Parsing | Tree-sitter | **Keep** + AST-aware (cAST) chunking |
| Symbols / facts | Semgrep | **Replace** → SCIP indexers + native package tooling |
| Embeddings | pgvector | **Keep**; model = self-hosted `jina-code-embeddings-1.5b` (private source) or `voyage-code-3` (API ok) |
| Graph store | Neo4j / Memgraph | **Locked** → Apache AGE on managed Azure Postgres (GA); Neo4j spike-only |
| Retrieval | GraphRAG | **Replace** → hybrid (vector + BM25 + deterministic graph traversal + rerank) |
| Agents | LangGraph (Python) | **Keep** (decided) → isolated Python service behind MCP |
| MCP server | (unspecified lang) | **Keep**, build in Go (official SDK) |

With those four changes it's a coherent, on-stack, production-grade design. The shape was right; the component picks needed work.

---

## The competitive reality (read this before writing any code)

Every layer of the specced stack exists as free, Apache-2.0, runnable code, and one project already assembled the *whole thing* with your exact target capabilities:

- **potpie** (5.4k★, Apache-2.0, v1.1.0 May 2026) — "turns your codebase into a knowledge graph" in Neo4j via Tree-sitter, with prebuilt Q&A / debugging / impact ("blast radius") / spec agents and an MCP-style tool service. **This is your PRD, already built and open.**
- **DeepWiki** (Cognition) — free, no account: repo Q&A + architecture diagrams + dependency maps for 50k+ public repos, plus a no-auth MCP server. If your headline feature is "chat with a repo + see its architecture," it's already $0.
- **GitHub natively** — Copilot Spaces + knowledge bases give bundled repo Q&A to teams who already pay for GitHub. That's the gravity well that kills standalone Q&A.
- **The tell:** both **Greptile** and **Qodo** pivoted *away* from repo-Q&A to AI code review because Q&A wouldn't monetize. Cognition gives understanding away (DeepWiki) and monetizes *action* (Devin). When the best-funded players treat your headline feature as a loss-leader, it's commoditized.

**What is NOT commoditized** (i.e., where a wedge exists): high-accuracy *auditable* impact analysis / blast-radius (everyone does this badly — over-predicts or misses cross-service effects); truly air-gapped self-hosted for regulated industries (only Tabnine/Qodo seriously compete; Sourcegraph Cody's AI still needs cloud callbacks); architecture-level *reasoning* (not diagrams) at monorepo scale; and traceability/provenance for change-control. Note the overlap with AaraMinds' existing SOC 2 / ISO 27001 / Azure-brownfield positioning — that's not a coincidence, it's the wedge.

---

## The hard problems the docs gloss over

These are the make-or-break engineering realities. None are mentioned in the kickoff.

1. **Incremental re-indexing.** Re-parsing/re-embedding a repo per commit doesn't scale (whole-project SCIP indexing is minutes-to-hours). You need three-lane invalidation: re-parse only changed files; recompute graph edges for the changed file's 1-hop importers (Stack-Graphs principle: push cross-file resolution to query time); re-embed only chunks whose content hash changed (dedup by Git blob SHA, like GitHub Blackbird). Get this wrong and the factory can't keep up with an active repo.

2. **Call-graph recall has a hard ceiling.** Tree-sitter gives syntax, not resolved symbols. Real-world recall tops out around **70–88%** even for mature tooling — PyCG on real Python is ~99% precision but **~70% recall**; Java static analyses median ~0.88 recall; and ISSTA 2024's finding is that *ground truth for real programs is fundamentally unobtainable*. Dynamic dispatch, reflection, Spring DI, and cross-service calls are invisible to syntactic analysis. **Tier every edge with a confidence + evidence label** (`exact` / `probable` / `inferred`); never present the graph as complete.

3. **Impact analysis amplifies that imperfection transitively.** "What breaks if I change X" inherits the recall gap and then compounds it. Return *ranked, depth-bounded, confidence-scored* reachability, not a transitive closure (which on a hub node returns half the repo). Build a dedicated Spring/DI extractor — it's the highest-value language-specific investment for a Java shop. Intersect static impact with test coverage to recover dynamic edges.

4. **Scale: embedding cost/latency breaks first**, not parsing. ~20 bytes/LOC and ~20 ns/LOC means a 100M-LOC monorepo is ~2 GB of vectors and ~2 s/query naively. Quantize, embed at function granularity, keep the structural graph as the primary substrate, tier repos by size.

5. **The single biggest risk that sinks the project:** treating a sub-100%-recall graph as ground truth and shipping **silent false negatives**. If the tool says "nothing else is affected" and a reflectively-wired or cross-service caller wasn't in the graph, you've actively caused a production break and lost all trust. The mitigation is *product honesty* — "here's what I'm confident breaks, and here's what I can't see" — not a better algorithm.

---

## Rewriting the success criteria (currently unfalsifiable)

The PRD's targets have no definition, dataset, or measurement method, and two are arithmetically suspicious. Rewrite before committing:

- **"Repository Understanding ≥ 80%"** → On a frozen gold set of ≥300 human-authored Q/A pairs across ≥5 representative repos (SWE-QA style; tagged definition/usage/dataflow/cross-file/cross-service), correct on ≥X% overall *and* ≥Y% per type, LLM-judge validated to ≥90% human agreement. **Reality check: rigorous SOTA on SWE-QA is ~48% overall, so 80% is almost certainly mis-specified** unless your question distribution is much easier. Set X/Y from a measured baseline.
- **"Impact Analysis ≥ 70%"** → Split into **recall** (the safety metric — of entities that actually needed follow-up change per Git history, the fraction flagged) and **precision** (of flagged, the fraction real, human-sampled n≥100), reported *separately* per tier (same-language static / DI-wired / cross-service), because the achievable ceiling differs sharply. A single blended 70% is meaningless.
- **"Traceability = 100%"** → Redefine as *self-citation completeness*: 100% of emitted nodes/edges/answer-citations carry a resolvable `repo@commit:path:line` reference, asserted in CI. This is verifiable. It explicitly does **not** mean "the graph captures 100% of true relationships" — that's call-graph soundness, which is provably <100%. Conflating the two is the trap.

---

## Recommended next steps

1. **Fork decided: internal delivery leverage** (accelerate onboarding onto brownfield client/own codebases; not a product for now). Everything downstream optimizes for ops-simplicity and stack-fit over multi-tenant productization.
2. **Spike, don't spec.** Stand up potpie (or assemble SCIP + AGE + pgvector) against *one real brownfield repo you actually work on*, and measure: indexing time, call-graph recall on 20 hand-labeled edges, impact-analysis precision/recall on 10 real historical changes. One week, real numbers.
3. **Benchmark AGE traversal** on your top 5 impact queries at representative scale → settles AGE vs FalkorDB.
4. **Build the eval set first** (the rewritten criteria above), so every later decision is measurable rather than vibes.
5. **Pick the wedge** if it's a product: auditable impact analysis for regulated brownfield monorepos, on Azure, air-gappable. Build the wedge, adopt the plumbing.

---

## Sources

**Technology**
- Tree-sitter v0.26.8 — https://github.com/tree-sitter/tree-sitter/releases
- SCIP (replaces LSIF) — https://sourcegraph.com/blog/announcing-scip · https://github.com/sourcegraph/scip
- Stack Graphs (archived Sept 2025) — https://github.blog/open-source/introducing-stack-graphs/
- Semgrep data-flow limits / OpenGrep — https://semgrep.dev/docs/writing-rules/data-flow/data-flow-overview · https://appsecsanta.com/sast-tools/opengrep-vs-semgrep
- voyage-code-3 — https://blog.voyageai.com/2024/12/04/voyage-code-3/
- jina-code-embeddings — https://jina.ai/news/jina-code-embeddings-sota-code-retrieval-at-0-5b-and-1-5b/
- cAST AST-aware chunking (arXiv 2506.15655) — https://arxiv.org/abs/2506.15655
- Apache AGE vs Neo4j — https://dev.to/pawnsapprentice/apache-age-vs-neo4j-battle-of-the-graph-databases-2m4 · https://medium.com/trendyol-tech/migrating-graph-operations-to-apache-age-from-writes-to-reads-3b8334628e1c
- Cosmos DB Gremlin overview — https://learn.microsoft.com/en-us/azure/cosmos-db/gremlin/overview
- FalkorDB code-graph — https://github.com/FalkorDB/code-graph
- LazyGraphRAG — https://www.microsoft.com/en-us/research/blog/lazygraphrag-setting-a-new-standard-for-quality-and-cost/
- AST-derived vs LLM-extracted KG for code (arXiv 2601.08773) — https://arxiv.org/pdf/2601.08773
- Official Go MCP SDK v1.6.0 — https://github.com/modelcontextprotocol/go-sdk
- MCP spec 2025-11-25 / 2026 RC — https://modelcontextprotocol.io/specification/2025-11-25 · https://blog.modelcontextprotocol.io/posts/2026-07-28-release-candidate/

**Competitive**
- potpie — https://github.com/potpie-ai/potpie
- DeepWiki — https://cognition.ai/blog/deepwiki · https://github.com/CognitionAI/deepwiki
- Sourcegraph pricing/licensing — https://sourcegraph.com/pricing · https://devclass.com/2024/08/21/sourcegraph-makes-core-repository-private-co-founder-complains-open-source-means-extra-work-and-risk/
- Greptile — https://www.greptile.com/pricing
- GitHub Copilot Spaces — https://docs.github.com/en/copilot/concepts/context/spaces
- Augment context engine (scale) — https://www.augmentcode.com/context-engine
- Tabnine air-gapped — https://intuitionlabs.ai/articles/enterprise-ai-code-assistants-air-gapped-environments
- Qodo — https://www.qodo.ai/

**Hard problems & eval**
- GitHub Blackbird incremental index — https://github.blog/engineering/architecture-optimization/the-technology-behind-githubs-new-code-search/
- Glean incremental indexing — https://glean.software/blog/incremental/
- PyCG (Python call graphs, ~70% recall) — https://arxiv.org/pdf/2103.00587
- Total Recall? static call graphs (ISSTA 2024) — https://sse.cs.tu-dortmund.de/storages/sse-cs/r/Publications/Preprints/dyncg-issta-2024.pdf
- Call graph + runtime info (FSE 2025) — https://jacquesklein2302.github.io/papers/2025-FSE-IVR-call_graph.pdf
- Augment 100M-line quantized vector search — https://www.augmentcode.com/blog/repo-scale-100M-line-codebase-quantized-vector-search
- SWE-bench — https://github.com/swe-bench/SWE-bench
- SWE-QA (repo-level QA, ~48% SOTA) — https://arxiv.org/html/2509.14635v2
