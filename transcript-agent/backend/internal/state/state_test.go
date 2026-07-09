package state

import (
	"testing"

	"github.com/aaraminds/transcript-agent/internal/domain"
)

func TestLegalTransitions(t *testing.T) {
	legal := [][2]domain.Status{
		{domain.StatusSubmitted, domain.StatusQueued},
		{domain.StatusQueued, domain.StatusValidating},
		{domain.StatusValidating, domain.StatusMetadataExtracted},
		{domain.StatusValidating, domain.StatusNeedsUserAction},
		{domain.StatusMetadataExtracted, domain.StatusCaptionChecked},
		{domain.StatusMetadataExtracted, domain.StatusExtractingAudio},
		{domain.StatusCaptionChecked, domain.StatusNeedsUserAction},
		{domain.StatusCaptionChecked, domain.StatusNormalizing}, // caption reuse skips STT
		{domain.StatusCaptionChecked, domain.StatusExtractingAudio},
		{domain.StatusNeedsUserAction, domain.StatusQueued},         // replace media
		{domain.StatusNeedsUserAction, domain.StatusCaptionChecked}, // caption decision
		{domain.StatusExtractingAudio, domain.StatusTranscribing},
		{domain.StatusExtractingAudio, domain.StatusNeedsUserAction},
		{domain.StatusTranscribing, domain.StatusNormalizing},
		{domain.StatusTranscribing, domain.StatusQueued}, // quota exhaustion (14.7)
		{domain.StatusNormalizing, domain.StatusQualityChecking},
		{domain.StatusQualityChecking, domain.StatusDrafted},
		{domain.StatusDrafted, domain.StatusInReview},
		{domain.StatusInReview, domain.StatusApproved},
		{domain.StatusApproved, domain.StatusExported},
		{domain.StatusApproved, domain.StatusInReview}, // reopen (11.4)
		{domain.StatusExported, domain.StatusInReview}, // reopen (11.4)
		{domain.StatusSubmitted, domain.StatusCancelled},
		{domain.StatusTranscribing, domain.StatusCancelled},
		{domain.StatusApproved, domain.StatusCancelled},
	}
	for _, pair := range legal {
		job := &domain.Job{Status: pair[0]}
		if err := Transition(job, pair[1]); err != nil {
			t.Errorf("expected %s -> %s legal, got %v", pair[0], pair[1], err)
		}
		if job.Status != pair[1] {
			t.Errorf("job status not updated for %s -> %s", pair[0], pair[1])
		}
	}
}

func TestIllegalTransitions(t *testing.T) {
	illegal := [][2]domain.Status{
		{domain.StatusSubmitted, domain.StatusApproved},
		{domain.StatusSubmitted, domain.StatusInReview},
		{domain.StatusQueued, domain.StatusTranscribing},
		{domain.StatusInReview, domain.StatusExported}, // export requires approval first
		{domain.StatusDrafted, domain.StatusApproved},  // approval only from in_review
		{domain.StatusFailed, domain.StatusQueued},     // failed is terminal (R9)
		{domain.StatusCancelled, domain.StatusQueued},  // cancelled is terminal
		{domain.StatusApproved, domain.StatusDrafted},
		{domain.StatusExported, domain.StatusApproved},
		{domain.StatusNormalizing, domain.StatusInReview},
	}
	for _, pair := range illegal {
		job := &domain.Job{Status: pair[0]}
		err := Transition(job, pair[1])
		if err == nil {
			t.Errorf("expected %s -> %s illegal", pair[0], pair[1])
			continue
		}
		if domain.CodeOf(err) != domain.CodeInvalidStateTransition {
			t.Errorf("expected INVALID_STATE_TRANSITION, got %s", domain.CodeOf(err))
		}
		if job.Status != pair[0] {
			t.Errorf("job status mutated on illegal transition %s -> %s", pair[0], pair[1])
		}
	}
}

func TestUnknownStatusRejected(t *testing.T) {
	job := &domain.Job{Status: domain.StatusQueued}
	if err := Transition(job, domain.Status("bogus")); err == nil {
		t.Fatal("expected error for unknown status")
	}
}

func TestTerminal(t *testing.T) {
	if !domain.StatusFailed.Terminal() || !domain.StatusCancelled.Terminal() {
		t.Fatal("failed/cancelled must be terminal")
	}
	if domain.StatusNeedsUserAction.Terminal() {
		t.Fatal("needs_user_action is a resumable state, not terminal (R9)")
	}
}
