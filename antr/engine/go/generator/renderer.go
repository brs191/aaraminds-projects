package generator

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/netip"
	"sort"
	"strings"

	"github.com/aaraminds/azure-nettopo-engine/internal/graph"
)

// ProjectionBaseline carries subscription-owned AVNM state into the fixture projection.
// It is intentionally narrow: only fields that Analyze() reads and that are not owned
// by TopologySpec are carried in.
type ProjectionBaseline struct {
	AVNMSecurityAdminRules []graph.AdminRule
}

// TerraformPlan is the renderer output.
type TerraformPlan struct {
	// Files maps HCL filename to HCL content.
	Files map[string]string

	// SpecHash is the SHA-256 of canonical JSON of the TopologySpec.
	SpecHash string

	// RegistrySnapshotSHA is reserved — empty string in Phase 3.
	RegistrySnapshotSHA string

	// FixtureProjection is the graph.Fixture for ValidateBeforeEmit → Analyze().
	FixtureProjection *graph.Fixture
}

// ErrUnknownNSGIntent is returned when a spec contains an intent outside the approved vocabulary.
type ErrUnknownNSGIntent struct{ Intent string }

func (e ErrUnknownNSGIntent) Error() string {
	return fmt.Sprintf("unknown NSG intent %q: not in approved vocabulary", e.Intent)
}

// RenderTerraform translates a validated TopologySpec into a TerraformPlan.
// Pure function — no I/O, no randomness, no timestamps, no external calls.
func RenderTerraform(spec TopologySpec, registry ModuleRegistry, baseline ProjectionBaseline) (TerraformPlan, error) {
	// Step 1: Structural validation
	if err := spec.Validate(); err != nil {
		return TerraformPlan{}, fmt.Errorf("spec validation failed: %w", err)
	}

	// Step 2: Validate all NSGIntents against the 16-value vocabulary
	for _, v := range spec.VNets {
		for _, sn := range v.Subnets {
			for _, intent := range sn.NSGIntents {
				if _, err := expandIntent(intent); err != nil {
					return TerraformPlan{}, err // already ErrUnknownNSGIntent
				}
			}
		}
	}

	// Step 3: CIDR overlap check across peered VNets
	if spec.PeeringTopology != "none" {
		if err := checkCIDROverlap(spec); err != nil {
			return TerraformPlan{}, err
		}
	}

	// Step 4: firewallEnabled requires AzureFirewallSubnet
	if spec.FirewallEnabled {
		found := false
	outer:
		for _, v := range spec.VNets {
			for _, sn := range v.Subnets {
				if sn.Name == "AzureFirewallSubnet" {
					found = true
					break outer
				}
			}
		}
		if !found {
			return TerraformPlan{}, fmt.Errorf("firewallEnabled=true but no AzureFirewallSubnet found in any VNet")
		}
	}

	// Check: routeToFirewall requires firewallEnabled
	for _, v := range spec.VNets {
		for _, sn := range v.Subnets {
			if sn.RouteToFirewall && !spec.FirewallEnabled {
				return TerraformPlan{}, fmt.Errorf("VNet %q subnet %q: routeToFirewall=true requires firewallEnabled=true (NVA IP unknown without firewall)", v.Name, sn.Name)
			}
		}
	}

	// Step 5: Compute SpecHash = SHA-256(canonical JSON with sorted map keys)
	specHash, err := computeSpecHash(spec)
	if err != nil {
		return TerraformPlan{}, fmt.Errorf("computing spec hash: %w", err)
	}

	// Step 6: Generate HCL files
	files, err := generateHCL(spec, registry, specHash)
	if err != nil {
		return TerraformPlan{}, fmt.Errorf("generating HCL: %w", err)
	}

	// Step 7: Build FixtureProjection
	fixture := ProjectFixture(spec, baseline)

	return TerraformPlan{
		Files:               files,
		SpecHash:            specHash,
		RegistrySnapshotSHA: "", // reserved
		FixtureProjection:   fixture,
	}, nil
}

