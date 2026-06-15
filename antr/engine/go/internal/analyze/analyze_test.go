package analyze

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/aaraminds/azure-nettopo-engine/internal/graph"
)

func load(t *testing.T, name string) *graph.Fixture {
	t.Helper()
	fx, err := graph.Load(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatalf("load %s: %v", name, err)
	}
	return fx
}

func reachableHigh(fs []Finding, res string) bool {
	for _, f := range fs {
		if f.Reachable && (f.Severity == "High" || f.Severity == "Critical") && f.Resource == res {
			return true
		}
	}
	return false
}

func has(fs []Finding, typeSub, res string) bool {
	for _, f := range fs {
		if strings.Contains(f.Type, typeSub) && (res == "" || f.Resource == res) {
			return true
		}
	}
	return false
}

func anyReachableInternet(fs []Finding) bool {
	for _, f := range fs {
		if f.Reachable && strings.Contains(f.Evidence, "->Internet") {
			return true
		}
	}
	return false
}

func TestF1_InternetExposureRealVsLatent(t *testing.T) {
	fs := Analyze(load(t, "fixture-1-internet-exposure.json"))
	if !reachableHigh(fs, "nic-vm-web-a") {
		t.Error("spoke-a SSH must be reachable High")
	}
	if reachableHigh(fs, "nic-vm-web-b") {
		t.Error("TRAP: spoke-b (firewalled, no public IP) must NOT be reachable")
	}
	if !has(fs, "orphaned", "pip-orphan-01") {
		t.Error("orphaned public IP must be flagged")
	}
}

func TestF2_DefaultAllowVnetInbound(t *testing.T) {
	fs := Analyze(load(t, "fixture-2-segmentation-peering.json"))
	if !reachableHigh(fs, "nic-db1") {
		t.Error("sensitive db reachable VNet-wide (default AllowVnetInBound) must be High")
	}
	if anyReachableInternet(fs) {
		t.Error("no internet exposure expected (no public IPs)")
	}
}

func TestF3_AVNMSourceScopeAndCIDR(t *testing.T) {
	fs := Analyze(load(t, "fixture-3-cidr-avnm.json"))
	if !reachableHigh(fs, "nic-edge1") {
		t.Error("edge 443 via AVNM AlwaysAllow must be reachable High")
	}
	if reachableHigh(fs, "nic-mgmt1") {
		t.Error("TRAP: mgmt RDP internet path closed by AVNM Deny must NOT be reachable")
	}
	if !has(fs, "CIDR overlap", "") {
		t.Error("ov-a/ov-b CIDR overlap must be flagged")
	}
}

func TestH1_FirewallDNAT(t *testing.T) {
	fs := Analyze(load(t, "fixture-h1-dnat-multihop.json"))
	if !reachableHigh(fs, "nic-backend1") {
		t.Error("backend1 reachable via firewall DNAT despite no public IP")
	}
	if reachableHigh(fs, "nic-backend2") {
		t.Error("TRAP: backend2 has no DNAT rule -> must NOT be reachable")
	}
}

func TestH2_BlackholeAndTags(t *testing.T) {
	fs := Analyze(load(t, "fixture-h2-blackhole-tags.json"))
	if !reachableHigh(fs, "nic-edge") {
		t.Error("edge 443 from Internet must be reachable High")
	}
	if !reachableHigh(fs, "nic-api") {
		t.Error("api 443 from AzureCloud (cross-tenant) must be reachable High")
	}
	if reachableHigh(fs, "nic-dark") {
		t.Error("TRAP: darkpool black-holed (route None) must NOT be reachable")
	}
}

