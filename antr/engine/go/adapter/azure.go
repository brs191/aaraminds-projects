// Package adapter fetches a complete network topology snapshot from live Azure
// APIs and assembles a *graph.Fixture for the deterministic analysis engine.
//
// Usage:
//
//	cred, _ := azidentity.NewDefaultAzureCredential(nil)
//	fixture, err := adapter.FetchFixture(ctx, cred, subscriptionID)
//	findings := analyze.Analyze(fixture)
package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"golang.org/x/sync/errgroup"

	"github.com/aaraminds/azure-nettopo-engine/internal/graph"
)

const armBase = "https://management.azure.com"

// FetchFixture fetches a complete topology fixture for a single Azure subscription.
// cred: DefaultAzureCredential (Managed Identity in prod, az login in dev)
// subscriptionID: Azure subscription GUID
func FetchFixture(ctx context.Context, cred azcore.TokenCredential, subscriptionID string) (*graph.Fixture, error) {
	a := &adapter{
		cred:           cred,
		subscriptionID: subscriptionID,
		httpClient:     &http.Client{Timeout: 90 * time.Second},
	}

	// Step A: parallel Resource Graph bulk queries
	rg, err := a.fetchResourceGraph(ctx)
	if err != nil {
		return nil, fmt.Errorf("resource graph: %w", err)
	}

	// Steps B+C (NW effective rules/routes), D (AVNM), E (Firewall) run in parallel.
	var (
		nwData nwResult
		avnm   graph.AVNM
		fws    []*graph.Firewall
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		nwData, err = a.fetchNetworkWatcher(gctx, rg.nicMetas, rg.nwLocations)
		return err
	})

	g.Go(func() error {
		var err error
		avnm, err = a.fetchAVNM(gctx)
		return err
	})

	if len(rg.rawFWs) > 0 {
		g.Go(func() error {
			for _, rf := range rg.rawFWs {
				f, err := a.fetchFirewall(gctx, rf, rg.ResourceGraph.PublicIPAddresses)
				if err != nil {
					return err
				}
				if f != nil {
					fws = append(fws, f)
				}
			}
			// Deterministic order regardless of ARG row order / fetch scheduling.
			sort.Slice(fws, func(i, j int) bool { return fws[i].Name < fws[j].Name })
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Step F: assemble the fixture
	fixture := &graph.Fixture{
		Subscription:  subscriptionID,
		ResourceGraph: rg.ResourceGraph,
		NetworkWatcher: graph.NetworkWatcher{
			EffectiveSecurityRules: nwData.effectiveRules,
			EffectiveRoutes:        nwData.effectiveRoutes,
			IncompleteNICs:         nwData.incompleteNICs,
		},
		AVNM:                      avnm,
		AzureFirewalls:            fws,
		CrossSubscriptionPeerings: deriveCrossSubPeerings(rg.ResourceGraph.VirtualNetworks),
	}
	return fixture, nil
}

// ─── internal adapter ─────────────────────────────────────────────────────────

type adapter struct {
	cred           azcore.TokenCredential
	subscriptionID string
	httpClient     *http.Client
}

// ─── NW result ────────────────────────────────────────────────────────────────

type nwResult struct {
	effectiveRules  map[string][]graph.SecRule
	effectiveRoutes map[string][]graph.Route
	incompleteNICs  []string
}

// ─── NIC metadata (adapter-internal; not in graph.NIC) ────────────────────────

type nicMeta struct {
	nic           graph.NIC
	resourceGroup string
	location      string
}

// ─── Network Watcher location map ────────────────────────────────────────────

type nwLocation struct {
	name          string
	resourceGroup string
}

// ─── Detected firewall (intermediate) ────────────────────────────────────────

type rawFirewall struct {
	name             string
	resourceGroup    string
	privateIP        string
	publicIPID       string
	firewallPolicyID string
	sku              string
}

// ─── Resource Graph result ────────────────────────────────────────────────────

type rgResult struct {
	graph.ResourceGraph
	nicMetas    []nicMeta
	nwLocations map[string]nwLocation
	rawFWs      []*rawFirewall // ALL detected firewalls (external review F5)
}

// ─── HTTP helpers ─────────────────────────────────────────────────────────────

// bearerToken obtains a short-lived Azure AD access token for ARM.
func (a *adapter) bearerToken(ctx context.Context) (string, error) {
	tok, err := a.cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	})
	if err != nil {
		return "", fmt.Errorf("get token: %w", err)
	}
	return tok.Token, nil
}

