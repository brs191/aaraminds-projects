// Package exporter generates deterministic export artifacts from approved
// transcript segments (PRD R8, 14.12) and validates srt/vtt output by
// parsing it back (PRD 17.2: SRT/VTT validation pass rate 100%). No LLM is
// involved — export formatting is deterministic code (PRD 20.2 rule 7).
package exporter

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aaraminds/transcript-agent/internal/domain"
)

// Generate renders segments in the given format.
func Generate(format string, segments []*domain.Segment) ([]byte, error) {
	switch format {
	case "txt":
		return TXT(segments), nil
	case "md":
		return MD(segments), nil
	case "srt":
		return SRT(segments)
	case "vtt":
		return VTT(segments)
	default:
		return nil, domain.E(domain.CodeValidationError,
			"unsupported export format %q; supported: %s", format, strings.Join(domain.ExportFormats, ", "))
	}
}

// Validate checks a rendered artifact. txt/md always pass; srt/vtt are parsed
// back and their cue ordering and timestamps verified.
func Validate(format string, data []byte) error {
	switch format {
	case "txt", "md":
		if len(data) == 0 {
			return domain.E(domain.CodeFormatValidationFailed, "%s export is empty", format)
		}
		return nil
	case "srt":
		return ValidateSRT(data)
	case "vtt":
		return ValidateVTT(data)
	default:
		return domain.E(domain.CodeValidationError, "unsupported export format %q", format)
	}
}

// cueSegments filters out segments with no renderable text (a cleanup pass
// can legally empty a pure-filler segment).
func cueSegments(segments []*domain.Segment) []*domain.Segment {
	out := make([]*domain.Segment, 0, len(segments))
	for _, s := range segments {
		if strings.TrimSpace(s.Text) != "" {
			out = append(out, s)
		}
	}
	return out
}

func ts(ms int, sep string) string {
	h := ms / 3600000
	m := ms % 3600000 / 60000
	s := ms % 60000 / 1000
	frac := ms % 1000
	return fmt.Sprintf("%02d:%02d:%02d%s%03d", h, m, s, sep, frac)
}

// TXT renders "[HH:MM:SS.mmm] Speaker: text" lines.
func TXT(segments []*domain.Segment) []byte {
	var b strings.Builder
	for _, s := range cueSegments(segments) {
		fmt.Fprintf(&b, "[%s] %s: %s\n", ts(s.StartMS, "."), s.SpeakerLabel, s.Text)
	}
	return []byte(b.String())
}

// MD renders a Markdown transcript.
func MD(segments []*domain.Segment) []byte {
	var b strings.Builder
	b.WriteString("# Transcript\n\n")
	for _, s := range cueSegments(segments) {
		fmt.Fprintf(&b, "**[%s] %s:** %s\n\n", ts(s.StartMS, "."), s.SpeakerLabel, s.Text)
	}
	return []byte(b.String())
}

// SRT renders SubRip: 1-based index, "HH:MM:SS,mmm --> HH:MM:SS,mmm", text.
func SRT(segments []*domain.Segment) ([]byte, error) {
	segs := cueSegments(segments)
	if len(segs) == 0 {
		return nil, domain.E(domain.CodeFormatValidationFailed, "no segments to export")
	}
	var b strings.Builder
	for i, s := range segs {
		if s.EndMS <= s.StartMS {
			return nil, domain.E(domain.CodeFormatValidationFailed,
				"segment %d has end_ms %d <= start_ms %d", i+1, s.EndMS, s.StartMS)
		}
		fmt.Fprintf(&b, "%d\n%s --> %s\n%s: %s\n\n",
			i+1, ts(s.StartMS, ","), ts(s.EndMS, ","), s.SpeakerLabel, s.Text)
	}
	return []byte(b.String()), nil
}

// VTT renders WebVTT.
func VTT(segments []*domain.Segment) ([]byte, error) {
	segs := cueSegments(segments)
	if len(segs) == 0 {
		return nil, domain.E(domain.CodeFormatValidationFailed, "no segments to export")
	}
	var b strings.Builder
	b.WriteString("WEBVTT\n\n")
	for i, s := range segs {
		if s.EndMS <= s.StartMS {
			return nil, domain.E(domain.CodeFormatValidationFailed,
				"segment %d has end_ms %d <= start_ms %d", i+1, s.EndMS, s.StartMS)
		}
		fmt.Fprintf(&b, "%s --> %s\n%s: %s\n\n",
			ts(s.StartMS, "."), ts(s.EndMS, "."), s.SpeakerLabel, s.Text)
	}
	return []byte(b.String()), nil
}

// Cue is a parsed cue used by the validators.
type Cue struct {
	StartMS int
	EndMS   int
	Text    string
}

