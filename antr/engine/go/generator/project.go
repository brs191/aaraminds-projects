package generator

import (
	"fmt"
	"net/netip"
	"sort"
	"strings"

	"github.com/aaraminds/azure-nettopo-engine/internal/graph"
)

// peGroupIdToZone maps a Private Endpoint groupId to the canonical Azure Private
// DNS zone name. This is a replica of the same map in internal/analyze/analyze.go
// kept here so ProjectFixture has no dependency on unexported analyze internals.
// Both maps MUST stay in sync with the engine's zone resolution logic.
var peGroupIdToZone = map[string]string{
	"blob":            "privatelink.blob.core.windows.net",
	"file":            "privatelink.file.core.windows.net",
	"queue":           "privatelink.queue.core.windows.net",
	"table":           "privatelink.table.core.windows.net",
	"dfs":             "privatelink.dfs.core.windows.net",
	"web":             "privatelink.web.core.windows.net",
	"vault":           "privatelink.vaultcore.azure.net",
	"sql":             "privatelink.database.windows.net",
	"sqlOnDemand":     "privatelink.sql.azuresynapse.net",
	"registry":        "privatelink.azurecr.io",
	"sites":           "privatelink.azurewebsites.net",
	"namespace":       "privatelink.servicebus.windows.net",
	"managedInstance": "privatelink.database.windows.net",
	"searchService":   "privatelink.search.windows.net",
	"azurecosmosdb":   "privatelink.documents.azure.com",
	"redisCache":      "privatelink.redis.cache.windows.net",
	"openai":          "privatelink.openai.azure.com",
	"account":         "privatelink.purview.azure.com",
}

// nsgVocabulary is the closed set of accepted NSGIntent strings (16 values including aliases).
var nsgVocabulary = map[string]bool{
	"allow-https-from-internet":  true,
	"allow-http-from-internet":   true,
	"allow-ssh-from-bastion":     true,
	"allow-rdp-from-bastion":     true,
	"allow-bastion-rdp-ssh":      true,
	"allow-internal-vnet":        true,
	"allow-app-tier-only":        true,
	"allow-appgw-management":     true,
	"allow-lb-probes":            true,
	"deny-internet-inbound":      true,
	"deny-all-inbound-other":     true,
	"deny-all-inbound":           true, // alias for deny-all-inbound-other
	"deny-all-outbound-internet": true,
	"allow-azure-monitor":        true,
	"allow-key-vault":            true,
	"allow-storage":              true,
}

