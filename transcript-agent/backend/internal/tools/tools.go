// Package tools implements one Go function per MCP-style tool contract in
// PRD section 14. Every tool validates input, acts through providers/stores,
// emits an audit event, and returns structured error codes. Contract
// standards (PRD 14): no durable external writes, configuration is read from
// the job_config snapshot via job_config_id, low-confidence results are never
// hidden.
package tools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/aaraminds/transcript-agent/internal/audit"
	"github.com/aaraminds/transcript-agent/internal/domain"
	"github.com/aaraminds/transcript-agent/internal/objectstore"
	"github.com/aaraminds/transcript-agent/internal/providers/captions"
	"github.com/aaraminds/transcript-agent/internal/providers/llm"
	"github.com/aaraminds/transcript-agent/internal/providers/media"
	"github.com/aaraminds/transcript-agent/internal/providers/stt"
	"github.com/aaraminds/transcript-agent/internal/store"
)

// Toolset wires the tool functions to stores and providers.
type Toolset struct {
	Stores      store.Stores
	Objects     objectstore.ObjectStore
	STT         stt.Provider
	LLM         llm.Provider
	Media       media.Processor
	Captions    captions.Provider
	STTProvider string // provider name recorded in job_config snapshots
	// ConfigDefaults optionally overrides the PRD defaults used when creating
	// job_config snapshots (admin-tunable via env; see cmd/server).
	ConfigDefaults *domain.JobConfig
	Auditor        *audit.Writer
	Log            *slog.Logger
}

func (t *Toolset) log() *slog.Logger {
	if t.Log != nil {
		return t.Log
	}
	return slog.Default()
}

func (t *Toolset) auditor() *audit.Writer {
	if t.Auditor != nil {
		return t.Auditor
	}
	return audit.New(t.Stores.Audit, t.Log)
}

// Audit appends an audit event (PRD 13.3 audit_events, append-only) via the
// audit writer helper.
func (t *Toolset) Audit(ctx context.Context, jobID *uuid.UUID, actorType, actorID, eventType string, payload map[string]any) error {
	return t.auditor().Event(ctx, jobID, actorType, actorID, eventType, payload)
}

// URIHash returns a short hash of a URI for audit payloads (PRD 18.1: prefer
// hashed values over full media URLs).
func URIHash(uri string) string {
	sum := sha256.Sum256([]byte(uri))
	return hex.EncodeToString(sum[:8])
}

// UploadURIScheme is the only scheme accepted for real upload sources: it
// resolves to a staged artifact created by POST /api/v1/uploads, never to an
// arbitrary server path (audit M5). MockURIScheme is the test/demo escape
// hatch handled entirely by the deterministic stub providers.
const (
	UploadURIScheme = "upload://"
	MockURIScheme   = "mock://"
)

func validateSourceURI(sourceType, sourceURI string) error {
	if strings.TrimSpace(sourceURI) == "" {
		return domain.E(domain.CodeInvalidSourceURI, "source_uri is required")
	}
	switch sourceType {
	case domain.SourceYouTube:
		if !strings.Contains(sourceURI, "youtube.com") && !strings.Contains(sourceURI, "youtu.be") {
			return domain.E(domain.CodeInvalidSourceURI,
				"source_uri does not look like a YouTube URL: %s", sourceURI)
		}
	case domain.SourceUpload:
		switch {
		case strings.HasPrefix(sourceURI, UploadURIScheme):
			// Resolved against the staged-artifact store by the caller.
		case strings.HasPrefix(sourceURI, MockURIScheme):
			ext := strings.ToLower(strings.TrimPrefix(pathExt(sourceURI), "."))
			ok := false
			for _, s := range domain.SupportedUploadExtensions {
				if ext == s {
					ok = true
				}
			}
			if !ok {
				return domain.E(domain.CodeUnsupportedFormat,
					"this file type is not supported (got %q); supported: %s",
					ext, strings.Join(domain.SupportedUploadExtensions, ", "))
			}
		default:
			// Raw filesystem paths and file:// URIs are rejected outright.
			return domain.E(domain.CodeInvalidSourceURI,
				"upload source_uri must be an upload:// URI returned by POST /api/v1/uploads")
		}
	default:
		return domain.E(domain.CodeUnsupportedSourceType,
			"source_type must be %q or %q, got %q", domain.SourceYouTube, domain.SourceUpload, sourceType)
	}
	return nil
}

