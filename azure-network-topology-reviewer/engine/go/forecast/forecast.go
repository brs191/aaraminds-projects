package forecast

import (
	"context"
	"fmt"
	"strings"

	"github.com/aaraminds/azure-nettopo-engine/internal/graph"
	"github.com/aaraminds/azure-nettopo-engine/simulator"
)

// CostForecast is the output of ForecastCost for a given TopologyDelta.
//
// Fixed costs are EXACT — derived from Azure Retail Prices API list prices.
// Variable costs are a BAND (low/high) — estimated from traffic volume data.
// All amounts are monthly USD. Positive = cost increase. Negative = cost decrease.
//
// IMPORTANT: All values are Retail Prices list rates. EA/MCA contract discounts,
// reserved instances, and commitment savings are NOT applied. Never present these
// values as actual billed amounts. Use Azure Cost Management for actuals.
type CostForecast struct {
	// FixedDeltaUSD is the exact monthly cost change from fixed-price resources
	// (PIP additions/removals). Retail Prices list price.
	FixedDeltaUSD float64 `json:"fixedDeltaUsd"`

	// VariableDeltaUSDLow is the lower bound of the variable cost band.
	// Computed as: point_estimate × (1 − ConfidenceBandPct/100).
	VariableDeltaUSDLow float64 `json:"variableDeltaUsdLow"`

	// VariableDeltaUSDHigh is the upper bound of the variable cost band.
	// Computed as: point_estimate × (1 + ConfidenceBandPct/100).
	VariableDeltaUSDHigh float64 `json:"variableDeltaUsdHigh"`

	// ConfidenceBandPct is the uncertainty factor (30 = ±30%, 50 = ±50%).
	// 30: flow log data available.
	// 50: heuristic estimate, flow logs disabled.
	ConfidenceBandPct int `json:"confidenceBandPct"`

	// PriceSourceDate is the earliest effectiveStartDate across all Retail API
	// price entries fetched. Format: "YYYY-MM-DD". Empty if no fixed lookups ran.
	PriceSourceDate string `json:"priceSourceDate"`

	// ExistingFixedMonthlyUSD is the current monthly fixed cost of the topology
	// before the delta. List price only — NOT actual billed spend.
	// Informational context only; not part of the delta forecast.
	ExistingFixedMonthlyUSD float64 `json:"existingFixedMonthlyUsd"`

	// LineItems breaks down the fixed delta into per-resource components.
	// Each item represents one billable resource change.
	LineItems []CostLineItem `json:"lineItems"`

	// Caveats is a list of advisory strings explaining estimate limitations.
	// Always contains at least the three mandatory caveats from §8.1.
	Caveats []string `json:"caveats"`
}

// CostLineItem is one billable resource change in the fixed cost breakdown.
type CostLineItem struct {
	Resource     string  `json:"resource"`
	ResourceType string  `json:"resourceType"` // "PublicIP" | "VPNGateway" | "ExpressRouteGateway" | "AzureFirewall" | "PrivateEndpoint"
	ChangeType   string  `json:"changeType"`   // "Add" | "Remove" | "Existing"
	SKU          string  `json:"sku"`
	MonthlyUSD   float64 `json:"monthlyUsd"`  // positive = cost added, negative = cost removed; 0 = informational
	Region       string  `json:"region"`
	PriceSource  string  `json:"priceSource"` // always "retail-prices-api"
}

// mandatoryCaveats are appended to every CostForecast (§8.1 of SIMULATION_MODEL.md).
var mandatoryCaveats = []string{
	"Fixed costs reflect Azure Retail Prices API list prices; EA/MCA contract discounts are not applied.",
	"Variable costs are estimated from traffic volume data and may not reflect actual billing.",
	"Costs reflect the change introduced by the delta; they do not represent the total topology cost.",
}

