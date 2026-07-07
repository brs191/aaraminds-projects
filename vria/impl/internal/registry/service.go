package registry

import (
	"fmt"
	"time"
)

// Service implements the load_use_cases contract (contracts/09 §3.1):
// staging never auto-promotes; promotion is an explicit, audited action.
type Service struct {
	store Store
	newID func(prefix string) string
}

func NewService(store Store) *Service {
	n := 0
	return &Service{store: store, newID: func(p string) string {
		n++
		return fmt.Sprintf("%s-%06d", p, n)
	}}
}

// ImportResult mirrors the load_use_cases output payload.
type ImportResult struct {
	ImportBatchID    string            `json:"import_batch_id"`
	RecordsLoaded    int               `json:"records_loaded"`
	RecordsRejected  int               `json:"records_rejected"`
	ValidationErrors []ValidationError `json:"validation_errors"`
	AuditID          string            `json:"audit_id"`
}

type ValidationError struct {
	RowRef    string `json:"row_ref"`
	Field     string `json:"field"`
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
}

// Import stages rows. Rejected rows are reported, never silently dropped or
// promoted (intake principle 6: surface, don't merge).
func (s *Service) Import(sourceID, requestedBy string, rows []map[string]string) (ImportResult, error) {
	staged, rejected := StageImport(rows)
	batch := ImportBatch{
		ImportBatchID: s.newID("imp"),
		SourceID:      sourceID,
		RequestedBy:   requestedBy,
		Staged:        staged,
		CreatedAt:     time.Now().UTC(),
	}
	if err := s.store.StageBatch(batch); err != nil {
		return ImportResult{}, err
	}
	res := ImportResult{
		ImportBatchID:   batch.ImportBatchID,
		RecordsLoaded:   len(staged) - rejected,
		RecordsRejected: rejected,
	}
	for _, r := range staged {
		for _, e := range r.Errors {
			res.ValidationErrors = append(res.ValidationErrors, ValidationError{
				RowRef: r.UseCaseID, Field: "delivery_status/identity",
				ErrorCode: "VALIDATION_FAILED", Message: e,
			})
		}
	}
	audit := AuditEvent{
		AuditID: s.newID("aud"), ActorID: requestedBy, Action: "registry.import_staged",
		TargetType: "ImportBatch", TargetID: batch.ImportBatchID,
	}
	s.store.AppendAudit(audit)
	res.AuditID = audit.AuditID
	return res, nil
}

// Promote moves clean staged records into the active registry.
func (s *Service) Promote(batchID, actorID string) ([]UseCase, error) {
	return s.store.PromoteBatch(batchID, actorID, false)
}
