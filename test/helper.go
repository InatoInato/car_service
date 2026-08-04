package test

import (
	"net/http"
	"testing"
	"time"
)

const baseURL = "http://localhost:8080"

func waitForServer(t *testing.T) {
	t.Helper()

	client := &http.Client{
		Timeout: time.Second,
	}

	for i := 0; i < 30; i++ {
		resp, err := client.Get(baseURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}

		time.Sleep(time.Second)
	}

	t.Fatal("server is not ready")
}