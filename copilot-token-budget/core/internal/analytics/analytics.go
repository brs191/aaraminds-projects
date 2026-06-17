// Package analytics provides passive, pure aggregations over a slice of sessions.
//
// Every function is pure: it takes the sessions (and, where cost is involved, a
// pricing.Config) and returns derived values without touching the file system,
// the clock, or any global state. Date bucketing always uses Session.BillingTime,
// normalized to UTC, so spend lands in the day/week/month it settled — unambiguously
// in UTC and matching the TypeScript port regardless of the host timezone.
//
// Credits are computed via budget.FromNanoAIU so this package agrees with the
// budget package's totals to the last nano.
package analytics

import (
	"math"
	"sort"
	"time"

	"github.com/aaraminds/copilot-token-budget/internal/budget"
	"github.com/aaraminds/copilot-token-budget/internal/pricing"
	"github.com/aaraminds/copilot-token-budget/internal/session"
)

// Bucket is one time slice (day, ISO week, or month) of aggregated usage.
type Bucket struct {
	// Key is the human-stable bucket label ("2006-01-02", "2006-W01", "2006-01").
	Key string
	// Start is the bucket's lower time bound, in UTC.
	Start time.Time
	// Sessions is the count of sessions attributed to this bucket.
	Sessions int
	// Credits is total credits consumed in the bucket.
	Credits float64
	// InputTokens / OutputTokens are token totals across the bucket's sessions.
	InputTokens  int64
	OutputTokens int64
	// ByModel maps model name -> credits consumed by that model in the bucket.
	ByModel map[string]float64
}

// Consumer is a ranked aggregate row (a session, model, or project).
type Consumer struct {
	// Name is the display label (project name or session id, model name, etc.).
	Name string
	// Credits is the consumer's total credit spend.
	Credits float64
	// InputTokens / OutputTokens are the consumer's token totals.
	InputTokens  int64
	OutputTokens int64
	// Model is the primary model, where one consumer maps to a single model.
	Model string
}

// sessionCredits returns a session's credit cost from its settled nanoAIU.
func sessionCredits(s session.Session) float64 {
	return budget.FromNanoAIU(s.TotalNanoAIU)
}

// DailySeries returns one Bucket per calendar day that has data, keyed
// "2006-01-02" and sorted ascending by Start.
func DailySeries(sessions []session.Session) []Bucket {
	return series(sessions, func(t time.Time) (string, time.Time) {
		day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
		return day.Format("2006-01-02"), day
	})
}

// WeeklySeries returns one Bucket per ISO week that has data, keyed "2006-W01"
// (ISO-year + ISO-week) and sorted ascending by Start. Start is the Monday of
// the ISO week.
func WeeklySeries(sessions []session.Session) []Bucket {
	return series(sessions, func(t time.Time) (string, time.Time) {
		year, week := t.ISOWeek()
		// Monday of the ISO week containing t.
		weekday := int(t.Weekday())
		if weekday == 0 { // time.Sunday == 0; ISO treats Sunday as day 7.
			weekday = 7
		}
		monday := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).
			AddDate(0, 0, -(weekday - 1))
		key := isoWeekKey(year, week)
		return key, monday
	})
}

// MonthlySeries returns one Bucket per calendar month that has data, keyed
// "2006-01" and sorted ascending by Start.
func MonthlySeries(sessions []session.Session) []Bucket {
	return series(sessions, func(t time.Time) (string, time.Time) {
		month := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
		return month.Format("2006-01"), month
	})
}

// isoWeekKey formats an ISO year/week pair as "2006-W01" with a zero-padded week.
func isoWeekKey(year, week int) string {
	w := []byte{'0' + byte(week/10), '0' + byte(week%10)}
	return itoa4(year) + "-W" + string(w)
}

// itoa4 formats a (non-negative, <= 4 digit) year without importing strconv-heavy
// paths; falls back to time formatting semantics by building the string directly.
func itoa4(year int) string {
	if year < 0 {
		year = 0
	}
	return string([]byte{
		'0' + byte((year/1000)%10),
		'0' + byte((year/100)%10),
		'0' + byte((year/10)%10),
		'0' + byte(year%10),
	})
}

// series is the shared bucketing engine. keyFn maps a BillingTime to a (key,
// bucketStart) pair; sessions sharing a key are aggregated and the result is
// sorted ascending by Start. Only buckets with at least one session are returned.
func series(sessions []session.Session, keyFn func(time.Time) (string, time.Time)) []Bucket {
	byKey := make(map[string]*Bucket)
	for _, s := range sessions {
		// Normalize to UTC so bucketing is timezone-independent and matches the
		// TypeScript port (which buckets in UTC). The keyFns build their day/week/
		// month boundary in t.Location(), which is UTC after this conversion.
		key, start := keyFn(s.BillingTime().UTC())
		b := byKey[key]
		if b == nil {
			b = &Bucket{Key: key, Start: start, ByModel: make(map[string]float64)}
			byKey[key] = b
		}
		b.Sessions++
		b.Credits += sessionCredits(s)
		b.InputTokens += s.TotalInputTokens()
		b.OutputTokens += s.TotalOutputTokens()
		for _, m := range s.ModelMetrics {
			b.ByModel[m.Model] += budget.FromNanoAIU(m.NanoAIU)
		}
	}

	out := make([]Bucket, 0, len(byKey))
	for _, b := range byKey {
		out = append(out, *b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start.Before(out[j].Start) })
	return out
}

