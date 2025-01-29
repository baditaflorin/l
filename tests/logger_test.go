// File: tests/logger_test.go
package tests

import (
	"bytes"
	"encoding/json"
	"github.com/baditaflorin/l"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestLoggerBasicFunctionality(t *testing.T) {
	var buf bytes.Buffer
	factory := l.NewStandardFactory()
	config := l.Config{
		Output:     &buf,
		JsonFormat: true,
		AddSource:  false,
	}

	logger, err := factory.CreateLogger(config)
	require.NoError(t, err)
	require.NotNil(t, logger)
	defer func() {
		err := logger.Flush()
		assert.NoError(t, err)
		err = logger.Close()
		assert.NoError(t, err)
	}()

	testCases := []struct {
		name     string
		logFunc  func(string, ...any)
		message  string
		level    string
		extraKey string
		extraVal interface{}
	}{
		{
			name:     "Info logging",
			logFunc:  logger.Info,
			message:  "test info message",
			level:    "INFO",
			extraKey: "user_id",
			extraVal: "123",
		},
		{
			name:     "Error logging",
			logFunc:  logger.Error,
			message:  "test error message",
			level:    "ERROR",
			extraKey: "error_code",
			extraVal: float64(500),
		},
		{
			name:    "Warning logging",
			logFunc: logger.Warn,
			message: "" +
				"test warning message",
			level:    "WARN",
			extraKey: "warning_type",
			extraVal: "resource_low",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf.Reset()
			tc.logFunc(tc.message, tc.extraKey, tc.extraVal)

			// Ensure logs are flushed after each test case
			err := logger.Flush()
			require.NoError(t, err)

			var logEntry map[string]interface{}
			err = json.Unmarshal(buf.Bytes(), &logEntry)
			require.NoError(t, err)

			assert.Equal(t, tc.message, logEntry["msg"])
			assert.Equal(t, tc.level, logEntry["level"])
			assert.Equal(t, tc.extraVal, logEntry[tc.extraKey])
			assert.NotEmpty(t, logEntry["time"])
		})
	}
}

func TestLoggerWithContext(t *testing.T) {
	var buf bytes.Buffer
	factory := l.NewStandardFactory()
	config := l.Config{
		Output:     &buf,
		JsonFormat: true,
	}

	logger, err := factory.CreateLogger(config)
	require.NoError(t, err)
	defer func() {
		err := logger.Flush()
		assert.NoError(t, err)
		err = logger.Close()
		assert.NoError(t, err)
	}()

	contextLogger := logger.With(
		"service", "test-service",
		"environment", "testing",
	)

	contextLogger.Info("test message",
		"request_id", "req123",
	)

	err = logger.Flush()
	require.NoError(t, err)

	var logEntry map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "test-service", logEntry["service"])
	assert.Equal(t, "testing", logEntry["environment"])
	assert.Equal(t, "req123", logEntry["request_id"])
}

func TestLoggerFlushAndClose(t *testing.T) {
	var buf bytes.Buffer
	factory := l.NewStandardFactory()
	config := l.Config{
		Output:     &buf,
		JsonFormat: true,
		AsyncWrite: true,
		BufferSize: 1024,
	}

	logger, err := factory.CreateLogger(config)
	require.NoError(t, err)

	// Write some logs
	const numMessages = 100
	for i := 0; i < numMessages; i++ {
		logger.Info("test message", "index", i)
	}

	// Give async writes some time to process
	time.Sleep(50 * time.Millisecond)

	// Test flush before close
	err = logger.Flush()
	assert.NoError(t, err)

	// Give the flush operation time to complete
	time.Sleep(50 * time.Millisecond)

	// Verify logs were written
	logContent := buf.String()
	assert.NotEmpty(t, logContent, "Buffer should contain logs")

	// Count the number of valid JSON entries
	logLines := bytes.Split(buf.Bytes(), []byte("\n"))
	var validEntries int
	for _, line := range logLines {
		if len(line) > 0 {
			var entry map[string]interface{}
			if err := json.Unmarshal(line, &entry); err == nil {
				validEntries++
			}
		}
	}
	assert.Equal(t, numMessages, validEntries,
		"Expected %d log entries, got %d", numMessages, validEntries)

	// Close the logger
	err = logger.Close()
	assert.NoError(t, err)

	// After close, the logger should handle writes gracefully
	// Note: Info() doesn't return an error per the interface
	initialCount := validEntries
	logger.Info("should be handled gracefully", "after", "close")

	// Verify no new logs were written after close
	logLines = bytes.Split(buf.Bytes(), []byte("\n"))
	validEntries = 0
	for _, line := range logLines {
		if len(line) > 0 {
			var entry map[string]interface{}
			if err := json.Unmarshal(line, &entry); err == nil {
				validEntries++
			}
		}
	}
	assert.Equal(t, initialCount, validEntries,
		"No new entries should be written after close")
}

func TestLoggerConcurrentAccess(t *testing.T) {
	var buf bytes.Buffer
	factory := l.NewStandardFactory()
	config := l.Config{
		Output:     &buf,
		JsonFormat: true,
		AsyncWrite: true,
		BufferSize: 10000,
	}

	logger, err := factory.CreateLogger(config)
	require.NoError(t, err)

	concurrency := 10
	messages := 100
	var wg sync.WaitGroup
	wg.Add(concurrency)

	// Channel to signal when all goroutines are done
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// Start workers
	for i := 0; i < concurrency; i++ {
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < messages; j++ {
				logger.Info("concurrent test",
					"worker_id", workerID,
					"message_id", j,
				)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	<-done

	// Give some time for async writes and use multiple flush attempts
	for i := 0; i < 3; i++ {
		time.Sleep(50 * time.Millisecond)
		err = logger.Flush()
		if err != nil {
			t.Logf("Flush attempt %d failed: %v", i+1, err)
		}
	}

	// Close the logger
	err = logger.Close()
	assert.NoError(t, err)

	// Count valid JSON entries
	logLines := bytes.Split(buf.Bytes(), []byte("\n"))
	var validEntries int
	for _, line := range logLines {
		if len(line) > 0 {
			var entry map[string]interface{}
			if err := json.Unmarshal(line, &entry); err == nil {
				validEntries++
			}
		}
	}

	assert.Equal(t, concurrency*messages, validEntries,
		"Expected %d log entries, got %d", concurrency*messages, validEntries)
}
