package analyze

import (
	"sort"
	"testing"

	"github.com/aaraminds/azure-nettopo-engine/internal/graph"
)

// V4-07 regression — findings keyed by ARM resource id (id || name).
// Two NICs with the SAME bare name in different subscriptions must NOT merge:
// the engine keys by id when present, so each produces its own finding. Golden
// fixtures (no id) are unaffected — asserted by analyze_test.go.
func twoSameNamedNICs() *graph.Fixture {
	pipA, pipB := "pipA", "pipB"
	return &graph.Fixture{
		ResourceGraph: graph.ResourceGraph{
			VirtualNetworks: []graph.VNet{
				{Name: "vA", AddressSpace: []string{"10.0.0.0/16"},
					Subnets: []graph.Subnet{{Name: "web", AddressPrefix: "10.0.1.0/24"}}},
				{Name: "vB", AddressSpace: []string{"10.1.0.0/16"},
					Subnets: []graph.Subnet{{Name: "web", AddressPrefix: "10.1.1.0/24"}}},
			},
			NetworkInterfaces: []graph.NIC{
				{Name: "nic-web", ID: "/subs/subA/nic-web", Subnet: "vA/web", PublicIP: &pipA, PrivateIP: "10.0.1.4",
					Tags: map[string]string{"sensitive": "true"}},
				{Name: "nic-web", ID: "/subs/subB/nic-web", Subnet: "vB/web", PublicIP: &pipB, PrivateIP: "10.1.1.4"},
			},
		},
		NetworkWatcher: graph.NetworkWatcher{
			EffectiveSecurityRules: map[string][]graph.SecRule{
				"/subs/subA/nic-web": {{Name: "a", Direction: "Inbound", Access: "Allow", SourceAddressPrefix: "0.0.0.0/0", DestinationPortRange: "443"}},
				"/subs/subB/nic-web": {{Name: "a", Direction: "Inbound", Access: "Allow", SourceAddressPrefix: "0.0.0.0/0", DestinationPortRange: "443"}},
			},
			EffectiveRoutes: map[string][]graph.Route{
				"/subs/subA/nic-web": {{AddressPrefix: "0.0.0.0/0", NextHopType: "Internet"}},
				"/subs/subB/nic-web": {{AddressPrefix: "0.0.0.0/0", NextHopType: "Internet"}},
			},
		},
	}
}

func TestResourceID_NoMergeAcrossSubscriptions(t *testing.T) {
	findings := Analyze(twoSameNamedNICs())
	sev := map[string]string{}
	for _, f := range findings {
		if f.Type == "over-permissive NSG (reachable)" {
			sev[f.Resource] = f.Severity
		}
	}
	if sev["/subs/subA/nic-web"] != "Critical" {
		t.Fatalf("subA NIC: want Critical, got %q (merged by name?) — all: %v", sev["/subs/subA/nic-web"], sev)
	}
	if sev["/subs/subB/nic-web"] != "High" {
		t.Fatalf("subB NIC: want High, got %q — all: %v", sev["/subs/subB/nic-web"], sev)
	}
}

func TestResourceID_DeterministicTotalOrder(t *testing.T) {
	a := Analyze(twoSameNamedNICs())
	b := Analyze(twoSameNamedNICs())
	if len(a) != len(b) {
		t.Fatalf("non-deterministic length: %d vs %d", len(a), len(b))
	}
	keys := make([][3]string, len(a))
	for i, f := range a {
		keys[i] = [3]string{f.Resource, f.Type, f.Evidence}
	}
	sorted := make([][3]string, len(keys))
	copy(sorted, keys)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i][0] != sorted[j][0] {
			return sorted[i][0] < sorted[j][0]
		}
		if sorted[i][1] != sorted[j][1] {
			return sorted[i][1] < sorted[j][1]
		}
		return sorted[i][2] < sorted[j][2]
	})
	for i := range keys {
		if keys[i] != sorted[i] {
			t.Fatalf("findings not in total (Resource,Type,Evidence) order at %d: %v", i, keys[i])
		}
	}
}

// Audit M-3: a NIC whose Network Watcher enrichment failed must surface as an
// explicit "analysis incomplete" finding, not vanish silently.
func TestAnalysisIncomplete_SurfacedAsFinding(t *testing.T) {
	fx := &graph.Fixture{
		ResourceGraph:  graph.ResourceGraph{NetworkInterfaces: []graph.NIC{{Name: "nic-a", Subnet: "v/s"}}},
		NetworkWatcher: graph.NetworkWatcher{IncompleteNICs: []string{"nic-dark"}},
	}
	found := false
	for _, f := range Analyze(fx) {
		if f.Type == "analysis incomplete" && f.Resource == "nic-dark" && f.Severity == "Medium" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected Medium 'analysis incomplete' finding for nic-dark (audit M-3)")
	}
}
