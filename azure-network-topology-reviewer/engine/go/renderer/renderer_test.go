package renderer

import (
	"strings"
	"testing"

	"github.com/aaraminds/azure-nettopo-engine/internal/analyze"
	"github.com/aaraminds/azure-nettopo-engine/internal/graph"
)

// ── shared test fixtures ──────────────────────────────────────────────────────

func threeFindings() []analyze.Finding {
	return []analyze.Finding{
		{
			Type:      "over-permissive NSG (reachable)",
			Severity:  "Critical",
			Resource:  "nic-critical",
			Evidence:  "0.0.0.0/0:22 inbound + route 0.0.0.0/0->Internet + public IP 1.2.3.4",
			Reachable: true,
		},
		{
			Type:      "over-permissive NSG (latent)",
			Severity:  "Informational",
			Resource:  "nic-info",
			Evidence:  "0.0.0.0/0:3389 inbound but no public IP",
			Reachable: false,
		},
		{
			Type:      "CIDR overlap",
			Severity:  "Medium",
			Resource:  "vnet-a~vnet-b",
			Evidence:  "overlapping address space 10.0.0.0/16 / 10.0.1.0/24",
			Reachable: false,
		},
	}
}

func smallFixture() *graph.Fixture {
	return &graph.Fixture{
		Subscription: "00000000-0000-0000-0000-000000000001",
		ResourceGraph: graph.ResourceGraph{
			VirtualNetworks: []graph.VNet{
				{
					Name:         "vnet-hub",
					AddressSpace: []string{"10.0.0.0/16"},
					Subnets: []graph.Subnet{
						{Name: "GatewaySubnet", AddressPrefix: "10.0.0.0/24"},
						{Name: "AppSubnet", AddressPrefix: "10.0.1.0/24"},
					},
					Peerings: []graph.Peering{
						{
							RemoteVnet:            "vnet-spoke",
							State:                 "Connected",
							AllowForwardedTraffic: true,
						},
					},
				},
				{
					Name:         "vnet-spoke",
					AddressSpace: []string{"10.1.0.0/16"},
					Subnets: []graph.Subnet{
						{Name: "WorkloadSubnet", AddressPrefix: "10.1.0.0/24"},
					},
				},
			},
			NetworkInterfaces: []graph.NIC{
				{
					Name:      "nic-vm-01",
					Subnet:    "vnet-hub/GatewaySubnet",
					PrivateIP: "10.0.0.4",
				},
				{
					Name:      "nic-vm-02",
					Subnet:    "vnet-hub/AppSubnet",
					PrivateIP: "10.0.1.4",
				},
				{
					Name:      "nic-spoke-01",
					Subnet:    "vnet-spoke/WorkloadSubnet",
					PrivateIP: "10.1.0.4",
				},
			},
		},
	}
}

// ── ToMarkdown tests ──────────────────────────────────────────────────────────

func TestToMarkdown_Header(t *testing.T) {
	out := ToMarkdown("sub-abc-123", threeFindings())

	if !strings.Contains(out, "# Azure Network Topology Analysis — sub-abc-123") {
		t.Error("missing subscription in header")
	}
	if !strings.Contains(out, "Generated:") {
		t.Error("missing Generated timestamp line")
	}
}

func TestToMarkdown_SeverityTable(t *testing.T) {
	out := ToMarkdown("sub-test", threeFindings())

	for _, want := range []string{
		"## Summary",
		"| Severity | Count |",
		"🔴 Critical | 1",
		"🟠 High | 0",
		"🟡 Medium | 1",
		"🔵 Informational | 1",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in summary table; got:\n%s", want, out)
		}
	}
}

