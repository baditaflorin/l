package middleware

import (
	"github.com/baditaflorin/l"
	"net/http"
	"time"
)

func LogRequest(logger l.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create request-scoped logger
			reqLogger := logger.With(
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
			)

			reqLogger.Info("Request started")

			next.ServeHTTP(w, r)

			reqLogger.Info("Request completed",
				"duration_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}
