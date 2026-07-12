package api

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// handleStatic serves the built React UI (WEB_DIST) with SPA fallback:
//   - real files under the dist directory are served with their correct
//     content type (http.ServeFile / mime by extension);
//   - every other non-/api/, non-/debug/ GET path gets index.html so
//     client-side routes like /jobs/{id} deep-link correctly;
//   - directories are never listed — a directory path falls back to
//     index.html like any other unknown path;
//   - /api/v1/* and /debug/* keep their JSON 404 behavior (more specific
//     mux patterns already won for the registered routes).
//
// When StaticDir is unset the route answers 404, preserving the pre-static
// behavior of the server.
func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	if s.StaticDir == "" || strings.HasPrefix(p, "/api/") || strings.HasPrefix(p, "/debug/") {
		http.NotFound(w, r)
		return
	}
	// Root-anchored clean prevents traversal outside the dist directory;
	// http.ServeFile additionally rejects any raw ".." in the URL.
	clean := path.Clean("/" + p)
	name := filepath.Join(s.StaticDir, filepath.FromSlash(clean))
	if fi, err := os.Stat(name); err == nil && fi.Mode().IsRegular() {
		http.ServeFile(w, r, name)
		return
	}
	// SPA fallback: serve the shell and let the client router resolve.
	http.ServeFile(w, r, filepath.Join(s.StaticDir, "index.html"))
}
