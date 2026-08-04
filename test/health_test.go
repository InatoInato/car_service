package test

import (
	"net/http"
	"testing"
)

func TestHealth(t *testing.T) {
	waitForServer(t)

	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}