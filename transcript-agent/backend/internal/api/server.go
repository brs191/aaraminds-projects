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
	Tools               *tools.Toolset
	Orch                *orchestrator.Orchestrator
	Objects             objectstore.ObjectStore
	CORSOrigin          string
	AuthProxySecret     string
	DownloadTokenSecret []byte
	Log                 *slog.Logger

	mux *http.ServeMux
}

// NewServer wires routes. Method-qualified patterns require Go 1.22+.
func NewServer(ts *tools.Toolset, orch *orchestrator.Orchestrator, objects objectstore.ObjectStore, corsOrigin, authProxySecret string, downloadTokenSecret []byte, log *slog.Logger) *Server {
	if corsOrigin == "" {
		corsOrigin = "http://localhost:5173"
	}
	if len(downloadTokenSecret) == 0 {
		downloadTokenSecret = make([]byte, 32)
		if _, err := rand.Read(downloadTokenSecret); err != nil {
			downloadTokenSecret = []byte("transcript-agent-dev-download-secret")
		}
	}
	if log == nil {
		log = slog.Default()
	}
	s := &Server{
		Tools:               ts,
		Orch:                orch,
		Objects:             objects,
		CORSOrigin:          corsOrigin,
		AuthProxySecret:     authProxySecret,
		DownloadTokenSecret: downloadTokenSecret,
		Log:                 log,
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
