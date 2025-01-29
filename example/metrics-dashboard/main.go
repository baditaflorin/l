package main

import (
	"github.com/baditaflorin/l"
	"github.com/baditaflorin/l/example/metrics-dashboard/metrics"
	"net/http"
)

func main() {
	factory := l.NewStandardFactory()
	config := l.Config{
		JsonFormat: true,
		Metrics:    true,
	}

	reporter, err := metrics.NewReporter(factory, config)
	if err != nil {
		panic(err)
	}

	// Start metrics reporter
	go reporter.Start()

	// Setup metrics endpoint
	http.HandleFunc("/metrics", reporter.HandleMetrics)

	logger, err := factory.CreateLogger(config)
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	logger.Info("Starting metrics server", "port", 8080)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Error("Server failed", "error", err)
	}
}
