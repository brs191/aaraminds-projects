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
		nics[n.Name] = n
	}

	for name, nic := range nics {
		rules := effRules[name]
		routes := effRoutes[name]
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
			findings = append(findings, Finding{"orphaned public endpoint", "Low", pip.Name,
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
		allowVnet, denyVnet := false, false
		for _, r := range effRules[name] {
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

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Resource != findings[j].Resource {
			return findings[i].Resource < findings[j].Resource
		}
		return findings[i].Type < findings[j].Type
	})
	return findings
}
