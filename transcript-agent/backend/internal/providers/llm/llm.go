// Package llm defines the LLM provider interface used for transcript cleanup
// (PRD R6, 14.8, 15.2) and summary generation (PRD R10, 14.11, 15.3).
package llm

import "context"

// CleanupStats reports what normalization changed (14.8 change_summary).
type CleanupStats struct {
	SegmentsProcessed     int
	FillerWordsRemoved    int
	MeaningChangeDetected bool
}

// Provider performs the two LLM tasks in MVP scope. Cleanup receives and
// returns per-segment texts positionally so timestamps and speaker labels are
// preserved by construction (PRD 15.1 rule 6).
type Provider interface {
	// Cleanup applies the documented cleanup policy to each segment text.
	// It must never paraphrase, reorder, or change meaning.
	Cleanup(ctx context.Context, segmentTexts []string, stylePolicyID string) ([]string, CleanupStats, error)
	// Summarize produces a transcript-grounded summary of at most maxWords.
	Summarize(ctx context.Context, transcriptText string, maxWords int, style string) (string, error)
}
