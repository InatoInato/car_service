//go:build integration

package test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	router "github.com/InatoInato/car_service.git/internal"
	"github.com/InatoInato/car_service.git/internal/db"
	"github.com/InatoInato/car_service.git/internal/provider"
	"github.com/InatoInato/car_service.git/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const testDatabaseURL = "postgres://loadtest:loadtest@127.0.0.1:15432/cars_test?sslmode=disable"

// Defaults to the isolated test stack, never the developer's port-8080 API.
// An explicit CAR_SERVICE_BASE_URL retains the existing external-test workflow.
func TestMain(m *testing.M) { os.Exit(runIntegration(m)) }

func runIntegration(m *testing.M) int {
	if os.Getenv("CAR_SERVICE_BASE_URL") != "" {
		return m.Run()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, testDatabaseURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer pool.Close()
	if err = pool.Ping(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "Run make test-up first:", err)
		return 1
	}
	cache := redis.NewClient(&redis.Options{Addr: "127.0.0.1:16379", DialTimeout: time.Second, ReadTimeout: time.Second, WriteTimeout: time.Second})
	defer cache.Close()
	if err = cache.Ping(ctx).Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Test Redis unavailable:", err)
		return 1
	}
	catalogue, err := provider.NewGenerationCatalog()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httptest.NewServer(router.New(logger, service.NewCarService(db.New(pool), cache, logger), service.NewGenerationService(catalogue)))
	defer server.Close()
	baseURL = server.URL
	return m.Run()
}
