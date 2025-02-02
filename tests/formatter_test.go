// File: tests/formatter_test.go
package tests

import (
	"encoding/json"
	"github.com/baditaflorin/l"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestJSONFormatter(t *testing.T) {
	formatter := l.NewJSONFormatter()

	testCases := []struct {
		name           string
		record         l.LogRecord
		expectedFields map[string]interface{}
	}{
		{
			name: "Basic log entry",
			record: l.LogRecord{
				Level:   l.LevelInfo,
				Message: "test message",
				Time:    time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
				Args: []any{
					"key1", "value1",
					"key2", 42,
				},
			},
			expectedFields: map[string]interface{}{
				"level": "INFO",
				"msg":   "test message",
				"key1":  "value1",
				"key2":  float64(42),
				"time":  "2025-01-01T12:00:00Z",
			},
		},
		{
			name: "Error with stack trace",
			record: l.LogRecord{
				Level:   l.LevelError,
				Message: "error occurred",
				Time:    time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
				Args: []any{
					"error", "connection failed",
					"code", 500,
					"stack", []string{"func1", "func2"},
				},
			},
			expectedFields: map[string]interface{}{
				"level": "ERROR",
				"msg":   "error occurred",
				"error": "connection failed",
				"code":  float64(500),
				"stack": []interface{}{"func1", "func2"},
				"time":  "2025-01-01T12:00:00Z",
			},
		},
		{
			name: "Complex nested data",
			record: l.LogRecord{
				Level:   l.LevelInfo,
				Message: "nested data",
				Time:    time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
				Args: []any{
					"data", map[string]interface{}{
						"user": map[string]interface{}{
							"id":   123,
							"name": "test",
						},
					},
				},
				Attrs: []slog.Attr{
					slog.Group("metadata",
						slog.String("version", "1.0"),
						slog.Int("api_version", 2),
					),
				},
			},
			expectedFields: map[string]interface{}{
				"level": "INFO",
				"msg":   "nested data",
				"data": map[string]interface{}{
					"user": map[string]interface{}{
						"id":   float64(123),
						"name": "test",
					},
				},
				"metadata": map[string]interface{}{
					"version":     "1.0",
					"api_version": float64(2),
				},
				"time": "2025-01-01T12:00:00Z",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := formatter.Format(tc.record)
			require.NoError(t, err)

			var logEntry map[string]interface{}
			err = json.Unmarshal(output, &logEntry)
			require.NoError(t, err)

			for key, expectedValue := range tc.expectedFields {
				assert.Equal(t, expectedValue, logEntry[key], "Mismatch for field %s", key)
			}
		})
	}
}

func TestTextFormatter(t *testing.T) {
	formatter := l.NewTextFormatter()

	testCases := []struct {
		name     string
		record   l.LogRecord
		contains []string
		excludes []string
	}{
		{
			name: "Simple log message",
			record: l.LogRecord{
				Level:   l.LevelInfo,
				Message: "test message",
				Time:    time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
				Args: []any{
					"user", "testuser",
					"action", "login",
				},
			},
			contains: []string{
				"INFO",
				"test message",
				"user=testuser",
				"action=login",
				"2025-01-01",
			},
		},
		{
			name: "Error message with details",
			record: l.LogRecord{
				Level:   l.LevelError,
				Message: "operation failed",
				Time:    time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
				Args: []any{
					"error", "database connection failed",
					"retry_count", 3,
					"duration_ms", 1500,
				},
			},
			contains: []string{
				"ERROR",
				"operation failed",
				"error=database connection failed",
				"retry_count=3",
				"duration_ms=1500",
			},
		},
		{
			name: "Debug message with source",
			record: l.LogRecord{
				Level:     l.LevelDebug,
				Message:   "debug info",
				Time:      time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
				Source:    "main.go:123",
				AddSource: true,
				Args: []any{
					"detail", "connection pool stats",
					"active", 5,
					"idle", 2,
				},
			},
			contains: []string{
				"DEBUG",
				"debug info",
				"main.go:123",
				"detail=connection pool stats",
				"active=5",
				"idle=2",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := formatter.Format(tc.record)
			require.NoError(t, err)

			outputStr := string(output)

			// Check for required content
			for _, substr := range tc.contains {
				assert.True(t, strings.Contains(outputStr, substr),
					"Output should contain '%s', got: %s", substr, outputStr)
			}

			// Check for excluded content
			for _, substr := range tc.excludes {
				assert.False(t, strings.Contains(outputStr, substr),
					"Output should not contain '%s', got: %s", substr, outputStr)
			}
		})
	}
}

func TestFormatterOptions(t *testing.T) {
	testCases := []struct {
		name   string
		opts   l.FormatterOptions
		record l.LogRecord
		verify func(t *testing.T, output string)
	}{
		{
			name: "Custom time format",
			opts: l.FormatterOptions{
				TimeFormat: time.RFC822,
				UseColor:   false,
				Indent:     "",
			},
			record: l.LogRecord{
				Level:   l.LevelInfo,
				Message: "test message",
				Time:    time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
			},
			verify: func(t *testing.T, output string) {
				assert.Contains(t, output, "01 Jan 25 12:00 UTC")
			},
		},
		{
			name: "With indentation",
			opts: l.FormatterOptions{
				TimeFormat: time.RFC3339,
				UseColor:   false,
				Indent:     "    ",
			},
			record: l.LogRecord{
				Level:   l.LevelInfo,
				Message: "test message",
				Args:    []any{"key", "value"},
			},
			verify: func(t *testing.T, output string) {
				lines := strings.Split(output, "\n")
				for _, line := range lines {
					if line != "" {
						assert.True(t, strings.HasPrefix(line, "    "))
					}
				}
			},
		},
		{
			name: "With color enabled",
			opts: l.FormatterOptions{
				TimeFormat: time.RFC3339,
				UseColor:   true,
				Indent:     "",
			},
			record: l.LogRecord{
				Level:   l.LevelError,
				Message: "error message",
			},
			verify: func(t *testing.T, output string) {
				// Verify ANSI color codes are present
				assert.Contains(t, output, "\x1b[")
				assert.Contains(t, output, "\x1b[0m")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			formatter := l.NewTextFormatter().WithOptions(tc.opts)
			output, err := formatter.Format(tc.record)
			require.NoError(t, err)

			tc.verify(t, string(output))
		})
	}
}

func TestFormatterEdgeCases(t *testing.T) {
	jsonFormatter := l.NewJSONFormatter()
	textFormatter := l.NewTextFormatter()

	testCases := []struct {
		name      string
		record    l.LogRecord
		formatter l.Formatter
	}{
		{
			name: "Empty message",
			record: l.LogRecord{
				Level:   l.LevelInfo,
				Message: "",
				Time:    time.Now(),
			},
			formatter: jsonFormatter,
		},
		{
			name: "Nil args",
			record: l.LogRecord{
				Level:   l.LevelInfo,
				Message: "test",
				Time:    time.Now(),
				Args:    nil,
			},
			formatter: textFormatter,
		},
		{
			name: "Invalid args length",
			record: l.LogRecord{
				Level:   l.LevelInfo,
				Message: "test",
				Time:    time.Now(),
				Args:    []any{"key"}, // Missing value
			},
			formatter: jsonFormatter,
		},
		{
			name: "Non-string keys in args",
			record: l.LogRecord{
				Level:   l.LevelInfo,
				Message: "test",
				Time:    time.Now(),
				Args:    []any{123, "value"},
			},
			formatter: textFormatter,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := tc.formatter.Format(tc.record)
			assert.NoError(t, err, "Formatter should handle edge cases gracefully")
			assert.NotEmpty(t, output, "Formatter should produce non-empty output")
		})
	}
}
