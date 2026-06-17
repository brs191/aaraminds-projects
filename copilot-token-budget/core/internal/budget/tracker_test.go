package budget

import (
	"math"
	"testing"
)

func TestFromNanoAIU(t *testing.T) {
	cases := []struct {
		nanoAIU int64
		want    float64
	}{
		{0, 0},
		{1_000_000_000, 1.0},
		{500_000_000_000, 500.0},
		{656_539_080_000, 656.53908}, // real sample from Phase 0
	}
	for _, c := range cases {
		got := FromNanoAIU(c.nanoAIU)
		if math.Abs(got-c.want) > 0.0001 {
			t.Errorf("FromNanoAIU(%d) = %.5f, want %.5f", c.nanoAIU, got, c.want)
		}
	}
}

func TestToDollars(t *testing.T) {
	cases := []struct {
		credits float64
		want    float64
	}{
		{0, 0},
		{1.0, 0.01},
		{100.0, 1.00},
		{7000.0, 70.00},
	}
	for _, c := range cases {
		got := ToDollars(c.credits)
		if math.Abs(got-c.want) > 0.0001 {
			t.Errorf("ToDollars(%.2f) = %.4f, want %.4f", c.credits, got, c.want)
		}
	}
}

func TestCalculate_StatusThresholds(t *testing.T) {
	cases := []struct {
		desc       string
		nanoValues []int64
		allowance  int
		wantStatus BudgetStatus
		wantPctMin float64
		wantPctMax float64
	}{
		{
			desc:       "OK — below 60%",
			nanoValues: []int64{3_000_000_000_000}, // 3000 credits
			allowance:  7000,
			wantStatus: StatusOK,
			wantPctMin: 42,
			wantPctMax: 43,
		},
		{
			desc:       "WARNING — exactly 60%",
			nanoValues: []int64{4_200_000_000_000}, // 4200 credits = 60%
			allowance:  7000,
			wantStatus: StatusWarning,
			wantPctMin: 59.9,
			wantPctMax: 60.1,
		},
		{
			desc:       "WARNING — 89%",
			nanoValues: []int64{6_230_000_000_000}, // 6230 credits ≈ 89%
			allowance:  7000,
			wantStatus: StatusWarning,
			wantPctMin: 88,
			wantPctMax: 90,
		},
		{
			desc:       "CRITICAL — over 90%",
			nanoValues: []int64{14_144_656_785_000}, // real June 2026 data
			allowance:  7000,
			wantStatus: StatusCritical,
			wantPctMin: 200,
			wantPctMax: 203,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			state := Calculate(c.nanoValues, c.allowance)
			if state.Status != c.wantStatus {
				t.Errorf("Status = %q, want %q (pct=%.2f%%)", state.Status, c.wantStatus, state.UsedPct)
			}
			if state.UsedPct < c.wantPctMin || state.UsedPct > c.wantPctMax {
				t.Errorf("UsedPct = %.2f, want [%.1f, %.1f]", state.UsedPct, c.wantPctMin, c.wantPctMax)
			}
		})
	}
}

func TestCalculate_ZeroAllowanceFallback(t *testing.T) {
	state := Calculate([]int64{1_000_000_000_000}, 0)
	if state.AllowedCredits != MonthlyAllowanceCredits {
		t.Errorf("AllowedCredits = %d, want %d (default fallback)", state.AllowedCredits, MonthlyAllowanceCredits)
	}
}

func TestCalculate_NegativeAllowanceFallback(t *testing.T) {
	state := Calculate([]int64{1_000_000_000_000}, -1)
	if state.AllowedCredits != MonthlyAllowanceCredits {
		t.Errorf("AllowedCredits = %d, want %d (negative fallback)", state.AllowedCredits, MonthlyAllowanceCredits)
	}
}

func TestCalculate_EmptyInput(t *testing.T) {
	state := Calculate(nil, 7000)
	if state.UsedCredits != 0 {
		t.Errorf("UsedCredits = %.2f, want 0 for empty input", state.UsedCredits)
	}
	if state.Status != StatusOK {
		t.Errorf("Status = %q, want OK for zero usage", state.Status)
	}
	if state.RemainingCredit != 7000 {
		t.Errorf("RemainingCredit = %.2f, want 7000", state.RemainingCredit)
	}
}

func TestCalculate_MultipleValues(t *testing.T) {
	// Sum of three sessions: 500 + 200 + 300 = 1000 credits
	values := []int64{
		500_000_000_000,
		200_000_000_000,
		300_000_000_000,
	}
	state := Calculate(values, 7000)
	if math.Abs(state.UsedCredits-1000) > 0.001 {
		t.Errorf("UsedCredits = %.3f, want 1000", state.UsedCredits)
	}
	if math.Abs(state.RemainingCredit-6000) > 0.001 {
		t.Errorf("RemainingCredit = %.3f, want 6000", state.RemainingCredit)
	}
}

func TestEstimateInstructionCostPerSession(t *testing.T) {
	// 12,000 systemTokens (typical from Phase 0 data)
	credits, dollars := EstimateInstructionCostPerSession(12_000)

	// Expected: (12000 * 50 * 300) / 1_000_000 = 180_000_000 / 1_000_000 = 180 credits
	wantCredits := 180.0
	wantDollars := 1.80

	if math.Abs(credits-wantCredits) > 0.001 {
		t.Errorf("credits = %.3f, want %.3f", credits, wantCredits)
	}
	if math.Abs(dollars-wantDollars) > 0.001 {
		t.Errorf("dollars = %.3f, want %.3f", dollars, wantDollars)
	}
}

func TestEstimateInstructionCostPerSession_Zero(t *testing.T) {
	credits, dollars := EstimateInstructionCostPerSession(0)
	if credits != 0 || dollars != 0 {
		t.Errorf("expected 0,0 for zero tokens; got %.3f, %.3f", credits, dollars)
	}
}

func TestNoPanicOnLargeValues(t *testing.T) {
	// Ensure no overflow panic on very large nanoAIU accumulations
	large := []int64{
		math.MaxInt64 / 4,
		math.MaxInt64 / 4,
	}
	// Just must not panic
	state := Calculate(large, 7000)
	if state.Status != StatusCritical {
		t.Errorf("expected CRITICAL for very large values, got %q", state.Status)
	}
}
