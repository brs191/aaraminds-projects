// Package analyze is the deterministic analysis core — a direct port of
// reference/analyze.py. No model in the path; the same inputs always produce the
// same findings. The reference Python proves these functions against the fixtures.
package analyze

import (
	"fmt"
	"net/netip"
	"sort"
	"strings"

	"github.com/aaraminds/azure-nettopo-engine/internal/graph"
)

type Finding struct {
	Type      string `json:"type"`
	Severity  string `json:"severity"`
	Resource  string `json:"resource"`
	Evidence  string `json:"evidence"`
	Reachable bool   `json:"reachable"`
}

func cidrOverlap(a, b string) bool {
	pa, err1 := netip.ParsePrefix(a)
	pb, err2 := netip.ParsePrefix(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return pa.Overlaps(pb)
}

func isInternetSource(s string) bool {
	switch strings.ToLower(s) {
	case "0.0.0.0/0", "internet", "*":
		return true
	}
	return false
}

func isBroadTagSource(s string) bool { return strings.EqualFold(s, "azurecloud") }

func nicVnet(n graph.NIC) string {
	if i := strings.Index(n.Subnet, "/"); i >= 0 {
		return n.Subnet[:i]
	}
	return ""
}

// subnetToVnet extracts the VNet name from a "{vnetName}/{subnetName}" string.
func subnetToVnet(subnet string) string {
	if i := strings.Index(subnet, "/"); i >= 0 {
		return subnet[:i]
	}
	return ""
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// adminVerdict — Gate 1. Highest-priority inbound AVNM verdict governing an
// internet-sourced flow on port. Source-scope aware: an Internet-tag rule does
// not govern intra-VNet/peered sources.
func adminVerdict(rules []graph.AdminRule, vnet, port string) string {
	best := ""
	bestPri := 1 << 30
	for _, ar := range rules {
		if ar.Direction != "Inbound" || !contains(ar.AppliesTo, vnet) || ar.DestinationPortRange != port {
			continue
		}
		src := strings.ToLower(ar.SourceAddressPrefix)
		if src != "internet" && src != "0.0.0.0/0" && src != "*" {
			continue
		}
		if ar.Priority < bestPri {
			bestPri, best = ar.Priority, ar.Access
		}
	}
	return best
}

// rid returns the ARM resource id when present, else the bare name. Bare names
// are not unique across subscriptions, so multi-sub estates must carry id (V4-07).
func rid(name, id string) string {
	if id != "" {
		return id
	}
	return name
}

// Analyze computes the findings for a topology. Deterministic.
func Analyze(fx *graph.Fixture) []Finding {
	var findings []Finding
	rg := fx.ResourceGraph
	effRules := fx.NetworkWatcher.EffectiveSecurityRules
	effRoutes := fx.NetworkWatcher.EffectiveRoutes
	admin := fx.AVNM.SecurityAdminRules
	fw := fx.AzureFirewall

	nics := map[string]graph.NIC{}
	for _, n := range rg.NetworkInterfaces {
		nics[rid(n.Name, n.ID)] = n
	}

	for name, nic := range nics {
		// name is the rid (id||name); NW tables are id-keyed on a live multi-sub
		// estate but name-keyed in current fixtures — try rid, fall back to name.
		rules := effRules[name]
		if rules == nil {
			rules = effRules[nic.Name]
		}
		routes := effRoutes[name]
		if routes == nil {
			routes = effRoutes[nic.Name]
		}
		hasPIP := nic.PublicIP != nil && *nic.PublicIP != ""
		defaultHop := ""
		for _, r := range routes {
			if r.AddressPrefix == "0.0.0.0/0" {
				defaultHop = r.NextHopType
				break
			}
		}
		vnet := nicVnet(nic)

		for _, r := range rules {
			if r.Direction != "Inbound" {
				continue
			}
			src := r.SourceAddressPrefix
			broadNet, broadTag := isInternetSource(src), isBroadTagSource(src)
			if !broadNet && !broadTag {
				continue
			}
			port := r.DestinationPortRange
			av := adminVerdict(admin, vnet, port) // Gate 1
			openInternet := r.Access == "Allow"   // Gate 2
			switch av {
			case "AlwaysAllow":
				openInternet = true
			case "Deny":
				openInternet = false
			}
			reachable := openInternet && hasPIP && defaultHop == "Internet" // Gates 3+4

			if reachable {
				sev := "High"
				if strings.EqualFold(nic.Tags["sensitive"], "true") {
					sev = "Critical"
				}
				ev := fmt.Sprintf("%s:%s inbound + route 0.0.0.0/0->Internet + public IP %s", src, port, deref(nic.PublicIP))
				if av == "AlwaysAllow" {
					ev += " (AVNM AlwaysAllow overrides NSG)"
				}
				if broadTag {
					ev += " — AzureCloud tag = all Azure public IPs, cross-tenant"
				}
				findings = append(findings, Finding{"over-permissive NSG (reachable)", sev, name, ev, true})
			} else {
				var why []string
				if !hasPIP {
					why = append(why, "no public IP")
				}
				if defaultHop == "None" {
					why = append(why, "route 0.0.0.0/0->None (black-hole)")
				} else if defaultHop != "" && defaultHop != "Internet" {
					why = append(why, "route 0.0.0.0/0->"+defaultHop)
				}
				if av == "Deny" {
					why = append(why, "AVNM Deny closes the Internet source (east-west may remain open)")
				}
				reason := strings.Join(why, "; ")
				if reason == "" {
					reason = "not reachable"
				}
				findings = append(findings, Finding{"over-permissive NSG (latent)", "Informational", name,
					fmt.Sprintf("%s:%s inbound but %s", src, port, reason), false})
			}
		}

		if fw != nil {
			for _, nat := range fw.NatRules {
				if nat.TranslatedAddress == nic.PrivateIP {
					findings = append(findings, Finding{"over-permissive NSG (reachable)", "High", name,
						fmt.Sprintf("firewall DNAT %s:%d -> %s:%d (source %v); no public IP on the NIC",
							fw.PublicIP, nat.DestinationPort, nic.PrivateIP, nat.TranslatedPort, nat.SourceAddresses), true})
				}
			}
		}
	}

	for _, pip := range rg.PublicIPAddresses {
		if pip.IPConfiguration == nil || *pip.IPConfiguration == "" {
			findings = append(findings, Finding{"orphaned public endpoint", "Low", rid(pip.Name, pip.ID),
				fmt.Sprintf("public IP %s with null ipConfiguration", pip.IPAddress), false})
		}
	}

	vnets := rg.VirtualNetworks
	for i := 0; i < len(vnets); i++ {
		for j := i + 1; j < len(vnets); j++ {
			for _, pa := range vnets[i].AddressSpace {
				for _, pb := range vnets[j].AddressSpace {
					if cidrOverlap(pa, pb) {
						findings = append(findings, Finding{"CIDR overlap", "Medium",
							vnets[i].Name + "~" + vnets[j].Name,
							fmt.Sprintf("overlapping address space %s / %s", pa, pb), false})
					}
				}
			}
		}
	}

	// segmentation: sensitive subnet reachable VNet-wide via the default AllowVnetInBound
	for name, nic := range nics {
		if !strings.EqualFold(nic.Tags["sensitive"], "true") {
			continue
		}
		segRules := effRules[name]
		if segRules == nil {
			segRules = effRules[nic.Name]
		}
		allowVnet, denyVnet := false, false
		for _, r := range segRules {
			if r.Name == "AllowVnetInBound" || (r.Priority == 65000 && r.Access == "Allow") {
				allowVnet = true
			}
			if strings.Contains(r.Name, "DenyVnetInBound") {
				denyVnet = true
			}
		}
		if allowVnet && !denyVnet {
			findings = append(findings, Finding{"missing tier segmentation", "High", name,
				"sensitive subnet reachable VNet-wide via default AllowVnetInBound (no DenyVnetInBound above priority 65000)", true})
		}
	}

	findings = append(findings, checkPrivateDnsZoneMisconfiguration(rg)...)
	findings = append(findings, checkAppGatewayExposure(rg)...)
	findings = append(findings, checkAKSExposure(rg)...)
	findings = append(findings, checkCrossSubPeeringExposure(fx)...)
	findings = append(findings, checkLoadBalancerNAT(rg)...)
	findings = append(findings, checkAPIMExposure(rg)...)
	findings = append(findings, checkBastionBypass(rg, effRules)...)
	findings = append(findings, checkVirtualWAN(rg)...)
	findings = append(findings, checkFrontDoorExposure(rg)...)

	// Stable + total order: a NIC can emit two findings with the same
	// (Resource, Type) (e.g. two latent rules); Evidence breaks the tie so the
	// output order is fully deterministic across slice sizes / Go versions and
	// matches the Python reference twin's sort. (Adversarial review HIGH-2.)
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Resource != findings[j].Resource {
			return findings[i].Resource < findings[j].Resource
		}
		if findings[i].Type != findings[j].Type {
			return findings[i].Type < findings[j].Type
		}
		return findings[i].Evidence < findings[j].Evidence
	})
	return findings
}

