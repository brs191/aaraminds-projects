// Package hypothesis implements Epic 2 (gate-d-operations/12): value
// hypothesis drafts, validation per gate-a-value/03 §4, and approval-gated
// commit per contracts/18. Drafts never touch the active record; a commit
// requires an Approved approval request, and a rejected request leaves the
// original untouched.
package hypothesis

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aaraminds/vria/internal/approval"
	"github.com/aaraminds/vria/internal/enums"
)

// Hypothesis mirrors contracts/17 §4 (persisted fields only; operational
// fields like current_value live on MetricSnapshot — see 03 field-ownership
// note).
type Hypothesis struct {
	ValueHypothesisID string                  `json:"value_hypothesis_id"`
	UseCaseID         string                  `json:"use_case_id"`
	BusinessObjective string                  `json:"business_objective"`
	ExpectedBenefit   string                  `json:"expected_benefit"`
	BenefitType       string                  `json:"benefit_type"`
	PrimaryMetricID   string                  `json:"primary_metric_id"`
	BaselineValue     *float64                `json:"baseline_value"`
	TargetValue       *float64                `json:"target_value"`
	AttributionMethod enums.AttributionMethod `json:"attribution_method"`
	KnownConfounders  []string                `json:"known_confounders"`
	NetValueCheck     enums.NetValueCheck     `json:"net_value_check"`
	ValueOwner        string                  `json:"value_owner"`
	ApprovalState     enums.ArtifactState     `json:"approval_state"`
	RecordVersion     int                     `json:"record_version"`
	CreatedAt         time.Time               `json:"created_at"`
}

// allowedFields is exactly the draft_use_case_update contract
// (contracts/09 §3.4) plus core hypothesis fields from 17 §4.
var allowedFields = map[string]bool{
	"tier": true, "domain": true, "value_owner": true, "delivery_owner": true,
	"sponsor": true, "primary_metric_id": true, "expected_benefit": true,
	"attribution_method": true, "known_confounders": true,
	"initiative_cost_period": true, "net_value_check": true,
	"business_objective": true, "benefit_type": true,
	"baseline_value": true, "target_value": true,
}

// Draft is a proposed change set awaiting approval.
type Draft struct {
	DraftID          string                 `json:"draft_id"`
	UseCaseID        string                 `json:"use_case_id"`
	Proposed         map[string]interface{} `json:"proposed_changes"`
	ValidationStatus string                 `json:"validation_status"` // Valid | Invalid
	ValidationErrors []string               `json:"validation_errors"`
	RequiresApproval bool                   `json:"requires_approval"`
	Committed        bool                   `json:"committed"`
	CreatedBy        string                 `json:"created_by"`
	CreatedAt        time.Time              `json:"created_at"`
}

var (
	ErrNotFound         = errors.New("not found")
	ErrDisallowedField  = errors.New("field not permitted by draft_use_case_update contract")
	ErrInvalidDraft     = errors.New("draft failed validation")
	ErrAlreadyCommitted = errors.New("draft already committed")
	ErrApprovalRequired = approval.ErrApprovalRequired
)

// AuditSink decouples this package from the registry store.
type AuditSink func(action, targetType, targetID, actorID string)

type Service struct {
	mu       sync.Mutex
	byUC     map[string]*Hypothesis
	drafts   map[string]*Draft
	requests map[string]*approval.Request
	audit    AuditSink
	seq      int
}

func NewService(audit AuditSink) *Service {
	if audit == nil {
		audit = func(string, string, string, string) {}
	}
	return &Service{
		byUC:     map[string]*Hypothesis{},
		drafts:   map[string]*Draft{},
		requests: map[string]*approval.Request{},
		audit:    audit,
	}
}

func (s *Service) nextID(p string) string {
	s.seq++
	return fmt.Sprintf("%s-%06d", p, s.seq)
}

// Get returns the active hypothesis and its Gate A missing-field list
// (contracts/09 §3.3 output shape).
func (s *Service) Get(useCaseID string) (Hypothesis, []string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	h, ok := s.byUC[useCaseID]
	if !ok {
		return Hypothesis{}, nil, ErrNotFound
	}
	return *h, missingRequired(*h), nil
}

// missingRequired implements gate-a-value/03 §4 (required before Gate A exit).
func missingRequired(h Hypothesis) []string {
	var m []string
	if h.ValueOwner == "" {
		m = append(m, "value_owner")
	}
	if h.ExpectedBenefit == "" {
		m = append(m, "expected_benefit")
	}
	if h.PrimaryMetricID == "" {
		m = append(m, "primary_metric_id")
	}
	if h.BaselineValue == nil {
		m = append(m, "baseline_value")
	}
	if h.TargetValue == nil {
		m = append(m, "target_value")
	}
	return m
}

// numberFields must arrive as JSON numbers; a mistyped value (e.g. a string
// "42") would otherwise pass field-name validation, commit, and then be
// silently dropped by applyChanges' type assertion — a data-loss trap.
var numberFields = map[string]bool{"baseline_value": true, "target_value": true}

