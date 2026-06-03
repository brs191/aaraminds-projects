#!/usr/bin/env python3
"""Phase 0 - Workstream C: AGE traversal benchmark (the go/no-go gate).

Measures p50/p95 latency of the impact-analysis traversal pattern on Apache AGE.
This is the single test that decides whether AGE survives as the production graph
store, or whether we fall back (Cosmos Gremlin / FalkorDB on Container Apps).

Usage:
  pip install "psycopg[binary]"
  export PGCONN='host=<srv>.postgres.database.azure.com port=5432 dbname=repointel user=<u> password=<p> sslmode=require'

  # 1) load a synthetic graph at the credit-routing-service's scale (~2.5k nodes):
  python benchmark.py --generate --nodes 2500 --avg-degree 4

  # 2) run the benchmark:
  python benchmark.py --iterations 50

For a faithful result, replace the synthetic graph with the REAL deterministic graph
exported from the potpie/Neo4j spike (Workstream A) - files/classes/functions plus
CALLS / IMPORTS edges. The synthetic graph only proves AGE's traversal engine in
isolation; the real graph proves it on real shape.
"""
import argparse, os, random, statistics, time
import psycopg

GRAPH = "codegraph"


def _prelude(cur):
    cur.execute("LOAD 'age';")
    cur.execute('SET search_path = ag_catalog, "$user", public;')


def setup_graph(conn):
    with conn.cursor() as cur:
        cur.execute("CREATE EXTENSION IF NOT EXISTS age CASCADE;")
        _prelude(cur)
        cur.execute("SELECT count(*) FROM ag_graph WHERE name = %s;", (GRAPH,))
        if cur.fetchone()[0] == 0:
            cur.execute(f"SELECT create_graph('{GRAPH}');")
    conn.commit()


def generate(conn, n, avg_degree):
    """Create n Function nodes and ~n*avg_degree CALLS edges, biased toward a few hubs."""
    with conn.cursor() as cur:
        _prelude(cur)
        for i in range(n):
            cur.execute(
                f"SELECT * FROM cypher('{GRAPH}', $$ CREATE (:Function {{fqn:'fn{i}', file:'F{i % 200}.java'}}) $$) AS (v agtype);"
            )
        edges = n * avg_degree
        for _ in range(edges):
            a = random.randint(0, n - 1)
            b = int((random.random() ** 2) * n) % n  # bias toward low indices => hubs
            if a == b:
                continue
            cur.execute(
                f"SELECT * FROM cypher('{GRAPH}', $$ MATCH (x:Function {{fqn:'fn{a}'}}),(y:Function {{fqn:'fn{b}'}}) CREATE (x)-[:CALLS]->(y) $$) AS (e agtype);"
            )
    conn.commit()


# The five real impact-analysis traversal shapes. {t} = a target function's fqn.
QUERIES = {
    "direct_callers":       "MATCH (c)-[:CALLS]->(:Function {{fqn:'{t}'}}) RETURN c",
    "dependents_depth2":    "MATCH (c)-[:CALLS*1..2]->(:Function {{fqn:'{t}'}}) RETURN DISTINCT c",
    "dependents_depth3":    "MATCH (c)-[:CALLS*1..3]->(:Function {{fqn:'{t}'}}) RETURN DISTINCT c",
    "forward_chain_depth3": "MATCH (:Function {{fqn:'{t}'}})-[:CALLS*1..3]->(d) RETURN DISTINCT d",
    "blast_radius_depth3":  "MATCH (c)-[:CALLS*1..3]->(:Function {{fqn:'{t}'}}) RETURN DISTINCT c",
}


def pick_targets(conn, k):
    with conn.cursor() as cur:
        _prelude(cur)
        cur.execute(f"SELECT * FROM cypher('{GRAPH}', $$ MATCH (f:Function) RETURN f.fqn $$) AS (fqn agtype);")
        rows = [str(r[0]).strip('"') for r in cur.fetchall()]
    random.shuffle(rows)
    return rows[:k] if rows else []


def run(conn, iterations):
    targets = pick_targets(conn, iterations)
    if not targets:
        raise SystemExit("graph is empty - run `--generate` first (or load the real export).")
    results = {}
    with conn.cursor() as cur:
        _prelude(cur)
        for name, tmpl in QUERIES.items():
            lat = []
            for t in targets:
                q = tmpl.format(t=t)
                start = time.perf_counter()
                cur.execute(f"SELECT * FROM cypher('{GRAPH}', $$ {q} $$) AS (x agtype);")
                cur.fetchall()
                lat.append((time.perf_counter() - start) * 1000.0)
            lat.sort()
            p95 = lat[max(0, int(len(lat) * 0.95) - 1)]
            results[name] = (statistics.median(lat), p95)
    return results


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--generate", action="store_true", help="load a synthetic graph then exit")
    ap.add_argument("--nodes", type=int, default=2500)
    ap.add_argument("--avg-degree", type=int, default=4)
    ap.add_argument("--iterations", type=int, default=50)
    args = ap.parse_args()

    conn = psycopg.connect(os.environ["PGCONN"])
    setup_graph(conn)

    if args.generate:
        print(f"generating {args.nodes} nodes, ~{args.nodes * args.avg_degree} edges ...")
        generate(conn, args.nodes, args.avg_degree)
        print("done.")
        return

    res = run(conn, args.iterations)
    print(f"\nAGE traversal benchmark  (graph={GRAPH}, {args.iterations} samples/query)")
    print(f"{'query':24}{'p50 (ms)':>12}{'p95 (ms)':>12}")
    print("-" * 48)
    for name, (p50, p95) in res.items():
        print(f"{name:24}{p50:>12.1f}{p95:>12.1f}")
    print("\nGate: compare against the target you set BEFORE running")
    print("(e.g. p95 < 1500 ms at depth<=3). Over budget on a real-shaped graph => AGE no-go.")


if __name__ == "__main__":
    main()
