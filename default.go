// File: default.go
package l

import (
	"log/slog"
	"os"
	"sync"
	"time"
)

var (
	defaultLogger     Logger
	defaultLoggerOnce sync.Once
)

// Initialize the default logger with standard configuration
func init() {
	defaultLogger = mustCreateDefaultLogger()
}

// mustCreateDefaultLogger creates a default logger or panics
func mustCreateDefaultLogger() Logger {
	// Set up a default configuration that uses both new and legacy fields.
	config := Config{
		// New fields
		Level:       slog.LevelInfo,
		JSON:        true,
		ServiceName: "whisper-service",
		Environment: "development",
		TimeFormat:  time.RFC3339,

		// Legacy fields
		MinLevel:   slog.LevelInfo,
		JsonFormat: true,

		AddSource: true,
		Output:    os.Stdout,
		// Other fields can be set to defaults as needed.
		AsyncWrite:  false,
		BufferSize:  1024 * 1024,
		MaxFileSize: 50 * 1024 * 1024, // e.g. 50MB
		MaxBackups:  7,
	}

	logger, err := NewStandardLogger(NewStandardFactory(), config)
	if err != nil {
		panic(err)
	}

	return logger
}

// Package-level logging functions that use the default logger
func Info(msg string, args ...any)  { defaultLogger.Info(msg, args...) }
func Error(msg string, args ...any) { defaultLogger.Error(msg, args...) }
func Warn(msg string, args ...any)  { defaultLogger.Warn(msg, args...) }
func Debug(msg string, args ...any) { defaultLogger.Debug(msg, args...) }

// With returns a new logger with the given key-value pairs added to the context
func With(args ...any) Logger { return defaultLogger.With(args...) }

// SetDefaultLogger allows users to replace the default logger with their own instance
func SetDefaultLogger(logger Logger) {
	if logger != nil {
		defaultLogger = logger
	}
}

// GetDefaultLogger returns the current default logger instance
func GetDefaultLogger() Logger {
	return defaultLogger
}
