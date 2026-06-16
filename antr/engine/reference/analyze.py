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


def _go_list(xs):
    # Format like Go fmt %v ([a b c]) so DNAT evidence matches the Go twin (V4-07).
    return "[" + " ".join(str(x) for x in (xs or [])) + "]"


def rid(obj) -> str:
    """Stable identity for a resource: its ARM resource id when present, else its
    bare name. Bare names are NOT unique across subscriptions/resource groups, so
    multi-subscription estates must carry `id`. (V4-07 — keeps single-sub fixtures
    that have no `id` byte-identical.)"""
    return obj.get("id") or obj.get("name", "")


def nic_vnet(nic) -> str:
    s = nic.get("subnet", "")
    return s.split("/")[0] if "/" in s else ""


def subnet_to_vnet(subnet: str) -> str:
    s = subnet or ""
    return s.split("/")[0] if "/" in s else ""


# peGroupIdToZone maps a Private Endpoint groupId to the canonical Azure Private
# DNS zone name for that service sub-resource. Kept byte-identical to the Go twin
# (internal/analyze/analyze.go) so the H-1 families produce the same findings.
PE_GROUPID_TO_ZONE = {
    "blob": "privatelink.blob.core.windows.net",
    "file": "privatelink.file.core.windows.net",
    "queue": "privatelink.queue.core.windows.net",
    "table": "privatelink.table.core.windows.net",
    "dfs": "privatelink.dfs.core.windows.net",
    "web": "privatelink.web.core.windows.net",
    "vault": "privatelink.vaultcore.azure.net",
    "sql": "privatelink.database.windows.net",
    "sqlOnDemand": "privatelink.sql.azuresynapse.net",
    "registry": "privatelink.azurecr.io",
    "sites": "privatelink.azurewebsites.net",
    "namespace": "privatelink.servicebus.windows.net",
    "managedInstance": "privatelink.database.windows.net",
    "searchService": "privatelink.search.windows.net",
    "azurecosmosdb": "privatelink.documents.azure.com",
    "redisCache": "privatelink.redis.cache.windows.net",
    "openai": "privatelink.openai.azure.com",
    "account": "privatelink.purview.azure.com",
}


# ------------------------------------------------ Azure-specific families (H-1)
# Ported 1:1 from the Go engine's check* functions. Evidence strings are
# byte-identical (em-dash U+2014) so twin-drift covers the WHOLE engine, not just
# the shared core. These are the families that previously had no Python oracle.
def check_private_dns_zone(rg):
    pes = rg.get("privateEndpoints", [])
    if not pes:
        return []
    zone_linked = {}  # zoneName -> set of linked VNet names
    for z in rg.get("privateDnsZones", []):
        zone_linked.setdefault(z.get("name", ""), set()).update(z.get("linkedVnets", []) or [])
    out = []
    for pe in pes:
        state = pe.get("connectionState", "")
        if state != "Approved" and state != "":
            continue
        expected = PE_GROUPID_TO_ZONE.get(pe.get("groupId", ""))
        if expected is None:
            continue
        vnet = subnet_to_vnet(pe.get("subnet", ""))
        if vnet == "":
            continue
        name = pe.get("name", "")
        gid = pe.get("groupId", "")
        if expected not in zone_linked:
            out.append(finding(
                "private DNS zone missing", "High", name,
                f"Private endpoint \"{name}\" (service: {gid}) is in VNet \"{vnet}\" but "
                f"Private DNS zone \"{expected}\" does not exist in this subscription — "
                f"DNS resolution will use public endpoints", False))
            continue
        if vnet not in zone_linked[expected]:
            out.append(finding(
                "private DNS zone not linked to VNet", "High", name,
                f"Private endpoint \"{name}\" (service: {gid}) is in VNet \"{vnet}\" but zone "
                f"\"{expected}\" is not linked to that VNet — workloads in \"{vnet}\" resolve "
                f"this service via public DNS, bypassing the private endpoint", False))
    return out


def check_app_gateway(rg):
    out = []
    for gw in rg.get("applicationGateways", []):
        pip = gw.get("publicIp", "")
        if pip == "":
            continue
        name = gw.get("name", "")
        if not gw.get("wafEnabled", False):
            out.append(finding(
                "app gateway WAF disabled", "Medium", name,
                f"Application Gateway \"{name}\" has public IP {pip} but WAF is disabled — "
                f"no L7 protection on public ingress", True))
        elif (gw.get("wafMode", "") or "").lower() == "detection":
            out.append(finding(
                "app gateway WAF in detection mode", "Informational", name,
                f"Application Gateway \"{name}\" WAF is enabled but in Detection mode — "
                f"threats are logged but not blocked", False))
    return out


