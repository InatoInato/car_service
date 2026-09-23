//go:build integration

package test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"
)

type carPayload struct {
	Brand          string  `json:"brand"`
	Model          string  `json:"model"`
	ProductionYear int16   `json:"production_year"`
	Color          string  `json:"color"`
	Price          float64 `json:"price"`
}

type carResponse struct {
	ID             string      `json:"id"`
	Brand          string      `json:"brand"`
	Model          string      `json:"model"`
	ProductionYear int16       `json:"production_year"`
	Color          string      `json:"color"`
	Price          json.Number `json:"price"`
	CreatedAt      string      `json:"created_at"`
	UpdatedAt      string      `json:"updated_at"`
}

type carListResponse struct {
	Data  []carResponse `json:"data"`
	Page  int           `json:"page"`
	Limit int           `json:"limit"`
	Total int64         `json:"total"`
}

func TestIntegrationCarsCRUDAndFilters(t *testing.T) {
	waitForServer(t)

	client := &http.Client{Timeout: 5 * time.Second}
	uniqueName := "Integration-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	createdFrom := time.Now().UTC().Add(-2 * time.Second)

	first := createCar(t, client, carPayload{
		Brand:          uniqueName,
		Model:          "Alpha",
		ProductionYear: 2021,
		Color:          "Black",
		Price:          15001.25,
	})
	defer deleteCar(t, client, first.ID)

	second := createCar(t, client, carPayload{
		Brand:          uniqueName,
		Model:          "Beta",
		ProductionYear: 2022,
		Color:          "White",
		Price:          35002.75,
	})
	defer deleteCar(t, client, second.ID)

	t.Run("create and get", func(t *testing.T) {
		var fetched carResponse
		requestJSON(t, client, http.MethodGet, "/cars/"+first.ID, nil, http.StatusOK, &fetched)
		assertCar(t, fetched, first.ID, uniqueName, "Alpha", 2021, "15001.25")
	})

	t.Run("update is visible on subsequent get", func(t *testing.T) {
		payload := carPayload{
			Brand:          uniqueName,
			Model:          "Alpha Updated",
			ProductionYear: 2024,
			Color:          "Blue",
			Price:          25000.50,
		}

		var updated carResponse
		requestJSON(t, client, http.MethodPut, "/cars/"+first.ID, payload, http.StatusOK, &updated)
		assertCar(t, updated, first.ID, uniqueName, "Alpha Updated", 2024, "25000.50")

		var fetched carResponse
		requestJSON(t, client, http.MethodGet, "/cars/"+first.ID, nil, http.StatusOK, &fetched)
		assertCar(t, fetched, first.ID, uniqueName, "Alpha Updated", 2024, "25000.50")
	})

	t.Run("filter by name", func(t *testing.T) {
		result := listCars(t, client, url.Values{"name": {uniqueName}})
		assertListContainsOnly(t, result, first.ID, second.ID)
	})

	t.Run("filter by combined brand and model", func(t *testing.T) {
		result := listCars(t, client, url.Values{"name": {uniqueName + " Alpha Updated"}})
		assertListContainsOnly(t, result, first.ID)
	})

	t.Run("filter by year and production year alias", func(t *testing.T) {
		result := listCars(t, client, url.Values{
			"name": {uniqueName},
			"year": {"2024"},
		})
		assertListContainsOnly(t, result, first.ID)

		result = listCars(t, client, url.Values{
			"name":            {uniqueName},
			"production_year": {"2022"},
		})
		assertListContainsOnly(t, result, second.ID)
	})

	t.Run("filter by price", func(t *testing.T) {
		result := listCars(t, client, url.Values{
			"name":      {uniqueName},
			"min_price": {"30000"},
			"max_price": {"40000"},
		})
		assertListContainsOnly(t, result, second.ID)

		result = listCars(t, client, url.Values{
			"name":  {uniqueName},
			"price": {"25000.50"},
		})
		assertListContainsOnly(t, result, first.ID)
	})

	t.Run("filter by created time", func(t *testing.T) {
		result := listCars(t, client, url.Values{
			"name":         {uniqueName},
			"created_from": {createdFrom.Format(time.RFC3339Nano)},
			"created_to":   {time.Now().UTC().Add(2 * time.Second).Format(time.RFC3339Nano)},
		})
		assertListContainsOnly(t, result, first.ID, second.ID)

		result = listCars(t, client, url.Values{
			"name":       {uniqueName},
			"created_to": {createdFrom.Add(-time.Second).Format(time.RFC3339Nano)},
		})
		assertListContainsOnly(t, result)
	})

	t.Run("paginate filtered results", func(t *testing.T) {
		firstPage := listCars(t, client, url.Values{
			"name":  {uniqueName},
			"page":  {"1"},
			"limit": {"1"},
		})
		secondPage := listCars(t, client, url.Values{
			"name":  {uniqueName},
			"page":  {"2"},
			"limit": {"1"},
		})

		if firstPage.Total != 2 || secondPage.Total != 2 {
			t.Fatalf("expected filtered total 2, got %d and %d", firstPage.Total, secondPage.Total)
		}
		if len(firstPage.Data) != 1 || len(secondPage.Data) != 1 {
			t.Fatalf("expected one car per page, got %d and %d", len(firstPage.Data), len(secondPage.Data))
		}
		if firstPage.Data[0].ID == secondPage.Data[0].ID {
			t.Fatal("expected different cars on consecutive pages")
		}
	})

	t.Run("reject invalid filters", func(t *testing.T) {
		invalidQueries := []url.Values{
			{"year": {"1800"}},
			{"min_price": {"-1"}},
			{"min_price": {"20"}, "max_price": {"10"}},
			{"created_from": {"not-a-time"}},
			{
				"created_from": {"2026-02-01T00:00:00Z"},
				"created_to":   {"2026-01-01T00:00:00Z"},
			},
		}

		for _, query := range invalidQueries {
			requestJSON(t, client, http.MethodGet, "/cars?"+query.Encode(), nil, http.StatusBadRequest, nil)
		}
	})

	t.Run("delete and return not found", func(t *testing.T) {
		requestJSON(t, client, http.MethodDelete, "/cars/"+second.ID, nil, http.StatusNoContent, nil)
		requestJSON(t, client, http.MethodGet, "/cars/"+second.ID, nil, http.StatusNotFound, nil)
	})
}

