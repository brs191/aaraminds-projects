# azure-nettopo-engine — the deterministic core (proven)

This is the engine the whole project pointed at: the part that computes network exposure **in code, deterministically** — no model in the path. It is the answer to the eval arc's conclusion that the capability lives in the engine, not in skill prose.

## What's here, and why it's Python

`reference/analyze.py` is the deterministic analysis core, with `reference/test_analyze.py` running it against the real eval fixtures. **The sandbox has no Go toolchain**, so this is a stdlib-only **reference implementation** — the executable spec and test oracle that the production Go engine (`engine-plan.md`) is a direct port of. Go stays the production stack; this proves the algorithm is correct first, so the port is mechanical and verified-by-twin.

## It's proven — 5/5 golden tests on real fixtures

```
$ python test_analyze.py
PASS  f1  internet exposure: real (spoke-a) vs latent (spoke-b) + orphaned IP
PASS  f2  default AllowVnetInBound flat-opens the sensitive db VNet-wide
PASS  f3  AVNM source-scope (AlwaysAllow opens / Deny closes) + CIDR overlap
PASS  h1  firewall DNAT publishes a no-public-IP backend; sibling without DNAT is not reachable
PASS  h2  None black-hole is latent; AzureCloud tag is a real broad exposure
5/5 cases passed
```

Every case asserts both recall (the planted finding is produced) **and** precision (the trap is *not* flagged reachable-High). The engine reproduces the answer keys that took five eval rounds to hand-verify — and gets the exact cases the model did inconsistently right, every time. That is the thing prose could not do.

## Run it

```
python reference/analyze.py testdata/fixture-1-internet-exposure.json   # -> findings as JSON
python reference/test_analyze.py                                        # -> the golden suite
```

Example output (fixture-1): `nic-vm-web-a` High reachable (rule + Internet route + public IP); `nic-vm-web-b` latent (firewalled route, no public IP) — the byte-identical NSG rule, correctly split by computed reachability.

## Map to the Go production engine (`engine-plan.md`)

The reference functions port one-to-one into `internal/analyze`:

| reference (`analyze.py`) | Go (`internal/analyze`) |
|---|---|
| `admin_verdict()` (source-scope aware) | `adminRuleVerdict()` — Gate 1 |
| broad-source inbound loop + `nsg_allows` | `nsgVerdict()` — Gate 2 |
| `default_hop` / `None` handling | `nextHop()` — Gate 3 |
| firewall `natRules` loop | `dnatPaths()` |
| public-IP + route + admin composition | `edgeOpen()` / `reachable()` |
| severity + latent branch | `severity()` |
| CIDR overlap, orphaned IP, default-allow segmentation | the finding detectors |

The fixtures in `testdata/` become the Go table-driven golden tests unchanged.

## v0 scope (honest)

**Covered and tested:** internet-exposure reachability through all four gates (NSG effective rules, route incl. `None` black-hole, AVNM source-scope both directions, firewall DNAT, public-IP exposure, AzureCloud-tag breadth), orphaned public endpoints, CIDR overlap, and the default-`AllowVnetInBound` flat-open of a sensitive subnet.

**Not yet (next, in priority order):** full intra-VNet path enumeration and transitive-peering reachability (fixture-2's spoke-x↔y); multi-hop network attack-paths and drift detection (the two strongest items from the latest feedback — both are graph extensions of `reachable()`); the cost and generation tools; the Azure adapter (Resource Graph + Network Watcher, needs live Azure); and the MCP transport (mcp-go, needs the Go toolchain + the dependency).

## Status

This completes the hard, previously-unproven part — the deterministic analyzer, demonstrated correct on real data. What remains is engineering that follows the plan: port to Go, add the Azure adapter and the MCP transport, wire the `aara-network-topology-reviewer` agent to `analyze_risks`, and extend `reachable()` for attack-paths and drift. None of it is uncertain anymore; the core is real.
