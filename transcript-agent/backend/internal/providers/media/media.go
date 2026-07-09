// Package media defines the media processor interface (PRD 14.2 metadata,
// 14.6 audio extraction) with two implementations: an ffmpeg/ffprobe-backed
// processor for local upload files when the binaries are present, and a
// deterministic stub otherwise (dev / tests / YouTube sources without a
// downloader).
package media

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/aaraminds/transcript-agent/internal/domain"
)

// Metadata is the media property set from PRD 14.2.
type Metadata struct {
	DurationSeconds int
	Format          string
	AudioTracks     int
	VideoTracks     int
	Codec           string
	SampleRateHz    int
}

// Processor extracts metadata and normalized audio from source media.
type Processor interface {
	Metadata(ctx context.Context, sourceType, sourceURI string) (*Metadata, error)
	// ExtractAudio returns normalized mono 16 kHz WAV bytes plus metadata.
	ExtractAudio(ctx context.Context, sourceType, sourceURI string) ([]byte, *Metadata, error)
}

// Auto returns the ffmpeg-backed processor when ffmpeg and ffprobe are on
// PATH, otherwise the deterministic stub.
func Auto() Processor {
	if _, err := exec.LookPath("ffprobe"); err == nil {
		if _, err := exec.LookPath("ffmpeg"); err == nil {
			return &FFmpeg{fallback: NewStub()}
		}
	}
	return NewStub()
}

func extOf(uri string) string {
	e := strings.ToLower(strings.TrimPrefix(filepath.Ext(strings.SplitN(uri, "?", 2)[0]), "."))
	return e
}

