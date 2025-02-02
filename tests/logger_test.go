// File: tests/logger_test.go
package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/baditaflorin/l"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"path/filepath"
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

// createTestLogger returns a new logger using a bytes.Buffer for output.
// The async flag lets you choose between synchronous and asynchronous writes.
func createTestLogger(async bool) (l.Logger, *bytes.Buffer) {
	buf := new(bytes.Buffer)
	cfg := l.Config{
		Output:      buf,
		JsonFormat:  true,
		AsyncWrite:  async,
		BufferSize:  1024,
		AddSource:   true,
		Environment: "development",
		TimeFormat:  time.RFC3339,
		Level:       l.LevelInfo,
	}
	logger, err := l.NewStandardLogger(l.NewStandardFactory(), cfg)
	if err != nil {
		panic(err)
	}
	return logger, buf
}

// Test 1: Verify that the default logger logs an info message and the output contains the expected text.
func TestDefaultLoggerInfo(t *testing.T) {
	logger, buf := createTestLogger(false)
	logger.Info("test message", "key", "value")
	err := logger.Flush()
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "value")

	logger.Close()
}

// Test 2: Verify that the With() method adds fields correctly.
func TestLoggerWithFields(t *testing.T) {
	logger, _ := createTestLogger(false)
	loggerWithField := logger.With("user", "alice")
	fields := loggerWithField.GetFields()
	assert.Equal(t, "alice", fields["user"])
	logger.Close()
}

//
//// Test 3: Verify that an asynchronous logger flushes correctly.
//func TestFlushAsyncLogger(t *testing.T) {
//	logger, buf := createTestLogger(true)
//	logger.Info("async test message")
//	err := logger.Flush()
//	assert.NoError(t, err)
//
//	output := buf.String()
//	assert.Contains(t, output, "async test message")
//	logger.Close()
//}

// Test 4: Verify that an error log message includes a stack trace (when in development mode).
func TestErrorLoggingIncludesStack(t *testing.T) {
	logger, buf := createTestLogger(false)
	logger.Error("error occurred", "detail", "something went wrong")
	err := logger.Flush()
	assert.NoError(t, err)

	output := buf.String()
	// In non‑production environments we expect the log to include a "stack" field.
	assert.Contains(t, output, "stack")
	logger.Close()
}

// Test 5: Verify that metrics are incremented when logging.
func TestMetricsIncrement(t *testing.T) {
	logger, _ := createTestLogger(false)
	// Retrieve the metrics collector from the underlying StandardLogger.
	stdLogger, ok := logger.(*l.StandardLogger)
	assert.True(t, ok)

	metricsBefore := stdLogger.Metrics.GetMetrics()
	logger.Info("info message")
	logger.Error("error message")
	_ = logger.Flush()
	metricsAfter := stdLogger.Metrics.GetMetrics()

	assert.Equal(t, metricsBefore.TotalMessages+2, metricsAfter.TotalMessages)
	assert.Equal(t, metricsBefore.ErrorMessages+1, metricsAfter.ErrorMessages)
	logger.Close()
}

// Test 10: Verify that the DirectWriter writes data correctly and can be closed.
func TestDirectWriterWriteAndClose(t *testing.T) {
	buf := new(bytes.Buffer)
	writer := l.NewDirectWriter(buf)
	data := []byte("direct write test")
	n, err := writer.Write(data)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)

	err = writer.Close()
	assert.NoError(t, err)
	output := buf.String()
	assert.Equal(t, "direct write test", output)
}

// Test 1: Logging with an empty message should still produce a valid log entry.
func TestLoggingWithEmptyMessage(t *testing.T) {
	var buf bytes.Buffer
	factory := l.NewStandardFactory()
	config := l.Config{
		Output:     &buf,
		JsonFormat: true,
	}
	logger, err := factory.CreateLogger(config)
	require.NoError(t, err)

	logger.Info("")
	err = logger.Flush()
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "", result["msg"])
	logger.Close()
}

// Test 2: Logging using only an error message without extra key–value pairs.
func TestLoggerWithOnlyError(t *testing.T) {
	var buf bytes.Buffer
	factory := l.NewStandardFactory()
	config := l.Config{
		Output:     &buf,
		JsonFormat: true,
	}
	logger, err := factory.CreateLogger(config)
	require.NoError(t, err)

	logger.Error("error occurred")
	err = logger.Flush()
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "error occurred", result["msg"])
	logger.Close()
}

