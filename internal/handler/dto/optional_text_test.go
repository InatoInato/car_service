package dto

import (
	"encoding/json"
	"testing"
)

func TestOptionalTextRoundTrip(t *testing.T) {
	for _, body := range []string{`{}`, `{"image":null}`, `{"image":""}`, `{"image":"https://example.com/car.jpg"}`} {
		var req CreateCarRequest
		if err := json.Unmarshal([]byte(body), &req); err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(req)
		if err != nil {
			t.Fatal(err)
		}
		var got, want map[string]json.RawMessage
		if err = json.Unmarshal(encoded, &got); err != nil {
			t.Fatal(err)
		}
		if err = json.Unmarshal([]byte(body), &want); err != nil {
			t.Fatal(err)
		}
		if string(got["image"]) != string(want["image"]) {
			t.Fatalf("presence changed: %s -> %s", body, encoded)
		}
	}
}
