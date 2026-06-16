package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	armrg "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph"
	"golang.org/x/sync/errgroup"

	"github.com/aaraminds/azure-nettopo-engine/internal/graph"
)

// fetchResourceGraph runs all 7+ Resource Graph KQL queries in parallel and
// assembles the graph.ResourceGraph plus adapter-internal metadata.
func (a *adapter) fetchResourceGraph(ctx context.Context) (rgResult, error) {
	client, err := armrg.NewClient(a.cred, nil)
	if err != nil {
		return rgResult{}, fmt.Errorf("resource graph client: %w", err)
	}

	// Results channels (each query writes to its own slot via closure).
	var (
		vnets    []graph.VNet
		nsgs     []graph.NSG
		rts      []graph.RouteTable
		pips     []graph.PublicIP
		nicRaw   []map[string]interface{} // parsed NIC rows (need extra metadata)
		nwRaw    []map[string]interface{} // network watcher rows
		fwRaw    []map[string]interface{} // firewall rows
		peRaw    []map[string]interface{}
		lbRaw    []map[string]interface{}
		dnsRaw   []map[string]interface{}
		appgwRaw []map[string]interface{}
		aksRaw   []map[string]interface{}
		natgwRaw []map[string]interface{}
		vwanRaw  []map[string]interface{}
		vhubRaw  []map[string]interface{}
		apimRaw  []map[string]interface{}
		bastRaw  []map[string]interface{}
		vngRaw   []map[string]interface{}
		erRaw    []map[string]interface{}
		afdRaw   []map[string]interface{}
	)

	rg := func(kql string) ([]map[string]interface{}, error) {
		return a.runKQL(ctx, client, kql)
	}

	g, gctx := errgroup.WithContext(ctx)

	// VNets
	g.Go(func() error {
		rows, err := rg(kqlVNets(a.subscriptionID))
		if err != nil {
			return fmt.Errorf("vnets: %w", err)
		}
		vnets = parseVNets(rows)
		return nil
	})

	// NSGs
	g.Go(func() error {
		rows, err := rg(kqlNSGs(a.subscriptionID))
		if err != nil {
			return fmt.Errorf("nsgs: %w", err)
		}
		nsgs = parseNSGs(rows)
		return nil
	})

	// Route Tables
	g.Go(func() error {
		rows, err := rg(kqlRouteTables(a.subscriptionID))
		if err != nil {
			return fmt.Errorf("routetables: %w", err)
		}
		rts = parseRouteTables(rows)
		return nil
	})

	// Public IPs
	g.Go(func() error {
		rows, err := rg(kqlPublicIPs(a.subscriptionID))
		if err != nil {
			return fmt.Errorf("publicips: %w", err)
		}
		pips = parsePublicIPs(rows)
		return nil
	})

	// NICs (raw — needs rg+location for NW calls)
	g.Go(func() error {
		var err error
		nicRaw, err = rg(kqlNICs(a.subscriptionID))
		if err != nil {
			return fmt.Errorf("nics: %w", err)
		}
		return nil
	})

	// Network Watchers
	g.Go(func() error {
		var err error
		nwRaw, err = rg(kqlNetworkWatchers(a.subscriptionID))
		if err != nil {
			return fmt.Errorf("network watchers: %w", err)
		}
		return nil
	})

	// Azure Firewalls
	g.Go(func() error {
		var err error
		fwRaw, err = rg(kqlFirewalls(a.subscriptionID))
		if err != nil {
			return fmt.Errorf("firewalls: %w", err)
		}
		return nil
	})

	// Private Endpoints
	g.Go(func() error {
		var err error
		peRaw, err = rg(kqlPrivateEndpoints(a.subscriptionID))
		if err != nil {
			return fmt.Errorf("private endpoints: %w", err)
		}
		return nil
	})

	// Load Balancers
	g.Go(func() error {
		var err error
		lbRaw, err = rg(kqlLoadBalancers(a.subscriptionID))
		if err != nil {
			return fmt.Errorf("load balancers: %w", err)
		}
		return nil
	})

	// Private DNS Zones (metadata only; links fetched separately)
	g.Go(func() error {
		var err error
		dnsRaw, err = rg(kqlPrivateDNSZones())
		if err != nil {
			return fmt.Errorf("private dns zones: %w", err)
		}
		return nil
	})

	// Application Gateways
	g.Go(func() error {
		var err error
		appgwRaw, err = rg(kqlAppGateways())
		if err != nil {
			return fmt.Errorf("app gateways: %w", err)
		}
		return nil
	})

	// AKS
	g.Go(func() error {
		var err error
		aksRaw, err = rg(kqlAKS(a.subscriptionID))
		if err != nil {
			return fmt.Errorf("aks: %w", err)
		}
		return nil
	})

	// NAT Gateways
	g.Go(func() error {
		var err error
		natgwRaw, err = rg(kqlNATGateways(a.subscriptionID))
		if err != nil {
			return fmt.Errorf("nat gateways: %w", err)
		}
		return nil
	})

	// Virtual WANs
	g.Go(func() error {
		var err error
		vwanRaw, err = rg(kqlVirtualWANs(a.subscriptionID))
		if err != nil {
			return fmt.Errorf("virtual wans: %w", err)
		}
		return nil
	})

	// Virtual Hubs
	g.Go(func() error {
		var err error
		vhubRaw, err = rg(kqlVirtualHubs(a.subscriptionID))
		if err != nil {
			return fmt.Errorf("virtual hubs: %w", err)
		}
		return nil
	})

	// APIM
	g.Go(func() error {
		var err error
		apimRaw, err = rg(kqlAPIM(a.subscriptionID))
		if err != nil {
			return fmt.Errorf("apim: %w", err)
		}
		return nil
	})

	// Azure Bastion
	g.Go(func() error {
		var err error
		bastRaw, err = rg(kqlBastions(a.subscriptionID))
		if err != nil {
			return fmt.Errorf("bastions: %w", err)
		}
		return nil
	})

	// VNet Gateways
	g.Go(func() error {
		var err error
		vngRaw, err = rg(kqlVNetGateways(a.subscriptionID))
		if err != nil {
			return fmt.Errorf("vnet gateways: %w", err)
		}
		return nil
	})

	// ExpressRoute circuits
	g.Go(func() error {
		var err error
		erRaw, err = rg(kqlExpressRoutes(a.subscriptionID))
		if err != nil {
			return fmt.Errorf("expressroutes: %w", err)
		}
		return nil
	})

	// Azure Front Door
	g.Go(func() error {
		var err error
		afdRaw, err = rg(kqlFrontDoors(a.subscriptionID))
		if err != nil {
			return fmt.Errorf("front doors: %w", err)
		}
		return nil
	})

	_ = gctx // used by the errgroup
	if err := g.Wait(); err != nil {
		return rgResult{}, err
	}

	// Assemble NIC graph.NIC list and nicMeta list.
	nics, metas := parseNICs(nicRaw)

	// Build NW location map.
	nwLocs := buildNWLocationMap(nwRaw)

	// Detected firewall (at most one per subscription in the canonical model).
	var rawFW *rawFirewall
	if len(fwRaw) > 0 {
		rawFW = parseRawFirewall(fwRaw[0])
	}

	// Private DNS Zones — fetch VNet links for each zone.
	dnsZones, err := a.parsePrivateDNSZonesWithLinks(ctx, dnsRaw)
	if err != nil {
		// Non-fatal: log and continue with empty zones.
		dnsZones = nil
	}

	res := rgResult{
		ResourceGraph: graph.ResourceGraph{
			VirtualNetworks:        vnets,
			NetworkSecurityGroups:  nsgs,
			RouteTables:            rts,
			PublicIPAddresses:      pips,
			NetworkInterfaces:      nics,
			PrivateEndpoints:       parsePrivateEndpoints(peRaw),
			LoadBalancers:          parseLoadBalancers(lbRaw),
			PrivateDnsZones:        dnsZones,
			ApplicationGateways:    parseAppGateways(appgwRaw),
			AKSClusters:            parseAKS(aksRaw),
			NatGateways:            parseNATGateways(natgwRaw),
			VirtualWANs:            parseVirtualWANs(vwanRaw, vhubRaw),
			APIManagements:         parseAPIM(apimRaw),
			AzureBastions:          parseBastions(bastRaw),
			VirtualNetworkGateways: parseVNetGateways(vngRaw),
			ExpressRouteCircuits:   parseExpressRoutes(erRaw),
			AzureFrontDoors:        parseFrontDoors(afdRaw),
		},
		nicMetas:    metas,
		nwLocations: nwLocs,
		rawFW:       rawFW,
	}
	return res, nil
}

