// Package graph is the topology model and the Azure fixture parser. The model is
// Azure-shaped (NSG, AVNM, Azure Firewall, effective rules/routes, Private DNS,
// Application Gateway, AKS, NAT Gateway, Private Link Service, Express Route).
// Making it genuinely cloud-neutral — so the AWS adapter is a second adapter rather
// than a rewrite — is deferred until that adapter lands.
// It mirrors the reference implementation (reference/analyze.py).
package graph

import (
	"encoding/json"
	"os"
)

type Fixture struct {
	Subscription              string             `json:"subscription"`
	ResourceGraph             ResourceGraph      `json:"resourceGraph"`
	NetworkWatcher            NetworkWatcher     `json:"networkWatcher"`
	AVNM                      AVNM               `json:"avnm"`
	AzureFirewall             *Firewall          `json:"azureFirewall,omitempty"`
	CrossSubscriptionPeerings []CrossSubPeering  `json:"crossSubscriptionPeerings,omitempty"`
	// Enrichment holds optional P1 data from Defender for Cloud, Azure Policy,
	// and Activity Logs. Populated by the adapter only when the caller requests
	// enriched analysis. The engine does not read Enrichment; the MCP explainer
	// layer surfaces it as additional context alongside engine findings.
	Enrichment *Enrichment `json:"enrichment,omitempty"`
}

type ResourceGraph struct {
	VirtualNetworks        []VNet                `json:"virtualNetworks"`
	NetworkSecurityGroups  []NSG                 `json:"networkSecurityGroups"`
	RouteTables            []RouteTable          `json:"routeTables"`
	PublicIPAddresses      []PublicIP            `json:"publicIPAddresses"`
	NetworkInterfaces      []NIC                 `json:"networkInterfaces"`
	PrivateEndpoints       []PrivateEndpoint     `json:"privateEndpoints,omitempty"`
	LoadBalancers          []LoadBalancer        `json:"loadBalancers,omitempty"`
	PrivateDnsZones        []PrivateDnsZone      `json:"privateDnsZones,omitempty"`
	ApplicationGateways    []ApplicationGateway  `json:"applicationGateways,omitempty"`
	AKSClusters            []AKSCluster          `json:"aksClusters,omitempty"`
	NatGateways            []NatGateway          `json:"natGateways,omitempty"`
	PrivateLinkServices    []PrivateLinkService  `json:"privateLinkServices,omitempty"`
	ExpressRouteCircuits   []ExpressRouteCircuit `json:"expressRouteCircuits,omitempty"`
	APIManagements         []APIManagement       `json:"apiManagements,omitempty"`
	AzureBastions          []AzureBastion        `json:"azureBastions,omitempty"`
	VirtualNetworkGateways []VirtualNetworkGateway `json:"virtualNetworkGateways,omitempty"`
	// Virtual WAN — P0 per Microsoft docs. Structurally different from
	// traditional hub-spoke: spokes connect to vHubs (not via VNet peerings).
	// Absent from a subscription = traditional hub-spoke; present = vWAN topology.
	VirtualWANs            []VirtualWAN          `json:"virtualWans,omitempty"`
	// Phase 2: collected but no analysis rule yet
	DNSPrivateResolvers  []DNSPrivateResolver  `json:"dnsPrivateResolvers,omitempty"`
	AzureRouteServers    []AzureRouteServer    `json:"azureRouteServers,omitempty"`
	AzureFrontDoors      []AzureFrontDoor      `json:"azureFrontDoors,omitempty"`
	DDoSProtectionPlans  []DDoSProtectionPlan  `json:"ddosProtectionPlans,omitempty"`
	LocalNetworkGateways []LocalNetworkGateway `json:"localNetworkGateways,omitempty"`
}

type VNet struct {
	Name         string    `json:"name"`
	AddressSpace []string  `json:"addressSpace"`
	Subnets      []Subnet  `json:"subnets"`
	Peerings     []Peering `json:"peerings"`
}

type Subnet struct {
	Name                           string   `json:"name"`
	AddressPrefix                  string   `json:"addressPrefix"`
	NetworkSecurityGroup           string   `json:"networkSecurityGroup"`
	RouteTable                     string   `json:"routeTable"`
	ServiceEndpoints               []string `json:"serviceEndpoints,omitempty"`               // e.g. ["Microsoft.Storage","Microsoft.KeyVault"]
	Delegations                    []string `json:"delegations,omitempty"`                    // e.g. ["Microsoft.Sql/managedInstances"]
	PrivateEndpointNetworkPolicies string   `json:"privateEndpointNetworkPolicies,omitempty"` // "Enabled"|"Disabled"|"NetworkSecurityGroupEnabled"|"RouteTableEnabled"
}

