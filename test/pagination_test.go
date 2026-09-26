//go:build integration

package test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"testing"
	"time"

	router "github.com/InatoInato/car_service.git/internal"
	"github.com/InatoInato/car_service.git/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestIntegrationPaginationWithEqualCreationTimes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, testDatabaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(context.Background())
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())

	// Equal timestamps occur in bulk imports. Keep the fixture in a transaction
	// and serve requests through that transaction so rollback removes every row.
	brand := "Pagination-" + uuid.NewString()
	rows, err := tx.Query(ctx, `
		INSERT INTO cars (id, brand, model, production_year, color, price, created_at)
		SELECT gen_random_uuid(), $1, 'Same time', 2024, 'Black', 100,
		       '2024-01-01T00:00:00Z'::timestamptz
		FROM generate_series(1, 61)
		RETURNING id`, brand)
	if err != nil {
		t.Fatal(err)
	}
	var expected []string
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		expected = append(expected, id.String())
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(expected)))

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httptest.NewServer(router.New(logger, db.New(tx), nil, nil))
	defer server.Close()
	client := server.Client()
	client.Timeout = 5 * time.Second
	seen := make(map[string]bool)
	var returned []string
	for page := 1; page <= 5; page++ {
		res, err := client.Get(fmt.Sprintf("%s/cars?name=%s&page=%d&limit=20", server.URL, url.QueryEscape(brand), page))
		if err != nil {
			t.Fatal(err)
		}
		var result carListResponse
		err = json.NewDecoder(res.Body).Decode(&result)
		res.Body.Close()
		if err != nil || res.StatusCode != http.StatusOK {
			t.Fatalf("page %d: status=%d decode error=%v", page, res.StatusCode, err)
		}
		start := min((page-1)*20, len(expected))
		end := min(start+20, len(expected))
		if result.Total != int64(len(expected)) || len(result.Data) != end-start {
			t.Fatalf("page %d: total=%d rows=%d; want total=%d rows=%d", page, result.Total, len(result.Data), len(expected), end-start)
		}
		for _, car := range result.Data {
			if seen[car.ID] {
				t.Fatalf("car %s repeated on page %d", car.ID, page)
			}
			seen[car.ID] = true
			returned = append(returned, car.ID)
		}
	}
	if len(seen) != len(expected) {
		t.Fatalf("pagination returned %d unique cars; want %d", len(seen), len(expected))
	}
	for i, id := range returned {
		if id != expected[i] {
			t.Fatalf("item %d: got %s, want %s in stable ID order", i, id, expected[i])
		}
	}
}
