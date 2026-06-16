package adapter

import (
	"testing"

	"github.com/aaraminds/azure-nettopo-engine/internal/graph"
)

// C-2: validate the ARM field-path assumptions the parsers depend on, against
// representative Resource-Graph JSON (numbers as float64, as a real ARG response
// unmarshals). These were previously [VERIFY] items with no test.

// F6: cross-subscription peerings parsed per-VNet must be projected into the
// flat Fixture.CrossSubscriptionPeerings list (intra-sub peerings excluded), or
// the cross-sub-peering finding family is dead on live data.
func TestDeriveCrossSubPeerings(t *testing.T) {
	vnets := []graph.VNet{
		{Name: "hub", Peerings: []graph.Peering{
			{RemoteVnet: "spoke-local", State: "Connected"},                                                          // intra-sub: excluded
			{RemoteVnet: "remote-a", RemoteSubscriptionID: "sub-b", State: "Connected", AllowForwardedTraffic: true}, // cross-sub
		}},
		{Name: "edge", Peerings: []graph.Peering{
			{RemoteVnet: "remote-c", RemoteSubscriptionID: "sub-c", State: "Disconnected"}, // cross-sub (state preserved)
		}},
	}
	got := deriveCrossSubPeerings(vnets)
	if len(got) != 2 {
		t.Fatalf("want 2 cross-sub peerings (intra-sub excluded), got %d: %+v", len(got), got)
	}
	if got[0].LocalVnet != "hub" || got[0].RemoteVnet != "remote-a" || got[0].RemoteSubscriptionID != "sub-b" ||
		got[0].State != "Connected" || !got[0].AllowForwardedTraffic || got[0].HasHubFirewall {
		t.Errorf("first peering mis-projected: %+v", got[0])
	}
	if got[1].LocalVnet != "edge" || got[1].State != "Disconnected" {
		t.Errorf("second peering mis-projected: %+v", got[1])
	}
}

// F4: ARM returns inboundNatRules[].properties.backendIPConfiguration as an
// object {"id": ...} on the live path. Both the object shape and the legacy
// bare-string shape must resolve to the backend NIC, or LB-NAT exposure is missed.
func TestParseLBNatRules_BackendIPConfigShapes(t *testing.T) {
	const nicID = "/subscriptions/s/resourceGroups/rg/providers/Microsoft.Network/networkInterfaces/nic-app/ipConfigurations/ipconfig1"
	objShape := []interface{}{map[string]interface{}{
		"name": "ssh-nat",
		"properties": map[string]interface{}{
			"protocol":               "Tcp",
			"frontendPort":           float64(2222),
			"backendPort":            float64(22),
			"backendIPConfiguration": map[string]interface{}{"id": nicID}, // live ARM object shape
		},
	}}
	strShape := []interface{}{map[string]interface{}{
		"name": "ssh-nat",
		"properties": map[string]interface{}{
			"protocol":               "Tcp",
			"frontendPort":           float64(2222),
			"backendPort":            float64(22),
			"backendIPConfiguration": nicID, // legacy bare-string shape
		},
	}}
	for _, tc := range []struct {
		name string
		arr  []interface{}
	}{{"object", objShape}, {"string", strShape}} {
		got := parseLBNatRules(tc.arr)
		if len(got) != 1 {
			t.Fatalf("%s: want 1 rule, got %d", tc.name, len(got))
		}
		if got[0].BackendNic != "nic-app" {
			t.Errorf("%s: BackendNic = %q, want %q (exposure would be silently skipped)", tc.name, got[0].BackendNic, "nic-app")
		}
		if got[0].FrontendPort != 2222 || got[0].BackendPort != 22 {
			t.Errorf("%s: ports mis-parsed: %+v", tc.name, got[0])
		}
	}
}

// F3: App Gateway WAF state must come from the actual config/policy, never from
// the SKU. A WAF_v2 gateway with a disabled or Detection-mode policy must NOT be
// reported as protected.
func TestParseAppGateways_WAFFromPolicyNotSKU(t *testing.T) {
	// WAF_v2 SKU but the attached firewall policy is Disabled → WafEnabled=false.
	rows := []map[string]interface{}{{
		"name":           "appgw-prod",
		"skuTier":        "WAF_v2",
		"wafEnabled":     false, // no inline config
		"wafMode":        "",
		"wafPolicyState": "Disabled",
		"wafPolicyMode":  "Prevention",
	}}
	gw := parseAppGateways(rows)
	if len(gw) != 1 {
		t.Fatalf("want 1 gateway, got %d", len(gw))
	}
	if gw[0].WafEnabled {
		t.Error("WAF_v2 with a Disabled policy must report WafEnabled=false (SKU must not force it on)")
	}
	// WAF_v2 with an Enabled Detection-mode policy → enabled + Detection mode.
	rows2 := []map[string]interface{}{{
		"name": "appgw-det", "skuTier": "WAF_v2",
		"wafEnabled": false, "wafMode": "",
		"wafPolicyState": "Enabled", "wafPolicyMode": "Detection",
	}}
	gw2 := parseAppGateways(rows2)
	if !gw2[0].WafEnabled || gw2[0].WafMode != "Detection" {
		t.Errorf("policy state/mode not honored: %+v", gw2[0])
	}
}

// F3: Front Door WAF state must come from a real WAF-policy association + its
// mode, not from frontDoorId (which is always set). The parser must read wafMode.
func TestParseFrontDoors_WAFModeProjected(t *testing.T) {
	rows := []map[string]interface{}{
		{"name": "fd-unprotected", "sku": "Standard_AzureFrontDoor", "wafEnabled": false},
		{"name": "fd-detection", "sku": "Premium_AzureFrontDoor", "wafEnabled": true, "wafMode": "Detection"},
	}
	fds := parseFrontDoors(rows)
	if len(fds) != 2 {
		t.Fatalf("want 2 front doors, got %d", len(fds))
	}
	if fds[0].WafEnabled {
		t.Error("fd-unprotected must report WafEnabled=false (no WAF policy association)")
	}
	if !fds[1].WafEnabled || fds[1].WafMode != "Detection" {
		t.Errorf("fd-detection wafMode not projected: %+v", fds[1])
	}
}

