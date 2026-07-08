package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/aaraminds/rif/ingestion/cloneurl"
	"github.com/aaraminds/rif/ingestion/store"
)

// registerRepoRequest is the JSON body for POST /repos.
type registerRepoRequest struct {
	RepoID   string `json:"repo_id"`
	CloneURL string `json:"clone_url"`
}

// RegisterRepo returns a handler for POST /repos.
//
// Inserts the repo into rif_meta.repositories and returns:
//   - 201 {"repo_id":"..."} on success.
//   - 400 if repo_id or clone_url is missing.
//   - 409 if the repo_id is already registered.
//   - 500 on unexpected errors.
func RegisterRepo(rs *store.RunStore, allowedCloneHosts []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req registerRepoRequest
		if !decodeBody(w, r, &req) {
			return
		}
		req.RepoID = strings.TrimSpace(req.RepoID)
		if req.RepoID == "" || strings.TrimSpace(req.CloneURL) == "" {
			writeJSON(w, http.StatusBadRequest, errResponse("repo_id and clone_url are required"))
			return
		}
		cloneURL, err := cloneurl.Validate(req.CloneURL, allowedCloneHosts)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errResponse(err.Error()))
			return
		}

		if err := rs.RegisterRepo(r.Context(), req.RepoID, cloneURL); err != nil {
			if errors.Is(err, store.ErrRepoDuplicate) {
				writeJSON(w, http.StatusConflict, errResponse("repo already registered"))
				return
			}
			writeJSON(w, http.StatusInternalServerError, errResponse("failed to register repo"))
			return
		}

		writeJSON(w, http.StatusCreated, map[string]string{"repo_id": req.RepoID})
	}
}