// expandIntent expands one NSGIntent string into one or more graph.SecRules.
// Returns ErrUnknownNSGIntent if the intent is not in the approved vocabulary.
// This function is the single source of truth for the intent → rule translation.
func expandIntent(intent string) ([]graph.SecRule, error) {
	switch intent {
	case "allow-https-from-internet":
		return []graph.SecRule{{
			Name: "allow-https-from-internet", Priority: 100,
			Direction: "Inbound", Access: "Allow", Protocol: "TCP",
			SourceAddressPrefix: "Internet", DestinationPortRange: "443",
		}}, nil

	case "allow-http-from-internet":
		return []graph.SecRule{{
			Name: "allow-http-from-internet", Priority: 110,
			Direction: "Inbound", Access: "Allow", Protocol: "TCP",
			SourceAddressPrefix: "Internet", DestinationPortRange: "80",
		}}, nil

	case "allow-ssh-from-bastion":
		return []graph.SecRule{{
			Name: "allow-ssh-from-bastion", Priority: 200,
			Direction: "Inbound", Access: "Allow", Protocol: "TCP",
			SourceAddressPrefix: "VirtualNetwork", DestinationPortRange: "22",
		}}, nil

	case "allow-rdp-from-bastion":
		return []graph.SecRule{{
			Name: "allow-rdp-from-bastion", Priority: 210,
			Direction: "Inbound", Access: "Allow", Protocol: "TCP",
			SourceAddressPrefix: "VirtualNetwork", DestinationPortRange: "3389",
		}}, nil

	case "allow-bastion-rdp-ssh":
		// Expands to TWO rules: SSH + RDP (same as allow-ssh-from-bastion + allow-rdp-from-bastion)
		return []graph.SecRule{
			{
				Name: "allow-ssh-from-bastion", Priority: 200,
				Direction: "Inbound", Access: "Allow", Protocol: "TCP",
				SourceAddressPrefix: "VirtualNetwork", DestinationPortRange: "22",
			},
			{
				Name: "allow-rdp-from-bastion", Priority: 210,
				Direction: "Inbound", Access: "Allow", Protocol: "TCP",
				SourceAddressPrefix: "VirtualNetwork", DestinationPortRange: "3389",
			},
		}, nil

	case "allow-internal-vnet":
		return []graph.SecRule{{
			Name: "allow-internal-vnet", Priority: 300,
			Direction: "Inbound", Access: "Allow", Protocol: "*",
			SourceAddressPrefix: "VirtualNetwork", DestinationPortRange: "*",
		}}, nil

	case "allow-app-tier-only":
		return []graph.SecRule{{
			Name: "allow-app-tier-only", Priority: 300,
			Direction: "Inbound", Access: "Allow", Protocol: "TCP",
			SourceAddressPrefix: "VirtualNetwork", DestinationPortRange: "*",
		}}, nil

	case "allow-appgw-management":
		return []graph.SecRule{{
			Name: "allow-appgw-management", Priority: 100,
			Direction: "Inbound", Access: "Allow", Protocol: "TCP",
			SourceAddressPrefix: "GatewayManager", DestinationPortRange: "65200-65535",
		}}, nil

	case "allow-lb-probes":
		return []graph.SecRule{{
			Name: "allow-lb-probes", Priority: 400,
			Direction: "Inbound", Access: "Allow", Protocol: "TCP",
			SourceAddressPrefix: "AzureLoadBalancer", DestinationPortRange: "*",
		}}, nil

	case "deny-internet-inbound":
		return []graph.SecRule{{
			Name: "deny-internet-inbound", Priority: 1000,
			Direction: "Inbound", Access: "Deny", Protocol: "*",
			SourceAddressPrefix: "Internet", DestinationPortRange: "*",
		}}, nil

	case "deny-all-inbound-other", "deny-all-inbound":
		// deny-all-inbound is an alias for deny-all-inbound-other
		return []graph.SecRule{{
			Name: "deny-all-inbound-other", Priority: 4096,
			Direction: "Inbound", Access: "Deny", Protocol: "*",
			SourceAddressPrefix: "*", DestinationPortRange: "*",
		}}, nil

	case "deny-all-outbound-internet":
		return []graph.SecRule{{
			Name: "deny-all-outbound-internet", Priority: 1000,
			Direction: "Outbound", Access: "Deny", Protocol: "*",
			SourceAddressPrefix: "*", DestinationPortRange: "*",
		}}, nil

	case "allow-azure-monitor":
		return []graph.SecRule{{
			Name: "allow-azure-monitor", Priority: 200,
			Direction: "Outbound", Access: "Allow", Protocol: "TCP",
			SourceAddressPrefix: "VirtualNetwork", DestinationPortRange: "443",
		}}, nil

	case "allow-key-vault":
		return []graph.SecRule{{
			Name: "allow-key-vault", Priority: 210,
			Direction: "Outbound", Access: "Allow", Protocol: "TCP",
			SourceAddressPrefix: "AzureKeyVault", DestinationPortRange: "443",
		}}, nil

	case "allow-storage":
		return []graph.SecRule{{
			Name: "allow-storage", Priority: 220,
			Direction: "Outbound", Access: "Allow", Protocol: "TCP",
			SourceAddressPrefix: "Storage", DestinationPortRange: "443",
		}}, nil

	default:
		return nil, ErrUnknownNSGIntent{Intent: intent}
	}
}

