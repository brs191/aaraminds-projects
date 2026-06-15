package simulator

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/aaraminds/azure-nettopo-engine/internal/graph"
)

// ApplyDelta applies a single topology delta to a fixture and returns a new,
// independent fixture. The original fixture is NEVER mutated — a JSON
// round-trip deep copy is performed before any change is applied.
//
// Returns an error if:
//   - delta.Validate() fails
//   - the target resource does not exist in the fixture
//   - adding a resource would create a duplicate name
func ApplyDelta(fixture *graph.Fixture, delta TopologyDelta) (*graph.Fixture, error) {
	if err := delta.Validate(); err != nil {
		return nil, err
	}

	sim, err := deepCopy(fixture)
	if err != nil {
		return nil, fmt.Errorf("ApplyDelta: deep copy failed: %w", err)
	}

	switch {
	case delta.AddSubnet != nil:
		err = applyAddSubnet(sim, delta.AddSubnet)
	case delta.RemoveSubnet != nil:
		err = applyRemoveSubnet(sim, delta.RemoveSubnet)
	case delta.AddNSGRule != nil:
		err = applyAddNSGRule(sim, delta.AddNSGRule)
	case delta.RemoveNSGRule != nil:
		err = applyRemoveNSGRule(sim, delta.RemoveNSGRule)
	case delta.AddPeering != nil:
		err = applyAddPeering(sim, delta.AddPeering)
	case delta.RemovePeering != nil:
		err = applyRemovePeering(sim, delta.RemovePeering)
	case delta.AddPublicIP != nil:
		err = applyAddPublicIP(sim, delta.AddPublicIP)
	case delta.RemovePublicIP != nil:
		err = applyRemovePublicIP(sim, delta.RemovePublicIP)
	case delta.ModifyRoute != nil:
		err = applyModifyRoute(sim, delta.ModifyRoute)
	}
	if err != nil {
		return nil, err
	}
	return sim, nil
}

// deepCopy produces an independent copy of a Fixture via JSON round-trip.
// This is safe for all pointer fields (*string, *Firewall, *Enrichment),
// maps (EffectiveSecurityRules, EffectiveRoutes), and omitempty slices.
func deepCopy(src *graph.Fixture) (*graph.Fixture, error) {
	b, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}
	var dst graph.Fixture
	if err := json.Unmarshal(b, &dst); err != nil {
		return nil, err
	}
	return &dst, nil
}

// ---- per-operation apply functions ----

func applyAddSubnet(sim *graph.Fixture, op *AddSubnetOp) error {
	vnet := findVNet(sim, op.VNetName)
	if vnet == nil {
		return fmt.Errorf("AddSubnet: VNet %q not found", op.VNetName)
	}
	for _, s := range vnet.Subnets {
		if s.Name == op.Name {
			return fmt.Errorf("AddSubnet: subnet %q already exists in VNet %q", op.Name, op.VNetName)
		}
	}
	vnet.Subnets = append(vnet.Subnets, graph.Subnet{
		Name:                 op.Name,
		AddressPrefix:        op.AddressPrefix,
		NetworkSecurityGroup: op.NSGName,
		RouteTable:           op.RouteTableName,
	})
	return nil
}

