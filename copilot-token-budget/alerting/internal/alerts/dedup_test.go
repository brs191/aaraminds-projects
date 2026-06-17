package alerts

import (
	"testing"
	"time"
)

func TestShouldAlert(t *testing.T) {
	tests := []struct {
		name      string
		state     alertState
		threshold int
		now       time.Time
		want      bool
	}{
		{
			name:      "no prior alerts — should fire",
			state:     alertState{ThresholdAlerts: map[string]string{}},
			threshold: 60,
			now:       time.Date(2026, 6, 14, 10, 0, 0, 0, time.UTC),
			want:      true,
		},
		{
			name:      "alerted today — should skip",
			state:     alertState{ThresholdAlerts: map[string]string{"60": "2026-06-14"}},
			threshold: 60,
			now:       time.Date(2026, 6, 14, 15, 30, 0, 0, time.UTC),
			want:      false,
		},
		{
			name:      "alerted yesterday — should re-fire today",
			state:     alertState{ThresholdAlerts: map[string]string{"60": "2026-06-13"}},
			threshold: 60,
			now:       time.Date(2026, 6, 14, 10, 0, 0, 0, time.UTC),
			want:      true,
		},
		{
			name:      "different threshold alerted today — should fire for new threshold",
			state:     alertState{ThresholdAlerts: map[string]string{"90": "2026-06-14"}},
			threshold: 60,
			now:       time.Date(2026, 6, 14, 10, 0, 0, 0, time.UTC),
			want:      true,
		},
		{
			name:      "nil threshold map — should fire",
			state:     alertState{ThresholdAlerts: nil},
			threshold: 60,
			now:       time.Date(2026, 6, 14, 10, 0, 0, 0, time.UTC),
			want:      true,
		},
		{
			name:      "critical threshold alerted today — should skip",
			state:     alertState{ThresholdAlerts: map[string]string{"90": "2026-06-14"}},
			threshold: 90,
			now:       time.Date(2026, 6, 14, 23, 59, 59, 0, time.UTC),
			want:      false,
		},
		{
			name:      "year boundary — new year fires again",
			state:     alertState{ThresholdAlerts: map[string]string{"60": "2025-12-31"}},
			threshold: 60,
			now:       time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC),
			want:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldAlert(tc.state, tc.threshold, tc.now)
			if got != tc.want {
				t.Errorf("shouldAlert(threshold=%d, now=%s) = %v, want %v",
					tc.threshold, tc.now.UTC().Format("2006-01-02"), got, tc.want)
			}
		})
	}
}

// TestShouldAlertUTCDayStableAcrossTZ proves a single fixed instant near midnight
// maps to the same UTC dedup day regardless of the time.Time's location. Without UTC
// normalisation, a process running in a +HH zone would see a different local date and
// could re-fire (or miss) an alert across a TZ change between runs.
func TestShouldAlertUTCDayStableAcrossTZ(t *testing.T) {
	// 2026-06-14 23:30 UTC is the same instant as 2026-06-15 05:00 in IST (+05:30).
	// In UTC the dedup day is 2026-06-14 either way.
	instantUTC := time.Date(2026, 6, 14, 23, 30, 0, 0, time.UTC)
	ist := time.FixedZone("IST", 5*3600+30*60)
	instantIST := instantUTC.In(ist) // same instant, local date is 2026-06-15

	if instantIST.Format("2006-01-02") == instantIST.UTC().Format("2006-01-02") {
		t.Fatal("test precondition broken: local and UTC dates should differ near midnight")
	}

	state := alertState{ThresholdAlerts: map[string]string{"60": "2026-06-14"}}

	// Already alerted on UTC day 2026-06-14 — must skip from BOTH representations.
	if shouldAlert(state, 60, instantUTC) {
		t.Error("should skip: already alerted on UTC day 2026-06-14 (UTC instant)")
	}
	if shouldAlert(state, 60, instantIST) {
		t.Error("should skip: same instant in IST must resolve to the same UTC day")
	}
}
