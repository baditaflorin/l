// File: tests/field_provider_test.go
package tests

import (
	"github.com/baditaflorin/l"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"testing"
)

// Test FieldProvider functions with 0% coverage
func TestFieldProvider(t *testing.T) {
	factory := l.NewStandardFactory()
	config := l.Config{
		JsonFormat: true,
		Output:     io.Discard,
	}

	logger, err := factory.CreateLogger(config)
	require.NoError(t, err)
	defer logger.Close()

	t.Run("WithFields and GetFields", func(t *testing.T) {
		// Create fields
		fields := map[string]interface{}{
			"service":     "test-service",
			"environment": "testing",
			"version":     "1.0.0",
		}

		// Test WithFields
		fieldLogger := logger.WithFields(fields)
		require.NotNil(t, fieldLogger)

		// Test logging with fields
		fieldLogger.Info("test with fields")

		// Test GetFields
		retrievedFields := fieldLogger.GetFields()
		assert.Equal(t, fields["service"], retrievedFields["service"])
		assert.Equal(t, fields["environment"], retrievedFields["environment"])
		assert.Equal(t, fields["version"], retrievedFields["version"])
	})
}
