// Package scorecard implements the scorecard lifecycle (P5.1) per
// contracts/17 §9 and contracts/18: draft creation, approval-gated
// publication (GE-007), supersession, and invalidation. Published
// scorecards are never edited in place; every gated action lands in the
// append-only decision log.
package scorecard

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aaraminds/vria/internal/approval"
	"github.com/aaraminds/vria/internal/enums"
)

// Approval action types this service gates on (contracts/18 §3).
const (
	ActionPublication  = "ScorecardPublication"
	ActionSupersession = "ScorecardSupersession"
	ActionInvalidation = "ScorecardInvalidation"
)

var (
	ErrNotFound          = errors.New("not found")
	ErrUnknownAssessment = errors.New("unknown assessment id")
	ErrReasonRequired    = errors.New("invalidation requires a reason")
	ErrBadActionType     = errors.New("unsupported approval action type")
	// ErrApprovalRequired re-exports the gate error so callers can map it
	// to APPROVAL_REQUIRED without importing internal/approval.
	ErrApprovalRequired = approval.ErrApprovalRequired
)

// Period is the reporting period (contracts/17 §9).
type Period struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// Coverage is the evidence_coverage_summary (contracts/17 §9): every
// published scorecard must disclose how much of it is actually evidenced.
type Coverage struct {
	AssessmentsTotal int `json:"assessments_total"`
	WithCitations    int `json:"with_citations"`
	WithGaps         int `json:"with_gaps"`
}

// Scorecard mirrors contracts/17 §9.
type Scorecard struct {
	ScorecardID             string              `json:"scorecard_id"`
	Title                   string              `json:"title"`
	Period                  Period              `json:"period"`
	EvidenceCoverageSummary Coverage            `json:"evidence_coverage_summary"`
	ArtifactState           enums.ArtifactState `json:"artifact_state"`
	AssessmentIDs           []string            `json:"assessment_ids"`
	SupersedesScorecardID   string              `json:"supersedes_scorecard_id,omitempty"`
	DecisionLogPointer      string              `json:"decision_log_pointer,omitempty"`
	PublishedAt             *time.Time          `json:"published_at"`
	CreatedBy               string              `json:"created_by"`
	CreatedAt               time.Time           `json:"created_at"`
}

// AssessmentInfo is what coverage computation needs from a member
// assessment; with_citations = no missing evidence, with_gaps = the rest.
type AssessmentInfo struct {
	AssessmentID    string
	MissingEvidence []string
}

// AssessmentSource resolves member assessments without coupling this
// package to the assessment service.
type AssessmentSource func(assessmentID string) (AssessmentInfo, bool)

// AuditSink decouples this package from the audit store.
type AuditSink func(action, targetType, targetID, actorID string)

type Service struct {
	mu          sync.Mutex
	cards       map[string]*Scorecard
	requests    map[string]*approval.Request
	log         *approval.DecisionLog
	assessments AssessmentSource
	audit       AuditSink
	now         func() time.Time
	seq         int
}

