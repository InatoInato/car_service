package service

import (
	"context"
	"errors"
	"testing"

	"github.com/InatoInato/car_service.git/internal/db"
	"github.com/google/uuid"
)

type mockCarStore struct {
	createFn func(context.Context, db.CreateCarParams) (db.Car, error)
	getFn    func(context.Context, uuid.UUID) (db.Car, error)
	listFn   func(context.Context, db.ListCarsParams) ([]db.Car, error)
	updateFn func(context.Context, db.UpdateCarParams) (db.Car, error)
	deleteFn func(context.Context, uuid.UUID) error
}

func (m *mockCarStore) CreateCar(ctx context.Context, arg db.CreateCarParams) (db.Car, error) {
	return m.createFn(ctx, arg)
}

func (m *mockCarStore) GetCarByID(ctx context.Context, id uuid.UUID) (db.Car, error) {
	return m.getFn(ctx, id)
}

func (m *mockCarStore) ListCars(ctx context.Context, arg db.ListCarsParams) ([]db.Car, error) {
	return m.listFn(ctx, arg)
}

func (m *mockCarStore) UpdateCar(ctx context.Context, arg db.UpdateCarParams) (db.Car, error) {
	return m.updateFn(ctx, arg)
}

func (m *mockCarStore) DeleteCar(ctx context.Context, id uuid.UUID) error {
	return m.deleteFn(ctx, id)
}

func TestCreateCar(t *testing.T) {
	store := &mockCarStore{
		createFn: func(ctx context.Context, arg db.CreateCarParams) (db.Car, error) {

			if arg.Brand != "BMW" {
				t.Fatal("brand not passed")
			}

			return db.Car{
				ID: arg.ID,
				Brand: arg.Brand,
			}, nil
		},
	}

	svc := NewCarService(store)

	car, err := svc.CreateCar(context.Background(), db.CreateCarParams{
		ID: uuid.New(),
		Brand: "BMW",
	})

	if err != nil {
		t.Fatal(err)
	}

	if car.Brand != "BMW" {
		t.Fatal("unexpected brand")
	}
}

func TestGetCarByID(t *testing.T) {

	id := uuid.New()

	store := &mockCarStore{
		getFn: func(ctx context.Context, uid uuid.UUID) (db.Car, error) {

			if uid != id {
				t.Fatal("wrong id")
			}

			return db.Car{
				ID: id,
				Brand: "Honda",
			}, nil
		},
	}

	svc := NewCarService(store)

	car, err := svc.GetCarByID(context.Background(), id)

	if err != nil {
		t.Fatal(err)
	}

	if car.ID != id {
		t.Fatal("wrong car")
	}
}

func TestListCars(t *testing.T) {

	store := &mockCarStore{
		listFn: func(ctx context.Context, arg db.ListCarsParams) ([]db.Car, error) {

			if arg.Limit != 20 {
				t.Fatal("wrong limit")
			}

			if arg.Offset != 0 {
				t.Fatal("wrong offset")
			}

			return []db.Car{
				{Brand: "BMW"},
				{Brand: "Audi"},
			}, nil
		},
	}

	svc := NewCarService(store)

	cars, err := svc.ListCars(context.Background(), 20, 0)

	if err != nil {
		t.Fatal(err)
	}

	if len(cars) != 2 {
		t.Fatal("expected 2 cars")
	}
}

func TestUpdateCar(t *testing.T) {

	store := &mockCarStore{
		updateFn: func(ctx context.Context, arg db.UpdateCarParams) (db.Car, error) {

			arg.Brand = "Mercedes"

			return db.Car{
				ID: arg.ID,
				Brand: arg.Brand,
			}, nil
		},
	}

	svc := NewCarService(store)

	id := uuid.New()

	car, err := svc.UpdateCar(context.Background(), db.UpdateCarParams{
		ID: id,
	})

	if err != nil {
		t.Fatal(err)
	}

	if car.Brand != "Mercedes" {
		t.Fatal("update failed")
	}
}

func TestDeleteCar(t *testing.T) {

	called := false

	store := &mockCarStore{
		deleteFn: func(ctx context.Context, id uuid.UUID) error {
			called = true
			return nil
		},
	}

	svc := NewCarService(store)

	err := svc.DeleteCar(context.Background(), uuid.New())

	if err != nil {
		t.Fatal(err)
	}

	if !called {
		t.Fatal("delete wasn't called")
	}
}

func TestCreateCar_Error(t *testing.T) {

	store := &mockCarStore{
		createFn: func(ctx context.Context, arg db.CreateCarParams) (db.Car, error) {
			return db.Car{}, errors.New("db error")
		},
	}

	svc := NewCarService(store)

	_, err := svc.CreateCar(context.Background(), db.CreateCarParams{})

	if err == nil {
		t.Fatal("expected error")
	}
}