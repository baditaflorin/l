// example/structured-logging/main.go
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
	err := l.Setup(l.Options{
		Output:     os.Stdout,
		JsonFormat: true,
		AddSource:  true,
	})
	if err != nil {
		panic(err)
	}
	defer l.Close()

	// Create logger with common fields
	txLogger := l.With(
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
}