func TestToMarkdown_FindingSections(t *testing.T) {
	out := ToMarkdown("sub-test", threeFindings())

	for _, want := range []string{
		"## Findings",
		"### 🔴 CRITICAL — nic-critical",
		"**Type:** over-permissive NSG (reachable)",
		"**Reachable:** yes",
		"### 🟡 MEDIUM — vnet-a~vnet-b",
		"**Reachable:** no",
		"### 🔵 INFORMATIONAL — nic-info",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in findings; output:\n%s", want, out[:min(500, len(out))])
		}
	}
}

func TestToMarkdown_SortOrder(t *testing.T) {
	out := ToMarkdown("sub-test", threeFindings())

	critPos := strings.Index(out, "### 🔴 CRITICAL")
	medPos := strings.Index(out, "### 🟡 MEDIUM")
	infoPos := strings.Index(out, "### 🔵 INFORMATIONAL")

	if critPos == -1 || medPos == -1 || infoPos == -1 {
		t.Fatal("one or more severity sections missing")
	}
	if !(critPos < medPos && medPos < infoPos) {
		t.Errorf("findings not in Critical→Medium→Informational order: crit=%d med=%d info=%d",
			critPos, medPos, infoPos)
	}
}

func TestToMarkdown_Recommendations(t *testing.T) {
	out := ToMarkdown("sub-test", threeFindings())

	if !strings.Contains(out, "## Recommendations") {
		t.Error("missing Recommendations section")
	}
	if !strings.Contains(out, "Critical and High") {
		t.Error("missing 'Critical and High' recommendation text")
	}
	if !strings.Contains(out, "enrich=true") {
		t.Error("missing enrich=true recommendation")
	}
}

func TestToMarkdown_NoFindings(t *testing.T) {
	out := ToMarkdown("sub-clean", nil)

	if !strings.Contains(out, "_No findings._") {
		t.Error("expected '_No findings._' for empty findings slice")
	}
	if !strings.Contains(out, "🔴 Critical | 0") {
		t.Error("expected zero count for Critical")
	}
}

// ── ToDrawIO tests ────────────────────────────────────────────────────────────

func TestToDrawIO_ValidRoot(t *testing.T) {
	out := ToDrawIO(smallFixture(), threeFindings())

	if !strings.HasPrefix(out, "<mxfile") {
		t.Errorf("expected output to start with <mxfile; got: %q", out[:min(80, len(out))])
	}
	if !strings.Contains(out, "<mxCell") {
		t.Error("expected at least one <mxCell element")
	}
	if !strings.HasSuffix(strings.TrimSpace(out), "</mxfile>") {
		t.Error("expected output to end with </mxfile>")
	}
}

func TestToDrawIO_CellCount(t *testing.T) {
	out := ToDrawIO(smallFixture(), nil)

	// Count <mxCell occurrences.
	// Expected minimum:
	//   2 root cells (id=0, id=1)
	//   2 VNet swimlanes
	//   3 subnet swimlanes (2 in hub, 1 in spoke)
	//   3 NIC ellipses
	//   1 peering edge
	//   1 legend
	// Total ≥ 12
	count := strings.Count(out, "<mxCell")
	const minCells = 12
	if count < minCells {
		t.Errorf("expected at least %d mxCell elements, got %d", minCells, count)
	}
}

func TestToDrawIO_VNetCells(t *testing.T) {
	out := ToDrawIO(smallFixture(), nil)

	for _, id := range []string{"vnet-vnet-hub", "vnet-vnet-spoke"} {
		if !strings.Contains(out, `id="`+id+`"`) {
			t.Errorf("expected VNet cell with id=%q", id)
		}
	}
}

func TestToDrawIO_SubnetCells(t *testing.T) {
	out := ToDrawIO(smallFixture(), nil)

	for _, id := range []string{
		"subnet-vnet-hub-gatewaysubnet",
		"subnet-vnet-hub-appsubnet",
		"subnet-vnet-spoke-workloadsubnet",
	} {
		if !strings.Contains(out, `id="`+id+`"`) {
			t.Errorf("expected subnet cell with id=%q", id)
		}
	}
}

