package simulator_test

import (
	"testing"

	"github.com/aaraminds/azure-nettopo-engine/internal/analyze"
	"github.com/aaraminds/azure-nettopo-engine/internal/graph"
	"github.com/aaraminds/azure-nettopo-engine/simulator"
)

// ---- fixture helpers ----

// pip attaches the named public IP to a NIC.
func pip(name string) *string { return &name }

// baseFixture returns a minimal topology with one VNet, one NSG, one NIC
// reachable from the internet via SSH.
//
//	VNet: vnet-a (10.0.0.0/16)
//	  Subnet: sub-a, NSG: nsg-a, RouteTable: rt-a
//	NSG: nsg-a — allow-ssh (0.0.0.0/0:22 Inbound Allow)
//	RouteTable: rt-a — default 0.0.0.0/0 → Internet
//	NIC: nic-a, Subnet: vnet-a/sub-a, PublicIP: pip-a
//	Effective rules for nic-a: [allow-ssh, DenyAllInBound]
//	Effective routes for nic-a: [0.0.0.0/0 → Internet]
func baseFixture() *graph.Fixture {
	allowSSH := graph.SecRule{
		Name: "allow-ssh", Priority: 200, Direction: "Inbound", Access: "Allow",
		Protocol: "Tcp", SourceAddressPrefix: "0.0.0.0/0", DestinationPortRange: "22",
	}
	denyAll := graph.SecRule{
		Name: "DenyAllInBound", Priority: 65500, Direction: "Inbound", Access: "Deny",
	}
	nicName := "pip-a"
	return &graph.Fixture{
		Subscription: "sub-test",
		ResourceGraph: graph.ResourceGraph{
			VirtualNetworks: []graph.VNet{
				{
					Name:         "vnet-a",
					AddressSpace: []string{"10.0.0.0/16"},
					Subnets: []graph.Subnet{
						{Name: "sub-a", AddressPrefix: "10.0.1.0/24",
							NetworkSecurityGroup: "nsg-a", RouteTable: "rt-a"},
					},
				},
			},
			NetworkSecurityGroups: []graph.NSG{
				{Name: "nsg-a", SecurityRules: []graph.SecRule{allowSSH},
					AssociatedSubnets: []string{"vnet-a/sub-a"}},
			},
			RouteTables: []graph.RouteTable{
				{Name: "rt-a",
					Routes: []graph.Route{
						{Name: "default", AddressPrefix: "0.0.0.0/0", NextHopType: "Internet"},
					},
					AssociatedSubnets: []string{"vnet-a/sub-a"}},
			},
			PublicIPAddresses: []graph.PublicIP{
				{Name: "pip-a", IPAddress: "20.1.1.1", IPConfiguration: &nicName},
			},
			NetworkInterfaces: []graph.NIC{
				{Name: "nic-a", Subnet: "vnet-a/sub-a", PublicIP: pip("pip-a"), PrivateIP: "10.0.1.4"},
			},
		},
		NetworkWatcher: graph.NetworkWatcher{
			EffectiveSecurityRules: map[string][]graph.SecRule{
				"nic-a": {allowSSH, denyAll},
			},
			EffectiveRoutes: map[string][]graph.Route{
				"nic-a": {{AddressPrefix: "0.0.0.0/0", NextHopType: "Internet"}},
			},
		},
		AVNM: graph.AVNM{},
	}
}

// sensitiveNICFixture returns a topology with a sensitive NIC that is reachable
// (has PIP + internet route + open NSG) — baseline produces a Critical finding.
func sensitiveNICFixture() *graph.Fixture {
	fx := baseFixture()
	fx.ResourceGraph.NetworkInterfaces[0].Tags = map[string]string{"sensitive": "true"}
	return fx
}

// noPublicIPFixture returns a topology identical to baseFixture but without the PIP.
// Baseline produces a latent/Informational finding (NSG open but not reachable).
func noPublicIPFixture() *graph.Fixture {
	fx := baseFixture()
	fx.ResourceGraph.NetworkInterfaces[0].PublicIP = nil
	return fx
}

