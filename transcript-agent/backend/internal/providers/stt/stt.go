// Package stt defines the speech-to-text provider interface (PRD R4, 14.7).
package stt

import "context"

// Segment is one diarized, timestamped transcript segment from the provider.
type Segment struct {
	StartMS      int
	EndMS        int
	SpeakerLabel string
	Text         string
	Confidence   float64
}

// Result is the provider transcription output plus provider metadata that is
// recorded in the audit trail (PRD 18.1: provider request IDs).
type Result struct {
	Segments             []Segment
	Provider             string
	Model                string
	RequestID            string
	DiarizationAvailable bool
}

// Provider runs batch transcription with timestamps, diarization and
// segment confidence scores.
type Provider interface {
	Transcribe(ctx context.Context, audioArtifactURI, language string, enableDiarization bool) (*Result, error)
}

// SpeakerHinter is an optional Provider extension for backends whose
// diarization accepts an expected speaker count (job_config
// expected_speaker_count, PRD 13.3). The transcribe_audio tool prefers this
// method when the provider implements it; expectedSpeakerCount is nil when
// the job config carries no value.
type SpeakerHinter interface {
	TranscribeWithSpeakerHint(ctx context.Context, audioArtifactURI, language string, enableDiarization bool, expectedSpeakerCount *int) (*Result, error)
}
