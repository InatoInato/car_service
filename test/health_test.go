package test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	router "github.com/InatoInato/car_service.git/internal"
)

type healthPinger struct{ err error }

func (p healthPinger) Ping(context.Context) error { return p.err }

func TestHealth(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	for _, tc := range []struct {
		name   string
		pinger healthPinger
		status int
	}{
		{"database available", healthPinger{}, http.StatusOK},
		{"database unavailable", healthPinger{err: errors.New("database unavailable")}, http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler := router.New(logger, nil, nil, tc.pinger)
			request := httptest.NewRequest(http.MethodGet, "/health", nil)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != tc.status {
				t.Fatalf("expected %d, got %d", tc.status, recorder.Code)
			}
		})
	}
}
