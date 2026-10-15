package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/snehal/observability-incident-lab/internal/store"
)

type HealthHandler struct {
	db    store.Pinger
	cache store.Pinger
}

func NewHealthHandler(db, cache store.Pinger) *HealthHandler {
	return &HealthHandler{db: db, cache: cache}
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	failed := make([]string, 0, 2)

	dbCtx, dbCancel := context.WithTimeout(r.Context(), 2*time.Second)
	if err := h.db.Ping(dbCtx); err != nil {
		failed = append(failed, "database")
	}
	dbCancel()

	cacheCtx, cacheCancel := context.WithTimeout(r.Context(), 2*time.Second)
	if err := h.cache.Ping(cacheCtx); err != nil {
		failed = append(failed, "redis")
	}
	cacheCancel()

	if len(failed) > 0 {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status": "not_ready",
			"failed": failed,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
