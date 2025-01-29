package main

import (
	"github.com/baditaflorin/l"
	"github.com/baditaflorin/l/example/web-service/handlers"
	"github.com/baditaflorin/l/example/web-service/middleware"
	"net/http"
	"os"
)

func main() {
	factory := l.NewStandardFactory()
	config := l.Config{
		Output:     os.Stdout,
		JsonFormat: true,
		AddSource:  true,
		Metrics:    true,
	}

	logger, err := factory.CreateLogger(config)
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	// Create router and add logging middleware
	mux := http.NewServeMux()
	mux.Handle("/users", middleware.LogRequest(logger)(handlers.UsersHandler(logger)))

	logger.Info("Starting web service", "port", 8080)
	if err := http.ListenAndServe(":8080", mux); err != nil {
		logger.Error("Server failed", "error", err)
	}
}
