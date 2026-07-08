package handler

import (
	"net/http"

	"github.com/aaraminds/rif/graphstore"
)

// Health returns a handler for GET /healthz.
//
// Calls GraphStore.Ping and returns:
//   - 200 {"status":"ok"} when the graph store is reachable.
//   - 503 {"status":"degraded","detail":"..."} when Ping fails.
func Health(gs graphstore.GraphStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := gs.Ping(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status": "degraded",
				"detail": err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
