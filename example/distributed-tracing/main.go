package main

import (
	"context"
	"github.com/baditaflorin/l"
	"os"
	"time"
)

type TraceContext struct {
	TraceID      string
	SpanID       string
	ParentSpanID string
	ServiceName  string
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

	// Simulate a distributed transaction
	ctx := context.Background()
	trace := &TraceContext{
		TraceID:     "trace_abc123",
		SpanID:      "span_1",
		ServiceName: "order-service",
	}

	processOrder(ctx, trace)
}

func processOrder(ctx context.Context, trace *TraceContext) {
	logger := l.With(
		"trace_id", trace.TraceID,
		"span_id", trace.SpanID,
		"service", trace.ServiceName,
	)

	logger.Info("Starting order processing")

	// Call payment service
	paymentTrace := &TraceContext{
		TraceID:      trace.TraceID,
		SpanID:       "span_2",
		ParentSpanID: trace.SpanID,
		ServiceName:  "payment-service",
	}
	processPayment(ctx, paymentTrace)

	logger.Info("Order processing completed")
}

func processPayment(ctx context.Context, trace *TraceContext) {
	logger := l.With(
		"trace_id", trace.TraceID,
		"span_id", trace.SpanID,
		"parent_span_id", trace.ParentSpanID,
		"service", trace.ServiceName,
	)

	logger.Info("Processing payment")
	time.Sleep(time.Second) // Simulate API call
	logger.Info("Payment processed")
}