// ─── KQL runner ───────────────────────────────────────────────────────────────

// argQuerier is the subset of *armrg.Client.runKQL needs. Abstracted so the
// pagination loop can be unit-tested without a live Azure Resource Graph.
type argQuerier interface {
	Resources(ctx context.Context, query armrg.QueryRequest, options *armrg.ClientResourcesOptions) (armrg.ClientResourcesResponse, error)
}

// runKQL executes a Resource Graph query and returns ALL rows, following the
// SkipToken across pages. Resource Graph caps a single page at ~1000 rows;
// without this loop a subscription with >1000 resources of a type would be
// silently truncated and findings on the dropped resources never produced
// (audit C-1). Each page accumulates into `all`; the loop ends when the service
// returns no SkipToken.
func (a *adapter) runKQL(ctx context.Context, client argQuerier, kql string) ([]map[string]interface{}, error) {
	sub := a.subscriptionID
	var all []map[string]interface{}
	var skipToken *string
	for {
		resp, err := client.Resources(ctx, armrg.QueryRequest{
			Subscriptions: []*string{&sub},
			Query:         ptr(kql),
			Options: &armrg.QueryRequestOptions{
				ResultFormat: ptr(armrg.ResultFormatObjectArray),
				SkipToken:    skipToken,
			},
		}, nil)
		if err != nil {
			return nil, err
		}
		rows, terr := toRows(resp.Data)
		if terr != nil {
			return nil, terr
		}
		all = append(all, rows...)
		if resp.SkipToken == nil || *resp.SkipToken == "" {
			return all, nil
		}
		skipToken = resp.SkipToken
	}
}

// ─── KQL queries ──────────────────────────────────────────────────────────────

func kqlVNets(sub string) string {
	return fmt.Sprintf(`Resources
| where subscriptionId == %q
| where type == "microsoft.network/virtualnetworks"
| project
    name,
    resourceGroup,
    location,
    addressPrefixes = properties.addressSpace.addressPrefixes,
    subnets = properties.subnets,
    peerings = properties.virtualNetworkPeerings`, sub)
}

