package generator

import (
	"fmt"
	"net/netip"
	"strings"
)

// TopologySpec is the intermediate representation between architect intent (NL)
// and Terraform. Produced by the LLM (AskAT&T), consumed by RenderTerraform.
// Matches the Pydantic models in phase-3/generator/models.py field-for-field.
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
	// "hub-spoke" — one hub VNet peers to N spoke VNets (UseRemoteGateways on spokes)
	// "mesh"      — every VNet peers to every other VNet
	// "none"      — no VNet peering
	// "custom"    — explicit pairs in PeeringPairs
	PeeringTopology string `json:"peeringTopology"`

	// PeeringPairs is populated only when PeeringTopology == "custom".
	PeeringPairs []PeeringPairSpec `json:"peeringPairs,omitempty"`

	// HubVNetName identifies which VNet is the hub for hub-spoke topologies.
	HubVNetName string `json:"hubVnetName,omitempty"`

	// GatewayType is the connectivity gateway to provision in the hub VNet.
	// "vpn" | "expressroute" | "none"
	GatewayType string `json:"gatewayType"`

	// FirewallEnabled indicates whether Azure Firewall should be provisioned.
	FirewallEnabled bool `json:"firewallEnabled"`

	// AVNMEnabled indicates whether an existing AVNM instance is in scope.
	AVNMEnabled bool `json:"avnmEnabled"`

	// AVNMNetworkGroupID is the existing AVNM Network Group ID to reference.
	AVNMNetworkGroupID string `json:"avnmNetworkGroupId,omitempty"`

	// TierLabels is the ordered list of network tiers present in this topology.
	TierLabels []string `json:"tierLabels"`

	// Tags is the set of resource tags to apply to all generated resources.
	// AT&T mandates at minimum: "env", "owner", "costcenter", "appid".
	Tags map[string]string `json:"tags"`
}

// VNetSpec describes one virtual network to generate.
type VNetSpec struct {
	Name         string       `json:"name"`
	AddressSpace []string     `json:"addressSpace"`
	Subnets      []SubnetSpec `json:"subnets"`
	IsHub        bool         `json:"isHub,omitempty"`
}

// SubnetSpec describes one subnet to generate.
type SubnetSpec struct {
	Name                  string               `json:"name"`
	AddressPrefix         string               `json:"addressPrefix"`
	TierLabel             string               `json:"tierLabel"`
	Sensitive             bool                 `json:"sensitive"`
	NSGIntents            []string             `json:"nsgIntents"`
	RouteToFirewall       bool                 `json:"routeToFirewall,omitempty"`
	ServiceEndpoints      []string             `json:"serviceEndpoints,omitempty"`
	Delegations           []string             `json:"delegations,omitempty"`
	PrivateEndpointSubnet bool                 `json:"privateEndpointSubnet,omitempty"`
	PrivateEndpoints      []PrivateEndpointSpec `json:"privateEndpoints,omitempty"`
}

// PrivateEndpointSpec describes one generated private endpoint.
type PrivateEndpointSpec struct {
	Name              string `json:"name"`
	GroupID           string `json:"groupId"`
	ServiceResourceID string `json:"serviceResourceId"`
}

// PeeringPairSpec is used only when PeeringTopology == "custom".
type PeeringPairSpec struct {
	LocalVNet             string `json:"localVnet"`
	RemoteVNet            string `json:"remoteVnet"`
	AllowForwardedTraffic bool   `json:"allowForwardedTraffic,omitempty"`
	UseRemoteGateways     bool   `json:"useRemoteGateways,omitempty"`
	AllowGatewayTransit   bool   `json:"allowGatewayTransit,omitempty"`
}

// validPeeringTopologies is the closed set of allowed peeringTopology values.
var validPeeringTopologies = map[string]bool{
	"hub-spoke": true,
	"mesh":      true,
	"none":      true,
	"custom":    true,
}

