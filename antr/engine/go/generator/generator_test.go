package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aaraminds/azure-nettopo-engine/internal/graph"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

// minimalSpec returns a TopologySpec that passes Validate() with the given
// VNets and FirewallEnabled flag. Callers override fields as needed.
func minimalSpec(vnets []VNetSpec, firewallEnabled bool) TopologySpec {
	return TopologySpec{
		SpecVersion:     "1.0",
		Description:     "test spec description (>= 10 chars)",
		Region:          "eastus2",
		VNets:           vnets,
		PeeringTopology: "none",
		GatewayType:     "none",
		FirewallEnabled: firewallEnabled,
		AVNMEnabled:     false,
		TierLabels:      []string{"web"},
		Tags:            map[string]string{"env": "test", "owner": "test", "costcenter": "test", "appid": "test"},
	}
}

func defaultRegistry() ModuleRegistry { return LoadDefaultRegistry() }

// findNICName returns the synthetic NIC name for the given VNet + subnet.
func findNICName(vnet, subnet string) string {
	return fmt.Sprintf("synthetic-nic-%s-%s", vnet, subnet)
}

// hasFindings returns true if any finding in findings matches all supplied predicates.
func hasFindings(findings []finding, predicates ...func(finding) bool) bool {
	for _, f := range findings {
		match := true
		for _, p := range predicates {
			if !p(f) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// finding is a local alias to avoid import collision in test helpers.
type finding = struct {
	Type      string
	Severity  string
	Resource  string
	Evidence  string
	Reachable bool
}

// ─── 1. TestIntentExpansion ───────────────────────────────────────────────────

// TestIntentExpansion verifies that every entry in the approved vocabulary expands
// to at least one SecRule without error. Includes both deny-all-inbound-other and
// its alias deny-all-inbound.
func TestIntentExpansion(t *testing.T) {
	allIntents := []string{
		"allow-https-from-internet",
		"allow-http-from-internet",
		"allow-ssh-from-bastion",
		"allow-rdp-from-bastion",
		"allow-bastion-rdp-ssh",
		"allow-internal-vnet",
		"allow-app-tier-only",
		"allow-appgw-management",
		"allow-lb-probes",
		"deny-internet-inbound",
		"deny-all-inbound-other",
		"deny-all-inbound", // alias for deny-all-inbound-other
		"deny-all-outbound-internet",
		"allow-azure-monitor",
		"allow-key-vault",
		"allow-storage",
	}

	for _, intent := range allIntents {
		t.Run(intent, func(t *testing.T) {
			rules, err := expandIntent(intent)
			if err != nil {
				t.Fatalf("expandIntent(%q) returned unexpected error: %v", intent, err)
			}
			if len(rules) == 0 {
				t.Fatalf("expandIntent(%q) returned empty rule slice", intent)
			}
		})
	}

	// Special case: allow-bastion-rdp-ssh must expand to exactly two rules
	t.Run("allow-bastion-rdp-ssh expands to 2 rules", func(t *testing.T) {
		rules, err := expandIntent("allow-bastion-rdp-ssh")
		if err != nil {
			t.Fatal(err)
		}
		if len(rules) != 2 {
			t.Fatalf("expected 2 rules for allow-bastion-rdp-ssh, got %d", len(rules))
		}
	})

	// Internet-sourced intents must carry SourceAddressPrefix == "Internet" (GR-003)
	internetIntents := []string{
		"allow-https-from-internet",
		"allow-http-from-internet",
		"deny-internet-inbound",
	}
	for _, intent := range internetIntents {
		t.Run(intent+" has Internet source", func(t *testing.T) {
			rules, _ := expandIntent(intent)
			for _, r := range rules {
				if r.SourceAddressPrefix != "Internet" {
					t.Fatalf("intent %q: expected SourceAddressPrefix=Internet, got %q", intent, r.SourceAddressPrefix)
				}
			}
		})
	}
}

// ─── 2. TestGateFail_SensitiveNICWithInternetIngress ─────────────────────────

// TestGateFail_SensitiveNICWithInternetIngress verifies that Gate 2+4 produces a
// Critical finding for a sensitive subnet with direct internet ingress (no firewall,
// no routeToFirewall). This is the key Phase 3 security gate.
func TestGateFail_SensitiveNICWithInternetIngress(t *testing.T) {
	spec := minimalSpec([]VNetSpec{{
		Name:         "vnet-sensitive",
		AddressSpace: []string{"10.0.0.0/16"},
		Subnets: []SubnetSpec{{
			Name:            "snet-web",
			AddressPrefix:   "10.0.1.0/24",
			TierLabel:       "web",
			Sensitive:       true,
			NSGIntents:      []string{"allow-https-from-internet"},
			RouteToFirewall: false,
		}},
	}}, false /* firewallEnabled */)

	plan, err := RenderTerraform(spec, defaultRegistry(), ProjectionBaseline{})
	if err != nil {
		t.Fatalf("RenderTerraform failed unexpectedly: %v", err)
	}

	result := ValidateBeforeEmit(plan)

	if result.Approved {
		t.Error("expected Approved=false for sensitive NIC with direct internet ingress, got Approved=true")
	}

	var hasCritical bool
	for _, f := range result.Findings {
		if f.Severity == "Critical" {
			hasCritical = true
			break
		}
	}
	if !hasCritical {
		t.Errorf("expected at least one Critical finding, got findings: %+v", result.Findings)
	}
}

// ─── 3. TestGatePass_SensitiveNICDenied ──────────────────────────────────────

// TestGatePass_SensitiveNICDenied verifies that Gate 2+4 passes when a sensitive
// subnet has deny intents and routes traffic through the firewall.
func TestGatePass_SensitiveNICDenied(t *testing.T) {
	spec := minimalSpec([]VNetSpec{{
		Name:         "vnet-secure",
		AddressSpace: []string{"10.0.0.0/16"},
		Subnets: []SubnetSpec{
			{
				// Required when firewallEnabled=true
				Name:          "AzureFirewallSubnet",
				AddressPrefix: "10.0.0.0/26",
				TierLabel:     "mgmt",
				Sensitive:     false,
				NSGIntents:    []string{},
			},
			{
				Name:            "snet-data",
				AddressPrefix:   "10.0.1.0/24",
				TierLabel:       "data",
				Sensitive:       true,
				NSGIntents:      []string{"deny-all-inbound", "deny-internet-inbound"},
				RouteToFirewall: true,
			},
		},
	}}, true /* firewallEnabled */)
	spec.TierLabels = []string{"data"}

	plan, err := RenderTerraform(spec, defaultRegistry(), ProjectionBaseline{})
	if err != nil {
		t.Fatalf("RenderTerraform failed unexpectedly: %v", err)
	}

	result := ValidateBeforeEmit(plan)

	if !result.Approved {
		blocking := []string{}
		for _, f := range result.Findings {
			if f.Severity == "Critical" || f.Severity == "High" {
				blocking = append(blocking, fmt.Sprintf("%s/%s: %s", f.Severity, f.Type, f.Evidence))
			}
		}
		t.Errorf("expected Approved=true, got Approved=false; blocking findings: %v", blocking)
	}
}

// ─── 4. TestCIDROverlap_Advisory ─────────────────────────────────────────────

// TestCIDROverlap_Advisory verifies that two peered VNets with overlapping CIDRs
// cause RenderTerraform to return an error before building the fixture.
func TestCIDROverlap_Advisory(t *testing.T) {
	spec := TopologySpec{
		SpecVersion:     "1.0",
		Description:     "cidr overlap test spec description",
		Region:          "eastus2",
		PeeringTopology: "mesh",
		GatewayType:     "none",
		FirewallEnabled: false,
		AVNMEnabled:     false,
		TierLabels:      []string{"web"},
		Tags:            map[string]string{"env": "test", "owner": "test", "costcenter": "test", "appid": "test"},
		VNets: []VNetSpec{
			{
				Name:         "vnet1",
				AddressSpace: []string{"10.0.0.0/16"},
				Subnets:      []SubnetSpec{{Name: "snet1", AddressPrefix: "10.0.1.0/24", TierLabel: "web", Sensitive: false, NSGIntents: []string{}}},
			},
			{
				Name:         "vnet2",
				AddressSpace: []string{"10.0.0.0/16"}, // same CIDR — overlap!
				Subnets:      []SubnetSpec{{Name: "snet1", AddressPrefix: "10.0.2.0/24", TierLabel: "web", Sensitive: false, NSGIntents: []string{}}},
			},
		},
	}

	_, err := RenderTerraform(spec, defaultRegistry(), ProjectionBaseline{})
	if err == nil {
		t.Fatal("expected error for overlapping CIDRs in peered VNets, got nil")
	}
	if !strings.Contains(err.Error(), "CIDR overlap") && !strings.Contains(err.Error(), "overlap") {
		t.Errorf("expected error message to mention CIDR overlap, got: %v", err)
	}
}

// ─── 5. TestUnknownNSGIntent ──────────────────────────────────────────────────

// TestUnknownNSGIntent verifies that an intent outside the approved vocabulary
// causes RenderTerraform to return ErrUnknownNSGIntent, not panic.
func TestUnknownNSGIntent(t *testing.T) {
	spec := minimalSpec([]VNetSpec{{
		Name:         "vnet-test",
		AddressSpace: []string{"10.0.0.0/16"},
		Subnets: []SubnetSpec{{
			Name:          "snet-web",
			AddressPrefix: "10.0.1.0/24",
			TierLabel:     "web",
			Sensitive:     false,
			NSGIntents:    []string{"allow-everything"}, // not in vocabulary
		}},
	}}, false)

	_, err := RenderTerraform(spec, defaultRegistry(), ProjectionBaseline{})
	if err == nil {
		t.Fatal("expected ErrUnknownNSGIntent, got nil")
	}

	var nsgErr ErrUnknownNSGIntent
	// Check type assertion
	if e, ok := err.(ErrUnknownNSGIntent); ok {
		nsgErr = e
	} else {
		// expandIntent might be wrapped; check the message
		if !strings.Contains(err.Error(), "allow-everything") {
			t.Errorf("expected error to mention the unknown intent, got: %v", err)
		}
		return
	}

	if nsgErr.Intent != "allow-everything" {
		t.Errorf("expected Intent=%q, got %q", "allow-everything", nsgErr.Intent)
	}
}

// ─── 6. TestSpecHash_Deterministic ───────────────────────────────────────────

// TestSpecHash_Deterministic verifies that rendering the same spec twice produces
// identical SpecHash values (pure function, no randomness or timestamps).
func TestSpecHash_Deterministic(t *testing.T) {
	spec := minimalSpec([]VNetSpec{{
		Name:         "vnet-hash",
		AddressSpace: []string{"10.0.0.0/16"},
		Subnets: []SubnetSpec{{
			Name:          "snet-web",
			AddressPrefix: "10.0.1.0/24",
			TierLabel:     "web",
			Sensitive:     false,
			NSGIntents:    []string{"allow-https-from-internet", "deny-all-inbound-other"},
		}},
	}}, false)

	plan1, err1 := RenderTerraform(spec, defaultRegistry(), ProjectionBaseline{})
	plan2, err2 := RenderTerraform(spec, defaultRegistry(), ProjectionBaseline{})

	if err1 != nil || err2 != nil {
		t.Fatalf("RenderTerraform errors: %v / %v", err1, err2)
	}

	if plan1.SpecHash == "" {
		t.Fatal("SpecHash must not be empty")
	}
	if plan1.SpecHash != plan2.SpecHash {
		t.Errorf("SpecHash not deterministic: render1=%q, render2=%q", plan1.SpecHash, plan2.SpecHash)
	}

	// Hash must be 64 hex chars (SHA-256)
	if len(plan1.SpecHash) != 64 {
		t.Errorf("expected 64-char SHA-256 hex, got len=%d: %q", len(plan1.SpecHash), plan1.SpecHash)
	}
}

// ─── 7. TestDefaultRoute_Firewall ────────────────────────────────────────────

// TestDefaultRoute_Firewall verifies that when firewallEnabled=true and
// routeToFirewall=true, the synthetic NIC's effective route is VirtualAppliance.
func TestDefaultRoute_Firewall(t *testing.T) {
	spec := minimalSpec([]VNetSpec{{
		Name:         "vnet-fw",
		AddressSpace: []string{"10.0.0.0/16"},
		Subnets: []SubnetSpec{
			{
				Name:          "AzureFirewallSubnet",
				AddressPrefix: "10.0.0.0/26",
				TierLabel:     "mgmt",
				Sensitive:     false,
				NSGIntents:    []string{},
			},
			{
				Name:            "snet-app",
				AddressPrefix:   "10.0.1.0/24",
				TierLabel:       "app",
				Sensitive:       false,
				NSGIntents:      []string{"allow-internal-vnet"},
				RouteToFirewall: true,
			},
		},
	}}, true /* firewallEnabled */)
	spec.TierLabels = []string{"app"}

	plan, err := RenderTerraform(spec, defaultRegistry(), ProjectionBaseline{})
	if err != nil {
		t.Fatalf("RenderTerraform error: %v", err)
	}

	nicName := findNICName("vnet-fw", "snet-app")
	routes, ok := plan.FixtureProjection.NetworkWatcher.EffectiveRoutes[nicName]
	if !ok {
		t.Fatalf("no EffectiveRoutes for NIC %q; all keys: %v",
			nicName, routeKeys(plan.FixtureProjection))
	}

	var defaultRoute *graph.Route
	for i := range routes {
		if routes[i].AddressPrefix == "0.0.0.0/0" {
			defaultRoute = &routes[i]
			break
		}
	}
	if defaultRoute == nil {
		t.Fatal("no 0.0.0.0/0 route found in EffectiveRoutes")
	}
	if defaultRoute.NextHopType != "VirtualAppliance" {
		t.Errorf("expected NextHopType=VirtualAppliance, got %q", defaultRoute.NextHopType)
	}
}

// ─── 8. TestDefaultRoute_Internet ────────────────────────────────────────────

// TestDefaultRoute_Internet verifies that when firewallEnabled=false and
// routeToFirewall=false, the synthetic NIC routes to Internet, and the engine
// fires a non-Critical internet-reachable finding.
func TestDefaultRoute_Internet(t *testing.T) {
	spec := minimalSpec([]VNetSpec{{
		Name:         "vnet-open",
		AddressSpace: []string{"10.0.0.0/16"},
		Subnets: []SubnetSpec{{
			Name:            "snet-web",
			AddressPrefix:   "10.0.1.0/24",
			TierLabel:       "web",
			Sensitive:       false, // NOT sensitive — finding will be High, not Critical
			NSGIntents:      []string{"allow-https-from-internet"},
			RouteToFirewall: false,
		}},
	}}, false /* firewallEnabled */)

	plan, err := RenderTerraform(spec, defaultRegistry(), ProjectionBaseline{})
	if err != nil {
		t.Fatalf("RenderTerraform error: %v", err)
	}

	// Assert route is Internet
	nicName := findNICName("vnet-open", "snet-web")
	routes, ok := plan.FixtureProjection.NetworkWatcher.EffectiveRoutes[nicName]
	if !ok {
		t.Fatalf("no EffectiveRoutes for NIC %q", nicName)
	}
	var defaultRoute *graph.Route
	for i := range routes {
		if routes[i].AddressPrefix == "0.0.0.0/0" {
			defaultRoute = &routes[i]
			break
		}
	}
	if defaultRoute == nil {
		t.Fatal("no 0.0.0.0/0 route")
	}
	if defaultRoute.NextHopType != "Internet" {
		t.Errorf("expected NextHopType=Internet, got %q", defaultRoute.NextHopType)
	}

	// Assert engine fires an internet-reachable finding (High, not Critical)
	result := ValidateBeforeEmit(plan)
	var hasFinding bool
	for _, f := range result.Findings {
		if f.Reachable && f.Severity != "Critical" {
			hasFinding = true
			break
		}
	}
	if !hasFinding {
		t.Errorf("expected at least one non-Critical reachable finding; got: %+v", result.Findings)
	}
}

// ─── 9. TestAVNMBaseline_CarriedThrough ──────────────────────────────────────

// TestAVNMBaseline_CarriedThrough verifies GR-001: ProjectionBaseline.AVNMSecurityAdminRules
// is copied into FixtureProjection.AVNM.SecurityAdminRules (not discarded).
func TestAVNMBaseline_CarriedThrough(t *testing.T) {
	adminRules := []graph.AdminRule{
		{
			Name:                 "block-ssh-internet",
			Priority:             100,
			Direction:            "Inbound",
			Access:               "Deny",
			Protocol:             "TCP",
			SourceAddressPrefix:  "Internet",
			DestinationPortRange: "22",
			AppliesTo:            []string{"vnet-avnm"},
		},
	}
	baseline := ProjectionBaseline{AVNMSecurityAdminRules: adminRules}

	spec := minimalSpec([]VNetSpec{{
		Name:         "vnet-avnm",
		AddressSpace: []string{"10.0.0.0/16"},
		Subnets: []SubnetSpec{{
			Name:          "snet-web",
			AddressPrefix: "10.0.1.0/24",
			TierLabel:     "web",
			Sensitive:     false,
			NSGIntents:    []string{"allow-https-from-internet"},
		}},
	}}, false)

	plan, err := RenderTerraform(spec, defaultRegistry(), baseline)
	if err != nil {
		t.Fatalf("RenderTerraform error: %v", err)
	}

	got := plan.FixtureProjection.AVNM.SecurityAdminRules
	if len(got) == 0 {
		t.Fatal("expected AVNM.SecurityAdminRules to be populated from baseline, got empty")
	}
	if got[0].Name != "block-ssh-internet" {
		t.Errorf("expected first rule name %q, got %q", "block-ssh-internet", got[0].Name)
	}
}

// ─── 10. TestSyntheticPIP_InternetIngress ─────────────────────────────────────

// TestSyntheticPIP_InternetIngress verifies GR-002: a synthetic NIC with
// allow-https-from-internet and no routeToFirewall MUST get a non-nil PublicIP.
func TestSyntheticPIP_InternetIngress(t *testing.T) {
	spec := minimalSpec([]VNetSpec{{
		Name:         "vnet-pip",
		AddressSpace: []string{"10.0.0.0/16"},
		Subnets: []SubnetSpec{{
			Name:            "snet-web",
			AddressPrefix:   "10.0.1.0/24",
			TierLabel:       "web",
			Sensitive:       false,
			NSGIntents:      []string{"allow-https-from-internet"},
			RouteToFirewall: false,
		}},
	}}, false)

	plan, err := RenderTerraform(spec, defaultRegistry(), ProjectionBaseline{})
	if err != nil {
		t.Fatalf("RenderTerraform error: %v", err)
	}

	nicName := findNICName("vnet-pip", "snet-web")
	var nic *graph.NIC
	for i := range plan.FixtureProjection.ResourceGraph.NetworkInterfaces {
		if plan.FixtureProjection.ResourceGraph.NetworkInterfaces[i].Name == nicName {
			n := plan.FixtureProjection.ResourceGraph.NetworkInterfaces[i]
			nic = &n
			break
		}
	}
	if nic == nil {
		t.Fatalf("synthetic NIC %q not found in fixture", nicName)
	}
	if nic.PublicIP == nil || *nic.PublicIP == "" {
		t.Error("expected non-nil PublicIP on internet-ingress NIC (GR-002), got nil")
	}

	// Also verify a PublicIPAddress entry exists with non-empty IPConfiguration
	var pipFound bool
	for _, pip := range plan.FixtureProjection.ResourceGraph.PublicIPAddresses {
		if pip.IPConfiguration != nil && *pip.IPConfiguration == nicName {
			pipFound = true
			break
		}
	}
	if !pipFound {
		t.Errorf("expected a PublicIPAddress with IPConfiguration=%q, none found", nicName)
	}
}

// ─── 11. TestNilFixtureProjection_ValidationFails ────────────────────────────

// TestNilFixtureProjection_ValidationFails verifies the guard in ValidateBeforeEmit:
// a TerraformPlan with FixtureProjection==nil returns Approved=false with a High finding.
func TestNilFixtureProjection_ValidationFails(t *testing.T) {
	plan := TerraformPlan{
		Files:             map[string]string{"versions.tf": "# empty"},
		SpecHash:          "deadbeef",
		FixtureProjection: nil,
	}

	result := ValidateBeforeEmit(plan)

	if result.Approved {
		t.Error("expected Approved=false for nil FixtureProjection, got Approved=true")
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected at least one finding, got empty")
	}
	if result.Findings[0].Severity != "High" {
		t.Errorf("expected synthetic finding severity=High, got %q", result.Findings[0].Severity)
	}
	if result.Findings[0].Type != "projection-error" {
		t.Errorf("expected finding type=projection-error, got %q", result.Findings[0].Type)
	}
}

// ─── 12. TestRegistryVersionPin ──────────────────────────────────────────────

// TestRegistryVersionPin verifies that every entry in LoadDefaultRegistry() has a
// pinned (exact) version — no ">=", "~>", or "*" constraints.
func TestRegistryVersionPin(t *testing.T) {
	registry := LoadDefaultRegistry()
	unpinnedMarkers := []string{">=", "~>", "*"}

	for _, entry := range registry.entries {
		for _, marker := range unpinnedMarkers {
			if strings.Contains(entry.Version, marker) {
				t.Errorf("module %q has unpinned version %q (contains %q)", entry.ID, entry.Version, marker)
			}
		}
		if entry.Version == "" {
			t.Errorf("module %q has empty version", entry.ID)
		}
	}

	if len(registry.entries) != 12 {
		t.Errorf("expected 12 modules in default registry, got %d", len(registry.entries))
	}
}

// ─── 13. TestRegistryFileLoad ─────────────────────────────────────────────────

// TestRegistryFileLoad verifies that LoadRegistryFromFile loads valid YAML and
// rejects registry files with unpinned versions.
func TestRegistryFileLoad(t *testing.T) {
	dir := t.TempDir()

	t.Run("valid YAML loads successfully", func(t *testing.T) {
		content := `modules:
  - id: az-nsg
    source: Azure/network-security-group/azurerm
    version: "4.1.0"
    purpose: "NSG + security rules from intent vocabulary"
    handles:
      - nsg
    required_inputs:
      - resource_group_name
      - security_group_name
      - location
    notes: "test entry"
`
		path := filepath.Join(dir, "registry-valid.yaml")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		reg, err := LoadRegistryFromFile(path)
		if err != nil {
			t.Fatalf("expected no error for valid registry, got: %v", err)
		}
		entry, ok := reg.Select("nsg")
		if !ok {
			t.Fatal("expected to find entry with capability 'nsg'")
		}
		if entry.ID != "az-nsg" {
			t.Errorf("expected ID=az-nsg, got %q", entry.ID)
		}
		if entry.Version != "4.1.0" {
			t.Errorf("expected version=4.1.0, got %q", entry.Version)
		}
	})

	t.Run("unpinned version is rejected", func(t *testing.T) {
		content := `modules:
  - id: az-nsg
    source: Azure/network-security-group/azurerm
    version: ">= 1.0"
    purpose: "test"
    handles:
      - nsg
`
		path := filepath.Join(dir, "registry-unpinned.yaml")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := LoadRegistryFromFile(path)
		if err == nil {
			t.Fatal("expected error for unpinned version '>= 1.0', got nil")
		}
		if !strings.Contains(err.Error(), "unpinned") && !strings.Contains(err.Error(), ">=") {
			t.Errorf("expected error to mention unpinned version, got: %v", err)
		}
	})

	t.Run("tilde constraint is rejected", func(t *testing.T) {
		content := `modules:
  - id: az-nsg
    source: Azure/network-security-group/azurerm
    version: "~> 4.1"
    purpose: "test"
    handles:
      - nsg
`
		path := filepath.Join(dir, "registry-tilde.yaml")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := LoadRegistryFromFile(path)
		if err == nil {
			t.Fatal("expected error for tilde constraint '~> 4.1', got nil")
		}
	})

	t.Run("JSON format also loads", func(t *testing.T) {
		content := `{"modules": [{"id": "az-nsg", "source": "Azure/network-security-group/azurerm", "version": "4.1.0", "purpose": "test", "handles": ["nsg"]}]}`
		path := filepath.Join(dir, "registry.json")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		reg, err := LoadRegistryFromFile(path)
		if err != nil {
			t.Fatalf("expected no error for valid JSON registry, got: %v", err)
		}
		if _, ok := reg.Select("nsg"); !ok {
			t.Fatal("expected to find 'nsg' capability in JSON registry")
		}
	})
}

// ─── Additional edge-case tests ───────────────────────────────────────────────

// TestPrivateIPOffset verifies the synthetic private IP derivation adds offset 5.
func TestPrivateIPOffset(t *testing.T) {
	cases := []struct {
		cidr string
		want string
	}{
		{"10.0.1.0/24", "10.0.1.5"},
		{"10.3.2.0/24", "10.3.2.5"},
		{"192.168.0.0/24", "192.168.0.5"},
		{"172.16.0.0/20", "172.16.0.5"},
	}
	for _, tc := range cases {
		got := syntheticPrivateIP(tc.cidr)
		if got != tc.want {
			t.Errorf("syntheticPrivateIP(%q) = %q, want %q", tc.cidr, got, tc.want)
		}
	}
}

// TestPEDnsZoneProjection verifies GR-004: PrivateEndpointSpec with a known GroupID
// results in the correct PrivateDnsZone being linked to the hosting VNet.
func TestPEDnsZoneProjection(t *testing.T) {
	spec := TopologySpec{
		SpecVersion:     "1.0",
		Description:     "PE DNS zone projection test spec",
		Region:          "eastus2",
		PeeringTopology: "none",
		GatewayType:     "none",
		FirewallEnabled: false,
		AVNMEnabled:     false,
		TierLabels:      []string{"pe"},
		Tags:            map[string]string{"env": "test", "owner": "test", "costcenter": "test", "appid": "test"},
		VNets: []VNetSpec{{
			Name:         "vnet-pe",
			AddressSpace: []string{"10.0.0.0/16"},
			Subnets: []SubnetSpec{{
				Name:                  "snet-pe",
				AddressPrefix:         "10.0.1.0/24",
				TierLabel:             "pe",
				Sensitive:             false,
				NSGIntents:            []string{},
				PrivateEndpointSubnet: true,
				PrivateEndpoints: []PrivateEndpointSpec{{
					Name:              "pe-blob",
					GroupID:           "blob",
					ServiceResourceID: "/subscriptions/xxx/resourceGroups/rg/providers/Microsoft.Storage/storageAccounts/mystorage",
				}},
			}},
		}},
	}

	plan, err := RenderTerraform(spec, defaultRegistry(), ProjectionBaseline{})
	if err != nil {
		t.Fatalf("RenderTerraform error: %v", err)
	}

	zones := plan.FixtureProjection.ResourceGraph.PrivateDnsZones
	if len(zones) == 0 {
		t.Fatal("expected at least one PrivateDnsZone, got none")
	}

	var blobZone *graph.PrivateDnsZone
	for i := range zones {
		if zones[i].Name == "privatelink.blob.core.windows.net" {
			blobZone = &zones[i]
			break
		}
	}
	if blobZone == nil {
		t.Fatalf("expected zone privatelink.blob.core.windows.net, got: %v", zoneNames(zones))
	}

	var linked bool
	for _, vn := range blobZone.LinkedVnets {
		if vn == "vnet-pe" {
			linked = true
			break
		}
	}
	if !linked {
		t.Errorf("expected vnet-pe in LinkedVnets for blob zone, got: %v", blobZone.LinkedVnets)
	}
}

// TestHCL_SourcesFromRegistry verifies that all module source strings in generated
// HCL come from the registry and not from TopologySpec fields.
func TestHCL_SourcesFromRegistry(t *testing.T) {
	spec := minimalSpec([]VNetSpec{{
		Name:         "vnet-hcl",
		AddressSpace: []string{"10.0.0.0/16"},
		Subnets: []SubnetSpec{{
			Name:          "snet-web",
			AddressPrefix: "10.0.1.0/24",
			TierLabel:     "web",
			Sensitive:     false,
			NSGIntents:    []string{"allow-https-from-internet"},
		}},
	}}, false)

	plan, err := RenderTerraform(spec, defaultRegistry(), ProjectionBaseline{})
	if err != nil {
		t.Fatalf("RenderTerraform error: %v", err)
	}

	// Collect all approved module sources from the registry
	reg := defaultRegistry()
	approvedSources := make(map[string]bool, len(reg.entries))
	for _, e := range reg.entries {
		approvedSources[e.Source] = true
	}

	// Verify versions.tf and main.tf exist
	if _, ok := plan.Files["versions.tf"]; !ok {
		t.Error("versions.tf not generated")
	}
	if _, ok := plan.Files["main.tf"]; !ok {
		t.Error("main.tf not generated")
	}
	if _, ok := plan.Files["nsg.tf"]; !ok {
		t.Error("nsg.tf not generated")
	}

	// All module source = "..." values in generated HCL must be registry sources.
	// We match only the exact `source  =` attribute (not source_address_prefix, etc.).
	for filename, content := range plan.Files {
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			// Must start with "source " or "source=" but NOT "source_"
			if !strings.HasPrefix(trimmed, "source ") && !strings.HasPrefix(trimmed, "source=") {
				continue
			}
			if strings.HasPrefix(trimmed, "source_") {
				continue // e.g. source_address_prefix
			}
			// Extract the source value
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) < 2 {
				continue
			}
			src := strings.TrimSpace(parts[1])
			src = strings.Trim(src, `"`)
			// Skip hashicorp/azurerm (provider, not module)
			if src == "hashicorp/azurerm" {
				continue
			}
			if !approvedSources[src] {
				t.Errorf("file %s: found unapproved module source %q (not in registry)", filename, src)
			}
		}
	}
}

// ─── helpers used by tests ────────────────────────────────────────────────────

func routeKeys(fixture *graph.Fixture) []string {
	if fixture == nil {
		return nil
	}
	keys := make([]string, 0, len(fixture.NetworkWatcher.EffectiveRoutes))
	for k := range fixture.NetworkWatcher.EffectiveRoutes {
		keys = append(keys, k)
	}
	return keys
}

func zoneNames(zones []graph.PrivateDnsZone) []string {
	names := make([]string, len(zones))
	for i, z := range zones {
		names[i] = z.Name
	}
	return names
}

// Audit H-3: generated infrastructure with a Medium-severity security finding
// (here: two peered VNets with overlapping CIDR) must NOT be approved for auto-PR.
func TestValidateBeforeEmit_MediumBlocks(t *testing.T) {
	fx := &graph.Fixture{ResourceGraph: graph.ResourceGraph{VirtualNetworks: []graph.VNet{
		{Name: "vnet-a", AddressSpace: []string{"10.50.0.0/16"}},
		{Name: "vnet-b", AddressSpace: []string{"10.50.0.0/16"}},
	}}}
	res := ValidateBeforeEmit(TerraformPlan{FixtureProjection: fx})
	if res.Approved {
		t.Fatalf("Medium CIDR-overlap must block generation (audit H-3); findings=%v", res.Findings)
	}
}
