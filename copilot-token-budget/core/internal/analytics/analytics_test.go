package analytics

import (
	"testing"
	"time"

	"github.com/aaraminds/copilot-token-budget/internal/budget"
	"github.com/aaraminds/copilot-token-budget/internal/pricing"
	"github.com/aaraminds/copilot-token-budget/internal/session"
)

// mkSession builds a finalized session whose BillingTime is end, with a single
// model metric so token/credit aggregation has something to sum.
func mkSession(id, project, model string, end time.Time, nano, in, out int64) session.Session {
	return session.Session{
		ID:           id,
		ProjectName:  project,
		PrimaryModel: model,
		StartTime:    end.Add(-time.Hour),
		EndTime:      end,
		IsFinal:      true,
		TotalNanoAIU: nano,
		Tokens:       session.TokenBreakdown{CurrentTokens: in},
		ModelMetrics: []session.ModelMetric{
			{Model: model, InputTokens: in, OutputTokens: out, NanoAIU: nano},
		},
	}
}

const credit = int64(1_000_000_000) // 1 credit in nanoAIU

func TestDailySeries(t *testing.T) {
	loc := time.UTC
	d1 := time.Date(2026, 6, 10, 9, 0, 0, 0, loc)
	d1b := time.Date(2026, 6, 10, 18, 0, 0, 0, loc)
	d2 := time.Date(2026, 6, 11, 9, 0, 0, 0, loc)

	sessions := []session.Session{
		mkSession("a", "proj", "sonnet", d1, 2*credit, 100, 10),
		mkSession("b", "proj", "sonnet", d1b, 3*credit, 200, 20),
		mkSession("c", "proj", "opus", d2, 5*credit, 300, 30),
	}

	got := DailySeries(sessions)
	if len(got) != 2 {
		t.Fatalf("got %d buckets, want 2", len(got))
	}
	if got[0].Key != "2026-06-10" || got[1].Key != "2026-06-11" {
		t.Fatalf("keys = %q,%q; want 2026-06-10,2026-06-11", got[0].Key, got[1].Key)
	}
	if got[0].Sessions != 2 {
		t.Errorf("day1 sessions = %d, want 2", got[0].Sessions)
	}
	if got[0].Credits != 5 {
		t.Errorf("day1 credits = %v, want 5", got[0].Credits)
	}
	if got[0].InputTokens != 300 || got[0].OutputTokens != 30 {
		t.Errorf("day1 tokens = %d/%d, want 300/30", got[0].InputTokens, got[0].OutputTokens)
	}
	if got[0].ByModel["sonnet"] != 5 {
		t.Errorf("day1 ByModel[sonnet] = %v, want 5", got[0].ByModel["sonnet"])
	}
}

func TestWeeklySeries(t *testing.T) {
	// 2026-06-10 is a Wednesday; ISO week 24. 2026-06-15 is the next Monday, week 25.
	wedW24 := time.Date(2026, 6, 10, 9, 0, 0, 0, time.UTC)
	monW25 := time.Date(2026, 6, 15, 9, 0, 0, 0, time.UTC)

	sessions := []session.Session{
		mkSession("a", "p", "sonnet", wedW24, credit, 10, 1),
		mkSession("b", "p", "sonnet", monW25, credit, 10, 1),
	}
	got := WeeklySeries(sessions)
	if len(got) != 2 {
		t.Fatalf("got %d weekly buckets, want 2", len(got))
	}
	if got[0].Key != "2026-W24" {
		t.Errorf("week1 key = %q, want 2026-W24", got[0].Key)
	}
	if got[1].Key != "2026-W25" {
		t.Errorf("week2 key = %q, want 2026-W25", got[1].Key)
	}
	// Start of W24 bucket is Monday 2026-06-08.
	if !got[0].Start.Equal(time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("week1 Start = %v, want 2026-06-08", got[0].Start)
	}
}

