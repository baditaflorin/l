// File: tests/writer_test.go
package tests

import (
	"bytes"
	"fmt"
	"github.com/baditaflorin/l"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

	// Helper function to verify content with retries
	verifyContent := func(expected string, timeout time.Duration) bool {
		deadline := time.Now().Add(timeout)
		for time.Now().Before(deadline) {
			err := writer.Flush()
			assert.NoError(t, err)
			if strings.Contains(buf.String(), expected) {
				return true
			}
			time.Sleep(50 * time.Millisecond)
		}
		return false
	}

	t.Run("Single Write", func(t *testing.T) {
		buf.Reset()
		testData := []byte("test message\n")
		n, err := writer.Write(testData)
		assert.NoError(t, err)
		assert.Equal(t, len(testData), n)

		success := verifyContent("test message", time.Second)
		assert.True(t, success, "Expected content not found after timeout")
	})

	t.Run("Multiple Sequential Writes", func(t *testing.T) {
		buf.Reset()
		for i := 0; i < 10; i++ {
			data := []byte(fmt.Sprintf("message %d\n", i))
			_, err := writer.Write(data)
			assert.NoError(t, err)
		}

		success := verifyContent("message 9", time.Second)
		assert.True(t, success, "Expected content not found after timeout")
		assert.Equal(t, 10, bytes.Count(buf.Bytes(), []byte("\n")))
	})

	t.Run("Concurrent Writes", func(t *testing.T) {
		buf.Reset()
		var wg sync.WaitGroup
		numGoroutines := 5
		messagesPerRoutine := 100

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(routineID int) {
				defer wg.Done()
				for j := 0; j < messagesPerRoutine; j++ {
					msg := fmt.Sprintf("routine-%d-msg-%d\n", routineID, j)
					_, err := writer.Write([]byte(msg))
					assert.NoError(t, err)
				}
			}(i)
		}

		wg.Wait()
		err := writer.Flush()
		assert.NoError(t, err)
		time.Sleep(100 * time.Millisecond) // Give extra time for flush to complete

		totalExpectedLines := numGoroutines * messagesPerRoutine
		actualLines := bytes.Count(buf.Bytes(), []byte("\n"))
		assert.Equal(t, totalExpectedLines, actualLines,
			"Expected %d lines but got %d", totalExpectedLines, actualLines)
	})

	t.Run("Write After Close", func(t *testing.T) {
		buf.Reset()
		newWriter := l.NewBufferedWriter(&buf, 1024)
		testData := []byte("test before close\n")
		_, err := newWriter.Write(testData)
		assert.NoError(t, err)

		err = newWriter.Flush()
		assert.NoError(t, err)

		err = newWriter.Close()
		assert.NoError(t, err)

		// Try writing after close
		_, err = newWriter.Write([]byte("test after close\n"))
		assert.Error(t, err, "Write after close should return error")
		assert.Contains(t, err.Error(), "closed")
	})

	t.Run("Large Write Test", func(t *testing.T) {
		buf.Reset()
		largeData := bytes.Repeat([]byte("a"), 1000000) // 1MB of data
		n, err := writer.Write(largeData)
		assert.NoError(t, err)
		assert.Equal(t, len(largeData), n)

		err = writer.Flush()
		assert.NoError(t, err)
		assert.Equal(t, len(largeData), buf.Len())
	})

	t.Run("Buffer Full Behavior", func(t *testing.T) {
		buf.Reset()
		// Create a writer with a very small buffer
		smallWriter := l.NewBufferedWriter(&buf, 10) // Very small buffer

		// Write multiple chunks of data
		for i := 0; i < 10; i++ {
			data := []byte(fmt.Sprintf("chunk-%d\n", i))
			n, err := smallWriter.Write(data)
			assert.NoError(t, err, "Write should succeed even with full buffer")
			assert.Equal(t, len(data), n, "Should write all bytes")
		}

		// Give some time for async writes to complete
		time.Sleep(100 * time.Millisecond)

		err := smallWriter.Flush()
		assert.NoError(t, err, "Flush should succeed")

		// Verify all data was written
		content := buf.String()
		for i := 0; i < 10; i++ {
			expectedChunk := fmt.Sprintf("chunk-%d\n", i)
			assert.Contains(t, content, expectedChunk,
				"Output should contain chunk %d", i)
		}

		// Try a large write that exceeds buffer size
		largeData := bytes.Repeat([]byte("b"), 1000)
		n, err := smallWriter.Write(largeData)
		assert.NoError(t, err, "Large write should succeed")
		assert.Equal(t, len(largeData), n, "Should write all bytes of large write")

		err = smallWriter.Flush()
		assert.NoError(t, err, "Final flush should succeed")

		// Verify the large write was completed
		assert.True(t, bytes.Contains(buf.Bytes(), bytes.Repeat([]byte("b"), 1000)),
			"Output should contain the large write")

		err = smallWriter.Close()
		assert.NoError(t, err, "Close should succeed")
	})

	// Cleanup
	err := writer.Flush()
	assert.NoError(t, err)
	err = writer.Close()
	assert.NoError(t, err)
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
