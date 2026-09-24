package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"

	router "github.com/InatoInato/car_service.git/internal"
	"github.com/InatoInato/car_service.git/internal/generation"
	"github.com/InatoInato/car_service.git/internal/provider"
	"github.com/InatoInato/car_service.git/internal/service"
)

type unavailableGenerations struct{}

func (unavailableGenerations) Generations(context.Context, string, string) ([]generation.Candidate, error) {
	return nil, errors.New("private provider diagnostic")
}

func TestGenerationUnavailableIsNotEmptySuccess(t *testing.T) {
	for _, s := range []*service.GenerationService{nil, service.NewGenerationService(unavailableGenerations{})} {
		r := router.New(slog.New(slog.NewTextHandler(io.Discard, nil)), nil, s, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/cars/generations?brand=M&model=E&year=1995", nil))
		assertAPIError(t, w, 503)
		var body map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body["error"] == "private provider diagnostic" {
			t.Fatal("leaked provider failure")
		}
	}
}

func TestGenerationHTTPContract(t *testing.T) {
	catalogue, err := provider.NewGenerationCatalog()
	if err != nil {
		t.Fatal(err)
	}
	r := router.New(slog.New(slog.NewTextHandler(io.Discard, nil)), nil, service.NewGenerationService(catalogue), nil)
	for _, tc := range []struct {
		query         string
		status, count int
	}{
		{"brand=Mercedes-Benz&model=E280&year=1994", 200, 1},
		{"brand=Mercedes-Benz&model=E280&year=1993", 200, 1},
		{"brand=Mercedes-Benz&model=E280&year=1992", 200, 0},
		{"brand=mercedes+benz&model=E+280&year=1995", 200, 2},
		{"brand=Mercedes-Benz&model=E280&year=1999", 200, 1},
		{"brand=Mercedes-Benz&model=E280&year=2000", 200, 0},
		{"brand=Unknown&model=Unknown&year=1995", 200, 0},
		{"brand=Mercedes-Benz&model=E280+CDI&year=1995", 200, 0},
		{"", 400, 0}, {"brand=Mercedes-Benz&year=1995", 400, 0},
		{"brand=+&model=E280&year=1995", 400, 0},
		{"brand=M&model=E&year=32768", 400, 0},
		{"brand=M&model=E&year=x", 400, 0},
		{"brand=M&model=E&year=1995&year=1996", 400, 0},
	} {
		t.Run(tc.query, func(t *testing.T) {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest("GET", "/cars/generations?"+tc.query, nil))
			if w.Code != tc.status {
				t.Fatalf("HTTP %d: %s", w.Code, w.Body.String())
			}
			if tc.status != 200 {
				assertAPIError(t, w, tc.status)
				return
			}
			var result generation.Result
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Candidates == nil || len(result.Candidates) != tc.count || result.Notice == "" {
				t.Fatalf("unexpected result %+v", result)
			}
		})
	}
}