// peGroupIdToZone maps a Private Endpoint groupId to the canonical
// Azure Private DNS zone name for that service sub-resource.
// Deterministic: only zones where a PE actually exists in a VNet are checked.
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

// checkPrivateDnsZoneMisconfiguration detects Private Endpoints whose VNet
// does not have the correct Private DNS Zone linked. Without the zone link,
// DNS queries from that VNet resolve via public DNS — the PE's network
// isolation guarantee is silently broken.
//
// The check is deterministic: it maps PE.GroupId → expected zone name, then
// verifies the zone is both present and linked to the PE's VNet. No false
// positives on VNets that simply do not contain PEs for that service.
func checkPrivateDnsZoneMisconfiguration(rg graph.ResourceGraph) []Finding {
	if len(rg.PrivateEndpoints) == 0 {
		return nil
	}
	// Index private DNS zones by name for O(1) lookup.
	zoneLinked := map[string]map[string]bool{} // zoneName → set of linked VNet names
	for _, z := range rg.PrivateDnsZones {
		if _, ok := zoneLinked[z.Name]; !ok {
			zoneLinked[z.Name] = map[string]bool{}
		}
		for _, v := range z.LinkedVnets {
			zoneLinked[z.Name][v] = true
		}
	}

	var findings []Finding
	for _, pe := range rg.PrivateEndpoints {
		if pe.ConnectionState != "Approved" && pe.ConnectionState != "" {
			continue // only approved connections are active; pending/rejected have no traffic
		}
		expectedZone, known := peGroupIdToZone[pe.GroupId]
		if !known {
			continue // unknown service — skip rather than guess
		}
		vnet := subnetToVnet(pe.Subnet)
		if vnet == "" {
			continue
		}
		linkedVnets, zoneExists := zoneLinked[expectedZone]
		if !zoneExists {
			findings = append(findings, Finding{
				Type:      "private DNS zone missing",
				Severity:  "High",
				Resource:  pe.Name,
				Evidence:  fmt.Sprintf("Private endpoint %q (service: %s) is in VNet %q but Private DNS zone %q does not exist in this subscription — DNS resolution will use public endpoints", pe.Name, pe.GroupId, vnet, expectedZone),
				Reachable: false,
			})
			continue
		}
		if !linkedVnets[vnet] {
			findings = append(findings, Finding{
				Type:      "private DNS zone not linked to VNet",
				Severity:  "High",
				Resource:  pe.Name,
				Evidence:  fmt.Sprintf("Private endpoint %q (service: %s) is in VNet %q but zone %q is not linked to that VNet — workloads in %q resolve this service via public DNS, bypassing the private endpoint", pe.Name, pe.GroupId, vnet, expectedZone, vnet),
				Reachable: false,
			})
		}
	}
	return findings
}