func TestParseVNets_FieldPaths(t *testing.T) {
	rows := []map[string]interface{}{{
		"name":            "vnet-a",
		"addressPrefixes": []interface{}{"10.0.0.0/16"},
		"subnets": []interface{}{map[string]interface{}{
			"name": "web",
			"properties": map[string]interface{}{
				"addressPrefix":        "10.0.1.0/24",
				"networkSecurityGroup": map[string]interface{}{"id": "/subscriptions/s/resourceGroups/rg/providers/Microsoft.Network/networkSecurityGroups/nsg-web"},
				"routeTable":           map[string]interface{}{"id": "/subscriptions/s/resourceGroups/rg/providers/Microsoft.Network/routeTables/rt-web"},
			},
		}},
		"peerings": []interface{}{map[string]interface{}{
			"properties": map[string]interface{}{
				"remoteVirtualNetwork":  map[string]interface{}{"id": "/subscriptions/s2/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/vnet-b"},
				"peeringState":          "Connected",
				"allowForwardedTraffic": true,
			},
		}},
	}}
	vnets := parseVNets(rows)
	if len(vnets) != 1 {
		t.Fatalf("want 1 vnet, got %d", len(vnets))
	}
	v := vnets[0]
	if v.Name != "vnet-a" || len(v.AddressSpace) != 1 || v.AddressSpace[0] != "10.0.0.0/16" {
		t.Fatalf("vnet name/address wrong: %+v", v)
	}
	if len(v.Subnets) != 1 {
		t.Fatalf("want 1 subnet, got %d", len(v.Subnets))
	}
	sn := v.Subnets[0]
	if sn.Name != "web" || sn.AddressPrefix != "10.0.1.0/24" {
		t.Errorf("subnet name/prefix wrong: %+v", sn)
	}
	if sn.NetworkSecurityGroup != "nsg-web" {
		t.Errorf("subnet NSG field-path: got %q want nsg-web", sn.NetworkSecurityGroup)
	}
	if sn.RouteTable != "rt-web" {
		t.Errorf("subnet routeTable field-path: got %q want rt-web", sn.RouteTable)
	}
	if len(v.Peerings) != 1 || v.Peerings[0].RemoteVnet != "vnet-b" {
		t.Errorf("peering remoteVirtualNetwork field-path: %+v", v.Peerings)
	}
	if !v.Peerings[0].AllowForwardedTraffic {
		t.Errorf("peering allowForwardedTraffic not parsed")
	}
}

func TestParseNSGs_FieldPaths(t *testing.T) {
	rows := []map[string]interface{}{{
		"name": "nsg-web",
		"securityRules": []interface{}{map[string]interface{}{
			"name": "allow-https",
			"properties": map[string]interface{}{
				"priority":             float64(200),
				"direction":            "Inbound",
				"access":               "Allow",
				"protocol":             "Tcp",
				"sourceAddressPrefix":  "0.0.0.0/0",
				"destinationPortRange": "443",
			},
		}},
		"associatedSubnets": []interface{}{map[string]interface{}{
			"id": "/subscriptions/s/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/vnet-a/subnets/web",
		}},
	}}
	nsgs := parseNSGs(rows)
	if len(nsgs) != 1 || nsgs[0].Name != "nsg-web" {
		t.Fatalf("nsg parse: %+v", nsgs)
	}
	if len(nsgs[0].SecurityRules) != 1 {
		t.Fatalf("want 1 rule, got %d", len(nsgs[0].SecurityRules))
	}
	r := nsgs[0].SecurityRules[0]
	if r.Priority != 200 || r.Access != "Allow" || r.SourceAddressPrefix != "0.0.0.0/0" || r.DestinationPortRange != "443" {
		t.Errorf("rule field-paths wrong: %+v", r)
	}
	if len(nsgs[0].AssociatedSubnets) != 1 || nsgs[0].AssociatedSubnets[0] != "vnet-a/web" {
		t.Errorf("associatedSubnets path: %+v", nsgs[0].AssociatedSubnets)
	}
}

func TestParseRouteTables_FieldPaths(t *testing.T) {
	rows := []map[string]interface{}{{
		"name": "rt-app",
		"routes": []interface{}{map[string]interface{}{
			"name": "to-fw",
			"properties": map[string]interface{}{
				"addressPrefix":    "0.0.0.0/0",
				"nextHopType":      "VirtualAppliance",
				"nextHopIpAddress": "10.0.0.4",
			},
		}},
		"associatedSubnets": []interface{}{map[string]interface{}{
			"id": "/subscriptions/s/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/vnet-a/subnets/app",
		}},
	}}
	rts := parseRouteTables(rows)
	if len(rts) != 1 || rts[0].Name != "rt-app" || len(rts[0].Routes) != 1 {
		t.Fatalf("route table parse: %+v", rts)
	}
	rt := rts[0].Routes[0]
	if rt.AddressPrefix != "0.0.0.0/0" || rt.NextHopType != "VirtualAppliance" || rt.NextHopIPAddress != "10.0.0.4" {
		t.Errorf("route field-paths wrong: %+v", rt)
	}
	if len(rts[0].AssociatedSubnets) != 1 || rts[0].AssociatedSubnets[0] != "vnet-a/app" {
		t.Errorf("route table associatedSubnets path: %+v", rts[0].AssociatedSubnets)
	}
}