// Test 3: Ensure that concurrent writes produce output (using async writing).
func TestLoggerConcurrentWrites(t *testing.T) {
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

	var wg sync.WaitGroup
	numGoroutines := 20
	messagesPerGoroutine := 50
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				logger.Info("concurrent test", "goroutine", id, "message", j)
			}
		}(i)
	}
	wg.Wait()
	err = logger.Flush()
	require.NoError(t, err)
	// Check that some output was produced.
	assert.Greater(t, buf.Len(), 0)
	logger.Close()
}

//
//// Test 4: Calling Flush() after the logger has been closed should return an error.
//func TestLoggerFlushAfterClose(t *testing.T) {
//	var buf bytes.Buffer
//	factory := l.NewStandardFactory()
//	config := l.Config{
//		Output:     &buf,
//		JsonFormat: true,
//	}
//	logger, err := factory.CreateLogger(config)
//	require.NoError(t, err)
//
//	logger.Info("test message")
//	err = logger.Flush()
//	require.NoError(t, err)
//	err = logger.Close()
//	require.NoError(t, err)
//	err = logger.Flush()
//	assert.Error(t, err, "Flush after close should return an error")
//}

// Test 5: Logging with a nil value in the structured arguments does not cause a crash.
func TestLoggerWithNilArguments(t *testing.T) {
	var buf bytes.Buffer
	factory := l.NewStandardFactory()
	config := l.Config{
		Output:     &buf,
		JsonFormat: true,
	}
	logger, err := factory.CreateLogger(config)
	require.NoError(t, err)

	logger.Info("test nil", "key", nil)
	err = logger.Flush()
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	// Check that "key" is present in the JSON (its value will be null)
	_, ok := result["key"]
	assert.True(t, ok, "Expected key to be present in log entry")
	logger.Close()
}

// Test 6: Chaining logger.With() calls should merge fields from previous chainings.
func TestLoggerChainingWith(t *testing.T) {
	var buf bytes.Buffer
	factory := l.NewStandardFactory()
	config := l.Config{
		Output:     &buf,
		JsonFormat: true,
	}
	logger, err := factory.CreateLogger(config)
	require.NoError(t, err)

	// Add a base field.
	logger = logger.With("base", "value1")
	// Chain an additional field.
	chainedLogger := logger.With("additional", "value2")
	chainedLogger.Info("chained message")
	err = logger.Flush()
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "value1", result["base"])
	assert.Equal(t, "value2", result["additional"])
	logger.Close()
}

// Test 7: JSON-formatted output should include expected keys and values.
func TestLoggerJSONOutputStructure(t *testing.T) {
	var buf bytes.Buffer
	factory := l.NewStandardFactory()
	config := l.Config{
		Output:     &buf,
		JsonFormat: true,
	}
	logger, err := factory.CreateLogger(config)
	require.NoError(t, err)

	logger.Info("json structure test", "user", "testuser", "action", "login")
	err = logger.Flush()
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "json structure test", result["msg"])
	assert.Equal(t, "testuser", result["user"])
	assert.Equal(t, "login", result["action"])
	logger.Close()
}

// Test 8: Text-formatted output should contain the message and key=value pairs.
func TestLoggerTextOutputStructure(t *testing.T) {
	var buf bytes.Buffer
	factory := l.NewStandardFactory()
	config := l.Config{
		Output:     &buf,
		JsonFormat: false,
	}
	logger, err := factory.CreateLogger(config)
	require.NoError(t, err)

	logger.Info("text structure test", "user", "testuser", "action", "logout")
	err = logger.Flush()
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "text structure test")
	assert.Contains(t, output, "user=testuser")
	assert.Contains(t, output, "action=logout")
	logger.Close()
}

// Test 9: Verify that the RotationManager rotates the file when the max size is exceeded.
func TestLoggerRotationManager(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rotation_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	logPath := filepath.Join(tmpDir, "rotating.log")
	config := l.Config{
		FilePath:    logPath,
		JsonFormat:  true,
		MaxFileSize: 200, // Set a small size to force rotation
		MaxBackups:  2,
	}
	factory := l.NewStandardFactory()
	logger, err := factory.CreateLogger(config)
	require.NoError(t, err)

	// Write enough messages to trigger rotation.
	for i := 0; i < 50; i++ {
		logger.Info("rotation test", "iteration", i)
	}
	err = logger.Flush()
	require.NoError(t, err)
	// Look for rotated backup files.
	backups, _ := filepath.Glob(logPath + ".*")
	assert.GreaterOrEqual(t, len(backups), 1, "Expected at least one rotated backup file")
	logger.Close()
}

