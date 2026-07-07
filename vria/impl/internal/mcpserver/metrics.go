// metrics.go implements the get_metric_snapshot tool (09 §3.5).
//
// Reference adapter: CSV file at MetricsConfig.CSVPath with columns:
//   metric_id, use_case_id, period_start, period_end,
//   baseline_value, current_value, target_value, metric_unit,
//   source_system, source_owner, authority, freshness, cost, currency
//
// Missing metric → METRIC_UNAVAILABLE. Values are never inferred or fabricated
// (GE-013).
package mcpserver

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/aaraminds/vria/internal/enums"
)

// MetricsConfig configures the get_metric_snapshot handler.
type MetricsConfig struct {
	// CSVPath is the path to the metrics CSV file.
	CSVPath string
}

// --- input / output types (mirrors §3.5 JSON contract) ---

type metricSnapshotInput struct {
	MetricID  string       `json:"metric_id"`
	Period    *periodInput `json:"period"`
	UseCaseID string       `json:"use_case_id"`
}

type periodInput struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type metricSnapshotOutput struct {
	MetricSnapshot       metricSnapshot       `json:"metric_snapshot"`
	InitiativeCostPeriod initiativeCostPeriod `json:"initiative_cost_period"`
	SourceOwner          string               `json:"source_owner"`
	Freshness            enums.Freshness      `json:"freshness"`
	Authority            enums.Authority      `json:"authority"`
	AuditID              string               `json:"audit_id"`
}

type metricSnapshot struct {
	MetricID      string  `json:"metric_id"`
	UseCaseID     string  `json:"use_case_id"`
	PeriodStart   string  `json:"period_start"`
	PeriodEnd     string  `json:"period_end"`
	BaselineValue float64 `json:"baseline_value"`
	CurrentValue  float64 `json:"current_value"`
	TargetValue   float64 `json:"target_value"`
	MetricUnit    string  `json:"metric_unit"`
	SourceSystem  string  `json:"source_system"`
}

type initiativeCostPeriod struct {
	Start    string  `json:"start"`
	End      string  `json:"end"`
	Cost     float64 `json:"cost"`
	Currency string  `json:"currency"`
}

// csvMetricRow holds one parsed row from the CSV.
type csvMetricRow struct {
	metricID     string
	useCaseID    string
	periodStart  string
	periodEnd    string
	baselineVal  float64
	currentVal   float64
	targetVal    float64
	metricUnit   string
	sourceSystem string
	sourceOwner  string
	authority    string
	freshness    string
	cost         float64
	currency     string
}

const (
	csvColMetricID     = 0
	csvColUseCaseID    = 1
	csvColPeriodStart  = 2
	csvColPeriodEnd    = 3
	csvColBaselineVal  = 4
	csvColCurrentVal   = 5
	csvColTargetVal    = 6
	csvColMetricUnit   = 7
	csvColSourceSystem = 8
	csvColSourceOwner  = 9
	csvColAuthority    = 10
	csvColFreshness    = 11
	csvColCost         = 12
	csvColCurrency     = 13
	csvExpectedCols    = 14
)