type Peering struct {
	RemoteVnet            string `json:"remoteVnet"`
	RemoteSubscriptionID  string `json:"remoteSubscriptionId,omitempty"` // non-empty = cross-subscription peering
	State                 string `json:"state"`
	AllowForwardedTraffic bool   `json:"allowForwardedTraffic"`
	AllowGatewayTransit   bool   `json:"allowGatewayTransit"`
	UseRemoteGateways     bool   `json:"useRemoteGateways"`
	// RemoteVnetRegion and IsGlobalPeering are populated in Phase 2 (TMR-006).
	// IsGlobalPeering = true triggers cross-region peering egress cost in forecast_cost.
	RemoteVnetRegion string `json:"remoteVnetRegion,omitempty"`
	IsGlobalPeering  bool   `json:"isGlobalPeering,omitempty"`
}

type NSG struct {
	Name              string    `json:"name"`
	SecurityRules     []SecRule `json:"securityRules"`
	AssociatedSubnets []string  `json:"associatedSubnets"`
}

type SecRule struct {
	Name                 string `json:"name"`
	Priority             int    `json:"priority"`
	Direction            string `json:"direction"`
	Access               string `json:"access"`
	Protocol             string `json:"protocol"`
	SourceAddressPrefix  string `json:"sourceAddressPrefix"`
	DestinationPortRange string `json:"destinationPortRange"`
	Source               string `json:"source"`
}

type RouteTable struct {
	Name              string   `json:"name"`
	Routes            []Route  `json:"routes"`
	AssociatedSubnets []string `json:"associatedSubnets"`
	// DisableBgpRoutePropagation is populated in Phase 2 (TMR-007).
	// When true, BGP-learned routes are not propagated to subnets using this RT.
	DisableBgpRoutePropagation bool `json:"disableBgpRoutePropagation,omitempty"`
}

type Route struct {
	Name             string `json:"name"`
	AddressPrefix    string `json:"addressPrefix"`
	NextHopType      string `json:"nextHopType"`
	NextHopIPAddress string `json:"nextHopIpAddress"`
}

type PublicIP struct {
	Name            string  `json:"name"`
	IPAddress       string  `json:"ipAddress"`
	IPConfiguration *string `json:"ipConfiguration"` // null => orphaned
	// AllocationMethod and SKU are populated in Phase 2 (TMR-005).
	// Required for fixed-cost PIP pricing in forecast_cost.
	AllocationMethod string `json:"allocationMethod,omitempty"` // "Static" | "Dynamic"
	SKU              string `json:"sku,omitempty"`               // "Basic" | "Standard"
}

type NIC struct {
	Name                 string            `json:"name"`
	Subnet               string            `json:"subnet"`
	NetworkSecurityGroup *string           `json:"networkSecurityGroup"`
	PublicIP             *string           `json:"publicIp"`
	PrivateIP            string            `json:"privateIp"`
	Tags                 map[string]string `json:"tags"`
	// DNSServers lists the custom DNS servers configured on this NIC.
	// Empty = NIC uses Azure-provided DNS (168.63.129.16).
	// Non-empty = custom resolver — critical for hybrid DNS path analysis:
	// if a NIC uses an on-prem DNS server that doesn't forward privatelink.* zones
	// to Azure DNS, PE name resolution will silently use public IPs.
	DNSServers []string `json:"dnsServers,omitempty"`
}

type NetworkWatcher struct {
	EffectiveSecurityRules map[string][]SecRule `json:"effectiveSecurityRules"`
	EffectiveRoutes        map[string][]Route   `json:"effectiveRoutes"`
}

type AVNM struct {
	SecurityAdminRules []AdminRule `json:"securityAdminRules"`
}

type AdminRule struct {
	Name                 string   `json:"name"`
	Priority             int      `json:"priority"`
	Direction            string   `json:"direction"`
	Access               string   `json:"access"`
	Protocol             string   `json:"protocol"`
	SourceAddressPrefix  string   `json:"sourceAddressPrefix"`
	DestinationPortRange string   `json:"destinationPortRange"`
	AppliesTo            []string `json:"appliesTo"`
}