func parseClock(v, sep string) (int, error) {
	main, frac, ok := strings.Cut(v, sep)
	if !ok || len(frac) != 3 {
		return 0, fmt.Errorf("bad timestamp %q", v)
	}
	parts := strings.Split(main, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("bad timestamp %q", v)
	}
	var nums [3]int
	for i, p := range parts {
		if len(p) != 2 {
			return 0, fmt.Errorf("bad timestamp field %q in %q", p, v)
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return 0, fmt.Errorf("bad timestamp %q", v)
		}
		nums[i] = n
	}
	if nums[1] > 59 || nums[2] > 59 {
		return 0, fmt.Errorf("timestamp out of range %q", v)
	}
	ms, err := strconv.Atoi(frac)
	if err != nil {
		return 0, fmt.Errorf("bad timestamp %q", v)
	}
	return ((nums[0]*60+nums[1])*60+nums[2])*1000 + ms, nil
}

func parseTiming(line, sep string) (int, int, error) {
	parts := strings.Split(line, " --> ")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("bad timing line %q", line)
	}
	start, err := parseClock(strings.TrimSpace(parts[0]), sep)
	if err != nil {
		return 0, 0, err
	}
	endStr := strings.Fields(strings.TrimSpace(parts[1]))[0]
	end, err := parseClock(endStr, sep)
	if err != nil {
		return 0, 0, err
	}
	return start, end, nil
}

func checkCues(cues []Cue, format string) error {
	if len(cues) == 0 {
		return domain.E(domain.CodeFormatValidationFailed, "%s: no cues", format)
	}
	prevStart := -1
	for i, c := range cues {
		if c.EndMS <= c.StartMS {
			return domain.E(domain.CodeFormatValidationFailed,
				"%s: cue %d end %d <= start %d", format, i+1, c.EndMS, c.StartMS)
		}
		if c.StartMS < prevStart {
			return domain.E(domain.CodeFormatValidationFailed,
				"%s: cue %d starts before previous cue (ordering violation)", format, i+1)
		}
		if strings.TrimSpace(c.Text) == "" {
			return domain.E(domain.CodeFormatValidationFailed, "%s: cue %d has empty text", format, i+1)
		}
		prevStart = c.StartMS
	}
	return nil
}

// ParseSRT parses SRT content, enforcing sequential indices and blocks.
func ParseSRT(data []byte) ([]Cue, error) {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	blocks := strings.Split(strings.TrimRight(text, "\n"), "\n\n")
	var cues []Cue
	for bi, block := range blocks {
		lines := strings.Split(strings.TrimSpace(block), "\n")
		if len(lines) < 3 {
			return nil, domain.E(domain.CodeFormatValidationFailed,
				"srt: block %d has %d lines, need index, timing, text", bi+1, len(lines))
		}
		idx, err := strconv.Atoi(strings.TrimSpace(lines[0]))
		if err != nil || idx != bi+1 {
			return nil, domain.E(domain.CodeFormatValidationFailed,
				"srt: block %d has bad index %q (must be sequential from 1)", bi+1, lines[0])
		}
		start, end, err := parseTiming(lines[1], ",")
		if err != nil {
			return nil, domain.E(domain.CodeFormatValidationFailed, "srt: block %d: %v", bi+1, err)
		}
		cues = append(cues, Cue{StartMS: start, EndMS: end, Text: strings.Join(lines[2:], "\n")})
	}
	return cues, nil
}

// ValidateSRT parses SRT and verifies timestamps and cue ordering.
func ValidateSRT(data []byte) error {
	cues, err := ParseSRT(data)
	if err != nil {
		return err
	}
	return checkCues(cues, "srt")
}

// ParseVTT parses WebVTT content.
func ParseVTT(data []byte) ([]Cue, error) {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || !strings.HasPrefix(strings.TrimSpace(lines[0]), "WEBVTT") {
		return nil, domain.E(domain.CodeFormatValidationFailed, "vtt: missing WEBVTT header")
	}
	var cues []Cue
	i := 1
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "NOTE") {
			i++
			continue
		}
		if !strings.Contains(line, "-->") {
			// optional cue identifier
			i++
			if i >= len(lines) || !strings.Contains(lines[i], "-->") {
				return nil, domain.E(domain.CodeFormatValidationFailed, "vtt: expected timing after identifier %q", line)
			}
			line = strings.TrimSpace(lines[i])
		}
		start, end, err := parseTiming(line, ".")
		if err != nil {
			return nil, domain.E(domain.CodeFormatValidationFailed, "vtt: %v", err)
		}
		i++
		var textLines []string
		for i < len(lines) && strings.TrimSpace(lines[i]) != "" {
			textLines = append(textLines, strings.TrimSpace(lines[i]))
			i++
		}
		cues = append(cues, Cue{StartMS: start, EndMS: end, Text: strings.Join(textLines, "\n")})
	}
	return cues, nil
}

// ValidateVTT parses WebVTT and verifies timestamps and cue ordering.
func ValidateVTT(data []byte) error {
	cues, err := ParseVTT(data)
	if err != nil {
		return err
	}
	return checkCues(cues, "vtt")
}
