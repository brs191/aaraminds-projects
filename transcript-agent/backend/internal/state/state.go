// Package state implements the deterministic job lifecycle state machine
// (PRD 11.1 workflow, 11.2 exceptions, 11.4 post-approval correction, 13.3
// canonical status enum). Model-free by design: high-risk transitions are
// never decided by an LLM (PRD 12.3).
package state

import (
	"time"

	"github.com/aaraminds/transcript-agent/internal/domain"
)

// allowed maps each status to the set of statuses it may transition to.
var allowed = map[domain.Status][]domain.Status{
	domain.StatusSubmitted: {domain.StatusQueued, domain.StatusCancelled},
	domain.StatusQueued:    {domain.StatusValidating, domain.StatusCancelled},
	domain.StatusValidating: {
		domain.StatusMetadataExtracted, domain.StatusNeedsUserAction,
		domain.StatusFailed, domain.StatusCancelled,
	},
	domain.StatusMetadataExtracted: {
		domain.StatusCaptionChecked,  // youtube path: caption pre-check
		domain.StatusExtractingAudio, // upload path: no caption check
		domain.StatusNeedsUserAction, domain.StatusFailed, domain.StatusCancelled,
	},
	domain.StatusCaptionChecked: {
		domain.StatusNeedsUserAction, // pause for producer caption decision
		domain.StatusExtractingAudio, // fresh transcription
		domain.StatusNormalizing,     // caption reuse: parsed raw goes straight to normalization (11.1 step 7)
		domain.StatusFailed, domain.StatusCancelled,
	},
	domain.StatusNeedsUserAction: {
		domain.StatusQueued,         // replace_job_media re-runs from the top (14.13)
		domain.StatusCaptionChecked, // caption decision recorded, resume pipeline
		domain.StatusFailed, domain.StatusCancelled,
	},
	domain.StatusExtractingAudio: {
		domain.StatusTranscribing, domain.StatusNeedsUserAction,
		domain.StatusFailed, domain.StatusCancelled,
	},
	domain.StatusTranscribing: {
		domain.StatusNormalizing,
		domain.StatusQueued, // STT quota exhaustion returns job to queued (14.7)
		domain.StatusNeedsUserAction, domain.StatusFailed, domain.StatusCancelled,
	},
	domain.StatusNormalizing: {
		domain.StatusQualityChecking, domain.StatusFailed, domain.StatusCancelled,
	},
	domain.StatusQualityChecking: {
		domain.StatusDrafted, domain.StatusFailed, domain.StatusCancelled,
	},
	domain.StatusDrafted: {
		domain.StatusInReview, domain.StatusFailed, domain.StatusCancelled,
	},
	domain.StatusInReview: {
		domain.StatusApproved, domain.StatusCancelled,
	},
	domain.StatusApproved: {
		domain.StatusExported,
		domain.StatusInReview, // reopen for post-approval correction (11.4)
		domain.StatusCancelled,
	},
	domain.StatusExported: {
		domain.StatusInReview, // reopen (11.4)
		domain.StatusCancelled,
	},
	// failed and cancelled are terminal.
	domain.StatusFailed:    {},
	domain.StatusCancelled: {},
}

// CanTransition reports whether from -> to is a legal lifecycle transition.
func CanTransition(from, to domain.Status) bool {
	for _, s := range allowed[from] {
		if s == to {
			return true
		}
	}
	return false
}

// Transition validates and applies a status change to the job, updating
// UpdatedAt. It returns INVALID_STATE_TRANSITION on illegal moves.
func Transition(job *domain.Job, to domain.Status) error {
	if !to.Valid() {
		return domain.E(domain.CodeInvalidStateTransition, "unknown status %q", to)
	}
	if !CanTransition(job.Status, to) {
		return domain.E(domain.CodeInvalidStateTransition,
			"illegal transition %s -> %s", job.Status, to)
	}
	job.Status = to
	job.UpdatedAt = time.Now().UTC()
	return nil
}
