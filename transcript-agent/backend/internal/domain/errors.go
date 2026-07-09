package domain

import (
	"errors"
	"fmt"
)

// Error is the structured error type carried through tools and surfaced by the
// API as {"error":{"code","message"}}. Codes come from PRD sections 14 and 19.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

// E builds a structured domain error.
func E(code, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

// CodeOf extracts the structured code from an error chain, or INTERNAL_ERROR.
func CodeOf(err error) string {
	var de *Error
	if errors.As(err, &de) {
		return de.Code
	}
	return CodeInternalError
}

// AsError converts any error into a *Error, wrapping unknown errors as internal.
func AsError(err error) *Error {
	var de *Error
	if errors.As(err, &de) {
		return de
	}
	return &Error{Code: CodeInternalError, Message: err.Error()}
}

// Error codes — PRD section 14 tool contracts and section 19 failure matrix.
const (
	// 14.1 submit_media_job
	CodeOwnershipAttestationMissing = "OWNERSHIP_ATTESTATION_MISSING"
	CodeUnsupportedSourceType       = "UNSUPPORTED_SOURCE_TYPE"
	CodeInvalidSourceURI            = "INVALID_SOURCE_URI"

	// 14.2 get_media_metadata
	CodeMediaNotFound     = "MEDIA_NOT_FOUND"
	CodeUnsupportedFormat = "UNSUPPORTED_FORMAT"
	CodeNoAudioTrack      = "NO_AUDIO_TRACK"
	CodeMetadataTimeout   = "METADATA_TIMEOUT"

	// 14.3 check_youtube_captions
	CodeYouTubeAuthRequired   = "YOUTUBE_AUTH_REQUIRED"
	CodeVideoNotOwned         = "VIDEO_NOT_OWNED"
	CodeCaptionAPIUnavailable = "CAPTION_API_UNAVAILABLE"

	// 14.4 fetch_existing_captions
	CodeCaptionDownloadUnauthorized = "CAPTION_DOWNLOAD_UNAUTHORIZED"
	CodeCaptionTrackNotFound        = "CAPTION_TRACK_NOT_FOUND"
	CodeCaptionFormatUnsupported    = "CAPTION_FORMAT_UNSUPPORTED"

	// 14.5 parse_captions_to_transcript
	CodeCaptionParseFailed      = "CAPTION_PARSE_FAILED"
	CodeCaptionEmptyOrTruncated = "CAPTION_EMPTY_OR_TRUNCATED"
	CodeTimestampInvalid        = "TIMESTAMP_INVALID"

	// 14.6 extract_audio
	CodeExtractionFailed    = "EXTRACTION_FAILED"
	CodeArtifactWriteFailed = "ARTIFACT_WRITE_FAILED"

	// 14.7 transcribe_audio
	CodeSTTProviderTimeout       = "STT_PROVIDER_TIMEOUT"
	CodeSTTProviderQuotaExceeded = "STT_PROVIDER_QUOTA_EXCEEDED"
	CodeLanguageUnsupported      = "LANGUAGE_UNSUPPORTED"
	CodeDiarizationUnavailable   = "DIARIZATION_UNAVAILABLE"

	// 14.8 normalize_transcript
	CodeStylePolicyNotFound = "STYLE_POLICY_NOT_FOUND"
	CodeMeaningChangeRisk   = "MEANING_CHANGE_RISK"
	CodeLLMOutputInvalid    = "LLM_OUTPUT_INVALID"

	// 14.9 quality_check_transcript
	CodeTranscriptNotFound = "TRANSCRIPT_NOT_FOUND"
	CodeDurationMismatch   = "DURATION_MISMATCH"
	CodeQualityCheckFailed = "QUALITY_CHECK_FAILED"

	// 14.10 approve_transcript
	CodeUserNotAuthorized              = "USER_NOT_AUTHORIZED"
	CodeTranscriptVersionNotReviewable = "TRANSCRIPT_VERSION_NOT_REVIEWABLE"
	CodeOpenCriticalIssues             = "OPEN_CRITICAL_ISSUES"

	// 14.11 generate_summary
	CodeSummaryUngroundedClaimRisk = "SUMMARY_UNGROUNDED_CLAIM_RISK"
	CodeLLMProviderTimeout         = "LLM_PROVIDER_TIMEOUT"
	CodeTranscriptTooLong          = "TRANSCRIPT_TOO_LONG"

	// 14.12 export_transcript
	CodeApprovedTranscriptRequired = "APPROVED_TRANSCRIPT_REQUIRED"
	CodeFormatValidationFailed     = "FORMAT_VALIDATION_FAILED"

	// 14.13 replace_job_media / 14.14 cancel_job
	CodeJobNotInActionableState = "JOB_NOT_IN_ACTIONABLE_STATE"
	CodeJobAlreadyTerminal      = "JOB_ALREADY_TERMINAL"

	// 14.15 publish_caption_file (disabled in MVP)
	CodeDisabledInMVP = "DISABLED_IN_MVP"

	// Cross-cutting
	CodeStatusConflict             = "STATUS_CONFLICT"
	CodeRequestTooLarge            = "REQUEST_TOO_LARGE"
	CodeTokenInvalid               = "TOKEN_INVALID"
	CodeAudioNotAvailable          = "AUDIO_NOT_AVAILABLE"
	CodeTranscriptVersionImmutable = "TRANSCRIPT_VERSION_IMMUTABLE"
	CodeJobNotFound                = "JOB_NOT_FOUND"
	CodeSegmentNotFound            = "SEGMENT_NOT_FOUND"
	CodeSummaryNotFound            = "SUMMARY_NOT_FOUND"
	CodeExportNotFound             = "EXPORT_NOT_FOUND"
	CodeQualityReportNotFound      = "QUALITY_REPORT_NOT_FOUND"
	CodeValidationError            = "VALIDATION_ERROR"
	CodeUnauthenticated            = "UNAUTHENTICATED"
	CodeInvalidStateTransition     = "INVALID_STATE_TRANSITION"
	CodeAuditWriteFailed           = "AUDIT_WRITE_FAILED"
	CodeInternalError              = "INTERNAL_ERROR"
	CodeNotConfigured              = "NOT_CONFIGURED"
)

// ErrStatusConflict signals that a compare-and-swap status transition lost a
// race: the job's current status no longer matches the expected `from` value.
// API handlers surface it as 409 STATUS_CONFLICT; the orchestrator drops the
// step and lets the requeue scanner or the winning actor own the job.
var ErrStatusConflict = &Error{
	Code:    CodeStatusConflict,
	Message: "job status changed concurrently; reload the job and retry",
}

// ErrDisabledInMVP is returned by publish_caption_file (PRD 14.15).
var ErrDisabledInMVP = &Error{
	Code:    CodeDisabledInMVP,
	Message: "publish_caption_file is defined as an interface stub only and is disabled in the MVP; publishing requires a future approval-gated workflow (Level 3)",
}
