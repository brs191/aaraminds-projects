// Package mock is the deterministic caption provider for MODE=mock.
// A source_uri containing "captions=1" reports authorized official captions,
// which makes the orchestrator pause at needs_user_action/caption_decision
// (PRD 11.1 step 6).
package mock

import (
	"context"
	"strings"

	"github.com/aaraminds/transcript-agent/internal/domain"
	"github.com/aaraminds/transcript-agent/internal/providers/captions"
)

// TrackID is the deterministic official track id served by the mock.
const TrackID = "mock-track-en-official"

// Provider is the mock caption provider.
type Provider struct{}

// New returns the mock provider.
func New() *Provider { return &Provider{} }

func (p *Provider) Check(_ context.Context, sourceURI, language string) (*captions.CheckResult, error) {
	if strings.Contains(sourceURI, "caption-api-down") {
		return nil, domain.E(domain.CodeCaptionAPIUnavailable, "mock caption API unavailable")
	}
	res := &captions.CheckResult{AutoCaptionsFound: true}
	if strings.Contains(sourceURI, "captions=1") {
		res.OfficialCaptionsFound = true
		res.DownloadAuthorized = true
		res.Tracks = append(res.Tracks, captions.Track{
			CaptionTrackID: TrackID, Language: language,
			CaptionType: "official", Downloadable: true,
		})
	}
	// Auto-generated captions are reported but never reusable (PRD R2).
	res.Tracks = append(res.Tracks, captions.Track{
		CaptionTrackID: "mock-track-en-auto", Language: language,
		CaptionType: "auto_generated", Downloadable: false,
	})
	return res, nil
}

// mockVTT is a deterministic official caption file. No speaker cues, so the
// parsed transcript defaults to Speaker 1 with caption-origin flags (14.5).
const mockVTT = `WEBVTT

00:00:00.000 --> 00:00:04.500
Welcome back to the show, um, today we look at caption reuse.

00:00:04.700 --> 00:00:09.200
When official captions already exist, transcription cost can be avoided.

00:00:09.400 --> 00:00:14.100
The captions are parsed into the same transcript structures as speech to text.

00:00:14.300 --> 00:00:18.900
Caption derived segments carry no confidence scores at all.

00:00:19.100 --> 00:00:23.600
So the quality report marks the transcript confidence unavailable.

00:00:23.800 --> 00:00:28.400
Reviewers still edit, relabel speakers, and approve as usual.

00:00:28.600 --> 00:00:33.000
Exports are generated only from the approved version.

00:00:33.200 --> 00:00:36.800
And the audit trail records the caption reuse decision.
`

func (p *Provider) Fetch(_ context.Context, captionTrackID, format string) ([]byte, error) {
	if captionTrackID != TrackID {
		return nil, domain.E(domain.CodeCaptionTrackNotFound, "caption track %q not found", captionTrackID)
	}
	if format != "vtt" {
		return nil, domain.E(domain.CodeCaptionFormatUnsupported, "mock provider serves vtt only, got %q", format)
	}
	return []byte(mockVTT), nil
}

var _ captions.Provider = (*Provider)(nil)
