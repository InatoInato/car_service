package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	router "github.com/InatoInato/car_service.git/internal"
	"github.com/InatoInato/car_service.git/internal/db"
	"github.com/InatoInato/car_service.git/internal/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Exercise the real router, handlers, and service with only persistence stubbed.
// Each subtest owns its store so parallel tests share no mutable state.
type apiStore struct {
	service.CarStore
	car      db.Car
	err      error
	calls    int
	ctx      context.Context
	created  db.CreateCarParams
	updated  db.UpdateCarParams
	filtered db.FilterCarsParams
	id       uuid.UUID
}

func (s *apiStore) record(ctx context.Context) { s.calls++; s.ctx = ctx }
func (s *apiStore) CreateCar(ctx context.Context, p db.CreateCarParams) (db.Car, error) {
	s.record(ctx)
	s.created = p
	return s.car, s.err
}
func (s *apiStore) GetCarByID(ctx context.Context, id uuid.UUID) (db.Car, error) {
	s.record(ctx)
	s.id = id
	return s.car, s.err
}
func (s *apiStore) UpdateCar(ctx context.Context, p db.UpdateCarParams) (db.Car, error) {
	s.record(ctx)
	s.updated = p
	return s.car, s.err
}
func (s *apiStore) DeleteCar(ctx context.Context, id uuid.UUID) error {
	s.record(ctx)
	s.id = id
	return s.err
}
func (s *apiStore) FilterCars(ctx context.Context, p db.FilterCarsParams) ([]db.Car, error) {
	s.record(ctx)
	s.filtered = p
	return []db.Car{s.car}, s.err
}
func (s *apiStore) CountFilteredCars(ctx context.Context, p db.CountFilteredCarsParams) (int64, error) {
	s.record(ctx)
	return 1, s.err
}

func api(t *testing.T, store *apiStore) http.Handler {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return router.New(logger, service.NewCarService(store, nil, logger), nil)
}

const validCarJSON = `{"brand":"Toyota","model":"Camry","production_year":2024,"color":"Blue","price":12345.67}`

func TestAPIRejectsMalformedRequestsBeforePersistence(t *testing.T) {
	t.Parallel()
	tests := []struct{ name, method, path, body string }{
		{"empty body", "POST", "/cars", ""},
		{"malformed JSON", "POST", "/cars", "{"},
		{"wrong price type", "POST", "/cars", `{"price":"oops"}`},
		{"year overflow", "POST", "/cars", `{"production_year":999999}`},
		{"invalid get ID", "GET", "/cars/not-a-uuid", ""},
		{"invalid update ID", "PUT", "/cars/not-a-uuid", validCarJSON},
		{"invalid delete ID", "DELETE", "/cars/not-a-uuid", ""},
		{"invalid update body", "PUT", "/cars/" + uuid.NewString(), "{"},
		{"invalid page", "GET", "/cars?page=0", ""},
		{"invalid limit", "GET", "/cars?limit=101", ""},
		{"nonfinite price", "GET", "/cars?min_price=NaN", ""},
		{"invalid timestamp", "GET", "/cars?created_from=tomorrow", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			store := &apiStore{}
			res := httptest.NewRecorder()
			api(t, store).ServeHTTP(res, httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body)))
			assertAPIError(t, res, http.StatusBadRequest)
			if store.calls != 0 {
				t.Fatalf("invalid request reached persistence %d times", store.calls)
			}
		})
	}
}

func assertAPIError(t *testing.T, res *httptest.ResponseRecorder, status int) {
	t.Helper()
	if res.Code != status {
		t.Fatalf("want %d, got %d: %s", status, res.Code, res.Body.String())
	}
	if res.Header().Get("Content-Type") != "application/json" {
		t.Fatal("error must be JSON")
	}
	var body map[string]string
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil || body["error"] == "" {
		t.Fatalf("invalid error response: %s", res.Body.String())
	}
}

