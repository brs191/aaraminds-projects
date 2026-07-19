// Package app wires stores, providers, tools, orchestrator and API server
// into one unit. Used by cmd/server and by the end-to-end test suite so both
// exercise identical wiring.
package app

import (
	"log/slog"
	"time"

	"github.com/aaraminds/transcript-agent/internal/api"
	"github.com/aaraminds/transcript-agent/internal/audit"
	"github.com/aaraminds/transcript-agent/internal/domain"
	"github.com/aaraminds/transcript-agent/internal/objectstore"
	"github.com/aaraminds/transcript-agent/internal/orchestrator"
	"github.com/aaraminds/transcript-agent/internal/providers/captions"
	"github.com/aaraminds/transcript-agent/internal/providers/llm"
	"github.com/aaraminds/transcript-agent/internal/providers/media"
	"github.com/aaraminds/transcript-agent/internal/providers/stt"
	"github.com/aaraminds/transcript-agent/internal/store"
	"github.com/aaraminds/transcript-agent/internal/tools"
)

// Options configures the app.
type Options struct {
	Log      *slog.Logger
	Stores   store.Stores
	Objects  objectstore.ObjectStore
	STT      stt.Provider
	LLM      llm.Provider
	Media    media.Processor
	Captions captions.Provider
	// STTName is recorded in job_config snapshots (PRD 13.3 stt_provider).
	STTName string
	// ConfigDefaults overrides PRD job_config defaults when set.
	ConfigDefaults *domain.JobConfig
	CORSOrigin     string
	// AuthProxySecret, when set, requires a trusted reverse proxy to attach
	// X-Auth-Proxy-Secret on authenticated API requests.
	AuthProxySecret string
	// SigningSecret keys the HMAC for signed download/audio links
	// (SIGNING_SECRET). Empty means a random per-boot secret.
	SigningSecret []byte
	// MaxUploadBytes caps POST /api/v1/uploads bodies (MAX_UPLOAD_BYTES);
	// zero means the 2 GiB default.
	MaxUploadBytes int64
	// Sync drives jobs inline on Enqueue (tests / single-process demos).
	Sync bool
	// Backoff between a retryable failure and its single retry.
	Backoff time.Duration
	// DrainTimeout bounds in-flight step completion after SIGTERM
	// (DRAIN_TIMEOUT env; zero means the 30s default).
	DrainTimeout time.Duration
	// StuckJobThreshold is the updated_at age past which mid-pipeline jobs are
	// reclaimed to queued (STUCK_JOB_THRESHOLD env; zero means the 10m default).
	StuckJobThreshold time.Duration
	// RetentionDays sets media_artifacts.retention_until at creation
	// (RETENTION_DAYS env; zero means the 30-day default).
	RetentionDays int
	// LibraryPollInterval is the library feed poll cadence
	// (LIBRARY_POLL_INTERVAL env; zero means the 30m default).
	LibraryPollInterval time.Duration
	// LibraryAutoPerPoll caps auto-transcribed new episodes per feed per poll
	// (LIBRARY_AUTO_PER_POLL env; zero means the default of 3).
	LibraryAutoPerPoll int
	// LibraryMaxDownloadBytes caps library enclosure downloads
	// (LIBRARY_MAX_DOWNLOAD_BYTES env; zero means the 500 MiB default).
	LibraryMaxDownloadBytes int64
	// WebDist, when set, serves the built React UI from this directory at /
	// with SPA fallback (WEB_DIST env). Empty disables static serving.
	WebDist string
}

// App is the wired application.
type App struct {
	Tools *tools.Toolset
	Orch  *orchestrator.Orchestrator
	API   *api.Server
}

// New wires everything.
func New(o Options) *App {
	if o.Log == nil {
		o.Log = slog.Default()
	}
	ts := &tools.Toolset{
		Stores:                  o.Stores,
		Objects:                 o.Objects,
		STT:                     o.STT,
		LLM:                     o.LLM,
		Media:                   o.Media,
		Captions:                o.Captions,
		STTProvider:             o.STTName,
		ConfigDefaults:          o.ConfigDefaults,
		RetentionDays:           o.RetentionDays,
		LibraryMaxDownloadBytes: o.LibraryMaxDownloadBytes,
		Auditor:                 audit.New(o.Stores.Audit, o.Log),
		Log:                     o.Log,
	}
	orch := orchestrator.New(ts, o.Log, o.Backoff, o.Sync)
	if o.DrainTimeout > 0 {
		orch.DrainTimeout = o.DrainTimeout
	}
	if o.StuckJobThreshold > 0 {
		orch.StuckThreshold = o.StuckJobThreshold
	}
	if o.LibraryPollInterval > 0 {
		orch.LibraryPollInterval = o.LibraryPollInterval
	}
	if o.LibraryAutoPerPoll > 0 {
		orch.LibraryAutoPerPoll = o.LibraryAutoPerPoll
	}
	srv := api.NewServer(ts, orch, o.Objects, o.CORSOrigin, o.AuthProxySecret, o.SigningSecret, o.MaxUploadBytes, o.Log)
	srv.StaticDir = o.WebDist
	return &App{Tools: ts, Orch: orch, API: srv}
}