def check_aks(rg):
    out = []
    for aks in rg.get("aksClusters", []):
        if not aks.get("isPrivateCluster", False):
            name = aks.get("name", "")
            out.append(finding(
                "AKS non-private cluster", "Medium", name,
                f"AKS cluster \"{name}\" is not a private cluster — API server is reachable "
                f"from the public internet; use a private cluster with a private endpoint for "
                f"production workloads", True))
    return out


def check_cross_sub_peering(fx):
    out = []
    for xp in fx.get("crossSubscriptionPeerings", []):
        if (xp.get("state", "") or "").lower() == "connected" and not xp.get("hasHubFirewall", False):
            local, remote = xp.get("localVnet", ""), xp.get("remoteVnet", "")
            sub = xp.get("remoteSubscriptionId", "")
            out.append(finding(
                "cross-subscription peering without firewall", "Medium", local + "~" + remote,
                f"VNet \"{local}\" and \"{remote}\" (sub {sub}) are directly peered across "
                f"subscriptions with no hub firewall in path — lateral movement between "
                f"subscriptions is unrestricted", False))
    return out


def check_load_balancer_nat(rg):
    out = []
    for lb in rg.get("loadBalancers", []):
        if lb.get("isInternal", False) or lb.get("frontendIp", "") == "":
            continue
        lbname, feip = lb.get("name", ""), lb.get("frontendIp", "")
        for nat in lb.get("inboundNatRules", []) or []:
            bnic = nat.get("backendNic", "")
            if bnic == "":
                continue
            out.append(finding(
                "internet reachable via load balancer NAT", "High", bnic,
                f"load balancer \"{lbname}\" NAT rule \"{nat.get('name', '')}\" forwards public IP "
                f"{feip}:{nat.get('frontendPort', 0)} → NIC \"{bnic}\":{nat.get('backendPort', 0)} — "
                f"NIC is internet-reachable without a direct public IP", True))
    return out


def check_apim(rg):
    out = []
    for apim in rg.get("apiManagements", []):
        name, mode = apim.get("name", ""), apim.get("vnetMode", "")
        if mode == "None":
            out.append(finding(
                "APIM without VNet isolation", "Medium", name,
                f"API Management \"{name}\" is deployed without VNet injection (mode=None) — "
                f"gateway is publicly accessible and backend API calls bypass network controls", True))
        elif mode == "External":
            if not apim.get("hasWafFrontEnd", False):
                out.append(finding(
                    "APIM External mode without WAF", "Medium", name,
                    f"API Management \"{name}\" is VNet-injected in External mode (public endpoint "
                    f"{apim.get('publicIp', '')}) with no WAF upstream — API traffic reaches the "
                    f"gateway without L7 inspection", True))
    return out


def check_bastion_bypass(rg, eff_rules):
    if not rg.get("azureBastions", []):
        return []
    mgmt = {"22", "3389"}
    out = []
    for nic in rg.get("networkInterfaces", []):
        pip = nic.get("publicIp")
        if not pip:
            continue
        name = nic.get("name", "")
        for r in eff_rules.get(name, []) or []:
            if r.get("direction") != "Inbound" or r.get("access") != "Allow":
                continue
            if not is_internet_source(r.get("sourceAddressPrefix", "")):
                continue
            if str(r.get("destinationPortRange", "")) in mgmt:
                out.append(finding(
                    "Bastion bypass — direct management port exposed", "High", name,
                    f"Azure Bastion is deployed but NIC \"{name}\" has public IP {pip} with port "
                    f"{r.get('destinationPortRange', '')} open from internet — Bastion is intended "
                    f"to be the exclusive management ingress", True))
                break
    return out


def check_front_door(rg):
    out = []
    for fd in rg.get("azureFrontDoors", []):
        name = fd.get("name", "")
        if not fd.get("wafEnabled", False):
            out.append(finding(
                "Front Door WAF disabled", "Medium", name,
                f"Azure Front Door \"{name}\" has no WAF policy enabled — all internet-facing "
                f"endpoints lack L7 protection (OWASP Top 10, DDoS at app layer)", True))
        elif (fd.get("wafMode", "") or "").lower() == "detection":
            out.append(finding(
                "Front Door WAF in detection mode", "Informational", name,
                f"Azure Front Door \"{name}\" WAF is enabled but in Detection mode — threats are "
                f"logged but not blocked; switch to Prevention for active protection", False))
    return out