// syntheticPrivateIP returns the base address of the CIDR block plus offset 5.
// For "10.0.1.0/24" → "10.0.1.5". Deterministic.
func syntheticPrivateIP(cidr string) string {
	prefix, err := netip.ParsePrefix(cidr)
	if err != nil {
		return "10.0.0.5"
	}
	addr := prefix.Addr()
	if !addr.Is4() {
		return "10.0.0.5"
	}
	b := addr.As4()
	v := uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
	v += 5
	b[0] = byte(v >> 24)
	b[1] = byte(v >> 16)
	b[2] = byte(v >> 8)
	b[3] = byte(v)
	return netip.AddrFrom4(b).String()
}

// tiersThatGetSyntheticNICs is the set of tier labels that always get a synthetic NIC.
var tiersThatGetSyntheticNICs = map[string]bool{
	"web":  true,
	"app":  true,
	"data": true,
	"dmz":  true,
}

// internetIngressIntents is the set of allow intents that indicate direct internet ingress.
var internetIngressIntents = map[string]bool{
	"allow-https-from-internet": true,
	"allow-http-from-internet":  true,
}

// subnetHasInternetIngress reports whether the subnet has any direct internet ingress intent.
func subnetHasInternetIngress(sn SubnetSpec) bool {
	for _, intent := range sn.NSGIntents {
		if internetIngressIntents[intent] {
			return true
		}
	}
	return false
}

// syntheticPIPName returns a synthetic public IP name for the given NIC, or "" if none needed.
// A NIC gets a synthetic PIP if: has internet ingress intent AND routeToFirewall=false.
// This implements GR-002: internet-facing synthetic NICs MUST get a synthetic PublicIP.
func syntheticPIPName(vnetName, subnetName string, sn SubnetSpec) string {
	if subnetHasInternetIngress(sn) && !sn.RouteToFirewall {
		return fmt.Sprintf("pip-synthetic-nic-%s-%s", vnetName, subnetName)
	}
	return ""
}

