package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/snehal/observability-incident-lab/internal/model"
	"github.com/snehal/observability-incident-lab/internal/store"
)

type mockOrderStore struct {
	order     model.Order
	found     bool
	insertErr error
	getErr    error
}

func (m *mockOrderStore) Insert(_ context.Context, req model.CreateOrderRequest) (model.Order, error) {
	if m.insertErr != nil {
		return model.Order{}, m.insertErr
	}
	return model.Order{
		ID: 1, CustomerID: req.CustomerID, Status: req.Status,
		TotalCents: req.TotalCents, CreatedAt: time.Now(),
	}, nil
}

func (m *mockOrderStore) GetByID(_ context.Context, _ int64) (model.Order, bool, error) {
	return m.order, m.found, m.getErr
}

func (m *mockOrderStore) ListByCustomer(context.Context, int64) ([]model.Order, error) {
	return nil, nil
}

type mockCache struct {
	value []byte
	found bool
	err   error
}

func (m *mockCache) Get(context.Context, string) ([]byte, bool, error) {
	return m.value, m.found, m.err
}

func (m *mockCache) Set(context.Context, string, []byte, time.Duration) error {
	return m.err
}

func (m *mockCache) Delete(context.Context, string) error {
	return m.err
}

func (m *mockCache) Ping(context.Context) error {
	return m.err
}

func requestWithID(method, path, id string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", id)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}

func TestCreateOrder(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		insertErr  error
		wantStatus int
	}{
		{"created", `{"customer_id":1,"status":"pending","total_cents":500}`, nil, http.StatusCreated},
		{"bad json", `{`, nil, http.StatusBadRequest},
		{"trailing json", `{"customer_id":1,"status":"pending","total_cents":500} {}`, nil, http.StatusBadRequest},
		{"invalid status", `{"customer_id":1,"status":"invalid","total_cents":500}`, nil, http.StatusBadRequest},
		{"zero cents", `{"customer_id":1,"status":"pending","total_cents":0}`, nil, http.StatusBadRequest},
		{"unknown customer", `{"customer_id":99,"status":"pending","total_cents":500}`, store.ErrUnknownCustomer, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orders := &mockOrderStore{insertErr: tt.insertErr}
			cache := &mockCache{}
			h := NewOrdersHandler(orders, cache, time.Minute, nil)

			req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			h.Create(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestGetOrderNotFound(t *testing.T) {
	orders := &mockOrderStore{found: false}
	cache := &mockCache{}
	h := NewOrdersHandler(orders, cache, time.Minute, nil)

	req := requestWithID(http.MethodGet, "/orders/42", "42")
	rec := httptest.NewRecorder()
	h.GetByID(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestGetOrderCacheHit(t *testing.T) {
	cached := `{"id":42,"customer_id":1,"status":"paid","total_cents":500,"created_at":"2025-01-01T00:00:00Z"}`
	orders := &mockOrderStore{found: false}
	cache := &mockCache{value: []byte(cached), found: true}
	h := NewOrdersHandler(orders, cache, time.Minute, nil)

	req := requestWithID(http.MethodGet, "/orders/42", "42")
	rec := httptest.NewRecorder()
	h.GetByID(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestGetOrderDatabaseError(t *testing.T) {
	orders := &mockOrderStore{getErr: errors.New("database unavailable")}
	cache := &mockCache{}
	h := NewOrdersHandler(orders, cache, time.Minute, nil)

	req := requestWithID(http.MethodGet, "/orders/42", "42")
	rec := httptest.NewRecorder()
	h.GetByID(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