func TestMonthlySeries(t *testing.T) {
	may := time.Date(2026, 5, 20, 9, 0, 0, 0, time.UTC)
	jun := time.Date(2026, 6, 2, 9, 0, 0, 0, time.UTC)
	sessions := []session.Session{
		mkSession("a", "p", "sonnet", jun, 2*credit, 10, 1),
		mkSession("b", "p", "sonnet", may, 1*credit, 10, 1),
	}
	got := MonthlySeries(sessions)
	if len(got) != 2 {
		t.Fatalf("got %d monthly buckets, want 2", len(got))
	}
	if got[0].Key != "2026-05" || got[1].Key != "2026-06" {
		t.Fatalf("keys = %q,%q; want ascending 2026-05,2026-06", got[0].Key, got[1].Key)
	}
}

// TestNearMidnightBucketsUTC proves a session that settles at 23:30Z on the last day
// of June buckets to 2026-06-30 / 2026-W27... no — to the UTC day/week/month, NOT a
// local-time-shifted day, even when the host machine is in a non-UTC timezone. The Go
// side normalizes BillingTime to UTC in series(), so this is timezone-independent.
func TestNearMidnightBucketsUTC(t *testing.T) {
	// A fixed-offset zone east of UTC: 23:30Z is already 04:30 the NEXT local day,
	// so a naive local-time bucketer would land this in July. UTC bucketing must not.
	plus5 := time.FixedZone("UTC+5", 5*60*60)
	ts := time.Date(2026, 6, 30, 23, 30, 0, 0, time.UTC).In(plus5)

	s := mkSession("a", "p", "sonnet", ts, credit, 10, 1)

	daily := DailySeries([]session.Session{s})
	if len(daily) != 1 || daily[0].Key != "2026-06-30" {
		t.Fatalf("daily key = %v, want 2026-06-30 (UTC), not a local-shifted day", keysOf(daily))
	}
	monthly := MonthlySeries([]session.Session{s})
	if len(monthly) != 1 || monthly[0].Key != "2026-06" {
		t.Fatalf("monthly key = %v, want 2026-06 (UTC)", keysOf(monthly))
	}
	// 2026-06-30 is a Tuesday -> ISO week 27 of 2026.
	weekly := WeeklySeries([]session.Session{s})
	if len(weekly) != 1 || weekly[0].Key != "2026-W27" {
		t.Fatalf("weekly key = %v, want 2026-W27 (UTC)", keysOf(weekly))
	}
}

func keysOf(bs []Bucket) []string {
	out := make([]string, len(bs))
	for i, b := range bs {
		out[i] = b.Key
	}
	return out
}

func TestEmptySeries(t *testing.T) {
	if got := DailySeries(nil); len(got) != 0 {
		t.Errorf("DailySeries(nil) = %d buckets, want 0", len(got))
	}
}

func TestTopSessions(t *testing.T) {
	d := time.Date(2026, 6, 10, 9, 0, 0, 0, time.UTC)
	sessions := []session.Session{
		mkSession("id1", "alpha", "sonnet", d, 5*credit, 100, 10),
		mkSession("id2", "", "opus", d, 9*credit, 200, 20), // no project -> uses id
		mkSession("id3", "gamma", "haiku", d, 1*credit, 50, 5),
	}
	got := TopSessions(sessions, 2)
	if len(got) != 2 {
		t.Fatalf("got %d, want 2", len(got))
	}
	if got[0].Name != "id2" || got[0].Credits != 9 {
		t.Errorf("top = %+v, want id2/9", got[0])
	}
	if got[1].Name != "alpha" {
		t.Errorf("second = %q, want alpha", got[1].Name)
	}
}

func TestTopModels(t *testing.T) {
	d := time.Date(2026, 6, 10, 9, 0, 0, 0, time.UTC)
	sessions := []session.Session{
		mkSession("a", "p", "sonnet", d, 3*credit, 100, 10),
		mkSession("b", "p", "sonnet", d, 4*credit, 100, 10),
		mkSession("c", "p", "opus", d, 5*credit, 100, 10),
	}
	got := TopModels(sessions, 0) // 0 => all
	if len(got) != 2 {
		t.Fatalf("got %d models, want 2", len(got))
	}
	// sonnet aggregates to 7 > opus 5.
	if got[0].Name != "sonnet" || got[0].Credits != 7 {
		t.Errorf("top model = %+v, want sonnet/7", got[0])
	}
	if got[0].InputTokens != 200 {
		t.Errorf("sonnet inputTokens = %d, want 200", got[0].InputTokens)
	}
}

