# Phase 1 — Mac setup & run (handoff runbook)

## 0. What moves, and what doesn't

- **MOVE:** the factory tooling — `baseline/`, `phase-0/`, `phase-1/` (small; all text + scripts).
- **DON'T move:** the `clear/` snapshot of `credit-routing-service`. On the Mac you point the extractor at your **real** checkout instead.
- **First check:** if `aaraminds-projects/` is your synced Cowork folder, these files are **already on this Mac** — skip §1, go to §2.

## 1. Transfer (only if not already synced)

- **Bundle:** unpack next to your repos —
  `mkdir -p ~/projects/aaraminds-projects && tar xzf repo-intelligence-factory-tooling.tar.gz -C ~/projects/aaraminds-projects/`
- **Git (recommended long-term):** make `repo-intelligence-factory/` a repo, push it, clone on the Mac — so it versions and re-syncs cleanly.

## 2. Toolchain (one-time)

```bash
brew install openjdk@17 maven        # JDK 17 + Maven (arm64-native on Apple Silicon)
export JAVA_HOME=$(/usr/libexec/java_home -v 17)
java -version    # -> 17.x
mvn -version     # -> uses 17
python3 --version  # 3.9+ (ships with macOS / brew); git + unzip are present by default
```

## 3. Point the extractor at your REAL repo

The extractor needs a **built** checkout at the frozen SHA (so the classpath + Lombok/JAXB bytecode exist):

```bash
cd <your-credit-routing-service>
git worktree add /tmp/crs-44b6b86 44b6b865     # clean checkout at the frozen SHA
cd /tmp/crs-44b6b86 && mvn -DskipTests package # -> target/ + the Spring Boot fat jar
```

(If your real repo has moved past `44b6b86`, use your current SHA — the extractor stamps whatever HEAD you point it at into every `source_ref`.)

## 4. Run Phase 1

```bash
cd ~/projects/aaraminds-projects/repo-intelligence-factory/phase-1/extractor
chmod +x run.sh
./run.sh /tmp/crs-44b6b86 ./_work
# -> _work/lib/*.jar (194 deps), _work/graph.json, then provenance_check.py prints PASS
```

First run pulls JavaParser 3.26.2 + Jackson from Maven Central (needs network once).

## 5. G7 — load into AGE + run the go/no-go

- **Azure (recommended; arch-agnostic):** `phase-0/age-benchmark/provision.sh` (needs `az login`) → dev Postgres 16 + AGE; regenerate `loader/load_age.py` against `_work/graph.json`; `psql -f schema/age_schema.sql && psql -f loader/load.cypher`; then `python3 phase-0/age-benchmark/benchmark.py --iterations 50`. **Gate:** p95 < 1500 ms at depth ≤ 3.
- **Local (Docker):** `docker run -d -p 5432:5432 -e POSTGRES_PASSWORD=pw apache/age`. Note: the `apache/age` image is x86_64 — on Apple Silicon it runs under Rosetta emulation (fine for a benchmark, just slower), or build AGE for arm64.

## 6. Confirm the move worked (no toolchain needed)

```bash
cd phase-1 && python3 eval/provenance_check.py && python3 query/demo_queries.py
```

These run on the committed thin-slice fixture with zero dependencies — green means the tooling transferred intact.

## Portability notes

- All scripts use **relative paths + CLI args** — no absolute sandbox paths baked in.
- `build_thin_slice.py` hardcodes the fixture SHA (`44b6b86`); the real extractor takes `--repo`/`--sha`, so your run stamps your repo + SHA.
- `run.sh` uses only BSD-compatible tools (`unzip`, `paste`, `ls`) — runs as-is on macOS.