// checkCIDROverlap returns an error if any two VNets in the spec have overlapping address spaces.
func checkCIDROverlap(spec TopologySpec) error {
	for i := 0; i < len(spec.VNets); i++ {
		for j := i + 1; j < len(spec.VNets); j++ {
			vi, vj := spec.VNets[i], spec.VNets[j]
			for _, pa := range vi.AddressSpace {
				for _, pb := range vj.AddressSpace {
					if prefixOverlap(pa, pb) {
						return fmt.Errorf("CIDR overlap detected between VNet %q (%s) and VNet %q (%s); peered VNets must have non-overlapping address spaces",
							vi.Name, pa, vj.Name, pb)
					}
				}
			}
		}
	}
	return nil
}

// prefixOverlap reports whether two CIDR strings overlap.
func prefixOverlap(a, b string) bool {
	pa, err1 := netip.ParsePrefix(a)
	pb, err2 := netip.ParsePrefix(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return pa.Overlaps(pb)
}

// computeSpecHash returns the hex-encoded SHA-256 of the canonical JSON of spec.
// encoding/json marshals map[string]string with sorted keys, so this is deterministic.
func computeSpecHash(spec TopologySpec) (string, error) {
	data, err := json.Marshal(spec)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum), nil
}

// ─── HCL generation ──────────────────────────────────────────────────────────

// toIdentifier converts a resource name (possibly containing hyphens/dots) to a
// valid Terraform identifier (underscores only).
func toIdentifier(s string) string {
	r := strings.NewReplacer("-", "_", ".", "_", " ", "_", "/", "_")
	return r.Replace(s)
}

// hclTagsBlock renders a map[string]string as an HCL tags block with sorted keys.
func hclTagsBlock(tags map[string]string, indent string) string {
	if len(tags) == 0 {
		return "{}"
	}
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	sb.WriteString("{\n")
	for _, k := range keys {
		sb.WriteString(fmt.Sprintf("%s  %s = %q\n", indent, k, tags[k]))
	}
	sb.WriteString(indent + "  }")
	return sb.String()
}

// generateHCL generates all HCL files for the TerraformPlan.
// All module sources come from registry — never from TopologySpec fields.
func generateHCL(spec TopologySpec, registry ModuleRegistry, specHash string) (map[string]string, error) {
	files := make(map[string]string)

	files["versions.tf"] = genVersions(specHash)
	files["main.tf"] = genMain(spec, registry)
	files["nsg.tf"] = genNSG(spec, registry)

	if hasRouteToFirewall(spec) {
		files["routes.tf"] = genRoutes(spec, registry)
	}
	if spec.PeeringTopology != "none" {
		files["peering.tf"] = genPeering(spec, registry)
	}
	if spec.GatewayType != "none" {
		files["gateway.tf"] = genGateway(spec, registry)
	}
	if spec.FirewallEnabled {
		files["firewall.tf"] = genFirewall(spec, registry)
	}

	return files, nil
}

// hasRouteToFirewall reports whether any subnet in the spec has RouteToFirewall=true.
func hasRouteToFirewall(spec TopologySpec) bool {
	for _, v := range spec.VNets {
		for _, sn := range v.Subnets {
			if sn.RouteToFirewall {
				return true
			}
		}
	}
	return false
}

func genVersions(specHash string) string {
	return fmt.Sprintf(`# Generated by azure-nettopo-engine Phase 3 renderer
# SpecHash: %s
# DO NOT EDIT — regenerate by re-running generate_topology

terraform {
  required_version = ">= 1.5"
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
    }
  }
}
`, specHash)
}