func TestTopProjects(t *testing.T) {
	d := time.Date(2026, 6, 10, 9, 0, 0, 0, time.UTC)
	sessions := []session.Session{
		mkSession("a", "alpha", "sonnet", d, 3*credit, 100, 10),
		mkSession("b", "alpha", "sonnet", d, 4*credit, 100, 10),
		mkSession("c", "beta", "opus", d, 5*credit, 100, 10),
	}
	got := TopProjects(sessions, 0)
	if len(got) != 2 {
		t.Fatalf("got %d projects, want 2", len(got))
	}
	if got[0].Name != "alpha" || got[0].Credits != 7 {
		t.Errorf("top project = %+v, want alpha/7", got[0])
	}
}

func TestContextWindowPct(t *testing.T) {
	cfg := pricing.Default()
	s := mkSession("a", "p", "claude-sonnet-4.6", time.Now(), credit, 0, 0)
	s.Tokens.CurrentTokens = 100000 // half of 200000
	if got := ContextWindowPct(s, cfg); got != 50 {
		t.Errorf("ContextWindowPct = %v, want 50", got)
	}

	// Zero/unknown window guard.
	zeroCfg := pricing.Config{Default: pricing.ModelRate{ContextWindowTokens: 0}}
	if got := ContextWindowPct(s, zeroCfg); got != 0 {
		t.Errorf("zero-window ContextWindowPct = %v, want 0", got)
	}
}

func TestAnomalousDays(t *testing.T) {
	mkBucket := func(key string, credits float64) Bucket {
		return Bucket{Key: key, Credits: credits}
	}

	t.Run("too few points", func(t *testing.T) {
		in := []Bucket{mkBucket("d1", 1), mkBucket("d2", 100)}
		if got := AnomalousDays(in); got != nil {
			t.Errorf("got %v, want nil for <3 points", got)
		}
	})

	t.Run("flags spike", func(t *testing.T) {
		// Many normal days dilute the spike's effect on the population stddev so
		// the threshold (mean + 2*stddev) lands below the spike and flags it.
		in := []Bucket{
			mkBucket("d1", 10), mkBucket("d2", 12), mkBucket("d3", 11),
			mkBucket("d4", 13), mkBucket("d5", 10), mkBucket("d6", 12),
			mkBucket("d7", 11), mkBucket("d8", 13), mkBucket("d9", 10),
			mkBucket("d10", 60), // the spike
		}
		got := AnomalousDays(in)
		if len(got) != 1 || got[0].Key != "d10" {
			t.Fatalf("got %+v, want only d10 flagged", got)
		}
	})

	t.Run("flat series flags nothing", func(t *testing.T) {
		in := []Bucket{mkBucket("a", 5), mkBucket("b", 5), mkBucket("c", 5)}
		if got := AnomalousDays(in); len(got) != 0 {
			t.Errorf("got %+v, want none flagged on flat series", got)
		}
	})

	t.Run("deterministic", func(t *testing.T) {
		in := []Bucket{
			mkBucket("d1", 10), mkBucket("d2", 12), mkBucket("d3", 11),
			mkBucket("d4", 13), mkBucket("d5", 10), mkBucket("d6", 60),
		}
		a := AnomalousDays(in)
		b := AnomalousDays(in)
		if len(a) != len(b) {
			t.Fatalf("non-deterministic length: %d vs %d", len(a), len(b))
		}
		for i := range a {
			if a[i].Key != b[i].Key {
				t.Errorf("non-deterministic at %d: %q vs %q", i, a[i].Key, b[i].Key)
			}
		}
	})
}

func TestCreditsAgreeWithBudget(t *testing.T) {
	d := time.Date(2026, 6, 10, 9, 0, 0, 0, time.UTC)
	s := mkSession("a", "p", "sonnet", d, 7*credit+500_000_000, 100, 10)
	got := DailySeries([]session.Session{s})
	want := budget.FromNanoAIU(s.TotalNanoAIU)
	if got[0].Credits != want {
		t.Errorf("bucket credits = %v, want budget.FromNanoAIU = %v", got[0].Credits, want)
	}
}
