package exporter

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/aaraminds/transcript-agent/internal/domain"
)

func seg(startMS, endMS int, speaker, text string) *domain.Segment {
	return &domain.Segment{
		SegmentID: uuid.New(), StartMS: startMS, EndMS: endMS,
		SpeakerLabel: speaker, Text: text,
	}
}

func sampleSegments() []*domain.Segment {
	return []*domain.Segment{
		seg(0, 4200, "Speaker 1", "Welcome to the podcast."),
		seg(4400, 9000, "Speaker 2", "Thanks for having me."),
		seg(9200, 3661500, "Speaker 1", "Let's talk about long episodes."), // >1h end
	}
}

const goldenSRT = `1
00:00:00,000 --> 00:00:04,200
Speaker 1: Welcome to the podcast.

2
00:00:04,400 --> 00:00:09,000
Speaker 2: Thanks for having me.

3
00:00:09,200 --> 01:01:01,500
Speaker 1: Let's talk about long episodes.

`

const goldenVTT = `WEBVTT

00:00:00.000 --> 00:00:04.200
Speaker 1: Welcome to the podcast.

00:00:04.400 --> 00:00:09.000
Speaker 2: Thanks for having me.

00:00:09.200 --> 01:01:01.500
Speaker 1: Let's talk about long episodes.

`

func TestSRTGolden(t *testing.T) {
	out, err := SRT(sampleSegments())
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != goldenSRT {
		t.Errorf("srt output mismatch:\ngot:\n%s\nwant:\n%s", out, goldenSRT)
	}
}

func TestVTTGolden(t *testing.T) {
	out, err := VTT(sampleSegments())
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != goldenVTT {
		t.Errorf("vtt output mismatch:\ngot:\n%s\nwant:\n%s", out, goldenVTT)
	}
}

// Round-trip: generated output must pass the parse-back validator and yield
// the same timestamps.
func TestValidatorRoundTrip(t *testing.T) {
	segs := sampleSegments()

	srt, err := SRT(segs)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateSRT(srt); err != nil {
		t.Fatalf("generated srt failed validation: %v", err)
	}
	cues, err := ParseSRT(srt)
	if err != nil {
		t.Fatal(err)
	}
	if len(cues) != len(segs) {
		t.Fatalf("srt cue count %d != %d", len(cues), len(segs))
	}
	for i, c := range cues {
		if c.StartMS != segs[i].StartMS || c.EndMS != segs[i].EndMS {
			t.Errorf("srt cue %d timestamps %d-%d != %d-%d", i, c.StartMS, c.EndMS, segs[i].StartMS, segs[i].EndMS)
		}
	}

	vtt, err := VTT(segs)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateVTT(vtt); err != nil {
		t.Fatalf("generated vtt failed validation: %v", err)
	}
	vcues, err := ParseVTT(vtt)
	if err != nil {
		t.Fatal(err)
	}
	for i, c := range vcues {
		if c.StartMS != segs[i].StartMS || c.EndMS != segs[i].EndMS {
			t.Errorf("vtt cue %d timestamps %d-%d != %d-%d", i, c.StartMS, c.EndMS, segs[i].StartMS, segs[i].EndMS)
		}
	}
}

func TestValidatorsRejectBadInput(t *testing.T) {
	cases := map[string]struct {
		format string
		data   string
	}{
		"vtt missing header":   {"vtt", "00:00:00.000 --> 00:00:01.000\nhi\n"},
		"vtt end before start": {"vtt", "WEBVTT\n\n00:00:05.000 --> 00:00:01.000\nhi\n"},
		"vtt bad timestamp":    {"vtt", "WEBVTT\n\n00:00:xx.000 --> 00:00:01.000\nhi\n"},
		"srt bad index":        {"srt", "7\n00:00:00,000 --> 00:00:01,000\nhi\n"},
		"srt bad ordering":     {"srt", "1\n00:00:10,000 --> 00:00:12,000\nhi\n\n2\n00:00:01,000 --> 00:00:02,000\nbye\n"},
		"srt missing text":     {"srt", "1\n00:00:00,000 --> 00:00:01,000\n"},
		"srt vtt separator":    {"srt", "1\n00:00:00.000 --> 00:00:01.000\nhi\n"},
	}
	for name, tc := range cases {
		if err := Validate(tc.format, []byte(tc.data)); err == nil {
			t.Errorf("%s: expected validation failure", name)
		} else if domain.CodeOf(err) != domain.CodeFormatValidationFailed {
			t.Errorf("%s: want FORMAT_VALIDATION_FAILED, got %v", name, err)
		}
	}
}

func TestTxtAndMd(t *testing.T) {
	segs := sampleSegments()
	txt := TXT(segs)
	if !strings.Contains(string(txt), "[00:00:04.400] Speaker 2: Thanks for having me.") {
		t.Errorf("txt missing expected line:\n%s", txt)
	}
	md := MD(segs)
	if !strings.HasPrefix(string(md), "# Transcript") {
		t.Errorf("md missing header:\n%s", md)
	}
	if err := Validate("txt", txt); err != nil {
		t.Error(err)
	}
	if err := Validate("md", md); err != nil {
		t.Error(err)
	}
}

func TestEmptySegmentsSkipped(t *testing.T) {
	segs := []*domain.Segment{
		seg(0, 1000, "Speaker 1", "Hello."),
		seg(1200, 2000, "Speaker 2", "   "), // emptied by cleanup — must be skipped
		seg(2200, 3000, "Speaker 1", "Bye."),
	}
	srt, err := SRT(segs)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateSRT(srt); err != nil {
		t.Fatalf("srt with skipped empty segment failed validation: %v", err)
	}
	cues, _ := ParseSRT(srt)
	if len(cues) != 2 {
		t.Errorf("expected 2 cues after skipping empty segment, got %d", len(cues))
	}
}
