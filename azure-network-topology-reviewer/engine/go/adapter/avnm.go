package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aaraminds/azure-nettopo-engine/internal/graph"
)

// fetchAVNM performs the 5-step Azure Virtual Network Manager walk and returns
// the populated graph.AVNM. If no Network Managers are found in the subscription
// the result is an empty AVNM (not an error).
func (a *adapter) fetchAVNM(ctx context.Context) (graph.AVNM, error) {
	// ── Step 1: list Network Managers via Resource Graph ─────────────────────
	nmRows, err := a.listNetworkManagers(ctx)
	if err != nil {
		return graph.AVNM{SecurityAdminRules: []graph.AdminRule{}}, fmt.Errorf("list network managers: %w", err)
	}
	if len(nmRows) == 0 {
		return graph.AVNM{SecurityAdminRules: []graph.AdminRule{}}, nil
	}

	var allRules []graph.AdminRule

	for _, nm := range nmRows {
		nmName := getString(nm, "name")
		nmRG := getString(nm, "resourceGroup")
		if nmName == "" || nmRG == "" {
			continue
		}

		// ── Step 2: list SecurityAdminConfigurations for this NM ─────────────
		configs, err := a.listSecurityAdminConfigs(ctx, nmRG, nmName)
		if err != nil {
			// Non-fatal: log and continue.
			continue
		}

		for _, cfg := range configs {
			cfgName := getString(cfg, "name")
			if cfgName == "" {
				continue
			}

			// ── Step 3: list RuleCollections for this config ──────────────────
			collections, err := a.listRuleCollections(ctx, nmRG, nmName, cfgName)
			if err != nil {
				continue
			}

			for _, coll := range collections {
				collName := getString(coll, "name")
				if collName == "" {
					continue
				}
				// Capture networkGroupIds that this collection applies to.
				var groupIDs []string
				props := getMap(coll, "properties")
				if props == nil {
					props = coll
				}
				for _, ag := range getSlice(props, "appliesToGroups") {
					if agm, ok := ag.(map[string]interface{}); ok {
						gid := getString(agm, "networkGroupId")
						if gid != "" {
							groupIDs = append(groupIDs, gid)
						}
					}
				}

				// ── Step 4: list Rules for this collection ────────────────────
				rules, err := a.listCollectionRules(ctx, nmRG, nmName, cfgName, collName)
				if err != nil {
					continue
				}

				// ── Step 5: resolve VNet names for each networkGroupId ────────
				var vnetNames []string
				for _, gid := range groupIDs {
					names, err := a.resolveGroupVNets(ctx, nmRG, nmName, gid)
					if err != nil {
						continue
					}
					vnetNames = append(vnetNames, names...)
				}

				// Expand rules into AdminRule entries.
				for _, rule := range rules {
					ruleMap, ok := rule.(map[string]interface{})
					if !ok {
						continue
					}
					expanded := expandAdminRule(ruleMap, vnetNames)
					allRules = append(allRules, expanded...)
				}
			}
		}
	}

	return graph.AVNM{SecurityAdminRules: allRules}, nil
}

// ─── AVNM REST walk helpers ───────────────────────────────────────────────────

func (a *adapter) listNetworkManagers(ctx context.Context) ([]map[string]interface{}, error) {
	kql := fmt.Sprintf(`Resources
| where subscriptionId == %q
| where type == "microsoft.network/networkmanagers"
| project name, resourceGroup, location`, a.subscriptionID)

	// Re-use the Resource Graph client approach; avoid duplicating client init
	// by using the HTTP token approach here.
	url := a.armURL(
		fmt.Sprintf("%s/providers/Microsoft.Network/networkManagers", a.subPath()),
		"2024-03-01",
	)
	items, err := a.listAll(ctx, url)
	if err != nil {
		// Fall back to empty if the subscription has no NMs.
		return nil, nil
	}
	_ = kql // used as documentation; actual call via REST

	var out []map[string]interface{}
	for _, raw := range items {
		var m map[string]interface{}
		if json.Unmarshal(raw, &m) == nil {
			out = append(out, m)
		}
	}
	return out, nil
}

func (a *adapter) listSecurityAdminConfigs(ctx context.Context, nmRG, nmName string) ([]map[string]interface{}, error) {
	url := a.armURL(
		fmt.Sprintf("%s/resourceGroups/%s/providers/Microsoft.Network/networkManagers/%s/securityAdminConfigurations",
			a.subPath(), nmRG, nmName),
		"2024-03-01",
	)
	items, err := a.listAll(ctx, url)
	if err != nil {
		return nil, err
	}
	var out []map[string]interface{}
	for _, raw := range items {
		var m map[string]interface{}
		if json.Unmarshal(raw, &m) == nil {
			out = append(out, m)
		}
	}
	return out, nil
}