def check_virtual_wan(rg):
    out = []
    for wan in rg.get("virtualWans", []):
        for hub in wan.get("vHubs", []) or []:
            name = hub.get("name", "")
            if not hub.get("hasSecuredFirewall", False):
                spokes = len(hub.get("spokeConnections", []) or [])
                out.append(finding(
                    "vWAN hub unsecured — no firewall", "Medium", name,
                    f"Virtual WAN hub \"{name}\" has {spokes} spoke connection(s) but no secured "
                    f"Azure Firewall — all spoke-to-spoke and spoke-to-internet traffic is "
                    f"forwarded without inspection", False))
            elif not hub.get("routingPolicyPrivate", False):
                out.append(finding(
                    "vWAN hub firewall bypasses private traffic", "Medium", name,
                    f"Virtual WAN hub \"{name}\" has a secured firewall but RoutingPolicyPrivate=false "
                    f"— spoke-to-spoke (east-west) traffic bypasses the firewall; only "
                    f"internet-bound traffic is inspected", False))
    return out


# ---------------------------------------------------------------- Gate 1: AVNM admin rules
def _parse_port_range(s):
    """Parse an Azure destinationPortRange token into an inclusive (lo, hi), or
    None for '*' / non-numeric. Mirrors Go analyze.parsePortRange (twin parity)."""
    s = (s or "").strip()
    if "-" in s:
        a, _, b = s.partition("-")
        try:
            lo, hi = int(a.strip()), int(b.strip())
        except ValueError:
            return None
        return (lo, hi) if lo <= hi else None
    try:
        p = int(s)
    except ValueError:
        return None
    return (p, p)


def _admin_port_covers(admin_spec, nsg_port):
    """Whether an AVNM admin rule's port range governs the NSG rule's port. '*'
    covers all ports; a range covers any port it contains. Without this the verdict
    exact-matched strings, so a deny-all admin on '*'/'80-443' never governed an NSG
    allow on '443' (external review F2). Mirrors Go analyze.adminPortCovers."""
    a, n = (admin_spec or "").strip(), (nsg_port or "").strip()
    if a == "*":
        return True
    if a == n:
        return True
    ar, nr = _parse_port_range(a), _parse_port_range(n)
    if ar is None or nr is None:
        return False
    return ar[0] <= nr[0] and ar[1] >= nr[1]


