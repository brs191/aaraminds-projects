package simulator

import (
	"github.com/aaraminds/azure-nettopo-engine/internal/analyze"
)

// SecurityDelta is the result of comparing Analyze(original) with Analyze(simulated).
// It makes the security impact of a TopologyDelta explicit and actionable.
type SecurityDelta struct {
	// AddedRisks contains findings present in the simulated topology but not
	// the original — new risks introduced by the change.
	AddedRisks []analyze.Finding `json:"addedRisks"`

	// MitigatedRisks contains findings present in the original topology but not
	// the simulated — risks resolved by the change.
	MitigatedRisks []analyze.Finding `json:"mitigatedRisks"`

	// Unchanged contains findings present in both. Populated only when
	// includeUnchanged is true in DiffFindings.
	Unchanged []analyze.Finding `json:"unchanged,omitempty"`

	// OriginalFindingCount is the total findings before the change.
	OriginalFindingCount int `json:"originalFindingCount"`

	// SimulatedFindingCount is the total findings after the change.
	SimulatedFindingCount int `json:"simulatedFindingCount"`

	// RiskVector summarises net changes in severity counts.
	// Positive = more findings at that severity after the change (risk increase).
	// Negative = fewer findings (risk reduction).
	RiskVector RiskVector `json:"riskVector"`
}

// RiskVector captures net changes in finding severity counts.
type RiskVector struct {
	CriticalDelta      int `json:"criticalDelta"`
	HighDelta          int `json:"highDelta"`
	MediumDelta        int `json:"mediumDelta"`
	LowDelta           int `json:"lowDelta"`
	InformationalDelta int `json:"informationalDelta"`
}

// DiffFindings computes the SecurityDelta between an original and simulated
// findings set. Equality key is Type + "|" + Resource + "|" + Severity —
// Evidence is excluded because Evidence strings are dynamically constructed
// and can differ between two analyses of identical topologies, which would
// produce spurious added/mitigated pairs (SR-004 from the rubber-duck review).
//
// A severity escalation (e.g. High → Critical for the same resource) correctly
// appears as one MitigatedRisk (original) + one AddedRisk (new severity).
func DiffFindings(original, simulated []analyze.Finding) SecurityDelta {
	origMap := toFindingMap(original)
	simMap := toFindingMap(simulated)

	var added, mitigated, unchanged []analyze.Finding

	for k, f := range simMap {
		if _, exists := origMap[k]; exists {
			unchanged = append(unchanged, f)
		} else {
			added = append(added, f)
		}
	}
	for k, f := range origMap {
		if _, exists := simMap[k]; !exists {
			mitigated = append(mitigated, f)
		}
	}

	rv := buildRiskVector(added, mitigated)

	return SecurityDelta{
		AddedRisks:            added,
		MitigatedRisks:        mitigated,
		Unchanged:             unchanged,
		OriginalFindingCount:  len(original),
		SimulatedFindingCount: len(simulated),
		RiskVector:            rv,
	}
}

// findingKey returns the equality key for diffing.
func findingKey(f analyze.Finding) string {
	return f.Type + "|" + f.Resource + "|" + f.Severity
}

func toFindingMap(findings []analyze.Finding) map[string]analyze.Finding {
	m := make(map[string]analyze.Finding, len(findings))
	for _, f := range findings {
		m[findingKey(f)] = f
	}
	return m
}

func buildRiskVector(added, mitigated []analyze.Finding) RiskVector {
	var rv RiskVector
	for _, f := range added {
		incSeverity(&rv, f.Severity, +1)
	}
	for _, f := range mitigated {
		incSeverity(&rv, f.Severity, -1)
	}
	return rv
}

func incSeverity(rv *RiskVector, sev string, delta int) {
	switch sev {
	case "Critical":
		rv.CriticalDelta += delta
	case "High":
		rv.HighDelta += delta
	case "Medium":
		rv.MediumDelta += delta
	case "Low":
		rv.LowDelta += delta
	default:
		rv.InformationalDelta += delta
	}
}
