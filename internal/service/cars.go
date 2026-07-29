package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/InatoInato/car_service.git/internal/db"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type CarStore interface {
	CreateCar(ctx context.Context, arg db.CreateCarParams) (db.Car, error)
	GetCarByID(ctx context.Context, id uuid.UUID) (db.Car, error)
	ListCars(ctx context.Context, arg db.ListCarsParams) ([]db.Car, error)
	UpdateCar(ctx context.Context, arg db.UpdateCarParams) (db.Car, error)
	DeleteCar(ctx context.Context, id uuid.UUID) error
}

type CarService struct {
	store CarStore
	redis *redis.Client
}

func NewCarService(store CarStore, redis *redis.Client) *CarService {
	return &CarService{
		store: store,
		redis: redis,
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
	key := fmt.Sprintf("cars:%s", id.String())

	// 1. Try Redis cache hit
	if s.redis != nil {
		val, err := s.redis.Get(ctx, key).Result()
		if err == nil {
			var car db.Car
			if err := json.Unmarshal([]byte(val), &car); err == nil {
				return car, nil
			}
		}
	}

	// 2. Cache miss: Query PostgreSQL
	car, err := s.store.GetCarByID(ctx, id)
	if err != nil {
		return db.Car{}, err
	}

	// 3. Save to Redis with 10 minute TTL
	if s.redis != nil {
		if data, err := json.Marshal(car); err == nil {
			s.redis.Set(ctx, key, data, 10*time.Minute)
		}
	}

	return car, nil
}

func (s *CarService) ListCars(
	ctx context.Context,
	limit int32,
	offset int32,
) ([]db.Car, error) {
	return s.store.ListCars(ctx, db.ListCarsParams{
		Limit:  limit,
		Offset: offset,
	})
}

func (s *CarService) UpdateCar(
	ctx context.Context,
	params db.UpdateCarParams,
) (db.Car, error) {
	car, err := s.store.UpdateCar(ctx, params)
	if err != nil {
		return db.Car{}, err
	}

	// Invalidate old cache
	if s.redis != nil {
		key := fmt.Sprintf("cars:%s", car.ID.String())
		s.redis.Del(ctx, key)
	}

	return car, nil
}

func (s *CarService) DeleteCar(
	ctx context.Context,
	id uuid.UUID,
) error {
	if err := s.store.DeleteCar(ctx, id); err != nil {
		return err
	}

	// Invalidate old cache
	if s.redis != nil {
		key := fmt.Sprintf("cars:%s", id.String())
		s.redis.Del(ctx, key)
	}

	return nil
}