// ProjectFixture derives a graph.Fixture from TopologySpec + baseline.
// Used EXCLUSIVELY by ValidateBeforeEmit → Analyze().
// NOT used for live Azure API queries.
func ProjectFixture(spec TopologySpec, baseline ProjectionBaseline) *graph.Fixture {
	fixture := &graph.Fixture{
		Subscription: "synthetic",
	}

	// ─── Rule 1 + 2: VNets and Subnets ────────────────────────────────────────
	var gVNets []graph.VNet
	for _, v := range spec.VNets {
		var gSubnets []graph.Subnet
		for _, sn := range v.Subnets {
			gs := graph.Subnet{
				Name:                 sn.Name,
				AddressPrefix:        sn.AddressPrefix,
				NetworkSecurityGroup: fmt.Sprintf("nsg-%s", v.Name),
				ServiceEndpoints:     sn.ServiceEndpoints,
				Delegations:          sn.Delegations,
			}
			if sn.RouteToFirewall {
				gs.RouteTable = fmt.Sprintf("rt-%s", v.Name)
			}
			if sn.PrivateEndpointSubnet {
				gs.PrivateEndpointNetworkPolicies = "Disabled"
			}
			gSubnets = append(gSubnets, gs)
		}

		gv := graph.VNet{
			Name:         v.Name,
			AddressSpace: v.AddressSpace,
			Subnets:      gSubnets,
		}
		gVNets = append(gVNets, gv)
	}

	// ─── Rule 11: Peerings ────────────────────────────────────────────────────
	// Populate after initial VNet list is built
	vnetByName := make(map[string]*graph.VNet, len(gVNets))
	for i := range gVNets {
		vnetByName[gVNets[i].Name] = &gVNets[i]
	}
	projectPeerings(spec, vnetByName)

	fixture.ResourceGraph.VirtualNetworks = gVNets

	// ─── Rule 3: NSGs ─────────────────────────────────────────────────────────
	var gNSGs []graph.NSG
	for _, v := range spec.VNets {
		rulesSeen := make(map[string]bool)
		var allRules []graph.SecRule
		var assocSubnets []string
		for _, sn := range v.Subnets {
			assocSubnets = append(assocSubnets, fmt.Sprintf("%s/%s", v.Name, sn.Name))
			for _, intent := range sn.NSGIntents {
				expanded, err := expandIntent(intent)
				if err != nil {
					continue
				}
				for _, r := range expanded {
					if !rulesSeen[r.Name] {
						rulesSeen[r.Name] = true
						allRules = append(allRules, r)
					}
				}
			}
		}
		// Sort by priority for determinism
		sort.Slice(allRules, func(i, j int) bool {
			if allRules[i].Priority != allRules[j].Priority {
				return allRules[i].Priority < allRules[j].Priority
			}
			return allRules[i].Name < allRules[j].Name
		})
		gNSGs = append(gNSGs, graph.NSG{
			Name:              fmt.Sprintf("nsg-%s", v.Name),
			SecurityRules:     allRules,
			AssociatedSubnets: assocSubnets,
		})
	}
	fixture.ResourceGraph.NetworkSecurityGroups = gNSGs

	// ─── Rule 4: RouteTables ──────────────────────────────────────────────────
	var gRTs []graph.RouteTable
	for _, v := range spec.VNets {
		hasRT := false
		var assocSubnets []string
		for _, sn := range v.Subnets {
			if sn.RouteToFirewall {
				hasRT = true
				assocSubnets = append(assocSubnets, fmt.Sprintf("%s/%s", v.Name, sn.Name))
			}
		}
		if !hasRT {
			continue
		}
		gRTs = append(gRTs, graph.RouteTable{
			Name: fmt.Sprintf("rt-%s", v.Name),
			Routes: []graph.Route{{
				Name:             "default-route",
				AddressPrefix:    "0.0.0.0/0",
				NextHopType:      "VirtualAppliance",
				NextHopIPAddress: "10.0.0.4",
			}},
			AssociatedSubnets: assocSubnets,
		})
	}
	fixture.ResourceGraph.RouteTables = gRTs

	// ─── Rules 5, 6, 7, 8: Synthetic NICs + PIPs + EffectiveRules + Routes ───
	effRules := make(map[string][]graph.SecRule)
	effRoutes := make(map[string][]graph.Route)
	var gNICs []graph.NIC
	var gPIPs []graph.PublicIP

	for _, v := range spec.VNets {
		for _, sn := range v.Subnets {
			// Determine if this subnet gets a synthetic NIC (Rule 5)
			needsNIC := sn.Sensitive || tiersThatGetSyntheticNICs[sn.TierLabel]
			if !needsNIC {
				continue
			}

			nicName := fmt.Sprintf("synthetic-nic-%s-%s", v.Name, sn.Name)
			privateIP := syntheticPrivateIP(sn.AddressPrefix)
			nsgRef := fmt.Sprintf("nsg-%s", v.Name)

			// PublicIP: set for direct internet ingress (GR-002)
			pipName := syntheticPIPName(v.Name, sn.Name, sn)
			var publicIPPtr *string
			if pipName != "" {
				publicIPPtr = strPtr(pipName)
			}

			// Tags
			tags := map[string]string{"synthetic": "true"}
			if sn.Sensitive {
				tags["sensitive"] = "true"
			}

			nic := graph.NIC{
				Name:                 nicName,
				Subnet:               fmt.Sprintf("%s/%s", v.Name, sn.Name),
				NetworkSecurityGroup: strPtr(nsgRef),
				PublicIP:             publicIPPtr,
				PrivateIP:            privateIP,
				Tags:                 tags,
			}
			gNICs = append(gNICs, nic)

			// Rule 6: PublicIPAddresses entry for each NIC that has a PIP
			if pipName != "" {
				gPIPs = append(gPIPs, graph.PublicIP{
					Name:            pipName,
					IPAddress:       "203.0.113.5", // documentation range — synthetic
					IPConfiguration: strPtr(nicName),
				})
			}

			// Rule 7: EffectiveSecurityRules — expand all NSGIntents for this subnet
			// CRITICAL (GR-003): Internet-sourced intents → SourceAddressPrefix = "Internet"
			var subnetRules []graph.SecRule
			for _, intent := range sn.NSGIntents {
				expanded, err := expandIntent(intent)
				if err != nil {
					continue
				}
				subnetRules = append(subnetRules, expanded...)
			}
			// Sort by priority for determinism
			sort.Slice(subnetRules, func(i, j int) bool {
				if subnetRules[i].Priority != subnetRules[j].Priority {
					return subnetRules[i].Priority < subnetRules[j].Priority
				}
				return subnetRules[i].Name < subnetRules[j].Name
			})
			effRules[nicName] = subnetRules

			// Rule 8: EffectiveRoutes
			var routes []graph.Route
			if sn.RouteToFirewall && spec.FirewallEnabled {
				// Route via firewall
				routes = []graph.Route{{
					Name:          "default-route",
					AddressPrefix: "0.0.0.0/0",
					NextHopType:   "VirtualAppliance",
				}}
			} else {
				// Direct internet route
				routes = []graph.Route{{
					Name:          "default-route",
					AddressPrefix: "0.0.0.0/0",
					NextHopType:   "Internet",
				}}
			}
			effRoutes[nicName] = routes
		}
	}

	fixture.ResourceGraph.NetworkInterfaces = gNICs
	fixture.ResourceGraph.PublicIPAddresses = gPIPs
	fixture.NetworkWatcher = graph.NetworkWatcher{
		EffectiveSecurityRules: effRules,
		EffectiveRoutes:        effRoutes,
	}

	// ─── Rule 9: AVNM — copy from baseline (GR-001) ──────────────────────────
	fixture.AVNM = graph.AVNM{
		SecurityAdminRules: baseline.AVNMSecurityAdminRules,
	}

	// ─── Rule 10: AzureFirewall ───────────────────────────────────────────────
	if spec.FirewallEnabled {
		fixture.AzureFirewall = &graph.Firewall{
			Name:      "synthetic-fw",
			PrivateIP: "10.0.0.4",
			PublicIP:  "1.2.3.4",
			SKUTier:   "Standard",
			NatRules:  nil,
		}
	}

	// ─── Rule 12: PrivateEndpoints (GR-004) ──────────────────────────────────
	var gPEs []graph.PrivateEndpoint
	// Also collect DNS zones: zone name → set of VNets that need the link
	zoneVNets := make(map[string]map[string]bool)

	for _, v := range spec.VNets {
		for _, sn := range v.Subnets {
			for _, pe := range sn.PrivateEndpoints {
				subnetPath := fmt.Sprintf("%s/%s", v.Name, sn.Name)
				gPEs = append(gPEs, graph.PrivateEndpoint{
					Name:                 pe.Name,
					Subnet:               subnetPath,
					PrivateIP:            syntheticPrivateIP(sn.AddressPrefix), // approximate
					GroupId:              pe.GroupID,
					PrivateLinkServiceId: pe.ServiceResourceID,
					ConnectionState:      "Approved",
				})

				// Rule 13: PrivateDnsZones — derive zone from GroupID (GR-004)
				if zoneName, ok := peGroupIdToZone[pe.GroupID]; ok {
					if zoneVNets[zoneName] == nil {
						zoneVNets[zoneName] = make(map[string]bool)
					}
					zoneVNets[zoneName][v.Name] = true
				}
			}
		}
	}
	fixture.ResourceGraph.PrivateEndpoints = gPEs

	// Build PrivateDnsZones from collected zone → VNet mappings
	var gZones []graph.PrivateDnsZone
	// Sort zone names for determinism
	zoneNames := make([]string, 0, len(zoneVNets))
	for z := range zoneVNets {
		zoneNames = append(zoneNames, z)
	}
	sort.Strings(zoneNames)
	for _, zoneName := range zoneNames {
		linkedVnets := make([]string, 0, len(zoneVNets[zoneName]))
		for vn := range zoneVNets[zoneName] {
			linkedVnets = append(linkedVnets, vn)
		}
		sort.Strings(linkedVnets)
		gZones = append(gZones, graph.PrivateDnsZone{
			Name:        zoneName,
			LinkedVnets: linkedVnets,
		})
	}
	fixture.ResourceGraph.PrivateDnsZones = gZones

	// ─── Rule 14: ApplicationGateways ────────────────────────────────────────
	if containsLabel(spec.TierLabels, "appgw") {
		fixture.ResourceGraph.ApplicationGateways = []graph.ApplicationGateway{{
			Name:       "synthetic-appgw",
			PublicIP:   "appgw-pip",
			WafEnabled: true,
			WafMode:    "Prevention",
		}}
	}

	// ─── Rule 15: AzureBastions ───────────────────────────────────────────────
	if containsLabel(spec.TierLabels, "bastion") {
		fixture.ResourceGraph.AzureBastions = []graph.AzureBastion{{
			Name: "synthetic-bastion",
		}}
	}

	return fixture
}

