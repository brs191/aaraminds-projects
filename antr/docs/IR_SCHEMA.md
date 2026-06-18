# antr IR — schema & example reference (for external builders)

The **Graph IR** is the JSON object every antr view consumes (`graph.Fixture`). This file is
the **self-contained contract**: the full schema, the analysis output (`Finding`), the
severity **overlay** + node-id scheme + palette, and a complete worked example — so a build
in another system (e.g. GitHub Copilot) never has to reverse-engineer shapes from prose.

Authoritative source: `engine/go/internal/graph/model.go` (Go) mirrored by
`engine/reference/analyze.py` (Python). This doc tracks them; if they ever disagree, the Go
struct + JSON tag wins.

> Determinism (applies to every producer/consumer): sort lists before emit; ids from content
> or a monotonic counter; **no** timestamps / UUIDs / hash-set iteration order / wall-clock.

---

## 1. Top level — `Fixture`

```jsonc
{
  "subscription": "<guid>",                 // primary subscription of this capture
  "resourceGraph": { ... },                 // §2 — the topology (the part views read most)
  "networkWatcher": { ... },                // §3 — effective rules/routes (engine input)
  "avnm": { "securityAdminRules": [ ... ] },// §4 — AVNM admin gate (engine input)
  "azureFirewall":  { ... },                // §5 — single firewall (legacy)
  "azureFirewalls": [ { ... } ],            // §5 — ALL firewalls (engine unions both)
  "crossSubscriptionPeerings": [ ... ],     // §6
  "enrichment": { ... }                     // optional; engine does NOT read it
}
```

## 2. `resourceGraph` — families

Every list is `omitempty` (absent = empty). Field names below are the JSON tags exactly.

### virtualNetworks `[]`
```jsonc
{ "name":"hub", "addressSpace":["10.0.0.0/16"],
  "subnets":[ { "name":"web", "addressPrefix":"10.0.1.0/24",
                "networkSecurityGroup":"nsg-web", "routeTable":"rt-web",
                "serviceEndpoints":["Microsoft.Storage"], "delegations":["Microsoft.Sql/managedInstances"],
                "privateEndpointNetworkPolicies":"Disabled" } ],
  "peerings":[ { "remoteVnet":"spoke", "remoteSubscriptionId":"", "state":"Connected",
                 "allowForwardedTraffic":true, "allowGatewayTransit":false, "useRemoteGateways":false,
                 "remoteVnetRegion":"eastus", "isGlobalPeering":false } ] }
```
`remoteSubscriptionId` is non-empty **only** for a genuine cross-subscription peer.

### networkInterfaces `[]`  (the workload/compute leaf)
```jsonc
{ "name":"nic-web", "id":"/subscriptions/.../networkInterfaces/nic-web",  // id optional; identity key
  "subnet":"hub/web", "networkSecurityGroup":"nsg-web", "publicIp":"pip-web", // publicIp null = none
  "privateIp":"10.0.1.4", "tags":{"application":"customer-portal"}, "dnsServers":["10.0.0.4"] }
```

### publicIPAddresses `[]`
```jsonc
{ "name":"pip-web", "id":"...", "ipAddress":"20.40.0.10",
  "ipConfiguration":"/subscriptions/.../ipConfigurations/ipc",   // null => orphaned (finding)
  "allocationMethod":"Static", "sku":"Standard" }
```

### networkSecurityGroups `[]` · routeTables `[]`
```jsonc
{ "name":"nsg-web",
  "securityRules":[ { "name":"allow-https","priority":200,"direction":"Inbound","access":"Allow",
                      "protocol":"Tcp","sourceAddressPrefix":"0.0.0.0/0","destinationPortRange":"443","source":"subnet:web" } ],
  "associatedSubnets":["hub/web"] }
{ "name":"rt-web",
  "routes":[ { "name":"default","addressPrefix":"0.0.0.0/0","nextHopType":"Internet","nextHopIpAddress":"" } ],
  "associatedSubnets":["hub/web"], "disableBgpRoutePropagation":false }
```
`nextHopType` ∈ Internet · VirtualAppliance · VnetLocal · VirtualNetworkGateway · **None** (black-hole).

