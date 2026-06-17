package adapter

import (
	"context"
	"strings"
	"testing"

	armrg "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph"

	"github.com/aaraminds/azure-nettopo-engine/internal/analyze"
	"github.com/aaraminds/azure-nettopo-engine/internal/graph"
)

// recordedARG is an offline argQuerier that serves a recorded estate: it returns
// the rows for whichever resource type the KQL filters on. This drives the ENTIRE
// fetchResourceGraph fan-out (15+ parallel queries, parse wiring, app-layer joins,
// cross-sub derivation, multi-firewall detection) without touching Azure — closing
// the orchestration half of audit C-2. Field-path shapes mirror real ARG
// projections (numbers as float64, joined fields flattened, ids as objects).
type recordedARG struct {
	byType map[string][]interface{}
}

func (r *recordedARG) Resources(_ context.Context, q armrg.QueryRequest, _ *armrg.ClientResourcesOptions) (armrg.ClientResourcesResponse, error) {
	kql := ""
	if q.Query != nil {
		kql = *q.Query
	}
	// Dispatch to the FIRST (outermost) type literal present in the query, so the
	// join sub-queries (e.g. App Gateway → WAF policy) resolve to the outer type.
	best, bestPos := "", 1<<30
	for t := range r.byType {
		if i := strings.Index(kql, t); i >= 0 && i < bestPos {
			best, bestPos = t, i
		}
	}
	var rows []interface{}
	if best != "" {
		rows = r.byType[best]
	}
	return armrg.ClientResourcesResponse{QueryResponse: armrg.QueryResponse{Data: rows, SkipToken: nil}}, nil
}

func obj(kv ...interface{}) map[string]interface{} {
	m := map[string]interface{}{}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i].(string)] = kv[i+1]
	}
	return m
}

// A small but realistic estate that exercises the live-path families the audit
// flagged: app-layer WAF posture (F3), LB-NAT object shape (F4), multi-firewall
// (F5), cross-sub peering derivation (F6), and the same-sub-peering correctness
// fix (F10). NW effective rules/routes come from a separate REST path and are
// empty here; these families are computed from the resource graph alone.
func recordedEstate(localSub, remoteSub string) *recordedARG {
	t := func(s string) string { return `"` + s + `"` }
	return &recordedARG{byType: map[string][]interface{}{
		t("microsoft.network/virtualnetworks"): {
			obj("name", "hub", "addressPrefixes", []interface{}{"10.0.0.0/16"},
				"subnets", []interface{}{obj("name", "AzureFirewallSubnet", "properties", obj("addressPrefix", "10.0.0.0/24"))},
				"peerings", []interface{}{
					// same-subscription peer — must NOT be flagged cross-sub (F10)
					obj("properties", obj("remoteVirtualNetwork", obj("id", "/subscriptions/"+localSub+"/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/spoke"),
						"peeringState", "Connected", "allowForwardedTraffic", true)),
					// cross-subscription peer — MUST be flagged cross-sub (F6)
					obj("properties", obj("remoteVirtualNetwork", obj("id", "/subscriptions/"+remoteSub+"/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/partner"),
						"peeringState", "Connected", "allowForwardedTraffic", false)),
				}),
			obj("name", "spoke", "addressPrefixes", []interface{}{"10.1.0.0/16"},
				"subnets", []interface{}{obj("name", "web", "properties", obj("addressPrefix", "10.1.1.0/24"))},
				"peerings", []interface{}{}),
		},
		t("microsoft.network/azurefirewalls"): {
			obj("name", "afw-east", "resourceGroup", "rg", "privateIp", "10.0.0.4"),
			obj("name", "afw-west", "resourceGroup", "rg", "privateIp", "10.2.0.4"), // F5: 2nd firewall must be modeled
		},
		t("microsoft.network/loadbalancers"): {
			obj("name", "lb-pub", "sku", "Standard", "isInternal", false,
				"publicIPRef", "/subscriptions/"+localSub+"/resourceGroups/rg/providers/Microsoft.Network/publicIPAddresses/pip-lb",
				"inboundNatRules", []interface{}{
					obj("name", "ssh", "properties", obj("protocol", "Tcp",
						"frontendPort", float64(2222), "backendPort", float64(22),
						// F4: ARM returns backendIPConfiguration as an OBJECT {id}
						"backendIPConfiguration", obj("id", "/subscriptions/"+localSub+"/resourceGroups/rg/providers/Microsoft.Network/networkInterfaces/nic-app/ipConfigurations/ipcfg")))}),
		},
		t("microsoft.network/applicationgateways"): {
			// F3: WAF_v2 SKU but the attached WAF policy is Disabled → finding fires
			obj("name", "appgw-prod", "skuTier", "WAF_v2", "wafEnabled", false,
				"wafPolicyState", "Disabled", "wafPolicyMode", "Prevention",
				"gatewaySubnetRef", "/subscriptions/"+localSub+"/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/hub/subnets/appgw",
				"frontendIPs", []interface{}{obj("properties", obj("publicIPAddress", obj("id", "/subscriptions/"+localSub+"/.../publicIPAddresses/pip-agw")))}),
		},
		t("microsoft.containerservice/managedclusters"): {
			obj("name", "aks-public", "subnetId", "/subscriptions/"+localSub+"/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/spoke/subnets/web", "isPrivate", false),
		},
		t("microsoft.cdn/profiles"): {
			obj("name", "fd-edge", "sku", "Premium_AzureFrontDoor", "wafEnabled", false), // F3 Front Door WAF disabled
		},
	}}
}

