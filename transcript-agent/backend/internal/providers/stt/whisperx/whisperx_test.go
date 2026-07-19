package whisperx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aaraminds/transcript-agent/internal/domain"
)

// fakeSidecar implements the frozen sidecar API for contract tests:
// POST /v1/jobs -> 202, GET /v1/jobs/{id} -> 200, DELETE /v1/jobs/{id} -> 204.
type fakeSidecar struct {
	t *testing.T

	mu       sync.Mutex
	submits  []submittedJob
	deletes  []string
	getCount int

	// pollsUntilDone is how many GET polls return "processing" before the
	// terminal response is served.
	pollsUntilDone int
	terminal       map[string]any // body of the terminal GET response
	submitStatus   int            // non-zero: fail POST /v1/jobs with this code
	neverFinish    bool           // GETs always answer "processing"
}

type submittedJob struct {
	language          string
	enableDiarization string
	minSpeakers       string
	maxSpeakers       string
	filename          string
	fileBytes         string
}

func (f *fakeSidecar) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/jobs", func(w http.ResponseWriter, r *http.Request) {
		if f.submitStatus != 0 {
			http.Error(w, "sidecar rejected", f.submitStatus)
			return
		}
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			f.t.Errorf("parse multipart: %v", err)
			http.Error(w, "bad multipart", http.StatusBadRequest)
			return
		}
		sub := submittedJob{
			language:          r.FormValue("language"),
			enableDiarization: r.FormValue("enable_diarization"),
			minSpeakers:       r.FormValue("min_speakers"),
			maxSpeakers:       r.FormValue("max_speakers"),
		}
		file, hdr, err := r.FormFile("file")
		if err != nil {
			f.t.Errorf("form file: %v", err)
			http.Error(w, "missing file", http.StatusBadRequest)
			return
		}
		data, _ := io.ReadAll(file)
		file.Close()
		sub.filename = hdr.Filename
		sub.fileBytes = string(data)
		f.mu.Lock()
		f.submits = append(f.submits, sub)
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprint(w, `{"job_id":"job-abc-123","status":"queued"}`)
	})
	mux.HandleFunc("GET /v1/jobs/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.getCount++
		n := f.getCount
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if f.neverFinish || n <= f.pollsUntilDone {
			fmt.Fprintf(w, `{"job_id":%q,"status":"processing","error":null,"result":null}`, r.PathValue("id"))
			return
		}
		body := f.terminal
		if body == nil {
			body = map[string]any{"job_id": r.PathValue("id"), "status": "processing", "error": nil, "result": nil}
		}
		if err := json.NewEncoder(w).Encode(body); err != nil {
			f.t.Errorf("encode terminal body: %v", err)
		}
	})
	mux.HandleFunc("DELETE /v1/jobs/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.deletes = append(f.deletes, r.PathValue("id"))
		f.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
	return mux
}

func doneBody(diarizationApplied bool, segments []map[string]any) map[string]any {
	return map[string]any{
		"job_id": "job-abc-123",
		"status": "done",
		"error":  nil,
		"result": map[string]any{
			"language":            "en",
			"duration_seconds":    1234.5,
			"model":               "large-v3-turbo",
			"diarization_applied": diarizationApplied,
			"segments":            segments,
		},
	}
}

