package generator

import (
	"github.com/aaraminds/azure-nettopo-engine/internal/analyze"
)

// ValidationResult is the output of ValidateBeforeEmit.
// PR creation MUST receive a ValidationResult, not a raw bool.
// This enforces the §4.5 anti-bypass design: no callers can thread a plain bool.
type ValidationResult struct {
	Findings []analyze.Finding
	Approved bool // true iff no Critical or High findings
}

// ValidateBeforeEmit runs Analyze() on plan.FixtureProjection and returns a
// ValidationResult. NO bypass parameters. NO "force" flag. NO "dry_run". See §4.5.
//
// Gate logic:
//
//	Approved = len(findings where severity ∈ {"Critical","High"}) == 0
//
// If plan.FixtureProjection == nil, returns a synthetic High finding and Approved=false.
// PR creation must not be attempted when Approved==false.
func ValidateBeforeEmit(plan TerraformPlan) ValidationResult {
	if plan.FixtureProjection == nil {
		return ValidationResult{
			Findings: []analyze.Finding{{
				Type:      "projection-error",
				Severity:  "High",
				Resource:  "fixture",
				Evidence:  "nil fixture projection — rendering failed",
				Reachable: false,
			}},
			Approved: false,
		}
	}

	findings := analyze.Analyze(plan.FixtureProjection)

	approved := true
	for _, f := range findings {
		if f.Severity == "Critical" || f.Severity == "High" {
			approved = false
			break
		}
	}

	return ValidationResult{
		Findings: findings,
		Approved: approved,
	}
}
