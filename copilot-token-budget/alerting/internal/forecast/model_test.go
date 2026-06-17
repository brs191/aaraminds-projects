package forecast

import (
	"testing"

	"github.com/aaraminds/copilot-session-manager/internal/session"
)

func TestDailyBurnRate(t *testing.T) {
	tests := []struct {
		name        string
		sessions    []session.Session
		daysElapsed int
		want        float64
	}{
		{
			name:        "zero daysElapsed guard — no division by zero",
			sessions:    []session.Session{{TotalNanoAIU: 1_000_000_000}},
			daysElapsed: 0,
			want:        0,
		},
		{
			name:        "negative daysElapsed guard",
			sessions:    []session.Session{{TotalNanoAIU: 1_000_000_000}},
			daysElapsed: -5,
			want:        0,
		},
		{
			name:        "nil sessions — zero burn",
			sessions:    nil,
			daysElapsed: 10,
			want:        0,
		},
		{
			name:        "1 credit over 1 day",
			sessions:    []session.Session{{TotalNanoAIU: 1_000_000_000}},
			daysElapsed: 1,
			want:        1.0,
		},
		{
			name:        "7000 credits over 14 days",
			sessions:    []session.Session{{TotalNanoAIU: 7_000 * 1_000_000_000}},
			daysElapsed: 14,
			want:        500.0,
		},
		{
			name: "multiple sessions summed",
			sessions: []session.Session{
				{TotalNanoAIU: 3_000_000_000},
				{TotalNanoAIU: 2_000_000_000},
			},
			daysElapsed: 5,
			want:        1.0, // 5 credits / 5 days
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := DailyBurnRate(tc.sessions, tc.daysElapsed)
			if got != tc.want {
				t.Errorf("DailyBurnRate() = %f, want %f", got, tc.want)
			}
		})
	}
}

func TestMonthEndForecast(t *testing.T) {
	tests := []struct {
		name          string
		dailyBurn     float64
		daysRemaining int
		want          float64
	}{
		{"zero days remaining — month already over", 100, 0, 0},
		{"negative days remaining guard", 100, -3, 0},
		{"zero burn rate", 0, 15, 0},
		{"10 days at 100 cr/day", 100, 10, 1000},
		{"half month at 500 cr/day", 500, 15, 7500},
		{"1 day left at 250 cr/day", 250, 1, 250},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := MonthEndForecast(tc.dailyBurn, tc.daysRemaining)
			if got != tc.want {
				t.Errorf("MonthEndForecast(%.0f, %d) = %f, want %f",
					tc.dailyBurn, tc.daysRemaining, got, tc.want)
			}
		})
	}
}

func TestProjectedMonthEndTotal(t *testing.T) {
	tests := []struct {
		name          string
		usedCredits   float64
		dailyBurn     float64
		daysRemaining int
		want          float64
	}{
		{"last day — returns used credits only", 6800, 250, 0, 6800},
		{"negative days remaining — returns used credits", 6800, 250, -3, 6800},
		{"mid-month projection", 3000, 200, 10, 5000},
		{"zero burn — stays at used", 4200, 0, 12, 4200},
		{"fresh month, no usage yet", 0, 500, 30, 15000},
		{"one day left", 6500, 250, 1, 6750},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ProjectedMonthEndTotal(tc.usedCredits, tc.dailyBurn, tc.daysRemaining)
			if got != tc.want {
				t.Errorf("ProjectedMonthEndTotal(%.0f, %.0f, %d) = %f, want %f",
					tc.usedCredits, tc.dailyBurn, tc.daysRemaining, got, tc.want)
			}
		})
	}
}

func TestExceedsAllowance(t *testing.T) {
	tests := []struct {
		name      string
		forecast  float64
		allowance float64
		want      bool
	}{
		{"well under", 5000, 7000, false},
		{"exactly at allowance", 7000, 7000, false},
		{"one credit over", 7001, 7000, true},
		{"far over", 10000, 7000, true},
		{"zero forecast", 0, 7000, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ExceedsAllowance(tc.forecast, tc.allowance)
			if got != tc.want {
				t.Errorf("ExceedsAllowance(%.0f, %.0f) = %v, want %v",
					tc.forecast, tc.allowance, got, tc.want)
			}
		})
	}
}

func TestModelRoutingRecommendationNilGuards(t *testing.T) {
	// nil sessions → nil recommendations
	got := ModelRoutingRecommendation(nil, 1.0)
	if got != nil {
		t.Errorf("expected nil for nil sessions, got %v", got)
	}

	// zero avgCostPerToken → nil recommendations
	got = ModelRoutingRecommendation([]session.Session{{TotalNanoAIU: 1e9}}, 0)
	if got != nil {
		t.Errorf("expected nil for zero avgCostPerToken, got %v", got)
	}
}