// nvaRouteFixture returns a topology where the default route goes through an NVA.
// Baseline: NSG is open but not internet-reachable (Gate 3 blocks).
func nvaRouteFixture() *graph.Fixture {
	fx := baseFixture()
	fx.NetworkWatcher.EffectiveRoutes["nic-a"] = []graph.Route{
		{AddressPrefix: "0.0.0.0/0", NextHopType: "VirtualAppliance", NextHopIPAddress: "10.0.0.4"},
	}
	fx.ResourceGraph.RouteTables[0].Routes[0] = graph.Route{
		Name: "default", AddressPrefix: "0.0.0.0/0",
		NextHopType: "VirtualAppliance", NextHopIPAddress: "10.0.0.4",
	}
	return fx
}

// twoPeerFixture returns two VNets, each with one NIC. vnet-a → vnet-b peering exists.
func twoPeerFixture() *graph.Fixture {
	fx := baseFixture()
	nicBName := "nic-b"
	_ = nicBName
	fx.ResourceGraph.VirtualNetworks = append(fx.ResourceGraph.VirtualNetworks, graph.VNet{
		Name:         "vnet-b",
		AddressSpace: []string{"10.1.0.0/16"},
		Subnets: []graph.Subnet{
			{Name: "sub-b", AddressPrefix: "10.1.1.0/24"},
		},
	})
	return fx
}

// ---- deep-copy immutability test ----

func TestApplyDelta_OriginalNotMutated(t *testing.T) {
	orig := baseFixture()
	origPIPName := *orig.ResourceGraph.NetworkInterfaces[0].PublicIP

	delta := simulator.TopologyDelta{
		RemovePublicIP: &simulator.RemovePublicIPOp{NICName: "nic-a"},
	}
	sim, err := simulator.ApplyDelta(orig, delta)
	if err != nil {
		t.Fatalf("ApplyDelta: %v", err)
	}

	// Simulated NIC must have no PIP.
	if sim.ResourceGraph.NetworkInterfaces[0].PublicIP != nil {
		t.Error("simulated NIC should have no public IP after RemovePublicIP")
	}
	// Original NIC must still have its PIP.
	if orig.ResourceGraph.NetworkInterfaces[0].PublicIP == nil {
		t.Error("original NIC must not be mutated")
	}
	if *orig.ResourceGraph.NetworkInterfaces[0].PublicIP != origPIPName {
		t.Errorf("original PIP name changed: got %q want %q",
			*orig.ResourceGraph.NetworkInterfaces[0].PublicIP, origPIPName)
	}
}

// ---- validate tests ----

func TestValidate_NoOp(t *testing.T) {
	err := simulator.TopologyDelta{}.Validate()
	if err == nil {
		t.Error("expected error for empty delta")
	}
}

func TestValidate_TwoOps(t *testing.T) {
	err := simulator.TopologyDelta{
		AddSubnet:  &simulator.AddSubnetOp{VNetName: "v", Name: "s", AddressPrefix: "10.0.0.0/24"},
		RemoveNSGRule: &simulator.RemoveNSGRuleOp{NSGName: "n", RuleName: "r"},
	}.Validate()
	if err == nil {
		t.Error("expected error for two ops set")
	}
}

func TestValidate_ModifyRouteUnknownHop(t *testing.T) {
	err := simulator.TopologyDelta{
		ModifyRoute: &simulator.ModifyRouteOp{
			RouteTableName: "rt", RouteName: "r", NewNextHopType: "Bogus",
		},
	}.Validate()
	if err == nil {
		t.Error("expected error for unknown NextHopType")
	}
}

// ---- AddNSGRule test — creates a new High finding ----

