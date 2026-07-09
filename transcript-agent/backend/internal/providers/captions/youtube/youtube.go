// Package youtube is a config-gated skeleton client for the YouTube Data API
// v3 captions endpoints (PRD 14.3/14.4 verified API notes, July 2026):
//
//   - captions.list (GET https://www.googleapis.com/youtube/v3/captions?part=snippet&videoId=...)
//     returns caption-track metadata only, never caption text. Quota: 50 units.
//   - captions.download (GET https://www.googleapis.com/youtube/v3/captions/{id}?tfmt=vtt)
//     returns caption text. Quota: 200 units. Requires permission to edit the
//     video and an authorized OAuth scope (youtube.force-ssl or
//     youtubepartner) — so reuse is feasible only for owned/authorized
//     channel content; third-party videos must fall back to transcription.
//   - future captions.insert publishing costs 400 units and stays disabled in
//     MVP (14.15).
//
// Daily default quota is 10,000 units; at 1-10 episodes/month the caption
// pre-check (50) + download (200) is negligible, but the client should still
// surface quota errors as CAPTION_API_UNAVAILABLE for the retry path.
package youtube

import (
	"context"
	"net/http"
	"time"

	"github.com/aaraminds/transcript-agent/internal/domain"
	"github.com/aaraminds/transcript-agent/internal/providers/captions"
)

// Config holds OAuth material. Tokens live in a secrets manager (PRD 16.6);
// never log token values.
type Config struct {
	// OAuthToken is a short-lived bearer token minted from the stored
	// refresh token with minimum scopes.
	OAuthToken string
	// ChannelOwned confirms Phase 0 channel-ownership signoff (PRD 25).
	ChannelOwned bool
}

// Provider is the YouTube captions client skeleton.
type Provider struct {
	cfg    Config
	client *http.Client
}

// New returns the provider.
func New(cfg Config) *Provider {
	return &Provider{cfg: cfg, client: &http.Client{Timeout: 30 * time.Second}}
}

// Configured reports whether OAuth material is present.
func (p *Provider) Configured() bool { return p.cfg.OAuthToken != "" }

// Check would extract the video id from sourceURI, call captions.list, and
// classify tracks: snippet.trackKind == "asr" -> auto_generated (never
// reusable, PRD R2); otherwise official. download_authorized is true only
// when the channel is owned/authorized (cfg.ChannelOwned) — otherwise the
// caller falls back to transcription with VIDEO_NOT_OWNED semantics.
func (p *Provider) Check(ctx context.Context, sourceURI, language string) (*captions.CheckResult, error) {
	if !p.Configured() {
		return nil, domain.E(domain.CodeYouTubeAuthRequired,
			"youtube caption client not configured; connect the channel OAuth or continue to transcription")
	}
	_ = ctx
	return nil, domain.E(domain.CodeNotConfigured,
		"youtube captions.list client is a skeleton pending channel-ownership signoff; use MODE=mock")
}

// Fetch would call captions.download with tfmt mapped from format. HTTP 403
// maps to CAPTION_DOWNLOAD_UNAUTHORIZED (fall back to transcription), 404 to
// CAPTION_TRACK_NOT_FOUND.
func (p *Provider) Fetch(ctx context.Context, captionTrackID, format string) ([]byte, error) {
	if !p.Configured() {
		return nil, domain.E(domain.CodeCaptionDownloadUnauthorized,
			"youtube caption download not authorized; falling back to transcription")
	}
	_ = ctx
	return nil, domain.E(domain.CodeNotConfigured,
		"youtube captions.download client is a skeleton; use MODE=mock")
}

var _ captions.Provider = (*Provider)(nil)
