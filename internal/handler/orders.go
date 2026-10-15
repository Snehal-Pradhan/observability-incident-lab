package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/snehal/observability-incident-lab/internal/model"
	"github.com/snehal/observability-incident-lab/internal/observability"
	"github.com/snehal/observability-incident-lab/internal/store"
)

type OrdersHandler struct {
	orders   store.OrderStore
	cache    store.Cache
	cacheTTL time.Duration
	logger   *slog.Logger
}

func NewOrdersHandler(
	orders store.OrderStore,
	cache store.Cache,
	cacheTTL time.Duration,
	logger *slog.Logger,
) *OrdersHandler {
	if logger == nil {
		logger = slog.Default()
	}

	return &OrdersHandler{
		orders:   orders,
		cache:    cache,
		cacheTTL: cacheTTL,
		logger:   logger,
	}
}

func (h *OrdersHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateOrderRequest
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "request must contain exactly one JSON value")
		return
	}

	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	order, err := h.orders.Insert(ctx, req)
	if errors.Is(err, store.ErrUnknownCustomer) {
		writeError(w, http.StatusBadRequest, "unknown customer")
		return
	}
	if err != nil {
		h.logger.Error("insert order failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create order")
		return
	}

	cacheCtx, cacheCancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cacheCancel()

	if err := h.cache.Delete(cacheCtx, fmt.Sprintf("order:%d", order.ID)); err != nil {
		observability.CacheOperationsTotal.WithLabelValues("delete", "error").Inc()
		h.logger.Warn("cache invalidation failed", "order_id", order.ID, "error", err)
	} else {
		observability.CacheOperationsTotal.WithLabelValues("delete", "success").Inc()
	}

	writeJSON(w, http.StatusCreated, order)
}

func (h *OrdersHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	key := fmt.Sprintf("order:%d", id)

	cacheCtx, cacheCancel := context.WithTimeout(r.Context(), 5*time.Second)
	cached, found, cacheErr := h.cache.Get(cacheCtx, key)
	cacheCancel()

	switch {
	case cacheErr != nil:
		observability.CacheOperationsTotal.WithLabelValues("get", "error").Inc()
		h.logger.Warn("cache read failed", "order_id", id, "error", cacheErr)
	case found:
		observability.CacheOperationsTotal.WithLabelValues("get", "hit").Inc()

		var order model.Order
		if err := json.Unmarshal(cached, &order); err == nil {
			writeJSON(w, http.StatusOK, order)
			return
		}

		h.logger.Warn("invalid cached order JSON", "order_id", id)
	default:
		observability.CacheOperationsTotal.WithLabelValues("get", "miss").Inc()
	}

	dbCtx, dbCancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer dbCancel()

	order, found, err := h.orders.GetByID(dbCtx, id)
	if err != nil {
		h.logger.Error("get order failed", "order_id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to retrieve order")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}

	value, err := json.Marshal(order)
	if err != nil {
		h.logger.Error("marshal order failed", "order_id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to retrieve order")
		return
	}

	setCtx, setCancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer setCancel()

	if err := h.cache.Set(setCtx, key, value, h.cacheTTL); err != nil {
		observability.CacheOperationsTotal.WithLabelValues("set", "error").Inc()
		h.logger.Warn("cache write failed", "order_id", id, "error", err)
	} else {
		observability.CacheOperationsTotal.WithLabelValues("set", "success").Inc()
	}

	writeJSON(w, http.StatusOK, order)
}