func kqlNSGs(sub string) string {
	return fmt.Sprintf(`Resources
| where subscriptionId == %q
| where type == "microsoft.network/networksecuritygroups"
| project
    name,
    resourceGroup,
    location,
    securityRules = properties.securityRules,
    associatedSubnets = properties.subnets`, sub)
}

func kqlRouteTables(sub string) string {
	return fmt.Sprintf(`Resources
| where subscriptionId == %q
| where type == "microsoft.network/routetables"
| project
    name,
    resourceGroup,
    location,
    routes = properties.routes,
    associatedSubnets = properties.subnets,
    disableBgpRoutePropagation = properties.disableBgpRoutePropagation`, sub)
}

func kqlPublicIPs(sub string) string {
	return fmt.Sprintf(`Resources
| where subscriptionId == %q
| where type == "microsoft.network/publicipaddresses"
| project
    name,
    id,
    resourceGroup,
    location,
    ipAddress = properties.ipAddress,
    ipConfiguration = properties.ipConfiguration.id,
    allocationMethod = properties.publicIPAllocationMethod,
    sku = sku.name`, sub)
}

func kqlNICs(sub string) string {
	return fmt.Sprintf(`Resources
| where subscriptionId == %q
| where type == "microsoft.network/networkinterfaces"
| project
    name,
    id,
    resourceGroup,
    location,
    ipConfigurations = properties.ipConfigurations,
    nsgId = properties.networkSecurityGroup.id,
    tags,
    dnsSettings = properties.dnsSettings`, sub)
}

func kqlNetworkWatchers(sub string) string {
	return fmt.Sprintf(`Resources
| where subscriptionId == %q
| where type == "microsoft.network/networkwatchers"
| project name, resourceGroup, location`, sub)
}

func kqlFirewalls(sub string) string {
	return fmt.Sprintf(`Resources
| where subscriptionId == %q
| where type == "microsoft.network/azurefirewalls"
| project
    name,
    resourceGroup,
    location,
    privateIp = properties.ipConfigurations[0].properties.privateIPAddress,
    publicIpId = properties.ipConfigurations[0].properties.publicIPAddress.id,
    firewallPolicyId = properties.firewallPolicy.id,
    sku = properties.sku.tier`, sub)
}

func kqlPrivateEndpoints(sub string) string {
	return fmt.Sprintf(`Resources
| where subscriptionId == %q
| where type == "microsoft.network/privateendpoints"
| project
    name, resourceGroup,
    subnetId = properties.subnet.id,
    privateIp = properties.networkInterfaces[0].properties.ipConfigurations[0].properties.privateIPAddress,
    groupId = properties.privateLinkServiceConnections[0].properties.groupIds[0],
    privateLinkServiceId = properties.privateLinkServiceConnections[0].properties.privateLinkServiceId,
    connectionState = properties.privateLinkServiceConnections[0].properties.privateLinkServiceConnectionState.status`, sub)
}

func kqlLoadBalancers(sub string) string {
	return fmt.Sprintf(`Resources
| where subscriptionId == %q
| where type == "microsoft.network/loadbalancers"
| project
    name, resourceGroup, sku = sku.name,
    frontendIP = properties.frontendIPConfigurations[0].properties.privateIPAddress,
    publicIPRef = properties.frontendIPConfigurations[0].properties.publicIPAddress.id,
    isInternal = isnull(properties.frontendIPConfigurations[0].properties.publicIPAddress.id),
    inboundNatRules = properties.inboundNatRules,
    backendPools = properties.backendAddressPools`, sub)
}

// kqlPrivateDNSZones does not filter by subscriptionId (matches spec — see note).
func kqlPrivateDNSZones() string {
	return `resources
| where type == "microsoft.network/privatednszones"
| project name, id, resourceGroup, subscriptionId`
}

func kqlAppGateways() string {
	return `resources
| where type == "microsoft.network/applicationgateways"
| extend wafc = properties.webApplicationFirewallConfiguration
| extend sku  = properties.sku
| project
    name, resourceGroup,
    gatewaySubnetRef   = tostring(properties.gatewayIPConfigurations[0].properties.subnet.id),
    wafEnabled         = tobool(coalesce(wafc.enabled, todynamic('false'))),
    wafMode            = tostring(wafc.firewallMode),
    skuTier            = tostring(sku.tier),
    frontendIPs        = properties.frontendIPConfigurations,
    backendPools       = properties.backendAddressPools,
    firewallPolicy     = tostring(properties.firewallPolicy.id)`
}

func kqlAKS(sub string) string {
	return fmt.Sprintf(`Resources
| where subscriptionId == %q
| where type == "microsoft.containerservice/managedclusters"
| project
    name, resourceGroup,
    subnetId = properties.agentPoolProfiles[0].vnetSubnetID,
    podCidr = properties.networkProfile.podCidr,
    serviceCidr = properties.networkProfile.serviceCidr,
    isPrivate = tobool(coalesce(properties.apiServerAccessProfile.enablePrivateCluster, todynamic('false'))),
    apiServerIp = properties.apiServerAccessProfile.privateEndpointDNSZone`, sub)
}

func kqlNATGateways(sub string) string {
	return fmt.Sprintf(`Resources
| where subscriptionId == %q
| where type == "microsoft.network/natgateways"
| project
    name, resourceGroup,
    publicIps = properties.publicIpAddresses,
    subnets = properties.subnets`, sub)
}

