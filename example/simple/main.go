package main

import (
	"errors"
	"github.com/baditaflorin/l"
)

func main() {
	// Simple usage with default logger
	l.Info("Starting application")

	// Log with additional context
	l.Info("User logged in",
		"user_id", "123",
		"ip", "192.168.1.1",
	)

	// Create a logger with persistent fields
	userLogger := l.With(
		"user_id", "123",
		"service", "auth",
	)

	userLogger.Info("Permission granted",
		"resource", "api/users",
		"action", "read",
	)

	// Error logging
	if err := someOperation(); err != nil {
		l.Error("Operation failed",
			"error", err,
			"operation", "user_update",
		)
	}
}

func someOperation() error {
	// Some operation that might return an error
	return error(errors.New("some error"))
}
