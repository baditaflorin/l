package tests

import (
	"context"
	"github.com/baditaflorin/l"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoggerIntegration(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "logger_integration_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "test.log")
	t.Logf("Log file path: %s", logPath)

	// Configure logger with all features enabled
	config := l.Config{
		FilePath:    logPath,
		JsonFormat:  true,
		AsyncWrite:  false, // Set to false for debugging
		MaxFileSize: 1024,  // 1KB to trigger rotation quickly
		MaxBackups:  3,
		AddSource:   true,
		Metrics:     true,
		MinLevel:    l.LevelDebug,
		Output:      io.Discard,
	}

	factory := l.NewStandardFactory()
	logger, err := factory.CreateLogger(config)
	require.NoError(t, err)

	// Verify initial file creation
	_, err = os.Stat(logPath)
	require.NoError(t, err, "Log file should be created")
	t.Log("Initial log file created successfully")

	// Test context propagation
	ctx := context.WithValue(context.Background(), "test_key", "test_value")
	contextLogger := logger.WithContext(ctx)

	// Write enough data to trigger rotation
	largePayload := string(make([]byte, 200)) // 200 bytes per message
	for i := 0; i < 10; i++ {
		contextLogger.Info("test message",
			"iteration", i,
			"payload", largePayload,
		)

		// Flush after each write for debugging
		err = logger.Flush()
		require.NoError(t, err)

		// Check file size after each write
		if info, err := os.Stat(logPath); err == nil {
			t.Logf("Current file size after iteration %d: %d bytes", i, info.Size())
		}
	}

	// Force flush and wait
	err = logger.Flush()
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	// Check for rotated files
	pattern := logPath + "*"
	files, err := filepath.Glob(pattern)
	require.NoError(t, err)

	// Log all found files
	t.Log("Found files:")
	for _, file := range files {
		if info, err := os.Stat(file); err == nil {
			t.Logf("  - %s (size: %d bytes)", file, info.Size())
		}
	}

	assert.True(t, len(files) > 1, "Expected rotated log files, got %d files", len(files))

	// Clean up
	err = logger.Close()
	assert.NoError(t, err)
}
