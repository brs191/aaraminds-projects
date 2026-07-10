package domain

// Status is the canonical job lifecycle enum from PRD section 13.3.
type Status string

const (
	StatusSubmitted         Status = "submitted"
	StatusQueued            Status = "queued"
	StatusValidating        Status = "validating"
	StatusMetadataExtracted Status = "metadata_extracted"
	StatusCaptionChecked    Status = "caption_checked"
	StatusNeedsUserAction   Status = "needs_user_action"
	StatusExtractingAudio   Status = "extracting_audio"
	StatusTranscribing      Status = "transcribing"
	StatusNormalizing       Status = "normalizing"
	StatusQualityChecking   Status = "quality_checking"
	StatusDrafted           Status = "drafted"
	StatusInReview          Status = "in_review"
	StatusApproved          Status = "approved"
	StatusExported          Status = "exported"
	StatusFailed            Status = "failed"
	StatusCancelled         Status = "cancelled"
)

// AllStatuses lists every canonical status value.
var AllStatuses = []Status{
	StatusSubmitted, StatusQueued, StatusValidating, StatusMetadataExtracted,
	StatusCaptionChecked, StatusNeedsUserAction, StatusExtractingAudio,
	StatusTranscribing, StatusNormalizing, StatusQualityChecking, StatusDrafted,
	StatusInReview, StatusApproved, StatusExported, StatusFailed, StatusCancelled,
}

// Valid reports whether s is a canonical status value.
func (s Status) Valid() bool {
	for _, v := range AllStatuses {
		if s == v {
			return true
		}
	}
	return false
}

// Terminal reports whether the status is terminal (PRD R9: failed is terminal;
// cancelled is terminal).
func (s Status) Terminal() bool {
	return s == StatusFailed || s == StatusCancelled
}

// Transcript version types (PRD 13.3 transcript_versions.version_type).
const (
	VersionRaw      = "raw"
	VersionClean    = "clean"
	VersionReviewed = "reviewed"
	VersionApproved = "approved"
)

// Source types.
const (
	SourceYouTube = "youtube"
	SourceUpload  = "upload"
)

// Artifact types (PRD 13.3 media_artifacts.artifact_type).
const (
	ArtifactSourceMedia   = "source_media"
	ArtifactAudioExtract  = "audio_extract"
	ArtifactCaptionSource = "caption_source"
	ArtifactExport        = "export"
)

// Action-required values surfaced with needs_user_action.
const (
	ActionCaptionDecision  = "caption_decision"
	ActionReplaceMedia     = "replace_media"
	ActionDurationExceeded = "duration_exceeded" // media longer than max_duration_seconds (PRD 20.2)
)

// Roles accepted by the API auth middleware.
const (
	RoleProducer = "producer"
	RoleReviewer = "reviewer"
	RoleAdmin    = "admin"
)

// Export formats supported in MVP (PRD R8).
var ExportFormats = []string{"txt", "md", "srt", "vtt"}

// SupportedUploadExtensions per PRD R1.
var SupportedUploadExtensions = []string{"mp3", "m4a", "wav", "mp4", "mov"}
