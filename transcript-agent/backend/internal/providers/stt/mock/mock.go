// Package mock is a deterministic fake STT provider. It emits a plausible
// two-speaker podcast (~2 minutes) with mixed confidences, including several
// segments below the 0.80 default threshold, so the low-confidence review
// path (PRD R5) is exercisable end-to-end without a real provider.
package mock

import (
	"context"
	"fmt"
	"hash/fnv"
	"math/rand"
	"strings"
	"sync"

	"github.com/aaraminds/transcript-agent/internal/domain"
	"github.com/aaraminds/transcript-agent/internal/providers/stt"
)

// Provider is the deterministic mock STT provider.
type Provider struct {
	mu    sync.Mutex
	calls int
}

// New returns a mock provider.
func New() *Provider { return &Provider{} }

type line struct {
	speaker string
	text    string
	conf    float64
	durMS   int
}

// script is a fixed two-speaker podcast conversation with deliberate filler
// words (um, uh, you know, like) so the cleanup policy (PRD 15.2) has real
// work to do, and mixed confidences including values below 0.80.
var script = []line{
	{"Speaker 1", "Um, welcome back to the show, you know, today we are talking about shipping small reliable systems.", 0.94, 5200},
	{"Speaker 2", "Thanks for having me, uh, it is great to be here again.", 0.91, 3800},
	{"Speaker 1", "So, like, let's start with the basics, what does a minimum viable pipeline actually look like?", 0.88, 5400},
	{"Speaker 2", "Uh, in my experience it is three things, intake, processing, and a human approval gate.", 0.93, 5600},
	{"Speaker 1", "Right, and, you know, the approval gate is the part most teams skip first.", 0.86, 4400},
	{"Speaker 2", "Which is exactly backwards, because trust is the whole product.", 0.95, 4200},
	{"Speaker 1", "Um, can you give a concrete example from a project you shipped?", 0.90, 4000},
	{"Speaker 2", "Sure, we built a transcript workflow where every export required an approved version.", 0.72, 5400},
	{"Speaker 1", "That sounds slow though, did reviewers push back on the extra step?", 0.89, 4600},
	{"Speaker 2", "Uh, at first yes, but the audit trail saved us in the first compliance review.", 0.78, 5200},
	{"Speaker 1", "You know, that matches what we hear from other teams about governance.", 0.84, 4200},
	{"Speaker 2", "The other lesson is to keep raw output immutable, never edit the source of truth.", 0.96, 5000},
	{"Speaker 1", "Um, how do you handle sections where the audio quality drops?", 0.87, 4200},
	{"Speaker 2", "We flag low confidence segments and route reviewer attention there first.", 0.68, 4800},
	{"Speaker 1", "So the machine does the pass and the human does the judgment.", 0.92, 4200},
	{"Speaker 2", "Exactly, and, uh, that division of labor is what makes the effort reduction real.", 0.83, 5000},
	{"Speaker 1", "Like, what about summaries, do you generate those automatically too?", 0.79, 4400},
	{"Speaker 2", "Yes, but grounded only in transcript content, no invented claims allowed.", 0.94, 4600},
	{"Speaker 1", "Um, that seems like the right guardrail for derived content.", 0.85, 4000},
	{"Speaker 2", "It is, and every summary records which transcript version it came from.", 0.91, 4400},
	{"Speaker 1", "You know, before we wrap up, what is the one thing listeners should do tomorrow?", 0.88, 5000},
	{"Speaker 2", "Write down your approval boundary, uh, decide what a human must sign off on.", 0.75, 5200},
	{"Speaker 1", "That is a great note to end on, thanks so much for joining us.", 0.93, 4400},
	{"Speaker 2", "Thank you, this was fun.", 0.97, 2600},
}

// Transcribe returns the deterministic script. Special audio URIs trigger
// provider failure modes for testing the retry matrix (PRD 19):
//
//	contains "stt-timeout-once"  -> STT_PROVIDER_TIMEOUT on the first call only
//	contains "stt-quota"         -> STT_PROVIDER_QUOTA_EXCEEDED
//	contains "no-diarization"    -> diarization unavailable, single speaker
func (p *Provider) Transcribe(_ context.Context, audioArtifactURI, language string, enableDiarization bool) (*stt.Result, error) {
	if language != "en" {
		return nil, domain.E(domain.CodeLanguageUnsupported, "mock STT supports language 'en' only, got %q", language)
	}
	if strings.Contains(audioArtifactURI, "stt-quota") {
		return nil, domain.E(domain.CodeSTTProviderQuotaExceeded, "mock STT quota exhausted")
	}
	p.mu.Lock()
	p.calls++
	calls := p.calls
	p.mu.Unlock()
	if strings.Contains(audioArtifactURI, "stt-timeout-once") && calls == 1 {
		return nil, domain.E(domain.CodeSTTProviderTimeout, "mock STT timed out (transient)")
	}

	diarized := enableDiarization && !strings.Contains(audioArtifactURI, "no-diarization")
	res := &stt.Result{
		Provider:             "mock",
		Model:                "mock-stt-v1",
		RequestID:            fmt.Sprintf("mock-req-%06d", calls),
		DiarizationAvailable: diarized,
	}
	// Deterministic pseudo-random confidence jitter seeded by the audio
	// artifact URI (which embeds the job ID), clamped to [0.55, 0.99]. The
	// jitter is at most ±0.04 so the script's below-threshold segments (≤0.78)
	// always stay under the 0.80 default threshold.
	h := fnv.New64a()
	h.Write([]byte(audioArtifactURI))
	rng := rand.New(rand.NewSource(int64(h.Sum64())))
	cursor := 0
	for _, l := range script {
		speaker := l.speaker
		if !diarized {
			speaker = "Speaker 1"
		}
		conf := l.conf + (rng.Float64()-0.5)*0.08
		if conf < 0.55 {
			conf = 0.55
		}
		if conf > 0.99 {
			conf = 0.99
		}
		res.Segments = append(res.Segments, stt.Segment{
			StartMS:      cursor,
			EndMS:        cursor + l.durMS,
			SpeakerLabel: speaker,
			Text:         l.text,
			Confidence:   conf,
		})
		cursor += l.durMS + 200 // small inter-segment gap
	}
	return res, nil
}

var _ stt.Provider = (*Provider)(nil)