// TopSessions returns up to n sessions ranked by credits descending. Each row's
// Name is the project name, falling back to the session id when the project is
// unknown. Ties break by Name then by ID-bearing Name for determinism.
func TopSessions(sessions []session.Session, n int) []Consumer {
	rows := make([]Consumer, 0, len(sessions))
	for _, s := range sessions {
		name := s.ProjectName
		if name == "" {
			name = s.ID
		}
		rows = append(rows, Consumer{
			Name:         name,
			Credits:      sessionCredits(s),
			InputTokens:  s.TotalInputTokens(),
			OutputTokens: s.TotalOutputTokens(),
			Model:        s.PrimaryModel,
		})
	}
	sortConsumers(rows)
	return topN(rows, n)
}

// TopModels aggregates per-model credits across all sessions (summing each
// session's ByModel contribution) and returns up to n models by credits desc.
func TopModels(sessions []session.Session, n int) []Consumer {
	type agg struct {
		credits      float64
		inputTokens  int64
		outputTokens int64
	}
	byModel := make(map[string]*agg)
	for _, s := range sessions {
		for _, m := range s.ModelMetrics {
			a := byModel[m.Model]
			if a == nil {
				a = &agg{}
				byModel[m.Model] = a
			}
			a.credits += budget.FromNanoAIU(m.NanoAIU)
			a.inputTokens += m.InputTokens
			a.outputTokens += m.OutputTokens
		}
	}
	rows := make([]Consumer, 0, len(byModel))
	for name, a := range byModel {
		rows = append(rows, Consumer{
			Name:         name,
			Credits:      a.credits,
			InputTokens:  a.inputTokens,
			OutputTokens: a.outputTokens,
			Model:        name,
		})
	}
	sortConsumers(rows)
	return topN(rows, n)
}

// TopProjects aggregates credits and tokens by project name across all sessions
// and returns up to n projects by credits descending. Sessions with no project
// name aggregate under their id.
func TopProjects(sessions []session.Session, n int) []Consumer {
	type agg struct {
		credits      float64
		inputTokens  int64
		outputTokens int64
		model        string
	}
	byProject := make(map[string]*agg)
	for _, s := range sessions {
		name := s.ProjectName
		if name == "" {
			name = s.ID
		}
		a := byProject[name]
		if a == nil {
			a = &agg{}
			byProject[name] = a
		}
		a.credits += sessionCredits(s)
		a.inputTokens += s.TotalInputTokens()
		a.outputTokens += s.TotalOutputTokens()
		if a.model == "" {
			a.model = s.PrimaryModel
		}
	}
	rows := make([]Consumer, 0, len(byProject))
	for name, a := range byProject {
		rows = append(rows, Consumer{
			Name:         name,
			Credits:      a.credits,
			InputTokens:  a.inputTokens,
			OutputTokens: a.outputTokens,
			Model:        a.model,
		})
	}
	sortConsumers(rows)
	return topN(rows, n)
}

// sortConsumers orders by credits desc, breaking ties by Name asc for stable,
// deterministic output.
func sortConsumers(rows []Consumer) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Credits != rows[j].Credits {
			return rows[i].Credits > rows[j].Credits
		}
		return rows[i].Name < rows[j].Name
	})
}

// topN returns the first n rows, or all of them when n <= 0 or n >= len.
func topN(rows []Consumer, n int) []Consumer {
	if n <= 0 || n >= len(rows) {
		return rows
	}
	return rows[:n]
}

// ContextWindowPct returns how full a session's context window is, as a percent
// of the primary model's ContextWindowTokens from cfg. Returns 0 when the window
// is unknown/zero to avoid a divide-by-zero.
func ContextWindowPct(s session.Session, cfg pricing.Config) float64 {
	window := cfg.RateFor(s.PrimaryModel).ContextWindowTokens
	if window <= 0 {
		return 0
	}
	return float64(s.Tokens.CurrentTokens) / float64(window) * 100
}

// AnomalousDays flags days whose Credits exceed mean + 2*populationStdDev of the
// supplied daily series. It returns the flagged buckets in the input order. The
// result is empty when there are fewer than 3 data points (too few to define a
// distribution). The computation is deterministic.
//
// Callers should pass the output of DailySeries; any series of Buckets works.
func AnomalousDays(daily []Bucket) []Bucket {
	if len(daily) < 3 {
		return nil
	}
	n := float64(len(daily))
	var sum float64
	for _, b := range daily {
		sum += b.Credits
	}
	mean := sum / n

	var variance float64
	for _, b := range daily {
		d := b.Credits - mean
		variance += d * d
	}
	variance /= n // population variance
	threshold := mean + 2*math.Sqrt(variance)

	var out []Bucket
	for _, b := range daily {
		if b.Credits > threshold {
			out = append(out, b)
		}
	}
	return out
}
