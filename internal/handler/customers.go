package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/snehal/observability-incident-lab/internal/model"
	"github.com/snehal/observability-incident-lab/internal/store"
)

type OrdersByCustomerHandler struct {
	orders store.OrderStore
	logger *slog.Logger
}

func NewOrdersByCustomerHandler(orders store.OrderStore, logger *slog.Logger) *OrdersByCustomerHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &OrdersByCustomerHandler{orders: orders, logger: logger}
}

func (h *OrdersByCustomerHandler) List(w http.ResponseWriter, r *http.Request) {
	customerID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || customerID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	orders, err := h.orders.ListByCustomer(ctx, customerID)
	if err != nil {
		h.logger.Error("list orders by customer failed", "customer_id", customerID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list orders")
		return
	}

	if orders == nil {
		orders = make([]model.Order, 0)
	}

	writeJSON(w, http.StatusOK, orders)
}
