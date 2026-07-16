package handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/InatoInato/car_service.git/internal/db"
	"github.com/InatoInato/car_service.git/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// mockCarStore implements the service.CarStore interface for testing handlers.
type mockCarStore struct {
	CreateCarFunc  func(ctx context.Context, arg db.CreateCarParams) (db.Car, error)
	GetCarByIDFunc func(ctx context.Context, id uuid.UUID) (db.Car, error)
	ListCarsFunc   func(ctx context.Context) ([]db.Car, error)
	UpdateCarFunc  func(ctx context.Context, arg db.UpdateCarParams) (db.Car, error)
	DeleteCarFunc  func(ctx context.Context, id uuid.UUID) error
}

func (m *mockCarStore) CreateCar(ctx context.Context, arg db.CreateCarParams) (db.Car, error) {
	return m.CreateCarFunc(ctx, arg)
}

func (m *mockCarStore) GetCarByID(ctx context.Context, id uuid.UUID) (db.Car, error) {
	return m.GetCarByIDFunc(ctx, id)
}

func (m *mockCarStore) ListCars(ctx context.Context) ([]db.Car, error) {
	return m.ListCarsFunc(ctx)
}

func (m *mockCarStore) UpdateCar(ctx context.Context, arg db.UpdateCarParams) (db.Car, error) {
	return m.UpdateCarFunc(ctx, arg)
}

func (m *mockCarStore) DeleteCar(ctx context.Context, id uuid.UUID) error {
	return m.DeleteCarFunc(ctx, id)
}

// helper to easily construct route context for URL params (like standard chi routing)
func addChiURLParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestCarHandler_Create(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		mockCreateFunc func(ctx context.Context, arg db.CreateCarParams) (db.Car, error)
		wantStatus     int
		wantBodySub    string
	}{
		{
			name: "Success Created",
			body: `{"brand":"Tesla","model":"Model Y","production_year":2023,"color":"Red","price":45000}`,
			mockCreateFunc: func(ctx context.Context, arg db.CreateCarParams) (db.Car, error) {
				var price pgtype.Numeric
				_ = price.Scan("45000")
				return db.Car{
					ID:             arg.ID,
					Brand:          "Tesla",
					Model:          "Model Y",
					ProductionYear: 2023,
					Color:          "Red",
					Price:          price,
				}, nil
			},
			wantStatus:  http.StatusCreated,
			wantBodySub: `"brand":"Tesla"`,
		},
		{
			name:        "Invalid JSON request",
			body:        `{invalid json`,
			wantStatus:  http.StatusBadRequest,
			wantBodySub: `"error":"invalid request"`,
		},
		{
			name:        "Invalid price type format",
			body:        `{"brand":"Tesla","model":"Model Y","price":"abc"}`,
			wantStatus:  http.StatusBadRequest,
			wantBodySub: `"error":"invalid request"`,
		},
		{
			name: "Database service error",
			body: `{"brand":"Tesla","model":"Model Y","production_year":2023,"color":"Red","price":45000}`,
			mockCreateFunc: func(ctx context.Context, arg db.CreateCarParams) (db.Car, error) {
				return db.Car{}, errors.New("db error connection refused")
			},
			wantStatus:  http.StatusInternalServerError,
			wantBodySub: "db error connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockCarStore{CreateCarFunc: tt.mockCreateFunc}
			svc := service.NewCarService(store)
			handler := NewCarHandler(svc)

			req := httptest.NewRequest(http.MethodPost, "/cars", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.Create(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
			if !strings.Contains(rec.Body.String(), tt.wantBodySub) {
				t.Errorf("expected body to contain %q, got %q", tt.wantBodySub, rec.Body.String())
			}
		})
	}
}

