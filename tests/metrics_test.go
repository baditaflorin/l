// File: tests/metrics_test.go
package tests

import (
	"bytes"
	"github.com/baditaflorin/l"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestMetricsCollection(t *testing.T) {
	factory := l.NewStandardFactory()
	config := l.Config{
		JsonFormat: true,
		Metrics:    true,
	}

	collector, err := factory.CreateMetricsCollector(config)
	require.NoError(t, err)

	// Test incrementing metrics
	expectedTotal := uint64(100)
	expectedErrors := uint64(5)
	expectedDropped := uint64(2)

	for i := uint64(0); i < expectedTotal; i++ {
		collector.IncrementTotal()
	}

	for i := uint64(0); i < expectedErrors; i++ {
		collector.IncrementErrors()
	}

	for i := uint64(0); i < expectedDropped; i++ {
		collector.IncrementDropped()
	}

	// Set timestamps
	now := time.Now()
	collector.SetLastError(now)
	collector.SetLastFlush(now)

	// Get and verify metrics
	metrics := collector.GetMetrics()
	assert.Equal(t, expectedTotal, metrics.TotalMessages)
	assert.Equal(t, expectedErrors, metrics.ErrorMessages)
	assert.Equal(t, expectedDropped, metrics.DroppedMessages)
	assert.Equal(t, now.Unix(), metrics.LastError.Unix())
	assert.Equal(t, now.Unix(), metrics.LastFlush.Unix())

	// Test reset
	collector.Reset()
	metrics = collector.GetMetrics()
	assert.Equal(t, uint64(0), metrics.TotalMessages)
	assert.Equal(t, uint64(0), metrics.ErrorMessages)
	assert.Equal(t, uint64(0), metrics.DroppedMessages)
}

func TestErrorHandler(t *testing.T) {
	var lastError error
	errorCallback := func(err error) {
		lastError = err
	}

	handler := l.NewStandardErrorHandler(errorCallback)

	// Test error handling
	testError := assert.AnError
	handler.Handle(testError)
	assert.Equal(t, testError, lastError)

	// Test panic recovery
	err := handler.WithRecovery(func() error {
		panic("test panic")
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "test panic")

	// Test normal execution
	err = handler.WithRecovery(func() error {
		return nil
	})
	assert.NoError(t, err)
}

func TestLoggerWithMetrics(t *testing.T) {
	var buf bytes.Buffer
	factory := l.NewStandardFactory()
	config := l.Config{
		Output:     &buf,
		JsonFormat: true,
		Metrics:    true,
	}

	// Create the logger first
	logger, err := factory.CreateLogger(config)
	require.NoError(t, err)
	defer logger.Close()

	// Get the metrics collector from the factory - this ensures we use the same one
	collector, err := factory.CreateMetricsCollector(config)
	require.NoError(t, err)

	testCases := []struct {
		name           string
		logFunc        func()
		expectedTotal  uint64
		expectedErrors uint64
	}{
		{
			name: "Info messages only",
			logFunc: func() {
				logger.Info("test info 1")
				logger.Info("test info 2")
			},
			expectedTotal:  2,
			expectedErrors: 0,
		},
		{
			name: "Error messages",
			logFunc: func() {
				logger.Error("test error 1")
				logger.Error("test error 2")
			},
			expectedTotal:  2,
			expectedErrors: 2,
		},
		{
			name: "Mixed messages",
			logFunc: func() {
				logger.Info("test info")
				logger.Error("test error")
				logger.Warn("test warning")
			},
			expectedTotal:  3,
			expectedErrors: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf.Reset()
			collector.Reset() // Reset metrics before each test case

			tc.logFunc()

			// Ensure logs are flushed and processed
			err = logger.Flush()
			require.NoError(t, err)

			// Wait for any async operations
			time.Sleep(100 * time.Millisecond)

			metrics := collector.GetMetrics()
			assert.Equal(t, tc.expectedTotal, metrics.TotalMessages, "Total messages mismatch")
			assert.Equal(t, tc.expectedErrors, metrics.ErrorMessages, "Error messages mismatch")
		})
	}
}

func TestConcurrentMetricsCollection(t *testing.T) {
	factory := l.NewStandardFactory()
	config := l.Config{
		JsonFormat: true,
		Metrics:    true,
	}

	collector, err := factory.CreateMetricsCollector(config)
	require.NoError(t, err)

	// Test concurrent metric updates
	concurrency := 10
	messagesPerRoutine := 1000
	done := make(chan bool)

	for i := 0; i < concurrency; i++ {
		go func() {
			for j := 0; j < messagesPerRoutine; j++ {
				collector.IncrementTotal()
				if j%20 == 0 { // 5% error rate
					collector.IncrementErrors()
				}
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < concurrency; i++ {
		<-done
	}

	metrics := collector.GetMetrics()
	expectedTotal := uint64(concurrency * messagesPerRoutine)
	expectedErrors := uint64(concurrency * (messagesPerRoutine / 20))

	assert.Equal(t, expectedTotal, metrics.TotalMessages)
	assert.Equal(t, expectedErrors, metrics.ErrorMessages)
}

func TestMetricsWithHealthCheck(t *testing.T) {
	factory := l.NewStandardFactory()
	config := l.Config{
		JsonFormat: true,
		Metrics:    true,
	}

	logger, err := factory.CreateLogger(config)
	require.NoError(t, err)
	defer logger.Close()

	healthChecker, err := factory.CreateHealthChecker(logger)
	require.NoError(t, err)

	// Start health checker
	healthChecker.Start(logger.GetContext())
	defer healthChecker.Stop()

	// Perform some logging operations
	for i := 0; i < 100; i++ {
		logger.Info("test message", "index", i)
		if i%10 == 0 {
			logger.Error("test error", "index", i)
		}
	}

	// Check health
	err = healthChecker.Check()
	assert.NoError(t, err)

	// Wait for metrics to be updated
	time.Sleep(100 * time.Millisecond)

	// Verify metrics are being collected
	collector, err := factory.CreateMetricsCollector(config)
	require.NoError(t, err)
	metrics := collector.GetMetrics()

	assert.True(t, metrics.TotalMessages >= 100)
	assert.True(t, metrics.ErrorMessages >= 10)
}

func TestMetricsReset(t *testing.T) {
	factory := l.NewStandardFactory()
	config := l.Config{
		JsonFormat: true,
		Metrics:    true,
	}

	collector, err := factory.CreateMetricsCollector(config)
	require.NoError(t, err)

	// Record some metrics
	for i := 0; i < 100; i++ {
		collector.IncrementTotal()
		if i%10 == 0 {
			collector.IncrementErrors()
		}
	}

	// Verify initial state
	metrics := collector.GetMetrics()
	assert.Equal(t, uint64(100), metrics.TotalMessages)
	assert.Equal(t, uint64(10), metrics.ErrorMessages)

	// Test reset
	collector.Reset()
	metricsAfterReset := collector.GetMetrics()

	// Verify everything is zeroed out
	assert.Equal(t, uint64(0), metricsAfterReset.TotalMessages)
	assert.Equal(t, uint64(0), metricsAfterReset.ErrorMessages)
	assert.Equal(t, uint64(0), metricsAfterReset.DroppedMessages)

	// Verify we can still record metrics after reset
	collector.IncrementTotal()
	collector.IncrementErrors()

	metricsAfterNewRecording := collector.GetMetrics()
	assert.Equal(t, uint64(1), metricsAfterNewRecording.TotalMessages)
	assert.Equal(t, uint64(1), metricsAfterNewRecording.ErrorMessages)
}
