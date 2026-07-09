// Package api implements the frozen REST contract under /api/v1 that the
// React review UI (../web) was built against. Auth is header-based
// (X-User-Id, X-User-Role), errors are {"error":{"code","message"}}, and CORS
// allows the local Vite dev server.
package api

import (
	"crypto/rand"
	"log/slog"
	"net/http"

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
	Log            *slog.Logger

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

	mux.HandleFunc("POST /api/v1/signed-links", s.handleCreateSignedLink)
	mux.HandleFunc("GET /api/v1/jobs/{jobID}/audio", s.handleJobAudio)
	mux.HandleFunc("GET /api/v1/exports/{exportID}/download", s.handleDownloadExport)

	s.mux = mux
	return s
}

// Handler returns the middleware-wrapped root handler.
func (s *Server) Handler() http.Handler {
	return s.middleware(s.mux)
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
