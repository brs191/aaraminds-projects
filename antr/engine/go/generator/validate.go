package generator

import (
	"github.com/aaraminds/azure-nettopo-engine/internal/analyze"
)

// ValidationResult is the output of ValidateBeforeEmit.
// PR creation MUST receive a ValidationResult, not a raw bool.
// This enforces the §4.5 anti-bypass design: no callers can thread a plain bool.
type ValidationResult struct {
	Findings []analyze.Finding
	Approved bool // true iff no Critical, High, OR Medium findings (generated infra: stricter bar, audit H-3)
}

// ValidateBeforeEmit runs Analyze() on plan.FixtureProjection and returns a
// ValidationResult. NO bypass parameters. NO "force" flag. NO "dry_run". See §4.5.
//
// Gate logic:
//
//	Approved = len(findings where severity ∈ {"Critical","High","Medium"}) == 0  (generated infra)
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
		// Generated infrastructure is held to a STRICTER bar than a review of an
		// existing estate: we authored it, so it must not auto-PR with any
		// Critical, High, OR Medium security finding — Medium includes
		// WAF-disabled App Gateway / Front Door, public (non-private) AKS,
		// cross-sub peering without a firewall, vWAN unsecured, and CIDR overlap
		// (audit H-3). Low / Informational (latent, hygiene) do not block.
		if f.Severity == "Critical" || f.Severity == "High" || f.Severity == "Medium" {
			approved = false
			break
		}
	}

	return ValidationResult{
		Findings: findings,
		Approved: approved,
	}
}