func genMain(spec TopologySpec, registry ModuleRegistry) string {
	vnetMod, _ := registry.Select("vnet")

	var sb strings.Builder
	sb.WriteString("# VNet modules\n\n")

	// Sort VNets for determinism (they're already in spec order, but we iterate by index)
	for _, v := range spec.VNets {
		addrSpaceJSON, _ := json.Marshal(v.AddressSpace)
		tagsStr := hclTagsBlock(spec.Tags, "  ")
		id := toIdentifier(v.Name)
		sb.WriteString(fmt.Sprintf(`module "vnet_%s" {
  source  = %q
  version = %q

  vnet_name     = %q
  address_space = %s
  location      = %q
  tags          = %s
}

`, id, vnetMod.Source, vnetMod.Version,
			v.Name, string(addrSpaceJSON), spec.Region, tagsStr))

		// Subnets module
		subnetMod, _ := registry.Select("subnets")
		subnetNames := make([]string, len(v.Subnets))
		subnetPrefixes := make([]string, len(v.Subnets))
		for i, sn := range v.Subnets {
			subnetNames[i] = sn.Name
			subnetPrefixes[i] = sn.AddressPrefix
		}
		namesJSON, _ := json.Marshal(subnetNames)
		prefixesJSON, _ := json.Marshal(subnetPrefixes)
		sb.WriteString(fmt.Sprintf(`module "subnets_%s" {
  source  = %q
  version = %q

  resource_group_name  = var.resource_group_name
  virtual_network_name = module.vnet_%s.vnet_name
  subnet_names         = %s
  subnet_prefixes      = %s
  location             = %q

  depends_on = [module.vnet_%s]
}

`, id, subnetMod.Source, subnetMod.Version,
			id, string(namesJSON), string(prefixesJSON), spec.Region, id))
	}

	return sb.String()
}

func genNSG(spec TopologySpec, registry ModuleRegistry) string {
	nsgMod, _ := registry.Select("nsg")
	tagsStr := hclTagsBlock(spec.Tags, "  ")

	var sb strings.Builder
	sb.WriteString("# NSG modules — one per VNet\n\n")

	for _, v := range spec.VNets {
		id := toIdentifier(v.Name)
		nsgName := fmt.Sprintf("nsg-%s", v.Name)

		// Collect all rules across subnets in this VNet, de-duplicate by name
		rulesSeen := make(map[string]bool)
		var rules []graph.SecRule
		for _, sn := range v.Subnets {
			for _, intent := range sn.NSGIntents {
				expanded, err := expandIntent(intent)
				if err != nil {
					continue // already validated upstream
				}
				for _, r := range expanded {
					if !rulesSeen[r.Name] {
						rulesSeen[r.Name] = true
						rules = append(rules, r)
					}
				}
			}
		}

		// Sort rules by priority for determinism
		sort.Slice(rules, func(i, j int) bool {
			if rules[i].Priority != rules[j].Priority {
				return rules[i].Priority < rules[j].Priority
			}
			return rules[i].Name < rules[j].Name
		})

		// Build custom_rules block
		var rulesBlock strings.Builder
		rulesBlock.WriteString("[\n")
		for _, r := range rules {
			rulesBlock.WriteString(fmt.Sprintf(`    {
      name                       = %q
      priority                   = %d
      direction                  = %q
      access                     = %q
      protocol                   = %q
      source_address_prefix      = %q
      destination_port_range     = %q
    },
`, r.Name, r.Priority, r.Direction, r.Access, r.Protocol,
				r.SourceAddressPrefix, r.DestinationPortRange))
		}
		rulesBlock.WriteString("  ]")

		sb.WriteString(fmt.Sprintf(`module "nsg_%s" {
  source  = %q
  version = %q

  resource_group_name = var.resource_group_name
  security_group_name = %q
  location            = %q
  custom_rules        = %s
  tags                = %s

  depends_on = [module.subnets_%s]
}

`, id, nsgMod.Source, nsgMod.Version,
			nsgName, spec.Region, rulesBlock.String(), tagsStr, id))
	}
	return sb.String()
}

