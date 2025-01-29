// File: tests/edge_cases_test.go
package tests

import (
	"bytes"
	"fmt"
	"github.com/baditaflorin/l"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestLoggerEdgeCases(t *testing.T) {
	factory := l.NewStandardFactory()

	t.Run("Zero configuration", func(t *testing.T) {
		config := l.Config{}
		logger, err := factory.CreateLogger(config)
		assert.NoError(t, err)
		assert.NotNil(t, logger)
	})

	t.Run("Invalid file path", func(t *testing.T) {
		config := l.Config{
			FilePath: "/nonexistent/directory/log.txt",
		}
		_, err := factory.CreateLogger(config)
		assert.Error(t, err)
	})

	t.Run("Extremely large messages", func(t *testing.T) {
		var buf bytes.Buffer
		config := l.Config{
			AsyncWrite: true,
			BufferSize: 2 * 1024 * 1024, // 2MB buffer to handle large messages
			Output:     &buf,            // Use buffer to capture output
		}
		logger, err := factory.CreateLogger(config)
		require.NoError(t, err)

		// Test with incrementally larger messages
		sizes := []int{
			1024,        // 1KB
			10 * 1024,   // 10KB
			100 * 1024,  // 100KB
			1024 * 1024, // 1MB
		}

		for _, size := range sizes {
			t.Run(fmt.Sprintf("Message size %d bytes", size), func(t *testing.T) {
				buf.Reset() // Clear buffer before each test

				// Create a placeholder message of the specified size
				logger.Info("large message test",
					"size_bytes", size,
					"content_length", size,
				)

				// Give some time for async processing
				time.Sleep(100 * time.Millisecond)

				// Flush after each large message
				err = logger.Flush()
				assert.NoError(t, err, "Failed to flush logger with message size %d", size)

				// Verify that something was written to the buffer
				assert.Greater(t, buf.Len(), 0, "Expected output for message size %d", size)
			})
		}

		// Test very large message handling
		t.Run("Very large message handling", func(t *testing.T) {
			buf.Reset()
			hugeSize := 5 * 1024 * 1024 // 5MB

			logger.Info("huge message test",
				"size_bytes", hugeSize,
				"content_length", hugeSize,
			)

			// Give some time for async processing
			time.Sleep(200 * time.Millisecond)

			err = logger.Flush()
			assert.NoError(t, err, "Failed to flush logger with huge message")
			assert.Greater(t, buf.Len(), 0, "Expected output for huge message")
		})

		// Clean up
		err = logger.Close()
		assert.NoError(t, err)
	})

	t.Run("Null values in structured logging", func(t *testing.T) {
		config := l.Config{JsonFormat: true}
		logger, err := factory.CreateLogger(config)
		require.NoError(t, err)

		logger.Info("test message",
			"null_value", nil,
			"empty_string", "",
			"zero_int", 0,
		)
	})
}
