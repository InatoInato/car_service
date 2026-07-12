package service

import (
	"context"

	"github.com/InatoInato/car_service.git/internal/db"
	"github.com/google/uuid"
)

type CarStore interface {
	CreateCar(ctx context.Context, arg db.CreateCarParams) (db.Car, error)
	GetCarByID(ctx context.Context, id uuid.UUID) (db.Car, error)
	ListCars(ctx context.Context) ([]db.Car, error)
	UpdateCar(ctx context.Context, arg db.UpdateCarParams) (db.Car, error)
	DeleteCar(ctx context.Context, id uuid.UUID) error
}

type CarService struct {
	store CarStore
}

func NewCarService(store CarStore) *CarService {
	return &CarService{
		store: store,
	}
}

func (s *CarService) CreateCar(
	ctx context.Context,
	params db.CreateCarParams,
) (db.Car, error) {
	return s.store.CreateCar(ctx, params)
}

func (s *CarService) GetCarByID(
	ctx context.Context,
	id uuid.UUID,
) (db.Car, error) {
	return s.store.GetCarByID(ctx, id)
}

func (s *CarService) ListCars(
	ctx context.Context,
) ([]db.Car, error) {
	return s.store.ListCars(ctx)
}

func (s *CarService) UpdateCar(
	ctx context.Context,
	params db.UpdateCarParams,
) (db.Car, error) {
	return s.store.UpdateCar(ctx, params)
}

func (s *CarService) DeleteCar(
	ctx context.Context,
	id uuid.UUID,
) error {
	return s.store.DeleteCar(ctx, id)
}