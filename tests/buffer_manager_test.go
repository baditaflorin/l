// File: tests/buffer_manager_test.go
package tests

import (
	"github.com/baditaflorin/l"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// Test BufferManager functions with 0% coverage
func TestBufferManager(t *testing.T) {
	factory := l.NewStandardFactory()

	t.Run("StandardBufferManager Operations", func(t *testing.T) {
		bufferSize := 1024
		manager, err := factory.CreateBufferManager(l.Config{
			BufferSize: bufferSize,
		})
		require.NoError(t, err)

		// Test Write
		data := []byte("test message")
		n, err := manager.Write(data)
		assert.NoError(t, err)
		assert.Equal(t, len(data), n)

		// Test IsFull
		isFull := manager.IsFull()
		assert.False(t, isFull)

		// Test Flush
		err = manager.Flush()
		assert.NoError(t, err)

		// Test Close
		err = manager.Close()
		assert.NoError(t, err)

		// Test Write after close
		_, err = manager.Write([]byte("test after close"))
		assert.Error(t, err)
	})
}
