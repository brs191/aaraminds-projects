#!/usr/bin/env python3
"""V4-07 regression — resource identity keying.

Two NICs with the same bare name in different subscriptions must NOT merge:
the engine keys by ARM `id` when present, so each produces its own finding.
Golden fixtures (no `id`) are unaffected — that is asserted by test_analyze.py.

Run:  python3 test_resource_id.py
"""
import analyze as eng


def _fx():
    return {
        "resourceGraph": {
            "virtualNetworks": [
                {"name": "vA", "subscriptionId": "subA", "addressSpace": ["10.0.0.0/16"],
                 "subnets": [{"name": "web", "addressPrefix": "10.0.1.0/24"}], "peerings": []},
                {"name": "vB", "subscriptionId": "subB", "addressSpace": ["10.1.0.0/16"],
                 "subnets": [{"name": "web", "addressPrefix": "10.1.1.0/24"}], "peerings": []},
            ],
            "publicIPAddresses": [],
            "networkInterfaces": [
                {"id": "/subs/subA/nic-web", "name": "nic-web", "subnet": "vA/web",
                 "publicIp": "pipA", "privateIp": "10.0.1.4", "tags": {"sensitive": "true"}},
                {"id": "/subs/subB/nic-web", "name": "nic-web", "subnet": "vB/web",
                 "publicIp": "pipB", "privateIp": "10.1.1.4"},
            ],
        },
        "networkWatcher": {
            "effectiveSecurityRules": {
                "/subs/subA/nic-web": [{"name": "a", "direction": "Inbound", "access": "Allow",
                                        "sourceAddressPrefix": "0.0.0.0/0", "destinationPortRange": "443"}],
                "/subs/subB/nic-web": [{"name": "a", "direction": "Inbound", "access": "Allow",
                                        "sourceAddressPrefix": "0.0.0.0/0", "destinationPortRange": "443"}],
            },
            "effectiveRoutes": {
                "/subs/subA/nic-web": [{"addressPrefix": "0.0.0.0/0", "nextHopType": "Internet"}],
                "/subs/subB/nic-web": [{"addressPrefix": "0.0.0.0/0", "nextHopType": "Internet"}],
            },
        },
        "avnm": {"securityAdminRules": []},
    }


def test_no_merge_across_subscriptions():
    findings = eng.analyze(_fx())
    resources = {f["resource"] for f in findings}
    assert "/subs/subA/nic-web" in resources, "subA NIC finding lost (merged by name)"
    assert "/subs/subB/nic-web" in resources, "subB NIC finding lost (merged by name)"
    # subA: sensitive + internet-exposed -> Critical; subB: exposed, not sensitive -> High
    by_res = {}
    for f in findings:
        if f["type"].startswith("over-permissive NSG (reachable)"):
            by_res[f["resource"]] = f["severity"]
    assert by_res.get("/subs/subA/nic-web") == "Critical", by_res
    assert by_res.get("/subs/subB/nic-web") == "High", by_res


def test_determinism_total_order():
    a = eng.analyze(_fx())
    b = eng.analyze(_fx())
    assert a == b, "analyze() not deterministic"
    keys = [(f["resource"], f["type"], f["evidence"]) for f in a]
    assert keys == sorted(keys), "findings not in total (resource, type, evidence) order"


if __name__ == "__main__":
    test_no_merge_across_subscriptions()
    test_determinism_total_order()
    print("PASS  V4-07 resource-id keying: same-named NICs across subs do not merge; deterministic total order")
