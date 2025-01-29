// File: tests/default_test.go
package tests

import (
	"github.com/baditaflorin/l"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
	"sync"
	"testing"
)

func TestDefaultLoggerInitialization(t *testing.T) {
	// Test initial default logger
	logger := l.GetDefaultLogger()
	require.NotNil(t, logger, "Default logger should not be nil")

	// Test setting a new default logger
	factory := l.NewStandardFactory()
	config := l.Config{
		Output:     io.Discard,
		JsonFormat: true,
		MinLevel:   slog.LevelInfo,
		AddSource:  true,
	}

	newLogger, err := factory.CreateLogger(config)
	require.NoError(t, err)

	// Set new default logger
	l.SetDefaultLogger(newLogger)

	// Verify new logger is set
	currentLogger := l.GetDefaultLogger()
	assert.Equal(t, newLogger, currentLogger)

	// Test concurrent access to default logger
	var wg sync.WaitGroup
	concurrency := 10
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			l.Info("test message")
			logger := l.GetDefaultLogger()
			assert.NotNil(t, logger)
		}()
	}

	wg.Wait()

	// Test nil logger handling
	l.SetDefaultLogger(nil)
	assert.NotNil(t, l.GetDefaultLogger(), "Default logger should not be nil even when setting nil")

	// Restore original logger
	l.SetDefaultLogger(logger)
}

func TestDefaultLoggerConcurrentUsage(t *testing.T) {
	var wg sync.WaitGroup
	concurrency := 10
	messagesPerRoutine := 100
	wg.Add(concurrency)

	// Test setting a new default logger
	factory := l.NewStandardFactory()
	config := l.Config{
		Output:     io.Discard,
		JsonFormat: true,
		MinLevel:   slog.LevelInfo,
		AddSource:  true,
	}

	newLogger, err := factory.CreateLogger(config)
	require.NoError(t, err)

	// Set new default logger
	l.SetDefaultLogger(newLogger)

	for i := 0; i < concurrency; i++ {
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < messagesPerRoutine; j++ {
				l.Info("test message",
					"routine_id", routineID,
					"iteration", j)

				l.Error("test error",
					"routine_id", routineID,
					"iteration", j)

				l.Warn("test warning",
					"routine_id", routineID,
					"iteration", j)

				l.Debug("test debug",
					"routine_id", routineID,
					"iteration", j)
			}
		}(i)
	}

	wg.Wait()
}

func TestDefaultLoggerWithContextAndFields(t *testing.T) {
	// Test With
	logger := l.With("test_key", "test_value")
	require.NotNil(t, logger)

	// Test chaining With calls
	chainedLogger := logger.With("another_key", "another_value")
	require.NotNil(t, chainedLogger)

	// Test logging with context
	logger.Info("test with context")
	chainedLogger.Info("test with chained context")

	// Test all log levels
	levels := []struct {
		name string
		fn   func(string, ...any)
	}{
		{"Info", l.Info},
		{"Error", l.Error},
		{"Warn", l.Warn},
		{"Debug", l.Debug},
	}

	for _, level := range levels {
		t.Run(level.name, func(t *testing.T) {
			level.fn("test message", "level", level.name)
		})
	}
}
