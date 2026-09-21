package handler

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/InatoInato/car_service.git/internal/handler/dto"
)

func TestOptionalListingFields(t *testing.T) {
	for _, tc := range []struct {
		name, json     string
		present, valid bool
	}{
		{"omitted", `{}`, false, false},
		{"null", `{"image":null,"model_generation":null,"description":null}`, true, false},
		{"blank", `{"image":" ","model_generation":"","description":"\n\t"}`, true, false},
		{"values", `{"image":" https://example.com/car.jpg ","model_generation":" W124 facelift ","description":"Service history\nAvailable"}`, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var req dto.CreateCarRequest
			if err := json.Unmarshal([]byte(tc.json), &req); err != nil {
				t.Fatal(err)
			}
			if req.Image.Present != tc.present || req.ModelGeneration.Present != tc.present || req.Description.Present != tc.present {
				t.Fatal("JSON presence lost")
			}
			p, err := createCarParams(req)
			if err != nil {
				t.Fatal(err)
			}
			if p.Image.Valid != tc.valid || p.ModelGeneration.Valid != tc.valid || p.Description.Valid != tc.valid {
				t.Fatalf("incorrect NULL mapping: %+v", p)
			}
			if tc.valid && (p.Image.String != "https://example.com/car.jpg" || p.ModelGeneration.String != "W124 facelift" || p.Description.String != "Service history\nAvailable") {
				t.Fatalf("incorrect values: %+v", p)
			}
		})
	}
}

func TestOptionalListingFieldsRejectInvalidInput(t *testing.T) {
	for _, body := range []string{`{"image":42}`, `{"description":{}}`, `{"model_generation":[]}`} {
		var req dto.CreateCarRequest
		if err := json.Unmarshal([]byte(body), &req); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
	for _, value := range []string{"javascript:alert(1)", "data:image/png;base64,a", "file:///tmp/a.jpg", "/tmp/a.jpg", "https://user:secret@example.com/a", "https://example.com/a b", "https://example.com/a#fragment", "https://", strings.Repeat("x", 2049)} {
		req := dto.CreateCarRequest{Image: dto.OptionalText{Present: true, Value: &value}}
		if _, err := parseListingDetails(req); err == nil {
			t.Fatalf("accepted image %q", value)
		}
	}
	for _, field := range []string{"description", "model_generation"} {
		for _, value := range []string{strings.Repeat("界", 10001), "bad\x00text"} {
			body, _ := json.Marshal(map[string]string{field: value})
			var req dto.CreateCarRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatal(err)
			}
			if _, err := parseListingDetails(req); err == nil {
				t.Fatalf("accepted invalid %s", field)
			}
		}
	}
}
