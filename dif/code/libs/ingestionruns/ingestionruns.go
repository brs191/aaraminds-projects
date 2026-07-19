// Package ingestionruns models DIF ingestion run lifecycle and promotion
// safety rules.
package ingestionruns

import (
	"errors"
	"fmt"
	"strings"
)

const (
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"

	ReasonPromotable            Reason = "promotable"
	ReasonRunNotCompleted       Reason = "run_not_completed"
	ReasonDegenerateNoDocuments Reason = "degenerate_no_documents"
	ReasonDegenerateNoNodes     Reason = "degenerate_no_nodes"
	ReasonDegenerateNoAnchors   Reason = "degenerate_no_anchors"
	ReasonDegenerateNoPassages  Reason = "degenerate_no_passages"
	ReasonInvalidRun            Reason = "invalid_run"
	PromotionDecisionAllow      string = "allow"
	PromotionDecisionDeny       string = "deny"
)

// Status is the lifecycle status stored in dif_meta.ingestion_runs.status.
type Status string

// Reason is the explicit promotion decision reason.
type Reason string

// Run contains the ingestion-run fields needed for lifecycle validation and
// promotion decisions.
type Run struct {
	RunID         string
	CorpusID      string
	SourceID      string
	Status        Status
	Stage         string
	DocumentCount int
	NodeCount     int
	EdgeCount     int
	AnchorCount   int
	PassageCount  int
	CaveatCount   int
	ErrorMessage  string
}

// Decision is the result of evaluating whether a run can promote.
type Decision struct {
	CanPromote bool
	Reason     Reason
	Err        error
}

// Record is the safe write-shape for dif_meta.ingestion_runs scaffold tests.
type Record struct {
	RunID        string
	CorpusID     string
	SourceID     string
	Status       Status
	Stage        string
	Counts       Counts
	RunMetrics   map[string]string
	ErrorMessage string
	Promoted     bool
}

// Counts groups non-negative ingestion output counts.
type Counts struct {
	DocumentCount int
	NodeCount     int
	EdgeCount     int
	AnchorCount   int
	PassageCount  int
	CaveatCount   int
}

// ValidationError reports invalid ingestion-run shape before promotion logic.
type ValidationError struct {
	Fields []string
}

// Error returns the invalid field list in stable order.
func (e ValidationError) Error() string {
	if len(e.Fields) == 0 {
		return "invalid ingestion run"
	}
	return "invalid ingestion run: " + strings.Join(e.Fields, ", ")
}

// IsValidationError reports whether err is an ingestion-run validation error.
func IsValidationError(err error) bool {
	var validation ValidationError
	return errors.As(err, &validation)
}

// Validate verifies required fields, known lifecycle status, and non-negative
// counts. It does not decide promotion.
func (r Run) Validate() error {
	var fields []string
	if strings.TrimSpace(r.RunID) == "" {
		fields = append(fields, "run_id")
	}
	if strings.TrimSpace(r.CorpusID) == "" {
		fields = append(fields, "corpus_id")
	}
	if !ValidStatus(r.Status) {
		fields = append(fields, "status")
	}
	for _, count := range []struct {
		name  string
		value int
	}{
		{"document_count", r.DocumentCount},
		{"node_count", r.NodeCount},
		{"edge_count", r.EdgeCount},
		{"anchor_count", r.AnchorCount},
		{"passage_count", r.PassageCount},
		{"caveat_count", r.CaveatCount},
	} {
		if count.value < 0 {
			fields = append(fields, count.name)
		}
	}
	if len(fields) > 0 {
		return ValidationError{Fields: fields}
	}
	return nil
}

// PromotionDecision evaluates the exact P0 promotion guard. It only allows
// completed runs with documents, nodes, anchors, and passages.
func (r Run) PromotionDecision() Decision {
	if err := r.Validate(); err != nil {
		return Decision{CanPromote: false, Reason: ReasonInvalidRun, Err: err}
	}
	if r.Status != StatusCompleted {
		return Decision{CanPromote: false, Reason: ReasonRunNotCompleted, Err: NonPromotableError{RunID: r.RunID, Reason: ReasonRunNotCompleted}}
	}
	if r.DocumentCount <= 0 {
		return Decision{CanPromote: false, Reason: ReasonDegenerateNoDocuments, Err: NonPromotableError{RunID: r.RunID, Reason: ReasonDegenerateNoDocuments}}
	}
	if r.NodeCount <= 0 {
		return Decision{CanPromote: false, Reason: ReasonDegenerateNoNodes, Err: NonPromotableError{RunID: r.RunID, Reason: ReasonDegenerateNoNodes}}
	}
	if r.AnchorCount <= 0 {
		return Decision{CanPromote: false, Reason: ReasonDegenerateNoAnchors, Err: NonPromotableError{RunID: r.RunID, Reason: ReasonDegenerateNoAnchors}}
	}
	if r.PassageCount <= 0 {
		return Decision{CanPromote: false, Reason: ReasonDegenerateNoPassages, Err: NonPromotableError{RunID: r.RunID, Reason: ReasonDegenerateNoPassages}}
	}
	return Decision{CanPromote: true, Reason: ReasonPromotable}
}

// ToRecord returns the write-shape with promotion metrics populated. It never
// marks a non-promotable run as promoted.
func (r Run) ToRecord() Record {
	decision := r.PromotionDecision()
	promotion := PromotionDecisionDeny
	if decision.CanPromote {
		promotion = PromotionDecisionAllow
	}
	return Record{
		RunID:    strings.TrimSpace(r.RunID),
		CorpusID: strings.TrimSpace(r.CorpusID),
		SourceID: strings.TrimSpace(r.SourceID),
		Status:   r.Status,
		Stage:    strings.TrimSpace(r.Stage),
		Counts: Counts{
			DocumentCount: r.DocumentCount,
			NodeCount:     r.NodeCount,
			EdgeCount:     r.EdgeCount,
			AnchorCount:   r.AnchorCount,
			PassageCount:  r.PassageCount,
			CaveatCount:   r.CaveatCount,
		},
		RunMetrics: map[string]string{
			"promotion_decision": promotion,
			"promotion_reason":   string(decision.Reason),
		},
		ErrorMessage: strings.TrimSpace(r.ErrorMessage),
		Promoted:     decision.CanPromote,
	}
}

// NonPromotableError explicitly reports a safe, structured blocked-promotion
// reason.
type NonPromotableError struct {
	RunID  string
	Reason Reason
}

// Error returns a stable non-promotable message.
func (e NonPromotableError) Error() string {
	return fmt.Sprintf("ingestion run %q is not promotable: %s", strings.TrimSpace(e.RunID), e.Reason)
}

// IsNonPromotable reports whether err is a NonPromotableError.
func IsNonPromotable(err error) bool {
	var nonPromotable NonPromotableError
	return errors.As(err, &nonPromotable)
}

// ValidStatus reports whether status is one of the P0 lifecycle statuses.
func ValidStatus(status Status) bool {
	switch status {
	case StatusRunning, StatusCompleted, StatusFailed, StatusCancelled:
		return true
	default:
		return false
	}
}
