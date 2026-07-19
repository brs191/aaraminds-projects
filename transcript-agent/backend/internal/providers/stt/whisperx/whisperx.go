// Package whisperx implements the STT provider interface against the local
// WhisperX sidecar (stt-sidecar/): an async transcription service with
// faster-whisper + wav2vec2 alignment + optional pyannote diarization.
//
// The sidecar API is frozen (see stt-sidecar/README.md):
//
//	POST   /v1/jobs          multipart (file, language, enable_diarization,
//	                         min_speakers?, max_speakers?) -> 202 {job_id,status}
//	GET    /v1/jobs/{id}     -> 200 {job_id,status,error,result}
//	DELETE /v1/jobs/{id}     -> 204 (cancel/cleanup)
//
// The provider submits the audio artifact, polls with a context-aware backoff
// until the job reaches done/error, and maps the result into the local STT
// segment contract (SPEAKER_00 -> "Speaker 1" by first appearance).
package whisperx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aaraminds/transcript-agent/internal/domain"
	"github.com/aaraminds/transcript-agent/internal/providers/stt"
)

const (
	localScheme         = "local://"
	defaultBaseURL      = "http://localhost:9090"
	defaultPollInterval = 5 * time.Second
	defaultTimeout      = 2 * time.Hour
	// maxConsecutivePollFailures tolerates transient network blips while a
	// long transcription is running before the whole attempt is failed as
	// retryable STT_PROVIDER_TIMEOUT.
	maxConsecutivePollFailures = 3
)

// Config holds the sidecar connection settings (env: WHISPERX_URL,
// WHISPERX_POLL_INTERVAL, WHISPERX_TIMEOUT; DATA_DIR for LocalDataDir).
type Config struct {
	// BaseURL is the sidecar base URL, default http://localhost:9090.
	BaseURL string
	// LocalDataDir is the DATA_DIR used by the local object store. It lets the
	// provider resolve local:// artifacts without accepting user-supplied paths.
	LocalDataDir string
	// PollInterval is the wait between GET /v1/jobs/{id} polls (default 5s).
	PollInterval time.Duration
	// Timeout bounds the whole transcription (submit + poll), default 2h.
	Timeout time.Duration
	// HTTPClient overrides the HTTP client (tests). Default has no client
	// timeout: the overall deadline comes from Timeout via the context.
	HTTPClient *http.Client
}

// Provider is the WhisperX sidecar client.
type Provider struct {
	cfg    Config
	client *http.Client
}

// New returns the provider with defaults applied. It does not probe the
// sidecar eagerly; an unreachable sidecar surfaces as a retryable
// STT_PROVIDER_TIMEOUT at transcription time.
func New(cfg Config) *Provider {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = defaultPollInterval
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultTimeout
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{} // no client timeout; ctx deadline governs
	}
	return &Provider{cfg: cfg, client: client}
}

// Transcribe implements stt.Provider.
func (p *Provider) Transcribe(ctx context.Context, audioArtifactURI, language string, enableDiarization bool) (*stt.Result, error) {
	return p.TranscribeWithSpeakerHint(ctx, audioArtifactURI, language, enableDiarization, nil)
}

// TranscribeWithSpeakerHint implements stt.SpeakerHinter: when the job config
// carries expected_speaker_count, it is forwarded to the sidecar as
// min_speakers = max_speakers (pyannote diarization bounds).
func (p *Provider) TranscribeWithSpeakerHint(ctx context.Context, audioArtifactURI, language string, enableDiarization bool, expectedSpeakerCount *int) (*stt.Result, error) {
	if strings.TrimSpace(language) == "" {
		language = "en"
	}
	ctx, cancel := context.WithTimeout(ctx, p.cfg.Timeout)
	defer cancel()

	jobID, err := p.submit(ctx, audioArtifactURI, language, enableDiarization, expectedSpeakerCount)
	if err != nil {
		return nil, err
	}
	res, err := p.poll(ctx, jobID)
	if err != nil {
		// Best-effort cancel/cleanup on the sidecar; the job would otherwise
		// keep occupying the single-worker queue until its TTL.
		p.cancelJob(jobID)
		return nil, err
	}
	result, err := mapResult(res, enableDiarization)
	if err != nil {
		return nil, err
	}
	result.RequestID = jobID
	return result, nil
}

