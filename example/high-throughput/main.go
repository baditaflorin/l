package main

import (
	"context"
	"github.com/baditaflorin/l"
	"sync"
	"time"
)

func main() {
	// Create factory and config
	factory := l.NewStandardFactory()
	config := l.Config{
		FilePath:    "high-throughput/high-throughput.log",
		MaxFileSize: 100 * 1024 * 1024, // 100MB
		JsonFormat:  true,
		AsyncWrite:  true,
		BufferSize:  1024 * 1024, // 1MB buffer
		MinLevel:    l.LevelInfo,
		AddSource:   true,
		MaxBackups:  5,
		Metrics:     true,
	}

	// Create logger
	logger, err := factory.CreateLogger(config)
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	// Create metrics collector
	metricsCollector, err := factory.CreateMetricsCollector(config)
	if err != nil {
		panic(err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const (
		numGoroutines  = 100
		logsPerRoutine = 10000
	)

	var wg sync.WaitGroup
	start := time.Now()

	// Channel for backpressure
	semaphore := make(chan struct{}, numGoroutines*2)

	// Start worker goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()

			routineLogger := logger.With("routine_id", routineID)

			for j := 0; j < logsPerRoutine; j++ {
				select {
				case semaphore <- struct{}{}:
					routineLogger.Info("High throughput test",
						"count", j,
						"data", generateLargePayload(10),
					)
					metricsCollector.IncrementTotal()
					<-semaphore
				case <-ctx.Done():
					return
				}
			}
		}(i)
	}

	// Monitor metrics
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				metrics := metricsCollector.GetMetrics()
				logger.Info("Logging metrics",
					"total_messages", metrics.TotalMessages,
					"dropped_messages", metrics.DroppedMessages,
					"error_messages", metrics.ErrorMessages,
				)
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for all worker goroutines
	wg.Wait()

	// Log completion metrics
	duration := time.Since(start)
	logger.Info("Test completed",
		"duration_seconds", duration.Seconds(),
		"messages_per_second", float64(numGoroutines*logsPerRoutine)/duration.Seconds(),
	)

	// Ensure all logs are written
	if err := logger.Flush(); err != nil {
		logger.Error("Failed to flush logs", "error", err)
	}
}

func generateLargePayload(size int) string {
	payload := make([]byte, size)
	for i := range payload {
		payload[i] = 'A' + byte(i%26)
	}
	return string(payload)
}
