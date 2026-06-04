#!/usr/bin/env python3
"""azure-nettopo-engine — deterministic analysis core (reference implementation).

This is the executable spec / test oracle for the production Go engine described in
NetworkTopologyReviewer-engine-plan.md. It is stdlib-only (no deps) and computes network
exposure deterministically from a topology export — no model in the path. The Go engine
(internal/analyze) is a direct port of these functions.

Run:  python analyze.py <fixture.json>   -> prints findings as JSON
"""
import ipaddress
import json
import sys


# ---------------------------------------------------------------- model / helpers
def cidr_overlap(a: str, b: str) -> bool:
    try:
        return ipaddress.ip_network(a, strict=False).overlaps(ipaddress.ip_network(b, strict=False))
    except ValueError:
        return False


def is_internet_source(src: str) -> bool:
    return (src or "").lower() in ("0.0.0.0/0", "internet", "*")


def is_broad_tag_source(src: str) -> bool:
    # service tags broader than they look (cross-tenant) — not raw internet, but over-permissive
    return (src or "").lower() in ("azurecloud",)


def finding(ftype, severity, resource, evidence, reachable):
    return {"type": ftype, "severity": severity, "resource": resource,
            "evidence": evidence, "reachable": reachable}


def nic_vnet(nic) -> str:
    s = nic.get("subnet", "")
    return s.split("/")[0] if "/" in s else ""


# ---------------------------------------------------------------- Gate 1: AVNM admin rules
def admin_verdict(admin_rules, vnet, port, want_source="internet"):
    """Highest-priority inbound admin verdict that governs an `want_source`-sourced flow on `port`.
    Source-scope aware: an Internet-tag rule does NOT govern intra-VNet/peered sources."""
    best, best_pri = None, 10 ** 9
    for ar in admin_rules:
        if ar.get("direction") != "Inbound":
            continue
        if vnet not in (ar.get("appliesTo") or []):
            continue
        if str(ar.get("destinationPortRange")) != str(port):
            continue
        ars = (ar.get("sourceAddressPrefix") or "").lower()
        # only rules whose source covers the source under test apply
        if want_source == "internet" and ars not in ("internet", "0.0.0.0/0", "*"):
            continue
        pri = ar.get("priority", 10 ** 9)
        if pri < best_pri:
            best, best_pri = ar.get("access"), pri
    return best  # 'Allow' | 'AlwaysAllow' | 'Deny' | None