// ResolveUploadURI maps an upload://<uuid> URI to its staged source_media
// artifact. Anything that does not resolve is INVALID_SOURCE_URI.
func (t *Toolset) ResolveUploadURI(ctx context.Context, uri string) (*domain.MediaArtifact, error) {
	id, err := uuid.Parse(strings.TrimPrefix(uri, UploadURIScheme))
	if err != nil {
		return nil, domain.E(domain.CodeInvalidSourceURI, "invalid upload URI %q", uri)
	}
	art, err := t.Stores.Artifacts.GetArtifact(ctx, id)
	if err != nil || art.ArtifactType != domain.ArtifactSourceMedia {
		return nil, domain.E(domain.CodeInvalidSourceURI,
			"upload URI %q does not resolve to an uploaded media artifact", uri)
	}
	return art, nil
}

// mediaSourceURI returns the URI handed to the media processor: upload://
// sources resolve to the staged artifact's object-store URI; everything else
// (youtube URLs, mock:// markers) passes through unchanged.
func (t *Toolset) mediaSourceURI(ctx context.Context, job *domain.Job) (string, error) {
	if job.SourceType == domain.SourceUpload && strings.HasPrefix(job.SourceURI, UploadURIScheme) {
		art, err := t.ResolveUploadURI(ctx, job.SourceURI)
		if err != nil {
			return "", err
		}
		return art.URI, nil
	}
	return job.SourceURI, nil
}

func pathExt(uri string) string {
	base := strings.SplitN(uri, "?", 2)[0]
	if i := strings.LastIndex(base, "."); i >= 0 {
		return base[i:]
	}
	return ""
}

// ---------------------------------------------------------------------
// 14.1 submit_media_job
// ---------------------------------------------------------------------

// SubmitMediaJobInput is the 14.1 input contract.
type SubmitMediaJobInput struct {
	SourceType        string `json:"source_type"`
	SourceURI         string `json:"source_uri"`
	Language          string `json:"language"`
	SubmittedBy       string `json:"submitted_by"`
	OwnershipAttested bool   `json:"ownership_attested"`
}