func TestF6_PrivateDNSMisconfiguration(t *testing.T) {
	fs := Analyze(load(t, "fixture-f6-pe-dns-misconfiguration.json"))
	// pe-storage-a is in spoke-a-vnet (groupId=blob) but zone is NOT linked to spoke-a-vnet → High finding.
	if !has(fs, "private DNS zone not linked", "pe-storage-a") {
		t.Error("pe-storage-a (blob) in spoke-a-vnet — zone not linked to spoke-a → must produce 'private DNS zone not linked to VNet' High finding on the PE resource")
	}
	// pe-storage-c is in spoke-c-vnet (groupId=blob) AND zone IS linked to spoke-c-vnet → no finding.
	for _, f := range fs {
		if strings.Contains(f.Type, "private DNS zone") && f.Resource == "pe-storage-c" {
			t.Errorf("TRAP: pe-storage-c zone is correctly linked to spoke-c-vnet, must not be flagged; got: %v", f)
		}
	}
	// spoke-b-vnet has compute NICs only (no PE for blob) — must NOT produce a DNS finding.
	for _, f := range fs {
		if strings.Contains(f.Type, "private DNS zone") && strings.Contains(f.Evidence, "spoke-b-vnet") {
			t.Errorf("TRAP: spoke-b-vnet has no PE for blob, must not get a DNS finding; got: %v", f)
		}
	}
	// No internet reachability expected.
	if anyReachableInternet(fs) {
		t.Error("no internet exposure expected in this fixture")
	}
}

func TestF7_AppGatewayWAF(t *testing.T) {
	fs := Analyze(load(t, "fixture-f7-appgw-waf-disabled.json"))
	// appgw-prod: public IP + WAF disabled → Medium finding
	if !has(fs, "app gateway WAF disabled", "appgw-prod") {
		t.Error("appgw-prod has public IP with WAF disabled — must produce 'app gateway WAF disabled' Medium finding")
	}
	// appgw-staging: public IP + WAF in Detection → Informational
	if !has(fs, "app gateway WAF in detection mode", "appgw-staging") {
		t.Error("appgw-staging WAF is in Detection mode — must produce informational finding")
	}
	// appgw-internal: no public IP → no WAF finding
	for _, f := range fs {
		if strings.Contains(f.Type, "app gateway") && f.Resource == "appgw-internal" {
			t.Errorf("TRAP: appgw-internal is internal-only, must not produce a WAF finding; got: %v", f)
		}
	}
}

func TestF8_AKSAndCrossSubPeering(t *testing.T) {
	fs := Analyze(load(t, "fixture-f8-aks-and-crosssub-peering.json"))
	// aks-public is not a private cluster → Medium finding
	if !has(fs, "AKS non-private cluster", "aks-public") {
		t.Error("aks-public is not a private cluster — must produce 'AKS non-private cluster' Medium finding")
	}
	// aks-private IS private → no finding
	for _, f := range fs {
		if strings.Contains(f.Type, "AKS non-private") && f.Resource == "aks-private" {
			t.Errorf("TRAP: aks-private is correctly private, must not be flagged; got: %v", f)
		}
	}
	// Cross-sub peering without firewall → Medium finding
	if !has(fs, "cross-subscription peering without firewall", "spoke-a-vnet~remote-vnet") {
		t.Error("cross-subscription peering with no hub firewall must produce a Medium finding")
	}
}

func TestF10_ELBNATPorts(t *testing.T) {
	fs := Analyze(load(t, "fixture-f10-elb-nat.json"))
	// ELB has two inbound NAT rules → both backend NICs must be flagged High.
	if !has(fs, "internet reachable via load balancer NAT", "nic-vm-dmz1") {
		t.Error("nic-vm-dmz1 receives ELB NAT port 22 from public IP — must be flagged High (internet reachable via LB NAT)")
	}
	if !has(fs, "internet reachable via load balancer NAT", "nic-vm-dmz2") {
		t.Error("nic-vm-dmz2 receives ELB NAT port 3389 from public IP — must be flagged High (internet reachable via LB NAT)")
	}
	// ILB (internal, private frontend) must NOT produce any LB NAT finding.
	for _, f := range fs {
		if strings.Contains(f.Type, "load balancer NAT") && f.Resource == "nic-vm-backend" {
			t.Errorf("TRAP: nic-vm-backend is behind an ILB (private frontend), must not produce internet-exposure finding; got: %v", f)
		}
	}
}

func TestF11_APIMExposure(t *testing.T) {
	fs := Analyze(load(t, "fixture-f11-apim-exposure.json"))
	// apim-public: VNetMode=None → Medium finding
	if !has(fs, "APIM without VNet isolation", "apim-public") {
		t.Error("apim-public has VNetMode=None — must produce 'APIM without VNet isolation' Medium finding")
	}
	// apim-external: VNetMode=External, no WAF → Medium finding
	if !has(fs, "APIM External mode without WAF", "apim-external") {
		t.Error("apim-external has VNetMode=External with no WAF upstream — must produce 'APIM External mode without WAF' Medium finding")
	}
	// apim-internal: VNetMode=Internal → no finding
	for _, f := range fs {
		if strings.Contains(f.Type, "APIM") && f.Resource == "apim-internal" {
			t.Errorf("TRAP: apim-internal is correctly locked down (Internal mode), must not produce any APIM finding; got: %v", f)
		}
	}
	// apim-protected: VNetMode=External but hasWafFrontEnd=true → no finding
	for _, f := range fs {
		if strings.Contains(f.Type, "APIM") && f.Resource == "apim-protected" {
			t.Errorf("TRAP: apim-protected has a WAF upstream, must not produce an APIM exposure finding; got: %v", f)
		}
	}
}