func writeArtifact(t *testing.T) (dataDir, uri string) {
	t.Helper()
	dataDir = t.TempDir()
	key := filepath.Join("job-1", "audio", "normalized.wav")
	path := filepath.Join(dataDir, key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("RIFF fake wav"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dataDir, localScheme + key
}

func newTestProvider(srvURL, dataDir string, timeout time.Duration) *Provider {
	return New(Config{
		BaseURL:      srvURL,
		LocalDataDir: dataDir,
		PollInterval: 5 * time.Millisecond,
		Timeout:      timeout,
	})
}

func TestTranscribeHappyPathWithDiarization(t *testing.T) {
	dataDir, uri := writeArtifact(t)
	fake := &fakeSidecar{
		t:              t,
		pollsUntilDone: 2, // queued/processing polls before done
		terminal: doneBody(true, []map[string]any{
			// SPEAKER_01 appears first: first-appearance ordinal mapping must
			// label it "Speaker 1" and SPEAKER_00 "Speaker 2".
			{"start_ms": 0, "end_ms": 4200, "text": "  Hello there. ", "speaker": "SPEAKER_01", "confidence": 0.93},
			{"start_ms": 4400, "end_ms": 8000, "text": "Hi back.", "speaker": "SPEAKER_00", "confidence": 0.87},
			{"start_ms": 8200, "end_ms": 9000, "text": "Great.", "speaker": "SPEAKER_01", "confidence": 0.71},
			{"start_ms": 9100, "end_ms": 9100, "text": "Yes.", "speaker": nil, "confidence": nil},
		}),
	}
	srv := httptest.NewServer(fake.handler())
	defer srv.Close()

	p := newTestProvider(srv.URL, dataDir, time.Minute)
	speakers := 2
	res, err := p.TranscribeWithSpeakerHint(context.Background(), uri, "en", true, &speakers)
	if err != nil {
		t.Fatal(err)
	}

	if len(fake.submits) != 1 {
		t.Fatalf("submits %d", len(fake.submits))
	}
	sub := fake.submits[0]
	if sub.language != "en" || sub.enableDiarization != "true" {
		t.Fatalf("submitted form %+v", sub)
	}
	if sub.minSpeakers != "2" || sub.maxSpeakers != "2" {
		t.Fatalf("speaker hint not forwarded: %+v", sub)
	}
	if sub.filename != "normalized.wav" || sub.fileBytes != "RIFF fake wav" {
		t.Fatalf("audio upload %+v", sub)
	}

	if res.Provider != "whisperx" || res.Model != "large-v3-turbo" || res.RequestID != "job-abc-123" {
		t.Fatalf("metadata %+v", res)
	}
	if !res.DiarizationAvailable {
		t.Fatal("diarization should be available")
	}
	if len(res.Segments) != 4 {
		t.Fatalf("segments %d", len(res.Segments))
	}
	if res.Segments[0].SpeakerLabel != "Speaker 1" || res.Segments[1].SpeakerLabel != "Speaker 2" || res.Segments[2].SpeakerLabel != "Speaker 1" {
		t.Fatalf("speaker mapping %+v", res.Segments)
	}
	if res.Segments[3].SpeakerLabel != "Speaker 1" {
		t.Fatalf("null speaker should fall back to Speaker 1: %+v", res.Segments[3])
	}
	if res.Segments[0].Text != "Hello there." {
		t.Fatalf("text not trimmed: %q", res.Segments[0].Text)
	}
	if res.Segments[0].StartMS != 0 || res.Segments[0].EndMS != 4200 {
		t.Fatalf("timing %+v", res.Segments[0])
	}
	if res.Segments[0].Confidence != 0.93 || res.Segments[2].Confidence != 0.71 {
		t.Fatalf("confidence passthrough %+v", res.Segments)
	}
	if res.Segments[3].Confidence != 0.8 {
		t.Fatalf("null confidence should default to 0.8, got %v", res.Segments[3].Confidence)
	}
	if res.Segments[3].EndMS != res.Segments[3].StartMS+1 {
		t.Fatalf("zero-length segment not fixed: %+v", res.Segments[3])
	}
}

func TestTranscribeNoDiarizationPath(t *testing.T) {
	dataDir, uri := writeArtifact(t)
	fake := &fakeSidecar{
		t: t,
		terminal: doneBody(false, []map[string]any{
			{"start_ms": 0, "end_ms": 4200, "text": "Solo voice.", "speaker": nil, "confidence": 0.9},
			{"start_ms": 4400, "end_ms": 8000, "text": "Still solo.", "speaker": nil, "confidence": 0.85},
		}),
	}
	srv := httptest.NewServer(fake.handler())
	defer srv.Close()

	p := newTestProvider(srv.URL, dataDir, time.Minute)
	// Diarization requested but the sidecar could not apply it (no HF_TOKEN).
	res, err := p.Transcribe(context.Background(), uri, "en", true)
	if err != nil {
		t.Fatal(err)
	}
	if res.DiarizationAvailable {
		t.Fatal("diarization must be reported unavailable when the sidecar did not apply it")
	}
	for _, s := range res.Segments {
		if s.SpeakerLabel != "Speaker 1" {
			t.Fatalf("null speakers must map to Speaker 1: %+v", s)
		}
	}
	if fake.submits[0].minSpeakers != "" || fake.submits[0].maxSpeakers != "" {
		t.Fatalf("no speaker hint expected: %+v", fake.submits[0])
	}
}

func TestTranscribeDiarizationNotRequested(t *testing.T) {
	dataDir, uri := writeArtifact(t)
	fake := &fakeSidecar{
		t: t,
		terminal: doneBody(false, []map[string]any{
			{"start_ms": 0, "end_ms": 1000, "text": "Hello.", "speaker": nil, "confidence": 0.9},
		}),
	}
	srv := httptest.NewServer(fake.handler())
	defer srv.Close()

	p := newTestProvider(srv.URL, dataDir, time.Minute)
	res, err := p.Transcribe(context.Background(), uri, "", false)
	if err != nil {
		t.Fatal(err)
	}
	// Parity with the azure provider: not requested != unavailable.
	if !res.DiarizationAvailable {
		t.Fatal("diarization must not be marked unavailable when it was not requested")
	}
	if fake.submits[0].enableDiarization != "false" {
		t.Fatalf("enable_diarization %q", fake.submits[0].enableDiarization)
	}
	if fake.submits[0].language != "en" {
		t.Fatalf("empty language must default to en, got %q", fake.submits[0].language)
	}
}

func TestTranscribeSidecarJobError(t *testing.T) {
	dataDir, uri := writeArtifact(t)
	cases := []struct {
		code, want string
	}{
		{"LANGUAGE_UNSUPPORTED", domain.CodeLanguageUnsupported},
		{"AUDIO_DECODE_FAILED", domain.CodeNoAudioTrack},
		{"TRANSCRIBE_FAILED", domain.CodeInternalError},
	}
	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			fake := &fakeSidecar{
				t: t,
				terminal: map[string]any{
					"job_id": "job-abc-123",
					"status": "error",
					"error":  map[string]any{"code": tc.code, "message": "boom"},
					"result": nil,
				},
			}
			srv := httptest.NewServer(fake.handler())
			defer srv.Close()

			p := newTestProvider(srv.URL, dataDir, time.Minute)
			_, err := p.Transcribe(context.Background(), uri, "en", true)
			if got := domain.CodeOf(err); got != tc.want {
				t.Fatalf("code %s, want %s (err=%v)", got, tc.want, err)
			}
		})
	}
}

