package adapter

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	armnet "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"

	"github.com/aaraminds/azure-nettopo-engine/internal/graph"
)

const (
	nwMaxRetries    = 5
	nwSemaphoreSize = 10
	nwPollFrequency = 2 * time.Second
	nwCallDeadline  = 60 * time.Second
)

// fetchNetworkWatcher runs Effective Security Rules and Effective Routes in
// parallel for all eligible NICs, bounded by a shared semaphore of size 10.
//
// Fail-open: a NIC that fails NW enrichment is omitted from the result maps
// (not inserted with empty slices). Errors are logged but do not abort the batch.
func (a *adapter) fetchNetworkWatcher(
	ctx context.Context,
	metas []nicMeta,
	nwLocs map[string]nwLocation,
) (nwResult, error) {
	log.Printf("INFO: starting NW enrichment for %d NICs", len(metas))
	start := time.Now()

	ifaceClient, err := armnet.NewInterfacesClient(a.subscriptionID, a.cred, nil)
	if err != nil {
		return nwResult{}, fmt.Errorf("interfaces client: %w", err)
	}

	// One shared semaphore across ALL NW calls (ESR + ER for all NICs).
	sem := make(chan struct{}, nwSemaphoreSize)

	type esrItem struct {
		nicName string
		rules   []graph.SecRule
		err     error
	}
	type erItem struct {
		nicName string
		routes  []graph.Route
		err     error
	}

	// Filter to NICs that have a Network Watcher in their location.
	type eligible struct {
		meta nicMeta
		nw   nwLocation
	}
	var eligible_ []eligible
	for _, m := range metas {
		nw, ok := nwLocs[m.location]
		if !ok {
			log.Printf("WARN: no Network Watcher for location %s; skipping NIC %s", m.location, m.nic.Name)
			continue
		}
		eligible_ = append(eligible_, eligible{m, nw})
	}

	esrResults := make([]esrItem, len(eligible_))
	erResults := make([]erItem, len(eligible_))

	var wg sync.WaitGroup
	for i, e := range eligible_ {
		i, e := i, e // capture
		wg.Add(2)

		// ESR goroutine
		go func() {
			defer wg.Done()
			rules, err := a.fetchESR(ctx, ifaceClient, e.meta, sem)
			esrResults[i] = esrItem{e.meta.nic.Name, rules, err}
		}()

		// ER goroutine
		go func() {
			defer wg.Done()
			routes, err := a.fetchER(ctx, ifaceClient, e.meta, sem)
			erResults[i] = erItem{e.meta.nic.Name, routes, err}
		}()
	}
	wg.Wait()

	effectiveRules := make(map[string][]graph.SecRule)
	effectiveRoutes := make(map[string][]graph.Route)
	errCount := 0

	for _, r := range esrResults {
		if r.err != nil {
			log.Printf("ERROR: NW ESR call failed for NIC %s; omitting from results: %v", r.nicName, r.err)
			errCount++
		} else if r.nicName != "" {
			effectiveRules[r.nicName] = r.rules
		}
	}
	for _, r := range erResults {
		if r.err != nil {
			log.Printf("ERROR: NW ER call failed for NIC %s; omitting from results: %v", r.nicName, r.err)
			errCount++
		} else if r.nicName != "" {
			effectiveRoutes[r.nicName] = r.routes
		}
	}

	elapsed := time.Since(start).Milliseconds()
	log.Printf("INFO: NW enrichment complete; %d NICs processed in %dms; %d errors",
		len(effectiveRules), elapsed, errCount)

	return nwResult{effectiveRules, effectiveRoutes}, nil
}

