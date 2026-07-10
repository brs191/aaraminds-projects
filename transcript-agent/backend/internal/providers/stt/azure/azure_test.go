package azure

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aaraminds/transcript-agent/internal/domain"
)

func TestTranscribeLocalArtifact(t *testing.T) {
	dir := t.TempDir()
	audioKey := filepath.Join("job-1", "audio", "normalized.wav")
	audioPath := filepath.Join(dir, audioKey)
	if err := os.MkdirAll(filepath.Dir(audioPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(audioPath, []byte("RIFF fake wav"), 0o644); err != nil {
		t.Fatal(err)
	}

	var sawAudio bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/speechtotext/transcriptions:transcribe" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("api-version"); got != defaultAPIVersion {
			t.Fatalf("api-version %q", got)
		}
		if got := r.Header.Get("Ocp-Apim-Subscription-Key"); got != "test-key" {
			t.Fatalf("subscription key %q", got)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Fatalf("content-type %q", r.Header.Get("Content-Type"))
		}
		mr, err := r.MultipartReader()
		if err != nil {
			t.Fatal(err)
		}
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(part)
			if err != nil {
				t.Fatal(err)
			}
			switch part.FormName() {
			case "definition":
				var def fastDefinition
				if err := json.Unmarshal(data, &def); err != nil {
					t.Fatalf("definition JSON: %v", err)
				}
				if len(def.Locales) != 1 || def.Locales[0] != "en-US" {
					t.Fatalf("locales %+v", def.Locales)
				}
				if def.Diarization == nil || !def.Diarization.Enabled || def.Diarization.MaxSpeakers != 2 {
					t.Fatalf("diarization %+v", def.Diarization)
				}
			case "audio":
				sawAudio = true
				if part.FileName() != "normalized.wav" {
					t.Fatalf("audio filename %q", part.FileName())
				}
				if string(data) != "RIFF fake wav" {
					t.Fatalf("audio data %q", data)
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("x-ms-request-id", "req-123")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"durationMilliseconds": 2000,
			"combinedPhrases": [{"text": "Hello there. Hi back."}],
			"phrases": [
				{"speaker": 1, "offsetMilliseconds": 100, "durationMilliseconds": 500, "text": "Hello there.", "confidence": 0.91},
				{"speaker": 0, "offsetMilliseconds": 700, "durationMilliseconds": 400, "text": "Hi back.", "confidence": 0.87}
			]
		}`))
	}))
	defer srv.Close()

	p := New(Config{
		Endpoint:     srv.URL,
		Key:          "test-key",
		Model:        "custom-model",
		LocalDataDir: dir,
		MaxSpeakers:  2,
	})
	res, err := p.Transcribe(context.Background(), localScheme+audioKey, "en", true)
	if err != nil {
		t.Fatal(err)
	}
	if !sawAudio {
		t.Fatal("multipart audio field was not sent")
	}
	if res.Provider != "azure" || res.Model != "custom-model" || res.RequestID != "req-123" {
		t.Fatalf("metadata %+v", res)
	}
	if !res.DiarizationAvailable {
		t.Fatal("diarization should be available when speaker IDs are returned")
	}
	if len(res.Segments) != 2 {
		t.Fatalf("segments %d", len(res.Segments))
	}
	if res.Segments[0].SpeakerLabel != "Speaker 1" || res.Segments[1].SpeakerLabel != "Speaker 2" {
		t.Fatalf("speaker labels %+v", res.Segments)
	}
	if res.Segments[0].StartMS != 100 || res.Segments[0].EndMS != 600 {
		t.Fatalf("timing %+v", res.Segments[0])
	}
}

func TestTranscribePublicURLUsesDefinitionAudioURL(t *testing.T) {
	var sawAudioURL bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mr, err := r.MultipartReader()
		if err != nil {
			t.Fatal(err)
		}
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			data, _ := io.ReadAll(part)
			if part.FormName() == "audio" {
				t.Fatal("public URL request should not include inline audio")
			}
			if part.FormName() == "definition" {
				var def fastDefinition
				if err := json.Unmarshal(data, &def); err != nil {
					t.Fatal(err)
				}
				sawAudioURL = def.AudioURL == "https://example.com/podcast.wav"
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"durationMilliseconds": 1000,
			"combinedPhrases": [{"text": "Only combined text."}]
		}`))
	}))
	defer srv.Close()

	p := New(Config{Endpoint: srv.URL, Key: "test-key"})
	res, err := p.Transcribe(context.Background(), "https://example.com/podcast.wav", "en-US", false)
	if err != nil {
		t.Fatal(err)
	}
	if !sawAudioURL {
		t.Fatal("definition.audioUrl was not sent")
	}
	if len(res.Segments) != 1 || res.Segments[0].Text != "Only combined text." {
		t.Fatalf("combined fallback not mapped: %+v", res.Segments)
	}
	if !res.DiarizationAvailable {
		t.Fatal("diarization should not be marked unavailable when it was not requested")
	}
}

func TestTranscribeMapsQuotaError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "quota exceeded", http.StatusTooManyRequests)
	}))
	defer srv.Close()

	p := New(Config{Endpoint: srv.URL, Key: "test-key"})
	_, err := p.Transcribe(context.Background(), "https://example.com/podcast.wav", "en", true)
	if got := domain.CodeOf(err); got != domain.CodeSTTProviderQuotaExceeded {
		t.Fatalf("code %s, want %s (err=%v)", got, domain.CodeSTTProviderQuotaExceeded, err)
	}
}
