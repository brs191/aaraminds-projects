// Package simulator implements the Phase 2 topology what-if simulation.
// ApplyDelta takes an immutable *graph.Fixture and a TopologyDelta, deep-copies
// the fixture, applies the structural change, projects simulated effective
// rules/routes onto affected NICs, and returns the new fixture. DiffFindings
// then compares Analyze(original) with Analyze(simulated) to produce a
// SecurityDelta showing added and mitigated risks.
package simulator

import (
	"fmt"

	"github.com/aaraminds/azure-nettopo-engine/internal/graph"
)

// TopologyDelta describes a single proposed change to a topology.
// Exactly one operation field must be non-nil; Validate enforces this.
// Multiple changes must be applied as sequential ApplyDelta calls so callers
// see the incremental security impact of each individual change.
type TopologyDelta struct {
	AddSubnet      *AddSubnetOp      `json:"addSubnet,omitempty"`
	RemoveSubnet   *RemoveSubnetOp   `json:"removeSubnet,omitempty"`
	AddNSGRule     *AddNSGRuleOp     `json:"addNsgRule,omitempty"`
	RemoveNSGRule  *RemoveNSGRuleOp  `json:"removeNsgRule,omitempty"`
	AddPeering     *AddPeeringOp     `json:"addPeering,omitempty"`
	RemovePeering  *RemovePeeringOp  `json:"removePeering,omitempty"`
	AddPublicIP    *AddPublicIPOp    `json:"addPublicIp,omitempty"`
	RemovePublicIP *RemovePublicIPOp `json:"removePublicIp,omitempty"`
	ModifyRoute    *ModifyRouteOp    `json:"modifyRoute,omitempty"`
}

// Validate returns a non-nil error if the delta is structurally invalid.
// It does not check that named resources exist — that is ApplyDelta's job.
func (d TopologyDelta) Validate() error {
	count := 0
	if d.AddSubnet != nil {
		count++
	}
	if d.RemoveSubnet != nil {
		count++
	}
	if d.AddNSGRule != nil {
		count++
	}
	if d.RemoveNSGRule != nil {
		count++
	}
	if d.AddPeering != nil {
		count++
	}
	if d.RemovePeering != nil {
		count++
	}
	if d.AddPublicIP != nil {
		count++
	}
	if d.RemovePublicIP != nil {
		count++
	}
	if d.ModifyRoute != nil {
		count++
	}
	if count == 0 {
		return fmt.Errorf("TopologyDelta: no operation set; exactly one must be non-nil")
	}
	if count > 1 {
		return fmt.Errorf("TopologyDelta: %d operations set; exactly one must be non-nil (apply sequentially for multi-step changes)", count)
	}

	// Per-operation field validation.
	if op := d.AddSubnet; op != nil {
		if op.VNetName == "" {
			return fmt.Errorf("AddSubnet: VNetName is required")
		}
		if op.Name == "" {
			return fmt.Errorf("AddSubnet: Name is required")
		}
		if op.AddressPrefix == "" {
			return fmt.Errorf("AddSubnet: AddressPrefix is required")
		}
	}
	if op := d.RemoveSubnet; op != nil {
		if op.VNetName == "" {
			return fmt.Errorf("RemoveSubnet: VNetName is required")
		}
		if op.SubnetName == "" {
			return fmt.Errorf("RemoveSubnet: SubnetName is required")
		}
	}
	if op := d.AddNSGRule; op != nil {
		if op.NSGName == "" {
			return fmt.Errorf("AddNSGRule: NSGName is required")
		}
		if op.Rule.Name == "" {
			return fmt.Errorf("AddNSGRule: Rule.Name is required")
		}
	}
	if op := d.RemoveNSGRule; op != nil {
		if op.NSGName == "" {
			return fmt.Errorf("RemoveNSGRule: NSGName is required")
		}
		if op.RuleName == "" {
			return fmt.Errorf("RemoveNSGRule: RuleName is required")
		}
	}
	if op := d.AddPeering; op != nil {
		if op.LocalVNet == "" {
			return fmt.Errorf("AddPeering: LocalVNet is required")
		}
		if op.RemoteVNet == "" {
			return fmt.Errorf("AddPeering: RemoteVNet is required")
		}
	}
	if op := d.RemovePeering; op != nil {
		if op.LocalVNet == "" {
			return fmt.Errorf("RemovePeering: LocalVNet is required")
		}
		if op.RemoteVNet == "" {
			return fmt.Errorf("RemovePeering: RemoteVNet is required")
		}
	}
	if op := d.AddPublicIP; op != nil {
		if op.NICName == "" {
			return fmt.Errorf("AddPublicIP: NICName is required")
		}
		if op.PIPName == "" {
			return fmt.Errorf("AddPublicIP: PIPName is required")
		}
	}
	if op := d.RemovePublicIP; op != nil {
		if op.NICName == "" {
			return fmt.Errorf("RemovePublicIP: NICName is required")
		}
	}
	if op := d.ModifyRoute; op != nil {
		if op.RouteTableName == "" {
			return fmt.Errorf("ModifyRoute: RouteTableName is required")
		}
		if op.RouteName == "" {
			return fmt.Errorf("ModifyRoute: RouteName is required")
		}
		switch op.NewNextHopType {
		case "Internet", "VirtualAppliance", "None", "VirtualNetworkGateway", "VnetLocal":
		default:
			if op.NewNextHopType == "" {
				return fmt.Errorf("ModifyRoute: NewNextHopType is required")
			}
			return fmt.Errorf("ModifyRoute: unknown NextHopType %q; expected Internet|VirtualAppliance|None|VirtualNetworkGateway|VnetLocal", op.NewNextHopType)
		}
		if op.NewNextHopType == "VirtualAppliance" && op.NewNextHopIP == "" {
			return fmt.Errorf("ModifyRoute: NewNextHopIP is required when NewNextHopType is VirtualAppliance")
		}
	}
	return nil
}