func supportedExt(ext string) bool {
	for _, s := range domain.SupportedUploadExtensions {
		if ext == s {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------- Stub ---

// Stub is the deterministic processor used in mock mode. Behavior triggers
// (for tests and demos):
//
//	source_uri contains "missing"           -> MEDIA_NOT_FOUND
//	source_uri contains "noaudio"           -> zero audio tracks / NO_AUDIO_TRACK
//	source_uri contains "meta-timeout-once" -> METADATA_TIMEOUT on first call only
//	unsupported upload extension            -> UNSUPPORTED_FORMAT
type Stub struct {
	mu       sync.Mutex
	metaHits map[string]int
}

// NewStub returns a stub processor.
func NewStub() *Stub { return &Stub{metaHits: map[string]int{}} }

func (s *Stub) Metadata(_ context.Context, sourceType, sourceURI string) (*Metadata, error) {
	if strings.Contains(sourceURI, "missing") {
		return nil, domain.E(domain.CodeMediaNotFound, "source media not found: %s", sourceURI)
	}
	if strings.Contains(sourceURI, "meta-timeout-once") {
		s.mu.Lock()
		s.metaHits[sourceURI]++
		first := s.metaHits[sourceURI] == 1
		s.mu.Unlock()
		if first {
			return nil, domain.E(domain.CodeMetadataTimeout, "metadata probe timed out (transient)")
		}
	}
	m := &Metadata{DurationSeconds: 120, AudioTracks: 1, Codec: "aac", SampleRateHz: 48000}
	if sourceType == domain.SourceYouTube {
		m.Format = "mp4"
		m.VideoTracks = 1
	} else {
		ext := extOf(sourceURI)
		if !supportedExt(ext) {
			return nil, domain.E(domain.CodeUnsupportedFormat,
				"unsupported format %q; supported: %s", ext, strings.Join(domain.SupportedUploadExtensions, ", "))
		}
		m.Format = ext
		if ext == "mp4" || ext == "mov" {
			m.VideoTracks = 1
		}
	}
	if strings.Contains(sourceURI, "noaudio") {
		m.AudioTracks = 0
		m.Codec = ""
		m.SampleRateHz = 0
	}
	return m, nil
}

func (s *Stub) ExtractAudio(ctx context.Context, sourceType, sourceURI string) ([]byte, *Metadata, error) {
	meta, err := s.Metadata(ctx, sourceType, sourceURI)
	if err != nil {
		return nil, nil, err
	}
	if meta.AudioTracks == 0 {
		return nil, nil, domain.E(domain.CodeNoAudioTrack, "no audio track detected in %s", sourceURI)
	}
	out := &Metadata{DurationSeconds: meta.DurationSeconds, Format: "wav", AudioTracks: 1, Codec: "pcm_s16le", SampleRateHz: 16000}
	// Deterministic fake WAV payload (RIFF header + marker text).
	payload := append([]byte("RIFF....WAVEfmt mock-normalized-audio:"), []byte(sourceURI)...)
	return payload, out, nil
}

var _ Processor = (*Stub)(nil)

// -------------------------------------------------------------- FFmpeg ---

// FFmpeg shells out to ffprobe/ffmpeg for local upload files. YouTube
// sources fall back to the stub (media download is out of MVP backend scope;
// caption reuse or a pre-downloaded upload covers YouTube in MVP).
type FFmpeg struct {
	fallback *Stub
}

func localPath(uri string) (string, bool) {
	p := strings.TrimPrefix(uri, "file://")
	if !filepath.IsAbs(p) {
		return "", false
	}
	if _, err := os.Stat(p); err != nil {
		return "", false
	}
	return p, true
}

type ffprobeOut struct {
	Format struct {
		Duration   string `json:"duration"`
		FormatName string `json:"format_name"`
	} `json:"format"`
	Streams []struct {
		CodecType  string `json:"codec_type"`
		CodecName  string `json:"codec_name"`
		SampleRate string `json:"sample_rate"`
	} `json:"streams"`
}

func (f *FFmpeg) Metadata(ctx context.Context, sourceType, sourceURI string) (*Metadata, error) {
	p, ok := localPath(sourceURI)
	if sourceType != domain.SourceUpload || !ok {
		return f.fallback.Metadata(ctx, sourceType, sourceURI)
	}
	if !supportedExt(extOf(sourceURI)) {
		return nil, domain.E(domain.CodeUnsupportedFormat,
			"unsupported format %q; supported: %s", extOf(sourceURI), strings.Join(domain.SupportedUploadExtensions, ", "))
	}
	out, err := exec.CommandContext(ctx, "ffprobe", "-v", "quiet",
		"-print_format", "json", "-show_format", "-show_streams", p).Output()
	if err != nil {
		return nil, domain.E(domain.CodeMetadataTimeout, "ffprobe failed: %v", err)
	}
	var probe ffprobeOut
	if err := json.Unmarshal(out, &probe); err != nil {
		return nil, domain.E(domain.CodeMetadataTimeout, "ffprobe output parse failed: %v", err)
	}
	m := &Metadata{Format: extOf(sourceURI)}
	if d, err := strconv.ParseFloat(probe.Format.Duration, 64); err == nil {
		m.DurationSeconds = int(d)
	}
	for _, s := range probe.Streams {
		switch s.CodecType {
		case "audio":
			m.AudioTracks++
			m.Codec = s.CodecName
			if sr, err := strconv.Atoi(s.SampleRate); err == nil {
				m.SampleRateHz = sr
			}
		case "video":
			m.VideoTracks++
		}
	}
	return m, nil
}

func (f *FFmpeg) ExtractAudio(ctx context.Context, sourceType, sourceURI string) ([]byte, *Metadata, error) {
	p, ok := localPath(sourceURI)
	if sourceType != domain.SourceUpload || !ok {
		return f.fallback.ExtractAudio(ctx, sourceType, sourceURI)
	}
	meta, err := f.Metadata(ctx, sourceType, sourceURI)
	if err != nil {
		return nil, nil, err
	}
	if meta.AudioTracks == 0 {
		return nil, nil, domain.E(domain.CodeNoAudioTrack, "no audio track detected in %s", sourceURI)
	}
	tmp, err := os.CreateTemp("", "transcript-agent-*.wav")
	if err != nil {
		return nil, nil, domain.E(domain.CodeExtractionFailed, "temp file: %v", err)
	}
	tmpName := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpName)
	// Normalize to mono 16 kHz PCM WAV per 14.6 defaults.
	cmd := exec.CommandContext(ctx, "ffmpeg", "-y", "-i", p, "-vn", "-ac", "1", "-ar", "16000", "-f", "wav", tmpName)
	if err := cmd.Run(); err != nil {
		return nil, nil, domain.E(domain.CodeExtractionFailed, "ffmpeg extraction failed: %v", err)
	}
	data, err := os.ReadFile(tmpName)
	if err != nil {
		return nil, nil, domain.E(domain.CodeExtractionFailed, "read extracted audio: %v", err)
	}
	return data, &Metadata{DurationSeconds: meta.DurationSeconds, Format: "wav", AudioTracks: 1, Codec: "pcm_s16le", SampleRateHz: 16000}, nil
}

var _ Processor = (*FFmpeg)(nil)