func TestToDrawIO_NICColors(t *testing.T) {
	// nic-vm-01 has a Critical finding via nicFindings["nic-critical"] — but
	// that NIC is NOT in the fixture. Let us use the actual fixture NIC with a
	// Critical finding on it by building a fixture + matching findings.
	fx := smallFixture()
	findings := []analyze.Finding{
		{Severity: "Critical", Resource: "nic-vm-01", Type: "t", Evidence: "e"},
	}
	out := ToDrawIO(fx, findings)

	// nic-vm-01 must have Critical fill color.
	if !strings.Contains(out, "fillColor=#FF0000") {
		t.Error("expected Critical fill color #FF0000 for nic-vm-01")
	}
}

func TestToDrawIO_PeeringEdge(t *testing.T) {
	out := ToDrawIO(smallFixture(), nil)

	if !strings.Contains(out, `edge="1"`) {
		t.Error("expected at least one edge cell for VNet peering")
	}
	// The peering AllowForwardedTraffic=true → should NOT be dashed.
	if strings.Contains(out, "dashed=1") {
		t.Error("peering with AllowForwardedTraffic=true should not be dashed")
	}
}

func TestToDrawIO_PeeringDashed(t *testing.T) {
	fx := smallFixture()
	// Override to AllowForwardedTraffic=false.
	fx.ResourceGraph.VirtualNetworks[0].Peerings[0].AllowForwardedTraffic = false
	out := ToDrawIO(fx, nil)

	if !strings.Contains(out, "dashed=1") {
		t.Error("peering with AllowForwardedTraffic=false should be dashed")
	}
}

func TestToDrawIO_Legend(t *testing.T) {
	out := ToDrawIO(smallFixture(), nil)

	if !strings.Contains(out, `id="legend"`) {
		t.Error("expected legend cell")
	}
	if !strings.Contains(out, "Critical") {
		t.Error("expected 'Critical' in legend")
	}
}

func TestToDrawIO_UnplacedNICs(t *testing.T) {
	fx := smallFixture()
	// Add a NIC with an unknown subnet.
	fx.ResourceGraph.NetworkInterfaces = append(fx.ResourceGraph.NetworkInterfaces,
		graph.NIC{
			Name:      "nic-orphan",
			Subnet:    "vnet-unknown/SubnetX",
			PrivateIP: "172.16.0.5",
		},
	)
	out := ToDrawIO(fx, nil)

	if !strings.Contains(out, "Unplaced NICs") {
		t.Error("expected 'Unplaced NICs' container for NIC with unknown subnet")
	}
	if !strings.Contains(out, "nic-nic-orphan") {
		t.Error("expected nic-orphan to appear in Unplaced NICs container")
	}
}

func TestToDrawIO_Firewall(t *testing.T) {
	fx := smallFixture()
	// Add firewall with private IP inside vnet-hub (10.0.0.0/16).
	fwName := "fw-main"
	fx.AzureFirewall = &graph.Firewall{
		Name:      fwName,
		PrivateIP: "10.0.255.4",
		PublicIP:  "52.1.2.3",
	}
	out := ToDrawIO(fx, nil)

	if !strings.Contains(out, "fw-fw-main") {
		t.Errorf("expected firewall cell id 'fw-fw-main'; output snippet:\n%s",
			out[:min(1000, len(out))])
	}
	if !strings.Contains(out, "fillColor=#f0a30a") {
		t.Error("expected firewall fill color #f0a30a")
	}
}

func TestToDrawIO_LoadBalancer(t *testing.T) {
	fx := smallFixture()
	fx.ResourceGraph.LoadBalancers = []graph.LoadBalancer{
		{Name: "lb-frontend", Sku: "Standard", FrontendIP: "203.0.113.5"},
	}
	out := ToDrawIO(fx, nil)

	if !strings.Contains(out, "lb-lb-frontend") {
		t.Error("expected load balancer cell id 'lb-lb-frontend'")
	}
}