// NewMetricSnapshotHandler returns a Handler for get_metric_snapshot backed by
// a CSV file.
func NewMetricSnapshotHandler(cfg MetricsConfig) Handler {
	return func(ctx context.Context, input json.RawMessage) (interface{}, *ToolError) {
		var req metricSnapshotInput
		if err := json.Unmarshal(input, &req); err != nil {
			return nil, &ToolError{Code: ErrInvalidInput, Message: "cannot parse input: " + err.Error()}
		}
		if strings.TrimSpace(req.MetricID) == "" {
			return nil, &ToolError{Code: ErrInvalidInput, Message: "missing required field: metric_id"}
		}

		row, msg := findMetricRow(cfg.CSVPath, req.MetricID, req.UseCaseID, req.Period)
		if msg != "" {
			return nil, &ToolError{Code: ErrMetricUnavailable, Message: msg}
		}

		out := metricSnapshotOutput{
			MetricSnapshot: metricSnapshot{
				MetricID:      row.metricID,
				UseCaseID:     row.useCaseID,
				PeriodStart:   row.periodStart,
				PeriodEnd:     row.periodEnd,
				BaselineValue: row.baselineVal,
				CurrentValue:  row.currentVal,
				TargetValue:   row.targetVal,
				MetricUnit:    row.metricUnit,
				SourceSystem:  row.sourceSystem,
			},
			InitiativeCostPeriod: initiativeCostPeriod{
				Start:    row.periodStart,
				End:      row.periodEnd,
				Cost:     row.cost,
				Currency: row.currency,
			},
			SourceOwner: row.sourceOwner,
			Freshness:   canonicalFreshness(row.freshness),
			Authority:   canonicalAuthority(row.authority),
			AuditID:     newAuditID(),
		}
		return out, nil
	}
}

// findMetricRow scans the CSV for the first row matching metric_id and
// optionally use_case_id. Period filtering is applied when both start and end
// are non-empty. Returns a string message (non-empty means not found / error).
func findMetricRow(path, metricID, useCaseID string, period *periodInput) (*csvMetricRow, string) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Sprintf("cannot open metrics CSV: %s", err.Error())
	}
	defer f.Close()

	r := csv.NewReader(f)
	// Skip header row.
	if _, err := r.Read(); err != nil {
		return nil, fmt.Sprintf("cannot read CSV header: %s", err.Error())
	}

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue // skip malformed rows
		}
		if len(record) < csvExpectedCols {
			continue
		}
		if record[csvColMetricID] != metricID {
			continue
		}
		if useCaseID != "" && record[csvColUseCaseID] != useCaseID {
			continue
		}
		// Optional period filter: row must overlap requested period.
		if period != nil && period.Start != "" && period.End != "" {
			if !periodOverlaps(record[csvColPeriodStart], record[csvColPeriodEnd], period.Start, period.End) {
				continue
			}
		}

		row := &csvMetricRow{
			metricID:     record[csvColMetricID],
			useCaseID:    record[csvColUseCaseID],
			periodStart:  record[csvColPeriodStart],
			periodEnd:    record[csvColPeriodEnd],
			metricUnit:   record[csvColMetricUnit],
			sourceSystem: record[csvColSourceSystem],
			sourceOwner:  record[csvColSourceOwner],
			authority:    record[csvColAuthority],
			freshness:    record[csvColFreshness],
			currency:     record[csvColCurrency],
		}
		row.baselineVal, _ = strconv.ParseFloat(record[csvColBaselineVal], 64)
		row.currentVal, _ = strconv.ParseFloat(record[csvColCurrentVal], 64)
		row.targetVal, _ = strconv.ParseFloat(record[csvColTargetVal], 64)
		row.cost, _ = strconv.ParseFloat(record[csvColCost], 64)
		return row, ""
	}

	return nil, fmt.Sprintf("metric %q not found in CSV", metricID)
}

// periodOverlaps is a simple string-based date overlap check for ISO-8601
// date strings (YYYY-MM-DD). Row [rs,re] overlaps request [qs,qe] when
// rs <= qe AND re >= qs.
func periodOverlaps(rowStart, rowEnd, queryStart, queryEnd string) bool {
	return rowStart <= queryEnd && rowEnd >= queryStart
}

func canonicalFreshness(s string) enums.Freshness {
	switch s {
	case string(enums.Fresh):
		return enums.Fresh
	case string(enums.Aging):
		return enums.Aging
	case string(enums.Stale):
		return enums.Stale
	default:
		return enums.FreshnessUnknown
	}
}

func canonicalAuthority(s string) enums.Authority {
	switch s {
	case string(enums.Authoritative):
		return enums.Authoritative
	case string(enums.Secondary):
		return enums.Secondary
	default:
		return enums.AuthorityUnknown
	}
}
