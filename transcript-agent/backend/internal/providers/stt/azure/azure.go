// Package azure implements Azure Speech fast transcription for the STT
// provider interface. It supports the local object-store artifacts produced by
// this backend (local://...) and public audio URLs.
package azure

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aaraminds/transcript-agent/internal/domain"
	"github.com/aaraminds/transcript-agent/internal/providers/stt"
)

const (
	defaultAPIVersion = "2025-10-15"
	defaultModelName  = "azure-fast-transcription"
	localScheme       = "local://"
)

// Config holds the Azure Speech resource settings. Key material must come
// from a secrets manager in production (PRD 16.6); env vars in dev only.
type Config struct {
	// Endpoint is the Speech/Foundry resource endpoint, for example:
	// https://<resource>.cognitiveservices.azure.com
	Endpoint string
	// Region is a legacy fallback used to construct
	// https://<region>.api.cognitive.microsoft.com when Endpoint is omitted.
	Region string
	Key    string
	Model  string
	// LocalDataDir is the DATA_DIR used by the local object store. It lets the
	// provider resolve local:// artifacts without accepting user-supplied paths.
	LocalDataDir string
	APIVersion   string
	MaxSpeakers  int
}

// Provider is the Azure Speech fast transcription client.
type Provider struct {
	cfg    Config
	client *http.Client
}

// New returns the provider. It does not validate credentials eagerly.
func New(cfg Config) *Provider {
	return &Provider{cfg: cfg, client: &http.Client{Timeout: 10 * time.Minute}}
}

// Configured reports whether the provider has enough config to be used.
func (p *Provider) Configured() bool {
	return p.cfg.Key != "" && (p.cfg.Endpoint != "" || p.cfg.Region != "")
}

// Transcribe submits an audio file to Azure fast transcription and maps the
// synchronous response into the local STT segment contract.
func (p *Provider) Transcribe(ctx context.Context, audioArtifactURI, language string, enableDiarization bool) (*stt.Result, error) {
	if !p.Configured() {
		return nil, domain.E(domain.CodeNotConfigured,
			"azure STT provider is not configured (set AZURE_SPEECH_ENDPOINT or AZURE_SPEECH_REGION, plus AZURE_SPEECH_KEY)")
	}
	locale, err := azureLocale(language)
	if err != nil {
		return nil, err
	}
	body, contentType, err := p.multipartBody(audioArtifactURI, locale, enableDiarization)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.transcribeURL(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Ocp-Apim-Subscription-Key", p.cfg.Key)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")

	client := p.client
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(req)
	if err != nil {
		if timeoutErr(ctx, err) {
			return nil, domain.E(domain.CodeSTTProviderTimeout, "azure STT request timed out: %v", err)
		}
		return nil, domain.E(domain.CodeSTTProviderTimeout, "azure STT request failed: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		msg := limitedBody(res.Body)
		return nil, azureHTTPError(res.StatusCode, msg)
	}
	var out fastResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, domain.E(domain.CodeInternalError, "decode azure STT response: %v", err)
	}
	result, err := mapFastResponse(out, enableDiarization)
	if err != nil {
		return nil, err
	}
	result.RequestID = firstHeader(res.Header, "x-ms-request-id", "apim-request-id", "x-request-id")
	result.Model = p.cfg.Model
	if result.Model == "" {
		result.Model = defaultModelName
	}
	return result, nil
}

func (p *Provider) transcribeURL() string {
	endpoint := strings.TrimRight(p.cfg.Endpoint, "/")
	if endpoint == "" {
		endpoint = fmt.Sprintf("https://%s.api.cognitive.microsoft.com", strings.TrimSpace(p.cfg.Region))
	}
	version := p.cfg.APIVersion
	if version == "" {
		version = defaultAPIVersion
	}
	return endpoint + "/speechtotext/transcriptions:transcribe?api-version=" + url.QueryEscape(version)
}

type fastDefinition struct {
	AudioURL    string              `json:"audioUrl,omitempty"`
	Locales     []string            `json:"locales,omitempty"`
	Diarization *diarizationOptions `json:"diarization,omitempty"`
}

type diarizationOptions struct {
	Enabled     bool `json:"enabled"`
	MaxSpeakers int  `json:"maxSpeakers,omitempty"`
}

func (p *Provider) multipartBody(audioArtifactURI, locale string, enableDiarization bool) (io.ReadCloser, string, error) {
	def := fastDefinition{Locales: []string{locale}}
	if enableDiarization {
		def.Diarization = &diarizationOptions{Enabled: true, MaxSpeakers: p.cfg.MaxSpeakers}
	}
	if strings.HasPrefix(audioArtifactURI, "http://") || strings.HasPrefix(audioArtifactURI, "https://") {
		def.AudioURL = audioArtifactURI
		return bufferedMultipart(def, "", nil)
	}
	if !strings.HasPrefix(audioArtifactURI, localScheme) {
		return nil, "", domain.E(domain.CodeMediaNotFound,
			"azure STT can transcribe local:// artifacts or public http(s) audio URLs, got %q", audioArtifactURI)
	}
	if p.cfg.LocalDataDir == "" {
		return nil, "", domain.E(domain.CodeNotConfigured,
			"azure STT local artifact resolution requires DATA_DIR / LocalDataDir")
	}
	path, err := localPath(p.cfg.LocalDataDir, audioArtifactURI)
	if err != nil {
		return nil, "", err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, "", domain.E(domain.CodeMediaNotFound, "audio artifact not found: %s", audioArtifactURI)
	}
	body, contentType := streamingMultipart(def, filepath.Base(path), file)
	return body, contentType, nil
}