// CreateDraft validates a proposed change set. Disallowed fields and type
// mismatches reject the draft outright; the original record is never touched
// (09 §3.4 failure rule). Deep-copies the proposed map so a caller mutating
// it after the call cannot bypass validation or race the commit read.
func (s *Service) CreateDraft(useCaseID, actorID string, proposed map[string]interface{}) (Draft, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make(map[string]interface{}, len(proposed))
	for k, v := range proposed {
		cp[k] = v
	}
	d := Draft{
		DraftID: s.nextID("drf"), UseCaseID: useCaseID, Proposed: cp,
		RequiresApproval: true, CreatedBy: actorID, CreatedAt: time.Now().UTC(),
		ValidationStatus: "Valid",
	}
	for field, val := range cp {
		if !allowedFields[field] {
			d.ValidationStatus = "Invalid"
			d.ValidationErrors = append(d.ValidationErrors,
				fmt.Sprintf("%s: %v", ErrDisallowedField, field))
			continue
		}
		if numberFields[field] {
			if _, ok := val.(float64); !ok && val != nil {
				d.ValidationStatus = "Invalid"
				d.ValidationErrors = append(d.ValidationErrors,
					fmt.Sprintf("%s must be a number, got %T", field, val))
			}
		}
	}
	s.drafts[d.DraftID] = &d
	s.audit("hypothesis.draft_created", "Draft", d.DraftID, actorID)
	return d, nil
}

// SubmitForApproval opens an approval request of type RegistryUpdate.
func (s *Service) SubmitForApproval(draftID, requestedBy string) (approval.Request, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.drafts[draftID]
	if !ok {
		return approval.Request{}, ErrNotFound
	}
	if d.ValidationStatus != "Valid" {
		return approval.Request{}, fmt.Errorf("%w: %v", ErrInvalidDraft, d.ValidationErrors)
	}
	req := approval.Request{
		ApprovalID: s.nextID("apr"), ActionType: "RegistryUpdate",
		TargetID: draftID, TargetType: "UseCase",
		RequestedBy: requestedBy, State: enums.ReqSubmitted,
	}
	s.requests[req.ApprovalID] = &req
	s.audit("approval.submitted", "ApprovalRequest", req.ApprovalID, requestedBy)
	return req, nil
}

// Decide applies an approver decision. decidedBy comes from the
// authenticated principal, never the payload (contracts/18).
func (s *Service) Decide(approvalID, action, decidedBy string) (approval.Request, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	req, ok := s.requests[approvalID]
	if !ok {
		return approval.Request{}, ErrNotFound
	}
	// Separation of duties: the requester cannot approve their own request
	// (contracts/18 §3). Checked before the transition so a self-approval
	// attempt leaves the request untouched.
	if err := approval.CheckApprover(req, action, decidedBy); err != nil {
		return *req, err
	}
	next, err := approval.RequestTransition(req.State, action)
	if err != nil {
		return *req, err
	}
	req.State = next
	req.DecidedBy = decidedBy
	s.audit("approval.decided", "ApprovalRequest", approvalID, decidedBy)
	return *req, nil
}

// Commit applies an approved draft to the active hypothesis, bumping
// record_version (contracts/17 §11). Without an Approved request of type
// RegistryUpdate for this draft, nothing changes.
func (s *Service) Commit(draftID, approvalID, actorID string) (Hypothesis, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.drafts[draftID]
	if !ok {
		return Hypothesis{}, ErrNotFound
	}
	if d.Committed {
		return Hypothesis{}, ErrAlreadyCommitted
	}
	req, ok := s.requests[approvalID]
	if !ok || req.State != enums.ReqApproved ||
		req.ActionType != "RegistryUpdate" || req.TargetID != draftID {
		return Hypothesis{}, ErrApprovalRequired
	}
	h, ok := s.byUC[d.UseCaseID]
	if !ok {
		h = &Hypothesis{
			ValueHypothesisID: s.nextID("vh"), UseCaseID: d.UseCaseID,
			AttributionMethod: enums.AttributionUnknown,
			NetValueCheck:     enums.NetUnknown,
			ApprovalState:     enums.ArtDraft,
			CreatedAt:         time.Now().UTC(),
		}
		s.byUC[d.UseCaseID] = h
	}
	applyChanges(h, d.Proposed)
	h.RecordVersion++
	h.ApprovalState = enums.ArtApproved
	d.Committed = true
	s.audit("hypothesis.committed", "ValueHypothesis", h.ValueHypothesisID, actorID)
	return *h, nil
}

func applyChanges(h *Hypothesis, p map[string]interface{}) {
	if v, ok := p["business_objective"].(string); ok {
		h.BusinessObjective = v
	}
	if v, ok := p["expected_benefit"].(string); ok {
		h.ExpectedBenefit = v
	}
	if v, ok := p["benefit_type"].(string); ok {
		h.BenefitType = v
	}
	if v, ok := p["primary_metric_id"].(string); ok {
		h.PrimaryMetricID = v
	}
	if v, ok := p["value_owner"].(string); ok {
		h.ValueOwner = v
	}
	if v, ok := p["attribution_method"].(string); ok {
		h.AttributionMethod = enums.AttributionMethod(v)
	}
	if v, ok := p["net_value_check"].(string); ok {
		h.NetValueCheck = enums.NetValueCheck(v)
	}
	if v, ok := p["baseline_value"].(float64); ok {
		h.BaselineValue = &v
	}
	if v, ok := p["target_value"].(float64); ok {
		h.TargetValue = &v
	}
	if v, ok := p["known_confounders"].([]interface{}); ok {
		h.KnownConfounders = nil
		for _, c := range v {
			if cs, ok := c.(string); ok {
				h.KnownConfounders = append(h.KnownConfounders, cs)
			}
		}
	}
}