func kqlVirtualWANs(sub string) string {
	return fmt.Sprintf(`Resources
| where subscriptionId == %q
| where type == "microsoft.network/virtualwans"
| project name, resourceGroup, sku = properties.type`, sub)
}

func kqlVirtualHubs(sub string) string {
	return fmt.Sprintf(`Resources
| where subscriptionId == %q
| where type == "microsoft.network/virtualhubs"
| project
    name, resourceGroup,
    addressPrefix = properties.addressPrefix,
    location,
    spokeConnections = properties.virtualNetworkConnections,
    hasSecuredFw = isnull(properties.azureFirewall.id) == false,
    firewallPrivateIp = properties.azureFirewall.id,
    routingPolicies = properties.routingPolicies,
    routingPolicyInternet = isnotnull(properties.routingPolicies) and array_length(properties.routingPolicies) > 0,
    virtualWanId = properties.virtualWan.id`, sub)
}

func kqlAPIM(sub string) string {
	return fmt.Sprintf(`Resources
| where subscriptionId == %q
| where type == "microsoft.apimanagement/service"
| project
    name, resourceGroup,
    subnetId = properties.virtualNetworkConfiguration.subnetResourceId,
    publicIp = properties.publicIPAddresses[0],
    vnetMode = properties.virtualNetworkType,
    gatewayUrl = properties.gatewayUrl,
    sku = sku.name`, sub)
}

func kqlBastions(sub string) string {
	return fmt.Sprintf(`Resources
| where subscriptionId == %q
| where type == "microsoft.network/bastionhosts"
| project
    name, resourceGroup,
    subnetId = properties.ipConfigurations[0].properties.subnet.id,
    publicIp = properties.ipConfigurations[0].properties.publicIPAddress.id,
    sku = sku.name`, sub)
}

func kqlVNetGateways(sub string) string {
	return fmt.Sprintf(`Resources
| where subscriptionId == %q
| where type == "microsoft.network/virtualnetworkgateways"
| project
    name, resourceGroup,
    subnetId = properties.ipConfigurations[0].properties.subnet.id,
    gatewayType = properties.gatewayType,
    sku = properties.sku.name,
    publicIp = properties.ipConfigurations[0].properties.publicIPAddress.id,
    enableBgp = tobool(properties.enableBgp),
    enableForcedTunneling = tobool(coalesce(properties.enableForcedTunneling, todynamic('false')))`, sub)
}

func kqlExpressRoutes(sub string) string {
	return fmt.Sprintf(`Resources
| where subscriptionId == %q
| where type == "microsoft.network/expressroutecircuits"
| project
    name, resourceGroup,
    peeringLocation = properties.serviceProviderProperties.peeringLocation,
    bandwidthMbps = properties.serviceProviderProperties.bandwidthInMbps,
    connectedVnet = ""`, sub)
}

func kqlFrontDoors(sub string) string {
	return fmt.Sprintf(`Resources
| where subscriptionId == %q
| where type == "microsoft.cdn/profiles"
| where sku.name in ("Standard_AzureFrontDoor", "Premium_AzureFrontDoor")
| project
    name, resourceGroup,
    sku = sku.name,
    wafEnabled = isnotnull(properties.frontDoorId)`, sub)
}

// ─── Parse helpers ────────────────────────────────────────────────────────────

func parseVNets(rows []map[string]interface{}) []graph.VNet {
	var out []graph.VNet
	for _, row := range rows {
		vnet := graph.VNet{
			Name:         getString(row, "name"),
			AddressSpace: getStringSlice(row, "addressPrefixes"),
			Subnets:      parseSubnets(getSlice(row, "subnets")),
			Peerings:     parsePeerings(getSlice(row, "peerings")),
		}
		out = append(out, vnet)
	}
	return out
}

func parseSubnets(arr []interface{}) []graph.Subnet {
	var out []graph.Subnet
	for _, item := range arr {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		props := getMap(m, "properties")
		if props == nil {
			props = m
		}
		sub := graph.Subnet{
			Name:          getString(m, "name"),
			AddressPrefix: getString(props, "addressPrefix"),
		}
		if nsg := getMap(props, "networkSecurityGroup"); nsg != nil {
			sub.NetworkSecurityGroup = extractResourceName(getString(nsg, "id"))
		}
		if rt := getMap(props, "routeTable"); rt != nil {
			sub.RouteTable = extractResourceName(getString(rt, "id"))
		}
		// Service endpoints
		for _, se := range getSlice(props, "serviceEndpoints") {
			if sm, ok := se.(map[string]interface{}); ok {
				if svc := getString(sm, "service"); svc != "" {
					sub.ServiceEndpoints = append(sub.ServiceEndpoints, svc)
				}
			}
		}
		// Delegations
		for _, d := range getSlice(props, "delegations") {
			if dm, ok := d.(map[string]interface{}); ok {
				dp := getMap(dm, "properties")
				if dp == nil {
					dp = dm
				}
				if svcName := getString(dp, "serviceName"); svcName != "" {
					sub.Delegations = append(sub.Delegations, svcName)
				}
			}
		}
		sub.PrivateEndpointNetworkPolicies = getString(props, "privateEndpointNetworkPolicies")
		out = append(out, sub)
	}
	return out
}