// UnsupportedDeltaError is returned by ApplyDelta when the caller requests
// an operation that is explicitly out of scope for Phase 2.
type UnsupportedDeltaError struct {
	Operation string
	Reason    string
}

func (e *UnsupportedDeltaError) Error() string {
	return fmt.Sprintf("unsupported delta operation %q: %s", e.Operation, e.Reason)
}

// ---- operation structs ----

type AddSubnetOp struct {
	// VNetName is the name of the existing VNet to add the subnet to.
	VNetName string `json:"vnetName"`
	// Name is the new subnet name; must not already exist in the VNet.
	Name string `json:"name"`
	// AddressPrefix is the CIDR for the new subnet (e.g. "10.1.5.0/24").
	AddressPrefix string `json:"addressPrefix"`
	// NSGName is the bare NSG name to associate; empty = no NSG.
	NSGName string `json:"nsgName"`
	// RouteTableName is the bare route table name to associate; empty = no RT.
	RouteTableName string `json:"routeTableName"`
}

type RemoveSubnetOp struct {
	VNetName   string `json:"vnetName"`
	SubnetName string `json:"subnetName"`
}

type AddNSGRuleOp struct {
	// NSGName is the name of the existing NSG to add the rule to.
	NSGName string `json:"nsgName"`
	// Rule is the SecRule to add; Rule.Name must be unique within the NSG.
	Rule graph.SecRule `json:"rule"`
}

type RemoveNSGRuleOp struct {
	NSGName  string `json:"nsgName"`
	RuleName string `json:"ruleName"`
}

type AddPeeringOp struct {
	LocalVNet  string `json:"localVnet"`
	RemoteVNet string `json:"remoteVnet"`
	// State is "Connected" | "Initiated".
	State                 string `json:"state"`
	AllowForwardedTraffic bool   `json:"allowForwardedTraffic"`
	AllowGatewayTransit   bool   `json:"allowGatewayTransit"`
	UseRemoteGateways     bool   `json:"useRemoteGateways"`
}

type RemovePeeringOp struct {
	LocalVNet  string `json:"localVnet"`
	RemoteVNet string `json:"remoteVnet"`
}

type AddPublicIPOp struct {
	NICName string `json:"nicName"`
	PIPName string `json:"pipName"`
	// IPAddress is a simulated IP for the new PIP (e.g. "20.10.10.50").
	IPAddress string `json:"ipAddress"`
}

type RemovePublicIPOp struct {
	// NICName is the NIC whose PIP is to be detached.
	NICName string `json:"nicName"`
}

type ModifyRouteOp struct {
	RouteTableName string `json:"routeTableName"`
	RouteName      string `json:"routeName"`
	// NewNextHopType: "Internet" | "VirtualAppliance" | "None" | "VirtualNetworkGateway" | "VnetLocal"
	NewNextHopType string `json:"newNextHopType"`
	// NewNextHopIP is required when NewNextHopType == "VirtualAppliance".
	NewNextHopIP string `json:"newNextHopIp,omitempty"`
}
