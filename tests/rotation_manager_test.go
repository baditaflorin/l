// File: tests/rotation_manager_test.go
package tests

import (
	"github.com/baditaflorin/l"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

// Test RotationManager functions with 0% coverage
func TestRotationManager(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rotation_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	logFile := filepath.Join(tmpDir, "test.log")
	config := l.Config{
		FilePath:    logFile,
		MaxFileSize: 1024,
		MaxBackups:  3,
	}

	t.Run("RotationManager Operations", func(t *testing.T) {
		factory := l.NewStandardFactory()
		manager, err := factory.CreateRotationManager(config)
		require.NoError(t, err)

		// Test GetCurrentFile
		writer, err := manager.GetCurrentFile()
		assert.NoError(t, err)
		assert.NotNil(t, writer)

		// Test Cleanup
		err = manager.Cleanup()
		assert.NoError(t, err)
	})
}