func parsePeerings(arr []interface{}) []graph.Peering {
	var out []graph.Peering
	for _, item := range arr {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		props := getMap(m, "properties")
		if props == nil {
			props = m
		}
		remoteVnetMap := getMap(props, "remoteVirtualNetwork")
		remoteVnetID := ""
		if remoteVnetMap != nil {
			remoteVnetID = getString(remoteVnetMap, "id")
		}
		remoteVnet := extractResourceName(remoteVnetID)
		remoteSub := extractSubscriptionID(remoteVnetID)
		// Only set RemoteSubscriptionID if it differs from the local subscription.
		p := graph.Peering{
			RemoteVnet:            remoteVnet,
			State:                 getString(props, "peeringState"),
			AllowForwardedTraffic: getBool(props, "allowForwardedTraffic"),
			AllowGatewayTransit:   getBool(props, "allowGatewayTransit"),
			UseRemoteGateways:     getBool(props, "useRemoteGateways"),
		}
		if remoteSub != "" {
			p.RemoteSubscriptionID = remoteSub
		}
		out = append(out, p)
	}
	return out
}

func parseNSGs(rows []map[string]interface{}) []graph.NSG {
	var out []graph.NSG
	for _, row := range rows {
		nsg := graph.NSG{
			Name:          getString(row, "name"),
			SecurityRules: parseNSGRules(getSlice(row, "securityRules")),
		}
		for _, item := range getSlice(row, "associatedSubnets") {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			subnetID := getString(m, "id")
			if s := extractSubnet(subnetID); s != "" {
				nsg.AssociatedSubnets = append(nsg.AssociatedSubnets, s)
			}
		}
		out = append(out, nsg)
	}
	return out
}

func parseNSGRules(arr []interface{}) []graph.SecRule {
	var out []graph.SecRule
	for _, item := range arr {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		props := getMap(m, "properties")
		if props == nil {
			props = m
		}
		src := getString(props, "sourceAddressPrefix")
		r := graph.SecRule{
			Name:                 getString(m, "name"),
			Priority:             getInt(props, "priority"),
			Direction:            getString(props, "direction"),
			Access:               getString(props, "access"),
			Protocol:             getString(props, "protocol"),
			SourceAddressPrefix:  src,
			DestinationPortRange: getString(props, "destinationPortRange"),
			Source:               src, // invariant: Source = SourceAddressPrefix
		}
		out = append(out, r)
	}
	return out
}

func parseRouteTables(rows []map[string]interface{}) []graph.RouteTable {
	var out []graph.RouteTable
	for _, row := range rows {
		rt := graph.RouteTable{
			Name:   getString(row, "name"),
			Routes: parseRoutes(getSlice(row, "routes")),
		}
		for _, item := range getSlice(row, "associatedSubnets") {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			subnetID := getString(m, "id")
			if s := extractSubnet(subnetID); s != "" {
				rt.AssociatedSubnets = append(rt.AssociatedSubnets, s)
			}
		}
		out = append(out, rt)
	}
	return out
}

func parseRoutes(arr []interface{}) []graph.Route {
	var out []graph.Route
	for _, item := range arr {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		props := getMap(m, "properties")
		if props == nil {
			props = m
		}
		r := graph.Route{
			Name:             getString(m, "name"),
			AddressPrefix:    getString(props, "addressPrefix"),
			NextHopType:      getString(props, "nextHopType"),
			NextHopIPAddress: getString(props, "nextHopIpAddress"),
		}
		out = append(out, r)
	}
	return out
}

func parsePublicIPs(rows []map[string]interface{}) []graph.PublicIP {
	var out []graph.PublicIP
	for _, row := range rows {
		ipCfg := getString(row, "ipConfiguration")
		pip := graph.PublicIP{
			Name:      getString(row, "name"),
			ID:        getString(row, "id"),
			IPAddress: getString(row, "ipAddress"),
		}
		if ipCfg != "" {
			// Store just the resource name portion for the ipConfiguration reference.
			// The engine checks: pip.IPConfiguration == nil => orphaned.
			pip.IPConfiguration = ptr(ipCfg)
		}
		out = append(out, pip)
	}
	return out
}

// parseNICs returns both the graph.NIC slice and the adapter-internal nicMeta slice.
func parseNICs(rows []map[string]interface{}) ([]graph.NIC, []nicMeta) {
	var nics []graph.NIC
	var metas []nicMeta

	for _, row := range rows {
		name := getString(row, "name")
		rg := getString(row, "resourceGroup")
		loc := strings.ToLower(getString(row, "location"))

		ipConfigs := getSlice(row, "ipConfigurations")
		var subnet, privateIP, publicIPName string
		if len(ipConfigs) > 0 {
			ipcfg, ok := ipConfigs[0].(map[string]interface{})
			if ok {
				props := getMap(ipcfg, "properties")
				if props == nil {
					props = ipcfg
				}
				privateIP = getString(props, "privateIPAddress")
				if subnetMap := getMap(props, "subnet"); subnetMap != nil {
					subnet = extractSubnet(getString(subnetMap, "id"))
				}
				if pipMap := getMap(props, "publicIPAddress"); pipMap != nil {
					publicIPName = extractResourceName(getString(pipMap, "id"))
				}
			}
		}

		nsgName := extractResourceName(getString(row, "nsgId"))

		// DNS servers
		var dnsServers []string
		if dnsSett := getMap(row, "dnsSettings"); dnsSett != nil {
			dnsServers = getStringSlice(dnsSett, "dnsServers")
		}

		// Tags
		tags := map[string]string{}
		if tv, ok := row["tags"]; ok && tv != nil {
			if tm, ok := tv.(map[string]interface{}); ok {
				for k, v := range tm {
					if vs, ok := v.(string); ok {
						tags[k] = vs
					}
				}
			}
		}

		nic := graph.NIC{
			Name:       name,
			ID:         getString(row, "id"),
			Subnet:     subnet,
			PrivateIP:  privateIP,
			Tags:       tags,
			DNSServers: dnsServers,
		}
		if nsgName != "" {
			nic.NetworkSecurityGroup = ptr(nsgName)
		}
		if publicIPName != "" {
			nic.PublicIP = ptr(publicIPName)
		}

		nics = append(nics, nic)
		metas = append(metas, nicMeta{
			nic:           nic,
			resourceGroup: rg,
			location:      loc,
		})
	}
	return nics, metas
}