func TestIntegrationPriceIsRequiredAndExact(t *testing.T) {
	waitForServer(t)
	client := &http.Client{Timeout: 5 * time.Second}
	base := `{"brand":"PriceTest","model":"Exact","production_year":2024,"color":"Blue"`
	for _, suffix := range []string{
		`}`,
		`,"price":null}`,
		`,"price":1.00000000000000001}`,
		`,"price":10000000000}`,
	} {
		requestJSON(t, client, http.MethodPost, "/cars", json.RawMessage(base+suffix), http.StatusBadRequest, nil)
	}
	var created carResponse
	requestJSON(t, client, http.MethodPost, "/cars", json.RawMessage(base+`,"price":0.29}`), http.StatusCreated, &created)
	t.Cleanup(func() { deleteCar(t, client, created.ID) })
	if created.Price.String() != "0.29" {
		t.Fatalf("price changed on create: %s", created.Price)
	}
	requestJSON(t, client, http.MethodPut, "/cars/"+created.ID, json.RawMessage(base+`,"price":1.00000000000000001}`), http.StatusBadRequest, nil)
	var unchanged carResponse
	requestJSON(t, client, http.MethodGet, "/cars/"+created.ID, nil, http.StatusOK, &unchanged)
	if unchanged.Price.String() != "0.29" {
		t.Fatalf("invalid update changed price: %s", unchanged.Price)
	}
	var updated carResponse
	requestJSON(t, client, http.MethodPut, "/cars/"+created.ID, json.RawMessage(base+`,"price":9999999999.99}`), http.StatusOK, &updated)
	if updated.Price.String() != "9999999999.99" {
		t.Fatalf("price lost precision at upper bound: %s", updated.Price)
	}
}

func createCar(t *testing.T, client *http.Client, payload carPayload) carResponse {
	t.Helper()

	var car carResponse
	requestJSON(t, client, http.MethodPost, "/cars", payload, http.StatusCreated, &car)
	if car.ID == "" {
		t.Fatal("create response did not contain a car ID")
	}
	if _, err := time.Parse(time.RFC3339Nano, car.CreatedAt); err != nil {
		t.Fatalf("create response contains invalid created_at %q: %v", car.CreatedAt, err)
	}
	return car
}

func deleteCar(t *testing.T, client *http.Client, id string) {
	t.Helper()
	requestJSON(t, client, http.MethodDelete, "/cars/"+id, nil, http.StatusNoContent, nil)
}

func listCars(t *testing.T, client *http.Client, query url.Values) carListResponse {
	t.Helper()

	var result carListResponse
	requestJSON(t, client, http.MethodGet, "/cars?"+query.Encode(), nil, http.StatusOK, &result)
	return result
}

func requestJSON(
	t *testing.T,
	client *http.Client,
	method string,
	path string,
	body any,
	wantStatus int,
	response any,
) {
	t.Helper()

	var requestBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("encode request body: %v", err)
		}
		requestBody = bytes.NewReader(encoded)
	}

	request, err := http.NewRequest(method, baseURL+path, requestBody)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	httpResponse, err := client.Do(request)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer httpResponse.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(httpResponse.Body, 1<<20))
	if err != nil {
		t.Fatalf("read %s %s response: %v", method, path, err)
	}
	if httpResponse.StatusCode != wantStatus {
		t.Fatalf(
			"%s %s: expected status %d, got %d: %s",
			method,
			path,
			wantStatus,
			httpResponse.StatusCode,
			responseBody,
		)
	}
	if response != nil {
		if err := json.Unmarshal(responseBody, response); err != nil {
			t.Fatalf("decode %s %s response: %v; body: %s", method, path, err, responseBody)
		}
	}
}

func assertCar(
	t *testing.T,
	car carResponse,
	id string,
	brand string,
	model string,
	year int16,
	price string,
) {
	t.Helper()

	if car.ID != id || car.Brand != brand || car.Model != model || car.ProductionYear != year {
		t.Fatalf("unexpected car response: %+v", car)
	}
	actualPrice, err := strconv.ParseFloat(car.Price.String(), 64)
	if err != nil {
		t.Fatalf("parse response price %q: %v", car.Price, err)
	}
	wantPrice, err := strconv.ParseFloat(price, 64)
	if err != nil {
		t.Fatalf("parse expected price %q: %v", price, err)
	}
	if actualPrice != wantPrice {
		t.Fatalf("expected price %s, got %s", price, car.Price)
	}
}

func assertListContainsOnly(t *testing.T, result carListResponse, ids ...string) {
	t.Helper()

	if result.Total != int64(len(ids)) {
		t.Fatalf("expected total %d, got %d", len(ids), result.Total)
	}
	if len(result.Data) != len(ids) {
		t.Fatalf("expected %d cars, got %d", len(ids), len(result.Data))
	}

	wanted := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		wanted[id] = struct{}{}
	}
	for _, car := range result.Data {
		if _, ok := wanted[car.ID]; !ok {
			t.Fatalf("unexpected car %s in filtered response", car.ID)
		}
		delete(wanted, car.ID)
	}
	if len(wanted) != 0 {
		t.Fatalf("filtered response is missing car IDs: %v", wanted)
	}
}
