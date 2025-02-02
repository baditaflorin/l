package tests

import (
	"github.com/baditaflorin/l"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestCreateFormatter(t *testing.T) {
	factory := l.NewStandardFactory()

	testCases := []struct {
		name        string
		config      l.Config
		expectJSON  bool
		expectError bool
	}{
		{
			name: "JSON Formatter",
			config: l.Config{
				JsonFormat: true,
			},
			expectJSON:  true,
			expectError: false,
		},
		{
			name: "Text Formatter",
			config: l.Config{
				JsonFormat: false,
			},
			expectJSON:  false,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			formatter, err := factory.CreateFormatter(tc.config)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, formatter)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, formatter)

			// Test formatter type
			if tc.expectJSON {
				_, ok := formatter.(*l.JSONFormatter)
				assert.True(t, ok, "Expected JSONFormatter")
			} else {
				_, ok := formatter.(*l.TextFormatter)
				assert.True(t, ok, "Expected TextFormatter")
			}

			// Test basic formatting
			record := l.LogRecord{
				Level:   l.LevelInfo,
				Message: "test message",
				Time:    time.Now(),
			}

			output, err := formatter.Format(record)
			assert.NoError(t, err)
			assert.NotEmpty(t, output)
		})
	}
}

func TestGetLevelColorAllLevels(t *testing.T) {
	formatter := l.NewTextFormatter().WithOptions(l.FormatterOptions{
		UseColor: true,
	})

	testCases := []struct {
		name          string
		level         l.Level
		expectedColor string
	}{
		{
			name:          "Debug Level",
			level:         l.LevelDebug,
			expectedColor: "\x1b[36m", // Cyan
		},
		{
			name:          "Info Level",
			level:         l.LevelInfo,
			expectedColor: "\x1b[32m", // Green
		},
		{
			name:          "Warn Level",
			level:         l.LevelWarn,
			expectedColor: "\x1b[33m", // Yellow
		},
		{
			name:          "Error Level",
			level:         l.LevelError,
			expectedColor: "\x1b[31m", // Red
		},
		{
			name:          "Unknown Level",
			level:         l.Level(999),
			expectedColor: "\x1b[37m", // White
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			record := l.LogRecord{
				Level:   tc.level,
				Message: "test message",
				Time:    time.Now(),
			}

			output, err := formatter.Format(record)
			require.NoError(t, err)
			assert.Contains(t, string(output), tc.expectedColor)
		})
	}
}
