// l/middleware.go
package l

import (
	"fmt"
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture status and size.
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

// RequestLogger returns middleware that logs HTTP requests.
func (l *StandardLogger) RequestLogger() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := newResponseWriter(w)
			// Process request.
			next.ServeHTTP(rw, r)
			duration := time.Since(start)
			l.WithFields(map[string]interface{}{
				"method":      r.Method,
				"path":        r.URL.Path,
				"status":      rw.status,
				"duration_ms": duration.Milliseconds(),
				"size":        rw.size,
			}).Info("Request completed")
		})
	}
}

// RecoveryMiddleware returns middleware that recovers from panics.
func (l *StandardLogger) RecoveryMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					l.LogError(r.Context(), fmt.Errorf("panic: %v", rec))
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
