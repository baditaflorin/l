// example/security-logging/main.go
package main

import (
	"github.com/baditaflorin/l"
	"os"
	"time"
)

type SecurityEvent struct {
	EventType   string
	Severity    string
	Source      string
	Description string
	UserID      string
	IPAddress   string
	Timestamp   time.Time
}

func main() {
	// Setup secure logging with both console and file output
	err := l.Setup(l.Options{
		Output:      os.Stdout,
		FilePath:    "security/events.log",
		JsonFormat:  true,
		AddSource:   true,
		MaxFileSize: 50 * 1024 * 1024, // 50MB
		MaxBackups:  90,               // Keep 90 days of security logs
	})
	if err != nil {
		panic(err)
	}
	defer l.Close()

	// Simulate security events
	events := []SecurityEvent{
		{
			EventType:   "FAILED_LOGIN",
			Severity:    "WARNING",
			Source:      "auth_service",
			Description: "Multiple failed login attempts",
			UserID:      "user123",
			IPAddress:   "192.168.1.100",
			Timestamp:   time.Now(),
		},
		{
			EventType:   "PERMISSION_CHANGE",
			Severity:    "HIGH",
			Source:      "admin_panel",
			Description: "User role elevated to admin",
			UserID:      "user456",
			IPAddress:   "192.168.1.101",
			Timestamp:   time.Now(),
		},
	}

	for _, event := range events {
		l.Error("Security event detected",
			"event_type", event.EventType,
			"severity", event.Severity,
			"source", event.Source,
			"description", event.Description,
			"user_id", event.UserID,
			"ip_address", event.IPAddress,
			"timestamp", event.Timestamp,
		)
	}
}
