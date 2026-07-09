package mock

import (
	"context"
	"strings"
	"testing"

	"github.com/aaraminds/transcript-agent/internal/domain"
)

func TestCleanupRemovesFillers(t *testing.T) {
	p := New()
	in := []string{
		"Um, welcome back to the show, you know, today we are talking about systems.",
		"Thanks for having me, uh, it is great to be here.",
		"So, like, let's start with the basics.",
		"The other lesson is to keep raw output immutable.",
	}
	out, stats, err := p.Cleanup(context.Background(), in, StylePolicyDefault)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != len(in) {
		t.Fatalf("cleanup must be positional: got %d texts for %d segments", len(out), len(in))
	}
	for i, cleaned := range out {
		low := " " + strings.ToLower(cleaned) + " "
		for _, filler := range []string{" um ", " um, ", " uh ", " uh, ", " you know, ", ", like, "} {
			if strings.Contains(low, filler) {
				t.Errorf("segment %d still contains filler %q: %q", i, filler, cleaned)
			}
		}
	}
	if out[3] != in[3] {
		t.Errorf("segment without fillers must be unchanged, got %q", out[3])
	}
	if stats.FillerWordsRemoved == 0 {
		t.Error("expected filler words removed count > 0")
	}
	if stats.SegmentsProcessed != len(in) {
		t.Errorf("segments processed = %d, want %d", stats.SegmentsProcessed, len(in))
	}
	if stats.MeaningChangeDetected {
		t.Error("cleanup flagged a meaning change on allowed edits")
	}
}

// TestCleanupPreservesMeaning: every non-filler word of the original must
// survive, and no new word may appear (word-subset check, PRD R6).
func TestCleanupPreservesMeaning(t *testing.T) {
	p := New()
	in := []string{
		"Um, we built a transcript workflow where every export required an approved version.",
		"You know, that matches what we hear from other teams about governance.",
	}
	out, _, err := p.Cleanup(context.Background(), in, StylePolicyDefault)
	if err != nil {
		t.Fatal(err)
	}
	fillers := map[string]bool{"um": true, "uh": true, "you": true, "know": true, "like": true}
	for i := range in {
		origSet := contentWords(in[i])
		cleanSet := contentWords(out[i])
		// no invented words
		for w := range cleanSet {
			if !origSet[w] {
				t.Errorf("cleanup invented word %q in %q", w, out[i])
			}
		}
		// no non-filler word lost
		for w := range origSet {
			if !fillers[w] && !cleanSet[w] {
				t.Errorf("cleanup dropped non-filler word %q from %q", w, in[i])
			}
		}
	}
}

func TestCleanupUnknownStylePolicy(t *testing.T) {
	p := New()
	_, _, err := p.Cleanup(context.Background(), []string{"hello"}, "no-such-policy")
	if domain.CodeOf(err) != domain.CodeStylePolicyNotFound {
		t.Fatalf("want STYLE_POLICY_NOT_FOUND, got %v", err)
	}
}

func TestSummarizeRespectsBudgetAndGrounding(t *testing.T) {
	p := New()
	transcript := strings.Repeat("the pipeline needs an approval gate before export. ", 60)
	sum, err := p.Summarize(context.Background(), transcript, 50, "neutral-professional")
	if err != nil {
		t.Fatal(err)
	}
	if n := len(strings.Fields(sum)); n > 50 {
		t.Errorf("summary word count %d exceeds budget 50", n)
	}
	tset := contentWords(transcript)
	for w := range contentWords(sum) {
		if !tset[w] {
			t.Errorf("summary contains ungrounded word %q", w)
		}
	}
}
