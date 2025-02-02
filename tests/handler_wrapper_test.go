// File: tests/handler_wrapper_test.go
package tests

import (
	"github.com/baditaflorin/l"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"os"
	"testing"
)

// Test HandlerWrapper functions with 0% coverage
func TestHandlerWrapper(t *testing.T) {
	factory := l.NewStandardFactory()
	config := l.Config{
		JsonFormat: true,
	}

	logger, err := factory.CreateLogger(config)
	require.NoError(t, err)
	defer logger.Close()

	t.Run("HandlerWrapper Methods", func(t *testing.T) {
		// Test Enabled
		wrapper := factory.CreateHandlerWrapper(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}),
			logger,
		)

		enabled := wrapper.Enabled(nil, l.LevelInfo)
		assert.True(t, enabled)

		// Test WithAttrs
		attrs := []slog.Attr{
			slog.String("key", "value"),
		}
		newHandler := wrapper.WithAttrs(attrs)
		assert.NotNil(t, newHandler)

		// Test WithGroup
		groupHandler := wrapper.WithGroup("test_group")
		assert.NotNil(t, groupHandler)
	})
}
