// Package renderer produces human-readable and diagram output from analysis results.
// ToMarkdown renders a structured Markdown report; ToDrawIO renders valid mxGraph XML
// for import into draw.io. Both are deterministic and depend only on stdlib +
// internal packages — no external dependencies.
package renderer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aaraminds/azure-nettopo-engine/internal/analyze"
)

// sevOrder returns a numeric sort key for a severity (lower = higher priority).
func sevOrder(sev string) int {
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
		return 4
	}
}

// sevEmoji returns the emoji for a severity level.
func sevEmoji(sev string) string {
	switch sev {
	case "Critical":
		return "🔴"
	case "High":
		return "🟠"
	case "Medium":
		return "🟡"
	case "Informational":
		return "🔵"
	default:
		return "⚪"
	}
}

// ToMarkdown renders a structured Markdown report from analysis findings.
// Findings are sorted Critical → High → Medium → Informational; within the same
// severity they are ordered alphabetically by resource name.
func ToMarkdown(sub string, findings []analyze.Finding) string {
	// Sort a copy — do not mutate the caller's slice.
	sorted := make([]analyze.Finding, len(findings))
	copy(sorted, findings)
	sort.SliceStable(sorted, func(i, j int) bool {
		oi, oj := sevOrder(sorted[i].Severity), sevOrder(sorted[j].Severity)
		if oi != oj {
			return oi < oj
		}
		return sorted[i].Resource < sorted[j].Resource
	})

	// Count by severity.
	counts := map[string]int{
		"Critical":      0,
		"High":          0,
		"Medium":        0,
		"Informational": 0,
	}
	for _, f := range findings {
		if _, ok := counts[f.Severity]; ok {
			counts[f.Severity]++
		}
	}

	var sb strings.Builder

	// ── Header ──────────────────────────────────────────────────────────────
	fmt.Fprintf(&sb, "# Azure Network Topology Analysis — %s\n", sub)
	// Deterministic by design: the report must be byte-reproducible for the same
	// fixture (no wall-clock). A run timestamp, if wanted, is added by the caller
	// at the non-deterministic MCP boundary. (Adversarial review HIGH-1.)
	fmt.Fprintf(&sb, "Generated: deterministic render\n\n")

	// ── Summary table ────────────────────────────────────────────────────────
	sb.WriteString("## Summary\n")
	sb.WriteString("| Severity | Count |\n")
	sb.WriteString("|---|---|\n")
	fmt.Fprintf(&sb, "| 🔴 Critical | %d |\n", counts["Critical"])
	fmt.Fprintf(&sb, "| 🟠 High | %d |\n", counts["High"])
	fmt.Fprintf(&sb, "| 🟡 Medium | %d |\n", counts["Medium"])
	fmt.Fprintf(&sb, "| 🔵 Informational | %d |\n\n", counts["Informational"])

	// ── Findings ─────────────────────────────────────────────────────────────
	sb.WriteString("## Findings\n\n")
	if len(sorted) == 0 {
		sb.WriteString("_No findings._\n\n")
	}
	for _, f := range sorted {
		reachStr := "no"
		if f.Reachable {
			reachStr = "yes"
		}
		fmt.Fprintf(&sb, "### %s %s — %s\n",
			sevEmoji(f.Severity), strings.ToUpper(f.Severity), f.Resource)
		fmt.Fprintf(&sb, "**Type:** %s\n", f.Type)
		fmt.Fprintf(&sb, "**Evidence:** %s\n", f.Evidence)
		fmt.Fprintf(&sb, "**Reachable:** %s\n\n", reachStr)
	}

	// ── Recommendations ───────────────────────────────────────────────────────
	sb.WriteString("## Recommendations\n")
	sb.WriteString("- Review all Critical and High findings immediately.\n")
	sb.WriteString("- Run with `enrich=true` for Defender for Cloud correlation.\n")

	return sb.String()
}
