// Sustainment scheduler (P4.2, contracts/20 §7, gate-b-behavior/06 §8):
// post-Realized value must keep proving itself. Each Realized use case gets
// a check per reporting window; a missing or stale snapshot counts as a
// failed check; two consecutive failures regress the value claim (GE-006)
// by generating a NEW Regressed assessment — never by editing history.
package assessment

import (
	"time"

	"github.com/aaraminds/vria/internal/enums"
	"github.com/aaraminds/vria/internal/hypothesis"
	"github.com/aaraminds/vria/internal/scoring"
)

// DefaultReportingWindow is the default metric reporting window
// (gate-b-behavior/06 §8): 30 days.
const DefaultReportingWindow = 30 * 24 * time.Hour

// SchedulerActor is the audited actor for scheduler-generated assessments.
const SchedulerActor = "vria-sustainment-scheduler"

// Notifier is the owner-notification callback hook. It fires on AtRisk
// (first failed check — state stays Realized) and on Regressed.
type Notifier func(useCaseID, valueOwner string, status enums.SustainmentStatus)

// CheckOutcome reports one executed sustainment check.
type CheckOutcome struct {
	UseCaseID       string                  `json:"use_case_id"`
	Status          enums.SustainmentStatus `json:"sustainment_status"`
	CheckFailed     bool                    `json:"check_failed"`
	NewAssessmentID string                  `json:"new_assessment_id,omitempty"`
	NextCheckAt     time.Time               `json:"next_check_at"`
}

// Scheduler runs due sustainment checks against the assessment service.
// It holds no goroutines: callers drive it with RunDue(now), which keeps
// the whole flow deterministic and testable.
type Scheduler struct {
	svc     *Service
	window  time.Duration
	notify  Notifier
	nextDue map[string]time.Time
}

func NewScheduler(svc *Service, window time.Duration, notify Notifier) *Scheduler {
	if window <= 0 {
		window = DefaultReportingWindow
	}
	if notify == nil {
		notify = func(string, string, enums.SustainmentStatus) {}
	}
	return &Scheduler{svc: svc, window: window, nextDue: map[string]time.Time{}, notify: notify}
}

// RunDue executes every due check. For each use case whose latest
// assessment is Realized:
//
//  1. pull the current SustainmentCheck from the MetricProvider — no
//     result means the snapshot is missing, which counts as failed
//     (contracts/20 §7);
//  2. append it to the append-only check history and fold the history via
//     scoring.EvaluateSustainment;
//  3. schedule the next check at now + reporting window;
//  4. notify the value owner on AtRisk or Regressed;
//  5. on Regressed, generate a NEW assessment — the scoring engine maps
//     the regressed sustainment status to ValueState Regressed and
//     recommendation Fix. Prior assessments are never mutated.
func (sch *Scheduler) RunDue(now time.Time) []CheckOutcome {
	var out []CheckOutcome
	for _, uc := range sch.svc.useCases() {
		latest, ok := sch.svc.Latest(uc)
		if !ok || latest.ValueState != enums.Realized {
			continue
		}
		if due, seen := sch.nextDue[uc]; seen && now.Before(due) {
			continue
		}
		h, _, err := sch.svc.hyp.Get(uc)
		if err != nil {
			h = hypothesis.Hypothesis{UseCaseID: uc}
		}
		chk, ok := sch.svc.provider.SustainmentCheck(uc, h.PrimaryMetricID)
		if !ok {
			// Missing snapshot = failed check, never a skipped one.
			chk = scoring.SustainmentCheck{}
		}
		status := sch.svc.appendCheck(uc, chk)
		sch.nextDue[uc] = now.Add(sch.window)
		oc := CheckOutcome{
			UseCaseID:   uc,
			Status:      status,
			CheckFailed: chk.Failed(),
			NextCheckAt: sch.nextDue[uc],
		}
		if status == enums.SustainAtRisk || status == enums.SustainRegressed {
			sch.notify(uc, h.ValueOwner, status)
		}
		if status == enums.SustainRegressed {
			if a, err := sch.svc.GenerateAssessment(uc, SchedulerActor); err == nil {
				oc.NewAssessmentID = a.AssessmentID
			}
		}
		out = append(out, oc)
	}
	return out
}

// NextCheckAt reports when the next check for a use case is due; zero time
// means no check has run yet (due immediately).
func (sch *Scheduler) NextCheckAt(useCaseID string) time.Time {
	return sch.nextDue[useCaseID]
}