type Firewall struct {
	Name      string    `json:"name"`
	PrivateIP string    `json:"privateIp"`
	PublicIP  string    `json:"publicIp"`
	NatRules  []NatRule `json:"natRules"`
	// PolicyRef is non-empty when this is a policy-based firewall. The adapter
	// resolves NAT rules from the policy's RuleCollectionGroups and normalises
	// them into NatRules above so the engine sees a single, unified shape.
	PolicyRef string `json:"policyRef,omitempty"`
	// SKUTier is the firewall tier — populated in Phase 2 (TMR-004).
	// Required for fixed-cost reporting in forecast_cost.
	SKUTier string `json:"skuTier,omitempty"` // "Basic" | "Standard" | "Premium"
}

type NatRule struct {
	Name               string   `json:"name"`
	Protocol           string   `json:"protocol"`
	SourceAddresses    []string `json:"sourceAddresses"`
	DestinationAddress string   `json:"destinationAddress"`
	DestinationPort    int      `json:"destinationPort"`
	TranslatedAddress  string   `json:"translatedAddress"`
	TranslatedPort     int      `json:"translatedPort"`
}

// PrivateEndpoint is the consumer side of a Private Link connection.
// GroupId identifies the Azure service sub-resource (e.g. "blob", "vault",
// "sql") and is the key that determines which Private DNS Zone the VNet must
// have linked. Without GroupId the DNS-zone misconfiguration check would
// degrade to a heuristic; with it the check is deterministic.
type PrivateEndpoint struct {
	Name                 string `json:"name"`
	Subnet               string `json:"subnet"`                // "{vnetName}/{subnetName}"
	PrivateIP            string `json:"privateIp"`             // IP of the PE NIC
	GroupId              string `json:"groupId"`               // service sub-resource: "blob", "vault", "sql", "registry", etc.
	PrivateLinkServiceId string `json:"privateLinkServiceId"`  // ARM resource ID of the target service
	ConnectionState      string `json:"connectionState"`       // "Approved" | "Pending" | "Rejected"
}

// LoadBalancer models both External (internet-facing, public frontend) and
// Internal (private frontend, NVA HA) load balancers.
// The critical analysis case is an External LB with inbound NAT rules — a NIC
// in the backend pool is internet-reachable via port-forwarding even without a
// direct public IP (the same DNAT pattern as Azure Firewall NatRules).
type LoadBalancer struct {
	Name           string          `json:"name"`
	Sku            string          `json:"sku"`            // "Standard" | "Basic"
	FrontendIP     string          `json:"frontendIp"`     // public IP for ELB; private IP for ILB
	IsInternal     bool            `json:"isInternal"`     // false = internet-facing (public frontend)
	InboundNatRules []LBNatRule   `json:"inboundNatRules,omitempty"`
	BackendPools   []LBBackendPool `json:"backendPools,omitempty"`
}

// LBNatRule is a single port-forwarding rule on a Load Balancer.
// FrontendPort on the LB's frontend IP is forwarded to BackendPort on BackendNic.
type LBNatRule struct {
	Name         string `json:"name"`
	Protocol     string `json:"protocol"`     // "Tcp" | "Udp"
	FrontendPort int    `json:"frontendPort"`
	BackendPort  int    `json:"backendPort"`
	BackendNic   string `json:"backendNic"`   // NIC name receiving forwarded traffic
}

// LBBackendPool is the set of NICs (or IP addresses) that receive load-balanced traffic.
type LBBackendPool struct {
	Name    string   `json:"name"`
	NicRefs []string `json:"nicRefs"` // NIC names in this backend pool
}

// PrivateDnsZone represents an Azure Private DNS Zone and its VNet links.
// A zone not linked to a VNet that hosts a PE for that service means DNS
// resolution in that VNet goes public — the PE's security guarantee is broken.
type PrivateDnsZone struct {
	Name       string       `json:"name"`       // e.g. "privatelink.blob.core.windows.net"
	LinkedVnets []string    `json:"linkedVnets"` // VNet names with autoregistration or manual link
	ARecords   []DnsARecord `json:"aRecords,omitempty"`
}

type DnsARecord struct {
	Name string `json:"name"` // relative hostname (e.g. "myaccount")
	IP   string `json:"ip"`   // private IP of the PE NIC
}

