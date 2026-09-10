package handler

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestListCarsParams(t *testing.T) {
	request := httptest.NewRequest(
		"GET",
		"/cars?name=BMW+X5&year=2023&created_from=2026-01-01T00%3A00%3A00Z&created_to=2026-12-31T23%3A59%3A59Z&min_price=10000.50&max_price=60000&page=2&limit=10",
		nil,
	)

	params, page, limit, err := listCarsParams(request)
	if err != nil {
		t.Fatal(err)
	}
	if page != 2 || limit != 10 || params.OffsetCount != 10 || params.LimitCount != 10 {
		t.Fatalf("unexpected pagination: page=%d limit=%d offset=%d", page, limit, params.OffsetCount)
	}
	if params.Name != "BMW X5" {
		t.Fatalf("unexpected name filter %q", params.Name)
	}
	if !params.Year.Valid || params.Year.Int16 != 2023 {
		t.Fatalf("unexpected year filter: %+v", params.Year)
	}
	if !params.CreatedFrom.Valid || !params.CreatedTo.Valid {
		t.Fatal("created-time filters were not parsed")
	}
	if !params.MinPrice.Valid || !params.MaxPrice.Valid {
		t.Fatal("price filters were not parsed")
	}
}

func TestListCarsParamsSupportsExactCreatedTimeAndPrice(t *testing.T) {
	request := httptest.NewRequest(
		"GET",
		"/cars?created_at=2026-08-06T12%3A00%3A00Z&price=49500",
		nil,
	)

	params, _, _, err := listCarsParams(request)
	if err != nil {
		t.Fatal(err)
	}
	wantTime := time.Date(2026, time.August, 6, 12, 0, 0, 0, time.UTC)
	if !params.CreatedFrom.Valid || !params.CreatedFrom.Time.Equal(wantTime) ||
		!params.CreatedTo.Valid || !params.CreatedTo.Time.Equal(wantTime) {
		t.Fatal("exact created_at filter was not applied to both time bounds")
	}
	if !params.MinPrice.Valid || !params.MaxPrice.Valid {
		t.Fatal("exact price filter was not applied to both price bounds")
	}
}

func TestListCarsParamsRejectsInvalidRanges(t *testing.T) {
	tests := []string{
		"/cars?year=1800",
		"/cars?created_from=2026-02-01T00%3A00%3A00Z&created_to=2026-01-01T00%3A00%3A00Z",
		"/cars?min_price=20&max_price=10",
		"/cars?min_price=-1",
		"/cars?page=214748365",
	}

	for _, target := range tests {
		t.Run(target, func(t *testing.T) {
			request := httptest.NewRequest("GET", target, nil)
			if _, _, _, err := listCarsParams(request); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
