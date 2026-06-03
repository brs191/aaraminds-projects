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