func TestRecordedEstate_FetchResourceGraphOrchestration(t *testing.T) {
	const localSub, remoteSub = "11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222"
	a := &adapter{subscriptionID: localSub, argClient: recordedEstate(localSub, remoteSub)}

	res, err := a.fetchResourceGraph(context.Background())
	if err != nil {
		t.Fatalf("fetchResourceGraph: %v", err)
	}
	rg := res.ResourceGraph

	if len(rg.VirtualNetworks) != 2 {
		t.Fatalf("want 2 vnets, got %d", len(rg.VirtualNetworks))
	}
	// F5: both firewalls detected
	if len(res.rawFWs) != 2 {
		t.Errorf("F5: want 2 firewalls detected, got %d", len(res.rawFWs))
	}
	// F4: LB NAT backend NIC resolved from the OBJECT-shaped backendIPConfiguration
	if len(rg.LoadBalancers) != 1 || len(rg.LoadBalancers[0].InboundNatRules) != 1 ||
		rg.LoadBalancers[0].InboundNatRules[0].BackendNic != "nic-app" {
		t.Errorf("F4: LB NAT backend NIC mis-parsed: %+v", rg.LoadBalancers)
	}
	// F3: App Gateway WAF reported disabled (policy state, not SKU)
	if len(rg.ApplicationGateways) != 1 || rg.ApplicationGateways[0].WafEnabled {
		t.Errorf("F3: WAF_v2 with Disabled policy must report WafEnabled=false: %+v", rg.ApplicationGateways)
	}

	// F6 + F10: build the fixture and assert cross-sub derivation is CORRECT —
	// exactly one cross-sub peering (the partner in remoteSub), and the same-sub
	// spoke peering is NOT counted.
	fx := &graph.Fixture{ResourceGraph: rg, CrossSubscriptionPeerings: deriveCrossSubPeerings(rg.VirtualNetworks)}
	if len(fx.CrossSubscriptionPeerings) != 1 {
		t.Fatalf("F10: want exactly 1 cross-sub peering (same-sub must be excluded), got %d: %+v",
			len(fx.CrossSubscriptionPeerings), fx.CrossSubscriptionPeerings)
	}
	if fx.CrossSubscriptionPeerings[0].RemoteVnet != "partner" {
		t.Errorf("cross-sub peer should be 'partner', got %q", fx.CrossSubscriptionPeerings[0].RemoteVnet)
	}

	// End-to-end: the assembled fixture runs through Analyze and surfaces the
	// app-layer exposures the live path must catch.
	want := map[string]bool{
		"app gateway WAF disabled":                    false,
		"AKS non-private cluster":                     false,
		"Front Door WAF disabled":                     false,
		"internet reachable via load balancer NAT":    false,
		"cross-subscription peering without firewall": false,
	}
	for _, f := range analyze.Analyze(fx) {
		if _, ok := want[f.Type]; ok {
			want[f.Type] = true
		}
	}
	for typ, seen := range want {
		if !seen {
			t.Errorf("expected finding %q from the recorded estate, not produced", typ)
		}
	}
}

// F10 focused regression: a same-subscription peering must not be marked cross-sub.
func TestParsePeerings_SameSubNotCrossSub(t *testing.T) {
	const sub = "aaaa1111-bbbb-2222-cccc-333344445555"
	rows := []map[string]interface{}{obj("name", "v", "addressPrefixes", []interface{}{"10.0.0.0/16"},
		"peerings", []interface{}{
			obj("properties", obj("remoteVirtualNetwork", obj("id", "/subscriptions/"+sub+"/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/local-peer"), "peeringState", "Connected")),
			obj("properties", obj("remoteVirtualNetwork", obj("id", "/subscriptions/ffff9999-0000-1111-2222-333344445555/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/remote-peer"), "peeringState", "Connected")),
		})}
	vnets := parseVNets(rows, sub)
	p := vnets[0].Peerings
	if len(p) != 2 {
		t.Fatalf("want 2 peerings, got %d", len(p))
	}
	if p[0].RemoteSubscriptionID != "" {
		t.Errorf("same-sub peer must have empty RemoteSubscriptionID, got %q", p[0].RemoteSubscriptionID)
	}
	if p[1].RemoteSubscriptionID == "" {
		t.Errorf("cross-sub peer must carry RemoteSubscriptionID")
	}
}
