package approval

import (
	"errors"
	"testing"

	"github.com/aaraminds/vria/internal/enums"
)

func TestRequestLifecycle(t *testing.T) {
	s, err := RequestTransition(enums.ReqDraft, "submit_for_approval")
	if err != nil || s != enums.ReqSubmitted {
		t.Fatalf("submit: %v %s", err, s)
	}
	s, err = RequestTransition(s, "request_changes")
	if err != nil || s != enums.ReqChangesRequested {
		t.Fatalf("request_changes: %v %s", err, s)
	}
	s, err = RequestTransition(s, "resubmit")
	if err != nil || s != enums.ReqSubmitted {
		t.Fatalf("resubmit: %v %s", err, s)
	}
	s, err = RequestTransition(s, "approve")
	if err != nil || s != enums.ReqApproved {
		t.Fatalf("approve: %v %s", err, s)
	}
}

func TestRejectedIsTerminal(t *testing.T) {
	s, _ := RequestTransition(enums.ReqSubmitted, "reject")
	if s != enums.ReqRejected {
		t.Fatalf("reject: %s", s)
	}
	if _, err := RequestTransition(s, "resubmit"); !errors.Is(err, ErrTerminalState) {
		t.Fatalf("Rejected must be terminal, got %v", err)
	}
}

func TestArtifactLifecycle(t *testing.T) {
	s, err := ArtifactTransition(enums.ArtDraft, "approve")
	if err != nil || s != enums.ArtApproved {
		t.Fatalf("approve: %v %s", err, s)
	}
	s, err = ArtifactTransition(s, "publish")
	if err != nil || s != enums.ArtPublished {
		t.Fatalf("publish: %v %s", err, s)
	}
	s, err = ArtifactTransition(s, "invalidate")
	if err != nil || s != enums.ArtInvalidated {
		t.Fatalf("invalidate: %v %s", err, s)
	}
	if _, err = ArtifactTransition(s, "publish"); !errors.Is(err, ErrTerminalState) {
		t.Fatal("Invalidated must be terminal")
	}
}

func TestNoWithdrawnFromPublished(t *testing.T) {
	if _, err := ArtifactTransition(enums.ArtPublished, "withdraw"); err == nil {
		t.Fatal("withdraw must not exist on the artifact lifecycle")
	}
}

func TestDecisionLogAppendOnly(t *testing.T) {
	var l DecisionLog
	if err := l.Append(DecisionRecord{DecisionRecordID: "d1", DecidedBy: "u1", ApprovalID: "a1"}); err != nil {
		t.Fatal(err)
	}
	if err := l.Append(DecisionRecord{DecisionRecordID: "d2"}); err == nil {
		t.Fatal("record without decided_by/approval_id must be rejected")
	}
	if len(l.Records()) != 1 {
		t.Fatalf("records = %d, want 1", len(l.Records()))
	}
}
