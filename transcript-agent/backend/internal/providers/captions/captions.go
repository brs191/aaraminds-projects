// Package captions defines the caption provider interface (PRD R2, 14.3,
// 14.4) and a WebVTT parser shared by the caption-reuse path (14.5).
package captions

import (
	"context"
	"strconv"
	"strings"

	"github.com/aaraminds/transcript-agent/internal/domain"
)

// Track mirrors caption_tracks in the 14.3 output contract.
type Track struct {
	CaptionTrackID string `json:"caption_track_id"`
	Language       string `json:"language"`
	CaptionType    string `json:"caption_type"` // official | auto_generated
	Downloadable   bool   `json:"downloadable"`
}

// CheckResult mirrors the 14.3 output contract.
type CheckResult struct {
	OfficialCaptionsFound bool    `json:"official_captions_found"`
	AutoCaptionsFound     bool    `json:"auto_captions_found"`
	DownloadAuthorized    bool    `json:"download_authorized"`
	Tracks                []Track `json:"caption_tracks"`
}

// Provider checks and fetches official captions. Auto-generated captions are
// never treated as reusable in MVP (PRD R2).
type Provider interface {
	// Check inspects the source for reusable official caption tracks.
	Check(ctx context.Context, sourceURI, language string) (*CheckResult, error)
	// Fetch downloads a caption track in the requested format (vtt|srt|text).
	Fetch(ctx context.Context, captionTrackID, format string) ([]byte, error)
}

// Cue is one parsed caption cue.
type Cue struct {
	StartMS int
	EndMS   int
	Text    string
}

// ParseVTT parses a WebVTT document into cues. Errors use the 14.5 failure
// codes so the orchestrator can fall back to transcription.
func ParseVTT(data []byte) ([]Cue, error) {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || !strings.HasPrefix(strings.TrimSpace(lines[0]), "WEBVTT") {
		return nil, domain.E(domain.CodeCaptionParseFailed, "missing WEBVTT header")
	}
	var cues []Cue
	i := 1
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "NOTE") {
			i++
			continue
		}
		// Optional cue identifier line before the timing line.
		if !strings.Contains(line, "-->") {
			i++
			if i >= len(lines) || !strings.Contains(lines[i], "-->") {
				return nil, domain.E(domain.CodeCaptionParseFailed, "expected timing line near %q", line)
			}
			line = strings.TrimSpace(lines[i])
		}
		parts := strings.SplitN(line, "-->", 2)
		start, err := ParseVTTTimestamp(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, err
		}
		endField := strings.Fields(strings.TrimSpace(parts[1])) // strip cue settings
		if len(endField) == 0 {
			return nil, domain.E(domain.CodeTimestampInvalid, "missing end timestamp in %q", line)
		}
		end, err := ParseVTTTimestamp(endField[0])
		if err != nil {
			return nil, err
		}
		if end <= start {
			return nil, domain.E(domain.CodeTimestampInvalid, "cue end %d <= start %d", end, start)
		}
		i++
		var textLines []string
		for i < len(lines) && strings.TrimSpace(lines[i]) != "" {
			textLines = append(textLines, strings.TrimSpace(lines[i]))
			i++
		}
		cues = append(cues, Cue{StartMS: start, EndMS: end, Text: strings.Join(textLines, " ")})
	}
	if len(cues) == 0 {
		return nil, domain.E(domain.CodeCaptionEmptyOrTruncated, "caption file contains no cues")
	}
	return cues, nil
}

// ParseVTTTimestamp parses "HH:MM:SS.mmm" or "MM:SS.mmm" into milliseconds.
func ParseVTTTimestamp(ts string) (int, error) {
	main, frac, ok := strings.Cut(ts, ".")
	if !ok || len(frac) != 3 {
		return 0, domain.E(domain.CodeTimestampInvalid, "invalid VTT timestamp %q", ts)
	}
	parts := strings.Split(main, ":")
	if len(parts) != 2 && len(parts) != 3 {
		return 0, domain.E(domain.CodeTimestampInvalid, "invalid VTT timestamp %q", ts)
	}
	nums := make([]int, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return 0, domain.E(domain.CodeTimestampInvalid, "invalid VTT timestamp %q", ts)
		}
		nums[i] = n
	}
	ms, err := strconv.Atoi(frac)
	if err != nil {
		return 0, domain.E(domain.CodeTimestampInvalid, "invalid VTT timestamp %q", ts)
	}
	var total int
	if len(nums) == 3 {
		total = ((nums[0]*60+nums[1])*60 + nums[2]) * 1000
	} else {
		total = (nums[0]*60 + nums[1]) * 1000
	}
	return total + ms, nil
}
