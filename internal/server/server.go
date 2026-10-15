package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/riandyrn/otelchi"

	"github.com/snehal/observability-incident-lab/internal/config"
	"github.com/snehal/observability-incident-lab/internal/handler"
	"github.com/snehal/observability-incident-lab/internal/middleware"
	"github.com/snehal/observability-incident-lab/internal/observability"
	"github.com/snehal/observability-incident-lab/internal/store"
)

func New(cfg config.Config, orders store.OrderStore, db store.Pinger, cache store.Cache, logger *slog.Logger) *chi.Mux {
	if logger == nil {
		logger = slog.Default()
	}

	r := chi.NewRouter()
	r.Use(otelchi.Middleware(cfg.OTelServiceName))
	r.Use(middleware.Logging(logger))

	// Expose Prometheus metrics without counting metrics scrapes as API traffic.
	r.Handle("/metrics", observability.MetricsHandler())

	// Record application HTTP requests, excluding /metrics.
	r.Group(func(api chi.Router) {
		api.Use(observability.MetricsMiddleware)

		orderHandler := handler.NewOrdersHandler(
			orders,
			cache,
			time.Duration(cfg.CacheTTLSeconds)*time.Second,
			logger,
		)
		ordersByCustomerHandler := handler.NewOrdersByCustomerHandler(orders, logger)
		healthHandler := handler.NewHealthHandler(db, cache)

		api.Get("/health", healthHandler.Health)
		api.Get("/ready", healthHandler.Ready)
		api.Post("/orders", orderHandler.Create)
		api.Get("/orders/{id}", orderHandler.GetByID)
		api.Get("/customers/{id}/orders", ordersByCustomerHandler.List)
	})

	return r
}

func HTTPServer(cfg config.Config, router http.Handler) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
