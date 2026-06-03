#!/usr/bin/env python3
"""Golden tests for the deterministic engine core, using the real eval fixtures.

Each case asserts the planted findings ARE produced and the precision traps are NOT
flagged reachable-High. Green here = the deterministic algorithm reproduces the answer
keys with no model in the path. Run: python test_analyze.py
"""
import json
import os
import sys

from analyze import analyze

HERE = os.path.dirname(os.path.abspath(__file__))
TD = os.path.join(HERE, "..", "testdata")


def load(n):
    return json.load(open(os.path.join(TD, n), encoding="utf-8"))


def reachable_high(F, resource):
    return any(f["reachable"] and f["severity"] in ("High", "Critical") and f["resource"] == resource for f in F)


def has(F, type_substr, resource=None):
    return any(type_substr in f["type"] and (resource is None or f["resource"] == resource) for f in F)


CASES = []


def case(name):
    def deco(fn):
        CASES.append((name, fn))
        return fn
    return deco


@case("f1  internet exposure: real (spoke-a) vs latent (spoke-b) + orphaned IP")
def _():
    F = analyze(load("fixture-1-internet-exposure.json"))
    assert reachable_high(F, "nic-vm-web-a"), "spoke-a SSH must be reachable High"
    assert not reachable_high(F, "nic-vm-web-b"), "TRAP: spoke-b (firewalled, no public IP) must NOT be reachable"
    assert has(F, "orphaned", "pip-orphan-01"), "orphaned public IP must be flagged"


@case("f2  default AllowVnetInBound flat-opens the sensitive db VNet-wide")
def _():
    F = analyze(load("fixture-2-segmentation-peering.json"))
    assert reachable_high(F, "nic-db1"), "sensitive db reachable VNet-wide (no DenyVnetInBound) must be High"
    assert not any(f["reachable"] and "->Internet" in f.get("evidence", "") for f in F), "no internet exposure (no public IPs)"


@case("f3  AVNM source-scope (AlwaysAllow opens / Deny closes) + CIDR overlap")
def _():
    F = analyze(load("fixture-3-cidr-avnm.json"))
    assert reachable_high(F, "nic-edge1"), "edge 443 opened by AVNM AlwaysAllow must be reachable High"
    assert not reachable_high(F, "nic-mgmt1"), "TRAP: mgmt RDP internet path closed by AVNM Deny must NOT be reachable"
    assert has(F, "CIDR overlap"), "ov-a/ov-b overlapping address space must be flagged"


@case("h1  firewall DNAT publishes a no-public-IP backend; sibling without DNAT is not reachable")
def _():
    F = analyze(load("fixture-h1-dnat-multihop.json"))
    assert reachable_high(F, "nic-backend1"), "backend1 reachable via firewall DNAT despite no public IP"
    assert not reachable_high(F, "nic-backend2"), "TRAP: backend2 has no DNAT rule -> must NOT be reachable"


@case("h2  None black-hole is latent; AzureCloud tag is a real broad exposure")
def _():
    F = analyze(load("fixture-h2-blackhole-tags.json"))
    assert reachable_high(F, "nic-edge"), "edge 443 from Internet must be reachable High"
    assert reachable_high(F, "nic-api"), "api 443 from AzureCloud (cross-tenant) must be reachable High"
    assert not reachable_high(F, "nic-dark"), "TRAP: darkpool 0.0.0.0/0:22 black-holed (route None) must NOT be reachable"


def run():
    fails = 0
    for name, fn in CASES:
        try:
            fn()
            print(f"PASS  {name}")
        except AssertionError as e:
            fails += 1
            print(f"FAIL  {name}\n        -> {e}")
        except Exception as e:
            fails += 1
            print(f"ERROR {name}\n        -> {type(e).__name__}: {e}")
    print(f"\n{len(CASES) - fails}/{len(CASES)} cases passed")
    sys.exit(1 if fails else 0)


if __name__ == "__main__":
    run()
