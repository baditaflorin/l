// File: example/structured-logging/main.go
package main

import (
	"github.com/baditaflorin/l"
	"os"
	"time"
)

type TransactionLog struct {
	TransactionID string
	Amount        float64
	Currency      string
	Status        string
	CustomerID    string
	Timestamp     time.Time
}

func main() {
	factory := l.NewStandardFactory()
	config := l.Config{
		Output:     os.Stdout,
		JsonFormat: true,
		AddSource:  true,
	}

	logger, err := factory.CreateLogger(config)
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	// Create logger with common fields
	txLogger := logger.With(
		"service", "payment-processor",
		"environment", "production",
	)

	// Log a complex transaction
	tx := TransactionLog{
		TransactionID: "tx_123456",
		Amount:        99.99,
		Currency:      "USD",
		Status:        "pending",
		CustomerID:    "cust_789",
		Timestamp:     time.Now(),
	}

	txLogger.Info("Processing transaction",
		"transaction_id", tx.TransactionID,
		"amount", tx.Amount,
		"currency", tx.Currency,
		"customer_id", tx.CustomerID,
	)

	// Simulate transaction processing
	time.Sleep(time.Second)

	// Log transaction completion
	txLogger.Info("Transaction completed",
		"transaction_id", tx.TransactionID,
		"status", "success",
		"processing_time_ms", 1000,
	)

	if err := logger.Flush(); err != nil {
		panic(err)
	}
}
