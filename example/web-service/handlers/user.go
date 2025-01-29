package handlers

import (
	"github.com/baditaflorin/l"
	"net/http"
)

func UsersHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l.Info("Processing user request",
			"method", r.Method,
			"path", r.URL.Path,
		)

		// Simulate error
		if r.Method == "POST" {
			l.Error("Invalid user data",
				"status", 400,
				"client_ip", r.RemoteAddr,
			)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