// checkAppGatewayExposure detects Application Gateways with a public IP but WAF
// disabled (or in detection-only mode). WAF is the primary L7 protection for
// public ingress; disabling it exposes backends to OWASP Top 10 attacks.
func checkAppGatewayExposure(rg graph.ResourceGraph) []Finding {
	var findings []Finding
	for _, gw := range rg.ApplicationGateways {
		if gw.PublicIP == "" {
			continue
		}
		if !gw.WafEnabled {
			findings = append(findings, Finding{
				Type:      "app gateway WAF disabled",
				Severity:  "Medium",
				Resource:  gw.Name,
				Evidence:  fmt.Sprintf("Application Gateway %q has public IP %s but WAF is disabled — no L7 protection on public ingress", gw.Name, gw.PublicIP),
				Reachable: true,
			})
		} else if strings.EqualFold(gw.WafMode, "Detection") {
			findings = append(findings, Finding{
				Type:      "app gateway WAF in detection mode",
				Severity:  "Informational",
				Resource:  gw.Name,
				Evidence:  fmt.Sprintf("Application Gateway %q WAF is enabled but in Detection mode — threats are logged but not blocked", gw.Name),
				Reachable: false,
			})
		}
	}
	return findings
}

// checkAKSExposure detects AKS clusters where the API server is not private.
// A non-private cluster has its Kubernetes API server reachable from the public
// internet; an authenticated client anywhere can attempt cluster control-plane access.
func checkAKSExposure(rg graph.ResourceGraph) []Finding {
	var findings []Finding
	for _, aks := range rg.AKSClusters {
		if !aks.IsPrivateCluster {
			findings = append(findings, Finding{
				Type:      "AKS non-private cluster",
				Severity:  "Medium",
				Resource:  aks.Name,
				Evidence:  fmt.Sprintf("AKS cluster %q is not a private cluster — API server is reachable from the public internet; use a private cluster with a private endpoint for production workloads", aks.Name),
				Reachable: true,
			})
		}
	}
	return findings
}

