package forecast

import (
	"strings"

	"github.com/aaraminds/azure-nettopo-engine/internal/graph"
	"github.com/aaraminds/azure-nettopo-engine/simulator"
)

const (
	// heuristicGBPerNICPerMonth is the AT&T estate empirical P50 baseline
	// (from 2025 Traffic Analytics data). Used when Flow Logs are disabled.
	heuristicGBPerNICPerMonth = 50.0
)

// FlowSummary holds observed monthly traffic volumes from Traffic Analytics.
// When FlowLogsEnabled is false (or the map has no entry for the affected resource),
// EstimateTrafficGB falls back to the subscription heuristic.
type FlowSummary struct {
	// MonthlyGBByNSG maps NSG name → average observed monthly GB (last 30 days).
	// Populated from: AzureNetworkAnalytics_CL | summarize sum(BytesSent_d) by NSGName_s
	MonthlyGBByNSG map[string]float64

	// MonthlyGBByVNet maps VNet name → aggregate monthly GB across all NSGs in the VNet.
	MonthlyGBByVNet map[string]float64

	// FlowLogsEnabled indicates whether Traffic Analytics data is available for
	// the subscription. When false, all estimates use the heuristic.
	FlowLogsEnabled bool
}

// EstimateTrafficGB returns (estimatedMonthlyGB, flowLogsUsed) for the segment
// affected by the delta. "Affected segment" depends on the delta type:
//
//   - AddNSGRule / RemoveNSGRule → NSG's observed traffic
//   - ModifyRoute → VNet(s) containing the route table's associated subnets
//   - AddPeering / RemovePeering → local VNet's aggregate traffic
//   - AddSubnet → new subnet, no history: returns (0, false)
//   - AddPublicIP / RemovePublicIP → no variable cost: returns (0, false)
//
// Falls back to the heuristic (NIC count × 50 GB/NIC/month) when flow log
// data is absent or unavailable for the specific resource.
func EstimateTrafficGB(fx *graph.Fixture, delta simulator.TopologyDelta, flows FlowSummary) (estimatedGB float64, flowLogsUsed bool) {
	switch {
	case delta.AddNSGRule != nil:
		return trafficForNSG(fx, delta.AddNSGRule.NSGName, flows)
	case delta.RemoveNSGRule != nil:
		return trafficForNSG(fx, delta.RemoveNSGRule.NSGName, flows)
	case delta.ModifyRoute != nil:
		return trafficForRouteTable(fx, delta.ModifyRoute.RouteTableName, flows)
	case delta.AddPeering != nil:
		return trafficForVNet(fx, delta.AddPeering.LocalVNet, flows)
	case delta.RemovePeering != nil:
		return trafficForVNet(fx, delta.RemovePeering.LocalVNet, flows)
	default:
		// AddSubnet (new, no NICs), AddPublicIP, RemovePublicIP: no variable cost.
		return 0, false
	}
}

func trafficForNSG(fx *graph.Fixture, nsgName string, flows FlowSummary) (float64, bool) {
	if flows.FlowLogsEnabled {
		if gb, ok := flows.MonthlyGBByNSG[nsgName]; ok && gb > 0 {
			return gb, true
		}
	}
	nicCount := countNICsForNSG(fx, nsgName)
	return float64(nicCount) * heuristicGBPerNICPerMonth, false
}

func trafficForRouteTable(fx *graph.Fixture, rtName string, flows FlowSummary) (float64, bool) {
	vnetNames := vnetsForRouteTable(fx, rtName)
	if len(vnetNames) == 0 {
		return 0, false
	}
	var total float64
	anyFlowLogs := false
	for _, vn := range vnetNames {
		gb, used := trafficForVNet(fx, vn, flows)
		total += gb
		if used {
			anyFlowLogs = true
		}
	}
	return total, anyFlowLogs
}

func trafficForVNet(fx *graph.Fixture, vnetName string, flows FlowSummary) (float64, bool) {
	if flows.FlowLogsEnabled {
		if gb, ok := flows.MonthlyGBByVNet[vnetName]; ok && gb > 0 {
			return gb, true
		}
	}
	nicCount := countNICsInVNet(fx, vnetName)
	return float64(nicCount) * heuristicGBPerNICPerMonth, false
}

// ---- helpers ----

// countNICsForNSG counts NICs governed by the named NSG (subnet-level or NIC-level).
func countNICsForNSG(fx *graph.Fixture, nsgName string) int {
	subnetNSG := buildSubnetNSGMap(fx)
	count := 0
	for _, nic := range fx.ResourceGraph.NetworkInterfaces {
		if subnetNSG[nic.Subnet] == nsgName {
			count++
			continue
		}
		if nic.NetworkSecurityGroup != nil && *nic.NetworkSecurityGroup == nsgName {
			count++
		}
	}
	return count
}

// countNICsInVNet counts all NICs whose Subnet starts with "{vnetName}/".
func countNICsInVNet(fx *graph.Fixture, vnetName string) int {
	prefix := vnetName + "/"
	count := 0
	for _, nic := range fx.ResourceGraph.NetworkInterfaces {
		if strings.HasPrefix(nic.Subnet, prefix) {
			count++
		}
	}
	return count
}

// vnetsForRouteTable returns the unique VNet names that contain subnets
// listed in the route table's AssociatedSubnets.
func vnetsForRouteTable(fx *graph.Fixture, rtName string) []string {
	seen := map[string]bool{}
	for _, rt := range fx.ResourceGraph.RouteTables {
		if rt.Name != rtName {
			continue
		}
		for _, sub := range rt.AssociatedSubnets {
			if vn := vnetFromSubnetKey(sub); vn != "" {
				seen[vn] = true
			}
		}
	}
	out := make([]string, 0, len(seen))
	for vn := range seen {
		out = append(out, vn)
	}
	return out
}

// buildSubnetNSGMap returns a map from "{vnet}/{subnet}" → NSG name.
func buildSubnetNSGMap(fx *graph.Fixture) map[string]string {
	m := make(map[string]string)
	for _, vn := range fx.ResourceGraph.VirtualNetworks {
		for _, sn := range vn.Subnets {
			m[vn.Name+"/"+sn.Name] = sn.NetworkSecurityGroup
		}
	}
	return m
}

// vnetFromSubnetKey extracts the VNet name from a "{vnet}/{subnet}" key.
func vnetFromSubnetKey(key string) string {
	if i := strings.Index(key, "/"); i >= 0 {
		return key[:i]
	}
	return ""
}
