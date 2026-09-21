package test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	router "github.com/InatoInato/car_service.git/internal"
)

func TestHealth(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := router.New(logger, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}