// TestAddNSGRule_CreatesHighFinding verifies that adding an allow-all-inbound
// rule to an NSG that governs a NIC with a public IP and Internet default route
// causes a new High finding in the SecurityDelta.
func TestAddNSGRule_CreatesHighFinding(t *testing.T) {
	// Baseline: NIC has no NSG rule open to internet (NSG is empty).
	fx := &graph.Fixture{
		Subscription: "sub-test",
		ResourceGraph: graph.ResourceGraph{
			VirtualNetworks: []graph.VNet{
				{Name: "vnet-a", AddressSpace: []string{"10.0.0.0/16"},
					Subnets: []graph.Subnet{
						{Name: "sub-a", AddressPrefix: "10.0.1.0/24",
							NetworkSecurityGroup: "nsg-a"},
					}},
			},
			NetworkSecurityGroups: []graph.NSG{
				{Name: "nsg-a", SecurityRules: []graph.SecRule{},
					AssociatedSubnets: []string{"vnet-a/sub-a"}},
			},
			RouteTables: []graph.RouteTable{},
			PublicIPAddresses: []graph.PublicIP{
				{Name: "pip-a", IPAddress: "20.1.1.1"},
			},
			NetworkInterfaces: []graph.NIC{
				{Name: "nic-a", Subnet: "vnet-a/sub-a", PublicIP: pip("pip-a"), PrivateIP: "10.0.1.4"},
			},
		},
		NetworkWatcher: graph.NetworkWatcher{
			EffectiveSecurityRules: map[string][]graph.SecRule{
				"nic-a": {
					{Name: "DenyAllInBound", Priority: 65500, Direction: "Inbound", Access: "Deny"},
				},
			},
			EffectiveRoutes: map[string][]graph.Route{
				"nic-a": {{AddressPrefix: "0.0.0.0/0", NextHopType: "Internet"}},
			},
		},
		AVNM: graph.AVNM{},
	}

	origFindings := analyze.Analyze(fx)
	// Baseline must have no reachable finding.
	for _, f := range origFindings {
		if f.Reachable && f.Resource == "nic-a" {
			t.Fatalf("baseline must not have a reachable finding for nic-a; got %+v", f)
		}
	}

	delta := simulator.TopologyDelta{
		AddNSGRule: &simulator.AddNSGRuleOp{
			NSGName: "nsg-a",
			Rule: graph.SecRule{
				Name: "allow-http", Priority: 100, Direction: "Inbound",
				Access: "Allow", Protocol: "Tcp",
				SourceAddressPrefix: "0.0.0.0/0", DestinationPortRange: "80",
			},
		},
	}

	simFx, err := simulator.ApplyDelta(fx, delta)
	if err != nil {
		t.Fatalf("ApplyDelta: %v", err)
	}

	simFindings := analyze.Analyze(simFx)
	sd := simulator.DiffFindings(origFindings, simFindings)

	// Must have at least one added High (or Critical) finding for nic-a.
	foundAdded := false
	for _, f := range sd.AddedRisks {
		if f.Resource == "nic-a" && (f.Severity == "High" || f.Severity == "Critical") && f.Reachable {
			foundAdded = true
		}
	}
	if !foundAdded {
		t.Errorf("expected added High/Critical finding for nic-a; AddedRisks=%+v", sd.AddedRisks)
	}
	if sd.RiskVector.HighDelta <= 0 && sd.RiskVector.CriticalDelta <= 0 {
		t.Errorf("expected positive HighDelta or CriticalDelta; RiskVector=%+v", sd.RiskVector)
	}
}

// ---- RemovePublicIP test — removes a reachable finding ----

// TestRemovePublicIP_RemovesReachableFinding verifies that detaching the PIP
// from a NIC that has an internet-reachable finding removes that finding
// from the SecurityDelta (MitigatedRisks).
func TestRemovePublicIP_RemovesReachableFinding(t *testing.T) {
	fx := baseFixture()

	origFindings := analyze.Analyze(fx)
	reachableInOrig := false
	for _, f := range origFindings {
		if f.Reachable && f.Resource == "nic-a" {
			reachableInOrig = true
		}
	}
	if !reachableInOrig {
		t.Fatal("baseFixture must have a reachable finding for nic-a")
	}

	delta := simulator.TopologyDelta{
		RemovePublicIP: &simulator.RemovePublicIPOp{NICName: "nic-a"},
	}

	simFx, err := simulator.ApplyDelta(fx, delta)
	if err != nil {
		t.Fatalf("ApplyDelta: %v", err)
	}

	simFindings := analyze.Analyze(simFx)
	sd := simulator.DiffFindings(origFindings, simFindings)

	// Must have at least one mitigated finding for nic-a.
	foundMit := false
	for _, f := range sd.MitigatedRisks {
		if f.Resource == "nic-a" && f.Reachable {
			foundMit = true
		}
	}
	if !foundMit {
		t.Errorf("expected mitigated reachable finding for nic-a; MitigatedRisks=%+v", sd.MitigatedRisks)
	}
	if sd.RiskVector.HighDelta >= 0 && sd.RiskVector.CriticalDelta >= 0 {
		// At least one severity bucket should be negative.
		t.Errorf("expected negative delta in High or Critical; RiskVector=%+v", sd.RiskVector)
	}
}