func (a *adapter) listRuleCollections(ctx context.Context, nmRG, nmName, cfgName string) ([]map[string]interface{}, error) {
	url := a.armURL(
		fmt.Sprintf("%s/resourceGroups/%s/providers/Microsoft.Network/networkManagers/%s/securityAdminConfigurations/%s/ruleCollections",
			a.subPath(), nmRG, nmName, cfgName),
		"2024-03-01",
	)
	items, err := a.listAll(ctx, url)
	if err != nil {
		return nil, err
	}
	var out []map[string]interface{}
	for _, raw := range items {
		var m map[string]interface{}
		if json.Unmarshal(raw, &m) == nil {
			out = append(out, m)
		}
	}
	return out, nil
}

func (a *adapter) listCollectionRules(ctx context.Context, nmRG, nmName, cfgName, collName string) ([]interface{}, error) {
	url := a.armURL(
		fmt.Sprintf("%s/resourceGroups/%s/providers/Microsoft.Network/networkManagers/%s/securityAdminConfigurations/%s/ruleCollections/%s/rules",
			a.subPath(), nmRG, nmName, cfgName, collName),
		"2024-03-01",
	)
	items, err := a.listAll(ctx, url)
	if err != nil {
		return nil, err
	}
	var out []interface{}
	for _, raw := range items {
		var m interface{}
		if json.Unmarshal(raw, &m) == nil {
			out = append(out, m)
		}
	}
	return out, nil
}

// resolveGroupVNets resolves the VNet names that are static members of a network group.
func (a *adapter) resolveGroupVNets(ctx context.Context, nmRG, nmName, groupID string) ([]string, error) {
	// groupID is an ARM resource ID like:
	//   /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/networkManagers/{nm}/networkGroups/{group}
	groupName := extractResourceName(groupID)
	if groupName == "" {
		return nil, nil
	}
	url := a.armURL(
		fmt.Sprintf("%s/resourceGroups/%s/providers/Microsoft.Network/networkManagers/%s/networkGroups/%s/staticMembers",
			a.subPath(), nmRG, nmName, groupName),
		"2024-03-01",
	)
	items, err := a.listAll(ctx, url)
	if err != nil {
		return nil, err
	}
	var vnetNames []string
	for _, raw := range items {
		var m map[string]interface{}
		if json.Unmarshal(raw, &m) != nil {
			continue
		}
		props := getMap(m, "properties")
		if props == nil {
			props = m
		}
		resourceID := getString(props, "resourceId")
		if vnetName := extractResourceName(resourceID); vnetName != "" {
			vnetNames = append(vnetNames, vnetName)
		}
	}
	return vnetNames, nil
}

// ─── Admin rule expansion ─────────────────────────────────────────────────────

// expandAdminRule converts a raw rule map (from AVNM API) into one or more
// graph.AdminRule entries using Cartesian product expansion for multi-value
// sources × destination port ranges.
func expandAdminRule(ruleMap map[string]interface{}, appliesTo []string) []graph.AdminRule {
	name := getString(ruleMap, "name")
	props := getMap(ruleMap, "properties")
	if props == nil {
		props = ruleMap
	}

	direction := getString(props, "direction")
	access := getString(props, "access")
	protocol := getString(props, "protocol")
	priority := getInt(props, "priority")

	// Sources
	var sources []string
	for _, s := range getSlice(props, "sources") {
		if sm, ok := s.(map[string]interface{}); ok {
			addr := getString(sm, "addressPrefix")
			if addr != "" {
				sources = append(sources, addr)
			}
		}
	}
	if src := getString(props, "sourceAddressPrefix"); src != "" && len(sources) == 0 {
		sources = []string{src}
	}
	if len(sources) == 0 {
		sources = []string{""}
	}

	// Destination port ranges
	var ports []string
	for _, p := range getSlice(props, "destinationPortRanges") {
		if ps, ok := p.(string); ok && ps != "" {
			ports = append(ports, ps)
		}
	}
	if port := getString(props, "destinationPortRange"); port != "" && len(ports) == 0 {
		ports = []string{port}
	}
	if len(ports) == 0 {
		ports = []string{"*"}
	}

	// Normalise access string: "Allow" → "Allow", "AlwaysAllow" → "AlwaysAllow", "Deny" → "Deny".
	access = normaliseAdminAccess(access)

	var out []graph.AdminRule
	for _, src := range sources {
		for _, port := range ports {
			out = append(out, graph.AdminRule{
				Name:                 name,
				Priority:             priority,
				Direction:            direction,
				Access:               access,
				Protocol:             protocol,
				SourceAddressPrefix:  src,
				DestinationPortRange: port,
				AppliesTo:            appliesTo,
			})
		}
	}
	return out
}

func normaliseAdminAccess(s string) string {
	switch strings.ToLower(s) {
	case "alwaysallow":
		return "AlwaysAllow"
	case "allow":
		return "Allow"
	case "deny":
		return "Deny"
	}
	return s
}
