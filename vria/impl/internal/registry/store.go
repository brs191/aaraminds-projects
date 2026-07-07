package registry

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aaraminds/vria/internal/enums"
)

// UseCase mirrors contracts/17 §3.
type UseCase struct {
	UseCaseID       string               `json:"use_case_id"`
	Name            string               `json:"name"`
	Tier            enums.UseCaseTier    `json:"tier"`
	Domain          string               `json:"domain,omitempty"`
	ValueOwner      string               `json:"value_owner,omitempty"`
	DeliveryOwner   string               `json:"delivery_owner,omitempty"`
	Sponsor         string               `json:"sponsor,omitempty"`
	DeliveryStatus  enums.DeliveryStatus `json:"delivery_status"`
	PrimaryMetricID string               `json:"primary_metric_id,omitempty"`
	ApprovalState   enums.ArtifactState  `json:"approval_state"`
	RecordVersion   int                  `json:"record_version"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
}

// AuditEvent mirrors contracts/19 audit_events. Append-only.
type AuditEvent struct {
	AuditID    string    `json:"audit_id"`
	ActorID    string    `json:"actor_id"`
	Action     string    `json:"action"`
	TargetType string    `json:"target_type"`
	TargetID   string    `json:"target_id"`
	TraceID    string    `json:"trace_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// ImportBatch is one staged import (contracts/09 §3.1).
type ImportBatch struct {
	ImportBatchID string         `json:"import_batch_id"`
	SourceID      string         `json:"source_id"`
	RequestedBy   string         `json:"requested_by"`
	Staged        []StagedRecord `json:"staged"`
	Promoted      bool           `json:"promoted"`
	CreatedAt     time.Time      `json:"created_at"`
}

var (
	ErrNotFound         = errors.New("not found")
	ErrAlreadyPromoted  = errors.New("batch already promoted")
	ErrRejectedRecords  = errors.New("batch has rejected records; resolve or exclude before promotion")
	ErrDuplicateUseCase = errors.New("use case already exists in registry")
	ErrNothingToPromote = errors.New("batch has no promotable records; correct rejects and re-import")
)

// Store is the persistence boundary. The in-memory implementation backs
// tests and local runs; the PostgreSQL implementation (contracts/19) plugs
// in behind the same interface.
type Store interface {
	StageBatch(b ImportBatch) error
	GetBatch(id string) (ImportBatch, error)
	// PromoteBatch moves clean staged records into the active registry. It
	// fails atomically: partial promotion is never permitted (09 §3.1). When
	// failOnRejected is true, any rejected row aborts the whole promotion;
	// when false, rejected rows are skipped and left for re-import after
	// triage. A batch with no promotable rows returns ErrNothingToPromote.
	PromoteBatch(id, actorID string, failOnRejected bool) ([]UseCase, error)
	ListUseCases() []UseCase
	GetUseCase(id string) (UseCase, error)
	AppendAudit(e AuditEvent)
	AuditTrail() []AuditEvent
}

type MemStore struct {
	mu       sync.RWMutex
	batches  map[string]*ImportBatch
	useCases map[string]UseCase
	order    []string
	audit    []AuditEvent
	seq      int
}

func NewMemStore() *MemStore {
	return &MemStore{
		batches:  map[string]*ImportBatch{},
		useCases: map[string]UseCase{},
	}
}

func (s *MemStore) nextID(prefix string) string {
	s.seq++
	return fmt.Sprintf("%s-%06d", prefix, s.seq)
}

func (s *MemStore) StageBatch(b ImportBatch) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if b.ImportBatchID == "" {
		return errors.New("import_batch_id required")
	}
	cp := b
	s.batches[b.ImportBatchID] = &cp
	return nil
}

func (s *MemStore) GetBatch(id string) (ImportBatch, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.batches[id]
	if !ok {
		return ImportBatch{}, ErrNotFound
	}
	return *b, nil
}

func (s *MemStore) PromoteBatch(id, actorID string, failOnRejected bool) ([]UseCase, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.batches[id]
	if !ok {
		return nil, ErrNotFound
	}
	if b.Promoted {
		return nil, ErrAlreadyPromoted
	}
	// Validate the whole batch before mutating anything (atomic promotion).
	var toPromote []StagedRecord
	for _, r := range b.Staged {
		if len(r.Errors) > 0 {
			if failOnRejected {
				return nil, fmt.Errorf("%w: %s", ErrRejectedRecords, r.UseCaseID)
			}
			continue // rejected records are skipped; re-import after triage
		}
		if _, exists := s.useCases[r.UseCaseID]; exists {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateUseCase, r.UseCaseID)
		}
		toPromote = append(toPromote, r)
	}
	if len(toPromote) == 0 {
		// An all-rejected batch must not report success with an empty list.
		return nil, ErrNothingToPromote
	}
	now := time.Now().UTC()
	var promoted []UseCase
	for _, r := range toPromote {
		uc := UseCase{
			UseCaseID:      r.UseCaseID,
			Name:           r.Name,
			Tier:           r.Tier,
			DeliveryStatus: r.DeliveryStatus,
			ApprovalState:  enums.ArtDraft,
			RecordVersion:  1,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		s.useCases[uc.UseCaseID] = uc
		s.order = append(s.order, uc.UseCaseID)
		promoted = append(promoted, uc)
	}
	b.Promoted = true
	s.audit = append(s.audit, AuditEvent{
		AuditID: s.nextID("aud"), ActorID: actorID, Action: "registry.promote_batch",
		TargetType: "ImportBatch", TargetID: id, CreatedAt: now,
	})
	return promoted, nil
}

func (s *MemStore) ListUseCases() []UseCase {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]UseCase, 0, len(s.order))
	for _, id := range s.order {
		out = append(out, s.useCases[id])
	}
	return out
}

func (s *MemStore) GetUseCase(id string) (UseCase, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	uc, ok := s.useCases[id]
	if !ok {
		return UseCase{}, ErrNotFound
	}
	return uc, nil
}

func (s *MemStore) AppendAudit(e AuditEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.AuditID == "" {
		e.AuditID = s.nextID("aud")
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	s.audit = append(s.audit, e)
}

func (s *MemStore) AuditTrail() []AuditEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]AuditEvent, len(s.audit))
	copy(out, s.audit)
	return out
}
