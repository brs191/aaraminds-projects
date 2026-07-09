// Command server runs the Podcast Transcript Agent backend (PRD v1.5 P0).
// Configuration is environment-only; see README.md for the full variable
// list. Defaults run fully in-memory with mock providers so the React UI can
// exercise the complete workflow without external services.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aaraminds/transcript-agent/internal/app"
	"github.com/aaraminds/transcript-agent/internal/domain"
	"github.com/aaraminds/transcript-agent/internal/objectstore"
	"github.com/aaraminds/transcript-agent/internal/providers/captions"
	capmock "github.com/aaraminds/transcript-agent/internal/providers/captions/mock"
	"github.com/aaraminds/transcript-agent/internal/providers/captions/youtube"
	"github.com/aaraminds/transcript-agent/internal/providers/llm"
	"github.com/aaraminds/transcript-agent/internal/providers/llm/anthropic"
	llmmock "github.com/aaraminds/transcript-agent/internal/providers/llm/mock"
	"github.com/aaraminds/transcript-agent/internal/providers/media"
	"github.com/aaraminds/transcript-agent/internal/providers/stt"
	"github.com/aaraminds/transcript-agent/internal/providers/stt/azure"
	sttmock "github.com/aaraminds/transcript-agent/internal/providers/stt/mock"
	"github.com/aaraminds/transcript-agent/internal/store"
	"github.com/aaraminds/transcript-agent/internal/store/memory"
	"github.com/aaraminds/transcript-agent/internal/store/postgres"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// --- storage -----------------------------------------------------------
	var stores store.Stores
	switch storage := env("STORAGE", "memory"); storage {
	case "memory":
		stores = memory.New().Stores()
		log.Info("storage: in-memory (data is lost on restart)")
	case "postgres":
		dsn := os.Getenv("DATABASE_URL")
		if dsn == "" {
			log.Error("STORAGE=postgres requires DATABASE_URL")
			os.Exit(1)
		}
		pool, err := pgxpool.New(ctx, dsn)
		if err != nil {
			log.Error("connect postgres", "error", err)
			os.Exit(1)
		}
		defer pool.Close()
		if err := postgres.Migrate(ctx, pool, env("MIGRATIONS_DIR", "migrations")); err != nil {
			log.Error("apply migrations", "error", err)
			os.Exit(1)
		}
		stores = postgres.New(pool).Stores()
		log.Info("storage: postgres", "migrations_dir", env("MIGRATIONS_DIR", "migrations"))
	default:
		log.Error("unknown STORAGE value", "storage", storage)
		os.Exit(1)
	}

	// --- object store --------------------------------------------------------
	dataDir := env("DATA_DIR", "./data")
	objects, err := objectstore.NewLocal(dataDir)
	if err != nil {
		log.Error("init object store", "error", err)
		os.Exit(1)
	}

	// --- providers -----------------------------------------------------------
	var sttProvider stt.Provider
	sttName := env("STT_PROVIDER", "mock")
	switch sttName {
	case "mock":
		sttProvider = sttmock.New()
	case "azure":
		sttProvider = azure.New(azure.Config{
			Region: os.Getenv("AZURE_SPEECH_REGION"),
			Key:    os.Getenv("AZURE_SPEECH_KEY"),
			Model:  os.Getenv("AZURE_SPEECH_MODEL"),
		})
	default:
		log.Error("unknown STT_PROVIDER", "value", sttName)
		os.Exit(1)
	}

	var llmProvider llm.Provider
	switch p := env("LLM_PROVIDER", "mock"); p {
	case "mock":
		llmProvider = llmmock.New()
	case "anthropic":
		llmProvider = anthropic.New(anthropic.Config{
			APIKey:       os.Getenv("ANTHROPIC_API_KEY"),
			CleanupModel: env("ANTHROPIC_CLEANUP_MODEL", "claude-haiku-4-5"),
			SummaryModel: env("ANTHROPIC_SUMMARY_MODEL", "claude-sonnet-4-5"),
		})
	default:
		log.Error("unknown LLM_PROVIDER", "value", p)
		os.Exit(1)
	}

	var captionProvider captions.Provider
	switch p := env("CAPTION_PROVIDER", "mock"); p {
	case "mock":
		captionProvider = capmock.New()
	case "youtube":
		captionProvider = youtube.New(youtube.Config{
			OAuthToken:   os.Getenv("YOUTUBE_OAUTH_TOKEN"),
			ChannelOwned: env("YOUTUBE_CHANNEL_OWNED", "false") == "true",
		})
	default:
		log.Error("unknown CAPTION_PROVIDER", "value", p)
		os.Exit(1)
	}

	var mediaProcessor media.Processor
	switch p := env("MEDIA_PROVIDER", "mock"); p {
	case "mock":
		mediaProcessor = media.NewStub()
	case "ffmpeg", "auto":
		mediaProcessor = media.Auto() // ffmpeg/ffprobe if on PATH, stub otherwise
	default:
		log.Error("unknown MEDIA_PROVIDER", "value", p)
		os.Exit(1)
	}

	// --- job_config defaults (admin-tunable; snapshotted per job) -------------
	defaults := domain.DefaultJobConfig(sttName)
	defaults.ConfidenceThreshold = envFloat("DEFAULT_CONFIDENCE_THRESHOLD", defaults.ConfidenceThreshold)
	defaults.SummaryMaxWords = envInt("DEFAULT_SUMMARY_MAX_WORDS", defaults.SummaryMaxWords)
	defaults.SummaryStyle = env("DEFAULT_SUMMARY_STYLE", defaults.SummaryStyle)
	defaults.StylePolicyID = env("DEFAULT_STYLE_POLICY_ID", defaults.StylePolicyID)

	// --- wiring ----------------------------------------------------------------
	a := app.New(app.Options{
		Log:                 log,
		Stores:              stores,
		Objects:             objects,
		STT:                 sttProvider,
		LLM:                 llmProvider,
		Media:               mediaProcessor,
		Captions:            captionProvider,
		STTName:             sttName,
		ConfigDefaults:      &defaults,
		CORSOrigin:          env("CORS_ORIGIN", "http://localhost:5173"),
		AuthProxySecret:     os.Getenv("AUTH_PROXY_SECRET"),
		DownloadTokenSecret: []byte(os.Getenv("DOWNLOAD_TOKEN_SECRET")),
		Sync:                false,
		Backoff:             envDuration("RETRY_BACKOFF", 2*time.Second),
	})
	if os.Getenv("AUTH_PROXY_SECRET") == "" {
		log.Warn("AUTH_PROXY_SECRET is not set; header auth is running in development mode and must not be exposed directly")
	}
	a.Orch.Start(ctx, envInt("WORKERS", 2), envDuration("REQUEUE_INTERVAL", 3*time.Second))

	// --- http server -------------------------------------------------------------
	addr := ":" + env("PORT", "8080")
	srv := &http.Server{
		Addr:              addr,
		Handler:           a.API.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		log.Info("listening", "addr", addr, "stt_provider", sttName, "data_dir", dataDir)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown", "error", err)
	}
	fmt.Println("bye")
}
