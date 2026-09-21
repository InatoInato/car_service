package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/InatoInato/car_service.git/internal/db"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
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
	store  CarStore
	redis  *redis.Client
	logger *slog.Logger
}

func NewCarService(store CarStore, redis *redis.Client, logger *slog.Logger) *CarService {
	if logger == nil {
		logger = slog.Default()
	}

	return &CarService{
		store:  store,
		redis:  redis,
		logger: logger,
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
	key := carCacheKey(id)

	if s.redis != nil {
		val, err := s.redis.Get(ctx, key).Result()
		switch err {
		case nil:
			var car db.Car
			if err := json.Unmarshal([]byte(val), &car); err == nil {
				s.logger.Info("redis cache hit", "key", key)
				return car, nil
			} else {
				s.logger.Warn("failed to unmarshal redis cache value", "key", key, "error", err)
			}
		case redis.Nil:
			s.logger.Info("redis cache miss", "key", key)
		default:
			s.logger.Warn("redis cache read failed", "key", key, "error", err)
		}
	}

	car, err := s.store.GetCarByID(ctx, id)
	if err != nil {
		return db.Car{}, err
	}

	if s.redis != nil {
		if data, err := json.Marshal(car); err == nil {
			if err := s.redis.Set(ctx, key, data, 10*time.Minute).Err(); err != nil {
				s.logger.Warn("redis cache write failed", "key", key, "error", err)
			} else {
				s.logger.Info("redis cache write", "key", key, "ttl", 10*time.Minute)
			}
		} else {
			s.logger.Warn("failed to marshal redis cache value", "key", key, "error", err)
		}
	}

	return car, nil
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
	car, err := s.store.UpdateCar(ctx, params)
	if err != nil {
		return db.Car{}, err
	}

	if s.redis != nil {
		key := carCacheKey(car.ID)
		if err := s.redis.Del(ctx, key).Err(); err != nil {
			s.logger.Warn("redis cache invalidation failed", "key", key, "error", err)
		} else {
			s.logger.Info("redis cache invalidated", "key", key)
		}
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

	if s.redis != nil {
		key := carCacheKey(id)
		if err := s.redis.Del(ctx, key).Err(); err != nil {
			s.logger.Warn("redis cache invalidation failed", "key", key, "error", err)
		} else {
			s.logger.Info("redis cache invalidated", "key", key)
		}
	}

	return nil
}

// A schema-versioned namespace avoids decoding old cached records that omit
// listing details. Old keys expire naturally; no Redis-wide flush is needed.
func carCacheKey(id uuid.UUID) string { return "cars:v2:" + id.String() }
