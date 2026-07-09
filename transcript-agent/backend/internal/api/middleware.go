package api

import (
	"context"
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

// authExempt reports whether a path skips authentication: health checks and
// the export download endpoint (frozen contract: no auth on download).
func authExempt(r *http.Request) bool {
	p := r.URL.Path
	if p == "/healthz" || p == "/api/v1/healthz" {
		return true
	}
	if (r.Method == http.MethodGet || r.Method == http.MethodHead) &&
		strings.HasPrefix(p, "/api/v1/exports/") && strings.HasSuffix(p, "/download") {
		return true
	}
	return false
}

// middleware wraps the mux with CORS, request logging, and header auth.
func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS for the local React dev server (frozen contract).
		origin := r.Header.Get("Origin")
		if origin != "" && origin == s.CORSOrigin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-User-Id, X-User-Role")
			w.Header().Set("Access-Control-Max-Age", "600")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		if !authExempt(r) {
			userID := strings.TrimSpace(r.Header.Get("X-User-Id"))
			role := strings.TrimSpace(r.Header.Get("X-User-Role"))
			if s.AuthProxySecret != "" && r.Header.Get("X-Auth-Proxy-Secret") != s.AuthProxySecret {
				writeError(rec, domain.E(domain.CodeUnauthenticated,
					"authentication proxy secret missing or invalid"))
				s.logRequest(r, rec, start, "")
				return
			}
			if userID == "" || !validRole(role) {
				writeError(rec, domain.E(domain.CodeUnauthenticated,
					"authentication required: send X-User-Id and X-User-Role (producer|reviewer|admin)"))
				s.logRequest(r, rec, start, "")
				return
			}
			r = r.WithContext(context.WithValue(r.Context(), identityKey, Identity{UserID: userID, Role: role}))
			next.ServeHTTP(rec, r)
			s.logRequest(r, rec, start, userID)
			return
		}
		next.ServeHTTP(rec, r)
		s.logRequest(r, rec, start, "")
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
	)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
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
