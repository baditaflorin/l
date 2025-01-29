// example/audit-trail/main.go
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
	// Setup logger with JSON format and file output for audit trail
	err := l.Setup(l.Options{
		FilePath:    "audit/audit.log",
		JsonFormat:  true,
		MaxFileSize: 10 * 1024 * 1024, // 10MB
		MaxBackups:  30,               // Keep 30 days of audit logs
	})
	if err != nil {
		panic(err)
	}
	defer l.Close()

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

	for _, event := range events {
		l.Info("Audit event",
			"user_id", event.UserID,
			"action", event.Action,
			"resource", event.Resource,
			"timestamp", event.Timestamp,
		)
	}
}