// fetchESR fetches Effective Security Rules for a single NIC with retry/backoff.
func (a *adapter) fetchESR(
	ctx context.Context,
	client *armnet.InterfacesClient,
	nm nicMeta,
	sem chan struct{},
) ([]graph.SecRule, error) {
	nicName := nm.nic.Name
	nicRG := nm.resourceGroup

	for attempt := 0; attempt < nwMaxRetries; attempt++ {
		if err := acquireSem(ctx, sem); err != nil {
			return nil, err
		}

		callCtx, cancel := context.WithTimeout(ctx, nwCallDeadline)
		poller, err := client.BeginListEffectiveNetworkSecurityGroups(callCtx, nicRG, nicName, nil)
		if err != nil {
			cancel()
			releaseSem(sem)
			if is429(err) && attempt < nwMaxRetries-1 {
				log.Printf("WARN: NW call throttled for NIC %s; retrying (%d/%d)", nicName, attempt+1, nwMaxRetries)
				nwBackoff(attempt)
				continue
			}
			log.Printf("ERROR: NW call failed for NIC %s after %d retries; omitting from results", nicName, nwMaxRetries)
			return nil, fmt.Errorf("BeginListEffectiveNSG %s: %w", nicName, err)
		}

		result, err := poller.PollUntilDone(callCtx, &runtime.PollUntilDoneOptions{Frequency: nwPollFrequency})
		cancel()
		releaseSem(sem)

		if err != nil {
			if is429(err) && attempt < nwMaxRetries-1 {
				log.Printf("WARN: NW call throttled for NIC %s; retrying (%d/%d)", nicName, attempt+1, nwMaxRetries)
				nwBackoff(attempt)
				continue
			}
			log.Printf("ERROR: NW call failed for NIC %s after %d retries; omitting from results", nicName, nwMaxRetries)
			return nil, fmt.Errorf("PollUntilDone ESR %s: %w", nicName, err)
		}

		// Flatten all value[*].effectiveSecurityRules[] — do NOT deduplicate.
		var rules []graph.SecRule
		for _, ensg := range result.Value {
			if ensg == nil {
				continue
			}
			for _, r := range ensg.EffectiveSecurityRules {
				rules = append(rules, expandEffectiveRule(r)...)
			}
		}
		return rules, nil
	}

	log.Printf("ERROR: NW call failed for NIC %s after %d retries; omitting from results", nicName, nwMaxRetries)
	return nil, fmt.Errorf("max retries exceeded for NIC %s ESR", nicName)
}

// fetchER fetches Effective Routes for a single NIC with retry/backoff.
func (a *adapter) fetchER(
	ctx context.Context,
	client *armnet.InterfacesClient,
	nm nicMeta,
	sem chan struct{},
) ([]graph.Route, error) {
	nicName := nm.nic.Name
	nicRG := nm.resourceGroup

	for attempt := 0; attempt < nwMaxRetries; attempt++ {
		if err := acquireSem(ctx, sem); err != nil {
			return nil, err
		}

		callCtx, cancel := context.WithTimeout(ctx, nwCallDeadline)
		poller, err := client.BeginGetEffectiveRouteTable(callCtx, nicRG, nicName, nil)
		if err != nil {
			cancel()
			releaseSem(sem)
			if is429(err) && attempt < nwMaxRetries-1 {
				log.Printf("WARN: NW call throttled for NIC %s; retrying (%d/%d)", nicName, attempt+1, nwMaxRetries)
				nwBackoff(attempt)
				continue
			}
			log.Printf("ERROR: NW call failed for NIC %s after %d retries; omitting from results", nicName, nwMaxRetries)
			return nil, fmt.Errorf("BeginGetEffectiveRouteTable %s: %w", nicName, err)
		}

		result, err := poller.PollUntilDone(callCtx, &runtime.PollUntilDoneOptions{Frequency: nwPollFrequency})
		cancel()
		releaseSem(sem)

		if err != nil {
			if is429(err) && attempt < nwMaxRetries-1 {
				log.Printf("WARN: NW call throttled for NIC %s; retrying (%d/%d)", nicName, attempt+1, nwMaxRetries)
				nwBackoff(attempt)
				continue
			}
			log.Printf("ERROR: NW call failed for NIC %s after %d retries; omitting from results", nicName, nwMaxRetries)
			return nil, fmt.Errorf("PollUntilDone ER %s: %w", nicName, err)
		}

		var routes []graph.Route
		for _, r := range result.Value {
			routes = append(routes, convertEffectiveRoute(r))
		}
		return routes, nil
	}

	log.Printf("ERROR: NW call failed for NIC %s after %d retries; omitting from results", nicName, nwMaxRetries)
	return nil, fmt.Errorf("max retries exceeded for NIC %s ER", nicName)
}

// ─── Semaphore helpers ────────────────────────────────────────────────────────