// ---------------------------------------------------------------------------
// submit
// ---------------------------------------------------------------------------

type submitResponse struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

func (p *Provider) submit(ctx context.Context, audioArtifactURI, language string, enableDiarization bool, expectedSpeakerCount *int) (string, error) {
	audio, filename, err := p.openAudio(ctx, audioArtifactURI)
	if err != nil {
		return "", err
	}
	body, contentType := multipartBody(audio, filename, language, enableDiarization, expectedSpeakerCount)
	defer body.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.BaseURL+"/v1/jobs", body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")

	res, err := p.client.Do(req)
	if err != nil {
		return "", transportError(ctx, "submit", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusAccepted && (res.StatusCode < 200 || res.StatusCode >= 300) {
		return "", httpError(res.StatusCode, limitedBody(res.Body))
	}
	var out submitResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return "", domain.E(domain.CodeInternalError, "decode whisperx submit response: %v", err)
	}
	if out.JobID == "" {
		return "", domain.E(domain.CodeInternalError, "whisperx sidecar returned no job_id")
	}
	return out.JobID, nil
}

// openAudio resolves the audio artifact into a reader. local:// artifacts are
// resolved against LocalDataDir (mirrors the azure provider — never
// user-supplied raw paths); public http(s) URLs are fetched and streamed.
func (p *Provider) openAudio(ctx context.Context, uri string) (io.ReadCloser, string, error) {
	if strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://") {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
		if err != nil {
			return nil, "", domain.E(domain.CodeInvalidSourceURI, "invalid audio url %q: %v", uri, err)
		}
		res, err := p.client.Do(req)
		if err != nil {
			return nil, "", domain.E(domain.CodeMediaNotFound, "fetch audio url %q: %v", uri, err)
		}
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			res.Body.Close()
			return nil, "", domain.E(domain.CodeMediaNotFound, "fetch audio url %q: HTTP %d", uri, res.StatusCode)
		}
		name := filepath.Base(req.URL.Path)
		if name == "" || name == "/" || name == "." {
			name = "audio"
		}
		return res.Body, name, nil
	}
	if !strings.HasPrefix(uri, localScheme) {
		return nil, "", domain.E(domain.CodeMediaNotFound,
			"whisperx STT can transcribe local:// artifacts or http(s) audio URLs, got %q", uri)
	}
	if p.cfg.LocalDataDir == "" {
		return nil, "", domain.E(domain.CodeNotConfigured,
			"whisperx STT local artifact resolution requires DATA_DIR / LocalDataDir")
	}
	path, err := localPath(p.cfg.LocalDataDir, uri)
	if err != nil {
		return nil, "", err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, "", domain.E(domain.CodeMediaNotFound, "audio artifact not found: %s", uri)
	}
	return file, filepath.Base(path), nil
}

func localPath(baseDir, uri string) (string, error) {
	key := strings.TrimPrefix(uri, localScheme)
	clean := filepath.Clean(key)
	if clean == "." || strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return "", domain.E(domain.CodeValidationError, "invalid local artifact uri %q", uri)
	}
	return filepath.Join(baseDir, clean), nil
}

// multipartBody streams the audio into a multipart request without buffering
// the full payload (audio artifacts can be large).
func multipartBody(audio io.ReadCloser, filename, language string, enableDiarization bool, expectedSpeakerCount *int) (io.ReadCloser, string) {
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		defer audio.Close()
		err := mw.WriteField("language", language)
		if err == nil {
			err = mw.WriteField("enable_diarization", strconv.FormatBool(enableDiarization))
		}
		if err == nil && enableDiarization && expectedSpeakerCount != nil && *expectedSpeakerCount > 0 {
			err = mw.WriteField("min_speakers", strconv.Itoa(*expectedSpeakerCount))
			if err == nil {
				err = mw.WriteField("max_speakers", strconv.Itoa(*expectedSpeakerCount))
			}
		}
		if err == nil {
			var part io.Writer
			part, err = mw.CreateFormFile("file", filename)
			if err == nil {
				_, err = io.Copy(part, audio)
			}
		}
		if closeErr := mw.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		_ = pw.Close()
	}()
	return pr, mw.FormDataContentType()
}

// ---------------------------------------------------------------------------
// poll
// ---------------------------------------------------------------------------

