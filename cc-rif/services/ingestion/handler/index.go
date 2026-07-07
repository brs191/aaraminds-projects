package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/att/rif/ingestion/service"
	"github.com/att/rif/ingestion/store"
)

// triggerIndexRequest is the optional JSON body for POST /repos/{repoID}/index.
type triggerIndexRequest struct {
	SHA string `json:"sha"`
}

// TriggerIndex returns a handler for POST /repos/{repoID}/index.
//
// Starts an asynchronous indexing run and returns:
//   - 202 {"run_id":"..."} immediately (caller polls GET /repos/{repoID}/status).
//   - 400 if the request body is present but malformed.
//   - 404 if the repo_id is not registered.
//   - 500 on unexpected errors.
//
// The SHA field is optional. If absent the pipeline resolves HEAD after cloning.
func TriggerIndex(svc *service.IndexService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := chi.URLParam(r, "repoID")

		// Body is optional — a POST with no body or an empty object is valid.
		var req triggerIndexRequest
		if r.ContentLength != 0 {
			if !decodeBody(w, r, &req) {
				return
			}
		}

		sha := strings.TrimSpace(req.SHA)
		if sha != "" {
			normalized := normalizedNonZeroSHA(sha)
			if normalized == "" {
				writeJSON(w, http.StatusBadRequest, errResponse("sha must be a non-zero 40-character lowercase hex commit SHA"))
				return
			}
			sha = normalized
		}

		runID, err := svc.TriggerIndex(r.Context(), repoID, sha)
		if err != nil {
			if errors.Is(err, store.ErrRepoNotFound) {
				writeJSON(w, http.StatusNotFound, errResponse("repo not found"))
				return
			}
			if errors.Is(err, store.ErrIndexRunInProgress) {
				writeJSON(w, http.StatusConflict, errResponse("index run already in progress"))
				return
			}
			writeJSON(w, http.StatusInternalServerError, errResponse("failed to trigger index"))
			return
		}

		writeJSON(w, http.StatusAccepted, map[string]string{"run_id": runID})
	}
}
