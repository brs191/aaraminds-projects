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

type Request struct {
	ApprovalID string
	ActionType string
	TargetID   string
	State      enums.ApprovalRequestState
	DecidedBy  string
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

type DecisionRecord struct {
	DecisionRecordID string
	DecisionType     string
	TargetID         string
	Decision         string
	Rationale        string
	DecidedBy        string
	ApprovalID       string
	CreatedAt        time.Time
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