// checkCrossSubPeeringExposure detects cross-subscription VNet peerings where no
// hub firewall is in the traffic path. Direct cross-subscription peering without
// a firewall allows unrestricted lateral movement between subscriptions.
func checkCrossSubPeeringExposure(fx *graph.Fixture) []Finding {
	var findings []Finding
	for _, xp := range fx.CrossSubscriptionPeerings {
		if strings.EqualFold(xp.State, "Connected") && !xp.HasHubFirewall {
			findings = append(findings, Finding{
				Type:      "cross-subscription peering without firewall",
				Severity:  "Medium",
				Resource:  xp.LocalVnet + "~" + xp.RemoteVnet,
				Evidence:  fmt.Sprintf("VNet %q and %q (sub %s) are directly peered across subscriptions with no hub firewall in path — lateral movement between subscriptions is unrestricted", xp.LocalVnet, xp.RemoteVnet, xp.RemoteSubscriptionID),
				Reachable: false,
			})
		}
	}
	return findings
}

// checkLoadBalancerNAT detects NICs that are internet-reachable via External
// Load Balancer inbound NAT rules, even though the NIC itself has no public IP.
// An ELB with a public frontend + inbound NAT rules is functionally equivalent
// to the NIC having a public IP for inbound traffic — the same threat model as
// AzureFirewall DNAT. Gate 4 (hasPIP) misses this without explicit LB modeling.
func checkLoadBalancerNAT(rg graph.ResourceGraph) []Finding {
	var findings []Finding
	for _, lb := range rg.LoadBalancers {
		if lb.IsInternal || lb.FrontendIP == "" {
			continue // ILB does not expose to internet; skip
		}
		for _, nat := range lb.InboundNatRules {
			if nat.BackendNic == "" {
				continue
			}
			findings = append(findings, Finding{
				Type:      "internet reachable via load balancer NAT",
				Severity:  "High",
				Resource:  nat.BackendNic,
				Evidence:  fmt.Sprintf("load balancer %q NAT rule %q forwards public IP %s:%d → NIC %q:%d — NIC is internet-reachable without a direct public IP", lb.Name, nat.Name, lb.FrontendIP, nat.FrontendPort, nat.BackendNic, nat.BackendPort),
				Reachable: true,
			})
		}
	}
	return findings
}

// checkAPIMExposure detects API Management instances that expose the API gateway
// to the internet without adequate L7 protection.
//   - VNetMode="None": no VNet isolation — the gateway is publicly accessible and
//     backend calls to private services bypass all network controls.
//   - VNetMode="External": VNet-injected with a public endpoint — exposure is
//     reduced but the public gateway is unprotected if no WAF sits upstream.
//   - VNetMode="Internal": correctly locked down — no internet exposure, no finding.
func checkAPIMExposure(rg graph.ResourceGraph) []Finding {
	var findings []Finding
	for _, apim := range rg.APIManagements {
		switch apim.VNetMode {
		case "None":
			findings = append(findings, Finding{
				Type:      "APIM without VNet isolation",
				Severity:  "Medium",
				Resource:  apim.Name,
				Evidence:  fmt.Sprintf("API Management %q is deployed without VNet injection (mode=None) — gateway is publicly accessible and backend API calls bypass network controls", apim.Name),
				Reachable: true,
			})
		case "External":
			if !apim.HasWAFFrontEnd {
				findings = append(findings, Finding{
					Type:      "APIM External mode without WAF",
					Severity:  "Medium",
					Resource:  apim.Name,
					Evidence:  fmt.Sprintf("API Management %q is VNet-injected in External mode (public endpoint %s) with no WAF upstream — API traffic reaches the gateway without L7 inspection", apim.Name, apim.PublicIP),
					Reachable: true,
				})
			}
		}
	}
	return findings
}

