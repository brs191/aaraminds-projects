package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/aaraminds/azure-nettopo-engine/internal/graph"
)

// fetchFirewall resolves NAT rules for the detected Azure Firewall, handling
// both classic (natRuleCollections on the firewall resource) and policy-based
// (ruleCollectionGroups on the linked Firewall Policy) firewalls.
//
// The public IP is resolved from the PublicIPAddresses collection collected by
// Resource Graph; we match by resource ID (last segment = name).
func (a *adapter) fetchFirewall(
	ctx context.Context,
	fw *rawFirewall,
	pips []graph.PublicIP,
) (*graph.Firewall, error) {
	if fw == nil {
		return nil, nil
	}

	// Resolve Public IP address from the PIP collection.
	publicIPAddr := ""
	wantName := extractResourceName(fw.publicIPID)
	for _, pip := range pips {
		if pip.Name == wantName {
			publicIPAddr = pip.IPAddress
			break
		}
	}

	result := &graph.Firewall{
		Name:      fw.name,
		PrivateIP: fw.privateIP,
		PublicIP:  publicIPAddr,
	}

	if fw.firewallPolicyID == "" {
		// ── Classic firewall: GET the resource and parse natRuleCollections ──
		natRules, err := a.fetchClassicFirewallNAT(ctx, fw)
		if err != nil {
			return result, fmt.Errorf("classic firewall NAT: %w", err)
		}
		result.NatRules = natRules
	} else {
		// ── Policy-based firewall: walk RuleCollectionGroups ──────────────────
		result.PolicyRef = fw.firewallPolicyID
		natRules, err := a.fetchPolicyFirewallNAT(ctx, fw.firewallPolicyID)
		if err != nil {
			return result, fmt.Errorf("policy firewall NAT: %w", err)
		}
		result.NatRules = natRules
	}
	return result, nil
}

// fetchClassicFirewallNAT fetches NAT rules from a classic Azure Firewall resource.
// Uses ARM GET: /azureFirewalls/{name}?api-version=2024-03-01
func (a *adapter) fetchClassicFirewallNAT(ctx context.Context, fw *rawFirewall) ([]graph.NatRule, error) {
	url := a.armURL(
		fmt.Sprintf("%s/resourceGroups/%s/providers/Microsoft.Network/azureFirewalls/%s",
			a.subPath(), fw.resourceGroup, fw.name),
		"2024-03-01",
	)
	var fwResource map[string]interface{}
	if err := a.getJSON(ctx, url, &fwResource); err != nil {
		return nil, err
	}

	props := getMap(fwResource, "properties")
	if props == nil {
		return nil, nil
	}

	var natRules []graph.NatRule
	for _, coll := range getSlice(props, "natRuleCollections") {
		cm, ok := coll.(map[string]interface{})
		if !ok {
			continue
		}
		collProps := getMap(cm, "properties")
		if collProps == nil {
			collProps = cm
		}
		for _, rule := range getSlice(collProps, "rules") {
			rm, ok := rule.(map[string]interface{})
			if !ok {
				continue
			}
			nr := parseClassicNATRule(rm)
			if nr.Name != "" {
				natRules = append(natRules, nr)
			}
		}
	}
	return natRules, nil
}

func parseClassicNATRule(rm map[string]interface{}) graph.NatRule {
	name := getString(rm, "name")
	protocols := getStringSlice(rm, "protocols")
	protocol := ""
	if len(protocols) > 0 {
		protocol = protocols[0]
	}

	dstAddrs := getStringSlice(rm, "destinationAddresses")
	dstAddr := ""
	if len(dstAddrs) > 0 {
		dstAddr = dstAddrs[0]
	}

	dstPorts := getStringSlice(rm, "destinationPorts")
	dstPort := 0
	if len(dstPorts) > 0 {
		dstPort = parsePort(dstPorts[0])
	}

	translatedAddr := getString(rm, "translatedAddress")
	translatedPort := parsePort(getString(rm, "translatedPort"))

	return graph.NatRule{
		Name:               name,
		Protocol:           protocol,
		SourceAddresses:    getStringSlice(rm, "sourceAddresses"),
		DestinationAddress: dstAddr,
		DestinationPort:    dstPort,
		TranslatedAddress:  translatedAddr,
		TranslatedPort:     translatedPort,
	}
}

// fetchPolicyFirewallNAT fetches NAT rules from a Firewall Policy's
// RuleCollectionGroups. Only FirewallPolicyNatRuleCollection entries are processed.
func (a *adapter) fetchPolicyFirewallNAT(ctx context.Context, policyID string) ([]graph.NatRule, error) {
	policyRG := extractResourceGroup(policyID)
	policyName := extractResourceName(policyID)
	if policyName == "" || policyRG == "" {
		return nil, nil
	}

	url := a.armURL(
		fmt.Sprintf("%s/resourceGroups/%s/providers/Microsoft.Network/firewallPolicies/%s/ruleCollectionGroups",
			a.subPath(), policyRG, policyName),
		"2024-03-01",
	)
	items, err := a.listAll(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("list rule collection groups: %w", err)
	}

	var natRules []graph.NatRule
	for _, raw := range items {
		var group map[string]interface{}
		if json.Unmarshal(raw, &group) != nil {
			continue
		}
		props := getMap(group, "properties")
		if props == nil {
			props = group
		}
		for _, coll := range getSlice(props, "ruleCollections") {
			cm, ok := coll.(map[string]interface{})
			if !ok {
				continue
			}
			// Only process NAT rule collections.
			ruleCollType := getString(cm, "ruleCollectionType")
			if ruleCollType != "FirewallPolicyNatRuleCollection" {
				continue
			}
			collProps := getMap(cm, "properties")
			if collProps == nil {
				collProps = cm
			}
			for _, rule := range getSlice(collProps, "rules") {
				rm, ok := rule.(map[string]interface{})
				if !ok {
					continue
				}
				nr := parsePolicyNATRule(rm)
				if nr.Name != "" {
					natRules = append(natRules, nr)
				}
			}
		}
	}
	return natRules, nil
}

func parsePolicyNATRule(rm map[string]interface{}) graph.NatRule {
	name := getString(rm, "name")

	// ipProtocols[] for policy-based rules
	protocols := getStringSlice(rm, "ipProtocols")
	protocol := ""
	if len(protocols) > 0 {
		protocol = protocols[0]
	}

	// destinationAddresses[]
	dstAddrs := getStringSlice(rm, "destinationAddresses")
	dstAddr := ""
	if len(dstAddrs) > 0 {
		dstAddr = dstAddrs[0]
	}

	// destinationPorts[]
	dstPorts := getStringSlice(rm, "destinationPorts")
	dstPort := 0
	if len(dstPorts) > 0 {
		dstPort = parsePort(dstPorts[0])
	}

	translatedAddr := getString(rm, "translatedAddress")
	translatedPort := parsePort(getString(rm, "translatedPort"))

	return graph.NatRule{
		Name:               name,
		Protocol:           protocol,
		SourceAddresses:    getStringSlice(rm, "sourceAddresses"),
		DestinationAddress: dstAddr,
		DestinationPort:    dstPort,
		TranslatedAddress:  translatedAddr,
		TranslatedPort:     translatedPort,
	}
}

// parsePort converts a port string to int; returns 0 on parse failure.
func parsePort(s string) int {
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}
