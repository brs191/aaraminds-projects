// Package renderer — drawio.go produces valid Draw.io mxGraph XML from a topology
// fixture and analysis findings. Output can be imported directly into draw.io or
// Confluence using the "Insert > Diagram from file" / "Edit > XML" feature.
// All XML is built with stdlib fmt/strings only — no external XML library.
package renderer

import (
	"fmt"
	"net/netip"
	"strings"
	"unicode"

	"github.com/aaraminds/azure-nettopo-engine/internal/analyze"
	"github.com/aaraminds/azure-nettopo-engine/internal/graph"
)

// ── Cell-ID helpers ───────────────────────────────────────────────────────────

// slugify replaces all non-alphanumeric characters with "-" and lowercases the result.
func slugify(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteByte('-')
		}
	}
	return b.String()
}

// xmlAttr XML-escapes a string for safe use in an attribute value.
// Encodes &, <, >, ", and '.
func xmlAttr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// drawioLabel builds a cell label, XML-escaping each line and joining with &#xa;
// (the draw.io newline entity) so multi-line labels render correctly.
func drawioLabel(lines ...string) string {
	escaped := make([]string, len(lines))
	for i, l := range lines {
		escaped[i] = xmlAttr(l)
	}
	return strings.Join(escaped, "&#xa;")
}

// ── Severity helpers ──────────────────────────────────────────────────────────

// sevOrderDIO mirrors the priority for the drawio renderer (separate from sevOrder
// in markdown.go to keep the two files independent).
func sevOrderDIO(sev string) int {
	switch sev {
	case "Critical":
		return 0
	case "High":
		return 1
	case "Medium":
		return 2
	case "Informational":
		return 3
	default:
		return 99
	}
}

// highestSeverity returns the highest-priority severity string from a set of
// findings, or "Clean" if the slice is empty.
func highestSeverity(findings []analyze.Finding) string {
	best := 99
	result := "Clean"
	for _, f := range findings {
		if o := sevOrderDIO(f.Severity); o < best {
			best = o
			result = f.Severity
		}
	}
	return result
}

// nicStyle returns the mxCell style string for a NIC ellipse based on its
// highest finding severity.
func nicStyle(sev string) string {
	base := "ellipse;whiteSpace=wrap;html=1;"
	switch sev {
	case "Critical":
		return base + "fillColor=#FF0000;strokeColor=#CC0000;fontColor=#ffffff;"
	case "High":
		return base + "fillColor=#f8cecc;strokeColor=#b85450;"
	case "Medium":
		return base + "fillColor=#fff2cc;strokeColor=#d6b656;"
	case "Informational":
		return base + "fillColor=#dae8fc;strokeColor=#6c8ebf;"
	default: // Clean
		return base + "fillColor=#d5e8d4;strokeColor=#82b366;"
	}
}

// ── IP helper ─────────────────────────────────────────────────────────────────

// ipInVNet returns true if ipStr falls within any of vnet's address prefixes.
func ipInVNet(vnet graph.VNet, ipStr string) bool {
	if ipStr == "" {
		return false
	}
	ip, err := netip.ParseAddr(ipStr)
	if err != nil {
		return false
	}
	for _, cidr := range vnet.AddressSpace {
		pfx, err := netip.ParsePrefix(cidr)
		if err != nil {
			continue
		}
		if pfx.Contains(ip) {
			return true
		}
	}
	return false
}

// ── XML cell writers ──────────────────────────────────────────────────────────

// writeVertex emits a vertex mxCell element with explicit geometry.
func writeVertex(w *strings.Builder, id, value, style, parent string, x, y, width, height int) {
	fmt.Fprintf(w,
		"        <mxCell id=%q value=%q style=%q vertex=\"1\" parent=%q>\n"+
			"          <mxGeometry x=\"%d\" y=\"%d\" width=\"%d\" height=\"%d\" as=\"geometry\"/>\n"+
			"        </mxCell>\n",
		id, value, style, parent,
		x, y, width, height,
	)
}

// writeEdge emits an edge mxCell between two vertex cells.
func writeEdge(w *strings.Builder, id, value, style, srcID, tgtID string) {
	fmt.Fprintf(w,
		"        <mxCell id=%q value=%q style=%q edge=\"1\" source=%q target=%q parent=\"1\">\n"+
			"          <mxGeometry relative=\"1\" as=\"geometry\"/>\n"+
			"        </mxCell>\n",
		id, value, style, srcID, tgtID,
	)
}