// Test 10: Metrics reset functionality works as expected.
func TestLoggerMetricsReset(t *testing.T) {
	factory := l.NewStandardFactory()
	config := l.Config{
		JsonFormat: true,
		Metrics:    true,
	}
	collector, err := factory.CreateMetricsCollector(config)
	require.NoError(t, err)
	// Record some metrics.
	for i := 0; i < 10; i++ {
		collector.IncrementTotal()
		if i%2 == 0 {
			collector.IncrementErrors()
		}
	}
	metricsBefore := collector.GetMetrics()
	require.Greater(t, metricsBefore.TotalMessages, uint64(0))
	collector.Reset()
	metricsAfter := collector.GetMetrics()
	assert.Equal(t, uint64(0), metricsAfter.TotalMessages)
	assert.Equal(t, uint64(0), metricsAfter.ErrorMessages)
}

// Test 1: Logging with an odd number of arguments.
func TestLoggerWithOddArgs(t *testing.T) {
	var buf bytes.Buffer
	factory := l.NewStandardFactory()
	config := l.Config{
		Output:     &buf,
		JsonFormat: true,
	}
	logger, err := factory.CreateLogger(config)
	require.NoError(t, err)

	// Provide an odd number of key/value args; expect the last key to be ignored.
	logger.Info("odd args test", "key1", "value1", "key2")
	err = logger.Flush()
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "value1", result["key1"])
	_, ok := result["key2"]
	assert.False(t, ok, "Expected 'key2' to be ignored due to missing value")
	logger.Close()
}

// Test 2: Logging with a non-string key.
func TestLoggerWithNonStringKey(t *testing.T) {
	var buf bytes.Buffer
	factory := l.NewStandardFactory()
	config := l.Config{
		Output:     &buf,
		JsonFormat: true,
	}
	logger, err := factory.CreateLogger(config)
	require.NoError(t, err)

	// Use an integer as a key (non-string). Depending on implementation,
	// the logger might convert it to a string or skip it.
	logger.Info("non-string key test", 123, "value123", "normal", "value")
	err = logger.Flush()
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	// Verify at least the normal key/value appears.
	assert.Equal(t, "value", result["normal"])
	logger.Close()
}

// Test 3: Logging an error should include a stack trace.
func TestLoggerErrorStackTrace(t *testing.T) {
	var buf bytes.Buffer
	factory := l.NewStandardFactory()
	config := l.Config{
		Output:     &buf,
		JsonFormat: true,
	}
	logger, err := factory.CreateLogger(config)
	require.NoError(t, err)

	// Log an error (the logger enriches error logs with a stack trace).
	logger.Error("error with stack", "error", "simulated error")
	err = logger.Flush()
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	_, ok := result["stack"]
	assert.True(t, ok, "Expected 'stack' field to be present in error log")
	logger.Close()
}

// Test 4: Logging after the logger’s context has been canceled.
func TestLoggerAfterContextCancellation(t *testing.T) {
	var buf bytes.Buffer
	factory := l.NewStandardFactory()
	config := l.Config{
		Output:     &buf,
		JsonFormat: true,
	}
	logger, err := factory.CreateLogger(config)
	require.NoError(t, err)

	// Cancel the logger's context.
	ctx, cancel := context.WithCancel(logger.GetContext())
	cancel()
	cancelledLogger := logger.WithContext(ctx)
	cancelledLogger.Info("logging after cancellation", "test", "value")
	err = logger.Flush()
	require.NoError(t, err)
	assert.Greater(t, buf.Len(), 0, "Expected log output even with a canceled context")
	logger.Close()
}

// Test 5: Logger configuration with a nil Output should default to os.Stdout.
func TestLoggerWithNilOutput(t *testing.T) {
	// Redirect os.Stdout to capture output.
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	factory := l.NewStandardFactory()
	config := l.Config{
		Output:     nil, // nil output; should default internally to os.Stdout.
		JsonFormat: true,
	}
	logger, err := factory.CreateLogger(config)
	require.NoError(t, err)
	logger.Info("nil output test", "key", "value")
	err = logger.Flush()
	require.NoError(t, err)
	logger.Close()

	w.Close()
	var outBuf bytes.Buffer
	_, err = io.Copy(&outBuf, r)
	require.NoError(t, err)
	os.Stdout = oldStdout

	output := outBuf.String()
	assert.Contains(t, output, "nil output test")
}