func buildNWLocationMap(rows []map[string]interface{}) map[string]nwLocation {
	m := make(map[string]nwLocation)
	for _, row := range rows {
		loc := strings.ToLower(getString(row, "location"))
		m[loc] = nwLocation{
			name:          getString(row, "name"),
			resourceGroup: getString(row, "resourceGroup"),
		}
	}
	return m
}

func parseRawFirewall(row map[string]interface{}) *rawFirewall {
	if row == nil {
		return nil
	}
	return &rawFirewall{
		name:             getString(row, "name"),
		resourceGroup:    getString(row, "resourceGroup"),
		privateIP:        getString(row, "privateIp"),
		publicIPID:       getString(row, "publicIpId"),
		firewallPolicyID: getString(row, "firewallPolicyId"),
		sku:              getString(row, "sku"),
	}
}

func parsePrivateEndpoints(rows []map[string]interface{}) []graph.PrivateEndpoint {
	var out []graph.PrivateEndpoint
	for _, row := range rows {
		out = append(out, graph.PrivateEndpoint{
			Name:                 getString(row, "name"),
			Subnet:               extractSubnet(getString(row, "subnetId")),
			PrivateIP:            getString(row, "privateIp"),
			GroupId:              getString(row, "groupId"),
			PrivateLinkServiceId: getString(row, "privateLinkServiceId"),
			ConnectionState:      getString(row, "connectionState"),
		})
	}
	return out
}

func parseLoadBalancers(rows []map[string]interface{}) []graph.LoadBalancer {
	var out []graph.LoadBalancer
	for _, row := range rows {
		isInternal := getBool(row, "isInternal")
		frontendIP := getString(row, "frontendIP")
		// For ELB, frontendIP comes from the public IP resource; resolve its actual IP
		// via the PublicIPAddresses collection at assembly time. For now store the name.
		if !isInternal {
			pubIPID := getString(row, "publicIPRef")
			if pubIPID != "" {
				frontendIP = extractResourceName(pubIPID) // store PIP name; resolved in assembly
			}
		}
		lb := graph.LoadBalancer{
			Name:       getString(row, "name"),
			Sku:        getString(row, "sku"),
			FrontendIP: frontendIP,
			IsInternal: isInternal,
		}
		lb.InboundNatRules = parseLBNatRules(getSlice(row, "inboundNatRules"))
		lb.BackendPools = parseLBBackendPools(getSlice(row, "backendPools"))
		out = append(out, lb)
	}
	return out
}

func parseLBNatRules(arr []interface{}) []graph.LBNatRule {
	var out []graph.LBNatRule
	for _, item := range arr {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		props := getMap(m, "properties")
		if props == nil {
			props = m
		}
		backendNicID := getString(props, "backendIPConfiguration")
		// backendIPConfiguration is ".../networkInterfaces/{nicName}/ipConfigurations/..."
		// Extract NIC name: segment before "/ipconfigurations/"
		backendNic := extractNICFromIPConfig(backendNicID)
		r := graph.LBNatRule{
			Name:         getString(m, "name"),
			Protocol:     getString(props, "protocol"),
			FrontendPort: getInt(props, "frontendPort"),
			BackendPort:  getInt(props, "backendPort"),
			BackendNic:   backendNic,
		}
		out = append(out, r)
	}
	return out
}

func extractNICFromIPConfig(armID string) string {
	// ".../networkInterfaces/{nicName}/ipConfigurations/..."
	lower := strings.ToLower(armID)
	idx := strings.Index(lower, "/networkinterfaces/")
	if idx < 0 {
		return ""
	}
	rest := armID[idx+len("/networkinterfaces/"):]
	if i := strings.Index(rest, "/"); i >= 0 {
		return rest[:i]
	}
	return rest
}

func parseLBBackendPools(arr []interface{}) []graph.LBBackendPool {
	var out []graph.LBBackendPool
	for _, item := range arr {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		props := getMap(m, "properties")
		if props == nil {
			props = m
		}
		pool := graph.LBBackendPool{Name: getString(m, "name")}
		for _, be := range getSlice(props, "backendIPConfigurations") {
			if bm, ok := be.(map[string]interface{}); ok {
				if nicName := extractNICFromIPConfig(getString(bm, "id")); nicName != "" {
					pool.NicRefs = append(pool.NicRefs, nicName)
				}
			}
		}
		out = append(out, pool)
	}
	return out
}