// ForecastCost estimates the monthly cost impact of the given topology delta.
//
// region is the Azure ARM region name (e.g. "eastus") used for Retail Prices API
// lookups. When region is empty ForecastCost defaults to "eastus" and appends a
// caveat recommending the caller supply the correct region.
//
// cache must be non-nil; create one with NewPriceCache() or NewPriceCacheWithClient().
// flows may be a zero-value FlowSummary (triggers heuristic-only variable cost).
//
// All fixed costs are EXACT list prices. Variable costs are a ±30% band (±50% when
// flow log data is unavailable). Do not conflate with Azure Cost Management actuals.
func ForecastCost(
	ctx context.Context,
	fx *graph.Fixture,
	delta simulator.TopologyDelta,
	cache *PriceCache,
	flows FlowSummary,
	region string,
) (CostForecast, error) {
	if cache == nil {
		return CostForecast{}, fmt.Errorf("ForecastCost: cache must be non-nil")
	}

	if region == "" {
		region = "eastus"
	}

	fc := CostForecast{
		Caveats: append([]string(nil), mandatoryCaveats...),
	}

	// ---- fixed costs from the delta ----
	var priceSourceDate string
	if err := applyFixedCosts(ctx, fx, delta, cache, region, &fc, &priceSourceDate); err != nil {
		return fc, err
	}

	// ---- existing fixed costs (informational context) ----
	if err := applyExistingCosts(ctx, fx, cache, region, &fc, &priceSourceDate); err != nil {
		// Non-fatal — existing cost context is informational only.
		fc.Caveats = append(fc.Caveats,
			fmt.Sprintf("existing fixed cost estimate partially unavailable: %v", err))
	}

	fc.PriceSourceDate = priceSourceDate

	// ---- variable costs (band estimate) ----
	applyVariableCosts(fx, delta, flows, &fc)

	return fc, nil
}

// applyFixedCosts evaluates fixed-price resource changes introduced by the delta.
func applyFixedCosts(ctx context.Context, fx *graph.Fixture, delta simulator.TopologyDelta, cache *PriceCache, region string, fc *CostForecast, sourceDate *string) error {
	switch {
	case delta.AddPublicIP != nil:
		return pipAddCost(ctx, delta.AddPublicIP.PIPName, "Standard Static IPv4", region, +1, "Add", cache, fc, sourceDate)

	case delta.RemovePublicIP != nil:
		sku, pipName := pipSKUFromNIC(fx, delta.RemovePublicIP.NICName)
		if pipName == "" {
			fc.Caveats = append(fc.Caveats,
				fmt.Sprintf("RemovePublicIP: NIC %q has no public IP; fixed cost delta is $0", delta.RemovePublicIP.NICName))
			return nil
		}
		return pipAddCost(ctx, pipName, sku, region, -1, "Remove", cache, fc, sourceDate)

	case delta.ModifyRoute != nil && delta.ModifyRoute.NewNextHopType == "VirtualNetworkGateway":
		// Routing through an existing gateway — report gateway cost as context.
		return gatewayContextCosts(ctx, fx, cache, region, fc, sourceDate)

	case delta.AddPeering != nil && (delta.AddPeering.AllowGatewayTransit || delta.AddPeering.UseRemoteGateways):
		// Gateway transit peering — report gateway cost as context.
		return gatewayContextCosts(ctx, fx, cache, region, fc, sourceDate)
	}
	return nil
}

// pipAddCost adds a PIP line item (positive or negative depending on sign).
func pipAddCost(ctx context.Context, pipName, sku, region string, sign float64, changeType string, cache *PriceCache, fc *CostForecast, sourceDate *string) error {
	filter := PIPFilter(region, sku)
	price, sd, err := cache.Lookup(ctx, filter)
	if err != nil {
		return fmt.Errorf("PIP cost lookup (%s): %w", sku, err)
	}
	if price == 0 && sd == "" {
		fc.Caveats = append(fc.Caveats,
			fmt.Sprintf("PIP SKU %q not found in Retail Prices API for region %q; fixed cost delta may be incomplete", sku, region))
	}
	advanceSourceDate(sourceDate, sd)

	delta := price * sign
	fc.FixedDeltaUSD += delta
	fc.LineItems = append(fc.LineItems, CostLineItem{
		Resource:     pipName,
		ResourceType: "PublicIP",
		ChangeType:   changeType,
		SKU:          sku,
		MonthlyUSD:   delta,
		Region:       region,
		PriceSource:  "retail-prices-api",
	})
	return nil
}

