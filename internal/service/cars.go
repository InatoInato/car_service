package service

import (
	"context"
	"fmt"

	"github.com/InatoInato/car_service.git/internal/db"
	"github.com/google/uuid"
)

type CarStore interface {
	CreateCar(ctx context.Context, arg db.CreateCarParams) (db.Car, error)
	GetCarByID(ctx context.Context, id uuid.UUID) (db.Car, error)
	ListCars(ctx context.Context, arg db.ListCarsParams) ([]db.Car, error)
	CountCars(ctx context.Context) (int64, error)
	UpdateCar(ctx context.Context, arg db.UpdateCarParams) (db.Car, error)
	DeleteCar(ctx context.Context, id uuid.UUID) error
}

type FilteredCarStore interface {
	FilterCars(ctx context.Context, arg db.FilterCarsParams) ([]db.Car, error)
	CountFilteredCars(ctx context.Context, arg db.CountFilteredCarsParams) (int64, error)
}

type CarService struct {
	store CarStore
}

func NewCarService(store CarStore) *CarService {
	return &CarService{store: store}
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
	// A mutable listing must be read from the source of truth. A cache
	// invalidation failure after a committed write cannot safely be hidden.
	return s.store.GetCarByID(ctx, id)
}

func (s *CarService) ListCars(
	ctx context.Context,
	limit int32,
	offset int32,
) ([]db.Car, int64, error) {
	cars, err := s.store.ListCars(ctx, db.ListCarsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := s.store.CountCars(ctx)
	if err != nil {
		return nil, 0, err
	}

	return cars, total, nil
}

func (s *CarService) FilterCars(
	ctx context.Context,
	params db.FilterCarsParams,
) ([]db.Car, int64, error) {
	store, ok := s.store.(FilteredCarStore)
	if !ok {
		return nil, 0, fmt.Errorf("car store does not support filtering")
	}

	cars, err := store.FilterCars(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	total, err := store.CountFilteredCars(ctx, db.CountFilteredCarsParams{
		Name:        params.Name,
		Year:        params.Year,
		CreatedFrom: params.CreatedFrom,
		CreatedTo:   params.CreatedTo,
		MinPrice:    params.MinPrice,
		MaxPrice:    params.MaxPrice,
	})
	if err != nil {
		return nil, 0, err
	}

	return cars, total, nil
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