func TestF12_BastionBypass(t *testing.T) {
	fs := Analyze(load(t, "fixture-f12-bastion-bypass.json"))
	// nic-vm-exposed: public IP + SSH port 22 open from internet while Bastion is deployed → High
	if !has(fs, "Bastion bypass", "nic-vm-exposed") {
		t.Error("nic-vm-exposed has public IP with SSH open while Bastion is deployed — must produce 'Bastion bypass' High finding")
	}
	// nic-vm-rdp: public IP + RDP port 3389 open from Internet tag while Bastion is deployed → High
	if !has(fs, "Bastion bypass", "nic-vm-rdp") {
		t.Error("nic-vm-rdp has public IP with RDP open (Internet tag) while Bastion is deployed — must produce 'Bastion bypass' High finding")
	}
	// nic-vm-safe: public IP but only port 443 open — not a management port → no Bastion bypass finding
	for _, f := range fs {
		if strings.Contains(f.Type, "Bastion bypass") && f.Resource == "nic-vm-safe" {
			t.Errorf("TRAP: nic-vm-safe only has port 443 open (not a management port), must not produce a Bastion bypass finding; got: %v", f)
		}
	}
	// nic-vm-nopip: no public IP → no Bastion bypass finding (no direct internet path)
	for _, f := range fs {
		if strings.Contains(f.Type, "Bastion bypass") && f.Resource == "nic-vm-nopip" {
			t.Errorf("TRAP: nic-vm-nopip has no public IP, must not produce a Bastion bypass finding; got: %v", f)
		}
	}
}

func TestF13_VirtualWAN(t *testing.T) {
	fs := Analyze(load(t, "fixture-f13-vwan-unsecured.json"))
	// hub-westus: no firewall → Medium "vWAN hub unsecured" finding
	if !has(fs, "vWAN hub unsecured", "hub-westus") {
		t.Error("hub-westus has no secured firewall — must produce 'vWAN hub unsecured' Medium finding")
	}
	// hub-eastus: secured firewall but RoutingPolicyPrivate=false → Medium "bypass private traffic" finding
	if !has(fs, "vWAN hub firewall bypasses private traffic", "hub-eastus") {
		t.Error("hub-eastus has firewall but RoutingPolicyPrivate=false — must produce 'vWAN hub firewall bypasses private traffic' Medium finding")
	}
	// hub-centralus: fully secured (firewall + both routing policies) → no finding
	for _, f := range fs {
		if strings.Contains(f.Type, "vWAN") && f.Resource == "hub-centralus" {
			t.Errorf("TRAP: hub-centralus is fully secured, must not produce any vWAN finding; got: %v", f)
		}
	}
}

func TestF14_FrontDoorWAF(t *testing.T) {
	fx := load(t, "fixture-f14-frontdoor-waf.json")
	got := Analyze(fx)

	// fd-no-waf: no WAF policy → Medium
	if !has(got, "Front Door WAF disabled", "fd-no-waf") {
		t.Error("fd-no-waf has no WAF enabled — must produce 'Front Door WAF disabled' Medium finding")
	}
	// fd-detection-mode: WAF in Detection → Informational
	if !has(got, "Front Door WAF in detection mode", "fd-detection-mode") {
		t.Error("fd-detection-mode WAF is in Detection mode — must produce 'Front Door WAF in detection mode' Informational finding")
	}
	// trap: fd-prevention-mode: WAF in Prevention → no finding
	for _, f := range got {
		if strings.Contains(f.Type, "Front Door") && f.Resource == "fd-prevention-mode" {
			t.Errorf("TRAP: fd-prevention-mode has WAF in Prevention — must not produce any Front Door finding; got: %v", f)
		}
	}
}
