package scoring

import "github.com/aaraminds/vria/internal/enums"

// SustainmentCheck is one scheduled check (one per metric reporting_window,
// gate-b-behavior/06 §8). A missing or stale snapshot counts as failed
// (contracts/20 §7).
type SustainmentCheck struct {
	MeasuredBenefit *float64 // nil = snapshot missing
	TargetValue     float64
	Threshold       float64 // fraction of target; 0 means default
	SnapshotStale   bool
}

// DefaultSustainmentThreshold is 80% of target (contracts/20 §7).
const DefaultSustainmentThreshold = 0.80

func (c SustainmentCheck) Failed() bool {
	if c.MeasuredBenefit == nil || c.SnapshotStale {
		return true
	}
	th := c.Threshold
	if th == 0 {
		th = DefaultSustainmentThreshold
	}
	if c.TargetValue == 0 {
		// No target to sustain against: a present measurement passes.
		return false
	}
	// Fail only when the ratio is clearly below threshold. The epsilon
	// absorbs float rounding so a measurement exactly at the threshold (e.g.
	// 0.3 against 10% of 3) is not spuriously failed.
	const epsilon = 1e-9
	return *c.MeasuredBenefit/c.TargetValue+epsilon < th
}

// EvaluateSustainment folds the post-Realized check history into a status:
// first failed check → AtRisk (state stays Realized, owner notified);
// two consecutive failures → Regressed (contracts/20 §7, GE-006).
//
// The status reflects the TRAILING run of checks, not the whole history: a
// passing check after a regression clears it (recovery), so Regressed is not
// a permanent latch. The fold does not early-return for this reason.
func EvaluateSustainment(history []SustainmentCheck) enums.SustainmentStatus {
	if len(history) == 0 {
		return enums.SustainNotStarted
	}
	consecutive := 0
	regressed := false
	for _, c := range history {
		if c.Failed() {
			consecutive++
			if consecutive >= 2 {
				regressed = true
			}
		} else {
			consecutive = 0
			regressed = false // a passing check recovers the claim
		}
	}
	switch {
	case regressed:
		return enums.SustainRegressed
	case consecutive == 1:
		return enums.SustainAtRisk
	default:
		return enums.SustainOk
	}
}
