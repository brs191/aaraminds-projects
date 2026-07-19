package domain

import (
	"time"

	"github.com/google/uuid"
)

// ErrorInfo is the last_error shape surfaced on the Job JSON.
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Job is the jobs row (PRD 13.3) plus orchestration bookkeeping fields
// (action_required, last_error, caption decision state) needed by the
// workflow in PRD 11.1 and the REST contract.
type Job struct {
	JobID             uuid.UUID
	SourceType        string // youtube | upload
	SourceURI         string
	Status            Status
	SubmittedBy       string
	OwnershipAttested bool
	Language          string
	JobConfigID       *uuid.UUID
	DurationSeconds   int
	ActionRequired    string // "" | caption_decision | replace_media
	LastError         *ErrorInfo
	CaptionsAvailable bool   // official, authorized, downloadable captions found
	CaptionTrackID    string // first reusable official track
	CaptionReuse      *bool  // producer decision; nil = undecided
	CancelReason      string
	// LibraryMode marks personal-use library jobs (RSS episodes): the pipeline
	// stops at drafted (no review gate) and the summary is auto-generated.
	LibraryMode bool
	// SourceBasis records the ownership basis when attestation is programmatic
	// (library jobs: "open_rss_personal_use"). Empty for manual attestations.
	SourceBasis string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Feed is a feeds row (library mode): a subscribed podcast RSS feed.
type Feed struct {
	FeedID         uuid.UUID
	FeedURL        string
	Title          string
	Description    string
	ImageURL       string // "" renders as null
	AutoTranscribe bool
	LastPolledAt   *time.Time
	PollError      string // "" renders as null; recorded on poll failure, feed stays
	CreatedAt      time.Time
	DeletedAt      *time.Time // soft delete: episodes/jobs/transcripts are kept
}

// Episode is an episodes row (library mode), unique per (feed_id, guid).
type Episode struct {
	EpisodeID       uuid.UUID
	FeedID          uuid.UUID
	GUID            string
	Title           string
	Description     string
	AudioURL        string
	PublishedAt     *time.Time
	DurationSeconds *int
	MediaArtifactID *uuid.UUID // set once the enclosure is downloaded
	JobID           *uuid.UUID // set once a transcription job exists
	CreatedAt       time.Time
}

// JobConfig is the per-job configuration snapshot (PRD 13.3 job_config).
// Tools read configuration from here via job_config_id — never from
// independent per-tool threshold inputs.
type JobConfig struct {
	JobConfigID          uuid.UUID
	JobID                uuid.UUID
	Language             string
	ConfidenceThreshold  float64
	EnableDiarization    bool
	ExpectedSpeakerCount *int
	StylePolicyID        string
	SummaryMaxWords      int
	SummaryStyle         string
	STTProvider          string
	STTModel             string
	MaxDurationSeconds   *int
	CreatedBy            string
	CreatedAt            time.Time
}

// DefaultJobConfig returns the PRD default configuration snapshot values.
func DefaultJobConfig(sttProvider string) JobConfig {
	return JobConfig{
		Language:            "en",
		ConfidenceThreshold: 0.80,
		EnableDiarization:   true,
		StylePolicyID:       "default-clean-v1",
		SummaryMaxWords:     150,
		SummaryStyle:        "neutral-professional",
		STTProvider:         sttProvider,
	}
}

// TranscriptVersion is a transcript_versions row (PRD 13.3).
type TranscriptVersion struct {
	TranscriptVersionID uuid.UUID
	JobID               uuid.UUID
	VersionType         string // raw | clean | reviewed | approved
	SourceVersionID     *uuid.UUID
	CreatedBy           string
	IsImmutable         bool
	CreatedAt           time.Time
}

// Segment is a transcript_segments row (PRD 13.3). Confidence is nil for
// caption-derived segments (PRD 14.5 null-confidence rule).
type Segment struct {
	SegmentID           uuid.UUID
	TranscriptVersionID uuid.UUID
	StartMS             int
	EndMS               int
	SpeakerLabel        string
	Text                string
	Confidence          *float64
	Flags               map[string]bool // low_confidence, caption_origin, ...
}

// Summary is a summaries row (PRD 13.3).
type Summary struct {
	SummaryID                 uuid.UUID
	JobID                     uuid.UUID
	SourceTranscriptVersionID uuid.UUID
	Text                      string
	ValidationStatus          string // passed | needs_review | failed
	ValidationNotes           string
	CreatedBy                 string
	CreatedAt                 time.Time
	UpdatedAt                 *time.Time
}

// QualityIssue is one entry of quality_reports.issue_summary_json.
type QualityIssue struct {
	IssueType string `json:"issue_type"`
	Severity  string `json:"severity"`
	StartMS   int    `json:"start_ms"`
	EndMS     int    `json:"end_ms"`
	Message   string `json:"message"`
}

// QualityReport is a quality_reports row (PRD 13.3).
type QualityReport struct {
	QualityReportID           uuid.UUID
	JobID                     uuid.UUID
	TranscriptVersionID       uuid.UUID
	JobConfigID               uuid.UUID
	ConfidenceThreshold       float64
	QualityScore              *float64
	AverageConfidence         *float64
	LowConfidenceSegmentCount int
	CoverageGapSeconds        int
	TimestampGapCount         int
	DiarizationWarningCount   int
	ConfidenceUnavailable     bool // caption-derived transcripts (PRD R5)
	Issues                    []QualityIssue
	CreatedAt                 time.Time
}

// Approval is an approvals row (PRD 13.3, incl. supersede chain per 11.4).
type Approval struct {
	ApprovalID                  uuid.UUID
	JobID                       uuid.UUID
	ApprovedTranscriptVersionID uuid.UUID
	ApprovedBy                  string
	ApprovedAt                  time.Time
	ApprovalNote                string
	SupersededByApprovalID      *uuid.UUID
}

// AuditEvent is an audit_events row (PRD 13.3). Append-only.
type AuditEvent struct {
	AuditEventID uuid.UUID
	JobID        *uuid.UUID
	ActorType    string // user | system | tool
	ActorID      string
	EventType    string
	EventPayload map[string]any
	CreatedAt    time.Time
}

// MediaArtifact is a media_artifacts row (PRD 13.3).
type MediaArtifact struct {
	ArtifactID     uuid.UUID
	JobID          uuid.UUID
	ArtifactType   string // source_media | audio_extract | caption_source | export
	URI            string
	MimeType       string
	SizeBytes      int64
	Superseded     bool
	RetentionUntil *time.Time
	CreatedAt      time.Time
}

// ExportRecord links an export artifact to the approved transcript version it
// was generated from (PRD 14.12 / R8). Superseded is set on every prior
// export when the job is re-approved (PRD 13.2 r5): the artifact stays
// downloadable but responses carry X-Superseded: true.
type ExportRecord struct {
	ExportID                    uuid.UUID
	JobID                       uuid.UUID
	ApprovedTranscriptVersionID uuid.UUID
	Format                      string
	ArtifactURI                 string
	ValidationStatus            string // passed | failed
	Superseded                  bool
	CreatedBy                   string
	CreatedAt                   time.Time
}
