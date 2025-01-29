// File: example/custom/main.go
package main

import (
	"github.com/baditaflorin/l"
	"io"
	"os"
	"time"
)

func main() {
	// Create custom configuration
	config := l.Config{
		Output:      io.Discard,
		FilePath:    "logs/app.log",
		JsonFormat:  true,
		AsyncWrite:  true,
		BufferSize:  1024 * 1024,      // 1MB buffer
		MaxFileSize: 10 * 1024 * 1024, // 10MB max file size
		MaxBackups:  5,
		AddSource:   true,
		Metrics:     true,
	}

	// Create custom logger
	logger, err := l.NewStandardLogger(l.NewStandardFactory(), config)
	if err != nil {
		panic(err)
	}
	defer func() {
		logger.Flush() // Ensure all logs are written
		logger.Close()
	}()

	// Set as default logger
	l.SetDefaultLogger(logger)

	// Use both the default package-level functions and the logger instance
	l.Info("Starting application with custom configuration",
		"time", time.Now(),
		"pid", os.Getpid(),
	)

	logger.Info("Direct logger usage",
		"custom_field", "test value",
	)

	// Add some error logging
	logger.Error("Example error message",
		"error_code", 500,
		"details", "This is a test error",
	)

	// Use logger with context
	userLogger := logger.With(
		"user_id", "123",
		"session", "abc456",
	)

	userLogger.Info("User action logged",
		"action", "login",
		"ip", "192.168.1.1",
	)

	// Wait a moment to ensure async writes complete
	time.Sleep(100 * time.Millisecond)
}
