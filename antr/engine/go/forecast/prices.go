// Package forecast implements the Phase 2 cost impact estimation for topology deltas.
// Fixed costs are EXACT — fetched from the Azure Retail Prices API (public, unauthenticated).
// Variable costs are a BAND — estimated from VNet Flow Logs or the subscription heuristic.
//
// IMPORTANT: All prices are Azure Retail Prices list rates. EA/MCA contract discounts and
// reservation commitments are NOT applied. Do not present these values as actual billed spend.
package forecast

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	retailPricesBase       = "https://prices.azure.com/api/retail/prices"
	retailPricesAPIVersion = "2023-01-01-preview"
	// cacheTTL is how long a price entry is considered fresh. Prices change
	// monthly, not hourly — 24h is conservative and safe for a session.
	cacheTTL = 24 * time.Hour
)

// priceResult holds one cached Retail Prices API lookup result.
type priceResult struct {
	monthlyUSD float64
	sourceDate string    // "YYYY-MM-DD" from effectiveStartDate
	fetchedAt  time.Time
}

// retailResponse is the JSON envelope returned by the Retail Prices API.
type retailResponse struct {
	Items        []retailItem `json:"Items"`
	NextPageLink *string      `json:"NextPageLink"`
}

// retailItem holds the fields we consume from each price record.
type retailItem struct {
	UnitPrice          float64 `json:"unitPrice"`
	RetailPrice        float64 `json:"retailPrice"`
	UnitOfMeasure      string  `json:"unitOfMeasure"`
	SkuName            string  `json:"skuName"`
	ServiceName        string  `json:"serviceName"`
	ArmRegionName      string  `json:"armRegionName"`
	Type               string  `json:"type"`
	EffectiveStartDate string  `json:"effectiveStartDate"` // "2024-01-01T00:00:00Z"
}

// PriceCache stores Retail Prices API results in memory.
// TTL is 24h (prices change monthly). Thread-safe for concurrent callers.
// Cache is NOT persisted across process restarts — prices are re-fetched on startup.
type PriceCache struct {
	mu         sync.RWMutex
	items      map[string]priceResult
	httpClient *http.Client
	clock      func() time.Time // injectable; defaults to time.Now
}

// NewPriceCache creates a PriceCache with default HTTP client and clock.
func NewPriceCache() *PriceCache {
	return &PriceCache{
		items:      make(map[string]priceResult),
		httpClient: &http.Client{Timeout: 15 * time.Second},
		clock:      time.Now,
	}
}

// NewPriceCacheWithClient creates a PriceCache with an injected HTTP client.
// Use this in tests to point at an httptest.Server instead of the real API.
func NewPriceCacheWithClient(client *http.Client) *PriceCache {
	return &PriceCache{
		items:      make(map[string]priceResult),
		httpClient: client,
		clock:      time.Now,
	}
}

// SetClock replaces the internal clock function. Used in tests to simulate TTL expiry.
func (c *PriceCache) SetClock(fn func() time.Time) {
	c.mu.Lock()
	c.clock = fn
	c.mu.Unlock()
}

// Lookup returns the monthly USD unit price and effectiveStartDate for the given
// OData filter string. Results are cached for 24h. When the API returns HTTP 429,
// Lookup returns a *RateLimitError — callers should back off then retry.
// Returns (0.0, "", nil) when the filter matches no items (unknown SKU/region).
func (c *PriceCache) Lookup(ctx context.Context, oDataFilter string) (float64, string, error) {
	// Fast path under read lock.
	c.mu.RLock()
	if entry, ok := c.items[oDataFilter]; ok {
		if c.clock().Sub(entry.fetchedAt) < cacheTTL {
			c.mu.RUnlock()
			return entry.monthlyUSD, entry.sourceDate, nil
		}
	}
	c.mu.RUnlock()

	// Slow path: fetch fresh data and cache.
	price, sourceDate, err := c.fetchFromAPI(ctx, oDataFilter)
	if err != nil {
		return 0, "", err
	}

	c.mu.Lock()
	c.items[oDataFilter] = priceResult{
		monthlyUSD: price,
		sourceDate: sourceDate,
		fetchedAt:  c.clock(),
	}
	c.mu.Unlock()

	return price, sourceDate, nil
}

