package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := &responseWriter{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			next.ServeHTTP(rw, r)

			duration := time.Since(start)

			// 1. Lightweight health check log at DEBUG level
			if r.URL.Path == "/health" {
				logger.Debug("health check", "status", rw.status, "duration_ms", duration.Milliseconds())
				return
			}

			// 2. Dynamic log level based on status code
			logFn := logger.Info
			if rw.status >= 500 {
				logFn = logger.Error
			} else if rw.status >= 400 {
				logFn = logger.Warn
			}

			// 3. Compact production API log
			logFn(
				"http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.status,
				"duration_ms", duration.Milliseconds(),
			)
		})
	}
}