package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockPinger struct {
	err error
}

func (m mockPinger) Ping(context.Context) error {
	return m.err
}

func TestHealth(t *testing.T) {
	h := NewHealthHandler(mockPinger{}, mockPinger{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	h.Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestReady(t *testing.T) {
	tests := []struct {
		name       string
		dbErr      error
		redisErr   error
		wantStatus int
	}{
		{"ready", nil, nil, http.StatusOK},
		{"database down", errors.New("db unavailable"), nil, http.StatusServiceUnavailable},
		{"redis down", nil, errors.New("redis unavailable"), http.StatusServiceUnavailable},
		{"both down", errors.New("db unavailable"), errors.New("redis unavailable"), http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHealthHandler(mockPinger{err: tt.dbErr}, mockPinger{err: tt.redisErr})
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/ready", nil)

			h.Ready(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}
