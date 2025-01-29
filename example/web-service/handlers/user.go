package handlers

import (
	"github.com/baditaflorin/l"
	"net/http"
)

func UsersHandler(logger l.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create handler-specific logger with request context
		reqLogger := logger.With(
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)

		reqLogger.Info("Processing user request")

		// Simulate error
		if r.Method == "POST" {
			reqLogger.Error("Invalid user data",
				"status", 400,
			)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