// ApplicationGateway represents an Azure Application Gateway.
// WAF mode and public IP presence drive security findings.
type ApplicationGateway struct {
	Name         string             `json:"name"`
	Subnet       string             `json:"subnet"`              // "{vnetName}/{subnetName}"
	PublicIP     string             `json:"publicIp,omitempty"`  // empty = internal only
	WafEnabled   bool               `json:"wafEnabled"`
	WafMode      string             `json:"wafMode,omitempty"`   // "Prevention" | "Detection" | ""
	BackendPools []AppGWBackendPool `json:"backendPools,omitempty"`
}

type AppGWBackendPool struct {
	Name    string   `json:"name"`
	Targets []string `json:"targets"` // private IPs or FQDNs of backend members
}

// AKSCluster represents an Azure Kubernetes Service cluster.
// IsPrivateCluster=false means the API server is reachable from the public internet.
type AKSCluster struct {
	Name             string `json:"name"`
	Subnet           string `json:"subnet"`               // "{vnetName}/{subnetName}" of node pool subnet
	PodCidr          string `json:"podCidr,omitempty"`
	ServiceCidr      string `json:"serviceCidr,omitempty"`
	IsPrivateCluster bool   `json:"isPrivateCluster"`
	ApiServerIP      string `json:"apiServerIp,omitempty"` // PE IP for private cluster API server
}

// NatGateway represents an Azure NAT Gateway.
// Subnets with a NAT GW have explicit outbound internet regardless of route table.
type NatGateway struct {
	Name              string   `json:"name"`
	PublicIPs         []string `json:"publicIps,omitempty"`
	AssociatedSubnets []string `json:"associatedSubnets"` // "{vnetName}/{subnetName}"
}

// PrivateLinkService represents an Azure Private Link Service (PLS).
// PLS is the provider side of a private link connection (vs PE which is the consumer).
// Bastion NVA patterns and cross-tenant access use PLS.
type PrivateLinkService struct {
	Name                  string   `json:"name"`
	Subnet                string   `json:"subnet"`                 // "{vnetName}/{subnetName}"
	NatIPConfig           string   `json:"natIpConfig,omitempty"`  // private IP used for SNAT
	LinkedPrivateEndpoints []string `json:"linkedPrivateEndpoints,omitempty"` // PE names connected to this PLS
}

// ExpressRouteCircuit represents an Azure ExpressRoute circuit.
// When BGP advertises a default route (0.0.0.0/0) via ER, Gateway Subnet routes
// override UDRs on connected VNets — Gate 3 must account for this.
type ExpressRouteCircuit struct {
	Name                    string `json:"name"`
	PeeringLocation         string `json:"peeringLocation,omitempty"`
	BandwidthMbps           int    `json:"bandwidthMbps,omitempty"`
	ConnectedVnet           string `json:"connectedVnet,omitempty"`         // VNet name of the connected virtual network gateway
	BGPAdvertisesDefaultRoute bool `json:"bgpAdvertisesDefaultRoute"`       // true = on-prem advertises 0.0.0.0/0 via BGP → overrides local internet routing
}

// CrossSubPeering captures a VNet peering that crosses a subscription boundary.
// These are not returned in a single-subscription Resource Graph query and require
// separate collection. HasHubFirewall=false means traffic is unrestricted between subs.
type CrossSubPeering struct {
	LocalVnet            string `json:"localVnet"`
	RemoteVnet           string `json:"remoteVnet"`
	RemoteSubscriptionID string `json:"remoteSubscriptionId"`
	State                string `json:"state"`
	AllowForwardedTraffic bool  `json:"allowForwardedTraffic"`
	HasHubFirewall       bool   `json:"hasHubFirewall"` // true = a firewall sits in the peering path
}

