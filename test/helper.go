package test

import (
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

var baseURL = func() string {
	if value := strings.TrimRight(os.Getenv("CAR_SERVICE_BASE_URL"), "/"); value != "" {
		return value
	}
	return "http://localhost:8080"
}()

func waitForServer(t *testing.T) {
	t.Helper()

	client := &http.Client{
		Timeout: time.Second,
	}

	for i := 0; i < 30; i++ {
		resp, err := client.Get(baseURL + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}

		time.Sleep(time.Second)
	}

	t.Fatalf("server at %s is not ready", baseURL)
}