# ---------------------------------------------------------------- the analysis
def analyze(fx):
    rg = fx.get("resourceGraph", {})
    nw = fx.get("networkWatcher", {})
    admin_rules = fx.get("avnm", {}).get("securityAdminRules", [])
    fw = fx.get("azureFirewall")
    eff_rules = nw.get("effectiveSecurityRules", {})
    eff_routes = nw.get("effectiveRoutes", {})
    nics = {n["name"]: n for n in rg.get("networkInterfaces", [])}
    findings = []

    # ---- per-NIC internet exposure (Gates: AVNM source-scope -> NSG -> route -> public IP) ----
    for name, nic in nics.items():
        rules = eff_rules.get(name, [])
        routes = eff_routes.get(name, [])
        has_pip = bool(nic.get("publicIp"))
        default_hop = next((r.get("nextHopType") for r in routes if r.get("addressPrefix") == "0.0.0.0/0"), None)
        vnet = nic_vnet(nic)

        for r in rules:
            if r.get("direction") != "Inbound":
                continue
            src = r.get("sourceAddressPrefix", "")
            broad_net, broad_tag = is_internet_source(src), is_broad_tag_source(src)
            if not (broad_net or broad_tag):
                continue
            port = r.get("destinationPortRange", "")
            admin = admin_verdict(admin_rules, vnet, port)            # Gate 1
            nsg_allows = (r.get("access") == "Allow")                 # Gate 2 (effective)
            if admin == "AlwaysAllow":
                open_internet = True                                 # admin force-opens past NSG
            elif admin == "Deny":
                open_internet = False                                # admin closes the internet source
            else:
                open_internet = nsg_allows
            reachable = open_internet and has_pip and default_hop == "Internet"  # Gates 3+4

            if reachable:
                sensitive = str(nic.get("tags", {}).get("sensitive", "")).lower() == "true"
                sev = "Critical" if sensitive else "High"
                ev = f"{src}:{port} inbound + route 0.0.0.0/0->Internet + public IP {nic.get('publicIp')}"
                if admin == "AlwaysAllow":
                    ev += " (AVNM AlwaysAllow overrides NSG)"
                if broad_tag:
                    ev += " — AzureCloud tag = all Azure public IPs, cross-tenant"
                findings.append(finding("over-permissive NSG (reachable)", sev, name, ev, True))
            else:
                why = []
                if not has_pip:
                    why.append("no public IP")
                if default_hop == "None":
                    why.append("route 0.0.0.0/0->None (black-hole)")
                elif default_hop and default_hop != "Internet":
                    why.append(f"route 0.0.0.0/0->{default_hop}")
                if admin == "Deny":
                    why.append("AVNM Deny closes the Internet source (east-west may remain open)")
                findings.append(finding("over-permissive NSG (latent)", "Informational", name,
                                        f"{src}:{port} inbound but " + "; ".join(why) or "not reachable", False))

        # inbound firewall DNAT publishes a no-public-IP backend
        if fw:
            for nat in fw.get("natRules", []):
                if nat.get("translatedAddress") == nic.get("privateIp"):
                    findings.append(finding(
                        "over-permissive NSG (reachable)", "High", name,
                        f"firewall DNAT {fw.get('publicIp')}:{nat.get('destinationPort')} -> "
                        f"{nic.get('privateIp')}:{nat.get('translatedPort')} (source {nat.get('sourceAddresses')}); "
                        f"no public IP on the NIC", True))

    # ---- orphaned public endpoints ----
    for pip in rg.get("publicIPAddresses", []):
        if not pip.get("ipConfiguration"):
            findings.append(finding("orphaned public endpoint", "Low", pip["name"],
                                    f"public IP {pip.get('ipAddress')} with null ipConfiguration", False))

    # ---- CIDR / address-space overlap ----
    vnets = rg.get("virtualNetworks", [])
    for i in range(len(vnets)):
        for j in range(i + 1, len(vnets)):
            for pa in vnets[i].get("addressSpace", []):
                for pb in vnets[j].get("addressSpace", []):
                    if cidr_overlap(pa, pb):
                        findings.append(finding("CIDR overlap", "Medium",
                                                f"{vnets[i]['name']}~{vnets[j]['name']}",
                                                f"overlapping address space {pa} / {pb}", False))

    # ---- segmentation: sensitive subnet reachable VNet-wide via the default AllowVnetInBound ----
    for name, nic in nics.items():
        if str(nic.get("tags", {}).get("sensitive", "")).lower() != "true":
            continue
        rules = eff_rules.get(name, [])
        allow_vnet = any(r.get("name") == "AllowVnetInBound" or
                         (r.get("priority") == 65000 and r.get("access") == "Allow") for r in rules)
        deny_vnet = any("DenyVnetInBound" in (r.get("name", "")) for r in rules)
        if allow_vnet and not deny_vnet:
            findings.append(finding("missing tier segmentation", "High", name,
                                    "sensitive subnet reachable VNet-wide via default AllowVnetInBound "
                                    "(no DenyVnetInBound above priority 65000)", True))

    # Canonical, deterministic ordering — must match the Go engine's sort
    # (internal/analyze/analyze.go: by resource, then type) so the reference and
    # the port emit findings in the same order and stay true twins.
    findings.sort(key=lambda f: (f["resource"], f["type"]))
    return findings


def main():
    if len(sys.argv) != 2:
        sys.exit("usage: python analyze.py <fixture.json>")
    fx = json.load(open(sys.argv[1], encoding="utf-8"))
    print(json.dumps(analyze(fx), indent=2))


if __name__ == "__main__":
    main()
