package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/aaraminds/transcript-agent/internal/domain"
	"github.com/aaraminds/transcript-agent/internal/metrics"
	"github.com/aaraminds/transcript-agent/internal/objectstore"
	"github.com/aaraminds/transcript-agent/internal/providers/llm"
	"github.com/aaraminds/transcript-agent/internal/providers/stt"
)

// ---------------------------------------------------------------------
// 14.6 extract_audio
// ---------------------------------------------------------------------

// ExtractAudio extracts a normalized audio artifact (PRD 14.6) and stores it
// with an audio_extract media_artifacts record.
func (t *Toolset) ExtractAudio(ctx context.Context, job *domain.Job) (string, error) {
	sourceURI, err := t.mediaSourceURI(ctx, job)
	if err != nil {
		t.Audit(ctx, &job.JobID, "tool", "extract_audio", "tool.extract_audio.failed",
			map[string]any{"error_code": domain.CodeOf(err), "error": err.Error()})
		return "", err
	}
	data, meta, err := t.Media.ExtractAudio(ctx, job.SourceType, sourceURI)
	if err != nil {
		t.Audit(ctx, &job.JobID, "tool", "extract_audio", "tool.extract_audio.failed",
			map[string]any{"error_code": domain.CodeOf(err), "error": err.Error()})
		return "", err
	}
	// Propagate mock behavior markers from the source URI into the artifact
	// key so provider fault injection works end-to-end in mock mode.
	name := "normalized"
	for _, marker := range []string{"stt-timeout-once", "stt-quota", "no-diarization"} {
		if strings.Contains(job.SourceURI, marker) {
			name += "-" + marker
		}
	}
	key := objectstore.KeyFor(job.JobID.String(), "audio", name+".wav")
	uri, err := t.Objects.Put(ctx, key, data)
	if err != nil {
		t.Audit(ctx, &job.JobID, "tool", "extract_audio", "tool.extract_audio.failed",
			map[string]any{"error_code": domain.CodeOf(err), "error": err.Error()})
		return "", err
	}
	now := time.Now().UTC()
	art := &domain.MediaArtifact{
		ArtifactID: uuid.New(), JobID: job.JobID,
		ArtifactType: domain.ArtifactAudioExtract, URI: uri,
		MimeType: "audio/wav", SizeBytes: int64(len(data)),
		RetentionUntil: t.RetentionUntil(now), CreatedAt: now,
	}
	if err := t.Stores.Artifacts.CreateArtifact(ctx, art); err != nil {
		return "", err
	}
	t.Audit(ctx, &job.JobID, "tool", "extract_audio", "tool.extract_audio.completed",
		map[string]any{
			"artifact_id":      art.ArtifactID.String(),
			"duration_seconds": meta.DurationSeconds,
			"format":           meta.Format,
			"sample_rate_hz":   meta.SampleRateHz,
			"channels":         1,
		})
	return uri, nil
}

// LatestAudioArtifactURI returns the newest audio_extract artifact for a job.
func (t *Toolset) LatestAudioArtifactURI(ctx context.Context, jobID uuid.UUID) (string, error) {
	arts, err := t.Stores.Artifacts.ListArtifactsByJob(ctx, jobID, domain.ArtifactAudioExtract)
	if err != nil {
		return "", err
	}
	if len(arts) == 0 {
		return "", domain.E(domain.CodeMediaNotFound, "no audio artifact for job %s", jobID)
	}
	return arts[len(arts)-1].URI, nil
}

// ---------------------------------------------------------------------
// 14.7 transcribe_audio
// ---------------------------------------------------------------------

