// File: tests/writer_test.go
package tests

import (
	"bytes"
	"github.com/baditaflorin/l"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDirectWriter(t *testing.T) {
	var buf bytes.Buffer
	writer := l.NewDirectWriter(&buf)

	testData := []byte("test message\n")
	n, err := writer.Write(testData)
	assert.NoError(t, err)
	assert.Equal(t, len(testData), n)
	assert.Equal(t, testData, buf.Bytes())

	err = writer.Close()
	assert.NoError(t, err)
}

func TestBufferedWriter(t *testing.T) {
	var buf bytes.Buffer
	writer := l.NewBufferedWriter(&buf, 1024)

	// Test single write
	testData := []byte("test message\n")
	n, err := writer.Write(testData)
	assert.NoError(t, err)
	assert.Equal(t, len(testData), n)

	// Flush and verify
	err = writer.Flush()
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "test message")

	// Test multiple concurrent writes
	done := make(chan bool)
	messages := 100
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < messages; j++ {
				_, _ = writer.Write([]byte("concurrent message\n"))
			}
			done <- true
		}()
	}

	// Wait for all writes to complete
	for i := 0; i < 5; i++ {
		<-done
	}

	// Flush and close
	err = writer.Flush()
	assert.NoError(t, err)
	err = writer.Close()
	assert.NoError(t, err)

	// Verify total messages
	lines := bytes.Count(buf.Bytes(), []byte("\n"))
	assert.Equal(t, 5*messages+1, lines) // +1 for the initial test message
}

func TestFileWriter(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "logger_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create config with test file path
	config := l.Config{
		FilePath:    filepath.Join(tmpDir, "test.log"),
		MaxFileSize: 1024,
		MaxBackups:  3,
	}

	// Create file writer
	writer, err := l.NewFileWriter(config)
	require.NoError(t, err)

	// Test writing
	testData := []byte("test message\n")
	n, err := writer.Write(testData)
	assert.NoError(t, err)
	assert.Equal(t, len(testData), n)

	// Verify file content
	content, err := os.ReadFile(config.FilePath)
	require.NoError(t, err)
	assert.Equal(t, testData, content)

	// Test rotation
	for i := 0; i < 1000; i++ {
		_, err = writer.Write([]byte("test message for rotation\n"))
		require.NoError(t, err)
	}

	// Check if backup files were created
	backupFiles, err := filepath.Glob(config.FilePath + ".*")
	require.NoError(t, err)
	assert.NotEmpty(t, backupFiles)

	// Close writer
	err = writer.Close()
	assert.NoError(t, err)
}

func TestWriterPerformance(t *testing.T) {
	var buf bytes.Buffer
	writer := l.NewBufferedWriter(&buf, 1024*1024) // 1MB buffer

	// Create test data
	testData := bytes.Repeat([]byte("performance test message\n"), 100)

	// Measure write performance
	iterations := 1000
	start := time.Now()

	for i := 0; i < iterations; i++ {
		_, err := writer.Write(testData)
		require.NoError(t, err)
	}

	err := writer.Flush()
	require.NoError(t, err)

	duration := time.Since(start)
	bytesWritten := len(testData) * iterations

	t.Logf("Wrote %d bytes in %v (%.2f MB/s)",
		bytesWritten,
		duration,
		float64(bytesWritten)/1024/1024/duration.Seconds(),
	)

	// Close writer
	err = writer.Close()
	assert.NoError(t, err)
}