func NewService(assessments AssessmentSource, audit AuditSink) *Service {
	if assessments == nil {
		assessments = func(string) (AssessmentInfo, bool) { return AssessmentInfo{}, false }
	}
	if audit == nil {
		audit = func(string, string, string, string) {}
	}
	return &Service{
		cards:       map[string]*Scorecard{},
		requests:    map[string]*approval.Request{},
		log:         &approval.DecisionLog{},
		assessments: assessments,
		audit:       audit,
		now:         func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) nextID(p string) string {
	s.seq++
	return fmt.Sprintf("%s-%06d", p, s.seq)
}

// CreateDraft assembles a draft scorecard from existing assessments and
// computes the evidence coverage summary. Every member assessment must
// resolve; a scorecard never references assessments that do not exist.
func (s *Service) CreateDraft(title string, period Period, assessmentIDs []string, actorID string) (Scorecard, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cov := Coverage{AssessmentsTotal: len(assessmentIDs)}
	for _, id := range assessmentIDs {
		info, ok := s.assessments(id)
		if !ok {
			return Scorecard{}, fmt.Errorf("%w: %s", ErrUnknownAssessment, id)
		}
		if len(info.MissingEvidence) == 0 {
			cov.WithCitations++
		} else {
			cov.WithGaps++
		}
	}
	c := &Scorecard{
		ScorecardID:             s.nextID("scd"),
		Title:                   title,
		Period:                  period,
		EvidenceCoverageSummary: cov,
		ArtifactState:           enums.ArtDraft,
		AssessmentIDs:           append([]string(nil), assessmentIDs...),
		CreatedBy:               actorID,
		CreatedAt:               s.now(),
	}
	s.cards[c.ScorecardID] = c
	s.audit("scorecard.generated", "Scorecard", c.ScorecardID, actorID)
	return *c, nil
}

// Get returns one scorecard by ID.
func (s *Service) Get(scorecardID string) (Scorecard, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.cards[scorecardID]
	if !ok {
		return Scorecard{}, ErrNotFound
	}
	return *c, nil
}

// SubmitForApproval opens an approval request for a gated scorecard action
// (contracts/18 §4 submit_for_approval). Publication, supersession, and
// invalidation each need their own request.
func (s *Service) SubmitForApproval(scorecardID, actionType, requestedBy string) (approval.Request, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.cards[scorecardID]; !ok {
		return approval.Request{}, ErrNotFound
	}
	switch actionType {
	case ActionPublication, ActionSupersession, ActionInvalidation:
	case "":
		actionType = ActionPublication
	default:
		return approval.Request{}, fmt.Errorf("%w: %s", ErrBadActionType, actionType)
	}
	// The "scapr" prefix keeps scorecard approval IDs disjoint from the
	// hypothesis service's "apr" namespace; the shared decision endpoint
	// routes by ID across both services.
	req := &approval.Request{
		ApprovalID: s.nextID("scapr"), ActionType: actionType,
		TargetID: scorecardID, TargetType: "Scorecard",
		RequestedBy: requestedBy, State: enums.ReqSubmitted,
	}
	s.requests[req.ApprovalID] = req
	s.audit("approval.submitted", "ApprovalRequest", req.ApprovalID, requestedBy)
	return *req, nil
}

// Decide applies an approver decision via the request state machine.
// decidedBy always comes from the authenticated principal, never payload
// data (contracts/18 §4). Approving a publication request also moves the
// draft artifact to Approved (contracts/18 §2.2).
func (s *Service) Decide(approvalID, action, decidedBy string) (approval.Request, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	req, ok := s.requests[approvalID]
	if !ok {
		return approval.Request{}, ErrNotFound
	}
	// Separation of duties: the requester cannot approve their own scorecard
	// action (contracts/18 §3).
	if err := approval.CheckApprover(req, action, decidedBy); err != nil {
		return *req, err
	}
	next, err := approval.RequestTransition(req.State, action)
	if err != nil {
		return *req, err
	}
	req.State = next
	req.DecidedBy = decidedBy
	if next == enums.ReqApproved && req.ActionType == ActionPublication {
		if c, ok := s.cards[req.TargetID]; ok && c.ArtifactState == enums.ArtDraft {
			if st, aerr := approval.ArtifactTransition(c.ArtifactState, "approve"); aerr == nil {
				c.ArtifactState = st
			}
		}
	}
	s.audit("approval.decided", "ApprovalRequest", approvalID, decidedBy)
	return *req, nil
}

// Publish executes publication under the GE-007 gate: only an Approved
// ScorecardPublication request for this scorecard unlocks it. On success
// the decision lands in the append-only decision log and the scorecard
// carries the log pointer (contracts/21 §5 scorecard.published).
func (s *Service) Publish(scorecardID, approvalID, actorID string) (Scorecard, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.cards[scorecardID]
	if !ok {
		return Scorecard{}, ErrNotFound
	}
	var req *approval.Request
	if r, ok := s.requests[approvalID]; ok {
		cp := *r
		req = &cp
	}
	next, err := approval.PublishScorecard(scorecardID, c.ArtifactState, req)
	if err != nil {
		return *c, err // no publication, no state change
	}
	now := s.now()
	rec := approval.DecisionRecord{
		DecisionRecordID: s.nextID("dec"),
		DecisionType:     ActionPublication,
		TargetID:         scorecardID,
		TargetType:       "Scorecard",
		Decision:         "Approved",
		DecidedBy:        req.DecidedBy,
		ApprovalID:       approvalID,
		CreatedAt:        now,
	}
	if err := s.log.Append(rec); err != nil {
		return *c, err
	}
	c.ArtifactState = next
	c.PublishedAt = &now
	c.DecisionLogPointer = rec.DecisionRecordID
	s.audit("scorecard.published", "Scorecard", scorecardID, actorID)
	return *c, nil
}

// Supersede retires a scorecard in favor of a replacement, gated by an
// Approved ScorecardSupersession request for the old scorecard. The
// replacement gets the supersedes link; the published original is never
// edited in place (contracts/18 §4 supersede_scorecard).
func (s *Service) Supersede(oldID, replacementID, approvalID, actorID string) (Scorecard, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.cards[oldID]
	if !ok {
		return Scorecard{}, ErrNotFound
	}
	repl, ok := s.cards[replacementID]
	if !ok {
		return Scorecard{}, fmt.Errorf("%w: replacement %s", ErrNotFound, replacementID)
	}
	req, ok := s.requests[approvalID]
	if !ok || req.State != enums.ReqApproved ||
		req.ActionType != ActionSupersession || req.TargetID != oldID {
		return *old, ErrApprovalRequired
	}
	next, err := approval.ArtifactTransition(old.ArtifactState, "supersede")
	if err != nil {
		return *old, err
	}
	now := s.now()
	rec := approval.DecisionRecord{
		DecisionRecordID: s.nextID("dec"),
		DecisionType:     ActionSupersession,
		TargetID:         oldID,
		TargetType:       "Scorecard",
		Decision:         "Approved",
		Rationale:        "superseded by " + replacementID,
		DecidedBy:        req.DecidedBy,
		ApprovalID:       approvalID,
		CreatedAt:        now,
	}
	if err := s.log.Append(rec); err != nil {
		return *old, err
	}
	old.ArtifactState = next
	repl.SupersedesScorecardID = oldID
	s.audit("scorecard.superseded", "Scorecard", oldID, actorID)
	return *repl, nil
}

// Invalidate retires a published scorecard without replacement, gated by
// an Approved ScorecardInvalidation request. A reason is mandatory and is
// preserved in the immutable decision record (contracts/18 §4).
func (s *Service) Invalidate(scorecardID, approvalID, reason, actorID string) (Scorecard, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.cards[scorecardID]
	if !ok {
		return Scorecard{}, ErrNotFound
	}
	if reason == "" {
		return *c, ErrReasonRequired
	}
	req, ok := s.requests[approvalID]
	if !ok || req.State != enums.ReqApproved ||
		req.ActionType != ActionInvalidation || req.TargetID != scorecardID {
		return *c, ErrApprovalRequired
	}
	next, err := approval.ArtifactTransition(c.ArtifactState, "invalidate")
	if err != nil {
		return *c, err
	}
	now := s.now()
	rec := approval.DecisionRecord{
		DecisionRecordID: s.nextID("dec"),
		DecisionType:     ActionInvalidation,
		TargetID:         scorecardID,
		TargetType:       "Scorecard",
		Decision:         "Approved",
		Rationale:        reason,
		DecidedBy:        req.DecidedBy,
		ApprovalID:       approvalID,
		CreatedAt:        now,
	}
	if err := s.log.Append(rec); err != nil {
		return *c, err
	}
	c.ArtifactState = next
	s.audit("scorecard.invalidated", "Scorecard", scorecardID, actorID)
	return *c, nil
}

// DecisionRecords exposes the append-only decision log (read-only copy).
func (s *Service) DecisionRecords() []approval.DecisionRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.log.Records()
}