// TranscribeAudio runs batch STT with diarization and creates the raw
// transcript version (PRD 14.7). Confidence flagging uses the threshold from
// the job_config snapshot only.
func (t *Toolset) TranscribeAudio(ctx context.Context, job *domain.Job, audioArtifactURI string, cfg *domain.JobConfig) (*domain.TranscriptVersion, error) {
	var (
		res *stt.Result
		err error
	)
	// Providers that support an expected speaker count for diarization get it
	// from the job_config snapshot (PRD 13.3 expected_speaker_count).
	if hinter, ok := t.STT.(stt.SpeakerHinter); ok {
		res, err = hinter.TranscribeWithSpeakerHint(ctx, audioArtifactURI, cfg.Language, cfg.EnableDiarization, cfg.ExpectedSpeakerCount)
	} else {
		res, err = t.STT.Transcribe(ctx, audioArtifactURI, cfg.Language, cfg.EnableDiarization)
	}
	if err != nil {
		t.Audit(ctx, &job.JobID, "tool", "transcribe_audio", "tool.transcribe_audio.failed",
			map[string]any{"error_code": domain.CodeOf(err), "error": err.Error(), "job_config_id": cfg.JobConfigID.String()})
		return nil, err
	}
	version := &domain.TranscriptVersion{
		TranscriptVersionID: uuid.New(), JobID: job.JobID,
		VersionType: domain.VersionRaw, CreatedBy: "tool:transcribe_audio",
		IsImmutable: false, CreatedAt: time.Now().UTC(),
	}
	lowCount := 0
	segs := make([]*domain.Segment, 0, len(res.Segments))
	for _, s := range res.Segments {
		conf := s.Confidence
		flags := map[string]bool{}
		if conf < cfg.ConfidenceThreshold {
			flags["low_confidence"] = true
			lowCount++
		}
		if !res.DiarizationAvailable {
			flags["diarization_unavailable"] = true
		}
		segs = append(segs, &domain.Segment{
			SegmentID: uuid.New(), TranscriptVersionID: version.TranscriptVersionID,
			StartMS: s.StartMS, EndMS: s.EndMS,
			SpeakerLabel: s.SpeakerLabel, Text: s.Text,
			Confidence: &conf, Flags: flags,
		})
	}
	if err := t.Stores.Transcripts.CreateVersion(ctx, version, segs); err != nil {
		return nil, err
	}
	metrics.STTSecondsProcessed.Add(int64(job.DurationSeconds))
	t.Audit(ctx, &job.JobID, "tool", "transcribe_audio", "tool.transcribe_audio.completed",
		map[string]any{
			"provider":              res.Provider,
			"model":                 res.Model,
			"request_id":            res.RequestID,
			"transcript_version_id": version.TranscriptVersionID.String(),
			"segment_count":         len(segs),
			"job_config_id":         cfg.JobConfigID.String(),
			"confidence_threshold":  cfg.ConfidenceThreshold,
			"low_confidence_count":  lowCount,
			"diarization_available": res.DiarizationAvailable,
		})
	return version, nil
}

// ---------------------------------------------------------------------
// 14.8 normalize_transcript
// ---------------------------------------------------------------------

