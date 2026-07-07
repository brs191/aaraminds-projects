// Command ingestion-service is the Phase 1 Ingestion Service for the
// Repo Intelligence Factory.
//
// It owns four concerns:
//   - Repo registration (POST /repos)
//   - On-demand indexing (POST /repos/{repoID}/index)
//   - Run status polling (GET /repos/{repoID}/status)
//   - GitHub webhook enqueue endpoint (POST /webhook/github)
//   - Health probes (GET /healthz, GET /health)
//
// All configuration is read from environment variables via [config.Load].
// The service connects to Postgres (with Apache AGE), wraps the graph store
// in a LoggingStore, and starts a Chi HTTP router with graceful shutdown on
// SIGTERM/SIGINT.
package main

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"github.com/att/rif/graphstore"
	"github.com/att/rif/ingestion/config"
	"github.com/att/rif/ingestion/handler"
	"github.com/att/rif/ingestion/service"
	"github.com/att/rif/ingestion/store"
	"github.com/att/rif/phase5/ingestion/queue"
	"github.com/att/rif/phase5/ingestion/reconcile"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// ── Config ────────────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	// ── Logger ─────────────────────────────────────────────────────────────────
	var logLevel slog.Level
	if err := logLevel.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		logLevel = slog.LevelInfo
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	// ── Postgres connection pool with AGE bootstrap ───────────────────────────
	// Every new connection loads the AGE shared library and sets search_path so
	// that Cypher queries, rif_meta tables, and pg_catalog builtins are all
	// accessible without schema-qualifying.
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("parse database url: %w", err)
	}
	poolCfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		if _, err := conn.Exec(ctx, "LOAD 'age'"); err != nil {
			return fmt.Errorf("load age extension: %w", err)
		}
		if _, err := conn.Exec(ctx, "SET search_path = ag_catalog, rif_meta, public"); err != nil {
			return fmt.Errorf("set search_path: %w", err)
		}
		return nil
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return fmt.Errorf("pgxpool: %w", err)
	}
	defer pool.Close()

	// ── GraphStore ────────────────────────────────────────────────────────────
	ageStore, err := graphstore.NewAGEStore(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("age store: %w", err)
	}
	// Wrap with structured logging + BlastRadius audit trail.
	gs := graphstore.NewLoggingStore(ageStore, logger).WithAuditPath(cfg.AuditLogPath)

	// ── Application layer ─────────────────────────────────────────────────────
	runStore := store.NewRunStore(pool)
	idxSvc := service.NewIndexService(runStore, gs, cfg, logger)
	queueStore := queue.NewStore(pool)
	incrementalSvc := service.NewIncrementalService(idxSvc, pool, queueStore)
	queueDispatcher := service.NewQueueDispatcher(incrementalSvc)
	queueWorker := queue.NewWorker(queueStore, queueDispatcher)
	reconciler := reconcile.NewReconciler(pool, queueStore)

	if cfg.IncrementalEnabled {
		if err := queueStore.EnsureSchema(context.Background()); err != nil {
			return fmt.Errorf("ensure phase5 queue schema: %w", err)
		}
	}
	if cfg.GitHubWebhookSecret == "" {
		logger.Warn("GITHUB_WEBHOOK_SECRET is empty; webhook signature verification is disabled")
	}
	if cfg.APIToken == "" {
		logger.Warn("RIF_API_TOKEN is empty; state-changing API endpoints are unauthenticated")
	}

	// ── Router ────────────────────────────────────────────────────────────────
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(requestLogger(logger))

	r.Get("/healthz", handler.Health(gs))
	r.Get("/health", handler.Health(gs))
	r.Group(func(r chi.Router) {
		r.Use(bearerTokenMiddleware(cfg.APIToken))
		r.Post("/repos", handler.RegisterRepo(runStore, cfg.AllowedCloneHosts))
		r.Post("/repos/{repoID}/index", handler.TriggerIndex(idxSvc))
	})
	r.Get("/repos/{repoID}/status", handler.GetStatus(runStore))
	r.Post("/webhook/github", handler.GithubWebhook(queueStore, cfg.GitHubWebhookSecret))

	// ── HTTP server ───────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// ── Graceful shutdown with errgroup ───────────────────────────────────────
	g, gctx := errgroup.WithContext(context.Background())

	g.Go(func() error {
		logger.Info("ingestion service starting",
			slog.String("addr", srv.Addr),
			slog.String("log_level", cfg.LogLevel),
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("listen: %w", err)
		}
		return nil
	})

	if cfg.IncrementalEnabled {
		g.Go(func() error {
			logger.Info("phase5 queue worker starting")
			if err := queueWorker.Run(gctx); err != nil && !errors.Is(err, context.Canceled) {
				return fmt.Errorf("phase5 queue worker: %w", err)
			}
			return nil
		})
		g.Go(func() error {
			logger.Info("phase5 reconciler starting")
			if err := reconciler.Run(gctx); err != nil && !errors.Is(err, context.Canceled) {
				return fmt.Errorf("phase5 reconciler: %w", err)
			}
			return nil
		})
	}

	g.Go(func() error {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
		select {
		case sig := <-sigCh:
			logger.Info("shutdown signal received", slog.String("signal", sig.String()))
		case <-gctx.Done():
			// Other goroutine exited (e.g. listen error); propagate.
			return gctx.Err()
		}

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("graceful shutdown: %w", err)
		}
		logger.Info("server shutdown complete")
		_ = gs.Close()
		return nil
	})

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

func bearerTokenMiddleware(expectedToken string) func(http.Handler) http.Handler {
	expectedToken = strings.TrimSpace(expectedToken)
	return func(next http.Handler) http.Handler {
		if expectedToken == "" {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			const prefix = "Bearer "
			auth := strings.TrimSpace(r.Header.Get("Authorization"))
			if !strings.HasPrefix(auth, prefix) {
				http.Error(w, "missing bearer token", http.StatusUnauthorized)
				return
			}
			got := strings.TrimSpace(strings.TrimPrefix(auth, prefix))
			if subtle.ConstantTimeCompare([]byte(got), []byte(expectedToken)) != 1 {
				http.Error(w, "invalid bearer token", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// requestLogger returns a Chi middleware that emits a structured slog record
// for every completed request, including method, path, status code, and duration.
func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			logger.InfoContext(r.Context(), "http request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.Status()),
				slog.Int64("duration_ms", time.Since(start).Milliseconds()),
				slog.String("request_id", middleware.GetReqID(r.Context())),
			)
		})
	}
}
