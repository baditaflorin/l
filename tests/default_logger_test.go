// File: tests/default_logger_test.go
package tests

import (
	"github.com/baditaflorin/l"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
	"testing"
)

// Test default logger functions with 0% coverage
func TestDefaultLogger(t *testing.T) {

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

	t.Run("Default Logger Package Functions", func(t *testing.T) {
		// Test Info
		l.Info("test info message", "key", "value")

		// Test Error
		l.Error("test error message", "error", "test error")

		// Test Warn
		l.Warn("test warning message", "warning", "test warning")

		// Test Debug
		l.Debug("test debug message", "debug", "test debug")

		// Test With
		logger := l.With("context", "test")
		require.NotNil(t, logger)
		logger.Info("test with context")
	})

	t.Run("SetDefaultLogger and GetDefaultLogger", func(t *testing.T) {
		// Get current default logger
		originalLogger := l.GetDefaultLogger()
		require.NotNil(t, originalLogger)

		// Create a new logger
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
		assert.NotEqual(t, originalLogger, currentLogger)

		// Test logging with new default logger
		l.Info("test with new default logger")

		// Reset to original logger
		l.SetDefaultLogger(originalLogger)
	})
}
