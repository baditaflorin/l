package main

import (
	"context"
	"github.com/baditaflorin/l"
	"sync"
	"time"
)

func main() {
	// Configure for high throughput with larger buffer
	err := l.Setup(l.Options{
		//Output:     os.Stdout,
		FilePath:    "high-throughput/high-throughput.log",
		MaxFileSize: 10 * 1024 * 1024, // 10MB
		JsonFormat:  true,
		AsyncWrite:  true,
		BufferSize:  1024 * 1024, // Increase buffer size to 1MB
	})
	if err != nil {
		panic(err)
	}
	defer l.Close()

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

	// WaitGroup for all goroutines (workers + metrics)
	var allDone sync.WaitGroup

	// Start metrics monitoring with proper cleanup
	allDone.Add(1)
	metricsStop := make(chan struct{})
	go func() {
		defer allDone.Done()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				metrics := l.GetMetrics()
				l.Info("Logging metrics",
					"total_messages", metrics.TotalMessages.Load(),
					"dropped_messages", metrics.DroppedMessages.Load(),
					"error_messages", metrics.ErrorMessages.Load(),
				)
			case <-metricsStop:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	// Start worker goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		allDone.Add(1)
		go func(routineID int) {
			defer wg.Done()
			defer allDone.Done()

			for j := 0; j < logsPerRoutine; j++ {
				select {
				case semaphore <- struct{}{}:
					l.Info("High throughput test",
						"routine_id", routineID,
						"count", j,
						"data", generateLargePayload(10),
					)
					<-semaphore
				case <-ctx.Done():
					return
				}
			}
		}(i)
	}

	// Wait for all worker goroutines
	wg.Wait()

	// Log completion metrics
	duration := time.Since(start)
	l.Info("Test completed",
		"duration_seconds", duration.Seconds(),
		"messages_per_second", float64(numGoroutines*logsPerRoutine)/duration.Seconds(),
	)

	// Stop metrics monitoring
	close(metricsStop)

	// Wait for all goroutines to finish
	allDone.Wait()
}

func generateLargePayload(size int) string {
	payload := make([]byte, size)
	for i := range payload {
		payload[i] = 'A' + byte(i%26)
	}
	return string(payload)
}
