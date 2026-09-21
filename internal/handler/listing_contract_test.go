package handler_test

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestListingRequestErrorsDoNotReachPersistence(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"null object", "null", 400},
		{"trailing object", validCarJSON + ` {}`, 400},
		{"wrong optional type", strings.TrimSuffix(validCarJSON, "}") + `,"image":12}`, 400},
		{"unsafe URL", strings.TrimSuffix(validCarJSON, "}") + `,"image":"javascript:alert(1)"}`, 400},
		{"invalid database text", strings.TrimSuffix(validCarJSON, "}") + `,"description":"bad\u0000text"}`, 400},
		{"long generation", strings.TrimSuffix(validCarJSON, "}") + `,"model_generation":"` + strings.Repeat("x", 101) + `"}`, 400},
		{"large body", strings.TrimSuffix(validCarJSON, "}") + `,"description":"` + strings.Repeat("x", 128<<10) + `"}`, 413},
	} {
		for _, method := range []string{"POST", "PUT"} {
			t.Run(method+"/"+tc.name, func(t *testing.T) {
				store := &apiStore{}
				path := "/cars"
				if method == "PUT" {
					path += "/" + uuid.NewString()
				}
				w := httptest.NewRecorder()
				api(t, store).ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(tc.body)))
				assertAPIError(t, w, tc.status)
				if store.calls != 0 {
					t.Fatal("invalid request reached persistence")
				}
			})
		}
	}
}