type sidecarError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type sidecarSegment struct {
	StartMS    int      `json:"start_ms"`
	EndMS      int      `json:"end_ms"`
	Text       string   `json:"text"`
	Speaker    *string  `json:"speaker"`
	Confidence *float64 `json:"confidence"`
}

type sidecarResult struct {
	Language           string           `json:"language"`
	DurationSeconds    float64          `json:"duration_seconds"`
	Model              string           `json:"model"`
	DiarizationApplied bool             `json:"diarization_applied"`
	Segments           []sidecarSegment `json:"segments"`
}

type jobStatusResponse struct {
	JobID  string         `json:"job_id"`
	Status string         `json:"status"`
	Error  *sidecarError  `json:"error"`
	Result *sidecarResult `json:"result"`
}

func (p *Provider) poll(ctx context.Context, jobID string) (*sidecarResult, error) {
	consecutiveFailures := 0
	for {
		status, err := p.getJob(ctx, jobID)
		switch {
		case err != nil && ctx.Err() != nil:
			return nil, pollDeadlineError(ctx, jobID, p.cfg.Timeout)
		case err != nil:
			consecutiveFailures++
			if consecutiveFailures >= maxConsecutivePollFailures {
				return nil, err
			}
		default:
			consecutiveFailures = 0
			switch status.Status {
			case "done":
				if status.Result == nil {
					return nil, domain.E(domain.CodeInternalError,
						"whisperx job %s reported done with no result", jobID)
				}
				return status.Result, nil
			case "error":
				return nil, mapSidecarError(jobID, status.Error)
			case "queued", "processing":
				// keep polling
			default:
				return nil, domain.E(domain.CodeInternalError,
					"whisperx job %s reported unknown status %q", jobID, status.Status)
			}
		}
		select {
		case <-ctx.Done():
			return nil, pollDeadlineError(ctx, jobID, p.cfg.Timeout)
		case <-time.After(p.cfg.PollInterval):
		}
	}
}

func (p *Provider) getJob(ctx context.Context, jobID string) (*jobStatusResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.cfg.BaseURL+"/v1/jobs/"+jobID, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	res, err := p.client.Do(req)
	if err != nil {
		return nil, transportError(ctx, "poll", err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		// Job evicted (sidecar restart or TTL). Retryable: resubmission
		// starts a fresh sidecar job.
		return nil, domain.E(domain.CodeSTTProviderTimeout,
			"whisperx job %s no longer exists on the sidecar (restarted or expired)", jobID)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, httpError(res.StatusCode, limitedBody(res.Body))
	}
	var out jobStatusResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, domain.E(domain.CodeInternalError, "decode whisperx job status: %v", err)
	}
	return &out, nil
}

// cancelJob is best-effort cleanup, detached from the (likely already
// cancelled) request context.
func (p *Provider) cancelJob(jobID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, p.cfg.BaseURL+"/v1/jobs/"+jobID, nil)
	if err != nil {
		return
	}
	if res, err := p.client.Do(req); err == nil {
		io.Copy(io.Discard, io.LimitReader(res.Body, 4096)) //nolint:errcheck
		res.Body.Close()
	}
}

// ---------------------------------------------------------------------------
// mapping
// ---------------------------------------------------------------------------

// mapResult maps the sidecar result into the local STT contract. Diarization
// speaker IDs (SPEAKER_00, SPEAKER_01, ...) become "Speaker 1", "Speaker 2"
// by first appearance; segments without a speaker fall back to "Speaker 1".
func mapResult(in *sidecarResult, enableDiarization bool) (*stt.Result, error) {
	res := &stt.Result{
		Provider:             "whisperx",
		Model:                in.Model,
		DiarizationAvailable: !enableDiarization,
	}
	if res.Model == "" {
		res.Model = "whisperx"
	}
	speakerOrdinals := map[string]int{}
	nextSpeaker := 1
	anySpeaker := false
	for _, seg := range in.Segments {
		text := strings.TrimSpace(seg.Text)
		if text == "" {
			continue
		}
		start := seg.StartMS
		end := seg.EndMS
		if end <= start {
			end = start + 1
		}
		label := "Speaker 1"
		if seg.Speaker != nil && *seg.Speaker != "" {
			anySpeaker = true
			ordinal, ok := speakerOrdinals[*seg.Speaker]
			if !ok {
				ordinal = nextSpeaker
				speakerOrdinals[*seg.Speaker] = ordinal
				nextSpeaker++
			}
			label = fmt.Sprintf("Speaker %d", ordinal)
		}
		conf := 0.8 // parity with the azure provider when confidence is absent
		if seg.Confidence != nil && *seg.Confidence > 0 {
			conf = *seg.Confidence
			if conf > 1 {
				conf = 1
			}
		}
		res.Segments = append(res.Segments, stt.Segment{
			StartMS:      start,
			EndMS:        end,
			SpeakerLabel: label,
			Text:         text,
			Confidence:   conf,
		})
	}
	if len(res.Segments) == 0 {
		return nil, domain.E(domain.CodeInternalError, "whisperx returned no transcript segments")
	}
	if enableDiarization {
		// Diarization was requested but the sidecar could not apply it (no
		// HF_TOKEN, pipeline failure, ...): the pipeline flags every segment
		// diarization_unavailable, mirroring the mock's no-diarization path.
		res.DiarizationAvailable = in.DiarizationApplied && anySpeaker
	}
	return res, nil
}

