package handler_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type handlerTransport struct{ handler http.Handler }

func (t handlerTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	recorder := httptest.NewRecorder()
	t.handler.ServeHTTP(recorder, request)
	return recorder.Result(), nil
}

func httpClient(handler http.Handler) *http.Client {
	return &http.Client{Transport: handlerTransport{handler: handler}}
}

// These checks use net/http request and response types through an in-memory
// transport. The broader api_test.go suite covers handler mapping in detail.
func TestHTTPBoundaryContracts(t *testing.T) {
	t.Parallel()
	store := &apiStore{}
	client := httpClient(api(t, store))

	t.Run("health is JSON", func(t *testing.T) {
		res, err := client.Get("http://car-service.test/health")
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK || res.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("unexpected health response: status=%d content-type=%q", res.StatusCode, res.Header.Get("Content-Type"))
		}
		var body map[string]string
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil || body["status"] != "ok" {
			t.Fatalf("unexpected health body: body=%v error=%v", body, err)
		}
	})

	tests := []struct {
		name, method, path, body string
		status                   int
	}{
		{"oversized create", http.MethodPost, "/cars", `{"description":"` + strings.Repeat("x", 129<<10) + `"}`, http.StatusRequestEntityTooLarge},
		{"oversized update", http.MethodPut, "/cars/00000000-0000-4000-8000-000000000001", `{"description":"` + strings.Repeat("x", 129<<10) + `"}`, http.StatusRequestEntityTooLarge},
		{"multiple JSON values", http.MethodPost, "/cars", validCarJSON + validCarJSON, http.StatusBadRequest},
		{"JSON null", http.MethodPost, "/cars", "null", http.StatusBadRequest},
		{"unsupported method", http.MethodPatch, "/cars/00000000-0000-4000-8000-000000000001", validCarJSON, http.StatusMethodNotAllowed},
		{"unknown route", http.MethodGet, "/not-an-endpoint", "", http.StatusNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, "http://car-service.test"+tc.path, strings.NewReader(tc.body))
			if err != nil {
				t.Fatal(err)
			}
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			res, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			data, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
			if err != nil {
				t.Fatal(err)
			}
			if res.StatusCode != tc.status {
				t.Fatalf("want %d, got %d: %.200s", tc.status, res.StatusCode, data)
			}
		})
	}
	if store.calls != 0 {
		t.Fatalf("boundary failures reached persistence %d times", store.calls)
	}
}

func TestHTTPGenerationFailureIsNonBlockingJSON(t *testing.T) {
	t.Parallel()
	client := httpClient(api(t, &apiStore{}))
	res, err := client.Get("http://car-service.test/cars/generations?brand=BMW&model=320i&year=2020")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d", res.StatusCode)
	}
	if res.Header.Get("Content-Type") != "application/json" || res.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("unexpected headers: content-type=%q cache-control=%q", res.Header.Get("Content-Type"), res.Header.Get("Cache-Control"))
	}
	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil || body["error"] == "" {
		t.Fatalf("unexpected error response: body=%v error=%v", body, err)
	}
}
