package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/aaraminds/transcript-agent/internal/domain"
)

// Identity is the authenticated caller (PRD 16.1: all users authenticate;
// MVP transport is trusted headers behind enterprise SSO/reverse proxy).
type Identity struct {
	UserID string
	Role   string // producer | reviewer | admin (PRD 16.2 MVP-minimum roles)
}

type ctxKey int

const identityKey ctxKey = iota

// identityFrom returns the caller identity stored by the auth middleware.
func identityFrom(ctx context.Context) Identity {
	id, _ := ctx.Value(identityKey).(Identity)
	return id
}

func validRole(role string) bool {
	switch role {
	case domain.RoleProducer, domain.RoleReviewer, domain.RoleAdmin:
		return true
	}
	return false
}

// authExempt reports whether a path skips authentication entirely: health
// checks, the internal expvar metrics endpoint (PRD 18.2; deploy behind the
// trusted proxy, not on the public edge), and everything outside /api/ —
// static UI assets and SPA routes are public; the API contract under /api/
// stays authenticated.
func authExempt(r *http.Request) bool {
	p := r.URL.Path
	if !strings.HasPrefix(p, "/api/") {
		return true // /healthz, /debug/vars, static UI + SPA fallback
	}
	return p == "/api/v1/healthz"
}

// tokenCapable reports whether the endpoint accepts EITHER a signed ?token=
// OR auth headers (export download, job audio). The middleware attaches the
// header identity when present but never rejects here — the handler enforces
// token-or-auth (audit H2: downloads are no longer fully open).
func tokenCapable(r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	p := r.URL.Path
	if strings.HasPrefix(p, "/api/v1/exports/") && strings.HasSuffix(p, "/download") {
		return true
	}
	if strings.HasPrefix(p, "/api/v1/jobs/") && strings.HasSuffix(p, "/audio") {
		return true
	}
	return false
}

// maxJSONBodyBytes caps every request body except the multipart upload
// endpoint (audit H4).
const maxJSONBodyBytes = 1 << 20 // 1 MiB

func isUploadEndpoint(r *http.Request) bool {
	return r.Method == http.MethodPost && r.URL.Path == "/api/v1/uploads"
}

// identityFromHeaders extracts a valid header identity, or ok=false. When an
// auth proxy secret is configured it must match too.
func (s *Server) identityFromHeaders(r *http.Request) (Identity, bool) {
	if s.AuthProxySecret != "" && r.Header.Get("X-Auth-Proxy-Secret") != s.AuthProxySecret {
		return Identity{}, false
	}
	userID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	role := strings.TrimSpace(r.Header.Get("X-User-Role"))
	if userID == "" || !validRole(role) {
		return Identity{}, false
	}
	return Identity{UserID: userID, Role: role}, true
}

// middleware wraps the mux with CORS, body limits, request logging, and
// header auth.
func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS for the local React dev server (frozen contract).
		origin := r.Header.Get("Origin")
		if origin != "" && origin == s.CORSOrigin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-User-Id, X-User-Role")
			w.Header().Set("Access-Control-Expose-Headers", "X-Superseded, X-Request-Id")
			w.Header().Set("Access-Control-Max-Age", "600")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Body limit on every JSON route (the upload handler applies its own
		// MAX_UPLOAD_BYTES limit).
		if r.Body != nil && !isUploadEndpoint(r) {
			r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
		}

		start := time.Now()
		reqID := newRequestID()
		w.Header().Set("X-Request-Id", reqID)
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK, reqID: reqID, log: s.Log}

		switch {
		case authExempt(r):
			next.ServeHTTP(rec, r)
			s.logRequest(r, rec, start, "")
		case tokenCapable(r):
			// Attach identity when header auth is present; the handler decides
			// between token and identity.
			userID := ""
			if ident, ok := s.identityFromHeaders(r); ok {
				r = r.WithContext(context.WithValue(r.Context(), identityKey, ident))
				userID = ident.UserID
			}
			next.ServeHTTP(rec, r)
			s.logRequest(r, rec, start, userID)
		default:
			ident, ok := s.identityFromHeaders(r)
			if !ok {
				if s.AuthProxySecret != "" && r.Header.Get("X-Auth-Proxy-Secret") != s.AuthProxySecret {
					writeError(rec, domain.E(domain.CodeUnauthenticated,
						"authentication proxy secret missing or invalid"))
				} else {
					writeError(rec, domain.E(domain.CodeUnauthenticated,
						"authentication required: send X-User-Id and X-User-Role (producer|reviewer|admin)"))
				}
				s.logRequest(r, rec, start, "")
				return
			}
			r = r.WithContext(context.WithValue(r.Context(), identityKey, ident))
			next.ServeHTTP(rec, r)
			s.logRequest(r, rec, start, ident.UserID)
		}
	})
}

func (s *Server) logRequest(r *http.Request, rec *statusRecorder, start time.Time, userID string) {
	log := s.Log
	if log == nil {
		log = slog.Default()
	}
	log.Info("http",
		"method", r.Method,
		"path", r.URL.Path,
		"status", rec.status,
		"duration_ms", time.Since(start).Milliseconds(),
		"user", userID,
		"request_id", rec.reqID,
	)
}

// newRequestID returns a short random request identifier for log correlation.
func newRequestID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b[:])
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	reqID  string
	log    *slog.Logger
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// logInternalError records the full internal error server-side (writeError
// sends the client only a generic INTERNAL_ERROR — audit M-sanitize-500s).
func (r *statusRecorder) logInternalError(err error) {
	log := r.log
	if log == nil {
		log = slog.Default()
	}
	log.Error("internal error", "request_id", r.reqID, "error", err)
}

// requireRole enforces endpoint-level RBAC (PRD 16.2). Returns an error when
// the caller's role is not in the allowed set.
func requireRole(id Identity, roles ...string) error {
	for _, role := range roles {
		if id.Role == role {
			return nil
		}
	}
	return domain.E(domain.CodeUserNotAuthorized,
		"role %q is not permitted for this action (requires %s)", id.Role, strings.Join(roles, " or "))
}
