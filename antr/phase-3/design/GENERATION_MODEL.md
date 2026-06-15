# Topology Generation Model — Phase 3

**Document:** `phase-3/design/GENERATION_MODEL.md`
**Date:** 2026-06-13
**Status:** REVIEWED — Phase 3 design. Phase 2 design: `phase-2/design/SIMULATION_MODEL.md` (Step 2.1 complete).
**Scope:** Defines the end-to-end topology generation pipeline — from architect natural-language intent to
validated Terraform emitted as a GitHub PR. The `Analyze()` function that detects risks in deployed
topologies is the identical gate that blocks generation of insecure topologies. No LLM output escapes
validation.

**Upstream references:**
- `engine/go/internal/graph/model.go` — `graph.Fixture` (32-struct topology model)
- `engine/go/internal/analyze/analyze.go` — `Analyze(*graph.Fixture) []Finding` (13 deterministic rules)
- `engine/go/simulator/delta.go` — `TopologyDelta`, `ApplyDelta` (Phase 2 simulator)
- `phase-2/design/SIMULATION_MODEL.md` — SR-002, SR-003 limitations being closed here
- `phase-1/design/TOPOLOGY_MODEL.md` — Phase 1 deferred items (east-west, DNS, AVNM delta)

---

## Contents

1. [TopologySpec Schema](#1-topologyspec-schema)
2. [Module Registry](#2-module-registry)
3. [Renderer Contract](#3-renderer-contract)
4. [Validation Gate](#4-validation-gate)
5. [PR Workflow](#5-pr-workflow)
6. [Non-Negotiable Guardrails](#6-non-negotiable-guardrails)
7. [Phase 3 Placeholder Reconciliation](#7-phase-3-placeholder-reconciliation)
8. [`generate_topology` MCP Tool Contract](#8-generate_topology-mcp-tool-contract)
9. [Rubber-Duck Review Findings](#9-rubber-duck-review-findings)

---

## 1. `TopologySpec` Schema

**Purpose:** The structured intermediate representation the LLM produces from architect intent.
`TopologySpec` sits between natural language and Terraform. The LLM reasons about *intent*; the renderer
translates intent to Terraform. The LLM never touches raw NSG rule structs, route table entries, or
firewall rules.

### 1.1 Design principles

| Principle | Rationale |
|---|---|
| Intent, not configuration | `nsg_intent` expresses desired access patterns; the renderer translates to `azurerm_network_security_rule` blocks |
| Expressible as JSON Schema | Required for LLM structured output / tool-call response schema (AskAT&T function calling) |
| Unambiguous tier labels | `tier_label` on subnets drives NSG intent expansion and `sensitive: true` propagation |
| No raw security rules in output | Hard guardrail — see §6.1. The LLM never produces a `priority` / `access` / `direction` tuple |
| Flat, round-trippable | `TopologySpec` must survive JSON serialisation and hashing without loss; no function pointers or interfaces |

### 1.2 Go type definition

```go
// Package generator is the Phase 3 topology generation pipeline.
// TopologySpec is the structured output of the LLM step and the input to
// RenderTerraform. It intentionally omits all raw security-rule details —
// those are the renderer's domain, driven by the NSGIntent vocabulary (§3.3).
package generator

// TopologySpec is the intermediate representation between architect intent (NL)
// and Terraform. Produced by the LLM (AskAT&T), consumed by RenderTerraform.
type TopologySpec struct {
    // SpecVersion allows future schema evolution without breaking cached specs.
    // Current version: "1.0".
    SpecVersion string `json:"specVersion"`

    // Description is the architect's original intent, reproduced verbatim.
    // Stored in audit trail and PR body. Never used for generation logic.
    Description string `json:"description"`

    // Region is the primary Azure region for all resources (e.g. "eastus2").
    Region string `json:"region"`

    // VNets is the ordered list of virtual networks to create.
    // Hub VNet (if present) must be first by convention.
    VNets []VNetSpec `json:"vnets"`

    // PeeringTopology describes the overall peering pattern.
    // "hub-spoke"  — one hub VNet peers to N spoke VNets (UseRemoteGateways on spokes)
    // "mesh"       — every VNet peers to every other VNet
    // "none"       — no VNet peering
    // "custom"     — explicit pairs in PeeringPairs (hub-spoke and mesh are synthetic sugar)
    PeeringTopology string `json:"peeringTopology"`

    // PeeringPairs is populated only when PeeringTopology == "custom".
    // Each entry specifies a directed peering (from Local to Remote).
    PeeringPairs []PeeringPairSpec `json:"peeringPairs,omitempty"`

    // HubVNetName identifies which VNet is the hub for hub-spoke topologies.
    // Must reference a name in VNets. Empty when PeeringTopology != "hub-spoke".
    HubVNetName string `json:"hubVnetName,omitempty"`

    // GatewayType is the connectivity gateway to provision in the hub VNet.
    // "vpn"         — Azure VPN Gateway (VirtualNetworkGateway, VpnGateway SKU)
    // "expressroute" — ExpressRoute Gateway
    // "none"        — no gateway
    GatewayType string `json:"gatewayType"`

    // FirewallEnabled indicates whether Azure Firewall should be provisioned in
    // the hub VNet. When true, the renderer emits the AzureFirewall + Policy
    // resources and forces a 0.0.0.0/0 → VirtualAppliance UDR on spoke subnets.
    FirewallEnabled bool `json:"firewallEnabled"`

    // AVNMEnabled indicates whether the subscription already has an Azure Virtual
    // Network Manager instance whose admin rules must be respected. When true, the
    // renderer does NOT emit AVNM resources (they already exist); it emits a data
    // source reference only. When false, no AVNM resources are emitted.
    AVNMEnabled bool `json:"avnmEnabled"`

    // AVNMNetworkGroupID is the existing AVNM Network Group ID to reference when
    // AVNMEnabled == true. [VERIFY] AT&T AVNM Network Group naming convention.
    AVNMNetworkGroupID string `json:"avnmNetworkGroupId,omitempty"`

    // TierLabels is the ordered list of network tiers present in this topology.
    // Standard AT&T labels: "dmz", "web", "app", "data", "mgmt", "shared".
    // Used by the renderer to derive subnet names and NSG intent defaults.
    TierLabels []string `json:"tierLabels"`

    // Tags is the set of resource tags to apply to all generated resources.
    // AT&T mandates at minimum: "env", "owner", "costcenter", "appid".
    // [VERIFY] AT&T mandatory tag policy — confirm required keys.
    Tags map[string]string `json:"tags"`
}

// VNetSpec describes one virtual network to generate.
type VNetSpec struct {
    // Name is the VNet resource name. Must be globally unique within the spec.
    Name string `json:"name"`

    // AddressSpace is the list of CIDR blocks for this VNet.
    // The renderer validates that no two VNets in the spec overlap when
    // PeeringTopology != "none" (peered VNets must not have overlapping CIDRs).
    AddressSpace []string `json:"addressSpace"`

    // Subnets is the ordered list of subnets within this VNet.
    Subnets []SubnetSpec `json:"subnets"`

    // IsHub marks this VNet as the hub in a hub-spoke topology.
    // Must be consistent with TopologySpec.HubVNetName.
    IsHub bool `json:"isHub,omitempty"`
}

// SubnetSpec describes one subnet to generate.
type SubnetSpec struct {
    // Name is the subnet resource name within its parent VNet.
    Name string `json:"name"`

    // AddressPrefix is the CIDR for this subnet (e.g. "10.1.1.0/24").
    AddressPrefix string `json:"addressPrefix"`

    // TierLabel is the logical tier this subnet belongs to.
    // Must be one of the values in TopologySpec.TierLabels.
    // The renderer uses this to look up default NSG intents and routing.
    TierLabel string `json:"tierLabel"`

    // Sensitive marks this subnet as containing sensitive workloads.
    // Maps to graph.NIC.Tags["sensitive"] = "true" in the fixture projection.
    // When true: ValidateBeforeEmit will apply Critical severity to internet-
    // reachable NICs in this subnet (engine rule: sensitive=true + Gate 2).
    Sensitive bool `json:"sensitive"`

    // NSGIntents is the list of named access intents for this subnet's NSG.
    // These are vocabulary items from the intent table (§3.3).
    // The renderer translates each intent to one or more azurerm_network_security_rule blocks.
    // Intents outside the approved vocabulary are rejected, not guessed.
    NSGIntents []string `json:"nsgIntents"`

    // RouteToFirewall forces a 0.0.0.0/0 → VirtualAppliance UDR pointing at the
    // hub firewall private IP for this subnet. Requires FirewallEnabled == true.
    // Automatically set to true for all spoke subnets when FirewallEnabled == true.
    RouteToFirewall bool `json:"routeToFirewall,omitempty"`

    // ServiceEndpoints is the list of Azure service endpoints to enable.
    // e.g. ["Microsoft.Storage", "Microsoft.KeyVault"]
    ServiceEndpoints []string `json:"serviceEndpoints,omitempty"`

    // Delegations is the list of service delegations.
    // e.g. ["Microsoft.Sql/managedInstances"]
    Delegations []string `json:"delegations,omitempty"`

    // PrivateEndpointSubnet marks this subnet as the dedicated private endpoint
    // subnet. Sets PrivateEndpointNetworkPolicies = "Disabled" in the rendered HCL.
    PrivateEndpointSubnet bool `json:"privateEndpointSubnet,omitempty"`

    // PrivateEndpoints declares the private endpoints that terminate in this subnet.
    // This is required for fixture projection correctness: Analyze() reads
    // ResourceGraph.PrivateEndpoints and PrivateDnsZones for the private DNS gate.
    // Empty = this subnet hosts no private endpoints.
    PrivateEndpoints []PrivateEndpointSpec `json:"privateEndpoints,omitempty"`
}

// PrivateEndpointSpec describes one generated private endpoint.
type PrivateEndpointSpec struct {
    // Name is the private endpoint resource name.
    Name string `json:"name"`

    // GroupID is the Azure sub-resource groupId used by the engine's
    // peGroupIdToZone mapping (e.g. "blob", "vault", "sql", "openai").
    GroupID string `json:"groupId"`

    // ServiceResourceID is the ARM resource ID of the target service.
    ServiceResourceID string `json:"serviceResourceId"`
}

// PeeringPairSpec is used only when PeeringTopology == "custom".
type PeeringPairSpec struct {
    LocalVNet             string `json:"localVnet"`
    RemoteVNet            string `json:"remoteVnet"`
    AllowForwardedTraffic bool   `json:"allowForwardedTraffic"`
    UseRemoteGateways     bool   `json:"useRemoteGateways"`
    AllowGatewayTransit   bool   `json:"allowGatewayTransit"`
}
```

### 1.3 JSON Schema (for AskAT&T structured output)

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "urn:att:nettopo:topology-spec:1.0",
  "title": "TopologySpec",
  "type": "object",
  "required": ["specVersion", "description", "region", "vnets",
               "peeringTopology", "gatewayType", "firewallEnabled",
               "avnmEnabled", "tierLabels", "tags"],
  "additionalProperties": false,
  "properties": {
    "specVersion": { "type": "string", "const": "1.0" },
    "description": { "type": "string", "minLength": 10 },
    "region": {
      "type": "string",
      "description": "Primary Azure region slug, e.g. eastus2, westus3"
    },
    "vnets": {
      "type": "array",
      "minItems": 1,
      "maxItems": 20,
      "items": { "$ref": "#/$defs/VNetSpec" }
    },
    "peeringTopology": {
      "type": "string",
      "enum": ["hub-spoke", "mesh", "none", "custom"]
    },
    "peeringPairs": {
      "type": "array",
      "items": { "$ref": "#/$defs/PeeringPairSpec" }
    },
    "hubVnetName": { "type": "string" },
    "gatewayType": {
      "type": "string",
      "enum": ["vpn", "expressroute", "none"]
    },
    "firewallEnabled": { "type": "boolean" },
    "avnmEnabled": { "type": "boolean" },
    "avnmNetworkGroupId": { "type": "string" },
    "tierLabels": {
      "type": "array",
      "items": {
        "type": "string",
        "enum": ["dmz", "web", "app", "data", "mgmt", "shared", "gateway",
                 "bastion", "pe", "aks", "appgw"]
      }
    },
    "tags": {
      "type": "object",
      "required": ["env", "owner", "costcenter", "appid"],
      "additionalProperties": { "type": "string" }
    }
  },
  "$defs": {
    "VNetSpec": {
      "type": "object",
      "required": ["name", "addressSpace", "subnets"],
      "additionalProperties": false,
      "properties": {
        "name": { "type": "string", "minLength": 1 },
        "addressSpace": {
          "type": "array",
          "items": { "type": "string", "pattern": "^\\d+\\.\\d+\\.\\d+\\.\\d+/\\d+$" }
        },
        "subnets": {
          "type": "array",
          "minItems": 1,
          "items": { "$ref": "#/$defs/SubnetSpec" }
        },
        "isHub": { "type": "boolean" }
      }
    },
    "SubnetSpec": {
      "type": "object",
      "required": ["name", "addressPrefix", "tierLabel", "sensitive", "nsgIntents"],
      "additionalProperties": false,
      "properties": {
        "name": { "type": "string" },
        "addressPrefix": { "type": "string", "pattern": "^\\d+\\.\\d+\\.\\d+\\.\\d+/\\d+$" },
        "tierLabel": { "type": "string" },
        "sensitive": { "type": "boolean" },
        "nsgIntents": {
          "type": "array",
          "items": { "type": "string" }
        },
        "routeToFirewall": { "type": "boolean" },
        "serviceEndpoints": { "type": "array", "items": { "type": "string" } },
        "delegations": { "type": "array", "items": { "type": "string" } },
        "privateEndpointSubnet": { "type": "boolean" },
        "privateEndpoints": {
          "type": "array",
          "items": { "$ref": "#/$defs/PrivateEndpointSpec" }
        }
      }
    },
    "PrivateEndpointSpec": {
      "type": "object",
      "required": ["name", "groupId", "serviceResourceId"],
      "additionalProperties": false,
      "properties": {
        "name": { "type": "string", "minLength": 1 },
        "groupId": { "type": "string", "minLength": 1 },
        "serviceResourceId": { "type": "string", "minLength": 1 }
      }
    },
    "PeeringPairSpec": {
      "type": "object",
      "required": ["localVnet", "remoteVnet"],
      "additionalProperties": false,
      "properties": {
        "localVnet": { "type": "string" },
        "remoteVnet": { "type": "string" },
        "allowForwardedTraffic": { "type": "boolean" },
        "useRemoteGateways": { "type": "boolean" },
        "allowGatewayTransit": { "type": "boolean" }
      }
    }
  }
}
```

### 1.4 LLM system prompt constraint (non-negotiable)

The AskAT&T system prompt for topology generation MUST include the following verbatim constraint block.
This is enforced by the `generate_topology` tool handler; any spec that contains raw priority/access/direction
tuples at the top level is rejected before reaching the renderer.

```
CONSTRAINT — SECURITY RULES:
You MUST NOT produce raw NSG security rule objects with fields: priority, access, direction,
destinationPortRange, sourceAddressPrefix. Instead, express desired access as named intents
from the approved vocabulary in the nsgIntents array of each subnet. The renderer translates
intents to rules. Intents outside the approved vocabulary will be rejected.

CONSTRAINT — RESOURCE IDENTIFIERS:
Do not invent Terraform resource IDs, module source paths, or version strings.
The renderer selects modules from the AT&T-approved registry. Your output is TopologySpec JSON only.
```

### 1.5 Worked example — AT&T 3-tier hub-spoke with sensitive DB tier

**Intent:** "Create an AT&T hub-spoke network for the BCLM payment processing platform in East US 2.
Hub with Azure Firewall. Three spokes: web (public-facing), app (internal APIs), data (sensitive,
PostgreSQL Flexible Server). VPN gateway in hub for on-premises connectivity. Tag with env=prod,
owner=network-team, costcenter=BCLM-NET, appid=PAY-001."

**Resulting `TopologySpec` JSON:**

```json
{
  "specVersion": "1.0",
  "description": "AT&T BCLM payment processing platform — hub-spoke, East US 2, Azure Firewall, VPN gateway, 3 spoke tiers (web/app/data), data tier sensitive.",
  "region": "eastus2",
  "peeringTopology": "hub-spoke",
  "hubVnetName": "hub-vnet-bclm-prod",
  "gatewayType": "vpn",
  "firewallEnabled": true,
  "avnmEnabled": true,
  "avnmNetworkGroupId": "/subscriptions/XXXX/resourceGroups/rg-avnm/providers/Microsoft.Network/networkManagers/nm-att-prod/networkGroups/ng-bclm-prod",
  "tierLabels": ["gateway", "mgmt", "web", "app", "data", "pe"],
  "tags": {
    "env": "prod",
    "owner": "network-team",
    "costcenter": "BCLM-NET",
    "appid": "PAY-001"
  },
  "vnets": [
    {
      "name": "hub-vnet-bclm-prod",
      "addressSpace": ["10.0.0.0/16"],
      "isHub": true,
      "subnets": [
        {
          "name": "AzureFirewallSubnet",
          "addressPrefix": "10.0.0.0/26",
          "tierLabel": "mgmt",
          "sensitive": false,
          "nsgIntents": []
        },
        {
          "name": "GatewaySubnet",
          "addressPrefix": "10.0.1.0/27",
          "tierLabel": "gateway",
          "sensitive": false,
          "nsgIntents": []
        },
        {
          "name": "AzureBastionSubnet",
          "addressPrefix": "10.0.2.0/26",
          "tierLabel": "mgmt",
          "sensitive": false,
          "nsgIntents": ["allow-https-from-internet", "deny-all-inbound-other"]
        },
        {
          "name": "snet-mgmt",
          "addressPrefix": "10.0.3.0/24",
          "tierLabel": "mgmt",
          "sensitive": false,
          "nsgIntents": ["allow-bastion-rdp-ssh", "deny-internet-inbound"],
          "routeToFirewall": true
        }
      ]
    },
    {
      "name": "spoke-vnet-web-prod",
      "addressSpace": ["10.1.0.0/16"],
      "subnets": [
        {
          "name": "snet-web",
          "addressPrefix": "10.1.1.0/24",
          "tierLabel": "web",
          "sensitive": false,
          "nsgIntents": ["allow-https-from-internet", "allow-http-from-internet", "deny-all-inbound-other"],
          "routeToFirewall": true
        },
        {
          "name": "snet-appgw",
          "addressPrefix": "10.1.2.0/24",
          "tierLabel": "appgw",
          "sensitive": false,
          "nsgIntents": ["allow-appgw-management", "allow-https-from-internet"]
        }
      ]
    },
    {
      "name": "spoke-vnet-app-prod",
      "addressSpace": ["10.2.0.0/16"],
      "subnets": [
        {
          "name": "snet-app",
          "addressPrefix": "10.2.1.0/24",
          "tierLabel": "app",
          "sensitive": false,
          "nsgIntents": ["allow-internal-vnet", "deny-internet-inbound"],
          "routeToFirewall": true
        }
      ]
    },
    {
      "name": "spoke-vnet-data-prod",
      "addressSpace": ["10.3.0.0/16"],
      "subnets": [
        {
          "name": "snet-data",
          "addressPrefix": "10.3.1.0/24",
          "tierLabel": "data",
          "sensitive": true,
          "nsgIntents": ["allow-app-tier-only", "deny-internet-inbound", "deny-all-inbound-other"],
          "routeToFirewall": true,
          "serviceEndpoints": ["Microsoft.Storage", "Microsoft.KeyVault"]
        },
        {
          "name": "snet-pe",
          "addressPrefix": "10.3.2.0/24",
          "tierLabel": "pe",
          "sensitive": true,
          "nsgIntents": ["deny-internet-inbound"],
          "privateEndpointSubnet": true,
          "routeToFirewall": true
        }
      ]
    }
  ]
}
```

**Validation expectation after `ValidateBeforeEmit`:** Zero Critical, zero High findings.
The `snet-data` subnet is `sensitive: true` but has `deny-internet-inbound` intent + `routeToFirewall: true`,
so the fixture projection will show no internet-reachable path. The gate passes.

---

## 2. Module Registry

**Purpose:** The vetted set of Terraform modules the renderer selects from. The renderer never invents
module sources. Any module source not in this registry causes the renderer to return an error.

### 2.1 Registry entry schema

```go
// ModuleRegistryEntry describes one approved Terraform module.
type ModuleRegistryEntry struct {
    // ID is the stable internal identifier referenced by renderer selection rules.
    ID string

    // Source is the Terraform module source string as it appears in HCL.
    // For public registry: "Azure/network/azurerm"
    // For AT&T internal: "artifactory.att.com/tf-modules/att-vnet/azurerm" [VERIFY]
    Source string

    // Version is the pinned version. MUST be an exact version, never a constraint
    // like ">= 3.0". The renderer uses this exact string in the version = "..." field.
    Version string

    // Purpose describes what this module provisions. Used in generated PR comments.
    Purpose string

    // Handles is the list of TopologySpec capabilities this module can satisfy.
    Handles []string

    // RequiredInputs lists the module input variables the renderer must always set.
    RequiredInputs []string

    // Notes contains AT&T-specific guidance or [VERIFY] items.
    Notes string
}
```

### 2.2 Approved module table

| ID | Source | Pinned version | Handles | Notes |
|---|---|---|---|---|
| `att-vnet` | `artifactory.att.com/tf-modules/att-vnet/azurerm` | `2.4.1` | Hub VNet, spoke VNet, address space, DNS servers | **[VERIFY]** AT&T internal module name and version. Falls back to `az-vnet` if unavailable. |
| `az-vnet` | `Azure/network/azurerm` | `5.3.0` | VNet creation, address space | Public fallback when `att-vnet` not available |
| `az-subnets` | `Azure/subnets/azurerm` | `1.0.0` | Subnet creation, NSG association, route table association, service endpoints, delegations | Used for all subnet creation |
| `az-nsg` | `Azure/network-security-group/azurerm` | `4.1.0` | NSG + security rules from intent vocabulary | Renderer maps NSGIntent strings to module rule inputs (§3.3) |
| `az-hub-spoke` | `Azure/caf-enterprise-scale/azurerm` | `6.2.0` | Hub-spoke peering topology, UDR propagation | Preferred for hub-spoke peering. Scope-limits to connectivity module only. |
| `az-vpn-gw` | `Azure/vpn-gateway/azurerm` | `1.3.2` | VPN Gateway in GatewaySubnet | Used when `gatewayType == "vpn"` |
| `az-er-gw` | `Azure/expressroute-gateway/azurerm` | `1.1.0` | ExpressRoute Gateway | Used when `gatewayType == "expressroute"` |
| `az-firewall` | `Azure/firewall/azurerm` | `2.2.1` | Azure Firewall + Firewall Policy, Standard/Premium SKU | Used when `firewallEnabled == true` |
| `az-bastion` | `Azure/bastion/azurerm` | `2.0.0` | Azure Bastion in AzureBastionSubnet | Used when `tierLabels` contains `"bastion"` |
| `az-appgw-waf` | `Azure/application-gateway/azurerm` | `3.1.0` | Application Gateway with WAF v2 policy | Used when `tierLabels` contains `"appgw"`. **WAF policy MUST be enabled — renderer rejects WAFMode=Detection.** |
| `az-private-endpoint` | `Azure/private-endpoint/azurerm` | `1.2.0` | Private Endpoint + Private DNS Zone link | Used when any subnet declares `privateEndpoints[]` |
| `az-route-table` | `Azure/route-table/azurerm` | `1.1.0` | UDR / route table creation | Used when `routeToFirewall: true` on any subnet |

> **[VERIFY]** `att-vnet` module: confirm source path `artifactory.att.com/tf-modules/att-vnet/azurerm`,
> exact pinned version, and which AT&T team owns the module registry. If the AT&T internal registry
> uses a different source format (e.g., `git::https://...`), update the `Source` field accordingly.

> **[VERIFY]** JFrog Artifactory Terraform registry URL format for AT&T. The renderer must use the
> correct `TF_CLI_ARGS_init` or `provider_installation` block to resolve modules from Artifactory.

### 2.3 Module selection rules

The renderer applies these rules in order to determine which modules to instantiate:

| Condition | Module(s) selected | Notes |
|---|---|---|
| Always (at least 1 VNet) | `att-vnet` (or `az-vnet` fallback) per VNet | One module call per VNet spec entry |
| Always (subnets present) | `az-subnets` per VNet | Handles subnet + NSG + RT association |
| Any subnet has `nsgIntents` | `az-nsg` per VNet | One NSG per VNet; renderer groups intents |
| `routeToFirewall: true` on any subnet | `az-route-table` per VNet | Creates UDR + 0.0.0.0/0 → NVA |
| `peeringTopology == "hub-spoke"` | `az-hub-spoke` once | Peering wired by hub-spoke module |
| `peeringTopology == "mesh"` | `az-vnet` VNet peering blocks (direct) | No dedicated mesh module; renderer emits `azurerm_virtual_network_peering` resources directly |
| `gatewayType == "vpn"` | `az-vpn-gw` once | Placed in `GatewaySubnet` of hub VNet |
| `gatewayType == "expressroute"` | `az-er-gw` once | Placed in `GatewaySubnet` of hub VNet |
| `firewallEnabled == true` | `az-firewall` once + `az-route-table` for each spoke subnet | Firewall in `AzureFirewallSubnet` of hub VNet |
| `tierLabels` contains `"bastion"` | `az-bastion` once | Placed in `AzureBastionSubnet` of hub VNet |
| `tierLabels` contains `"appgw"` | `az-appgw-waf` once | WAF policy enforced; WAFMode=Prevention required |
| Any subnet has `privateEndpoints[]` entries | `az-private-endpoint` per private endpoint | One module call per `PrivateEndpointSpec` entry |

### 2.4 Module parameterisation contract

The renderer translates `TopologySpec` fields to module input variables using the following lookup table:

| Module ID | `TopologySpec` field | Module input variable |
|---|---|---|
| `att-vnet` / `az-vnet` | `VNetSpec.Name` | `vnet_name` |
| `att-vnet` / `az-vnet` | `VNetSpec.AddressSpace` | `address_space` |
| `att-vnet` / `az-vnet` | `TopologySpec.Region` | `location` |
| `att-vnet` / `az-vnet` | `TopologySpec.Tags` | `tags` |
| `az-subnets` | `SubnetSpec.Name` | `subnet_names` (list) |
| `az-subnets` | `SubnetSpec.AddressPrefix` | `subnet_prefixes` (list) |
| `az-subnets` | `SubnetSpec.ServiceEndpoints` | `subnet_service_endpoints` (map) |
| `az-subnets` | `SubnetSpec.Delegations` | `subnet_delegation` (map) |
| `az-nsg` | `SubnetSpec.NSGIntents` (expanded) | `predefined_rules` + `custom_rules` |
| `az-nsg` | `SubnetSpec.Name` | `security_group_name` (derived: `nsg-{vnet}-{subnet}`) |
| `az-hub-spoke` | `HubVNetName` | `hub_virtual_network_resource_id` |
| `az-hub-spoke` | Spoke VNet IDs | `virtual_network_resource_ids_to_peer_to_hub` |
| `az-firewall` | `TopologySpec.Region` | `location` |
| `az-firewall` | Hub VNet ID | `virtual_network_id` |
| `az-firewall` | `TopologySpec.Tags` | `tags` |
| `az-vpn-gw` | Hub `GatewaySubnet` ID | `subnet_id` |
| `az-vpn-gw` | `TopologySpec.Tags` | `tags` |
| `az-route-table` | Firewall private IP | `next_hop_ip_address` (for 0.0.0.0/0 route) |
| `az-route-table` | Spoke subnet IDs | `subnet_ids` |
| `az-appgw-waf` | `TopologySpec.Region` | `location` |
| `az-appgw-waf` | `"WAF_v2"` (hardcoded) | `sku_name` |
| `az-appgw-waf` | `"Prevention"` (hardcoded) | `waf_configuration.firewall_mode` |
| `az-private-endpoint` | `PrivateEndpointSpec.Name` | `name` |
| `az-private-endpoint` | `PrivateEndpointSpec.GroupID` | `subresource_names` / DNS-zone derivation |
| `az-private-endpoint` | `PrivateEndpointSpec.ServiceResourceID` | `private_connection_resource_id` |

### 2.5 Registry versioning rules

1. **Exact pin only:** Every module entry in the registry has an exact version string (e.g., `"5.3.0"`).
   The renderer never emits `version = ">= 5.0"` or `version = "~> 5.3"`. Unpinned versions are
   rejected at registry load time.
2. **Registry snapshot commit SHA:** Each `generate_topology` invocation records the Git commit SHA of
   the registry file at the time of generation. This becomes part of the audit trail (§6.7).
3. **Version updates require registry PR:** Module version bumps require a separate PR to the module
   registry snapshot file. The generator tool reads the registry from a well-known path in the
   infrastructure repository. [VERIFY] Path in AT&T infrastructure repository.
4. **Source immutability:** Module sources in the registry may not be overridden by `TopologySpec` fields.
   The LLM never specifies a module source.

---

## 3. Renderer Contract

**Purpose:** Deterministic, pure-function translation of `TopologySpec` → `TerraformPlan`.
Same spec always produces identical Terraform. The renderer also produces a `graph.Fixture`
projection for `ValidateBeforeEmit`.

### 3.1 Function signature

```go
// RenderTerraform translates a validated TopologySpec into a TerraformPlan.
// It is a pure function: no I/O, no randomness, no timestamps.
// The same spec always produces the same plan (modulo registry snapshot hash).
//
// Returns an error if:
//   - spec.Validate() fails
//   - any NSGIntent in spec is not in the approved vocabulary (§3.3)
//   - any required module is not found in registry
//   - CIDR overlap detected across peered VNets
//   - firewallEnabled == true but no AzureFirewallSubnet present in hub VNet
//   - a declared PrivateEndpointSpec.GroupID cannot be mapped to a Private DNS zone
// The baseline parameter carries subscription-owned inputs that the generator must
// respect but does not create (currently AVNM Security Admin Rules only).
func RenderTerraform(spec TopologySpec, registry ModuleRegistry, baseline ProjectionBaseline) (TerraformPlan, error)

// ProjectionBaseline is the read-only subscription context used by fixture projection.
// It is intentionally narrow: only fields that Analyze() reads and that are not owned
// by TopologySpec are copied in.
type ProjectionBaseline struct {
    AVNMSecurityAdminRules []graph.AdminRule
}

// TerraformPlan is the renderer's output.
type TerraformPlan struct {
    // Files maps HCL filename to HCL content.
    // Standard filenames: "main.tf", "variables.tf", "outputs.tf", "versions.tf",
    // "nsg.tf", "routes.tf", "peering.tf", "gateway.tf", "firewall.tf".
    Files map[string]string

    // SpecHash is the SHA-256 of the canonical JSON serialisation of the input
    // TopologySpec. Used for cache invalidation and audit trail.
    SpecHash string

    // RegistrySnapshotSHA is the Git commit SHA of the module registry file.
    RegistrySnapshotSHA string

    // FixtureProjection is the graph.Fixture representation of the generated topology.
    // This is the input to ValidateBeforeEmit → Analyze(). See §3.5.
    FixtureProjection *graph.Fixture
}
```

### 3.2 Determinism guarantees

| Guarantee | Mechanism |
|---|---|
| No timestamps in identifiers | Resource names derive from `spec.Tags["appid"]` + tier label + VNet name. No `time.Now()` calls. |
| Stable map iteration | All `map[string]string` fields are serialised with sorted keys before generating HCL. |
| Stable slice ordering | VNets are rendered in spec order. Subnets rendered in spec order. NSG rules rendered by priority (ascending), derived from intent position in vocab table. |
| Idempotent `SpecHash` | `json.Marshal` with sorted keys → `sha256.Sum256`. Any field reorder in the input spec will not change the hash if the semantic content is identical. |
| No external calls | `RenderTerraform` makes no network calls. Module sources are read from the in-memory registry. |

### 3.3 NSG intent vocabulary

The renderer recognises exactly these 16 intent strings. Any intent outside this table causes
`RenderTerraform` to return `ErrUnknownNSGIntent`. The LLM is instructed to use only these values.

| Intent string | Direction | Access | Protocol | Source | Destination port | Priority | Notes |
|---|---|---|---|---|---|---|---|
| `allow-https-from-internet` | Inbound | Allow | TCP | Internet | 443 | 100 | Web-tier ingress |
| `allow-http-from-internet` | Inbound | Allow | TCP | Internet | 80 | 110 | Web-tier HTTP (usually redirect to HTTPS) |
| `allow-ssh-from-bastion` | Inbound | Allow | TCP | VirtualNetwork | 22 | 200 | SSH from Bastion-managed subnet only; renderer sets SourceAddressPrefix to AzureBastionSubnet CIDR |
| `allow-rdp-from-bastion` | Inbound | Allow | TCP | VirtualNetwork | 3389 | 210 | RDP from Bastion-managed subnet only |
| `allow-bastion-rdp-ssh` | Inbound | Allow | TCP | VirtualNetwork | 22,3389 | 200 | Combined RDP+SSH from bastion; used on mgmt tier |
| `allow-internal-vnet` | Inbound | Allow | `*` | VirtualNetwork | `*` | 300 | Allow all intra-VNet; typical for app tier. **Note:** ValidateBeforeEmit fires segmentation warning if `sensitive: true` subnet uses this intent without a paired deny; architect must pair with `deny-all-inbound-other`. |
| `allow-app-tier-only` | Inbound | Allow | TCP | app-tier CIDR | `*` | 300 | Restricts data tier to app-tier source CIDR; renderer derives CIDR from spec |
| `allow-appgw-management` | Inbound | Allow | TCP | GatewayManager | 65200-65535 | 100 | Required for Application Gateway v2 management traffic |
| `allow-lb-probes` | Inbound | Allow | TCP | AzureLoadBalancer | `*` | 400 | Standard Load Balancer health probes |
| `deny-internet-inbound` | Inbound | Deny | `*` | Internet | `*` | 1000 | Explicit deny of all internet-sourced traffic |
| `deny-all-inbound-other` | Inbound | Deny | `*` | `*` | `*` | 4096 | Catch-all deny at maximum priority |
| `deny-all-outbound-internet` | Outbound | Deny | `*` | `*` | `*` | 1000 | Force all outbound through firewall; pair with `routeToFirewall: true` |
| `allow-azure-monitor` | Outbound | Allow | TCP | VirtualNetwork | 443 | 200 | Azure Monitor / Log Analytics outbound |
| `allow-key-vault` | Outbound | Allow | TCP | `AzureKeyVault` service tag | 443 | 210 | Key Vault outbound over service endpoint |
| `allow-storage` | Outbound | Allow | TCP | `Storage` service tag | 443 | 220 | Azure Storage outbound over service endpoint |
| `deny-all-inbound-other` | Inbound | Deny | `*` | `*` | `*` | 4096 | *(alias — same as row 11 above; accepted as duplicate for ergonomics)* |

> **Renderer behaviour on unknown intent:** Returns `ErrUnknownNSGIntent{Intent: "<value>"}`.
> The `generate_topology` handler treats this as a generation failure — the LLM is called again
> with the error injected into the context (up to `max_iterations`).

**Projection invariant for Gate 2:** Any intent whose source is "Internet" MUST project to
`graph.SecRule.SourceAddressPrefix == "Internet"` in both the declared NSG rule set and
`NetworkWatcher.EffectiveSecurityRules`. For Phase 3 this applies at minimum to:
`allow-https-from-internet`, `allow-http-from-internet`, and `deny-internet-inbound`.
Using an empty source, module-local alias, or any value other than `"Internet"` / `"0.0.0.0/0"`
would cause `Analyze()` to miss the rule entirely.

### 3.4 NIC and VM generation — Phase 3 scope decision

**Decision: The renderer does NOT emit compute resources (NICs, VMs, VMSS, AKS node pools).**

**Rationale:**
1. Phase 3 scope is network topology — VNets, subnets, NSGs, route tables, peerings, gateways.
   Compute resources introduce IaC coupling (OS image versions, SKUs, availability zones) that belongs
   in application team repositories, not a shared network topology generator.
2. `ValidateBeforeEmit` needs a `graph.Fixture` with NICs to fire NIC-level findings (Gate 2, Gate 4).
   The fixture projection (§3.5) synthesises *synthetic NICs* for the subnets that matter to the
   engine — sensitive tiers plus direct-ingress / lateral-movement tiers such as `web`, `app`,
   `data`, and `dmz`. This is sufficient to trigger the engine's critical paths without emitting
   real compute HCL.
3. Emitting VM HCL would require AT&T-specific image references, boot diagnostics settings, and
   compliance tag requirements that are outside the network team's governance boundary.

**Consequence for `ValidateBeforeEmit`:** The fixture projection generates synthetic NICs (see §3.5).
These are annotated with `Tags["synthetic"] = "true"` so findings referencing them are labelled
"(synthetic NIC — validate after compute deployment)" in the PR description.

### 3.5 Fixture projection — HCL → `graph.Fixture`

The fixture projection is computed *from the `TopologySpec`* (not from parsed HCL — the spec is the
authoritative source). This is the `FixtureProjection` field of `TerraformPlan`.

```go
// ProjectFixture derives a graph.Fixture from a TopologySpec plus the read-only
// subscription baseline needed for engine parity.
// This fixture is used exclusively by ValidateBeforeEmit → Analyze().
// It is NOT used for any live Azure API query.
//
// Projection rules:
//   - One graph.VNet per VNetSpec, with AddressSpace and Subnets populated.
//   - One graph.Subnet per SubnetSpec, with NSG and RouteTable names derived.
//   - One graph.NSG per VNet. SecurityRules populated from NSGIntent expansion (§3.3).
//   - One graph.RouteTable per VNet with routeToFirewall subnets.
//     Routes: ["0.0.0.0/0 → VirtualAppliance → <firewall_private_ip>"] when firewallEnabled.
//     Routes: ["0.0.0.0/0 → Internet"] when firewallEnabled == false and routeToFirewall == false.
//   - Synthetic NICs: one per subnet where sensitive==true or tierLabel∈{"web","app","data","dmz"}.
//     NIC fields: Name="synthetic-nic-{subnet}", Subnet="{vnet}/{subnet}", PrivateIP="<derived>",
//     Tags={"sensitive":"true"} if sensitive==true else Tags={}.
//     PublicIP: deterministic synthetic PIP when the subnet is a direct-internet ingress tier
//     (e.g. NSG intents include allow-http(s)-from-internet and the topology does not put an
//     Application Gateway or Bastion in front of that subnet); nil otherwise.
//   - ResourceGraph.PublicIPAddresses: populated for every synthetic NIC PublicIP so Gate 4
//     evaluates the same inputs Analyze() expects.
//   - EffectiveSecurityRules (NetworkWatcher): projected from expanded NSG rules for each synthetic NIC.
//     Projection logic mirrors engine/go/simulator/apply.go projectEffectiveRules().
//     Internet-facing intents MUST produce SourceAddressPrefix="Internet" (or "0.0.0.0/0").
//   - EffectiveRoutes (NetworkWatcher): projected from RouteTable routes for each synthetic NIC's subnet.
//   - Peerings: populated from PeeringPairs (or derived from peeringTopology).
//   - AzureFirewall: populated from FirewallEnabled flag (synthetic Firewall struct with SKUTier="Standard");
//     NatRules is empty because Phase 3 does not generate DNAT authoring paths.
//   - AVNM.SecurityAdminRules: copied from baseline.AVNMSecurityAdminRules so Gate 1 sees the
//     subscription's existing admin-rule posture.
//   - PrivateEndpoints: one graph.PrivateEndpoint per PrivateEndpointSpec with
//     ConnectionState="Approved" and GroupId/PrivateLinkServiceId carried through.
//   - PrivateDnsZones: derived from PrivateEndpointSpec.GroupID using the same mapping as the engine,
//     with the hosting VNet linked so the private DNS gate evaluates correctly.
//   - ApplicationGateways: populated when the renderer selects az-appgw-waf; the projection sets
//     PublicIP, WafEnabled=true, and WafMode="Prevention".
//   - AzureBastions: populated when the renderer selects az-bastion; its presence is required for
//     Analyze() to enforce the Bastion bypass rule.
//   - AKSClusters, LoadBalancers, APIManagements, AzureFrontDoors, VirtualWANs, and
//     CrossSubscriptionPeerings: explicitly not generated in Phase 3 and therefore left empty.
func ProjectFixture(spec TopologySpec, baseline ProjectionBaseline) *graph.Fixture
```

**Engine-field coverage matrix (must stay in lock-step with `Analyze()`):**

| Engine-read field | Projection / disposition |
|---|---|
| `ResourceGraph.NetworkInterfaces[]` | Populated with synthetic NICs |
| `NIC.Tags["sensitive"]` | Copied from `SubnetSpec.Sensitive` |
| `NIC.PublicIP` + `ResourceGraph.PublicIPAddresses[]` | Deterministic synthetic PIP only for direct-ingress synthetic NICs; otherwise explicitly nil / empty |
| `NetworkWatcher.EffectiveSecurityRules` | Populated from canonical NSG intent expansion |
| `NetworkWatcher.EffectiveRoutes` | Populated from projected route tables and `routeToFirewall` |
| `AVNM.SecurityAdminRules` | Copied from `ProjectionBaseline` |
| `AzureFirewall` | Populated when `firewallEnabled == true`; `NatRules` empty by design in Phase 3 |
| `ResourceGraph.VirtualNetworks[].AddressSpace` | Populated from `VNetSpec.AddressSpace` |
| `ResourceGraph.PrivateEndpoints[]` | Populated from `SubnetSpec.PrivateEndpoints[]` |
| `ResourceGraph.PrivateDnsZones[]` | Populated from PE `GroupID` → zone mapping |
| `ResourceGraph.ApplicationGateways[]` | Populated when app gateway module is rendered |
| `ResourceGraph.AzureBastions[]` | Populated when bastion module is rendered |
| `ResourceGraph.AKSClusters[]`, `LoadBalancers[]`, `APIManagements[]`, `AzureFrontDoors[]`, `VirtualWANs[]`, `CrossSubscriptionPeerings[]` | Explicitly not generated in Phase 3; projection leaves them empty |

**Critical projection rule — route table NextHopType:**

| Condition | NextHopType for 0.0.0.0/0 | Engine Gate 3 verdict |
|---|---|---|
| `firewallEnabled == true` AND `routeToFirewall == true` | `VirtualAppliance` | Not internet-reachable via route |
| `firewallEnabled == false` AND `routeToFirewall == false` | `Internet` | Internet-reachable via route |
| `firewallEnabled == true` AND `routeToFirewall == false` | `Internet` | **Internet-reachable** (engine will fire if sensitive NIC in subnet) |
| `firewallEnabled == false` AND `routeToFirewall == true` | `VirtualAppliance` | Render error — NVA IP unknown; renderer rejects this combination |

**CIDR overlap pre-check in projection:** Before calling `Analyze()`, `ProjectFixture` runs a
CIDR-overlap check across all VNets in peering scope. If overlap is detected, it returns a
`ProjectionError{Type: "cidr-overlap"}` without calling `Analyze()` — this is a structural error
that the LLM cannot resolve by refining NSG intents.

---

## 4. Validation Gate

**Purpose:** `ValidateBeforeEmit` is the mandatory security gate between the renderer and PR creation.
It runs `Analyze()` on the fixture projection and blocks emission of Terraform if any High or Critical
findings are present. It is not bypassable.

### 4.1 Function signature

```go
// ValidateBeforeEmit runs Analyze() on the TerraformPlan's FixtureProjection and
// returns a ValidationResult. PR creation consumes this object directly; callers do
// not thread a raw "approved bool" through the system.
//
// Gate logic:
//   approved = len(findings where severity ∈ {"Critical", "High"}) == 0
//
// If approved == false:
//   - The TerraformPlan.Files are NOT written to any output.
//   - The findings are returned to the caller for LLM refinement (§4.3) or final failure.
//   - No GitHub PR is created.
//
// If approved == true:
//   - The full findings slice (including Medium/Low/Informational) is returned.
//   - Medium/Low/Informational findings are advisory — they appear in the PR description
//     but do not block PR creation.
//
// ValidateBeforeEmit has NO parameters to bypass the gate. There is no
// "force", "skip_validation", or "dry_run" parameter. See §6.2.
type ValidationResult struct {
    Findings []analyze.Finding
    Approved bool
}

func ValidateBeforeEmit(plan TerraformPlan) ValidationResult
```

### 4.2 Finding classification

| Severity | Gate effect | PR effect |
|---|---|---|
| Critical | **Blocks emission** | Never reaches PR (gate failed) |
| High | **Blocks emission** | Never reaches PR (gate failed) |
| Medium | Advisory — does not block | Appears in PR description under "⚠️ Advisory Findings" |
| Low | Advisory — does not block | Appears in PR description under "ℹ️ Low-Severity Notes" |
| Informational | Advisory — does not block | Appears in PR description (collapsed section) |

### 4.3 Iterative refinement loop

When `ValidateBeforeEmit` returns `approved == false`, the `generate_topology` handler may call the
LLM again with the blocking findings injected as context. The loop has a hard limit.

```
var validation ValidationResult
for iteration := 1; iteration <= maxIterations; iteration++ {
    spec, err = callLLM(intent, failingFindings)   // findings from previous iteration (or nil on first)
    plan, err = RenderTerraform(spec, registry, baseline)
    validation = ValidateBeforeEmit(plan)
    if validation.Approved {
        break
    }
    failingFindings = filterBlocking(validation.Findings) // Critical + High only
    if containsUnfixableBlockingFinding(failingFindings) {
        break // do not spend the remaining iterations on a finding the spec cannot change
    }
}
if !validation.Approved {
    return GenerationResult{GatePass: false, Findings: failingFindings, Iterations: maxIterations}
    // Caller receives the findings. No PR is created. No silent retry.
}
```

**Refinement prompt injection:** On iteration 2+, the LLM receives the following additional context
appended to the system prompt:

```
VALIDATION FAILURE — iteration {N} of {max}.
The previous TopologySpec produced the following blocking security findings:

{findings as JSON array}

You MUST revise the TopologySpec to eliminate these findings. Common fixes:
- Add "deny-internet-inbound" to sensitive subnet nsgIntents
- Set routeToFirewall: true on subnets with sensitive: true
- Remove allow-https-from-internet from data/app tier subnets
- Pair allow-internal-vnet with deny-all-inbound-other on sensitive subnets
- For `private DNS zone missing` / `private DNS zone not linked to VNet`, add or correct
  the relevant `privateEndpoints[]` declaration so the renderer emits the required Private DNS link
- For Bastion findings, replace direct internet management intents with `allow-bastion-rdp-ssh`
- If AVNM `AlwaysAllow` keeps a port exposed, remove direct internet ingress on that subnet or
  force the subnet through the firewall so Gate 3 closes the path

Do NOT change the intent described by the architect. Only change NSG intents,
routeToFirewall flags, subnet labelling, and other generator-owned fields (for example
`privateEndpoints[]`) to satisfy the security gate.
```

**Hard limits:**

| Parameter | Value | Rationale |
|---|---|---|
| `max_iterations` default | 2 | Two attempts is sufficient for well-described intent; more suggests ambiguous input |
| `max_iterations` caller-settable range | 1–3 | Never 0 (at least one attempt required); never >3 (infinite loop prevention) |
| On hard fail | Return `GatePass: false` with full findings | Caller (architect) must revise intent and resubmit |

### 4.4 Findings that always block (engine rule registry)

The following findings will always cause gate failure when they appear at Critical or High:

| Engine finding type | Severity (engine) | Gate-blocking | Typical cause in generated topology |
|---|---|---|---|
| `over-permissive NSG (reachable)` | Critical / High | Yes | Internet-sourced allow rule + `NIC.PublicIP` + `0.0.0.0/0 -> Internet`; severity becomes Critical when `NIC.Tags["sensitive"] == "true"` |
| `missing tier segmentation` | High | Yes | `sensitive: true` NIC with `AllowVnetInBound` and no `DenyVnetInBound` |
| `internet reachable via load balancer NAT` | High | Yes | Load balancer NAT rule exposing internet traffic (explicitly not generated in Phase 3) |
| `Bastion bypass — direct management port exposed` | High | Yes | Direct NSG allow of RDP/SSH from internet instead of from bastion |
| `private DNS zone missing` / `private DNS zone not linked to VNet` | High | Yes | Private endpoint exists but required Private DNS zone is absent or not linked |
| `cross-sub peering without firewall` | Medium | No (advisory) | Cross-subscription peering detected; firewall presence advisory |
| `App Gateway WAF disabled` | Medium | No (advisory) | WAF disabled on AppGW (renderer enforces Prevention mode; this fires for pre-existing AppGWs in fixture) |
| `AKS non-private` | Medium | No (advisory) | AKS API server reachable from internet (not generated by renderer in Phase 3) |
| `orphaned PIP` | Low | No | Unassociated public IP resource |
| `CIDR overlap` | Medium | No (advisory) | Overlapping address spaces in peered VNets |

### 4.5 Anti-bypass design

The gate bypass prevention is structural, not policy:

1. **No `ForceEmit` parameter exists** in `ValidateBeforeEmit` or any caller-facing API.
2. **`TerraformPlan.Files` is not accessible** to the PR creation code path except through a
   `ValidationResult` returned by `ValidateBeforeEmit`. The PR workflow function signature is:
   ```go
   func CreatePR(ctx context.Context, plan TerraformPlan, validation ValidationResult) (prURL string, err error)
   ```
   `CreatePR` returns `ErrGateFailed` if `validation.Approved == false`. There is no separate
   caller-supplied `approved` flag to spoof or accidentally stale-cache across branches.
3. **Audit log integrity:** The audit entry for every `generate_topology` call includes the
   `ValidateBeforeEmit` findings hash. If `validation.Approved == true` but the findings hash is empty (no
   findings at all), the audit log records a warning — zero findings on a real topology is unusual
   and warrants review.

---

## 5. PR Workflow

**Purpose:** How a validated `TerraformPlan` becomes a pull request against the AT&T infrastructure
Terraform repository, with human approval enforced before any apply.

### 5.1 PR workflow overview

```
generate_topology MCP tool (`validation.Approved == true`)
  │
  ├── 1. Write Terraform files to ephemeral workspace (in-memory or tmpfs — never persisted to disk)
  ├── 2. Commit to a new branch: att-nettopo/{appid}/{spec-hash-prefix}
  ├── 3. Push branch to AT&T infrastructure repository [VERIFY repo name below]
  ├── 4. Create PR via GitHub API (gh CLI or REST)
  │       PR title:  "[nettopo] {spec.Tags["appid"]} — {spec.Description[:80]}"
  │       PR body:   see §5.4
  │       PR labels: ["network-topology", "generated", "needs-review"]
  └── 5. Return pr_url to MCP tool caller
```

### 5.2 GitHub Actions workflow — OIDC auth

```yaml
# .github/workflows/nettopo-generate.yml
# Triggered by the generate_topology MCP tool via gh CLI or workflow_dispatch.
# OIDC — no AZURE_CLIENT_SECRET ever.

name: Network Topology Generation Gate

on:
  workflow_dispatch:
    inputs:
      spec_hash:
        description: 'SHA-256 of the TopologySpec (from generate_topology output)'
        required: true
      plan_artifact:
        description: 'Artifact name containing TerraformPlan.Files'
        required: true

permissions:
  id-token: write       # OIDC token for Azure federated credential
  contents: write       # Push branch + create PR
  pull-requests: write  # Create PR

jobs:
  validate-and-pr:
    runs-on: ubuntu-latest
    environment: production   # Required reviewers enforced here — see §5.5

    steps:
      - name: Checkout infrastructure repo
        uses: actions/checkout@v4
        with:
          repository: ${{ vars.INFRA_REPO }}   # [VERIFY] AT&T infra repo name
          token: ${{ secrets.GH_APP_TOKEN }}

      - name: Azure login (OIDC — no secret)
        uses: azure/login@v2
        with:
          client-id: ${{ vars.AZURE_CLIENT_ID }}
          tenant-id: ${{ vars.AZURE_TENANT_ID }}
          subscription-id: ${{ vars.AZURE_SUBSCRIPTION_ID }}
          # No client-secret — OIDC federated identity only

      - name: Configure JFrog Artifactory for Terraform
        run: |
          cat > ~/.terraformrc << 'EOF'
          credentials "artifactory.att.com" {
            token = "${{ secrets.JFROG_TOKEN }}"
          }
          provider_installation {
            network_mirror {
              url = "https://artifactory.att.com/artifactory/terraform-registry/"
              include = ["registry.terraform.io/*/*"]
            }
            direct {
              exclude = ["registry.terraform.io/*/*"]
            }
          }
          EOF
        # [VERIFY] JFrog Artifactory Terraform registry URL for AT&T

      - name: Download plan artifact
        uses: actions/download-artifact@v4
        with:
          name: ${{ inputs.plan_artifact }}
          path: ./generated-topology/

      - name: Terraform init (module resolution from Artifactory)
        working-directory: ./generated-topology
        run: terraform init -backend=false

      - name: Terraform validate (syntax check only — no apply)
        working-directory: ./generated-topology
        run: terraform validate

      - name: Create PR branch and push
        run: |
          git config user.email "nettopo-bot@att.com"
          git config user.name  "AT&T NetTopo Generator"
          BRANCH="att-nettopo/${{ inputs.spec_hash_prefix }}"
          git checkout -b "${BRANCH}"
          git add generated-topology/
          git commit -m "feat(nettopo): generated topology ${BRANCH}"
          git push origin "${BRANCH}"

      - name: Create pull request
        env:
          GH_TOKEN: ${{ secrets.GH_APP_TOKEN }}
        run: |
          gh pr create \
            --repo "${{ vars.INFRA_REPO }}" \
            --base main \
            --head "att-nettopo/${{ inputs.spec_hash_prefix }}" \
            --title "[nettopo] ${{ inputs.appid }} — network topology" \
            --body-file generated-topology/PR_BODY.md \
            --label "network-topology,generated,needs-review"
```

> **[VERIFY]** `vars.INFRA_REPO` — confirm the AT&T infrastructure Terraform repository name and org.
> The generator tool must not push to `azure-network-topology-reviewer` itself.

> **[VERIFY]** `secrets.GH_APP_TOKEN` — confirm whether a GitHub App token or PAT is used for
> infrastructure repo access. GitHub App is preferred (auditable, scoped).

> **[VERIFY]** `secrets.JFROG_TOKEN` — confirm secret name and whether JFrog Artifactory requires
> a Terraform-specific API token or uses the same service account token as Docker.

### 5.3 AVNM vs direct AzureRM — trade-off decision

The renderer must choose between two Terraform approaches for network security configuration:

| Approach | When to use | When NOT to use |
|---|---|---|
| `azurerm_network_manager_*` (AVNM) | Managing security admin rules and network groups on NEW deployments in a subscription where AVNM is already deployed. When `avnmEnabled == true` in spec. | Creating new VNets/subnets for the first time — AVNM admin rules reference Network Groups by membership, which is resolved by AVNM, not Terraform; bootstrapping is a chicken-and-egg problem. |
| Direct `azurerm_virtual_network` + `azurerm_subnet` + `azurerm_network_security_group` | Creating new VNets, subnets, NSGs (the default Phase 3 path). When `avnmEnabled == false`. When `avnmEnabled == true` but only data source referencing is needed. | Modifying AVNM security admin rules on existing network groups (use AVNM resources instead). |

**Phase 3 rule:** The renderer always uses **direct AzureRM resources** for new VNet/subnet/NSG
creation. When `avnmEnabled == true`, the renderer emits a `data "azurerm_network_manager" ...`
data source block to reference the existing AVNM instance, but does NOT emit `azurerm_network_manager_*`
resource blocks. Rationale: the AVNM instance is managed separately by the network platform team;
the topology generator creates within the AVNM-governed perimeter, not alongside it.

### 5.4 PR content specification

The PR body (`PR_BODY.md`) is generated by the renderer and contains:

```markdown
## Network Topology Generation — Automated PR

**Architect intent:** {spec.Description}
**AppID:** {spec.Tags["appid"]}
**Region:** {spec.Region}
**Generated:** {RFC3339 timestamp}
**Spec hash:** `{plan.SpecHash}`
**Module registry snapshot:** `{plan.RegistrySnapshotSHA}`
**Iterations required:** {iterations}

---

## Topology Summary

{rendered summary table: VNets × subnets × NSG intents × sensitive flags}

---

## ValidateBeforeEmit Result

✅ **Gate PASSED** — 0 Critical, 0 High findings.

{findings table — only Medium/Low/Informational if present}

> Advisory findings do not block this PR. Review and address post-merge if applicable.

---

## Attached: TopologySpec

<details>
<summary>TopologySpec JSON (expand)</summary>

```json
{spec JSON — pretty-printed}
```

</details>

---

## Audit Trail

| Field | Value |
|---|---|
| Intent hash | `SHA-256({spec.Description})` |
| SpecHash | `{plan.SpecHash}` |
| ValidateBeforeEmit findings hash | `SHA-256({findings JSON})` |
| Registry snapshot SHA | `{plan.RegistrySnapshotSHA}` |
| Generator version | `{engine version tag}` |

---

⚠️ **This PR was generated by the AT&T Network Topology Generator. Apply requires human approval.**
**No auto-merge. Reviewer must confirm the topology matches the stated intent before approving.**
```

### 5.5 Approval gate

| Enforcement point | Mechanism | Configuration |
|---|---|---|
| Required reviewers | GitHub Actions `environment: production` | At least 1 reviewer from `att-network-approvers` team required |
| Branch protection | `main` branch requires PR + passing status checks | Configured in infrastructure repo settings [VERIFY] |
| No auto-merge | `--auto-merge` flag is never set in `gh pr create` call | Enforced in workflow — no auto-merge path exists |
| terraform apply scope | Managed Identity holds Reader role only | The workflow never runs `terraform apply`; apply is triggered post-merge by a separate, human-initiated pipeline |

> **[VERIFY]** GitHub team name for required reviewers: `att-network-approvers`. Confirm team slug in
> AT&T GitHub org.

> **[VERIFY]** The post-merge apply pipeline is assumed to be a separate, manually-triggered workflow
> owned by the network platform team. The generator tool has no visibility into it.

### 5.6 JFrog Artifactory image push (if applicable)

The `nettopo-generate` workflow does not push container images — it pushes Terraform files.
If a future Phase adds a custom Terraform provider built by the AT&T team, that provider binary MUST
be published to JFrog Artifactory (not GitHub Packages, not Azure ACR). The workflow would add:

```yaml
      - name: Push provider to Artifactory
        run: |
          jfrog rt upload \
            ./terraform-provider-att-nettopo_v*.zip \
            "att-terraform-providers/att-nettopo/"
        env:
          JFROG_URL: ${{ vars.JFROG_URL }}
          JFROG_USER: ${{ vars.JFROG_USER }}
          JFROG_TOKEN: ${{ secrets.JFROG_TOKEN }}
```

This is not required for Phase 3 (all modules are from the public registry or existing AT&T internal
registry). Documented here as the pattern for future phases.

---

## 6. Non-Negotiable Guardrails

**Purpose:** AT&T security and governance requirements that are enforced structurally, not by policy
convention. Each is individually documented with the enforcement mechanism.

| # | Guardrail | Enforcement mechanism |
|---|---|---|
| 1 | LLM scope boundary | `TopologySpec` schema + `ErrUnknownNSGIntent` |
| 2 | `ValidateBeforeEmit` not bypassable | Function signature — no bypass parameter |
| 3 | No write/apply permission | Managed Identity Reader role + workflow has no `terraform apply` |
| 4 | Human approval required | GitHub Actions `environment: production` + branch protection |
| 5 | AskAT&T only | LLM client hardcoded endpoint + Key Vault secret |
| 6 | Registry pinning | Exact version strings + registry load-time validation |
| 7 | Audit trail | Mandatory fields in `TerraformPlan` + PR body |

### 6.1 LLM scope boundary

**What:** The LLM (AskAT&T) is permitted to:
- Produce `TopologySpec` JSON matching the approved schema.
- Select `NSGIntent` values from the approved vocabulary (§3.3).
- Select `tierLabel` values from the approved list.
- Set `sensitive: bool` and `routeToFirewall: bool` per subnet.

The LLM is explicitly prohibited from:
- Producing raw `azurerm_network_security_rule` HCL blocks.
- Producing `priority`, `access`, `direction`, `destinationPortRange` fields at any level.
- Specifying Terraform module source paths or version strings.
- Writing route table entries (`address_prefix`, `next_hop_type`, `next_hop_in_ip_address`).
- Writing firewall rules (`nat_rule_collections`, `network_rule_collections`, `application_rule_collections`).

**Enforcement:**
1. JSON Schema validation rejects any `TopologySpec` with fields outside the defined schema
   (`additionalProperties: false` on all objects).
2. `RenderTerraform` validates every `NSGIntent` string against the vocabulary table before rendering.
   Unknown intents return `ErrUnknownNSGIntent` immediately.
3. The AskAT&T system prompt contains the verbatim CONSTRAINT block (§1.4).
4. The LLM response is parsed as `TopologySpec` only — any raw HCL in the response is discarded.
5. Terraform `module.source` is resolved only from `ModuleRegistryEntry.Source` selected by renderer rule;
   no `TopologySpec` field is ever interpolated into a module `source = ...` argument.

### 6.2 `ValidateBeforeEmit` is not bypassable

**What:** There is no code path from `RenderTerraform` output to GitHub PR creation that does not
pass through `ValidateBeforeEmit`.

**Enforcement:**
1. `CreatePR` accepts a `ValidationResult` from `ValidateBeforeEmit`, not a raw `approved bool`.
   It returns `ErrGateFailed` if `validation.Approved == false`.
2. There is no `ForceEmit`, `SkipValidation`, `DryRun`, or `--insecure` flag anywhere in the codebase.
3. The `generate_topology` tool handler calls `ValidateBeforeEmit` and passes the result directly to
   `CreatePR`. There is no intermediate storage of `plan.Files` that bypasses this call.
4. Code review policy: any PR to `engine/go/generator/` that adds a parameter bypassing validation
   must be rejected. Document this in `CONTRIBUTING.md`.

### 6.3 No write/apply permission

**What:** The Managed Identity used by the MCP server (and the GitHub Actions workflow) holds the
Azure `Reader` built-in role at the subscription scope. It has no `Contributor`, `Network Contributor`,
or custom write roles.

**Enforcement:**
1. Azure RBAC: Managed Identity assigned `Reader` role. Role assignments are infrastructure-as-code
   managed and require a separate approval. [VERIFY] Which team manages the Managed Identity role assignments.
2. GitHub Actions workflow: `terraform apply` is never called in any step of `nettopo-generate.yml`.
   The workflow runs `terraform init` + `terraform validate` only.
3. Apply is triggered post-PR-merge by a separate pipeline owned by the network platform team.
   The generator tool has no trigger, credential, or API access to that pipeline.
4. The `AZURE_CLIENT_SECRET` environment variable is never set — not in GitHub Actions, not in the
   MCP server. Authentication is OIDC federated identity only.

### 6.4 Human approval required

**What:** Every PR created by `generate_topology` requires at least one human reviewer from the
`att-network-approvers` team before merge. No auto-merge.

**Enforcement:**
1. GitHub Actions `environment: production` on the `validate-and-pr` job. Environments with required
   reviewers block the job until approval is granted.
2. Branch protection rule on `main` in the infrastructure repository: "Require pull request reviews
   before merging" with minimum 1 required reviewer. [VERIFY] Infrastructure repo branch protection config.
3. `gh pr create` in the workflow never sets `--auto-merge`.
4. The PR label `needs-review` is added; the infrastructure repo has a policy blocking merge of PRs
   with `needs-review` label until explicitly removed by a reviewer. [VERIFY] Label-based merge block policy.

### 6.5 AskAT&T only

**What:** All LLM calls use the AskAT&T endpoint with client-credentials JWT bearer authentication.
No calls to `api.openai.com`, `*.openai.azure.com`, `generativelanguage.googleapis.com`, or any
other external LLM endpoint.

**Enforcement:**
1. The LLM client in `engine/go/generator/llm.go` has a hardcoded base URL constant for the AskAT&T
   endpoint. The URL is not configurable via environment variable.
2. Client-credentials JWT is fetched from Azure Key Vault on startup (Key Vault reference injected
   via Container Apps secret). The secret is never logged, never exported to environment variables
   in workflow logs.
3. AskAT&T client-credentials flow: `POST /oauth2/token` with `client_id` + `client_secret` from
   Key Vault → bearer token with TTL (refreshed before expiry). The `client_secret` is the Key Vault
   secret value; it is held in memory only, never written to disk.
4. The MCP server's audit log records only `"llm_endpoint": "askatt"` — never logs the token or secret.

```go
// LLMClient is the AskAT&T client. The endpoint is not configurable.
type LLMClient struct {
    endpoint    string        // AskAT&T base URL — set at compile time from ldflags
    tokenSource TokenSource   // client-credentials token source backed by Key Vault
    httpClient  *http.Client
}

// TokenSource abstracts the AskAT&T client-credentials flow.
// Implementations must refresh the token before expiry.
type TokenSource interface {
    Token(ctx context.Context) (string, error)
}
```

> **[VERIFY]** AskAT&T endpoint URL and client-credentials token URL. Confirm whether AskAT&T uses
> standard OAuth2 client_credentials grant or a proprietary auth flow.

> **[VERIFY]** AskAT&T structured output / function calling API contract — confirm the API supports
> JSON Schema-constrained responses equivalent to OpenAI function calling `strict: true`.

### 6.6 Registry pinning

**What:** Every module in the registry has an exact, immutable version string. The renderer never
emits `>= version`, `~> version`, or an unpinned source.

**Enforcement:**
1. `ModuleRegistry.Load()` validates every entry at startup: if `Version` does not match
   `^[0-9]+\.[0-9]+\.[0-9]+$`, startup fails with a fatal error.
2. The renderer emits `version = "{exact}"` in every `module` block. There is no code path
   that emits a version constraint.
3. Registry updates are gated by PR to the registry snapshot file. The registry snapshot file
   is a checked-in JSON file at a well-known path in the infrastructure repository.

### 6.7 Audit trail

**What:** Every `generate_topology` invocation produces an immutable audit record.

**Required audit fields:**

| Field | Value | How derived |
|---|---|---|
| `intent_hash` | `SHA-256(spec.Description)` | Hashes the original NL intent |
| `spec_hash` | `plan.SpecHash` | SHA-256 of canonical spec JSON (§3.1) |
| `findings_hash` | `SHA-256(json.Marshal(findings))` | Hash of full findings slice from `ValidateBeforeEmit` |
| `gate_pass` | `bool` | Direct from `ValidateBeforeEmit` return |
| `iterations` | `int` | Number of LLM calls in the refinement loop |
| `registry_snapshot_sha` | `plan.RegistrySnapshotSHA` | Git SHA of module registry at generation time |
| `generator_version` | `string` | Engine version tag (set via `ldflags` at build time) |
| `llm_endpoint` | `"askatt"` | Constant — never logs token or URL with credentials |
| `pr_url` | `string` | GitHub PR URL (empty if gate failed) |
| `timestamp` | RFC3339 | UTC time of `generate_topology` invocation |
| `subscription_id` | `string` | Target subscription from tool input |

**Audit log destination:** Structured JSON to `slog` (same logger as existing MCP server),
written to stderr → forwarded to Azure Monitor via Container Apps log collection.
[VERIFY] AT&T log ingestion endpoint and workspace ID.

---

## 7. Phase 3 Placeholder Reconciliation

**Purpose:** Map Phase 2 documented limitations and Phase 1 deferred items to their Phase 3 closures.
Every item below is either closed by a specific Phase 3 addition or explicitly re-deferred with rationale.

| Ref | Phase 2 / Phase 1 note | Phase 3 disposition | Mechanism |
|---|---|---|---|
| SR-002 | `AddSubnet` produces zero `SecurityDelta` — no NICs in new subnet | **Closed with workflow constraint** | Phase 3 adds `AddNICOp` to `TopologyDelta` (§7.1). `ProjectFixture` generates synthetic NICs per subnet for generation-time validation, and simulator callers use `AddSubnet` + `AddNIC` sequentially. Pure `AddSubnet` remains cost-only. |
| SR-003 | `AddPeering` zero SecurityDelta vs Phase 1 engine — `VNet.Peerings[]` not read by any Phase 1 rule | **Closed** | Phase 3 adds `checkIntraVNetSegmentation` analysis rule (§7.2) that reads `VNet.Peerings[]` for intra-subscription peering topology. |
| P1-EW | East-west lateral movement analysis (deferred from Phase 1) | **Closed** | Phase 3 adds `checkLateralMovement` rule (§7.3) and requires it to read `VNet.Peerings[]` when evaluating peered-VNet paths. |
| P1-DNS | Hybrid DNS resolution path (deferred from Phase 1) | **Closed** | Phase 3 adds `checkHybridDNS` rule (§7.4). |
| P2-AVNM | AVNM Admin Rule delta simulation (out of scope in Phase 2) | **Closed** | Phase 3 adds `AddAVNMAdminRuleOp` and `RemoveAVNMAdminRuleOp` to `TopologyDelta` (§7.5). |
| P1-DPR | `DNSPrivateResolvers` collected but no analysis rule (Phase 2 stub) | **Closed** | Phase 3 `checkHybridDNS` rule consumes `DNSPrivateResolvers` (§7.4). |
| P1-RS | `AzureRouteServers` collected but no analysis rule (Phase 2 stub) | **Deferred to Phase 4** | Route Server BGP analysis requires on-prem route advertisement data not available from Resource Graph alone. |
| P1-DDOS | `DDoSProtectionPlans` collected but no analysis rule | **Deferred to Phase 4** | DDoS analysis is compliance-level, not reachability-level; Phase 4 scope. |
| P1-LNG | `LocalNetworkGateways` collected but no analysis rule | **Deferred to Phase 4** | LNG analysis requires VPN connection status data; out of scope for generation model. |

### 7.1 New delta operation: `AddNICOp`

Closes SR-002 for any workflow that models the workload-bearing step explicitly. Enables `simulate_change`
to model adding a NIC to a subnet (triggering Gate 2 and Gate 4 analysis). No new analysis rule is required:
the existing Phase 1 engine already reasons over NICs once one exists.

```go
// AddNICOp adds a synthetic NIC to an existing subnet.
// After application, the engine will fire findings for this NIC if the subnet's
// NSG/route configuration permits internet reachability.
// Used by Phase 3 generator to validate subnets by projecting synthetic NICs
// (§3.5) and by Phase 2 simulator to close SR-002.
type AddNICOp struct {
    // NICName is the new NIC resource name; must not already exist in fixture.
    NICName string `json:"nicName"`
    // VNetName and SubnetName identify the target subnet.
    VNetName   string `json:"vnetName"`
    SubnetName string `json:"subnetName"`
    // Sensitive sets Tags["sensitive"] = "true" if true.
    Sensitive bool `json:"sensitive"`
    // PublicIP if non-nil attaches a synthetic PIP, triggering Gate 4.
    PublicIP *AddPublicIPOp `json:"publicIp,omitempty"`
}
```

**`ApplyDelta` changes:** Adds `AddNICOp` case. Creates a `graph.NIC` with `PrivateIP` derived from
subnet CIDR (first host address of subnet that is not already allocated). Projects effective rules
and routes for the new NIC using `projectEffectiveRules` / `projectEffectiveRoutes` (existing Phase 2
logic). `SecurityDelta` will now be non-empty for `AddSubnet` + `AddNIC` sequential deltas; a pure
`AddSubnet` operation remains intentionally non-security-bearing until a NIC is added.

### 7.2 New analysis rule: `checkIntraVNetSegmentation`

Closes SR-003. Reads `VNet.Peerings[]` to detect cases where a peered spoke VNet has a sensitive
subnet reachable from another spoke without firewall intermediation.

```go
// checkIntraVNetSegmentation fires when:
//   - VNet A peers to VNet B (VNet.Peerings contains RemoteVnet=B)
//   - VNet B has a subnet with a NIC tagged sensitive=true
//   - No Azure Firewall is present (Fixture.AzureFirewall == nil) OR
//     the route table for VNet B's subnet does not have 0.0.0.0/0 → VirtualAppliance
//   - No AVNM Deny rule blocks the path
// Severity: High (sensitive NIC reachable across peered VNet without firewall control)
// Finding type: "intra-vnet segmentation gap"
```

### 7.3 New analysis rule: `checkLateralMovement`

Closes P1-EW east-west analysis deferral.

```go
// checkLateralMovement fires when:
//   - Source and destination NICs are in the same VNet OR in VNets connected by
//     VNet.Peerings[] with State="Connected"
//   - A NIC in tier T1 has effective NSG rules permitting ALL ports to a NIC in tier T2
//     (i.e., AllowVnetInBound with no port restriction)
//   - T1 and T2 have different tier labels (not intra-tier)
//   - T2 subnet is sensitive=true
// Severity: High
// Finding type: "unrestricted lateral movement to sensitive tier"
// Evidence: "NIC {name} in {tier} has AllowVnetInBound to sensitive subnet {subnet}"
```

### 7.4 New analysis rule: `checkHybridDNS`

Closes P1-DNS deferral and P1-DPR stub.

```go
// checkHybridDNS fires when:
//   - A PrivateEndpoint exists in the fixture
//   - AND no PrivateDnsZone is linked to the VNet containing the private endpoint
//     (DNS zone link not present in PrivateDnsZone.LinkedVnets[])
//   - AND no DNSPrivateResolver is present that would forward the private zone queries
//     to an on-premises resolver
// Severity: High (private endpoint DNS misconfiguration)
// Finding type: "hybrid DNS misconfiguration — PE unreachable via DNS"
// Evidence: "PrivateEndpoint {name} has no DNS zone link and no DNS Private Resolver in VNet {vnet}"
//
// Note: this subsumes the existing "Private DNS misconfiguration" finding for the case
// where DNSPrivateResolvers is non-empty. If a resolver exists, the finding is downgraded
// to Informational ("DNS forwarding path requires validation").
```

### 7.5 New delta operations: AVNM Admin Rule delta

Closes P2-AVNM.

```go
// AddAVNMAdminRuleOp adds a security admin rule to the fixture's AVNM.
// After application, adminVerdict() (Gate 1) will consider the new rule.
type AddAVNMAdminRuleOp struct {
    Rule graph.AdminRule `json:"rule"` // AdminRule to add; Name must be unique
}

// RemoveAVNMAdminRuleOp removes a named security admin rule from the fixture's AVNM.
type RemoveAVNMAdminRuleOp struct {
    RuleName string `json:"ruleName"`
}
```

**`ApplyDelta` changes:** Adds `AddAVNMAdminRuleOp` and `RemoveAVNMAdminRuleOp` cases. Modifies
`Fixture.AVNM.SecurityAdminRules[]`. Gate 1 (`adminVerdict`) is re-evaluated on the simulated
fixture by `Analyze()` — no additional projection logic required since Gate 1 reads AVNM rules
directly from `Fixture.AVNM.SecurityAdminRules`.

---

## 8. `generate_topology` MCP Tool Contract

**Purpose:** The MCP tool that wires `TopologySpec` generation, Terraform rendering, validation, and
PR creation into a single architect-facing call.

### 8.1 Tool registration

```go
// RegisterGenerateTopologyTool registers the generate_topology MCP tool with the server.
// It captures the LLM client, module registry, and GitHub client for use in the handler.
func RegisterGenerateTopologyTool(
    s *server.MCPServer,
    llm LLMClient,
    registry ModuleRegistry,
    github GitHubClient,
    logger *slog.Logger,
    auditor *Auditor,
)
```

### 8.2 Tool input schema

```json
{
  "name": "generate_topology",
  "description": "Generate an Azure network topology from architect intent. Produces validated Terraform and a GitHub PR for human approval. The topology is validated by the same engine that analyzes live deployments — zero High/Critical findings required before a PR is created.",
  "inputSchema": {
    "type": "object",
    "required": ["intent", "subscription_id", "region"],
    "additionalProperties": false,
    "properties": {
      "intent": {
        "type": "string",
        "minLength": 20,
        "description": "Natural language description of the desired network topology. Include: VNet count, tier structure, connectivity model (hub-spoke/mesh), sensitive workloads, gateway requirements, Azure Firewall preference, and mandatory tags (env, owner, costcenter, appid)."
      },
      "subscription_id": {
        "type": "string",
        "pattern": "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$",
        "description": "Target Azure subscription ID. Used to fetch AVNM and Firewall context for the fixture projection baseline."
      },
      "region": {
        "type": "string",
        "description": "Primary Azure region slug (e.g. eastus2, westus3, centralus). All generated resources use this region."
      },
      "max_iterations": {
        "type": "integer",
        "minimum": 1,
        "maximum": 3,
        "default": 2,
        "description": "Maximum number of LLM refinement iterations if ValidateBeforeEmit fails. Default 2. Increase to 3 for complex topologies where the first attempt commonly needs adjustment."
      }
    }
  }
}
```

### 8.3 Tool output schema

```go
// GenerationResult is the JSON-serialised output of the generate_topology tool.
type GenerationResult struct {
    // Spec is the TopologySpec produced by the LLM.
    // Always present (even if gate failed) so the architect can inspect what was generated.
    Spec TopologySpec `json:"spec"`

    // Plan is the rendered TerraformPlan.
    // Present only when GatePass == true.
    // Files are base64-encoded HCL content keyed by filename.
    Plan *TerraformPlanSummary `json:"plan,omitempty"`

    // Findings is the full ValidateBeforeEmit result.
    // Empty slice (not nil) when GatePass == true.
    // Contains blocking findings when GatePass == false.
    Findings []analyze.Finding `json:"findings"`

    // PRURL is the GitHub PR URL.
    // Empty string when GatePass == false or PR creation failed.
    PRURL string `json:"prUrl"`

    // Iterations is the number of LLM calls made (1 = succeeded on first attempt).
    Iterations int `json:"iterations"`

    // GatePass is true if ValidateBeforeEmit approved the plan.
    GatePass bool `json:"gatePass"`

    // Error is a human-readable error message if generation failed for reasons
    // other than gate failure (e.g., LLM error, renderer error, PR creation error).
    Error string `json:"error,omitempty"`
}

// TerraformPlanSummary is a summary of TerraformPlan for the MCP response.
// Full HCL content is attached to the GitHub PR, not returned in the MCP response
// (responses are bounded by MCP protocol limits).
type TerraformPlanSummary struct {
    SpecHash            string   `json:"specHash"`
    RegistrySnapshotSHA string   `json:"registrySnapshotSha"`
    FileNames           []string `json:"fileNames"`
    ResourceCount       int      `json:"resourceCount"` // count of Terraform resource blocks
}
```

### 8.4 Tool handler — execution flow

```go
func generateTopologyHandler(llm LLMClient, registry ModuleRegistry, github GitHubClient,
    logger *slog.Logger, auditor *Auditor) server.ToolHandlerFunc {

    return func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {

        // 1. Parse and validate inputs
        intent, _         := req.RequireString("intent")
        subscriptionID, _ := req.RequireString("subscription_id")
        region, _         := req.RequireString("region")
        maxIter           := req.GetInt("max_iterations", 2)
        if maxIter < 1 || maxIter > 3 { maxIter = 2 }

        // 2. Fetch subscription context (AVNM + Firewall baseline) — read-only
        baseline, err := fetcher.FetchFixture(ctx, subscriptionID)
        // baseline is used to populate AVNMEnabled and existing FirewallEnabled context
        // fed into the LLM system prompt as JSON context

        // 3. LLM → TopologySpec (with refinement loop)
        var (
            spec          TopologySpec
            plan          TerraformPlan
            validation    ValidationResult
            failFindings  []analyze.Finding
        )

        for iteration := 1; iteration <= maxIter; iteration++ {
            // 3a. Call AskAT&T with structured output schema
            spec, err = llm.GenerateSpec(ctx, intent, baseline, failFindings, iteration)
            if err != nil { /* handle LLM error */ }

            // 3b. Validate TopologySpec schema
            if err = spec.Validate(); err != nil { /* reject — schema error */ }

            // 3c. Render Terraform
            plan, err = RenderTerraform(spec, registry, ProjectionBaseline{
                AVNMSecurityAdminRules: baseline.AVNM.SecurityAdminRules,
            })
            if err != nil { /* handle render error — ErrUnknownNSGIntent etc. */ }

            // 3d. Validate gate — MANDATORY, cannot be skipped
            validation = ValidateBeforeEmit(plan)

            if validation.Approved { break }

            // Gate failed — collect blocking findings for next iteration
            failFindings = filterBlocking(validation.Findings) // Critical + High only
            if containsUnfixableBlockingFinding(failFindings) { break }
            logger.Info("gate failed, refining", "iteration", iteration,
                        "blocking_findings", len(failFindings))
        }

        // 4. Write audit log — always, regardless of gate outcome
        auditor.Record(ctx, AuditEntry{
            IntentHash:          sha256hex(intent),
            SpecHash:            plan.SpecHash,
            FindingsHash:        sha256hex(mustMarshal(validation.Findings)),
            GatePass:            validation.Approved,
            Iterations:          /* iteration count */,
            RegistrySnapshotSHA: plan.RegistrySnapshotSHA,
            SubscriptionID:      subscriptionID,
        })

        // 5. If gate failed — return findings, no PR
        if !validation.Approved {
            return mcpgo.NewToolResultText(mustMarshal(GenerationResult{
                Spec:       spec,
                Findings:   validation.Findings,
                GatePass:   false,
                Iterations: /* count */,
                Error:      fmt.Sprintf("gate failed after %d iterations: %d blocking finding(s)", ...),
            })), nil
        }

        // 6. Create PR (gate passed)
        prURL, err := github.CreatePR(ctx, plan, validation)
        if err != nil { /* handle PR creation error */ }

        // 7. Return result
        return mcpgo.NewToolResultText(mustMarshal(GenerationResult{
            Spec:       spec,
            Plan:       summarise(plan),
            Findings:   validation.Findings,  // advisory findings (Medium/Low/Info)
            PRURL:      prURL,
            GatePass:   true,
            Iterations: /* count */,
        })), nil
    }
}
```

### 8.5 Error taxonomy

| Error type | Returned as | Recovery |
|---|---|---|
| `ErrInvalidSubscriptionID` | `GenerationResult.Error` | Caller fixes input |
| LLM call failure (timeout, auth) | `GenerationResult.Error` | Retry with backoff; AskAT&T token may need refresh |
| `ErrSchemaValidation` | `GenerationResult.Error` | LLM output did not match TopologySpec schema; retry |
| `ErrUnknownNSGIntent{Intent}` | `GenerationResult.Error` with intent name | LLM produced out-of-vocabulary intent; refinement loop handles |
| `ErrCIDROverlap` | `GenerationResult.Error` | Architect must revise address spaces in intent |
| `ErrGateFailed` (after max iterations) | `GenerationResult.GatePass = false` | Architect must revise intent |
| `ErrPRCreationFailed` | `GenerationResult.Error` | GitHub API error; plan is valid but PR not created; plan files available in audit log |
| `UnsupportedDeltaError` (from simulator) | `GenerationResult.Error` | Phase 3 delta not supported; upgrade required |

### 8.6 Example invocation

**MCP tool call:**
```json
{
  "name": "generate_topology",
  "arguments": {
    "intent": "Create an AT&T hub-spoke network for the BCLM payment processing platform in East US 2. Hub with Azure Firewall and VPN Gateway. Three spoke VNets: web (public-facing HTTPS), app (internal APIs), data (sensitive PostgreSQL). All spoke subnets route through the firewall. Tag with env=prod, owner=network-team, costcenter=BCLM-NET, appid=PAY-001.",
    "subscription_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "region": "eastus2",
    "max_iterations": 2
  }
}
```

**Successful response (gate passed, 1 iteration):**
```json
{
  "spec": { "specVersion": "1.0", "description": "...", "... (TopologySpec) ..." },
  "plan": {
    "specHash": "3a7f9c2e1b4d8f0e5a6c3b2d1e9f7a4b3c8d2e5f1a6b9c0d4e7f2a3b8c1d6e9",
    "registrySnapshotSha": "abc123def456",
    "fileNames": ["main.tf", "nsg.tf", "routes.tf", "peering.tf", "firewall.tf", "gateway.tf", "versions.tf"],
    "resourceCount": 34
  },
  "findings": [],
  "prUrl": "https://github.com/att-network/att-infra-terraform/pull/842",
  "iterations": 1,
  "gatePass": true
}
```

**Failed response (gate failed after 2 iterations):**
```json
{
  "spec": { "specVersion": "1.0", "... last attempted spec ..." },
  "findings": [
    {
      "type": "over-permissive NSG (reachable)",
      "severity": "Critical",
      "resource": "synthetic-nic-snet-data",
      "evidence": "Internet:443 inbound + route 0.0.0.0/0->Internet + public IP 20.10.10.50",
      "reachable": true
    }
  ],
  "prUrl": "",
  "iterations": 2,
  "gatePass": false,
  "error": "gate failed after 2 iterations: 1 blocking finding(s). Revise intent to specify that the data tier should not have internet-facing NSG intents and should route through firewall."
}
```

---

## Appendix A — `[VERIFY]` Item Registry

All items requiring AT&T team confirmation before implementation:

| # | Section | Item | Owner |
|---|---|---|---|
| V-01 | §2.2 | AT&T internal module registry source path `artifactory.att.com/tf-modules/att-vnet/azurerm` and pinned version `2.4.1` | AT&T network platform team |
| V-02 | §2.2 | JFrog Artifactory Terraform registry URL format (provider_installation block) | AT&T DevOps / Artifactory admin |
| V-03 | §2.4 | Module version pinning update process — PR target in infrastructure repository | AT&T network platform team |
| V-04 | §5.2 | Infrastructure Terraform repository name (`vars.INFRA_REPO`) | AT&T network platform team |
| V-05 | §5.2 | GitHub App token vs PAT for infrastructure repo access (`secrets.GH_APP_TOKEN`) | AT&T GitHub admin |
| V-06 | §5.2 | JFrog token secret name (`secrets.JFROG_TOKEN`) and whether a Terraform-specific token is needed | AT&T Artifactory admin |
| V-07 | §5.5 | GitHub team slug for required reviewers (`att-network-approvers`) | AT&T GitHub admin |
| V-08 | §5.5 | Branch protection configuration in infrastructure repository | AT&T GitHub admin |
| V-09 | §5.5 | Label-based merge block policy for `needs-review` label | AT&T GitHub admin |
| V-10 | §6.3 | Which team manages Managed Identity role assignments | AT&T IAM team |
| V-11 | §6.5 | AskAT&T endpoint URL and client-credentials token URL | AT&T AI platform team |
| V-12 | §6.5 | AskAT&T structured output / function calling API contract (JSON Schema `strict: true` equivalent) | AT&T AI platform team |
| V-13 | §6.7 | Azure Monitor log ingestion workspace ID for audit logs | AT&T security / observability team |
| V-14 | §1.2 | AT&T mandatory tag policy — confirm required tag keys beyond `env`, `owner`, `costcenter`, `appid` | AT&T cloud governance team |
| V-15 | §5.6 | Post-merge apply pipeline ownership and trigger mechanism | AT&T network platform team |

## 9. Rubber-Duck Review Findings

### GR-001
- **Severity:** High
- **Section cited:** §3.1, §3.5
- **Risk:** The original fixture projection dropped `Fixture.AVNM.SecurityAdminRules` even though `Analyze()` Gate 1 reads them. That would let the generation gate ignore subscription-baseline AVNM `Deny` / `AlwaysAllow` posture and return the wrong approval result.
- **Fix applied:** Added `ProjectionBaseline` to `RenderTerraform` / `ProjectFixture` and documented that `AVNM.SecurityAdminRules` are copied from the fetched subscription baseline into `FixtureProjection`.

### GR-002
- **Severity:** High
- **Section cited:** §3.5
- **Risk:** The original projection set every synthetic NIC `PublicIP=nil`, which means `Analyze()` could never satisfy Gate 4 for direct internet-exposed synthetic workloads. Unsafe web-tier topologies would degrade to latent or non-blocking findings instead of blocking PR creation.
- **Fix applied:** Updated projection rules so direct-ingress synthetic NICs receive deterministic synthetic public IPs and matching `ResourceGraph.PublicIPAddresses` entries; also corrected the worked failure example to match the real engine evidence.

### GR-003
- **Severity:** Medium
- **Section cited:** §3.3, §3.5
- **Risk:** The NSG intent vocabulary said "Internet" at the intent level but did not require fixture projection to emit `SecRule.SourceAddressPrefix="Internet"` / `"0.0.0.0/0"`. If a module-local alias or empty source leaked into `EffectiveSecurityRules`, Gate 2 would silently miss internet ingress.
- **Fix applied:** Added an explicit projection invariant requiring canonical Internet source prefixes in both declared NSG rules and `NetworkWatcher.EffectiveSecurityRules`.

### GR-004
- **Severity:** High
- **Section cited:** §1.2, §2.3, §2.4, §3.5, §4.3
- **Risk:** The design selected a private-endpoint module and treated private-DNS findings as blocking, but `TopologySpec` had no way to declare a private endpoint target and the refinement prompt gave no repair path. That made the private-DNS gate both under-modelled and non-convergent.
- **Fix applied:** Added `PrivateEndpointSpec`, wired it into the schema/module mapping/fixture projection, and expanded the refinement prompt so DNS-related blocking findings can be corrected by changing generator-owned fields.

### GR-005
- **Severity:** Medium
- **Section cited:** §4.1, §4.5, §6.2, §8.4
- **Risk:** The anti-bypass design originally relied on threading `approved bool` into `CreatePR`. That is not structural protection: a future caller could fabricate or stale-cache `true` without a trustworthy validation object.
- **Fix applied:** Replaced the naked approval flag in the design with a `ValidationResult` flow from `ValidateBeforeEmit` into `CreatePR`, and updated the handler pseudocode accordingly.

### GR-006
- **Severity:** Medium
- **Section cited:** §7.1, §7.3
- **Risk:** Placeholder reconciliation overstated two closures: `AddNICOp` closes SR-002 only when paired with workload creation, and the east-west rule text did not actually require reading `VNet.Peerings[]`, the exact unread field called out by SR-003.
- **Fix applied:** Clarified the SR-002 workflow constraint (`AddSubnet` + `AddNIC`) and updated `checkLateralMovement` to explicitly cover same-VNet or peered-VNet paths via `VNet.Peerings[]`.