// checkBastionBypass detects management-port exposure that circumvents Azure Bastion.
// When Bastion is deployed, it establishes a security contract: SSH/RDP to VMs should
// flow exclusively through Bastion, not via direct public IPs.
// A NIC with a public IP that has port 22 or 3389 permitted from the internet while
// Bastion is deployed is a bypass — the VM is reachable via a path Bastion was meant to eliminate.
func checkBastionBypass(rg graph.ResourceGraph, effRules map[string][]graph.SecRule) []Finding {
	if len(rg.AzureBastions) == 0 {
		return nil // Bastion not deployed; no contract to enforce
	}
	mgmtPorts := map[string]bool{"22": true, "3389": true}
	var findings []Finding
	for _, nic := range rg.NetworkInterfaces {
		if nic.PublicIP == nil || *nic.PublicIP == "" {
			continue
		}
		for _, r := range effRules[nic.Name] {
			if r.Direction != "Inbound" || r.Access != "Allow" {
				continue
			}
			if !isInternetSource(r.SourceAddressPrefix) {
				continue
			}
			if mgmtPorts[r.DestinationPortRange] {
				findings = append(findings, Finding{
					Type:      "Bastion bypass — direct management port exposed",
					Severity:  "High",
					Resource:  nic.Name,
					Evidence:  fmt.Sprintf("Azure Bastion is deployed but NIC %q has public IP %s with port %s open from internet — Bastion is intended to be the exclusive management ingress", nic.Name, *nic.PublicIP, r.DestinationPortRange),
					Reachable: true,
				})
				break // one finding per NIC is enough
			}
		}
	}
	return findings
}

// checkFrontDoorExposure detects Azure Front Door profiles where WAF protection is
// absent or in detection-only mode. Front Door is the outermost internet ingress point
// for many enterprise APIs and portals — WAF is the primary L7 defence at that layer.
//   - WafEnabled=false: no WAF policy associated with any endpoint → unprotected L7 ingress.
//   - WafMode="Detection": WAF logs threats but does not block them → exposure remains.
func checkFrontDoorExposure(rg graph.ResourceGraph) []Finding {
	var findings []Finding
	for _, fd := range rg.AzureFrontDoors {
		if !fd.WafEnabled {
			findings = append(findings, Finding{
				Type:      "Front Door WAF disabled",
				Severity:  "Medium",
				Resource:  fd.Name,
				Evidence:  fmt.Sprintf("Azure Front Door %q has no WAF policy enabled — all internet-facing endpoints lack L7 protection (OWASP Top 10, DDoS at app layer)", fd.Name),
				Reachable: true,
			})
		} else if strings.EqualFold(fd.WafMode, "Detection") {
			findings = append(findings, Finding{
				Type:      "Front Door WAF in detection mode",
				Severity:  "Informational",
				Resource:  fd.Name,
				Evidence:  fmt.Sprintf("Azure Front Door %q WAF is enabled but in Detection mode — threats are logged but not blocked; switch to Prevention for active protection", fd.Name),
				Reachable: false,
			})
		}
	}
	return findings
}

// In a vWAN topology, all spoke-to-spoke and spoke-to-internet traffic flows
// through the vHub. A vHub without a secured Azure Firewall (HasSecuredFirewall=false)
// means all inter-spoke traffic is forwarded without inspection — unrestricted
// lateral movement across all connected spokes.
//
// Additionally: if a secured vHub has RoutingPolicyPrivate=false, private
// (spoke-to-spoke) traffic bypasses the firewall even though it is present —
// the firewall only sees internet-bound traffic.
func checkVirtualWAN(rg graph.ResourceGraph) []Finding {
	var findings []Finding
	for _, wan := range rg.VirtualWANs {
		for _, hub := range wan.VHubs {
			if !hub.HasSecuredFirewall {
				findings = append(findings, Finding{
					Type:      "vWAN hub unsecured — no firewall",
					Severity:  "Medium",
					Resource:  hub.Name,
					Evidence:  fmt.Sprintf("Virtual WAN hub %q has %d spoke connection(s) but no secured Azure Firewall — all spoke-to-spoke and spoke-to-internet traffic is forwarded without inspection", hub.Name, len(hub.SpokeConnections)),
					Reachable: false,
				})
			} else if !hub.RoutingPolicyPrivate {
				findings = append(findings, Finding{
					Type:      "vWAN hub firewall bypasses private traffic",
					Severity:  "Medium",
					Resource:  hub.Name,
					Evidence:  fmt.Sprintf("Virtual WAN hub %q has a secured firewall but RoutingPolicyPrivate=false — spoke-to-spoke (east-west) traffic bypasses the firewall; only internet-bound traffic is inspected", hub.Name),
					Reachable: false,
				})
			}
		}
	}
	return findings
}