func localPath(baseDir, uri string) (string, error) {
	key := strings.TrimPrefix(uri, localScheme)
	clean := filepath.Clean(key)
	if clean == "." || strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return "", domain.E(domain.CodeValidationError, "invalid local artifact uri %q", uri)
	}
	return filepath.Join(baseDir, clean), nil
}

func bufferedMultipart(def fastDefinition, filename string, data []byte) (io.ReadCloser, string, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	raw, err := json.Marshal(def)
	if err != nil {
		return nil, "", err
	}
	if err := mw.WriteField("definition", string(raw)); err != nil {
		return nil, "", err
	}
	if filename != "" {
		part, err := mw.CreateFormFile("audio", filename)
		if err != nil {
			return nil, "", err
		}
		if _, err := part.Write(data); err != nil {
			return nil, "", err
		}
	}
	if err := mw.Close(); err != nil {
		return nil, "", err
	}
	return io.NopCloser(&buf), mw.FormDataContentType(), nil
}

func streamingMultipart(def fastDefinition, filename string, file *os.File) (io.ReadCloser, string) {
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		defer file.Close()
		raw, err := json.Marshal(def)
		if err == nil {
			err = mw.WriteField("definition", string(raw))
		}
		if err == nil {
			var part io.Writer
			part, err = mw.CreateFormFile("audio", filename)
			if err == nil {
				_, err = io.Copy(part, file)
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

func azureLocale(language string) (string, error) {
	switch strings.TrimSpace(language) {
	case "", "en", "en-US":
		return "en-US", nil
	default:
		return "", domain.E(domain.CodeLanguageUnsupported,
			"azure STT MVP supports English only (language=en), got %q", language)
	}
}

type fastResponse struct {
	DurationMilliseconds int `json:"durationMilliseconds"`
	CombinedPhrases      []struct {
		Text string `json:"text"`
	} `json:"combinedPhrases"`
	Phrases []struct {
		Channel              int     `json:"channel"`
		Speaker              *int    `json:"speaker"`
		OffsetMilliseconds   int     `json:"offsetMilliseconds"`
		DurationMilliseconds int     `json:"durationMilliseconds"`
		Text                 string  `json:"text"`
		Locale               string  `json:"locale"`
		Confidence           float64 `json:"confidence"`
	} `json:"phrases"`
}

func mapFastResponse(in fastResponse, enableDiarization bool) (*stt.Result, error) {
	res := &stt.Result{Provider: "azure", DiarizationAvailable: !enableDiarization}
	speakerOrdinals := map[int]int{}
	nextSpeaker := 1
	anySpeaker := false
	for _, phrase := range in.Phrases {
		text := strings.TrimSpace(phrase.Text)
		if text == "" {
			continue
		}
		start := phrase.OffsetMilliseconds
		end := start + phrase.DurationMilliseconds
		if end <= start {
			end = start + 1
		}
		label := "Speaker 1"
		if phrase.Speaker != nil {
			anySpeaker = true
			id := *phrase.Speaker
			ordinal, ok := speakerOrdinals[id]
			if !ok {
				ordinal = nextSpeaker
				speakerOrdinals[id] = ordinal
				nextSpeaker++
			}
			label = fmt.Sprintf("Speaker %d", ordinal)
		}
		conf := phrase.Confidence
		if conf <= 0 {
			conf = 0.8
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
		for _, combined := range in.CombinedPhrases {
			text := strings.TrimSpace(combined.Text)
			if text == "" {
				continue
			}
			end := in.DurationMilliseconds
			if end <= 0 {
				end = 1
			}
			res.Segments = append(res.Segments, stt.Segment{
				StartMS:      0,
				EndMS:        end,
				SpeakerLabel: "Speaker 1",
				Text:         text,
				Confidence:   0.8,
			})
			break
		}
	}
	if len(res.Segments) == 0 {
		return nil, domain.E(domain.CodeInternalError, "azure STT returned no transcript phrases")
	}
	if enableDiarization {
		res.DiarizationAvailable = anySpeaker
	}
	return res, nil
}

func azureHTTPError(status int, body string) error {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return domain.E(domain.CodeNotConfigured, "azure STT authentication failed (HTTP %d): %s", status, body)
	case http.StatusTooManyRequests:
		return domain.E(domain.CodeSTTProviderQuotaExceeded, "azure STT quota exceeded (HTTP 429): %s", body)
	case http.StatusBadRequest:
		return domain.E(domain.CodeLanguageUnsupported, "azure STT rejected the request (HTTP 400): %s", body)
	case http.StatusRequestTimeout, http.StatusGatewayTimeout:
		return domain.E(domain.CodeSTTProviderTimeout, "azure STT timed out (HTTP %d): %s", status, body)
	}
	if status >= 500 {
		return domain.E(domain.CodeSTTProviderTimeout, "azure STT unavailable (HTTP %d): %s", status, body)
	}
	return domain.E(domain.CodeInternalError, "azure STT returned HTTP %d: %s", status, body)
}

func timeoutErr(ctx context.Context, err error) bool {
	if ctx.Err() != nil {
		return true
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

func limitedBody(r io.Reader) string {
	data, _ := io.ReadAll(io.LimitReader(r, 4096))
	msg := strings.TrimSpace(string(data))
	if msg == "" {
		return http.StatusText(http.StatusInternalServerError)
	}
	return msg
}

func firstHeader(h http.Header, keys ...string) string {
	for _, k := range keys {
		if v := h.Get(k); v != "" {
			return v
		}
	}
	return ""
}

var _ stt.Provider = (*Provider)(nil)
