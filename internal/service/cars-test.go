package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/InatoInato/car_service.git/internal/db"
	"github.com/google/uuid"
)

// mockCarStore implements the CarStore interface for testing purposes.
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

// --- Tests ---

func TestCarService_CreateCar(t *testing.T) {
	ctx := context.Background()
	carID := uuid.New()
	mockCar := db.Car{ID: carID}
	mockParams := db.CreateCarParams{}
	dbErr := errors.New("database connection failed")

	tests := []struct {
		name    string
		mockFn  func(ctx context.Context, arg db.CreateCarParams) (db.Car, error)
		params  db.CreateCarParams
		want    db.Car
		wantErr bool
	}{
		{
			name: "Success case",
			mockFn: func(ctx context.Context, arg db.CreateCarParams) (db.Car, error) {
				return mockCar, nil
			},
			params:  mockParams,
			want:    mockCar,
			wantErr: false,
		},
		{
			name: "Database error case",
			mockFn: func(ctx context.Context, arg db.CreateCarParams) (db.Car, error) {
				return db.Car{}, dbErr
			},
			params:  mockParams,
			want:    db.Car{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockCarStore{CreateCarFunc: tt.mockFn}
			s := NewCarService(store)

			got, err := s.CreateCar(ctx, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateCar() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateCar() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCarService_GetCarByID(t *testing.T) {
	ctx := context.Background()
	carID := uuid.New()
	mockCar := db.Car{ID: carID}
	dbErr := errors.New("car not found")

	tests := []struct {
		name    string
		mockFn  func(ctx context.Context, id uuid.UUID) (db.Car, error)
		id      uuid.UUID
		want    db.Car
		wantErr bool
	}{
		{
			name: "Success case",
			mockFn: func(ctx context.Context, id uuid.UUID) (db.Car, error) {
				return mockCar, nil
			},
			id:      carID,
			want:    mockCar,
			wantErr: false,
		},
		{
			name: "Not found error case",
			mockFn: func(ctx context.Context, id uuid.UUID) (db.Car, error) {
				return db.Car{}, dbErr
			},
			id:      carID,
			want:    db.Car{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockCarStore{GetCarByIDFunc: tt.mockFn}
			s := NewCarService(store)

			got, err := s.GetCarByID(ctx, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCarByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCarByID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCarService_ListCars(t *testing.T) {
	ctx := context.Background()
	mockCars := []db.Car{{ID: uuid.New()}, {ID: uuid.New()}}

	store := &mockCarStore{
		ListCarsFunc: func(ctx context.Context) ([]db.Car, error) {
			return mockCars, nil
		},
	}
	s := NewCarService(store)

	got, err := s.ListCars(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, mockCars) {
		t.Errorf("ListCars() = %v, want %v", got, mockCars)
	}
}

func TestCarService_DeleteCar(t *testing.T) {
	ctx := context.Background()
	carID := uuid.New()

	store := &mockCarStore{
		DeleteCarFunc: func(ctx context.Context, id uuid.UUID) error {
			if id != carID {
				return errors.New("wrong ID passed")
			}
			return nil
		},
	}
	s := NewCarService(store)

	err := s.DeleteCar(ctx, carID)
	if err != nil {
		t.Errorf("unexpected error on delete: %v", err)
	}
}