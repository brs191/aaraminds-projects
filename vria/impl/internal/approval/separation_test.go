package approval

import (
	"errors"
	"testing"
)

// Fix #2: the requester must not be able to approve their own request.
func TestCheckApproverRejectsSelfApproval(t *testing.T) {
	req := &Request{RequestedBy: "mallory"}
	if err := CheckApprover(req, "approve", "mallory"); !errors.Is(err, ErrSelfApproval) {
		t.Fatalf("self-approval must be rejected, got %v", err)
	}
	if err := CheckApprover(req, "approve", "reviewer"); err != nil {
		t.Fatalf("distinct approver must be allowed, got %v", err)
	}
}

// reject/request_changes are not approvals — separation does not apply.
func TestCheckApproverAllowsNonApproveVerbs(t *testing.T) {
	req := &Request{RequestedBy: "mallory"}
	for _, verb := range []string{"reject", "request_changes"} {
		if err := CheckApprover(req, verb, "mallory"); err != nil {
			t.Fatalf("%s by requester must be allowed, got %v", verb, err)
		}
	}
}

// An ApproverIDs allowlist, when set, is enforced as defense in depth.
func TestCheckApproverEnforcesAllowlist(t *testing.T) {
	req := &Request{RequestedBy: "owner", ApproverIDs: []string{"lead", "governance"}}
	if err := CheckApprover(req, "approve", "random"); !errors.Is(err, ErrNotDesignatedApprover) {
		t.Fatalf("non-listed approver must be rejected, got %v", err)
	}
	if err := CheckApprover(req, "approve", "governance"); err != nil {
		t.Fatalf("listed approver must be allowed, got %v", err)
	}
}