func TestCarHandler_GetByID(t *testing.T) {
	carID := uuid.New()

	tests := []struct {
		name            string
		urlID           string
		mockGetByIDFunc func(ctx context.Context, id uuid.UUID) (db.Car, error)
		wantStatus      int
		wantBodySub     string
	}{
		{
			name:  "Success Found",
			urlID: carID.String(),
			mockGetByIDFunc: func(ctx context.Context, id uuid.UUID) (db.Car, error) {
				return db.Car{ID: id, Brand: "Porsche"}, nil
			},
			wantStatus:  http.StatusOK,
			wantBodySub: `"brand":"Porsche"`,
		},
		{
			name:        "Invalid UUID Format",
			urlID:       "not-a-valid-uuid",
			wantStatus:  http.StatusBadRequest,
			wantBodySub: `"error":"invalid car id"`,
		},
		{
			name:  "Car Not Found",
			urlID: carID.String(),
			mockGetByIDFunc: func(ctx context.Context, id uuid.UUID) (db.Car, error) {
				return db.Car{}, pgx.ErrNoRows
			},
			wantStatus:  http.StatusNotFound,
			wantBodySub: `"error":"car not found"`,
		},
		{
			name:  "Server Database Error",
			urlID: carID.String(),
			mockGetByIDFunc: func(ctx context.Context, id uuid.UUID) (db.Car, error) {
				return db.Car{}, errors.New("something went wrong")
			},
			wantStatus:  http.StatusInternalServerError,
			wantBodySub: `"error":"something went wrong"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockCarStore{GetCarByIDFunc: tt.mockGetByIDFunc}
			svc := service.NewCarService(store)
			handler := NewCarHandler(svc)

			req := httptest.NewRequest(http.MethodGet, "/cars/"+tt.urlID, nil)
			req = addChiURLParams(req, map[string]string{"id": tt.urlID})
			rec := httptest.NewRecorder()

			handler.GetByID(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
			if !strings.Contains(rec.Body.String(), tt.wantBodySub) {
				t.Errorf("expected body to contain %q, got %q", tt.wantBodySub, rec.Body.String())
			}
		})
	}
}

func TestCarHandler_List(t *testing.T) {
	tests := []struct {
		name         string
		mockListFunc func(ctx context.Context) ([]db.Car, error)
		wantStatus   int
		wantBodySub  string
	}{
		{
			name: "Success empty array",
			mockListFunc: func(ctx context.Context) ([]db.Car, error) {
				return []db.Car{}, nil
			},
			wantStatus:  http.StatusOK,
			wantBodySub: `[]`,
		},
		{
			name: "Success multi-element",
			mockListFunc: func(ctx context.Context) ([]db.Car, error) {
				return []db.Car{{Brand: "Audi"}, {Brand: "BMW"}}, nil
			},
			wantStatus:  http.StatusOK,
			wantBodySub: `"brand":"Audi"`,
		},
		{
			name: "Database read failure",
			mockListFunc: func(ctx context.Context) ([]db.Car, error) {
				return nil, errors.New("read failed")
			},
			wantStatus:  http.StatusInternalServerError,
			wantBodySub: `"error":"failed to fetch cars"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockCarStore{ListCarsFunc: tt.mockListFunc}
			svc := service.NewCarService(store)
			handler := NewCarHandler(svc)

			req := httptest.NewRequest(http.MethodGet, "/cars", nil)
			rec := httptest.NewRecorder()

			handler.List(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
			if !strings.Contains(rec.Body.String(), tt.wantBodySub) {
				t.Errorf("expected body to contain %q, got %q", tt.wantBodySub, rec.Body.String())
			}
		})
	}
}