// validGatewayTypes is the closed set of allowed gatewayType values.
var validGatewayTypes = map[string]bool{
	"vpn":          true,
	"expressroute": true,
	"none":         true,
}

// requiredTags are the AT&T-mandated tag keys every TopologySpec must carry.
var requiredTags = []string{"env", "owner", "costcenter", "appid"}

// Validate checks structural constraints before rendering.
// Returns the first validation error encountered.
func (s TopologySpec) Validate() error {
	if s.SpecVersion != "1.0" {
		return fmt.Errorf("specVersion must be \"1.0\", got %q", s.SpecVersion)
	}
	if len(s.Description) < 10 {
		return fmt.Errorf("description must be at least 10 characters, got %d", len(s.Description))
	}
	if len(s.VNets) == 0 {
		return fmt.Errorf("vnets must contain at least one entry")
	}
	if !validPeeringTopologies[s.PeeringTopology] {
		return fmt.Errorf("peeringTopology must be one of {hub-spoke, mesh, none, custom}, got %q", s.PeeringTopology)
	}
	if !validGatewayTypes[s.GatewayType] {
		return fmt.Errorf("gatewayType must be one of {vpn, expressroute, none}, got %q", s.GatewayType)
	}
	if s.PeeringTopology == "hub-spoke" {
		if s.HubVNetName == "" {
			return fmt.Errorf("peeringTopology=hub-spoke requires hubVnetName to be set")
		}
		found := false
		for _, v := range s.VNets {
			if v.Name == s.HubVNetName {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("hubVnetName %q does not reference any VNet in vnets", s.HubVNetName)
		}
	}
	if s.PeeringTopology == "custom" && len(s.PeeringPairs) == 0 {
		return fmt.Errorf("peeringTopology=custom requires at least one entry in peeringPairs")
	}
	for _, key := range requiredTags {
		if _, ok := s.Tags[key]; !ok {
			return fmt.Errorf("tags must include required key %q", key)
		}
	}

	// VNet name uniqueness
	vnetNames := make(map[string]bool, len(s.VNets))
	for _, v := range s.VNets {
		if v.Name == "" {
			return fmt.Errorf("all VNets must have a non-empty name")
		}
		if vnetNames[v.Name] {
			return fmt.Errorf("duplicate VNet name %q", v.Name)
		}
		vnetNames[v.Name] = true

		// Subnet name uniqueness within each VNet
		subnetNames := make(map[string]bool, len(v.Subnets))
		for _, sn := range v.Subnets {
			if sn.Name == "" {
				return fmt.Errorf("VNet %q: all subnets must have a non-empty name", v.Name)
			}
			if subnetNames[sn.Name] {
				return fmt.Errorf("VNet %q: duplicate subnet name %q", v.Name, sn.Name)
			}
			subnetNames[sn.Name] = true

			// Validate address prefixes parse
			if _, err := netip.ParsePrefix(sn.AddressPrefix); err != nil {
				return fmt.Errorf("VNet %q subnet %q: invalid addressPrefix %q: %v", v.Name, sn.Name, sn.AddressPrefix, err)
			}
		}

		// Validate VNet address space
		for _, cidr := range v.AddressSpace {
			if _, err := netip.ParsePrefix(cidr); err != nil {
				return fmt.Errorf("VNet %q: invalid addressSpace entry %q: %v", v.Name, cidr, err)
			}
		}
	}

	// Validate peering pair references
	for i, pp := range s.PeeringPairs {
		if !vnetNames[pp.LocalVNet] {
			return fmt.Errorf("peeringPairs[%d]: localVnet %q not found in vnets", i, pp.LocalVNet)
		}
		if !vnetNames[pp.RemoteVNet] {
			return fmt.Errorf("peeringPairs[%d]: remoteVnet %q not found in vnets", i, pp.RemoteVNet)
		}
	}

	// Validate region
	if strings.TrimSpace(s.Region) == "" {
		return fmt.Errorf("region must not be empty")
	}

	return nil
}
