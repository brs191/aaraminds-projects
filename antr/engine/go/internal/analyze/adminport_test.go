package analyze

import (
	"strings"
	"testing"

	"github.com/aaraminds/azure-nettopo-engine/internal/graph"
)

// External review F5 — DNAT must be evaluated across ALL firewalls, not just the
// first. Two firewalls each publishing a different no-public-IP backend must both
// produce a reachable DNAT finding.
func TestMultipleFirewalls_AllDNATPathsFound(t *testing.T) {
	fx := &graph.Fixture{
		ResourceGraph: graph.ResourceGraph{
			NetworkInterfaces: []graph.NIC{
				{Name: "nic-east", Subnet: "hub-east/workload", PrivateIP: "10.0.1.4"},
				{Name: "nic-west", Subnet: "hub-west/workload", PrivateIP: "10.1.1.4"},
			},
		},
		AzureFirewalls: []*graph.Firewall{
			{Name: "afw-east", PublicIP: "20.70.0.10", NatRules: []graph.NatRule{
				{Name: "dnat-e", Protocol: "Tcp", SourceAddresses: []string{"*"}, DestinationPort: 443, TranslatedAddress: "10.0.1.4", TranslatedPort: 443}}},
			{Name: "afw-west", PublicIP: "20.71.0.10", NatRules: []graph.NatRule{
				{Name: "dnat-w", Protocol: "Tcp", SourceAddresses: []string{"*"}, DestinationPort: 443, TranslatedAddress: "10.1.1.4", TranslatedPort: 443}}},
		},
	}
	hits := map[string]bool{}
	for _, f := range Analyze(fx) {
		if f.Type == "over-permissive NSG (reachable)" && strings.Contains(f.Evidence, "firewall DNAT") {
			hits[f.Resource] = true
		}
	}
	if !hits["nic-east"] || !hits["nic-west"] {
		t.Fatalf("both firewalls' DNAT paths must be found; got %v", hits)
	}
}

// External review F2 — AVNM admin rules with wildcard or range ports must govern
// the NSG port they cover. Unit-level coverage of the matching predicate.
func TestAdminPortCovers(t *testing.T) {
	cases := []struct {
		admin, nsg string
		want       bool
	}{
		{"*", "443", true},      // all-ports admin governs any NSG port
		{"443", "443", true},    // exact single port
		{"80-443", "443", true}, // range contains the port
		{"80-443", "80", true},  // lower bound inclusive
		{"80-443", "443", true}, // upper bound inclusive
		{"80-442", "443", false},
		{"80-443", "8080", false},
		{"443", "80", false},
		{"*", "*", true},    // all vs all
		{"443", "*", false}, // a specific admin port does not govern an all-ports NSG rule
		{"", "443", false},  // empty admin spec governs nothing
	}
	for _, c := range cases {
		if got := adminPortCovers(c.admin, c.nsg); got != c.want {
			t.Errorf("adminPortCovers(%q,%q)=%v want %v", c.admin, c.nsg, got, c.want)
		}
	}
}

// External review F7 — a rule merely NAMED like a deny must not suppress the
// missing-tier-segmentation finding unless it actually overrides AllowVnetInBound:
// inbound Deny, VNet-scoped, priority < 65000. A lower-precedence "DenyVnetInBound"
// (priority 65100) does not override, so the finding must still fire.
func TestSegmentation_LowerPrecedenceDenyDoesNotSuppress(t *testing.T) {
	mkFx := func(denyPriority int) *graph.Fixture {
		return &graph.Fixture{
			ResourceGraph: graph.ResourceGraph{
				VirtualNetworks: []graph.VNet{{Name: "svc-vnet", AddressSpace: []string{"10.60.0.0/16"},
					Subnets: []graph.Subnet{{Name: "secure", AddressPrefix: "10.60.1.0/24"}}}},
				NetworkInterfaces: []graph.NIC{{Name: "nic-secure", Subnet: "svc-vnet/secure",
					PrivateIP: "10.60.1.4", Tags: map[string]string{"sensitive": "true"}}},
			},
			NetworkWatcher: graph.NetworkWatcher{EffectiveSecurityRules: map[string][]graph.SecRule{
				"nic-secure": {
					{Name: "AllowVnetInBound", Priority: 65000, Direction: "Inbound", Access: "Allow", SourceAddressPrefix: "VirtualNetwork", DestinationPortRange: "*"},
					{Name: "DenyVnetInBound-custom", Priority: denyPriority, Direction: "Inbound", Access: "Deny", SourceAddressPrefix: "VirtualNetwork", DestinationPortRange: "*"},
				},
			}},
		}
	}
	has := func(fs []Finding) bool {
		for _, f := range fs {
			if f.Type == "missing tier segmentation" {
				return true
			}
		}
		return false
	}
	if !has(Analyze(mkFx(65100))) {
		t.Error("lower-precedence (65100) deny must NOT suppress segmentation finding")
	}
	if has(Analyze(mkFx(4000))) {
		t.Error("real higher-precedence (4000) deny SHOULD suppress segmentation finding")
	}
}

// Analyze-level: a Deny admin rule on "*" closes an otherwise-reachable NSG allow
// on 443, downgrading the finding from reachable (High) to latent (Informational).
func TestAdminWildcardDenyClosesInternet(t *testing.T) {
	pip := "pip-web"
	fx := &graph.Fixture{
		ResourceGraph: graph.ResourceGraph{
			VirtualNetworks: []graph.VNet{{Name: "svc-vnet", AddressSpace: []string{"10.50.0.0/16"},
				Subnets: []graph.Subnet{{Name: "web", AddressPrefix: "10.50.1.0/24"}}}},
			NetworkInterfaces: []graph.NIC{{Name: "nic-web", Subnet: "svc-vnet/web", PublicIP: &pip, PrivateIP: "10.50.1.4"}},
		},
		NetworkWatcher: graph.NetworkWatcher{
			EffectiveSecurityRules: map[string][]graph.SecRule{
				"nic-web": {{Name: "allow-https", Priority: 200, Direction: "Inbound", Access: "Allow",
					SourceAddressPrefix: "0.0.0.0/0", DestinationPortRange: "443"}},
			},
			EffectiveRoutes: map[string][]graph.Route{
				"nic-web": {{AddressPrefix: "0.0.0.0/0", NextHopType: "Internet"}},
			},
		},
		AVNM: graph.AVNM{SecurityAdminRules: []graph.AdminRule{
			{Name: "deny-all", Priority: 10, Direction: "Inbound", Access: "Deny",
				SourceAddressPrefix: "Internet", DestinationPortRange: "*", AppliesTo: []string{"svc-vnet"}},
		}},
	}
	for _, f := range Analyze(fx) {
		if f.Type == "over-permissive NSG (reachable)" {
			t.Fatalf("wildcard admin Deny should have closed the 443 flow; got reachable finding: %+v", f)
		}
	}
	// And it must still surface as a latent finding (not vanish).
	latent := false
	for _, f := range Analyze(fx) {
		if f.Type == "over-permissive NSG (latent)" && f.Resource == "nic-web" {
			latent = true
		}
	}
	if !latent {
		t.Fatal("expected a latent finding for nic-web after admin Deny closed the internet source")
	}
}