// APIManagement represents an Azure API Management instance.
// VNetMode determines the exposure surface:
//   - "None"     → no VNet injection; gateway is fully public, backend calls bypass network controls
//   - "External" → VNet-injected with a public-facing gateway; network controls apply to backend
//   - "Internal" → VNet-injected with no public gateway; only accessible within VNet/peered networks
//
// HasWAFFrontEnd should be set true when an Application Gateway (WAF) or Front Door (WAF)
// sits in front of the APIM gateway URL — determined by the adapter by matching APIM's
// gateway IP against APP GW backend pools.
type APIManagement struct {
	Name           string `json:"name"`
	Subnet         string `json:"subnet,omitempty"`      // "{vnetName}/{subnetName}" — empty when VNetMode=None
	PublicIP       string `json:"publicIp,omitempty"`
	VNetMode       string `json:"vnetMode"`               // "External" | "Internal" | "None"
	GatewayURL     string `json:"gatewayUrl,omitempty"`
	HasWAFFrontEnd bool   `json:"hasWafFrontEnd"`         // true = APP GW or Front Door WAF is upstream
	SkuName        string `json:"skuName,omitempty"`      // "Developer"|"Basic"|"Standard"|"Premium"
}

// AzureBastion represents an Azure Bastion host.
// Its presence establishes a security contract: management access (SSH/RDP) to VMs
// in the same subscription should flow exclusively through Bastion.
// A NIC with a direct public IP AND management ports (22/3389) open from the internet
// while Bastion is deployed is a Bastion bypass — the VM is reachable by a path
// that circumvents the Bastion controls.
type AzureBastion struct {
	Name     string `json:"name"`
	Subnet   string `json:"subnet"`    // must be "AzureBastionSubnet" in the VNet
	PublicIP string `json:"publicIp"`
	SKU      string `json:"sku,omitempty"` // "Basic" | "Standard"
}

// VirtualNetworkGateway represents an Azure VPN or ExpressRoute gateway.
// Required by the adapter to:
//   (a) Associate ExpressRoute circuits with VNets via the gateway resource
//   (b) Detect forced tunneling (EnableForcedTunneling or BGP default route) intent
//   (c) Enable GatewaySubnet NSG/UDR validation via the existing NSG analysis
//
// The engine does NOT need VNG for Gate 3 correctness — NW effective routes already
// capture the BGP outcome at the NIC level. VNG is adapter context and Phase-2 analysis.
type VirtualNetworkGateway struct {
	Name                  string   `json:"name"`
	Subnet                string   `json:"subnet"`                        // must be "GatewaySubnet"
	GatewayType           string   `json:"gatewayType"`                   // "Vpn" | "ExpressRoute"
	SKU                   string   `json:"sku,omitempty"`                 // "ErGw1AZ"|"ErGw2AZ"|"ErGw3AZ"|"VpnGw1"-"VpnGw5"|"Basic"
	PublicIP              string   `json:"publicIp,omitempty"`
	EnableBGP             bool     `json:"enableBgp"`
	EnableForcedTunneling bool     `json:"enableForcedTunneling"`         // true = BGP default route advertisement → on-prem forces internet traffic
	ConnectedCircuitIds   []string `json:"connectedCircuitIds,omitempty"` // ARM IDs of connected ER circuits
}

// ─── Phase 2 placeholder structs ───────────────────────────────────────────────
// These are collected by the adapter (Phase 1) but have no analysis rule yet.
// Analysis rules will be added in Phase 2 steps.

// DNSPrivateResolver enables hybrid DNS: on-prem clients → Azure Private DNS and
// Azure → on-prem DNS. Analysis (Phase 2): if no inbound endpoint covers the
// privatelink.* zones, on-prem clients resolve PEs via public DNS.
type DNSPrivateResolver struct {
	Name             string               `json:"name"`
	VNet             string               `json:"vnet"`
	InboundEndpoints []DNSResolverEndpoint `json:"inboundEndpoints,omitempty"`  // on-prem → Azure
	OutboundEndpoints []DNSResolverEndpoint `json:"outboundEndpoints,omitempty"` // Azure → on-prem
}

type DNSResolverEndpoint struct {
	Name      string `json:"name"`
	Subnet    string `json:"subnet"` // "{vnetName}/{subnetName}"
	IPAddress string `json:"ipAddress,omitempty"`
}

// AzureRouteServer enables BGP between NVAs and Azure SDN. When present, NVA-learned
// routes propagate to ALL VNets connected to the Route Server VNet — can override UDRs.
// Analysis (Phase 2): if Route Server is present and NVA BGP peers advertise 0.0.0.0/0,
// effective routes on all connected spokes would route to the NVA, not the internet.
type AzureRouteServer struct {
	Name         string   `json:"name"`
	Subnet       string   `json:"subnet"`          // must be "RouteServerSubnet"
	PublicIP     string   `json:"publicIp,omitempty"`
	BGPPeerASNs  []int    `json:"bgpPeerAsns,omitempty"`
	ConnectedVNets []string `json:"connectedVnets,omitempty"`
}