func TestAPIPersistenceErrors(t *testing.T) {
	t.Parallel()
	id := uuid.NewString()
	tests := []struct {
		method, path, body string
		err                error
		status             int
	}{
		{"POST", "/cars", validCarJSON, errors.New("secret database details"), 500},
		{"GET", "/cars/" + id, "", errors.New("secret database details"), 500},
		{"GET", "/cars", "", errors.New("secret database details"), 500},
		{"PUT", "/cars/" + id, validCarJSON, errors.New("secret database details"), 500},
		{"DELETE", "/cars/" + id, "", errors.New("secret database details"), 500},
		{"GET", "/cars/" + id, "", pgx.ErrNoRows, 404},
		{"PUT", "/cars/" + id, validCarJSON, pgx.ErrNoRows, 404},
	}
	for _, tc := range tests {
		t.Run(tc.method+tc.path+tc.err.Error(), func(t *testing.T) {
			t.Parallel()
			store := &apiStore{err: tc.err}
			res := httptest.NewRecorder()
			api(t, store).ServeHTTP(res, httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body)))
			assertAPIError(t, res, tc.status)
			if store.calls != 1 {
				t.Fatalf("want one persistence call, got %d", store.calls)
			}
			if strings.Contains(res.Body.String(), "secret database details") {
				t.Fatal("internal error leaked to client")
			}
		})
	}
}

func TestAPISuccessContracts(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	for _, method := range []string{"POST", "GET", "PUT", "DELETE", "LIST"} {
		t.Run(method, func(t *testing.T) {
			t.Parallel()
			store := &apiStore{car: db.Car{ID: id, Brand: "Toyota", Model: "Camry"}}
			path, verb, status := "/cars/"+id.String(), method, 200
			if method == "POST" {
				path, status = "/cars", 201
			}
			if method == "DELETE" {
				status = 204
			}
			if method == "LIST" {
				verb, path = "GET", "/cars?name=Toyota&year=2024&page=2&limit=5"
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			req := httptest.NewRequest(verb, path, strings.NewReader(validCarJSON)).WithContext(ctx)
			res := httptest.NewRecorder()
			api(t, store).ServeHTTP(res, req)
			if res.Code != status {
				t.Fatalf("want %d, got %d: %s", status, res.Code, res.Body.String())
			}
			cancel()
			if store.ctx == nil || store.ctx.Err() != context.Canceled {
				t.Fatal("request cancellation not propagated to persistence")
			}
			if method == "DELETE" {
				if store.id != id || res.Body.Len() != 0 {
					t.Fatal("delete must pass ID and return an empty body")
				}
				return
			}
			if res.Header().Get("Content-Type") != "application/json" {
				t.Fatal("response must be JSON")
			}
			if method == "LIST" {
				var body struct {
					Data        []db.Car `json:"data"`
					Page, Limit int
					Total       int64
				}
				if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body.Page != 2 || body.Limit != 5 || body.Total != 1 || len(body.Data) != 1 || body.Data[0].ID != id {
					t.Fatalf("invalid list response: %s", res.Body.String())
				}
				if store.filtered.Name != "Toyota" || store.filtered.OffsetCount != 5 || store.filtered.Year.Int16 != 2024 {
					t.Fatalf("filters lost: %+v", store.filtered)
				}
				return
			}
			var body db.Car
			if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.ID != id || body.Brand != "Toyota" {
				t.Fatalf("incorrect car response: %+v", body)
			}
			switch method {
			case "POST":
				p := store.created
				if p.ID == uuid.Nil || p.Brand != "Toyota" || p.Model != "Camry" || p.Color != "Blue" || p.ProductionYear != 2024 || !p.CreatedAt.Valid || !p.UpdatedAt.Valid {
					t.Fatalf("create mapping: %+v", p)
				}
				price, _ := p.Price.MarshalJSON()
				if string(price) != "12345.67" {
					t.Fatalf("price changed: %s", price)
				}
			case "PUT":
				p := store.updated
				if p.ID != id || p.Brand != "Toyota" || p.Model != "Camry" || p.Color != "Blue" || p.ProductionYear != 2024 {
					t.Fatalf("update mapping: %+v", p)
				}
			case "GET":
				if store.id != id {
					t.Fatal("incorrect lookup ID")
				}
			}
		})
	}
}
