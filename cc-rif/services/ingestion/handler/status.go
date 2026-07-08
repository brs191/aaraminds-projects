package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/aaraminds/rif/ingestion/store"
)

// runStatusResponse is the JSON body returned by GET /repos/{repoID}/status.
type runStatusResponse struct {
	RunID       string     `json:"run_id"`
	Status      string     `json:"status"`
	SHA         string     `json:"sha"`
	NodeCount   *int32     `json:"node_count"`
	EdgeCount   *int32     `json:"edge_count"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

// GetStatus returns a handler for GET /repos/{repoID}/status.
//
// Returns the latest index_run for the repo:
//   - 200 with the run status JSON.
//   - 404 if no run exists for this repo or the repo itself is not registered.
//   - 500 on unexpected errors.
func GetStatus(rs *store.RunStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "repoID")

		rs2, err := rs.GetLatestRun(r.Context(), repoID)
		if err != nil {
			if errors.Is(err, store.ErrRunNotFound) {
				writeJSON(w, http.StatusNotFound, errResponse("no runs found for repo"))
				return
			}
			writeJSON(w, http.StatusInternalServerError, errResponse("failed to fetch run status"))
			return
		}

		writeJSON(w, http.StatusOK, runStatusResponse{
			RunID:       rs2.RunID,
			Status:      rs2.Stage,
			SHA:         rs2.SHA,
			NodeCount:   rs2.NodeCount,
			EdgeCount:   rs2.EdgeCount,
			StartedAt:   rs2.StartedAt,
			CompletedAt: rs2.CompletedAt,
		})
	}
}