// InvalidateAndRefetch removes the cached entry for oDataFilter and immediately
// fetches a fresh value. Called when the upstream responds with HTTP 429 and
// the caller has already waited through the Retry-After interval.
func (c *PriceCache) InvalidateAndRefetch(ctx context.Context, oDataFilter string) (float64, string, error) {
	c.mu.Lock()
	delete(c.items, oDataFilter)
	c.mu.Unlock()
	return c.Lookup(ctx, oDataFilter)
}

func (c *PriceCache) fetchFromAPI(ctx context.Context, oDataFilter string) (float64, string, error) {
	params := url.Values{}
	params.Set("api-version", retailPricesAPIVersion)
	params.Set("$filter", oDataFilter)
	reqURL := retailPricesBase + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, "", fmt.Errorf("retail prices: build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("retail prices: HTTP GET: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return 0, "", &RateLimitError{Filter: oDataFilter}
	}
	if resp.StatusCode != http.StatusOK {
		return 0, "", fmt.Errorf("retail prices: HTTP %d for filter %q", resp.StatusCode, oDataFilter)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("retail prices: read body: %w", err)
	}

	var rr retailResponse
	if err := json.Unmarshal(body, &rr); err != nil {
		return 0, "", fmt.Errorf("retail prices: unmarshal: %w", err)
	}
	if len(rr.Items) == 0 {
		return 0.0, "", nil // unknown SKU — callers add a caveat
	}

	item := rr.Items[0]
	sourceDate := ""
	if len(item.EffectiveStartDate) >= 10 {
		sourceDate = item.EffectiveStartDate[:10] // "YYYY-MM-DD"
	}
	return item.UnitPrice, sourceDate, nil
}

// RateLimitError is returned when the Retail Prices API responds with HTTP 429.
// Callers should honour Retry-After and call InvalidateAndRefetch.
type RateLimitError struct {
	Filter string
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("retail prices API rate limited (HTTP 429) for filter: %s", e.Filter)
}

// IsRateLimit returns true if err is a *RateLimitError.
func IsRateLimit(err error) bool {
	var rl *RateLimitError
	return errors.As(err, &rl)
}

// ---- OData filter constructors — one per resource type ----

// PIPFilter builds the OData filter for a Public IP Address SKU.
// sku is the combined SKU string e.g. "Standard Static IPv4".
func PIPFilter(region, sku string) string {
	return fmt.Sprintf(
		"serviceName eq 'Virtual Network' and productName eq 'IP Addresses' and armRegionName eq '%s' and skuName eq '%s' and type eq 'Consumption'",
		region, sku)
}

// VPNGatewayFilter builds the OData filter for a VPN Gateway SKU.
// sku is the SKU name e.g. "VpnGw1 Gateway".
func VPNGatewayFilter(region, sku string) string {
	return fmt.Sprintf(
		"serviceName eq 'VPN Gateway' and armRegionName eq '%s' and skuName eq '%s' and type eq 'Consumption'",
		region, sku)
}

// ERGatewayFilter builds the OData filter for an ExpressRoute Gateway SKU.
func ERGatewayFilter(region, sku string) string {
	return fmt.Sprintf(
		"serviceName eq 'ExpressRoute' and productName eq 'ExpressRoute Gateway' and armRegionName eq '%s' and skuName eq '%s' and type eq 'Consumption'",
		region, sku)
}

// FirewallFilter builds the OData filter for an Azure Firewall SKU tier.
// skuTier is e.g. "Standard" or "Premium".
func FirewallFilter(region, skuTier string) string {
	return fmt.Sprintf(
		"serviceName eq 'Azure Firewall' and armRegionName eq '%s' and skuName eq '%s' and type eq 'Consumption'",
		region, skuTier)
}

// PrivateEndpointFilter builds the OData filter for Private Endpoint hourly pricing.
func PrivateEndpointFilter(region string) string {
	return fmt.Sprintf(
		"serviceName eq 'Azure Private Link' and productName eq 'Private Endpoint' and armRegionName eq '%s' and type eq 'Consumption'",
		region)
}
