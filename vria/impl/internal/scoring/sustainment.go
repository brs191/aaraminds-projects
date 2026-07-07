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
	return *c.MeasuredBenefit < th*c.TargetValue
}

// EvaluateSustainment folds the post-Realized check history into a status:
// first failed check → AtRisk (state stays Realized, owner notified);
// two consecutive failures → Regressed (contracts/20 §7, GE-006).
func EvaluateSustainment(history []SustainmentCheck) enums.SustainmentStatus {
	if len(history) == 0 {
		return enums.SustainNotStarted
	}
	consecutive := 0
	status := enums.SustainOk
	for _, c := range history {
		if c.Failed() {
			consecutive++
			if consecutive >= 2 {
				return enums.SustainRegressed
			}
			status = enums.SustainAtRisk
		} else {
			consecutive = 0
			status = enums.SustainOk
		}
	}
	return status
}