// ---- RemoveNSGRule test — removes an open rule that was blocking a finding ----

func TestRemoveNSGRule_NoFindingWithoutOpenRule(t *testing.T) {
	fx := baseFixture()
	origFindings := analyze.Analyze(fx)

	// Remove the allow-ssh rule — nic-a should no longer be reachable.
	delta := simulator.TopologyDelta{
		RemoveNSGRule: &simulator.RemoveNSGRuleOp{NSGName: "nsg-a", RuleName: "allow-ssh"},
	}
	simFx, err := simulator.ApplyDelta(fx, delta)
	if err != nil {
		t.Fatalf("ApplyDelta: %v", err)
	}
	simFindings := analyze.Analyze(simFx)
	sd := simulator.DiffFindings(origFindings, simFindings)

	for _, f := range sd.AddedRisks {
		if f.Reachable && f.Resource == "nic-a" {
			t.Errorf("unexpected added reachable finding for nic-a after RemoveNSGRule; finding=%+v", f)
		}
	}
}

// ---- ModifyRoute test — adding NVA route removes internet reachability ----

func TestModifyRoute_NVARouteMitigatesReachability(t *testing.T) {
	fx := baseFixture()

	origFindings := analyze.Analyze(fx)
	if !hasReachable(origFindings, "nic-a") {
		t.Fatal("baseFixture must have reachable finding for nic-a")
	}

	// Redirect default route to NVA — should mitigate the internet-reachable finding.
	delta := simulator.TopologyDelta{
		ModifyRoute: &simulator.ModifyRouteOp{
			RouteTableName: "rt-a",
			RouteName:      "default",
			NewNextHopType: "VirtualAppliance",
			NewNextHopIP:   "10.0.0.4",
		},
	}
	simFx, err := simulator.ApplyDelta(fx, delta)
	if err != nil {
		t.Fatalf("ApplyDelta: %v", err)
	}
	simFindings := analyze.Analyze(simFx)

	if hasReachable(simFindings, "nic-a") {
		t.Error("nic-a must NOT be reachable after default route → VirtualAppliance")
	}

	sd := simulator.DiffFindings(origFindings, simFindings)
	foundMit := false
	for _, f := range sd.MitigatedRisks {
		if f.Resource == "nic-a" && f.Reachable {
			foundMit = true
		}
	}
	if !foundMit {
		t.Errorf("expected mitigated reachable finding; MitigatedRisks=%+v", sd.MitigatedRisks)
	}
}

// ---- AddSubnet test — cost-only; SecurityDelta is empty ----

func TestAddSubnet_NoSecurityDelta(t *testing.T) {
	fx := baseFixture()
	origFindings := analyze.Analyze(fx)

	delta := simulator.TopologyDelta{
		AddSubnet: &simulator.AddSubnetOp{
			VNetName:      "vnet-a",
			Name:          "sub-new",
			AddressPrefix: "10.0.2.0/24",
			NSGName:       "nsg-a",
		},
	}
	simFx, err := simulator.ApplyDelta(fx, delta)
	if err != nil {
		t.Fatalf("ApplyDelta: %v", err)
	}

	// The new subnet must exist.
	found := false
	for _, s := range simFx.ResourceGraph.VirtualNetworks[0].Subnets {
		if s.Name == "sub-new" {
			found = true
		}
	}
	if !found {
		t.Error("new subnet sub-new should exist in simulated fixture")
	}

	// No new NICs → SecurityDelta should be zero.
	simFindings := analyze.Analyze(simFx)
	sd := simulator.DiffFindings(origFindings, simFindings)
	if len(sd.AddedRisks) != 0 || len(sd.MitigatedRisks) != 0 {
		t.Errorf("AddSubnet (no NICs) must produce zero SecurityDelta; got added=%v mitigated=%v",
			sd.AddedRisks, sd.MitigatedRisks)
	}
}

