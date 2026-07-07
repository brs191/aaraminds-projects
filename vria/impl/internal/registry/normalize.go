// Package registry implements intake normalization for Epic 1
// (gate-a-value/02 §2–§4). Reject-don't-guess: an unmapped or ambiguous
// source status surfaces an error instead of a silent merge
// (intake principle 6).
package registry

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aaraminds/vria/internal/enums"
)

var (
	ErrUnmappedStatus  = errors.New("unmapped delivery status; add to mapping table")
	ErrAmbiguousStatus = errors.New("ambiguous/conflicting delivery status; surface to portfolio lead")
)

// statusMapping is the reviewable mapping table (P1.1 constraint: a table,
// not inline logic). Keys are normalized to lowercase, trimmed.
var statusMapping = map[string]enums.DeliveryStatus{
	"draft":                              enums.DSDraft,
	"discovery":                          enums.DSDiscovery,
	"training":                           enums.DSTraining,
	"training; ptb not started; pending": enums.DSTraining,
	"ptb":                                enums.DSPTBInProgress,
	"ptb not started":                    enums.DSPTBNotStarted,
	"ptb in progress":                    enums.DSPTBInProgress,
	"ptb approved":                       enums.DSPTBApproved,
	"pto not started":                    enums.DSPTONotStarted,
	"pto in progress":                    enums.DSPTOInProgress,
	"pto approved":                       enums.DSPTOApproved,
	"in progress":                        enums.DSInProgress,
	"pilot":                              enums.DSPilot,
	"production":                         enums.DSProduction,
	"blocked":                            enums.DSBlocked,
	"stopped":                            enums.DSStopped,
	"not captured":                       enums.DSUnknown,
	"":                                   enums.DSUnknown,
}

// ambiguousMarkers flag source strings that carry conflicting states and
// must be resolved by a human during Gate A (internal/99 note: "mixed
// PTB/PTO values must be resolved before production scoring").
var ambiguousMarkers = []string{"mixed", "not started/completed", "/completed/pending"}

// NormalizeDeliveryStatus maps a raw source status to the canonical enum.
func NormalizeDeliveryStatus(raw string) (enums.DeliveryStatus, error) {
	key := strings.ToLower(strings.TrimSpace(raw))
	for _, m := range ambiguousMarkers {
		if strings.Contains(key, m) {
			return enums.DSUnknown, fmt.Errorf("%w: %q", ErrAmbiguousStatus, raw)
		}
	}
	if s, ok := statusMapping[key]; ok {
		return s, nil
	}
	return enums.DSUnknown, fmt.Errorf("%w: %q", ErrUnmappedStatus, raw)
}

// StagedRecord is one row in the import staging area (contracts/09 §3.1).
type StagedRecord struct {
	UseCaseID      string
	Name           string
	Tier           enums.UseCaseTier
	DeliveryStatus enums.DeliveryStatus
	Errors         []string
}

// StageImport validates raw rows into staging. Rows with errors are staged
// with rejection reasons — never silently promoted (contracts/09 §3.1
// failure behavior).
func StageImport(rows []map[string]string) (staged []StagedRecord, rejected int) {
	for _, row := range rows {
		rec := StagedRecord{UseCaseID: row["use_case_id"], Name: row["name"]}
		if rec.UseCaseID == "" {
			rec.Errors = append(rec.Errors, "missing use_case_id")
		}
		if rec.Name == "" {
			rec.Errors = append(rec.Errors, "missing name")
		}
		switch enums.UseCaseTier(row["tier"]) {
		case enums.TierTool, enums.TierAgent, enums.TierLayer:
			rec.Tier = enums.UseCaseTier(row["tier"])
		default:
			rec.Tier = enums.TierUnclassified
		}
		ds, err := NormalizeDeliveryStatus(row["delivery_status"])
		rec.DeliveryStatus = ds
		if err != nil {
			rec.Errors = append(rec.Errors, err.Error())
		}
		if len(rec.Errors) > 0 {
			rejected++
		}
		staged = append(staged, rec)
	}
	return staged, rejected
}
