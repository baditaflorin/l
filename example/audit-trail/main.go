package main

import (
	"github.com/baditaflorin/l"
	"time"
)

type AuditEvent struct {
	UserID    string
	Action    string
	Resource  string
	Timestamp time.Time
}

func main() {
	// Create factory and config
	factory := l.NewStandardFactory()
	config := l.Config{
		FilePath:    "audit/audit.log",
		JsonFormat:  true,
		MaxFileSize: 10 * 1024 * 1024, // 10MB
		MaxBackups:  30,               // Keep 30 days of audit logs
		AddSource:   true,             // Include source file and line for audit trail
	}

	// Create logger
	logger, err := factory.CreateLogger(config)
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	// Create a logger with common audit context
	auditLogger := logger.With(
		"component", "audit-system",
		"environment", "production",
	)

	// Simulate some audit events
	events := []AuditEvent{
		{
			UserID:    "user123",
			Action:    "LOGIN",
			Resource:  "webapp",
			Timestamp: time.Now(),
		},
		{
			UserID:    "user123",
			Action:    "VIEW_DOCUMENT",
			Resource:  "doc456",
			Timestamp: time.Now(),
		},
		{
			UserID:    "admin789",
			Action:    "MODIFY_SETTINGS",
			Resource:  "system_config",
			Timestamp: time.Now(),
		},
	}

	// Log each audit event
	for _, event := range events {
		auditLogger.Info("Audit event",
			"user_id", event.UserID,
			"action", event.Action,
			"resource", event.Resource,
			"timestamp", event.Timestamp,
		)
	}

	// Ensure all audit logs are written
	if err := logger.Flush(); err != nil {
		panic(err)
	}
}
