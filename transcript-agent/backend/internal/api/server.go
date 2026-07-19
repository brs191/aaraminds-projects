// Package api implements the frozen REST contract under /api/v1 that the
// React review UI (../web) was built against. Auth is header-based
// (X-User-Id, X-User-Role), errors are {"error":{"code","message"}}, and CORS
// allows the local Vite dev server.
package api

import (
	"crypto/rand"
	"expvar"
	"log/slog"
	"net/http"

	"github.com/aaraminds/transcript-agent/internal/domain"
	"github.com/aaraminds/transcript-agent/internal/objectstore"
	"github.com/aaraminds/transcript-agent/internal/orchestrator"
	"github.com/aaraminds/transcript-agent/internal/tools"
)

// Server holds handler dependencies.
type Server struct {
	Tools           *tools.Toolset
	Orch            *orchestrator.Orchestrator
	Objects         objectstore.ObjectStore
	CORSOrigin      string
	AuthProxySecret string
	// SigningSecret keys the HMAC for signed download/audio links
	// (SIGNING_SECRET env; random per boot when unset — links then die on
	// restart).
	SigningSecret []byte
	// MaxUploadBytes caps POST /api/v1/uploads bodies (MAX_UPLOAD_BYTES env).
	MaxUploadBytes int64
	// StaticDir, when set, serves the built React UI (WEB_DIST env) at /
	// with SPA fallback. Empty disables static serving (dev mode: Vite on
	// :5173 talks to the API cross-origin via CORS).
	StaticDir string
	Log       *slog.Logger

	mux *http.ServeMux
}

// DefaultMaxUploadBytes is the frozen 2 GiB default for MAX_UPLOAD_BYTES.
const DefaultMaxUploadBytes = int64(2) << 30

// NewServer wires routes. Method-qualified patterns require Go 1.22+.
func NewServer(ts *tools.Toolset, orch *orchestrator.Orchestrator, objects objectstore.ObjectStore, corsOrigin, authProxySecret string, signingSecret []byte, maxUploadBytes int64, log *slog.Logger) *Server {
	if corsOrigin == "" {
		corsOrigin = "http://localhost:5173"
	}
	if log == nil {
		log = slog.Default()
	}
	if len(signingSecret) == 0 {
		signingSecret = make([]byte, 32)
		if _, err := rand.Read(signingSecret); err != nil {
			panic("api: cannot generate ephemeral signing secret: " + err.Error())
		}
		log.Warn("SIGNING_SECRET is not set; using a random per-boot secret — signed links stop working on restart")
	}
	if maxUploadBytes <= 0 {
		maxUploadBytes = DefaultMaxUploadBytes
	}
	s := &Server{
		Tools:           ts,
		Orch:            orch,
		Objects:         objects,
		CORSOrigin:      corsOrigin,
		AuthProxySecret: authProxySecret,
		SigningSecret:   signingSecret,
		MaxUploadBytes:  maxUploadBytes,
		Log:             log,
	}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /api/v1/healthz", s.handleHealthz)
	// Internal operational metrics (PRD 18.2); auth-exempt, keep off the
	// public edge.
	mux.Handle("GET /debug/vars", expvar.Handler())

	mux.HandleFunc("POST /api/v1/jobs", s.handleSubmitJob)
	mux.HandleFunc("POST /api/v1/uploads", s.handleUploadMedia)
	mux.HandleFunc("GET /api/v1/jobs", s.handleListJobs)
	mux.HandleFunc("GET /api/v1/jobs/{jobID}", s.handleGetJob)
	mux.HandleFunc("POST /api/v1/jobs/{jobID}/caption-decision", s.handleCaptionDecision)
	mux.HandleFunc("POST /api/v1/jobs/{jobID}/replace-media", s.handleReplaceMedia)
	mux.HandleFunc("POST /api/v1/jobs/{jobID}/cancel", s.handleCancelJob)
	mux.HandleFunc("GET /api/v1/jobs/{jobID}/transcripts", s.handleListTranscripts)
	mux.HandleFunc("POST /api/v1/jobs/{jobID}/review", s.handleCreateReview)
	mux.HandleFunc("POST /api/v1/jobs/{jobID}/approve", s.handleApprove)
	mux.HandleFunc("GET /api/v1/jobs/{jobID}/approvals", s.handleListApprovals)
	mux.HandleFunc("POST /api/v1/jobs/{jobID}/reopen", s.handleReopen)
	mux.HandleFunc("POST /api/v1/jobs/{jobID}/summary", s.handleGenerateSummary)
	mux.HandleFunc("GET /api/v1/jobs/{jobID}/summary", s.handleGetSummary)
	mux.HandleFunc("GET /api/v1/jobs/{jobID}/quality-report", s.handleQualityReport)
	mux.HandleFunc("POST /api/v1/jobs/{jobID}/exports", s.handleCreateExports)
	mux.HandleFunc("GET /api/v1/jobs/{jobID}/exports", s.handleListExports)
	mux.HandleFunc("GET /api/v1/jobs/{jobID}/audit", s.handleAudit)

	mux.HandleFunc("GET /api/v1/transcripts/{versionID}/segments", s.handleListSegments)
	mux.HandleFunc("PATCH /api/v1/transcripts/{versionID}/segments/{segmentID}", s.handleEditSegment)

	mux.HandleFunc("PATCH /api/v1/summaries/{summaryID}", s.handleEditSummary)

	// Library mode (personal-use RSS feeds; frozen contract with the UI).
	mux.HandleFunc("POST /api/v1/library/feeds", s.handleAddFeed)
	mux.HandleFunc("GET /api/v1/library/feeds", s.handleListFeeds)
	mux.HandleFunc("DELETE /api/v1/library/feeds/{feedID}", s.handleDeleteFeed)
	mux.HandleFunc("POST /api/v1/library/feeds/{feedID}/poll", s.handlePollFeed)
	mux.HandleFunc("GET /api/v1/library/episodes", s.handleListEpisodes)
	mux.HandleFunc("POST /api/v1/library/episodes/{episodeID}/transcribe", s.handleTranscribeEpisode)
	mux.HandleFunc("GET /api/v1/library/search", s.handleLibrarySearch)

	mux.HandleFunc("POST /api/v1/signed-links", s.handleCreateSignedLink)
	mux.HandleFunc("GET /api/v1/jobs/{jobID}/audio", s.handleJobAudio)
	mux.HandleFunc("GET /api/v1/exports/{exportID}/download", s.handleDownloadExport)

	// Catch-all: built web UI with SPA fallback (no-op 404 until StaticDir
	// is set). Every registered pattern above is more specific and wins.
	mux.HandleFunc("GET /", s.handleStatic)

	s.mux = mux
	return s
}

// Handler returns the middleware-wrapped root handler.
func (s *Server) Handler() http.Handler {
	return s.middleware(s.mux)
}

// handleHealthz pings the job store (cheap status list) and the object store
// (small write probe). Either failing answers 503 {"status":"degraded"}
// (PRD 18.2).
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if _, err := s.Tools.Stores.Jobs.ListJobsByStatus(ctx, domain.StatusSubmitted); err != nil {
		s.Log.Error("healthz: job store ping failed", "error", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "degraded"})
		return
	}
	if _, err := s.Objects.Put(ctx, "healthz/probe", []byte("ok")); err != nil {
		s.Log.Error("healthz: object store probe failed", "error", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "degraded"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