// ---- AddPeering test — new peering added to simulated fixture ----

func TestAddPeering_PeeringAppearsInSimulation(t *testing.T) {
	fx := twoPeerFixture()

	origFindings := analyze.Analyze(fx)

	delta := simulator.TopologyDelta{
		AddPeering: &simulator.AddPeeringOp{
			LocalVNet:  "vnet-a",
			RemoteVNet: "vnet-b",
			State:      "Connected",
		},
	}
	simFx, err := simulator.ApplyDelta(fx, delta)
	if err != nil {
		t.Fatalf("ApplyDelta: %v", err)
	}

	// Peering must appear in simulated VNet.
	found := false
	for _, p := range simFx.ResourceGraph.VirtualNetworks[0].Peerings {
		if p.RemoteVnet == "vnet-b" {
			found = true
		}
	}
	if !found {
		t.Error("peering to vnet-b must appear in simulated vnet-a")
	}

	// Original must be unchanged.
	for _, p := range fx.ResourceGraph.VirtualNetworks[0].Peerings {
		if p.RemoteVnet == "vnet-b" {
			t.Error("original vnet-a must not have the new peering")
		}
	}

	// Diff is computed — no assertion on count (Phase 1 engine rule limitation
	// documented in SR-003: checkCrossSubPeeringExposure reads CrossSubscriptionPeerings,
	// not VNet.Peerings for intra-sub topology).
	sd := simulator.DiffFindings(origFindings, analyze.Analyze(simFx))
	_ = sd
}

// ---- AddPublicIP test — adds PIP to a NIC that had none ----

func TestAddPublicIP_CreatesReachableFinding(t *testing.T) {
	fx := noPublicIPFixture()
	origFindings := analyze.Analyze(fx)

	// Baseline must not have reachable finding.
	if hasReachable(origFindings, "nic-a") {
		t.Fatal("noPublicIPFixture must not have reachable finding")
	}

	delta := simulator.TopologyDelta{
		AddPublicIP: &simulator.AddPublicIPOp{
			NICName:   "nic-a",
			PIPName:   "pip-new",
			IPAddress: "20.2.2.2",
		},
	}
	simFx, err := simulator.ApplyDelta(fx, delta)
	if err != nil {
		t.Fatalf("ApplyDelta: %v", err)
	}

	simFindings := analyze.Analyze(simFx)
	sd := simulator.DiffFindings(origFindings, simFindings)

	foundAdded := false
	for _, f := range sd.AddedRisks {
		if f.Resource == "nic-a" && f.Reachable {
			foundAdded = true
		}
	}
	if !foundAdded {
		t.Errorf("expected added reachable finding after AddPublicIP; AddedRisks=%+v", sd.AddedRisks)
	}
}

// ---- DiffFindings — severity escalation appears as add+mitigate ----

func TestDiffFindings_SeverityEscalation(t *testing.T) {
	orig := []analyze.Finding{
		{Type: "over-permissive NSG (reachable)", Severity: "High", Resource: "nic-a", Reachable: true},
	}
	sim := []analyze.Finding{
		{Type: "over-permissive NSG (reachable)", Severity: "Critical", Resource: "nic-a", Reachable: true},
	}
	sd := simulator.DiffFindings(orig, sim)

	if len(sd.AddedRisks) != 1 {
		t.Fatalf("expected 1 AddedRisk (Critical); got %d", len(sd.AddedRisks))
	}
	if sd.AddedRisks[0].Severity != "Critical" {
		t.Errorf("AddedRisk should be Critical; got %s", sd.AddedRisks[0].Severity)
	}
	if len(sd.MitigatedRisks) != 1 {
		t.Fatalf("expected 1 MitigatedRisk (High); got %d", len(sd.MitigatedRisks))
	}
	if sd.MitigatedRisks[0].Severity != "High" {
		t.Errorf("MitigatedRisk should be High; got %s", sd.MitigatedRisks[0].Severity)
	}
	if sd.RiskVector.CriticalDelta != 1 {
		t.Errorf("CriticalDelta want 1; got %d", sd.RiskVector.CriticalDelta)
	}
	if sd.RiskVector.HighDelta != -1 {
		t.Errorf("HighDelta want -1; got %d", sd.RiskVector.HighDelta)
	}
}