// acquireSem blocks until a semaphore slot is available or ctx is cancelled.
func acquireSem(ctx context.Context, sem chan struct{}) error {
	select {
	case sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// releaseSem releases a semaphore slot.
func releaseSem(sem chan struct{}) { <-sem }

// ─── Retry helpers ────────────────────────────────────────────────────────────

// is429 returns true if err is an Azure HTTP 429 (Too Many Requests) response.
func is429(err error) bool {
	var respErr *azcore.ResponseError
	if errors.As(err, &respErr) {
		return respErr.StatusCode == 429
	}
	return false
}

// nwBackoff sleeps for 2s * 2^attempt with ±20% jitter.
func nwBackoff(attempt int) {
	base := float64(2*time.Second) * math.Pow(2, float64(attempt))
	jitter := 0.8 + rand.Float64()*0.4 // [0.8, 1.2]
	d := time.Duration(base * jitter)
	time.Sleep(d)
}

// ─── Effective rule conversion ────────────────────────────────────────────────

// expandEffectiveRule converts one armnetwork.EffectiveNetworkSecurityRule into
// one or more graph.SecRule values.
//
// Multi-value expansion: if sourceAddressPrefixes has >1 element OR
// destinationPortRanges has >1 element, emit the Cartesian product.
func expandEffectiveRule(r *armnet.EffectiveNetworkSecurityRule) []graph.SecRule {
	if r == nil {
		return nil
	}

	name := derefStr(r.Name)
	priority := int32(0)
	if r.Priority != nil {
		priority = *r.Priority
	}
	direction := ""
	if r.Direction != nil {
		direction = string(*r.Direction)
	}
	access := ""
	if r.Access != nil {
		access = string(*r.Access)
	}
	protocol := ""
	if r.Protocol != nil {
		protocol = string(*r.Protocol)
	}

	sources := effectiveSourcePrefixes(r)
	ports := effectiveDestPorts(r)

	var out []graph.SecRule
	for _, src := range sources {
		for _, port := range ports {
			out = append(out, graph.SecRule{
				Name:                 name,
				Priority:             int(priority),
				Direction:            direction,
				Access:               access,
				Protocol:             protocol,
				SourceAddressPrefix:  src,
				DestinationPortRange: port,
				Source:               src, // invariant: Source = SourceAddressPrefix
			})
		}
	}
	return out
}

// effectiveSourcePrefixes returns the source address prefixes for an effective rule.
// Prefers the singular sourceAddressPrefix; falls back to the plural array.
func effectiveSourcePrefixes(r *armnet.EffectiveNetworkSecurityRule) []string {
	if r.SourceAddressPrefix != nil && *r.SourceAddressPrefix != "" {
		return []string{*r.SourceAddressPrefix}
	}
	var out []string
	for _, s := range r.SourceAddressPrefixes {
		if s != nil && *s != "" {
			out = append(out, *s)
		}
	}
	if len(out) == 0 {
		return []string{""}
	}
	return out
}

// effectiveDestPorts returns the destination port ranges for an effective rule.
// Prefers the singular destinationPortRange; falls back to the plural array.
func effectiveDestPorts(r *armnet.EffectiveNetworkSecurityRule) []string {
	if r.DestinationPortRange != nil && *r.DestinationPortRange != "" {
		return []string{*r.DestinationPortRange}
	}
	var out []string
	for _, p := range r.DestinationPortRanges {
		if p != nil && *p != "" {
			out = append(out, *p)
		}
	}
	if len(out) == 0 {
		return []string{""}
	}
	return out
}

// convertEffectiveRoute converts one armnetwork.EffectiveRoute to a graph.Route.
func convertEffectiveRoute(r *armnet.EffectiveRoute) graph.Route {
	if r == nil {
		return graph.Route{}
	}
	addrPrefix := ""
	if len(r.AddressPrefix) > 0 && r.AddressPrefix[0] != nil {
		addrPrefix = *r.AddressPrefix[0]
	}
	nextHopIP := ""
	if len(r.NextHopIPAddress) > 0 && r.NextHopIPAddress[0] != nil {
		nextHopIP = *r.NextHopIPAddress[0]
	}
	nextHopType := ""
	if r.NextHopType != nil {
		nextHopType = string(*r.NextHopType)
	}
	return graph.Route{
		Name:             derefStr(r.Name),
		AddressPrefix:    addrPrefix,
		NextHopType:      nextHopType,
		NextHopIPAddress: nextHopIP,
	}
}