// Test 6: Concurrent flush and write operations.
func TestConcurrentFlushAndWrite(t *testing.T) {
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

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			logger.Info("concurrent write", "iteration", i)
			time.Sleep(10 * time.Millisecond)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			err := logger.Flush()
			require.NoError(t, err)
			time.Sleep(25 * time.Millisecond)
		}
	}()
	wg.Wait()
	err = logger.Flush()
	require.NoError(t, err)
	logger.Close()
	assert.Greater(t, buf.Len(), 0)
}

// Test 7: Logging using a custom blocking writer.
type blockingWriter struct {
	inner io.Writer
	delay time.Duration
}

func (bw *blockingWriter) Write(p []byte) (int, error) {
	time.Sleep(bw.delay)
	return bw.inner.Write(p)
}

func (bw *blockingWriter) Close() error {
	if closer, ok := bw.inner.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (bw *blockingWriter) Flush() error {
	if flusher, ok := bw.inner.(interface{ Flush() error }); ok {
		return flusher.Flush()
	}
	return nil
}

//
//func TestLoggerWithBlockingWriter(t *testing.T) {
//	var buf bytes.Buffer
//	bw := &blockingWriter{
//		inner: &buf,
//		delay: 50 * time.Millisecond,
//	}
//	factory := l.NewStandardFactory()
//	config := l.Config{
//		Output:     bw,
//		JsonFormat: true,
//		AsyncWrite: true,
//		BufferSize: 512,
//	}
//	logger, err := factory.CreateLogger(config)
//	require.NoError(t, err)
//
//	start := time.Now()
//	logger.Info("blocking writer test", "test", "value")
//	err = logger.Flush()
//	require.NoError(t, err)
//	duration := time.Since(start)
//	assert.GreaterOrEqual(t, duration, 50*time.Millisecond, "Expected delay from blocking writer")
//	logger.Close()
//	output := buf.String()
//	assert.Contains(t, output, "blocking writer test")
//}
//
//// Test 8: Using a custom time format.
//func TestLoggerCustomTimeFormat(t *testing.T) {
//	var buf bytes.Buffer
//	factory := l.NewStandardFactory()
//	customTime := "2006/01/02 15:04:05"
//	config := l.Config{
//		Output:     &buf,
//		JsonFormat: false,
//		TimeFormat: customTime,
//	}
//	logger, err := factory.CreateLogger(config)
//	require.NoError(t, err)
//
//	logger.Info("custom time test")
//	err = logger.Flush()
//	require.NoError(t, err)
//	output := buf.String()
//	// Check for the presence of "/" in the time stamp.
//	assert.Contains(t, output, "/", "Expected custom time format to use '/' as separator")
//	logger.Close()
//}

// Test 9: Logging with a non-serializable value (e.g. a channel).
func TestLoggerWithNonSerializableValue(t *testing.T) {
	var buf bytes.Buffer
	factory := l.NewStandardFactory()
	config := l.Config{
		Output:     &buf,
		JsonFormat: true,
	}
	logger, err := factory.CreateLogger(config)
	require.NoError(t, err)

	ch := make(chan int)
	// Log a channel value; the JSON encoder may omit or represent it in some default way.
	logger.Info("non serializable", "channel", ch)
	err = logger.Flush()
	require.NoError(t, err)
	logger.Close()
	output := buf.String()
	assert.Contains(t, output, "non serializable")
}

// Test 10: Logging with huge numeric values.
func TestLoggerWithHugeNumericValues(t *testing.T) {
	var buf bytes.Buffer
	factory := l.NewStandardFactory()
	config := l.Config{
		Output:     &buf,
		JsonFormat: true,
	}
	logger, err := factory.CreateLogger(config)
	require.NoError(t, err)

	hugeInt := int64(1 << 62)
	hugeFloat := 1.0e308
	logger.Info("huge numbers", "hugeInt", hugeInt, "hugeFloat", hugeFloat)
	err = logger.Flush()
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	// JSON numbers become float64.
	assert.Equal(t, float64(hugeInt), result["hugeInt"])
	assert.Equal(t, hugeFloat, result["hugeFloat"])
	logger.Close()
}
