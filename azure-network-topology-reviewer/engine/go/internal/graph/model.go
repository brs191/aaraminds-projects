// Package graph is the topology model and the Azure fixture parser. The model is
// Azure-shaped in v1 (NSG, AVNM, Azure Firewall, effective rules/routes); making it
// genuinely cloud-neutral — so the AWS adapter is a second adapter rather than a
// rewrite — is deferred until that adapter lands (see engine-plan.md "Risks").
// It mirrors the reference implementation (reference/analyze.py).
package graph

import (
	"encoding/json"
	"os"
)

type Fixture struct {
	Subscription   string         `json:"subscription"`
	ResourceGraph  ResourceGraph  `json:"resourceGraph"`
	NetworkWatcher NetworkWatcher `json:"networkWatcher"`
	AVNM           AVNM           `json:"avnm"`
	AzureFirewall  *Firewall      `json:"azureFirewall,omitempty"`
}

type ResourceGraph struct {
	VirtualNetworks       []VNet       `json:"virtualNetworks"`
	NetworkSecurityGroups []NSG        `json:"networkSecurityGroups"`
	RouteTables           []RouteTable `json:"routeTables"`
	PublicIPAddresses     []PublicIP   `json:"publicIPAddresses"`
	NetworkInterfaces     []NIC        `json:"networkInterfaces"`
}

type VNet struct {
	Name         string    `json:"name"`
	AddressSpace []string  `json:"addressSpace"`
	Subnets      []Subnet  `json:"subnets"`
	Peerings     []Peering `json:"peerings"`
}

type Subnet struct {
	Name                 string `json:"name"`
	AddressPrefix        string `json:"addressPrefix"`
	NetworkSecurityGroup string `json:"networkSecurityGroup"`
	RouteTable           string `json:"routeTable"`
}

type Peering struct {
	RemoteVnet            string `json:"remoteVnet"`
	State                 string `json:"state"`
	AllowForwardedTraffic bool   `json:"allowForwardedTraffic"`
	AllowGatewayTransit   bool   `json:"allowGatewayTransit"`
	UseRemoteGateways     bool   `json:"useRemoteGateways"`
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
}

type NIC struct {
	Name                 string            `json:"name"`
	Subnet               string            `json:"subnet"`
	NetworkSecurityGroup *string           `json:"networkSecurityGroup"`
	PublicIP             *string           `json:"publicIp"`
	PrivateIP            string            `json:"privateIp"`
	Tags                 map[string]string `json:"tags"`
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