// AzureFrontDoor represents a Front Door profile (Standard/Premium tier with WAF support).
// WAF policy is per-endpoint in Front Door Standard/Premium (unlike classic Front Door
// where it was profile-level). An endpoint without a WAF policy ID has no L7 protection.
// Analysis: FD with WafEnabled=false or WafMode="Detection" → Medium/Informational finding.
type AzureFrontDoor struct {
	Name      string              `json:"name"`
	SKU       string              `json:"sku,omitempty"`     // "Standard_AzureFrontDoor" | "Premium_AzureFrontDoor"
	WafEnabled bool               `json:"wafEnabled"`        // true = at least one WAF policy is associated
	WafMode   string              `json:"wafMode,omitempty"` // "Prevention" | "Detection" — worst-case mode across all endpoints
	Endpoints []FrontDoorEndpoint `json:"endpoints,omitempty"`
}

// FrontDoorEndpoint is a Front Door routing endpoint.
// WafPolicyId empty means no WAF policy linked to this endpoint — unprotected internet exposure.
type FrontDoorEndpoint struct {
	Name        string `json:"name"`
	Hostname    string `json:"hostname"`              // e.g. "api-contoso.z01.azurefd.net"
	WafPolicyId string `json:"wafPolicyId,omitempty"` // ARM resource ID of the linked WAF policy; empty = unprotected
	Enabled     bool   `json:"enabled"`
}

// DDoSProtectionPlan is linked to one or more VNets. VNets without a DDoS plan
// rely on basic/default protection only — volumetric attacks are not mitigated.
// Analysis (Phase 2): VNet without linked DDoS plan → Informational.
type DDoSProtectionPlan struct {
	Name           string   `json:"name"`
	LinkedVNets    []string `json:"linkedVnets"`
}

// LocalNetworkGateway represents an on-premises network in a VPN connection.
// It holds the on-prem CIDR ranges and BGP settings.
// Analysis (Phase 2): overlapping on-prem ranges with Azure VNets (shadow routing).
type LocalNetworkGateway struct {
	Name            string   `json:"name"`
	GatewayIPAddress string  `json:"gatewayIpAddress"`    // public IP of on-prem VPN device
	AddressPrefixes  []string `json:"addressPrefixes"`     // on-prem CIDR ranges
	BGPAsn           int      `json:"bgpAsn,omitempty"`
}

// VirtualWAN represents an Azure Virtual WAN resource.
// vWAN is architecturally different from traditional hub-spoke:
//   - Spoke VNets connect to vHubs (Microsoft-managed routing appliances), not to each other via peerings
//   - vHub routing tables determine traffic paths — not UDRs on spoke subnets
//   - NW effective routes at the NIC level DO account for vWAN routing (Gate 3 remains valid)
//   - The adapter must detect vWAN presence and collect vHub topology separately
//
// Analysis rule: vHub without a secured firewall (HasSecuredFirewall=false) means
// spoke-to-spoke and spoke-to-internet traffic is NOT inspected — lateral movement is unrestricted.
type VirtualWAN struct {
	Name     string        `json:"name"`
	SKU      string        `json:"sku"`   // "Basic" | "Standard"
	VHubs    []VirtualHub  `json:"vHubs"`
}

// VirtualHub is a Microsoft-managed routing appliance inside a vWAN.
// A secured vHub has an Azure Firewall instance (via Firewall Manager) that
// intercepts internet-bound and/or private (spoke-to-spoke) traffic.
// RoutingPolicyInternet=true → internet-bound traffic routes through the firewall.
// RoutingPolicyPrivate=true → private (spoke-to-spoke) traffic routes through the firewall.
type VirtualHub struct {
	Name                  string   `json:"name"`
	AddressPrefix         string   `json:"addressPrefix"`         // vHub private address space
	Location              string   `json:"location,omitempty"`
	SpokeConnections      []string `json:"spokeConnections"`      // names of connected VNets
	HasSecuredFirewall    bool     `json:"hasSecuredFirewall"`    // true = Azure Firewall in vHub
	FirewallPrivateIP     string   `json:"firewallPrivateIp,omitempty"`
	RoutingPolicyInternet bool     `json:"routingPolicyInternet"` // internet traffic through FW
	RoutingPolicyPrivate  bool     `json:"routingPolicyPrivate"`  // private traffic through FW
}