func genRoutes(spec TopologySpec, registry ModuleRegistry) string {
	rtMod, _ := registry.Select("route-table")
	tagsStr := hclTagsBlock(spec.Tags, "  ")

	var sb strings.Builder
	sb.WriteString("# Route table modules — one per VNet with routeToFirewall subnets\n\n")

	for _, v := range spec.VNets {
		// Only emit a route table if at least one subnet routes to firewall
		hasRT := false
		for _, sn := range v.Subnets {
			if sn.RouteToFirewall {
				hasRT = true
				break
			}
		}
		if !hasRT {
			continue
		}

		id := toIdentifier(v.Name)
		rtName := fmt.Sprintf("rt-%s", v.Name)

		sb.WriteString(fmt.Sprintf(`module "rt_%s" {
  source  = %q
  version = %q

  resource_group_name = var.resource_group_name
  location            = %q
  name                = %q

  routes = [
    {
      name                   = "default-route"
      address_prefix         = "0.0.0.0/0"
      next_hop_type          = "VirtualAppliance"
      next_hop_in_ip_address = "10.0.0.4"
    }
  ]

  subnet_ids = [
`, id, rtMod.Source, rtMod.Version, spec.Region, rtName))

		for _, sn := range v.Subnets {
			if sn.RouteToFirewall {
				sb.WriteString(fmt.Sprintf("    module.subnets_%s.subnet_ids[%q],\n", id, sn.Name))
			}
		}
		sb.WriteString(fmt.Sprintf(`  ]
  tags = %s
}

`, tagsStr))
	}
	return sb.String()
}

func genPeering(spec TopologySpec, registry ModuleRegistry) string {
	var sb strings.Builder
	sb.WriteString("# Peering topology: " + spec.PeeringTopology + "\n\n")

	switch spec.PeeringTopology {
	case "hub-spoke":
		hubSpokeMod, _ := registry.Select("hub-spoke-peering")
		// Collect spoke VNet IDs
		var spokeIDs []string
		for _, v := range spec.VNets {
			if v.Name != spec.HubVNetName {
				spokeIDs = append(spokeIDs, fmt.Sprintf(`    module.vnet_%s.vnet_id,`, toIdentifier(v.Name)))
			}
		}
		hubID := toIdentifier(spec.HubVNetName)
		sb.WriteString(fmt.Sprintf(`module "hub_spoke_peering" {
  source  = %q
  version = %q

  hub_virtual_network_resource_id = module.vnet_%s.vnet_id

  virtual_network_resource_ids_to_peer_to_hub = [
%s
  ]

  depends_on = [
    module.vnet_%s,
  ]
}

`, hubSpokeMod.Source, hubSpokeMod.Version, hubID,
			strings.Join(spokeIDs, "\n"), hubID))

	case "mesh":
		// Direct azurerm_virtual_network_peering resources (no dedicated module per §2.3)
		vnets := spec.VNets
		for i := 0; i < len(vnets); i++ {
			for j := 0; j < len(vnets); j++ {
				if i == j {
					continue
				}
				local := vnets[i]
				remote := vnets[j]
				peerID := fmt.Sprintf("peer_%s_to_%s", toIdentifier(local.Name), toIdentifier(remote.Name))
				sb.WriteString(fmt.Sprintf(`resource "azurerm_virtual_network_peering" "%s" {
  name                      = "peer-%s-to-%s"
  resource_group_name       = var.resource_group_name
  virtual_network_name      = %q
  remote_virtual_network_id = module.vnet_%s.vnet_id
  allow_forwarded_traffic   = true
  allow_gateway_transit     = false
  use_remote_gateways       = false

  depends_on = [module.vnet_%s, module.vnet_%s]
}

`, peerID, local.Name, remote.Name, local.Name, toIdentifier(remote.Name),
					toIdentifier(local.Name), toIdentifier(remote.Name)))
			}
		}

	case "custom":
		for _, pp := range spec.PeeringPairs {
			peerID := fmt.Sprintf("peer_%s_to_%s", toIdentifier(pp.LocalVNet), toIdentifier(pp.RemoteVNet))
			sb.WriteString(fmt.Sprintf(`resource "azurerm_virtual_network_peering" "%s" {
  name                      = "peer-%s-to-%s"
  resource_group_name       = var.resource_group_name
  virtual_network_name      = %q
  remote_virtual_network_id = module.vnet_%s.vnet_id
  allow_forwarded_traffic   = %v
  allow_gateway_transit     = %v
  use_remote_gateways       = %v

  depends_on = [module.vnet_%s, module.vnet_%s]
}

`, peerID, pp.LocalVNet, pp.RemoteVNet, pp.LocalVNet, toIdentifier(pp.RemoteVNet),
				pp.AllowForwardedTraffic, pp.AllowGatewayTransit, pp.UseRemoteGateways,
				toIdentifier(pp.LocalVNet), toIdentifier(pp.RemoteVNet)))
		}
	}
	return sb.String()
}

