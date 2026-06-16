package forecast_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aaraminds/azure-nettopo-engine/internal/graph"
	"github.com/aaraminds/azure-nettopo-engine/simulator"

	"github.com/aaraminds/azure-nettopo-engine/forecast"
)

// ---- mock HTTP server helpers ----

// priceServer returns an httptest.Server that responds to Retail Prices API
// requests with a configurable per-filter price table.
func priceServer(t *testing.T, prices map[string]float64) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		filter := r.URL.Query().Get("$filter")
		price := 0.0
		for k, v := range prices {
			if strings.Contains(filter, k) {
				price = v
				break
			}
		}
		resp := map[string]interface{}{
			"Items": []map[string]interface{}{
				{
					"unitPrice":          price,
					"retailPrice":        price,
					"unitOfMeasure":      "1 Month",
					"skuName":            "test-sku",
					"serviceName":        "test-service",
					"armRegionName":      "eastus",
					"type":               "Consumption",
					"effectiveStartDate": "2025-01-01T00:00:00Z",
				},
			},
			"NextPageLink": nil,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

// emptyPriceServer returns a server that always responds with an empty Items list.
func emptyPriceServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"Items":        []interface{}{},
			"NextPageLink": nil,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

// rateLimitServer returns a server that always responds with HTTP 429.
func rateLimitServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
}

// clientFor returns an *http.Client whose transport rewrites the real Retail Prices
// base URL to the mock server URL, so PriceCache filter URLs still work normally.
func clientFor(t *testing.T, srv *httptest.Server) *http.Client {
	t.Helper()
	return &http.Client{
		Transport: &urlRewriter{base: srv.URL},
	}
}

// urlRewriter redirects all requests to a fixed base URL (the mock server).
type urlRewriter struct {
	base string
	real http.RoundTripper
}

func (u *urlRewriter) RoundTrip(req *http.Request) (*http.Response, error) {
	// Replace scheme+host with mock server, preserve path+query.
	newURL := u.base + req.URL.Path + "?" + req.URL.RawQuery
	newReq, _ := http.NewRequestWithContext(req.Context(), req.Method, newURL, req.Body)
	if u.real != nil {
		return u.real.RoundTrip(newReq)
	}
	return http.DefaultTransport.RoundTrip(newReq)
}

// ---- minimal fixtures ----

func pip(name string) *string { return &name }

// nicWithPIP returns a minimal fixture with one NIC that has a public IP, an
// open NSG rule, and an Internet default route — reachable from the internet.
func nicWithPIP() *graph.Fixture {
	nicName := "nic-a"
	return &graph.Fixture{
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
				{Name: "nsg-a", AssociatedSubnets: []string{"vnet-a/sub-a"}},
			},
			RouteTables: []graph.RouteTable{
				{Name: "rt-a", AssociatedSubnets: []string{"vnet-a/sub-a"},
					Routes: []graph.Route{
						{Name: "default", AddressPrefix: "0.0.0.0/0", NextHopType: "Internet"},
					}},
			},
			PublicIPAddresses: []graph.PublicIP{
				{Name: "pip-a", IPAddress: "20.1.1.1", IPConfiguration: &nicName},
			},
			NetworkInterfaces: []graph.NIC{
				{Name: "nic-a", Subnet: "vnet-a/sub-a", PublicIP: pip("pip-a"), PrivateIP: "10.0.1.4"},
			},
		},
		NetworkWatcher: graph.NetworkWatcher{
			EffectiveSecurityRules: map[string][]graph.SecRule{"nic-a": {}},
			EffectiveRoutes: map[string][]graph.Route{
				"nic-a": {{AddressPrefix: "0.0.0.0/0", NextHopType: "Internet"}},
			},
		},
		AVNM: graph.AVNM{},
	}
}

// nicNoPIP returns the same fixture but without a public IP on nic-a.
func nicNoPIP() *graph.Fixture {
	fx := nicWithPIP()
	fx.ResourceGraph.NetworkInterfaces[0].PublicIP = nil
	return fx
}