### app / edge / data families
```jsonc
// applicationGateways[]
{ "name":"appgw-prod","subnet":"hub/appgw","publicIp":"pip-agw","wafEnabled":false,"wafMode":"",
  "backendPools":[ { "name":"bp","targets":["10.1.1.4"] } ] }            // targets = backend NIC private IPs / FQDNs
// loadBalancers[]
{ "name":"lb-pub","sku":"Standard","frontendIp":"pip-lb","isInternal":false,
  "inboundNatRules":[ { "name":"ssh","protocol":"Tcp","frontendPort":2222,"backendPort":22,"backendNic":"nic-app" } ],
  "backendPools":[ { "name":"bp","nicRefs":["nic-app"] } ] }              // nicRefs = NIC names
// aksClusters[]
{ "name":"aks-1","subnet":"spoke/aks","podCidr":"10.244.0.0/16","serviceCidr":"10.0.0.0/16",
  "isPrivateCluster":false,"apiServerIp":"" }
// privateEndpoints[]
{ "name":"pe-sql","subnet":"spoke/data","privateIp":"10.1.2.4","groupId":"sql",
  "privateLinkServiceId":"/subscriptions/.../Microsoft.Sql/servers/sql-prod","connectionState":"Approved" }
// privateDnsZones[]
{ "name":"privatelink.database.windows.net","linkedVnets":["hub","spoke"],
  "aRecords":[ { "name":"sql-prod","ip":"10.1.2.4" } ] }
// natGateways[]
{ "name":"natgw-1","publicIps":["pip-nat"],"associatedSubnets":["spoke/web"] }
// azureFrontDoors[]   { "name":"fd-edge","sku":"Premium_AzureFrontDoor","wafEnabled":false,"wafMode":"","endpoints":[ {"name":"e1","hostname":"x.azurefd.net","wafPolicyId":"","enabled":true} ] }
// apiManagements[]    { "name":"apim-1","subnet":"hub/apim","publicIp":"","vnetMode":"Internal","gatewayUrl":"","hasWafFrontEnd":false,"skuName":"Premium" }
// azureBastions[]     { "name":"bastion-1","subnet":"hub/AzureBastionSubnet","publicIp":"pip-bas","sku":"Standard" }
// virtualNetworkGateways[]  { "name":"vpngw","gatewayType":"Vpn", ... }   // ExpressRoute | Vpn
// expressRouteCircuits[]    { "name":"er-1","peeringLocation":"...","bandwidthMbps":1000,"connectedVnet":"hub","bgpAdvertisesDefaultRoute":false }
// privateLinkServices[]     { "name":"pls-1","subnet":"hub/pls","natIpConfig":"10.0.3.4","linkedPrivateEndpoints":["pe-x"] }
// virtualWans[]             { "name":"vwan","sku":"Standard","vHubs":[ {"name":"hub-e","addressPrefix":"192.168.1.0/24","spokeConnections":["spoke"],"hasSecuredFirewall":false,"routingPolicyInternet":false,"routingPolicyPrivate":false} ] }
```
> NOT yet in the IR (see `FULL_ESTATE_VIEW_REQUIREMENTS.md` G1): the **backing data services**
> (SQL/Storage/Redis/…) themselves. Today only the Private Endpoint + `groupId` +
> `privateLinkServiceId` exist; the data node is logical until G1 adds `DataService` discovery.

## 3. `networkWatcher`
```jsonc
{ "effectiveSecurityRules": { "<nic id or name>": [ SecRule, ... ] },   // keyed by NIC id when present, else name
  "effectiveRoutes":        { "<nic id or name>": [ Route,   ... ] },
  "incompleteNics": ["nic-dark"] }                                       // NICs whose NW enrichment failed
```

## 4. `avnm.securityAdminRules[]`
```jsonc
{ "name":"deny-rdp","priority":10,"direction":"Inbound","access":"Deny",     // Allow | AlwaysAllow | Deny
  "protocol":"Tcp","sourceAddressPrefix":"Internet","destinationPortRange":"3389","appliesTo":["hub"] }
```
`destinationPortRange` may be a single port, a range `"80-443"`, or `"*"`.

## 5. firewall(s) — `azureFirewall` (object) and/or `azureFirewalls` (array)
```jsonc
{ "name":"afw","privateIp":"10.0.0.4","publicIp":"20.70.0.10","policyRef":"","skuTier":"Standard",
  "natRules":[ { "name":"dnat","protocol":"Tcp","sourceAddresses":["*"],"destinationAddress":"20.70.0.10",
                 "destinationPort":443,"translatedAddress":"10.1.1.4","translatedPort":443 } ] }
```

