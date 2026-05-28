// grading-predictor API — cert + defect dataset for grade prediction.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/FG-CollectLabs/grading-predictor-backend/internal/config"
	"github.com/FG-CollectLabs/grading-predictor-backend/internal/db"
	"github.com/FG-CollectLabs/grading-predictor-backend/internal/httpx"
	"github.com/FG-CollectLabs/grading-predictor-backend/internal/predictor"
)

func main() {
	setupLogger(os.Getenv("LOG_LEVEL"))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("open db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	// GCS client wired when GCS_BUCKET env is set and ADC is available.
	// For v0.1 deploy without GCS, images are stored as path-only in DB.
	h := &predictor.Handler{
		DB:        pool,
		GCSClient: nil,
		GCSBucket: cfg.GCSBucket,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		pingCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(pingCtx); err != nil {
			httpx.WriteError(w, http.StatusServiceUnavailable, "db_down", err.Error())
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})

	auth := httpx.BearerAuth(cfg.AdminAPIToken)
	h.Routes(mux, auth)

	handler := httpx.Chain(
		httpx.RecoverMiddleware,
		httpx.LoggingMiddleware,
		httpx.CORSMiddleware(cfg.CORSOrigins),
	)(mux)

	srv := &http.Server{
		Addr:              cfg.APIAddr,
		Handler:           handler,
		ReadHeaderTimeout: config.HTTPTimeout,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		slog.Info("api listening", "addr", cfg.APIAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("listen", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown", "err", err)
	}
}

func setupLogger(level string) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})))
}
