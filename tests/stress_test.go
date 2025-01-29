// File: tests/stress_test.go
package tests

import (
	"bytes"
	"encoding/json"
	"github.com/baditaflorin/l"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLoggerUnderLoad(t *testing.T) {
	var buf bytes.Buffer
	config := l.Config{
		AsyncWrite:  true,
		BufferSize:  1024 * 1024, // 1MB buffer
		JsonFormat:  true,
		MaxFileSize: 10 * 1024 * 1024, // 10MB
		Metrics:     true,
		Output:      &buf,
	}

	factory := l.NewStandardFactory()
	logger, err := factory.CreateLogger(config)
	require.NoError(t, err)

	const (
		numGoroutines = 100 // Number of concurrent goroutines
		numMessages   = 100 // Messages per goroutine
		totalMessages = numGoroutines * numMessages
	)

	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Use atomic counter for tracking successful writes
	var successfulWrites uint64
	var errorCount uint64

	// Create a buffered channel for sending test completion signals
	done := make(chan struct{}, numGoroutines)

	// Run test goroutines
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer func() {
				wg.Done()
				done <- struct{}{}
			}()

			// Recover from any panics in the goroutine
			defer func() {
				if r := recover(); r != nil {
					atomic.AddUint64(&errorCount, 1)
					t.Errorf("Goroutine %d panicked: %v", routineID, r)
				}
			}()

			for j := 0; j < numMessages; j++ {
				// Add some random data to make the test more realistic
				data := map[string]interface{}{
					"msg":       "stress_test",
					"routine":   routineID,
					"iteration": j,
					"timestamp": time.Now().UnixNano(),
				}

				logger.Info("Stress test message",
					"data", data,
					"routine", routineID,
					"iteration", j,
				)

				atomic.AddUint64(&successfulWrites, 1)
			}
		}(i)
	}

	// Wait for all goroutines with timeout
	waitChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitChan)
	}()

	// Set a reasonable timeout
	timeout := time.After(30 * time.Second)

	select {
	case <-waitChan:
		// All goroutines completed successfully
	case <-timeout:
		t.Fatal("Test timed out waiting for goroutines to complete")
	}

	// Calculate duration before flush
	duration := time.Since(start)

	// Ensure all messages are written
	err = logger.Flush()
	assert.NoError(t, err, "Failed to flush logger")

	// Calculate and log performance metrics
	messagesPerSecond := float64(totalMessages) / duration.Seconds()
	t.Logf("Performance: %d messages in %.2f seconds (%.2f msgs/sec)",
		totalMessages, duration.Seconds(), messagesPerSecond)

	// Report any errors
	if errorCount > 0 {
		t.Errorf("Encountered %d errors during test", errorCount)
	}

	// Verify metrics
	metrics, err := factory.CreateMetricsCollector(config)
	require.NoError(t, err, "Failed to create metrics collector")

	actualMessages := metrics.GetMetrics().TotalMessages
	assert.Equal(t, uint64(totalMessages), actualMessages,
		"Expected %d messages, got %d", totalMessages, actualMessages)

	// Verify buffer contains valid JSON
	output := buf.String()
	assert.Greater(t, len(output), 0, "Expected non-empty output")

	// Verify JSON structure of random samples
	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	numSamples := 10
	if len(lines) < numSamples {
		numSamples = len(lines)
	}

	for i := 0; i < numSamples; i++ {
		var logEntry map[string]interface{}
		err = json.Unmarshal(lines[i], &logEntry)
		assert.NoError(t, err, "Failed to parse JSON line %d", i)

		// Verify required fields
		assert.Contains(t, logEntry, "msg", "Log entry should contain 'msg' field")
		assert.Contains(t, logEntry, "data", "Log entry should contain 'data' field")
		assert.Contains(t, logEntry, "routine", "Log entry should contain 'routine' field")
		assert.Contains(t, logEntry, "iteration", "Log entry should contain 'iteration' field")
	}

	// Close logger and verify successful shutdown
	err = logger.Close()
	assert.NoError(t, err, "Failed to close logger")

	// Final verification of total message count
	successfulWritesCount := atomic.LoadUint64(&successfulWrites)
	assert.Equal(t, uint64(totalMessages), successfulWritesCount,
		"Expected %d successful writes, got %d", totalMessages, successfulWritesCount)
}