// gatewayContextCosts adds existing VPN/ER gateway costs as informational line items
// (the gateways exist already; they are not being added or removed in Phase 2).
func gatewayContextCosts(ctx context.Context, fx *graph.Fixture, cache *PriceCache, region string, fc *CostForecast, sourceDate *string) error {
	for _, gw := range fx.ResourceGraph.VirtualNetworkGateways {
		sku := gwSKUName(gw)
		var filter string
		if strings.EqualFold(gw.GatewayType, "ExpressRoute") {
			filter = ERGatewayFilter(region, sku)
		} else {
			filter = VPNGatewayFilter(region, sku)
		}
		price, sd, err := cache.Lookup(ctx, filter)
		if err != nil {
			return fmt.Errorf("gateway cost lookup (%s): %w", gw.Name, err)
		}
		advanceSourceDate(sourceDate, sd)
		fc.ExistingFixedMonthlyUSD += price
		fc.LineItems = append(fc.LineItems, CostLineItem{
			Resource:     gw.Name,
			ResourceType: resourceTypeForGateway(gw.GatewayType),
			ChangeType:   "Existing",
			SKU:          sku,
			MonthlyUSD:   0, // informational — not part of delta
			Region:       region,
			PriceSource:  "retail-prices-api",
		})
	}
	return nil
}

// applyExistingCosts populates ExistingFixedMonthlyUSD with the current topology's
// fixed costs (firewalls, Private Endpoints). Informational — not part of the delta.
func applyExistingCosts(ctx context.Context, fx *graph.Fixture, cache *PriceCache, region string, fc *CostForecast, sourceDate *string) error {
	// Azure Firewall.
	if fw := fx.AzureFirewall; fw != nil {
		skuTier := fw.SKUTier
		if skuTier == "" {
			skuTier = "Standard" // safe default
		}
		price, sd, err := cache.Lookup(ctx, FirewallFilter(region, skuTier))
		if err != nil {
			return fmt.Errorf("firewall cost lookup: %w", err)
		}
		advanceSourceDate(sourceDate, sd)
		fc.ExistingFixedMonthlyUSD += price
		fc.LineItems = append(fc.LineItems, CostLineItem{
			Resource:     fw.Name,
			ResourceType: "AzureFirewall",
			ChangeType:   "Existing",
			SKU:          skuTier,
			MonthlyUSD:   0,
			Region:       region,
			PriceSource:  "retail-prices-api",
		})
	}

	// Private Endpoints — fixed hourly charge × count.
	if n := len(fx.ResourceGraph.PrivateEndpoints); n > 0 {
		price, sd, err := cache.Lookup(ctx, PrivateEndpointFilter(region))
		if err != nil {
			return fmt.Errorf("private endpoint cost lookup: %w", err)
		}
		advanceSourceDate(sourceDate, sd)
		// price is per-endpoint hourly rate (unitOfMeasure="1 Hour"). Convert to monthly.
		// API returns unitPrice for "744 Hours" measure — treat as monthly directly.
		fc.ExistingFixedMonthlyUSD += price * float64(n)
	}

	return nil
}

