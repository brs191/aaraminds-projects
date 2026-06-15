# Topology Data Model — Phase 1

**Document:** `phase-1/design/TOPOLOGY_MODEL.md`
**Date:** 2026-06-12
**Status:** LOCKED — 5 rework passes complete (2026-06-12). 13 analysis rules; 13/13 golden tests pass. Model is frozen for Step 1.3 adapter implementation. See §16 for full resource coverage and rule registry.
**Scope:** Defines every Azure API source, query, and assembly step required to populate `graph.Fixture` from a live Azure subscription. This document is the authoritative specification for `phase-1/adapter/`.

---

## Contents

1. [Field Mapping Table](#1-field-mapping-table)
2. [Query Catalogue](#2-query-catalogue)
3. [Network Watcher Calls](#3-network-watcher-calls)
4. [Assembly Sequence](#4-assembly-sequence)
5. [Dual-Field Resolution — `Source` vs `SourceAddressPrefix`](#5-dual-field-resolution)
6. [RBAC Role Set](#6-rbac-role-set)
7. [Phase 2 Placeholders](#7-phase-2-placeholders)
8. [PrivateDnsZone — Field Mapping and KQL](#8-privatednszonestructure)
9. [ApplicationGateway — Field Mapping and KQL](#9-applicationgateway-structure)
10. [AKSCluster — Field Mapping and KQL](#10-akscluster-structure)
11. [NatGateway — Field Mapping and KQL](#11-natgateway-structure)
12. [PrivateLinkService — Field Mapping and KQL](#12-privatelinkservice-structure)
13. [ExpressRouteCircuit — Field Mapping and KQL](#13-expressroutecircuit-structure)
14. [Multi-Subscription Support](#14-multi-subscription-support)

---

## 1. Field Mapping Table

### Notation

| Symbol | Meaning |
|---|---|
| **Declared config** | Reflects what an operator configured; may not reflect what traffic actually sees |
| **Evaluated truth** | Reflects what Azure's control plane has resolved after inheritance, defaults, and overrides — this is what the engine needs for correctness |
| **Yes** | Required for current analysis gates |
| **No** | Populated for completeness / observability, not read by engine |
| **Phase-2** | Not in current analysis; required for `simulate_change` / `forecast_cost` |

---

### 1.1 `Fixture` (top-level envelope)

| Field | Go type | Azure API source | Config vs Truth | Example value | Required |
|---|---|---|---|---|---|
| `Subscription` | `string` | Caller input / `az account show --query id` | n/a | `"xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"` | Yes |
| `ResourceGraph` | `ResourceGraph` | Resource Graph bulk KQL — see §2 | Declared config | *(composite)* | Yes |
| `NetworkWatcher` | `NetworkWatcher` | NW REST API — per-NIC POST calls — see §3 | **Evaluated truth** | *(composite)* | Yes |
| `AVNM` | `AVNM` | AVNM REST API walk or Resource Graph KQL — see §2.6 | Declared config | *(composite)* | Yes |
| `AzureFirewall` | `*Firewall` | Resource Graph KQL `microsoft.network/azurefirewalls` — see §2.7 | Declared config | `nil` if no firewall | Conditional |

---

### 1.2 `VNet`

ARM type: `microsoft.network/virtualnetworks`
ARM property root: `properties`

| Field | Go type | ARM JSON path | Config vs Truth | Example value | Required |
|---|---|---|---|---|---|
| `Name` | `string` | `name` (top-level resource name) | Declared | `"hub-vnet"` | Yes |
| `AddressSpace` | `[]string` | `properties.addressSpace.addressPrefixes` | Declared | `["10.0.0.0/16"]` | Yes — CIDR overlap detection |
| `Subnets` | `[]Subnet` | `properties.subnets[]` | Declared | *(see 1.3)* | Yes |
| `Peerings` | `[]Peering` | `properties.virtualNetworkPeerings[]` | Declared (state is evaluated) | *(see 1.4)* | Yes |

---

### 1.3 `Subnet`

Nested inside `VNet.Subnets`. ARM source: `properties.subnets[]` within the VNet resource.

| Field | Go type | ARM JSON path | Config vs Truth | Example value | Required |
|---|---|---|---|---|---|
| `Name` | `string` | `properties.subnets[].name` | Declared | `"web"` | Yes |
| `AddressPrefix` | `string` | `properties.subnets[].properties.addressPrefix` | Declared | `"10.1.1.0/24"` | Yes |
| `NetworkSecurityGroup` | `string` | `properties.subnets[].properties.networkSecurityGroup.id` → extract `name` segment | Declared | `"nsg-web-a"` | Yes |
| `RouteTable` | `string` | `properties.subnets[].properties.routeTable.id` → extract `name` segment | Declared | `"rt-spoke-a"` | Yes |

**Adapter note — name extraction:** ARM resource IDs follow the pattern `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/networkSecurityGroups/{name}`. Extract the final path segment after the last `/`. Both `NetworkSecurityGroup` and `RouteTable` must be set to the bare resource name (e.g., `"nsg-web-a"`), not the full resource ID. Set to empty string `""` if the field is `null` in ARM.

---

### 1.4 `Peering`

Nested inside `VNet.Peerings`. ARM source: `properties.virtualNetworkPeerings[]` within the VNet resource.

| Field | Go type | ARM JSON path | Config vs Truth | Example value | Required |
|---|---|---|---|---|---|
| `RemoteVnet` | `string` | `properties.virtualNetworkPeerings[].properties.remoteVirtualNetwork.id` → extract VNet name | Declared | `"spoke-a-vnet"` | Yes |
| `State` | `string` | `properties.virtualNetworkPeerings[].properties.peeringState` | **Evaluated** — Azure resolves both sides | `"Connected"` \| `"Disconnected"` \| `"Initiated"` | Yes |
| `AllowForwardedTraffic` | `bool` | `properties.virtualNetworkPeerings[].properties.allowForwardedTraffic` | Declared | `true` | Phase-2 |
| `AllowGatewayTransit` | `bool` | `properties.virtualNetworkPeerings[].properties.allowGatewayTransit` | Declared | `false` | Phase-2 |
| `UseRemoteGateways` | `bool` | `properties.virtualNetworkPeerings[].properties.useRemoteGateways` | Declared | `true` | Phase-2 |

**Adapter note — RemoteVnet name extraction:** The `remoteVirtualNetwork.id` ARM resource ID contains the VNet name as the segment after `virtualNetworks/`. Extract it with a split on `/` and take the element after `virtualNetworks`. Example: `.../virtualNetworks/spoke-a-vnet` → `"spoke-a-vnet"`.

**Adapter note — Phase-2 peering fields (TMR-005):** `AllowForwardedTraffic`, `AllowGatewayTransit`, and `UseRemoteGateways` are marked Phase-2 because `analyze.go` does not currently read them. However, the adapter **must still populate them** from ARM during Phase 1 data collection — they are directly available in the Resource Graph VNet response and require no additional API calls. Omitting them now would require a separate re-fetch when Phase 2 simulation lands. Populate them; do not consume them in Phase 1 analysis.

---

### 1.5 `NSG`

ARM type: `microsoft.network/networksecuritygroups`

| Field | Go type | ARM JSON path | Config vs Truth | Example value | Required |
|---|---|---|---|---|---|
| `Name` | `string` | `name` | Declared | `"nsg-web-a"` | Yes |
| `SecurityRules` | `[]SecRule` | `properties.securityRules[]` | **Declared** — user-defined rules only; does **not** include Azure default rules | *(see 1.6)* | Yes — used for NSG resource finding; engine uses effective rules for gates |
| `AssociatedSubnets` | `[]string` | `properties.subnets[].id` → extract `{vnetName}/{subnetName}` | Declared | `["spoke-a-vnet/web"]` | Yes |

**Critical distinction:** `NSG.SecurityRules` (declared) and `NetworkWatcher.EffectiveSecurityRules[nicName]` (evaluated) are different data sources for different purposes. The engine's 4-gate analysis reads **only** effective rules. The declared `NSG.SecurityRules` are used solely to emit the NSG resource in findings and spot-checks. Do not conflate them.

**Adapter note — AssociatedSubnets format:** The ARM field `properties.subnets[].id` is a full resource ID like `.../virtualNetworks/{vnetName}/subnets/{subnetName}`. The adapter must reformat this as `"{vnetName}/{subnetName}"` (same convention as `NIC.Subnet`). This is a custom abbreviated ID used throughout `graph.Fixture` — it is not an ARM resource ID.

---

### 1.6 `SecRule`

Used in two distinct contexts:
- `NSG.SecurityRules[]` — declared rules, sourced from Resource Graph
- `NetworkWatcher.EffectiveSecurityRules[nicName][]` — evaluated/effective rules, sourced from NW REST API

The struct is identical for both uses. The `Source` field is only populated for effective rules (see §5).

| Field | Go type | ARM / NW response JSON path | Config vs Truth | Example value | Required |
|---|---|---|---|---|---|
| `Name` | `string` | `name` (Resource Graph) or `name` (NW response) | Both | `"allow-ssh"` | Yes |
| `Priority` | `int` | `properties.priority` (RG) or `priority` (NW) | Both | `200` | Yes — Gate 2 rule ordering |
| `Direction` | `string` | `properties.direction` (RG) or `direction` (NW) | Both | `"Inbound"` \| `"Outbound"` | Yes — engine filters on `"Inbound"` |
| `Access` | `string` | `properties.access` (RG) or `access` (NW) | Both | `"Allow"` \| `"Deny"` | Yes — Gate 2 verdict |
| `Protocol` | `string` | `properties.protocol` (RG) or `protocol` (NW) | Both | `"Tcp"` \| `"Udp"` \| `"*"` | Yes |
| `SourceAddressPrefix` | `string` | `properties.sourceAddressPrefix` (RG) or `sourceAddressPrefix` (NW) | Both | `"0.0.0.0/0"` \| `"Internet"` \| `"AzureCloud"` \| `"*"` | **Yes — canonical** — engine reads this exclusively |
| `DestinationPortRange` | `string` | `properties.destinationPortRange` (RG) or `destinationPortRange` (NW) | Both | `"22"` \| `"443"` \| `"0-65535"` | Yes — Gate 2 port check |
| `Source` | `string` | Adapter-populated only — no native ARM field | n/a — adapter convention | `"subnet:web"` | No — not read by engine; set equal to `SourceAddressPrefix` for forward compat (see §5) |

**Multi-value handling:** The NW effective rules response may return `sourceAddressPrefixes` (plural array) and `destinationPortRanges` (plural array) when a rule covers multiple values. The adapter must expand multi-value responses into separate `SecRule` entries — one per `(sourceAddressPrefix, destinationPortRange)` combination — because the engine performs exact string comparisons on these fields.

---

### 1.7 `RouteTable`

ARM type: `microsoft.network/routetables`

| Field | Go type | ARM JSON path | Config vs Truth | Example value | Required |
|---|---|---|---|---|---|
| `Name` | `string` | `name` | Declared | `"rt-spoke-a"` | Yes |
| `Routes` | `[]Route` | `properties.routes[]` | **Declared** — user-defined UDRs only; does **not** include system routes or BGP-learned routes | *(see 1.8)* | Yes — adapter must populate; engine uses effective routes for gate 3 |
| `AssociatedSubnets` | `[]string` | `properties.subnets[].id` → extract `{vnetName}/{subnetName}` | Declared | `["spoke-a-vnet/web"]` | Yes |

**Same declared vs evaluated distinction as NSG:** `RouteTable.Routes` (declared UDRs) and `NetworkWatcher.EffectiveRoutes[nicName]` (evaluated, including system routes + BGP) are different sources. Gate 3 (`0.0.0.0/0 → Internet` check) reads **only** effective routes.

---

### 1.8 `Route`

Used in two contexts:
- `RouteTable.Routes[]` — declared UDRs, sourced from Resource Graph
- `NetworkWatcher.EffectiveRoutes[nicName][]` — evaluated effective routes, sourced from NW REST API

| Field | Go type | ARM / NW response JSON path | Config vs Truth | Example value | Required |
|---|---|---|---|---|---|
| `Name` | `string` | `name` (RG) or `name` (NW, may be empty for system routes) | Both | `"default-to-fw"` | No — not read by engine for gate logic |
| `AddressPrefix` | `string` | `properties.addressPrefix` (RG) or `addressPrefix[]` first element (NW) | Both | `"0.0.0.0/0"` \| `"10.1.0.0/16"` | Yes — engine matches `"0.0.0.0/0"` exactly |
| `NextHopType` | `string` | `properties.nextHopType` (RG) or `nextHopType` (NW) | Declared (RG) / **Evaluated** (NW) | `"Internet"` \| `"VirtualAppliance"` \| `"VnetLocal"` \| `"None"` \| `"VnetPeering"` | Yes — Gate 3 checks `== "Internet"` |
| `NextHopIPAddress` | `string` | `properties.nextHopIpAddress` (RG) or `nextHopIpAddresses[]` first element (NW) | Declared (RG) / **Evaluated** (NW) | `"10.0.0.4"` | No — not read by engine directly (used for evidence strings) |

**Adapter note — NW effective routes array:** The NW response `addressPrefix` field is an array. Take `addressPrefix[0]`. Similarly `nextHopIpAddresses` is an array; take `[0]` or empty string if absent.

---

### 1.9 `PublicIP`

ARM type: `microsoft.network/publicipaddresses`

| Field | Go type | ARM JSON path | Config vs Truth | Example value | Required |
|---|---|---|---|---|---|
| `Name` | `string` | `name` | Declared | `"pip-vm-web-a"` | Yes |
| `IPAddress` | `string` | `properties.ipAddress` | **Evaluated** — assigned by Azure (empty if not yet allocated) | `"20.51.10.10"` | Yes — included in evidence string |
| `IPConfiguration` | `*string` | `properties.ipConfiguration.id` → extract `{resourceType}/{resourceName}/ipconfig{N}` short form | Declared | `"nic-vm-web-a/ipconfig1"` | Yes — `nil` (JSON `null`) signals orphaned endpoint |

**Orphaned PIP detection:** The engine fires the `"orphaned public endpoint"` finding when `pip.IPConfiguration == nil || *pip.IPConfiguration == ""`. The ARM field `properties.ipConfiguration` is `null` when the PIP is not attached to any resource. Set Go pointer to `nil` when the ARM value is absent or null; set to a non-empty string otherwise. The exact string format of the value is not parsed by the engine — any non-empty string is sufficient to suppress the finding.

---

### 1.10 `NIC`

ARM type: `microsoft.network/networkinterfaces`

| Field | Go type | ARM JSON path | Config vs Truth | Example value | Required |
|---|---|---|---|---|---|
| `Name` | `string` | `name` | Declared | `"nic-vm-web-a"` | Yes — primary key for NW lookup maps |
| `Subnet` | `string` | `properties.ipConfigurations[0].properties.subnet.id` → extract `"{vnetName}/{subnetName}"` | Declared | `"spoke-a-vnet/web"` | Yes — `nicVnet()` extracts VNet name from this via `strings.Index(subnet, "/")` |
| `NetworkSecurityGroup` | `*string` | `properties.networkSecurityGroup.id` → extract `name`; `nil` if absent | Declared | `"nsg-vm"` or `nil` | No — engine reads effective rules from NW, not NIC-level NSG ref |
| `PublicIP` | `*string` | `properties.ipConfigurations[0].properties.publicIPAddress.id` → extract `name`; `nil` if absent | Declared | `"pip-vm-web-a"` or `nil` | Yes — **Gate 4**: `nic.PublicIP != nil && *nic.PublicIP != ""` |
| `PrivateIP` | `string` | `properties.ipConfigurations[0].properties.privateIPAddress` | **Evaluated** — assigned by Azure | `"10.1.1.4"` | Yes — matched against `Firewall.NatRules[].TranslatedAddress` for DNAT path |
| `Tags` | `map[string]string` | `tags` | Declared | `{"sensitive": "true", "tier": "web"}` | Yes — `Tags["sensitive"] == "true"` escalates severity to Critical |

**Adapter note — Subnet format:** The ARM `subnet.id` is a full resource ID. Extract the VNet name and subnet name from it:
- VNet: segment after `virtualNetworks/`
- Subnet: segment after `subnets/`

Combine as `"{vnetName}/{subnetName}"`. The engine's `nicVnet(nic)` function does `strings.Index(n.Subnet, "/")` and takes everything before the first `/` as the VNet name. This convention must be preserved exactly.

**Multi-IP configuration:** Take `ipConfigurations[0]` (the primary IP configuration) for `Subnet`, `PublicIP`, and `PrivateIP`. Phase 2 may need secondary IPs.

---

### 1.11 `NetworkWatcher`

Not sourced from Resource Graph. Populated exclusively from per-NIC REST API calls (see §3).

| Field | Go type | Source | Config vs Truth | Example key | Required |
|---|---|---|---|---|---|
| `EffectiveSecurityRules` | `map[string][]SecRule` | NW REST API — `effectiveNetworkSecurityGroups` POST per NIC | **Evaluated truth** — includes inherited defaults, NSG rules, AVNM-applied rules at the NIC layer | `"nic-vm-web-a"` → `[]SecRule` | Yes — Gate 2 |
| `EffectiveRoutes` | `map[string][]Route` | NW REST API — `effectiveRouteTable` POST per NIC | **Evaluated truth** — includes UDRs, system routes, BGP-learned routes | `"nic-vm-web-a"` → `[]Route` | Yes — Gate 3 |

Map key is the **NIC bare name** (not the full resource ID), matching `NIC.Name`. This is the key the engine uses to look up rules and routes per NIC.

---

### 1.12 `AVNM`

| Field | Go type | Source | Config vs Truth | Example | Required |
|---|---|---|---|---|---|
| `SecurityAdminRules` | `[]AdminRule` | AVNM REST API walk (see §2.6) or Resource Graph `microsoft.network/networkmanagers/securityadminconfigurations/rulecollections/rules` | Declared config — Network Manager admin rules override NSG Deny but are themselves declared | *(see 1.13)* | Yes — Gate 1 |

---

### 1.13 `AdminRule`

| Field | Go type | ARM JSON path | Config vs Truth | Example value | Required |
|---|---|---|---|---|---|
| `Name` | `string` | `name` | Declared | `"deny-rdp-from-internet"` | Yes |
| `Priority` | `int` | `properties.priority` | Declared | `10` | Yes — `adminVerdict()` picks lowest priority matching rule |
| `Direction` | `string` | `properties.direction` | Declared | `"Inbound"` | Yes — engine filters on `"Inbound"` |
| `Access` | `string` | `properties.access` | Declared | `"Deny"` \| `"Allow"` \| `"AlwaysAllow"` | Yes — Gate 1 verdict |
| `Protocol` | `string` | `properties.protocol` | Declared | `"Tcp"` | Yes |
| `SourceAddressPrefix` | `string` | `properties.sources[0].addressPrefix` or `properties.sourceAddressPrefixes[0]` | Declared | `"Internet"` \| `"0.0.0.0/0"` \| `"*"` | Yes — `adminVerdict()` matches only `"internet"`, `"0.0.0.0/0"`, `"*"` (lowercased) |
| `DestinationPortRange` | `string` | `properties.destinationPortRanges[0]` or `properties.destinationPortRange` | Declared | `"3389"` \| `"443"` | Yes — exact string match in `adminVerdict()`; expand multi-value arrays, do not keep only `[0]` |
| `AppliesTo` | `[]string` | Derived from rule collection's `appliesToGroups[].networkGroupId` → resolve to VNet names | Declared | `["svc-vnet"]` | Yes — `adminVerdict()` checks `contains(ar.AppliesTo, vnet)` |

**Adapter note — AppliesTo resolution:** This is the most complex field. The rule collection's `appliesToGroups[]` contains Network Group resource IDs. Each Network Group's membership must be resolved to VNet names. Resolution path: Network Group → members (static or dynamic policy) → VNet names. See §2.6 for the full walk. For Phase 1, static membership groups are sufficient; dynamic (Azure Policy) membership requires [VERIFY] on live environment.

**Adapter note — SourceAddressPrefix for AVNM:** The AVNM REST API returns `sources[]` (an array of `{addressPrefix, addressPrefixType}` objects) rather than a flat `sourceAddressPrefix` string. If multiple source prefixes exist in a rule, expand into separate `AdminRule` entries.

**Adapter note — DestinationPortRange for AVNM:** `adminVerdict()` does an exact string comparison on `DestinationPortRange`. If AVNM returns `destinationPortRanges[]`, the adapter must expand them into separate `AdminRule` entries instead of keeping only `[0]`. When both `sources[]` and `destinationPortRanges[]` contain multiple values, emit the full Cartesian product so every `(sourceAddressPrefix, destinationPortRange)` pair is represented explicitly.

---

### 1.14 `Firewall`

ARM type: `microsoft.network/azurefirewalls`

| Field | Go type | ARM JSON path | Config vs Truth | Example value | Required |
|---|---|---|---|---|---|
| `Name` | `string` | `name` | Declared | `"afw-hub"` | Yes |
| `PrivateIP` | `string` | `properties.ipConfigurations[0].properties.privateIPAddress` | **Evaluated** | `"10.0.0.4"` | Yes — used in DNAT evidence string |
| `PublicIP` | `string` | `properties.ipConfigurations[0].properties.publicIPAddress.id` → follow to PIP resource → `properties.ipAddress` | **Evaluated** | `"20.70.0.10"` | Yes — used in DNAT evidence string |
| `NatRules` | `[]NatRule` | ARM GET on classic firewall `properties.natRuleCollections[].properties.rules[]` or Firewall Policy rule collection groups | Declared | *(see 1.15)* | Yes — DNAT path detection |

**Policy-based firewall:** When `properties.firewallPolicy.id` is set, the firewall uses a Firewall Policy and does **not** have inline `natRuleCollections`. The adapter must detect this and [VERIFY] whether the Resource Graph `microsoft.network/firewallpolicies/rulecollectiongroups` type exposes NAT rules, or whether an ARM GET on the policy resource is required. Mark as `[VERIFY]` in adapter implementation.

---

### 1.15 `NatRule`

Nested in `Firewall.NatRules`. ARM source:
- Classic firewall: `properties.natRuleCollections[].properties.rules[]` from an ARM GET on the firewall resource
- Policy-based firewall: `properties.ruleCollections[].rules[]` within Firewall Policy rule collection groups

| Field | Go type | ARM JSON path | Config vs Truth | Example value | Required |
|---|---|---|---|---|---|
| `Name` | `string` | `name` | Declared | `"dnat-https-to-backend1"` | Yes |
| `Protocol` | `string` | `properties.protocols[0]` | Declared | `"Tcp"` | Yes |
| `SourceAddresses` | `[]string` | `properties.sourceAddresses[]` | Declared | `["*"]` | Yes — included in evidence string |
| `DestinationAddress` | `string` | `properties.destinationAddresses[0]` | Declared | `"20.70.0.10"` | Yes — should match `Firewall.PublicIP` |
| `DestinationPort` | `int` | `properties.destinationPorts[0]` → parse string to int | Declared | `443` | Yes |
| `TranslatedAddress` | `string` | `properties.translatedAddress` | Declared | `"10.8.1.4"` | Yes — **critical**: matched against `NIC.PrivateIP` to detect DNAT exposure |
| `TranslatedPort` | `int` | `properties.translatedPort` → parse string to int | Declared | `443` | Yes |

---

## 2. Query Catalogue

All queries are subscription-scoped using `where subscriptionId == "{sub}"`. Replace `{sub}` with the target subscription GUID at runtime. All queries use the `Resources` table. Return raw properties objects — the adapter will map them to Go structs.

**API version for Resource Graph queries:** `2024-04-01` (stable). Pass via the `$skipToken` query parameter pattern when result sets exceed 1000 rows.

---

### 2.1 Virtual Networks

```kql
Resources
| where subscriptionId == "{sub}"
| where type == "microsoft.network/virtualnetworks"
| project
    name,
    resourceGroup,
    location,
    addressPrefixes = properties.addressSpace.addressPrefixes,
    subnets = properties.subnets,
    peerings = properties.virtualNetworkPeerings
```

The `subnets` column is a JSON array of subnet objects. Each element contains:
- `.name` → `Subnet.Name`
- `.properties.addressPrefix` → `Subnet.AddressPrefix`
- `.properties.networkSecurityGroup.id` → `Subnet.NetworkSecurityGroup` (extract name)
- `.properties.routeTable.id` → `Subnet.RouteTable` (extract name)

The `peerings` column is a JSON array of peering objects. Each element contains:
- `.name` → (not in struct, use for logging)
- `.properties.remoteVirtualNetwork.id` → `Peering.RemoteVnet` (extract VNet name)
- `.properties.peeringState` → `Peering.State`
- `.properties.allowForwardedTraffic` → `Peering.AllowForwardedTraffic`
- `.properties.allowGatewayTransit` → `Peering.AllowGatewayTransit`
- `.properties.useRemoteGateways` → `Peering.UseRemoteGateways`

**Pagination note:** A subscription with 500+ VNets exceeds the Resource Graph 1000-row default. Use `$skipToken` to page through results. The adapter must implement a paging loop.

---

### 2.2 Network Security Groups

```kql
Resources
| where subscriptionId == "{sub}"
| where type == "microsoft.network/networksecuritygroups"
| project
    name,
    resourceGroup,
    location,
    securityRules = properties.securityRules,
    associatedSubnets = properties.subnets
```

The `securityRules` column is an array of rule objects. Each element contains:
- `.name` → `SecRule.Name`
- `.properties.priority` → `SecRule.Priority`
- `.properties.direction` → `SecRule.Direction`
- `.properties.access` → `SecRule.Access`
- `.properties.protocol` → `SecRule.Protocol`
- `.properties.sourceAddressPrefix` → `SecRule.SourceAddressPrefix` (also copy to `SecRule.Source`)
- `.properties.destinationPortRange` → `SecRule.DestinationPortRange`

The `associatedSubnets` column is an array of subnet resource ID objects (`[].id`). Each ID must be converted to `"{vnetName}/{subnetName}"` format.

**Default rules exclusion:** The ARM API returns both user-defined rules in `properties.securityRules` and Azure default rules in `properties.defaultSecurityRules`. The KQL above projects only `properties.securityRules`. Default rules (AllowVnetInBound, DenyAllInBound, etc.) appear in Network Watcher effective rules — do not include them in `NSG.SecurityRules`. They are not needed for `NSG.SecurityRules`; the engine only uses effective rules for gate analysis.

---

### 2.3 Route Tables

```kql
Resources
| where subscriptionId == "{sub}"
| where type == "microsoft.network/routetables"
| project
    name,
    resourceGroup,
    location,
    routes = properties.routes,
    associatedSubnets = properties.subnets,
    disableBgpRoutePropagation = properties.disableBgpRoutePropagation
```

The `routes` column is an array. Each element contains:
- `.name` → `Route.Name`
- `.properties.addressPrefix` → `Route.AddressPrefix`
- `.properties.nextHopType` → `Route.NextHopType`
- `.properties.nextHopIpAddress` → `Route.NextHopIPAddress` (may be absent; use `""`)

The `associatedSubnets` column is an array of subnet resource ID objects (`[].id`), convert to `"{vnetName}/{subnetName}"` format.

**`disableBgpRoutePropagation`** is not in `RouteTable` struct but is needed for Phase 2 analysis. Capture it now — mark as Phase-2. See §7.

---

### 2.4 Public IP Addresses

```kql
Resources
| where subscriptionId == "{sub}"
| where type == "microsoft.network/publicipaddresses"
| project
    name,
    resourceGroup,
    location,
    ipAddress = properties.ipAddress,
    ipConfiguration = properties.ipConfiguration.id,
    allocationMethod = properties.publicIPAllocationMethod,
    sku = sku.name
```

Mapping:
- `name` → `PublicIP.Name`
- `ipAddress` → `PublicIP.IPAddress` (may be `""` if Dynamic + not attached; set to `""` not null)
- `ipConfiguration` → `PublicIP.IPConfiguration` as `*string`: `null` → Go `nil`; any string value → Go pointer to that string. The exact string value is not parsed by the engine — any non-nil non-empty value is sufficient.
- `allocationMethod` and `sku` — **Phase-2 fields**, captured now for cost estimation. See §7.

---

### 2.5 Network Interfaces

```kql
Resources
| where subscriptionId == "{sub}"
| where type == "microsoft.network/networkinterfaces"
| project
    name,
    resourceGroup,
    location,
    ipConfigurations = properties.ipConfigurations,
    nsgId = properties.networkSecurityGroup.id,
    tags
```

The `ipConfigurations` column is an array. Use `ipConfigurations[0]` for all NIC-level fields:
- `ipConfigurations[0].properties.subnet.id` → `NIC.Subnet` (convert to `"{vnetName}/{subnetName}"`)
- `ipConfigurations[0].properties.publicIPAddress.id` → `NIC.PublicIP` (extract name → `*string`; `null` → `nil`)
- `ipConfigurations[0].properties.privateIPAddress` → `NIC.PrivateIP`

Additional mappings:
- `nsgId` → `NIC.NetworkSecurityGroup` (extract name → `*string`; `null` → `nil`)
- `tags` → `NIC.Tags` as `map[string]string`

**Resource Group capture:** Capture `resourceGroup` and `location` from this query — they are required to construct the Network Watcher API URL in §3. The resource group and location are not in the `NIC` struct but must be tracked in the adapter's internal NIC metadata.

---

### 2.6 AVNM Security Admin Rules

AVNM resources are not reliably available in Resource Graph for all rule collection types. The authoritative source is the AVNM REST API. Use the following walk:

#### Step 1 — Discover Network Managers (Resource Graph)

```kql
Resources
| where subscriptionId == "{sub}"
| where type == "microsoft.network/networkmanagers"
| project
    name,
    resourceGroup,
    id
```

#### Step 2 — List Security Admin Configurations (ARM REST)

For each Network Manager discovered:

```
GET https://management.azure.com/subscriptions/{sub}/resourceGroups/{rg}/providers/
    Microsoft.Network/networkManagers/{nm}/securityAdminConfigurations
    ?api-version=2024-03-01
```

#### Step 3 — List Rule Collections (ARM REST)

For each Security Admin Configuration:

```
GET https://management.azure.com/subscriptions/{sub}/resourceGroups/{rg}/providers/
    Microsoft.Network/networkManagers/{nm}/securityAdminConfigurations/{config}/ruleCollections
    ?api-version=2024-03-01
```

Each rule collection has `properties.appliesToGroups[]` — capture these for step 5.

#### Step 4 — List Rules (ARM REST)

For each Rule Collection:

```
GET https://management.azure.com/subscriptions/{sub}/resourceGroups/{rg}/providers/
    Microsoft.Network/networkManagers/{nm}/securityAdminConfigurations/{config}/
    ruleCollections/{collection}/rules
    ?api-version=2024-03-01
```

Response fields → `AdminRule` struct:
- `name` → `AdminRule.Name`
- `properties.priority` → `AdminRule.Priority`
- `properties.direction` → `AdminRule.Direction`
- `properties.access` → `AdminRule.Access`
- `properties.protocol` → `AdminRule.Protocol`
- `properties.sources[]` → `AdminRule.SourceAddressPrefix` after expansion to one entry per source prefix
- `properties.destinationPortRanges[]` or `properties.destinationPortRange` → `AdminRule.DestinationPortRange` after expansion to one entry per destination port range
- `AppliesTo` → resolved from the parent rule collection's `appliesToGroups` (see step 5)

#### Step 5 — Resolve Network Groups to VNet Names (ARM REST)

For each Network Group referenced in `appliesToGroups[].networkGroupId`:

```
GET https://management.azure.com/subscriptions/{sub}/resourceGroups/{rg}/providers/
    Microsoft.Network/networkManagers/{nm}/networkGroups/{group}/staticMembers
    ?api-version=2024-03-01
```

Each static member has `properties.resourceId` pointing to a VNet resource ID. Extract the VNet name. Populate `AdminRule.AppliesTo` with the list of VNet names from all members of all network groups referenced by the rule collection.

**`[VERIFY]`** Dynamic membership (Azure Policy-based groups) is not returned by `staticMembers`. Confirm whether the live environment uses static or dynamic group membership. Dynamic membership resolution requires a separate Azure Policy evaluation call and is deferred to Phase 2.

**`[VERIFY]`** Confirm that the `microsoft.network/networkmanagers/securityadminconfigurations/rulecollections/rules` Resource Graph type is available and returns complete rule data in the target subscription's tenant. If it is, step 4 can be replaced with a single KQL query. Test with:
```kql
Resources
| where subscriptionId == "{sub}"
| where type == "microsoft.network/networkmanagers/securityadminconfigurations/rulecollections/rules"
| project name, properties, id
| limit 5
```
Fall back to the REST walk if this returns zero rows.

---

### 2.7 Azure Firewall

```kql
Resources
| where subscriptionId == "{sub}"
| where type == "microsoft.network/azurefirewalls"
| project
    name,
    resourceGroup,
    location,
    privateIp = properties.ipConfigurations[0].properties.privateIPAddress,
    publicIpId = properties.ipConfigurations[0].properties.publicIPAddress.id,
    natRuleCollections = properties.natRuleCollections,
    firewallPolicyId = properties.firewallPolicy.id,
    sku = properties.sku.tier
```

**Classic inline firewall:** If `firewallPolicyId` is null, Resource Graph can be used to discover that the firewall is classic, but `natRuleCollections` must **not** be treated as authoritative for parsing NAT rules because deeply nested arrays may be truncated in Resource Graph responses. For every classic firewall, perform an ARM GET on the firewall resource and read `properties.natRuleCollections[].properties.rules[]` from that response:

```
GET https://management.azure.com/subscriptions/{sub}/resourceGroups/{rg}/providers/
    Microsoft.Network/azureFirewalls/{firewallName}
    ?api-version=2024-03-01
```

Each element of `natRuleCollections[].properties.rules[]` maps to `NatRule`:
- `.name` → `NatRule.Name`
- `.properties.protocols[0]` → `NatRule.Protocol`
- `.properties.sourceAddresses[]` → `NatRule.SourceAddresses`
- `.properties.destinationAddresses[0]` → `NatRule.DestinationAddress`
- `.properties.destinationPorts[0]` → `NatRule.DestinationPort` (parse string to int)
- `.properties.translatedAddress` → `NatRule.TranslatedAddress`
- `.properties.translatedPort` → `NatRule.TranslatedPort` (parse string to int)

**Policy-based firewall:** If `firewallPolicyId` is non-null, perform a separate ARM GET:

```
GET https://management.azure.com/subscriptions/{sub}/resourceGroups/{rg}/providers/
    Microsoft.Network/firewallPolicies/{policyName}/ruleCollectionGroups
    ?api-version=2024-03-01
```

Iterate rule collection groups → rule collections → NAT rules. Map fields from `properties.rules[]` in collections of type `FirewallPolicyNatRuleCollection`.

**Silent data-loss guard:** Do not rely on the Resource Graph `natRuleCollections` payload for classic firewalls. Use the ARM GET above as the authoritative source for NAT rules; use Resource Graph only for discovery fields (`name`, `resourceGroup`, `location`, `publicIpId`, `firewallPolicyId`, `sku`).

**PublicIP resolution:** The `publicIpId` is a resource ID. Resolve the actual IP address by cross-referencing `PublicIPAddresses` already collected in Step A (match by resource ID → extract `IPAddress`).

---

## 3. Network Watcher Calls

Network Watcher (NW) APIs provide **evaluated truth** — the routing and security rules as actually resolved by Azure's dataplane, including system defaults, inherited rules, and BGP-learned routes. These cannot be obtained from Resource Graph.

### 3.1 Pre-condition — Network Watcher Discovery

Each Azure region where NICs reside must have a Network Watcher instance. Network Watchers are provisioned per-subscription per-region, typically in a resource group named `NetworkWatcherRG`.

**Discover Network Watchers:**

```kql
Resources
| where subscriptionId == "{sub}"
| where type == "microsoft.network/networkwatchers"
| project name, resourceGroup, location
```

Build a map: `location → {watcherName, watcherResourceGroup}`. For each NIC, look up the watcher by the NIC's location. If no watcher exists for a region, log a warning and skip NW calls for NICs in that region — the adapter must not fail hard; degrade gracefully (omit the NIC from `EffectiveSecurityRules` / `EffectiveRoutes` maps).

**`[VERIFY]`** Confirm whether all target subscription regions have Network Watchers provisioned. If a watcher is absent, the adapter must log `WARN: no NetworkWatcher in region {region}; {count} NICs will have no effective rule/route data`.

---

### 3.2 Effective Security Rules

**Purpose:** Returns all inbound and outbound security rules effective on a NIC, including NSG rules from subnet-level and NIC-level NSGs plus Azure defaults. This is the authoritative evaluated set that the engine reads for Gate 2.

**Endpoint:**
```
POST https://management.azure.com/subscriptions/{sub}/resourceGroups/{watcherRG}/providers/
     Microsoft.Network/networkWatchers/{watcherName}/effectiveNetworkSecurityGroups
     ?api-version=2023-11-01
```

**HTTP method:** `POST`

**Request body:**
```json
{
  "targetResourceId": "/subscriptions/{sub}/resourceGroups/{nicRG}/providers/Microsoft.Network/networkInterfaces/{nicName}"
}
```

**Required parameters:**

| Parameter | Source | Example |
|---|---|---|
| `sub` | Fixture.Subscription | `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx` |
| `watcherRG` | Discovered in pre-condition step | `NetworkWatcherRG` |
| `watcherName` | Discovered in pre-condition step | `NetworkWatcher_eastus` |
| `nicRG` | NIC resource group from §2.5 query | `rg-prod-eastus` |
| `nicName` | `NIC.Name` | `nic-vm-web-a` |

**Async operation:** This call returns `202 Accepted` with an `Azure-AsyncOperation` or `Location` header. The adapter must poll the operation URL until the response is `200 OK` or a terminal failure status is returned. Use a 2-second poll interval with an overall per-call timeout of 60 seconds; do **not** cap success polling at 10 retries because Azure documents this operation as typically taking 5–20 seconds and tail latency beyond 20 seconds must not be treated as a hard failure.

**Response structure:**
```json
{
  "value": [
    {
      "networkSecurityGroup": { "id": "..." },
      "association": { "subnet": { ... }, "networkInterface": { ... } },
      "effectiveSecurityRules": [
        {
          "name": "allow-ssh",
          "protocol": "Tcp",
          "sourcePortRange": "0-65535",
          "sourcePortRanges": ["0-65535"],
          "destinationPortRange": "22",
          "destinationPortRanges": ["22"],
          "sourceAddressPrefix": "0.0.0.0/0",
          "sourceAddressPrefixes": ["0.0.0.0/0"],
          "destinationAddressPrefix": "*",
          "destinationAddressPrefixes": ["*"],
          "expandedSourceAddressPrefix": null,
          "expandedDestinationAddressPrefix": null,
          "access": "Allow",
          "priority": 200,
          "direction": "Inbound"
        }
      ]
    }
  ]
}
```

**Response JSON path to rules array:** `value[*].effectiveSecurityRules[]`

Iterate all elements of `value[]` (there may be multiple NSG associations — subnet NSG and NIC NSG). Flatten all `effectiveSecurityRules[]` arrays into a single list and preserve every expanded rule entry. Do **not** deduplicate by `name` + `priority` + `direction`: different effective rules can legitimately share those fields while differing in source prefix, port range, access, or association, and collapsing them would silently drop engine-visible data.

**Mapping to `SecRule` struct:**

| Response field | `SecRule` field | Notes |
|---|---|---|
| `name` | `Name` | |
| `priority` | `Priority` | |
| `direction` | `Direction` | |
| `access` | `Access` | |
| `protocol` | `Protocol` | |
| `sourceAddressPrefix` | `SourceAddressPrefix` | If empty, fall back to `sourceAddressPrefixes[0]` |
| `destinationPortRange` | `DestinationPortRange` | If empty, fall back to `destinationPortRanges[0]` |
| *(adapter-set)* | `Source` | Set equal to `SourceAddressPrefix` (see §5) |

**Multi-value expansion:** If `sourceAddressPrefixes` has more than one element, create a separate `SecRule` entry for each prefix, copying all other fields. Same for `destinationPortRanges`. The engine performs exact string comparisons — a rule covering `["22", "3389"]` must be expanded into two separate `SecRule` records.

---

### 3.3 Effective Routes

**Purpose:** Returns the effective routing table for a NIC, including UDRs, system routes, and BGP-learned routes. The engine reads this for Gate 3 (`0.0.0.0/0 → Internet` check).

**Endpoint:**
```
POST https://management.azure.com/subscriptions/{sub}/resourceGroups/{watcherRG}/providers/
     Microsoft.Network/networkWatchers/{watcherName}/effectiveRouteTable
     ?api-version=2023-11-01
```

**HTTP method:** `POST`

**Request body:**
```json
{
  "targetResourceId": "/subscriptions/{sub}/resourceGroups/{nicRG}/providers/Microsoft.Network/networkInterfaces/{nicName}"
}
```

**Required parameters:** Same as §3.2 (sub, watcherRG, watcherName, nicRG, nicName).

**Async operation:** Same polling pattern as §3.2.

**Response structure:**
```json
{
  "value": [
    {
      "name": "default-to-fw",
      "source": "User",
      "state": "Active",
      "addressPrefix": ["0.0.0.0/0"],
      "nextHopType": "VirtualAppliance",
      "nextHopIpAddress": ["10.0.0.4"],
      "disableBgpRoutePropagation": false
    }
  ]
}
```

**Response JSON path to routes array:** `value[]`

**Mapping to `Route` struct:**

| Response field | `Route` field | Notes |
|---|---|---|
| `name` | `Name` | May be empty for system routes; use `""` |
| `addressPrefix[0]` | `AddressPrefix` | Take first element only |
| `nextHopType` | `NextHopType` | Valid values: `"Internet"`, `"VirtualAppliance"`, `"VnetLocal"`, `"VnetPeering"`, `"None"`, `"HyperNetGateway"`, `"VirtualNetworkGateway"` |
| `nextHopIpAddress[0]` | `NextHopIPAddress` | Take first element; use `""` if array is empty or absent |

**Engine gate dependency:** The engine checks:
```go
if r.AddressPrefix == "0.0.0.0/0" {
    defaultHop = r.NextHopType
}
```
The string `"0.0.0.0/0"` must match exactly. Azure API returns this as the first element of `addressPrefix[]`.

---

### 3.4 Parallelism Strategy

Network Watcher APIs are throttled at the Azure subscription level. Exceeding limits returns `429 Too Many Requests`.

**Known throttle limits:** Approximately 100 NW data-plane operations per 5-minute window per subscription. Treat this as an unverified lower bound until confirmed in the target subscription; both effective-rules and effective-routes calls consume the same shared budget. `[VERIFY]` exact limit in target subscription.

**Strategy — bounded concurrency with exponential backoff:**

```
MaxConcurrentCalls = 10   // shared semaphore across both NW APIs; max in-flight calls, not NICs
PollInterval = 2s         // async operation poll cadence
AsyncTimeout = 60s        // per-call deadline for the Azure async operation
BaseRetryDelay = 2s       // initial backoff on 429
MaxRetries = 5            // give up after 5 POST retries (total ~62s of backoff)
JitterFactor = 0.2        // ±20% jitter on each retry delay to prevent thundering herd
```

**Implementation pattern:**

1. Create one shared semaphore (buffered channel of size 10) for **all** Network Watcher calls across both APIs.
2. Treat each POST (`effectiveNetworkSecurityGroups` or `effectiveRouteTable`) as one independently scheduled call that acquires a semaphore slot before it starts.
3. For a single NIC, the rules call and the routes call may be launched concurrently, but only if two shared slots are available at that moment; otherwise one waits. The concurrency ceiling is 10 total in-flight NW calls, not 10 NICs × 2 calls.
4. Poll the Azure async operation every 2 seconds until success, terminal failure, or `AsyncTimeout`. Polling does not start a second POST; it is completion tracking for the original call.
5. On `429` response to the initial POST: release the semaphore, wait `BaseRetryDelay * 2^attempt * (1 ± jitter)`, re-acquire, and retry.
6. On non-retryable error (4xx != 429, 5xx, or async terminal failure): log error with NIC name and continue — do not fail the entire batch.
7. Release the semaphore after each call completes or fails terminally.

**Observability requirements:**

- Log `INFO: starting NW enrichment for {n} NICs` before the loop.
- Log `INFO: NW enrichment complete; {n} NICs processed in {duration}ms; {k} errors` after the loop.
- Log `WARN: NW call throttled for NIC {name}; retrying (attempt {i}/{max})` on each 429.
- Log `ERROR: NW call failed for NIC {name} after {max} retries; omitting from results` on terminal failure.

**Sizing expectation:** Lower bound throughput is approximately `(2 × NIC count / MaxConcurrentCalls) × avgCallDuration` because there are two NW calls per NIC and both APIs share the same pool. Example: 200 NICs at 5 seconds average per completed NW call and a 10-call pool yields roughly `(400 / 10) × 5 = 200 seconds` before retries. Treat this as optimistic; subscription throttling and backoff can extend total runtime materially. Log the total duration; if > 300s, emit a warning and suggest reviewing NIC count and throttle behaviour with operators.

---

## 4. Assembly Sequence

Build `graph.Fixture` in the following ordered steps. Steps B, C, and D can run in parallel after Step A completes. Step E can run in parallel with B/C/D if a firewall was detected in Step A.

```
Step A ──────────────────────────────────────────────────────────── (serial)
  Resource Graph bulk queries for all entity types
  Outputs: NIC list (with RG+location), VNet list, NSG list, RT list, PIP list, Firewall presence

Step B ──────────────────────── Step C ──────────────────── Step D ── (parallel)
  NW Effective Security           NW Effective Routes         AVNM walk
  Rules per NIC                   per NIC                     (REST API)
  (async, semaphore 10)           (async, semaphore 10)

                                                    Step E ── (parallel, conditional)
                                                      Firewall NAT rules
                                                      (only if firewall detected in A)

Step F ──────────────────────────────────────────────────────────── (serial, after B/C/D/E)
  Assemble graph.Fixture
```

---

### Step A — Resource Graph Bulk Queries

Issue all Resource Graph KQL queries in parallel (they are independent). Collect results into adapter-internal structs.

1. Run §2.1 VNet query → parse into `[]VNet`
2. Run §2.2 NSG query → parse into `[]NSG`
3. Run §2.3 Route Table query → parse into `[]RouteTable`
4. Run §2.4 Public IP query → parse into `[]PublicIP`
5. Run §2.5 NIC query → parse into `[]NIC` + internal `map[string]NICMeta{resourceGroup, location}`
6. Run §2.7 Firewall query → parse into `*Firewall` (nil if no results)
7. Run Network Watcher discovery query (§3.1) → build `map[string]WatcherRef{name, rg}` keyed by location

All 7 queries can be issued in parallel. Wait for all to complete before proceeding to Step B.

**Pagination:** Implement `$skipToken` paging for queries that may return >1000 rows (VNets, NICs in large subscriptions). Loop until no `$skipToken` is returned.

---

### Step B — NW Effective Security Rules per NIC

For each NIC in the NIC list from Step A:
1. Look up the NW watcher for `NICMeta.location`.
2. Issue the `effectiveNetworkSecurityGroups` POST (§3.2) with bounded concurrency (semaphore 10).
3. Poll the async operation until completion.
4. Parse the response into `[]SecRule`.
5. Write to `map[string][]SecRule` keyed by `NIC.Name`.

Failures are logged and the NIC key is omitted from the map (not inserted with an empty slice). The engine treats absent map keys as empty slices — no rules → no inbound findings for that NIC.

---

### Step C — NW Effective Routes per NIC

Same pattern as Step B, using the `effectiveRouteTable` POST (§3.3). Write to `map[string][]Route` keyed by `NIC.Name`.

Steps B and C share the same semaphore pool. For each NIC, the two calls may be issued independently, but each consumes its own shared semaphore slot; if only one slot is available, one call waits. Do not model the pool as "10 NICs concurrently" — it is "10 NW calls concurrently".

---

### Step D — AVNM Security Admin Rules

Execute the AVNM REST API walk defined in §2.6 (steps 1–5). This is typically a small number of API calls (one per network manager, configuration, and rule collection) and does not require bounded concurrency. Collect all `AdminRule` entries into `[]AdminRule`.

If the subscription has no Network Managers (§2.6 Step 1 returns zero rows), `AVNM.SecurityAdminRules` is set to an empty slice `[]`. Do not emit an error — absence of AVNM is valid.

---

### Step E — Azure Firewall NAT Rules (Conditional)

Only execute if Step A's firewall query returned a non-nil result.

- **Classic firewall:** Do **not** parse NAT rules from the Resource Graph payload. Issue an ARM GET on the firewall resource (§2.7) and parse `properties.natRuleCollections[].properties.rules[]` from that authoritative response.
- **Policy-based firewall:** Issue ARM GET on the Firewall Policy to retrieve rule collection groups (§2.7). Parse NAT rule collection groups into `[]NatRule`.

Resolve `Firewall.PublicIP` from the `PublicIPAddresses` collected in Step A by matching `properties.ipConfiguration.id` against the firewall's resource ID.

---

### Step F — Assemble `graph.Fixture`

After Steps A–E are complete, construct `graph.Fixture` with the following exact field mapping:

```
graph.Fixture{
    Subscription: <caller-supplied subscription ID>,

    ResourceGraph: graph.ResourceGraph{
        VirtualNetworks:       <parsed []VNet from Step A §2.1>,
        NetworkSecurityGroups: <parsed []NSG from Step A §2.2>,
        RouteTables:           <parsed []RouteTable from Step A §2.3>,
        PublicIPAddresses:     <parsed []PublicIP from Step A §2.4>,
        NetworkInterfaces:     <parsed []NIC from Step A §2.5>,
    },

    NetworkWatcher: graph.NetworkWatcher{
        EffectiveSecurityRules: <map[nicName][]SecRule from Step B>,
        EffectiveRoutes:        <map[nicName][]Route from Step C>,
    },

    AVNM: graph.AVNM{
        SecurityAdminRules: <[]AdminRule from Step D>,
    },

    AzureFirewall: <*Firewall from Step E; nil if no firewall>,
}
```

**Validation before returning:**
- Assert that every `NIC.Name` that appears in `EffectiveSecurityRules` map has a corresponding entry in `ResourceGraph.NetworkInterfaces`. Log a warning if unknown NIC names appear.
- Assert that `Fixture.Subscription` is non-empty.
- Log the counts: `INFO: fixture assembled; vnets={n} nsgs={n} nics={n} pips={n} effRules={n} effRoutes={n} adminRules={n} firewall={present|absent}`.

---

## 5. Dual-Field Resolution

### The problem

`SecRule` has two fields that both carry source address information:

```go
type SecRule struct {
    // ...
    SourceAddressPrefix  string `json:"sourceAddressPrefix"`  // used by engine
    Source               string `json:"source"`               // not used by engine
}
```

The engine in `analyze.go` reads **exclusively** `SourceAddressPrefix`:

```go
src := r.SourceAddressPrefix
broadNet, broadTag := isInternetSource(src), isBroadTagSource(src)
```

```go
src := strings.ToLower(ar.SourceAddressPrefix)
if src != "internet" && src != "0.0.0.0/0" && src != "*" { continue }
```

`Source` is never read by the engine.

### What `Source` appears to represent

Looking at the golden fixtures, `Source` is populated with values like `"subnet:web"` and `"subnet:backend"` — indicating which subnet's NSG contributed the rule. This is metadata about the rule's origin (which NSG association produced it), not the rule's network source predicate. It is consistent with Network Watcher's concept of rule provenance.

The ARM Network Watcher effective security rules response does not have a native `source` field with this provenance format — this appears to be a fixture-level annotation added during Phase 0 to document rule origins for debugging. It is not sourced from any Azure API.

### Canonical field: `SourceAddressPrefix`

**Rule for the adapter:** Always populate `SourceAddressPrefix` from the Azure API response field `sourceAddressPrefix` (NW effective rules response) or `properties.sourceAddressPrefix` (Resource Graph NSG rules). This is the field the engine reads.

**Rule for `Source`:** Set `Source` equal to `SourceAddressPrefix` for every `SecRule` the adapter creates. This ensures forward compatibility if a future engine version reads `Source`, and preserves the round-trip JSON representation from the fixtures.

```
// Adapter code pattern (pseudocode):
rule.SourceAddressPrefix = apiResponse.sourceAddressPrefix
rule.Source = rule.SourceAddressPrefix   // mirror, not independent data
```

Do **not** attempt to derive rule provenance (which NSG supplied the rule) for the `Source` field. The engine does not need it, and the computation is non-trivial (requires correlating rule names across multiple NSG resources).

---

## 6. RBAC Role Set

The Managed Identity running the adapter requires the following permissions. All roles are **read-only** — the adapter never writes to Azure.

### 6.1 Required roles

| Permission purpose | Minimum Azure role | Scope | Notes |
|---|---|---|---|
| Resource Graph queries (all entity types) | **Reader** (built-in) | Subscription | Grants `Microsoft.ResourceGraph/resources/read` implicitly. Resource Graph queries inherit the caller's ARM read permissions. |
| Network Watcher Effective Security Rules | Custom role with `Microsoft.Network/networkWatchers/effectiveNetworkSecurityGroups/action` | Subscription | Not included in Reader. See note below. |
| Network Watcher Effective Routes | Custom role with `Microsoft.Network/networkWatchers/effectiveRouteTable/action` | Subscription | Not included in Reader. See note below. |
| AVNM REST API read (Network Managers, Security Admin Configurations, Rule Collections, Rules) | **Reader** (built-in) | Subscription | `Microsoft.Network/networkManagers/read` is included in Reader. |
| AVNM Network Group static members read | **Reader** (built-in) | Subscription | `Microsoft.Network/networkManagers/networkGroups/staticMembers/read` included in Reader. |
| Azure Firewall read | **Reader** (built-in) | Subscription | `Microsoft.Network/azureFirewalls/read` included in Reader. |
| Firewall Policy read | **Reader** (built-in) | Subscription | `Microsoft.Network/firewallPolicies/read` included in Reader. |

### 6.2 Recommended role assignment

Assign **two** roles to the Managed Identity:

1. **Reader** (built-in `acdd72a7-3385-48ef-bd42-f606fba81ae7`) at subscription scope.
2. A **custom role** containing exactly the two NW data-plane actions:

```json
{
  "Name": "Network Topology Reviewer (NW Read-Only)",
  "Description": "Read-only access to Network Watcher data-plane APIs for topology analysis",
  "Actions": [
    "Microsoft.Network/networkWatchers/effectiveNetworkSecurityGroups/action",
    "Microsoft.Network/networkWatchers/effectiveRouteTable/action"
  ],
  "NotActions": [],
  "AssignableScopes": ["/subscriptions/{sub}"]
}
```

**Do not** assign **Network Contributor** (`4d97b98b-1d4f-4787-a291-c67834d212e7`) to the Managed Identity. Network Contributor grants write permissions (`Microsoft.Network/*/write`) which violate the read-only mandate established in Phase 0.

### 6.3 `[VERIFY]` items

| # | Item to verify | Why |
|---|---|---|
| V1 | The two `Microsoft.Network/networkWatchers/effective*` actions are sufficient for the target tenant — confirm they are not blocked by a deny assignment or Azure Policy. | Some enterprise tenants have deny assignments overriding Reader + custom roles. |
| V2 | Network Watcher exists in every region where NICs are deployed in the target subscription. | Absent NW → NIC degradation. |
| V3 | The `Microsoft.Network/networkManagers/securityAdminConfigurations/*/read` permission chain is included in the subscription's Reader role assignment. | Some tenants scope Reader to resource groups, not subscriptions. |
| V4 | AVNM rule collection Resource Graph type (`microsoft.network/networkmanagers/securityadminconfigurations/rulecollections/rules`) is available in the target tenant's Resource Graph index. | Resource Graph coverage for newer resource types can lag by days in some tenants. |
| V5 | Firewall Policy rule collection groups are returned in full by ARM GET (not truncated). | Some firewall policies have hundreds of rules; confirm ARM pagination is handled. |
| V6 | NW throttle limit in the target subscription. | The ~100 ops/5min limit is documented as a default; enterprise subscriptions may have a higher quota. Confirm with the subscription owner. |

---

## 7. Phase 2 Placeholders

The following fields are not required for Phase 1 analysis but are needed for `simulate_change` (topology what-if) and `forecast_cost` (Azure cost estimation) in Phase 2. They are documented here so the adapter can optionally capture them during Phase 1 data collection — avoiding a second full sweep of Azure APIs in Phase 2.

Each field is marked `// Phase 2` in the proposed struct extension.

---

### 7.1 VNet Gateway (new struct)

Required for: fixed monthly cost forecast (VPN Gateway or ExpressRoute Gateway SKU billing).

```go
// Phase 2
type VNetGateway struct {
    Name        string `json:"name"`         // ARM: name
    VNetName    string `json:"vnetName"`     // derived from subnet resource ID
    GatewayType string `json:"gatewayType"`  // ARM: properties.gatewayType — "Vpn" | "ExpressRoute"
    VpnType     string `json:"vpnType"`      // ARM: properties.vpnType — "RouteBased" | "PolicyBased"
    SKU         string `json:"sku"`          // ARM: properties.sku.name — "Basic" | "VpnGw1" | "VpnGw5AZ" | "ErGw1Az" | etc.
    ActiveActive bool   `json:"activeActive"` // ARM: properties.activeActive — doubles the gateway cost
}
```

Add `VNetGateways []VNetGateway` to `ResourceGraph`.

**Azure API source:** Resource Graph `microsoft.network/virtualnetworkgateways`, `properties.gatewayType`, `properties.vpnType`, `properties.sku.name`, `properties.activeActive`.

---

### 7.2 Private Endpoints (new struct)

Required for: Private Link cost estimation (per-endpoint fixed charge + data processing fee).

```go
// Phase 2
type PrivateEndpoint struct {
    Name               string   `json:"name"`               // ARM: name
    Subnet             string   `json:"subnet"`             // "{vnetName}/{subnetName}" — same convention as NIC.Subnet
    PrivateLinkService string   `json:"privateLinkService"` // ARM: properties.privateLinkServiceConnections[0].properties.privateLinkServiceId → extract resource type (e.g. "Microsoft.Storage/storageAccounts")
    GroupIDs           []string `json:"groupIds"`           // ARM: properties.privateLinkServiceConnections[0].properties.groupIds[] — e.g. ["blob", "file"]
}
```

Add `PrivateEndpoints []PrivateEndpoint` to `ResourceGraph`.

**Azure API source:** Resource Graph `microsoft.network/privateendpoints`, `properties.subnet.id`, `properties.privateLinkServiceConnections[0].properties.privateLinkServiceId`, `properties.privateLinkServiceConnections[0].properties.groupIds`.

---

### 7.3 VM SKU (for NIC bandwidth estimation)

Required for: variable cost estimation — NIC throughput and VM-level bandwidth caps affect data transfer cost modelling.

```go
// Phase 2
// Add to NIC struct:
VMSize string `json:"vmSize,omitempty"` // ARM: follow NIC.properties.virtualMachine.id → VM resource → properties.hardwareProfile.vmSize — e.g. "Standard_D4s_v5"
```

**Azure API source:** The NIC ARM resource has `properties.virtualMachine.id`. Follow this resource ID to the VM resource (`microsoft.compute/virtualmachines`) and read `properties.hardwareProfile.vmSize`. This requires a second Resource Graph join or a separate VM query. Recommended approach: add a Phase 2 VM query:

```kql
// Phase 2 query
Resources
| where subscriptionId == "{sub}"
| where type == "microsoft.compute/virtualmachines"
| project name, resourceGroup, vmSize = properties.hardwareProfile.vmSize,
    nicIds = properties.networkProfile.networkInterfaces[*].id
```

Join by NIC resource ID to populate `NIC.VMSize`.

---

### 7.4 Azure Firewall SKU Tier

Required for: firewall cost forecast (Basic ≈ $300/month, Standard ≈ $1,500/month, Premium ≈ $2,500/month, plus data processing).

```go
// Phase 2
// Add to Firewall struct:
SKUTier string `json:"skuTier,omitempty"` // ARM: properties.sku.tier — "Basic" | "Standard" | "Premium"
```

**Azure API source:** Already available in the §2.7 KQL query (`sku = properties.sku.tier`). This field is captured by the Phase 1 query but not mapped to the struct. No additional Azure call needed — the adapter just needs to populate this field in Phase 2.

---

### 7.5 Public IP Allocation Method and SKU

Required for: PIP cost forecast (Standard Static ≈ $3.65/month; Dynamic Basic ≈ free when attached; cross-region transfer differs by SKU).

```go
// Phase 2
// Add to PublicIP struct:
AllocationMethod string `json:"allocationMethod,omitempty"` // ARM: properties.publicIPAllocationMethod — "Static" | "Dynamic"
SKU              string `json:"sku,omitempty"`              // ARM: sku.name — "Basic" | "Standard"
```

**Azure API source:** Already available in the §2.4 KQL query (`allocationMethod`, `sku`). Captured by Phase 1 query, not yet mapped to struct.

---

### 7.6 VNet Peering Cross-Region Metadata

Required for: cross-region VNet peering data transfer cost (billed per GB, rate varies by source/destination region pair; global peering is more expensive than regional).

```go
// Phase 2
// Add to Peering struct:
RemoteVnetRegion string `json:"remoteVnetRegion,omitempty"` // resolved from remoteVirtualNetwork.id → VNet resource → location
IsGlobalPeering  bool   `json:"isGlobalPeering,omitempty"` // true when local VNet location != RemoteVnetRegion
```

**Azure API source:** The remote VNet's location is available from the VNet resources already collected in Step A (§2.1). Resolve `RemoteVnet` name → VNet record → `location`. Compute `IsGlobalPeering = (localVNetLocation != remoteVNetLocation)`. No additional Azure API calls needed — pure join on already-collected data.

---

### 7.7 Route Table BGP Propagation Flag

Required for: topology simulation (disabling BGP propagation affects route resolution and therefore the effective `0.0.0.0/0` next hop analysis in simulated topologies).

```go
// Phase 2
// Add to RouteTable struct:
DisableBgpRoutePropagation bool `json:"disableBgpRoutePropagation,omitempty"` // ARM: properties.disableBgpRoutePropagation
```

**Azure API source:** Already available in the §2.3 KQL query (`disableBgpRoutePropagation = properties.disableBgpRoutePropagation`). Captured by Phase 1 query, not yet mapped to struct.

---

*End of TOPOLOGY_MODEL.md*

---

## 8. PrivateDnsZone Structure

**ARM type:** `microsoft.network/privatednszones`
**Analysis rule:** `checkPrivateDnsZoneMisconfiguration` — fires when a `privatelink.*` zone exists but is not linked to every VNet that hosts NICs.

### 8.1 Field Mapping

| Field | Go type | ARM/API source | Config vs Truth | Example value | Required |
|---|---|---|---|---|---|
| `Name` | `string` | `name` | Declared | `"privatelink.blob.core.windows.net"` | Yes |
| `LinkedVnets` | `[]string` | `GET /privateDnsZones/{zone}/virtualNetworkLinks` → `properties.virtualNetwork.id` → VNet name | Evaluated (link must be provisioned) | `["spoke-a-vnet","hub-vnet"]` | Yes |
| `ARecords` | `[]DnsARecord` | `GET /privateDnsZones/{zone}/A` → `properties.aRecords[].ipv4Address` | Declared | `[{"name":"myaccount","ip":"10.1.2.5"}]` | Informational |

### 8.2 KQL Query

```kql
resources
| where type == "microsoft.network/privatednszones"
| project
    name,
    id,
    resourceGroup,
    subscriptionId
```

> **Note:** KQL returns zone metadata only. Virtual network links require a separate ARM list call per zone:
> `GET /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/privateDnsZones/{zone}/virtualNetworkLinks?api-version=2020-06-01`
>
> A-records require:
> `GET /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/privateDnsZones/{zone}/A?api-version=2020-06-01`
>
> Batch these in parallel using the same semaphore pool as NW calls (see §3).

---

## 9. ApplicationGateway Structure

**ARM type:** `microsoft.network/applicationgateways`
**Analysis rules:** `checkAppGatewayExposure` — WAF disabled (Medium) / WAF in Detection mode (Informational).

### 9.1 Field Mapping

| Field | Go type | ARM JSON path | Config vs Truth | Example value | Required |
|---|---|---|---|---|---|
| `Name` | `string` | `name` | Declared | `"appgw-prod"` | Yes |
| `Subnet` | `string` (vnet/subnet) | `properties.gatewayIPConfigurations[0].properties.subnet.id` → extract vnet+subnet names | Declared | `"hub-vnet/AppGatewaySubnet"` | Yes |
| `PublicIP` | `string` | `properties.frontendIPConfigurations[].properties.publicIPAddress.id` → resolve PIP `ipAddress` | Evaluated | `"20.10.10.1"` | Yes |
| `WafEnabled` | `bool` | `properties.webApplicationFirewallConfiguration.enabled` OR `properties.sku.tier == "WAF_v2"` | Declared | `false` | Yes |
| `WafMode` | `string` | `properties.webApplicationFirewallConfiguration.firewallMode` | Declared | `"Prevention"` | Yes |
| `BackendPools` | `[]AppGWBackendPool` | `properties.backendAddressPools[].properties.backendAddresses[].ipAddress` | Declared | *(see struct)* | No (Phase-2 reachability) |

### 9.2 KQL Query

```kql
resources
| where type == "microsoft.network/applicationgateways"
| extend wafc = properties.webApplicationFirewallConfiguration
| extend sku  = properties.sku
| project
    name,
    resourceGroup,
    subscriptionId,
    gatewaySubnetRef   = tostring(properties.gatewayIPConfigurations[0].properties.subnet.id),
    wafEnabled         = tobool(coalesce(wafc.enabled, todynamic('false'))),
    wafMode            = tostring(wafc.firewallMode),
    skuTier            = tostring(sku.tier),
    frontendIPs        = properties.frontendIPConfigurations,
    backendPools       = properties.backendAddressPools
```

> **Important:** WAF_v2 SKU tier with no `webApplicationFirewallConfiguration` means WAF is linked via an external WAF Policy resource (`properties.firewallPolicy`). A separate ARM GET on the policy is needed to determine `policySettings.state` (`Enabled`/`Disabled`) and `policySettings.mode` (`Prevention`/`Detection`). [VERIFY] in live environment.

---

## 10. AKSCluster Structure

**ARM type:** `microsoft.containerservice/managedclusters`
**Analysis rule:** `checkAKSExposure` — non-private cluster (Medium).

### 10.1 Field Mapping

| Field | Go type | ARM JSON path | Config vs Truth | Example value | Required |
|---|---|---|---|---|---|
| `Name` | `string` | `name` | Declared | `"aks-prod"` | Yes |
| `Subnet` | `string` (vnet/subnet) | `properties.agentPoolProfiles[0].vnetSubnetID` → extract vnet+subnet names | Declared | `"spoke-a-vnet/aks-nodes"` | Yes |
| `PodCidr` | `string` | `properties.networkProfile.podCidr` | Declared | `"192.168.0.0/16"` | Phase-2 (CIDR conflict) |
| `ServiceCidr` | `string` | `properties.networkProfile.serviceCidr` | Declared | `"172.16.0.0/16"` | Phase-2 |
| `IsPrivateCluster` | `bool` | `properties.apiServerAccessProfile.enablePrivateCluster` | Declared | `false` | Yes |
| `ApiServerIP` | `string` | `properties.privateFQDN` or NIC lookup if private | Evaluated | `"10.1.2.100"` | Phase-2 |

### 10.2 KQL Query

```kql
resources
| where type == "microsoft.containerservice/managedclusters"
| extend apiProfile = properties.apiServerAccessProfile
| extend netProfile = properties.networkProfile
| project
    name,
    resourceGroup,
    subscriptionId,
    isPrivateCluster   = tobool(coalesce(apiProfile.enablePrivateCluster, todynamic('false'))),
    privateFqdn        = tostring(properties.privateFQDN),
    podCidr            = tostring(netProfile.podCidr),
    serviceCidr        = tostring(netProfile.serviceCidr),
    agentPools         = properties.agentPoolProfiles
```

> **Subnet extraction:** `agentPools[].vnetSubnetID` is a full ARM resource ID (`/subscriptions/.../subnets/{name}`). Parse the VNet name and subnet name from the path segments. All node pools may use different subnets — Phase 1 captures only the first system pool; Phase 2 should capture all pools.

---

## 11. NatGateway Structure

**ARM type:** `microsoft.network/natgateways`
**Analysis note:** NAT Gateway provides deterministic outbound internet for subnets. This does not affect inbound reachability (Gates 1–4 are inbound-focused), but changes the effective outbound path. Gate 3 (`defaultHop == "Internet"`) remains correct: NAT GW uses `Internet` next-hop type. No new analysis rule required in Phase 1; collected for completeness and Phase-2 egress analysis.

### 11.1 Field Mapping

| Field | Go type | ARM JSON path | Config vs Truth | Example value | Required |
|---|---|---|---|---|---|
| `Name` | `string` | `name` | Declared | `"nat-gw-spoke-a"` | Yes |
| `PublicIPs` | `[]string` | `properties.publicIpAddresses[].id` → resolve PIP `ipAddress` | Evaluated | `["20.0.0.1"]` | Yes |
| `AssociatedSubnets` | `[]string` | `properties.subnets[].id` → extract vnet/subnet names | Declared | `["spoke-a-vnet/workload"]` | Yes |

### 11.2 KQL Query

```kql
resources
| where type == "microsoft.network/natgateways"
| project
    name,
    resourceGroup,
    subscriptionId,
    publicIpIds       = properties.publicIpAddresses,
    associatedSubnets = properties.subnets
```

---

## 12. PrivateLinkService Structure

**ARM type:** `microsoft.network/privatelinkservices`
**Analysis note:** A Private Link Service (PLS) is the provider-side construct (e.g., Bastion NVA PLS in the BCLM topology). It enables cross-tenant or cross-subscription access via a Private Endpoint on the consumer side. No Phase-1 analysis rule exists; collected for Phase-2 exposure mapping.

### 12.1 Field Mapping

| Field | Go type | ARM JSON path | Config vs Truth | Example value | Required |
|---|---|---|---|---|---|
| `Name` | `string` | `name` | Declared | `"pls-bastion-nva"` | Yes |
| `Subnet` | `string` (vnet/subnet) | `properties.ipConfigurations[0].properties.subnet.id` → extract vnet/subnet | Declared | `"hub-vnet/pls-subnet"` | Yes |
| `NatIPConfig` | `string` | `properties.ipConfigurations[0].properties.privateIPAddress` | Evaluated | `"10.0.5.4"` | Phase-2 |
| `LinkedPrivateEndpoints` | `[]string` | `properties.privateEndpointConnections[].id` → PE names | Evaluated | `["pe-consumer-tenant"]` | Phase-2 |

### 12.2 KQL Query

```kql
resources
| where type == "microsoft.network/privatelinkservices"
| project
    name,
    resourceGroup,
    subscriptionId,
    ipConfigs       = properties.ipConfigurations,
    peConnections   = properties.privateEndpointConnections
```

---

## 13. ExpressRouteCircuit Structure

**ARM type:** `microsoft.network/expressroutecircuits`
**Analysis note:** When `BGPAdvertisesDefaultRoute=true`, on-premises BGP advertises `0.0.0.0/0` into the connected VNet via the Virtual Network Gateway. This **overrides UDRs** on spoke VNets using `UseRemoteGateways=true` — Gate 3 (route `0.0.0.0/0→Internet`) may be invalidated because the effective route is `0.0.0.0/0→VirtualNetworkGateway` (to on-prem). This is a Phase-2 concern; Phase-1 collects the field.

### 13.1 Field Mapping

| Field | Go type | ARM JSON path | Config vs Truth | Example value | Required |
|---|---|---|---|---|---|
| `Name` | `string` | `name` | Declared | `"er-circuit-bclm"` | Yes |
| `PeeringLocation` | `string` | `properties.peeringLocation` | Declared | `"Dallas"` | Informational |
| `BandwidthMbps` | `int` | `properties.bandwidthInMbps` | Declared | `1000` | Phase-2 (cost) |
| `ConnectedVnet` | `string` | Via `microsoft.network/virtualnetworkgateways` → `expressRouteCircuit.id` join | Declared | `"hub-vnet"` | Phase-2 |
| `BGPAdvertisesDefaultRoute` | `bool` | Requires BGP summary from on-premises router or MSEE — **not available in Resource Graph** [VERIFY] | Evaluated | `false` | Phase-2 (Gate-3 override) |

### 13.2 KQL Query

```kql
resources
| where type == "microsoft.network/expressroutecircuits"
| project
    name,
    resourceGroup,
    subscriptionId,
    peeringLocation  = tostring(properties.peeringLocation),
    bandwidthMbps    = toint(properties.bandwidthInMbps),
    serviceState     = tostring(properties.serviceProviderProvisioningState)
```

> **BGP default route:** `BGPAdvertisesDefaultRoute` cannot be determined from Resource Graph. It requires querying BGP learned routes from the Virtual Network Gateway via `POST /virtualNetworkGateways/{gwName}/getBgpPeerStatus` or reading on-premises router configuration. Mark as `[VERIFY]` and default to `false` in Phase 1. **Phase 2:** add a NW BGP routes call to determine this at runtime.

---

## 14. Multi-Subscription Support

### 14.1 Current model

`Fixture.Subscription` holds the subscription ID for the scope of one adapter run. All Resource Graph KQL queries in §2 are subscription-scoped. This is correct for Phase 1 (single-subscription target).

### 14.2 Cross-subscription peerings

The `Peering.RemoteSubscriptionID` field (added in rework) captures cross-subscription peering links. When a VNet's peering `remoteVirtualNetwork.id` references a subscription ID different from the current fixture's subscription, the adapter MUST:

1. Set `Peering.RemoteSubscriptionID` to the remote subscription ID.
2. If the adapter has Reader access to the remote subscription: collect the remote VNet in a second Resource Graph query and populate `Fixture.CrossSubscriptionPeerings[]`.
3. If the adapter does NOT have access to the remote subscription: set `CrossSubPeering.State = "Connected"` but leave `HasHubFirewall = false` with a warning in the analysis output.

**Assembly step for cross-subscription peerings:**

```
Step G (cross-subscription, conditional):
  For each peering where RemoteSubscriptionID != current subscription:
    1. Query remote subscription: GET /subscriptions/{remoteSubId}/resourceGroups/*/providers/Microsoft.Network/virtualNetworks/{remoteVnet}
    2. Check if a hub firewall exists in either VNet's resource group or peering path
    3. Populate CrossSubscriptionPeerings[]:
       {localVnet, remoteVnet, remoteSubscriptionId, state, allowForwardedTraffic, hasHubFirewall}
```

### 14.3 Multi-subscription scope (BCLM full estate)

For the BCLM 6-subscription topology, the MCP `analyze_risks` tool should accept a `subscriptionIds []string` parameter. The adapter runs once per subscription producing one Fixture per subscription. The MCP server:

1. Runs `Analyze()` per fixture for intra-subscription findings.
2. Joins cross-subscription peerings across fixtures for inter-subscription findings.
3. Returns a merged finding list with a `subscriptionId` field on each finding.

This is a **Phase-2 MCP server concern** — the engine's `Analyze()` function remains single-fixture and deterministic.

### 14.4 RBAC for cross-subscription access

| Scope | Role | Purpose |
|---|---|---|
| Each target subscription | `Reader` (built-in) | Resource Graph queries |
| Each target subscription | Custom NW read role | NW Effective Rules/Routes |
| Remote subscription (peering target) | `Network Reader` or `Reader` | Cross-subscription peering metadata only |

> [VERIFY] Whether `Reader` on the remote subscription is sufficient to resolve `remoteVirtualNetwork.id` without a full network read permission.


---

## 15. Additional Network Watcher APIs — Adapter Guidance

> Based on official Microsoft documentation: [Network Watcher Overview](https://learn.microsoft.com/en-us/azure/network-watcher/network-watcher-monitoring-overview)

The Phase 1 adapter uses two NW data-plane APIs (§3). Three additional NW APIs
are available that materially improve analysis quality. This section documents
how each should be used by the adapter.

### 15.1 Next Hop API

**Endpoint:** `POST /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/networkWatchers/{nw}/nextHop`

**What it does:** For a specific source IP → destination IP pair on a NIC, returns:
- The actual next-hop IP
- The next-hop type (`Internet`, `VirtualAppliance`, `VnetLocal`, `VirtualNetworkGateway`, `None`)
- The ID of the route table entry that made the decision

**Difference from Effective Routes:** Effective Routes show all route table entries. Next Hop computes the *actual winning route* for a specific destination — including BGP-learned routes that may not be visible in the static effective route table.

**Phase 1 adapter usage:** Use as a **spot-check validator** for Gate 3. After populating `EffectiveRoutes`, call Next Hop for `dst=8.8.8.8` (internet) on NICs where Gate 3 status is ambiguous (e.g., multiple overlapping routes, BGP routes present). If Next Hop returns `Internet` but EffectiveRoutes shows `VirtualNetworkGateway` → BGP default route has taken over; update the effective route entry accordingly.

**Phase 2 adapter usage:** East-West path analysis — call Next Hop with specific spoke VNet CIDR as destination to determine if traffic flows through the hub firewall or directly between spokes.

### 15.2 IP Flow Verify API

**Endpoint:** `POST /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/networkWatchers/{nw}/ipFlowVerify`

**What it does:** For a specific 5-tuple (direction, protocol, local IP:port, remote IP:port), returns:
- `Access`: `Allow` or `Deny`
- The name of the NSG rule that made the decision
- The NSG resource that contains the decisive rule

**Why this matters:** This is the **authoritative Azure answer** to Gates 2+3 — it asks Azure's control plane directly rather than simulating NSG evaluation. It correctly handles:
- NSG rule priority resolution
- Augmented security rules (default rules)
- Service tag expansion at evaluation time
- Multiple NSG associations (subnet + NIC)

**Phase 1 adapter usage:** Use as an **eval-time validator** only — NOT as a replacement for the engine's simulation. Rationale:
1. IP Flow Verify is O(flows) — expensive for large subscriptions
2. It does NOT account for AVNM Security Admin Rules (Gate 1) — only NSG + routing
3. Our simulation adds AVNM, AzureCloud cross-tenant scope, and multi-value expansion that IP Flow Verify doesn't model

**Recommended pattern:** In the eval fixture generation step (Step 1.6), use IP Flow Verify to produce ground-truth Gate 2+3 verdicts for the golden fixtures. Compare against engine output to measure precision.

**Cost:** ~100 NW data-plane calls per 5 minutes per subscription (shared with ESR + ER calls). Use the existing semaphore-10 pool. Priority: after ESR and ER calls complete.

### 15.3 NW Topology API

**Endpoint:** `GET /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/networkWatchers/{nw}/topology`

**What it does:** Returns a JSON graph of all network resources in the specified resource group and their `contains`/`isConnectedTo`/`isAssociatedWith` relationships (VMs → NICs → Subnets → NSGs → VNets, etc.).

**Phase 1 adapter usage:** Optional discovery shortcut. Use to verify the Resource Graph query results are complete — cross-reference NIC count from Resource Graph vs. NW Topology. Catches resources in resource groups not in the Resource Graph scope.

**Limitation:** Scoped per resource group, not per subscription. A subscription with 50 resource groups would require 50 Topology API calls. Use Resource Graph as the primary source.

---

## 16. Model Lock Declaration

> **Status as of 2026-06-12: MODEL LOCKED FOR PHASE 1 ADAPTER IMPLEMENTATION**

The `graph.Fixture` schema is stable. All Phase 1 adapter work (Step 1.3) must target
this schema exactly. Schema changes after Step 1.3 begins require a formal ADR.

### Complete resource coverage (as of model lock)

**P0 — Fully modeled with analysis rules:**
VNet · Subnet (with ServiceEndpoints/Delegations/PENPolicies) · NSG · RouteTable · NIC · PublicIP ·
VNet Peering · Cross-Subscription Peering · AVNM Security Admin Rules · Azure Firewall (classic + policy) ·
Private Endpoint · Private DNS Zone · Load Balancer (ELB+ILB) · Application Gateway (WAF) ·
AKS · APIM · Azure Bastion · Virtual WAN (vHub) · NW Effective Security Rules · NW Effective Routes

**P0 — Collected, Phase-2 analysis rules:**
ExpressRoute Circuit · VirtualNetworkGateway (VPN+ER) · NAT Gateway · Private Link Service ·
DNS Private Resolver · Azure Route Server · Azure Front Door · DDoS Protection Plan · Local Network Gateway

**P1 — Enrichment envelope (optional, no engine rule):**
`Fixture.Enrichment.DefenderAssessments` · `PolicyFindings` · `RecentChanges` (Activity Log)

### Analysis rules (12 active, deterministic, golden-tested)

| Rule | Severity | Test |
|---|---|---|
| 4-gate internet reachability (AVNM→NSG→Route→PIP) | High/Critical | F1,F3,H1,H2 |
| Orphaned Public IP | Low | F1 |
| CIDR overlap | Medium | F3 |
| Missing tier segmentation | High | F2 |
| Azure Firewall DNAT reachability | High | H1 |
| PE DNS zone not linked | High | F6 |
| App Gateway WAF disabled/Detection | Medium/Info | F7 |
| AKS non-private cluster | Medium | F8 |
| Cross-subscription peering without firewall | Medium | F8 |
| ELB inbound NAT port forwarding | High | F10 |
| APIM External/None without WAF | Medium | F11 |
| Bastion bypass — direct management port | High | F12 |
| vWAN hub unsecured / private traffic bypass | Medium | F13 |
| Front Door WAF disabled / detection mode | Medium/Info | F14 |

**Golden fixture corpus: 13/13 tests — 100% pass rate**

