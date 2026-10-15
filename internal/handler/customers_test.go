package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/snehal/observability-incident-lab/internal/model"
)

type mockListStore struct {
	orders  []model.Order
	listErr error
	gotID   int64
}

func (m *mockListStore) Insert(context.Context, model.CreateOrderRequest) (model.Order, error) {
	return model.Order{}, nil
}

func (m *mockListStore) GetByID(context.Context, int64) (model.Order, bool, error) {
	return model.Order{}, false, nil
}

func (m *mockListStore) ListByCustomer(_ context.Context, customerID int64) ([]model.Order, error) {
	m.gotID = customerID
	return m.orders, m.listErr
}

func TestListOrdersByCustomer(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name       string
		id         string
		orders     []model.Order
		listErr    error
		wantStatus int
	}{
		{
			"ok with orders", "7",
			[]model.Order{{ID: 1, CustomerID: 7, Status: "paid", TotalCents: 500, CreatedAt: now}},
			nil, http.StatusOK,
		},
		{"ok empty list", "7", nil, nil, http.StatusOK},
		{"bad id", "abc", nil, nil, http.StatusBadRequest},
		{"zero id", "0", nil, nil, http.StatusBadRequest},
		{"store error", "7", nil, errors.New("db down"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockListStore{orders: tt.orders, listErr: tt.listErr}
			h := NewOrdersByCustomerHandler(store, nil)

			req := requestWithID(http.MethodGet, "/customers/"+tt.id+"/orders", tt.id)
			rec := httptest.NewRecorder()
			h.List(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var got []model.Order
				if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
					t.Fatalf("invalid JSON: %v", err)
				}
				if got == nil {
					t.Fatal("empty list must serialize as [], not null")
				}
				if store.gotID != 7 {
					t.Fatalf("store called with customer_id = %d, want 7", store.gotID)
				}
			}
		})
	}
}