// mapSidecarError maps sidecar job error codes onto PRD 14.7 error codes.
func mapSidecarError(jobID string, e *sidecarError) error {
	if e == nil {
		return domain.E(domain.CodeInternalError, "whisperx job %s failed without error detail", jobID)
	}
	switch e.Code {
	case domain.CodeLanguageUnsupported:
		return domain.E(domain.CodeLanguageUnsupported, "whisperx: %s", e.Message)
	case "STT_PROVIDER_TIMEOUT", "CANCELLED":
		// Transient on the sidecar side; retryable per the PRD 19 matrix.
		return domain.E(domain.CodeSTTProviderTimeout, "whisperx: %s", e.Message)
	case "AUDIO_DECODE_FAILED", "NO_AUDIO_TRACK":
		return domain.E(domain.CodeNoAudioTrack, "whisperx: %s", e.Message)
	}
	return domain.E(domain.CodeInternalError, "whisperx job %s failed (%s): %s", jobID, e.Code, e.Message)
}

// httpError maps sidecar HTTP-level failures. 5xx and 429 follow the azure
// provider convention: retryable timeout / quota respectively.
func httpError(status int, body string) error {
	switch status {
	case http.StatusTooManyRequests:
		return domain.E(domain.CodeSTTProviderQuotaExceeded, "whisperx sidecar rejected the job (HTTP 429): %s", body)
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return domain.E(domain.CodeValidationError, "whisperx sidecar rejected the request (HTTP %d): %s", status, body)
	case http.StatusRequestTimeout, http.StatusGatewayTimeout:
		return domain.E(domain.CodeSTTProviderTimeout, "whisperx sidecar timed out (HTTP %d): %s", status, body)
	}
	if status >= 500 {
		return domain.E(domain.CodeSTTProviderTimeout, "whisperx sidecar unavailable (HTTP %d): %s", status, body)
	}
	return domain.E(domain.CodeInternalError, "whisperx sidecar returned HTTP %d: %s", status, body)
}

// transportError maps connection-level failures (sidecar down, DNS, ctx) to
// the retryable STT_PROVIDER_TIMEOUT, matching orchestrator retry semantics.
func transportError(ctx context.Context, phase string, err error) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return domain.E(domain.CodeSTTProviderTimeout, "whisperx %s timed out: %v", phase, err)
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return domain.E(domain.CodeSTTProviderTimeout, "whisperx %s timed out: %v", phase, err)
	}
	return domain.E(domain.CodeSTTProviderTimeout, "whisperx sidecar unreachable (%s): %v", phase, err)
}

func pollDeadlineError(ctx context.Context, jobID string, timeout time.Duration) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return domain.E(domain.CodeSTTProviderTimeout,
			"whisperx job %s did not finish within WHISPERX_TIMEOUT=%s", jobID, timeout)
	}
	return domain.E(domain.CodeSTTProviderTimeout, "whisperx transcription cancelled: %v", ctx.Err())
}

func limitedBody(r io.Reader) string {
	data, _ := io.ReadAll(io.LimitReader(r, 4096))
	msg := strings.TrimSpace(string(data))
	if msg == "" {
		return http.StatusText(http.StatusInternalServerError)
	}
	return msg
}

var (
	_ stt.Provider      = (*Provider)(nil)
	_ stt.SpeakerHinter = (*Provider)(nil)
)