// ---- error cases ----

func TestApplyDelta_MissingResource(t *testing.T) {
	fx := baseFixture()
	cases := []struct {
		name  string
		delta simulator.TopologyDelta
	}{
		{"AddNSGRule-unknown-NSG", simulator.TopologyDelta{
			AddNSGRule: &simulator.AddNSGRuleOp{NSGName: "no-such-nsg",
				Rule: graph.SecRule{Name: "r", Priority: 100, Direction: "Inbound",
					Access: "Allow", SourceAddressPrefix: "*", DestinationPortRange: "80"}},
		}},
		{"RemoveNSGRule-unknown-NSG", simulator.TopologyDelta{
			RemoveNSGRule: &simulator.RemoveNSGRuleOp{NSGName: "no-such-nsg", RuleName: "r"},
		}},
		{"ModifyRoute-unknown-RT", simulator.TopologyDelta{
			ModifyRoute: &simulator.ModifyRouteOp{RouteTableName: "no-rt", RouteName: "r", NewNextHopType: "Internet"},
		}},
		{"AddPublicIP-unknown-NIC", simulator.TopologyDelta{
			AddPublicIP: &simulator.AddPublicIPOp{NICName: "no-nic", PIPName: "p", IPAddress: "1.2.3.4"},
		}},
		{"RemovePublicIP-unknown-NIC", simulator.TopologyDelta{
			RemovePublicIP: &simulator.RemovePublicIPOp{NICName: "no-nic"},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := simulator.ApplyDelta(fx, tc.delta)
			if err == nil {
				t.Error("expected error for missing resource")
			}
		})
	}
}

// ---- projection correctness: system default not stripped ----

// TestProjection_SystemDefaultPreserved verifies that adding a rule named
// "AllowVnetInBound" at priority 200 (user-defined range) does not strip
// the system-default "AllowVnetInBound" at priority 65000.
func TestProjection_SystemDefaultPreserved(t *testing.T) {
	fx := baseFixture()
	// Manually add a system-default AllowVnetInBound to the effective rules.
	sysDefault := graph.SecRule{
		Name: "AllowVnetInBound", Priority: 65000, Direction: "Inbound", Access: "Allow",
	}
	fx.NetworkWatcher.EffectiveSecurityRules["nic-a"] = append(
		fx.NetworkWatcher.EffectiveSecurityRules["nic-a"], sysDefault,
	)

	delta := simulator.TopologyDelta{
		AddNSGRule: &simulator.AddNSGRuleOp{
			NSGName: "nsg-a",
			Rule: graph.SecRule{
				Name: "AllowVnetInBound", Priority: 200, Direction: "Inbound",
				Access: "Allow", Protocol: "Tcp",
				SourceAddressPrefix: "VirtualNetwork", DestinationPortRange: "443",
			},
		},
	}
	simFx, err := simulator.ApplyDelta(fx, delta)
	if err != nil {
		t.Fatalf("ApplyDelta: %v", err)
	}

	effective := simFx.NetworkWatcher.EffectiveSecurityRules["nic-a"]
	systemDefaultPresent := false
	for _, r := range effective {
		if r.Name == "AllowVnetInBound" && r.Priority == 65000 {
			systemDefaultPresent = true
		}
	}
	if !systemDefaultPresent {
		t.Error("system default AllowVnetInBound at priority 65000 must be preserved after projection")
	}
}

// ---- helpers ----

func hasReachable(fs []analyze.Finding, resource string) bool {
	for _, f := range fs {
		if f.Reachable && f.Resource == resource {
			return true
		}
	}
	return false
}
