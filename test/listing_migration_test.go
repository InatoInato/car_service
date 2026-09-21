//go:build integration

package test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestIntegrationListingMigrationRoundTrip(t *testing.T) {
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
	// Transaction-local, unique schema: no application or other test table is touched.
	schema := pgx.Identifier{"migration_" + uuid.NewString()}.Sanitize()
	for _, sql := range []string{"CREATE SCHEMA " + schema, "SET LOCAL search_path TO " + schema} {
		if _, err = tx.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	apply := func(name string) {
		t.Helper()
		data, err := os.ReadFile("../database/migrations/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = tx.Exec(ctx, string(data)); err != nil {
			t.Fatal(err)
		}
	}
	apply("000001_create_cars.up.sql")
	id := uuid.New()
	if _, err = tx.Exec(ctx, `INSERT INTO cars(id,brand,model,production_year,color,price) VALUES($1,'Existing','Car',1995,'Black',100)`, id); err != nil {
		t.Fatal(err)
	}
	apply("000002_car_listing_details.up.sql")
	for _, field := range []struct {
		name  string
		limit int
	}{{"image", 2048}, {"model_generation", 100}, {"description", 10000}} {
		savepoint, err := tx.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		_, err = savepoint.Exec(ctx, "UPDATE cars SET "+pgx.Identifier{field.name}.Sanitize()+"=repeat('x',$1)", field.limit+1)
		var databaseError *pgconn.PgError
		if !errors.As(err, &databaseError) || databaseError.Code != "23514" {
			t.Fatalf("missing %s length constraint: %v", field.name, err)
		}
		if err = savepoint.Rollback(ctx); err != nil {
			t.Fatal(err)
		}
	}
	var nulls bool
	if err = tx.QueryRow(ctx, `SELECT image IS NULL AND model_generation IS NULL AND description IS NULL FROM cars WHERE id=$1`, id).Scan(&nulls); err != nil || !nulls {
		t.Fatalf("old car was not preserved with NULLs: %v", err)
	}
	if _, err = tx.Exec(ctx, `UPDATE cars SET image='https://example.com/car.jpg',model_generation='Custom',description='Saved' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	apply("000002_car_listing_details.down.sql")
	var count int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM cars WHERE id=$1 AND brand='Existing'`, id).Scan(&count); err != nil || count != 1 {
		t.Fatalf("rollback lost old car: %v", err)
	}
	apply("000002_car_listing_details.up.sql")
	if err = tx.QueryRow(ctx, `SELECT image IS NULL AND model_generation IS NULL AND description IS NULL FROM cars WHERE id=$1`, id).Scan(&nulls); err != nil || !nulls {
		t.Fatalf("up/down/up failed: %v", err)
	}
	// The outer rollback removes the temporary schema and its data.
}