## 6. `crossSubscriptionPeerings[]`
```jsonc
{ "localVnet":"hub","remoteVnet":"partner","remoteSubscriptionId":"<other-guid>",
  "state":"Connected","allowForwardedTraffic":false,"hasHubFirewall":false }
```

---

## 7. Analysis output — `Finding`

`Analyze(fixture)` returns a sorted `[]Finding`:
```jsonc
{ "type":"over-permissive NSG (reachable)", "severity":"High",
  "resource":"nic-web", "evidence":"0.0.0.0/0:443 inbound + route 0.0.0.0/0->Internet + public IP pip-web",
  "reachable": true }
```
`severity` ∈ Critical · High · Medium · Low · Informational. `reachable:true` = the engine
proved an internet-reachable path. **A view reads these; it never recomputes them.**

## 8. The overlay (severity per node) + node-id scheme

`overlay = compute_overlay(fixture)` returns:
```jsonc
{ "<kind>:<name>": { "severity":"High", "bucket":"High", "findings":[ Finding, ... ] } }
```
Only nodes that carry a finding appear; `severity_of(overlay, id)` returns `"Clean"` for any
absent id. **Node-id scheme (these strings are the join keys — match them exactly):**

| id | resource |
|---|---|
| `nic:<rid>` | NIC (`rid` = `id` if present else `name`) |
| `pip:<rid>` | public IP |
| `vnet:<a>` `vnet:<b>` | CIDR-overlap / cross-sub pair |
| `appgw:` `aks:` `apim:` `fd:` `vhub:` `pe:<name>` | app-layer families |

## 9. Severity palette (draw.io fill/stroke + legend badge)

| bucket | fill | stroke | badge |
|---|---|---|---|
| Critical | `#f8cecc` | `#b85450` | 🔴 |
| High | `#ffe6cc` | `#d79b00` | 🟠 |
| Medium | `#fff2cc` | `#d6b656` | 🟡 |
| Info (Low+Informational) | `#dae8fc` | `#6c8ebf` | 🔵 |
| Clean | `#d5e8d4` | `#82b366` | 🟢 |

Structural / unscored nodes (boundary, data-node, Bastion): grey `fill=#f5f5f5 stroke=#666666`;
subnets `fill=#fafafa stroke=#b3b3b3` — **disjoint** from the severity palette so grey is never
misread as a severity.

## 10. draw.io output header (determinism)

Emit a **static** header — no timestamp/etag/modified:
```xml
<mxfile host="app.diagrams.net">
  <diagram name="..." id="...">
    <mxGraphModel ... >
      <root>
        <mxCell id="0" /><mxCell id="1" parent="0" />
        <!-- vertices: <mxCell ... vertex="1" parent="1"><mxGeometry x y width height as="geometry"/></mxCell> -->
        <!-- edges:    <mxCell ... edge="1" parent="1" source target><mxGeometry relative="1" as="geometry"/></mxCell> -->
```
Invariants: all cell ids globally unique (vertices + edges); every edge `source`/`target` is a
drawn vertex id (no dangling); byte-identical re-render.

---

## 11. Worked examples (real, committed)

Copy these as test inputs — they exercise the families above and are already gated:

| Fixture | Exercises |
|---|---|
| `phase-4/fixtures/estate-multisub.json` | multi-sub hub/spoke, peerings, exposure + segmentation findings |
| `engine/go/testdata/fixture-f8-aks-and-crosssub-peering.json` | AKS + cross-sub peering |
| `engine/go/testdata/fixture-f6-pe-dns-misconfiguration.json` | Private Endpoint + private DNS zone |
| `engine/go/testdata/fixture-f7-appgw-waf-disabled.json` | App Gateway + WAF posture |

To see the IR → findings → overlay pipeline on any of them:
```bash
python3 engine/reference/analyze.py <fixture.json>        # -> [] Finding (JSON)
python3 phase-4/viz/overlay.py <fixture.json> --print     # -> overlay (node -> severity)
make demo FX=<fixture.json>                                # analyze -> view families -> report
```
