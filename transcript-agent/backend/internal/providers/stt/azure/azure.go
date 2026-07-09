// Package azure is a config-gated skeleton client for Azure AI Speech batch
// transcription (PRD 20.1: batch transcription billed per audio second;
// batch diarization included in Standard/Custom batch pricing).
//
// It is never called in MODE=mock. Wiring it in requires AZURE_SPEECH_KEY and
// AZURE_SPEECH_REGION plus STT_PROVIDER=azure.
//
// Batch flow (Speech to text REST API v3.2):
//  1. POST   https://{region}.api.cognitive.microsoft.com/speechtotext/v3.2/transcriptions
//     body: {"contentUrls":[audioURL],"locale":"en-US","displayName":...,
//     "properties":{"diarizationEnabled":true,"wordLevelTimestampsEnabled":true}}
//  2. GET    .../transcriptions/{id}         — poll status until Succeeded/Failed
//  3. GET    .../transcriptions/{id}/files   — list result files
//  4. GET    result file contentUrl          — recognizedPhrases[] with offsets,
//     durations, speaker numbers and confidence per phrase.
package azure

import (
	"context"
	"net/http"
	"time"

	"github.com/aaraminds/transcript-agent/internal/domain"
	"github.com/aaraminds/transcript-agent/internal/providers/stt"
)

// Config holds the Azure Speech resource settings. Key material must come
// from a secrets manager in production (PRD 16.6); env vars in dev only.
type Config struct {
	Region string // e.g. "eastus"
	Key    string // Azure Speech resource key
	Model  string // optional custom model/deployment id
}

// Provider is the Azure Speech batch transcription client skeleton.
type Provider struct {
	cfg    Config
	client *http.Client
}

// New returns the provider. It does not validate credentials eagerly.
func New(cfg Config) *Provider {
	return &Provider{cfg: cfg, client: &http.Client{Timeout: 60 * time.Second}}
}

// Configured reports whether the provider has enough config to be used.
func (p *Provider) Configured() bool { return p.cfg.Region != "" && p.cfg.Key != "" }

// Transcribe submits a batch transcription job and polls for the result.
//
// Skeleton status: request/poll/fetch wiring is intentionally not implemented
// until the STT bake-off decision (PRD section 25, target July 13 2026) picks
// the provider. The interface contract, error mapping, and audit metadata
// shape are final; only the HTTP calls remain.
func (p *Provider) Transcribe(ctx context.Context, audioArtifactURI, language string, enableDiarization bool) (*stt.Result, error) {
	if !p.Configured() {
		return nil, domain.E(domain.CodeNotConfigured,
			"azure STT provider is not configured (set AZURE_SPEECH_REGION and AZURE_SPEECH_KEY)")
	}
	// Implementation outline (see package doc for endpoints):
	//   1. Upload/point contentUrls at a SAS URL for the audio artifact.
	//   2. POST the transcription with diarizationEnabled=enableDiarization,
	//      locale mapped from language ("en" -> "en-US").
	//   3. Poll GET /transcriptions/{id} with backoff; map HTTP 429 to
	//      STT_PROVIDER_QUOTA_EXCEEDED and client timeouts to
	//      STT_PROVIDER_TIMEOUT so the orchestrator retry matrix (PRD 19)
	//      applies unchanged.
	//   4. Fetch recognizedPhrases and map to stt.Segment: offsetInTicks/1e4
	//      -> StartMS, speaker -> "Speaker N", confidence per phrase.
	_ = ctx
	return nil, domain.E(domain.CodeNotConfigured,
		"azure batch transcription client is a skeleton pending STT provider bake-off; use MODE=mock")
}

var _ stt.Provider = (*Provider)(nil)
