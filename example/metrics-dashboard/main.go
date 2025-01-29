package main

import (
	"github.com/baditaflorin/l"
	"github.com/baditaflorin/l/example/metrics-dashboard/metrics"
	"net/http"
)

func main() {
	err := l.Setup(l.Options{
		JsonFormat: true,
		Metrics:    true,
	})
	if err != nil {
		panic(err)
	}
	defer l.Close()

	// Start metrics reporter
	reporter := metrics.NewReporter()
	go reporter.Start()

	// Setup metrics endpoint
	http.HandleFunc("/metrics", reporter.HandleMetrics)

	l.Info("Starting metrics server", "port", 8080)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		l.Error("Server failed", "error", err)
	}
}