// SubmitMediaJob creates a transcript job (PRD 14.1).
func (t *Toolset) SubmitMediaJob(ctx context.Context, in SubmitMediaJobInput) (*domain.Job, error) {
	if !in.OwnershipAttested {
		return nil, domain.E(domain.CodeOwnershipAttestationMissing,
			"please confirm ownership/licensing before submission")
	}
	if err := validateSourceURI(in.SourceType, in.SourceURI); err != nil {
		return nil, err
	}
	var staged *domain.MediaArtifact
	if in.SourceType == domain.SourceUpload && strings.HasPrefix(in.SourceURI, UploadURIScheme) {
		var err error
		if staged, err = t.ResolveUploadURI(ctx, in.SourceURI); err != nil {
			return nil, err
		}
	}
	lang := in.Language
	if lang == "" {
		lang = "en"
	}
	if lang != "en" {
		return nil, domain.E(domain.CodeLanguageUnsupported, "MVP is English-only (language=en), got %q", lang)
	}
	now := time.Now().UTC()
	job := &domain.Job{
		JobID:             uuid.New(),
		SourceType:        in.SourceType,
		SourceURI:         in.SourceURI,
		Status:            domain.StatusSubmitted,
		SubmittedBy:       in.SubmittedBy,
		OwnershipAttested: true,
		Language:          lang,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := t.Stores.Jobs.CreateJob(ctx, job); err != nil {
		return nil, err
	}
	if staged != nil {
		// Link the staged upload bytes to the job as its source_media artifact.
		if err := t.Stores.Artifacts.CreateArtifact(ctx, &domain.MediaArtifact{
			ArtifactID: uuid.New(), JobID: job.JobID,
			ArtifactType: domain.ArtifactSourceMedia, URI: staged.URI,
			MimeType: staged.MimeType, SizeBytes: staged.SizeBytes, CreatedAt: now,
		}); err != nil {
			return nil, err
		}
	}
	t.Audit(ctx, &job.JobID, "user", in.SubmittedBy, "job.submitted", map[string]any{
		"source_type":        in.SourceType,
		"source_uri_hash":    URIHash(in.SourceURI),
		"language":           lang,
		"ownership_attested": true,
	})
	return job, nil
}

// CreateConfigSnapshot snapshots job configuration during validating
// (PRD 13.2 rule 7). All later tools read config via job_config_id.
func (t *Toolset) CreateConfigSnapshot(ctx context.Context, job *domain.Job, createdBy string) (*domain.JobConfig, error) {
	cfg := domain.DefaultJobConfig(t.STTProvider)
	if t.ConfigDefaults != nil {
		cfg = *t.ConfigDefaults
		if cfg.STTProvider == "" {
			cfg.STTProvider = t.STTProvider
		}
	}
	if cfg.STTProvider == "" {
		cfg.STTProvider = "mock"
	}
	cfg.JobConfigID = uuid.New()
	cfg.JobID = job.JobID
	cfg.Language = job.Language
	cfg.CreatedBy = createdBy
	cfg.CreatedAt = time.Now().UTC()
	if err := t.Stores.Jobs.CreateJobConfig(ctx, &cfg); err != nil {
		return nil, err
	}
	updated, err := t.Stores.Jobs.TransitionJob(ctx, job.JobID, job.Status, func(j *domain.Job) error {
		j.JobConfigID = &cfg.JobConfigID
		return nil
	})
	if err != nil {
		return nil, err
	}
	*job = *updated
	t.Audit(ctx, &job.JobID, "system", createdBy, "job_config.snapshot_created", map[string]any{
		"job_config_id":        cfg.JobConfigID.String(),
		"confidence_threshold": cfg.ConfidenceThreshold,
		"enable_diarization":   cfg.EnableDiarization,
		"style_policy_id":      cfg.StylePolicyID,
		"summary_max_words":    cfg.SummaryMaxWords,
		"summary_style":        cfg.SummaryStyle,
		"stt_provider":         cfg.STTProvider,
	})
	return &cfg, nil
}

// Config loads the job's configuration snapshot.
func (t *Toolset) Config(ctx context.Context, job *domain.Job) (*domain.JobConfig, error) {
	if job.JobConfigID == nil {
		return nil, domain.E(domain.CodeValidationError, "job %s has no config snapshot yet", job.JobID)
	}
	return t.Stores.Jobs.GetJobConfig(ctx, *job.JobConfigID)
}

// ---------------------------------------------------------------------
// 14.2 get_media_metadata
// ---------------------------------------------------------------------

// GetMediaMetadata extracts media properties and updates job duration
// (PRD 14.2). NO_AUDIO_TRACK is returned as an error so the orchestrator can
// route to needs_user_action/replace_media.
func (t *Toolset) GetMediaMetadata(ctx context.Context, job *domain.Job) (*media.Metadata, error) {
	sourceURI, err := t.mediaSourceURI(ctx, job)
	if err != nil {
		t.Audit(ctx, &job.JobID, "tool", "get_media_metadata", "tool.get_media_metadata.failed",
			map[string]any{"error_code": domain.CodeOf(err), "error": err.Error()})
		return nil, err
	}
	meta, err := t.Media.Metadata(ctx, job.SourceType, sourceURI)
	if err != nil {
		t.Audit(ctx, &job.JobID, "tool", "get_media_metadata", "tool.get_media_metadata.failed",
			map[string]any{"error_code": domain.CodeOf(err), "error": err.Error()})
		return nil, err
	}
	if meta.AudioTracks == 0 {
		err := domain.E(domain.CodeNoAudioTrack,
			"no audio track was detected; please replace the media with an audio-bearing file or confirm cancellation")
		t.Audit(ctx, &job.JobID, "tool", "get_media_metadata", "tool.get_media_metadata.failed",
			map[string]any{"error_code": domain.CodeNoAudioTrack})
		return nil, err
	}
	updated, err := t.Stores.Jobs.TransitionJob(ctx, job.JobID, job.Status, func(j *domain.Job) error {
		j.DurationSeconds = meta.DurationSeconds
		return nil
	})
	if err != nil {
		return nil, err
	}
	*job = *updated
	t.Audit(ctx, &job.JobID, "tool", "get_media_metadata", "tool.get_media_metadata.completed",
		map[string]any{
			"duration_seconds": meta.DurationSeconds,
			"format":           meta.Format,
			"audio_tracks":     meta.AudioTracks,
			"video_tracks":     meta.VideoTracks,
			"codec":            meta.Codec,
			"sample_rate_hz":   meta.SampleRateHz,
		})
	return meta, nil
}

// ---------------------------------------------------------------------
// 14.3 check_youtube_captions
// ---------------------------------------------------------------------

// CheckYouTubeCaptions runs the caption pre-check (PRD R2, 14.3). It also
// records the reusable-track decision inputs on the job.
func (t *Toolset) CheckYouTubeCaptions(ctx context.Context, job *domain.Job) (*captions.CheckResult, error) {
	if job.SourceType != domain.SourceYouTube {
		return nil, domain.E(domain.CodeValidationError, "caption check applies to youtube jobs only")
	}
	res, err := t.Captions.Check(ctx, job.SourceURI, job.Language)
	if err != nil {
		t.Audit(ctx, &job.JobID, "tool", "check_youtube_captions", "tool.check_youtube_captions.failed",
			map[string]any{"error_code": domain.CodeOf(err), "error": err.Error()})
		return nil, err
	}
	available, trackID := false, ""
	if res.OfficialCaptionsFound && res.DownloadAuthorized {
		for _, tr := range res.Tracks {
			// Only official downloadable tracks are reusable; auto-generated
			// captions are never reused in MVP (PRD R2).
			if tr.CaptionType == "official" && tr.Downloadable {
				available = true
				trackID = tr.CaptionTrackID
				break
			}
		}
	}
	updated, err := t.Stores.Jobs.TransitionJob(ctx, job.JobID, job.Status, func(j *domain.Job) error {
		j.CaptionsAvailable = available
		j.CaptionTrackID = trackID
		return nil
	})
	if err != nil {
		return nil, err
	}
	*job = *updated
	t.Audit(ctx, &job.JobID, "tool", "check_youtube_captions", "tool.check_youtube_captions.completed",
		map[string]any{
			"video_uri_hash":          URIHash(job.SourceURI),
			"official_captions_found": res.OfficialCaptionsFound,
			"auto_captions_found":     res.AutoCaptionsFound,
			"download_authorized":     res.DownloadAuthorized,
			"reusable":                job.CaptionsAvailable,
		})
	return res, nil
}

// ---------------------------------------------------------------------
// 14.4 fetch_existing_captions
// ---------------------------------------------------------------------

// FetchExistingCaptions downloads an authorized official caption track and
// stores it as a caption_source artifact (PRD 14.4).
func (t *Toolset) FetchExistingCaptions(ctx context.Context, job *domain.Job, captionTrackID string) (string, error) {
	data, err := t.Captions.Fetch(ctx, captionTrackID, "vtt")
	if err != nil {
		t.Audit(ctx, &job.JobID, "tool", "fetch_existing_captions", "tool.fetch_existing_captions.failed",
			map[string]any{"error_code": domain.CodeOf(err), "error": err.Error()})
		return "", err
	}
	key := objectstore.KeyFor(job.JobID.String(), "captions", "official.vtt")
	uri, err := t.Objects.Put(ctx, key, data)
	if err != nil {
		return "", err
	}
	art := &domain.MediaArtifact{
		ArtifactID: uuid.New(), JobID: job.JobID,
		ArtifactType: domain.ArtifactCaptionSource, URI: uri,
		MimeType: "text/vtt", SizeBytes: int64(len(data)), CreatedAt: time.Now().UTC(),
	}
	if err := t.Stores.Artifacts.CreateArtifact(ctx, art); err != nil {
		return "", err
	}
	t.Audit(ctx, &job.JobID, "tool", "fetch_existing_captions", "tool.fetch_existing_captions.completed",
		map[string]any{"caption_track_id": captionTrackID, "artifact_id": art.ArtifactID.String(), "format": "vtt", "size_bytes": len(data)})
	return uri, nil
}

// ---------------------------------------------------------------------
// 14.5 parse_captions_to_transcript
// ---------------------------------------------------------------------

// ParseCaptionsToTranscript parses a caption artifact into a raw transcript
// version (PRD 14.5). Segments carry null confidence and a caption_origin
// flag; speaker_label defaults to Speaker 1.
func (t *Toolset) ParseCaptionsToTranscript(ctx context.Context, job *domain.Job, captionArtifactURI string, cfg *domain.JobConfig) (*domain.TranscriptVersion, int, error) {
	data, err := t.Objects.Get(ctx, captionArtifactURI)
	if err != nil {
		return nil, 0, domain.E(domain.CodeCaptionParseFailed, "read caption artifact: %v", err)
	}
	cues, err := captions.ParseVTT(data)
	if err != nil {
		t.Audit(ctx, &job.JobID, "tool", "parse_captions_to_transcript", "tool.parse_captions_to_transcript.failed",
			map[string]any{"error_code": domain.CodeOf(err), "error": err.Error()})
		return nil, 0, err
	}
	version := &domain.TranscriptVersion{
		TranscriptVersionID: uuid.New(), JobID: job.JobID,
		VersionType: domain.VersionRaw, CreatedBy: "tool:parse_captions_to_transcript",
		IsImmutable: false, CreatedAt: time.Now().UTC(),
	}
	segs := make([]*domain.Segment, 0, len(cues))
	for _, c := range cues {
		segs = append(segs, &domain.Segment{
			SegmentID: uuid.New(), TranscriptVersionID: version.TranscriptVersionID,
			StartMS: c.StartMS, EndMS: c.EndMS,
			SpeakerLabel: "Speaker 1", // caption formats carry no speaker cues here (14.5)
			Text:         c.Text,
			Confidence:   nil, // null confidence per 14.5 behavior rules
			Flags:        map[string]bool{"caption_origin": true},
		})
	}
	if err := t.Stores.Transcripts.CreateVersion(ctx, version, segs); err != nil {
		return nil, 0, err
	}
	t.Audit(ctx, &job.JobID, "tool", "parse_captions_to_transcript", "tool.parse_captions_to_transcript.completed",
		map[string]any{
			"caption_artifact_uri_hash": URIHash(captionArtifactURI),
			"transcript_version_id":     version.TranscriptVersionID.String(),
			"segment_count":             len(segs),
			"confidence_available":      false,
			"speaker_labels_available":  false,
			"job_config_id":             cfg.JobConfigID.String(),
		})
	return version, len(segs), nil
}

// ---------------------------------------------------------------------
// 14.15 publish_caption_file — future, disabled in MVP
// ---------------------------------------------------------------------

// PublishCaptionFile is an interface stub only (PRD 14.15). It always
// returns DISABLED_IN_MVP and audits the blocked attempt.
func (t *Toolset) PublishCaptionFile(ctx context.Context, jobID uuid.UUID, actorID string) error {
	if err := t.Audit(ctx, &jobID, "tool", "publish_caption_file", "caption.publish.blocked",
		map[string]any{"reason": "disabled in MVP; requires Level 3 approval-gated workflow", "actor": actorID}); err != nil {
		return err
	}
	return domain.ErrDisabledInMVP
}