// ── Main renderer ─────────────────────────────────────────────────────────────

// ToDrawIO produces valid Draw.io mxGraph XML from a topology fixture and findings.
//
// Layout rules (all coordinates in px):
//   - VNets: 2-column grid, col_x = col*560+20, col_y = row*400+60, w=500
//   - Subnets: nested swimlanes inside VNet, stacked at y=30+i*90, w=460, h=80
//   - NICs: ellipses inside subnet, 3-wide grid at x=10+(j%3)*160, y=25+(j/3)*50, 140×40
//   - Firewall: 120×50 rectangle at x=20,y=60 in hub VNet (matched by private IP), else below grid
//   - Private Endpoints: rhombus 120×50 inside their subnet
//   - Load Balancers: 120×40 rectangles below VNet grid
//   - Peering edges: orthogonal, dashed if AllowForwardedTraffic=false
//   - Legend: 180×130 in top-right at x=1200,y=10
func ToDrawIO(fixture *graph.Fixture, findings []analyze.Finding) string {
	rg := fixture.ResourceGraph

	// ── Index NIC → findings ────────────────────────────────────────────────
	// TODO(V4-07): index by id-or-name once renders carry NIC ids; superseded by phase-4/viz.
	nicFindings := make(map[string][]analyze.Finding, len(findings))
	for _, f := range findings {
		nicFindings[f.Resource] = append(nicFindings[f.Resource], f)
	}

	// ── Index subnet key → NICs / PEs ───────────────────────────────────────
	// subnet key = "{vnetName}/{subnetName}"
	knownSubnets := make(map[string]bool)
	for _, vnet := range rg.VirtualNetworks {
		for _, sub := range vnet.Subnets {
			knownSubnets[vnet.Name+"/"+sub.Name] = true
		}
	}

	subnetNICs := make(map[string][]graph.NIC)
	var unplacedNICs []graph.NIC
	for _, nic := range rg.NetworkInterfaces {
		if knownSubnets[nic.Subnet] {
			subnetNICs[nic.Subnet] = append(subnetNICs[nic.Subnet], nic)
		} else {
			unplacedNICs = append(unplacedNICs, nic)
		}
	}

	subnetPEs := make(map[string][]graph.PrivateEndpoint)
	for _, pe := range rg.PrivateEndpoints {
		if knownSubnets[pe.Subnet] {
			subnetPEs[pe.Subnet] = append(subnetPEs[pe.Subnet], pe)
		}
	}

	// ── Locate firewall VNet (match private IP against address spaces) ───────
	fwVNet := ""
	if fixture.AzureFirewall != nil {
		for _, vnet := range rg.VirtualNetworks {
			if ipInVNet(vnet, fixture.AzureFirewall.PrivateIP) {
				fwVNet = vnet.Name
				break
			}
		}
	}

	// ── Track max grid row for placing out-of-grid elements ─────────────────
	maxRow := 0
	if n := len(rg.VirtualNetworks); n > 0 {
		maxRow = (n - 1) / 2
	}

	var sb strings.Builder

	// ── XML preamble ─────────────────────────────────────────────────────────
	sb.WriteString("<mxfile version=\"21.0.0\">\n")
	sb.WriteString("  <diagram name=\"Azure Network Topology\">\n")
	sb.WriteString("    <mxGraphModel dx=\"1422\" dy=\"762\" grid=\"1\" gridSize=\"10\" guides=\"1\"" +
		" tooltips=\"1\" connect=\"1\" arrows=\"1\" fold=\"1\" page=\"1\" pageScale=\"1\"" +
		" pageWidth=\"1654\" pageHeight=\"1169\" math=\"0\" shadow=\"0\">\n")
	sb.WriteString("      <root>\n")
	sb.WriteString("        <mxCell id=\"0\"/>\n")
	sb.WriteString("        <mxCell id=\"1\" parent=\"0\"/>\n")

	// ── VNet swimlanes ────────────────────────────────────────────────────────
	for vi, vnet := range rg.VirtualNetworks {
		col := vi % 2
		row := vi / 2
		vnetX := col*560 + 20
		vnetY := row*400 + 60

		subnetCount := len(vnet.Subnets)
		vnetHeight := 40 + 90*subnetCount
		if vnetHeight < 200 {
			vnetHeight = 200
		}
		// Reserve space for in-VNet firewall.
		if fixture.AzureFirewall != nil && fwVNet == vnet.Name {
			vnetHeight += 70
		}

		vnetSlug := slugify(vnet.Name)
		vnetID := "vnet-" + vnetSlug

		vnetLabelParts := []string{vnet.Name}
		if len(vnet.AddressSpace) > 0 {
			vnetLabelParts = append(vnetLabelParts, "("+strings.Join(vnet.AddressSpace, ", ")+")")
		}
		vnetLabel := drawioLabel(vnetLabelParts...)
		vnetStyle := "swimlane;startSize=30;fillColor=#dae8fc;strokeColor=#6c8ebf;fontStyle=1;fontSize=12;"
		fmt.Fprintf(&sb,
			"        <mxCell id=%q value=%q style=%q vertex=\"1\" parent=\"1\">\n"+
				"          <mxGeometry x=\"%d\" y=\"%d\" width=\"500\" height=\"%d\" as=\"geometry\"/>\n"+
				"        </mxCell>\n",
			vnetID, vnetLabel, vnetStyle, vnetX, vnetY, vnetHeight,
		)

		// Firewall placed inside this VNet.
		if fixture.AzureFirewall != nil && fwVNet == vnet.Name {
			fw := fixture.AzureFirewall
			fwID := "fw-" + slugify(fw.Name)
			fwLabel := drawioLabel("🔥 Firewall", fw.Name)
			fwStyle := "rounded=0;whiteSpace=wrap;html=1;fillColor=#f0a30a;strokeColor=#BD7000;fontStyle=1;"
			writeVertex(&sb, fwID, fwLabel, fwStyle, vnetID, 20, 60, 120, 50)
		}

		// Subnet swimlanes.
		for si, sub := range vnet.Subnets {
			subKey := vnet.Name + "/" + sub.Name
			subSlug := slugify(sub.Name)
			subID := "subnet-" + vnetSlug + "-" + subSlug

			subLabelParts := []string{sub.Name}
			if sub.AddressPrefix != "" {
				subLabelParts = append(subLabelParts, "("+sub.AddressPrefix+")")
			}
			subLabel := drawioLabel(subLabelParts...)
			subStyle := "swimlane;startSize=20;fillColor=#f5f5f5;strokeColor=#666666;fontSize=11;"
			subY := 30 + si*90
			fmt.Fprintf(&sb,
				"        <mxCell id=%q value=%q style=%q vertex=\"1\" parent=%q>\n"+
					"          <mxGeometry x=\"20\" y=\"%d\" width=\"460\" height=\"80\" as=\"geometry\"/>\n"+
					"        </mxCell>\n",
				subID, subLabel, subStyle, vnetID, subY,
			)

			// NIC ellipses inside this subnet.
			nicsInSub := subnetNICs[subKey]
			for j, nic := range nicsInSub {
				nicID := "nic-" + slugify(nic.Name)
				nicSev := highestSeverity(nicFindings[nic.Name])
				labelParts := []string{nic.Name, nic.PrivateIP}
				if nicSev != "Clean" {
					labelParts = append(labelParts, "["+nicSev+"]")
				}
				nicLabel := drawioLabel(labelParts...)
				nicX := 10 + (j%3)*160
				nicY := 25 + (j/3)*50
				writeVertex(&sb, nicID, nicLabel, nicStyle(nicSev), subID, nicX, nicY, 140, 40)
			}

			// Private Endpoint rhombuses inside this subnet.
			pesInSub := subnetPEs[subKey]
			for k, pe := range pesInSub {
				peID := "pe-" + slugify(pe.Name)
				peLabel := drawioLabel("PE", pe.Name)
				peStyle := "rhombus;whiteSpace=wrap;html=1;fillColor=#e1d5e7;strokeColor=#9673a6;"
				peX := 290 + k*130 // right side of subnet, after NICs
				writeVertex(&sb, peID, peLabel, peStyle, subID, peX, 15, 120, 50)
			}
		}
	}

	// ── Unplaced NICs container ───────────────────────────────────────────────
	if len(unplacedNICs) > 0 {
		unplacedRows := (len(unplacedNICs) + 2) / 3
		unplacedHeight := 60 + unplacedRows*50
		if unplacedHeight < 100 {
			unplacedHeight = 100
		}
		unplacedY := (maxRow+1)*400 + 100
		unplacedStyle := "swimlane;startSize=30;fillColor=#ffe6cc;strokeColor=#d6b656;fontStyle=1;"
		fmt.Fprintf(&sb,
			"        <mxCell id=\"unplaced-nics\" value=\"Unplaced NICs\""+
				" style=%q vertex=\"1\" parent=\"1\">\n"+
				"          <mxGeometry x=\"20\" y=\"%d\" width=\"500\" height=\"%d\" as=\"geometry\"/>\n"+
				"        </mxCell>\n",
			unplacedStyle, unplacedY, unplacedHeight,
		)
		for j, nic := range unplacedNICs {
			nicID := "nic-" + slugify(nic.Name)
			nicSev := highestSeverity(nicFindings[nic.Name])
			labelParts := []string{nic.Name, nic.PrivateIP}
			if nicSev != "Clean" {
				labelParts = append(labelParts, "["+nicSev+"]")
			}
			nicLabel := drawioLabel(labelParts...)
			nicX := 10 + (j%3)*160
			nicY := 35 + (j/3)*50
			writeVertex(&sb, nicID, nicLabel, nicStyle(nicSev), "unplaced-nics", nicX, nicY, 140, 40)
		}
	}

	// ── Standalone firewall (no matching VNet) ───────────────────────────────
	if fixture.AzureFirewall != nil && fwVNet == "" {
		fw := fixture.AzureFirewall
		fwID := "fw-" + slugify(fw.Name)
		fwLabel := drawioLabel("🔥 Firewall", fw.Name)
		fwStyle := "rounded=0;whiteSpace=wrap;html=1;fillColor=#f0a30a;strokeColor=#BD7000;fontStyle=1;"
		fwY := (maxRow+1)*400 + 260
		writeVertex(&sb, fwID, fwLabel, fwStyle, "1", 20, fwY, 120, 50)
	}

	// ── Load Balancers (row below VNet grid) ─────────────────────────────────
	if len(rg.LoadBalancers) > 0 {
		lbY := (maxRow+1)*400 + 60 + 20
		for li, lb := range rg.LoadBalancers {
			lbID := "lb-" + slugify(lb.Name)
			lbLabel := xmlAttr("LB: " + lb.Name)
			lbStyle := "rounded=1;whiteSpace=wrap;html=1;fillColor=#dae8fc;strokeColor=#6c8ebf;"
			lbX := 20 + li*140
			writeVertex(&sb, lbID, lbLabel, lbStyle, "1", lbX, lbY, 120, 40)
		}
	}

	// ── VNet peering edges ────────────────────────────────────────────────────
	peerSeen := make(map[string]bool)
	for _, vnet := range rg.VirtualNetworks {
		srcSlug := slugify(vnet.Name)
		srcID := "vnet-" + srcSlug
		for _, peer := range vnet.Peerings {
			tgtSlug := slugify(peer.RemoteVnet)
			// Deduplicate bidirectional peerings.
			key := srcSlug + "--" + tgtSlug
			rev := tgtSlug + "--" + srcSlug
			if peerSeen[key] || peerSeen[rev] {
				continue
			}
			peerSeen[key] = true

			edgeID := "peer-" + srcSlug + "-" + tgtSlug
			tgtID := "vnet-" + tgtSlug
			edgeStyle := "edgeStyle=orthogonalEdgeStyle;"
			if !peer.AllowForwardedTraffic {
				edgeStyle += "dashed=1;"
			}
			writeEdge(&sb, edgeID, xmlAttr(peer.State), edgeStyle, srcID, tgtID)
		}
	}

	// ── Legend ────────────────────────────────────────────────────────────────
	legendValue := "&lt;b&gt;Legend&lt;/b&gt;&lt;br&gt;" +
		"🔴 Critical&lt;br&gt;" +
		"🟠 High&lt;br&gt;" +
		"🟡 Medium&lt;br&gt;" +
		"🔵 Info&lt;br&gt;" +
		"🟢 Clean"
	legendStyle := "rounded=1;whiteSpace=wrap;html=1;" +
		"fillColor=#ffffff;strokeColor=#666666;fontSize=11;align=left;spacingLeft=5;"
	fmt.Fprintf(&sb,
		"        <mxCell id=\"legend\" value=%q style=%q vertex=\"1\" parent=\"1\">\n"+
			"          <mxGeometry x=\"1200\" y=\"10\" width=\"180\" height=\"130\" as=\"geometry\"/>\n"+
			"        </mxCell>\n",
		legendValue, legendStyle,
	)

	// ── XML closing ───────────────────────────────────────────────────────────
	sb.WriteString("      </root>\n")
	sb.WriteString("    </mxGraphModel>\n")
	sb.WriteString("  </diagram>\n")
	sb.WriteString("</mxfile>\n")

	return sb.String()
}
