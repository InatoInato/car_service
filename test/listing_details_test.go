//go:build integration

package test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/InatoInato/car_service.git/internal/handler/dto"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func TestIntegrationListingDetails(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}
	for _, mode := range []string{"omitted", "null", "empty", "values"} {
		t.Run(mode, func(t *testing.T) {
			brand := "Optional-" + uuid.NewString()
			body := map[string]any{"brand": brand, "model": "Unknown model", "production_year": 1995, "color": "Black", "price": 100}
			if mode != "omitted" {
				var image, generation, description any
				switch mode {
				case "empty":
					image, generation, description = " ", "", "\n"
				case "values":
					image, generation, description = "https://example.com/car.jpg", "Owner supplied generation", "History\nSecond line"
				}
				body["image"], body["model_generation"], body["description"] = image, generation, description
			}
			var created dto.CarResponse
			requestJSON(t, client, "POST", "/cars", body, 201, &created)
			if created.ID == "" {
				t.Fatal("missing car ID")
			}
			t.Cleanup(func() { deleteCar(t, client, created.ID) })
			path := "/cars/" + created.ID
			if mode == "values" {
				assertDetails(t, created, "https://example.com/car.jpg", "Owner supplied generation", "History\nSecond line")
			} else {
				assertDetails(t, created, "", "", "")
			}
			var got dto.CarResponse
			requestJSON(t, client, "GET", path, nil, 200, &got) // warm Redis
			if mode == "values" {
				assertDetails(t, got, "https://example.com/car.jpg", "Owner supplied generation", "History\nSecond line")
			} else {
				assertDetails(t, got, "", "", "")
			}
			body["image"], body["model_generation"], body["description"] = "http://localhost:3000/car.png", "Custom generation", "Changed"
			requestJSON(t, client, "PUT", path, body, 200, &got)
			assertDetails(t, got, "http://localhost:3000/car.png", "Custom generation", "Changed")
			requestJSON(t, client, "GET", path, nil, 200, &got)
			assertDetails(t, got, "http://localhost:3000/car.png", "Custom generation", "Changed")
			// Legacy clients omit these fields: changing price must not erase details.
			delete(body, "image")
			delete(body, "model_generation")
			delete(body, "description")
			body["price"] = 200
			requestJSON(t, client, "PUT", path, body, 200, &got)
			assertDetails(t, got, "http://localhost:3000/car.png", "Custom generation", "Changed")
			var list struct {
				Data []dto.CarResponse `json:"data"`
			}
			requestJSON(t, client, "GET", "/cars?name="+url.QueryEscape(brand), nil, 200, &list)
			if len(list.Data) != 1 {
				t.Fatalf("unexpected list %+v", list)
			}
			assertDetails(t, list.Data[0], "http://localhost:3000/car.png", "Custom generation", "Changed")
			// A partial optional edit leaves the other optional fields untouched.
			body["description"] = "Only description changed"
			requestJSON(t, client, "PUT", path, body, 200, &got)
			assertDetails(t, got, "http://localhost:3000/car.png", "Custom generation", "Only description changed")
			body["image"], body["model_generation"], body["description"] = nil, " ", nil
			requestJSON(t, client, "PUT", path, body, 200, &got)
			assertDetails(t, got, "", "", "")
			var raw map[string]any
			requestJSON(t, client, "GET", path, nil, 200, &raw)
			for _, key := range []string{"image", "model_generation", "description"} {
				value, ok := raw[key]
				if !ok || value != nil {
					t.Fatalf("%s must be present JSON null: %v", key, raw)
				}
			}
		})
	}
}

func assertDetails(t *testing.T, car dto.CarResponse, image, generation, description string) {
	t.Helper()
	for _, field := range []struct {
		name, want string
		got        *string
	}{{"image", image, car.Image}, {"generation", generation, car.ModelGeneration}, {"description", description, car.Description}} {
		if field.want == "" {
			if field.got != nil {
				t.Fatalf("%s should be null, got %q", field.name, *field.got)
			}
		} else if field.got == nil || *field.got != field.want {
			t.Fatalf("%s want %q, got %v", field.name, field.want, field.got)
		}
	}
}

func TestIntegrationIgnoresPreMigrationCache(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}
	body := map[string]any{"brand": "Cache-" + uuid.NewString(), "model": "M", "production_year": 1995, "color": "Black", "price": 100, "image": "https://example.com/image.jpg"}
	var created dto.CarResponse
	requestJSON(t, client, "POST", "/cars", body, 201, &created)
	t.Cleanup(func() { deleteCar(t, client, created.ID) })
	cache := redis.NewClient(&redis.Options{Addr: "127.0.0.1:16379"})
	defer cache.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	legacy, _ := json.Marshal(map[string]any{"id": created.ID, "brand": created.Brand, "model": "Old cache"})
	key := "cars:" + created.ID
	if err := cache.Set(ctx, key, legacy, time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	defer cache.Del(context.Background(), key)
	var got dto.CarResponse
	requestJSON(t, client, "GET", "/cars/"+created.ID, nil, 200, &got)
	assertDetails(t, got, "https://example.com/image.jpg", "", "")
}
