//go:build integration && load

package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"

	router "github.com/InatoInato/car_service.git/internal"
	"github.com/InatoInato/car_service.git/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The application runs in-process so -race instruments real handlers/services.
// Only the dedicated test databases are used; CAR_SERVICE_BASE_URL is ignored.
func loadServer(t *testing.T) (*http.Client, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, "postgres://loadtest:loadtest@127.0.0.1:15432/cars_test?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("start docker-compose.test.yml first: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httptest.NewServer(router.New(logger, db.New(pool), nil, pool))
	t.Cleanup(server.Close)
	client := server.Client()
	client.Timeout = 5 * time.Second
	return client, server.URL
}

func loadRequest(client *http.Client, endpoint, method, path string, body any, status int, out any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(method, endpoint+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	if res.StatusCode != status {
		return fmt.Errorf("%s %s: want %d, got %d: %.200s", method, path, status, res.StatusCode, data)
	}
	if out != nil {
		return json.Unmarshal(data, out)
	}
	return nil
}

func loadFixture(t *testing.T, client *http.Client, endpoint string) (carPayload, carResponse) {
	t.Helper()
	payload := carPayload{Brand: "Load-" + uuid.NewString(), Model: "initial", Color: "initial", ProductionYear: 2024, Price: 12345.67}
	var car carResponse
	if err := loadRequest(client, endpoint, "POST", "/cars", payload, 201, &car); err != nil {
		t.Fatal(err)
	}
	if car.ID == "" {
		t.Fatal("missing created car ID")
	}
	t.Cleanup(func() {
		if err := loadRequest(client, endpoint, "DELETE", "/cars/"+car.ID, nil, 204, nil); err != nil {
			t.Errorf("fixture cleanup: %v", err)
		}
	})
	return payload, car
}

// Concurrent calls share one service and HTTP router. Individual overlapping
// reads may see an older version; this checks coherent records, not linearizability.
func TestConcurrentCarReadUpdate(t *testing.T) {
	client, endpoint := loadServer(t)
	payload, car := loadFixture(t, client, endpoint)
	const workers, iterations = 8, 25
	start := make(chan struct{})
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			<-start
			for i := 0; i < iterations; i++ {
				var got carResponse
				method := "GET"
				var body any
				if worker%2 == 0 {
					update := payload
					update.Model = fmt.Sprintf("w%d-i%d", worker, i)
					update.Color = update.Model
					body, method = update, "PUT"
				}
				if err := loadRequest(client, endpoint, method, "/cars/"+car.ID, body, 200, &got); err != nil {
					t.Error(err)
					return
				}
				if got.ID != car.ID || got.Brand != payload.Brand || got.Model != got.Color {
					t.Errorf("incoherent concurrent response: %+v", got)
					return
				}
			}
		}(worker)
	}
	close(start)
	wg.Wait()
}

// Independent records allow strict lifecycle assertions while exercising all
// write/read handlers concurrently. No goroutine calls Fatal/FailNow.
func TestConcurrentCarLifecycles(t *testing.T) {
	client, endpoint := loadServer(t)
	const workers, iterations = 8, 10
	start := make(chan struct{})
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for i := 0; i < iterations; i++ {
				payload := carPayload{Brand: "Concurrent-" + uuid.NewString(), Model: "before", ProductionYear: 2024, Color: "Blue", Price: 100}
				var created carResponse
				if err := loadRequest(client, endpoint, "POST", "/cars", payload, 201, &created); err != nil {
					t.Error(err)
					return
				}
				if _, err := uuid.Parse(created.ID); err != nil {
					t.Errorf("invalid created ID: %q", created.ID)
					return
				}
				path := "/cars/" + created.ID
				t.Cleanup(func() {
					if err := loadRequest(client, endpoint, "DELETE", path, nil, 204, nil); err != nil {
						t.Errorf("cleanup: %v", err)
					}
				})
				payload.Model = "after"
				var updated, fetched carResponse
				if err := loadRequest(client, endpoint, "PUT", path, payload, 200, &updated); err != nil {
					t.Error(err)
					return
				}
				if err := loadRequest(client, endpoint, "GET", path, nil, 200, &fetched); err != nil {
					t.Error(err)
					return
				}
				if updated.ID != created.ID || fetched.ID != created.ID || updated.Model != "after" || fetched.Model != "after" || fetched.Brand != payload.Brand {
					t.Errorf("incorrect lifecycle response: updated=%+v fetched=%+v", updated, fetched)
					return
				}
				if err := loadRequest(client, endpoint, "DELETE", path, nil, 204, nil); err != nil {
					t.Error(err)
					return
				}
				if err := loadRequest(client, endpoint, "GET", path, nil, 404, nil); err != nil {
					t.Error(err)
					return
				}
			}
		}()
	}
	close(start)
	wg.Wait()
}

func loadSetting(t *testing.T, key string, fallback, maximum int) int {
	t.Helper()
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 1 || n > maximum {
		t.Fatalf("%s must be between 1 and %d", key, maximum)
	}
	return n
}