// ─── P1 Enrichment envelope ─────────────────────────────────────────────────────
// Enrichment holds optional data from Defender for Cloud, Azure Policy, and Activity
// Logs. The deterministic engine (Analyze) does NOT read Enrichment — it is surfaced
// by the MCP explainer layer as supporting context alongside engine findings.
//
// Phase 1 adapter: populate only when caller passes enrich=true.
// Phase 2: engine rules for attack path cross-correlation (e.g., engine finds NIC
// internet-reachable + Defender marks same resource as "MFA not enforced" → Critical).

type Enrichment struct {
	DefenderAssessments []DefenderAssessment `json:"defenderAssessments,omitempty"`
	PolicyFindings      []PolicyFinding      `json:"policyFindings,omitempty"`
	RecentChanges       []ActivityLogEntry   `json:"recentChanges,omitempty"` // last 90 days, network-plane only
	// FlowLogStatuses captures whether NSG/VNet flow logging is enabled per resource.
	// Source: Network Watcher Flow Log API (GET /networkWatchers/{nw}/flowLogs).
	// A resource without flow logs enabled means traffic forensics are unavailable
	// for that segment — critical for incident response and compliance (e.g., PCI-DSS).
	FlowLogStatuses []FlowLogSummary `json:"flowLogStatuses,omitempty"`
}

// DefenderAssessment is a single Defender for Cloud security recommendation for a resource.
// Source: GET /subscriptions/{sub}/providers/Microsoft.Security/assessments
type DefenderAssessment struct {
	ResourceId        string `json:"resourceId"`
	AssessmentName    string `json:"assessmentName"`     // human-readable recommendation name
	Status            string `json:"status"`             // "Healthy" | "Unhealthy" | "NotApplicable"
	Severity          string `json:"severity"`           // "High" | "Medium" | "Low"
	RemediationSteps  string `json:"remediationSteps,omitempty"`
}

// PolicyFinding is the compliance state of a resource against an Azure Policy definition.
// Source: GET /subscriptions/{sub}/providers/Microsoft.PolicyInsights/policyStates/latest/queryResults
type PolicyFinding struct {
	ResourceId          string `json:"resourceId"`
	PolicyDefinitionName string `json:"policyDefinitionName"`
	ComplianceState     string `json:"complianceState"` // "Compliant" | "NonCompliant" | "Exempt"
	PolicySetName       string `json:"policySetName,omitempty"` // initiative (e.g., "Azure Security Benchmark")
}

// ActivityLogEntry is a network-plane change event from the Azure Activity Log.
// Source: GET /subscriptions/{sub}/providers/microsoft.insights/eventtypes/management/values
// Filtered to: resourceType in (NSG, RouteTable, VNet, Firewall, APIM, LB, ...), last 90 days.
// Supports drift detection: "this NSG rule was modified 3 days ago by service principal X."
type ActivityLogEntry struct {
	Timestamp     string `json:"timestamp"`
	ResourceId    string `json:"resourceId"`
	OperationName string `json:"operationName"` // e.g. "Microsoft.Network/networkSecurityGroups/write"
	ChangedBy     string `json:"changedBy"`     // UPN or service principal ID
	Status        string `json:"status"`        // "Succeeded" | "Failed"
}

// FlowLogSummary captures the flow logging status of an NSG or VNet.
// Source: GET /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/networkWatchers/{nw}/flowLogs
// A segment without flow logs is a forensics blind spot — traffic cannot be reconstructed
// after a security incident, and compliance posture (PCI-DSS 10.x, SOC 2) is weakened.
type FlowLogSummary struct {
	ResourceId    string `json:"resourceId"`              // ARM ID of the NSG or VNet
	ResourceName  string `json:"resourceName"`
	ResourceType  string `json:"resourceType"`            // "NSG" | "VNet"
	Enabled       bool   `json:"enabled"`
	StorageAccount string `json:"storageAccount,omitempty"` // destination storage account name
	RetentionDays int    `json:"retentionDays,omitempty"`  // 0 = forever
}

// Load parses a topology export (the Azure adapter will produce this shape from
// Resource Graph + Network Watcher in production).
func Load(path string) (*Fixture, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f Fixture
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	return &f, nil
}