// getJSON makes an authenticated GET against the ARM API and decodes the response.
func (a *adapter) getJSON(ctx context.Context, url string, out interface{}) error {
	token, err := a.bearerToken(ctx)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return json.Unmarshal(body, out)
}

// armURL builds a fully qualified ARM URL.
func (a *adapter) armURL(path string, apiVersion string) string {
	if strings.Contains(path, "?") {
		return armBase + path + "&api-version=" + apiVersion
	}
	return armBase + path + "?api-version=" + apiVersion
}

// subPath returns the path prefix for the subscription.
func (a *adapter) subPath() string {
	return "/subscriptions/" + a.subscriptionID
}

// ─── Paginated ARM response envelope ─────────────────────────────────────────

type armPage struct {
	Value    []json.RawMessage `json:"value"`
	NextLink string            `json:"nextLink"`
}

// listAll follows nextLink pagination and returns all values.
func (a *adapter) listAll(ctx context.Context, url string) ([]json.RawMessage, error) {
	var all []json.RawMessage
	for url != "" {
		var page armPage
		if err := a.getJSON(ctx, url, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Value...)
		url = page.NextLink
	}
	return all, nil
}

// ─── JSON map helpers ─────────────────────────────────────────────────────────

func getString(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func getBool(m map[string]interface{}, key string) bool {
	v, ok := m[key]
	if !ok || v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

func getInt(m map[string]interface{}, key string) int {
	v, ok := m[key]
	if !ok || v == nil {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	case float64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	}
	return 0
}

func getStringSlice(m map[string]interface{}, key string) []string {
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	var out []string
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func getSlice(m map[string]interface{}, key string) []interface{} {
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	return arr
}

func getMap(m map[string]interface{}, key string) map[string]interface{} {
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	sub, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	return sub
}

// toRows marshals data as JSON and unmarshals it into []map[string]interface{}.
// Safe for any internal representation returned by Resource Graph.
func toRows(data interface{}) ([]map[string]interface{}, error) {
	if data == nil {
		return nil, nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var rows []map[string]interface{}
	return rows, json.Unmarshal(b, &rows)
}

// ─── ARM ID helpers ───────────────────────────────────────────────────────────

// extractSubnet extracts "{vnetName}/{subnetName}" from an ARM subnet resource ID.
//
//	ARM: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/virtualNetworks/{vnet}/subnets/{subnet}
//	Result: "{vnet}/{subnet}"
func extractSubnet(armID string) string {
	if armID == "" {
		return ""
	}
	lower := strings.ToLower(armID)
	vnIdx := strings.Index(lower, "/virtualnetworks/")
	if vnIdx < 0 {
		return ""
	}
	rest := armID[vnIdx+len("/virtualnetworks/"):]
	snIdx := strings.Index(strings.ToLower(rest), "/subnets/")
	if snIdx < 0 {
		return ""
	}
	vnet := rest[:snIdx]
	subnet := rest[snIdx+len("/subnets/"):]
	// Trim any sub-resource path segments after the subnet name.
	if i := strings.Index(subnet, "/"); i >= 0 {
		subnet = subnet[:i]
	}
	return vnet + "/" + subnet
}

// extractResourceName returns the last path segment of an ARM resource ID.
// E.g. ".../networkSecurityGroups/my-nsg" → "my-nsg"
func extractResourceName(armID string) string {
	if armID == "" {
		return ""
	}
	parts := strings.Split(strings.TrimRight(armID, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

// extractResourceGroup extracts the resource group name from an ARM resource ID.
func extractResourceGroup(armID string) string {
	lower := strings.ToLower(armID)
	idx := strings.Index(lower, "/resourcegroups/")
	if idx < 0 {
		return ""
	}
	rest := armID[idx+len("/resourcegroups/"):]
	if i := strings.Index(rest, "/"); i >= 0 {
		return rest[:i]
	}
	return rest
}

// ptr returns a pointer to v.
func ptr[T any](v T) *T { return &v }

// derefStr dereferences a *string; returns "" if nil.
func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
