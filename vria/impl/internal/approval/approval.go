// Package approval implements the two state machines in
// contracts/18_VRIA_Approval_Workflow_Spec.md §2.
package approval

import (
	"errors"
	"fmt"
	"time"

	"github.com/aaraminds/vria/internal/enums"
)

var (
	ErrInvalidTransition = errors.New("invalid transition")
	ErrApprovalRequired  = errors.New("action requires an Approved approval request")
	ErrTerminalState     = errors.New("state is terminal; open a new request")
)

// --- Approval request lifecycle (ApprovalRequestState) ---

var requestTransitions = map[enums.ApprovalRequestState]map[string]enums.ApprovalRequestState{
	enums.ReqDraft: {
		"submit_for_approval": enums.ReqSubmitted,
		"withdraw":            enums.ReqWithdrawn,
	},
	enums.ReqSubmitted: {
		"approve":         enums.ReqApproved,
		"reject":          enums.ReqRejected,
		"request_changes": enums.ReqChangesRequested,
	},
	enums.ReqChangesRequested: {
		"resubmit": enums.ReqSubmitted,
		"withdraw": enums.ReqWithdrawn,
	},
	// Approved, Rejected, Withdrawn are terminal.
}

// RequestTransition applies an action to an approval request.
// Rejected is terminal by design: revisions require a new request.
func RequestTransition(from enums.ApprovalRequestState, action string) (enums.ApprovalRequestState, error) {
	actions, ok := requestTransitions[from]
	if !ok {
		return from, fmt.Errorf("%w: %s", ErrTerminalState, from)
	}
	to, ok := actions[action]
	if !ok {
		return from, fmt.Errorf("%w: %s from %s", ErrInvalidTransition, action, from)
	}
	return to, nil
}

// --- Target artifact lifecycle (ArtifactState) ---

var artifactTransitions = map[enums.ArtifactState]map[string]enums.ArtifactState{
	enums.ArtDraft: {
		"approve": enums.ArtApproved,
	},
	enums.ArtApproved: {
		"publish":   enums.ArtPublished,
		"supersede": enums.ArtSuperseded,
	},
	enums.ArtPublished: {
		"supersede":  enums.ArtSuperseded,
		"invalidate": enums.ArtInvalidated,
	},
	// Superseded and Invalidated are terminal.
}

func ArtifactTransition(from enums.ArtifactState, action string) (enums.ArtifactState, error) {
	actions, ok := artifactTransitions[from]
	if !ok {
		return from, fmt.Errorf("%w: %s", ErrTerminalState, from)
	}
	to, ok := actions[action]
	if !ok {
		return from, fmt.Errorf("%w: %s from %s", ErrInvalidTransition, action, from)
	}
	return to, nil
}

// --- Publication gate (contracts/18 §4: publish_scorecard) ---

// Request mirrors contracts/17 §8 (ApprovalRequest). RequestedBy and
// ApproverIDs carry the separation-of-duties semantics the DB columns
// (migration 0003 requested_by / approver_ids) expect.
type Request struct {
	ApprovalID    string                     `json:"approval_id"`
	ActionType    string                     `json:"action_type"`
	TargetID      string                     `json:"target_id"`
	TargetType    string                     `json:"target_type,omitempty"`
	RequestedBy   string                     `json:"requested_by"`
	ApproverIDs   []string                   `json:"approver_ids,omitempty"`
	State         enums.ApprovalRequestState `json:"approval_state"`
	RiskTier      string                     `json:"risk_tier,omitempty"`
	Rationale     string                     `json:"rationale,omitempty"`
	DecidedBy     string                     `json:"decided_by,omitempty"`
	DecisionNotes string                     `json:"decision_comments,omitempty"`
}

// ErrSelfApproval is returned when the decider is the requester. Separation
// of duties (contracts/18 §3, gate-c-runtime/10 §3) forbids self-approval.
var ErrSelfApproval = errors.New("requester cannot approve their own request")

// ErrNotDesignatedApprover is returned when an ApproverIDs allowlist is set
// and the decider is not on it.
var ErrNotDesignatedApprover = errors.New("principal is not a designated approver for this request")

// CheckApprover enforces separation of duties for an approval decision.
// Applies only to the approve verb; reject/request_changes may come from any
// authorized reviewer. When ApproverIDs is populated it is treated as an
// allowlist (the real role check happens at the OIDC gateway; this is
// defense in depth).
func CheckApprover(req *Request, action, decidedBy string) error {
	if action != "approve" {
		return nil
	}
	if decidedBy == "" {
		return ErrApprovalRequired
	}
	if req.RequestedBy != "" && decidedBy == req.RequestedBy {
		return ErrSelfApproval
	}
	if len(req.ApproverIDs) > 0 {
		for _, a := range req.ApproverIDs {
			if a == decidedBy {
				return nil
			}
		}
		return ErrNotDesignatedApprover
	}
	return nil
}

// PublishScorecard enforces GE-007: publication may execute only with an
// Approved approval request of type ScorecardPublication for this target.
// Anything else must go through submit_for_approval — never direct execution.
func PublishScorecard(scorecardID string, artifact enums.ArtifactState, req *Request) (enums.ArtifactState, error) {
	if req == nil || req.State != enums.ReqApproved ||
		req.ActionType != "ScorecardPublication" || req.TargetID != scorecardID {
		return artifact, ErrApprovalRequired
	}
	return ArtifactTransition(artifact, "publish")
}

// --- Append-only decision log (contracts/17 §10, contracts/19 §5a) ---

// DecisionRecord mirrors contracts/17 §10. JSON tags emit the canonical
// snake_case names the schema and the decision-log API require.
type DecisionRecord struct {
	DecisionRecordID string    `json:"decision_record_id"`
	DecisionType     string    `json:"decision_type"`
	TargetID         string    `json:"target_id"`
	TargetType       string    `json:"target_type"`
	Decision         string    `json:"decision"`
	Rationale        string    `json:"rationale,omitempty"`
	DecidedBy        string    `json:"decided_by"`
	ApprovalID       string    `json:"approval_id"`
	CreatedAt        time.Time `json:"created_at"`
}

// DecisionLog exposes append and read only. There is deliberately no update
// or delete surface; the database revokes them as well.
type DecisionLog struct {
	records []DecisionRecord
}

func (l *DecisionLog) Append(r DecisionRecord) error {
	if r.DecidedBy == "" || r.ApprovalID == "" {
		return errors.New("decision record requires decided_by and approval_id")
	}
	l.records = append(l.records, r)
	return nil
}

func (l *DecisionLog) Records() []DecisionRecord {
	out := make([]DecisionRecord, len(l.records))
	copy(out, l.records)
	return out
}
