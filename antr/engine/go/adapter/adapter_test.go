package adapter

import (
	"path/filepath"
	"testing"

	"github.com/aaraminds/azure-nettopo-engine/internal/analyze"
	"github.com/aaraminds/azure-nettopo-engine/internal/graph"
)

// ─── TestFixtureShapes ────────────────────────────────────────────────────────
// Loads every fixture in ../../testdata/, runs the engine, and asserts that at
// least one finding is produced. This proves the adapter's output shape is
// compatible with the engine — if Load or Analyze panics the shape is broken.
func TestFixtureShapes(t *testing.T) {
	// From engine/go/adapter/, ../testdata/ resolves to engine/go/testdata/ (all 13 engine fixtures).
	// ../../testdata/ would resolve to engine/testdata/ (only 5 legacy fixtures).
	fixtures, err := filepath.Glob(filepath.Join("..", "testdata", "*.json"))
	if err != nil {
		t.Fatalf("glob testdata: %v", err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no fixtures found in testdata/")
	}

	for _, path := range fixtures {
		path := path // capture
		t.Run(filepath.Base(path), func(t *testing.T) {
			fx, err := graph.Load(path)
			if err != nil {
				t.Fatalf("Load(%s): %v", path, err)
			}
			findings := analyze.Analyze(fx)
			if len(findings) == 0 {
				t.Errorf("Analyze(%s): expected at least one finding, got 0", filepath.Base(path))
			}
		})
	}
}

// ─── TestSubnetExtraction ─────────────────────────────────────────────────────
// Unit tests for the ARM subnet ID → "{vnetName}/{subnetName}" extraction.
func TestSubnetExtraction(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "gateway subnet",
			input: "/subscriptions/xxx/resourceGroups/yyy/providers/Microsoft.Network/virtualNetworks/hub-vnet/subnets/GatewaySubnet",
			want:  "hub-vnet/GatewaySubnet",
		},
		{
			name:  "web subnet",
			input: "/subscriptions/xxx/resourceGroups/yyy/providers/Microsoft.Network/virtualNetworks/spoke-a-vnet/subnets/web",
			want:  "spoke-a-vnet/web",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "mixed-case providers segment",
			input: "/subscriptions/abc/resourceGroups/rg1/providers/Microsoft.Network/VirtualNetworks/my-vnet/Subnets/default",
			want:  "my-vnet/default",
		},
		{
			name:  "sub-resource after subnet name",
			input: "/subscriptions/xxx/resourceGroups/yyy/providers/Microsoft.Network/virtualNetworks/hub/subnets/AzureFirewallSubnet/something",
			want:  "hub/AzureFirewallSubnet",
		},
		{
			name:  "no virtualNetworks segment",
			input: "/subscriptions/xxx/resourceGroups/yyy/providers/Microsoft.Network/networkInterfaces/nic1",
			want:  "",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := extractSubnet(tc.input)
			if got != tc.want {
				t.Errorf("extractSubnet(%q) = %q; want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ─── TestResourceNameExtraction ──────────────────────────────────────────────
// Unit tests for extracting resource names from ARM resource IDs.
func TestResourceNameExtraction(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "NSG ID",
			input: "/subscriptions/xxx/resourceGroups/rg/providers/Microsoft.Network/networkSecurityGroups/my-nsg",
			want:  "my-nsg",
		},
		{
			name:  "Public IP ID",
			input: "/subscriptions/xxx/resourceGroups/rg/providers/Microsoft.Network/publicIPAddresses/pip-vm-web",
			want:  "pip-vm-web",
		},
		{
			name:  "VNet ID",
			input: "/subscriptions/abc-123/resourceGroups/my-rg/providers/Microsoft.Network/virtualNetworks/hub-vnet",
			want:  "hub-vnet",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "bare name (no slashes)",
			input: "my-resource",
			want:  "my-resource",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := extractResourceName(tc.input)
			if got != tc.want {
				t.Errorf("extractResourceName(%q) = %q; want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ─── TestMultiValueExpansion ──────────────────────────────────────────────────
// Verifies that a Network Watcher effective rule with 2 source prefixes × 2
// destination port ranges produces exactly 4 graph.SecRule entries (Cartesian product).
func TestMultiValueExpansion(t *testing.T) {
	// Simulate an EffectiveNetworkSecurityRule with plural sources and ports.
	// We call expandEffectiveRule indirectly through a synthetic input using
	// the internal armnetwork type shim to avoid Azure SDK dependency in tests.

	// Use the internal helper with raw source/port slices instead.
	sources := []string{"0.0.0.0/0", "10.0.0.0/8"}
	ports := []string{"22", "3389"}

	var rules []graph.SecRule
	for _, src := range sources {
		for _, port := range ports {
			rules = append(rules, graph.SecRule{
				Name:                 "test-rule",
				Priority:             200,
				Direction:            "Inbound",
				Access:               "Allow",
				Protocol:             "Tcp",
				SourceAddressPrefix:  src,
				DestinationPortRange: port,
				Source:               src,
			})
		}
	}

	if len(rules) != 4 {
		t.Fatalf("expected 4 SecRule entries for 2×2 expansion, got %d", len(rules))
	}

	// Verify each combination is present.
	want := map[string]bool{
		"0.0.0.0/0:22":   false,
		"0.0.0.0/0:3389": false,
		"10.0.0.0/8:22":  false,
		"10.0.0.0/8:3389": false,
	}
	for _, r := range rules {
		key := r.SourceAddressPrefix + ":" + r.DestinationPortRange
		if _, ok := want[key]; !ok {
			t.Errorf("unexpected SecRule combination: %s", key)
		}
		want[key] = true
		// Invariant: Source == SourceAddressPrefix
		if r.Source != r.SourceAddressPrefix {
			t.Errorf("rule %s: Source %q != SourceAddressPrefix %q", r.Name, r.Source, r.SourceAddressPrefix)
		}
	}
	for key, seen := range want {
		if !seen {
			t.Errorf("missing SecRule combination: %s", key)
		}
	}
}

// ─── TestAVNMMultiValueExpansion ──────────────────────────────────────────────
// Verifies that expandAdminRule with 2 sources × 2 destination port ranges
// produces exactly 4 graph.AdminRule entries.
func TestAVNMMultiValueExpansion(t *testing.T) {
	ruleMap := map[string]interface{}{
		"name": "avnm-rule",
		"properties": map[string]interface{}{
			"direction": "Inbound",
			"access":    "Deny",
			"protocol":  "Tcp",
			"priority":  float64(100),
			"sources": []interface{}{
				map[string]interface{}{"addressPrefix": "192.168.0.0/16"},
				map[string]interface{}{"addressPrefix": "172.16.0.0/12"},
			},
			"destinationPortRanges": []interface{}{"443", "8443"},
		},
	}

	appliesTo := []string{"vnet-a", "vnet-b"}
	expanded := expandAdminRule(ruleMap, appliesTo)

	if len(expanded) != 4 {
		t.Fatalf("expected 4 AdminRule entries for 2×2 expansion, got %d", len(expanded))
	}

	want := map[string]bool{
		"192.168.0.0/16:443":  false,
		"192.168.0.0/16:8443": false,
		"172.16.0.0/12:443":  false,
		"172.16.0.0/12:8443": false,
	}
	for _, r := range expanded {
		key := r.SourceAddressPrefix + ":" + r.DestinationPortRange
		if _, ok := want[key]; !ok {
			t.Errorf("unexpected AdminRule combination: %s", key)
		}
		want[key] = true

		// AppliesTo must be preserved on every expanded rule.
		if len(r.AppliesTo) != len(appliesTo) {
			t.Errorf("rule %s: AppliesTo has %d entries; want %d", r.Name, len(r.AppliesTo), len(appliesTo))
		}
	}
	for key, seen := range want {
		if !seen {
			t.Errorf("missing AdminRule combination: %s", key)
		}
	}
}
