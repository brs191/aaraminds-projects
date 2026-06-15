// Package generator is the Phase 3 topology generation pipeline.
// It translates a validated TopologySpec (LLM-produced structured intent) into a
// TerraformPlan and validates the plan against the deterministic Analyze() engine
// before any Terraform is emitted to a GitHub PR.
package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ModuleRegistryEntry describes one approved Terraform module.
type ModuleRegistryEntry struct {
	ID             string   `json:"id"             yaml:"id"`
	Source         string   `json:"source"         yaml:"source"`
	Version        string   `json:"version"        yaml:"version"`
	Purpose        string   `json:"purpose"        yaml:"purpose"`
	Handles        []string `json:"handles"        yaml:"handles"`
	RequiredInputs []string `json:"requiredInputs" yaml:"required_inputs"`
	Notes          string   `json:"notes"          yaml:"notes"`
}

// ModuleRegistry is the in-memory approved module registry.
type ModuleRegistry struct {
	entries []ModuleRegistryEntry
}

// registryFile is the top-level wrapper for YAML/JSON registry files.
type registryFile struct {
	Modules []ModuleRegistryEntry `json:"modules" yaml:"modules"`
}

// LoadRegistryFromFile loads a YAML or JSON registry file from the given path.
// Returns error if any entry has an unpinned version (contains ">=", "~>", or "*").
func LoadRegistryFromFile(path string) (ModuleRegistry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ModuleRegistry{}, fmt.Errorf("LoadRegistryFromFile: read %q: %w", path, err)
	}

	var rf registryFile
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".json" {
		if err := json.Unmarshal(data, &rf); err != nil {
			return ModuleRegistry{}, fmt.Errorf("LoadRegistryFromFile: JSON unmarshal: %w", err)
		}
	} else {
		// YAML (default)
		if err := yaml.Unmarshal(data, &rf); err != nil {
			return ModuleRegistry{}, fmt.Errorf("LoadRegistryFromFile: YAML unmarshal: %w", err)
		}
	}

	for i, e := range rf.Modules {
		if err := validateVersion(e.Version); err != nil {
			return ModuleRegistry{}, fmt.Errorf("LoadRegistryFromFile: entry %d (%q): %w", i, e.ID, err)
		}
	}
	return ModuleRegistry{entries: rf.Modules}, nil
}

// validateVersion rejects unpinned version constraints.
func validateVersion(v string) error {
	for _, bad := range []string{">=", "~>", "*"} {
		if strings.Contains(v, bad) {
			return fmt.Errorf("unpinned version %q (contains %q); registry requires exact versions only", v, bad)
		}
	}
	return nil
}

