// File: default.go
package l

import (
	"log/slog"
	"os"
	"sync"
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
	config := Config{
		Output:     os.Stdout,
		JsonFormat: true,
		MinLevel:   slog.LevelInfo, // Fixed: Using slog.LevelInfo instead of LevelInfo
		AddSource:  true,
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
