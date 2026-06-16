#!/usr/bin/env python3
"""Generate a large, deterministic, multi-subscription / multi-hub Azure estate
fixture — a stand-in for an enterprise estate like the BCLM reference (which is a
hand-drawn diagram, not a machine fixture). Used to prove the pipeline scales:
connected, cross-subscription, boundary-rich, with a real severity spread.

Run:
    python3 synth_estate.py --hubs 3 --spokes 30 --subs 5 --seed 42 --out <file.json>
"""
import argparse
import json
import random


def gen(hubs=3, spokes=30, subs=5, seed=42):
    rnd = random.Random(seed)
    subscriptions = [f"sub-conn-{i:02d}" for i in range(hubs)] + \
                    [f"sub-wl-{i:02d}" for i in range(subs)]
    vnets, nsgs, rts, pips, nics = [], [], [], [], []
    eff_rules, eff_routes = {}, {}
    gws, ercs, natgws = [], [], []
    xsub = []
    fw = {"name": "afw-hub-0", "publicIp": "20.0.0.1", "vnet": "hub-00", "natRules": []}

    # hubs (one per connectivity subscription)
    hub_names = []
    for h in range(hubs):
        hn = f"hub-{h:02d}"
        hub_names.append(hn)
        vnets.append({"name": hn, "subscriptionId": f"sub-conn-{h:02d}",
                      "addressSpace": [f"10.{h}.0.0/16"],
                      "subnets": [{"name": "AzureFirewallSubnet", "addressPrefix": f"10.{h}.0.0/26"},
                                  {"name": "GatewaySubnet", "addressPrefix": f"10.{h}.1.0/27"}],
                      "peerings": []})
        gws.append({"name": f"vpngw-{hn}", "gatewayType": "Vpn", "vnet": hn})
        if h == 0:
            gws.append({"name": f"ergw-{hn}", "gatewayType": "ExpressRoute", "vnet": hn})
            ercs.append({"name": "er-circuit-att", "peeringLocation": "Silicon Valley"})

    # spokes
    for s in range(spokes):
        sub = rnd.choice(subscriptions[hubs:])  # a workload subscription
        sn = f"spoke-{s:02d}"
        base = 16 + s
        vnets.append({"name": sn, "subscriptionId": sub,
                      "addressSpace": [f"10.{base}.0.0/16"],
                      "subnets": [{"name": "web", "addressPrefix": f"10.{base}.1.0/24",
                                   "networkSecurityGroup": f"nsg-{sn}-web", "routeTable": f"rt-{sn}-web"},
                                  {"name": "app", "addressPrefix": f"10.{base}.2.0/24",
                                   "networkSecurityGroup": f"nsg-{sn}-app", "routeTable": f"rt-{sn}-app"}],
                      "peerings": []})
        # peer spoke -> a hub (cross-subscription by construction)
        hub = hub_names[s % hubs]
        _peer(vnets, sn, hub)
        _peer(vnets, hub, sn)
        # ~25% spoke-to-spoke peering
        if s >= 2 and rnd.random() < 0.25:
            other = f"spoke-{rnd.randint(0, s-1):02d}"
            _peer(vnets, sn, other)
            _peer(vnets, other, sn)
        # ~15% cross-subscription out-of-scope peer (external stub)
        if rnd.random() < 0.15:
            xsub.append({"localVnet": sn, "remoteVnet": f"shared-{s:02d}",
                         "remoteSubscriptionId": "sub-shared-99", "state": "Connected",
                         "allowForwardedTraffic": True, "hasHubFirewall": False})

        # web NIC: ~30% internet-exposed; ~half of those sensitive
        exposed = rnd.random() < 0.30
        sensitive = exposed and rnd.random() < 0.5
        webnic = f"nic-{sn}-web"
        pip = None
        if exposed:
            pip = f"pip-{sn}-web"
            pips.append({"name": pip, "ipAddress": f"20.60.{s}.10", "ipConfiguration": f"{webnic}/ipconfig1"})
        nic = {"name": webnic, "subnet": f"{sn}/web", "publicIp": pip, "privateIp": f"10.{base}.1.4"}
        if sensitive:
            nic["tags"] = {"sensitive": "true"}
        nics.append(nic)
        rules = [{"name": "allow-https", "priority": 200, "direction": "Inbound", "access": "Allow",
                  "protocol": "Tcp", "sourceAddressPrefix": "0.0.0.0/0", "destinationPortRange": "443"},
                 {"name": "AllowVnetInBound", "priority": 65000, "direction": "Inbound", "access": "Allow",
                  "sourceAddressPrefix": "VirtualNetwork", "destinationPortRange": "*"},
                 {"name": "DenyAllInBound", "priority": 65500, "direction": "Inbound", "access": "Deny"}]
        eff_rules[webnic] = rules
        eff_routes[webnic] = [{"addressPrefix": f"10.{base}.0.0/16", "nextHopType": "VnetLocal"},
                              {"addressPrefix": "0.0.0.0/0",
                               "nextHopType": "Internet" if exposed else "VirtualAppliance"}]
        nsgs.append({"name": f"nsg-{sn}-web", "securityRules": rules, "associatedSubnets": [f"{sn}/web"]})

        # app NIC: always firewalled, no public IP -> latent or clean
        appnic = f"nic-{sn}-app"
        nics.append({"name": appnic, "subnet": f"{sn}/app", "publicIp": None, "privateIp": f"10.{base}.2.4"})
        eff_rules[appnic] = [{"name": "AllowVnetInBound", "priority": 65000, "direction": "Inbound",
                              "access": "Allow", "sourceAddressPrefix": "VirtualNetwork", "destinationPortRange": "*"},
                             {"name": "DenyAllInBound", "priority": 65500, "direction": "Inbound", "access": "Deny"}]
        eff_routes[appnic] = [{"addressPrefix": f"10.{base}.0.0/16", "nextHopType": "VnetLocal"},
                             {"addressPrefix": "0.0.0.0/0", "nextHopType": "VirtualAppliance"}]
        rts.append({"name": f"rt-{sn}-web", "routes": [], "associatedSubnets": [f"{sn}/web"]})
        rts.append({"name": f"rt-{sn}-app",
                    "routes": [{"addressPrefix": "0.0.0.0/0", "nextHopType": "VirtualAppliance"}],
                    "associatedSubnets": [f"{sn}/app"]})

    natgws.append({"name": "natgw-shared", "subnet": "spoke-00/app"})
    pips.append({"name": "pip-orphan-01", "ipAddress": "20.99.99.99", "ipConfiguration": None})

    return {
        "subscription": "sub-conn-00",
        "_scenario": f"Synthetic enterprise estate: {hubs} hubs, {spokes} spokes, {len(subscriptions)} subscriptions.",
        "resourceGraph": {"virtualNetworks": vnets, "networkSecurityGroups": nsgs, "routeTables": rts,
                          "publicIPAddresses": pips, "networkInterfaces": nics,
                          "virtualNetworkGateways": gws, "expressRouteCircuits": ercs, "natGateways": natgws},
        "networkWatcher": {"effectiveSecurityRules": eff_rules, "effectiveRoutes": eff_routes},
        "avnm": {"securityAdminRules": []},
        "azureFirewall": fw,
        "crossSubscriptionPeerings": xsub,
    }


def _peer(vnets, a, b):
    for v in vnets:
        if v["name"] == a:
            if not any(p["remoteVnet"] == b for p in v["peerings"]):
                v["peerings"].append({"remoteVnet": b, "state": "Connected", "allowForwardedTraffic": True})
            return


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--hubs", type=int, default=3)
    ap.add_argument("--spokes", type=int, default=30)
    ap.add_argument("--subs", type=int, default=5)
    ap.add_argument("--seed", type=int, default=42)
    ap.add_argument("--out", default="phase-4/fixtures/estate-synth-large.json")
    a = ap.parse_args()
    fx = gen(a.hubs, a.spokes, a.subs, a.seed)
    json.dump(fx, open(a.out, "w", encoding="utf-8"), indent=1)
    rg = fx["resourceGraph"]
    print(f"wrote {a.out}: {len(rg['virtualNetworks'])} vnets, {len(rg['networkInterfaces'])} nics, "
          f"{sum(len(v['peerings']) for v in rg['virtualNetworks'])} peerings, "
          f"{len(fx['crossSubscriptionPeerings'])} x-sub")


if __name__ == "__main__":
    main()