def admin_verdict(admin_rules, vnet, port, want_source="internet"):
    """Highest-priority inbound admin verdict that governs an `want_source`-sourced flow on `port`.
    Source-scope aware: an Internet-tag rule does NOT govern intra-VNet/peered sources.
    Port-scope aware: wildcard/range admin ports govern any NSG port they cover (F2)."""
    best, best_pri = None, 10 ** 9
    for ar in admin_rules:
        if ar.get("direction") != "Inbound":
            continue
        if vnet not in (ar.get("appliesTo") or []):
            continue
        if not _admin_port_covers(ar.get("destinationPortRange", ""), port):
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
    # DNAT is evaluated across ALL firewalls (plural) plus the legacy singular
    # field, so a published backend behind any firewall is caught (external review
    # F5). Mirrors Go analyze.go's union.
    fws = list(fx.get("azureFirewalls") or [])
    _fw1 = fx.get("azureFirewall")
    if _fw1:
        fws.append(_fw1)
    eff_rules = nw.get("effectiveSecurityRules", {})
    eff_routes = nw.get("effectiveRoutes", {})
    # Key NICs by stable identity (ARM id when present, else name). Keying by bare
    # name dropped same-named NICs across subscriptions at the INPUT stage (V4-07).
    nics = {rid(n): n for n in rg.get("networkInterfaces", [])}
    findings = []

    # Surface NICs whose Network Watcher enrichment failed (audit M-3).
    for _name in nw.get("incompleteNics", []):
        findings.append(finding("analysis incomplete", "Medium", _name,
                                "Network Watcher enrichment failed \u2014 effective rules/routes unavailable; NIC not evaluated for internet exposure", False))

    def eff_for(nic, table):
        # Network Watcher tables are keyed by NIC id in a real multi-sub estate;
        # current single-sub fixtures key by name. Try id, fall back to name.
        v = table.get(rid(nic))
        return v if v is not None else table.get(nic.get("name", ""), [])

    # ---- per-NIC internet exposure (Gates: AVNM source-scope -> NSG -> route -> public IP) ----
    for key, nic in nics.items():
        rules = eff_for(nic, eff_rules)
        routes = eff_for(nic, eff_routes)
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
                findings.append(finding("over-permissive NSG (reachable)", sev, rid(nic), ev, True))
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
                # NOTE: parenthesize the `or` — `+` binds tighter, so the fallback
                # "not reachable" was previously dead code (LOW-1).
                findings.append(finding("over-permissive NSG (latent)", "Informational", rid(nic),
                                        f"{src}:{port} inbound but " + ("; ".join(why) or "not reachable"), False))

        # inbound firewall DNAT publishes a no-public-IP backend (any firewall, F5)
        for fw in fws:
            for nat in fw.get("natRules", []):
                if nat.get("translatedAddress") == nic.get("privateIp"):
                    findings.append(finding(
                        "over-permissive NSG (reachable)", "High", rid(nic),
                        f"firewall DNAT {fw.get('publicIp')}:{nat.get('destinationPort')} -> "
                        f"{nic.get('privateIp')}:{nat.get('translatedPort')} (source {_go_list(nat.get('sourceAddresses'))}); "
                        f"no public IP on the NIC", True))

    # ---- orphaned public endpoints ----
    for pip in rg.get("publicIPAddresses", []):
        if not pip.get("ipConfiguration"):
            findings.append(finding("orphaned public endpoint", "Low", rid(pip),
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
    for key, nic in nics.items():
        if str(nic.get("tags", {}).get("sensitive", "")).lower() != "true":
            continue
        rules = eff_for(nic, eff_rules)
        allow_vnet = any(r.get("name") == "AllowVnetInBound" or
                         (r.get("priority") == 65000 and r.get("access") == "Allow") for r in rules)
        # A real segmentation control overrides the default AllowVnetInBound only
        # if it is an INBOUND DENY, VNet-scoped, at HIGHER precedence (priority <
        # 65000). Trusting the rule NAME alone let a non-overriding rule falsely
        # suppress the finding (external review F7). Mirrors Go analyze.go.
        deny_vnet = any(
            r.get("direction") == "Inbound" and r.get("access") == "Deny" and
            (r.get("priority") or 0) < 65000 and
            ((r.get("sourceAddressPrefix", "") or "").lower() == "virtualnetwork"
             or "DenyVnetInBound" in (r.get("name", "")))
            for r in rules)
        if allow_vnet and not deny_vnet:
            findings.append(finding("missing tier segmentation", "High", rid(nic),
                                    "sensitive subnet reachable VNet-wide via default AllowVnetInBound "
                                    "(no DenyVnetInBound above priority 65000)", True))

    # ---- Azure-specific families (H-1): ported 1:1 from the Go engine so the
    # twin covers the WHOLE engine, not just the shared core. Order of appends
    # mirrors analyze.go; the final sort makes order irrelevant anyway.
    findings += check_private_dns_zone(rg)
    findings += check_app_gateway(rg)
    findings += check_aks(rg)
    findings += check_cross_sub_peering(fx)
    findings += check_load_balancer_nat(rg)
    findings += check_apim(rg)
    findings += check_bastion_bypass(rg, eff_rules)
    findings += check_virtual_wan(rg)
    findings += check_front_door(rg)

    # Canonical, deterministic ordering — must match the Go engine's sort
    # (internal/analyze/analyze.go: by resource, then type) so the reference and
    # the port emit findings in the same order and stay true twins.
    # Total order incl. evidence — a NIC can emit two findings with the same
    # (resource, type) (e.g. two latent rules); evidence breaks the tie so the
    # order is fully determined and matches the Go engine's SliceStable comparator.
    findings.sort(key=lambda f: (f["resource"], f["type"], f["evidence"]))
    return findings


def main():
    if len(sys.argv) != 2:
        sys.exit("usage: python analyze.py <fixture.json>")
    fx = json.load(open(sys.argv[1], encoding="utf-8"))
    print(json.dumps(analyze(fx), indent=2))


if __name__ == "__main__":
    main()
