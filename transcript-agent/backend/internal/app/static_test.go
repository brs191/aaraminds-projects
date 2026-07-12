package app_test

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aaraminds/transcript-agent/internal/app"
)

// TestStaticSPAFallback covers WEB_DIST serving: real assets come back
// verbatim with correct content types, unknown non-/api/ GET paths fall back
// to index.html (client-side routes like /jobs/{id}), and /api/v1 keeps
// priority (routes still authenticated, unknown API paths 404 as JSON-side
// not-found rather than the SPA shell).
func TestStaticSPAFallback(t *testing.T) {
	dist := t.TempDir()
	indexHTML := `<!doctype html><html><body><div id="root">transcript-agent-shell</div></body></html>`
	assetJS := `console.log("transcript-agent asset");`
	if err := os.WriteFile(filepath.Join(dist, "index.html"), []byte(indexHTML), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dist, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dist, "assets", "app.js"), []byte(assetJS), 0o644); err != nil {
		t.Fatal(err)
	}

	e := newEnvWith(t, nil, func(o *app.Options) { o.WebDist = dist })

	fetch := func(path string) (int, string, string) {
		t.Helper()
		res, err := http.Get(e.srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		return res.StatusCode, string(body), res.Header.Get("Content-Type")
	}

	// 1. Root serves the shell.
	status, body, ctype := fetch("/")
	if status != http.StatusOK || body != indexHTML {
		t.Fatalf("GET /: status %d body %q", status, body)
	}
	if !strings.HasPrefix(ctype, "text/html") {
		t.Fatalf("GET /: content type %q, want text/html", ctype)
	}

	// 2. Unknown client-side route falls back to the shell (no auth needed).
	status, body, ctype = fetch("/jobs/abc")
	if status != http.StatusOK || body != indexHTML {
		t.Fatalf("GET /jobs/abc: status %d body %q, want SPA fallback", status, body)
	}
	if !strings.HasPrefix(ctype, "text/html") {
		t.Fatalf("GET /jobs/abc: content type %q, want text/html", ctype)
	}

	// 3. Real asset files are served verbatim with a JS content type.
	status, body, ctype = fetch("/assets/app.js")
	if status != http.StatusOK || body != assetJS {
		t.Fatalf("GET /assets/app.js: status %d body %q", status, body)
	}
	if !strings.Contains(ctype, "javascript") {
		t.Fatalf("GET /assets/app.js: content type %q, want javascript", ctype)
	}

	// 4. No directory listing: a directory path gets the SPA shell, never
	// an index of files.
	status, body, _ = fetch("/assets/")
	if status != http.StatusOK || body != indexHTML {
		t.Fatalf("GET /assets/: status %d body %q, want SPA shell (no listing)", status, body)
	}

	// 5. /api/v1 keeps priority: healthz still answers JSON, authenticated
	// routes still enforce auth, and unknown API paths are 404 — never the
	// SPA shell.
	status, body, _ = fetch("/api/v1/healthz")
	if status != http.StatusOK || !strings.Contains(body, `"status":"ok"`) {
		t.Fatalf("GET /api/v1/healthz: status %d body %q", status, body)
	}
	st := e.do(http.MethodGet, "/api/v1/jobs", nil, nil, nil)
	if st != http.StatusUnauthorized {
		t.Fatalf("GET /api/v1/jobs without identity: status %d, want 401", st)
	}
	st = e.do(http.MethodGet, "/api/v1/jobs", producer, nil, nil)
	if st != http.StatusOK {
		t.Fatalf("GET /api/v1/jobs as producer: status %d, want 200", st)
	}
	// Unknown API paths never get the SPA shell: 401 unauthenticated (auth
	// runs before routing, as before), 404 authenticated.
	req, err := http.NewRequest(http.MethodGet, e.srv.URL+"/api/v1/does-not-exist", nil)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range producer {
		req.Header.Set(k, v)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound || strings.Contains(string(raw), "transcript-agent-shell") {
		t.Fatalf("GET /api/v1/does-not-exist: status %d body %q, want 404 without SPA shell", res.StatusCode, raw)
	}

	// 6. Path traversal cannot escape the dist directory.
	status, body, _ = fetch("/../go.mod")
	if strings.Contains(body, "module ") {
		t.Fatalf("GET /../go.mod leaked file contents (status %d)", status)
	}
}

// TestStaticDisabled pins the pre-WEB_DIST behavior: without a static dir the
// root path stays a plain 404.
func TestStaticDisabled(t *testing.T) {
	e := newEnv(t, nil)
	res, err := http.Get(e.srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("GET / without WEB_DIST: status %d, want 404", res.StatusCode)
	}
}