// projectPeerings populates the Peerings slice on each VNet pointer in vnetByName
// based on the spec's PeeringTopology.
func projectPeerings(spec TopologySpec, vnetByName map[string]*graph.VNet) {
	switch spec.PeeringTopology {
	case "hub-spoke":
		// Each spoke → hub; hub → each spoke
		for _, v := range spec.VNets {
			if v.Name == spec.HubVNetName {
				continue
			}
			// Spoke gets a peering to hub
			spoke := vnetByName[v.Name]
			spoke.Peerings = append(spoke.Peerings, graph.Peering{
				RemoteVnet:            spec.HubVNetName,
				State:                 "Connected",
				AllowForwardedTraffic: true,
				UseRemoteGateways:     spec.GatewayType != "none",
			})
			// Hub gets a peering back to spoke
			hub := vnetByName[spec.HubVNetName]
			hub.Peerings = append(hub.Peerings, graph.Peering{
				RemoteVnet:            v.Name,
				State:                 "Connected",
				AllowForwardedTraffic: true,
				AllowGatewayTransit:   spec.GatewayType != "none",
			})
		}

	case "mesh":
		// Every VNet peers to every other VNet
		vnets := spec.VNets
		for i := 0; i < len(vnets); i++ {
			for j := 0; j < len(vnets); j++ {
				if i == j {
					continue
				}
				src := vnetByName[vnets[i].Name]
				src.Peerings = append(src.Peerings, graph.Peering{
					RemoteVnet:            vnets[j].Name,
					State:                 "Connected",
					AllowForwardedTraffic: true,
				})
			}
		}

	case "custom":
		for _, pp := range spec.PeeringPairs {
			local := vnetByName[pp.LocalVNet]
			if local == nil {
				continue
			}
			local.Peerings = append(local.Peerings, graph.Peering{
				RemoteVnet:            pp.RemoteVNet,
				State:                 "Connected",
				AllowForwardedTraffic: pp.AllowForwardedTraffic,
				UseRemoteGateways:     pp.UseRemoteGateways,
				AllowGatewayTransit:   pp.AllowGatewayTransit,
			})
		}
	}
}

// strPtr returns a pointer to a copy of s.
func strPtr(s string) *string {
	return &s
}

// subnetVNet extracts the VNet name from a "{vnetName}/{subnetName}" path.
func subnetVNet(path string) string {
	if i := strings.Index(path, "/"); i >= 0 {
		return path[:i]
	}
	return ""
}