// parsePrivateDNSZonesWithLinks fetches VNet links for each Private DNS Zone.
func (a *adapter) parsePrivateDNSZonesWithLinks(ctx context.Context, rows []map[string]interface{}) ([]graph.PrivateDnsZone, error) {
	var out []graph.PrivateDnsZone
	for _, row := range rows {
		zoneName := getString(row, "name")
		zoneRG := getString(row, "resourceGroup")
		zoneSub := getString(row, "subscriptionId")
		if zoneName == "" || zoneRG == "" {
			continue
		}
		zone := graph.PrivateDnsZone{Name: zoneName}

		// Fetch VNet links for this zone.
		linksURL := armBase + fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/privateDnsZones/%s/virtualNetworkLinks",
			zoneSub, zoneRG, zoneName) + "?api-version=2020-06-01"
		items, err := a.listAll(ctx, linksURL)
		if err == nil {
			for _, raw := range items {
				var link map[string]interface{}
				if json.Unmarshal(raw, &link) != nil {
					continue
				}
				props := getMap(link, "properties")
				if props == nil {
					continue
				}
				if vnMap := getMap(props, "virtualNetwork"); vnMap != nil {
					vnetID := getString(vnMap, "id")
					if vnetName := extractResourceName(vnetID); vnetName != "" {
						zone.LinkedVnets = append(zone.LinkedVnets, vnetName)
					}
				}
			}
		}
		out = append(out, zone)
	}
	return out, nil
}

func parseAppGateways(rows []map[string]interface{}) []graph.ApplicationGateway {
	var out []graph.ApplicationGateway
	for _, row := range rows {
		skuTier := getString(row, "skuTier")
		wafEnabled := getBool(row, "wafEnabled")
		// WAF_v2 tier with policy-based WAF: set WafEnabled=true even if inline config absent.
		if strings.EqualFold(skuTier, "WAF_v2") {
			wafEnabled = true
		}
		wafMode := getString(row, "wafMode")

		gw := graph.ApplicationGateway{
			Name:       getString(row, "name"),
			Subnet:     extractSubnet(getString(row, "gatewaySubnetRef")),
			WafEnabled: wafEnabled,
			WafMode:    wafMode,
		}

		// Resolve PublicIP from frontendIPConfigurations
		for _, fe := range getSlice(row, "frontendIPs") {
			if fm, ok := fe.(map[string]interface{}); ok {
				fProps := getMap(fm, "properties")
				if fProps == nil {
					fProps = fm
				}
				if pipMap := getMap(fProps, "publicIPAddress"); pipMap != nil {
					gw.PublicIP = extractResourceName(getString(pipMap, "id"))
				}
			}
		}

		// Backend pools
		for _, bp := range getSlice(row, "backendPools") {
			if bm, ok := bp.(map[string]interface{}); ok {
				pool := graph.AppGWBackendPool{Name: getString(bm, "name")}
				bProps := getMap(bm, "properties")
				if bProps == nil {
					bProps = bm
				}
				for _, addr := range getSlice(bProps, "backendAddresses") {
					if am, ok := addr.(map[string]interface{}); ok {
						if ip := getString(am, "ipAddress"); ip != "" {
							pool.Targets = append(pool.Targets, ip)
						} else if fqdn := getString(am, "fqdn"); fqdn != "" {
							pool.Targets = append(pool.Targets, fqdn)
						}
					}
				}
				gw.BackendPools = append(gw.BackendPools, pool)
			}
		}
		out = append(out, gw)
	}
	return out
}

func parseAKS(rows []map[string]interface{}) []graph.AKSCluster {
	var out []graph.AKSCluster
	for _, row := range rows {
		out = append(out, graph.AKSCluster{
			Name:             getString(row, "name"),
			Subnet:           extractSubnet(getString(row, "subnetId")),
			PodCidr:          getString(row, "podCidr"),
			ServiceCidr:      getString(row, "serviceCidr"),
			IsPrivateCluster: getBool(row, "isPrivate"),
			ApiServerIP:      getString(row, "apiServerIp"),
		})
	}
	return out
}

func parseNATGateways(rows []map[string]interface{}) []graph.NatGateway {
	var out []graph.NatGateway
	for _, row := range rows {
		natgw := graph.NatGateway{Name: getString(row, "name")}
		for _, item := range getSlice(row, "publicIps") {
			if m, ok := item.(map[string]interface{}); ok {
				if id := getString(m, "id"); id != "" {
					natgw.PublicIPs = append(natgw.PublicIPs, extractResourceName(id))
				}
			}
		}
		for _, item := range getSlice(row, "subnets") {
			if m, ok := item.(map[string]interface{}); ok {
				if id := getString(m, "id"); id != "" {
					if s := extractSubnet(id); s != "" {
						natgw.AssociatedSubnets = append(natgw.AssociatedSubnets, s)
					}
				}
			}
		}
		out = append(out, natgw)
	}
	return out
}

// parseVirtualWANs assembles VirtualWAN structs by cross-referencing vHub rows.
func parseVirtualWANs(wanRows, hubRows []map[string]interface{}) []graph.VirtualWAN {
	// Index hubs by virtualWanId (ARM resource ID)
	hubsByWanID := map[string][]graph.VirtualHub{}
	for _, row := range hubRows {
		wanID := getString(row, "virtualWanId")
		hub := parseVirtualHub(row)
		hubsByWanID[wanID] = append(hubsByWanID[wanID], hub)
	}

	var out []graph.VirtualWAN
	for _, row := range wanRows {
		wanName := getString(row, "name")
		// We need the WAN's ARM ID to match hubs. Build it from name+rg but
		// we don't have subscriptionID here — match by wanName as last segment.
		var matchedHubs []graph.VirtualHub
		for wanID, hubs := range hubsByWanID {
			if extractResourceName(wanID) == wanName {
				matchedHubs = append(matchedHubs, hubs...)
			}
		}
		out = append(out, graph.VirtualWAN{
			Name:  wanName,
			SKU:   getString(row, "sku"),
			VHubs: matchedHubs,
		})
	}
	return out
}