func TestTranscribeSubmitHTTPErrors(t *testing.T) {
	dataDir, uri := writeArtifact(t)
	cases := []struct {
		status int
		want   string
	}{
		{http.StatusTooManyRequests, domain.CodeSTTProviderQuotaExceeded},
		{http.StatusInternalServerError, domain.CodeSTTProviderTimeout},
		{http.StatusBadRequest, domain.CodeValidationError},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprint(tc.status), func(t *testing.T) {
			fake := &fakeSidecar{t: t, submitStatus: tc.status}
			srv := httptest.NewServer(fake.handler())
			defer srv.Close()

			p := newTestProvider(srv.URL, dataDir, time.Minute)
			_, err := p.Transcribe(context.Background(), uri, "en", true)
			if got := domain.CodeOf(err); got != tc.want {
				t.Fatalf("code %s, want %s (err=%v)", got, tc.want, err)
			}
		})
	}
}

func TestTranscribeSidecarUnreachable(t *testing.T) {
	dataDir, uri := writeArtifact(t)
	// Reserve a port and close it so nothing listens there.
	srv := httptest.NewServer(http.NotFoundHandler())
	base := srv.URL
	srv.Close()

	p := newTestProvider(base, dataDir, time.Second)
	_, err := p.Transcribe(context.Background(), uri, "en", true)
	if got := domain.CodeOf(err); got != domain.CodeSTTProviderTimeout {
		t.Fatalf("code %s, want %s (err=%v)", got, domain.CodeSTTProviderTimeout, err)
	}
	if !strings.Contains(err.Error(), "unreachable") {
		t.Fatalf("error should mention unreachable sidecar: %v", err)
	}
}

