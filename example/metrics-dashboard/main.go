package main

import (
	"fmt"
	"github.com/baditaflorin/l"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
)

// Global logger and metrics collector
var (
	logger   l.Logger
	reporter *MetricsReporter
)

// MetricsReporter wraps the metrics functionality
type MetricsReporter struct {
	logger           l.Logger
	metricsCollector l.MetricsCollector
	stopChan         chan struct{}
	mu               sync.RWMutex
}

func NewMetricsReporter(factory l.Factory, config l.Config) (*MetricsReporter, error) {
	logger, err := factory.CreateLogger(config)
	if err != nil {
		return nil, err
	}

	metricsCollector, err := factory.CreateMetricsCollector(config)
	if err != nil {
		return nil, err
	}

	return &MetricsReporter{
		logger:           logger,
		metricsCollector: metricsCollector,
		stopChan:         make(chan struct{}),
	}, nil
}

func (r *MetricsReporter) Start() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.reportMetrics()
		case <-r.stopChan:
			return
		}
	}
}

func (r *MetricsReporter) Stop() {
	close(r.stopChan)
}

func (r *MetricsReporter) reportMetrics() {
	metrics := r.metricsCollector.GetMetrics()
	r.logger.Info("Current metrics",
		"total_messages", metrics.TotalMessages,
		"error_messages", metrics.ErrorMessages,
		"dropped_messages", metrics.DroppedMessages,
	)
}

func (r *MetricsReporter) HandleMetrics(w http.ResponseWriter, req *http.Request) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	metrics := r.metricsCollector.GetMetrics()
	fmt.Fprintf(w, `{
        "total_messages": %d,
        "error_messages": %d,
        "dropped_messages": %d,
        "last_error": "%s",
        "last_flush": "%s"
    }`,
		metrics.TotalMessages,
		metrics.ErrorMessages,
		metrics.DroppedMessages,
		metrics.LastError.Format(time.RFC3339),
		metrics.LastFlush.Format(time.RFC3339))
}

// generateTestData simulates different types of log events
func generateTestData(logger l.Logger) {
	events := []struct {
		name     string
		severity string
		chance   float64
	}{
		{"Login attempt", "info", 0.7},
		{"Database query", "info", 0.8},
		{"API request", "info", 0.9},
		{"Authentication failure", "error", 0.1},
		{"Database timeout", "error", 0.05},
		{"Memory warning", "warn", 0.15},
	}

	for {
		for _, event := range events {
			if rand.Float64() < event.chance {
				switch event.severity {
				case "info":
					logger.Info(event.name,
						"user_id", fmt.Sprintf("user_%d", rand.Intn(1000)),
						"timestamp", time.Now(),
					)
				case "error":
					logger.Error(event.name,
						"error_code", rand.Intn(500),
						"timestamp", time.Now(),
					)
				case "warn":
					logger.Warn(event.name,
						"resource", fmt.Sprintf("resource_%d", rand.Intn(10)),
						"timestamp", time.Now(),
					)
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func main() {
	// Initialize the logger and metrics
	factory := l.NewStandardFactory()
	config := l.Config{
		Output:     os.Stdout,
		JsonFormat: true,
		Metrics:    true,
	}

	var err error
	reporter, err = NewMetricsReporter(factory, config)
	if err != nil {
		panic(err)
	}

	// Start the metrics reporter
	go reporter.Start()
	defer reporter.Stop()

	// Start generating test data in the background
	go generateTestData(reporter.logger)

	// Setup HTTP endpoints
	http.HandleFunc("/metrics", reporter.HandleMetrics)

	// Start the server
	fmt.Println("Server starting on :8080")
	fmt.Println("\nTry these curl commands:")
	fmt.Println("1. Basic metrics query:")
	fmt.Println("   curl http://localhost:8080/metrics")
	fmt.Println("\n2. Watch metrics in real-time:")
	fmt.Println("   watch -n 2 'curl -s http://localhost:8080/metrics | jq \".\"'")
	fmt.Println("\n3. Save metrics to file:")
	fmt.Println("   curl -o metrics.json http://localhost:8080/metrics")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
}

/*
To use this example:

1. Save this file as metrics_dashboard.go

2. Run the server:
   go run metrics_dashboard.go

3. In another terminal, try these curl commands:

   # Basic metrics query
   curl http://localhost:8080/metrics

   # Pretty print with jq (if installed)
   curl -s http://localhost:8080/metrics | jq '.'

   # Watch metrics in real-time
   watch -n 2 'curl -s http://localhost:8080/metrics | jq "."'

   # Save metrics to file
   curl -o metrics.json http://localhost:8080/metrics

   # Get metrics with headers
   curl -i http://localhost:8080/metrics

The server will automatically generate various types of log events with different
severities, allowing you to see the metrics change in real-time. The test data
generator creates:
- Regular info logs (high frequency)
- Error logs (low frequency)
- Warning logs (medium frequency)

This helps demonstrate how the metrics dashboard tracks different types of log
events and how they accumulate over time.
*/