// LoadDefaultRegistry returns a hardcoded registry with all 12 approved modules from §2.2.
// Used when no registry file path is configured.
func LoadDefaultRegistry() ModuleRegistry {
	return ModuleRegistry{entries: []ModuleRegistryEntry{
		{
			ID:             "att-vnet",
			Source:         "artifactory.att.com/tf-modules/att-vnet/azurerm",
			Version:        "2.4.1",
			Purpose:        "Hub VNet, spoke VNet, address space, DNS servers",
			Handles:        []string{"vnet"},
			RequiredInputs: []string{"vnet_name", "address_space", "location"},
			Notes:          "[VERIFY] AT&T internal module name and version. Falls back to az-vnet if unavailable.",
		},
		{
			ID:             "az-vnet",
			Source:         "Azure/network/azurerm",
			Version:        "5.3.0",
			Purpose:        "VNet creation, address space",
			Handles:        []string{"vnet-public"},
			RequiredInputs: []string{"vnet_name", "address_space", "location"},
			Notes:          "Public fallback when att-vnet not available",
		},
		{
			ID:             "az-subnets",
			Source:         "Azure/subnets/azurerm",
			Version:        "1.0.0",
			Purpose:        "Subnet creation, NSG association, route table association, service endpoints, delegations",
			Handles:        []string{"subnets"},
			RequiredInputs: []string{"resource_group_name", "virtual_network_name", "subnet_names", "subnet_prefixes"},
			Notes:          "Used for all subnet creation",
		},
		{
			ID:             "az-nsg",
			Source:         "Azure/network-security-group/azurerm",
			Version:        "4.1.0",
			Purpose:        "NSG + security rules from intent vocabulary",
			Handles:        []string{"nsg"},
			RequiredInputs: []string{"resource_group_name", "security_group_name", "location"},
			Notes:          "Renderer maps NSGIntent strings to module rule inputs",
		},
		{
			ID:             "az-hub-spoke",
			Source:         "Azure/caf-enterprise-scale/azurerm",
			Version:        "6.2.0",
			Purpose:        "Hub-spoke peering topology, UDR propagation",
			Handles:        []string{"hub-spoke-peering"},
			RequiredInputs: []string{"hub_virtual_network_resource_id", "virtual_network_resource_ids_to_peer_to_hub"},
			Notes:          "Preferred for hub-spoke peering. Scope-limits to connectivity module only.",
		},
		{
			ID:             "az-vpn-gw",
			Source:         "Azure/vpn-gateway/azurerm",
			Version:        "1.3.2",
			Purpose:        "VPN Gateway in GatewaySubnet",
			Handles:        []string{"vpn-gateway"},
			RequiredInputs: []string{"resource_group_name", "location", "subnet_id"},
			Notes:          "Used when gatewayType == vpn",
		},
		{
			ID:             "az-er-gw",
			Source:         "Azure/expressroute-gateway/azurerm",
			Version:        "1.1.0",
			Purpose:        "ExpressRoute Gateway",
			Handles:        []string{"expressroute-gateway"},
			RequiredInputs: []string{"resource_group_name", "location", "subnet_id"},
			Notes:          "Used when gatewayType == expressroute",
		},
		{
			ID:             "az-firewall",
			Source:         "Azure/firewall/azurerm",
			Version:        "2.2.1",
			Purpose:        "Azure Firewall + Firewall Policy, Standard/Premium SKU",
			Handles:        []string{"firewall"},
			RequiredInputs: []string{"resource_group_name", "location", "virtual_network_id"},
			Notes:          "Used when firewallEnabled == true",
		},
		{
			ID:             "az-bastion",
			Source:         "Azure/bastion/azurerm",
			Version:        "2.0.0",
			Purpose:        "Azure Bastion in AzureBastionSubnet",
			Handles:        []string{"bastion"},
			RequiredInputs: []string{"resource_group_name", "location", "virtual_network_name"},
			Notes:          "Used when tierLabels contains bastion",
		},
		{
			ID:             "az-appgw-waf",
			Source:         "Azure/application-gateway/azurerm",
			Version:        "3.1.0",
			Purpose:        "Application Gateway with WAF v2 policy",
			Handles:        []string{"appgw-waf"},
			RequiredInputs: []string{"resource_group_name", "location", "sku_name"},
			Notes:          "Used when tierLabels contains appgw. WAF policy MUST be enabled.",
		},
		{
			ID:             "az-private-endpoint",
			Source:         "Azure/private-endpoint/azurerm",
			Version:        "1.2.0",
			Purpose:        "Private Endpoint + Private DNS Zone link",
			Handles:        []string{"private-endpoint"},
			RequiredInputs: []string{"resource_group_name", "location", "name", "private_connection_resource_id"},
			Notes:          "Used when any subnet declares privateEndpoints[]",
		},
		{
			ID:             "az-route-table",
			Source:         "Azure/route-table/azurerm",
			Version:        "1.1.0",
			Purpose:        "UDR / route table creation",
			Handles:        []string{"route-table"},
			RequiredInputs: []string{"resource_group_name", "location"},
			Notes:          "Used when routeToFirewall: true on any subnet",
		},
	}}
}

// Select returns the ModuleRegistryEntry whose Handles slice contains the given capability.
// Returns (entry, true) if found; (zero, false) if not.
func (r ModuleRegistry) Select(capability string) (ModuleRegistryEntry, bool) {
	for _, e := range r.entries {
		for _, h := range e.Handles {
			if h == capability {
				return e, true
			}
		}
	}
	return ModuleRegistryEntry{}, false
}
