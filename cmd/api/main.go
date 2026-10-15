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

	"github.com/snehal/observability-incident-lab/internal/config"
	"github.com/snehal/observability-incident-lab/internal/server"
	"github.com/snehal/observability-incident-lab/internal/store"
	"github.com/snehal/observability-incident-lab/migrations"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration failed", "error", err)
		os.Exit(1)
	}

	startupCtx, startupCancel := context.WithTimeout(context.Background(), 5*time.Second)
	pool, err := store.NewPool(startupCtx, cfg.DatabaseURL, cfg.DBMaxConns)
	startupCancel()
	if err != nil {
		logger.Error("database startup failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	migrationCtx, migrationCancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := migrations.Apply(migrationCtx, pool); err != nil {
		migrationCancel()
		logger.Error("database migrations failed", "error", err)
		os.Exit(1)
	}
	migrationCancel()
	logger.Info("database migrations applied")

	redisCache := store.NewRedisCache(cfg.RedisAddr)
	defer func() {
		if err := redisCache.Close(); err != nil {
			logger.Error("redis close failed", "error", err)
		}
	}()

	pgStore := store.NewPostgresStore(pool)
	router := server.New(cfg, pgStore, pgStore, redisCache, logger)
	httpServer := server.HTTPServer(cfg, router)

	signalCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("HTTP server starting", "addr", httpServer.Addr)
		serverErr <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server failed", "error", err)
			os.Exit(1)
		}
	case <-signalCtx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		if closeErr := httpServer.Close(); closeErr != nil {
			logger.Error("forced server close failed", "error", closeErr)
		}
	}

	logger.Info("server stopped")
}