// Fixed-concurrency, closed-loop CRUD load; each worker owns its fixture so
// strict read-after-write checks are meaningful without concurrent writers.
func TestLoadCars(t *testing.T) {
	workers := loadSetting(t, "LOAD_WORKERS", 8, 64)
	iterations := loadSetting(t, "LOAD_ITERATIONS", 25, 1000)
	client, endpoint := loadServer(t)
	type fixture struct {
		payload carPayload
		car     carResponse
	}
	fixtures := make([]fixture, workers)
	for i := range fixtures {
		fixtures[i].payload, fixtures[i].car = loadFixture(t, client, endpoint)
	}
	type result struct {
		latency time.Duration
		err     error
	}
	results := make(chan result, workers*iterations*4)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			<-start
			f := fixtures[worker]
			for i := 0; i < iterations; i++ {
				update := f.payload
				update.Model = fmt.Sprintf("iteration-%d", i)
				for step := 0; step < 4; step++ {
					began := time.Now()
					var got carResponse
					var err error
					switch step {
					case 0:
						err = loadRequest(client, endpoint, "PUT", "/cars/"+f.car.ID, update, 200, &got)
					case 1, 2: // Both reads must reflect the committed DB update.
						err = loadRequest(client, endpoint, "GET", "/cars/"+f.car.ID, nil, 200, &got)
					case 3:
						var list carListResponse
						err = loadRequest(client, endpoint, "GET", "/cars?name="+url.QueryEscape(f.payload.Brand), nil, 200, &list)
						if err == nil && (list.Total != 1 || len(list.Data) != 1 || list.Data[0].ID != f.car.ID || list.Data[0].Model != update.Model) {
							err = fmt.Errorf("incorrect filtered list for worker %d", worker)
						}
					}
					if err == nil && step < 3 && (got.ID != f.car.ID || got.Model != update.Model) {
						err = fmt.Errorf("stale/incorrect read for worker %d, step %d", worker, step)
					}
					results <- result{time.Since(began), err}
				}
			}
		}(worker)
	}
	began := time.Now()
	close(start)
	wg.Wait()
	elapsed := time.Since(began)
	close(results)
	latencies := make([]time.Duration, 0, workers*iterations*4)
	failures := 0
	for r := range results {
		latencies = append(latencies, r.latency)
		if r.err != nil {
			failures++
			if failures <= 5 {
				t.Log(r.err)
			}
		}
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	percentile := func(p int) time.Duration { return latencies[(len(latencies)*p+99)/100-1] }
	t.Logf("RESULT | scenario=crud workers=%d requests=%d errors=%d duration=%s throughput=%.1f req/s p50=%s p95=%s p99=%s", workers, len(latencies), failures, elapsed, float64(len(latencies))/elapsed.Seconds(), percentile(50), percentile(95), percentile(99))
	if failures != 0 {
		t.Fatalf("%d requests failed", failures)
	}
}

// List requests exercise the HTTP
// transport, router and PostgreSQL connection pool under concurrent reads.
func TestLoadListBurst(t *testing.T) {
	workers := loadSetting(t, "LOAD_WORKERS", 8, 64)
	iterations := loadSetting(t, "LOAD_ITERATIONS", 25, 1000)
	client, endpoint := loadServer(t)
	prefix := "List-load-" + uuid.NewString()
	created := make([]carResponse, workers)
	for i := range created {
		payload := carPayload{Brand: prefix, Model: fmt.Sprintf("car-%d", i), ProductionYear: 2024, Color: "Silver", Price: 10000 + float64(i)}
		if err := loadRequest(client, endpoint, http.MethodPost, "/cars", payload, http.StatusCreated, &created[i]); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for _, car := range created {
			if err := loadRequest(client, endpoint, http.MethodDelete, "/cars/"+car.ID, nil, http.StatusNoContent, nil); err != nil {
				t.Errorf("fixture cleanup: %v", err)
			}
		}
	})

	type result struct {
		latency time.Duration
		err     error
	}
	results := make(chan result, workers*iterations)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for i := 0; i < iterations; i++ {
				began := time.Now()
				var list carListResponse
				err := loadRequest(client, endpoint, http.MethodGet, "/cars?name="+url.QueryEscape(prefix)+"&limit=100", nil, http.StatusOK, &list)
				if err == nil && (list.Total != int64(workers) || len(list.Data) != workers) {
					err = fmt.Errorf("incorrect list size: total=%d data=%d want=%d", list.Total, len(list.Data), workers)
				}
				results <- result{latency: time.Since(began), err: err}
			}
		}()
	}
	began := time.Now()
	close(start)
	wg.Wait()
	elapsed := time.Since(began)
	close(results)

	latencies := make([]time.Duration, 0, workers*iterations)
	failures := 0
	for result := range results {
		latencies = append(latencies, result.latency)
		if result.err != nil {
			failures++
			if failures <= 5 {
				t.Log(result.err)
			}
		}
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	percentile := func(p int) time.Duration { return latencies[(len(latencies)*p+99)/100-1] }
	t.Logf("RESULT | scenario=list-burst workers=%d requests=%d errors=%d duration=%s throughput=%.1f req/s p50=%s p95=%s p99=%s", workers, len(latencies), failures, elapsed, float64(len(latencies))/elapsed.Seconds(), percentile(50), percentile(95), percentile(99))
	if failures != 0 {
		t.Fatalf("%d requests failed", failures)
	}
}
