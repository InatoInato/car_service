//go:build integration

package test

import (
	"context"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestLockedUpdateTimesOutWithoutChangingCar(t *testing.T) {
	client := &http.Client{Timeout: 7 * time.Second}
	car := createCar(t, client, carPayload{
		Brand: "Timeout-" + uuid.NewString(), Model: "Before",
		ProductionYear: 2024, Color: "Black", Price: 100,
	})
	t.Cleanup(func() { deleteCar(t, client, car.ID) })

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
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
	var lockedID uuid.UUID
	if err := tx.QueryRow(ctx, "SELECT id FROM cars WHERE id = $1 FOR UPDATE", car.ID).Scan(&lockedID); err != nil {
		t.Fatal(err)
	}
	if lockedID.String() != car.ID {
		t.Fatal("locked the wrong car")
	}

	update := carPayload{
		Brand: "Timeout test", Model: "After",
		ProductionYear: 2024, Color: "Black", Price: 200,
	}
	started := time.Now()
	var response map[string]string
	requestJSON(t, client, http.MethodPut, "/cars/"+car.ID, update, http.StatusServiceUnavailable, &response)
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("blocked update took %s; database deadline should be shorter than the HTTP timeout", elapsed)
	}
	if response["error"] != "database request timed out" {
		t.Fatalf("unexpected timeout response: %+v", response)
	}

	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	var unchanged carResponse
	requestJSON(t, client, http.MethodGet, "/cars/"+car.ID, nil, http.StatusOK, &unchanged)
	price, err := strconv.ParseFloat(unchanged.Price.String(), 64)
	if err != nil || price != 100 || unchanged.Model != "Before" {
		t.Fatalf("timed-out update changed the car: %+v, price parse error: %v", unchanged, err)
	}

	var updated carResponse
	requestJSON(t, client, http.MethodPut, "/cars/"+car.ID, update, http.StatusOK, &updated)
	price, err = strconv.ParseFloat(updated.Price.String(), 64)
	if err != nil || price != 200 || updated.Model != "After" {
		t.Fatalf("normal update failed after releasing the lock: %+v, price parse error: %v", updated, err)
	}
}
