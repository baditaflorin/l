package examples

import (
	"github.com/baditaflorin/l"
	"github.com/baditaflorin/l/example/web-service/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLogMiddleware(t *testing.T) {
	// Create logger for testing
	factory := l.NewStandardFactory()
	config := l.Config{
		JsonFormat: true,
	}

	var buf strings.Builder
	config.Output = io.MultiWriter(&buf, io.Discard)

	logger, err := factory.CreateLogger(config)
	require.NoError(t, err)
	defer logger.Close()

	t.Run("Basic Request Logging", func(t *testing.T) {
		// Create a test handler
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("test response"))
		})

		// Wrap with logging middleware
		loggedHandler := middleware.LogRequest(logger)(handler)

		// Create test request
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		// Handle request
		loggedHandler.ServeHTTP(rec, req)

		// Verify response
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "test response", rec.Body.String())

		// Verify logs contain required fields
		logs := buf.String()
		assert.Contains(t, logs, "Request started")
		assert.Contains(t, logs, "Request completed")
		assert.Contains(t, logs, "GET")
		assert.Contains(t, logs, "/test")
	})

	t.Run("Request with Error", func(t *testing.T) {
		// Create a handler that returns an error
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		loggedHandler := middleware.LogRequest(logger)(handler)

		req := httptest.NewRequest("POST", "/error", nil)
		rec := httptest.NewRecorder()

		loggedHandler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)

		logs := buf.String()
		assert.Contains(t, logs, "POST")
		assert.Contains(t, logs, "/error")
	})

	t.Run("Request Duration Logging", func(t *testing.T) {
		// Create a slow handler
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		})

		loggedHandler := middleware.LogRequest(logger)(handler)

		req := httptest.NewRequest("GET", "/slow", nil)
		rec := httptest.NewRecorder()

		loggedHandler.ServeHTTP(rec, req)

		logs := buf.String()
		assert.Contains(t, logs, "duration_ms")
		assert.Contains(t, logs, "/slow")
	})

	t.Run("Request with Custom Headers", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		loggedHandler := middleware.LogRequest(logger)(handler)

		req := httptest.NewRequest("GET", "/headers", nil)
		req.Header.Set("X-Request-ID", "test-123")
		req.Header.Set("User-Agent", "test-agent")
		rec := httptest.NewRecorder()

		loggedHandler.ServeHTTP(rec, req)

		logs := buf.String()
		assert.Contains(t, logs, "/headers")
		assert.Contains(t, logs, "GET")
	})

	t.Run("Concurrent Requests", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		loggedHandler := middleware.LogRequest(logger)(handler)

		// Make multiple concurrent requests
		for i := 0; i < 10; i++ {
			go func() {
				req := httptest.NewRequest("GET", "/concurrent", nil)
				rec := httptest.NewRecorder()
				loggedHandler.ServeHTTP(rec, req)
				assert.Equal(t, http.StatusOK, rec.Code)
			}()
		}

		// Give time for requests to complete
		time.Sleep(100 * time.Millisecond)

		logs := buf.String()
		assert.Contains(t, logs, "/concurrent")
		assert.Contains(t, logs, "GET")
	})
}
