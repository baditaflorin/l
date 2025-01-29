package main

import (
	"github.com/baditaflorin/l"
	"github.com/baditaflorin/l/example/web-service/handlers"
	"github.com/baditaflorin/l/example/web-service/middleware"
	"net/http"
	"os"
)

func main() {
	// Setup logger with JSON format and source info
	err := l.Setup(l.Options{
		Output:     os.Stdout,
		JsonFormat: true,
		AddSource:  true,
		Metrics:    true,
	})
	if err != nil {
		panic(err)
	}
	defer l.Close()

	// Create router and add logging middleware
	mux := http.NewServeMux()
	mux.Handle("/users", middleware.LogRequest(handlers.UsersHandler()))

	l.Info("Starting web service", "port", 8080)
	if err := http.ListenAndServe(":8080", mux); err != nil {
		l.Error("Server failed", "error", err)
	}
}