func genGateway(spec TopologySpec, registry ModuleRegistry) string {
	var sb strings.Builder
	tagsStr := hclTagsBlock(spec.Tags, "  ")

	// Find hub VNet for gateway placement
	hubVNetName := spec.HubVNetName
	if hubVNetName == "" {
		// Fall back to first VNet
		if len(spec.VNets) > 0 {
			hubVNetName = spec.VNets[0].Name
		}
	}
	hubID := toIdentifier(hubVNetName)

	switch spec.GatewayType {
	case "vpn":
		vpnMod, _ := registry.Select("vpn-gateway")
		sb.WriteString(fmt.Sprintf(`module "vpn_gateway" {
  source  = %q
  version = %q

  resource_group_name = var.resource_group_name
  location            = %q
  subnet_id           = module.subnets_%s.subnet_ids["GatewaySubnet"]
  tags                = %s

  depends_on = [module.subnets_%s]
}
`, vpnMod.Source, vpnMod.Version, spec.Region, hubID, tagsStr, hubID))

	case "expressroute":
		erMod, _ := registry.Select("expressroute-gateway")
		sb.WriteString(fmt.Sprintf(`module "expressroute_gateway" {
  source  = %q
  version = %q

  resource_group_name = var.resource_group_name
  location            = %q
  subnet_id           = module.subnets_%s.subnet_ids["GatewaySubnet"]
  tags                = %s

  depends_on = [module.subnets_%s]
}
`, erMod.Source, erMod.Version, spec.Region, hubID, tagsStr, hubID))
	}
	return sb.String()
}

func genFirewall(spec TopologySpec, registry ModuleRegistry) string {
	fwMod, _ := registry.Select("firewall")
	tagsStr := hclTagsBlock(spec.Tags, "  ")

	// Find hub VNet for firewall placement
	hubVNetName := spec.HubVNetName
	if hubVNetName == "" && len(spec.VNets) > 0 {
		hubVNetName = spec.VNets[0].Name
	}
	hubID := toIdentifier(hubVNetName)

	fwBlocks := fmt.Sprintf(`module "firewall" {
  source  = %q
  version = %q

  resource_group_name = var.resource_group_name
  location            = %q
  virtual_network_id  = module.vnet_%s.vnet_id
  sku_tier            = "Standard"
  tags                = %s

  depends_on = [module.subnets_%s]
}
`, fwMod.Source, fwMod.Version, spec.Region, hubID, tagsStr, hubID)

	// Add bastion if needed
	var bastionBlock string
	if containsLabel(spec.TierLabels, "bastion") {
		bastionMod, _ := registry.Select("bastion")
		bastionBlock = fmt.Sprintf(`
module "bastion" {
  source  = %q
  version = %q

  resource_group_name  = var.resource_group_name
  location             = %q
  virtual_network_name = %q
  tags                 = %s

  depends_on = [module.subnets_%s]
}
`, bastionMod.Source, bastionMod.Version, spec.Region, hubVNetName, tagsStr, hubID)
	}

	// Add AppGW if needed
	var appgwBlock string
	if containsLabel(spec.TierLabels, "appgw") {
		appgwMod, _ := registry.Select("appgw-waf")
		appgwBlock = fmt.Sprintf(`
module "appgw_waf" {
  source  = %q
  version = %q

  resource_group_name = var.resource_group_name
  location            = %q
  sku_name            = "WAF_v2"
  tags                = %s
  waf_configuration = {
    enabled        = true
    firewall_mode  = "Prevention"
    rule_set_type  = "OWASP"
    rule_set_version = "3.2"
  }
}
`, appgwMod.Source, appgwMod.Version, spec.Region, tagsStr)
	}

	return fwBlocks + bastionBlock + appgwBlock
}

// containsLabel reports whether the label is present in the tier labels slice.
func containsLabel(labels []string, label string) bool {
	for _, l := range labels {
		if l == label {
			return true
		}
	}
	return false
}
