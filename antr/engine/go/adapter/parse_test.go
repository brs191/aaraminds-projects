package adapter

import "testing"

// C-2: validate the ARM field-path assumptions the parsers depend on, against
// representative Resource-Graph JSON (numbers as float64, as a real ARG response
// unmarshals). These were previously [VERIFY] items with no test.

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