func applyRemoveSubnet(sim *graph.Fixture, op *RemoveSubnetOp) error {
	vnet := findVNet(sim, op.VNetName)
	if vnet == nil {
		return fmt.Errorf("RemoveSubnet: VNet %q not found", op.VNetName)
	}
	idx := -1
	for i, s := range vnet.Subnets {
		if s.Name == op.SubnetName {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("RemoveSubnet: subnet %q not found in VNet %q", op.SubnetName, op.VNetName)
	}
	vnet.Subnets = append(vnet.Subnets[:idx], vnet.Subnets[idx+1:]...)
	return nil
}

func applyAddNSGRule(sim *graph.Fixture, op *AddNSGRuleOp) error {
	nsg := findNSG(sim, op.NSGName)
	if nsg == nil {
		return fmt.Errorf("AddNSGRule: NSG %q not found", op.NSGName)
	}
	for _, r := range nsg.SecurityRules {
		if r.Name == op.Rule.Name {
			return fmt.Errorf("AddNSGRule: rule %q already exists in NSG %q", op.Rule.Name, op.NSGName)
		}
	}
	// normalise: Source == SourceAddressPrefix for backwards compat
	rule := op.Rule
	if rule.Source == "" {
		rule.Source = rule.SourceAddressPrefix
	}
	nsg.SecurityRules = append(nsg.SecurityRules, rule)
	projectEffectiveRules(sim, op.NSGName)
	return nil
}

func applyRemoveNSGRule(sim *graph.Fixture, op *RemoveNSGRuleOp) error {
	nsg := findNSG(sim, op.NSGName)
	if nsg == nil {
		return fmt.Errorf("RemoveNSGRule: NSG %q not found", op.NSGName)
	}
	idx := -1
	for i, r := range nsg.SecurityRules {
		if r.Name == op.RuleName {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("RemoveNSGRule: rule %q not found in NSG %q", op.RuleName, op.NSGName)
	}
	nsg.SecurityRules = append(nsg.SecurityRules[:idx], nsg.SecurityRules[idx+1:]...)
	projectEffectiveRules(sim, op.NSGName)
	return nil
}

func applyAddPeering(sim *graph.Fixture, op *AddPeeringOp) error {
	lv := findVNet(sim, op.LocalVNet)
	if lv == nil {
		return fmt.Errorf("AddPeering: LocalVNet %q not found", op.LocalVNet)
	}
	rv := findVNet(sim, op.RemoteVNet)
	if rv == nil {
		return fmt.Errorf("AddPeering: RemoteVNet %q not found", op.RemoteVNet)
	}
	_ = rv
	for _, p := range lv.Peerings {
		if p.RemoteVnet == op.RemoteVNet {
			return fmt.Errorf("AddPeering: peering to %q already exists in VNet %q", op.RemoteVNet, op.LocalVNet)
		}
	}
	state := op.State
	if state == "" {
		state = "Connected"
	}
	lv.Peerings = append(lv.Peerings, graph.Peering{
		RemoteVnet:            op.RemoteVNet,
		State:                 state,
		AllowForwardedTraffic: op.AllowForwardedTraffic,
		AllowGatewayTransit:   op.AllowGatewayTransit,
		UseRemoteGateways:     op.UseRemoteGateways,
	})
	return nil
}

func applyRemovePeering(sim *graph.Fixture, op *RemovePeeringOp) error {
	lv := findVNet(sim, op.LocalVNet)
	if lv == nil {
		return fmt.Errorf("RemovePeering: LocalVNet %q not found", op.LocalVNet)
	}
	idx := -1
	for i, p := range lv.Peerings {
		if p.RemoteVnet == op.RemoteVNet {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("RemovePeering: peering to %q not found in VNet %q", op.RemoteVNet, op.LocalVNet)
	}
	lv.Peerings = append(lv.Peerings[:idx], lv.Peerings[idx+1:]...)
	return nil
}

func applyAddPublicIP(sim *graph.Fixture, op *AddPublicIPOp) error {
	nic := findNIC(sim, op.NICName)
	if nic == nil {
		return fmt.Errorf("AddPublicIP: NIC %q not found", op.NICName)
	}
	if nic.PublicIP != nil && *nic.PublicIP != "" {
		return fmt.Errorf("AddPublicIP: NIC %q already has public IP %q; remove it first", op.NICName, *nic.PublicIP)
	}
	name := op.PIPName
	nic.PublicIP = &name
	// Add the PIP to the resource list with IPConfiguration set so it is not
	// treated as orphaned in the simulation.
	nicName := op.NICName
	sim.ResourceGraph.PublicIPAddresses = append(sim.ResourceGraph.PublicIPAddresses, graph.PublicIP{
		Name:            op.PIPName,
		IPAddress:       op.IPAddress,
		IPConfiguration: &nicName,
	})
	// Propagate the NIC change back (findNIC returns a pointer into the slice).
	return nil
}

func applyRemovePublicIP(sim *graph.Fixture, op *RemovePublicIPOp) error {
	nic := findNIC(sim, op.NICName)
	if nic == nil {
		return fmt.Errorf("RemovePublicIP: NIC %q not found", op.NICName)
	}
	nic.PublicIP = nil
	// Mark the detached PIP as orphaned (IPConfiguration = nil).
	for i := range sim.ResourceGraph.PublicIPAddresses {
		pip := &sim.ResourceGraph.PublicIPAddresses[i]
		if pip.IPConfiguration != nil && *pip.IPConfiguration != "" {
			// Check if this PIP's ipconfig references our NIC.
			if pip.IPConfiguration != nil && strings.Contains(*pip.IPConfiguration, op.NICName) {
				pip.IPConfiguration = nil
			}
		}
	}
	return nil
}

func applyModifyRoute(sim *graph.Fixture, op *ModifyRouteOp) error {
	rt := findRouteTable(sim, op.RouteTableName)
	if rt == nil {
		return fmt.Errorf("ModifyRoute: RouteTable %q not found", op.RouteTableName)
	}
	idx := -1
	for i, r := range rt.Routes {
		if r.Name == op.RouteName {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("ModifyRoute: route %q not found in RouteTable %q", op.RouteName, op.RouteTableName)
	}
	rt.Routes[idx].NextHopType = op.NewNextHopType
	rt.Routes[idx].NextHopIPAddress = op.NewNextHopIP
	projectEffectiveRoutes(sim, op.RouteTableName)
	return nil
}

// ---- projection functions ----

// projectEffectiveRules rebuilds NetworkWatcher.EffectiveSecurityRules for all
// NICs governed by nsgName after the NSG's SecurityRules have been modified.
//
// Strip criterion: remove effective rules whose Name is in the original
// declared set AND Priority < 65000 (user-defined range). This preserves
// Azure system defaults (AllowVnetInBound at 65000, DenyAllInBound at 65500)
// even when a user rule shares the same name.
func projectEffectiveRules(sim *graph.Fixture, nsgName string) {
	nsg := findNSG(sim, nsgName)
	if nsg == nil {
		return
	}

	// Build declared name set for this NSG (post-modification state).
	declaredNames := map[string]bool{}
	for _, r := range nsg.SecurityRules {
		declaredNames[r.Name] = true
	}

	affectedNICs := nicsByNSG(sim, nsgName)

	for _, nicName := range affectedNICs {
		base := sim.NetworkWatcher.EffectiveSecurityRules[nicName]

		// Strip pre-delta declared rules: Name in declared set AND user-range priority.
		stripped := base[:0:0] // new slice, shares no backing array
		for _, r := range base {
			isDeclared := declaredNames[r.Name]
			isUserRange := r.Priority > 0 && r.Priority < 65000
			if isDeclared && isUserRange {
				continue // remove — will be replaced by the post-delta declared rules
			}
			stripped = append(stripped, r)
		}

		// Inject the post-delta declared rules.
		stripped = append(stripped, nsg.SecurityRules...)

		// Sort by priority ascending (lower number = higher precedence).
		sort.Slice(stripped, func(i, j int) bool {
			return stripped[i].Priority < stripped[j].Priority
		})

		if sim.NetworkWatcher.EffectiveSecurityRules == nil {
			sim.NetworkWatcher.EffectiveSecurityRules = map[string][]graph.SecRule{}
		}
		sim.NetworkWatcher.EffectiveSecurityRules[nicName] = stripped
	}
}

// projectEffectiveRoutes rebuilds NetworkWatcher.EffectiveRoutes for all NICs
// in subnets associated with routeTableName after Routes have been modified.
func projectEffectiveRoutes(sim *graph.Fixture, routeTableName string) {
	rt := findRouteTable(sim, routeTableName)
	if rt == nil {
		return
	}

	// Find all NICs in subnets associated with this route table.
	subnetSet := map[string]bool{}
	for _, s := range rt.AssociatedSubnets {
		subnetSet[s] = true
	}

	for i := range sim.ResourceGraph.NetworkInterfaces {
		nic := &sim.ResourceGraph.NetworkInterfaces[i]
		if !subnetSet[nic.Subnet] {
			continue
		}
		base := sim.NetworkWatcher.EffectiveRoutes[nic.Name]

		// For each modified route, replace any existing effective route with
		// the same AddressPrefix (UDRs override system routes for same prefix),
		// or append if not present.
		routeMap := map[string]int{} // addressPrefix → index in base
		for idx, r := range base {
			routeMap[r.AddressPrefix] = idx
		}

		result := make([]graph.Route, len(base))
		copy(result, base)

		for _, udr := range rt.Routes {
			if idx, exists := routeMap[udr.AddressPrefix]; exists {
				result[idx] = udr
			} else {
				result = append(result, udr)
			}
		}

		if sim.NetworkWatcher.EffectiveRoutes == nil {
			sim.NetworkWatcher.EffectiveRoutes = map[string][]graph.Route{}
		}
		sim.NetworkWatcher.EffectiveRoutes[nic.Name] = result
	}
}

// ---- lookup helpers ----

func findVNet(sim *graph.Fixture, name string) *graph.VNet {
	for i := range sim.ResourceGraph.VirtualNetworks {
		if sim.ResourceGraph.VirtualNetworks[i].Name == name {
			return &sim.ResourceGraph.VirtualNetworks[i]
		}
	}
	return nil
}

func findNSG(sim *graph.Fixture, name string) *graph.NSG {
	for i := range sim.ResourceGraph.NetworkSecurityGroups {
		if sim.ResourceGraph.NetworkSecurityGroups[i].Name == name {
			return &sim.ResourceGraph.NetworkSecurityGroups[i]
		}
	}
	return nil
}

func findRouteTable(sim *graph.Fixture, name string) *graph.RouteTable {
	for i := range sim.ResourceGraph.RouteTables {
		if sim.ResourceGraph.RouteTables[i].Name == name {
			return &sim.ResourceGraph.RouteTables[i]
		}
	}
	return nil
}

func findNIC(sim *graph.Fixture, name string) *graph.NIC {
	for i := range sim.ResourceGraph.NetworkInterfaces {
		if sim.ResourceGraph.NetworkInterfaces[i].Name == name {
			return &sim.ResourceGraph.NetworkInterfaces[i]
		}
	}
	return nil
}

// nicsByNSG returns the names of all NICs governed by the named NSG.
// A NIC is governed if: (a) its subnet's NSG field matches, or (b) its
// NIC-level NetworkSecurityGroup field matches.
func nicsByNSG(sim *graph.Fixture, nsgName string) []string {
	// Build subnet→NSG mapping.
	subnetNSG := map[string]string{} // "{vnet}/{subnet}" → nsgName
	for _, vn := range sim.ResourceGraph.VirtualNetworks {
		for _, sn := range vn.Subnets {
			key := vn.Name + "/" + sn.Name
			subnetNSG[key] = sn.NetworkSecurityGroup
		}
	}

	var names []string
	for _, nic := range sim.ResourceGraph.NetworkInterfaces {
		if subnetNSG[nic.Subnet] == nsgName {
			names = append(names, nic.Name)
			continue
		}
		if nic.NetworkSecurityGroup != nil && *nic.NetworkSecurityGroup == nsgName {
			names = append(names, nic.Name)
		}
	}
	return names
}
