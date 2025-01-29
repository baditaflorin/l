// File: tests/recovery_test.go
package tests

import (
	"bytes"
	"github.com/baditaflorin/l"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestLoggerRecovery(t *testing.T) {
	var buf bytes.Buffer
	config := l.Config{
		Output:     &buf,
		AsyncWrite: true,
		Metrics:    true,
	}

	factory := l.NewStandardFactory()
	logger, err := factory.CreateLogger(config)
	require.NoError(t, err)

	// Test recovery from panics in different scenarios
	t.Run("Recover from panic in logging goroutine", func(t *testing.T) {
		buf.Reset()
		panicHandled := make(chan struct{})

		defer func() {
			if r := recover(); r != nil {
				t.Error("Logger should handle panics gracefully")
			}
		}()

		go func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("Recovered from panic", "error_type", "test_panic")
					// Ensure log is written
					err := logger.Flush()
					assert.NoError(t, err)
					close(panicHandled)
				}
			}()
			panic("test panic")
		}()

		// Wait for panic to be handled or timeout
		select {
		case <-panicHandled:
			// Success - panic was handled
			// Give a small amount of time for flush to complete
			time.Sleep(50 * time.Millisecond)
			output := buf.String()
			assert.Contains(t, output, "test_panic", "Expected panic to be logged")
			assert.Contains(t, output, "Recovered from panic", "Expected recovery message")
		case <-time.After(time.Second):
			t.Error("Timeout waiting for panic recovery")
		}
	})

	t.Run("Recover from buffer overflow", func(t *testing.T) {
		var overflowBuf bytes.Buffer
		smallConfig := l.Config{
			Output:     &overflowBuf,
			AsyncWrite: true,
			BufferSize: 10,
			Metrics:    true,
		}

		smallLogger, err := factory.CreateLogger(smallConfig)
		require.NoError(t, err)

		// Try to overwhelm the buffer with a smaller number of messages
		msgCount := 50
		for i := 0; i < msgCount; i++ {
			smallLogger.Info("overflow test", "test_iteration", i)
			// Periodically flush to ensure some messages get through
			if i%10 == 0 {
				err := smallLogger.Flush()
				assert.NoError(t, err)
			}
		}

		// Final flush to ensure all messages are written
		err = smallLogger.Flush()
		assert.NoError(t, err)

		// Give some time for async operations to complete
		time.Sleep(50 * time.Millisecond)

		// Logger should not crash and should have written something
		err = smallLogger.Close()
		assert.NoError(t, err)

		output := overflowBuf.String()
		assert.Greater(t, len(output), 0, "Expected some output from overflow test")
		assert.Contains(t, output, "overflow test", "Expected test messages to be logged")
	})

	t.Run("Recovery with metrics", func(t *testing.T) {
		var metricsBuf bytes.Buffer
		metricsConfig := l.Config{
			Output:     &metricsBuf,
			AsyncWrite: true,
			Metrics:    true,
		}

		metricsLogger, err := factory.CreateLogger(metricsConfig)
		require.NoError(t, err)

		metrics, err := factory.CreateMetricsCollector(metricsConfig)
		require.NoError(t, err)

		initialErrors := metrics.GetMetrics().ErrorMessages

		// Trigger a recoverable error
		metricsLogger.Error("intentional error for testing")

		// Ensure error is logged
		err = metricsLogger.Flush()
		assert.NoError(t, err)

		// Give metrics time to update
		time.Sleep(50 * time.Millisecond)

		// Verify error was counted
		currentErrors := metrics.GetMetrics().ErrorMessages
		assert.Greater(t, currentErrors, initialErrors, "Error count should increment")

		// Clean up
		err = metricsLogger.Close()
		assert.NoError(t, err)
	})

	// Clean up main logger
	err = logger.Close()
	assert.NoError(t, err)
}