// fxWithFirewall adds an Azure Firewall to the base fixture.
func fxWithFirewall(skuTier string) *graph.Fixture {
	fx := nicWithPIP()
	fx.AzureFirewall = &graph.Firewall{
		Name:     "fw-hub",
		PrivateIP: "10.0.0.4",
		PublicIP:  "20.99.99.99",
		SKUTier:  skuTier,
	}
	return fx
}

// ---- PriceCache tests ----

func TestPriceCache_ReturnsCachedPrice(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := map[string]interface{}{
			"Items": []map[string]interface{}{
				{"unitPrice": 3.65, "effectiveStartDate": "2025-01-01T00:00:00Z"},
			},
			"NextPageLink": nil,
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	cache := forecast.NewPriceCacheWithClient(clientFor(t, srv))
	ctx := context.Background()

	p1, d1, err := cache.Lookup(ctx, "serviceName eq 'Virtual Network'")
	if err != nil {
		t.Fatalf("first Lookup: %v", err)
	}
	if p1 != 3.65 {
		t.Errorf("price want 3.65; got %f", p1)
	}
	if d1 != "2025-01-01" {
		t.Errorf("sourceDate want 2025-01-01; got %s", d1)
	}

	// Second call with same filter must NOT hit the server again.
	_, _, _ = cache.Lookup(ctx, "serviceName eq 'Virtual Network'")
	if callCount != 1 {
		t.Errorf("expected 1 HTTP call (cache hit); got %d", callCount)
	}
}

func TestPriceCache_RefreshesAfterTTL(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := map[string]interface{}{
			"Items": []map[string]interface{}{
				{"unitPrice": 5.00, "effectiveStartDate": "2025-06-01T00:00:00Z"},
			},
			"NextPageLink": nil,
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	now := time.Now()
	cache := forecast.NewPriceCacheWithClient(clientFor(t, srv))
	// Artificially age the cache by manipulating the clock after first call.
	ctx := context.Background()

	_, _, _ = cache.Lookup(ctx, "my-filter")
	if callCount != 1 {
		t.Fatalf("expected 1 call; got %d", callCount)
	}

	// Simulate 25h elapsed — TTL is 24h.
	cache.SetClock(func() time.Time { return now.Add(25 * time.Hour) })
	_, _, _ = cache.Lookup(ctx, "my-filter")
	if callCount != 2 {
		t.Errorf("expected refresh after TTL expiry; callCount=%d", callCount)
	}
}

func TestPriceCache_RateLimitReturnsError(t *testing.T) {
	srv := rateLimitServer(t)
	defer srv.Close()

	cache := forecast.NewPriceCacheWithClient(clientFor(t, srv))
	_, _, err := cache.Lookup(context.Background(), "any-filter")
	if err == nil {
		t.Fatal("expected error on HTTP 429")
	}
	if !forecast.IsRateLimit(err) {
		t.Errorf("expected RateLimitError; got %T: %v", err, err)
	}
}

func TestPriceCache_EmptyItemsReturnsZero(t *testing.T) {
	srv := emptyPriceServer(t)
	defer srv.Close()

	cache := forecast.NewPriceCacheWithClient(clientFor(t, srv))
	price, sd, err := cache.Lookup(context.Background(), "unknown-sku")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if price != 0 || sd != "" {
		t.Errorf("expected (0, ''); got (%f, %q)", price, sd)
	}
}

// ---- ForecastCost fixed cost tests ----

func TestForecastCost_AddPublicIP_PositiveFixedDelta(t *testing.T) {
	srv := priceServer(t, map[string]float64{
		"IP Addresses": 3.65, // Standard Static IPv4
	})
	defer srv.Close()

	cache := forecast.NewPriceCacheWithClient(clientFor(t, srv))
	fx := nicNoPIP()
	delta := simulator.TopologyDelta{
		AddPublicIP: &simulator.AddPublicIPOp{
			NICName: "nic-a", PIPName: "pip-new", IPAddress: "20.2.2.2",
		},
	}

	fc, err := forecast.ForecastCost(context.Background(), fx, delta, cache, forecast.FlowSummary{}, "eastus")
	if err != nil {
		t.Fatalf("ForecastCost: %v", err)
	}
	if fc.FixedDeltaUSD <= 0 {
		t.Errorf("AddPublicIP must produce positive FixedDeltaUSD; got %f", fc.FixedDeltaUSD)
	}
	if len(fc.LineItems) == 0 {
		t.Error("expected at least one LineItem")
	}
	if fc.LineItems[0].ResourceType != "PublicIP" {
		t.Errorf("LineItem ResourceType want PublicIP; got %s", fc.LineItems[0].ResourceType)
	}
	if fc.LineItems[0].ChangeType != "Add" {
		t.Errorf("LineItem ChangeType want Add; got %s", fc.LineItems[0].ChangeType)
	}
}

func TestForecastCost_RemovePublicIP_NegativeFixedDelta(t *testing.T) {
	srv := priceServer(t, map[string]float64{
		"IP Addresses": 3.65,
	})
	defer srv.Close()

	cache := forecast.NewPriceCacheWithClient(clientFor(t, srv))
	fx := nicWithPIP()
	delta := simulator.TopologyDelta{
		RemovePublicIP: &simulator.RemovePublicIPOp{NICName: "nic-a"},
	}

	fc, err := forecast.ForecastCost(context.Background(), fx, delta, cache, forecast.FlowSummary{}, "eastus")
	if err != nil {
		t.Fatalf("ForecastCost: %v", err)
	}
	if fc.FixedDeltaUSD >= 0 {
		t.Errorf("RemovePublicIP must produce negative FixedDeltaUSD; got %f", fc.FixedDeltaUSD)
	}
	if len(fc.LineItems) == 0 {
		t.Error("expected at least one LineItem")
	}
	if fc.LineItems[0].ChangeType != "Remove" {
		t.Errorf("LineItem ChangeType want Remove; got %s", fc.LineItems[0].ChangeType)
	}
}

func TestForecastCost_ExistingFirewallReported(t *testing.T) {
	srv := priceServer(t, map[string]float64{
		"Azure Firewall": 1500.0,
	})
	defer srv.Close()

	cache := forecast.NewPriceCacheWithClient(clientFor(t, srv))
	fx := fxWithFirewall("Standard")
	delta := simulator.TopologyDelta{
		AddSubnet: &simulator.AddSubnetOp{
			VNetName: "vnet-a", Name: "sub-new", AddressPrefix: "10.0.2.0/24",
		},
	}

	fc, err := forecast.ForecastCost(context.Background(), fx, delta, cache, forecast.FlowSummary{}, "eastus")
	if err != nil {
		t.Fatalf("ForecastCost: %v", err)
	}
	if fc.ExistingFixedMonthlyUSD <= 0 {
		t.Errorf("existing firewall cost should be > 0; got %f", fc.ExistingFixedMonthlyUSD)
	}
	// Delta should be $0 — AddSubnet has no fixed-cost delta.
	if fc.FixedDeltaUSD != 0 {
		t.Errorf("AddSubnet FixedDeltaUSD should be 0; got %f", fc.FixedDeltaUSD)
	}
}

// ---- ForecastCost variable cost tests ----

func TestForecastCost_ModifyRouteToNVA_VariableCostBand(t *testing.T) {
	srv := emptyPriceServer(t) // no fixed cost for ModifyRoute
	defer srv.Close()

	cache := forecast.NewPriceCacheWithClient(clientFor(t, srv))
	fx := nicWithPIPAndRT()

	// 1 NIC in the route table's subnet → 50 GB/NIC heuristic
	delta := simulator.TopologyDelta{
		ModifyRoute: &simulator.ModifyRouteOp{
			RouteTableName: "rt-a",
			RouteName:      "default",
			NewNextHopType: "VirtualAppliance",
			NewNextHopIP:   "10.0.0.4",
		},
	}

	fc, err := forecast.ForecastCost(context.Background(), fx, delta, cache, forecast.FlowSummary{}, "eastus")
	if err != nil {
		t.Fatalf("ForecastCost: %v", err)
	}
	if fc.VariableDeltaUSDLow >= fc.VariableDeltaUSDHigh {
		t.Errorf("VariableDeltaUSDLow must be < High; got [%f, %f]", fc.VariableDeltaUSDLow, fc.VariableDeltaUSDHigh)
	}
	if fc.ConfidenceBandPct != 50 {
		// No flow logs → 50% band
		t.Errorf("ConfidenceBandPct want 50 (no flow logs); got %d", fc.ConfidenceBandPct)
	}
}

func TestForecastCost_ModifyRoute_FlowLogsBand30(t *testing.T) {
	srv := emptyPriceServer(t)
	defer srv.Close()

	cache := forecast.NewPriceCacheWithClient(clientFor(t, srv))
	fx := nicWithPIPAndRT()

	flows := forecast.FlowSummary{
		FlowLogsEnabled: true,
		MonthlyGBByVNet: map[string]float64{"vnet-a": 1000.0},
	}
	delta := simulator.TopologyDelta{
		ModifyRoute: &simulator.ModifyRouteOp{
			RouteTableName: "rt-a",
			RouteName:      "default",
			NewNextHopType: "VirtualAppliance",
			NewNextHopIP:   "10.0.0.4",
		},
	}

	fc, err := forecast.ForecastCost(context.Background(), fx, delta, cache, flows, "eastus")
	if err != nil {
		t.Fatalf("ForecastCost: %v", err)
	}
	if fc.ConfidenceBandPct != 30 {
		t.Errorf("ConfidenceBandPct want 30 (flow logs available); got %d", fc.ConfidenceBandPct)
	}
	// Band should be ±30% of point estimate.
	// point = 1000 GB × $0.016/GB = $16; low = $11.2, high = $20.8
	expectedLow := 1000.0 * 0.016 * 0.70
	expectedHigh := 1000.0 * 0.016 * 1.30
	if abs(fc.VariableDeltaUSDLow-expectedLow) > 0.01 {
		t.Errorf("VariableDeltaUSDLow want %.2f; got %.2f", expectedLow, fc.VariableDeltaUSDLow)
	}
	if abs(fc.VariableDeltaUSDHigh-expectedHigh) > 0.01 {
		t.Errorf("VariableDeltaUSDHigh want %.2f; got %.2f", expectedHigh, fc.VariableDeltaUSDHigh)
	}
}

// ---- mandatory caveats test ----

func TestForecastCost_MandatoryCaveatsPresent(t *testing.T) {
	srv := emptyPriceServer(t)
	defer srv.Close()

	cache := forecast.NewPriceCacheWithClient(clientFor(t, srv))
	fc, err := forecast.ForecastCost(context.Background(), nicNoPIP(),
		simulator.TopologyDelta{
			AddSubnet: &simulator.AddSubnetOp{VNetName: "vnet-a", Name: "s", AddressPrefix: "10.0.9.0/24"},
		},
		cache, forecast.FlowSummary{}, "eastus")
	if err != nil {
		t.Fatalf("ForecastCost: %v", err)
	}

	required := []string{
		"EA/MCA contract discounts are not applied",
		"Variable costs are estimated from traffic volume data",
		"they do not represent the total topology cost",
	}
	for _, must := range required {
		found := false
		for _, c := range fc.Caveats {
			if strings.Contains(c, must) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("mandatory caveat missing (substring %q); caveats=%v", must, fc.Caveats)
		}
	}
}

// ---- price_source_date populated ----

func TestForecastCost_PriceSourceDatePopulated(t *testing.T) {
	srv := priceServer(t, map[string]float64{"IP Addresses": 3.65})
	defer srv.Close()

	cache := forecast.NewPriceCacheWithClient(clientFor(t, srv))
	fc, err := forecast.ForecastCost(context.Background(), nicNoPIP(),
		simulator.TopologyDelta{
			AddPublicIP: &simulator.AddPublicIPOp{NICName: "nic-a", PIPName: "pip-x", IPAddress: "1.2.3.4"},
		},
		cache, forecast.FlowSummary{}, "eastus")
	if err != nil {
		t.Fatalf("ForecastCost: %v", err)
	}
	if fc.PriceSourceDate == "" {
		t.Error("PriceSourceDate must be populated when fixed cost lookups succeed")
	}
}

// ---- line item price source label ----

func TestForecastCost_LineItemPriceSource(t *testing.T) {
	srv := priceServer(t, map[string]float64{"IP Addresses": 3.65})
	defer srv.Close()

	cache := forecast.NewPriceCacheWithClient(clientFor(t, srv))
	fc, err := forecast.ForecastCost(context.Background(), nicNoPIP(),
		simulator.TopologyDelta{
			AddPublicIP: &simulator.AddPublicIPOp{NICName: "nic-a", PIPName: "pip-x", IPAddress: "1.2.3.4"},
		},
		cache, forecast.FlowSummary{}, "eastus")
	if err != nil {
		t.Fatalf("ForecastCost: %v", err)
	}
	for _, item := range fc.LineItems {
		if item.PriceSource != "retail-prices-api" {
			t.Errorf("LineItem.PriceSource must always be 'retail-prices-api'; got %q", item.PriceSource)
		}
	}
}

// ---- nil cache returns error ----

func TestForecastCost_NilCacheError(t *testing.T) {
	_, err := forecast.ForecastCost(context.Background(), nicNoPIP(),
		simulator.TopologyDelta{
			AddSubnet: &simulator.AddSubnetOp{VNetName: "vnet-a", Name: "s", AddressPrefix: "10.0.9.0/24"},
		},
		nil, forecast.FlowSummary{}, "eastus")
	if err == nil {
		t.Error("expected error when cache is nil")
	}
}

// ---- OData filter helpers ----

func TestFilterHelpers_ContainExpectedFragments(t *testing.T) {
	cases := []struct {
		name   string
		filter string
		wants  []string
	}{
		{"PIP", forecast.PIPFilter("eastus", "Standard Static IPv4"),
			[]string{"IP Addresses", "eastus", "Standard Static IPv4", "Consumption"}},
		{"VPNGateway", forecast.VPNGatewayFilter("westus", "VpnGw2 Gateway"),
			[]string{"VPN Gateway", "westus", "VpnGw2 Gateway"}},
		{"ERGateway", forecast.ERGatewayFilter("northeurope", "ErGw1AZ Gateway"),
			[]string{"ExpressRoute", "northeurope", "ErGw1AZ Gateway"}},
		{"Firewall", forecast.FirewallFilter("eastus", "Standard"),
			[]string{"Azure Firewall", "eastus", "Standard"}},
		{"PrivateEndpoint", forecast.PrivateEndpointFilter("eastus"),
			[]string{"Private Link", "eastus"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, want := range tc.wants {
				if !strings.Contains(tc.filter, want) {
					t.Errorf("filter %q missing %q", tc.filter, want)
				}
			}
		})
	}
}

// ---- helpers ----

// nicWithPIPAndRT is like nicWithPIP but includes a proper RT with the default route.
func nicWithPIPAndRT() *graph.Fixture {
	fx := nicWithPIP()
	// Ensure the route table associates with the subnet.
	if len(fx.ResourceGraph.RouteTables) == 0 {
		fx.ResourceGraph.RouteTables = []graph.RouteTable{
			{
				Name:              "rt-a",
				AssociatedSubnets: []string{"vnet-a/sub-a"},
				Routes: []graph.Route{
					{Name: "default", AddressPrefix: "0.0.0.0/0", NextHopType: "Internet"},
				},
			},
		}
	}
	return fx
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