func TestTranscribePollTimeout(t *testing.T) {
	dataDir, uri := writeArtifact(t)
	fake := &fakeSidecar{t: t, neverFinish: true}
	srv := httptest.NewServer(fake.handler())
	defer srv.Close()

	p := newTestProvider(srv.URL, dataDir, 60*time.Millisecond)
	_, err := p.Transcribe(context.Background(), uri, "en", true)
	if got := domain.CodeOf(err); got != domain.CodeSTTProviderTimeout {
		t.Fatalf("code %s, want %s (err=%v)", got, domain.CodeSTTProviderTimeout, err)
	}
	if !strings.Contains(err.Error(), "WHISPERX_TIMEOUT") {
		t.Fatalf("timeout error should name WHISPERX_TIMEOUT: %v", err)
	}
	// Best-effort cleanup: the abandoned sidecar job gets a DELETE.
	deadline := time.Now().Add(2 * time.Second)
	for {
		fake.mu.Lock()
		n := len(fake.deletes)
		fake.mu.Unlock()
		if n == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected 1 DELETE for the abandoned job, got %d", n)
		}
		time.Sleep(5 * time.Millisecond)
	}
	fake.mu.Lock()
	deleted := fake.deletes[0]
	fake.mu.Unlock()
	if deleted != "job-abc-123" {
		t.Fatalf("deleted job %q", deleted)
	}
}

func TestTranscribeContextCancellation(t *testing.T) {
	dataDir, uri := writeArtifact(t)
	fake := &fakeSidecar{t: t, neverFinish: true}
	srv := httptest.NewServer(fake.handler())
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	p := newTestProvider(srv.URL, dataDir, time.Hour)
	start := time.Now()
	_, err := p.Transcribe(ctx, uri, "en", true)
	if got := domain.CodeOf(err); got != domain.CodeSTTProviderTimeout {
		t.Fatalf("code %s, want %s (err=%v)", got, domain.CodeSTTProviderTimeout, err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("cancellation took %s; poll loop is not ctx-aware", elapsed)
	}
}

func TestTranscribeJobEvictedReturns404AsRetryable(t *testing.T) {
	dataDir, uri := writeArtifact(t)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/jobs", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprint(w, `{"job_id":"gone-1","status":"queued"}`)
	})
	mux.HandleFunc("GET /v1/jobs/{id}", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":{"code":"JOB_NOT_FOUND","message":"unknown job"}}`, http.StatusNotFound)
	})
	mux.HandleFunc("DELETE /v1/jobs/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	p := newTestProvider(srv.URL, dataDir, time.Minute)
	_, err := p.Transcribe(context.Background(), uri, "en", true)
	if got := domain.CodeOf(err); got != domain.CodeSTTProviderTimeout {
		t.Fatalf("code %s, want %s (err=%v)", got, domain.CodeSTTProviderTimeout, err)
	}
}

func TestTranscribeEmptySegmentsIsError(t *testing.T) {
	dataDir, uri := writeArtifact(t)
	fake := &fakeSidecar{t: t, terminal: doneBody(true, []map[string]any{})}
	srv := httptest.NewServer(fake.handler())
	defer srv.Close()

	p := newTestProvider(srv.URL, dataDir, time.Minute)
	_, err := p.Transcribe(context.Background(), uri, "en", true)
	if got := domain.CodeOf(err); got != domain.CodeInternalError {
		t.Fatalf("code %s, want %s (err=%v)", got, domain.CodeInternalError, err)
	}
}

func TestTranscribeRejectsUnknownScheme(t *testing.T) {
	p := newTestProvider("http://localhost:1", t.TempDir(), time.Minute)
	_, err := p.Transcribe(context.Background(), "mock://uploads/x.mp3", "en", true)
	if got := domain.CodeOf(err); got != domain.CodeMediaNotFound {
		t.Fatalf("code %s, want %s (err=%v)", got, domain.CodeMediaNotFound, err)
	}
	_, err = p.Transcribe(context.Background(), localScheme+"../etc/passwd", "en", true)
	if got := domain.CodeOf(err); got != domain.CodeValidationError {
		t.Fatalf("path traversal must be rejected, got %s (err=%v)", got, err)
	}
}