func TestCarHandler_Update(t *testing.T) {
	carID := uuid.New()

	tests := []struct {
		name           string
		urlID          string
		body           string
		mockUpdateFunc func(ctx context.Context, arg db.UpdateCarParams) (db.Car, error)
		wantStatus     int
		wantBodySub    string
	}{
		{
			name:  "Success Update",
			urlID: carID.String(),
			body:  `{"brand":"Toyota","model":"Camry","production_year":2022,"color":"White","price":28000}`,
			mockUpdateFunc: func(ctx context.Context, arg db.UpdateCarParams) (db.Car, error) {
				if arg.ID != carID {
					return db.Car{}, errors.New("incorrect ID injected")
				}
				return db.Car{
					ID:             arg.ID,
					Brand:          arg.Brand,
					Model:          arg.Model,
					ProductionYear: arg.ProductionYear,
					Color:          arg.Color,
					Price:          arg.Price,
				}, nil
			},
			wantStatus:  http.StatusOK,
			wantBodySub: `"brand":"Toyota"`,
		},
		{
			name:        "Invalid ID",
			urlID:       "bad-id",
			body:        `{"brand":"Toyota"}`,
			wantStatus:  http.StatusBadRequest,
			wantBodySub: `"error":"invalid car id"`,
		},
		{
			name:        "Invalid payload",
			urlID:       carID.String(),
			body:        `{bad body}`,
			wantStatus:  http.StatusBadRequest,
			wantBodySub: `"error":"invalid request body"`,
		},
		{
			name:  "Car not found on update",
			urlID: carID.String(),
			body:  `{"brand":"Toyota","price":28000}`,
			mockUpdateFunc: func(ctx context.Context, arg db.UpdateCarParams) (db.Car, error) {
				return db.Car{}, pgx.ErrNoRows
			},
			wantStatus:  http.StatusNotFound,
			wantBodySub: `"error":"car not found"`,
		},
		{
			name:  "Internal DB error",
			urlID: carID.String(),
			body:  `{"brand":"Toyota","price":28000}`,
			mockUpdateFunc: func(ctx context.Context, arg db.UpdateCarParams) (db.Car, error) {
				return db.Car{}, errors.New("something went wrong")
			},
			wantStatus:  http.StatusInternalServerError,
			wantBodySub: `"error":"failed to update car"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockCarStore{UpdateCarFunc: tt.mockUpdateFunc}
			svc := service.NewCarService(store)
			handler := NewCarHandler(svc)

			req := httptest.NewRequest(http.MethodPut, "/cars/"+tt.urlID, bytes.NewBufferString(tt.body))
			req = addChiURLParams(req, map[string]string{"id": tt.urlID})
			rec := httptest.NewRecorder()

			handler.Update(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
			if !strings.Contains(rec.Body.String(), tt.wantBodySub) {
				t.Errorf("expected body to contain %q, got %q", tt.wantBodySub, rec.Body.String())
			}
		})
	}
}

func TestCarHandler_Delete(t *testing.T) {
	carID := uuid.New()

	tests := []struct {
		name           string
		urlID          string
		mockDeleteFunc func(ctx context.Context, id uuid.UUID) error
		wantStatus     int
		wantBodySub    string
	}{
		{
			name:  "Success Delete",
			urlID: carID.String(),
			mockDeleteFunc: func(ctx context.Context, id uuid.UUID) error {
				if id != carID {
					return errors.New("wrong ID")
				}
				return nil
			},
			wantStatus:  http.StatusNoContent,
			wantBodySub: "",
		},
		{
			name:        "Invalid ID",
			urlID:       "bad-id",
			wantStatus:  http.StatusBadRequest,
			wantBodySub: `"error":"invalid car id"`,
		},
		{
			name:  "Internal failure",
			urlID: carID.String(),
			mockDeleteFunc: func(ctx context.Context, id uuid.UUID) error {
				return errors.New("delete failed")
			},
			wantStatus:  http.StatusInternalServerError,
			wantBodySub: `"error":"failed to delete car"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockCarStore{DeleteCarFunc: tt.mockDeleteFunc}
			svc := service.NewCarService(store)
			handler := NewCarHandler(svc)

			req := httptest.NewRequest(http.MethodDelete, "/cars/"+tt.urlID, nil)
			req = addChiURLParams(req, map[string]string{"id": tt.urlID})
			rec := httptest.NewRecorder()

			handler.Delete(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
			if tt.wantBodySub != "" && !strings.Contains(rec.Body.String(), tt.wantBodySub) {
				t.Errorf("expected body to contain %q, got %q", tt.wantBodySub, rec.Body.String())
			}
		})
	}
}