// NormalizeTranscript creates the clean transcript from the raw version
// (PRD 14.8, cleanup policy 15.2). Timestamps, speaker labels, confidence
// values, and flags are preserved segment-for-segment.
func (t *Toolset) NormalizeTranscript(ctx context.Context, job *domain.Job, source *domain.TranscriptVersion, cfg *domain.JobConfig) (*domain.TranscriptVersion, llm.CleanupStats, error) {
	segs, err := t.Stores.Transcripts.ListSegments(ctx, source.TranscriptVersionID)
	if err != nil {
		return nil, llm.CleanupStats{}, err
	}
	texts := make([]string, len(segs))
	for i, s := range segs {
		texts[i] = s.Text
	}
	cleaned, stats, err := t.LLM.Cleanup(ctx, texts, cfg.StylePolicyID)
	if err != nil {
		t.Audit(ctx, &job.JobID, "tool", "normalize_transcript", "tool.normalize_transcript.failed",
			map[string]any{"error_code": domain.CodeOf(err), "error": err.Error(), "job_config_id": cfg.JobConfigID.String()})
		return nil, stats, err
	}
	if len(cleaned) != len(segs) {
		err := domain.E(domain.CodeLLMOutputInvalid,
			"cleanup returned %d texts for %d segments", len(cleaned), len(segs))
		t.Audit(ctx, &job.JobID, "tool", "normalize_transcript", "tool.normalize_transcript.failed",
			map[string]any{"error_code": domain.CodeLLMOutputInvalid})
		return nil, stats, err
	}
	srcID := source.TranscriptVersionID
	version := &domain.TranscriptVersion{
		TranscriptVersionID: uuid.New(), JobID: job.JobID,
		VersionType: domain.VersionClean, SourceVersionID: &srcID,
		CreatedBy: "tool:normalize_transcript", IsImmutable: false, CreatedAt: time.Now().UTC(),
	}
	newSegs := make([]*domain.Segment, len(segs))
	for i, s := range segs {
		c := *s
		c.SegmentID = uuid.New()
		c.TranscriptVersionID = version.TranscriptVersionID
		c.Text = cleaned[i]
		if stats.MeaningChangeDetected {
			if c.Flags == nil {
				c.Flags = map[string]bool{}
			}
			c.Flags["meaning_change_risk"] = true // flag for human review (14.8)
		}
		newSegs[i] = &c
	}
	if err := t.Stores.Transcripts.CreateVersion(ctx, version, newSegs); err != nil {
		return nil, stats, err
	}
	t.Audit(ctx, &job.JobID, "tool", "normalize_transcript", "tool.normalize_transcript.completed",
		map[string]any{
			"source_transcript_version_id": source.TranscriptVersionID.String(),
			"clean_transcript_version_id":  version.TranscriptVersionID.String(),
			"job_config_id":                cfg.JobConfigID.String(),
			"style_policy_id_used":         cfg.StylePolicyID,
			"segments_processed":           stats.SegmentsProcessed,
			"filler_words_removed":         stats.FillerWordsRemoved,
			"meaning_change_detected":      stats.MeaningChangeDetected,
		})
	return version, stats, nil
}

// ---------------------------------------------------------------------
// 14.9 quality_check_transcript
// ---------------------------------------------------------------------

const timestampGapThresholdMS = 2000 // PRD 17.2: >2s drift flags review

