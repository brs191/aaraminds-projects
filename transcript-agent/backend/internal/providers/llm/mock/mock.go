// Package mock is a deterministic rule-based LLM provider. Cleanup applies
// exactly the allowed edits from PRD 15.2 (remove standalone um/uh/you know/
// like, tidy punctuation and capitalization) and never paraphrases. Summarize
// is purely extractive: it copies leading sentences from the transcript up to
// the configured word budget, so every summary claim is grounded by
// construction (PRD 15.3).
package mock

import (
	"context"
	"strings"
	"unicode"

	"github.com/aaraminds/transcript-agent/internal/domain"
	"github.com/aaraminds/transcript-agent/internal/providers/llm"
)

// StylePolicyDefault is the only style policy the mock knows (PRD job_config
// default default-clean-v1). Any other policy id fails with
// STYLE_POLICY_NOT_FOUND per 14.8.
const StylePolicyDefault = "default-clean-v1"

// Provider is the rule-based mock LLM.
type Provider struct{}

// New returns the mock provider.
func New() *Provider { return &Provider{} }

func trimCore(w string) string {
	return strings.ToLower(strings.TrimFunc(w, func(r rune) bool {
		return unicode.IsPunct(r)
	}))
}

func endsClause(w string) bool {
	return w == "" || strings.HasSuffix(w, ",") || strings.HasSuffix(w, ".") ||
		strings.HasSuffix(w, "!") || strings.HasSuffix(w, "?") || strings.HasSuffix(w, ";")
}

// cleanText removes allowed fillers from one segment text and returns the
// cleaned text plus the number of filler words removed.
func cleanText(text string) (string, int) {
	words := strings.Fields(text)
	var out []string
	removed := 0
	prev := func() string {
		if len(out) == 0 {
			return ""
		}
		return out[len(out)-1]
	}
	for i := 0; i < len(words); i++ {
		w := words[i]
		core := trimCore(w)
		// "um"/"uh" are always non-semantic disfluencies.
		if core == "um" || core == "uh" {
			removed++
			continue
		}
		// "you know," when set off as an aside (clause boundary before,
		// comma after) is clearly non-semantic.
		if core == "you" && i+1 < len(words) {
			next := words[i+1]
			if trimCore(next) == "know" && strings.HasSuffix(next, ",") && endsClause(prev()) {
				removed += 2
				i++
				continue
			}
		}
		// "like," standalone between clause boundaries.
		if core == "like" && strings.HasSuffix(w, ",") && endsClause(prev()) {
			removed++
			continue
		}
		out = append(out, w)
	}
	if len(out) == 0 {
		// Never silently drop an entire segment (PRD 15.1 rule 3).
		return text, 0
	}
	cleaned := strings.Join(out, " ")
	// Capitalization fix: first letter uppercase (allowed edit per 15.2).
	r := []rune(cleaned)
	r[0] = unicode.ToUpper(r[0])
	return string(r), removed
}

// contentWords returns the lowercase word set of a text (punctuation trimmed).
func contentWords(text string) map[string]bool {
	set := map[string]bool{}
	for _, w := range strings.Fields(text) {
		if c := trimCore(w); c != "" {
			set[c] = true
		}
	}
	return set
}

// Cleanup applies the cleanup policy per segment. Output is positional:
// index i of the result is the cleaned text of segmentTexts[i], so the caller
// preserves timestamps and speaker labels exactly.
func (p *Provider) Cleanup(_ context.Context, segmentTexts []string, stylePolicyID string) ([]string, llm.CleanupStats, error) {
	if stylePolicyID != StylePolicyDefault {
		return nil, llm.CleanupStats{}, domain.E(domain.CodeStylePolicyNotFound,
			"style policy %q not found; known: %s", stylePolicyID, StylePolicyDefault)
	}
	out := make([]string, len(segmentTexts))
	stats := llm.CleanupStats{SegmentsProcessed: len(segmentTexts)}
	for i, t := range segmentTexts {
		cleaned, removed := cleanText(t)
		// Meaning guard: every cleaned word must exist in the original.
		orig := contentWords(t)
		for w := range contentWords(cleaned) {
			if !orig[w] {
				stats.MeaningChangeDetected = true
			}
		}
		out[i] = cleaned
		stats.FillerWordsRemoved += removed
	}
	return out, stats, nil
}

// Summarize returns the leading sentences of the transcript, truncated at the
// word budget. Extractive by design so the grounding check always passes.
func (p *Provider) Summarize(_ context.Context, transcriptText string, maxWords int, style string) (string, error) {
	_ = style // mock ignores style; real providers apply neutral-professional etc.
	if maxWords <= 0 {
		maxWords = 150
	}
	words := strings.Fields(transcriptText)
	if len(words) == 0 {
		return "", domain.E(domain.CodeLLMOutputInvalid, "empty transcript text")
	}
	if len(words) > maxWords {
		words = words[:maxWords]
	}
	s := strings.Join(words, " ")
	if !strings.HasSuffix(s, ".") {
		s += "..."
	}
	return s, nil
}

var _ llm.Provider = (*Provider)(nil)