// applyVariableCosts estimates data-transfer variable costs and writes them into fc.
func applyVariableCosts(fx *graph.Fixture, delta simulator.TopologyDelta, flows FlowSummary, fc *CostForecast) {
	estimatedGB, flowLogsUsed := EstimateTrafficGB(fx, delta, flows)
	if estimatedGB == 0 {
		return // no variable cost trigger for this delta type
	}

	pricePerGB, category := variablePricePerGB(delta)
	if pricePerGB == 0 {
		return // delta type has no variable cost (e.g. AddPublicIP, AddSubnet with no NAT GW)
	}

	pointEstimate := estimatedGB * pricePerGB
	bandPct := 30
	if !flowLogsUsed {
		bandPct = 50
		fc.Caveats = append(fc.Caveats,
			fmt.Sprintf("Variable cost estimated from subscription heuristic — Flow Logs not enabled for affected segment; enable Flow Logs for a tighter estimate."))
	}
	fc.Caveats = append(fc.Caveats,
		fmt.Sprintf("Variable cost category: %s (%.1f GB/mo × $%.4f/GB)", category, estimatedGB, pricePerGB))

	factor := float64(bandPct) / 100.0
	fc.VariableDeltaUSDLow = pointEstimate * (1 - factor)
	fc.VariableDeltaUSDHigh = pointEstimate * (1 + factor)
	fc.ConfidenceBandPct = bandPct
}

// variablePricePerGB returns ($/GB, category) for the delta type that triggers
// variable data-transfer costs. Returns (0, "") when the delta has no variable cost.
func variablePricePerGB(delta simulator.TopologyDelta) (float64, string) {
	switch {
	case delta.ModifyRoute != nil && delta.ModifyRoute.NewNextHopType == "VirtualAppliance":
		// Traffic now routes through an NVA or Azure Firewall.
		return 0.016, "Firewall/NVA data processing"
	case delta.AddPeering != nil:
		// Cross-region peering egress (conservative mid-range; intra-region is cheaper).
		return 0.035, "VNet peering data transfer"
	case delta.AddSubnet != nil:
		// NAT Gateway data processing.
		return 0.045, "NAT Gateway data processing"
	default:
		return 0, ""
	}
}

// ---- helper functions ----

// pipSKUFromNIC returns (sku, pipName) for the PIP currently attached to nicName.
// Falls back to "Standard Static IPv4" if Phase 2 adapter fields are absent.
func pipSKUFromNIC(fx *graph.Fixture, nicName string) (sku, pipName string) {
	for _, nic := range fx.ResourceGraph.NetworkInterfaces {
		if nic.Name != nicName || nic.PublicIP == nil {
			continue
		}
		pipName = *nic.PublicIP
		break
	}
	if pipName == "" {
		return "", ""
	}
	for _, pip := range fx.ResourceGraph.PublicIPAddresses {
		if pip.Name == pipName {
			if pip.SKU != "" && pip.AllocationMethod != "" {
				return pip.SKU + " " + pip.AllocationMethod + " IPv4", pipName
			}
		}
	}
	return "Standard Static IPv4", pipName
}

// gwSKUName returns the formatted SKU name for a gateway (e.g. "VpnGw1 Gateway").
func gwSKUName(gw graph.VirtualNetworkGateway) string {
	if gw.SKU == "" {
		if strings.EqualFold(gw.GatewayType, "ExpressRoute") {
			return "ErGw1AZ Gateway"
		}
		return "VpnGw1 Gateway"
	}
	if strings.HasSuffix(gw.SKU, " Gateway") {
		return gw.SKU
	}
	return gw.SKU + " Gateway"
}

func resourceTypeForGateway(gwType string) string {
	if strings.EqualFold(gwType, "ExpressRoute") {
		return "ExpressRouteGateway"
	}
	return "VPNGateway"
}

// advanceSourceDate updates *current to newDate if newDate is earlier (oldest wins).
// "oldest" means earliest price data — conservative for reporting.
func advanceSourceDate(current *string, newDate string) {
	if newDate == "" {
		return
	}
	if *current == "" || newDate < *current {
		*current = newDate
	}
}