// QualityCheckTranscript computes real quality metrics from segments and
// persists a quality_reports row (PRD 14.9, 13.3). Caption-derived
// transcripts are marked confidence_unavailable and exempt from threshold
// flagging (PRD R5).
func (t *Toolset) QualityCheckTranscript(ctx context.Context, job *domain.Job, version *domain.TranscriptVersion, cfg *domain.JobConfig) (*domain.QualityReport, error) {
	segs, err := t.Stores.Transcripts.ListSegments(ctx, version.TranscriptVersionID)
	if err != nil || len(segs) == 0 {
		e := domain.E(domain.CodeTranscriptNotFound, "no segments for transcript version %s", version.TranscriptVersionID)
		t.Audit(ctx, &job.JobID, "tool", "quality_check_transcript", "tool.quality_check_transcript.failed",
			map[string]any{"error_code": domain.CodeTranscriptNotFound})
		return nil, e
	}
	sort.SliceStable(segs, func(i, k int) bool { return segs[i].StartMS < segs[k].StartMS })

	report := &domain.QualityReport{
		QualityReportID:     uuid.New(),
		JobID:               job.JobID,
		TranscriptVersionID: version.TranscriptVersionID,
		JobConfigID:         cfg.JobConfigID,
		ConfidenceThreshold: cfg.ConfidenceThreshold,
		CreatedAt:           time.Now().UTC(),
	}

	// Confidence metrics.
	var confSum float64
	confN := 0
	speakers := map[string]bool{}
	for _, s := range segs {
		speakers[s.SpeakerLabel] = true
		if s.Confidence == nil {
			continue
		}
		confSum += *s.Confidence
		confN++
		if *s.Confidence < cfg.ConfidenceThreshold {
			report.LowConfidenceSegmentCount++
			report.Issues = append(report.Issues, domain.QualityIssue{
				IssueType: "LOW_CONFIDENCE_SEGMENT", Severity: "medium",
				StartMS: s.StartMS, EndMS: s.EndMS,
				Message: fmt.Sprintf("Segment confidence %.2f below threshold %.2f.", *s.Confidence, cfg.ConfidenceThreshold),
			})
		}
	}
	if confN > 0 {
		avg := confSum / float64(confN)
		report.AverageConfidence = &avg
	} else {
		report.ConfidenceUnavailable = true
		report.Issues = append(report.Issues, domain.QualityIssue{
			IssueType: "CONFIDENCE_UNAVAILABLE", Severity: "info",
			StartMS: 0, EndMS: segs[len(segs)-1].EndMS,
			Message: "Caption-derived transcript: provider confidence scores are unavailable; threshold flagging skipped.",
		})
	}

	// Coverage and timestamp continuity.
	durationMS := job.DurationSeconds * 1000
	coveredMS := 0
	gapMS := segs[0].StartMS
	prevEnd := segs[0].StartMS
	for _, s := range segs {
		coveredMS += s.EndMS - s.StartMS
		if gap := s.StartMS - prevEnd; gap > 0 {
			gapMS += gap
			if gap > timestampGapThresholdMS {
				report.TimestampGapCount++
				report.Issues = append(report.Issues, domain.QualityIssue{
					IssueType: "TIMESTAMP_GAP", Severity: "low",
					StartMS: prevEnd, EndMS: s.StartMS,
					Message: fmt.Sprintf("Timestamp discontinuity of %dms between segments.", gap),
				})
			}
		}
		if s.EndMS > prevEnd {
			prevEnd = s.EndMS
		}
	}
	if durationMS > prevEnd {
		gapMS += durationMS - prevEnd
	}
	report.CoverageGapSeconds = gapMS / 1000

	// Duration sanity (14.9 DURATION_MISMATCH -> flag for review).
	if durationMS > 0 {
		diff := durationMS - prevEnd
		if diff < 0 {
			diff = -diff
		}
		if float64(diff) > 0.3*float64(durationMS) {
			report.Issues = append(report.Issues, domain.QualityIssue{
				IssueType: "DURATION_MISMATCH", Severity: "medium",
				StartMS: 0, EndMS: prevEnd,
				Message: fmt.Sprintf("Transcript span %ds differs from media duration %ds.", prevEnd/1000, job.DurationSeconds),
			})
		}
	}

	// Diarization sanity: STT path with diarization enabled should yield >=2
	// speakers on podcast content.
	if cfg.EnableDiarization && !report.ConfidenceUnavailable && len(speakers) < 2 {
		report.DiarizationWarningCount++
		report.Issues = append(report.Issues, domain.QualityIssue{
			IssueType: "DIARIZATION_WARNING", Severity: "medium",
			StartMS: 0, EndMS: prevEnd,
			Message: "Speaker detection produced fewer than two speakers; transcript needs manual speaker review.",
		})
	}

	// Composite score: average confidence weighted by coverage ratio.
	if !report.ConfidenceUnavailable {
		ratio := 1.0
		if durationMS > 0 {
			ratio = float64(coveredMS) / float64(durationMS)
			if ratio > 1 {
				ratio = 1
			}
		}
		score := *report.AverageConfidence * ratio
		report.QualityScore = &score
	}

	if err := t.Stores.Quality.CreateReport(ctx, report); err != nil {
		return nil, err
	}
	t.Audit(ctx, &job.JobID, "tool", "quality_check_transcript", "tool.quality_check_transcript.completed",
		map[string]any{
			"quality_report_id":            report.QualityReportID.String(),
			"job_config_id":                cfg.JobConfigID.String(),
			"confidence_threshold_used":    cfg.ConfidenceThreshold,
			"low_confidence_segment_count": report.LowConfidenceSegmentCount,
			"coverage_gap_seconds":         report.CoverageGapSeconds,
			"timestamp_gap_count":          report.TimestampGapCount,
			"diarization_warning_count":    report.DiarizationWarningCount,
			"confidence_unavailable":       report.ConfidenceUnavailable,
			"issue_count":                  len(report.Issues),
		})
	return report, nil
}