func parseVirtualHub(row map[string]interface{}) graph.VirtualHub {
	hub := graph.VirtualHub{
		Name:               getString(row, "name"),
		AddressPrefix:      getString(row, "addressPrefix"),
		Location:           getString(row, "location"),
		HasSecuredFirewall: getBool(row, "hasSecuredFw"),
	}

	// Resolve firewallPrivateIp: it's currently the ARM ID of the firewall.
	// The engine doesn't use FirewallPrivateIP directly; leave empty.
	// hub.FirewallPrivateIP = ...

	// Spoke connections: virtualNetworkConnections array
	for _, item := range getSlice(row, "spokeConnections") {
		if m, ok := item.(map[string]interface{}); ok {
			props := getMap(m, "properties")
			if props == nil {
				props = m
			}
			remoteVNet := getMap(props, "remoteVirtualNetwork")
			if remoteVNet == nil {
				// Sometimes the ID is directly on the connection
				remoteVNet = m
			}
			if id := getString(remoteVNet, "id"); id != "" {
				if vnetName := extractResourceName(id); vnetName != "" {
					hub.SpokeConnections = append(hub.SpokeConnections, vnetName)
				}
			}
		}
	}

	// Routing policies: determine RoutingPolicyInternet and RoutingPolicyPrivate
	policies := getSlice(row, "routingPolicies")
	for _, p := range policies {
		if pm, ok := p.(map[string]interface{}); ok {
			name := strings.ToLower(getString(pm, "name"))
			if strings.Contains(name, "internet") {
				hub.RoutingPolicyInternet = true
			}
			if strings.Contains(name, "private") {
				hub.RoutingPolicyPrivate = true
			}
		}
	}
	// Also check the simple boolean from KQL
	if getBool(row, "routingPolicyInternet") && len(policies) == 0 {
		hub.RoutingPolicyInternet = true
	}

	return hub
}

func parseAPIM(rows []map[string]interface{}) []graph.APIManagement {
	var out []graph.APIManagement
	for _, row := range rows {
		vnetMode := getString(row, "vnetMode")
		if vnetMode == "" {
			vnetMode = "None"
		}
		out = append(out, graph.APIManagement{
			Name:       getString(row, "name"),
			Subnet:     extractSubnet(getString(row, "subnetId")),
			PublicIP:   getString(row, "publicIp"),
			VNetMode:   vnetMode,
			GatewayURL: getString(row, "gatewayUrl"),
			SkuName:    getString(row, "sku"),
		})
	}
	return out
}

func parseBastions(rows []map[string]interface{}) []graph.AzureBastion {
	var out []graph.AzureBastion
	for _, row := range rows {
		subnetRef := getString(row, "subnetId")
		// Bastion subnet name should be "AzureBastionSubnet"
		subnetShort := extractSubnet(subnetRef)
		out = append(out, graph.AzureBastion{
			Name:     getString(row, "name"),
			Subnet:   subnetShort,
			PublicIP: extractResourceName(getString(row, "publicIp")),
			SKU:      getString(row, "sku"),
		})
	}
	return out
}

func parseVNetGateways(rows []map[string]interface{}) []graph.VirtualNetworkGateway {
	var out []graph.VirtualNetworkGateway
	for _, row := range rows {
		out = append(out, graph.VirtualNetworkGateway{
			Name:                  getString(row, "name"),
			Subnet:                extractSubnet(getString(row, "subnetId")),
			GatewayType:           getString(row, "gatewayType"),
			SKU:                   getString(row, "sku"),
			PublicIP:              extractResourceName(getString(row, "publicIp")),
			EnableBGP:             getBool(row, "enableBgp"),
			EnableForcedTunneling: getBool(row, "enableForcedTunneling"),
		})
	}
	return out
}

func parseExpressRoutes(rows []map[string]interface{}) []graph.ExpressRouteCircuit {
	var out []graph.ExpressRouteCircuit
	for _, row := range rows {
		out = append(out, graph.ExpressRouteCircuit{
			Name:            getString(row, "name"),
			PeeringLocation: getString(row, "peeringLocation"),
			BandwidthMbps:   getInt(row, "bandwidthMbps"),
			ConnectedVnet:   getString(row, "connectedVnet"),
		})
	}
	return out
}

func parseFrontDoors(rows []map[string]interface{}) []graph.AzureFrontDoor {
	var out []graph.AzureFrontDoor
	for _, row := range rows {
		out = append(out, graph.AzureFrontDoor{
			Name:       getString(row, "name"),
			SKU:        getString(row, "sku"),
			WafEnabled: getBool(row, "wafEnabled"),
		})
	}
	return out
}

// ─── ARM ID utilities ─────────────────────────────────────────────────────────

// extractSubscriptionID extracts the subscription GUID from an ARM resource ID.
func extractSubscriptionID(armID string) string {
	lower := strings.ToLower(armID)
	idx := strings.Index(lower, "/subscriptions/")
	if idx < 0 {
		return ""
	}
	rest := armID[idx+len("/subscriptions/"):]
	if i := strings.Index(rest, "/"); i >= 0 {
		return rest[:i]
	}
	return rest
}
